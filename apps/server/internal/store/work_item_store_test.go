package store_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/spine"
	"github.com/heurema/goalrail/apps/server/internal/store"
)

func TestWorkItemStoreCreateGetAndGetByApprovedContractID(t *testing.T) {
	workItems := store.NewWorkItemStore()
	created := validWorkItem()

	if err := workItems.Create(context.Background(), created); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, ok, err := workItems.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !ok {
		t.Fatal("Get() ok = false, want true")
	}
	if !reflect.DeepEqual(got, created) {
		t.Fatalf("Get() = %#v, want %#v", got, created)
	}

	byApproved, ok, err := workItems.GetByApprovedContractID(context.Background(), created.ApprovedContractID)
	if err != nil {
		t.Fatalf("GetByApprovedContractID() error = %v", err)
	}
	if !ok {
		t.Fatal("GetByApprovedContractID() ok = false, want true")
	}
	if byApproved.ID != created.ID {
		t.Fatalf("GetByApprovedContractID() id = %q, want %q", byApproved.ID, created.ID)
	}
}

func TestWorkItemStorePreventsDuplicateForApprovedContract(t *testing.T) {
	workItems := store.NewWorkItemStore()
	created := validWorkItem()
	if err := workItems.Create(context.Background(), created); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	duplicate := created
	duplicate.ID = "work-item-2"
	if err := workItems.Create(context.Background(), duplicate); err != store.ErrWorkItemAlreadyPlanned {
		t.Fatalf("duplicate Create() error = %v, want %v", err, store.ErrWorkItemAlreadyPlanned)
	}
}

func validWorkItem() spine.WorkItem {
	return spine.WorkItem{
		ID:                   "work-item-1",
		OrganizationID:       "organization-1",
		ProjectID:            "project-1",
		ContractID:           "contract-1",
		ApprovedContractID:   "approved-contract-1",
		RepoBindingID:        "repo-binding-1",
		Title:                "Planned task",
		Summary:              "Simple v0 planned task",
		Scope:                []string{"Scope"},
		AcceptanceRefs:       []string{"acceptance_criteria[0]"},
		ProofExpectationRefs: []string{"proof_expectations[0]"},
		Status:               spine.WorkItemStatusPlanned,
		SourceRefs: []spine.SourceRef{
			{Kind: "approved_contract", ID: "approved-contract-1"},
		},
		CreatedAt: time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC),
	}
}
