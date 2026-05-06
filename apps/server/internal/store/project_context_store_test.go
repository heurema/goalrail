package store

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

func TestProjectContextStoreBuildsInstallationUpsertWithModeAndPublicBaseURL(t *testing.T) {
	ctx := context.Background()
	exec := &recordingProjectContextExecer{}
	store := NewProjectContextStoreWithExecutor(exec)
	now := time.Date(2026, 5, 3, 10, 0, 0, 0, time.UTC)

	err := store.UpsertInstallation(ctx, spine.Installation{
		ID:            "018f0000-0000-7000-8000-000000000006",
		Mode:          spine.InstallationModeSelfHosted,
		PublicBaseURL: "http://localhost:8080",
		State:         spine.EntityStateActive,
		CreatedAt:     now,
		UpdatedAt:     now,
	})
	if err != nil {
		t.Fatalf("UpsertInstallation() error = %v", err)
	}

	if len(exec.calls) != 1 {
		t.Fatalf("Exec calls = %d, want 1", len(exec.calls))
	}
	call := exec.calls[0]
	if !strings.Contains(call.sql, "INSERT INTO installations") {
		t.Fatalf("SQL = %q, want installations insert", call.sql)
	}
	if !strings.Contains(call.sql, "ON CONFLICT (id) DO UPDATE SET") {
		t.Fatalf("SQL = %q, want idempotent conflict clause", call.sql)
	}
	if !strings.Contains(call.sql, "$1") {
		t.Fatalf("SQL = %q, want PostgreSQL placeholders", call.sql)
	}
	if got, want := len(call.args), 6; got != want {
		t.Fatalf("args len = %d, want %d", got, want)
	}
}

func TestProjectContextStoreBuildsOrganizationUpsertWithInstallationID(t *testing.T) {
	ctx := context.Background()
	exec := &recordingProjectContextExecer{}
	store := NewProjectContextStoreWithExecutor(exec)
	now := time.Date(2026, 5, 3, 10, 0, 0, 0, time.UTC)

	err := store.UpsertOrganization(ctx, spine.Organization{
		ID:             "018f0000-0000-7000-8000-000000000002",
		InstallationID: "018f0000-0000-7000-8000-000000000006",
		Slug:           "dev-default",
		DisplayName:    "Goalrail Dev Organization",
		State:          spine.EntityStateActive,
		CreatedAt:      now,
		UpdatedAt:      now,
	})
	if err != nil {
		t.Fatalf("UpsertOrganization() error = %v", err)
	}

	if len(exec.calls) != 1 {
		t.Fatalf("Exec calls = %d, want 1", len(exec.calls))
	}
	call := exec.calls[0]
	if !strings.Contains(call.sql, "INSERT INTO organizations") {
		t.Fatalf("SQL = %q, want organizations insert", call.sql)
	}
	if !strings.Contains(call.sql, "installation_id") {
		t.Fatalf("SQL = %q, want installation_id column", call.sql)
	}
	if got, want := len(call.args), 7; got != want {
		t.Fatalf("args len = %d, want %d", got, want)
	}
}

func TestProjectContextStoreBuildsRepoBindingUpsertWithSquirrelPlaceholders(t *testing.T) {
	ctx := context.Background()
	exec := &recordingProjectContextExecer{}
	store := NewProjectContextStoreWithExecutor(exec)
	now := time.Date(2026, 4, 26, 10, 0, 0, 0, time.UTC)

	err := store.UpsertRepoBinding(ctx, spine.RepoBinding{
		ID:                 "018f0000-0000-7000-8000-000000000004",
		OrganizationID:     "018f0000-0000-7000-8000-000000000002",
		ProjectID:          "018f0000-0000-7000-8000-000000000003",
		CreatedByUserID:    "018f0000-0000-7000-8000-000000000001",
		Provider:           "custom_git",
		RepositoryFullName: "heurema/goalrail",
		RepositoryURL:      "https://example.invalid/heurema/goalrail.git",
		DefaultBranch:      "main",
		WorkflowBaseBranch: "main",
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
	if !strings.Contains(call.sql, "workflow_base_branch") {
		t.Fatalf("SQL = %q, want workflow_base_branch column", call.sql)
	}
	if got, want := len(call.args), 16; got != want {
		t.Fatalf("args len = %d, want %d", got, want)
	}
}

func TestProjectContextStoreBuildsCreateProject(t *testing.T) {
	ctx := context.Background()
	exec := &recordingProjectContextExecer{}
	store := NewProjectContextStoreWithExecutor(exec)
	now := time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC)

	err := store.CreateProject(ctx, spine.Project{
		ID:              "018f0000-0000-7000-8000-000000000003",
		OrganizationID:  "018f0000-0000-7000-8000-000000000002",
		CreatedByUserID: "018f0000-0000-7000-8000-000000000001",
		Slug:            "github-acme-frontend",
		DisplayName:     "acme/frontend",
		State:           spine.EntityStateActive,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	if len(exec.calls) != 1 {
		t.Fatalf("Exec calls = %d, want 1", len(exec.calls))
	}
	call := exec.calls[0]
	if !strings.Contains(call.sql, "INSERT INTO projects") {
		t.Fatalf("SQL = %q, want projects insert", call.sql)
	}
	if strings.Contains(call.sql, "ON CONFLICT") {
		t.Fatalf("SQL = %q, want create without upsert conflict clause", call.sql)
	}
	if got, want := len(call.args), 8; got != want {
		t.Fatalf("args len = %d, want %d", got, want)
	}
}

func TestProjectContextStoreGetsProject(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC)
	query := &recordingProjectContextQuerier{
		row: fakeProjectContextRow{
			values: []any{
				"018f0000-0000-7000-8000-000000000003",
				"018f0000-0000-7000-8000-000000000002",
				"018f0000-0000-7000-8000-000000000001",
				"default",
				"Default Project",
				"active",
				now,
				now,
			},
		},
	}
	store := NewProjectContextStoreWithExecutorAndQuerier(&recordingProjectContextExecer{}, query)

	project, ok, err := store.GetProject(ctx, "018f0000-0000-7000-8000-000000000003")
	if err != nil {
		t.Fatalf("GetProject() error = %v", err)
	}
	if !ok {
		t.Fatal("GetProject() ok = false, want true")
	}
	if project.OrganizationID != "018f0000-0000-7000-8000-000000000002" {
		t.Fatalf("organization_id = %q, want 018f0000-0000-7000-8000-000000000002", project.OrganizationID)
	}
	if len(query.calls) != 1 {
		t.Fatalf("QueryRow calls = %d, want 1", len(query.calls))
	}
	call := query.calls[0]
	if !strings.Contains(call.sql, "FROM projects") {
		t.Fatalf("SQL = %q, want projects select", call.sql)
	}
	if !strings.Contains(call.sql, "$1") {
		t.Fatalf("SQL = %q, want PostgreSQL placeholders", call.sql)
	}
}

func TestProjectContextStoreGetsProjectByOrganizationAndSlug(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC)
	query := &recordingProjectContextQuerier{
		row: fakeProjectContextRow{
			values: []any{
				"018f0000-0000-7000-8000-000000000003",
				"018f0000-0000-7000-8000-000000000002",
				"018f0000-0000-7000-8000-000000000001",
				"github-acme-frontend",
				"acme/frontend",
				"active",
				now,
				now,
			},
		},
	}
	store := NewProjectContextStoreWithExecutorAndQuerier(&recordingProjectContextExecer{}, query)

	project, ok, err := store.GetProjectByOrganizationAndSlug(ctx, "018f0000-0000-7000-8000-000000000002", "github-acme-frontend")
	if err != nil {
		t.Fatalf("GetProjectByOrganizationAndSlug() error = %v", err)
	}
	if !ok {
		t.Fatal("GetProjectByOrganizationAndSlug() ok = false, want true")
	}
	if project.Slug != "github-acme-frontend" || project.DisplayName != "acme/frontend" {
		t.Fatalf("project = %#v, want repo-backed project", project)
	}
	if len(query.calls) != 1 {
		t.Fatalf("QueryRow calls = %d, want 1", len(query.calls))
	}
	call := query.calls[0]
	if !strings.Contains(call.sql, "FROM projects") || !strings.Contains(call.sql, "organization_id") || !strings.Contains(call.sql, "slug") {
		t.Fatalf("SQL = %q, want organization slug project lookup", call.sql)
	}
	if got, want := len(call.args), 2; got != want {
		t.Fatalf("args len = %d, want %d", got, want)
	}
}

func TestProjectContextStoreGetsActiveRepoBindingForProject(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC)
	query := &recordingProjectContextQuerier{
		row: fakeProjectContextRow{
			values: []any{
				"018f0000-0000-7000-8000-000000000004",
				"018f0000-0000-7000-8000-000000000002",
				"018f0000-0000-7000-8000-000000000003",
				"018f0000-0000-7000-8000-000000000001",
				"",
				"github",
				"",
				"heurema/goalrail",
				"git@github.com:heurema/goalrail.git",
				"main",
				"main",
				".",
				"metadata_only",
				"active",
				now,
				now,
			},
		},
	}
	store := NewProjectContextStoreWithExecutorAndQuerier(&recordingProjectContextExecer{}, query)

	binding, ok, err := store.GetActiveRepoBindingForProject(ctx, "018f0000-0000-7000-8000-000000000003")
	if err != nil {
		t.Fatalf("GetActiveRepoBindingForProject() error = %v", err)
	}
	if !ok {
		t.Fatal("GetActiveRepoBindingForProject() ok = false, want true")
	}
	if binding.WorkflowBaseBranch != "main" {
		t.Fatalf("workflow_base_branch = %q, want main", binding.WorkflowBaseBranch)
	}
	if len(query.calls) != 1 {
		t.Fatalf("QueryRow calls = %d, want 1", len(query.calls))
	}
	call := query.calls[0]
	if !strings.Contains(call.sql, "FROM repo_bindings") {
		t.Fatalf("SQL = %q, want repo_bindings select", call.sql)
	}
	if !strings.Contains(call.sql, "workflow_base_branch") {
		t.Fatalf("SQL = %q, want workflow_base_branch column", call.sql)
	}
}

func TestProjectContextStoreGetsActiveRepoBindingByOrganizationRepository(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC)
	query := &recordingProjectContextQuerier{
		row: fakeProjectContextRow{
			values: []any{
				"018f0000-0000-7000-8000-000000000004",
				"018f0000-0000-7000-8000-000000000002",
				"018f0000-0000-7000-8000-000000000003",
				"018f0000-0000-7000-8000-000000000001",
				"",
				"github",
				"",
				"acme/frontend",
				"git@github.com:acme/frontend.git",
				"main",
				"main",
				".",
				"metadata_only",
				"active",
				now,
				now,
			},
		},
	}
	store := NewProjectContextStoreWithExecutorAndQuerier(&recordingProjectContextExecer{}, query)

	binding, ok, err := store.GetActiveRepoBindingByOrganizationAndRepository(ctx, "018f0000-0000-7000-8000-000000000002", "GitHub", "ACME/frontend")
	if err != nil {
		t.Fatalf("GetActiveRepoBindingByOrganizationAndRepository() error = %v", err)
	}
	if !ok {
		t.Fatal("GetActiveRepoBindingByOrganizationAndRepository() ok = false, want true")
	}
	if binding.RepositoryFullName != "acme/frontend" {
		t.Fatalf("repository_full_name = %q, want acme/frontend", binding.RepositoryFullName)
	}
	if len(query.calls) != 1 {
		t.Fatalf("QueryRow calls = %d, want 1", len(query.calls))
	}
	call := query.calls[0]
	if !strings.Contains(call.sql, "lower(provider)") || !strings.Contains(call.sql, "lower(repository_full_name)") {
		t.Fatalf("SQL = %q, want lower-case repository identity lookup", call.sql)
	}
	if got, want := len(call.args), 4; got != want {
		t.Fatalf("args len = %d, want %d", got, want)
	}
}

func TestProjectContextStoreGetsRepoBinding(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC)
	query := &recordingProjectContextQuerier{
		row: fakeProjectContextRow{
			values: []any{
				"018f0000-0000-7000-8000-000000000004",
				"018f0000-0000-7000-8000-000000000002",
				"018f0000-0000-7000-8000-000000000003",
				"018f0000-0000-7000-8000-000000000001",
				"",
				"github",
				"",
				"heurema/goalrail",
				"git@github.com:heurema/goalrail.git",
				"main",
				"main",
				".",
				"metadata_only",
				"active",
				now,
				now,
			},
		},
	}
	store := NewProjectContextStoreWithExecutorAndQuerier(&recordingProjectContextExecer{}, query)

	binding, ok, err := store.GetRepoBinding(ctx, "018f0000-0000-7000-8000-000000000004")
	if err != nil {
		t.Fatalf("GetRepoBinding() error = %v", err)
	}
	if !ok {
		t.Fatal("GetRepoBinding() ok = false, want true")
	}
	if binding.ID != "018f0000-0000-7000-8000-000000000004" || binding.RepositoryFullName != "heurema/goalrail" {
		t.Fatalf("binding = %#v, want persisted repo binding", binding)
	}
	if !strings.Contains(query.calls[0].sql, "FROM repo_bindings") {
		t.Fatalf("SQL = %q, want repo_bindings select", query.calls[0].sql)
	}
}

func TestProjectContextStoreBuildsRepositoryContextSnapshotCreate(t *testing.T) {
	ctx := context.Background()
	exec := &recordingProjectContextExecer{}
	store := NewProjectContextStoreWithExecutor(exec)

	err := store.CreateRepositoryContextSnapshot(ctx, spine.RepositoryContextSnapshotRecord{
		ID:             "018f0000-0000-7000-8000-000000000301",
		OrganizationID: "018f0000-0000-7000-8000-000000000002",
		ProjectID:      "018f0000-0000-7000-8000-000000000003",
		RepoBindingID:  "018f0000-0000-7000-8000-000000000004",
		Source:         "goalrail_cli_init",
		SchemaVersion:  1,
		Fingerprint:    "sha256:abc123",
		Snapshot:       []byte(`{"schema_version":1}`),
		CreatedAt:      time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("CreateRepositoryContextSnapshot() error = %v", err)
	}

	if len(exec.calls) != 1 {
		t.Fatalf("Exec calls = %d, want 1", len(exec.calls))
	}
	call := exec.calls[0]
	if !strings.Contains(call.sql, "INSERT INTO repository_context_snapshots") {
		t.Fatalf("SQL = %q, want repository_context_snapshots insert", call.sql)
	}
	if strings.Contains(call.sql, "ON CONFLICT") {
		t.Fatalf("SQL = %q, want create without upsert conflict clause", call.sql)
	}
	if got, want := len(call.args), 9; got != want {
		t.Fatalf("args len = %d, want %d", got, want)
	}
}

func TestProjectContextStoreGetsRepositoryContextSnapshotByFingerprint(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC)
	query := &recordingProjectContextQuerier{
		row: fakeProjectContextRow{
			values: []any{
				"018f0000-0000-7000-8000-000000000301",
				"018f0000-0000-7000-8000-000000000002",
				"018f0000-0000-7000-8000-000000000003",
				"018f0000-0000-7000-8000-000000000004",
				"goalrail_cli_init",
				1,
				"sha256:abc123",
				[]byte(`{"schema_version":1}`),
				now,
			},
		},
	}
	store := NewProjectContextStoreWithExecutorAndQuerier(&recordingProjectContextExecer{}, query)

	record, ok, err := store.GetRepositoryContextSnapshotByFingerprint(ctx, "018f0000-0000-7000-8000-000000000004", "sha256:abc123")
	if err != nil {
		t.Fatalf("GetRepositoryContextSnapshotByFingerprint() error = %v", err)
	}
	if !ok {
		t.Fatal("GetRepositoryContextSnapshotByFingerprint() ok = false, want true")
	}
	if record.ID != "018f0000-0000-7000-8000-000000000301" || record.Fingerprint != "sha256:abc123" {
		t.Fatalf("record = %#v, want persisted snapshot", record)
	}
	if !strings.Contains(query.calls[0].sql, "FROM repository_context_snapshots") {
		t.Fatalf("SQL = %q, want repository_context_snapshots select", query.calls[0].sql)
	}
	if got, want := len(query.calls[0].args), 2; got != want {
		t.Fatalf("args len = %d, want %d", got, want)
	}
}

func TestProjectContextStoreResolvesRepoBindingContext(t *testing.T) {
	ctx := context.Background()
	query := &recordingProjectContextQuerier{
		row: fakeProjectContextRow{
			values: []any{"018f0000-0000-7000-8000-000000000002", "018f0000-0000-7000-8000-000000000003", "018f0000-0000-7000-8000-000000000004"},
		},
	}
	store := NewProjectContextStoreWithExecutorAndQuerier(&recordingProjectContextExecer{}, query)

	resolved, ok, err := store.ResolveRepoBinding(ctx, "018f0000-0000-7000-8000-000000000004")
	if err != nil {
		t.Fatalf("ResolveRepoBinding() error = %v", err)
	}
	if !ok {
		t.Fatal("ResolveRepoBinding() ok = false, want true")
	}
	if resolved.OrganizationID != "018f0000-0000-7000-8000-000000000002" {
		t.Fatalf("organization_id = %q, want 018f0000-0000-7000-8000-000000000002", resolved.OrganizationID)
	}
	if resolved.ProjectID != "018f0000-0000-7000-8000-000000000003" {
		t.Fatalf("project_id = %q, want 018f0000-0000-7000-8000-000000000003", resolved.ProjectID)
	}
	if resolved.RepoBindingID != "018f0000-0000-7000-8000-000000000004" {
		t.Fatalf("repo_binding_id = %q, want 018f0000-0000-7000-8000-000000000004", resolved.RepoBindingID)
	}
	if len(query.calls) != 1 {
		t.Fatalf("QueryRow calls = %d, want 1", len(query.calls))
	}
	call := query.calls[0]
	if !strings.Contains(call.sql, "FROM repo_bindings") {
		t.Fatalf("SQL = %q, want repo_bindings select", call.sql)
	}
	if !strings.Contains(call.sql, "$1") {
		t.Fatalf("SQL = %q, want PostgreSQL placeholders", call.sql)
	}
	if got, want := len(call.args), 1; got != want {
		t.Fatalf("args len = %d, want %d", got, want)
	}
}

func TestProjectContextStoreResolveRepoBindingReturnsNotFound(t *testing.T) {
	ctx := context.Background()
	query := &recordingProjectContextQuerier{row: fakeProjectContextRow{err: pgx.ErrNoRows}}
	store := NewProjectContextStoreWithExecutorAndQuerier(&recordingProjectContextExecer{}, query)

	_, ok, err := store.ResolveRepoBinding(ctx, "018f0000-0000-7000-8000-000000000099")
	if err != nil {
		t.Fatalf("ResolveRepoBinding() error = %v", err)
	}
	if ok {
		t.Fatal("ResolveRepoBinding() ok = true, want false")
	}
}

type recordingProjectContextExecer struct {
	calls []recordedExecCall
	tag   pgconn.CommandTag
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
	if r.tag.String() != "" {
		return r.tag, nil
	}
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}

type recordingProjectContextQuerier struct {
	calls []recordedExecCall
	row   pgx.Row
}

func (r *recordingProjectContextQuerier) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	r.calls = append(r.calls, recordedExecCall{
		sql:  sql,
		args: append([]any(nil), args...),
	})
	return r.row
}

type fakeProjectContextRow struct {
	values []any
	err    error
}

func (r fakeProjectContextRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) != len(r.values) {
		return errors.New("unexpected scan destination count")
	}
	for i := range dest {
		switch target := dest[i].(type) {
		case *string:
			value, ok := r.values[i].(string)
			if !ok {
				return errors.New("uuid value is not string")
			}
			*target = value
		case *[]byte:
			if r.values[i] == nil {
				*target = nil
				continue
			}
			value, ok := r.values[i].([]byte)
			if !ok {
				return errors.New("json value is not bytes")
			}
			*target = append([]byte(nil), value...)
		case *bool:
			value, ok := r.values[i].(bool)
			if !ok {
				return errors.New("bool value is not bool")
			}
			*target = value
		case *int:
			value, ok := r.values[i].(int)
			if !ok {
				return errors.New("int value is not int")
			}
			*target = value
		case *time.Time:
			value, ok := r.values[i].(time.Time)
			if !ok {
				return errors.New("time value is not time")
			}
			*target = value
		case *pgtype.Int4:
			if r.values[i] == nil {
				*target = pgtype.Int4{Valid: false}
				continue
			}
			value, ok := r.values[i].(int32)
			if !ok {
				return errors.New("int4 value is not int32")
			}
			*target = pgtype.Int4{Int32: value, Valid: true}
		case *pgtype.Timestamptz:
			if r.values[i] == nil {
				*target = pgtype.Timestamptz{Valid: false}
				continue
			}
			value, ok := r.values[i].(time.Time)
			if !ok {
				return errors.New("timestamptz value is not time")
			}
			*target = pgtype.Timestamptz{Time: value, Valid: true}
		case *pgtype.UUID:
			if r.values[i] == nil {
				*target = pgtype.UUID{Valid: false}
				continue
			}
			value, ok := r.values[i].(string)
			if !ok {
				return errors.New("uuid value is not string")
			}
			if err := target.Scan(value); err != nil {
				return err
			}
		case *spine.IntakeID:
			value, ok := r.values[i].(string)
			if !ok {
				return errors.New("intake id value is not string")
			}
			*target = spine.IntakeID(value)
		case *spine.IntakeState:
			value, ok := r.values[i].(spine.IntakeState)
			if !ok {
				return errors.New("intake state value is not intake state")
			}
			*target = value
		case *spine.GoalID:
			value, ok := r.values[i].(string)
			if !ok {
				return errors.New("goal id value is not string")
			}
			*target = spine.GoalID(value)
		case *spine.GoalState:
			value, ok := r.values[i].(spine.GoalState)
			if !ok {
				return errors.New("goal state value is not goal state")
			}
			*target = value
		case *spine.ClarificationRequestState:
			value, ok := r.values[i].(spine.ClarificationRequestState)
			if !ok {
				return errors.New("clarification request state value is not clarification request state")
			}
			*target = value
		case *spine.OrganizationID:
			value, ok := r.values[i].(string)
			if !ok {
				return errors.New("organization id value is not string")
			}
			*target = spine.OrganizationID(value)
		case *spine.ProjectID:
			value, ok := r.values[i].(string)
			if !ok {
				return errors.New("project id value is not string")
			}
			*target = spine.ProjectID(value)
		case *spine.RepoBindingID:
			value, ok := r.values[i].(string)
			if !ok {
				return errors.New("repo binding id value is not string")
			}
			*target = spine.RepoBindingID(value)
		default:
			return errors.New("unexpected scan target")
		}
	}
	return nil
}
