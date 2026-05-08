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
	rows  postgresRowsQuerier
	psql  squirrel.StatementBuilderType
}

func NewProjectContextStore(pool *pgxpool.Pool) *ProjectContextStore {
	db := newPostgresDB(pool)
	return newProjectContextStore(db, db, db)
}

func NewProjectContextStoreWithExecutor(exec postgresExecer) *ProjectContextStore {
	return newProjectContextStore(exec, nil, nil)
}

func NewProjectContextStoreWithExecutorAndQuerier(exec postgresExecer, query postgresRowQuerier) *ProjectContextStore {
	var rows postgresRowsQuerier
	if candidate, ok := query.(postgresRowsQuerier); ok {
		rows = candidate
	}
	return newProjectContextStore(exec, query, rows)
}

func NewProjectContextStoreWithExecutorQuerierAndRows(exec postgresExecer, query postgresRowQuerier, rows postgresRowsQuerier) *ProjectContextStore {
	return newProjectContextStore(exec, query, rows)
}

func newProjectContextStore(exec postgresExecer, query postgresRowQuerier, rows postgresRowsQuerier) *ProjectContextStore {
	return &ProjectContextStore{
		exec:  exec,
		query: query,
		rows:  rows,
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

func (s *ProjectContextStore) GetOrganization(ctx context.Context, organizationID spine.OrganizationID) (spine.Organization, bool, error) {
	if s.query == nil {
		return spine.Organization{}, false, fmt.Errorf("project context query executor is nil")
	}
	id, err := uuidValue(organizationID, "organization id")
	if err != nil {
		return spine.Organization{}, false, err
	}
	stmt := s.psql.
		Select(organizationColumns()...).
		From("organizations").
		Where(squirrel.Eq{"id": id})

	sqlText, args, err := stmt.ToSql()
	if err != nil {
		return spine.Organization{}, false, fmt.Errorf("get organization SQL: %w", err)
	}

	organization, err := scanOrganization(s.query.QueryRow(ctx, sqlText, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return spine.Organization{}, false, nil
		}
		return spine.Organization{}, false, fmt.Errorf("get organization: %w", err)
	}
	return organization, true, nil
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

func (s *ProjectContextStore) CreateProject(ctx context.Context, project spine.Project) error {
	stmt, err := s.projectInsert(project)
	if err != nil {
		return err
	}
	if err := s.execSQL(ctx, "create project", stmt); err != nil {
		return wrapUniqueConstraint(err)
	}
	return nil
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
	if err := s.execSQL(ctx, "create repo binding", stmt); err != nil {
		return wrapUniqueConstraint(err)
	}
	return nil
}

func (s *ProjectContextStore) projectInsert(project spine.Project) (squirrel.InsertBuilder, error) {
	projectID, err := uuidValue(project.ID, "project id")
	if err != nil {
		return squirrel.InsertBuilder{}, err
	}
	orgID, err := uuidValue(project.OrganizationID, "project organization id")
	if err != nil {
		return squirrel.InsertBuilder{}, err
	}
	createdByUserID, err := uuidValue(project.CreatedByUserID, "project created by user id")
	if err != nil {
		return squirrel.InsertBuilder{}, err
	}
	stmt := s.psql.
		Insert("projects").
		Columns("id", "organization_id", "created_by_user_id", "slug", "display_name", "state", "created_at", "updated_at").
		Values(projectID, orgID, createdByUserID, project.Slug, project.DisplayName, project.State, project.CreatedAt, project.UpdatedAt)
	return stmt, nil
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
		Select(projectColumns()...).
		From("projects").
		Where(squirrel.Eq{"id": id})

	sqlText, args, err := stmt.ToSql()
	if err != nil {
		return spine.Project{}, false, fmt.Errorf("get project SQL: %w", err)
	}

	project, err := scanProject(s.query.QueryRow(ctx, sqlText, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return spine.Project{}, false, nil
		}
		return spine.Project{}, false, fmt.Errorf("get project: %w", err)
	}
	return project, true, nil
}

func (s *ProjectContextStore) GetProjectByOrganizationAndSlug(ctx context.Context, organizationID spine.OrganizationID, slug string) (spine.Project, bool, error) {
	if s.query == nil {
		return spine.Project{}, false, fmt.Errorf("project context query executor is nil")
	}
	orgID, err := uuidValue(organizationID, "project organization id")
	if err != nil {
		return spine.Project{}, false, err
	}
	stmt := s.psql.
		Select(projectColumns()...).
		From("projects").
		Where(squirrel.Eq{"organization_id": orgID, "slug": slug})

	sqlText, args, err := stmt.ToSql()
	if err != nil {
		return spine.Project{}, false, fmt.Errorf("get project by organization and slug SQL: %w", err)
	}

	project, err := scanProject(s.query.QueryRow(ctx, sqlText, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return spine.Project{}, false, nil
		}
		return spine.Project{}, false, fmt.Errorf("get project by organization and slug: %w", err)
	}
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

func (s *ProjectContextStore) GetActiveRepoBindingByOrganizationAndRepository(ctx context.Context, organizationID spine.OrganizationID, provider string, repositoryFullName string) (spine.RepoBinding, bool, error) {
	if s.query == nil {
		return spine.RepoBinding{}, false, fmt.Errorf("project context query executor is nil")
	}
	orgID, err := uuidValue(organizationID, "repo binding organization id")
	if err != nil {
		return spine.RepoBinding{}, false, err
	}
	stmt := s.psql.
		Select(repoBindingColumns()...).
		From("repo_bindings").
		Where(squirrel.Expr("organization_id = ? AND lower(provider) = lower(?) AND lower(repository_full_name) = lower(?) AND state = ?", orgID, provider, repositoryFullName, spine.EntityStateActive))

	sqlText, args, err := stmt.ToSql()
	if err != nil {
		return spine.RepoBinding{}, false, fmt.Errorf("get active repo binding by organization repository SQL: %w", err)
	}

	binding, err := scanRepoBinding(s.query.QueryRow(ctx, sqlText, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return spine.RepoBinding{}, false, nil
		}
		return spine.RepoBinding{}, false, fmt.Errorf("get active repo binding by organization repository: %w", err)
	}
	return binding, true, nil
}

func (s *ProjectContextStore) ListActiveProjectRepoBindingContexts(ctx context.Context, organizationID spine.OrganizationID) ([]spine.ProjectRepoBindingContext, error) {
	if s.rows == nil {
		return nil, fmt.Errorf("project context rows executor is nil")
	}
	orgID, err := uuidValue(organizationID, "organization id")
	if err != nil {
		return nil, err
	}

	projectColumns := prefixedColumns("p", projectColumns())
	repoBindingColumns := prefixedColumns("rb", repoBindingColumns())
	stmt := s.psql.
		Select(append(projectColumns, repoBindingColumns...)...).
		From("projects p").
		Join("repo_bindings rb ON rb.project_id = p.id").
		Where(squirrel.Eq{
			"p.organization_id":  orgID,
			"p.state":            spine.EntityStateActive,
			"rb.organization_id": orgID,
			"rb.state":           spine.EntityStateActive,
		}).
		OrderBy("p.created_at ASC", "p.id ASC", "rb.created_at ASC", "rb.id ASC")

	sqlText, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("list active project repo binding contexts SQL: %w", err)
	}
	rows, err := s.rows.Query(ctx, sqlText, args...)
	if err != nil {
		return nil, fmt.Errorf("list active project repo binding contexts: %w", err)
	}
	defer rows.Close()

	var contexts []spine.ProjectRepoBindingContext
	for rows.Next() {
		project, binding, err := scanProjectRepoBindingContext(rows)
		if err != nil {
			return nil, fmt.Errorf("scan project repo binding context: %w", err)
		}
		contexts = append(contexts, spine.ProjectRepoBindingContext{
			Project:     project,
			RepoBinding: binding,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate project repo binding contexts: %w", err)
	}
	return contexts, nil
}

func (s *ProjectContextStore) GetRepoBinding(ctx context.Context, repoBindingID spine.RepoBindingID) (spine.RepoBinding, bool, error) {
	if s.query == nil {
		return spine.RepoBinding{}, false, fmt.Errorf("project context query executor is nil")
	}
	id, err := uuidValue(repoBindingID, "repo binding id")
	if err != nil {
		return spine.RepoBinding{}, false, err
	}
	stmt := s.psql.
		Select(repoBindingColumns()...).
		From("repo_bindings").
		Where(squirrel.Eq{"id": id})

	sqlText, args, err := stmt.ToSql()
	if err != nil {
		return spine.RepoBinding{}, false, fmt.Errorf("get repo binding SQL: %w", err)
	}

	binding, err := scanRepoBinding(s.query.QueryRow(ctx, sqlText, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return spine.RepoBinding{}, false, nil
		}
		return spine.RepoBinding{}, false, fmt.Errorf("get repo binding: %w", err)
	}
	return binding, true, nil
}

func (s *ProjectContextStore) CreateRepositoryContextSnapshot(ctx context.Context, record spine.RepositoryContextSnapshotRecord) error {
	snapshotID, err := uuidValue(record.ID, "repository context snapshot id")
	if err != nil {
		return err
	}
	orgID, err := uuidValue(record.OrganizationID, "repository context snapshot organization id")
	if err != nil {
		return err
	}
	projectID, err := uuidValue(record.ProjectID, "repository context snapshot project id")
	if err != nil {
		return err
	}
	repoBindingID, err := uuidValue(record.RepoBindingID, "repository context snapshot repo binding id")
	if err != nil {
		return err
	}
	stmt := s.psql.
		Insert("repository_context_snapshots").
		Columns(
			"id",
			"organization_id",
			"project_id",
			"repo_binding_id",
			"source",
			"schema_version",
			"fingerprint",
			"snapshot",
			"created_at",
		).
		Values(
			snapshotID,
			orgID,
			projectID,
			repoBindingID,
			record.Source,
			record.SchemaVersion,
			record.Fingerprint,
			record.Snapshot,
			record.CreatedAt.UTC(),
		)
	if err := s.execSQL(ctx, "create repository context snapshot", stmt); err != nil {
		return wrapUniqueConstraint(err)
	}
	return nil
}

func (s *ProjectContextStore) GetRepositoryContextSnapshotByFingerprint(ctx context.Context, repoBindingID spine.RepoBindingID, fingerprint string) (spine.RepositoryContextSnapshotRecord, bool, error) {
	if s.query == nil {
		return spine.RepositoryContextSnapshotRecord{}, false, fmt.Errorf("project context query executor is nil")
	}
	id, err := uuidValue(repoBindingID, "repository context snapshot repo binding id")
	if err != nil {
		return spine.RepositoryContextSnapshotRecord{}, false, err
	}
	stmt := s.psql.
		Select(repositoryContextSnapshotColumns()...).
		From("repository_context_snapshots").
		Where(squirrel.Eq{"repo_binding_id": id, "fingerprint": fingerprint})

	sqlText, args, err := stmt.ToSql()
	if err != nil {
		return spine.RepositoryContextSnapshotRecord{}, false, fmt.Errorf("get repository context snapshot by fingerprint SQL: %w", err)
	}

	record, err := scanRepositoryContextSnapshot(s.query.QueryRow(ctx, sqlText, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return spine.RepositoryContextSnapshotRecord{}, false, nil
		}
		return spine.RepositoryContextSnapshotRecord{}, false, fmt.Errorf("get repository context snapshot by fingerprint: %w", err)
	}
	return record, true, nil
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

func projectColumns() []string {
	return []string{
		"id",
		"organization_id",
		"created_by_user_id",
		"slug",
		"display_name",
		"state",
		"created_at",
		"updated_at",
	}
}

func organizationColumns() []string {
	return []string{
		"id",
		"installation_id",
		"slug",
		"display_name",
		"state",
		"created_at",
		"updated_at",
	}
}

func prefixedColumns(prefix string, columns []string) []string {
	out := make([]string, 0, len(columns))
	for _, column := range columns {
		out = append(out, prefix+"."+column)
	}
	return out
}

func repositoryContextSnapshotColumns() []string {
	return []string{
		"id",
		"organization_id",
		"project_id",
		"repo_binding_id",
		"source",
		"schema_version",
		"fingerprint",
		"snapshot",
		"created_at",
	}
}

func scanProject(row pgx.Row) (spine.Project, error) {
	var id string
	var organizationID string
	var createdByUserID string
	var state string
	var project spine.Project
	if err := row.Scan(
		&id,
		&organizationID,
		&createdByUserID,
		&project.Slug,
		&project.DisplayName,
		&state,
		&project.CreatedAt,
		&project.UpdatedAt,
	); err != nil {
		return spine.Project{}, err
	}
	project.ID = spine.ProjectID(id)
	project.OrganizationID = spine.OrganizationID(organizationID)
	project.CreatedByUserID = spine.UserID(createdByUserID)
	project.State = spine.EntityState(state)
	project.CreatedAt = project.CreatedAt.UTC()
	project.UpdatedAt = project.UpdatedAt.UTC()
	return project, nil
}

func scanOrganization(row pgx.Row) (spine.Organization, error) {
	var id string
	var installationID string
	var state string
	var organization spine.Organization
	if err := row.Scan(
		&id,
		&installationID,
		&organization.Slug,
		&organization.DisplayName,
		&state,
		&organization.CreatedAt,
		&organization.UpdatedAt,
	); err != nil {
		return spine.Organization{}, err
	}
	organization.ID = spine.OrganizationID(id)
	organization.InstallationID = spine.InstallationID(installationID)
	organization.State = spine.EntityState(state)
	organization.CreatedAt = organization.CreatedAt.UTC()
	organization.UpdatedAt = organization.UpdatedAt.UTC()
	return organization, nil
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

func scanProjectRepoBindingContext(row pgx.Row) (spine.Project, spine.RepoBinding, error) {
	var projectID string
	var projectOrganizationID string
	var projectCreatedByUserID string
	var projectState string
	var project spine.Project
	var bindingID string
	var bindingOrganizationID string
	var bindingProjectID string
	var bindingCreatedByUserID string
	var bindingAccessMode string
	var bindingState string
	var binding spine.RepoBinding

	if err := row.Scan(
		&projectID,
		&projectOrganizationID,
		&projectCreatedByUserID,
		&project.Slug,
		&project.DisplayName,
		&projectState,
		&project.CreatedAt,
		&project.UpdatedAt,
		&bindingID,
		&bindingOrganizationID,
		&bindingProjectID,
		&bindingCreatedByUserID,
		&binding.VcsConnectionID,
		&binding.Provider,
		&binding.RepositoryExternalID,
		&binding.RepositoryFullName,
		&binding.RepositoryURL,
		&binding.DefaultBranch,
		&binding.WorkflowBaseBranch,
		&binding.PathScope,
		&bindingAccessMode,
		&bindingState,
		&binding.CreatedAt,
		&binding.UpdatedAt,
	); err != nil {
		return spine.Project{}, spine.RepoBinding{}, err
	}

	project.ID = spine.ProjectID(projectID)
	project.OrganizationID = spine.OrganizationID(projectOrganizationID)
	project.CreatedByUserID = spine.UserID(projectCreatedByUserID)
	project.State = spine.EntityState(projectState)
	project.CreatedAt = project.CreatedAt.UTC()
	project.UpdatedAt = project.UpdatedAt.UTC()

	binding.ID = spine.RepoBindingID(bindingID)
	binding.OrganizationID = spine.OrganizationID(bindingOrganizationID)
	binding.ProjectID = spine.ProjectID(bindingProjectID)
	binding.CreatedByUserID = spine.UserID(bindingCreatedByUserID)
	binding.AccessMode = spine.RepoBindingAccessMode(bindingAccessMode)
	binding.State = spine.EntityState(bindingState)
	binding.CreatedAt = binding.CreatedAt.UTC()
	binding.UpdatedAt = binding.UpdatedAt.UTC()
	return project, binding, nil
}

func scanRepositoryContextSnapshot(row pgx.Row) (spine.RepositoryContextSnapshotRecord, error) {
	var record spine.RepositoryContextSnapshotRecord
	var id string
	var organizationID string
	var projectID string
	var repoBindingID string
	if err := row.Scan(
		&id,
		&organizationID,
		&projectID,
		&repoBindingID,
		&record.Source,
		&record.SchemaVersion,
		&record.Fingerprint,
		&record.Snapshot,
		&record.CreatedAt,
	); err != nil {
		return spine.RepositoryContextSnapshotRecord{}, err
	}
	record.ID = spine.RepositoryContextSnapshotID(id)
	record.OrganizationID = spine.OrganizationID(organizationID)
	record.ProjectID = spine.ProjectID(projectID)
	record.RepoBindingID = spine.RepoBindingID(repoBindingID)
	record.CreatedAt = record.CreatedAt.UTC()
	if record.Snapshot != nil {
		record.Snapshot = append([]byte(nil), record.Snapshot...)
	}
	return record, nil
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
		return uuid.Nil, spine.MalformedIDError{Field: field, Reason: "must be uuid", Err: err}
	}
	if id.Version() != 7 {
		return uuid.Nil, spine.MalformedIDError{Field: field, Reason: "must be uuidv7"}
	}
	return id, nil
}
