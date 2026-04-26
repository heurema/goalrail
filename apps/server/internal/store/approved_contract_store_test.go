package store_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/spine"
	"github.com/heurema/goalrail/apps/server/internal/store"
)

func TestApprovedContractStoreCreateAndGet(t *testing.T) {
	approvedStore := store.NewApprovedContractStore()
	created := validApprovedContract()

	if err := approvedStore.Create(context.Background(), created); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, ok, err := approvedStore.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !ok {
		t.Fatal("Get() ok = false, want true")
	}
	if !reflect.DeepEqual(got, created) {
		t.Fatalf("Get() = %#v, want %#v", got, created)
	}

	byDraft, ok, err := approvedStore.GetByContractDraftID(context.Background(), created.ContractDraftID)
	if err != nil {
		t.Fatalf("GetByContractDraftID() error = %v", err)
	}
	if !ok {
		t.Fatal("GetByContractDraftID() ok = false, want true")
	}
	if byDraft.ID != created.ID {
		t.Fatalf("GetByContractDraftID() id = %q, want %q", byDraft.ID, created.ID)
	}
}

func TestApprovedContractStorePreventsDuplicateForDraft(t *testing.T) {
	approvedStore := store.NewApprovedContractStore()
	created := validApprovedContract()
	if err := approvedStore.Create(context.Background(), created); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	duplicate := created
	duplicate.ID = "approved-contract-2"
	if err := approvedStore.Create(context.Background(), duplicate); err != store.ErrApprovedContractAlreadyApproved {
		t.Fatalf("duplicate Create() error = %v, want %v", err, store.ErrApprovedContractAlreadyApproved)
	}
}

func validApprovedContract() spine.ApprovedContract {
	return spine.ApprovedContract{
		ID:                 "approved-contract-1",
		OrganizationID:     "organization-1",
		ProjectID:          "project-1",
		ContractDraftID:    "contract-draft-1",
		ContractSeedID:     "contract-seed-1",
		GoalID:             "goal-1",
		RepoBindingID:      "repo-binding-1",
		Title:              "Approved title",
		IntentSummary:      "Approved summary",
		Scope:              []string{"Approved scope"},
		NonGoals:           []string{},
		Constraints:        []string{},
		AcceptanceCriteria: []string{"Approved acceptance"},
		ExpectedChecks:     []string{},
		ProofExpectations:  []string{"Approved proof"},
		RiskHints:          []string{},
		ApprovedBy:         spine.ActorRef{Kind: "user", ID: "dev_approver"},
		ApprovedAt:         time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC),
		SourceRefs: []spine.SourceRef{
			{Kind: "contract_draft", ID: "contract-draft-1"},
		},
		State: spine.ApprovedContractStateApproved,
	}
}
