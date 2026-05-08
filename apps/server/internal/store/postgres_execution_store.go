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

type PostgresExecutionJobStore struct {
	exec  postgresExecer
	query postgresRowQuerier
	psql  squirrel.StatementBuilderType
}

func NewPostgresExecutionJobStore(pool *pgxpool.Pool) *PostgresExecutionJobStore {
	db := newPostgresDB(pool)
	return NewPostgresExecutionJobStoreWithExecutorAndQuerier(db, db)
}

func NewPostgresExecutionJobStoreWithExecutorAndQuerier(exec postgresExecer, query postgresRowQuerier) *PostgresExecutionJobStore {
	return &PostgresExecutionJobStore{
		exec:  exec,
		query: query,
		psql:  squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (s *PostgresExecutionJobStore) Create(ctx context.Context, job spine.ExecutionJob) error {
	id, err := uuidValue(job.ID, "execution job id")
	if err != nil {
		return err
	}
	orgID, err := uuidValue(job.OrganizationID, "execution job organization id")
	if err != nil {
		return err
	}
	projectID, err := uuidValue(job.ProjectID, "execution job project id")
	if err != nil {
		return err
	}
	taskID, err := uuidValue(job.TaskID, "execution job task id")
	if err != nil {
		return err
	}
	contractID, err := uuidValue(job.ContractID, "execution job contract id")
	if err != nil {
		return err
	}
	approvedContractID, err := uuidValue(job.ApprovedContractID, "execution job approved contract id")
	if err != nil {
		return err
	}
	planID, err := uuidValue(job.PlanID, "execution job plan id")
	if err != nil {
		return err
	}
	proposalID, err := uuidValue(job.ProposalID, "execution job proposal id")
	if err != nil {
		return err
	}
	repoBindingID, err := uuidValue(job.RepoBindingID, "execution job repo binding id")
	if err != nil {
		return err
	}
	checkoutJobID, err := uuidValue(job.CheckoutJobID, "execution job checkout job id")
	if err != nil {
		return err
	}
	checkoutReceiptID, err := uuidValue(job.CheckoutReceiptID, "execution job checkout receipt id")
	if err != nil {
		return err
	}
	requestedBy, err := json.Marshal(job.RequestedBy)
	if err != nil {
		return fmt.Errorf("marshal execution job requested_by: %w", err)
	}
	createdAt := job.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	updatedAt := job.UpdatedAt.UTC()
	if updatedAt.IsZero() {
		updatedAt = createdAt
	}
	stmt := s.psql.
		Insert("execution_jobs").
		Columns(
			"id",
			"organization_id",
			"project_id",
			"task_id",
			"contract_id",
			"approved_contract_id",
			"plan_id",
			"proposal_id",
			"repo_binding_id",
			"checkout_job_id",
			"checkout_receipt_id",
			"state",
			"requested_by",
			"execution_mode",
			"created_at",
			"updated_at",
		).
		Values(
			id,
			orgID,
			projectID,
			taskID,
			contractID,
			approvedContractID,
			planID,
			proposalID,
			repoBindingID,
			checkoutJobID,
			checkoutReceiptID,
			job.State,
			requestedBy,
			job.ExecutionMode,
			createdAt,
			updatedAt,
		)
	if err := execSQL(ctx, s.exec, "create execution job", stmt); err != nil {
		if uniqueViolationConstraint(err) == "execution_jobs_task_receipt_unique" {
			return ErrExecutionJobAlreadyPrepared
		}
		if isUniqueViolation(err) {
			return ErrExecutionJobAlreadyExists
		}
		return err
	}
	return nil
}

func (s *PostgresExecutionJobStore) GetByTaskAndCheckoutReceipt(ctx context.Context, taskID spine.WorkItemID, checkoutReceiptID spine.CheckoutReceiptID) (spine.ExecutionJob, bool, error) {
	parsedTaskID, err := uuidValue(taskID, "execution job task id")
	if err != nil {
		return spine.ExecutionJob{}, false, err
	}
	parsedReceiptID, err := uuidValue(checkoutReceiptID, "execution job checkout receipt id")
	if err != nil {
		return spine.ExecutionJob{}, false, err
	}
	return s.getOne(ctx, "get execution job by task and checkout receipt", squirrel.Eq{
		"task_id":             parsedTaskID,
		"checkout_receipt_id": parsedReceiptID,
	})
}

func (s *PostgresExecutionJobStore) getOne(ctx context.Context, op string, where squirrel.Eq) (spine.ExecutionJob, bool, error) {
	stmt := s.psql.Select(executionJobColumns()...).From("execution_jobs").Where(where)
	row, err := queryRow(ctx, s.query, op, stmt)
	if err != nil {
		return spine.ExecutionJob{}, false, err
	}
	job, err := scanExecutionJob(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return spine.ExecutionJob{}, false, nil
		}
		return spine.ExecutionJob{}, false, fmt.Errorf("%s: %w", op, err)
	}
	return job, true, nil
}

func scanExecutionJob(row pgx.Row) (spine.ExecutionJob, error) {
	var job spine.ExecutionJob
	var id string
	var organizationID string
	var projectID string
	var taskID string
	var contractID string
	var approvedContractID string
	var planID string
	var proposalID string
	var repoBindingID string
	var checkoutJobID string
	var checkoutReceiptID string
	var state string
	var requestedBy []byte
	if err := row.Scan(
		&id,
		&organizationID,
		&projectID,
		&taskID,
		&contractID,
		&approvedContractID,
		&planID,
		&proposalID,
		&repoBindingID,
		&checkoutJobID,
		&checkoutReceiptID,
		&state,
		&requestedBy,
		&job.ExecutionMode,
		&job.CreatedAt,
		&job.UpdatedAt,
	); err != nil {
		return spine.ExecutionJob{}, err
	}
	job.ID = spine.ExecutionJobID(id)
	job.OrganizationID = spine.OrganizationID(organizationID)
	job.ProjectID = spine.ProjectID(projectID)
	job.TaskID = spine.WorkItemID(taskID)
	job.ContractID = spine.ContractID(contractID)
	job.ApprovedContractID = spine.ApprovedContractID(approvedContractID)
	job.PlanID = spine.WorkItemPlanID(planID)
	job.ProposalID = spine.WorkItemPlanProposalID(proposalID)
	job.RepoBindingID = spine.RepoBindingID(repoBindingID)
	job.CheckoutJobID = spine.CheckoutJobID(checkoutJobID)
	job.CheckoutReceiptID = spine.CheckoutReceiptID(checkoutReceiptID)
	job.State = spine.ExecutionJobState(state)
	if err := json.Unmarshal(requestedBy, &job.RequestedBy); err != nil {
		return spine.ExecutionJob{}, fmt.Errorf("unmarshal execution job requested_by: %w", err)
	}
	job.CreatedAt = job.CreatedAt.UTC()
	job.UpdatedAt = job.UpdatedAt.UTC()
	return job, nil
}

func executionJobColumns() []string {
	return []string{
		"id",
		"organization_id",
		"project_id",
		"task_id",
		"contract_id",
		"approved_contract_id",
		"plan_id",
		"proposal_id",
		"repo_binding_id",
		"checkout_job_id",
		"checkout_receipt_id",
		"state",
		"requested_by",
		"execution_mode",
		"created_at",
		"updated_at",
	}
}
