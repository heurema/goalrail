package store

import (
	"context"
	"errors"
	"sync"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

var (
	ErrContractDraftAlreadyExists  = errors.New("contract draft already exists")
	ErrContractDraftAlreadyDrafted = errors.New("contract seed already drafted")
	ErrContractDraftNotFound       = errors.New("contract draft not found")
)

type ContractDraftStore struct {
	mu     sync.RWMutex
	drafts map[spine.ContractDraftID]spine.ContractDraft
	bySeed map[spine.ContractSeedID]spine.ContractDraftID
}

func NewContractDraftStore() *ContractDraftStore {
	return &ContractDraftStore{
		drafts: make(map[spine.ContractDraftID]spine.ContractDraft),
		bySeed: make(map[spine.ContractSeedID]spine.ContractDraftID),
	}
}

func (s *ContractDraftStore) Create(_ context.Context, created spine.ContractDraft) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.drafts[created.ID]; exists {
		return ErrContractDraftAlreadyExists
	}
	if _, exists := s.bySeed[created.ContractSeedID]; exists {
		return ErrContractDraftAlreadyDrafted
	}

	s.drafts[created.ID] = cloneContractDraft(created)
	s.bySeed[created.ContractSeedID] = created.ID
	return nil
}

func (s *ContractDraftStore) Update(_ context.Context, updated spine.ContractDraft) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.drafts[updated.ID]
	if !exists {
		return ErrContractDraftNotFound
	}

	updated.ContractSeedID = existing.ContractSeedID
	updated.GoalID = existing.GoalID
	updated.RepoBindingID = existing.RepoBindingID
	updated.SourceRefs = append([]spine.SourceRef(nil), existing.SourceRefs...)
	updated.State = existing.State
	updated.CreatedAt = existing.CreatedAt

	s.drafts[updated.ID] = cloneContractDraft(updated)
	return nil
}

func (s *ContractDraftStore) MarkReadyForApproval(_ context.Context, updated spine.ContractDraft) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.drafts[updated.ID]
	if !exists {
		return ErrContractDraftNotFound
	}

	existing.State = spine.ContractDraftStateReadyForApproval
	s.drafts[updated.ID] = cloneContractDraft(existing)
	return nil
}

func (s *ContractDraftStore) Get(_ context.Context, id spine.ContractDraftID) (spine.ContractDraft, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	draft, ok := s.drafts[id]
	if !ok {
		return spine.ContractDraft{}, false, nil
	}
	return cloneContractDraft(draft), true, nil
}

func (s *ContractDraftStore) GetByContractSeedID(_ context.Context, seedID spine.ContractSeedID) (spine.ContractDraft, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	draftID, ok := s.bySeed[seedID]
	if !ok {
		return spine.ContractDraft{}, false, nil
	}
	draft, ok := s.drafts[draftID]
	if !ok {
		return spine.ContractDraft{}, false, nil
	}
	return cloneContractDraft(draft), true, nil
}

func cloneContractDraft(draft spine.ContractDraft) spine.ContractDraft {
	draft.ProposedScope = cloneStringSlice(draft.ProposedScope)
	draft.ProposedNonGoals = cloneStringSlice(draft.ProposedNonGoals)
	draft.ProposedConstraints = cloneStringSlice(draft.ProposedConstraints)
	draft.ProposedAcceptanceCriteria = cloneStringSlice(draft.ProposedAcceptanceCriteria)
	draft.ProposedExpectedChecks = cloneStringSlice(draft.ProposedExpectedChecks)
	draft.ProposedProofExpectations = cloneStringSlice(draft.ProposedProofExpectations)
	draft.RiskHints = cloneStringSlice(draft.RiskHints)
	draft.SourceRefs = append([]spine.SourceRef(nil), draft.SourceRefs...)
	return draft
}

func cloneStringSlice(values []string) []string {
	if values == nil {
		return nil
	}
	return append([]string{}, values...)
}
