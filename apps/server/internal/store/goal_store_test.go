package store_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/spine"
	"github.com/heurema/goalrail/apps/server/internal/store"
)

func TestGoalStoreCreateAndGet(t *testing.T) {
	goalStore := store.NewGoalStore()
	created := validGoal()

	if err := goalStore.Create(context.Background(), created); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, ok, err := goalStore.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !ok {
		t.Fatal("Get() ok = false, want true")
	}
	if !reflect.DeepEqual(got, created) {
		t.Fatalf("Get() = %#v, want %#v", got, created)
	}

	gotByIntake, ok, err := goalStore.GetByIntakeID(context.Background(), created.IntakeID)
	if err != nil {
		t.Fatalf("GetByIntakeID() error = %v", err)
	}
	if !ok {
		t.Fatal("GetByIntakeID() ok = false, want true")
	}
	if !reflect.DeepEqual(gotByIntake, created) {
		t.Fatalf("GetByIntakeID() = %#v, want %#v", gotByIntake, created)
	}

	duplicate := created
	duplicate.ID = "goal-2"
	if err := goalStore.Create(context.Background(), duplicate); err != store.ErrGoalAlreadyExists {
		t.Fatalf("duplicate intake Create() error = %v, want %v", err, store.ErrGoalAlreadyExists)
	}
}

func TestGoalStoreUpdateState(t *testing.T) {
	goalStore := store.NewGoalStore()
	created := validGoal()
	if err := goalStore.Create(context.Background(), created); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	updated, ok, err := goalStore.UpdateState(context.Background(), created.ID, spine.GoalStateNeedsClarification)
	if err != nil {
		t.Fatalf("UpdateState() error = %v", err)
	}
	if !ok {
		t.Fatal("UpdateState() ok = false, want true")
	}
	if updated.State != spine.GoalStateNeedsClarification {
		t.Fatalf("updated state = %q, want %q", updated.State, spine.GoalStateNeedsClarification)
	}

	stored, ok, err := goalStore.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !ok {
		t.Fatal("Get() ok = false, want true")
	}
	if stored.State != spine.GoalStateNeedsClarification {
		t.Fatalf("stored state = %q, want %q", stored.State, spine.GoalStateNeedsClarification)
	}

	if _, ok, err := goalStore.UpdateState(context.Background(), "missing", spine.GoalStateRejected); err != nil {
		t.Fatalf("missing UpdateState() error = %v", err)
	} else if ok {
		t.Fatal("missing UpdateState() ok = true, want false")
	}
}

func validGoal() spine.Goal {
	return spine.Goal{
		ID:             "goal-1",
		IntakeID:       "intake-1",
		RepoBindingID:  "repo_demo_1",
		Title:          "Refactor CSV export filters",
		Summary:        "Current code duplicates filter logic. Preserve current behavior.",
		ScopeHint:      "Refactor duplicate filter logic",
		AcceptanceHint: "Existing CSV export behavior is preserved",
		SourceRefs: []spine.SourceRef{
			{Kind: "intake", ID: "intake-1"},
		},
		RequestAuthor: spine.ActorRef{Kind: "user", ID: "dev_1"},
		IntentOwner:   spine.ActorRef{Kind: "user", ID: "dev_1"},
		State:         spine.GoalStateCreated,
		CreatedAt:     time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC),
	}
}
