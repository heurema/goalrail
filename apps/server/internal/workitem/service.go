package workitem

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

const (
	EventTypeWorkItemCreated      = "work_item.created"
	EntityTypeWorkItem            = "WorkItem"
	SourceRefKindApprovedContract = "approved_contract"
)

var (
	ErrWorkItemNotFound      = errors.New("work item not found")
	ErrMembershipRequired    = errors.New("active organization membership is required")
	ErrOrganizationForbidden = errors.New("user is not allowed to read this work item")
	ErrProjectMismatch       = errors.New("work item project expectation does not match work item")
	ErrRepoBindingMismatch   = errors.New("work item repo binding expectation does not match work item")
)

type Store interface {
	Get(context.Context, spine.WorkItemID) (spine.WorkItem, bool, error)
}

type ContractReader interface {
	Get(context.Context, spine.ContractID) (spine.Contract, bool, error)
}

type Service struct {
	WorkItems Store
	Contracts ContractReader
}

func NewService(workItems Store, contracts ContractReader) *Service {
	return &Service{
		WorkItems: workItems,
		Contracts: contracts,
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

func (s *Service) GetDetail(ctx context.Context, id spine.WorkItemID, input spine.WorkItemDetailRequest, membership spine.OrganizationMembership) (spine.WorkItemDetail, error) {
	item, err := s.Get(ctx, id)
	if err != nil {
		return spine.WorkItemDetail{}, err
	}
	if err := authorizeTaskAccess(membership, item.OrganizationID); err != nil {
		return spine.WorkItemDetail{}, err
	}
	if strings.TrimSpace(string(input.ProjectID)) != "" && input.ProjectID != item.ProjectID {
		return spine.WorkItemDetail{}, ErrProjectMismatch
	}
	if strings.TrimSpace(string(input.RepoBindingID)) != "" && input.RepoBindingID != item.RepoBindingID {
		return spine.WorkItemDetail{}, ErrRepoBindingMismatch
	}
	detail := detailFromWorkItem(item)
	if s.Contracts != nil {
		contract, ok, err := s.Contracts.Get(ctx, item.ContractID)
		if err != nil {
			return spine.WorkItemDetail{}, fmt.Errorf("get work item contract: %w", err)
		}
		if ok && contract.OrganizationID == item.OrganizationID && contract.ProjectID == item.ProjectID && contract.RepoBindingID == item.RepoBindingID {
			detail.GoalID = contract.GoalID
		}
	}
	return detail, nil
}

func authorizeTaskAccess(membership spine.OrganizationMembership, organizationID spine.OrganizationID) error {
	if membership.State != spine.EntityStateActive || strings.TrimSpace(string(membership.OrganizationID)) == "" {
		return ErrMembershipRequired
	}
	if membership.OrganizationID != organizationID {
		return ErrOrganizationForbidden
	}
	return nil
}

func detailFromWorkItem(item spine.WorkItem) spine.WorkItemDetail {
	nextAction := spine.WorkItemNextAction{
		Kind:      "prepare_checkout",
		Blocking:  false,
		Available: item.Status == spine.WorkItemStatusPlanned,
	}
	if nextAction.Available {
		nextAction.Command = fmt.Sprintf("goalrail work checkout prepare --task-id %s --format json", item.ID)
	}
	return spine.WorkItemDetail{
		ID:                   item.ID,
		WorkItemID:           item.ID,
		TaskID:               item.ID,
		ProjectID:            item.ProjectID,
		ContractID:           item.ContractID,
		ApprovedContractID:   item.ApprovedContractID,
		PlanID:               item.PlanID,
		ProposalID:           item.ProposalID,
		RepoBindingID:        item.RepoBindingID,
		Status:               item.Status,
		Title:                item.Title,
		Summary:              item.Summary,
		Scope:                append([]string(nil), item.Scope...),
		AcceptanceRefs:       append([]string(nil), item.AcceptanceRefs...),
		ProofExpectationRefs: append([]string(nil), item.ProofExpectationRefs...),
		SourceRefs:           append([]spine.SourceRef(nil), item.SourceRefs...),
		OwnerHint:            item.OwnerHint,
		OrderIndex:           item.OrderIndex,
		NextAction:           nextAction,
	}
}
