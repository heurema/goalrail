package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func TestMigrationsApplyFromScratchAndRecordInitSchemaParityVersion(t *testing.T) {
	db := openDockerPostgresForMigrationTest(t)
	configureGooseForMigrationTest(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := goose.UpContext(ctx, db, "."); err != nil {
		t.Fatalf("goose.UpContext() error = %v", err)
	}

	assertMigrationVersionApplied(t, db, 11)
}

func TestInitSchemaParityMigrationAppliesWhenEffectsAlreadyExistButVersionMissing(t *testing.T) {
	db := openDockerPostgresForMigrationTest(t)
	configureGooseForMigrationTest(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := goose.UpToContext(ctx, db, ".", 10); err != nil {
		t.Fatalf("goose.UpToContext(version=10) error = %v", err)
	}

	assertMigrationVersionNotApplied(t, db, 11)
	assertColumnExists(t, db, "repo_bindings", "workflow_base_branch")
	assertTableExists(t, db, "repository_context_snapshots")
	assertTableExists(t, db, "work_item_plan_leases")
	assertRelationExists(t, db, "users_email_lower_unique")
	assertRelationExists(t, db, "repo_bindings_one_active_per_org_repository_idx")

	if err := goose.UpContext(ctx, db, "."); err != nil {
		t.Fatalf("goose.UpContext() after existing effects error = %v", err)
	}

	assertMigrationVersionApplied(t, db, 11)
}

func openDockerPostgresForMigrationTest(t *testing.T) *sql.DB {
	t.Helper()
	if os.Getenv("GOALRAIL_RUN_POSTGRES_MIGRATION_TESTS") != "1" {
		t.Skip("set GOALRAIL_RUN_POSTGRES_MIGRATION_TESTS=1 to run docker-backed migration integration tests")
	}
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skipf("docker not available: %v", err)
	}

	containerName := fmt.Sprintf("goalrail-migrations-%d", time.Now().UnixNano())
	run := exec.Command(
		"docker", "run", "--rm", "-d",
		"--name", containerName,
		"-e", "POSTGRES_USER=postgres",
		"-e", "POSTGRES_PASSWORD=goalrailtest",
		"-e", "POSTGRES_DB=goalrail_test",
		"-p", "127.0.0.1::5432",
		"postgres:16-alpine",
	)
	containerID, err := run.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			t.Fatalf("docker run failed: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		t.Fatalf("docker run failed: %v", err)
	}
	t.Cleanup(func() {
		_ = exec.Command("docker", "rm", "-f", strings.TrimSpace(string(containerID))).Run()
	})

	portOutput, err := exec.Command("docker", "port", containerName, "5432/tcp").Output()
	if err != nil {
		t.Fatalf("docker port failed: %v", err)
	}
	endpoint := strings.TrimSpace(strings.SplitN(string(portOutput), "\n", 2)[0])
	_, port, ok := strings.Cut(endpoint, ":")
	if !ok || port == "" {
		t.Fatalf("unexpected docker port output: %q", strings.TrimSpace(string(portOutput)))
	}

	db, err := sql.Open("pgx", fmt.Sprintf("postgres://postgres:goalrailtest@127.0.0.1:%s/goalrail_test?sslmode=disable", port))
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	deadline := time.Now().Add(30 * time.Second)
	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		err := db.PingContext(ctx)
		cancel()
		if err == nil {
			return db
		}
		if time.Now().After(deadline) {
			t.Fatalf("postgres container did not become ready")
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func configureGooseForMigrationTest(t *testing.T) {
	t.Helper()
	goose.SetBaseFS(FS)
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("goose.SetDialect() error = %v", err)
	}
}

func assertMigrationVersionApplied(t *testing.T, db *sql.DB, version int) {
	t.Helper()
	if !migrationVersionApplied(t, db, version) {
		t.Fatalf("migration version %d is not applied", version)
	}
}

func assertMigrationVersionNotApplied(t *testing.T, db *sql.DB, version int) {
	t.Helper()
	if migrationVersionApplied(t, db, version) {
		t.Fatalf("migration version %d is unexpectedly applied", version)
	}
}

func migrationVersionApplied(t *testing.T, db *sql.DB, version int) bool {
	t.Helper()
	var applied bool
	row := db.QueryRow(
		"SELECT EXISTS (SELECT 1 FROM goose_db_version WHERE version_id = $1 AND is_applied)",
		version,
	)
	if err := row.Scan(&applied); err != nil {
		t.Fatalf("query migration version %d: %v", version, err)
	}
	return applied
}

func assertTableExists(t *testing.T, db *sql.DB, table string) {
	t.Helper()
	var exists bool
	row := db.QueryRow("SELECT to_regclass($1) IS NOT NULL", "public."+table)
	if err := row.Scan(&exists); err != nil {
		t.Fatalf("query table %s: %v", table, err)
	}
	if !exists {
		t.Fatalf("table %s does not exist", table)
	}
}

func assertRelationExists(t *testing.T, db *sql.DB, relation string) {
	t.Helper()
	var exists bool
	row := db.QueryRow("SELECT to_regclass($1) IS NOT NULL", "public."+relation)
	if err := row.Scan(&exists); err != nil {
		t.Fatalf("query relation %s: %v", relation, err)
	}
	if !exists {
		t.Fatalf("relation %s does not exist", relation)
	}
}

func assertColumnExists(t *testing.T, db *sql.DB, table, column string) {
	t.Helper()
	var exists bool
	row := db.QueryRow(
		`SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = 'public'
				AND table_name = $1
				AND column_name = $2
		)`,
		table,
		column,
	)
	if err := row.Scan(&exists); err != nil {
		t.Fatalf("query column %s.%s: %v", table, column, err)
	}
	if !exists {
		t.Fatalf("column %s.%s does not exist", table, column)
	}
}
