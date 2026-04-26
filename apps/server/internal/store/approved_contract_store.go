package store

import (
	"context"
	"errors"
	"sync"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

var (
	ErrApprovedContractAlreadyExists   = errors.New("approved contract already exists")
	ErrApprovedContractAlreadyApproved = errors.New("contract draft already approved")
	ErrApprovedContractNotFound        = errors.New("approved contract not found")
)

type ApprovedContractStore struct {
	mu        sync.RWMutex
	contracts map[spine.ApprovedContractID]spine.ApprovedContract
	byDraft   map[spine.ContractDraftID]spine.ApprovedContractID
}

func NewApprovedContractStore() *ApprovedContractStore {
	return &ApprovedContractStore{
		contracts: make(map[spine.ApprovedContractID]spine.ApprovedContract),
		byDraft:   make(map[spine.ContractDraftID]spine.ApprovedContractID),
	}
}

func (s *ApprovedContractStore) Create(_ context.Context, approved spine.ApprovedContract) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.contracts[approved.ID]; exists {
		return ErrApprovedContractAlreadyExists
	}
	if _, exists := s.byDraft[approved.ContractDraftID]; exists {
		return ErrApprovedContractAlreadyApproved
	}

	s.contracts[approved.ID] = cloneApprovedContract(approved)
	s.byDraft[approved.ContractDraftID] = approved.ID
	return nil
}

func (s *ApprovedContractStore) Get(_ context.Context, id spine.ApprovedContractID) (spine.ApprovedContract, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	approved, ok := s.contracts[id]
	if !ok {
		return spine.ApprovedContract{}, false, nil
	}
	return cloneApprovedContract(approved), true, nil
}

func (s *ApprovedContractStore) GetByContractDraftID(_ context.Context, id spine.ContractDraftID) (spine.ApprovedContract, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	approvedID, ok := s.byDraft[id]
	if !ok {
		return spine.ApprovedContract{}, false, nil
	}
	approved, ok := s.contracts[approvedID]
	if !ok {
		return spine.ApprovedContract{}, false, nil
	}
	return cloneApprovedContract(approved), true, nil
}

func cloneApprovedContract(approved spine.ApprovedContract) spine.ApprovedContract {
	approved.Scope = cloneStringSlice(approved.Scope)
	approved.NonGoals = cloneStringSlice(approved.NonGoals)
	approved.Constraints = cloneStringSlice(approved.Constraints)
	approved.AcceptanceCriteria = cloneStringSlice(approved.AcceptanceCriteria)
	approved.ExpectedChecks = cloneStringSlice(approved.ExpectedChecks)
	approved.ProofExpectations = cloneStringSlice(approved.ProofExpectations)
	approved.RiskHints = cloneStringSlice(approved.RiskHints)
	approved.SourceRefs = append([]spine.SourceRef(nil), approved.SourceRefs...)
	return approved
}
