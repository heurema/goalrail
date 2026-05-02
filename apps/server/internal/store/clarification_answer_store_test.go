package store_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/spine"
	"github.com/heurema/goalrail/apps/server/internal/store"
)

func TestClarificationAnswerStoreCreateAndGet(t *testing.T) {
	answerStore := store.NewClarificationAnswerStore()
	created := validClarificationAnswer()

	if err := answerStore.Create(context.Background(), created); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, ok, err := answerStore.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !ok {
		t.Fatal("Get() ok = false, want true")
	}
	if !reflect.DeepEqual(got, created) {
		t.Fatalf("Get() = %#v, want %#v", got, created)
	}

	byRequest, ok, err := answerStore.GetByRequestID(context.Background(), created.RequestID)
	if err != nil {
		t.Fatalf("GetByRequestID() error = %v", err)
	}
	if !ok {
		t.Fatal("GetByRequestID() ok = false, want true")
	}
	if !reflect.DeepEqual(byRequest, created) {
		t.Fatalf("GetByRequestID() = %#v, want %#v", byRequest, created)
	}
}

func TestClarificationAnswerStorePreventsDuplicateAnswerForRequest(t *testing.T) {
	answerStore := store.NewClarificationAnswerStore()
	created := validClarificationAnswer()
	if err := answerStore.Create(context.Background(), created); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	duplicate := created
	duplicate.ID = "answer-2"
	if err := answerStore.Create(context.Background(), duplicate); err != store.ErrClarificationAnswerAlreadyRecorded {
		t.Fatalf("duplicate Create() error = %v, want %v", err, store.ErrClarificationAnswerAlreadyRecorded)
	}
}

func TestClarificationAnswerStorePreventsDuplicateApplication(t *testing.T) {
	answerStore := store.NewClarificationAnswerStore()
	created := validClarificationAnswer()
	if err := answerStore.Create(context.Background(), created); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	marked, err := answerStore.MarkApplied(context.Background(), created.ID, spine.ActorRef{Kind: "user", ID: "dev_1"}, created.CreatedAt)
	if err != nil {
		t.Fatalf("first MarkApplied() error = %v", err)
	}
	if !marked {
		t.Fatal("first MarkApplied() marked = false, want true")
	}

	marked, err = answerStore.MarkApplied(context.Background(), created.ID, spine.ActorRef{Kind: "user", ID: "dev_1"}, created.CreatedAt)
	if err != nil {
		t.Fatalf("second MarkApplied() error = %v", err)
	}
	if marked {
		t.Fatal("second MarkApplied() marked = true, want false")
	}
}

func validClarificationAnswer() spine.ClarificationAnswer {
	return spine.ClarificationAnswer{
		ID:        "answer-1",
		RequestID: "clarification-1",
		GoalID:    "goal-1",
		Answers: []spine.ClarificationAnswerItem{
			{QuestionID: "question-1", Value: "High-level scope"},
		},
		SubmittedBy: spine.ActorRef{Kind: "user", ID: "dev_1"},
		State:       spine.ClarificationAnswerStateRecorded,
		CreatedAt:   time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC),
	}
}
