package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	squirrel "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

type ApprovedContractExecer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type ApprovedContractQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type PostgresApprovedContractStore struct {
	exec  ApprovedContractExecer
	query ApprovedContractQuerier
	psql  squirrel.StatementBuilderType
}

func NewPostgresApprovedContractStore(pool *pgxpool.Pool) *PostgresApprovedContractStore {
	db := newPostgresDB(pool)
	return NewPostgresApprovedContractStoreWithExecutorAndQuerier(db, db)
}

func NewPostgresApprovedContractStoreWithExecutorAndQuerier(exec ApprovedContractExecer, query ApprovedContractQuerier) *PostgresApprovedContractStore {
	return &PostgresApprovedContractStore{
		exec:  exec,
		query: query,
		psql:  squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (s *PostgresApprovedContractStore) Create(ctx context.Context, approved spine.ApprovedContract) error {
	id, err := uuidValue(approved.ID, "approved contract id")
	if err != nil {
		return err
	}
	orgID, err := uuidValue(approved.OrganizationID, "approved contract organization id")
	if err != nil {
		return err
	}
	projectID, err := uuidValue(approved.ProjectID, "approved contract project id")
	if err != nil {
		return err
	}
	repoBindingID, err := uuidValue(approved.RepoBindingID, "approved contract repo binding id")
	if err != nil {
		return err
	}
	draftID, err := uuidValue(approved.ContractDraftID, "approved contract contract draft id")
	if err != nil {
		return err
	}
	seedID, err := uuidValue(approved.ContractSeedID, "approved contract contract seed id")
	if err != nil {
		return err
	}
	goalID, err := uuidValue(approved.GoalID, "approved contract goal id")
	if err != nil {
		return err
	}

	scope, err := json.Marshal(nonNilStrings(approved.Scope))
	if err != nil {
		return fmt.Errorf("marshal approved contract scope: %w", err)
	}
	nonGoals, err := json.Marshal(nonNilStrings(approved.NonGoals))
	if err != nil {
		return fmt.Errorf("marshal approved contract non-goals: %w", err)
	}
	constraints, err := json.Marshal(nonNilStrings(approved.Constraints))
	if err != nil {
		return fmt.Errorf("marshal approved contract constraints: %w", err)
	}
	acceptanceCriteria, err := json.Marshal(nonNilStrings(approved.AcceptanceCriteria))
	if err != nil {
		return fmt.Errorf("marshal approved contract acceptance criteria: %w", err)
	}
	expectedChecks, err := json.Marshal(nonNilStrings(approved.ExpectedChecks))
	if err != nil {
		return fmt.Errorf("marshal approved contract expected checks: %w", err)
	}
	proofExpectations, err := json.Marshal(nonNilStrings(approved.ProofExpectations))
	if err != nil {
		return fmt.Errorf("marshal approved contract proof expectations: %w", err)
	}
	riskHints, err := json.Marshal(nonNilStrings(approved.RiskHints))
	if err != nil {
		return fmt.Errorf("marshal approved contract risk hints: %w", err)
	}
	approvedBy, err := json.Marshal(approved.ApprovedBy)
	if err != nil {
		return fmt.Errorf("marshal approved contract approved by: %w", err)
	}
	sourceRefsValue := approved.SourceRefs
	if sourceRefsValue == nil {
		sourceRefsValue = []spine.SourceRef{}
	}
	sourceRefs, err := json.Marshal(sourceRefsValue)
	if err != nil {
		return fmt.Errorf("marshal approved contract source refs: %w", err)
	}

	approvedAt := approved.ApprovedAt.UTC()
	if approvedAt.IsZero() {
		approvedAt = time.Now().UTC()
	}
	stmt := s.psql.
		Insert("approved_contracts").
		Columns(
			"id",
			"organization_id",
			"project_id",
			"repo_binding_id",
			"contract_draft_id",
			"contract_seed_id",
			"goal_id",
			"title",
			"intent_summary",
			"scope",
			"non_goals",
			"constraints",
			"acceptance_criteria",
			"expected_checks",
			"proof_expectations",
			"risk_hints",
			"approved_by",
			"approved_at",
			"source_refs",
			"state",
			"created_at",
			"updated_at",
		).
		Values(
			id,
			orgID,
			projectID,
			repoBindingID,
			draftID,
			seedID,
			goalID,
			approved.Title,
			approved.IntentSummary,
			scope,
			nonGoals,
			constraints,
			acceptanceCriteria,
			expectedChecks,
			proofExpectations,
			riskHints,
			approvedBy,
			approvedAt,
			sourceRefs,
			approved.State,
			approvedAt,
			approvedAt,
		)

	if err := s.execSQL(ctx, "create approved contract", stmt); err != nil {
		if uniqueViolationConstraint(err) == "approved_contracts_contract_draft_id_unique" {
			return ErrApprovedContractAlreadyApproved
		}
		if isUniqueViolation(err) {
			return ErrApprovedContractAlreadyExists
		}
		return err
	}
	return nil
}

func (s *PostgresApprovedContractStore) Get(ctx context.Context, id spine.ApprovedContractID) (spine.ApprovedContract, bool, error) {
	approvedID, err := uuidValue(id, "approved contract id")
	if err != nil {
		return spine.ApprovedContract{}, false, err
	}
	return s.getOne(ctx, "get approved contract", squirrel.Eq{"id": approvedID})
}

func (s *PostgresApprovedContractStore) GetByContractDraftID(ctx context.Context, id spine.ContractDraftID) (spine.ApprovedContract, bool, error) {
	draftID, err := uuidValue(id, "approved contract contract draft id")
	if err != nil {
		return spine.ApprovedContract{}, false, err
	}
	return s.getOne(ctx, "get approved contract by contract draft id", squirrel.Eq{"contract_draft_id": draftID})
}

func (s *PostgresApprovedContractStore) getOne(ctx context.Context, op string, where squirrel.Eq) (spine.ApprovedContract, bool, error) {
	stmt := s.psql.
		Select(approvedContractColumns()...).
		From("approved_contracts").
		Where(where)

	return s.queryApprovedContract(ctx, op, stmt)
}

func (s *PostgresApprovedContractStore) queryApprovedContract(ctx context.Context, op string, sqlizer squirrel.Sqlizer) (spine.ApprovedContract, bool, error) {
	if s.query == nil {
		return spine.ApprovedContract{}, false, fmt.Errorf("%s query executor is nil", op)
	}
	sqlText, args, err := sqlizer.ToSql()
	if err != nil {
		return spine.ApprovedContract{}, false, fmt.Errorf("%s SQL: %w", op, err)
	}
	approved, err := scanApprovedContract(s.query.QueryRow(ctx, sqlText, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return spine.ApprovedContract{}, false, nil
		}
		return spine.ApprovedContract{}, false, fmt.Errorf("%s: %w", op, err)
	}
	return approved, true, nil
}

func scanApprovedContract(row pgx.Row) (spine.ApprovedContract, error) {
	var approved spine.ApprovedContract
	var id string
	var organizationID string
	var projectID string
	var repoBindingID string
	var draftID string
	var seedID string
	var goalID string
	var scope []byte
	var nonGoals []byte
	var constraints []byte
	var acceptanceCriteria []byte
	var expectedChecks []byte
	var proofExpectations []byte
	var riskHints []byte
	var approvedBy []byte
	var sourceRefs []byte
	var state string
	if err := row.Scan(
		&id,
		&organizationID,
		&projectID,
		&repoBindingID,
		&draftID,
		&seedID,
		&goalID,
		&approved.Title,
		&approved.IntentSummary,
		&scope,
		&nonGoals,
		&constraints,
		&acceptanceCriteria,
		&expectedChecks,
		&proofExpectations,
		&riskHints,
		&approvedBy,
		&approved.ApprovedAt,
		&sourceRefs,
		&state,
	); err != nil {
		return spine.ApprovedContract{}, err
	}
	approved.ID = spine.ApprovedContractID(id)
	approved.OrganizationID = spine.OrganizationID(organizationID)
	approved.ProjectID = spine.ProjectID(projectID)
	approved.RepoBindingID = spine.RepoBindingID(repoBindingID)
	approved.ContractDraftID = spine.ContractDraftID(draftID)
	approved.ContractSeedID = spine.ContractSeedID(seedID)
	approved.GoalID = spine.GoalID(goalID)
	approved.State = spine.ApprovedContractState(state)
	if err := json.Unmarshal(scope, &approved.Scope); err != nil {
		return spine.ApprovedContract{}, fmt.Errorf("unmarshal approved contract scope: %w", err)
	}
	if err := json.Unmarshal(nonGoals, &approved.NonGoals); err != nil {
		return spine.ApprovedContract{}, fmt.Errorf("unmarshal approved contract non-goals: %w", err)
	}
	if err := json.Unmarshal(constraints, &approved.Constraints); err != nil {
		return spine.ApprovedContract{}, fmt.Errorf("unmarshal approved contract constraints: %w", err)
	}
	if err := json.Unmarshal(acceptanceCriteria, &approved.AcceptanceCriteria); err != nil {
		return spine.ApprovedContract{}, fmt.Errorf("unmarshal approved contract acceptance criteria: %w", err)
	}
	if err := json.Unmarshal(expectedChecks, &approved.ExpectedChecks); err != nil {
		return spine.ApprovedContract{}, fmt.Errorf("unmarshal approved contract expected checks: %w", err)
	}
	if err := json.Unmarshal(proofExpectations, &approved.ProofExpectations); err != nil {
		return spine.ApprovedContract{}, fmt.Errorf("unmarshal approved contract proof expectations: %w", err)
	}
	if err := json.Unmarshal(riskHints, &approved.RiskHints); err != nil {
		return spine.ApprovedContract{}, fmt.Errorf("unmarshal approved contract risk hints: %w", err)
	}
	if err := json.Unmarshal(approvedBy, &approved.ApprovedBy); err != nil {
		return spine.ApprovedContract{}, fmt.Errorf("unmarshal approved contract approved by: %w", err)
	}
	if err := json.Unmarshal(sourceRefs, &approved.SourceRefs); err != nil {
		return spine.ApprovedContract{}, fmt.Errorf("unmarshal approved contract source refs: %w", err)
	}
	approved.ApprovedAt = approved.ApprovedAt.UTC()
	return approved, nil
}

func approvedContractColumns() []string {
	return []string{
		"id",
		"organization_id",
		"project_id",
		"repo_binding_id",
		"contract_draft_id",
		"contract_seed_id",
		"goal_id",
		"title",
		"intent_summary",
		"scope",
		"non_goals",
		"constraints",
		"acceptance_criteria",
		"expected_checks",
		"proof_expectations",
		"risk_hints",
		"approved_by",
		"approved_at",
		"source_refs",
		"state",
	}
}

func (s *PostgresApprovedContractStore) execSQL(ctx context.Context, op string, sqlizer squirrel.Sqlizer) error {
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
