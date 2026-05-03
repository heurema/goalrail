package workitem

import (
	"context"
	"errors"
	"fmt"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

const (
	EventTypeWorkItemCreated      = "work_item.created"
	EntityTypeWorkItem            = "WorkItem"
	SourceRefKindApprovedContract = "approved_contract"
)

var ErrWorkItemNotFound = errors.New("work item not found")

type Store interface {
	Get(context.Context, spine.WorkItemID) (spine.WorkItem, bool, error)
}

type Service struct {
	WorkItems Store
}

func NewService(workItems Store) *Service {
	return &Service{
		WorkItems: workItems,
	}
}

func (s *Service) Get(ctx context.Context, id spine.WorkItemID) (spine.WorkItem, error) {
	item, ok, err := s.WorkItems.Get(ctx, id)
	if err != nil {
		return spine.WorkItem{}, fmt.Errorf("get work item: %w", err)
	}
	if !ok {
		return spine.WorkItem{}, ErrWorkItemNotFound
	}
	return item, nil
}
