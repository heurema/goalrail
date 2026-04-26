package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/spine"
	"github.com/heurema/goalrail/apps/server/internal/store"
)

func TestIntakeStoreCreateAndGet(t *testing.T) {
	intakeStore := store.NewIntakeStore()
	record := spine.IntakeRecord{
		ID:            "intake-1",
		RepoBindingID: "018f0000-0000-7000-8000-000000000004",
		Source: spine.IntakeSource{
			Kind: "codex_skill",
		},
		Title: "Refactor CSV export filters",
		RequestAuthor: spine.ActorRef{
			Kind: "user",
			ID:   "018f0000-0000-7000-8000-000000000001",
		},
		IntentOwner: spine.ActorRef{
			Kind: "user",
			ID:   "018f0000-0000-7000-8000-000000000001",
		},
		State:                    spine.IntakeStateReceived,
		CanonicalContractCreated: false,
		CreatedAt:                time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC),
	}

	if err := intakeStore.Create(context.Background(), record); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, ok, err := intakeStore.Get(context.Background(), record.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !ok {
		t.Fatal("Get() ok = false, want true")
	}
	if got != record {
		t.Fatalf("Get() = %#v, want %#v", got, record)
	}
}
