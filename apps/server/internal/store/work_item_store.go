package store

import (
	"context"
	"errors"
	"sync"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

var (
	ErrWorkItemAlreadyExists  = errors.New("work item already exists")
	ErrWorkItemAlreadyPlanned = errors.New("approved contract already planned")
	ErrWorkItemNotFound       = errors.New("work item not found")
)

type WorkItemStore struct {
	mu                 sync.RWMutex
	items              map[spine.WorkItemID]spine.WorkItem
	byApprovedContract map[spine.ApprovedContractID][]spine.WorkItemID
}

func NewWorkItemStore() *WorkItemStore {
	return &WorkItemStore{
		items:              make(map[spine.WorkItemID]spine.WorkItem),
		byApprovedContract: make(map[spine.ApprovedContractID][]spine.WorkItemID),
	}
}

func (s *WorkItemStore) Create(_ context.Context, item spine.WorkItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.items[item.ID]; exists {
		return ErrWorkItemAlreadyExists
	}

	s.items[item.ID] = cloneWorkItem(item)
	s.byApprovedContract[item.ApprovedContractID] = append(s.byApprovedContract[item.ApprovedContractID], item.ID)
	return nil
}

func (s *WorkItemStore) Get(_ context.Context, id spine.WorkItemID) (spine.WorkItem, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.items[id]
	if !ok {
		return spine.WorkItem{}, false, nil
	}
	return cloneWorkItem(item), true, nil
}

func (s *WorkItemStore) GetByApprovedContractID(_ context.Context, id spine.ApprovedContractID) (spine.WorkItem, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	itemIDs := s.byApprovedContract[id]
	if len(itemIDs) == 0 {
		return spine.WorkItem{}, false, nil
	}
	item, ok := s.items[itemIDs[0]]
	if !ok {
		return spine.WorkItem{}, false, nil
	}
	return cloneWorkItem(item), true, nil
}

func cloneWorkItem(item spine.WorkItem) spine.WorkItem {
	item.Scope = cloneStringSlice(item.Scope)
	item.AcceptanceRefs = cloneStringSlice(item.AcceptanceRefs)
	item.ProofExpectationRefs = cloneStringSlice(item.ProofExpectationRefs)
	item.SourceRefs = append([]spine.SourceRef(nil), item.SourceRefs...)
	if item.OrderIndex != nil {
		orderIndex := *item.OrderIndex
		item.OrderIndex = &orderIndex
	}
	return item
}
