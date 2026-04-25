package intake_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/eventlog"
	"github.com/heurema/goalrail/apps/server/internal/intake"
	"github.com/heurema/goalrail/apps/server/internal/spine"
	"github.com/heurema/goalrail/apps/server/internal/store"
)

func TestServiceSubmitAppendsReceivedEvent(t *testing.T) {
	events := eventlog.NewEventLog()
	service := intake.NewService(store.NewIntakeStore(), events, fixedClock{now: testTime()}, &sequenceIDs{})

	record, err := service.Submit(context.Background(), validSubmission())
	if err != nil {
		t.Fatalf("Submit() error = %v", err)
	}

	appended := events.Events()
	if len(appended) != 1 {
		t.Fatalf("events length = %d, want 1", len(appended))
	}
	event := appended[0]
	if event.Type != intake.EventTypeReceived {
		t.Fatalf("event type = %q, want %q", event.Type, intake.EventTypeReceived)
	}
	if event.EntityType != intake.EntityTypeIntake {
		t.Fatalf("entity type = %q, want %q", event.EntityType, intake.EntityTypeIntake)
	}
	if event.EntityID != string(record.ID) {
		t.Fatalf("entity id = %q, want %q", event.EntityID, record.ID)
	}
	if event.ID != "event-1" {
		t.Fatalf("event id = %q, want %q", event.ID, "event-1")
	}

	var payload spine.IntakeRecord
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatalf("unmarshal event payload: %v", err)
	}
	if payload.ID != record.ID {
		t.Fatalf("payload ID = %q, want %q", payload.ID, record.ID)
	}
	if payload.State != spine.IntakeStateReceived {
		t.Fatalf("payload state = %q, want %q", payload.State, spine.IntakeStateReceived)
	}
	if payload.CanonicalContractCreated {
		t.Fatal("payload CanonicalContractCreated = true, want false")
	}
}

func validSubmission() spine.IntakeSubmission {
	return spine.IntakeSubmission{
		RepoBindingID: "repo_demo_1",
		Source: spine.IntakeSource{
			Kind:       "codex_skill",
			ExternalID: "local-session-1",
		},
		Title: "Refactor CSV export filters",
		Body:  "Current code duplicates filter logic. Preserve current behavior.",
		RequestAuthor: spine.ActorRef{
			Kind:        "user",
			ID:          "dev_1",
			DisplayName: "Developer",
		},
	}
}

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

type sequenceIDs struct {
	intake int
	event  int
}

func (g *sequenceIDs) NewIntakeID() (spine.IntakeID, error) {
	g.intake++
	return spine.IntakeID(fmt.Sprintf("intake-%d", g.intake)), nil
}

func (g *sequenceIDs) NewEventID() (spine.EventID, error) {
	g.event++
	return spine.EventID(fmt.Sprintf("event-%d", g.event)), nil
}

func testTime() time.Time {
	return time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
}
