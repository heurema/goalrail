package store_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/spine"
	"github.com/heurema/goalrail/apps/server/internal/store"
)

func TestContractDraftStoreCreateAndGet(t *testing.T) {
	draftStore := store.NewContractDraftStore()
	created := validContractDraft()

	if err := draftStore.Create(context.Background(), created); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, ok, err := draftStore.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !ok {
		t.Fatal("Get() ok = false, want true")
	}
	if !reflect.DeepEqual(got, created) {
		t.Fatalf("Get() = %#v, want %#v", got, created)
	}

	bySeed, ok, err := draftStore.GetByContractSeedID(context.Background(), created.ContractSeedID)
	if err != nil {
		t.Fatalf("GetByContractSeedID() error = %v", err)
	}
	if !ok {
		t.Fatal("GetByContractSeedID() ok = false, want true")
	}
	if !reflect.DeepEqual(bySeed, created) {
		t.Fatalf("GetByContractSeedID() = %#v, want %#v", bySeed, created)
	}
}

func TestContractDraftStorePreventsDuplicateDraftForSeed(t *testing.T) {
	draftStore := store.NewContractDraftStore()
	created := validContractDraft()
	if err := draftStore.Create(context.Background(), created); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	duplicate := created
	duplicate.ID = "contract-draft-2"
	if err := draftStore.Create(context.Background(), duplicate); err != store.ErrContractDraftAlreadyDrafted {
		t.Fatalf("duplicate Create() error = %v, want %v", err, store.ErrContractDraftAlreadyDrafted)
	}
}

func validContractDraft() spine.ContractDraft {
	return spine.ContractDraft{
		ID:                         "contract-draft-1",
		ContractSeedID:             "contract-seed-1",
		GoalID:                     "goal-1",
		RepoBindingID:              "repo-binding-1",
		Title:                      "Refactor CSV export filters",
		IntentSummary:              "Current code duplicates filter logic.",
		ProposedScope:              []string{"Refactor duplicate filter logic"},
		ProposedNonGoals:           []string{},
		ProposedConstraints:        []string{},
		ProposedAcceptanceCriteria: []string{"Existing CSV export behavior is preserved"},
		ProposedExpectedChecks:     []string{},
		ProposedProofExpectations:  []string{"Provide evidence that acceptance criteria were checked."},
		RiskHints:                  []string{},
		SourceRefs: []spine.SourceRef{
			{Kind: "contract_seed", ID: "contract-seed-1"},
			{Kind: "goal", ID: "goal-1"},
			{Kind: "intake", ID: "intake-1"},
		},
		State:     spine.ContractDraftStateDraft,
		CreatedAt: time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC),
	}
}
