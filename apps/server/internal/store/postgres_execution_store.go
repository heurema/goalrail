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

	"github.com/heurema/goalrail/apps/server/internal/execution"
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
			"current_lease_id",
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
			checkoutJobID,
			checkoutReceiptID,
			job.State,
			requestedBy,
			job.ExecutionMode,
			nil,
			"",
			"",
			nil,
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

func (s *PostgresExecutionJobStore) Get(ctx context.Context, id spine.ExecutionJobID) (spine.ExecutionJob, bool, error) {
	jobID, err := uuidValue(id, "execution job id")
	if err != nil {
		return spine.ExecutionJob{}, false, err
	}
	return s.getOne(ctx, "get execution job", squirrel.Eq{"id": jobID})
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

func (s *PostgresExecutionJobStore) AcquireNextLease(ctx context.Context, input execution.JobLeaseInput) (spine.ExecutionLease, spine.ExecutionJob, bool, error) {
	if s.exec == nil || s.query == nil {
		return spine.ExecutionLease{}, spine.ExecutionJob{}, false, fmt.Errorf("execution job store executor is nil")
	}
	orgID, err := uuidValue(input.OrganizationID, "execution job lease organization id")
	if err != nil {
		return spine.ExecutionLease{}, spine.ExecutionJob{}, false, err
	}
	projectID, err := uuidValue(input.ProjectID, "execution job lease project id")
	if err != nil {
		return spine.ExecutionLease{}, spine.ExecutionJob{}, false, err
	}
	repoBindingID, err := uuidValue(input.RepoBindingID, "execution job lease repo binding id")
	if err != nil {
		return spine.ExecutionLease{}, spine.ExecutionJob{}, false, err
	}
	now := input.UpdatedAt.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	row := s.query.QueryRow(ctx, `
SELECT id, organization_id, project_id, task_id, contract_id, approved_contract_id, plan_id, proposal_id, repo_binding_id, checkout_job_id, checkout_receipt_id, state, requested_by, execution_mode, current_lease_id, current_runner_id, lease_token_hash, lease_expires_at, created_at, updated_at
FROM execution_jobs
WHERE organization_id = $1 AND project_id = $2 AND repo_binding_id = $3 AND (state = $4 OR (state = $5 AND lease_expires_at <= $6))
ORDER BY created_at ASC, id ASC
LIMIT 1
FOR UPDATE SKIP LOCKED
`, orgID, projectID, repoBindingID, spine.ExecutionJobStateQueued, spine.ExecutionJobStateLeased, now)
	job, err := scanExecutionJob(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return spine.ExecutionLease{}, spine.ExecutionJob{}, false, nil
		}
		return spine.ExecutionLease{}, spine.ExecutionJob{}, false, fmt.Errorf("select next execution job lease candidate: %w", err)
	}

	if job.State == spine.ExecutionJobStateLeased && job.CurrentLeaseID != nil {
		previousID, err := uuidValue(*job.CurrentLeaseID, "current execution lease id")
		if err != nil {
			return spine.ExecutionLease{}, spine.ExecutionJob{}, false, err
		}
		if _, err := s.exec.Exec(ctx, `
UPDATE execution_leases
SET state = $1, updated_at = $2
WHERE id = $3 AND state = $4
`, spine.ExecutionLeaseStateExpired, now, previousID, spine.ExecutionLeaseStateActive); err != nil {
			return spine.ExecutionLease{}, spine.ExecutionJob{}, false, fmt.Errorf("expire previous execution lease: %w", err)
		}
	}

	lease := spine.ExecutionLease{
		ID:                input.ID,
		ExecutionJobID:    job.ID,
		TaskID:            job.TaskID,
		CheckoutReceiptID: job.CheckoutReceiptID,
		RepoBindingID:     job.RepoBindingID,
		RunnerID:          input.RunnerID,
		State:             spine.ExecutionLeaseStateActive,
		LeaseTokenHash:    input.LeaseTokenHash,
		ExpiresAt:         input.ExpiresAt.UTC(),
		CreatedAt:         input.CreatedAt.UTC(),
		UpdatedAt:         input.UpdatedAt.UTC(),
	}
	if lease.CreatedAt.IsZero() {
		lease.CreatedAt = now
	}
	if lease.UpdatedAt.IsZero() {
		lease.UpdatedAt = now
	}
	if err := s.insertExecutionLease(ctx, lease); err != nil {
		return spine.ExecutionLease{}, spine.ExecutionJob{}, false, err
	}
	if err := s.markExecutionJobLeased(ctx, job.ID, lease); err != nil {
		return spine.ExecutionLease{}, spine.ExecutionJob{}, false, err
	}
	job.State = spine.ExecutionJobStateLeased
	job.CurrentLeaseID = &lease.ID
	job.CurrentRunnerID = lease.RunnerID
	job.LeaseTokenHash = lease.LeaseTokenHash
	job.LeaseExpiresAt = &lease.ExpiresAt
	job.UpdatedAt = lease.UpdatedAt
	return lease, job, true, nil
}

func (s *PostgresExecutionJobStore) MarkRunStarted(ctx context.Context, id spine.ExecutionJobID, leaseID spine.ExecutionLeaseID, runnerID string, tokenHash string, updatedAt time.Time) (bool, error) {
	jobID, err := uuidValue(id, "execution job id")
	if err != nil {
		return false, err
	}
	parsedLeaseID, err := uuidValue(leaseID, "execution lease id")
	if err != nil {
		return false, err
	}
	now := updatedAt.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	stmt := s.psql.
		Update("execution_jobs").
		Set("state", spine.ExecutionJobStateRunStarted).
		Set("updated_at", now).
		Where(squirrel.Eq{
			"id":                jobID,
			"state":             spine.ExecutionJobStateLeased,
			"current_lease_id":  parsedLeaseID,
			"current_runner_id": runnerID,
			"lease_token_hash":  tokenHash,
		}).
		Where(squirrel.Gt{"lease_expires_at": now})
	sqlText, args, err := stmt.ToSql()
	if err != nil {
		return false, fmt.Errorf("mark execution job run started SQL: %w", err)
	}
	result, err := s.exec.Exec(ctx, sqlText, args...)
	if err != nil {
		return false, fmt.Errorf("mark execution job run started: %w", err)
	}
	if result.RowsAffected() == 0 {
		return false, nil
	}
	leaseStmt := s.psql.
		Update("execution_leases").
		Set("state", spine.ExecutionLeaseStateRunStarted).
		Set("updated_at", now).
		Where(squirrel.Eq{"id": parsedLeaseID, "lease_token_hash": tokenHash, "state": spine.ExecutionLeaseStateActive}).
		Where(squirrel.Gt{"expires_at": now})
	if err := execUpdate(ctx, s.exec, "mark execution lease run started", ErrExecutionLeaseNotFound, leaseStmt); err != nil {
		return false, err
	}
	return true, nil
}

func (s *PostgresExecutionJobStore) MarkReceiptSubmitted(ctx context.Context, id spine.ExecutionJobID, updatedAt time.Time) (bool, error) {
	jobID, err := uuidValue(id, "execution job id")
	if err != nil {
		return false, err
	}
	now := updatedAt.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	stmt := s.psql.
		Update("execution_jobs").
		Set("state", spine.ExecutionJobStateReceiptSubmitted).
		Set("updated_at", now).
		Where(squirrel.Eq{"id": jobID, "state": spine.ExecutionJobStateRunStarted})
	sqlText, args, err := stmt.ToSql()
	if err != nil {
		return false, fmt.Errorf("mark execution job receipt submitted SQL: %w", err)
	}
	result, err := s.exec.Exec(ctx, sqlText, args...)
	if err != nil {
		return false, fmt.Errorf("mark execution job receipt submitted: %w", err)
	}
	return result.RowsAffected() > 0, nil
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
	var currentLeaseID pgtype.UUID
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
		&checkoutJobID,
		&checkoutReceiptID,
		&state,
		&requestedBy,
		&job.ExecutionMode,
		&currentLeaseID,
		&job.CurrentRunnerID,
		&job.LeaseTokenHash,
		&leaseExpiresAt,
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
	if value := uuidString(currentLeaseID); value != "" {
		leaseID := spine.ExecutionLeaseID(value)
		job.CurrentLeaseID = &leaseID
	}
	if leaseExpiresAt.Valid {
		value := leaseExpiresAt.Time.UTC()
		job.LeaseExpiresAt = &value
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
		"current_lease_id",
		"current_runner_id",
		"lease_token_hash",
		"lease_expires_at",
		"created_at",
		"updated_at",
	}
}

func (s *PostgresExecutionJobStore) insertExecutionLease(ctx context.Context, lease spine.ExecutionLease) error {
	id, err := uuidValue(lease.ID, "execution lease id")
	if err != nil {
		return err
	}
	jobID, err := uuidValue(lease.ExecutionJobID, "execution lease job id")
	if err != nil {
		return err
	}
	taskID, err := uuidValue(lease.TaskID, "execution lease task id")
	if err != nil {
		return err
	}
	checkoutReceiptID, err := uuidValue(lease.CheckoutReceiptID, "execution lease checkout receipt id")
	if err != nil {
		return err
	}
	repoBindingID, err := uuidValue(lease.RepoBindingID, "execution lease repo binding id")
	if err != nil {
		return err
	}
	stmt := s.psql.
		Insert("execution_leases").
		Columns("id", "execution_job_id", "task_id", "checkout_receipt_id", "repo_binding_id", "runner_id", "state", "lease_token_hash", "expires_at", "created_at", "updated_at").
		Values(id, jobID, taskID, checkoutReceiptID, repoBindingID, lease.RunnerID, lease.State, lease.LeaseTokenHash, lease.ExpiresAt.UTC(), lease.CreatedAt.UTC(), lease.UpdatedAt.UTC())
	if err := execSQL(ctx, s.exec, "create execution lease", stmt); err != nil {
		if isUniqueViolation(err) {
			return ErrExecutionLeaseAlreadyExists
		}
		return err
	}
	return nil
}

func (s *PostgresExecutionJobStore) markExecutionJobLeased(ctx context.Context, jobID spine.ExecutionJobID, lease spine.ExecutionLease) error {
	parsedJobID, err := uuidValue(jobID, "execution job id")
	if err != nil {
		return err
	}
	parsedLeaseID, err := uuidValue(lease.ID, "execution lease id")
	if err != nil {
		return err
	}
	stmt := s.psql.
		Update("execution_jobs").
		Set("state", spine.ExecutionJobStateLeased).
		Set("current_lease_id", parsedLeaseID).
		Set("current_runner_id", lease.RunnerID).
		Set("lease_token_hash", lease.LeaseTokenHash).
		Set("lease_expires_at", lease.ExpiresAt.UTC()).
		Set("updated_at", lease.UpdatedAt.UTC()).
		Where(squirrel.Eq{"id": parsedJobID})
	return execUpdate(ctx, s.exec, "mark execution job leased", ErrExecutionJobNotFound, stmt)
}

type PostgresRunStore struct {
	exec  postgresExecer
	query postgresRowQuerier
	psql  squirrel.StatementBuilderType
}

func NewPostgresRunStore(pool *pgxpool.Pool) *PostgresRunStore {
	db := newPostgresDB(pool)
	return NewPostgresRunStoreWithExecutorAndQuerier(db, db)
}

func NewPostgresRunStoreWithExecutorAndQuerier(exec postgresExecer, query postgresRowQuerier) *PostgresRunStore {
	return &PostgresRunStore{
		exec:  exec,
		query: query,
		psql:  squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (s *PostgresRunStore) Create(ctx context.Context, run spine.Run) error {
	id, err := uuidValue(run.ID, "run id")
	if err != nil {
		return err
	}
	jobID, err := uuidValue(run.ExecutionJobID, "run execution job id")
	if err != nil {
		return err
	}
	leaseID, err := uuidValue(run.ExecutionLeaseID, "run execution lease id")
	if err != nil {
		return err
	}
	taskID, err := uuidValue(run.TaskID, "run task id")
	if err != nil {
		return err
	}
	checkoutReceiptID, err := uuidValue(run.CheckoutReceiptID, "run checkout receipt id")
	if err != nil {
		return err
	}
	stmt := s.psql.
		Insert("runs").
		Columns("id", "execution_job_id", "execution_lease_id", "task_id", "checkout_receipt_id", "runner_id", "state", "started_at", "created_at", "updated_at").
		Values(id, jobID, leaseID, taskID, checkoutReceiptID, run.RunnerID, run.State, run.StartedAt.UTC(), run.CreatedAt.UTC(), run.UpdatedAt.UTC())
	if err := execSQL(ctx, s.exec, "create run", stmt); err != nil {
		if isUniqueViolation(err) {
			return ErrRunAlreadyExists
		}
		return err
	}
	return nil
}

func (s *PostgresRunStore) Get(ctx context.Context, id spine.RunID) (spine.Run, bool, error) {
	runID, err := uuidValue(id, "run id")
	if err != nil {
		return spine.Run{}, false, err
	}
	stmt := s.psql.Select(runColumns()...).From("runs").Where(squirrel.Eq{"id": runID})
	row, err := queryRow(ctx, s.query, "get run", stmt)
	if err != nil {
		return spine.Run{}, false, err
	}
	run, err := scanRun(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return spine.Run{}, false, nil
		}
		return spine.Run{}, false, fmt.Errorf("get run: %w", err)
	}
	return run, true, nil
}

func (s *PostgresRunStore) GetByExecutionLease(ctx context.Context, leaseID spine.ExecutionLeaseID) (spine.Run, bool, error) {
	parsedLeaseID, err := uuidValue(leaseID, "run execution lease id")
	if err != nil {
		return spine.Run{}, false, err
	}
	stmt := s.psql.Select(runColumns()...).From("runs").Where(squirrel.Eq{"execution_lease_id": parsedLeaseID})
	row, err := queryRow(ctx, s.query, "get run by execution lease", stmt)
	if err != nil {
		return spine.Run{}, false, err
	}
	run, err := scanRun(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return spine.Run{}, false, nil
		}
		return spine.Run{}, false, fmt.Errorf("get run by execution lease: %w", err)
	}
	return run, true, nil
}

func (s *PostgresRunStore) MarkReceiptSubmitted(ctx context.Context, id spine.RunID, finishedAt time.Time, updatedAt time.Time) (bool, error) {
	runID, err := uuidValue(id, "run id")
	if err != nil {
		return false, err
	}
	finished := finishedAt.UTC()
	if finished.IsZero() {
		finished = time.Now().UTC()
	}
	now := updatedAt.UTC()
	if now.IsZero() {
		now = finished
	}
	stmt := s.psql.
		Update("runs").
		Set("state", spine.RunStateReceiptSubmitted).
		Set("finished_at", finished).
		Set("updated_at", now).
		Where(squirrel.Eq{"id": runID, "state": spine.RunStateStarted})
	sqlText, args, err := stmt.ToSql()
	if err != nil {
		return false, fmt.Errorf("mark run receipt submitted SQL: %w", err)
	}
	result, err := s.exec.Exec(ctx, sqlText, args...)
	if err != nil {
		return false, fmt.Errorf("mark run receipt submitted: %w", err)
	}
	return result.RowsAffected() > 0, nil
}

func scanRun(row pgx.Row) (spine.Run, error) {
	var run spine.Run
	var id string
	var executionJobID string
	var executionLeaseID string
	var taskID string
	var checkoutReceiptID string
	var state string
	var finishedAt pgtype.Timestamptz
	if err := row.Scan(
		&id,
		&executionJobID,
		&executionLeaseID,
		&taskID,
		&checkoutReceiptID,
		&run.RunnerID,
		&state,
		&run.StartedAt,
		&finishedAt,
		&run.CreatedAt,
		&run.UpdatedAt,
	); err != nil {
		return spine.Run{}, err
	}
	run.ID = spine.RunID(id)
	run.ExecutionJobID = spine.ExecutionJobID(executionJobID)
	run.ExecutionLeaseID = spine.ExecutionLeaseID(executionLeaseID)
	run.TaskID = spine.WorkItemID(taskID)
	run.CheckoutReceiptID = spine.CheckoutReceiptID(checkoutReceiptID)
	run.State = spine.RunState(state)
	run.StartedAt = run.StartedAt.UTC()
	if finishedAt.Valid {
		value := finishedAt.Time.UTC()
		run.FinishedAt = &value
	}
	run.CreatedAt = run.CreatedAt.UTC()
	run.UpdatedAt = run.UpdatedAt.UTC()
	return run, nil
}

func runColumns() []string {
	return []string{
		"id",
		"execution_job_id",
		"execution_lease_id",
		"task_id",
		"checkout_receipt_id",
		"runner_id",
		"state",
		"started_at",
		"finished_at",
		"created_at",
		"updated_at",
	}
}

type PostgresExecutionReceiptStore struct {
	exec  postgresExecer
	query postgresRowQuerier
	psql  squirrel.StatementBuilderType
}

func NewPostgresExecutionReceiptStore(pool *pgxpool.Pool) *PostgresExecutionReceiptStore {
	db := newPostgresDB(pool)
	return NewPostgresExecutionReceiptStoreWithExecutorAndQuerier(db, db)
}

func NewPostgresExecutionReceiptStoreWithExecutorAndQuerier(exec postgresExecer, query postgresRowQuerier) *PostgresExecutionReceiptStore {
	return &PostgresExecutionReceiptStore{
		exec:  exec,
		query: query,
		psql:  squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (s *PostgresExecutionReceiptStore) Create(ctx context.Context, receipt spine.ExecutionReceipt) error {
	id, err := uuidValue(receipt.ID, "execution receipt id")
	if err != nil {
		return err
	}
	runID, err := uuidValue(receipt.RunID, "execution receipt run id")
	if err != nil {
		return err
	}
	jobID, err := uuidValue(receipt.ExecutionJobID, "execution receipt job id")
	if err != nil {
		return err
	}
	leaseID, err := uuidValue(receipt.ExecutionLeaseID, "execution receipt lease id")
	if err != nil {
		return err
	}
	taskID, err := uuidValue(receipt.TaskID, "execution receipt task id")
	if err != nil {
		return err
	}
	checkoutReceiptID, err := uuidValue(receipt.CheckoutReceiptID, "execution receipt checkout receipt id")
	if err != nil {
		return err
	}
	repoBindingID, err := uuidValue(receipt.RepoBindingID, "execution receipt repo binding id")
	if err != nil {
		return err
	}
	artifactRefs, err := json.Marshal(receipt.ArtifactRefs)
	if err != nil {
		return fmt.Errorf("marshal execution receipt artifact refs: %w", err)
	}
	changedPaths, err := json.Marshal(receipt.ChangedPathsSummary)
	if err != nil {
		return fmt.Errorf("marshal execution receipt changed paths summary: %w", err)
	}
	var exitCode any
	if receipt.ExitCode != nil {
		exitCode = *receipt.ExitCode
	}
	stmt := s.psql.
		Insert("execution_receipts").
		Columns(
			"id",
			"run_id",
			"execution_job_id",
			"execution_lease_id",
			"task_id",
			"checkout_receipt_id",
			"repo_binding_id",
			"runner_id",
			"workspace_ref",
			"commit_sha",
			"baseline_id",
			"overlay_id",
			"execution_mode",
			"process_status",
			"exit_code",
			"artifact_refs",
			"changed_paths_summary",
			"raw_source_uploaded",
			"started_at",
			"finished_at",
			"created_at",
			"updated_at",
		).
		Values(
			id,
			runID,
			jobID,
			leaseID,
			taskID,
			checkoutReceiptID,
			repoBindingID,
			receipt.RunnerID,
			receipt.WorkspaceRef,
			receipt.CommitSHA,
			receipt.BaselineID,
			receipt.OverlayID,
			receipt.ExecutionMode,
			receipt.ProcessStatus,
			exitCode,
			artifactRefs,
			changedPaths,
			receipt.RawSourceUploaded,
			receipt.StartedAt.UTC(),
			receipt.FinishedAt.UTC(),
			receipt.CreatedAt.UTC(),
			receipt.UpdatedAt.UTC(),
		)
	if err := execSQL(ctx, s.exec, "create execution receipt", stmt); err != nil {
		if uniqueViolationConstraint(err) == "execution_receipts_run_id_unique" {
			return ErrExecutionReceiptAlreadySubmitted
		}
		if isUniqueViolation(err) {
			return ErrExecutionReceiptAlreadyExists
		}
		return err
	}
	return nil
}

func (s *PostgresExecutionReceiptStore) GetByRun(ctx context.Context, runID spine.RunID) (spine.ExecutionReceipt, bool, error) {
	parsedRunID, err := uuidValue(runID, "execution receipt run id")
	if err != nil {
		return spine.ExecutionReceipt{}, false, err
	}
	stmt := s.psql.Select(executionReceiptColumns()...).From("execution_receipts").Where(squirrel.Eq{"run_id": parsedRunID})
	row, err := queryRow(ctx, s.query, "get execution receipt by run", stmt)
	if err != nil {
		return spine.ExecutionReceipt{}, false, err
	}
	receipt, err := scanExecutionReceipt(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return spine.ExecutionReceipt{}, false, nil
		}
		return spine.ExecutionReceipt{}, false, fmt.Errorf("get execution receipt by run: %w", err)
	}
	return receipt, true, nil
}

func scanExecutionReceipt(row pgx.Row) (spine.ExecutionReceipt, error) {
	var receipt spine.ExecutionReceipt
	var id string
	var runID string
	var jobID string
	var leaseID string
	var taskID string
	var checkoutReceiptID string
	var repoBindingID string
	var artifactRefs []byte
	var changedPaths []byte
	var exitCode pgtype.Int4
	if err := row.Scan(
		&id,
		&runID,
		&jobID,
		&leaseID,
		&taskID,
		&checkoutReceiptID,
		&repoBindingID,
		&receipt.RunnerID,
		&receipt.WorkspaceRef,
		&receipt.CommitSHA,
		&receipt.BaselineID,
		&receipt.OverlayID,
		&receipt.ExecutionMode,
		&receipt.ProcessStatus,
		&exitCode,
		&artifactRefs,
		&changedPaths,
		&receipt.RawSourceUploaded,
		&receipt.StartedAt,
		&receipt.FinishedAt,
		&receipt.CreatedAt,
		&receipt.UpdatedAt,
	); err != nil {
		return spine.ExecutionReceipt{}, err
	}
	receipt.ID = spine.ExecutionReceiptID(id)
	receipt.RunID = spine.RunID(runID)
	receipt.ExecutionJobID = spine.ExecutionJobID(jobID)
	receipt.ExecutionLeaseID = spine.ExecutionLeaseID(leaseID)
	receipt.TaskID = spine.WorkItemID(taskID)
	receipt.CheckoutReceiptID = spine.CheckoutReceiptID(checkoutReceiptID)
	receipt.RepoBindingID = spine.RepoBindingID(repoBindingID)
	if exitCode.Valid {
		value := int(exitCode.Int32)
		receipt.ExitCode = &value
	}
	if err := json.Unmarshal(artifactRefs, &receipt.ArtifactRefs); err != nil {
		return spine.ExecutionReceipt{}, fmt.Errorf("unmarshal execution receipt artifact refs: %w", err)
	}
	if err := json.Unmarshal(changedPaths, &receipt.ChangedPathsSummary); err != nil {
		return spine.ExecutionReceipt{}, fmt.Errorf("unmarshal execution receipt changed paths summary: %w", err)
	}
	receipt.StartedAt = receipt.StartedAt.UTC()
	receipt.FinishedAt = receipt.FinishedAt.UTC()
	receipt.CreatedAt = receipt.CreatedAt.UTC()
	receipt.UpdatedAt = receipt.UpdatedAt.UTC()
	receipt.NextAction = spine.ExecutionNextAction{
		Kind:         spine.ExecutionReceiptNextActionGateReview,
		Blocking:     true,
		Available:    false,
		PlannedSlice: spine.ExecutionReceiptNextActionPlannedSlice,
	}
	return receipt, nil
}

func executionReceiptColumns() []string {
	return []string{
		"id",
		"run_id",
		"execution_job_id",
		"execution_lease_id",
		"task_id",
		"checkout_receipt_id",
		"repo_binding_id",
		"runner_id",
		"workspace_ref",
		"commit_sha",
		"baseline_id",
		"overlay_id",
		"execution_mode",
		"process_status",
		"exit_code",
		"artifact_refs",
		"changed_paths_summary",
		"raw_source_uploaded",
		"started_at",
		"finished_at",
		"created_at",
		"updated_at",
	}
}
