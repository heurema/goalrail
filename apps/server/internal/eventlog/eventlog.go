package eventlog

import (
	"context"
	"sync"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

type EventLog struct {
	mu     sync.Mutex
	events []spine.Event
}

func NewEventLog() *EventLog {
	return &EventLog{}
}

func (l *EventLog) Append(_ context.Context, event spine.Event) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.events = append(l.events, cloneEvent(event))
	return nil
}

func (l *EventLog) Events() []spine.Event {
	l.mu.Lock()
	defer l.mu.Unlock()

	events := make([]spine.Event, len(l.events))
	for i, event := range l.events {
		events[i] = cloneEvent(event)
	}
	return events
}

func cloneEvent(event spine.Event) spine.Event {
	if event.Payload != nil {
		event.Payload = append([]byte(nil), event.Payload...)
	}
	return event
}
