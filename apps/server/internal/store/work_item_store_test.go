package store_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/spine"
	"github.com/heurema/goalrail/apps/server/internal/store"
)

func TestWorkItemStoreCreateAndGet(t *testing.T) {
	workItemStore := store.NewWorkItemStore()
	created := validWorkItem()

	if err := workItemStore.Create(context.Background(), created); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, ok, err := workItemStore.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !ok {
		t.Fatal("Get() ok = false, want true")
	}
	if !reflect.DeepEqual(got, created) {
		t.Fatalf("Get() = %#v, want %#v", got, created)
	}

	byApproved, ok, err := workItemStore.GetByApprovedContractID(context.Background(), created.ApprovedContractID)
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
	workItemStore := store.NewWorkItemStore()
	created := validWorkItem()
	if err := workItemStore.Create(context.Background(), created); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	duplicate := created
	duplicate.ID = "work-item-2"
	if err := workItemStore.Create(context.Background(), duplicate); err != store.ErrWorkItemAlreadyPlanned {
		t.Fatalf("duplicate Create() error = %v, want %v", err, store.ErrWorkItemAlreadyPlanned)
	}
}

func TestWorkItemStoreReturnsClones(t *testing.T) {
	workItemStore := store.NewWorkItemStore()
	created := validWorkItem()
	orderIndex := 0
	created.OrderIndex = &orderIndex
	if err := workItemStore.Create(context.Background(), created); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, ok, err := workItemStore.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !ok {
		t.Fatal("Get() ok = false, want true")
	}
	got.Scope[0] = "mutated"
	got.AcceptanceRefs[0] = "mutated"
	got.ProofExpectationRefs[0] = "mutated"
	got.SourceRefs[0] = spine.SourceRef{Kind: "mutated", ID: "mutated"}
	*got.OrderIndex = 99

	stored, ok, err := workItemStore.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("Get() second error = %v", err)
	}
	if !ok {
		t.Fatal("Get() second ok = false, want true")
	}
	if !reflect.DeepEqual(stored, created) {
		t.Fatalf("stored mutated through returned value: %#v want %#v", stored, created)
	}
}

func validWorkItem() spine.WorkItem {
	return spine.WorkItem{
		ID:                   "work-item-1",
		OrganizationID:       "organization-1",
		ProjectID:            "project-1",
		ApprovedContractID:   "approved-contract-1",
		RepoBindingID:        "repo-binding-1",
		Title:                "Approved title",
		Summary:              "Approved summary",
		Scope:                []string{"Approved scope"},
		AcceptanceRefs:       []string{"acceptance_criteria[0]"},
		ProofExpectationRefs: []string{"proof_expectations[0]"},
		Status:               spine.WorkItemStatusPlanned,
		SourceRefs: []spine.SourceRef{
			{Kind: "approved_contract", ID: "approved-contract-1"},
		},
		CreatedAt: time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC),
	}
}
