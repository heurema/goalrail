package store

import (
	"context"
	"fmt"
	"time"

	squirrel "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

type AuthExecer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type AuthQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type PostgresAuthStore struct {
	exec  AuthExecer
	query AuthQuerier
	psql  squirrel.StatementBuilderType
}

func NewPostgresAuthStore(pool *pgxpool.Pool) *PostgresAuthStore {
	db := newPostgresDB(pool)
	return NewPostgresAuthStoreWithExecutorAndQuerier(db, db)
}

func NewPostgresAuthStoreWithExecutorAndQuerier(exec AuthExecer, query AuthQuerier) *PostgresAuthStore {
	return &PostgresAuthStore{
		exec:  exec,
		query: query,
		psql:  squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (s *PostgresAuthStore) UpsertPasswordCredential(ctx context.Context, credential spine.UserPasswordCredential) error {
	userID, err := uuidValue(credential.UserID, "password credential user id")
	if err != nil {
		return err
	}
	passwordChangedAt := nullableTime(credential.PasswordChangedAt)
	createdAt := utcOrNow(credential.CreatedAt)
	updatedAt := utcOrDefault(credential.UpdatedAt, createdAt)

	stmt := s.psql.
		Insert("user_password_credentials").
		Columns(
			"user_id",
			"password_hash",
			"must_change_password",
			"password_changed_at",
			"created_at",
			"updated_at",
		).
		Values(
			userID,
			credential.PasswordHash,
			credential.MustChangePassword,
			passwordChangedAt,
			createdAt,
			updatedAt,
		).
		Suffix("ON CONFLICT (user_id) DO UPDATE SET password_hash = EXCLUDED.password_hash, must_change_password = EXCLUDED.must_change_password, password_changed_at = EXCLUDED.password_changed_at, updated_at = EXCLUDED.updated_at")
	return s.execSQL(ctx, "upsert password credential", stmt)
}

func (s *PostgresAuthStore) UpsertSession(ctx context.Context, session spine.UserSession) error {
	sessionID, err := uuidValue(session.ID, "user session id")
	if err != nil {
		return err
	}
	userID, err := uuidValue(session.UserID, "user session user id")
	if err != nil {
		return err
	}
	createdAt := utcOrNow(session.CreatedAt)
	updatedAt := utcOrDefault(session.UpdatedAt, createdAt)

	stmt := s.psql.
		Insert("user_sessions").
		Columns(
			"id",
			"user_id",
			"refresh_token_hash",
			"state",
			"created_at",
			"updated_at",
			"expires_at",
			"revoked_at",
			"last_used_at",
		).
		Values(
			sessionID,
			userID,
			session.RefreshTokenHash,
			session.State,
			createdAt,
			updatedAt,
			session.ExpiresAt.UTC(),
			nullableTime(session.RevokedAt),
			nullableTime(session.LastUsedAt),
		).
		Suffix("ON CONFLICT (id) DO UPDATE SET refresh_token_hash = EXCLUDED.refresh_token_hash, state = EXCLUDED.state, updated_at = EXCLUDED.updated_at, expires_at = EXCLUDED.expires_at, revoked_at = EXCLUDED.revoked_at, last_used_at = EXCLUDED.last_used_at")
	return s.execSQL(ctx, "upsert user session", stmt)
}

func (s *PostgresAuthStore) GetPasswordCredential(ctx context.Context, userID spine.UserID) (spine.UserPasswordCredential, bool, error) {
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

func (s *PostgresAuthStore) GetSession(ctx context.Context, sessionID spine.UserSessionID) (spine.UserSession, bool, error) {
	parsedSessionID, err := uuidValue(sessionID, "user session id")
	if err != nil {
		return spine.UserSession{}, false, err
	}
	stmt := s.psql.
		Select(userSessionColumns()...).
		From("user_sessions").
		Where(squirrel.Eq{"id": parsedSessionID})
	return s.getSession(ctx, "get user session", stmt)
}

func (s *PostgresAuthStore) GetSessionByRefreshTokenHash(ctx context.Context, refreshTokenHash string) (spine.UserSession, bool, error) {
	stmt := s.psql.
		Select(userSessionColumns()...).
		From("user_sessions").
		Where(squirrel.Eq{"refresh_token_hash": refreshTokenHash})
	return s.getSession(ctx, "get user session by refresh token hash", stmt)
}

func (s *PostgresAuthStore) GetUser(ctx context.Context, userID spine.UserID) (spine.User, bool, error) {
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

func (s *PostgresAuthStore) GetUserByEmail(ctx context.Context, email string) (spine.User, bool, error) {
	stmt := s.psql.
		Select(userColumns()...).
		From("users").
		Where("lower(email) = lower(?)", email).
		OrderBy("created_at ASC", "id ASC").
		Limit(1)
	return s.getUser(ctx, "get user by email", stmt)
}

func (s *PostgresAuthStore) GetPrimaryOrganizationMembership(ctx context.Context, userID spine.UserID) (spine.OrganizationMembership, bool, error) {
	parsedUserID, err := uuidValue(userID, "organization membership user id")
	if err != nil {
		return spine.OrganizationMembership{}, false, err
	}
	stmt := s.psql.
		Select(organizationMembershipColumns()...).
		From("organization_memberships").
		Where(squirrel.Eq{
			"user_id": parsedUserID,
			"state":   string(spine.EntityStateActive),
		}).
		OrderBy("created_at ASC", "id ASC").
		Limit(1)
	return s.getOrganizationMembership(ctx, "get primary organization membership", stmt)
}

func (s *PostgresAuthStore) getPasswordCredential(ctx context.Context, op string, stmt squirrel.SelectBuilder) (spine.UserPasswordCredential, bool, error) {
	if s.query == nil {
		return spine.UserPasswordCredential{}, false, fmt.Errorf("%s query executor is nil", op)
	}
	sqlText, args, err := stmt.ToSql()
	if err != nil {
		return spine.UserPasswordCredential{}, false, fmt.Errorf("%s SQL: %w", op, err)
	}
	credential, err := scanPasswordCredential(s.query.QueryRow(ctx, sqlText, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return spine.UserPasswordCredential{}, false, nil
		}
		return spine.UserPasswordCredential{}, false, fmt.Errorf("%s: %w", op, err)
	}
	return credential, true, nil
}

func (s *PostgresAuthStore) getSession(ctx context.Context, op string, stmt squirrel.SelectBuilder) (spine.UserSession, bool, error) {
	if s.query == nil {
		return spine.UserSession{}, false, fmt.Errorf("%s query executor is nil", op)
	}
	sqlText, args, err := stmt.ToSql()
	if err != nil {
		return spine.UserSession{}, false, fmt.Errorf("%s SQL: %w", op, err)
	}
	session, err := scanUserSession(s.query.QueryRow(ctx, sqlText, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return spine.UserSession{}, false, nil
		}
		return spine.UserSession{}, false, fmt.Errorf("%s: %w", op, err)
	}
	return session, true, nil
}

func (s *PostgresAuthStore) getUser(ctx context.Context, op string, stmt squirrel.SelectBuilder) (spine.User, bool, error) {
	if s.query == nil {
		return spine.User{}, false, fmt.Errorf("%s query executor is nil", op)
	}
	sqlText, args, err := stmt.ToSql()
	if err != nil {
		return spine.User{}, false, fmt.Errorf("%s SQL: %w", op, err)
	}
	user, err := scanUser(s.query.QueryRow(ctx, sqlText, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return spine.User{}, false, nil
		}
		return spine.User{}, false, fmt.Errorf("%s: %w", op, err)
	}
	return user, true, nil
}

func (s *PostgresAuthStore) getOrganizationMembership(ctx context.Context, op string, stmt squirrel.SelectBuilder) (spine.OrganizationMembership, bool, error) {
	if s.query == nil {
		return spine.OrganizationMembership{}, false, fmt.Errorf("%s query executor is nil", op)
	}
	sqlText, args, err := stmt.ToSql()
	if err != nil {
		return spine.OrganizationMembership{}, false, fmt.Errorf("%s SQL: %w", op, err)
	}
	membership, err := scanOrganizationMembership(s.query.QueryRow(ctx, sqlText, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return spine.OrganizationMembership{}, false, nil
		}
		return spine.OrganizationMembership{}, false, fmt.Errorf("%s: %w", op, err)
	}
	return membership, true, nil
}

func scanPasswordCredential(row pgx.Row) (spine.UserPasswordCredential, error) {
	var credential spine.UserPasswordCredential
	var userID string
	var passwordChangedAt pgtype.Timestamptz
	if err := row.Scan(
		&userID,
		&credential.PasswordHash,
		&credential.MustChangePassword,
		&passwordChangedAt,
		&credential.CreatedAt,
		&credential.UpdatedAt,
	); err != nil {
		return spine.UserPasswordCredential{}, err
	}
	credential.UserID = spine.UserID(userID)
	credential.PasswordChangedAt = timeFromPg(passwordChangedAt)
	credential.CreatedAt = credential.CreatedAt.UTC()
	credential.UpdatedAt = credential.UpdatedAt.UTC()
	return credential, nil
}

func scanUserSession(row pgx.Row) (spine.UserSession, error) {
	var session spine.UserSession
	var sessionID string
	var userID string
	var state string
	var revokedAt pgtype.Timestamptz
	var lastUsedAt pgtype.Timestamptz
	if err := row.Scan(
		&sessionID,
		&userID,
		&session.RefreshTokenHash,
		&state,
		&session.CreatedAt,
		&session.UpdatedAt,
		&session.ExpiresAt,
		&revokedAt,
		&lastUsedAt,
	); err != nil {
		return spine.UserSession{}, err
	}
	session.ID = spine.UserSessionID(sessionID)
	session.UserID = spine.UserID(userID)
	session.State = spine.UserSessionState(state)
	session.RevokedAt = timeFromPg(revokedAt)
	session.LastUsedAt = timeFromPg(lastUsedAt)
	session.CreatedAt = session.CreatedAt.UTC()
	session.UpdatedAt = session.UpdatedAt.UTC()
	session.ExpiresAt = session.ExpiresAt.UTC()
	return session, nil
}

func scanUser(row pgx.Row) (spine.User, error) {
	var user spine.User
	var id string
	var state string
	if err := row.Scan(
		&id,
		&user.DisplayName,
		&user.Email,
		&state,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		return spine.User{}, err
	}
	user.ID = spine.UserID(id)
	user.State = spine.EntityState(state)
	user.CreatedAt = user.CreatedAt.UTC()
	user.UpdatedAt = user.UpdatedAt.UTC()
	return user, nil
}

func scanOrganizationMembership(row pgx.Row) (spine.OrganizationMembership, error) {
	var membership spine.OrganizationMembership
	var id string
	var organizationID string
	var userID string
	var role string
	var state string
	if err := row.Scan(
		&id,
		&organizationID,
		&userID,
		&role,
		&state,
		&membership.CreatedAt,
		&membership.UpdatedAt,
	); err != nil {
		return spine.OrganizationMembership{}, err
	}
	membership.ID = spine.OrganizationMembershipID(id)
	membership.OrganizationID = spine.OrganizationID(organizationID)
	membership.UserID = spine.UserID(userID)
	membership.Role = spine.OrganizationMembershipRole(role)
	membership.State = spine.EntityState(state)
	membership.CreatedAt = membership.CreatedAt.UTC()
	membership.UpdatedAt = membership.UpdatedAt.UTC()
	return membership, nil
}

func passwordCredentialColumns() []string {
	return []string{
		"user_id",
		"password_hash",
		"must_change_password",
		"password_changed_at",
		"created_at",
		"updated_at",
	}
}

func userSessionColumns() []string {
	return []string{
		"id",
		"user_id",
		"refresh_token_hash",
		"state",
		"created_at",
		"updated_at",
		"expires_at",
		"revoked_at",
		"last_used_at",
	}
}

func userColumns() []string {
	return []string{
		"id",
		"display_name",
		"email",
		"state",
		"created_at",
		"updated_at",
	}
}

func organizationMembershipColumns() []string {
	return []string{
		"id",
		"organization_id",
		"user_id",
		"role",
		"state",
		"created_at",
		"updated_at",
	}
}

func (s *PostgresAuthStore) execSQL(ctx context.Context, op string, sqlizer squirrel.Sqlizer) error {
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

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC()
}

func timeFromPg(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	t := value.Time.UTC()
	return &t
}

func utcOrNow(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}
	return value.UTC()
}

func utcOrDefault(value time.Time, fallback time.Time) time.Time {
	if value.IsZero() {
		return fallback.UTC()
	}
	return value.UTC()
}
