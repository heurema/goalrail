package store

import (
	"context"
	"fmt"
	"time"

	squirrel "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

type PostgresUserManagementStore struct {
	exec  postgresExecer
	query postgresRowQuerier
	rows  postgresRowsQuerier
	psql  squirrel.StatementBuilderType
}

func NewPostgresUserManagementStore(pool *pgxpool.Pool) *PostgresUserManagementStore {
	db := newPostgresDB(pool)
	return NewPostgresUserManagementStoreWithExecutorAndQuerier(db, db, db)
}

func NewPostgresUserManagementStoreWithExecutorAndQuerier(exec postgresExecer, query postgresRowQuerier, rows postgresRowsQuerier) *PostgresUserManagementStore {
	return &PostgresUserManagementStore{
		exec:  exec,
		query: query,
		rows:  rows,
		psql:  squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (s *PostgresUserManagementStore) ListOrganizationMemberships(ctx context.Context, organizationID spine.OrganizationID) ([]spine.OrganizationMembership, error) {
	orgID, err := uuidValue(organizationID, "organization id")
	if err != nil {
		return nil, err
	}
	stmt := s.psql.
		Select(organizationMembershipColumns()...).
		From("organization_memberships").
		Where(squirrel.Eq{"organization_id": orgID}).
		OrderBy("created_at ASC", "id ASC")
	sqlText, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("list organization memberships SQL: %w", err)
	}
	rows, err := s.rows.Query(ctx, sqlText, args...)
	if err != nil {
		return nil, fmt.Errorf("list organization memberships: %w", err)
	}
	defer rows.Close()

	var memberships []spine.OrganizationMembership
	for rows.Next() {
		membership, err := scanOrganizationMembership(rows)
		if err != nil {
			return nil, fmt.Errorf("scan organization membership: %w", err)
		}
		memberships = append(memberships, membership)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate organization memberships: %w", err)
	}
	return memberships, nil
}

func (s *PostgresUserManagementStore) GetUser(ctx context.Context, userID spine.UserID) (spine.User, bool, error) {
	parsedUserID, err := uuidValue(userID, "user id")
	if err != nil {
		return spine.User{}, false, err
	}
	stmt := s.psql.
		Select(userColumns()...).
		From("users").
		Where(squirrel.Eq{"id": parsedUserID})
	return s.getUser(ctx, "get user", stmt)
}

func (s *PostgresUserManagementStore) GetUserByEmail(ctx context.Context, email string) (spine.User, bool, error) {
	stmt := s.psql.
		Select(userColumns()...).
		From("users").
		Where("lower(email) = lower(?)", email).
		OrderBy("created_at ASC", "id ASC").
		Limit(1)
	return s.getUser(ctx, "get user by email", stmt)
}

func (s *PostgresUserManagementStore) CreateUser(ctx context.Context, user spine.User) (bool, error) {
	userID, err := uuidValue(user.ID, "user id")
	if err != nil {
		return false, err
	}
	createdAt := utcOrNow(user.CreatedAt)
	updatedAt := utcOrDefault(user.UpdatedAt, createdAt)
	stmt := s.psql.
		Insert("users").
		Columns("id", "display_name", "email", "state", "created_at", "updated_at").
		Values(userID, user.DisplayName, user.Email, user.State, createdAt, updatedAt)
	sqlText, args, err := stmt.ToSql()
	if err != nil {
		return false, fmt.Errorf("create user SQL: %w", err)
	}
	tag, err := s.exec.Exec(ctx, sqlText, args...)
	if err != nil {
		if uniqueViolationConstraint(err) == "users_email_lower_unique" {
			return false, nil
		}
		return false, fmt.Errorf("create user: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (s *PostgresUserManagementStore) UpsertUser(ctx context.Context, user spine.User) error {
	userID, err := uuidValue(user.ID, "user id")
	if err != nil {
		return err
	}
	createdAt := utcOrNow(user.CreatedAt)
	updatedAt := utcOrDefault(user.UpdatedAt, createdAt)
	stmt := s.psql.
		Insert("users").
		Columns("id", "display_name", "email", "state", "created_at", "updated_at").
		Values(userID, user.DisplayName, user.Email, user.State, createdAt, updatedAt).
		Suffix("ON CONFLICT (id) DO UPDATE SET display_name = EXCLUDED.display_name, email = EXCLUDED.email, state = EXCLUDED.state, updated_at = EXCLUDED.updated_at")
	return execSQL(ctx, s.exec, "upsert user", stmt)
}

func (s *PostgresUserManagementStore) GetOrganizationMembership(ctx context.Context, organizationID spine.OrganizationID, userID spine.UserID) (spine.OrganizationMembership, bool, error) {
	orgID, err := uuidValue(organizationID, "organization membership organization id")
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
		Where(squirrel.Eq{
			"organization_id": orgID,
			"user_id":         parsedUserID,
		})
	return s.getOrganizationMembership(ctx, "get organization membership", stmt)
}

func (s *PostgresUserManagementStore) CreateOrganizationMembership(ctx context.Context, membership spine.OrganizationMembership) error {
	stmt, err := s.organizationMembershipInsert(membership)
	if err != nil {
		return err
	}
	return execSQL(ctx, s.exec, "create organization membership", stmt)
}

func (s *PostgresUserManagementStore) UpsertOrganizationMembership(ctx context.Context, membership spine.OrganizationMembership) error {
	stmt, err := s.organizationMembershipInsert(membership)
	if err != nil {
		return err
	}
	stmt = stmt.Suffix("ON CONFLICT (organization_id, user_id) DO UPDATE SET role = EXCLUDED.role, state = EXCLUDED.state, updated_at = EXCLUDED.updated_at")
	return execSQL(ctx, s.exec, "upsert organization membership", stmt)
}

func (s *PostgresUserManagementStore) organizationMembershipInsert(membership spine.OrganizationMembership) (squirrel.InsertBuilder, error) {
	membershipID, err := uuidValue(membership.ID, "organization membership id")
	if err != nil {
		return squirrel.InsertBuilder{}, err
	}
	orgID, err := uuidValue(membership.OrganizationID, "organization membership organization id")
	if err != nil {
		return squirrel.InsertBuilder{}, err
	}
	userID, err := uuidValue(membership.UserID, "organization membership user id")
	if err != nil {
		return squirrel.InsertBuilder{}, err
	}
	createdAt := utcOrNow(membership.CreatedAt)
	updatedAt := utcOrDefault(membership.UpdatedAt, createdAt)
	stmt := s.psql.
		Insert("organization_memberships").
		Columns("id", "organization_id", "user_id", "role", "state", "created_at", "updated_at").
		Values(membershipID, orgID, userID, membership.Role, membership.State, createdAt, updatedAt)
	return stmt, nil
}

func (s *PostgresUserManagementStore) GetPasswordCredential(ctx context.Context, userID spine.UserID) (spine.UserPasswordCredential, bool, error) {
	parsedUserID, err := uuidValue(userID, "password credential user id")
	if err != nil {
		return spine.UserPasswordCredential{}, false, err
	}
	stmt := s.psql.
		Select(passwordCredentialColumns()...).
		From("user_password_credentials").
		Where(squirrel.Eq{"user_id": parsedUserID})
	return s.getPasswordCredential(ctx, "get password credential", stmt)
}

func (s *PostgresUserManagementStore) UpsertPasswordCredential(ctx context.Context, credential spine.UserPasswordCredential) error {
	userID, err := uuidValue(credential.UserID, "password credential user id")
	if err != nil {
		return err
	}
	createdAt := utcOrNow(credential.CreatedAt)
	updatedAt := utcOrDefault(credential.UpdatedAt, createdAt)
	stmt := s.psql.
		Insert("user_password_credentials").
		Columns("user_id", "password_hash", "must_change_password", "password_changed_at", "created_at", "updated_at").
		Values(userID, credential.PasswordHash, credential.MustChangePassword, nullableTime(credential.PasswordChangedAt), createdAt, updatedAt).
		Suffix("ON CONFLICT (user_id) DO UPDATE SET password_hash = EXCLUDED.password_hash, must_change_password = EXCLUDED.must_change_password, password_changed_at = EXCLUDED.password_changed_at, updated_at = EXCLUDED.updated_at")
	return execSQL(ctx, s.exec, "upsert password credential", stmt)
}

func (s *PostgresUserManagementStore) LockActiveOwnerMemberships(ctx context.Context, organizationID spine.OrganizationID) error {
	orgID, err := uuidValue(organizationID, "organization id")
	if err != nil {
		return err
	}
	stmt := s.psql.
		Select("id::text").
		From("organization_memberships").
		Where(squirrel.Eq{
			"organization_id": orgID,
			"role":            string(spine.OrganizationMembershipRoleOwner),
			"state":           string(spine.EntityStateActive),
		}).
		OrderBy("id ASC").
		Suffix("FOR UPDATE")
	sqlText, args, err := stmt.ToSql()
	if err != nil {
		return fmt.Errorf("lock active owner memberships SQL: %w", err)
	}
	rows, err := s.rows.Query(ctx, sqlText, args...)
	if err != nil {
		return fmt.Errorf("lock active owner memberships: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("scan active owner membership lock: %w", err)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate active owner membership locks: %w", err)
	}
	return nil
}

func (s *PostgresUserManagementStore) CountActiveOwners(ctx context.Context, organizationID spine.OrganizationID) (int, error) {
	orgID, err := uuidValue(organizationID, "organization id")
	if err != nil {
		return 0, err
	}
	stmt := s.psql.
		Select("count(*)").
		From("organization_memberships m").
		Join("users u ON u.id = m.user_id").
		Where(squirrel.Eq{
			"m.organization_id": orgID,
			"m.role":            string(spine.OrganizationMembershipRoleOwner),
			"m.state":           string(spine.EntityStateActive),
			"u.state":           string(spine.EntityStateActive),
		})
	row, err := queryRow(ctx, s.query, "count active owners", stmt)
	if err != nil {
		return 0, err
	}
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("count active owners: %w", err)
	}
	return count, nil
}

func (s *PostgresUserManagementStore) RevokeActiveSessionsForUser(ctx context.Context, userID spine.UserID, revokedAt time.Time) error {
	parsedUserID, err := uuidValue(userID, "user session user id")
	if err != nil {
		return err
	}
	stmt := s.psql.
		Update("user_sessions").
		Set("state", string(spine.UserSessionStateRevoked)).
		Set("revoked_at", revokedAt.UTC()).
		Set("updated_at", revokedAt.UTC()).
		Where(squirrel.Eq{
			"user_id": parsedUserID,
			"state":   string(spine.UserSessionStateActive),
		})
	return execSQL(ctx, s.exec, "revoke active user sessions", stmt)
}

func (s *PostgresUserManagementStore) getUser(ctx context.Context, op string, stmt squirrel.SelectBuilder) (spine.User, bool, error) {
	row, err := queryRow(ctx, s.query, op, stmt)
	if err != nil {
		return spine.User{}, false, err
	}
	user, err := scanUser(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return spine.User{}, false, nil
		}
		return spine.User{}, false, fmt.Errorf("%s: %w", op, err)
	}
	return user, true, nil
}

func (s *PostgresUserManagementStore) getOrganizationMembership(ctx context.Context, op string, stmt squirrel.SelectBuilder) (spine.OrganizationMembership, bool, error) {
	row, err := queryRow(ctx, s.query, op, stmt)
	if err != nil {
		return spine.OrganizationMembership{}, false, err
	}
	membership, err := scanOrganizationMembership(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return spine.OrganizationMembership{}, false, nil
		}
		return spine.OrganizationMembership{}, false, fmt.Errorf("%s: %w", op, err)
	}
	return membership, true, nil
}

func (s *PostgresUserManagementStore) getPasswordCredential(ctx context.Context, op string, stmt squirrel.SelectBuilder) (spine.UserPasswordCredential, bool, error) {
	row, err := queryRow(ctx, s.query, op, stmt)
	if err != nil {
		return spine.UserPasswordCredential{}, false, err
	}
	credential, err := scanPasswordCredential(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return spine.UserPasswordCredential{}, false, nil
		}
		return spine.UserPasswordCredential{}, false, fmt.Errorf("%s: %w", op, err)
	}
	return credential, true, nil
}
