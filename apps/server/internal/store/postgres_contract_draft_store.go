package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	squirrel "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

type PostgresContractDraftStore struct {
	exec  postgresExecer
	query postgresRowQuerier
	psql  squirrel.StatementBuilderType
}

func NewPostgresContractDraftStore(pool *pgxpool.Pool) *PostgresContractDraftStore {
	db := newPostgresDB(pool)
	return NewPostgresContractDraftStoreWithExecutorAndQuerier(db, db)
}

func NewPostgresContractDraftStoreWithExecutorAndQuerier(exec postgresExecer, query postgresRowQuerier) *PostgresContractDraftStore {
	return &PostgresContractDraftStore{
		exec:  exec,
		query: query,
		psql:  squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (s *PostgresContractDraftStore) Create(ctx context.Context, created spine.ContractDraft) error {
	id, err := uuidValue(created.ID, "contract draft id")
	if err != nil {
		return err
	}
	orgID, err := uuidValue(created.OrganizationID, "contract draft organization id")
	if err != nil {
		return err
	}
	projectID, err := uuidValue(created.ProjectID, "contract draft project id")
	if err != nil {
		return err
	}
	repoBindingID, err := uuidValue(created.RepoBindingID, "contract draft repo binding id")
	if err != nil {
		return err
	}
	contractID, err := uuidValue(created.ContractID, "contract draft contract id")
	if err != nil {
		return err
	}
	seedID, err := uuidValue(created.ContractSeedID, "contract draft contract seed id")
	if err != nil {
		return err
	}
	goalID, err := uuidValue(created.GoalID, "contract draft goal id")
	if err != nil {
		return err
	}
	proposedScope, err := json.Marshal(nonNilStrings(created.ProposedScope))
	if err != nil {
		return fmt.Errorf("marshal contract draft proposed scope: %w", err)
	}
	proposedNonGoals, err := json.Marshal(nonNilStrings(created.ProposedNonGoals))
	if err != nil {
		return fmt.Errorf("marshal contract draft proposed non-goals: %w", err)
	}
	proposedConstraints, err := json.Marshal(nonNilStrings(created.ProposedConstraints))
	if err != nil {
		return fmt.Errorf("marshal contract draft proposed constraints: %w", err)
	}
	proposedAcceptanceCriteria, err := json.Marshal(nonNilStrings(created.ProposedAcceptanceCriteria))
	if err != nil {
		return fmt.Errorf("marshal contract draft proposed acceptance criteria: %w", err)
	}
	proposedExpectedChecks, err := json.Marshal(nonNilStrings(created.ProposedExpectedChecks))
	if err != nil {
		return fmt.Errorf("marshal contract draft proposed expected checks: %w", err)
	}
	proposedProofExpectations, err := json.Marshal(nonNilStrings(created.ProposedProofExpectations))
	if err != nil {
		return fmt.Errorf("marshal contract draft proposed proof expectations: %w", err)
	}
	riskHints, err := json.Marshal(nonNilStrings(created.RiskHints))
	if err != nil {
		return fmt.Errorf("marshal contract draft risk hints: %w", err)
	}
	sourceRefsValue := created.SourceRefs
	if sourceRefsValue == nil {
		sourceRefsValue = []spine.SourceRef{}
	}
	sourceRefs, err := json.Marshal(sourceRefsValue)
	if err != nil {
		return fmt.Errorf("marshal contract draft source refs: %w", err)
	}

	createdAt := created.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	stmt := s.psql.
		Insert("contract_drafts").
		Columns(
			"id",
			"organization_id",
			"project_id",
			"repo_binding_id",
			"contract_id",
			"contract_seed_id",
			"goal_id",
			"title",
			"intent_summary",
			"proposed_scope",
			"proposed_non_goals",
			"proposed_constraints",
			"proposed_acceptance_criteria",
			"proposed_expected_checks",
			"proposed_proof_expectations",
			"risk_hints",
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
			contractID,
			seedID,
			goalID,
			created.Title,
			created.IntentSummary,
			proposedScope,
			proposedNonGoals,
			proposedConstraints,
			proposedAcceptanceCriteria,
			proposedExpectedChecks,
			proposedProofExpectations,
			riskHints,
			sourceRefs,
			created.State,
			createdAt,
			createdAt,
		)

	if err := execSQL(ctx, s.exec, "create contract draft", stmt); err != nil {
		if uniqueViolationConstraint(err) == "contract_drafts_contract_seed_id_unique" {
			return ErrContractDraftAlreadyDrafted
		}
		if isUniqueViolation(err) {
			return ErrContractDraftAlreadyExists
		}
		return err
	}
	return nil
}

func (s *PostgresContractDraftStore) Get(ctx context.Context, id spine.ContractDraftID) (spine.ContractDraft, bool, error) {
	draftID, err := uuidValue(id, "contract draft id")
	if err != nil {
		return spine.ContractDraft{}, false, err
	}
	return s.getOne(ctx, "get contract draft", squirrel.Eq{"id": draftID})
}

func (s *PostgresContractDraftStore) GetByContractSeedID(ctx context.Context, id spine.ContractSeedID) (spine.ContractDraft, bool, error) {
	seedID, err := uuidValue(id, "contract draft contract seed id")
	if err != nil {
		return spine.ContractDraft{}, false, err
	}
	return s.getOne(ctx, "get contract draft by contract seed id", squirrel.Eq{"contract_seed_id": seedID})
}

func (s *PostgresContractDraftStore) Update(ctx context.Context, updated spine.ContractDraft) error {
	id, err := uuidValue(updated.ID, "contract draft id")
	if err != nil {
		return err
	}
	proposedScope, err := json.Marshal(nonNilStrings(updated.ProposedScope))
	if err != nil {
		return fmt.Errorf("marshal contract draft proposed scope: %w", err)
	}
	proposedNonGoals, err := json.Marshal(nonNilStrings(updated.ProposedNonGoals))
	if err != nil {
		return fmt.Errorf("marshal contract draft proposed non-goals: %w", err)
	}
	proposedConstraints, err := json.Marshal(nonNilStrings(updated.ProposedConstraints))
	if err != nil {
		return fmt.Errorf("marshal contract draft proposed constraints: %w", err)
	}
	proposedAcceptanceCriteria, err := json.Marshal(nonNilStrings(updated.ProposedAcceptanceCriteria))
	if err != nil {
		return fmt.Errorf("marshal contract draft proposed acceptance criteria: %w", err)
	}
	proposedExpectedChecks, err := json.Marshal(nonNilStrings(updated.ProposedExpectedChecks))
	if err != nil {
		return fmt.Errorf("marshal contract draft proposed expected checks: %w", err)
	}
	proposedProofExpectations, err := json.Marshal(nonNilStrings(updated.ProposedProofExpectations))
	if err != nil {
		return fmt.Errorf("marshal contract draft proposed proof expectations: %w", err)
	}
	riskHints, err := json.Marshal(nonNilStrings(updated.RiskHints))
	if err != nil {
		return fmt.Errorf("marshal contract draft risk hints: %w", err)
	}

	stmt := s.psql.
		Update("contract_drafts").
		Set("title", updated.Title).
		Set("intent_summary", updated.IntentSummary).
		Set("proposed_scope", proposedScope).
		Set("proposed_non_goals", proposedNonGoals).
		Set("proposed_constraints", proposedConstraints).
		Set("proposed_acceptance_criteria", proposedAcceptanceCriteria).
		Set("proposed_expected_checks", proposedExpectedChecks).
		Set("proposed_proof_expectations", proposedProofExpectations).
		Set("risk_hints", riskHints).
		Set("updated_at", time.Now().UTC()).
		Where(squirrel.Eq{"id": id})

	return execUpdate(ctx, s.exec, "update contract draft", ErrContractDraftNotFound, stmt)
}

func (s *PostgresContractDraftStore) MarkReadyForApproval(ctx context.Context, updated spine.ContractDraft) error {
	id, err := uuidValue(updated.ID, "contract draft id")
	if err != nil {
		return err
	}

	stmt := s.psql.
		Update("contract_drafts").
		Set("state", updated.State).
		Set("updated_at", time.Now().UTC()).
		Where(squirrel.Eq{"id": id})

	return execUpdate(ctx, s.exec, "mark contract draft ready for approval", ErrContractDraftNotFound, stmt)
}

func (s *PostgresContractDraftStore) getOne(ctx context.Context, op string, where squirrel.Eq) (spine.ContractDraft, bool, error) {
	stmt := s.psql.
		Select(contractDraftColumns()...).
		From("contract_drafts").
		Where(where)

	return s.queryContractDraft(ctx, op, stmt)
}

func (s *PostgresContractDraftStore) queryContractDraft(ctx context.Context, op string, sqlizer squirrel.Sqlizer) (spine.ContractDraft, bool, error) {
	row, err := queryRow(ctx, s.query, op, sqlizer)
	if err != nil {
		return spine.ContractDraft{}, false, err
	}
	draft, err := scanContractDraft(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return spine.ContractDraft{}, false, nil
		}
		return spine.ContractDraft{}, false, fmt.Errorf("%s: %w", op, err)
	}
	return draft, true, nil
}

func scanContractDraft(row pgx.Row) (spine.ContractDraft, error) {
	var draft spine.ContractDraft
	var id string
	var organizationID string
	var projectID string
	var repoBindingID string
	var contractID string
	var seedID string
	var goalID string
	var proposedScope []byte
	var proposedNonGoals []byte
	var proposedConstraints []byte
	var proposedAcceptanceCriteria []byte
	var proposedExpectedChecks []byte
	var proposedProofExpectations []byte
	var riskHints []byte
	var sourceRefs []byte
	var state string
	if err := row.Scan(
		&id,
		&organizationID,
		&projectID,
		&repoBindingID,
		&contractID,
		&seedID,
		&goalID,
		&draft.Title,
		&draft.IntentSummary,
		&proposedScope,
		&proposedNonGoals,
		&proposedConstraints,
		&proposedAcceptanceCriteria,
		&proposedExpectedChecks,
		&proposedProofExpectations,
		&riskHints,
		&sourceRefs,
		&state,
		&draft.CreatedAt,
	); err != nil {
		return spine.ContractDraft{}, err
	}
	draft.ID = spine.ContractDraftID(id)
	draft.OrganizationID = spine.OrganizationID(organizationID)
	draft.ProjectID = spine.ProjectID(projectID)
	draft.RepoBindingID = spine.RepoBindingID(repoBindingID)
	draft.ContractID = spine.ContractID(contractID)
	draft.ContractSeedID = spine.ContractSeedID(seedID)
	draft.GoalID = spine.GoalID(goalID)
	draft.State = spine.ContractDraftState(state)
	if err := json.Unmarshal(proposedScope, &draft.ProposedScope); err != nil {
		return spine.ContractDraft{}, fmt.Errorf("unmarshal contract draft proposed scope: %w", err)
	}
	if err := json.Unmarshal(proposedNonGoals, &draft.ProposedNonGoals); err != nil {
		return spine.ContractDraft{}, fmt.Errorf("unmarshal contract draft proposed non-goals: %w", err)
	}
	if err := json.Unmarshal(proposedConstraints, &draft.ProposedConstraints); err != nil {
		return spine.ContractDraft{}, fmt.Errorf("unmarshal contract draft proposed constraints: %w", err)
	}
	if err := json.Unmarshal(proposedAcceptanceCriteria, &draft.ProposedAcceptanceCriteria); err != nil {
		return spine.ContractDraft{}, fmt.Errorf("unmarshal contract draft proposed acceptance criteria: %w", err)
	}
	if err := json.Unmarshal(proposedExpectedChecks, &draft.ProposedExpectedChecks); err != nil {
		return spine.ContractDraft{}, fmt.Errorf("unmarshal contract draft proposed expected checks: %w", err)
	}
	if err := json.Unmarshal(proposedProofExpectations, &draft.ProposedProofExpectations); err != nil {
		return spine.ContractDraft{}, fmt.Errorf("unmarshal contract draft proposed proof expectations: %w", err)
	}
	if err := json.Unmarshal(riskHints, &draft.RiskHints); err != nil {
		return spine.ContractDraft{}, fmt.Errorf("unmarshal contract draft risk hints: %w", err)
	}
	if err := json.Unmarshal(sourceRefs, &draft.SourceRefs); err != nil {
		return spine.ContractDraft{}, fmt.Errorf("unmarshal contract draft source refs: %w", err)
	}
	draft.CreatedAt = draft.CreatedAt.UTC()
	return draft, nil
}

func contractDraftColumns() []string {
	return []string{
		"id",
		"organization_id",
		"project_id",
		"repo_binding_id",
		"contract_id",
		"contract_seed_id",
		"goal_id",
		"title",
		"intent_summary",
		"proposed_scope",
		"proposed_non_goals",
		"proposed_constraints",
		"proposed_acceptance_criteria",
		"proposed_expected_checks",
		"proposed_proof_expectations",
		"risk_hints",
		"source_refs",
		"state",
		"created_at",
	}
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
