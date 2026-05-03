package intake_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/heurema/goalrail/apps/server/internal/intake"
	"github.com/heurema/goalrail/apps/server/internal/spine"
	"testing"
	"time"
)

func TestServiceSubmitAppendsReceivedEvent(t *testing.T) {
	events := newFakeEventLog()
	service := intake.NewService(newFakeIntakeStore(), validProjectContextResolver(), events, fixedClock{now: testTime()}, &sequenceIDs{})

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
	if event.OrganizationID != "018f0000-0000-7000-8000-000000000002" {
		t.Fatalf("event organization_id = %q, want 018f0000-0000-7000-8000-000000000002", event.OrganizationID)
	}
	if event.ProjectID != "018f0000-0000-7000-8000-000000000003" {
		t.Fatalf("event project_id = %q, want 018f0000-0000-7000-8000-000000000003", event.ProjectID)
	}
	if event.RepoBindingID != "018f0000-0000-7000-8000-000000000004" {
		t.Fatalf("event repo_binding_id = %q, want 018f0000-0000-7000-8000-000000000004", event.RepoBindingID)
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
	if payload.OrganizationID != "018f0000-0000-7000-8000-000000000002" {
		t.Fatalf("payload organization_id = %q, want 018f0000-0000-7000-8000-000000000002", payload.OrganizationID)
	}
	if payload.ProjectID != "018f0000-0000-7000-8000-000000000003" {
		t.Fatalf("payload project_id = %q, want 018f0000-0000-7000-8000-000000000003", payload.ProjectID)
	}
	if payload.RepoBindingID != "018f0000-0000-7000-8000-000000000004" {
		t.Fatalf("payload repo_binding_id = %q, want 018f0000-0000-7000-8000-000000000004", payload.RepoBindingID)
	}
}

func TestValidateSubmissionRejectsMissingProjectID(t *testing.T) {
	submission := validSubmission()
	submission.ProjectID = ""

	err := intake.ValidateSubmission(submission)
	var validationErr *intake.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("ValidateSubmission() error = %v, want ValidationError", err)
	}
	if validationErr.Field != "project_id" {
		t.Fatalf("validation field = %q, want project_id", validationErr.Field)
	}
}

func TestServiceSubmitRejectsUnknownRepoBinding(t *testing.T) {
	events := newFakeEventLog()
	resolver := fakeProjectContextResolver{ok: false}
	service := intake.NewService(newFakeIntakeStore(), resolver, events, fixedClock{now: testTime()}, &sequenceIDs{})

	_, err := service.Submit(context.Background(), validSubmission())
	var validationErr *intake.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("Submit() error = %v, want ValidationError", err)
	}
	if validationErr.Field != "repo_binding_id" {
		t.Fatalf("validation field = %q, want repo_binding_id", validationErr.Field)
	}
	if got := len(events.Events()); got != 0 {
		t.Fatalf("events length = %d, want 0", got)
	}
}

func TestServiceSubmitRejectsRepoBindingForDifferentProject(t *testing.T) {
	events := newFakeEventLog()
	resolver := fakeProjectContextResolver{
		resolved: spine.ResolvedRepoBindingContext{
			OrganizationID: "018f0000-0000-7000-8000-000000000002",
			ProjectID:      "018f0000-0000-7000-8000-000000000006",
			RepoBindingID:  "018f0000-0000-7000-8000-000000000004",
		},
		ok: true,
	}
	service := intake.NewService(newFakeIntakeStore(), resolver, events, fixedClock{now: testTime()}, &sequenceIDs{})

	_, err := service.Submit(context.Background(), validSubmission())
	var validationErr *intake.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("Submit() error = %v, want ValidationError", err)
	}
	if validationErr.Field != "repo_binding_id" {
		t.Fatalf("validation field = %q, want repo_binding_id", validationErr.Field)
	}
	if got := len(events.Events()); got != 0 {
		t.Fatalf("events length = %d, want 0", got)
	}
}

func TestServiceSubmitRejectsUnavailableProjectContext(t *testing.T) {
	events := newFakeEventLog()
	service := intake.NewService(newFakeIntakeStore(), nil, events, fixedClock{now: testTime()}, &sequenceIDs{})

	_, err := service.Submit(context.Background(), validSubmission())
	if !errors.Is(err, intake.ErrProjectContextUnavailable) {
		t.Fatalf("Submit() error = %v, want ErrProjectContextUnavailable", err)
	}
	if got := len(events.Events()); got != 0 {
		t.Fatalf("events length = %d, want 0", got)
	}
}

func validSubmission() spine.IntakeSubmission {
	return spine.IntakeSubmission{
		ProjectID:     "018f0000-0000-7000-8000-000000000003",
		RepoBindingID: "018f0000-0000-7000-8000-000000000004",
		Source: spine.IntakeSource{
			Kind:       "codex_skill",
			ExternalID: "local-session-1",
		},
		Title: "Refactor CSV export filters",
		Body:  "Current code duplicates filter logic. Preserve current behavior.",
		RequestAuthor: spine.ActorRef{
			Kind:        "user",
			ID:          "018f0000-0000-7000-8000-000000000001",
			DisplayName: "Developer",
		},
	}
}

func validProjectContextResolver() fakeProjectContextResolver {
	return fakeProjectContextResolver{
		resolved: spine.ResolvedRepoBindingContext{
			OrganizationID: "018f0000-0000-7000-8000-000000000002",
			ProjectID:      "018f0000-0000-7000-8000-000000000003",
			RepoBindingID:  "018f0000-0000-7000-8000-000000000004",
		},
		ok: true,
	}
}

type fakeProjectContextResolver struct {
	resolved spine.ResolvedRepoBindingContext
	ok       bool
	err      error
}

func (r fakeProjectContextResolver) ResolveRepoBinding(context.Context, spine.RepoBindingID) (spine.ResolvedRepoBindingContext, bool, error) {
	return r.resolved, r.ok, r.err
}

type fakeIntakeStore struct {
	records map[spine.IntakeID]spine.IntakeRecord
}

func newFakeIntakeStore() *fakeIntakeStore {
	return &fakeIntakeStore{records: map[spine.IntakeID]spine.IntakeRecord{}}
}

func (s *fakeIntakeStore) Create(_ context.Context, record spine.IntakeRecord) error {
	s.records[record.ID] = record
	return nil
}

func (s *fakeIntakeStore) Get(_ context.Context, id spine.IntakeID) (spine.IntakeRecord, bool, error) {
	record, ok := s.records[id]
	return record, ok, nil
}

type fakeEventLog struct {
	events []spine.Event
}

func newFakeEventLog() *fakeEventLog {
	return &fakeEventLog{}
}

func (l *fakeEventLog) Append(_ context.Context, event spine.Event) error {
	l.events = append(l.events, cloneEvent(event))
	return nil
}

func (l *fakeEventLog) Events() []spine.Event {
	events := make([]spine.Event, len(l.events))
	for i, event := range l.events {
		events[i] = cloneEvent(event)
	}
	return events
}

func cloneEvent(event spine.Event) spine.Event {
	event.Payload = append([]byte(nil), event.Payload...)
	return event
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
