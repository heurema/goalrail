package workitem_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/spine"
	"github.com/heurema/goalrail/apps/server/internal/workitem"
)

func TestServiceGetsPlannedWorkItem(t *testing.T) {
	workItems := newFakeWorkItemStore()
	created := validWorkItem()
	if err := workItems.Create(context.Background(), created); err != nil {
		t.Fatalf("workItems.Create() error = %v", err)
	}
	contracts := newFakeContractStore()
	contracts.contracts[created.ContractID] = spine.Contract{
		ID:             created.ContractID,
		OrganizationID: created.OrganizationID,
		ProjectID:      created.ProjectID,
		RepoBindingID:  created.RepoBindingID,
		GoalID:         "goal-1",
	}
	service := workitem.NewService(workItems, contracts)

	got, err := service.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("Get() id = %q, want %q", got.ID, created.ID)
	}
	if got.ContractID != created.ContractID || got.ApprovedContractID != created.ApprovedContractID {
		t.Fatalf("Get() ids = %q/%q, want contract/approved ids", got.ContractID, got.ApprovedContractID)
	}
	if got.PlanID != created.PlanID || got.ProposalID != created.ProposalID {
		t.Fatalf("Get() planning trace = %q/%q, want %q/%q", got.PlanID, got.ProposalID, created.PlanID, created.ProposalID)
	}
}

func TestServiceGetUnknownWorkItemReturnsNotFound(t *testing.T) {
	service := workitem.NewService(newFakeWorkItemStore(), nil)

	_, err := service.Get(context.Background(), "missing")
	if !errors.Is(err, workitem.ErrWorkItemNotFound) {
		t.Fatalf("Get() error = %v, want ErrWorkItemNotFound", err)
	}
}

func TestServiceGetsAuthorizedWorkItemDetail(t *testing.T) {
	workItems := newFakeWorkItemStore()
	created := validWorkItem()
	if err := workItems.Create(context.Background(), created); err != nil {
		t.Fatalf("workItems.Create() error = %v", err)
	}
	contracts := newFakeContractStore()
	contracts.contracts[created.ContractID] = spine.Contract{
		ID:             created.ContractID,
		OrganizationID: created.OrganizationID,
		ProjectID:      created.ProjectID,
		RepoBindingID:  created.RepoBindingID,
		GoalID:         "goal-1",
	}
	service := workitem.NewService(workItems, contracts)

	got, err := service.GetDetail(context.Background(), created.ID, spine.WorkItemDetailRequest{
		ProjectID:     created.ProjectID,
		RepoBindingID: created.RepoBindingID,
	}, spine.OrganizationMembership{
		OrganizationID: created.OrganizationID,
		State:          spine.EntityStateActive,
	})
	if err != nil {
		t.Fatalf("GetDetail() error = %v", err)
	}
	if got.WorkItemID != created.ID || got.TaskID != created.ID || got.GoalID != "goal-1" {
		t.Fatalf("GetDetail() identity = %#v, want work item/task id and goal", got)
	}
	if got.Title != created.Title || got.Summary != created.Summary || len(got.Scope) != 1 {
		t.Fatalf("GetDetail() body = %#v, want WorkItem body fields", got)
	}
	if got.NextAction.Kind != "prepare_checkout" || !got.NextAction.Available {
		t.Fatalf("GetDetail() next_action = %#v, want available checkout preparation", got.NextAction)
	}
}

func TestServiceGetDetailRejectsWrongScope(t *testing.T) {
	workItems := newFakeWorkItemStore()
	created := validWorkItem()
	if err := workItems.Create(context.Background(), created); err != nil {
		t.Fatalf("workItems.Create() error = %v", err)
	}
	service := workitem.NewService(workItems, nil)
	membership := spine.OrganizationMembership{
		OrganizationID: created.OrganizationID,
		State:          spine.EntityStateActive,
	}

	if _, err := service.GetDetail(context.Background(), created.ID, spine.WorkItemDetailRequest{ProjectID: "other-project"}, membership); !errors.Is(err, workitem.ErrProjectMismatch) {
		t.Fatalf("GetDetail() project mismatch error = %v, want ErrProjectMismatch", err)
	}
	if _, err := service.GetDetail(context.Background(), created.ID, spine.WorkItemDetailRequest{}, spine.OrganizationMembership{OrganizationID: "other-org", State: spine.EntityStateActive}); !errors.Is(err, workitem.ErrOrganizationForbidden) {
		t.Fatalf("GetDetail() org mismatch error = %v, want ErrOrganizationForbidden", err)
	}
}

func validWorkItem() spine.WorkItem {
	orderIndex := 0
	return spine.WorkItem{
		ID:                   "work-item-1",
		OrganizationID:       "organization-1",
		ProjectID:            "project-1",
		ContractID:           "contract-1",
		ApprovedContractID:   "approved-contract-1",
		PlanID:               "plan-1",
		ProposalID:           "proposal-1",
		RepoBindingID:        "repo-binding-1",
		Title:                "Refactor CSV export filter builder",
		Summary:              "Extract duplicated filter construction logic.",
		Scope:                []string{"Update export filter construction code"},
		AcceptanceRefs:       []string{"acceptance_criteria[0]"},
		ProofExpectationRefs: []string{"proof_expectations[0]"},
		Status:               spine.WorkItemStatusPlanned,
		OrderIndex:           &orderIndex,
		SourceRefs: []spine.SourceRef{
			{Kind: workitem.SourceRefKindApprovedContract, ID: "approved-contract-1"},
			{Kind: "proposal", ID: "proposal-1"},
		},
		CreatedAt: time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC),
	}
}

type fakeWorkItemStore struct {
	items map[spine.WorkItemID]spine.WorkItem
}

func newFakeWorkItemStore() *fakeWorkItemStore {
	return &fakeWorkItemStore{items: map[spine.WorkItemID]spine.WorkItem{}}
}

func (s *fakeWorkItemStore) Create(_ context.Context, item spine.WorkItem) error {
	s.items[item.ID] = item
	return nil
}

func (s *fakeWorkItemStore) Get(_ context.Context, id spine.WorkItemID) (spine.WorkItem, bool, error) {
	item, ok := s.items[id]
	return item, ok, nil
}

type fakeContractStore struct {
	contracts map[spine.ContractID]spine.Contract
}

func (s *fakeContractStore) Get(_ context.Context, id spine.ContractID) (spine.Contract, bool, error) {
	contract, ok := s.contracts[id]
	return contract, ok, nil
}

func newFakeContractStore() *fakeContractStore {
	return &fakeContractStore{contracts: map[spine.ContractID]spine.Contract{}}
}
