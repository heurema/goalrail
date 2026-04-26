package store

import (
	"context"
	"errors"
	"sync"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

var ErrGoalAlreadyExists = errors.New("goal already exists")

type GoalStore struct {
	mu       sync.RWMutex
	goals    map[spine.GoalID]spine.Goal
	byIntake map[spine.IntakeID]spine.GoalID
}

func NewGoalStore() *GoalStore {
	return &GoalStore{
		goals:    make(map[spine.GoalID]spine.Goal),
		byIntake: make(map[spine.IntakeID]spine.GoalID),
	}
}

func (s *GoalStore) Create(_ context.Context, created spine.Goal) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.goals[created.ID]; exists {
		return ErrGoalAlreadyExists
	}
	if _, exists := s.byIntake[created.IntakeID]; exists {
		return ErrGoalAlreadyExists
	}
	s.goals[created.ID] = cloneGoal(created)
	s.byIntake[created.IntakeID] = created.ID
	return nil
}

func (s *GoalStore) Get(_ context.Context, id spine.GoalID) (spine.Goal, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	created, ok := s.goals[id]
	if !ok {
		return spine.Goal{}, false, nil
	}
	return cloneGoal(created), true, nil
}

func (s *GoalStore) GetByIntakeID(_ context.Context, id spine.IntakeID) (spine.Goal, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	goalID, ok := s.byIntake[id]
	if !ok {
		return spine.Goal{}, false, nil
	}
	created, ok := s.goals[goalID]
	if !ok {
		return spine.Goal{}, false, nil
	}
	return cloneGoal(created), true, nil
}

func (s *GoalStore) UpdateState(ctx context.Context, id spine.GoalID, state spine.GoalState) (spine.Goal, bool, error) {
	return s.UpdateReadiness(ctx, id, state, nil)
}

func (s *GoalStore) UpdateReadiness(_ context.Context, id spine.GoalID, state spine.GoalState, reasonCodes []spine.GoalReadinessReasonCode) (spine.Goal, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	updated, ok := s.goals[id]
	if !ok {
		return spine.Goal{}, false, nil
	}
	updated.State = state
	updated.LastReadinessReasonCodes = cloneReadinessReasonCodes(reasonCodes)
	s.goals[id] = cloneGoal(updated)
	return cloneGoal(updated), true, nil
}

func cloneGoal(created spine.Goal) spine.Goal {
	if created.SourceRefs != nil {
		created.SourceRefs = append([]spine.SourceRef(nil), created.SourceRefs...)
	}
	if created.LastReadinessReasonCodes != nil {
		created.LastReadinessReasonCodes = cloneReadinessReasonCodes(created.LastReadinessReasonCodes)
	}
	return created
}

func cloneReadinessReasonCodes(reasonCodes []spine.GoalReadinessReasonCode) []spine.GoalReadinessReasonCode {
	if reasonCodes == nil {
		return nil
	}
	return append([]spine.GoalReadinessReasonCode(nil), reasonCodes...)
}
