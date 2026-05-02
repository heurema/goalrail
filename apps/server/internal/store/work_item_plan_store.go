package store

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

var (
	ErrWorkItemPlanAlreadyExists      = errors.New("work item plan already exists")
	ErrWorkItemPlanAlreadyPlanned     = errors.New("contract already has work item plan")
	ErrWorkItemPlanNotFound           = errors.New("work item plan not found")
	ErrWorkItemPlanProposalExists     = errors.New("work item plan proposal already exists")
	ErrWorkItemPlanAlreadyHasProposal = errors.New("work item plan already has proposal")
	ErrWorkItemPlanProposalNotFound   = errors.New("work item plan proposal not found")
)

type WorkItemPlanStore struct {
	mu         sync.RWMutex
	plans      map[spine.WorkItemPlanID]spine.WorkItemPlan
	byContract map[spine.ContractID]spine.WorkItemPlanID
}

func NewWorkItemPlanStore() *WorkItemPlanStore {
	return &WorkItemPlanStore{
		plans:      make(map[spine.WorkItemPlanID]spine.WorkItemPlan),
		byContract: make(map[spine.ContractID]spine.WorkItemPlanID),
	}
}

func (s *WorkItemPlanStore) Create(_ context.Context, plan spine.WorkItemPlan) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.plans[plan.ID]; exists {
		return ErrWorkItemPlanAlreadyExists
	}
	if _, exists := s.byContract[plan.ContractID]; exists {
		return ErrWorkItemPlanAlreadyPlanned
	}

	s.plans[plan.ID] = cloneWorkItemPlan(plan)
	s.byContract[plan.ContractID] = plan.ID
	return nil
}

func (s *WorkItemPlanStore) Get(_ context.Context, id spine.WorkItemPlanID) (spine.WorkItemPlan, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	plan, ok := s.plans[id]
	if !ok {
		return spine.WorkItemPlan{}, false, nil
	}
	return cloneWorkItemPlan(plan), true, nil
}

func (s *WorkItemPlanStore) GetByContractID(_ context.Context, id spine.ContractID) (spine.WorkItemPlan, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	planID, ok := s.byContract[id]
	if !ok {
		return spine.WorkItemPlan{}, false, nil
	}
	plan, ok := s.plans[planID]
	if !ok {
		return spine.WorkItemPlan{}, false, nil
	}
	return cloneWorkItemPlan(plan), true, nil
}

func (s *WorkItemPlanStore) MarkProposalSubmitted(_ context.Context, id spine.WorkItemPlanID, updatedAt time.Time) error {
	return s.updateState(id, spine.WorkItemPlanStateProposalSubmitted, updatedAt)
}

func (s *WorkItemPlanStore) MarkAccepted(_ context.Context, id spine.WorkItemPlanID, updatedAt time.Time) error {
	return s.updateState(id, spine.WorkItemPlanStateAccepted, updatedAt)
}

func (s *WorkItemPlanStore) updateState(id spine.WorkItemPlanID, state spine.WorkItemPlanState, updatedAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	plan, exists := s.plans[id]
	if !exists {
		return ErrWorkItemPlanNotFound
	}
	plan.State = state
	plan.UpdatedAt = updatedAt.UTC()
	s.plans[id] = cloneWorkItemPlan(plan)
	return nil
}

type WorkItemPlanProposalStore struct {
	mu        sync.RWMutex
	proposals map[spine.WorkItemPlanProposalID]spine.WorkItemPlanProposal
	byPlan    map[spine.WorkItemPlanID]spine.WorkItemPlanProposalID
}

func NewWorkItemPlanProposalStore() *WorkItemPlanProposalStore {
	return &WorkItemPlanProposalStore{
		proposals: make(map[spine.WorkItemPlanProposalID]spine.WorkItemPlanProposal),
		byPlan:    make(map[spine.WorkItemPlanID]spine.WorkItemPlanProposalID),
	}
}

func (s *WorkItemPlanProposalStore) Create(_ context.Context, proposal spine.WorkItemPlanProposal) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.proposals[proposal.ID]; exists {
		return ErrWorkItemPlanProposalExists
	}
	if _, exists := s.byPlan[proposal.PlanID]; exists {
		return ErrWorkItemPlanAlreadyHasProposal
	}

	s.proposals[proposal.ID] = cloneWorkItemPlanProposal(proposal)
	s.byPlan[proposal.PlanID] = proposal.ID
	return nil
}

func (s *WorkItemPlanProposalStore) Get(_ context.Context, id spine.WorkItemPlanProposalID) (spine.WorkItemPlanProposal, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	proposal, ok := s.proposals[id]
	if !ok {
		return spine.WorkItemPlanProposal{}, false, nil
	}
	return cloneWorkItemPlanProposal(proposal), true, nil
}

func (s *WorkItemPlanProposalStore) GetByPlanID(_ context.Context, id spine.WorkItemPlanID) (spine.WorkItemPlanProposal, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	proposalID, ok := s.byPlan[id]
	if !ok {
		return spine.WorkItemPlanProposal{}, false, nil
	}
	proposal, ok := s.proposals[proposalID]
	if !ok {
		return spine.WorkItemPlanProposal{}, false, nil
	}
	return cloneWorkItemPlanProposal(proposal), true, nil
}

func (s *WorkItemPlanProposalStore) MarkAccepted(_ context.Context, id spine.WorkItemPlanProposalID, acceptedBy spine.ActorRef, acceptedAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	proposal, exists := s.proposals[id]
	if !exists {
		return ErrWorkItemPlanProposalNotFound
	}
	proposal.State = spine.WorkItemProposalStateAccepted
	proposal.AcceptedBy = cloneActorRefPointer(&acceptedBy)
	acceptedAt = acceptedAt.UTC()
	proposal.AcceptedAt = &acceptedAt
	proposal.UpdatedAt = acceptedAt
	s.proposals[id] = cloneWorkItemPlanProposal(proposal)
	return nil
}

func cloneWorkItemPlan(plan spine.WorkItemPlan) spine.WorkItemPlan {
	return plan
}

func cloneWorkItemPlanProposal(proposal spine.WorkItemPlanProposal) spine.WorkItemPlanProposal {
	proposal.SubmittedBy = cloneActorRef(proposal.SubmittedBy)
	proposal.Planner = cloneAnyMap(proposal.Planner)
	proposal.SourceSnapshotRefs = append([]spine.SourceRef(nil), proposal.SourceSnapshotRefs...)
	proposal.ProposedTasks = cloneProposedWorkItems(proposal.ProposedTasks)
	proposal.AcceptedBy = cloneActorRefPointer(proposal.AcceptedBy)
	if proposal.AcceptedAt != nil {
		acceptedAt := proposal.AcceptedAt.UTC()
		proposal.AcceptedAt = &acceptedAt
	}
	return proposal
}

func cloneProposedWorkItems(items []spine.ProposedWorkItem) []spine.ProposedWorkItem {
	out := make([]spine.ProposedWorkItem, len(items))
	for i, item := range items {
		out[i] = item
		out[i].Scope = cloneStringSlice(item.Scope)
		out[i].AcceptanceRefs = cloneStringSlice(item.AcceptanceRefs)
		out[i].ProofExpectationRefs = cloneStringSlice(item.ProofExpectationRefs)
		out[i].SourceRefs = append([]spine.SourceRef(nil), item.SourceRefs...)
		if item.OrderIndex != nil {
			orderIndex := *item.OrderIndex
			out[i].OrderIndex = &orderIndex
		}
	}
	return out
}

func cloneActorRef(value spine.ActorRef) spine.ActorRef {
	return value
}

func cloneActorRefPointer(value *spine.ActorRef) *spine.ActorRef {
	if value == nil {
		return nil
	}
	cloned := cloneActorRef(*value)
	return &cloned
}

func cloneAnyMap(values map[string]any) map[string]any {
	if values == nil {
		return nil
	}
	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}
