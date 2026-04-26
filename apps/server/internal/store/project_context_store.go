package store

import (
	"context"
	"fmt"

	squirrel "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

type ProjectContextExecer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type ProjectContextQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type ProjectContextStore struct {
	exec  ProjectContextExecer
	query ProjectContextQuerier
	psql  squirrel.StatementBuilderType
}

func NewProjectContextStore(pool *pgxpool.Pool) *ProjectContextStore {
	return newProjectContextStore(pool, pool)
}

func NewProjectContextStoreWithExecutor(exec ProjectContextExecer) *ProjectContextStore {
	return newProjectContextStore(exec, nil)
}

func NewProjectContextStoreWithExecutorAndQuerier(exec ProjectContextExecer, query ProjectContextQuerier) *ProjectContextStore {
	return newProjectContextStore(exec, query)
}

func newProjectContextStore(exec ProjectContextExecer, query ProjectContextQuerier) *ProjectContextStore {
	return &ProjectContextStore{
		exec:  exec,
		query: query,
		psql:  squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (s *ProjectContextStore) UpsertUser(ctx context.Context, user spine.User) error {
	stmt := s.psql.
		Insert("users").
		Columns("id", "display_name", "email", "state", "created_at", "updated_at").
		Values(user.ID, user.DisplayName, user.Email, user.State, user.CreatedAt, user.UpdatedAt).
		Suffix("ON CONFLICT (id) DO UPDATE SET display_name = EXCLUDED.display_name, email = EXCLUDED.email, state = EXCLUDED.state, updated_at = EXCLUDED.updated_at")

	return s.execSQL(ctx, "upsert user", stmt)
}

func (s *ProjectContextStore) UpsertOrganization(ctx context.Context, org spine.Organization) error {
	stmt := s.psql.
		Insert("organizations").
		Columns("id", "slug", "display_name", "state", "created_at", "updated_at").
		Values(org.ID, org.Slug, org.DisplayName, org.State, org.CreatedAt, org.UpdatedAt).
		Suffix("ON CONFLICT (id) DO UPDATE SET slug = EXCLUDED.slug, display_name = EXCLUDED.display_name, state = EXCLUDED.state, updated_at = EXCLUDED.updated_at")

	return s.execSQL(ctx, "upsert organization", stmt)
}

func (s *ProjectContextStore) UpsertOrganizationMembership(ctx context.Context, membership spine.OrganizationMembership) error {
	stmt := s.psql.
		Insert("organization_memberships").
		Columns("id", "organization_id", "user_id", "role", "state", "created_at", "updated_at").
		Values(membership.ID, membership.OrganizationID, membership.UserID, membership.Role, membership.State, membership.CreatedAt, membership.UpdatedAt).
		Suffix("ON CONFLICT (id) DO UPDATE SET role = EXCLUDED.role, state = EXCLUDED.state, updated_at = EXCLUDED.updated_at")

	return s.execSQL(ctx, "upsert organization membership", stmt)
}

func (s *ProjectContextStore) UpsertProject(ctx context.Context, project spine.Project) error {
	stmt := s.psql.
		Insert("projects").
		Columns("id", "organization_id", "slug", "display_name", "state", "created_at", "updated_at").
		Values(project.ID, project.OrganizationID, project.Slug, project.DisplayName, project.State, project.CreatedAt, project.UpdatedAt).
		Suffix("ON CONFLICT (id) DO UPDATE SET slug = EXCLUDED.slug, display_name = EXCLUDED.display_name, state = EXCLUDED.state, updated_at = EXCLUDED.updated_at")

	return s.execSQL(ctx, "upsert project", stmt)
}

func (s *ProjectContextStore) UpsertRepoBinding(ctx context.Context, binding spine.RepoBinding) error {
	stmt := s.psql.
		Insert("repo_bindings").
		Columns(
			"id",
			"organization_id",
			"project_id",
			"vcs_connection_id",
			"provider",
			"repository_external_id",
			"repository_full_name",
			"repository_url",
			"default_branch",
			"path_scope",
			"access_mode",
			"state",
			"created_at",
			"updated_at",
		).
		Values(
			binding.ID,
			binding.OrganizationID,
			binding.ProjectID,
			binding.VcsConnectionID,
			binding.Provider,
			binding.RepositoryExternalID,
			binding.RepositoryFullName,
			binding.RepositoryURL,
			binding.DefaultBranch,
			binding.PathScope,
			binding.AccessMode,
			binding.State,
			binding.CreatedAt,
			binding.UpdatedAt,
		).
		Suffix("ON CONFLICT (id) DO UPDATE SET provider = EXCLUDED.provider, repository_external_id = EXCLUDED.repository_external_id, repository_full_name = EXCLUDED.repository_full_name, repository_url = EXCLUDED.repository_url, default_branch = EXCLUDED.default_branch, path_scope = EXCLUDED.path_scope, access_mode = EXCLUDED.access_mode, state = EXCLUDED.state, updated_at = EXCLUDED.updated_at")

	return s.execSQL(ctx, "upsert repo binding", stmt)
}

func (s *ProjectContextStore) ResolveRepoBinding(ctx context.Context, repoBindingID spine.RepoBindingID) (spine.ResolvedRepoBindingContext, bool, error) {
	if s.query == nil {
		return spine.ResolvedRepoBindingContext{}, false, fmt.Errorf("project context query executor is nil")
	}

	stmt := s.psql.
		Select("organization_id", "project_id", "id").
		From("repo_bindings").
		Where(squirrel.Eq{"id": repoBindingID})

	sqlText, args, err := stmt.ToSql()
	if err != nil {
		return spine.ResolvedRepoBindingContext{}, false, fmt.Errorf("resolve repo binding SQL: %w", err)
	}

	var resolved spine.ResolvedRepoBindingContext
	if err := s.query.QueryRow(ctx, sqlText, args...).Scan(
		&resolved.OrganizationID,
		&resolved.ProjectID,
		&resolved.RepoBindingID,
	); err != nil {
		if err == pgx.ErrNoRows {
			return spine.ResolvedRepoBindingContext{}, false, nil
		}
		return spine.ResolvedRepoBindingContext{}, false, fmt.Errorf("resolve repo binding: %w", err)
	}

	return resolved, true, nil
}

func (s *ProjectContextStore) execSQL(ctx context.Context, op string, sqlizer squirrel.Sqlizer) error {
	sqlText, args, err := sqlizer.ToSql()
	if err != nil {
		return fmt.Errorf("%s SQL: %w", op, err)
	}
	if _, err := s.exec.Exec(ctx, sqlText, args...); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}
