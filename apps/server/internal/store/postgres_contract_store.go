package store

import (
	"context"
	"fmt"
	"time"

	squirrel "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

type PostgresContractStore struct {
	exec  postgresExecer
	query postgresRowQuerier
	rows  postgresRowsQuerier
	psql  squirrel.StatementBuilderType
}

func NewPostgresContractStore(pool *pgxpool.Pool) *PostgresContractStore {
	db := newPostgresDB(pool)
	return NewPostgresContractStoreWithExecutorQuerierAndRows(db, db, db)
}

func NewPostgresContractStoreWithExecutorAndQuerier(exec postgresExecer, query postgresRowQuerier) *PostgresContractStore {
	rows, _ := query.(postgresRowsQuerier)
	return NewPostgresContractStoreWithExecutorQuerierAndRows(exec, query, rows)
}

func NewPostgresContractStoreWithExecutorQuerierAndRows(exec postgresExecer, query postgresRowQuerier, rows postgresRowsQuerier) *PostgresContractStore {
	return &PostgresContractStore{
		exec:  exec,
		query: query,
		rows:  rows,
		psql:  squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (s *PostgresContractStore) Create(ctx context.Context, contract spine.Contract) error {
	id, err := uuidValue(contract.ID, "contract id")
	if err != nil {
		return err
	}
	orgID, err := uuidValue(contract.OrganizationID, "contract organization id")
	if err != nil {
		return err
	}
	projectID, err := uuidValue(contract.ProjectID, "contract project id")
	if err != nil {
		return err
	}
	repoBindingID, err := uuidValue(contract.RepoBindingID, "contract repo binding id")
	if err != nil {
		return err
	}
	goalID, err := uuidValue(contract.GoalID, "contract goal id")
	if err != nil {
		return err
	}
	currentSeedID, err := nullableUUIDValue(contractSeedIDPointerValue(contract.CurrentSeedID), "contract current seed id")
	if err != nil {
		return err
	}
	currentDraftID, err := nullableUUIDValue(contractDraftIDPointerValue(contract.CurrentDraftID), "contract current draft id")
	if err != nil {
		return err
	}
	approvedSnapshotID, err := nullableUUIDValue(approvedContractIDPointerValue(contract.ApprovedSnapshotID), "contract approved snapshot id")
	if err != nil {
		return err
	}

	createdAt := contract.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	updatedAt := contract.UpdatedAt.UTC()
	if updatedAt.IsZero() {
		updatedAt = createdAt
	}
	stmt := s.psql.
		Insert("contracts").
		Columns(
			"id",
			"organization_id",
			"project_id",
			"repo_binding_id",
			"goal_id",
			"state",
			"current_seed_id",
			"current_draft_id",
			"approved_snapshot_id",
			"created_at",
			"updated_at",
		).
		Values(
			id,
			orgID,
			projectID,
			repoBindingID,
			goalID,
			contract.State,
			currentSeedID,
			currentDraftID,
			approvedSnapshotID,
			createdAt,
			updatedAt,
		)

	if err := execSQL(ctx, s.exec, "create contract", stmt); err != nil {
		if uniqueViolationConstraint(err) == "contracts_goal_id_unique" {
			return ErrContractAlreadySeeded
		}
		if isUniqueViolation(err) {
			return ErrContractAlreadyExists
		}
		return err
	}
	return nil
}

func (s *PostgresContractStore) Get(ctx context.Context, id spine.ContractID) (spine.Contract, bool, error) {
	contractID, err := uuidValue(id, "contract id")
	if err != nil {
		return spine.Contract{}, false, err
	}
	return s.getOne(ctx, "get contract", squirrel.Eq{"id": contractID})
}

func (s *PostgresContractStore) GetByGoalID(ctx context.Context, id spine.GoalID) (spine.Contract, bool, error) {
	goalID, err := uuidValue(id, "contract goal id")
	if err != nil {
		return spine.Contract{}, false, err
	}
	return s.getOne(ctx, "get contract by goal id", squirrel.Eq{"goal_id": goalID})
}

func (s *PostgresContractStore) List(ctx context.Context, filter spine.ContractListFilter) ([]spine.Contract, error) {
	if s.rows == nil {
		return nil, fmt.Errorf("contract rows executor is nil")
	}
	orgID, err := uuidValue(filter.OrganizationID, "contract organization id")
	if err != nil {
		return nil, err
	}

	stmt := s.psql.
		Select(contractColumns()...).
		From("contracts").
		Where(squirrel.Eq{"organization_id": orgID}).
		OrderBy("updated_at DESC", "created_at DESC", "id DESC")

	if filter.ProjectID != "" {
		projectID, err := uuidValue(filter.ProjectID, "contract project id")
		if err != nil {
			return nil, err
		}
		stmt = stmt.Where(squirrel.Eq{"project_id": projectID})
	}
	if filter.RepoBindingID != "" {
		repoBindingID, err := uuidValue(filter.RepoBindingID, "contract repo binding id")
		if err != nil {
			return nil, err
		}
		stmt = stmt.Where(squirrel.Eq{"repo_binding_id": repoBindingID})
	}
	if filter.GoalID != "" {
		goalID, err := uuidValue(filter.GoalID, "contract goal id")
		if err != nil {
			return nil, err
		}
		stmt = stmt.Where(squirrel.Eq{"goal_id": goalID})
	}
	if filter.State != "" {
		stmt = stmt.Where(squirrel.Eq{"state": filter.State})
	}
	if filter.Limit > 0 {
		stmt = stmt.Limit(uint64(filter.Limit))
	}

	sqlText, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("list contracts SQL: %w", err)
	}
	rows, err := s.rows.Query(ctx, sqlText, args...)
	if err != nil {
		return nil, fmt.Errorf("list contracts: %w", err)
	}
	defer rows.Close()

	var contracts []spine.Contract
	for rows.Next() {
		contract, err := scanContract(rows)
		if err != nil {
			return nil, fmt.Errorf("scan contract: %w", err)
		}
		contracts = append(contracts, contract)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate contracts: %w", err)
	}
	return contracts, nil
}

func (s *PostgresContractStore) MarkDraftCreated(ctx context.Context, contractID spine.ContractID, draftID spine.ContractDraftID, updatedAt time.Time) error {
	id, err := uuidValue(contractID, "contract id")
	if err != nil {
		return err
	}
	draftUUID, err := uuidValue(draftID, "contract current draft id")
	if err != nil {
		return err
	}
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}

	stmt := s.psql.
		Update("contracts").
		Set("state", spine.ContractStateDraft).
		Set("current_draft_id", draftUUID).
		Set("updated_at", updatedAt.UTC()).
		Where(squirrel.Eq{"id": id})
	return execUpdate(ctx, s.exec, "mark contract draft created", ErrContractNotFound, stmt)
}

func (s *PostgresContractStore) MarkReadyForApproval(ctx context.Context, contractID spine.ContractID, updatedAt time.Time) error {
	id, err := uuidValue(contractID, "contract id")
	if err != nil {
		return err
	}
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}

	stmt := s.psql.
		Update("contracts").
		Set("state", spine.ContractStateReadyForApproval).
		Set("updated_at", updatedAt.UTC()).
		Where(squirrel.Eq{"id": id})
	return execUpdate(ctx, s.exec, "mark contract ready for approval", ErrContractNotFound, stmt)
}

func (s *PostgresContractStore) MarkApproved(ctx context.Context, contractID spine.ContractID, approvedSnapshotID spine.ApprovedContractID, updatedAt time.Time) error {
	id, err := uuidValue(contractID, "contract id")
	if err != nil {
		return err
	}
	approvedUUID, err := uuidValue(approvedSnapshotID, "contract approved snapshot id")
	if err != nil {
		return err
	}
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}

	stmt := s.psql.
		Update("contracts").
		Set("state", spine.ContractStateApproved).
		Set("approved_snapshot_id", approvedUUID).
		Set("updated_at", updatedAt.UTC()).
		Where(squirrel.Eq{"id": id})
	return execUpdate(ctx, s.exec, "mark contract approved", ErrContractNotFound, stmt)
}

func (s *PostgresContractStore) getOne(ctx context.Context, op string, where squirrel.Eq) (spine.Contract, bool, error) {
	stmt := s.psql.
		Select(contractColumns()...).
		From("contracts").
		Where(where)

	return s.queryContract(ctx, op, stmt)
}

func (s *PostgresContractStore) queryContract(ctx context.Context, op string, sqlizer squirrel.Sqlizer) (spine.Contract, bool, error) {
	row, err := queryRow(ctx, s.query, op, sqlizer)
	if err != nil {
		return spine.Contract{}, false, err
	}
	contract, err := scanContract(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return spine.Contract{}, false, nil
		}
		return spine.Contract{}, false, fmt.Errorf("%s: %w", op, err)
	}
	return contract, true, nil
}

func scanContract(row pgx.Row) (spine.Contract, error) {
	var contract spine.Contract
	var id string
	var organizationID string
	var projectID string
	var repoBindingID string
	var goalID string
	var state string
	var currentSeedID pgtype.UUID
	var currentDraftID pgtype.UUID
	var approvedSnapshotID pgtype.UUID
	if err := row.Scan(
		&id,
		&organizationID,
		&projectID,
		&repoBindingID,
		&goalID,
		&state,
		&currentSeedID,
		&currentDraftID,
		&approvedSnapshotID,
		&contract.CreatedAt,
		&contract.UpdatedAt,
	); err != nil {
		return spine.Contract{}, err
	}
	contract.ID = spine.ContractID(id)
	contract.OrganizationID = spine.OrganizationID(organizationID)
	contract.ProjectID = spine.ProjectID(projectID)
	contract.RepoBindingID = spine.RepoBindingID(repoBindingID)
	contract.GoalID = spine.GoalID(goalID)
	contract.State = spine.ContractState(state)
	if value := uuidString(currentSeedID); value != "" {
		seedID := spine.ContractSeedID(value)
		contract.CurrentSeedID = &seedID
	}
	if value := uuidString(currentDraftID); value != "" {
		draftID := spine.ContractDraftID(value)
		contract.CurrentDraftID = &draftID
	}
	if value := uuidString(approvedSnapshotID); value != "" {
		approvedID := spine.ApprovedContractID(value)
		contract.ApprovedSnapshotID = &approvedID
	}
	contract.CreatedAt = contract.CreatedAt.UTC()
	contract.UpdatedAt = contract.UpdatedAt.UTC()
	return contract, nil
}

func contractColumns() []string {
	return []string{
		"id",
		"organization_id",
		"project_id",
		"repo_binding_id",
		"goal_id",
		"state",
		"current_seed_id",
		"current_draft_id",
		"approved_snapshot_id",
		"created_at",
		"updated_at",
	}
}

func contractSeedIDPointerValue(value *spine.ContractSeedID) any {
	if value == nil {
		return ""
	}
	return *value
}

func contractDraftIDPointerValue(value *spine.ContractDraftID) any {
	if value == nil {
		return ""
	}
	return *value
}

func approvedContractIDPointerValue(value *spine.ApprovedContractID) any {
	if value == nil {
		return ""
	}
	return *value
}
