package store

import (
	"context"
	"errors"
	"sync"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

var (
	ErrClarificationRequestAlreadyExists = errors.New("clarification request already exists")
	ErrClarificationRequestAlreadyOpen   = errors.New("clarification request already open")
)

type ClarificationStore struct {
	mu         sync.RWMutex
	requests   map[spine.ClarificationRequestID]spine.ClarificationRequest
	openByGoal map[spine.GoalID]spine.ClarificationRequestID
}

func NewClarificationStore() *ClarificationStore {
	return &ClarificationStore{
		requests:   make(map[spine.ClarificationRequestID]spine.ClarificationRequest),
		openByGoal: make(map[spine.GoalID]spine.ClarificationRequestID),
	}
}

func (s *ClarificationStore) Create(_ context.Context, created spine.ClarificationRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.requests[created.ID]; exists {
		return ErrClarificationRequestAlreadyExists
	}
	if created.State == spine.ClarificationRequestStateOpen {
		if _, exists := s.openByGoal[created.GoalID]; exists {
			return ErrClarificationRequestAlreadyOpen
		}
	}

	s.requests[created.ID] = cloneClarificationRequest(created)
	if created.State == spine.ClarificationRequestStateOpen {
		s.openByGoal[created.GoalID] = created.ID
	}
	return nil
}

func (s *ClarificationStore) Get(_ context.Context, id spine.ClarificationRequestID) (spine.ClarificationRequest, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	request, ok := s.requests[id]
	if !ok {
		return spine.ClarificationRequest{}, false, nil
	}
	return cloneClarificationRequest(request), true, nil
}

func (s *ClarificationStore) GetOpenByGoalID(_ context.Context, goalID spine.GoalID) (spine.ClarificationRequest, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	requestID, ok := s.openByGoal[goalID]
	if !ok {
		return spine.ClarificationRequest{}, false, nil
	}
	request, ok := s.requests[requestID]
	if !ok {
		return spine.ClarificationRequest{}, false, nil
	}
	return cloneClarificationRequest(request), true, nil
}

func (s *ClarificationStore) UpdateState(_ context.Context, id spine.ClarificationRequestID, state spine.ClarificationRequestState) (spine.ClarificationRequest, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	updated, ok := s.requests[id]
	if !ok {
		return spine.ClarificationRequest{}, false, nil
	}
	if state == spine.ClarificationRequestStateOpen {
		if openID, exists := s.openByGoal[updated.GoalID]; exists && openID != id {
			return spine.ClarificationRequest{}, true, ErrClarificationRequestAlreadyOpen
		}
	}

	if updated.State == spine.ClarificationRequestStateOpen && state != spine.ClarificationRequestStateOpen {
		delete(s.openByGoal, updated.GoalID)
	}
	updated.State = state
	s.requests[id] = cloneClarificationRequest(updated)
	if state == spine.ClarificationRequestStateOpen {
		s.openByGoal[updated.GoalID] = id
	}

	return cloneClarificationRequest(updated), true, nil
}

func cloneClarificationRequest(request spine.ClarificationRequest) spine.ClarificationRequest {
	if request.ReasonCodes != nil {
		request.ReasonCodes = append([]spine.GoalReadinessReasonCode(nil), request.ReasonCodes...)
	}
	if request.Questions != nil {
		request.Questions = append([]spine.ClarificationQuestion(nil), request.Questions...)
	}
	if request.Target.ActorRef != nil {
		actor := *request.Target.ActorRef
		request.Target.ActorRef = &actor
	}
	return request
}
