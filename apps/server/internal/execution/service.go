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
	EventTypeExecutionJobCreated         = "execution_job.created"
	EventTypeExecutionJobLeased          = "execution_job.leased"
	EventTypeRunStarted                  = "run.started"
	EventTypeExecutionCommandPlanCreated = "execution_command_plan.created"
	EventTypeExecutionReceiptSubmitted   = "execution_receipt.submitted"

	EntityTypeExecutionJob         = "ExecutionJob"
	EntityTypeRun                  = "Run"
	EntityTypeExecutionCommandPlan = "ExecutionCommandPlan"
	EntityTypeExecutionReceipt     = "ExecutionReceipt"

	ExecutionModePrepareV0 = "prepare_v0"

	defaultLeaseTTL = 15 * time.Minute
	minLeaseTTL     = 30 * time.Second
	maxLeaseTTL     = 60 * time.Minute
)

var (
	ErrWorkItemNotFound             = errors.New("work item not found")
	ErrRepoBindingNotFound          = errors.New("repo binding not found")
	ErrCheckoutReceiptNotFound      = errors.New("checkout receipt not found")
	ErrCheckoutJobNotFound          = errors.New("checkout job not found")
	ErrExecutionJobNotFound         = errors.New("execution job not found")
	ErrRunNotFound                  = errors.New("run not found")
	ErrExecutionCommandPlanNotFound = errors.New("execution command plan not found")
	ErrExecutionReceiptNotFound     = errors.New("execution receipt not found")
	ErrInvalidWorkItemState         = errors.New("work item state does not allow execution preparation")
	ErrInvalidCheckoutState         = errors.New("checkout job state does not allow execution preparation")
	ErrInvalidExecutionState        = errors.New("execution job state does not allow this transition")
	ErrInvalidRunState              = errors.New("run state does not allow this transition")
	ErrInvalidCommandPlan           = errors.New("execution command plan is invalid")
	ErrLeaseExpired                 = errors.New("execution job lease expired")
	ErrInvalidLease                 = errors.New("execution job lease is invalid")
	ErrMembershipRequired           = errors.New("active organization membership is required")
	ErrOrganizationForbidden        = errors.New("user is not allowed to prepare execution for this work item")
	ErrProjectMismatch              = errors.New("execution project expectation does not match work item")
	ErrRepoBindingMismatch          = errors.New("execution repo binding expectation does not match work item")
	ErrCheckoutReceiptMismatch      = errors.New("checkout receipt does not match work item")
	ErrRawSourceUploaded            = errors.New("checkout receipt must not upload raw source")
	ErrExecutionRawSourceUploaded   = errors.New("execution receipt must not upload raw source")
	ErrRunAlreadyStarted            = errors.New("execution job already has run")
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
	GetByExecutionJob(context.Context, spine.ExecutionJobID) (spine.Run, bool, error)
	MarkReceiptSubmitted(context.Context, spine.RunID, time.Time, time.Time) (bool, error)
}

type ExecutionCommandPlanStore interface {
	Create(context.Context, spine.ExecutionCommandPlan) error
	Get(context.Context, spine.ExecutionCommandPlanID) (spine.ExecutionCommandPlan, bool, error)
	GetByRunAndAction(context.Context, spine.RunID, string, string) (spine.ExecutionCommandPlan, bool, error)
}

type ExecutionReceiptStore interface {
	Create(context.Context, spine.ExecutionReceipt) error
	Get(context.Context, spine.ExecutionReceiptID) (spine.ExecutionReceipt, bool, error)
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
	NewExecutionCommandPlanID() (spine.ExecutionCommandPlanID, error)
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
	CommandPlans      ExecutionCommandPlanStore
	ExecutionReceipts ExecutionReceiptStore
	Events            EventLog
	TxRunner          TransactionRunner
	Clock             Clock
	IDs               IDGenerator
}

func NewService(workItems WorkItemReader, repoBindings RepoBindingReader, checkoutReceipts CheckoutReceiptReader, checkoutJobs CheckoutJobReader, jobs JobStore, runs RunStore, commandPlans ExecutionCommandPlanStore, executionReceipts ExecutionReceiptStore, events EventLog, txRunner TransactionRunner, clock Clock, ids IDGenerator) *Service {
	return &Service{
		WorkItems:         workItems,
		RepoBindings:      repoBindings,
		CheckoutReceipts:  checkoutReceipts,
		CheckoutJobs:      checkoutJobs,
		Jobs:              jobs,
		Runs:              runs,
		CommandPlans:      commandPlans,
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
			if err := validateCurrentLeaseProof(job, input.LeaseID, input.RunnerID, leaseTokenHash(input.LeaseToken), now); err != nil {
				return spine.Run{}, false, err
			}
			existing, ok, err := s.Runs.GetByExecutionJob(ctx, job.ID)
			if err != nil {
				return spine.Run{}, false, fmt.Errorf("get run by execution job: %w", err)
			}
			if !ok {
				return spine.Run{}, false, ErrRunNotFound
			}
			return existing, false, nil
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

func (s *Service) CreateOrReturnCommandPlan(ctx context.Context, runID spine.RunID, input spine.ExecutionCommandPlanCreateRequest, membership spine.OrganizationMembership) (spine.ExecutionCommandPlan, bool, error) {
	policy, err := commandPlanPolicyForInput(input)
	if err != nil {
		return spine.ExecutionCommandPlan{}, false, err
	}
	run, ok, err := s.Runs.Get(ctx, runID)
	if err != nil {
		return spine.ExecutionCommandPlan{}, false, fmt.Errorf("get run: %w", err)
	}
	if !ok {
		return spine.ExecutionCommandPlan{}, false, ErrRunNotFound
	}
	job, ok, err := s.Jobs.Get(ctx, run.ExecutionJobID)
	if err != nil {
		return spine.ExecutionCommandPlan{}, false, fmt.Errorf("get execution job: %w", err)
	}
	if !ok {
		return spine.ExecutionCommandPlan{}, false, ErrExecutionJobNotFound
	}
	if err := authorizeTaskAccess(membership, job.OrganizationID); err != nil {
		return spine.ExecutionCommandPlan{}, false, err
	}
	if input.ProjectID != job.ProjectID {
		return spine.ExecutionCommandPlan{}, false, ErrProjectMismatch
	}
	if input.RepoBindingID != job.RepoBindingID {
		return spine.ExecutionCommandPlan{}, false, ErrRepoBindingMismatch
	}
	if run.State != spine.RunStateStarted {
		return spine.ExecutionCommandPlan{}, false, fmt.Errorf("%w: %s", ErrInvalidRunState, run.State)
	}
	if job.State != spine.ExecutionJobStateRunStarted {
		return spine.ExecutionCommandPlan{}, false, fmt.Errorf("%w: %s", ErrInvalidExecutionState, job.State)
	}
	policy, err = s.prepareCommandPlanPolicy(ctx, input, run, job, policy)
	if err != nil {
		return spine.ExecutionCommandPlan{}, false, err
	}
	if existing, ok, err := s.CommandPlans.GetByRunAndAction(ctx, run.ID, policy.CommandKind, policy.Action); err != nil {
		return spine.ExecutionCommandPlan{}, false, fmt.Errorf("get execution command plan by run and action: %w", err)
	} else if ok {
		if !commandPlanMatchesPolicy(existing, policy) {
			return spine.ExecutionCommandPlan{}, false, ErrInvalidCommandPlan
		}
		return decorateCommandPlanForResponse(existing), false, nil
	}
	planID, err := s.IDs.NewExecutionCommandPlanID()
	if err != nil {
		return spine.ExecutionCommandPlan{}, false, fmt.Errorf("new execution command plan id: %w", err)
	}
	now := s.Clock.Now().UTC()
	plan := spine.ExecutionCommandPlan{
		ID:                     planID,
		OrganizationID:         job.OrganizationID,
		ProjectID:              job.ProjectID,
		RepoBindingID:          job.RepoBindingID,
		TaskID:                 job.TaskID,
		CheckoutReceiptID:      job.CheckoutReceiptID,
		ExecutionJobID:         job.ID,
		RunID:                  run.ID,
		CommandKind:            policy.CommandKind,
		Action:                 policy.Action,
		ShellAllowed:           false,
		Argv:                   []string{},
		WorkingDirectory:       policy.WorkingDirectory,
		PathScope:              append([]string{}, policy.PathScope...),
		TimeoutSeconds:         policy.TimeoutSeconds,
		MaxStdoutBytes:         policy.MaxStdoutBytes,
		MaxStderrBytes:         policy.MaxStderrBytes,
		NetworkAllowed:         policy.NetworkAllowed,
		WorkspaceWriteAllowed:  policy.WorkspaceWriteAllowed,
		ScratchWriteAllowed:    policy.ScratchWriteAllowed,
		ChangedPathsAllowed:    policy.ChangedPathsAllowed,
		AllowedArtifactKinds:   []string{},
		RawSourceUploadAllowed: false,
		State:                  spine.ExecutionCommandPlanStatePlanned,
		CreatedAt:              now,
		UpdatedAt:              now,
	}
	if policy.SourceProjectProbeReceiptID != nil {
		sourceReceiptID := *policy.SourceProjectProbeReceiptID
		plan.SourceProjectProbeReceiptID = &sourceReceiptID
	}
	if policy.DeclaredTestTarget != nil {
		target := *policy.DeclaredTestTarget
		plan.SelectedTargetID = policy.SelectedTargetID
		plan.DeclaredTestTarget = &target
	}
	eventPayload := map[string]any{
		"execution_command_plan_id": plan.ID,
		"run_id":                    plan.RunID,
		"execution_job_id":          plan.ExecutionJobID,
		"task_id":                   plan.TaskID,
		"checkout_receipt_id":       plan.CheckoutReceiptID,
		"repo_binding_id":           plan.RepoBindingID,
		"command_kind":              plan.CommandKind,
		"action":                    plan.Action,
	}
	if plan.SourceProjectProbeReceiptID != nil {
		eventPayload["source_project_probe_receipt_id"] = *plan.SourceProjectProbeReceiptID
		eventPayload["selected_target_id"] = plan.SelectedTargetID
	}
	event, err := s.event(EventTypeExecutionCommandPlanCreated, EntityTypeExecutionCommandPlan, string(plan.ID), plan.OrganizationID, plan.ProjectID, plan.RepoBindingID, now, eventPayload)
	if err != nil {
		return spine.ExecutionCommandPlan{}, false, err
	}
	if err := s.TxRunner.RunReadCommitted(ctx, func(txCtx context.Context) error {
		if err := s.CommandPlans.Create(txCtx, plan); err != nil {
			return fmt.Errorf("create execution command plan: %w", err)
		}
		if err := s.Events.Append(txCtx, event); err != nil {
			return fmt.Errorf("append execution command plan created event: %w", err)
		}
		return nil
	}); err != nil {
		if existing, ok, lookupErr := s.CommandPlans.GetByRunAndAction(ctx, run.ID, policy.CommandKind, policy.Action); lookupErr != nil {
			return spine.ExecutionCommandPlan{}, false, fmt.Errorf("get execution command plan by run and action after create failure: %w", lookupErr)
		} else if ok {
			if !commandPlanMatchesPolicy(existing, policy) {
				return spine.ExecutionCommandPlan{}, false, ErrInvalidCommandPlan
			}
			return decorateCommandPlanForResponse(existing), false, nil
		}
		return spine.ExecutionCommandPlan{}, false, err
	}
	return decorateCommandPlanForResponse(plan), true, nil
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
	if err := validateCurrentLeaseProof(job, input.LeaseID, input.RunnerID, leaseTokenHash(input.LeaseToken), now); err != nil {
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
	var commandPlanID *spine.ExecutionCommandPlanID
	commandKind := strings.TrimSpace(input.CommandKind)
	action := strings.TrimSpace(input.Action)
	if input.ExecutionMode == spine.ExecutionReceiptModeBuiltinDiagnostic {
		plan, err := s.validateBuiltinDiagnosticReceipt(ctx, input, run, job)
		if err != nil {
			return spine.ExecutionReceipt{}, false, err
		}
		id := plan.ID
		commandPlanID = &id
		commandKind = plan.CommandKind
		action = plan.Action
	}
	var projectProbeMetadata *spine.ProjectProbeMetadata
	if input.ExecutionMode == spine.ExecutionReceiptModeProjectProbe {
		plan, err := s.validateProjectProbeReceipt(ctx, input, run, job)
		if err != nil {
			return spine.ExecutionReceipt{}, false, err
		}
		id := plan.ID
		commandPlanID = &id
		commandKind = plan.CommandKind
		action = plan.Action
		metadata := normalizeProjectProbeMetadata(*input.ProjectProbeMetadata)
		projectProbeMetadata = &metadata
	}
	receiptID, err := s.IDs.NewExecutionReceiptID()
	if err != nil {
		return spine.ExecutionReceipt{}, false, fmt.Errorf("new execution receipt id: %w", err)
	}
	executionMode := strings.TrimSpace(input.ExecutionMode)
	receipt := spine.ExecutionReceipt{
		ID:                   receiptID,
		RunID:                run.ID,
		ExecutionJobID:       run.ExecutionJobID,
		ExecutionLeaseID:     input.LeaseID,
		TaskID:               run.TaskID,
		CheckoutReceiptID:    run.CheckoutReceiptID,
		RepoBindingID:        job.RepoBindingID,
		RunnerID:             strings.TrimSpace(input.RunnerID),
		WorkspaceRef:         strings.TrimSpace(input.WorkspaceRef),
		CommitSHA:            strings.TrimSpace(input.CommitSHA),
		BaselineID:           strings.TrimSpace(input.BaselineID),
		OverlayID:            strings.TrimSpace(input.OverlayID),
		ExecutionMode:        executionMode,
		CommandPlanID:        commandPlanID,
		CommandKind:          commandKind,
		Action:               action,
		ProcessStatus:        strings.TrimSpace(input.ProcessStatus),
		ArtifactRefs:         append([]string{}, input.ArtifactRefs...),
		ChangedPathsSummary:  append([]string{}, input.ChangedPathsSummary...),
		RawSourceUploaded:    false,
		RunnerStartedAt:      utcTimePtr(input.RunnerStartedAt),
		RunnerFinishedAt:     utcTimePtr(input.RunnerFinishedAt),
		ProjectProbeMetadata: projectProbeMetadata,
		StartedAt:            run.StartedAt.UTC(),
		FinishedAt:           now,
		CreatedAt:            now,
		UpdatedAt:            now,
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
		"command_plan_id":      receipt.CommandPlanID,
		"command_kind":         receipt.CommandKind,
		"action":               receipt.Action,
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

type commandPlanPolicy struct {
	CommandKind                 string
	Action                      string
	WorkingDirectory            string
	PathScope                   []string
	TimeoutSeconds              int
	NetworkAllowed              bool
	WorkspaceWriteAllowed       bool
	ScratchWriteAllowed         bool
	MaxStdoutBytes              int
	MaxStderrBytes              int
	ChangedPathsAllowed         bool
	SourceProjectProbeReceiptID *spine.ExecutionReceiptID
	SelectedTargetID            string
	DeclaredTestTarget          *spine.ProjectProbeTestTargetCandidate
}

func commandPlanPolicyForInput(input spine.ExecutionCommandPlanCreateRequest) (commandPlanPolicy, error) {
	if strings.TrimSpace(string(input.ProjectID)) == "" {
		return commandPlanPolicy{}, &ValidationError{Field: "project_id", Message: "is required"}
	}
	if strings.TrimSpace(string(input.RepoBindingID)) == "" {
		return commandPlanPolicy{}, &ValidationError{Field: "repo_binding_id", Message: "is required"}
	}
	if len(input.Shell) != 0 {
		return commandPlanPolicy{}, &ValidationError{Field: "shell", Message: "is server-owned and must be omitted"}
	}
	if len(input.ShellAllowed) != 0 {
		return commandPlanPolicy{}, &ValidationError{Field: "shell_allowed", Message: "is server-owned and must be omitted"}
	}
	if len(input.Argv) != 0 {
		return commandPlanPolicy{}, &ValidationError{Field: "argv", Message: "is server-owned and must be omitted"}
	}
	if strings.TrimSpace(input.Command) != "" {
		return commandPlanPolicy{}, &ValidationError{Field: "command", Message: "is not allowed"}
	}
	if strings.TrimSpace(input.CommandString) != "" {
		return commandPlanPolicy{}, &ValidationError{Field: "command_string", Message: "is not allowed"}
	}
	if strings.TrimSpace(input.UserCommand) != "" {
		return commandPlanPolicy{}, &ValidationError{Field: "user_command", Message: "is not allowed"}
	}
	for _, field := range []struct {
		name  string
		value json.RawMessage
	}{
		{name: "run_all_tests", value: input.RunAllTests},
		{name: "stdout_capture", value: input.StdoutCapture},
		{name: "stderr_capture", value: input.StderrCapture},
		{name: "artifacts_allowed", value: input.ArtifactsAllowed},
		{name: "artifact_refs", value: input.ArtifactRefs},
		{name: "allowed_artifact_kinds", value: input.AllowedArtifactKinds},
		{name: "changed_paths_allowed", value: input.ChangedPathsAllowed},
		{name: "changed_paths_summary", value: input.ChangedPathsSummary},
		{name: "raw_source_upload", value: input.RawSourceUpload},
		{name: "raw_source_uploaded", value: input.RawSourceUploaded},
		{name: "raw_source_upload_allowed", value: input.RawSourceAllowed},
		{name: "network_allowed", value: input.NetworkAllowed},
		{name: "write_allowed", value: input.WriteAllowed},
		{name: "workspace_write_allowed", value: input.WorkspaceWriteAllowed},
		{name: "scratch_write_allowed", value: input.ScratchWriteAllowed},
	} {
		if len(field.value) != 0 {
			return commandPlanPolicy{}, &ValidationError{Field: field.name, Message: "is server-owned and must be omitted"}
		}
	}
	kind := strings.TrimSpace(input.CommandKind)
	action := strings.TrimSpace(input.Action)
	if kind == "" && action == "" {
		kind = spine.ExecutionCommandKindBuiltinDiagnostic
		action = spine.ExecutionCommandActionWorkspaceStatus
	}
	switch {
	case kind == spine.ExecutionCommandKindBuiltinDiagnostic && action == spine.ExecutionCommandActionWorkspaceStatus:
		if strings.TrimSpace(string(input.ProjectProbeReceiptID)) != "" {
			return commandPlanPolicy{}, &ValidationError{Field: "project_probe_receipt_id", Message: "must be omitted for builtin_diagnostic command plans"}
		}
		if strings.TrimSpace(input.SelectedTargetID) != "" {
			return commandPlanPolicy{}, &ValidationError{Field: "selected_target_id", Message: "must be omitted for builtin_diagnostic command plans"}
		}
		return commandPlanPolicy{
			CommandKind:      spine.ExecutionCommandKindBuiltinDiagnostic,
			Action:           spine.ExecutionCommandActionWorkspaceStatus,
			WorkingDirectory: ".",
			PathScope:        []string{"."},
			TimeoutSeconds:   30,
			MaxStdoutBytes:   0,
			MaxStderrBytes:   0,
		}, nil
	case kind == spine.ExecutionCommandKindProjectProbe && action == spine.ExecutionCommandActionDetectTestTargets:
		if strings.TrimSpace(string(input.ProjectProbeReceiptID)) != "" {
			return commandPlanPolicy{}, &ValidationError{Field: "project_probe_receipt_id", Message: "must be omitted for project_probe command plans"}
		}
		if strings.TrimSpace(input.SelectedTargetID) != "" {
			return commandPlanPolicy{}, &ValidationError{Field: "selected_target_id", Message: "must be omitted for project_probe command plans"}
		}
		return commandPlanPolicy{
			CommandKind:      spine.ExecutionCommandKindProjectProbe,
			Action:           spine.ExecutionCommandActionDetectTestTargets,
			WorkingDirectory: ".",
			PathScope:        []string{"."},
			TimeoutSeconds:   30,
			MaxStdoutBytes:   0,
			MaxStderrBytes:   0,
		}, nil
	case kind == spine.ExecutionCommandKindProjectTest && action == spine.ExecutionCommandActionRunTestTarget:
		if strings.TrimSpace(string(input.ProjectProbeReceiptID)) == "" {
			return commandPlanPolicy{}, &ValidationError{Field: "project_probe_receipt_id", Message: "is required for project_test command plans"}
		}
		if err := validateUUIDv7Field("project_probe_receipt_id", string(input.ProjectProbeReceiptID)); err != nil {
			return commandPlanPolicy{}, err
		}
		if strings.TrimSpace(input.SelectedTargetID) == "" {
			return commandPlanPolicy{}, &ValidationError{Field: "selected_target_id", Message: "is required for project_test command plans"}
		}
		sourceReceiptID := input.ProjectProbeReceiptID
		return commandPlanPolicy{
			CommandKind:                 spine.ExecutionCommandKindProjectTest,
			Action:                      spine.ExecutionCommandActionRunTestTarget,
			WorkingDirectory:            ".",
			PathScope:                   []string{"."},
			TimeoutSeconds:              120,
			NetworkAllowed:              false,
			WorkspaceWriteAllowed:       false,
			ScratchWriteAllowed:         false,
			MaxStdoutBytes:              0,
			MaxStderrBytes:              0,
			ChangedPathsAllowed:         false,
			SourceProjectProbeReceiptID: &sourceReceiptID,
			SelectedTargetID:            strings.TrimSpace(input.SelectedTargetID),
		}, nil
	case kind == spine.ExecutionCommandKindBuiltinDiagnostic:
		return commandPlanPolicy{}, &ValidationError{Field: "action", Message: "must be workspace_status"}
	case kind == spine.ExecutionCommandKindProjectProbe:
		return commandPlanPolicy{}, &ValidationError{Field: "action", Message: "must be detect_declared_test_targets"}
	case kind == spine.ExecutionCommandKindProjectTest:
		return commandPlanPolicy{}, &ValidationError{Field: "action", Message: "must be run_declared_test_target"}
	default:
		return commandPlanPolicy{}, &ValidationError{Field: "command_kind", Message: "must be builtin_diagnostic, project_probe, or project_test"}
	}
}

func validateUUIDv7Field(field string, value string) error {
	id, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		return &ValidationError{Field: field, Message: "must be a UUID"}
	}
	if id.Version() != 7 {
		return &ValidationError{Field: field, Message: "must be a UUIDv7"}
	}
	return nil
}

func (s *Service) prepareCommandPlanPolicy(ctx context.Context, input spine.ExecutionCommandPlanCreateRequest, run spine.Run, job spine.ExecutionJob, policy commandPlanPolicy) (commandPlanPolicy, error) {
	if policy.CommandKind != spine.ExecutionCommandKindProjectTest {
		return policy, nil
	}
	if policy.SourceProjectProbeReceiptID == nil {
		return commandPlanPolicy{}, &ValidationError{Field: "project_probe_receipt_id", Message: "is required for project_test command plans"}
	}
	probeReceipt, ok, err := s.ExecutionReceipts.Get(ctx, *policy.SourceProjectProbeReceiptID)
	if err != nil {
		return commandPlanPolicy{}, fmt.Errorf("get project probe execution receipt: %w", err)
	}
	if !ok {
		return commandPlanPolicy{}, ErrExecutionReceiptNotFound
	}
	if probeReceipt.ExecutionMode != spine.ExecutionReceiptModeProjectProbe ||
		probeReceipt.CommandKind != spine.ExecutionCommandKindProjectProbe ||
		probeReceipt.Action != spine.ExecutionCommandActionDetectTestTargets ||
		probeReceipt.ProjectProbeMetadata == nil {
		return commandPlanPolicy{}, ErrInvalidCommandPlan
	}
	if probeReceipt.TaskID != job.TaskID ||
		probeReceipt.CheckoutReceiptID != job.CheckoutReceiptID ||
		probeReceipt.RepoBindingID != job.RepoBindingID ||
		probeReceipt.TaskID != run.TaskID ||
		probeReceipt.CheckoutReceiptID != run.CheckoutReceiptID {
		return commandPlanPolicy{}, ErrInvalidCommandPlan
	}
	target, ok := findProjectProbeTarget(*probeReceipt.ProjectProbeMetadata, policy.SelectedTargetID)
	if !ok {
		return commandPlanPolicy{}, &ValidationError{Field: "selected_target_id", Message: "must match a declared project_probe test target"}
	}
	if !isSupportedProjectTestTarget(target) {
		return commandPlanPolicy{}, &ValidationError{Field: "selected_target_id", Message: "uses an unsupported target family"}
	}
	if !isSafeRelativePath(target.SourcePath) {
		return commandPlanPolicy{}, &ValidationError{Field: "selected_target_id", Message: "must point inside path_scope"}
	}
	selected := target
	policy.DeclaredTestTarget = &selected
	return policy, nil
}

func findProjectProbeTarget(metadata spine.ProjectProbeMetadata, selectedTargetID string) (spine.ProjectProbeTestTargetCandidate, bool) {
	normalized := normalizeProjectProbeMetadata(metadata)
	for _, candidate := range normalized.DeclaredTestTargetCandidates {
		candidate = normalizeProjectProbeTarget(candidate)
		if projectProbeTargetID(candidate) == selectedTargetID {
			return candidate, true
		}
	}
	return spine.ProjectProbeTestTargetCandidate{}, false
}

func normalizeProjectProbeTarget(candidate spine.ProjectProbeTestTargetCandidate) spine.ProjectProbeTestTargetCandidate {
	candidate.Name = strings.TrimSpace(candidate.Name)
	candidate.SourcePath = strings.TrimSpace(candidate.SourcePath)
	candidate.SourceKind = strings.TrimSpace(candidate.SourceKind)
	return candidate
}

func projectProbeTargetID(candidate spine.ProjectProbeTestTargetCandidate) string {
	candidate = normalizeProjectProbeTarget(candidate)
	return candidate.SourcePath + "#" + candidate.SourceKind + ":" + candidate.Name
}

func isSupportedProjectTestTarget(candidate spine.ProjectProbeTestTargetCandidate) bool {
	candidate = normalizeProjectProbeTarget(candidate)
	if candidate.SourceKind != "package_json_script" {
		return false
	}
	return candidate.Name == "test" || strings.HasPrefix(candidate.Name, "test:")
}

func commandPlanMatchesPolicy(plan spine.ExecutionCommandPlan, policy commandPlanPolicy) bool {
	if plan.CommandKind != policy.CommandKind ||
		plan.Action != policy.Action ||
		plan.WorkingDirectory != policy.WorkingDirectory ||
		plan.TimeoutSeconds != policy.TimeoutSeconds ||
		plan.MaxStdoutBytes != policy.MaxStdoutBytes ||
		plan.MaxStderrBytes != policy.MaxStderrBytes ||
		plan.NetworkAllowed != policy.NetworkAllowed ||
		plan.WorkspaceWriteAllowed != policy.WorkspaceWriteAllowed ||
		plan.ScratchWriteAllowed != policy.ScratchWriteAllowed ||
		plan.ChangedPathsAllowed != policy.ChangedPathsAllowed ||
		plan.ShellAllowed ||
		plan.RawSourceUploadAllowed ||
		len(plan.Argv) != 0 ||
		len(plan.AllowedArtifactKinds) != 0 ||
		len(plan.PathScope) != len(policy.PathScope) {
		return false
	}
	for i := range plan.PathScope {
		if plan.PathScope[i] != policy.PathScope[i] {
			return false
		}
	}
	if policy.SourceProjectProbeReceiptID == nil {
		return plan.SourceProjectProbeReceiptID == nil && plan.SelectedTargetID == "" && plan.DeclaredTestTarget == nil
	}
	if plan.SourceProjectProbeReceiptID == nil || *plan.SourceProjectProbeReceiptID != *policy.SourceProjectProbeReceiptID || plan.SelectedTargetID != policy.SelectedTargetID {
		return false
	}
	if policy.DeclaredTestTarget == nil || plan.DeclaredTestTarget == nil {
		return policy.DeclaredTestTarget == nil && plan.DeclaredTestTarget == nil
	}
	return *plan.DeclaredTestTarget == *policy.DeclaredTestTarget
}

func decorateCommandPlanForResponse(plan spine.ExecutionCommandPlan) spine.ExecutionCommandPlan {
	if plan.CommandKind == spine.ExecutionCommandKindProjectTest && plan.Action == spine.ExecutionCommandActionRunTestTarget {
		plan.NextAction = &spine.ExecutionNextAction{
			Kind:         spine.ExecutionCommandPlanNextActionRunnerProjectTestRequired,
			Blocking:     true,
			Available:    false,
			PlannedSlice: spine.ExecutionCommandPlanNextActionProjectTestPlannedSlice,
		}
	}
	return plan
}

func validateReceiptInput(input spine.ExecutionReceiptSubmitRequest) error {
	if strings.TrimSpace(string(input.ExecutionJobID)) == "" {
		return &ValidationError{Field: "execution_job_id", Message: "is required"}
	}
	if strings.TrimSpace(string(input.LeaseID)) == "" {
		return &ValidationError{Field: "lease_id", Message: "is required"}
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
	mode := strings.TrimSpace(input.ExecutionMode)
	status := strings.TrimSpace(input.ProcessStatus)
	if input.ExitCode != nil {
		return &ValidationError{Field: "exit_code", Message: "must be omitted for current execution receipts"}
	}
	if len(input.ArtifactRefs) != 0 {
		return &ValidationError{Field: "artifact_refs", Message: "must be empty for current execution receipts"}
	}
	if len(input.ChangedPathsSummary) != 0 {
		return &ValidationError{Field: "changed_paths_summary", Message: "must be empty for current execution receipts"}
	}
	if input.RawSourceUploaded {
		return ErrExecutionRawSourceUploaded
	}
	switch mode {
	case spine.ExecutionReceiptModeNoCommand:
		if strings.TrimSpace(string(input.CommandPlanID)) != "" {
			return &ValidationError{Field: "command_plan_id", Message: "must be omitted for no_command receipts"}
		}
		if strings.TrimSpace(input.CommandKind) != "" {
			return &ValidationError{Field: "command_kind", Message: "must be omitted for no_command receipts"}
		}
		if strings.TrimSpace(input.Action) != "" {
			return &ValidationError{Field: "action", Message: "must be omitted for no_command receipts"}
		}
		if input.RunnerStartedAt != nil {
			return &ValidationError{Field: "runner_started_at", Message: "must be omitted for no_command receipts"}
		}
		if input.RunnerFinishedAt != nil {
			return &ValidationError{Field: "runner_finished_at", Message: "must be omitted for no_command receipts"}
		}
		if input.ProjectProbeMetadata != nil {
			return &ValidationError{Field: "project_probe_metadata", Message: "must be omitted for no_command receipts"}
		}
		switch status {
		case spine.ExecutionReceiptStatusNotExecuted, spine.ExecutionReceiptStatusMetadataOnly:
		default:
			return &ValidationError{Field: "process_status", Message: "must be not_executed or metadata_only"}
		}
	case spine.ExecutionReceiptModeBuiltinDiagnostic:
		if strings.TrimSpace(string(input.CommandPlanID)) == "" {
			return &ValidationError{Field: "command_plan_id", Message: "is required for builtin_diagnostic receipts"}
		}
		if strings.TrimSpace(input.CommandKind) != spine.ExecutionCommandKindBuiltinDiagnostic {
			return &ValidationError{Field: "command_kind", Message: "must be builtin_diagnostic"}
		}
		if strings.TrimSpace(input.Action) != spine.ExecutionCommandActionWorkspaceStatus {
			return &ValidationError{Field: "action", Message: "must be workspace_status"}
		}
		if status != spine.ExecutionReceiptStatusMetadataOnly {
			return &ValidationError{Field: "process_status", Message: "must be metadata_only for builtin_diagnostic receipts"}
		}
		if input.RunnerStartedAt == nil {
			return &ValidationError{Field: "runner_started_at", Message: "is required for builtin_diagnostic receipts"}
		}
		if input.RunnerFinishedAt == nil {
			return &ValidationError{Field: "runner_finished_at", Message: "is required for builtin_diagnostic receipts"}
		}
		if input.ProjectProbeMetadata != nil {
			return &ValidationError{Field: "project_probe_metadata", Message: "must be omitted for builtin_diagnostic receipts"}
		}
	case spine.ExecutionReceiptModeProjectProbe:
		if strings.TrimSpace(string(input.CommandPlanID)) == "" {
			return &ValidationError{Field: "command_plan_id", Message: "is required for project_probe receipts"}
		}
		if strings.TrimSpace(input.CommandKind) != spine.ExecutionCommandKindProjectProbe {
			return &ValidationError{Field: "command_kind", Message: "must be project_probe"}
		}
		if strings.TrimSpace(input.Action) != spine.ExecutionCommandActionDetectTestTargets {
			return &ValidationError{Field: "action", Message: "must be detect_declared_test_targets"}
		}
		if status != spine.ExecutionReceiptStatusMetadataOnly {
			return &ValidationError{Field: "process_status", Message: "must be metadata_only for project_probe receipts"}
		}
		if input.RunnerStartedAt == nil {
			return &ValidationError{Field: "runner_started_at", Message: "is required for project_probe receipts"}
		}
		if input.RunnerFinishedAt == nil {
			return &ValidationError{Field: "runner_finished_at", Message: "is required for project_probe receipts"}
		}
		if input.ProjectProbeMetadata == nil {
			return &ValidationError{Field: "project_probe_metadata", Message: "is required for project_probe receipts"}
		}
		if err := validateProjectProbeMetadata(*input.ProjectProbeMetadata); err != nil {
			return err
		}
	default:
		return &ValidationError{Field: "execution_mode", Message: "must be no_command, builtin_diagnostic, or project_probe"}
	}
	return nil
}

func (s *Service) validateBuiltinDiagnosticReceipt(ctx context.Context, input spine.ExecutionReceiptSubmitRequest, run spine.Run, job spine.ExecutionJob) (spine.ExecutionCommandPlan, error) {
	plan, ok, err := s.CommandPlans.Get(ctx, input.CommandPlanID)
	if err != nil {
		return spine.ExecutionCommandPlan{}, fmt.Errorf("get execution command plan: %w", err)
	}
	if !ok {
		return spine.ExecutionCommandPlan{}, ErrExecutionCommandPlanNotFound
	}
	if plan.OrganizationID != job.OrganizationID || plan.ProjectID != job.ProjectID || plan.RepoBindingID != job.RepoBindingID {
		return spine.ExecutionCommandPlan{}, ErrInvalidCommandPlan
	}
	if plan.RunID != run.ID || plan.ExecutionJobID != job.ID || plan.TaskID != run.TaskID || plan.CheckoutReceiptID != run.CheckoutReceiptID {
		return spine.ExecutionCommandPlan{}, ErrInvalidCommandPlan
	}
	if plan.State != spine.ExecutionCommandPlanStatePlanned {
		return spine.ExecutionCommandPlan{}, ErrInvalidCommandPlan
	}
	if plan.CommandKind != spine.ExecutionCommandKindBuiltinDiagnostic || plan.Action != spine.ExecutionCommandActionWorkspaceStatus {
		return spine.ExecutionCommandPlan{}, ErrInvalidCommandPlan
	}
	if plan.ShellAllowed || len(plan.Argv) != 0 || plan.WorkingDirectory != "." || len(plan.PathScope) != 1 || plan.PathScope[0] != "." || plan.RawSourceUploadAllowed {
		return spine.ExecutionCommandPlan{}, ErrInvalidCommandPlan
	}
	if len(plan.AllowedArtifactKinds) != 0 || plan.MaxStdoutBytes != 0 || plan.MaxStderrBytes != 0 {
		return spine.ExecutionCommandPlan{}, ErrInvalidCommandPlan
	}
	if input.RunnerStartedAt != nil && input.RunnerFinishedAt != nil && input.RunnerFinishedAt.Before(*input.RunnerStartedAt) {
		return spine.ExecutionCommandPlan{}, &ValidationError{Field: "runner_finished_at", Message: "must be after runner_started_at"}
	}
	return plan, nil
}

func (s *Service) validateProjectProbeReceipt(ctx context.Context, input spine.ExecutionReceiptSubmitRequest, run spine.Run, job spine.ExecutionJob) (spine.ExecutionCommandPlan, error) {
	plan, ok, err := s.CommandPlans.Get(ctx, input.CommandPlanID)
	if err != nil {
		return spine.ExecutionCommandPlan{}, fmt.Errorf("get execution command plan: %w", err)
	}
	if !ok {
		return spine.ExecutionCommandPlan{}, ErrExecutionCommandPlanNotFound
	}
	if plan.OrganizationID != job.OrganizationID || plan.ProjectID != job.ProjectID || plan.RepoBindingID != job.RepoBindingID {
		return spine.ExecutionCommandPlan{}, ErrInvalidCommandPlan
	}
	if plan.RunID != run.ID || plan.ExecutionJobID != job.ID || plan.TaskID != run.TaskID || plan.CheckoutReceiptID != run.CheckoutReceiptID {
		return spine.ExecutionCommandPlan{}, ErrInvalidCommandPlan
	}
	if plan.State != spine.ExecutionCommandPlanStatePlanned {
		return spine.ExecutionCommandPlan{}, ErrInvalidCommandPlan
	}
	if plan.CommandKind != spine.ExecutionCommandKindProjectProbe || plan.Action != spine.ExecutionCommandActionDetectTestTargets {
		return spine.ExecutionCommandPlan{}, ErrInvalidCommandPlan
	}
	if plan.ShellAllowed || len(plan.Argv) != 0 || plan.WorkingDirectory != "." || len(plan.PathScope) != 1 || plan.PathScope[0] != "." || plan.RawSourceUploadAllowed {
		return spine.ExecutionCommandPlan{}, ErrInvalidCommandPlan
	}
	if len(plan.AllowedArtifactKinds) != 0 || plan.MaxStdoutBytes != 0 || plan.MaxStderrBytes != 0 {
		return spine.ExecutionCommandPlan{}, ErrInvalidCommandPlan
	}
	if input.RunnerStartedAt != nil && input.RunnerFinishedAt != nil && input.RunnerFinishedAt.Before(*input.RunnerStartedAt) {
		return spine.ExecutionCommandPlan{}, &ValidationError{Field: "runner_finished_at", Message: "must be after runner_started_at"}
	}
	return plan, nil
}

func validateProjectProbeMetadata(metadata spine.ProjectProbeMetadata) error {
	normalized := normalizeProjectProbeMetadata(metadata)
	if len(normalized.DetectedManifests) == 0 &&
		len(normalized.PackageManagerCandidates) == 0 &&
		len(normalized.DeclaredTestTargetCandidates) == 0 &&
		len(normalized.UnsupportedOrUnknowns) == 0 &&
		len(normalized.PartialityReasons) == 0 {
		return &ValidationError{Field: "project_probe_metadata", Message: "must include structured metadata or partiality reasons"}
	}
	for i, manifest := range metadata.DetectedManifests {
		if strings.TrimSpace(manifest.Path) == "" {
			return &ValidationError{Field: fmt.Sprintf("project_probe_metadata.detected_manifests[%d].path", i), Message: "is required"}
		}
		if strings.TrimSpace(manifest.Kind) == "" {
			return &ValidationError{Field: fmt.Sprintf("project_probe_metadata.detected_manifests[%d].kind", i), Message: "is required"}
		}
		if !isSafeRelativePath(manifest.Path) {
			return &ValidationError{Field: fmt.Sprintf("project_probe_metadata.detected_manifests[%d].path", i), Message: "must be a relative path within path_scope"}
		}
	}
	for i, candidate := range metadata.PackageManagerCandidates {
		if strings.TrimSpace(candidate.Name) == "" {
			return &ValidationError{Field: fmt.Sprintf("project_probe_metadata.package_manager_candidates[%d].name", i), Message: "is required"}
		}
		if strings.TrimSpace(candidate.SourcePath) == "" {
			return &ValidationError{Field: fmt.Sprintf("project_probe_metadata.package_manager_candidates[%d].source_path", i), Message: "is required"}
		}
		if !isSafeRelativePath(candidate.SourcePath) {
			return &ValidationError{Field: fmt.Sprintf("project_probe_metadata.package_manager_candidates[%d].source_path", i), Message: "must be a relative path within path_scope"}
		}
	}
	for i, candidate := range metadata.DeclaredTestTargetCandidates {
		if strings.TrimSpace(candidate.Name) == "" {
			return &ValidationError{Field: fmt.Sprintf("project_probe_metadata.declared_test_target_candidates[%d].name", i), Message: "is required"}
		}
		if strings.TrimSpace(candidate.SourcePath) == "" {
			return &ValidationError{Field: fmt.Sprintf("project_probe_metadata.declared_test_target_candidates[%d].source_path", i), Message: "is required"}
		}
		if strings.TrimSpace(candidate.SourceKind) == "" {
			return &ValidationError{Field: fmt.Sprintf("project_probe_metadata.declared_test_target_candidates[%d].source_kind", i), Message: "is required"}
		}
		if !isSafeRelativePath(candidate.SourcePath) {
			return &ValidationError{Field: fmt.Sprintf("project_probe_metadata.declared_test_target_candidates[%d].source_path", i), Message: "must be a relative path within path_scope"}
		}
	}
	for i, value := range metadata.UnsupportedOrUnknowns {
		if strings.TrimSpace(value) == "" {
			return &ValidationError{Field: fmt.Sprintf("project_probe_metadata.unsupported_or_unknowns[%d]", i), Message: "must not be blank"}
		}
	}
	for i, value := range metadata.PartialityReasons {
		if strings.TrimSpace(value) == "" {
			return &ValidationError{Field: fmt.Sprintf("project_probe_metadata.partiality_reasons[%d]", i), Message: "must not be blank"}
		}
	}
	return nil
}

func normalizeProjectProbeMetadata(metadata spine.ProjectProbeMetadata) spine.ProjectProbeMetadata {
	if metadata.DetectedManifests == nil {
		metadata.DetectedManifests = []spine.ProjectProbeManifest{}
	}
	if metadata.PackageManagerCandidates == nil {
		metadata.PackageManagerCandidates = []spine.ProjectProbePackageManagerCandidate{}
	}
	if metadata.DeclaredTestTargetCandidates == nil {
		metadata.DeclaredTestTargetCandidates = []spine.ProjectProbeTestTargetCandidate{}
	}
	metadata.UnsupportedOrUnknowns = nonNil(metadata.UnsupportedOrUnknowns)
	metadata.PartialityReasons = nonNil(metadata.PartialityReasons)
	return metadata
}

func nonNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func isSafeRelativePath(value string) bool {
	path := strings.TrimSpace(value)
	if path == "" || strings.HasPrefix(path, "/") || path == "." || path == ".." {
		return false
	}
	if strings.Contains(path, "\\") {
		return false
	}
	for _, part := range strings.Split(path, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
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

func utcTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	utc := value.UTC()
	return &utc
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

func (UUIDGenerator) NewExecutionCommandPlanID() (spine.ExecutionCommandPlanID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return spine.ExecutionCommandPlanID(id.String()), nil
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
