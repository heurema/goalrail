package store_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/spine"
	"github.com/heurema/goalrail/apps/server/internal/store"
)

func TestContractSeedStoreCreateAndGet(t *testing.T) {
	seedStore := store.NewContractSeedStore()
	created := validContractSeed()

	if err := seedStore.Create(context.Background(), created); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, ok, err := seedStore.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !ok {
		t.Fatal("Get() ok = false, want true")
	}
	if !reflect.DeepEqual(got, created) {
		t.Fatalf("Get() = %#v, want %#v", got, created)
	}

	byGoal, ok, err := seedStore.GetByGoalID(context.Background(), created.GoalID)
	if err != nil {
		t.Fatalf("GetByGoalID() error = %v", err)
	}
	if !ok {
		t.Fatal("GetByGoalID() ok = false, want true")
	}
	if !reflect.DeepEqual(byGoal, created) {
		t.Fatalf("GetByGoalID() = %#v, want %#v", byGoal, created)
	}
}

func TestContractSeedStorePreventsDuplicateSeedForGoal(t *testing.T) {
	seedStore := store.NewContractSeedStore()
	created := validContractSeed()
	if err := seedStore.Create(context.Background(), created); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	duplicate := created
	duplicate.ID = "contract-seed-2"
	if err := seedStore.Create(context.Background(), duplicate); err != store.ErrContractSeedAlreadySeeded {
		t.Fatalf("duplicate Create() error = %v, want %v", err, store.ErrContractSeedAlreadySeeded)
	}
}

func validContractSeed() spine.ContractSeed {
	return spine.ContractSeed{
		ID:             "contract-seed-1",
		GoalID:         "goal-1",
		RepoBindingID:  "repo-binding-1",
		Title:          "Refactor CSV export filters",
		IntentSummary:  "Current code duplicates filter logic.",
		IntentOwner:    spine.ActorRef{Kind: "user", ID: "dev_1"},
		ScopeHint:      "Refactor duplicate filter logic",
		AcceptanceHint: "Existing CSV export behavior is preserved",
		SourceRefs: []spine.SourceRef{
			{Kind: "goal", ID: "goal-1"},
			{Kind: "intake", ID: "intake-1"},
		},
		State:     spine.ContractSeedStateCreated,
		CreatedAt: time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC),
	}
}
