package store

import (
	"context"
	"errors"
	"sync"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

var (
	ErrContractSeedAlreadyExists = errors.New("contract seed already exists")
	ErrContractSeedAlreadySeeded = errors.New("contract seed already seeded")
)

type ContractSeedStore struct {
	mu     sync.RWMutex
	seeds  map[spine.ContractSeedID]spine.ContractSeed
	byGoal map[spine.GoalID]spine.ContractSeedID
}

func NewContractSeedStore() *ContractSeedStore {
	return &ContractSeedStore{
		seeds:  make(map[spine.ContractSeedID]spine.ContractSeed),
		byGoal: make(map[spine.GoalID]spine.ContractSeedID),
	}
}

func (s *ContractSeedStore) Create(_ context.Context, created spine.ContractSeed) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.seeds[created.ID]; exists {
		return ErrContractSeedAlreadyExists
	}
	if _, exists := s.byGoal[created.GoalID]; exists {
		return ErrContractSeedAlreadySeeded
	}

	s.seeds[created.ID] = cloneContractSeed(created)
	s.byGoal[created.GoalID] = created.ID
	return nil
}

func (s *ContractSeedStore) Get(_ context.Context, id spine.ContractSeedID) (spine.ContractSeed, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	seed, ok := s.seeds[id]
	if !ok {
		return spine.ContractSeed{}, false, nil
	}
	return cloneContractSeed(seed), true, nil
}

func (s *ContractSeedStore) GetByGoalID(_ context.Context, goalID spine.GoalID) (spine.ContractSeed, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	seedID, ok := s.byGoal[goalID]
	if !ok {
		return spine.ContractSeed{}, false, nil
	}
	seed, ok := s.seeds[seedID]
	if !ok {
		return spine.ContractSeed{}, false, nil
	}
	return cloneContractSeed(seed), true, nil
}

func (s *ContractSeedStore) Delete(_ context.Context, id spine.ContractSeedID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	seed, exists := s.seeds[id]
	if !exists {
		return nil
	}
	delete(s.seeds, id)
	if currentID, ok := s.byGoal[seed.GoalID]; ok && currentID == id {
		delete(s.byGoal, seed.GoalID)
	}
	return nil
}

func cloneContractSeed(seed spine.ContractSeed) spine.ContractSeed {
	if seed.SourceRefs != nil {
		seed.SourceRefs = append([]spine.SourceRef(nil), seed.SourceRefs...)
	}
	return seed
}
