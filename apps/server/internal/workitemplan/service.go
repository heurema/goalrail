package workitemplan

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/heurema/goalrail/apps/server/internal/spine"
	"github.com/heurema/goalrail/apps/server/internal/workitem"
)

const (
	SourceRefKindProposal = "proposal"
)

var (
	ErrContractNotFound                = errors.New("contract not found")
	ErrInvalidContractState            = errors.New("contract is not ready for planning")
	ErrContractMissingApprovedSnapshot = errors.New("contract approved snapshot is missing")
	ErrApprovedContractNotFound        = errors.New("approved contract not found")
	ErrInvalidApprovedContractState    = errors.New("approved contract is not approved")
	ErrAlreadyPlanned                  = errors.New("contract already has work item plan")
	ErrPlanNotFound                    = errors.New("work item plan not found")
	ErrInvalidPlanState                = errors.New("work item plan state is not valid for this transition")
	ErrAlreadyProposed                 = errors.New("work item plan already has proposal")
	ErrProposalNotFound                = errors.New("work item plan proposal not found")
	ErrInvalidProposalState            = errors.New("work item plan proposal state is not valid for this transition")
	ErrAlreadyAccepted                 = errors.New("work item plan proposal already accepted")
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

type ContractReader interface {
	Get(context.Context, spine.ContractID) (spine.Contract, bool, error)
}

type ApprovedContractReader interface {
	Get(context.Context, spine.ApprovedContractID) (spine.ApprovedContract, bool, error)
}

type PlanStore interface {
	Create(context.Context, spine.WorkItemPlan) error
	Get(context.Context, spine.WorkItemPlanID) (spine.WorkItemPlan, bool, error)
	GetByContractID(context.Context, spine.ContractID) (spine.WorkItemPlan, bool, error)
	MarkProposalSubmitted(context.Context, spine.WorkItemPlanID, time.Time) error
	MarkAccepted(context.Context, spine.WorkItemPlanID, time.Time) error
}

type ProposalStore interface {
	Create(context.Context, spine.WorkItemPlanProposal) error
	Get(context.Context, spine.WorkItemPlanProposalID) (spine.WorkItemPlanProposal, bool, error)
	GetByPlanID(context.Context, spine.WorkItemPlanID) (spine.WorkItemPlanProposal, bool, error)
	MarkAccepted(context.Context, spine.WorkItemPlanProposalID, spine.ActorRef, time.Time) error
}

type WorkItemStore interface {
	Create(context.Context, spine.WorkItem) error
	Get(context.Context, spine.WorkItemID) (spine.WorkItem, bool, error)
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
	NewWorkItemPlanID() (spine.WorkItemPlanID, error)
	NewWorkItemPlanProposalID() (spine.WorkItemPlanProposalID, error)
	NewWorkItemID() (spine.WorkItemID, error)
	NewEventID() (spine.EventID, error)
}

type Service struct {
	Contracts         ContractReader
	ApprovedContracts ApprovedContractReader
	Plans             PlanStore
	Proposals         ProposalStore
	WorkItems         WorkItemStore
	Events            EventLog
	TxRunner          TransactionRunner
	Clock             Clock
	IDs               IDGenerator
}

func NewService(contracts ContractReader, approvedContracts ApprovedContractReader, plans PlanStore, proposals ProposalStore, workItems WorkItemStore, events EventLog, txRunner TransactionRunner, clock Clock, ids IDGenerator) *Service {
	return &Service{
		Contracts:         contracts,
		ApprovedContracts: approvedContracts,
		Plans:             plans,
		Proposals:         proposals,
		WorkItems:         workItems,
		Events:            events,
		TxRunner:          txRunner,
		Clock:             clock,
		IDs:               ids,
	}
}

func (s *Service) CreatePlan(ctx context.Context, contractID spine.ContractID, input spine.WorkItemPlanCreateRequest) (spine.WorkItemPlan, error) {
	if err := validateActor("requested_by", input.RequestedBy); err != nil {
		return spine.WorkItemPlan{}, err
	}

	contract, approved, err := s.loadApprovedContract(ctx, contractID)
	if err != nil {
		return spine.WorkItemPlan{}, err
	}
	if _, ok, err := s.Plans.GetByContractID(ctx, contract.ID); err != nil {
		return spine.WorkItemPlan{}, fmt.Errorf("get work item plan by contract id: %w", err)
	} else if ok {
		return spine.WorkItemPlan{}, ErrAlreadyPlanned
	}

	planID, err := s.IDs.NewWorkItemPlanID()
	if err != nil {
		return spine.WorkItemPlan{}, fmt.Errorf("new work item plan id: %w", err)
	}
	now := s.Clock.Now().UTC()
	plan := spine.WorkItemPlan{
		ID:                 planID,
		OrganizationID:     contract.OrganizationID,
		ProjectID:          contract.ProjectID,
		ContractID:         contract.ID,
		ApprovedContractID: approved.ID,
		RepoBindingID:      contract.RepoBindingID,
		State:              spine.WorkItemPlanStateQueued,
		RequestedBy:        input.RequestedBy,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := s.Plans.Create(ctx, plan); err != nil {
		if _, ok, lookupErr := s.Plans.GetByContractID(ctx, contract.ID); lookupErr != nil {
			return spine.WorkItemPlan{}, fmt.Errorf("get work item plan by contract id after create failure: %w", lookupErr)
		} else if ok {
			return spine.WorkItemPlan{}, ErrAlreadyPlanned
		}
		return spine.WorkItemPlan{}, fmt.Errorf("create work item plan: %w", err)
	}
	return plan, nil
}

func (s *Service) GetPlan(ctx context.Context, id spine.WorkItemPlanID) (spine.WorkItemPlan, error) {
	plan, ok, err := s.Plans.Get(ctx, id)
	if err != nil {
		return spine.WorkItemPlan{}, fmt.Errorf("get work item plan: %w", err)
	}
	if !ok {
		return spine.WorkItemPlan{}, ErrPlanNotFound
	}
	return plan, nil
}

func (s *Service) SubmitProposal(ctx context.Context, planID spine.WorkItemPlanID, input spine.WorkItemPlanProposalSubmitRequest) (spine.WorkItemPlanProposal, error) {
	plan, err := s.GetPlan(ctx, planID)
	if err != nil {
		return spine.WorkItemPlanProposal{}, err
	}
	if _, ok, err := s.Proposals.GetByPlanID(ctx, plan.ID); err != nil {
		return spine.WorkItemPlanProposal{}, fmt.Errorf("get work item plan proposal by plan id: %w", err)
	} else if ok {
		return spine.WorkItemPlanProposal{}, ErrAlreadyProposed
	}
	if plan.State != spine.WorkItemPlanStateQueued {
		return spine.WorkItemPlanProposal{}, fmt.Errorf("%w: %s", ErrInvalidPlanState, plan.State)
	}
	if err := validateProposalInput(input); err != nil {
		return spine.WorkItemPlanProposal{}, err
	}

	proposalID, err := s.IDs.NewWorkItemPlanProposalID()
	if err != nil {
		return spine.WorkItemPlanProposal{}, fmt.Errorf("new work item plan proposal id: %w", err)
	}
	now := s.Clock.Now().UTC()
	proposedTasks := cloneProposedTasksWithOrder(input.ProposedTasks)
	proposal := spine.WorkItemPlanProposal{
		ID:                 proposalID,
		PlanID:             plan.ID,
		OrganizationID:     plan.OrganizationID,
		ProjectID:          plan.ProjectID,
		ContractID:         plan.ContractID,
		ApprovedContractID: plan.ApprovedContractID,
		RepoBindingID:      plan.RepoBindingID,
		State:              spine.WorkItemProposalStateSubmitted,
		SubmittedBy:        input.SubmittedBy,
		Planner:            nonNilPlanner(input.Planner),
		SourceSnapshotRefs: cloneValidSourceRefs(input.SourceSnapshotRefs),
		Rationale:          input.Rationale,
		ProposedTasks:      proposedTasks,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := s.Proposals.Create(ctx, proposal); err != nil {
		if _, ok, lookupErr := s.Proposals.GetByPlanID(ctx, plan.ID); lookupErr != nil {
			return spine.WorkItemPlanProposal{}, fmt.Errorf("get work item plan proposal by plan id after create failure: %w", lookupErr)
		} else if ok {
			return spine.WorkItemPlanProposal{}, ErrAlreadyProposed
		}
		return spine.WorkItemPlanProposal{}, fmt.Errorf("create work item plan proposal: %w", err)
	}
	if err := s.Plans.MarkProposalSubmitted(ctx, plan.ID, now); err != nil {
		return spine.WorkItemPlanProposal{}, fmt.Errorf("mark work item plan proposal submitted: %w", err)
	}
	return proposal, nil
}

func (s *Service) GetProposal(ctx context.Context, id spine.WorkItemPlanProposalID) (spine.WorkItemPlanProposal, error) {
	proposal, ok, err := s.Proposals.Get(ctx, id)
	if err != nil {
		return spine.WorkItemPlanProposal{}, fmt.Errorf("get work item plan proposal: %w", err)
	}
	if !ok {
		return spine.WorkItemPlanProposal{}, ErrProposalNotFound
	}
	return proposal, nil
}

func (s *Service) AcceptProposal(ctx context.Context, proposalID spine.WorkItemPlanProposalID, input spine.WorkItemPlanAcceptanceRequest) (spine.WorkItemPlanAcceptanceResult, error) {
	if err := validateActor("accepted_by", input.AcceptedBy); err != nil {
		return spine.WorkItemPlanAcceptanceResult{}, err
	}
	proposal, err := s.GetProposal(ctx, proposalID)
	if err != nil {
		return spine.WorkItemPlanAcceptanceResult{}, err
	}
	if proposal.State == spine.WorkItemProposalStateAccepted {
		return spine.WorkItemPlanAcceptanceResult{}, ErrAlreadyAccepted
	}
	if proposal.State != spine.WorkItemProposalStateSubmitted {
		return spine.WorkItemPlanAcceptanceResult{}, fmt.Errorf("%w: %s", ErrInvalidProposalState, proposal.State)
	}
	plan, err := s.GetPlan(ctx, proposal.PlanID)
	if err != nil {
		return spine.WorkItemPlanAcceptanceResult{}, err
	}
	if plan.State != spine.WorkItemPlanStateProposalSubmitted {
		return spine.WorkItemPlanAcceptanceResult{}, fmt.Errorf("%w: %s", ErrInvalidPlanState, plan.State)
	}

	acceptedAt := s.Clock.Now().UTC()
	items, events, err := s.materializedWorkItems(proposal, input.AcceptedBy, acceptedAt)
	if err != nil {
		return spine.WorkItemPlanAcceptanceResult{}, err
	}
	if err := s.TxRunner.RunReadCommitted(ctx, func(txCtx context.Context) error {
		for _, item := range items {
			if err := s.WorkItems.Create(txCtx, item); err != nil {
				return err
			}
		}
		if err := s.Proposals.MarkAccepted(txCtx, proposal.ID, input.AcceptedBy, acceptedAt); err != nil {
			return err
		}
		if err := s.Plans.MarkAccepted(txCtx, plan.ID, acceptedAt); err != nil {
			return err
		}
		for _, event := range events {
			if err := s.Events.Append(txCtx, event); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return spine.WorkItemPlanAcceptanceResult{}, fmt.Errorf("accept work item plan proposal transactionally: %w", err)
	}

	ids := make([]spine.WorkItemID, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return spine.WorkItemPlanAcceptanceResult{
		ProposalID:     proposal.ID,
		PlanID:         plan.ID,
		ContractID:     proposal.ContractID,
		State:          spine.WorkItemProposalStateAccepted,
		AcceptedBy:     input.AcceptedBy,
		AcceptedAt:     acceptedAt,
		CreatedTaskIDs: ids,
	}, nil
}

func (s *Service) loadApprovedContract(ctx context.Context, contractID spine.ContractID) (spine.Contract, spine.ApprovedContract, error) {
	contract, ok, err := s.Contracts.Get(ctx, contractID)
	if err != nil {
		return spine.Contract{}, spine.ApprovedContract{}, fmt.Errorf("get contract: %w", err)
	}
	if !ok {
		return spine.Contract{}, spine.ApprovedContract{}, ErrContractNotFound
	}
	if contract.State != spine.ContractStateApproved {
		return spine.Contract{}, spine.ApprovedContract{}, fmt.Errorf("%w: %s", ErrInvalidContractState, contract.State)
	}
	if contract.ApprovedSnapshotID == nil || strings.TrimSpace(string(*contract.ApprovedSnapshotID)) == "" {
		return spine.Contract{}, spine.ApprovedContract{}, ErrContractMissingApprovedSnapshot
	}
	approved, ok, err := s.ApprovedContracts.Get(ctx, *contract.ApprovedSnapshotID)
	if err != nil {
		return spine.Contract{}, spine.ApprovedContract{}, fmt.Errorf("get approved contract: %w", err)
	}
	if !ok {
		return spine.Contract{}, spine.ApprovedContract{}, ErrApprovedContractNotFound
	}
	if approved.State != spine.ApprovedContractStateApproved {
		return spine.Contract{}, spine.ApprovedContract{}, fmt.Errorf("%w: %s", ErrInvalidApprovedContractState, approved.State)
	}
	return contract, approved, nil
}

func (s *Service) materializedWorkItems(proposal spine.WorkItemPlanProposal, acceptedBy spine.ActorRef, acceptedAt time.Time) ([]spine.WorkItem, []spine.Event, error) {
	items := make([]spine.WorkItem, 0, len(proposal.ProposedTasks))
	events := make([]spine.Event, 0, len(proposal.ProposedTasks))
	for index, proposed := range proposal.ProposedTasks {
		itemID, err := s.IDs.NewWorkItemID()
		if err != nil {
			return nil, nil, fmt.Errorf("new work item id: %w", err)
		}
		orderIndex := index
		if proposed.OrderIndex != nil {
			orderIndex = *proposed.OrderIndex
		}
		item := spine.WorkItem{
			ID:                   itemID,
			OrganizationID:       proposal.OrganizationID,
			ProjectID:            proposal.ProjectID,
			ContractID:           proposal.ContractID,
			ApprovedContractID:   proposal.ApprovedContractID,
			PlanID:               proposal.PlanID,
			ProposalID:           proposal.ID,
			RepoBindingID:        proposal.RepoBindingID,
			Title:                proposed.Title,
			Summary:              proposed.Summary,
			Scope:                cloneNonBlankStrings(proposed.Scope),
			AcceptanceRefs:       cloneNonBlankStrings(proposed.AcceptanceRefs),
			ProofExpectationRefs: cloneNonBlankStrings(proposed.ProofExpectationRefs),
			Status:               spine.WorkItemStatusPlanned,
			OwnerHint:            proposed.OwnerHint,
			OrderIndex:           &orderIndex,
			SourceRefs:           sourceRefsForProposalTask(proposal, proposed),
			CreatedAt:            acceptedAt,
		}
		event, err := s.workItemCreatedEvent(item, acceptedBy)
		if err != nil {
			return nil, nil, err
		}
		items = append(items, item)
		events = append(events, event)
	}
	return items, events, nil
}

func (s *Service) workItemCreatedEvent(item spine.WorkItem, acceptedBy spine.ActorRef) (spine.Event, error) {
	eventID, err := s.IDs.NewEventID()
	if err != nil {
		return spine.Event{}, fmt.Errorf("new event id: %w", err)
	}
	payload := struct {
		WorkItemID         spine.WorkItemID             `json:"work_item_id"`
		ContractID         spine.ContractID             `json:"contract_id"`
		ApprovedContractID spine.ApprovedContractID     `json:"approved_contract_id"`
		PlanID             spine.WorkItemPlanID         `json:"plan_id"`
		ProposalID         spine.WorkItemPlanProposalID `json:"proposal_id"`
		RepoBindingID      spine.RepoBindingID          `json:"repo_binding_id"`
		Status             spine.WorkItemStatus         `json:"status"`
		AcceptedBy         spine.ActorRef               `json:"accepted_by"`
		SourceRefs         []spine.SourceRef            `json:"source_refs"`
		CreatedAt          time.Time                    `json:"created_at"`
	}{
		WorkItemID:         item.ID,
		ContractID:         item.ContractID,
		ApprovedContractID: item.ApprovedContractID,
		PlanID:             item.PlanID,
		ProposalID:         item.ProposalID,
		RepoBindingID:      item.RepoBindingID,
		Status:             item.Status,
		AcceptedBy:         acceptedBy,
		SourceRefs:         append([]spine.SourceRef(nil), item.SourceRefs...),
		CreatedAt:          item.CreatedAt,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return spine.Event{}, fmt.Errorf("marshal work item created event payload: %w", err)
	}
	return spine.Event{
		ID:             eventID,
		Type:           workitem.EventTypeWorkItemCreated,
		EntityType:     workitem.EntityTypeWorkItem,
		EntityID:       string(item.ID),
		OrganizationID: item.OrganizationID,
		ProjectID:      item.ProjectID,
		RepoBindingID:  item.RepoBindingID,
		Timestamp:      item.CreatedAt,
		Payload:        body,
	}, nil
}

func validateProposalInput(input spine.WorkItemPlanProposalSubmitRequest) error {
	if err := validateActor("submitted_by", input.SubmittedBy); err != nil {
		return err
	}
	if len(input.ProposedTasks) == 0 {
		return &ValidationError{Field: "proposed_tasks", Message: "must contain at least one task"}
	}
	for i, task := range input.ProposedTasks {
		prefix := fmt.Sprintf("proposed_tasks[%d]", i)
		if strings.TrimSpace(task.Title) == "" {
			return &ValidationError{Field: prefix + ".title", Message: "is required"}
		}
		if strings.TrimSpace(task.Summary) == "" {
			return &ValidationError{Field: prefix + ".summary", Message: "is required"}
		}
		if len(cloneNonBlankStrings(task.Scope)) == 0 {
			return &ValidationError{Field: prefix + ".scope", Message: "must contain at least one nonblank item"}
		}
		if len(cloneNonBlankStrings(task.AcceptanceRefs)) == 0 {
			return &ValidationError{Field: prefix + ".acceptance_refs", Message: "must contain at least one nonblank item"}
		}
		if len(cloneNonBlankStrings(task.ProofExpectationRefs)) == 0 {
			return &ValidationError{Field: prefix + ".proof_expectation_refs", Message: "must contain at least one nonblank item"}
		}
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

func cloneProposedTasksWithOrder(tasks []spine.ProposedWorkItem) []spine.ProposedWorkItem {
	out := make([]spine.ProposedWorkItem, 0, len(tasks))
	for i, task := range tasks {
		task.Scope = cloneNonBlankStrings(task.Scope)
		task.AcceptanceRefs = cloneNonBlankStrings(task.AcceptanceRefs)
		task.ProofExpectationRefs = cloneNonBlankStrings(task.ProofExpectationRefs)
		task.SourceRefs = cloneValidSourceRefs(task.SourceRefs)
		if task.OrderIndex == nil {
			orderIndex := i
			task.OrderIndex = &orderIndex
		} else {
			orderIndex := *task.OrderIndex
			task.OrderIndex = &orderIndex
		}
		out = append(out, task)
	}
	return out
}

func sourceRefsForProposalTask(proposal spine.WorkItemPlanProposal, task spine.ProposedWorkItem) []spine.SourceRef {
	refs := []spine.SourceRef{
		{Kind: workitem.SourceRefKindApprovedContract, ID: string(proposal.ApprovedContractID)},
		{Kind: SourceRefKindProposal, ID: string(proposal.ID)},
	}
	refs = append(refs, cloneValidSourceRefs(task.SourceRefs)...)
	return dedupeSourceRefs(refs)
}

func cloneValidSourceRefs(refs []spine.SourceRef) []spine.SourceRef {
	out := make([]spine.SourceRef, 0, len(refs))
	for _, ref := range refs {
		if strings.TrimSpace(ref.Kind) == "" || strings.TrimSpace(ref.ID) == "" {
			continue
		}
		out = append(out, ref)
	}
	return out
}

func dedupeSourceRefs(refs []spine.SourceRef) []spine.SourceRef {
	seen := make(map[string]bool, len(refs))
	out := make([]spine.SourceRef, 0, len(refs))
	for _, ref := range refs {
		key := ref.Kind + "\x00" + ref.ID
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, ref)
	}
	return out
}

func cloneNonBlankStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, value)
		}
	}
	return out
}

func nonNilPlanner(planner map[string]any) map[string]any {
	if planner == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(planner))
	for key, value := range planner {
		out[key] = value
	}
	return out
}

type SystemClock struct{}

func (SystemClock) Now() time.Time {
	return time.Now()
}

type UUIDGenerator struct{}

func (UUIDGenerator) NewWorkItemPlanID() (spine.WorkItemPlanID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return spine.WorkItemPlanID(id.String()), nil
}

func (UUIDGenerator) NewWorkItemPlanProposalID() (spine.WorkItemPlanProposalID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return spine.WorkItemPlanProposalID(id.String()), nil
}

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
