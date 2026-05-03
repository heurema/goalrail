package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	squirrel "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

type PostgresWorkItemPlanStore struct {
	exec  postgresExecer
	query postgresRowQuerier
	psql  squirrel.StatementBuilderType
}

func NewPostgresWorkItemPlanStore(pool *pgxpool.Pool) *PostgresWorkItemPlanStore {
	db := newPostgresDB(pool)
	return NewPostgresWorkItemPlanStoreWithExecutorAndQuerier(db, db)
}

func NewPostgresWorkItemPlanStoreWithExecutorAndQuerier(exec postgresExecer, query postgresRowQuerier) *PostgresWorkItemPlanStore {
	return &PostgresWorkItemPlanStore{
		exec:  exec,
		query: query,
		psql:  squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (s *PostgresWorkItemPlanStore) Create(ctx context.Context, plan spine.WorkItemPlan) error {
	id, err := uuidValue(plan.ID, "work item plan id")
	if err != nil {
		return err
	}
	orgID, err := uuidValue(plan.OrganizationID, "work item plan organization id")
	if err != nil {
		return err
	}
	projectID, err := uuidValue(plan.ProjectID, "work item plan project id")
	if err != nil {
		return err
	}
	contractID, err := uuidValue(plan.ContractID, "work item plan contract id")
	if err != nil {
		return err
	}
	approvedContractID, err := uuidValue(plan.ApprovedContractID, "work item plan approved contract id")
	if err != nil {
		return err
	}
	repoBindingID, err := uuidValue(plan.RepoBindingID, "work item plan repo binding id")
	if err != nil {
		return err
	}
	requestedBy, err := json.Marshal(plan.RequestedBy)
	if err != nil {
		return fmt.Errorf("marshal work item plan requested_by: %w", err)
	}
	createdAt := plan.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	updatedAt := plan.UpdatedAt.UTC()
	if updatedAt.IsZero() {
		updatedAt = createdAt
	}

	stmt := s.psql.
		Insert("work_item_plans").
		Columns(
			"id",
			"organization_id",
			"project_id",
			"contract_id",
			"approved_contract_id",
			"repo_binding_id",
			"state",
			"requested_by",
			"created_at",
			"updated_at",
		).
		Values(
			id,
			orgID,
			projectID,
			contractID,
			approvedContractID,
			repoBindingID,
			plan.State,
			requestedBy,
			createdAt,
			updatedAt,
		)
	if err := execSQL(ctx, s.exec, "create work item plan", stmt); err != nil {
		if uniqueViolationConstraint(err) == "work_item_plans_contract_id_unique" {
			return ErrWorkItemPlanAlreadyPlanned
		}
		if isUniqueViolation(err) {
			return ErrWorkItemPlanAlreadyExists
		}
		return err
	}
	return nil
}

func (s *PostgresWorkItemPlanStore) Get(ctx context.Context, id spine.WorkItemPlanID) (spine.WorkItemPlan, bool, error) {
	planID, err := uuidValue(id, "work item plan id")
	if err != nil {
		return spine.WorkItemPlan{}, false, err
	}
	return s.getOne(ctx, "get work item plan", squirrel.Eq{"id": planID})
}

func (s *PostgresWorkItemPlanStore) GetByContractID(ctx context.Context, id spine.ContractID) (spine.WorkItemPlan, bool, error) {
	contractID, err := uuidValue(id, "work item plan contract id")
	if err != nil {
		return spine.WorkItemPlan{}, false, err
	}
	return s.getOne(ctx, "get work item plan by contract id", squirrel.Eq{"contract_id": contractID})
}

func (s *PostgresWorkItemPlanStore) MarkProposalSubmitted(ctx context.Context, id spine.WorkItemPlanID, updatedAt time.Time) error {
	return s.updateState(ctx, id, spine.WorkItemPlanStateProposalSubmitted, updatedAt)
}

func (s *PostgresWorkItemPlanStore) MarkAccepted(ctx context.Context, id spine.WorkItemPlanID, updatedAt time.Time) error {
	return s.updateState(ctx, id, spine.WorkItemPlanStateAccepted, updatedAt)
}

func (s *PostgresWorkItemPlanStore) updateState(ctx context.Context, id spine.WorkItemPlanID, state spine.WorkItemPlanState, updatedAt time.Time) error {
	planID, err := uuidValue(id, "work item plan id")
	if err != nil {
		return err
	}
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}
	stmt := s.psql.
		Update("work_item_plans").
		Set("state", state).
		Set("updated_at", updatedAt.UTC()).
		Where(squirrel.Eq{"id": planID})
	return execUpdate(ctx, s.exec, "update work item plan state", ErrWorkItemPlanNotFound, stmt)
}

func (s *PostgresWorkItemPlanStore) getOne(ctx context.Context, op string, where squirrel.Eq) (spine.WorkItemPlan, bool, error) {
	stmt := s.psql.
		Select(workItemPlanColumns()...).
		From("work_item_plans").
		Where(where)
	row, err := queryRow(ctx, s.query, op, stmt)
	if err != nil {
		return spine.WorkItemPlan{}, false, err
	}
	plan, err := scanWorkItemPlan(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return spine.WorkItemPlan{}, false, nil
		}
		return spine.WorkItemPlan{}, false, fmt.Errorf("%s: %w", op, err)
	}
	return plan, true, nil
}

func scanWorkItemPlan(row pgx.Row) (spine.WorkItemPlan, error) {
	var plan spine.WorkItemPlan
	var id string
	var organizationID string
	var projectID string
	var contractID string
	var approvedContractID string
	var repoBindingID string
	var state string
	var requestedBy []byte
	if err := row.Scan(
		&id,
		&organizationID,
		&projectID,
		&contractID,
		&approvedContractID,
		&repoBindingID,
		&state,
		&requestedBy,
		&plan.CreatedAt,
		&plan.UpdatedAt,
	); err != nil {
		return spine.WorkItemPlan{}, err
	}
	plan.ID = spine.WorkItemPlanID(id)
	plan.OrganizationID = spine.OrganizationID(organizationID)
	plan.ProjectID = spine.ProjectID(projectID)
	plan.ContractID = spine.ContractID(contractID)
	plan.ApprovedContractID = spine.ApprovedContractID(approvedContractID)
	plan.RepoBindingID = spine.RepoBindingID(repoBindingID)
	plan.State = spine.WorkItemPlanState(state)
	if err := json.Unmarshal(requestedBy, &plan.RequestedBy); err != nil {
		return spine.WorkItemPlan{}, fmt.Errorf("unmarshal work item plan requested_by: %w", err)
	}
	plan.CreatedAt = plan.CreatedAt.UTC()
	plan.UpdatedAt = plan.UpdatedAt.UTC()
	return plan, nil
}

func workItemPlanColumns() []string {
	return []string{
		"id",
		"organization_id",
		"project_id",
		"contract_id",
		"approved_contract_id",
		"repo_binding_id",
		"state",
		"requested_by",
		"created_at",
		"updated_at",
	}
}

type PostgresWorkItemPlanProposalStore struct {
	exec  postgresExecer
	query postgresRowQuerier
	psql  squirrel.StatementBuilderType
}

func NewPostgresWorkItemPlanProposalStore(pool *pgxpool.Pool) *PostgresWorkItemPlanProposalStore {
	db := newPostgresDB(pool)
	return NewPostgresWorkItemPlanProposalStoreWithExecutorAndQuerier(db, db)
}

func NewPostgresWorkItemPlanProposalStoreWithExecutorAndQuerier(exec postgresExecer, query postgresRowQuerier) *PostgresWorkItemPlanProposalStore {
	return &PostgresWorkItemPlanProposalStore{
		exec:  exec,
		query: query,
		psql:  squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (s *PostgresWorkItemPlanProposalStore) Create(ctx context.Context, proposal spine.WorkItemPlanProposal) error {
	id, err := uuidValue(proposal.ID, "work item plan proposal id")
	if err != nil {
		return err
	}
	planID, err := uuidValue(proposal.PlanID, "work item plan proposal plan id")
	if err != nil {
		return err
	}
	orgID, err := uuidValue(proposal.OrganizationID, "work item plan proposal organization id")
	if err != nil {
		return err
	}
	projectID, err := uuidValue(proposal.ProjectID, "work item plan proposal project id")
	if err != nil {
		return err
	}
	contractID, err := uuidValue(proposal.ContractID, "work item plan proposal contract id")
	if err != nil {
		return err
	}
	approvedContractID, err := uuidValue(proposal.ApprovedContractID, "work item plan proposal approved contract id")
	if err != nil {
		return err
	}
	repoBindingID, err := uuidValue(proposal.RepoBindingID, "work item plan proposal repo binding id")
	if err != nil {
		return err
	}
	submittedBy, err := json.Marshal(proposal.SubmittedBy)
	if err != nil {
		return fmt.Errorf("marshal work item plan proposal submitted_by: %w", err)
	}
	planner := proposal.Planner
	if planner == nil {
		planner = map[string]any{}
	}
	plannerJSON, err := json.Marshal(planner)
	if err != nil {
		return fmt.Errorf("marshal work item plan proposal planner: %w", err)
	}
	sourceSnapshotRefs, err := json.Marshal(nonNilSourceRefs(proposal.SourceSnapshotRefs))
	if err != nil {
		return fmt.Errorf("marshal work item plan proposal source snapshot refs: %w", err)
	}
	proposedTasks, err := json.Marshal(nonNilProposedWorkItems(proposal.ProposedTasks))
	if err != nil {
		return fmt.Errorf("marshal work item plan proposal proposed tasks: %w", err)
	}
	createdAt := proposal.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	updatedAt := proposal.UpdatedAt.UTC()
	if updatedAt.IsZero() {
		updatedAt = createdAt
	}

	stmt := s.psql.
		Insert("work_item_plan_proposals").
		Columns(
			"id",
			"plan_id",
			"organization_id",
			"project_id",
			"contract_id",
			"approved_contract_id",
			"repo_binding_id",
			"state",
			"submitted_by",
			"planner",
			"source_snapshot_refs",
			"rationale",
			"proposed_tasks",
			"created_at",
			"updated_at",
		).
		Values(
			id,
			planID,
			orgID,
			projectID,
			contractID,
			approvedContractID,
			repoBindingID,
			proposal.State,
			submittedBy,
			plannerJSON,
			sourceSnapshotRefs,
			proposal.Rationale,
			proposedTasks,
			createdAt,
			updatedAt,
		)
	if err := execSQL(ctx, s.exec, "create work item plan proposal", stmt); err != nil {
		if uniqueViolationConstraint(err) == "work_item_plan_proposals_plan_id_unique" {
			return ErrWorkItemPlanAlreadyHasProposal
		}
		if isUniqueViolation(err) {
			return ErrWorkItemPlanProposalExists
		}
		return err
	}
	return nil
}

func (s *PostgresWorkItemPlanProposalStore) Get(ctx context.Context, id spine.WorkItemPlanProposalID) (spine.WorkItemPlanProposal, bool, error) {
	proposalID, err := uuidValue(id, "work item plan proposal id")
	if err != nil {
		return spine.WorkItemPlanProposal{}, false, err
	}
	return s.getOne(ctx, "get work item plan proposal", squirrel.Eq{"id": proposalID})
}

func (s *PostgresWorkItemPlanProposalStore) GetByPlanID(ctx context.Context, id spine.WorkItemPlanID) (spine.WorkItemPlanProposal, bool, error) {
	planID, err := uuidValue(id, "work item plan proposal plan id")
	if err != nil {
		return spine.WorkItemPlanProposal{}, false, err
	}
	return s.getOne(ctx, "get work item plan proposal by plan id", squirrel.Eq{"plan_id": planID})
}

func (s *PostgresWorkItemPlanProposalStore) MarkAccepted(ctx context.Context, id spine.WorkItemPlanProposalID, acceptedBy spine.ActorRef, acceptedAt time.Time) error {
	proposalID, err := uuidValue(id, "work item plan proposal id")
	if err != nil {
		return err
	}
	acceptedByJSON, err := json.Marshal(acceptedBy)
	if err != nil {
		return fmt.Errorf("marshal work item plan proposal accepted_by: %w", err)
	}
	if acceptedAt.IsZero() {
		acceptedAt = time.Now().UTC()
	}
	stmt := s.psql.
		Update("work_item_plan_proposals").
		Set("state", spine.WorkItemProposalStateAccepted).
		Set("accepted_by", acceptedByJSON).
		Set("accepted_at", acceptedAt.UTC()).
		Set("updated_at", acceptedAt.UTC()).
		Where(squirrel.Eq{"id": proposalID})
	return execUpdate(ctx, s.exec, "mark work item plan proposal accepted", ErrWorkItemPlanProposalNotFound, stmt)
}

func (s *PostgresWorkItemPlanProposalStore) getOne(ctx context.Context, op string, where squirrel.Eq) (spine.WorkItemPlanProposal, bool, error) {
	stmt := s.psql.
		Select(workItemPlanProposalColumns()...).
		From("work_item_plan_proposals").
		Where(where)
	row, err := queryRow(ctx, s.query, op, stmt)
	if err != nil {
		return spine.WorkItemPlanProposal{}, false, err
	}
	proposal, err := scanWorkItemPlanProposal(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return spine.WorkItemPlanProposal{}, false, nil
		}
		return spine.WorkItemPlanProposal{}, false, fmt.Errorf("%s: %w", op, err)
	}
	return proposal, true, nil
}

func scanWorkItemPlanProposal(row pgx.Row) (spine.WorkItemPlanProposal, error) {
	var proposal spine.WorkItemPlanProposal
	var id string
	var planID string
	var organizationID string
	var projectID string
	var contractID string
	var approvedContractID string
	var repoBindingID string
	var state string
	var submittedBy []byte
	var planner []byte
	var sourceSnapshotRefs []byte
	var proposedTasks []byte
	var acceptedBy []byte
	var acceptedAt pgtype.Timestamptz
	if err := row.Scan(
		&id,
		&planID,
		&organizationID,
		&projectID,
		&contractID,
		&approvedContractID,
		&repoBindingID,
		&state,
		&submittedBy,
		&planner,
		&sourceSnapshotRefs,
		&proposal.Rationale,
		&proposedTasks,
		&acceptedBy,
		&acceptedAt,
		&proposal.CreatedAt,
		&proposal.UpdatedAt,
	); err != nil {
		return spine.WorkItemPlanProposal{}, err
	}
	proposal.ID = spine.WorkItemPlanProposalID(id)
	proposal.PlanID = spine.WorkItemPlanID(planID)
	proposal.OrganizationID = spine.OrganizationID(organizationID)
	proposal.ProjectID = spine.ProjectID(projectID)
	proposal.ContractID = spine.ContractID(contractID)
	proposal.ApprovedContractID = spine.ApprovedContractID(approvedContractID)
	proposal.RepoBindingID = spine.RepoBindingID(repoBindingID)
	proposal.State = spine.WorkItemProposalState(state)
	if err := json.Unmarshal(submittedBy, &proposal.SubmittedBy); err != nil {
		return spine.WorkItemPlanProposal{}, fmt.Errorf("unmarshal work item plan proposal submitted_by: %w", err)
	}
	if err := json.Unmarshal(planner, &proposal.Planner); err != nil {
		return spine.WorkItemPlanProposal{}, fmt.Errorf("unmarshal work item plan proposal planner: %w", err)
	}
	if proposal.Planner == nil {
		proposal.Planner = map[string]any{}
	}
	if err := json.Unmarshal(sourceSnapshotRefs, &proposal.SourceSnapshotRefs); err != nil {
		return spine.WorkItemPlanProposal{}, fmt.Errorf("unmarshal work item plan proposal source snapshot refs: %w", err)
	}
	if err := json.Unmarshal(proposedTasks, &proposal.ProposedTasks); err != nil {
		return spine.WorkItemPlanProposal{}, fmt.Errorf("unmarshal work item plan proposal proposed tasks: %w", err)
	}
	if len(acceptedBy) > 0 {
		var actor spine.ActorRef
		if err := json.Unmarshal(acceptedBy, &actor); err != nil {
			return spine.WorkItemPlanProposal{}, fmt.Errorf("unmarshal work item plan proposal accepted_by: %w", err)
		}
		proposal.AcceptedBy = &actor
	}
	if acceptedAt.Valid {
		value := acceptedAt.Time.UTC()
		proposal.AcceptedAt = &value
	}
	proposal.CreatedAt = proposal.CreatedAt.UTC()
	proposal.UpdatedAt = proposal.UpdatedAt.UTC()
	return proposal, nil
}

func workItemPlanProposalColumns() []string {
	return []string{
		"id",
		"plan_id",
		"organization_id",
		"project_id",
		"contract_id",
		"approved_contract_id",
		"repo_binding_id",
		"state",
		"submitted_by",
		"planner",
		"source_snapshot_refs",
		"rationale",
		"proposed_tasks",
		"accepted_by",
		"accepted_at",
		"created_at",
		"updated_at",
	}
}

func nonNilSourceRefs(values []spine.SourceRef) []spine.SourceRef {
	if values == nil {
		return []spine.SourceRef{}
	}
	return values
}

func nonNilProposedWorkItems(values []spine.ProposedWorkItem) []spine.ProposedWorkItem {
	if values == nil {
		return []spine.ProposedWorkItem{}
	}
	return values
}
