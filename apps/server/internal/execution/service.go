package execution

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

const (
	EventTypeExecutionJobCreated = "execution_job.created"

	EntityTypeExecutionJob = "ExecutionJob"

	ExecutionModePrepareV0 = "prepare_v0"
)

var (
	ErrWorkItemNotFound        = errors.New("work item not found")
	ErrCheckoutReceiptNotFound = errors.New("checkout receipt not found")
	ErrCheckoutJobNotFound     = errors.New("checkout job not found")
	ErrInvalidWorkItemState    = errors.New("work item state does not allow execution preparation")
	ErrInvalidCheckoutState    = errors.New("checkout job state does not allow execution preparation")
	ErrMembershipRequired      = errors.New("active organization membership is required")
	ErrOrganizationForbidden   = errors.New("user is not allowed to prepare execution for this work item")
	ErrProjectMismatch         = errors.New("execution project expectation does not match work item")
	ErrRepoBindingMismatch     = errors.New("execution repo binding expectation does not match work item")
	ErrCheckoutReceiptMismatch = errors.New("checkout receipt does not match work item")
	ErrRawSourceUploaded       = errors.New("checkout receipt must not upload raw source")
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

type CheckoutReceiptReader interface {
	Get(context.Context, spine.CheckoutReceiptID) (spine.CheckoutReceipt, bool, error)
}

type CheckoutJobReader interface {
	Get(context.Context, spine.CheckoutJobID) (spine.CheckoutJob, bool, error)
}

type JobStore interface {
	Create(context.Context, spine.ExecutionJob) error
	GetByTaskAndCheckoutReceipt(context.Context, spine.WorkItemID, spine.CheckoutReceiptID) (spine.ExecutionJob, bool, error)
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
	NewEventID() (spine.EventID, error)
}

type Service struct {
	WorkItems        WorkItemReader
	CheckoutReceipts CheckoutReceiptReader
	CheckoutJobs     CheckoutJobReader
	Jobs             JobStore
	Events           EventLog
	TxRunner         TransactionRunner
	Clock            Clock
	IDs              IDGenerator
}

func NewService(workItems WorkItemReader, checkoutReceipts CheckoutReceiptReader, checkoutJobs CheckoutJobReader, jobs JobStore, events EventLog, txRunner TransactionRunner, clock Clock, ids IDGenerator) *Service {
	return &Service{
		WorkItems:        workItems,
		CheckoutReceipts: checkoutReceipts,
		CheckoutJobs:     checkoutJobs,
		Jobs:             jobs,
		Events:           events,
		TxRunner:         txRunner,
		Clock:            clock,
		IDs:              ids,
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

func (UUIDGenerator) NewEventID() (spine.EventID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return spine.EventID(id.String()), nil
}
