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
	psql  squirrel.StatementBuilderType
}

func NewPostgresContractStore(pool *pgxpool.Pool) *PostgresContractStore {
	db := newPostgresDB(pool)
	return NewPostgresContractStoreWithExecutorAndQuerier(db, db)
}

func NewPostgresContractStoreWithExecutorAndQuerier(exec postgresExecer, query postgresRowQuerier) *PostgresContractStore {
	return &PostgresContractStore{
		exec:  exec,
		query: query,
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
	row, err := queryContractLifecycleRow(ctx, s.query, op, sqlizer)
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

func queryContractLifecycleRow(ctx context.Context, query postgresRowQuerier, op string, sqlizer squirrel.Sqlizer) (pgx.Row, error) {
	if query == nil {
		return nil, fmt.Errorf("%s query executor is nil", op)
	}
	sqlText, args, err := sqlizer.ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s SQL: %w", op, err)
	}
	return query.QueryRow(ctx, sqlText, args...), nil
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
