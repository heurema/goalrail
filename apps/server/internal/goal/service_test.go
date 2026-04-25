package goal_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/eventlog"
	"github.com/heurema/goalrail/apps/server/internal/goal"
	"github.com/heurema/goalrail/apps/server/internal/spine"
	"github.com/heurema/goalrail/apps/server/internal/store"
)

func TestServicePromoteFromIntakeAppendsGoalEvents(t *testing.T) {
	intakes := store.NewIntakeStore()
	goals := store.NewGoalStore()
	events := eventlog.NewEventLog()
	service := goal.NewService(intakes, goals, events, fixedClock{now: testTime()}, &sequenceIDs{})
	intakeRecord := validIntakeRecord()
	if err := intakes.Create(context.Background(), intakeRecord); err != nil {
		t.Fatalf("Create intake: %v", err)
	}

	created, err := service.PromoteFromIntake(context.Background(), intakeRecord.ID)
	if err != nil {
		t.Fatalf("PromoteFromIntake() error = %v", err)
	}

	appended := events.Events()
	if len(appended) != 2 {
		t.Fatalf("events length = %d, want 2", len(appended))
	}

	goalCreated := appended[0]
	if goalCreated.Type != goal.EventTypeGoalCreated {
		t.Fatalf("first event type = %q, want %q", goalCreated.Type, goal.EventTypeGoalCreated)
	}
	if goalCreated.EntityType != goal.EntityTypeGoal {
		t.Fatalf("first entity type = %q, want %q", goalCreated.EntityType, goal.EntityTypeGoal)
	}
	if goalCreated.EntityID != string(created.ID) {
		t.Fatalf("first entity id = %q, want %q", goalCreated.EntityID, created.ID)
	}

	var goalPayload spine.Goal
	if err := json.Unmarshal(goalCreated.Payload, &goalPayload); err != nil {
		t.Fatalf("unmarshal goal.created payload: %v", err)
	}
	if goalPayload.ID != created.ID {
		t.Fatalf("payload goal id = %q, want %q", goalPayload.ID, created.ID)
	}
	if goalPayload.State != spine.GoalStateCreated {
		t.Fatalf("payload goal state = %q, want %q", goalPayload.State, spine.GoalStateCreated)
	}

	intakePromoted := appended[1]
	if intakePromoted.Type != goal.EventTypeIntakePromoted {
		t.Fatalf("second event type = %q, want %q", intakePromoted.Type, goal.EventTypeIntakePromoted)
	}
	if intakePromoted.EntityType != goal.EntityTypeIntake {
		t.Fatalf("second entity type = %q, want %q", intakePromoted.EntityType, goal.EntityTypeIntake)
	}
	if intakePromoted.EntityID != string(intakeRecord.ID) {
		t.Fatalf("second entity id = %q, want %q", intakePromoted.EntityID, intakeRecord.ID)
	}

	var promotedPayload struct {
		IntakeID spine.IntakeID `json:"intake_id"`
		GoalID   spine.GoalID   `json:"goal_id"`
	}
	if err := json.Unmarshal(intakePromoted.Payload, &promotedPayload); err != nil {
		t.Fatalf("unmarshal intake.promoted_to_goal payload: %v", err)
	}
	if promotedPayload.IntakeID != intakeRecord.ID {
		t.Fatalf("promoted payload intake id = %q, want %q", promotedPayload.IntakeID, intakeRecord.ID)
	}
	if promotedPayload.GoalID != created.ID {
		t.Fatalf("promoted payload goal id = %q, want %q", promotedPayload.GoalID, created.ID)
	}

	stored, ok, err := goals.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("goals.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("created goal not stored")
	}
	if stored.ID != created.ID {
		t.Fatalf("stored goal id = %q, want %q", stored.ID, created.ID)
	}
}

func TestServicePromoteFromIntakeUsesTitleAsSummaryWhenBodyEmpty(t *testing.T) {
	intakes := store.NewIntakeStore()
	service := goal.NewService(intakes, store.NewGoalStore(), eventlog.NewEventLog(), fixedClock{now: testTime()}, &sequenceIDs{})
	intakeRecord := validIntakeRecord()
	intakeRecord.Body = ""
	if err := intakes.Create(context.Background(), intakeRecord); err != nil {
		t.Fatalf("Create intake: %v", err)
	}

	created, err := service.PromoteFromIntake(context.Background(), intakeRecord.ID)
	if err != nil {
		t.Fatalf("PromoteFromIntake() error = %v", err)
	}
	if created.Summary != intakeRecord.Title {
		t.Fatalf("summary = %q, want title %q", created.Summary, intakeRecord.Title)
	}
}

func TestServicePromoteFromIntakeRejectsDuplicatePromotion(t *testing.T) {
	intakes := store.NewIntakeStore()
	goals := store.NewGoalStore()
	events := eventlog.NewEventLog()
	service := goal.NewService(intakes, goals, events, fixedClock{now: testTime()}, &sequenceIDs{})
	intakeRecord := validIntakeRecord()
	if err := intakes.Create(context.Background(), intakeRecord); err != nil {
		t.Fatalf("Create intake: %v", err)
	}

	if _, err := service.PromoteFromIntake(context.Background(), intakeRecord.ID); err != nil {
		t.Fatalf("first PromoteFromIntake() error = %v", err)
	}
	_, err := service.PromoteFromIntake(context.Background(), intakeRecord.ID)
	if !errors.Is(err, goal.ErrAlreadyPromoted) {
		t.Fatalf("second PromoteFromIntake() error = %v, want ErrAlreadyPromoted", err)
	}
	if got := len(events.Events()); got != 2 {
		t.Fatalf("events length after duplicate = %d, want 2", got)
	}
}

func TestServicePromoteFromIntakeUnknownIntake(t *testing.T) {
	service := goal.NewService(store.NewIntakeStore(), store.NewGoalStore(), eventlog.NewEventLog(), fixedClock{now: testTime()}, &sequenceIDs{})

	_, err := service.PromoteFromIntake(context.Background(), "missing")
	if !errors.Is(err, goal.ErrIntakeNotFound) {
		t.Fatalf("PromoteFromIntake() error = %v, want ErrIntakeNotFound", err)
	}
}

func TestServicePromoteFromIntakeRejectsNonReceivedIntake(t *testing.T) {
	intakes := store.NewIntakeStore()
	intakeRecord := validIntakeRecord()
	intakeRecord.State = "rejected"
	if err := intakes.Create(context.Background(), intakeRecord); err != nil {
		t.Fatalf("Create intake: %v", err)
	}
	service := goal.NewService(intakes, store.NewGoalStore(), eventlog.NewEventLog(), fixedClock{now: testTime()}, &sequenceIDs{})

	_, err := service.PromoteFromIntake(context.Background(), intakeRecord.ID)
	if !errors.Is(err, goal.ErrInvalidIntakeState) {
		t.Fatalf("PromoteFromIntake() error = %v, want ErrInvalidIntakeState", err)
	}
}

func TestServicePromoteFromIntakeValidatesStoredIntake(t *testing.T) {
	intakes := store.NewIntakeStore()
	intakeRecord := validIntakeRecord()
	intakeRecord.RepoBindingID = ""
	if err := intakes.Create(context.Background(), intakeRecord); err != nil {
		t.Fatalf("Create intake: %v", err)
	}
	service := goal.NewService(intakes, store.NewGoalStore(), eventlog.NewEventLog(), fixedClock{now: testTime()}, &sequenceIDs{})

	_, err := service.PromoteFromIntake(context.Background(), intakeRecord.ID)
	var validationErr *goal.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("PromoteFromIntake() error = %v, want ValidationError", err)
	}
}

func TestServicePromoteFromIntakeReturnsErrorWhenStoreFails(t *testing.T) {
	events := &recordingEventLog{}
	service := goal.NewService(
		intakeReader{record: validIntakeRecord(), ok: true},
		failingGoalStore{err: errors.New("store failed")},
		events,
		fixedClock{now: testTime()},
		&sequenceIDs{},
	)

	_, err := service.PromoteFromIntake(context.Background(), "intake-1")
	if err == nil {
		t.Fatal("PromoteFromIntake() error = nil, want error")
	}
	if len(events.events) != 0 {
		t.Fatalf("events length = %d, want 0", len(events.events))
	}
}

func TestServicePromoteFromIntakeReturnsErrorWhenSecondEventAppendFails(t *testing.T) {
	events := &recordingEventLog{failOn: 2, err: errors.New("append failed")}
	goals := &recordingGoalStore{}
	service := goal.NewService(
		intakeReader{record: validIntakeRecord(), ok: true},
		goals,
		events,
		fixedClock{now: testTime()},
		&sequenceIDs{},
	)

	_, err := service.PromoteFromIntake(context.Background(), "intake-1")
	if err == nil {
		t.Fatal("PromoteFromIntake() error = nil, want error")
	}
	if goals.created.ID == "" {
		t.Fatal("goal was not stored before append failure")
	}
	if len(events.events) != 1 {
		t.Fatalf("events length = %d, want first event only", len(events.events))
	}
}

func validIntakeRecord() spine.IntakeRecord {
	return spine.IntakeRecord{
		ID:            "intake-1",
		RepoBindingID: "repo_demo_1",
		Source:        spine.IntakeSource{Kind: "codex_skill"},
		Title:         "Refactor CSV export filters",
		Body:          "Current code duplicates filter logic. Preserve current behavior.",
		RequestAuthor: spine.ActorRef{
			Kind:        "user",
			ID:          "dev_1",
			DisplayName: "Developer",
		},
		IntentOwner: spine.ActorRef{
			Kind:        "user",
			ID:          "dev_1",
			DisplayName: "Developer",
		},
		State:                    spine.IntakeStateReceived,
		CanonicalContractCreated: false,
		CreatedAt:                testTime(),
	}
}

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

type sequenceIDs struct {
	goal  int
	event int
}

func (g *sequenceIDs) NewGoalID() (spine.GoalID, error) {
	g.goal++
	return spine.GoalID(fmt.Sprintf("goal-%d", g.goal)), nil
}

func (g *sequenceIDs) NewEventID() (spine.EventID, error) {
	g.event++
	return spine.EventID(fmt.Sprintf("event-%d", g.event)), nil
}

func testTime() time.Time {
	return time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
}

type intakeReader struct {
	record spine.IntakeRecord
	ok     bool
	err    error
}

func (r intakeReader) Get(context.Context, spine.IntakeID) (spine.IntakeRecord, bool, error) {
	return r.record, r.ok, r.err
}

type failingGoalStore struct {
	err error
}

func (s failingGoalStore) Create(context.Context, spine.Goal) error {
	return s.err
}

func (s failingGoalStore) Get(context.Context, spine.GoalID) (spine.Goal, bool, error) {
	return spine.Goal{}, false, nil
}

func (s failingGoalStore) GetByIntakeID(context.Context, spine.IntakeID) (spine.Goal, bool, error) {
	return spine.Goal{}, false, nil
}

type recordingGoalStore struct {
	created spine.Goal
}

func (s *recordingGoalStore) Create(_ context.Context, created spine.Goal) error {
	s.created = created
	return nil
}

func (s *recordingGoalStore) Get(_ context.Context, id spine.GoalID) (spine.Goal, bool, error) {
	if s.created.ID != id {
		return spine.Goal{}, false, nil
	}
	return s.created, true, nil
}

func (s *recordingGoalStore) GetByIntakeID(_ context.Context, id spine.IntakeID) (spine.Goal, bool, error) {
	if s.created.IntakeID != id {
		return spine.Goal{}, false, nil
	}
	return s.created, true, nil
}

type recordingEventLog struct {
	events []spine.Event
	failOn int
	err    error
}

func (l *recordingEventLog) Append(_ context.Context, event spine.Event) error {
	if l.failOn > 0 && len(l.events)+1 == l.failOn {
		return l.err
	}
	l.events = append(l.events, event)
	return nil
}
