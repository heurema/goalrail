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

	"github.com/heurema/goalrail/apps/server/internal/checkout"
	"github.com/heurema/goalrail/apps/server/internal/spine"
)

type PostgresCheckoutJobStore struct {
	exec  postgresExecer
	query postgresRowQuerier
	psql  squirrel.StatementBuilderType
}

func NewPostgresCheckoutJobStore(pool *pgxpool.Pool) *PostgresCheckoutJobStore {
	db := newPostgresDB(pool)
	return NewPostgresCheckoutJobStoreWithExecutorAndQuerier(db, db)
}

func NewPostgresCheckoutJobStoreWithExecutorAndQuerier(exec postgresExecer, query postgresRowQuerier) *PostgresCheckoutJobStore {
	return &PostgresCheckoutJobStore{
		exec:  exec,
		query: query,
		psql:  squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (s *PostgresCheckoutJobStore) Create(ctx context.Context, job spine.CheckoutJob) error {
	id, err := uuidValue(job.ID, "checkout job id")
	if err != nil {
		return err
	}
	orgID, err := uuidValue(job.OrganizationID, "checkout job organization id")
	if err != nil {
		return err
	}
	projectID, err := uuidValue(job.ProjectID, "checkout job project id")
	if err != nil {
		return err
	}
	taskID, err := uuidValue(job.TaskID, "checkout job task id")
	if err != nil {
		return err
	}
	contractID, err := uuidValue(job.ContractID, "checkout job contract id")
	if err != nil {
		return err
	}
	approvedContractID, err := uuidValue(job.ApprovedContractID, "checkout job approved contract id")
	if err != nil {
		return err
	}
	planID, err := uuidValue(job.PlanID, "checkout job plan id")
	if err != nil {
		return err
	}
	proposalID, err := uuidValue(job.ProposalID, "checkout job proposal id")
	if err != nil {
		return err
	}
	repoBindingID, err := uuidValue(job.RepoBindingID, "checkout job repo binding id")
	if err != nil {
		return err
	}
	requestedBy, err := json.Marshal(job.RequestedBy)
	if err != nil {
		return fmt.Errorf("marshal checkout job requested_by: %w", err)
	}
	instruction, err := json.Marshal(job.Instruction)
	if err != nil {
		return fmt.Errorf("marshal checkout job instruction: %w", err)
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
		Insert("checkout_jobs").
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
			"state",
			"requested_by",
			"instruction",
			"current_runner_id",
			"lease_token_hash",
			"lease_expires_at",
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
			job.State,
			requestedBy,
			instruction,
			"",
			"",
			nil,
			createdAt,
			updatedAt,
		)
	if err := execSQL(ctx, s.exec, "create checkout job", stmt); err != nil {
		if uniqueViolationConstraint(err) == "checkout_jobs_task_id_unique" {
			return ErrCheckoutJobAlreadyPrepared
		}
		if isUniqueViolation(err) {
			return ErrCheckoutJobAlreadyExists
		}
		return err
	}
	return nil
}

func (s *PostgresCheckoutJobStore) Get(ctx context.Context, id spine.CheckoutJobID) (spine.CheckoutJob, bool, error) {
	jobID, err := uuidValue(id, "checkout job id")
	if err != nil {
		return spine.CheckoutJob{}, false, err
	}
	return s.getOne(ctx, "get checkout job", squirrel.Eq{"id": jobID})
}

func (s *PostgresCheckoutJobStore) GetByTaskID(ctx context.Context, id spine.WorkItemID) (spine.CheckoutJob, bool, error) {
	taskID, err := uuidValue(id, "checkout job task id")
	if err != nil {
		return spine.CheckoutJob{}, false, err
	}
	return s.getOne(ctx, "get checkout job by task id", squirrel.Eq{"task_id": taskID})
}

func (s *PostgresCheckoutJobStore) AcquireNextLease(ctx context.Context, input checkout.JobLeaseInput) (spine.CheckoutJob, bool, error) {
	if s.exec == nil || s.query == nil {
		return spine.CheckoutJob{}, false, fmt.Errorf("checkout job store executor is nil")
	}
	orgID, err := uuidValue(input.OrganizationID, "checkout job lease organization id")
	if err != nil {
		return spine.CheckoutJob{}, false, err
	}
	now := input.UpdatedAt.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	row := s.query.QueryRow(ctx, `
SELECT id, organization_id, project_id, task_id, contract_id, approved_contract_id, plan_id, proposal_id, repo_binding_id, state, requested_by, instruction, current_runner_id, lease_token_hash, lease_expires_at, created_at, updated_at
FROM checkout_jobs
WHERE organization_id = $1 AND (state = $2 OR (state = $3 AND lease_expires_at <= $4))
ORDER BY created_at ASC, id ASC
LIMIT 1
FOR UPDATE SKIP LOCKED
`, orgID, spine.CheckoutJobStateQueued, spine.CheckoutJobStateLeased, now)
	job, err := scanCheckoutJob(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return spine.CheckoutJob{}, false, nil
		}
		return spine.CheckoutJob{}, false, fmt.Errorf("select next checkout job lease candidate: %w", err)
	}

	parsedJobID, err := uuidValue(job.ID, "checkout job id")
	if err != nil {
		return spine.CheckoutJob{}, false, err
	}
	stmt := s.psql.
		Update("checkout_jobs").
		Set("state", spine.CheckoutJobStateLeased).
		Set("current_runner_id", input.RunnerID).
		Set("lease_token_hash", input.LeaseTokenHash).
		Set("lease_expires_at", input.LeaseExpiresAt.UTC()).
		Set("updated_at", now).
		Where(squirrel.Eq{"id": parsedJobID})
	if err := execUpdate(ctx, s.exec, "mark checkout job leased", ErrCheckoutJobNotFound, stmt); err != nil {
		return spine.CheckoutJob{}, false, err
	}
	job.State = spine.CheckoutJobStateLeased
	job.CurrentRunnerID = input.RunnerID
	job.LeaseTokenHash = input.LeaseTokenHash
	job.LeaseExpiresAt = &input.LeaseExpiresAt
	job.UpdatedAt = now
	return job, true, nil
}

func (s *PostgresCheckoutJobStore) MarkReceiptSubmitted(ctx context.Context, id spine.CheckoutJobID, runnerID string, tokenHash string, updatedAt time.Time) (bool, error) {
	jobID, err := uuidValue(id, "checkout job id")
	if err != nil {
		return false, err
	}
	stmt := s.psql.
		Update("checkout_jobs").
		Set("state", spine.CheckoutJobStateReceiptSubmitted).
		Set("updated_at", updatedAt.UTC()).
		Where(squirrel.Eq{
			"id":                jobID,
			"state":             spine.CheckoutJobStateLeased,
			"current_runner_id": runnerID,
			"lease_token_hash":  tokenHash,
		}).
		Where(squirrel.Gt{"lease_expires_at": updatedAt.UTC()})
	sqlText, args, err := stmt.ToSql()
	if err != nil {
		return false, fmt.Errorf("mark checkout job receipt submitted SQL: %w", err)
	}
	result, err := s.exec.Exec(ctx, sqlText, args...)
	if err != nil {
		return false, fmt.Errorf("mark checkout job receipt submitted: %w", err)
	}
	return result.RowsAffected() > 0, nil
}

func (s *PostgresCheckoutJobStore) getOne(ctx context.Context, op string, where squirrel.Eq) (spine.CheckoutJob, bool, error) {
	stmt := s.psql.Select(checkoutJobColumns()...).From("checkout_jobs").Where(where)
	row, err := queryRow(ctx, s.query, op, stmt)
	if err != nil {
		return spine.CheckoutJob{}, false, err
	}
	job, err := scanCheckoutJob(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return spine.CheckoutJob{}, false, nil
		}
		return spine.CheckoutJob{}, false, fmt.Errorf("%s: %w", op, err)
	}
	return job, true, nil
}

func scanCheckoutJob(row pgx.Row) (spine.CheckoutJob, error) {
	var job spine.CheckoutJob
	var id string
	var organizationID string
	var projectID string
	var taskID string
	var contractID string
	var approvedContractID string
	var planID string
	var proposalID string
	var repoBindingID string
	var state string
	var requestedBy []byte
	var instruction []byte
	var currentRunnerID string
	var leaseExpiresAt pgtype.Timestamptz
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
		&state,
		&requestedBy,
		&instruction,
		&currentRunnerID,
		&job.LeaseTokenHash,
		&leaseExpiresAt,
		&job.CreatedAt,
		&job.UpdatedAt,
	); err != nil {
		return spine.CheckoutJob{}, err
	}
	job.ID = spine.CheckoutJobID(id)
	job.OrganizationID = spine.OrganizationID(organizationID)
	job.ProjectID = spine.ProjectID(projectID)
	job.TaskID = spine.WorkItemID(taskID)
	job.ContractID = spine.ContractID(contractID)
	job.ApprovedContractID = spine.ApprovedContractID(approvedContractID)
	job.PlanID = spine.WorkItemPlanID(planID)
	job.ProposalID = spine.WorkItemPlanProposalID(proposalID)
	job.RepoBindingID = spine.RepoBindingID(repoBindingID)
	job.State = spine.CheckoutJobState(state)
	if err := json.Unmarshal(requestedBy, &job.RequestedBy); err != nil {
		return spine.CheckoutJob{}, fmt.Errorf("unmarshal checkout job requested_by: %w", err)
	}
	if err := json.Unmarshal(instruction, &job.Instruction); err != nil {
		return spine.CheckoutJob{}, fmt.Errorf("unmarshal checkout job instruction: %w", err)
	}
	job.CurrentRunnerID = currentRunnerID
	if leaseExpiresAt.Valid {
		value := leaseExpiresAt.Time.UTC()
		job.LeaseExpiresAt = &value
	}
	job.CreatedAt = job.CreatedAt.UTC()
	job.UpdatedAt = job.UpdatedAt.UTC()
	return job, nil
}

func checkoutJobColumns() []string {
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
		"state",
		"requested_by",
		"instruction",
		"current_runner_id",
		"lease_token_hash",
		"lease_expires_at",
		"created_at",
		"updated_at",
	}
}

type PostgresCheckoutReceiptStore struct {
	exec postgresExecer
	psql squirrel.StatementBuilderType
}

func NewPostgresCheckoutReceiptStore(pool *pgxpool.Pool) *PostgresCheckoutReceiptStore {
	return NewPostgresCheckoutReceiptStoreWithExecutor(newPostgresDB(pool))
}

func NewPostgresCheckoutReceiptStoreWithExecutor(exec postgresExecer) *PostgresCheckoutReceiptStore {
	return &PostgresCheckoutReceiptStore{
		exec: exec,
		psql: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (s *PostgresCheckoutReceiptStore) Create(ctx context.Context, receipt spine.CheckoutReceipt) error {
	id, err := uuidValue(receipt.ID, "checkout receipt id")
	if err != nil {
		return err
	}
	jobID, err := uuidValue(receipt.JobID, "checkout receipt job id")
	if err != nil {
		return err
	}
	taskID, err := uuidValue(receipt.TaskID, "checkout receipt task id")
	if err != nil {
		return err
	}
	repoBindingID, err := uuidValue(receipt.RepoBindingID, "checkout receipt repo binding id")
	if err != nil {
		return err
	}
	partialReasons, err := json.Marshal(nonNilStrings(receipt.PartialReasons))
	if err != nil {
		return fmt.Errorf("marshal checkout receipt partial reasons: %w", err)
	}
	createdAt := receipt.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	stmt := s.psql.
		Insert("checkout_receipts").
		Columns(
			"id",
			"job_id",
			"task_id",
			"repo_binding_id",
			"runner_id",
			"workspace_ref",
			"commit_sha",
			"baseline_id",
			"overlay_id",
			"dirty",
			"partial",
			"partial_reasons",
			"raw_source_uploaded",
			"created_at",
		).
		Values(
			id,
			jobID,
			taskID,
			repoBindingID,
			receipt.RunnerID,
			receipt.WorkspaceRef,
			receipt.CommitSHA,
			receipt.BaselineID,
			receipt.OverlayID,
			receipt.Dirty,
			receipt.Partial,
			partialReasons,
			receipt.RawSourceUploaded,
			createdAt,
		)
	if err := execSQL(ctx, s.exec, "create checkout receipt", stmt); err != nil {
		if uniqueViolationConstraint(err) == "checkout_receipts_job_id_unique" {
			return checkout.ErrAlreadyReceipted
		}
		if isUniqueViolation(err) {
			return ErrCheckoutReceiptAlreadyExists
		}
		return err
	}
	return nil
}
