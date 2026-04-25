package store

import (
	"context"
	"errors"
	"sync"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

var ErrAlreadyExists = errors.New("intake record already exists")

type IntakeStore struct {
	mu      sync.RWMutex
	records map[spine.IntakeID]spine.IntakeRecord
}

func NewIntakeStore() *IntakeStore {
	return &IntakeStore{records: make(map[spine.IntakeID]spine.IntakeRecord)}
}

func (s *IntakeStore) Create(_ context.Context, record spine.IntakeRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.records[record.ID]; exists {
		return ErrAlreadyExists
	}
	s.records[record.ID] = record
	return nil
}

func (s *IntakeStore) Get(_ context.Context, id spine.IntakeID) (spine.IntakeRecord, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	record, ok := s.records[id]
	return record, ok, nil
}
