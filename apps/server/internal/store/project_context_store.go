package store

import (
	"context"
	"fmt"

	squirrel "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

type ProjectContextStore struct {
	exec  postgresExecer
	query postgresRowQuerier
	psql  squirrel.StatementBuilderType
}

func NewProjectContextStore(pool *pgxpool.Pool) *ProjectContextStore {
	db := newPostgresDB(pool)
	return newProjectContextStore(db, db)
}

func NewProjectContextStoreWithExecutor(exec postgresExecer) *ProjectContextStore {
	return newProjectContextStore(exec, nil)
}

func NewProjectContextStoreWithExecutorAndQuerier(exec postgresExecer, query postgresRowQuerier) *ProjectContextStore {
	return newProjectContextStore(exec, query)
}

func newProjectContextStore(exec postgresExecer, query postgresRowQuerier) *ProjectContextStore {
	return &ProjectContextStore{
		exec:  exec,
		query: query,
		psql:  squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (s *ProjectContextStore) UpsertUser(ctx context.Context, user spine.User) error {
	userID, err := uuidValue(user.ID, "user id")
	if err != nil {
		return err
	}
	stmt := s.psql.
		Insert("users").
		Columns("id", "display_name", "email", "state", "created_at", "updated_at").
		Values(userID, user.DisplayName, user.Email, user.State, user.CreatedAt, user.UpdatedAt).
		Suffix("ON CONFLICT (id) DO UPDATE SET display_name = EXCLUDED.display_name, email = EXCLUDED.email, state = EXCLUDED.state, updated_at = EXCLUDED.updated_at")

	return s.execSQL(ctx, "upsert user", stmt)
}

func (s *ProjectContextStore) UpsertInstallation(ctx context.Context, installation spine.Installation) error {
	installationID, err := uuidValue(installation.ID, "installation id")
	if err != nil {
		return err
	}
	stmt := s.psql.
		Insert("installations").
		Columns("id", "mode", "public_base_url", "state", "created_at", "updated_at").
		Values(installationID, installation.Mode, installation.PublicBaseURL, installation.State, installation.CreatedAt, installation.UpdatedAt).
		Suffix("ON CONFLICT (id) DO UPDATE SET mode = EXCLUDED.mode, public_base_url = EXCLUDED.public_base_url, state = EXCLUDED.state, updated_at = EXCLUDED.updated_at")

	return s.execSQL(ctx, "upsert installation", stmt)
}

func (s *ProjectContextStore) UpsertOrganization(ctx context.Context, org spine.Organization) error {
	orgID, err := uuidValue(org.ID, "organization id")
	if err != nil {
		return err
	}
	installationID, err := uuidValue(org.InstallationID, "organization installation id")
	if err != nil {
		return err
	}
	stmt := s.psql.
		Insert("organizations").
		Columns("id", "installation_id", "slug", "display_name", "state", "created_at", "updated_at").
		Values(orgID, installationID, org.Slug, org.DisplayName, org.State, org.CreatedAt, org.UpdatedAt).
		Suffix("ON CONFLICT (id) DO UPDATE SET installation_id = EXCLUDED.installation_id, slug = EXCLUDED.slug, display_name = EXCLUDED.display_name, state = EXCLUDED.state, updated_at = EXCLUDED.updated_at")

	return s.execSQL(ctx, "upsert organization", stmt)
}

func (s *ProjectContextStore) UpsertOrganizationMembership(ctx context.Context, membership spine.OrganizationMembership) error {
	membershipID, err := uuidValue(membership.ID, "organization membership id")
	if err != nil {
		return err
	}
	orgID, err := uuidValue(membership.OrganizationID, "organization membership organization id")
	if err != nil {
		return err
	}
	userID, err := uuidValue(membership.UserID, "organization membership user id")
	if err != nil {
		return err
	}
	stmt := s.psql.
		Insert("organization_memberships").
		Columns("id", "organization_id", "user_id", "role", "state", "created_at", "updated_at").
		Values(membershipID, orgID, userID, membership.Role, membership.State, membership.CreatedAt, membership.UpdatedAt).
		Suffix("ON CONFLICT (id) DO UPDATE SET role = EXCLUDED.role, state = EXCLUDED.state, updated_at = EXCLUDED.updated_at")

	return s.execSQL(ctx, "upsert organization membership", stmt)
}

func (s *ProjectContextStore) UpsertProject(ctx context.Context, project spine.Project) error {
	projectID, err := uuidValue(project.ID, "project id")
	if err != nil {
		return err
	}
	orgID, err := uuidValue(project.OrganizationID, "project organization id")
	if err != nil {
		return err
	}
	createdByUserID, err := uuidValue(project.CreatedByUserID, "project created by user id")
	if err != nil {
		return err
	}
	stmt := s.psql.
		Insert("projects").
		Columns("id", "organization_id", "created_by_user_id", "slug", "display_name", "state", "created_at", "updated_at").
		Values(projectID, orgID, createdByUserID, project.Slug, project.DisplayName, project.State, project.CreatedAt, project.UpdatedAt).
		Suffix("ON CONFLICT (id) DO UPDATE SET slug = EXCLUDED.slug, display_name = EXCLUDED.display_name, state = EXCLUDED.state, updated_at = EXCLUDED.updated_at")

	return s.execSQL(ctx, "upsert project", stmt)
}

func (s *ProjectContextStore) UpsertRepoBinding(ctx context.Context, binding spine.RepoBinding) error {
	stmt, err := s.repoBindingInsert(binding)
	if err != nil {
		return err
	}
	stmt = stmt.Suffix("ON CONFLICT (id) DO UPDATE SET provider = EXCLUDED.provider, repository_external_id = EXCLUDED.repository_external_id, repository_full_name = EXCLUDED.repository_full_name, repository_url = EXCLUDED.repository_url, default_branch = EXCLUDED.default_branch, workflow_base_branch = EXCLUDED.workflow_base_branch, path_scope = EXCLUDED.path_scope, access_mode = EXCLUDED.access_mode, state = EXCLUDED.state, updated_at = EXCLUDED.updated_at")

	return s.execSQL(ctx, "upsert repo binding", stmt)
}

func (s *ProjectContextStore) CreateRepoBinding(ctx context.Context, binding spine.RepoBinding) error {
	stmt, err := s.repoBindingInsert(binding)
	if err != nil {
		return err
	}
	return s.execSQL(ctx, "create repo binding", stmt)
}

func (s *ProjectContextStore) GetProject(ctx context.Context, projectID spine.ProjectID) (spine.Project, bool, error) {
	if s.query == nil {
		return spine.Project{}, false, fmt.Errorf("project context query executor is nil")
	}
	id, err := uuidValue(projectID, "project id")
	if err != nil {
		return spine.Project{}, false, err
	}
	stmt := s.psql.
		Select(
			"id",
			"organization_id",
			"created_by_user_id",
			"slug",
			"display_name",
			"state",
			"created_at",
			"updated_at",
		).
		From("projects").
		Where(squirrel.Eq{"id": id})

	sqlText, args, err := stmt.ToSql()
	if err != nil {
		return spine.Project{}, false, fmt.Errorf("get project SQL: %w", err)
	}

	var idValue string
	var organizationID string
	var createdByUserID string
	var state string
	var project spine.Project
	if err := s.query.QueryRow(ctx, sqlText, args...).Scan(
		&idValue,
		&organizationID,
		&createdByUserID,
		&project.Slug,
		&project.DisplayName,
		&state,
		&project.CreatedAt,
		&project.UpdatedAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			return spine.Project{}, false, nil
		}
		return spine.Project{}, false, fmt.Errorf("get project: %w", err)
	}
	project.ID = spine.ProjectID(idValue)
	project.OrganizationID = spine.OrganizationID(organizationID)
	project.CreatedByUserID = spine.UserID(createdByUserID)
	project.State = spine.EntityState(state)
	project.CreatedAt = project.CreatedAt.UTC()
	project.UpdatedAt = project.UpdatedAt.UTC()
	return project, true, nil
}

func (s *ProjectContextStore) GetActiveRepoBindingForProject(ctx context.Context, projectID spine.ProjectID) (spine.RepoBinding, bool, error) {
	if s.query == nil {
		return spine.RepoBinding{}, false, fmt.Errorf("project context query executor is nil")
	}
	id, err := uuidValue(projectID, "repo binding project id")
	if err != nil {
		return spine.RepoBinding{}, false, err
	}
	stmt := s.psql.
		Select(repoBindingColumns()...).
		From("repo_bindings").
		Where(squirrel.Eq{"project_id": id, "state": spine.EntityStateActive})

	sqlText, args, err := stmt.ToSql()
	if err != nil {
		return spine.RepoBinding{}, false, fmt.Errorf("get active repo binding SQL: %w", err)
	}

	binding, err := scanRepoBinding(s.query.QueryRow(ctx, sqlText, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return spine.RepoBinding{}, false, nil
		}
		return spine.RepoBinding{}, false, fmt.Errorf("get active repo binding: %w", err)
	}
	return binding, true, nil
}

func (s *ProjectContextStore) repoBindingInsert(binding spine.RepoBinding) (squirrel.InsertBuilder, error) {
	bindingID, err := uuidValue(binding.ID, "repo binding id")
	if err != nil {
		return squirrel.InsertBuilder{}, err
	}
	orgID, err := uuidValue(binding.OrganizationID, "repo binding organization id")
	if err != nil {
		return squirrel.InsertBuilder{}, err
	}
	projectID, err := uuidValue(binding.ProjectID, "repo binding project id")
	if err != nil {
		return squirrel.InsertBuilder{}, err
	}
	createdByUserID, err := uuidValue(binding.CreatedByUserID, "repo binding created by user id")
	if err != nil {
		return squirrel.InsertBuilder{}, err
	}
	stmt := s.psql.
		Insert("repo_bindings").
		Columns(
			"id",
			"organization_id",
			"project_id",
			"created_by_user_id",
			"vcs_connection_id",
			"provider",
			"repository_external_id",
			"repository_full_name",
			"repository_url",
			"default_branch",
			"workflow_base_branch",
			"path_scope",
			"access_mode",
			"state",
			"created_at",
			"updated_at",
		).
		Values(
			bindingID,
			orgID,
			projectID,
			createdByUserID,
			binding.VcsConnectionID,
			binding.Provider,
			binding.RepositoryExternalID,
			binding.RepositoryFullName,
			binding.RepositoryURL,
			binding.DefaultBranch,
			binding.WorkflowBaseBranch,
			binding.PathScope,
			binding.AccessMode,
			binding.State,
			binding.CreatedAt,
			binding.UpdatedAt,
		)
	return stmt, nil
}

func (s *ProjectContextStore) ResolveRepoBinding(ctx context.Context, repoBindingID spine.RepoBindingID) (spine.ResolvedRepoBindingContext, bool, error) {
	if s.query == nil {
		return spine.ResolvedRepoBindingContext{}, false, fmt.Errorf("project context query executor is nil")
	}
	bindingID, err := uuidValue(repoBindingID, "repo binding id")
	if err != nil {
		return spine.ResolvedRepoBindingContext{}, false, err
	}

	stmt := s.psql.
		Select("organization_id", "project_id", "id").
		From("repo_bindings").
		Where(squirrel.Eq{"id": bindingID})

	sqlText, args, err := stmt.ToSql()
	if err != nil {
		return spine.ResolvedRepoBindingContext{}, false, fmt.Errorf("resolve repo binding SQL: %w", err)
	}

	var organizationID string
	var projectID string
	var resolvedRepoBindingID string
	if err := s.query.QueryRow(ctx, sqlText, args...).Scan(
		&organizationID,
		&projectID,
		&resolvedRepoBindingID,
	); err != nil {
		if err == pgx.ErrNoRows {
			return spine.ResolvedRepoBindingContext{}, false, nil
		}
		return spine.ResolvedRepoBindingContext{}, false, fmt.Errorf("resolve repo binding: %w", err)
	}

	return spine.ResolvedRepoBindingContext{
		OrganizationID: spine.OrganizationID(organizationID),
		ProjectID:      spine.ProjectID(projectID),
		RepoBindingID:  spine.RepoBindingID(resolvedRepoBindingID),
	}, true, nil
}

func repoBindingColumns() []string {
	return []string{
		"id",
		"organization_id",
		"project_id",
		"created_by_user_id",
		"vcs_connection_id",
		"provider",
		"repository_external_id",
		"repository_full_name",
		"repository_url",
		"default_branch",
		"workflow_base_branch",
		"path_scope",
		"access_mode",
		"state",
		"created_at",
		"updated_at",
	}
}

func scanRepoBinding(row pgx.Row) (spine.RepoBinding, error) {
	var binding spine.RepoBinding
	var id string
	var organizationID string
	var projectID string
	var createdByUserID string
	var accessMode string
	var state string
	if err := row.Scan(
		&id,
		&organizationID,
		&projectID,
		&createdByUserID,
		&binding.VcsConnectionID,
		&binding.Provider,
		&binding.RepositoryExternalID,
		&binding.RepositoryFullName,
		&binding.RepositoryURL,
		&binding.DefaultBranch,
		&binding.WorkflowBaseBranch,
		&binding.PathScope,
		&accessMode,
		&state,
		&binding.CreatedAt,
		&binding.UpdatedAt,
	); err != nil {
		return spine.RepoBinding{}, err
	}
	binding.ID = spine.RepoBindingID(id)
	binding.OrganizationID = spine.OrganizationID(organizationID)
	binding.ProjectID = spine.ProjectID(projectID)
	binding.CreatedByUserID = spine.UserID(createdByUserID)
	binding.AccessMode = spine.RepoBindingAccessMode(accessMode)
	binding.State = spine.EntityState(state)
	binding.CreatedAt = binding.CreatedAt.UTC()
	binding.UpdatedAt = binding.UpdatedAt.UTC()
	return binding, nil
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

func uuidValue(value any, field string) (uuid.UUID, error) {
	id, err := uuid.Parse(fmt.Sprint(value))
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s must be uuid: %w", field, err)
	}
	if id.Version() != 7 {
		return uuid.Nil, fmt.Errorf("%s must be uuidv7", field)
	}
	return id, nil
}
