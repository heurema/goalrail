package store

import (
	"context"
	"errors"
	"sync"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

var (
	ErrClarificationAnswerAlreadyExists   = errors.New("clarification answer already exists")
	ErrClarificationAnswerAlreadyRecorded = errors.New("clarification answer already recorded")
)

type ClarificationAnswerStore struct {
	mu        sync.RWMutex
	answers   map[spine.ClarificationAnswerID]spine.ClarificationAnswer
	byRequest map[spine.ClarificationRequestID]spine.ClarificationAnswerID
	applied   map[spine.ClarificationAnswerID]bool
}

func NewClarificationAnswerStore() *ClarificationAnswerStore {
	return &ClarificationAnswerStore{
		answers:   make(map[spine.ClarificationAnswerID]spine.ClarificationAnswer),
		byRequest: make(map[spine.ClarificationRequestID]spine.ClarificationAnswerID),
		applied:   make(map[spine.ClarificationAnswerID]bool),
	}
}

func (s *ClarificationAnswerStore) Create(_ context.Context, created spine.ClarificationAnswer) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.answers[created.ID]; exists {
		return ErrClarificationAnswerAlreadyExists
	}
	if _, exists := s.byRequest[created.RequestID]; exists {
		return ErrClarificationAnswerAlreadyRecorded
	}

	s.answers[created.ID] = cloneClarificationAnswer(created)
	s.byRequest[created.RequestID] = created.ID
	return nil
}

func (s *ClarificationAnswerStore) Get(_ context.Context, id spine.ClarificationAnswerID) (spine.ClarificationAnswer, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	answer, ok := s.answers[id]
	if !ok {
		return spine.ClarificationAnswer{}, false, nil
	}
	return cloneClarificationAnswer(answer), true, nil
}

func (s *ClarificationAnswerStore) GetByRequestID(_ context.Context, requestID spine.ClarificationRequestID) (spine.ClarificationAnswer, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	answerID, ok := s.byRequest[requestID]
	if !ok {
		return spine.ClarificationAnswer{}, false, nil
	}
	answer, ok := s.answers[answerID]
	if !ok {
		return spine.ClarificationAnswer{}, false, nil
	}
	return cloneClarificationAnswer(answer), true, nil
}

func (s *ClarificationAnswerStore) MarkApplied(_ context.Context, id spine.ClarificationAnswerID) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.applied[id] {
		return false, nil
	}
	s.applied[id] = true
	return true, nil
}

func cloneClarificationAnswer(answer spine.ClarificationAnswer) spine.ClarificationAnswer {
	if answer.Answers != nil {
		answer.Answers = append([]spine.ClarificationAnswerItem(nil), answer.Answers...)
	}
	return answer
}
