package checkout

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
	EventTypeCheckoutJobCreated       = "checkout_job.created"
	EventTypeCheckoutReceiptSubmitted = "checkout_receipt.submitted"

	EntityTypeCheckoutJob     = "CheckoutJob"
	EntityTypeCheckoutReceipt = "CheckoutReceipt"

	defaultLeaseTTL = 15 * time.Minute
	minLeaseTTL     = 30 * time.Second
	maxLeaseTTL     = 60 * time.Minute
)

var (
	ErrWorkItemNotFound      = errors.New("work item not found")
	ErrRepoBindingNotFound   = errors.New("repo binding not found")
	ErrCheckoutJobNotFound   = errors.New("checkout job not found")
	ErrInvalidWorkItemState  = errors.New("work item state does not allow checkout preparation")
	ErrAlreadyReceipted      = errors.New("checkout receipt already submitted")
	ErrInvalidCheckoutState  = errors.New("checkout job state does not allow this transition")
	ErrLeaseExpired          = errors.New("checkout job lease expired")
	ErrInvalidLease          = errors.New("checkout job lease is invalid")
	ErrMembershipRequired    = errors.New("active organization membership is required")
	ErrOrganizationForbidden = errors.New("user is not allowed to prepare checkout for this work item")
	ErrProjectMismatch       = errors.New("checkout project expectation does not match work item")
	ErrRepoBindingMismatch   = errors.New("checkout repo binding expectation does not match work item")
	ErrRawSourceUploaded     = errors.New("checkout receipt must not upload raw source")
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

type JobLeaseInput struct {
	OrganizationID spine.OrganizationID
	ProjectID      spine.ProjectID
	RepoBindingID  spine.RepoBindingID
	RunnerID       string
	LeaseTokenHash string
	LeaseExpiresAt time.Time
	UpdatedAt      time.Time
}

type JobStore interface {
	Create(context.Context, spine.CheckoutJob) error
	Get(context.Context, spine.CheckoutJobID) (spine.CheckoutJob, bool, error)
	GetByTaskID(context.Context, spine.WorkItemID) (spine.CheckoutJob, bool, error)
	AcquireNextLease(context.Context, JobLeaseInput) (spine.CheckoutJob, bool, error)
	MarkReceiptSubmitted(context.Context, spine.CheckoutJobID, string, string, time.Time) (bool, error)
}

type ReceiptStore interface {
	Create(context.Context, spine.CheckoutReceipt) error
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
	NewCheckoutJobID() (spine.CheckoutJobID, error)
	NewCheckoutReceiptID() (spine.CheckoutReceiptID, error)
	NewEventID() (spine.EventID, error)
}

type Service struct {
	WorkItems    WorkItemReader
	RepoBindings RepoBindingReader
	Jobs         JobStore
	Receipts     ReceiptStore
	Events       EventLog
	TxRunner     TransactionRunner
	Clock        Clock
	IDs          IDGenerator
}

func NewService(workItems WorkItemReader, repoBindings RepoBindingReader, jobs JobStore, receipts ReceiptStore, events EventLog, txRunner TransactionRunner, clock Clock, ids IDGenerator) *Service {
	return &Service{
		WorkItems:    workItems,
		RepoBindings: repoBindings,
		Jobs:         jobs,
		Receipts:     receipts,
		Events:       events,
		TxRunner:     txRunner,
		Clock:        clock,
		IDs:          ids,
	}
}

func (s *Service) CreateOrReturnJob(ctx context.Context, taskID spine.WorkItemID, input spine.CheckoutJobCreateRequest, membership spine.OrganizationMembership) (spine.CheckoutJob, bool, error) {
	if err := validateActor("requested_by", input.RequestedBy); err != nil {
		return spine.CheckoutJob{}, false, err
	}
	task, binding, err := s.loadAuthorizedTaskContext(ctx, taskID, input.ProjectID, input.RepoBindingID, membership)
	if err != nil {
		return spine.CheckoutJob{}, false, err
	}
	if existing, ok, err := s.Jobs.GetByTaskID(ctx, task.ID); err != nil {
		return spine.CheckoutJob{}, false, fmt.Errorf("get checkout job by task id: %w", err)
	} else if ok {
		return existing, false, nil
	}

	jobID, err := s.IDs.NewCheckoutJobID()
	if err != nil {
		return spine.CheckoutJob{}, false, fmt.Errorf("new checkout job id: %w", err)
	}
	now := s.Clock.Now().UTC()
	job := spine.CheckoutJob{
		ID:                 jobID,
		OrganizationID:     task.OrganizationID,
		ProjectID:          task.ProjectID,
		TaskID:             task.ID,
		ContractID:         task.ContractID,
		ApprovedContractID: task.ApprovedContractID,
		PlanID:             task.PlanID,
		ProposalID:         task.ProposalID,
		RepoBindingID:      task.RepoBindingID,
		State:              spine.CheckoutJobStateQueued,
		RequestedBy:        input.RequestedBy,
		Instruction:        checkoutInstruction(jobID, task, binding),
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	event, err := s.event(EventTypeCheckoutJobCreated, EntityTypeCheckoutJob, string(job.ID), job.OrganizationID, job.ProjectID, job.RepoBindingID, now, map[string]any{
		"job_id":          job.ID,
		"task_id":         job.TaskID,
		"repo_binding_id": job.RepoBindingID,
		"requested_by":    job.RequestedBy,
	})
	if err != nil {
		return spine.CheckoutJob{}, false, err
	}
	if err := s.TxRunner.RunReadCommitted(ctx, func(txCtx context.Context) error {
		if err := s.Jobs.Create(txCtx, job); err != nil {
			return fmt.Errorf("create checkout job: %w", err)
		}
		if err := s.Events.Append(txCtx, event); err != nil {
			return fmt.Errorf("append checkout job created event: %w", err)
		}
		return nil
	}); err != nil {
		if existing, ok, lookupErr := s.Jobs.GetByTaskID(ctx, task.ID); lookupErr != nil {
			return spine.CheckoutJob{}, false, fmt.Errorf("get checkout job by task id after create failure: %w", lookupErr)
		} else if ok {
			return existing, false, nil
		}
		return spine.CheckoutJob{}, false, err
	}
	return job, true, nil
}

func (s *Service) AcquireNextLease(ctx context.Context, input spine.CheckoutJobLeaseCreateRequest, membership spine.OrganizationMembership) (spine.CheckoutJobLeaseCreated, bool, error) {
	if err := requireActiveMembership(membership); err != nil {
		return spine.CheckoutJobLeaseCreated{}, false, err
	}
	runnerID := strings.TrimSpace(input.RunnerID)
	if runnerID == "" {
		return spine.CheckoutJobLeaseCreated{}, false, &ValidationError{Field: "runner_id", Message: "is required"}
	}
	projectID := input.ProjectID
	if strings.TrimSpace(string(projectID)) == "" {
		return spine.CheckoutJobLeaseCreated{}, false, &ValidationError{Field: "project_id", Message: "is required"}
	}
	repoBindingID := input.RepoBindingID
	if strings.TrimSpace(string(repoBindingID)) == "" {
		return spine.CheckoutJobLeaseCreated{}, false, &ValidationError{Field: "repo_binding_id", Message: "is required"}
	}
	if err := s.validateLeaseScope(ctx, projectID, repoBindingID, membership); err != nil {
		return spine.CheckoutJobLeaseCreated{}, false, err
	}
	ttl, err := leaseTTL(input.TTLSeconds)
	if err != nil {
		return spine.CheckoutJobLeaseCreated{}, false, err
	}
	token, err := newLeaseToken()
	if err != nil {
		return spine.CheckoutJobLeaseCreated{}, false, fmt.Errorf("new checkout lease token: %w", err)
	}
	now := s.Clock.Now().UTC()
	leaseInput := JobLeaseInput{
		OrganizationID: membership.OrganizationID,
		ProjectID:      projectID,
		RepoBindingID:  repoBindingID,
		RunnerID:       runnerID,
		LeaseTokenHash: leaseTokenHash(token),
		LeaseExpiresAt: now.Add(ttl),
		UpdatedAt:      now,
	}
	var job spine.CheckoutJob
	var ok bool
	if err := s.TxRunner.RunReadCommitted(ctx, func(txCtx context.Context) error {
		var acquireErr error
		job, ok, acquireErr = s.Jobs.AcquireNextLease(txCtx, leaseInput)
		if acquireErr != nil {
			return fmt.Errorf("acquire checkout job lease: %w", acquireErr)
		}
		return nil
	}); err != nil {
		return spine.CheckoutJobLeaseCreated{}, false, err
	}
	if !ok {
		return spine.CheckoutJobLeaseCreated{}, false, nil
	}
	return spine.CheckoutJobLeaseCreated{
		JobID:          job.ID,
		TaskID:         job.TaskID,
		State:          job.State,
		RunnerID:       runnerID,
		LeaseToken:     token,
		LeaseExpiresAt: leaseInput.LeaseExpiresAt,
		Instruction:    job.Instruction,
		CreatedAt:      job.CreatedAt,
		UpdatedAt:      job.UpdatedAt,
	}, true, nil
}

func (s *Service) SubmitReceipt(ctx context.Context, jobID spine.CheckoutJobID, input spine.CheckoutReceiptSubmitRequest, membership spine.OrganizationMembership) (spine.CheckoutReceipt, error) {
	if err := validateReceiptInput(input); err != nil {
		return spine.CheckoutReceipt{}, err
	}
	job, ok, err := s.Jobs.Get(ctx, jobID)
	if err != nil {
		return spine.CheckoutReceipt{}, fmt.Errorf("get checkout job: %w", err)
	}
	if !ok {
		return spine.CheckoutReceipt{}, ErrCheckoutJobNotFound
	}
	if err := authorizeTaskAccess(membership, job.OrganizationID); err != nil {
		return spine.CheckoutReceipt{}, err
	}
	if job.State == spine.CheckoutJobStateReceiptSubmitted {
		return spine.CheckoutReceipt{}, ErrAlreadyReceipted
	}
	now := s.Clock.Now().UTC()
	if err := validateLeaseProof(job, input.RunnerID, leaseTokenHash(input.LeaseToken), now); err != nil {
		return spine.CheckoutReceipt{}, err
	}
	receiptID, err := s.IDs.NewCheckoutReceiptID()
	if err != nil {
		return spine.CheckoutReceipt{}, fmt.Errorf("new checkout receipt id: %w", err)
	}
	receipt := spine.CheckoutReceipt{
		ID:                receiptID,
		JobID:             job.ID,
		TaskID:            job.TaskID,
		RepoBindingID:     job.RepoBindingID,
		RunnerID:          strings.TrimSpace(input.RunnerID),
		WorkspaceRef:      strings.TrimSpace(input.WorkspaceRef),
		CommitSHA:         strings.TrimSpace(input.CommitSHA),
		BaselineID:        strings.TrimSpace(input.BaselineID),
		OverlayID:         strings.TrimSpace(input.OverlayID),
		Dirty:             input.Dirty,
		Partial:           input.Partial,
		PartialReasons:    cloneNonBlankStrings(input.PartialReasons),
		RawSourceUploaded: false,
		CreatedAt:         now,
	}
	event, err := s.event(EventTypeCheckoutReceiptSubmitted, EntityTypeCheckoutReceipt, string(receipt.ID), job.OrganizationID, job.ProjectID, job.RepoBindingID, now, map[string]any{
		"receipt_id":          receipt.ID,
		"job_id":              job.ID,
		"task_id":             job.TaskID,
		"repo_binding_id":     job.RepoBindingID,
		"runner_id":           receipt.RunnerID,
		"commit_sha":          receipt.CommitSHA,
		"raw_source_uploaded": receipt.RawSourceUploaded,
	})
	if err != nil {
		return spine.CheckoutReceipt{}, err
	}
	if err := s.TxRunner.RunReadCommitted(ctx, func(txCtx context.Context) error {
		if err := s.Receipts.Create(txCtx, receipt); err != nil {
			if errors.Is(err, ErrAlreadyReceipted) {
				return ErrAlreadyReceipted
			}
			return fmt.Errorf("create checkout receipt: %w", err)
		}
		updated, err := s.Jobs.MarkReceiptSubmitted(txCtx, job.ID, receipt.RunnerID, leaseTokenHash(input.LeaseToken), now)
		if err != nil {
			return fmt.Errorf("mark checkout job receipt submitted: %w", err)
		}
		if !updated {
			return ErrInvalidLease
		}
		if err := s.Events.Append(txCtx, event); err != nil {
			return fmt.Errorf("append checkout receipt submitted event: %w", err)
		}
		return nil
	}); err != nil {
		return spine.CheckoutReceipt{}, err
	}
	return receipt, nil
}

func (s *Service) validateLeaseScope(ctx context.Context, projectID spine.ProjectID, repoBindingID spine.RepoBindingID, membership spine.OrganizationMembership) error {
	binding, ok, err := s.RepoBindings.GetRepoBinding(ctx, repoBindingID)
	if err != nil {
		return fmt.Errorf("get checkout lease repo binding: %w", err)
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

func (s *Service) loadAuthorizedTaskContext(ctx context.Context, taskID spine.WorkItemID, projectID spine.ProjectID, repoBindingID spine.RepoBindingID, membership spine.OrganizationMembership) (spine.WorkItem, spine.RepoBinding, error) {
	task, ok, err := s.WorkItems.Get(ctx, taskID)
	if err != nil {
		return spine.WorkItem{}, spine.RepoBinding{}, fmt.Errorf("get work item: %w", err)
	}
	if !ok {
		return spine.WorkItem{}, spine.RepoBinding{}, ErrWorkItemNotFound
	}
	if task.Status != spine.WorkItemStatusPlanned {
		return spine.WorkItem{}, spine.RepoBinding{}, fmt.Errorf("%w: %s", ErrInvalidWorkItemState, task.Status)
	}
	if err := authorizeTaskAccess(membership, task.OrganizationID); err != nil {
		return spine.WorkItem{}, spine.RepoBinding{}, err
	}
	if strings.TrimSpace(string(projectID)) != "" && projectID != task.ProjectID {
		return spine.WorkItem{}, spine.RepoBinding{}, ErrProjectMismatch
	}
	if strings.TrimSpace(string(repoBindingID)) != "" && repoBindingID != task.RepoBindingID {
		return spine.WorkItem{}, spine.RepoBinding{}, ErrRepoBindingMismatch
	}
	binding, ok, err := s.RepoBindings.GetRepoBinding(ctx, task.RepoBindingID)
	if err != nil {
		return spine.WorkItem{}, spine.RepoBinding{}, fmt.Errorf("get repo binding: %w", err)
	}
	if !ok || binding.State != spine.EntityStateActive {
		return spine.WorkItem{}, spine.RepoBinding{}, ErrRepoBindingNotFound
	}
	if binding.OrganizationID != task.OrganizationID || binding.ProjectID != task.ProjectID || binding.ID != task.RepoBindingID {
		return spine.WorkItem{}, spine.RepoBinding{}, ErrRepoBindingMismatch
	}
	return task, binding, nil
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

func checkoutInstruction(jobID spine.CheckoutJobID, task spine.WorkItem, binding spine.RepoBinding) spine.CheckoutInstruction {
	pathScope := strings.TrimSpace(binding.PathScope)
	if pathScope == "" {
		pathScope = "."
	}
	return spine.CheckoutInstruction{
		JobID:              jobID,
		TaskID:             task.ID,
		RepoBindingID:      task.RepoBindingID,
		AccessMode:         binding.AccessMode,
		Provider:           binding.Provider,
		RepositoryFullName: binding.RepositoryFullName,
		RepositoryURL:      binding.RepositoryURL,
		WorkflowBaseBranch: binding.WorkflowBaseBranch,
		PathScope:          pathScope,
		SourceRef:          spine.SourceRef{Kind: "work_item", ID: string(task.ID)},
		RawSourceUploaded:  false,
	}
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

func validateReceiptInput(input spine.CheckoutReceiptSubmitRequest) error {
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
	if input.RawSourceUploaded {
		return ErrRawSourceUploaded
	}
	return nil
}

func validateLeaseProof(job spine.CheckoutJob, runnerID string, tokenHash string, now time.Time) error {
	if job.State != spine.CheckoutJobStateLeased {
		return fmt.Errorf("%w: %s", ErrInvalidCheckoutState, job.State)
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

func cloneNonBlankStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
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

func (UUIDGenerator) NewCheckoutJobID() (spine.CheckoutJobID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return spine.CheckoutJobID(id.String()), nil
}

func (UUIDGenerator) NewCheckoutReceiptID() (spine.CheckoutReceiptID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return spine.CheckoutReceiptID(id.String()), nil
}

func (UUIDGenerator) NewEventID() (spine.EventID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return spine.EventID(id.String()), nil
}
