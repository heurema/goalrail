package workitem

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
	EventTypeWorkItemCreated        = "work_item.created"
	EntityTypeWorkItem              = "WorkItem"
	SourceRefKindApprovedContract   = "approved_contract"
	ReasonMissingRepoBindingID      = "missing_repo_binding_id"
	ReasonMissingTitle              = "missing_title"
	ReasonMissingIntentSummary      = "missing_intent_summary"
	ReasonMissingScope              = "missing_scope"
	ReasonMissingAcceptanceCriteria = "missing_acceptance_criteria"
	ReasonMissingProofExpectations  = "missing_proof_expectations"
)

var (
	ErrApprovedContractNotFound     = errors.New("approved contract not found")
	ErrInvalidApprovedContractState = errors.New("approved contract is not plannable")
	ErrAlreadyPlanned               = errors.New("approved contract already planned")
)

type CompletenessError struct {
	ReasonCodes []string
}

func (e *CompletenessError) Error() string {
	if len(e.ReasonCodes) == 0 {
		return "work item planning checks failed"
	}
	return "work item planning checks failed: " + strings.Join(e.ReasonCodes, ",")
}

type ApprovedContractReader interface {
	Get(context.Context, spine.ApprovedContractID) (spine.ApprovedContract, bool, error)
}

type Store interface {
	Create(context.Context, spine.WorkItem) error
	Get(context.Context, spine.WorkItemID) (spine.WorkItem, bool, error)
	GetByApprovedContractID(context.Context, spine.ApprovedContractID) (spine.WorkItem, bool, error)
}

type EventLog interface {
	Append(context.Context, spine.Event) error
}

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	NewWorkItemID() (spine.WorkItemID, error)
	NewEventID() (spine.EventID, error)
}

type Service struct {
	ApprovedContracts ApprovedContractReader
	WorkItems         Store
	Events            EventLog
	Clock             Clock
	IDs               IDGenerator
}

func NewService(approvedContracts ApprovedContractReader, workItems Store, events EventLog, clock Clock, ids IDGenerator) *Service {
	return &Service{
		ApprovedContracts: approvedContracts,
		WorkItems:         workItems,
		Events:            events,
		Clock:             clock,
		IDs:               ids,
	}
}

func (s *Service) PlanApprovedContract(ctx context.Context, approvedContractID spine.ApprovedContractID) (spine.WorkItem, error) {
	if err := s.validateDependencies(); err != nil {
		return spine.WorkItem{}, err
	}

	approved, ok, err := s.ApprovedContracts.Get(ctx, approvedContractID)
	if err != nil {
		return spine.WorkItem{}, fmt.Errorf("get approved contract: %w", err)
	}
	if !ok {
		return spine.WorkItem{}, ErrApprovedContractNotFound
	}
	if approved.State != spine.ApprovedContractStateApproved {
		return spine.WorkItem{}, fmt.Errorf("%w: %s", ErrInvalidApprovedContractState, approved.State)
	}
	if _, ok, err := s.WorkItems.GetByApprovedContractID(ctx, approved.ID); err != nil {
		return spine.WorkItem{}, fmt.Errorf("get work item by approved contract id: %w", err)
	} else if ok {
		return spine.WorkItem{}, ErrAlreadyPlanned
	}
	if reasonCodes := planningReasonCodes(approved); len(reasonCodes) > 0 {
		return spine.WorkItem{}, &CompletenessError{ReasonCodes: reasonCodes}
	}

	workItemID, err := s.IDs.NewWorkItemID()
	if err != nil {
		return spine.WorkItem{}, fmt.Errorf("new work item id: %w", err)
	}
	now := s.Clock.Now().UTC()
	workItem := workItemFromApprovedContract(workItemID, approved, now)
	event, err := s.workItemCreatedEvent(workItem)
	if err != nil {
		return spine.WorkItem{}, err
	}

	if err := s.WorkItems.Create(ctx, workItem); err != nil {
		if _, ok, lookupErr := s.WorkItems.GetByApprovedContractID(ctx, approved.ID); lookupErr != nil {
			return spine.WorkItem{}, fmt.Errorf("get work item by approved contract id after create failure: %w", lookupErr)
		} else if ok {
			return spine.WorkItem{}, ErrAlreadyPlanned
		}
		return spine.WorkItem{}, fmt.Errorf("create work item: %w", err)
	}
	if err := s.Events.Append(ctx, event); err != nil {
		return spine.WorkItem{}, fmt.Errorf("append work item created event: %w", err)
	}

	return workItem, nil
}

func workItemFromApprovedContract(id spine.WorkItemID, approved spine.ApprovedContract, createdAt time.Time) spine.WorkItem {
	return spine.WorkItem{
		ID:                   id,
		OrganizationID:       approved.OrganizationID,
		ProjectID:            approved.ProjectID,
		ApprovedContractID:   approved.ID,
		RepoBindingID:        approved.RepoBindingID,
		Title:                approved.Title,
		Summary:              approved.IntentSummary,
		Scope:                cloneStrings(approved.Scope),
		AcceptanceRefs:       indexedRefs("acceptance_criteria", approved.AcceptanceCriteria),
		ProofExpectationRefs: indexedRefs("proof_expectations", approved.ProofExpectations),
		Status:               spine.WorkItemStatusPlanned,
		SourceRefs:           sourceRefsForApprovedContract(approved),
		CreatedAt:            createdAt,
	}
}

func planningReasonCodes(approved spine.ApprovedContract) []string {
	var reasonCodes []string
	if strings.TrimSpace(string(approved.RepoBindingID)) == "" {
		reasonCodes = append(reasonCodes, ReasonMissingRepoBindingID)
	}
	if strings.TrimSpace(approved.Title) == "" {
		reasonCodes = append(reasonCodes, ReasonMissingTitle)
	}
	if strings.TrimSpace(approved.IntentSummary) == "" {
		reasonCodes = append(reasonCodes, ReasonMissingIntentSummary)
	}
	if len(nonBlankStrings(approved.Scope)) == 0 {
		reasonCodes = append(reasonCodes, ReasonMissingScope)
	}
	if len(nonBlankStrings(approved.AcceptanceCriteria)) == 0 {
		reasonCodes = append(reasonCodes, ReasonMissingAcceptanceCriteria)
	}
	if len(nonBlankStrings(approved.ProofExpectations)) == 0 {
		reasonCodes = append(reasonCodes, ReasonMissingProofExpectations)
	}
	return reasonCodes
}

func nonBlankStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, value)
		}
	}
	return out
}

func indexedRefs(prefix string, values []string) []string {
	refs := make([]string, 0, len(values))
	for i, value := range values {
		if strings.TrimSpace(value) == "" {
			continue
		}
		refs = append(refs, fmt.Sprintf("%s[%d]", prefix, i))
	}
	return refs
}

func sourceRefsForApprovedContract(approved spine.ApprovedContract) []spine.SourceRef {
	refs := make([]spine.SourceRef, 0, len(approved.SourceRefs)+1)
	refs = append(refs, spine.SourceRef{Kind: SourceRefKindApprovedContract, ID: string(approved.ID)})
	for _, ref := range approved.SourceRefs {
		if strings.TrimSpace(ref.Kind) == "" || strings.TrimSpace(ref.ID) == "" {
			continue
		}
		refs = append(refs, ref)
	}
	return refs
}

func cloneStrings(values []string) []string {
	if values == nil {
		return nil
	}
	return append([]string{}, values...)
}

type workItemCreatedPayload struct {
	WorkItemID           spine.WorkItemID         `json:"work_item_id"`
	ApprovedContractID   spine.ApprovedContractID `json:"approved_contract_id"`
	RepoBindingID        spine.RepoBindingID      `json:"repo_binding_id"`
	Title                string                   `json:"title"`
	Summary              string                   `json:"summary"`
	Scope                []string                 `json:"scope"`
	AcceptanceRefs       []string                 `json:"acceptance_refs"`
	ProofExpectationRefs []string                 `json:"proof_expectation_refs"`
	Status               spine.WorkItemStatus     `json:"status"`
	OwnerHint            string                   `json:"owner_hint,omitempty"`
	OrderIndex           *int                     `json:"order_index,omitempty"`
	SourceRefs           []spine.SourceRef        `json:"source_refs"`
	CreatedAt            time.Time                `json:"created_at"`
}

func (s *Service) workItemCreatedEvent(workItem spine.WorkItem) (spine.Event, error) {
	eventID, err := s.IDs.NewEventID()
	if err != nil {
		return spine.Event{}, fmt.Errorf("new work item created event id: %w", err)
	}

	payload, err := json.Marshal(workItemCreatedPayload{
		WorkItemID:           workItem.ID,
		ApprovedContractID:   workItem.ApprovedContractID,
		RepoBindingID:        workItem.RepoBindingID,
		Title:                workItem.Title,
		Summary:              workItem.Summary,
		Scope:                cloneStrings(workItem.Scope),
		AcceptanceRefs:       cloneStrings(workItem.AcceptanceRefs),
		ProofExpectationRefs: cloneStrings(workItem.ProofExpectationRefs),
		Status:               workItem.Status,
		OwnerHint:            workItem.OwnerHint,
		OrderIndex:           cloneIntPointer(workItem.OrderIndex),
		SourceRefs:           append([]spine.SourceRef(nil), workItem.SourceRefs...),
		CreatedAt:            workItem.CreatedAt,
	})
	if err != nil {
		return spine.Event{}, fmt.Errorf("marshal work item created event payload: %w", err)
	}

	return spine.Event{
		ID:             eventID,
		Type:           EventTypeWorkItemCreated,
		EntityType:     EntityTypeWorkItem,
		EntityID:       string(workItem.ID),
		OrganizationID: workItem.OrganizationID,
		ProjectID:      workItem.ProjectID,
		RepoBindingID:  workItem.RepoBindingID,
		Timestamp:      workItem.CreatedAt,
		Payload:        payload,
	}, nil
}

func cloneIntPointer(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func (s *Service) validateDependencies() error {
	if s.ApprovedContracts == nil {
		return errors.New("work item service approved contract store is nil")
	}
	if s.WorkItems == nil {
		return errors.New("work item service work item store is nil")
	}
	if s.Events == nil {
		return errors.New("work item service event log is nil")
	}
	if s.Clock == nil {
		return errors.New("work item service clock is nil")
	}
	if s.IDs == nil {
		return errors.New("work item service id generator is nil")
	}
	return nil
}

type SystemClock struct{}

func (SystemClock) Now() time.Time {
	return time.Now()
}

type UUIDGenerator struct{}

func (UUIDGenerator) NewWorkItemID() (spine.WorkItemID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return spine.WorkItemID(id.String()), nil
}

func (UUIDGenerator) NewEventID() (spine.EventID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return spine.EventID(id.String()), nil
}
