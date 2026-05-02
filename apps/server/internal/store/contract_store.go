package store

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

var (
	ErrContractAlreadyExists = errors.New("contract already exists")
	ErrContractAlreadySeeded = errors.New("goal already has contract")
	ErrContractNotFound      = errors.New("contract not found")
)

type ContractStore struct {
	mu        sync.RWMutex
	contracts map[spine.ContractID]spine.Contract
	byGoal    map[spine.GoalID]spine.ContractID
}

func NewContractStore() *ContractStore {
	return &ContractStore{
		contracts: make(map[spine.ContractID]spine.Contract),
		byGoal:    make(map[spine.GoalID]spine.ContractID),
	}
}

func (s *ContractStore) Create(_ context.Context, contract spine.Contract) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.contracts[contract.ID]; exists {
		return ErrContractAlreadyExists
	}
	if _, exists := s.byGoal[contract.GoalID]; exists {
		return ErrContractAlreadySeeded
	}

	s.contracts[contract.ID] = cloneContract(contract)
	s.byGoal[contract.GoalID] = contract.ID
	return nil
}

func (s *ContractStore) Get(_ context.Context, id spine.ContractID) (spine.Contract, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	contract, ok := s.contracts[id]
	if !ok {
		return spine.Contract{}, false, nil
	}
	return cloneContract(contract), true, nil
}

func (s *ContractStore) GetByGoalID(_ context.Context, goalID spine.GoalID) (spine.Contract, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	contractID, ok := s.byGoal[goalID]
	if !ok {
		return spine.Contract{}, false, nil
	}
	contract, ok := s.contracts[contractID]
	if !ok {
		return spine.Contract{}, false, nil
	}
	return cloneContract(contract), true, nil
}

func (s *ContractStore) MarkDraftCreated(_ context.Context, contractID spine.ContractID, draftID spine.ContractDraftID, updatedAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	contract, exists := s.contracts[contractID]
	if !exists {
		return ErrContractNotFound
	}
	contract.State = spine.ContractStateDraft
	contract.CurrentDraftID = cloneContractDraftIDPointer(&draftID)
	contract.UpdatedAt = updatedAt.UTC()
	s.contracts[contractID] = cloneContract(contract)
	return nil
}

func (s *ContractStore) MarkReadyForApproval(_ context.Context, contractID spine.ContractID, updatedAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	contract, exists := s.contracts[contractID]
	if !exists {
		return ErrContractNotFound
	}
	contract.State = spine.ContractStateReadyForApproval
	contract.UpdatedAt = updatedAt.UTC()
	s.contracts[contractID] = cloneContract(contract)
	return nil
}

func (s *ContractStore) MarkApproved(_ context.Context, contractID spine.ContractID, approvedSnapshotID spine.ApprovedContractID, updatedAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	contract, exists := s.contracts[contractID]
	if !exists {
		return ErrContractNotFound
	}
	contract.State = spine.ContractStateApproved
	contract.ApprovedSnapshotID = cloneApprovedContractIDPointer(&approvedSnapshotID)
	contract.UpdatedAt = updatedAt.UTC()
	s.contracts[contractID] = cloneContract(contract)
	return nil
}

func cloneContract(contract spine.Contract) spine.Contract {
	contract.CurrentSeedID = cloneContractSeedIDPointer(contract.CurrentSeedID)
	contract.CurrentDraftID = cloneContractDraftIDPointer(contract.CurrentDraftID)
	contract.ApprovedSnapshotID = cloneApprovedContractIDPointer(contract.ApprovedSnapshotID)
	return contract
}

func cloneContractSeedIDPointer(value *spine.ContractSeedID) *spine.ContractSeedID {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneContractDraftIDPointer(value *spine.ContractDraftID) *spine.ContractDraftID {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneApprovedContractIDPointer(value *spine.ApprovedContractID) *spine.ApprovedContractID {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
