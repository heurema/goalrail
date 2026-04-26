package clarification_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/clarification"
	"github.com/heurema/goalrail/apps/server/internal/eventlog"
	"github.com/heurema/goalrail/apps/server/internal/spine"
	"github.com/heurema/goalrail/apps/server/internal/store"
)

func TestServiceCreateRequestCreatesOpenRequest(t *testing.T) {
	service, goals, _, _ := requestService(t)
	createdGoal := validGoal(spine.GoalReadinessReasonMissingScopeHint, spine.GoalReadinessReasonMissingAcceptanceHint)
	if err := goals.Create(context.Background(), createdGoal); err != nil {
		t.Fatalf("Create goal: %v", err)
	}

	created, err := service.CreateRequest(context.Background(), createdGoal.ID)
	if err != nil {
		t.Fatalf("CreateRequest() error = %v", err)
	}

	if created.State != spine.ClarificationRequestStateOpen {
		t.Fatalf("state = %q, want %q", created.State, spine.ClarificationRequestStateOpen)
	}
	if created.GoalID != createdGoal.ID {
		t.Fatalf("goal_id = %q, want %q", created.GoalID, createdGoal.ID)
	}
	if len(created.Questions) != 2 {
		t.Fatalf("questions length = %d, want 2", len(created.Questions))
	}
	if created.Target.Role != spine.ClarificationTargetRoleIntentOwner {
		t.Fatalf("target role = %q, want %q", created.Target.Role, spine.ClarificationTargetRoleIntentOwner)
	}
	if created.Target.ActorRef == nil || created.Target.ActorRef.ID != createdGoal.IntentOwner.ID {
		t.Fatalf("target actor = %#v, want intent owner", created.Target.ActorRef)
	}
}

func TestServiceCreateRequestQuestionMappings(t *testing.T) {
	tests := []struct {
		name       string
		reason     spine.GoalReadinessReasonCode
		wantText   string
		wantWhy    string
		wantMapsTo spine.ClarificationMapsTo
	}{
		{
			name:       "missing goal summary",
			reason:     spine.GoalReadinessReasonMissingGoalSummary,
			wantText:   "What is the goal summary?",
			wantWhy:    "Goal summary is required before contract seed readiness.",
			wantMapsTo: spine.ClarificationMapsToGoalSummary,
		},
		{
			name:       "missing intent owner",
			reason:     spine.GoalReadinessReasonMissingIntentOwner,
			wantText:   "Who owns the intent for this goal?",
			wantWhy:    "Intent owner is required before contract seed readiness.",
			wantMapsTo: spine.ClarificationMapsToGoalIntentOwner,
		},
		{
			name:       "missing scope hint",
			reason:     spine.GoalReadinessReasonMissingScopeHint,
			wantText:   "What is the intended scope at a high level?",
			wantWhy:    "A scope hint is required before contract seed readiness.",
			wantMapsTo: spine.ClarificationMapsToGoalScopeHint,
		},
		{
			name:       "missing acceptance hint",
			reason:     spine.GoalReadinessReasonMissingAcceptanceHint,
			wantText:   "What outcome would make this goal acceptable?",
			wantWhy:    "An acceptance hint is required before contract seed readiness.",
			wantMapsTo: spine.ClarificationMapsToGoalAcceptanceHint,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, goals, _, _ := requestService(t)
			createdGoal := validGoal(tt.reason)
			createdGoal.ID = spine.GoalID(fmt.Sprintf("goal-%d", i+1))
			createdGoal.IntakeID = spine.IntakeID(fmt.Sprintf("intake-%d", i+1))
			if err := goals.Create(context.Background(), createdGoal); err != nil {
				t.Fatalf("Create goal: %v", err)
			}

			created, err := service.CreateRequest(context.Background(), createdGoal.ID)
			if err != nil {
				t.Fatalf("CreateRequest() error = %v", err)
			}
			if len(created.Questions) != 1 {
				t.Fatalf("questions length = %d, want 1", len(created.Questions))
			}
			question := created.Questions[0]
			if question.Text != tt.wantText {
				t.Fatalf("question text = %q, want %q", question.Text, tt.wantText)
			}
			if question.WhyNeeded != tt.wantWhy {
				t.Fatalf("why_needed = %q, want %q", question.WhyNeeded, tt.wantWhy)
			}
			if question.AnswerType != spine.ClarificationAnswerTypeText {
				t.Fatalf("answer_type = %q, want %q", question.AnswerType, spine.ClarificationAnswerTypeText)
			}
			if question.MapsTo != tt.wantMapsTo {
				t.Fatalf("maps_to = %q, want %q", question.MapsTo, tt.wantMapsTo)
			}
		})
	}
}

func TestServiceCreateRequestAppendsClarificationRequestedEvent(t *testing.T) {
	service, goals, _, events := requestService(t)
	createdGoal := validGoal(spine.GoalReadinessReasonMissingScopeHint)
	if err := goals.Create(context.Background(), createdGoal); err != nil {
		t.Fatalf("Create goal: %v", err)
	}

	created, err := service.CreateRequest(context.Background(), createdGoal.ID)
	if err != nil {
		t.Fatalf("CreateRequest() error = %v", err)
	}

	appended := events.Events()
	if len(appended) != 1 {
		t.Fatalf("events length = %d, want 1", len(appended))
	}
	if appended[0].Type != clarification.EventTypeClarificationRequested {
		t.Fatalf("event type = %q, want %q", appended[0].Type, clarification.EventTypeClarificationRequested)
	}
	if appended[0].EntityType != clarification.EntityTypeClarificationRequest {
		t.Fatalf("entity type = %q, want %q", appended[0].EntityType, clarification.EntityTypeClarificationRequest)
	}
	if appended[0].EntityID != string(created.ID) {
		t.Fatalf("entity id = %q, want %q", appended[0].EntityID, created.ID)
	}

	var payload spine.ClarificationRequest
	if err := json.Unmarshal(appended[0].Payload, &payload); err != nil {
		t.Fatalf("unmarshal clarification.requested payload: %v", err)
	}
	if payload.ID != created.ID {
		t.Fatalf("payload request id = %q, want %q", payload.ID, created.ID)
	}
}

func TestServiceCreateRequestRejectsDuplicateOpenRequest(t *testing.T) {
	service, goals, _, events := requestService(t)
	createdGoal := validGoal(spine.GoalReadinessReasonMissingScopeHint)
	if err := goals.Create(context.Background(), createdGoal); err != nil {
		t.Fatalf("Create goal: %v", err)
	}

	if _, err := service.CreateRequest(context.Background(), createdGoal.ID); err != nil {
		t.Fatalf("first CreateRequest() error = %v", err)
	}
	_, err := service.CreateRequest(context.Background(), createdGoal.ID)
	if !errors.Is(err, clarification.ErrAlreadyOpen) {
		t.Fatalf("second CreateRequest() error = %v, want ErrAlreadyOpen", err)
	}
	if got := len(events.Events()); got != 1 {
		t.Fatalf("events length = %d, want 1", got)
	}
}

func TestServiceCreateRequestRejectsInvalidGoalState(t *testing.T) {
	service, goals, _, _ := requestService(t)
	createdGoal := validGoal(spine.GoalReadinessReasonMissingScopeHint)
	createdGoal.State = spine.GoalStateCreated
	if err := goals.Create(context.Background(), createdGoal); err != nil {
		t.Fatalf("Create goal: %v", err)
	}

	_, err := service.CreateRequest(context.Background(), createdGoal.ID)
	if !errors.Is(err, clarification.ErrInvalidGoalState) {
		t.Fatalf("CreateRequest() error = %v, want ErrInvalidGoalState", err)
	}
}

func TestServiceCreateRequestRejectsUnknownGoal(t *testing.T) {
	service, _, _, _ := requestService(t)

	_, err := service.CreateRequest(context.Background(), "missing")
	if !errors.Is(err, clarification.ErrGoalNotFound) {
		t.Fatalf("CreateRequest() error = %v, want ErrGoalNotFound", err)
	}
}

func TestServiceCreateRequestRejectsMissingReadinessReasons(t *testing.T) {
	service, goals, _, events := requestService(t)
	createdGoal := validGoal()
	if err := goals.Create(context.Background(), createdGoal); err != nil {
		t.Fatalf("Create goal: %v", err)
	}

	_, err := service.CreateRequest(context.Background(), createdGoal.ID)
	if !errors.Is(err, clarification.ErrMissingReadinessReasons) {
		t.Fatalf("CreateRequest() error = %v, want ErrMissingReadinessReasons", err)
	}
	if got := len(events.Events()); got != 0 {
		t.Fatalf("events length = %d, want 0", got)
	}
}

func TestServiceCreateRequestRejectsPolicyRejectedReason(t *testing.T) {
	service, goals, _, events := requestService(t)
	createdGoal := validGoal(spine.GoalReadinessReasonPolicyRejected)
	if err := goals.Create(context.Background(), createdGoal); err != nil {
		t.Fatalf("Create goal: %v", err)
	}

	_, err := service.CreateRequest(context.Background(), createdGoal.ID)
	if !errors.Is(err, clarification.ErrPolicyRejected) {
		t.Fatalf("CreateRequest() error = %v, want ErrPolicyRejected", err)
	}
	if got := len(events.Events()); got != 0 {
		t.Fatalf("events length = %d, want 0", got)
	}
}

func requestService(t *testing.T) (*clarification.Service, *store.GoalStore, *store.ClarificationStore, *eventlog.EventLog) {
	t.Helper()

	goals := store.NewGoalStore()
	clarifications := store.NewClarificationStore()
	events := eventlog.NewEventLog()
	service := clarification.NewService(goals, clarifications, events, fixedClock{now: testTime()}, &sequenceIDs{})
	return service, goals, clarifications, events
}

func validGoal(reasons ...spine.GoalReadinessReasonCode) spine.Goal {
	return spine.Goal{
		ID:            "goal-1",
		IntakeID:      "intake-1",
		RepoBindingID: "repo_demo_1",
		Title:         "Refactor CSV export filters",
		Summary:       "Current code duplicates filter logic. Preserve current behavior.",
		SourceRefs: []spine.SourceRef{
			{Kind: "intake", ID: "intake-1"},
		},
		RequestAuthor:            spine.ActorRef{Kind: "user", ID: "dev_1"},
		IntentOwner:              spine.ActorRef{Kind: "user", ID: "dev_1"},
		State:                    spine.GoalStateNeedsClarification,
		LastReadinessReasonCodes: append([]spine.GoalReadinessReasonCode(nil), reasons...),
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
	clarification int
	question      int
	event         int
}

func (g *sequenceIDs) NewClarificationRequestID() (spine.ClarificationRequestID, error) {
	g.clarification++
	return spine.ClarificationRequestID(fmt.Sprintf("clarification-%d", g.clarification)), nil
}

func (g *sequenceIDs) NewClarificationQuestionID() (spine.ClarificationQuestionID, error) {
	g.question++
	return spine.ClarificationQuestionID(fmt.Sprintf("question-%d", g.question)), nil
}

func (g *sequenceIDs) NewEventID() (spine.EventID, error) {
	g.event++
	return spine.EventID(fmt.Sprintf("event-%d", g.event)), nil
}

func testTime() time.Time {
	return time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
}
