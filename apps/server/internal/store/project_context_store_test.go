package store

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

func TestProjectContextStoreBuildsRepoBindingUpsertWithSquirrelPlaceholders(t *testing.T) {
	ctx := context.Background()
	exec := &recordingProjectContextExecer{}
	store := newProjectContextStore(exec)
	now := time.Date(2026, 4, 26, 10, 0, 0, 0, time.UTC)

	err := store.UpsertRepoBinding(ctx, spine.RepoBinding{
		ID:                 "rpb_dev_default",
		OrganizationID:     "org_dev_default",
		ProjectID:          "prj_dev_default",
		Provider:           "custom_git",
		RepositoryFullName: "heurema/goalrail",
		RepositoryURL:      "https://example.invalid/heurema/goalrail.git",
		DefaultBranch:      "main",
		PathScope:          ".",
		AccessMode:         spine.RepoBindingAccessModeMetadataOnly,
		State:              spine.EntityStateActive,
		CreatedAt:          now,
		UpdatedAt:          now,
	})
	if err != nil {
		t.Fatalf("UpsertRepoBinding() error = %v", err)
	}

	if len(exec.calls) != 1 {
		t.Fatalf("Exec calls = %d, want 1", len(exec.calls))
	}
	call := exec.calls[0]
	if !strings.Contains(call.sql, "INSERT INTO repo_bindings") {
		t.Fatalf("SQL = %q, want repo_bindings insert", call.sql)
	}
	if !strings.Contains(call.sql, "ON CONFLICT (id) DO UPDATE SET") {
		t.Fatalf("SQL = %q, want idempotent conflict clause", call.sql)
	}
	if !strings.Contains(call.sql, "$1") {
		t.Fatalf("SQL = %q, want PostgreSQL placeholders", call.sql)
	}
	if got, want := len(call.args), 14; got != want {
		t.Fatalf("args len = %d, want %d", got, want)
	}
}

type recordingProjectContextExecer struct {
	calls []recordedExecCall
}

type recordedExecCall struct {
	sql  string
	args []any
}

func (r *recordingProjectContextExecer) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	r.calls = append(r.calls, recordedExecCall{
		sql:  sql,
		args: append([]any(nil), args...),
	})
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}
