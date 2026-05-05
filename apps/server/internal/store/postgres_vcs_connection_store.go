package store

import (
	"context"
	"fmt"

	squirrel "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

type PostgresVcsConnectionStore struct {
	exec  postgresExecer
	query postgresRowQuerier
	psql  squirrel.StatementBuilderType
}

func NewPostgresVcsConnectionStore(pool *pgxpool.Pool) *PostgresVcsConnectionStore {
	db := newPostgresDB(pool)
	return NewPostgresVcsConnectionStoreWithExecutorAndQuerier(db, db)
}

func NewPostgresVcsConnectionStoreWithExecutorAndQuerier(exec postgresExecer, query postgresRowQuerier) *PostgresVcsConnectionStore {
	return &PostgresVcsConnectionStore{
		exec:  exec,
		query: query,
		psql:  squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (s *PostgresVcsConnectionStore) GetOrganization(ctx context.Context, organizationID spine.OrganizationID) (spine.Organization, bool, error) {
	id, err := uuidValue(organizationID, "organization id")
	if err != nil {
		return spine.Organization{}, false, err
	}
	stmt := s.psql.
		Select(organizationColumns()...).
		From("organizations").
		Where(squirrel.Eq{"id": id})

	row, err := queryRow(ctx, s.query, "get organization", stmt)
	if err != nil {
		return spine.Organization{}, false, err
	}
	organization, err := scanOrganization(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return spine.Organization{}, false, nil
		}
		return spine.Organization{}, false, fmt.Errorf("get organization: %w", err)
	}
	return organization, true, nil
}

func (s *PostgresVcsConnectionStore) CreatePendingSetup(ctx context.Context, connection spine.VcsConnection) error {
	connectionID, err := uuidValue(connection.ID, "VCS connection id")
	if err != nil {
		return err
	}
	installationID, err := uuidValue(connection.InstallationID, "VCS connection installation id")
	if err != nil {
		return err
	}
	organizationID, err := uuidValue(connection.OrganizationID, "VCS connection organization id")
	if err != nil {
		return err
	}
	createdByUserID, err := uuidValue(connection.CreatedByUserID, "VCS connection created by user id")
	if err != nil {
		return err
	}
	stmt := s.psql.
		Insert("vcs_connections").
		Columns(vcsConnectionColumns()...).
		Values(
			connectionID,
			installationID,
			organizationID,
			createdByUserID,
			connection.ProviderKind,
			connection.ProviderInstanceURL,
			connection.State,
			connection.SetupExpiresAt.UTC(),
			connection.CreatedAt.UTC(),
			connection.UpdatedAt.UTC(),
		)
	if err := execSQL(ctx, s.exec, "create pending setup VCS connection", stmt); err != nil {
		return wrapUniqueConstraint(err)
	}
	return nil
}

func (s *PostgresVcsConnectionStore) GetVcsConnection(ctx context.Context, connectionID spine.VcsConnectionID) (spine.VcsConnection, bool, error) {
	id, err := uuidValue(connectionID, "VCS connection id")
	if err != nil {
		return spine.VcsConnection{}, false, err
	}
	stmt := s.psql.
		Select(vcsConnectionColumns()...).
		From("vcs_connections").
		Where(squirrel.Eq{"id": id})

	row, err := queryRow(ctx, s.query, "get VCS connection", stmt)
	if err != nil {
		return spine.VcsConnection{}, false, err
	}
	connection, err := scanVcsConnection(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return spine.VcsConnection{}, false, nil
		}
		return spine.VcsConnection{}, false, fmt.Errorf("get VCS connection: %w", err)
	}
	return connection, true, nil
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

func vcsConnectionColumns() []string {
	return []string{
		"id",
		"installation_id",
		"organization_id",
		"created_by_user_id",
		"provider_kind",
		"provider_instance_url",
		"state",
		"setup_expires_at",
		"created_at",
		"updated_at",
	}
}

func scanOrganization(row pgx.Row) (spine.Organization, error) {
	var organization spine.Organization
	var id string
	var installationID string
	var state string
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

func scanVcsConnection(row pgx.Row) (spine.VcsConnection, error) {
	var connection spine.VcsConnection
	var id string
	var installationID string
	var organizationID string
	var createdByUserID string
	var state string
	if err := row.Scan(
		&id,
		&installationID,
		&organizationID,
		&createdByUserID,
		&connection.ProviderKind,
		&connection.ProviderInstanceURL,
		&state,
		&connection.SetupExpiresAt,
		&connection.CreatedAt,
		&connection.UpdatedAt,
	); err != nil {
		return spine.VcsConnection{}, err
	}
	connection.ID = spine.VcsConnectionID(id)
	connection.InstallationID = spine.InstallationID(installationID)
	connection.OrganizationID = spine.OrganizationID(organizationID)
	connection.CreatedByUserID = spine.UserID(createdByUserID)
	connection.State = spine.VcsConnectionState(state)
	connection.SetupExpiresAt = connection.SetupExpiresAt.UTC()
	connection.CreatedAt = connection.CreatedAt.UTC()
	connection.UpdatedAt = connection.UpdatedAt.UTC()
	return connection, nil
}
