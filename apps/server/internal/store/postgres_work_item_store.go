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

type PostgresWorkItemStore struct {
	exec  postgresExecer
	query postgresRowQuerier
	psql  squirrel.StatementBuilderType
}

func NewPostgresWorkItemStore(pool *pgxpool.Pool) *PostgresWorkItemStore {
	db := newPostgresDB(pool)
	return NewPostgresWorkItemStoreWithExecutorAndQuerier(db, db)
}

func NewPostgresWorkItemStoreWithExecutorAndQuerier(exec postgresExecer, query postgresRowQuerier) *PostgresWorkItemStore {
	return &PostgresWorkItemStore{
		exec:  exec,
		query: query,
		psql:  squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (s *PostgresWorkItemStore) Create(ctx context.Context, item spine.WorkItem) error {
	id, err := uuidValue(item.ID, "work item id")
	if err != nil {
		return err
	}
	orgID, err := uuidValue(item.OrganizationID, "work item organization id")
	if err != nil {
		return err
	}
	projectID, err := uuidValue(item.ProjectID, "work item project id")
	if err != nil {
		return err
	}
	contractID, err := uuidValue(item.ContractID, "work item contract id")
	if err != nil {
		return err
	}
	approvedContractID, err := uuidValue(item.ApprovedContractID, "work item approved contract id")
	if err != nil {
		return err
	}
	planID, err := uuidValue(item.PlanID, "work item plan id")
	if err != nil {
		return err
	}
	proposalID, err := uuidValue(item.ProposalID, "work item proposal id")
	if err != nil {
		return err
	}
	repoBindingID, err := uuidValue(item.RepoBindingID, "work item repo binding id")
	if err != nil {
		return err
	}
	scope, err := json.Marshal(nonNilStrings(item.Scope))
	if err != nil {
		return fmt.Errorf("marshal work item scope: %w", err)
	}
	acceptanceRefs, err := json.Marshal(nonNilStrings(item.AcceptanceRefs))
	if err != nil {
		return fmt.Errorf("marshal work item acceptance refs: %w", err)
	}
	proofExpectationRefs, err := json.Marshal(nonNilStrings(item.ProofExpectationRefs))
	if err != nil {
		return fmt.Errorf("marshal work item proof expectation refs: %w", err)
	}
	sourceRefsValue := item.SourceRefs
	if sourceRefsValue == nil {
		sourceRefsValue = []spine.SourceRef{}
	}
	sourceRefs, err := json.Marshal(sourceRefsValue)
	if err != nil {
		return fmt.Errorf("marshal work item source refs: %w", err)
	}

	createdAt := item.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	var orderIndex any
	if item.OrderIndex != nil {
		orderIndex = *item.OrderIndex
	}
	stmt := s.psql.
		Insert("work_items").
		Columns(
			"id",
			"organization_id",
			"project_id",
			"contract_id",
			"approved_contract_id",
			"plan_id",
			"proposal_id",
			"repo_binding_id",
			"title",
			"summary",
			"scope",
			"acceptance_refs",
			"proof_expectation_refs",
			"status",
			"owner_hint",
			"order_index",
			"source_refs",
			"created_at",
			"updated_at",
		).
		Values(
			id,
			orgID,
			projectID,
			contractID,
			approvedContractID,
			planID,
			proposalID,
			repoBindingID,
			item.Title,
			item.Summary,
			scope,
			acceptanceRefs,
			proofExpectationRefs,
			item.Status,
			item.OwnerHint,
			orderIndex,
			sourceRefs,
			createdAt,
			createdAt,
		)

	if err := execSQL(ctx, s.exec, "create work item", stmt); err != nil {
		if isUniqueViolation(err) {
			return ErrWorkItemAlreadyExists
		}
		return err
	}
	return nil
}

func (s *PostgresWorkItemStore) Get(ctx context.Context, id spine.WorkItemID) (spine.WorkItem, bool, error) {
	workItemID, err := uuidValue(id, "work item id")
	if err != nil {
		return spine.WorkItem{}, false, err
	}
	return s.getOne(ctx, "get work item", squirrel.Eq{"id": workItemID})
}

func (s *PostgresWorkItemStore) GetByApprovedContractID(ctx context.Context, id spine.ApprovedContractID) (spine.WorkItem, bool, error) {
	approvedContractID, err := uuidValue(id, "work item approved contract id")
	if err != nil {
		return spine.WorkItem{}, false, err
	}
	return s.getOne(ctx, "get work item by approved contract id", squirrel.Eq{"approved_contract_id": approvedContractID})
}

func (s *PostgresWorkItemStore) getOne(ctx context.Context, op string, where squirrel.Eq) (spine.WorkItem, bool, error) {
	stmt := s.psql.
		Select(workItemColumns()...).
		From("work_items").
		Where(where)

	return s.queryWorkItem(ctx, op, stmt)
}

func (s *PostgresWorkItemStore) queryWorkItem(ctx context.Context, op string, sqlizer squirrel.Sqlizer) (spine.WorkItem, bool, error) {
	if s.query == nil {
		return spine.WorkItem{}, false, fmt.Errorf("%s query executor is nil", op)
	}
	sqlText, args, err := sqlizer.ToSql()
	if err != nil {
		return spine.WorkItem{}, false, fmt.Errorf("%s SQL: %w", op, err)
	}
	item, err := scanWorkItem(s.query.QueryRow(ctx, sqlText, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return spine.WorkItem{}, false, nil
		}
		return spine.WorkItem{}, false, fmt.Errorf("%s: %w", op, err)
	}
	return item, true, nil
}

func scanWorkItem(row pgx.Row) (spine.WorkItem, error) {
	var item spine.WorkItem
	var id string
	var organizationID string
	var projectID string
	var contractID string
	var approvedContractID string
	var planID string
	var proposalID string
	var repoBindingID string
	var scope []byte
	var acceptanceRefs []byte
	var proofExpectationRefs []byte
	var status string
	var orderIndex pgtype.Int4
	var sourceRefs []byte
	if err := row.Scan(
		&id,
		&organizationID,
		&projectID,
		&contractID,
		&approvedContractID,
		&planID,
		&proposalID,
		&repoBindingID,
		&item.Title,
		&item.Summary,
		&scope,
		&acceptanceRefs,
		&proofExpectationRefs,
		&status,
		&item.OwnerHint,
		&orderIndex,
		&sourceRefs,
		&item.CreatedAt,
	); err != nil {
		return spine.WorkItem{}, err
	}
	item.ID = spine.WorkItemID(id)
	item.OrganizationID = spine.OrganizationID(organizationID)
	item.ProjectID = spine.ProjectID(projectID)
	item.ContractID = spine.ContractID(contractID)
	item.ApprovedContractID = spine.ApprovedContractID(approvedContractID)
	item.PlanID = spine.WorkItemPlanID(planID)
	item.ProposalID = spine.WorkItemPlanProposalID(proposalID)
	item.RepoBindingID = spine.RepoBindingID(repoBindingID)
	item.Status = spine.WorkItemStatus(status)
	if orderIndex.Valid {
		value := int(orderIndex.Int32)
		item.OrderIndex = &value
	}
	if err := json.Unmarshal(scope, &item.Scope); err != nil {
		return spine.WorkItem{}, fmt.Errorf("unmarshal work item scope: %w", err)
	}
	if err := json.Unmarshal(acceptanceRefs, &item.AcceptanceRefs); err != nil {
		return spine.WorkItem{}, fmt.Errorf("unmarshal work item acceptance refs: %w", err)
	}
	if err := json.Unmarshal(proofExpectationRefs, &item.ProofExpectationRefs); err != nil {
		return spine.WorkItem{}, fmt.Errorf("unmarshal work item proof expectation refs: %w", err)
	}
	if err := json.Unmarshal(sourceRefs, &item.SourceRefs); err != nil {
		return spine.WorkItem{}, fmt.Errorf("unmarshal work item source refs: %w", err)
	}
	item.CreatedAt = item.CreatedAt.UTC()
	return item, nil
}

func workItemColumns() []string {
	return []string{
		"id",
		"organization_id",
		"project_id",
		"contract_id",
		"approved_contract_id",
		"plan_id",
		"proposal_id",
		"repo_binding_id",
		"title",
		"summary",
		"scope",
		"acceptance_refs",
		"proof_expectation_refs",
		"status",
		"owner_hint",
		"order_index",
		"source_refs",
		"created_at",
	}
}
