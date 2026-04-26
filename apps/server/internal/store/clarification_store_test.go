package store_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/spine"
	"github.com/heurema/goalrail/apps/server/internal/store"
)

func TestClarificationStoreCreateAndGet(t *testing.T) {
	clarificationStore := store.NewClarificationStore()
	created := validClarificationRequest()

	if err := clarificationStore.Create(context.Background(), created); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, ok, err := clarificationStore.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !ok {
		t.Fatal("Get() ok = false, want true")
	}
	if !reflect.DeepEqual(got, created) {
		t.Fatalf("Get() = %#v, want %#v", got, created)
	}

	open, ok, err := clarificationStore.GetOpenByGoalID(context.Background(), created.GoalID)
	if err != nil {
		t.Fatalf("GetOpenByGoalID() error = %v", err)
	}
	if !ok {
		t.Fatal("GetOpenByGoalID() ok = false, want true")
	}
	if !reflect.DeepEqual(open, created) {
		t.Fatalf("GetOpenByGoalID() = %#v, want %#v", open, created)
	}
}

func TestClarificationStorePreventsDuplicateOpenRequestForGoal(t *testing.T) {
	clarificationStore := store.NewClarificationStore()
	created := validClarificationRequest()
	if err := clarificationStore.Create(context.Background(), created); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	duplicate := created
	duplicate.ID = "clarification-2"
	if err := clarificationStore.Create(context.Background(), duplicate); err != store.ErrClarificationRequestAlreadyOpen {
		t.Fatalf("duplicate open Create() error = %v, want %v", err, store.ErrClarificationRequestAlreadyOpen)
	}
}

func validClarificationRequest() spine.ClarificationRequest {
	actor := spine.ActorRef{Kind: "user", ID: "dev_1"}
	return spine.ClarificationRequest{
		ID:     "clarification-1",
		GoalID: "goal-1",
		ReasonCodes: []spine.GoalReadinessReasonCode{
			spine.GoalReadinessReasonMissingScopeHint,
			spine.GoalReadinessReasonMissingAcceptanceHint,
		},
		Questions: []spine.ClarificationQuestion{
			{
				ID:         "question-1",
				Text:       "What is the intended scope at a high level?",
				WhyNeeded:  "A scope hint is required before contract seed readiness.",
				AnswerType: spine.ClarificationAnswerTypeText,
				MapsTo:     spine.ClarificationMapsToGoalScopeHint,
			},
		},
		Target: spine.ClarificationTarget{
			Role:     spine.ClarificationTargetRoleIntentOwner,
			ActorRef: &actor,
		},
		State:     spine.ClarificationRequestStateOpen,
		CreatedAt: time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC),
	}
}
