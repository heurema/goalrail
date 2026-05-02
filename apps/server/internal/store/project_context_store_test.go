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
	if got, want := len(call.args), 15; got != want {
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
