package execution

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

const (
	EventTypeExecutionJobCreated       = "execution_job.created"
	EventTypeExecutionJobLeased        = "execution_job.leased"
	EventTypeRunStarted                = "run.started"
	EventTypeExecutionReceiptSubmitted = "execution_receipt.submitted"

	EntityTypeExecutionJob     = "ExecutionJob"
	EntityTypeRun              = "Run"
	EntityTypeExecutionReceipt = "ExecutionReceipt"

	ExecutionModePrepareV0 = "prepare_v0"

	defaultLeaseTTL = 15 * time.Minute
	minLeaseTTL     = 30 * time.Second
	maxLeaseTTL     = 60 * time.Minute
)

var (
	ErrWorkItemNotFound           = errors.New("work item not found")
	ErrRepoBindingNotFound        = errors.New("repo binding not found")
	ErrCheckoutReceiptNotFound    = errors.New("checkout receipt not found")
	ErrCheckoutJobNotFound        = errors.New("checkout job not found")
	ErrExecutionJobNotFound       = errors.New("execution job not found")
	ErrRunNotFound                = errors.New("run not found")
	ErrInvalidWorkItemState       = errors.New("work item state does not allow execution preparation")
	ErrInvalidCheckoutState       = errors.New("checkout job state does not allow execution preparation")
	ErrInvalidExecutionState      = errors.New("execution job state does not allow this transition")
	ErrInvalidRunState            = errors.New("run state does not allow this transition")
	ErrLeaseExpired               = errors.New("execution job lease expired")
	ErrInvalidLease               = errors.New("execution job lease is invalid")
	ErrMembershipRequired         = errors.New("active organization membership is required")
	ErrOrganizationForbidden      = errors.New("user is not allowed to prepare execution for this work item")
	ErrProjectMismatch            = errors.New("execution project expectation does not match work item")
	ErrRepoBindingMismatch        = errors.New("execution repo binding expectation does not match work item")
	ErrCheckoutReceiptMismatch    = errors.New("checkout receipt does not match work item")
	ErrRawSourceUploaded          = errors.New("checkout receipt must not upload raw source")
	ErrExecutionRawSourceUploaded = errors.New("execution receipt must not upload raw source")
	ErrRunAlreadyStarted          = errors.New("execution job already has run")
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Field == "" {
		return e.Message
	}
	return e.Field + ": " + e.Message
}

type WorkItemReader interface {
	Get(context.Context, spine.WorkItemID) (spine.WorkItem, bool, error)
}

type RepoBindingReader interface {
	GetRepoBinding(context.Context, spine.RepoBindingID) (spine.RepoBinding, bool, error)
}

type CheckoutReceiptReader interface {
	Get(context.Context, spine.CheckoutReceiptID) (spine.CheckoutReceipt, bool, error)
}

type CheckoutJobReader interface {
	Get(context.Context, spine.CheckoutJobID) (spine.CheckoutJob, bool, error)
}

type JobLeaseInput struct {
	ID             spine.ExecutionLeaseID
	OrganizationID spine.OrganizationID
	ProjectID      spine.ProjectID
	RepoBindingID  spine.RepoBindingID
	RunnerID       string
	LeaseTokenHash string
	ExpiresAt      time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type JobStore interface {
	Create(context.Context, spine.ExecutionJob) error
	Get(context.Context, spine.ExecutionJobID) (spine.ExecutionJob, bool, error)
	GetByTaskAndCheckoutReceipt(context.Context, spine.WorkItemID, spine.CheckoutReceiptID) (spine.ExecutionJob, bool, error)
	AcquireNextLease(context.Context, JobLeaseInput) (spine.ExecutionLease, spine.ExecutionJob, bool, error)
	MarkRunStarted(context.Context, spine.ExecutionJobID, spine.ExecutionLeaseID, string, string, time.Time) (bool, error)
	MarkReceiptSubmitted(context.Context, spine.ExecutionJobID, time.Time) (bool, error)
}

type RunStore interface {
	Create(context.Context, spine.Run) error
	Get(context.Context, spine.RunID) (spine.Run, bool, error)
	GetByExecutionLease(context.Context, spine.ExecutionLeaseID) (spine.Run, bool, error)
	MarkReceiptSubmitted(context.Context, spine.RunID, time.Time, time.Time) (bool, error)
}

type ExecutionReceiptStore interface {
	Create(context.Context, spine.ExecutionReceipt) error
	GetByRun(context.Context, spine.RunID) (spine.ExecutionReceipt, bool, error)
}

type EventLog interface {
	Append(context.Context, spine.Event) error
}

type TransactionRunner interface {
	RunReadCommitted(context.Context, func(context.Context) error) error
}

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	NewExecutionJobID() (spine.ExecutionJobID, error)
	NewExecutionLeaseID() (spine.ExecutionLeaseID, error)
	NewRunID() (spine.RunID, error)
	NewExecutionReceiptID() (spine.ExecutionReceiptID, error)
	NewEventID() (spine.EventID, error)
}

type Service struct {
	WorkItems         WorkItemReader
	RepoBindings      RepoBindingReader
	CheckoutReceipts  CheckoutReceiptReader
	CheckoutJobs      CheckoutJobReader
	Jobs              JobStore
	Runs              RunStore
	ExecutionReceipts ExecutionReceiptStore
	Events            EventLog
	TxRunner          TransactionRunner
	Clock             Clock
	IDs               IDGenerator
}

func NewService(workItems WorkItemReader, repoBindings RepoBindingReader, checkoutReceipts CheckoutReceiptReader, checkoutJobs CheckoutJobReader, jobs JobStore, runs RunStore, executionReceipts ExecutionReceiptStore, events EventLog, txRunner TransactionRunner, clock Clock, ids IDGenerator) *Service {
	return &Service{
		WorkItems:         workItems,
		RepoBindings:      repoBindings,
		CheckoutReceipts:  checkoutReceipts,
		CheckoutJobs:      checkoutJobs,
		Jobs:              jobs,
		Runs:              runs,
		ExecutionReceipts: executionReceipts,
		Events:            events,
		TxRunner:          txRunner,
		Clock:             clock,
		IDs:               ids,
	}
}

func (s *Service) CreateOrReturnJob(ctx context.Context, taskID spine.WorkItemID, input spine.ExecutionJobCreateRequest, membership spine.OrganizationMembership) (spine.ExecutionJob, bool, error) {
	if err := validateCreateInput(input); err != nil {
		return spine.ExecutionJob{}, false, err
	}
	task, receipt, checkoutJob, err := s.loadAuthorizedContext(ctx, taskID, input, membership)
	if err != nil {
		return spine.ExecutionJob{}, false, err
	}
	if existing, ok, err := s.Jobs.GetByTaskAndCheckoutReceipt(ctx, task.ID, receipt.ID); err != nil {
		return spine.ExecutionJob{}, false, fmt.Errorf("get execution job by task and checkout receipt: %w", err)
	} else if ok {
		return existing, false, nil
	}

	jobID, err := s.IDs.NewExecutionJobID()
	if err != nil {
		return spine.ExecutionJob{}, false, fmt.Errorf("new execution job id: %w", err)
	}
	now := s.Clock.Now().UTC()
	job := spine.ExecutionJob{
		ID:                 jobID,
		OrganizationID:     task.OrganizationID,
		ProjectID:          task.ProjectID,
		TaskID:             task.ID,
		ContractID:         task.ContractID,
		ApprovedContractID: task.ApprovedContractID,
		PlanID:             task.PlanID,
		ProposalID:         task.ProposalID,
		RepoBindingID:      task.RepoBindingID,
		CheckoutJobID:      checkoutJob.ID,
		CheckoutReceiptID:  receipt.ID,
		State:              spine.ExecutionJobStateQueued,
		RequestedBy:        input.RequestedBy,
		ExecutionMode:      ExecutionModePrepareV0,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	event, err := s.event(EventTypeExecutionJobCreated, EntityTypeExecutionJob, string(job.ID), job.OrganizationID, job.ProjectID, job.RepoBindingID, now, map[string]any{
		"execution_job_id":    job.ID,
		"task_id":             job.TaskID,
		"checkout_job_id":     job.CheckoutJobID,
		"checkout_receipt_id": job.CheckoutReceiptID,
		"repo_binding_id":     job.RepoBindingID,
		"requested_by":        job.RequestedBy,
	})
	if err != nil {
		return spine.ExecutionJob{}, false, err
	}
	if err := s.TxRunner.RunReadCommitted(ctx, func(txCtx context.Context) error {
		if err := s.Jobs.Create(txCtx, job); err != nil {
			return fmt.Errorf("create execution job: %w", err)
		}
		if err := s.Events.Append(txCtx, event); err != nil {
			return fmt.Errorf("append execution job created event: %w", err)
		}
		return nil
	}); err != nil {
		if existing, ok, lookupErr := s.Jobs.GetByTaskAndCheckoutReceipt(ctx, task.ID, receipt.ID); lookupErr != nil {
			return spine.ExecutionJob{}, false, fmt.Errorf("get execution job by task and checkout receipt after create failure: %w", lookupErr)
		} else if ok {
			return existing, false, nil
		}
		return spine.ExecutionJob{}, false, err
	}
	return job, true, nil
}

func (s *Service) AcquireNextLease(ctx context.Context, input spine.ExecutionJobLeaseCreateRequest, membership spine.OrganizationMembership) (spine.ExecutionJobLeaseCreated, bool, error) {
	if err := requireActiveMembership(membership); err != nil {
		return spine.ExecutionJobLeaseCreated{}, false, err
	}
	runnerID := strings.TrimSpace(input.RunnerID)
	if runnerID == "" {
		return spine.ExecutionJobLeaseCreated{}, false, &ValidationError{Field: "runner_id", Message: "is required"}
	}
	projectID := input.ProjectID
	if strings.TrimSpace(string(projectID)) == "" {
		return spine.ExecutionJobLeaseCreated{}, false, &ValidationError{Field: "project_id", Message: "is required"}
	}
	repoBindingID := input.RepoBindingID
	if strings.TrimSpace(string(repoBindingID)) == "" {
		return spine.ExecutionJobLeaseCreated{}, false, &ValidationError{Field: "repo_binding_id", Message: "is required"}
	}
	if err := s.validateLeaseScope(ctx, projectID, repoBindingID, membership); err != nil {
		return spine.ExecutionJobLeaseCreated{}, false, err
	}
	ttl, err := leaseTTL(input.TTLSeconds)
	if err != nil {
		return spine.ExecutionJobLeaseCreated{}, false, err
	}
	leaseID, err := s.IDs.NewExecutionLeaseID()
	if err != nil {
		return spine.ExecutionJobLeaseCreated{}, false, fmt.Errorf("new execution lease id: %w", err)
	}
	token, err := newLeaseToken()
	if err != nil {
		return spine.ExecutionJobLeaseCreated{}, false, fmt.Errorf("new execution lease token: %w", err)
	}
	now := s.Clock.Now().UTC()
	leaseInput := JobLeaseInput{
		ID:             leaseID,
		OrganizationID: membership.OrganizationID,
		ProjectID:      projectID,
		RepoBindingID:  repoBindingID,
		RunnerID:       runnerID,
		LeaseTokenHash: leaseTokenHash(token),
		ExpiresAt:      now.Add(ttl),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	var lease spine.ExecutionLease
	var job spine.ExecutionJob
	var ok bool
	if err := s.TxRunner.RunReadCommitted(ctx, func(txCtx context.Context) error {
		var acquireErr error
		lease, job, ok, acquireErr = s.Jobs.AcquireNextLease(txCtx, leaseInput)
		if acquireErr != nil {
			return fmt.Errorf("acquire execution job lease: %w", acquireErr)
		}
		if !ok {
			return nil
		}
		event, err := s.event(EventTypeExecutionJobLeased, EntityTypeExecutionJob, string(job.ID), job.OrganizationID, job.ProjectID, job.RepoBindingID, now, map[string]any{
			"execution_job_id":    job.ID,
			"execution_lease_id":  lease.ID,
			"task_id":             job.TaskID,
			"checkout_receipt_id": job.CheckoutReceiptID,
			"repo_binding_id":     job.RepoBindingID,
			"runner_id":           runnerID,
		})
		if err != nil {
			return err
		}
		if err := s.Events.Append(txCtx, event); err != nil {
			return fmt.Errorf("append execution job leased event: %w", err)
		}
		return nil
	}); err != nil {
		return spine.ExecutionJobLeaseCreated{}, false, err
	}
	if !ok {
		return spine.ExecutionJobLeaseCreated{}, false, nil
	}
	return spine.ExecutionJobLeaseCreated{
		ID:                lease.ID,
		ExecutionJobID:    lease.ExecutionJobID,
		TaskID:            lease.TaskID,
		CheckoutReceiptID: lease.CheckoutReceiptID,
		RepoBindingID:     lease.RepoBindingID,
		RunnerID:          lease.RunnerID,
		State:             lease.State,
		LeaseToken:        token,
		ExpiresAt:         lease.ExpiresAt,
		ExecutionJob:      job,
		CreatedAt:         lease.CreatedAt,
		UpdatedAt:         lease.UpdatedAt,
	}, true, nil
}

func (s *Service) StartRun(ctx context.Context, jobID spine.ExecutionJobID, input spine.RunStartRequest, membership spine.OrganizationMembership) (spine.Run, bool, error) {
	if err := validateRunStartInput(input); err != nil {
		return spine.Run{}, false, err
	}
	job, ok, err := s.Jobs.Get(ctx, jobID)
	if err != nil {
		return spine.Run{}, false, fmt.Errorf("get execution job: %w", err)
	}
	if !ok {
		return spine.Run{}, false, ErrExecutionJobNotFound
	}
	if err := authorizeTaskAccess(membership, job.OrganizationID); err != nil {
		return spine.Run{}, false, err
	}
	now := s.Clock.Now().UTC()
	if existing, ok, err := s.Runs.GetByExecutionLease(ctx, input.LeaseID); err != nil {
		return spine.Run{}, false, fmt.Errorf("get run by execution lease: %w", err)
	} else if ok {
		if existing.ExecutionJobID != job.ID {
			return spine.Run{}, false, ErrInvalidLease
		}
		if err := validateCurrentLeaseProof(job, input.LeaseID, input.RunnerID, leaseTokenHash(input.LeaseToken), now); err != nil {
			return spine.Run{}, false, err
		}
		return existing, false, nil
	}
	if job.State != spine.ExecutionJobStateLeased {
		if job.State == spine.ExecutionJobStateRunStarted {
			return spine.Run{}, false, ErrRunAlreadyStarted
		}
		return spine.Run{}, false, fmt.Errorf("%w: %s", ErrInvalidExecutionState, job.State)
	}
	if err := validateLeaseProof(job, input.LeaseID, input.RunnerID, leaseTokenHash(input.LeaseToken), now); err != nil {
		return spine.Run{}, false, err
	}
	runID, err := s.IDs.NewRunID()
	if err != nil {
		return spine.Run{}, false, fmt.Errorf("new run id: %w", err)
	}
	run := spine.Run{
		ID:                runID,
		ExecutionJobID:    job.ID,
		ExecutionLeaseID:  input.LeaseID,
		TaskID:            job.TaskID,
		CheckoutReceiptID: job.CheckoutReceiptID,
		RunnerID:          strings.TrimSpace(input.RunnerID),
		State:             spine.RunStateStarted,
		StartedAt:         now,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	event, err := s.event(EventTypeRunStarted, EntityTypeRun, string(run.ID), job.OrganizationID, job.ProjectID, job.RepoBindingID, now, map[string]any{
		"run_id":              run.ID,
		"execution_job_id":    run.ExecutionJobID,
		"execution_lease_id":  run.ExecutionLeaseID,
		"task_id":             run.TaskID,
		"checkout_receipt_id": run.CheckoutReceiptID,
		"runner_id":           run.RunnerID,
	})
	if err != nil {
		return spine.Run{}, false, err
	}
	if err := s.TxRunner.RunReadCommitted(ctx, func(txCtx context.Context) error {
		if err := s.Runs.Create(txCtx, run); err != nil {
			return fmt.Errorf("create run: %w", err)
		}
		updated, err := s.Jobs.MarkRunStarted(txCtx, job.ID, input.LeaseID, run.RunnerID, leaseTokenHash(input.LeaseToken), now)
		if err != nil {
			return fmt.Errorf("mark execution job run started: %w", err)
		}
		if !updated {
			return ErrInvalidLease
		}
		if err := s.Events.Append(txCtx, event); err != nil {
			return fmt.Errorf("append run started event: %w", err)
		}
		return nil
	}); err != nil {
		if existing, ok, lookupErr := s.Runs.GetByExecutionLease(ctx, input.LeaseID); lookupErr != nil {
			return spine.Run{}, false, fmt.Errorf("get run by execution lease after start failure: %w", lookupErr)
		} else if ok {
			return existing, false, nil
		}
		return spine.Run{}, false, err
	}
	return run, true, nil
}

func (s *Service) SubmitReceipt(ctx context.Context, runID spine.RunID, input spine.ExecutionReceiptSubmitRequest, membership spine.OrganizationMembership) (spine.ExecutionReceipt, bool, error) {
	if err := validateReceiptInput(input); err != nil {
		return spine.ExecutionReceipt{}, false, err
	}
	run, ok, err := s.Runs.Get(ctx, runID)
	if err != nil {
		return spine.ExecutionReceipt{}, false, fmt.Errorf("get run: %w", err)
	}
	if !ok {
		return spine.ExecutionReceipt{}, false, ErrRunNotFound
	}
	if input.ExecutionJobID != run.ExecutionJobID {
		return spine.ExecutionReceipt{}, false, ErrInvalidLease
	}
	job, ok, err := s.Jobs.Get(ctx, run.ExecutionJobID)
	if err != nil {
		return spine.ExecutionReceipt{}, false, fmt.Errorf("get execution job: %w", err)
	}
	if !ok {
		return spine.ExecutionReceipt{}, false, ErrExecutionJobNotFound
	}
	if err := authorizeTaskAccess(membership, job.OrganizationID); err != nil {
		return spine.ExecutionReceipt{}, false, err
	}
	now := s.Clock.Now().UTC()
	if err := validateCurrentLeaseProof(job, run.ExecutionLeaseID, input.RunnerID, leaseTokenHash(input.LeaseToken), now); err != nil {
		return spine.ExecutionReceipt{}, false, err
	}
	if existing, ok, err := s.ExecutionReceipts.GetByRun(ctx, run.ID); err != nil {
		return spine.ExecutionReceipt{}, false, fmt.Errorf("get execution receipt by run: %w", err)
	} else if ok {
		return existing, false, nil
	}
	if run.State != spine.RunStateStarted {
		return spine.ExecutionReceipt{}, false, fmt.Errorf("%w: %s", ErrInvalidRunState, run.State)
	}
	if job.State != spine.ExecutionJobStateRunStarted {
		return spine.ExecutionReceipt{}, false, fmt.Errorf("%w: %s", ErrInvalidExecutionState, job.State)
	}
	receiptID, err := s.IDs.NewExecutionReceiptID()
	if err != nil {
		return spine.ExecutionReceipt{}, false, fmt.Errorf("new execution receipt id: %w", err)
	}
	receipt := spine.ExecutionReceipt{
		ID:                  receiptID,
		RunID:               run.ID,
		ExecutionJobID:      run.ExecutionJobID,
		ExecutionLeaseID:    run.ExecutionLeaseID,
		TaskID:              run.TaskID,
		CheckoutReceiptID:   run.CheckoutReceiptID,
		RepoBindingID:       job.RepoBindingID,
		RunnerID:            strings.TrimSpace(input.RunnerID),
		WorkspaceRef:        strings.TrimSpace(input.WorkspaceRef),
		CommitSHA:           strings.TrimSpace(input.CommitSHA),
		BaselineID:          strings.TrimSpace(input.BaselineID),
		OverlayID:           strings.TrimSpace(input.OverlayID),
		ExecutionMode:       spine.ExecutionReceiptModeNoCommand,
		ProcessStatus:       strings.TrimSpace(input.ProcessStatus),
		ArtifactRefs:        append([]string{}, input.ArtifactRefs...),
		ChangedPathsSummary: append([]string{}, input.ChangedPathsSummary...),
		RawSourceUploaded:   false,
		StartedAt:           run.StartedAt.UTC(),
		FinishedAt:          now,
		CreatedAt:           now,
		UpdatedAt:           now,
		NextAction: spine.ExecutionNextAction{
			Kind:         spine.ExecutionReceiptNextActionGateReview,
			Blocking:     true,
			Available:    false,
			PlannedSlice: spine.ExecutionReceiptNextActionPlannedSlice,
		},
	}
	event, err := s.event(EventTypeExecutionReceiptSubmitted, EntityTypeExecutionReceipt, string(receipt.ID), job.OrganizationID, job.ProjectID, job.RepoBindingID, now, map[string]any{
		"execution_receipt_id": receipt.ID,
		"run_id":               receipt.RunID,
		"execution_job_id":     receipt.ExecutionJobID,
		"task_id":              receipt.TaskID,
		"checkout_receipt_id":  receipt.CheckoutReceiptID,
		"repo_binding_id":      receipt.RepoBindingID,
		"runner_id":            receipt.RunnerID,
		"execution_mode":       receipt.ExecutionMode,
		"process_status":       receipt.ProcessStatus,
	})
	if err != nil {
		return spine.ExecutionReceipt{}, false, err
	}
	if err := s.TxRunner.RunReadCommitted(ctx, func(txCtx context.Context) error {
		if err := s.ExecutionReceipts.Create(txCtx, receipt); err != nil {
			return fmt.Errorf("create execution receipt: %w", err)
		}
		runUpdated, err := s.Runs.MarkReceiptSubmitted(txCtx, run.ID, receipt.FinishedAt, now)
		if err != nil {
			return fmt.Errorf("mark run receipt submitted: %w", err)
		}
		if !runUpdated {
			return ErrInvalidRunState
		}
		jobUpdated, err := s.Jobs.MarkReceiptSubmitted(txCtx, job.ID, now)
		if err != nil {
			return fmt.Errorf("mark execution job receipt submitted: %w", err)
		}
		if !jobUpdated {
			return ErrInvalidExecutionState
		}
		if err := s.Events.Append(txCtx, event); err != nil {
			return fmt.Errorf("append execution receipt submitted event: %w", err)
		}
		return nil
	}); err != nil {
		if existing, ok, lookupErr := s.ExecutionReceipts.GetByRun(ctx, run.ID); lookupErr != nil {
			return spine.ExecutionReceipt{}, false, fmt.Errorf("get execution receipt by run after submit failure: %w", lookupErr)
		} else if ok {
			return existing, false, nil
		}
		return spine.ExecutionReceipt{}, false, err
	}
	return receipt, true, nil
}

func (s *Service) validateLeaseScope(ctx context.Context, projectID spine.ProjectID, repoBindingID spine.RepoBindingID, membership spine.OrganizationMembership) error {
	binding, ok, err := s.RepoBindings.GetRepoBinding(ctx, repoBindingID)
	if err != nil {
		return fmt.Errorf("get execution lease repo binding: %w", err)
	}
	if !ok || binding.State != spine.EntityStateActive {
		return ErrRepoBindingNotFound
	}
	if binding.OrganizationID != membership.OrganizationID {
		return ErrOrganizationForbidden
	}
	if binding.ProjectID != projectID {
		return ErrProjectMismatch
	}
	return nil
}

func (s *Service) loadAuthorizedContext(ctx context.Context, taskID spine.WorkItemID, input spine.ExecutionJobCreateRequest, membership spine.OrganizationMembership) (spine.WorkItem, spine.CheckoutReceipt, spine.CheckoutJob, error) {
	task, ok, err := s.WorkItems.Get(ctx, taskID)
	if err != nil {
		return spine.WorkItem{}, spine.CheckoutReceipt{}, spine.CheckoutJob{}, fmt.Errorf("get work item: %w", err)
	}
	if !ok {
		return spine.WorkItem{}, spine.CheckoutReceipt{}, spine.CheckoutJob{}, ErrWorkItemNotFound
	}
	if task.Status != spine.WorkItemStatusPlanned {
		return spine.WorkItem{}, spine.CheckoutReceipt{}, spine.CheckoutJob{}, fmt.Errorf("%w: %s", ErrInvalidWorkItemState, task.Status)
	}
	if err := authorizeTaskAccess(membership, task.OrganizationID); err != nil {
		return spine.WorkItem{}, spine.CheckoutReceipt{}, spine.CheckoutJob{}, err
	}
	if strings.TrimSpace(string(input.ProjectID)) != "" && input.ProjectID != task.ProjectID {
		return spine.WorkItem{}, spine.CheckoutReceipt{}, spine.CheckoutJob{}, ErrProjectMismatch
	}
	if strings.TrimSpace(string(input.RepoBindingID)) != "" && input.RepoBindingID != task.RepoBindingID {
		return spine.WorkItem{}, spine.CheckoutReceipt{}, spine.CheckoutJob{}, ErrRepoBindingMismatch
	}
	receipt, ok, err := s.CheckoutReceipts.Get(ctx, input.CheckoutReceiptID)
	if err != nil {
		return spine.WorkItem{}, spine.CheckoutReceipt{}, spine.CheckoutJob{}, fmt.Errorf("get checkout receipt: %w", err)
	}
	if !ok {
		return spine.WorkItem{}, spine.CheckoutReceipt{}, spine.CheckoutJob{}, ErrCheckoutReceiptNotFound
	}
	if receipt.RawSourceUploaded {
		return spine.WorkItem{}, spine.CheckoutReceipt{}, spine.CheckoutJob{}, ErrRawSourceUploaded
	}
	if receipt.TaskID != task.ID {
		return spine.WorkItem{}, spine.CheckoutReceipt{}, spine.CheckoutJob{}, ErrCheckoutReceiptMismatch
	}
	if receipt.RepoBindingID != task.RepoBindingID {
		return spine.WorkItem{}, spine.CheckoutReceipt{}, spine.CheckoutJob{}, ErrRepoBindingMismatch
	}
	checkoutJob, ok, err := s.CheckoutJobs.Get(ctx, receipt.JobID)
	if err != nil {
		return spine.WorkItem{}, spine.CheckoutReceipt{}, spine.CheckoutJob{}, fmt.Errorf("get checkout job: %w", err)
	}
	if !ok {
		return spine.WorkItem{}, spine.CheckoutReceipt{}, spine.CheckoutJob{}, ErrCheckoutJobNotFound
	}
	if checkoutJob.State != spine.CheckoutJobStateReceiptSubmitted {
		return spine.WorkItem{}, spine.CheckoutReceipt{}, spine.CheckoutJob{}, fmt.Errorf("%w: %s", ErrInvalidCheckoutState, checkoutJob.State)
	}
	if checkoutJob.OrganizationID != task.OrganizationID || checkoutJob.ProjectID != task.ProjectID || checkoutJob.TaskID != task.ID || checkoutJob.RepoBindingID != task.RepoBindingID {
		return spine.WorkItem{}, spine.CheckoutReceipt{}, spine.CheckoutJob{}, ErrCheckoutReceiptMismatch
	}
	if checkoutJob.ID != receipt.JobID {
		return spine.WorkItem{}, spine.CheckoutReceipt{}, spine.CheckoutJob{}, ErrCheckoutReceiptMismatch
	}
	return task, receipt, checkoutJob, nil
}

func validateCreateInput(input spine.ExecutionJobCreateRequest) error {
	if strings.TrimSpace(string(input.CheckoutReceiptID)) == "" {
		return &ValidationError{Field: "checkout_receipt_id", Message: "is required"}
	}
	if err := validateActor("requested_by", input.RequestedBy); err != nil {
		return err
	}
	return nil
}

func validateRunStartInput(input spine.RunStartRequest) error {
	if strings.TrimSpace(string(input.LeaseID)) == "" {
		return &ValidationError{Field: "lease_id", Message: "is required"}
	}
	if strings.TrimSpace(input.LeaseToken) == "" {
		return &ValidationError{Field: "lease_token", Message: "is required"}
	}
	if strings.TrimSpace(input.RunnerID) == "" {
		return &ValidationError{Field: "runner_id", Message: "is required"}
	}
	return nil
}

func validateReceiptInput(input spine.ExecutionReceiptSubmitRequest) error {
	if strings.TrimSpace(string(input.ExecutionJobID)) == "" {
		return &ValidationError{Field: "execution_job_id", Message: "is required"}
	}
	if strings.TrimSpace(input.LeaseToken) == "" {
		return &ValidationError{Field: "lease_token", Message: "is required"}
	}
	if strings.TrimSpace(input.RunnerID) == "" {
		return &ValidationError{Field: "runner_id", Message: "is required"}
	}
	if strings.TrimSpace(input.WorkspaceRef) == "" {
		return &ValidationError{Field: "workspace_ref", Message: "is required"}
	}
	if strings.TrimSpace(input.CommitSHA) == "" {
		return &ValidationError{Field: "commit_sha", Message: "is required"}
	}
	if strings.TrimSpace(input.ExecutionMode) != spine.ExecutionReceiptModeNoCommand {
		return &ValidationError{Field: "execution_mode", Message: "must be no_command"}
	}
	switch strings.TrimSpace(input.ProcessStatus) {
	case spine.ExecutionReceiptStatusNotExecuted, spine.ExecutionReceiptStatusMetadataOnly:
	default:
		return &ValidationError{Field: "process_status", Message: "must be not_executed or metadata_only"}
	}
	if input.ExitCode != nil {
		return &ValidationError{Field: "exit_code", Message: "must be omitted for no_command receipts"}
	}
	if len(input.ArtifactRefs) != 0 {
		return &ValidationError{Field: "artifact_refs", Message: "must be empty for no_command receipts"}
	}
	if len(input.ChangedPathsSummary) != 0 {
		return &ValidationError{Field: "changed_paths_summary", Message: "must be empty for no_command receipts"}
	}
	if input.RawSourceUploaded {
		return ErrExecutionRawSourceUploaded
	}
	return nil
}

func validateActor(field string, actor spine.ActorRef) error {
	if strings.TrimSpace(actor.Kind) == "" {
		return &ValidationError{Field: field + ".kind", Message: "is required"}
	}
	if strings.TrimSpace(actor.ID) == "" {
		return &ValidationError{Field: field + ".id", Message: "is required"}
	}
	return nil
}

func authorizeTaskAccess(membership spine.OrganizationMembership, organizationID spine.OrganizationID) error {
	if err := requireActiveMembership(membership); err != nil {
		return err
	}
	if membership.OrganizationID != organizationID {
		return ErrOrganizationForbidden
	}
	return nil
}

func requireActiveMembership(membership spine.OrganizationMembership) error {
	if membership.State != spine.EntityStateActive || strings.TrimSpace(string(membership.OrganizationID)) == "" {
		return ErrMembershipRequired
	}
	return nil
}

func validateLeaseProof(job spine.ExecutionJob, leaseID spine.ExecutionLeaseID, runnerID string, tokenHash string, now time.Time) error {
	if job.State != spine.ExecutionJobStateLeased {
		return fmt.Errorf("%w: %s", ErrInvalidExecutionState, job.State)
	}
	return validateCurrentLeaseProof(job, leaseID, runnerID, tokenHash, now)
}

func validateCurrentLeaseProof(job spine.ExecutionJob, leaseID spine.ExecutionLeaseID, runnerID string, tokenHash string, now time.Time) error {
	if job.CurrentLeaseID == nil || *job.CurrentLeaseID != leaseID {
		return ErrInvalidLease
	}
	if strings.TrimSpace(job.CurrentRunnerID) != strings.TrimSpace(runnerID) {
		return ErrInvalidLease
	}
	if strings.TrimSpace(job.LeaseTokenHash) == "" || job.LeaseTokenHash != tokenHash {
		return ErrInvalidLease
	}
	if job.LeaseExpiresAt == nil || !job.LeaseExpiresAt.After(now) {
		return ErrLeaseExpired
	}
	return nil
}

func leaseTTL(seconds int) (time.Duration, error) {
	if seconds == 0 {
		return defaultLeaseTTL, nil
	}
	ttl := time.Duration(seconds) * time.Second
	if ttl < minLeaseTTL {
		return 0, &ValidationError{Field: "ttl_seconds", Message: "must be at least 30"}
	}
	if ttl > maxLeaseTTL {
		return 0, &ValidationError{Field: "ttl_seconds", Message: "must be at most 3600"}
	}
	return ttl, nil
}

func newLeaseToken() (string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw[:]), nil
}

func leaseTokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (s *Service) event(eventType string, entityType string, entityID string, organizationID spine.OrganizationID, projectID spine.ProjectID, repoBindingID spine.RepoBindingID, occurredAt time.Time, payload any) (spine.Event, error) {
	eventID, err := s.IDs.NewEventID()
	if err != nil {
		return spine.Event{}, fmt.Errorf("new event id: %w", err)
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return spine.Event{}, fmt.Errorf("marshal event payload: %w", err)
	}
	return spine.Event{
		ID:             eventID,
		Type:           eventType,
		EntityType:     entityType,
		EntityID:       entityID,
		OrganizationID: organizationID,
		ProjectID:      projectID,
		RepoBindingID:  repoBindingID,
		Timestamp:      occurredAt.UTC(),
		Payload:        raw,
	}, nil
}

type SystemClock struct{}

func (SystemClock) Now() time.Time {
	return time.Now()
}

type UUIDGenerator struct{}

func (UUIDGenerator) NewExecutionJobID() (spine.ExecutionJobID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return spine.ExecutionJobID(id.String()), nil
}

func (UUIDGenerator) NewExecutionLeaseID() (spine.ExecutionLeaseID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return spine.ExecutionLeaseID(id.String()), nil
}

func (UUIDGenerator) NewRunID() (spine.RunID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return spine.RunID(id.String()), nil
}

func (UUIDGenerator) NewExecutionReceiptID() (spine.ExecutionReceiptID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return spine.ExecutionReceiptID(id.String()), nil
}

func (UUIDGenerator) NewEventID() (spine.EventID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return spine.EventID(id.String()), nil
}
