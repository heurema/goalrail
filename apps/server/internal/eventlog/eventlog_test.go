package eventlog_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/eventlog"
	"github.com/heurema/goalrail/apps/server/internal/spine"
)

func TestEventLogPreservesAppendOrder(t *testing.T) {
	log := eventlog.NewEventLog()
	events := []spine.Event{
		newEvent("event-1", "intake-1"),
		newEvent("event-2", "intake-2"),
	}

	for _, event := range events {
		if err := log.Append(context.Background(), event); err != nil {
			t.Fatalf("Append() error = %v", err)
		}
	}

	got := log.Events()
	if len(got) != len(events) {
		t.Fatalf("events length = %d, want %d", len(got), len(events))
	}
	for i := range events {
		if got[i].ID != events[i].ID {
			t.Fatalf("event[%d].ID = %q, want %q", i, got[i].ID, events[i].ID)
		}
		if got[i].EntityID != events[i].EntityID {
			t.Fatalf("event[%d].EntityID = %q, want %q", i, got[i].EntityID, events[i].EntityID)
		}
	}
}

func newEvent(id spine.EventID, entityID string) spine.Event {
	payload, _ := json.Marshal(map[string]string{"id": entityID})
	return spine.Event{
		ID:         id,
		Type:       "intake.received",
		EntityType: "IntakeRecord",
		EntityID:   entityID,
		Timestamp:  time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC),
		Payload:    payload,
	}
}
