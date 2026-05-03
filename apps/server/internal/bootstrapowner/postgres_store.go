package bootstrapowner

import (
	"context"
	"fmt"
	"time"

	squirrel "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

type Execer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type Querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type PostgresStore struct {
	exec  Execer
	query Querier
	psql  squirrel.StatementBuilderType
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return newPostgresStore(pool, pool)
}

func NewPostgresStoreWithExecutorAndQuerier(exec Execer, query Querier) *PostgresStore {
	return newPostgresStore(exec, query)
}

func newPostgresStore(exec Execer, query Querier) *PostgresStore {
	return &PostgresStore{
		exec:  exec,
		query: query,
		psql:  squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (s *PostgresStore) GetSelfHostedInstallation(ctx context.Context) (spine.Installation, bool, error) {
	stmt := s.psql.
		Select(installationColumns()...).
		From("installations").
		Where(squirrel.Eq{"mode": string(spine.InstallationModeSelfHosted)}).
		OrderBy("created_at ASC", "id ASC").
		Limit(1)
	return s.getInstallation(ctx, "get self-hosted installation", stmt)
}

func (s *PostgresStore) UpsertInstallation(ctx context.Context, installation spine.Installation) error {
	installationID, err := uuidValue(installation.ID, "installation id")
	if err != nil {
		return err
	}
	stmt := s.psql.
		Insert("installations").
		Columns("id", "mode", "public_base_url", "state", "created_at", "updated_at").
		Values(installationID, installation.Mode, installation.PublicBaseURL, installation.State, installation.CreatedAt.UTC(), installation.UpdatedAt.UTC()).
		Suffix("ON CONFLICT (id) DO UPDATE SET mode = EXCLUDED.mode, public_base_url = EXCLUDED.public_base_url, state = EXCLUDED.state, updated_at = EXCLUDED.updated_at")
	return s.execSQL(ctx, "upsert installation", stmt)
}

func (s *PostgresStore) GetPrimaryOrganization(ctx context.Context, installationID spine.InstallationID) (spine.Organization, bool, error) {
	parsedInstallationID, err := uuidValue(installationID, "organization installation id")
	if err != nil {
		return spine.Organization{}, false, err
	}
	stmt := s.psql.
		Select(organizationColumns()...).
		From("organizations").
		Where(squirrel.Eq{"installation_id": parsedInstallationID}).
		OrderBy("created_at ASC", "id ASC").
		Limit(1)
	return s.getOrganization(ctx, "get primary organization", stmt)
}

func (s *PostgresStore) UpsertOrganization(ctx context.Context, org spine.Organization) error {
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
		Values(orgID, installationID, org.Slug, org.DisplayName, org.State, org.CreatedAt.UTC(), org.UpdatedAt.UTC()).
		Suffix("ON CONFLICT (id) DO UPDATE SET installation_id = EXCLUDED.installation_id, slug = EXCLUDED.slug, display_name = EXCLUDED.display_name, state = EXCLUDED.state, updated_at = EXCLUDED.updated_at")
	return s.execSQL(ctx, "upsert organization", stmt)
}

func (s *PostgresStore) GetBootstrappedOwner(ctx context.Context, organizationID spine.OrganizationID) (spine.User, spine.UserPasswordCredential, bool, error) {
	parsedOrganizationID, err := uuidValue(organizationID, "bootstrapped owner organization id")
	if err != nil {
		return spine.User{}, spine.UserPasswordCredential{}, false, err
	}
	stmt := s.psql.
		Select(
			"u.id",
			"u.display_name",
			"u.email",
			"u.state",
			"u.created_at",
			"u.updated_at",
			"c.password_hash",
			"c.must_change_password",
			"c.password_changed_at",
			"c.created_at",
			"c.updated_at",
		).
		From("organization_memberships m").
		Join("users u ON u.id = m.user_id").
		Join("user_password_credentials c ON c.user_id = u.id").
		Where(squirrel.Eq{
			"m.organization_id": parsedOrganizationID,
			"m.role":            string(spine.OrganizationMembershipRoleOwner),
			"m.state":           string(spine.EntityStateActive),
			"u.state":           string(spine.EntityStateActive),
		}).
		OrderBy("m.created_at ASC", "m.id ASC").
		Limit(1)

	sqlText, args, err := stmt.ToSql()
	if err != nil {
		return spine.User{}, spine.UserPasswordCredential{}, false, fmt.Errorf("get bootstrapped owner SQL: %w", err)
	}

	var user spine.User
	var credential spine.UserPasswordCredential
	var userID string
	var userState string
	var passwordChangedAt pgtype.Timestamptz
	err = s.query.QueryRow(ctx, sqlText, args...).Scan(
		&userID,
		&user.DisplayName,
		&user.Email,
		&userState,
		&user.CreatedAt,
		&user.UpdatedAt,
		&credential.PasswordHash,
		&credential.MustChangePassword,
		&passwordChangedAt,
		&credential.CreatedAt,
		&credential.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return spine.User{}, spine.UserPasswordCredential{}, false, nil
		}
		return spine.User{}, spine.UserPasswordCredential{}, false, fmt.Errorf("get bootstrapped owner: %w", err)
	}

	user.ID = spine.UserID(userID)
	user.State = spine.EntityState(userState)
	user.CreatedAt = user.CreatedAt.UTC()
	user.UpdatedAt = user.UpdatedAt.UTC()

	credential.UserID = user.ID
	credential.PasswordChangedAt = timeFromPg(passwordChangedAt)
	credential.CreatedAt = credential.CreatedAt.UTC()
	credential.UpdatedAt = credential.UpdatedAt.UTC()

	return user, credential, true, nil
}

func (s *PostgresStore) GetUserByEmail(ctx context.Context, email string) (spine.User, bool, error) {
	stmt := s.psql.
		Select(userColumns()...).
		From("users").
		Where("lower(email) = lower(?)", email).
		OrderBy("created_at ASC", "id ASC").
		Limit(1)
	return s.getUser(ctx, "get user by email", stmt)
}

func (s *PostgresStore) UpsertUser(ctx context.Context, user spine.User) error {
	userID, err := uuidValue(user.ID, "user id")
	if err != nil {
		return err
	}
	stmt := s.psql.
		Insert("users").
		Columns("id", "display_name", "email", "state", "created_at", "updated_at").
		Values(userID, user.DisplayName, user.Email, user.State, user.CreatedAt.UTC(), user.UpdatedAt.UTC()).
		Suffix("ON CONFLICT (id) DO UPDATE SET display_name = EXCLUDED.display_name, email = EXCLUDED.email, state = EXCLUDED.state, updated_at = EXCLUDED.updated_at")
	return s.execSQL(ctx, "upsert user", stmt)
}

func (s *PostgresStore) GetOrganizationMembership(ctx context.Context, organizationID spine.OrganizationID, userID spine.UserID) (spine.OrganizationMembership, bool, error) {
	parsedOrganizationID, err := uuidValue(organizationID, "organization membership organization id")
	if err != nil {
		return spine.OrganizationMembership{}, false, err
	}
	parsedUserID, err := uuidValue(userID, "organization membership user id")
	if err != nil {
		return spine.OrganizationMembership{}, false, err
	}
	stmt := s.psql.
		Select(organizationMembershipColumns()...).
		From("organization_memberships").
		Where(squirrel.Eq{"organization_id": parsedOrganizationID, "user_id": parsedUserID}).
		Limit(1)
	return s.getOrganizationMembership(ctx, "get organization membership", stmt)
}

func (s *PostgresStore) UpsertOrganizationMembership(ctx context.Context, membership spine.OrganizationMembership) error {
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
		Values(membershipID, orgID, userID, membership.Role, membership.State, membership.CreatedAt.UTC(), membership.UpdatedAt.UTC()).
		Suffix("ON CONFLICT (organization_id, user_id) DO UPDATE SET role = EXCLUDED.role, state = EXCLUDED.state, updated_at = EXCLUDED.updated_at")
	return s.execSQL(ctx, "upsert organization membership", stmt)
}

func (s *PostgresStore) GetPasswordCredential(ctx context.Context, userID spine.UserID) (spine.UserPasswordCredential, bool, error) {
	parsedUserID, err := uuidValue(userID, "password credential user id")
	if err != nil {
		return spine.UserPasswordCredential{}, false, err
	}
	stmt := s.psql.
		Select(passwordCredentialColumns()...).
		From("user_password_credentials").
		Where(squirrel.Eq{"user_id": parsedUserID}).
		Limit(1)
	return s.getPasswordCredential(ctx, "get password credential", stmt)
}

func (s *PostgresStore) CreatePasswordCredential(ctx context.Context, credential spine.UserPasswordCredential) error {
	userID, err := uuidValue(credential.UserID, "password credential user id")
	if err != nil {
		return err
	}
	stmt := s.psql.
		Insert("user_password_credentials").
		Columns("user_id", "password_hash", "must_change_password", "password_changed_at", "created_at", "updated_at").
		Values(userID, credential.PasswordHash, credential.MustChangePassword, nullableTime(credential.PasswordChangedAt), credential.CreatedAt.UTC(), credential.UpdatedAt.UTC()).
		Suffix("ON CONFLICT (user_id) DO NOTHING")

	sqlText, args, err := stmt.ToSql()
	if err != nil {
		return fmt.Errorf("create password credential SQL: %w", err)
	}
	tag, err := s.exec.Exec(ctx, sqlText, args...)
	if err != nil {
		return fmt.Errorf("create password credential: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrExistingPasswordCredential
	}
	return nil
}

func (s *PostgresStore) getInstallation(ctx context.Context, op string, stmt squirrel.SelectBuilder) (spine.Installation, bool, error) {
	sqlText, args, err := stmt.ToSql()
	if err != nil {
		return spine.Installation{}, false, fmt.Errorf("%s SQL: %w", op, err)
	}
	var installation spine.Installation
	var id string
	var mode string
	var state string
	err = s.query.QueryRow(ctx, sqlText, args...).Scan(&id, &mode, &installation.PublicBaseURL, &state, &installation.CreatedAt, &installation.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return spine.Installation{}, false, nil
		}
		return spine.Installation{}, false, fmt.Errorf("%s: %w", op, err)
	}
	installation.ID = spine.InstallationID(id)
	installation.Mode = spine.InstallationMode(mode)
	installation.State = spine.EntityState(state)
	installation.CreatedAt = installation.CreatedAt.UTC()
	installation.UpdatedAt = installation.UpdatedAt.UTC()
	return installation, true, nil
}

func (s *PostgresStore) getOrganization(ctx context.Context, op string, stmt squirrel.SelectBuilder) (spine.Organization, bool, error) {
	sqlText, args, err := stmt.ToSql()
	if err != nil {
		return spine.Organization{}, false, fmt.Errorf("%s SQL: %w", op, err)
	}
	var org spine.Organization
	var id string
	var installationID string
	var state string
	err = s.query.QueryRow(ctx, sqlText, args...).Scan(&id, &installationID, &org.Slug, &org.DisplayName, &state, &org.CreatedAt, &org.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return spine.Organization{}, false, nil
		}
		return spine.Organization{}, false, fmt.Errorf("%s: %w", op, err)
	}
	org.ID = spine.OrganizationID(id)
	org.InstallationID = spine.InstallationID(installationID)
	org.State = spine.EntityState(state)
	org.CreatedAt = org.CreatedAt.UTC()
	org.UpdatedAt = org.UpdatedAt.UTC()
	return org, true, nil
}

func (s *PostgresStore) getUser(ctx context.Context, op string, stmt squirrel.SelectBuilder) (spine.User, bool, error) {
	sqlText, args, err := stmt.ToSql()
	if err != nil {
		return spine.User{}, false, fmt.Errorf("%s SQL: %w", op, err)
	}
	var user spine.User
	var id string
	var state string
	err = s.query.QueryRow(ctx, sqlText, args...).Scan(&id, &user.DisplayName, &user.Email, &state, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return spine.User{}, false, nil
		}
		return spine.User{}, false, fmt.Errorf("%s: %w", op, err)
	}
	user.ID = spine.UserID(id)
	user.State = spine.EntityState(state)
	user.CreatedAt = user.CreatedAt.UTC()
	user.UpdatedAt = user.UpdatedAt.UTC()
	return user, true, nil
}

func (s *PostgresStore) getOrganizationMembership(ctx context.Context, op string, stmt squirrel.SelectBuilder) (spine.OrganizationMembership, bool, error) {
	sqlText, args, err := stmt.ToSql()
	if err != nil {
		return spine.OrganizationMembership{}, false, fmt.Errorf("%s SQL: %w", op, err)
	}
	var membership spine.OrganizationMembership
	var id string
	var organizationID string
	var userID string
	var role string
	var state string
	err = s.query.QueryRow(ctx, sqlText, args...).Scan(&id, &organizationID, &userID, &role, &state, &membership.CreatedAt, &membership.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return spine.OrganizationMembership{}, false, nil
		}
		return spine.OrganizationMembership{}, false, fmt.Errorf("%s: %w", op, err)
	}
	membership.ID = spine.OrganizationMembershipID(id)
	membership.OrganizationID = spine.OrganizationID(organizationID)
	membership.UserID = spine.UserID(userID)
	membership.Role = spine.OrganizationMembershipRole(role)
	membership.State = spine.EntityState(state)
	membership.CreatedAt = membership.CreatedAt.UTC()
	membership.UpdatedAt = membership.UpdatedAt.UTC()
	return membership, true, nil
}

func (s *PostgresStore) getPasswordCredential(ctx context.Context, op string, stmt squirrel.SelectBuilder) (spine.UserPasswordCredential, bool, error) {
	sqlText, args, err := stmt.ToSql()
	if err != nil {
		return spine.UserPasswordCredential{}, false, fmt.Errorf("%s SQL: %w", op, err)
	}
	var credential spine.UserPasswordCredential
	var userID string
	var passwordChangedAt pgtype.Timestamptz
	err = s.query.QueryRow(ctx, sqlText, args...).Scan(&userID, &credential.PasswordHash, &credential.MustChangePassword, &passwordChangedAt, &credential.CreatedAt, &credential.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return spine.UserPasswordCredential{}, false, nil
		}
		return spine.UserPasswordCredential{}, false, fmt.Errorf("%s: %w", op, err)
	}
	credential.UserID = spine.UserID(userID)
	credential.PasswordChangedAt = timeFromPg(passwordChangedAt)
	credential.CreatedAt = credential.CreatedAt.UTC()
	credential.UpdatedAt = credential.UpdatedAt.UTC()
	return credential, true, nil
}

func (s *PostgresStore) execSQL(ctx context.Context, op string, sqlizer squirrel.Sqlizer) error {
	if s.exec == nil {
		return fmt.Errorf("%s executor is nil", op)
	}
	sqlText, args, err := sqlizer.ToSql()
	if err != nil {
		return fmt.Errorf("%s SQL: %w", op, err)
	}
	if _, err := s.exec.Exec(ctx, sqlText, args...); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func installationColumns() []string {
	return []string{"id", "mode", "public_base_url", "state", "created_at", "updated_at"}
}

func organizationColumns() []string {
	return []string{"id", "installation_id", "slug", "display_name", "state", "created_at", "updated_at"}
}

func userColumns() []string {
	return []string{"id", "display_name", "email", "state", "created_at", "updated_at"}
}

func organizationMembershipColumns() []string {
	return []string{"id", "organization_id", "user_id", "role", "state", "created_at", "updated_at"}
}

func passwordCredentialColumns() []string {
	return []string{"user_id", "password_hash", "must_change_password", "password_changed_at", "created_at", "updated_at"}
}

func timeFromPg(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	t := value.Time.UTC()
	return &t
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC()
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
