package clarification_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/clarification"
	"github.com/heurema/goalrail/apps/server/internal/spine"
)

func TestServiceCreateRequestCreatesOpenRequest(t *testing.T) {
	service, goals, _, _, _ := requestService(t)
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
			service, goals, _, _, _ := requestService(t)
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
	service, goals, _, _, events := requestService(t)
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

func TestServiceCreateRequestUsesTransactionRunnerWhenConfigured(t *testing.T) {
	goals := newFakeGoalStore()
	clarifications := newFakeClarificationStore()
	answers := newFakeClarificationAnswerStore()
	events := newFakeEventLog()
	txRunner := &fakeTransactionRunner{}
	service := clarification.NewService(
		goals,
		clarifications,
		answers,
		events,
		fixedClock{now: testTime()},
		&sequenceIDs{},
		clarification.WithTransactionRunner(txRunner),
	)
	createdGoal := validGoal(spine.GoalReadinessReasonMissingScopeHint)
	if err := goals.Create(context.Background(), createdGoal); err != nil {
		t.Fatalf("Create goal: %v", err)
	}

	created, err := service.CreateRequest(context.Background(), createdGoal.ID)
	if err != nil {
		t.Fatalf("CreateRequest() error = %v", err)
	}

	if txRunner.calls != 1 {
		t.Fatalf("TxRunner calls = %d, want 1", txRunner.calls)
	}
	if !clarifications.createSawTransaction {
		t.Fatal("clarification request Create did not run inside transaction runner")
	}
	if events.transactionalAppends != 1 {
		t.Fatalf("transactional event appends = %d, want 1", events.transactionalAppends)
	}
	stored, ok, err := clarifications.GetOpenByGoalID(context.Background(), createdGoal.ID)
	if err != nil {
		t.Fatalf("GetOpenByGoalID() error = %v", err)
	}
	if !ok {
		t.Fatal("stored request not found")
	}
	if stored.ID != created.ID {
		t.Fatalf("stored request id = %q, want %q", stored.ID, created.ID)
	}
}

func TestServiceCreateRequestTransactionRunnerFailureDoesNotUseFallbackStore(t *testing.T) {
	goals := newFakeGoalStore()
	clarifications := newFakeClarificationStore()
	answers := newFakeClarificationAnswerStore()
	events := newFakeEventLog()
	txRunner := &fakeTransactionRunner{err: errors.New("forced transaction failure")}
	service := clarification.NewService(
		goals,
		clarifications,
		answers,
		events,
		fixedClock{now: testTime()},
		&sequenceIDs{},
		clarification.WithTransactionRunner(txRunner),
	)
	createdGoal := validGoal(spine.GoalReadinessReasonMissingScopeHint)
	if err := goals.Create(context.Background(), createdGoal); err != nil {
		t.Fatalf("Create goal: %v", err)
	}

	_, err := service.CreateRequest(context.Background(), createdGoal.ID)
	if err == nil {
		t.Fatal("CreateRequest() error = nil, want transaction failure")
	}
	if txRunner.calls != 1 {
		t.Fatalf("TxRunner calls = %d, want 1", txRunner.calls)
	}
	if _, ok, err := clarifications.GetOpenByGoalID(context.Background(), createdGoal.ID); err != nil {
		t.Fatalf("GetOpenByGoalID() error = %v", err)
	} else if ok {
		t.Fatal("stored request found after transaction failure")
	}
	if got := len(events.Events()); got != 0 {
		t.Fatalf("events length = %d, want 0", got)
	}
}

func TestServiceCreateRequestDoesNotAppendReadinessOrContractEvents(t *testing.T) {
	service, goals, _, _, events := requestService(t)
	createdGoal := validGoal(spine.GoalReadinessReasonMissingScopeHint)
	if err := goals.Create(context.Background(), createdGoal); err != nil {
		t.Fatalf("Create goal: %v", err)
	}

	if _, err := service.CreateRequest(context.Background(), createdGoal.ID); err != nil {
		t.Fatalf("CreateRequest() error = %v", err)
	}

	for _, event := range events.Events() {
		switch event.Type {
		case "goal.readiness_recheck_requested", "contract.seed_created", "contract.created", "plan.created", "proposal.created", "task.created", "work_item.created", "run.created", "gate.decision_written", "proof.created":
			t.Fatalf("unexpected event type %q", event.Type)
		}
	}
}

func TestServiceCreateRequestRejectsDuplicateOpenRequest(t *testing.T) {
	service, goals, _, _, events := requestService(t)
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
	service, goals, _, _, _ := requestService(t)
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
	service, _, _, _, _ := requestService(t)

	_, err := service.CreateRequest(context.Background(), "missing")
	if !errors.Is(err, clarification.ErrGoalNotFound) {
		t.Fatalf("CreateRequest() error = %v, want ErrGoalNotFound", err)
	}
}

func TestServiceCreateRequestRejectsMissingReadinessReasons(t *testing.T) {
	service, goals, _, _, events := requestService(t)
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
	service, goals, _, _, events := requestService(t)
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

func TestServiceRecordAnswerRecordsAnswerAndMarksRequestAnswered(t *testing.T) {
	service, goals, requests, answers, _ := requestService(t)
	request := createRequest(t, service, goals, spine.GoalReadinessReasonMissingScopeHint, spine.GoalReadinessReasonMissingAcceptanceHint)

	recorded, err := service.RecordAnswer(context.Background(), request.ID, answerSubmission(request))
	if err != nil {
		t.Fatalf("RecordAnswer() error = %v", err)
	}

	if recorded.State != spine.ClarificationAnswerStateRecorded {
		t.Fatalf("answer state = %q, want %q", recorded.State, spine.ClarificationAnswerStateRecorded)
	}
	if recorded.RequestID != request.ID {
		t.Fatalf("request_id = %q, want %q", recorded.RequestID, request.ID)
	}
	if recorded.GoalID != request.GoalID {
		t.Fatalf("goal_id = %q, want %q", recorded.GoalID, request.GoalID)
	}

	storedAnswer, ok, err := answers.Get(context.Background(), recorded.ID)
	if err != nil {
		t.Fatalf("answers.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("stored answer not found")
	}
	if storedAnswer.ID != recorded.ID {
		t.Fatalf("stored answer id = %q, want %q", storedAnswer.ID, recorded.ID)
	}

	storedRequest, ok, err := requests.Get(context.Background(), request.ID)
	if err != nil {
		t.Fatalf("requests.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("stored request not found")
	}
	if storedRequest.State != spine.ClarificationRequestStateAnswered {
		t.Fatalf("request state = %q, want %q", storedRequest.State, spine.ClarificationRequestStateAnswered)
	}
}

func TestServiceRecordAnswerAppendsAnswerEvents(t *testing.T) {
	service, goals, _, _, events := requestService(t)
	request := createRequest(t, service, goals, spine.GoalReadinessReasonMissingScopeHint)

	recorded, err := service.RecordAnswer(context.Background(), request.ID, answerSubmission(request))
	if err != nil {
		t.Fatalf("RecordAnswer() error = %v", err)
	}

	appended := events.Events()
	if len(appended) != 3 {
		t.Fatalf("events length = %d, want 3", len(appended))
	}
	if appended[1].Type != clarification.EventTypeClarificationAnswerRecorded {
		t.Fatalf("event[1] type = %q, want %q", appended[1].Type, clarification.EventTypeClarificationAnswerRecorded)
	}
	if appended[1].EntityType != clarification.EntityTypeClarificationAnswer {
		t.Fatalf("event[1] entity type = %q, want %q", appended[1].EntityType, clarification.EntityTypeClarificationAnswer)
	}
	if appended[1].EntityID != string(recorded.ID) {
		t.Fatalf("event[1] entity id = %q, want %q", appended[1].EntityID, recorded.ID)
	}
	if appended[2].Type != clarification.EventTypeClarificationRequestAnswered {
		t.Fatalf("event[2] type = %q, want %q", appended[2].Type, clarification.EventTypeClarificationRequestAnswered)
	}
	if appended[2].EntityType != clarification.EntityTypeClarificationRequest {
		t.Fatalf("event[2] entity type = %q, want %q", appended[2].EntityType, clarification.EntityTypeClarificationRequest)
	}
	if appended[2].EntityID != string(request.ID) {
		t.Fatalf("event[2] entity id = %q, want %q", appended[2].EntityID, request.ID)
	}

	var answerPayload spine.ClarificationAnswer
	if err := json.Unmarshal(appended[1].Payload, &answerPayload); err != nil {
		t.Fatalf("unmarshal answer payload: %v", err)
	}
	if answerPayload.ID != recorded.ID {
		t.Fatalf("answer payload id = %q, want %q", answerPayload.ID, recorded.ID)
	}

	var requestPayload struct {
		RequestID     spine.ClarificationRequestID    `json:"request_id"`
		AnswerID      spine.ClarificationAnswerID     `json:"answer_id"`
		PreviousState spine.ClarificationRequestState `json:"previous_state"`
		NewState      spine.ClarificationRequestState `json:"new_state"`
	}
	if err := json.Unmarshal(appended[2].Payload, &requestPayload); err != nil {
		t.Fatalf("unmarshal request answered payload: %v", err)
	}
	if requestPayload.RequestID != request.ID {
		t.Fatalf("request payload request_id = %q, want %q", requestPayload.RequestID, request.ID)
	}
	if requestPayload.AnswerID != recorded.ID {
		t.Fatalf("request payload answer_id = %q, want %q", requestPayload.AnswerID, recorded.ID)
	}
	if requestPayload.PreviousState != spine.ClarificationRequestStateOpen {
		t.Fatalf("request payload previous_state = %q, want %q", requestPayload.PreviousState, spine.ClarificationRequestStateOpen)
	}
	if requestPayload.NewState != spine.ClarificationRequestStateAnswered {
		t.Fatalf("request payload new_state = %q, want %q", requestPayload.NewState, spine.ClarificationRequestStateAnswered)
	}
}

func TestServiceRecordAnswerUsesTransactionRunnerWhenConfigured(t *testing.T) {
	baseService, goals, requests, answers, events := requestService(t)
	request := createRequest(t, baseService, goals, spine.GoalReadinessReasonMissingScopeHint)
	txRunner := &fakeTransactionRunner{}
	service := clarification.NewService(
		goals,
		requests,
		answers,
		events,
		fixedClock{now: testTime()},
		&sequenceIDs{},
		clarification.WithTransactionRunner(txRunner),
	)

	recorded, err := service.RecordAnswer(context.Background(), request.ID, answerSubmission(request))
	if err != nil {
		t.Fatalf("RecordAnswer() error = %v", err)
	}

	if txRunner.calls != 1 {
		t.Fatalf("TxRunner calls = %d, want 1", txRunner.calls)
	}
	if !answers.createSawTransaction {
		t.Fatal("clarification answer Create did not run inside transaction runner")
	}
	if !requests.updateStateSawTransaction {
		t.Fatal("clarification request UpdateState did not run inside transaction runner")
	}
	if events.transactionalAppends != 2 {
		t.Fatalf("transactional event appends = %d, want 2", events.transactionalAppends)
	}
	if _, ok, err := answers.Get(context.Background(), recorded.ID); err != nil {
		t.Fatalf("answers.Get() error = %v", err)
	} else if !ok {
		t.Fatal("stored answer not found")
	}
}

func TestServiceRecordAnswerRejectsRepeatedAnswer(t *testing.T) {
	service, goals, _, _, events := requestService(t)
	request := createRequest(t, service, goals, spine.GoalReadinessReasonMissingScopeHint)
	input := answerSubmission(request)

	if _, err := service.RecordAnswer(context.Background(), request.ID, input); err != nil {
		t.Fatalf("first RecordAnswer() error = %v", err)
	}
	_, err := service.RecordAnswer(context.Background(), request.ID, input)
	if !errors.Is(err, clarification.ErrAlreadyAnswered) {
		t.Fatalf("second RecordAnswer() error = %v, want ErrAlreadyAnswered", err)
	}
	if got := len(events.Events()); got != 3 {
		t.Fatalf("events length = %d, want 3", got)
	}
}

func TestServiceRecordAnswerReturnsNotFoundWhenRequestUpdateFindsNoRow(t *testing.T) {
	baseService, goals, requests, answers, events := requestService(t)
	request := createRequest(t, baseService, goals, spine.GoalReadinessReasonMissingScopeHint)
	requests.updateStateMissing = true

	_, err := baseService.RecordAnswer(context.Background(), request.ID, answerSubmission(request))
	if !errors.Is(err, clarification.ErrRequestNotFound) {
		t.Fatalf("RecordAnswer() error = %v, want ErrRequestNotFound", err)
	}
	if got := len(events.Events()); got != 1 {
		t.Fatalf("events length = %d, want only original request event", got)
	}
	if _, ok, err := answers.GetByRequestID(context.Background(), request.ID); err != nil {
		t.Fatalf("answers.GetByRequestID() error = %v", err)
	} else if !ok {
		t.Fatal("answer was not created before request update returned not found")
	}
}

func TestServiceRecordAnswerValidation(t *testing.T) {
	tests := []struct {
		name string
		edit func(spine.ClarificationRequest, *spine.ClarificationAnswerSubmission)
	}{
		{
			name: "missing submitted_by kind",
			edit: func(_ spine.ClarificationRequest, input *spine.ClarificationAnswerSubmission) {
				input.SubmittedBy.Kind = ""
			},
		},
		{
			name: "missing submitted_by id",
			edit: func(_ spine.ClarificationRequest, input *spine.ClarificationAnswerSubmission) {
				input.SubmittedBy.ID = ""
			},
		},
		{
			name: "missing answers",
			edit: func(_ spine.ClarificationRequest, input *spine.ClarificationAnswerSubmission) {
				input.Answers = nil
			},
		},
		{
			name: "unknown question_id",
			edit: func(_ spine.ClarificationRequest, input *spine.ClarificationAnswerSubmission) {
				input.Answers[0].QuestionID = "unknown"
			},
		},
		{
			name: "duplicate question_id",
			edit: func(request spine.ClarificationRequest, input *spine.ClarificationAnswerSubmission) {
				input.Answers = []spine.ClarificationAnswerItem{
					{QuestionID: request.Questions[0].ID, Value: "Scope"},
					{QuestionID: request.Questions[0].ID, Value: "Duplicate"},
				}
			},
		},
		{
			name: "missing answer for one question",
			edit: func(_ spine.ClarificationRequest, input *spine.ClarificationAnswerSubmission) {
				input.Answers = input.Answers[:1]
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, goals, _, _, _ := requestService(t)
			request := createRequest(t, service, goals, spine.GoalReadinessReasonMissingScopeHint, spine.GoalReadinessReasonMissingAcceptanceHint)
			input := answerSubmission(request)
			tt.edit(request, &input)

			_, err := service.RecordAnswer(context.Background(), request.ID, input)
			var validationErr *clarification.ValidationError
			if !errors.As(err, &validationErr) {
				t.Fatalf("RecordAnswer() error = %v, want ValidationError", err)
			}
		})
	}
}

func TestServiceApplyAnswerUpdatesGoalHints(t *testing.T) {
	service, goals, _, _, _ := requestService(t)
	request := createRequest(t, service, goals,
		spine.GoalReadinessReasonMissingGoalSummary,
		spine.GoalReadinessReasonMissingScopeHint,
		spine.GoalReadinessReasonMissingAcceptanceHint,
	)
	recorded, err := service.RecordAnswer(context.Background(), request.ID, answerSubmissionWithValues(request, map[spine.ClarificationMapsTo]string{
		spine.ClarificationMapsToGoalSummary:        "Updated summary",
		spine.ClarificationMapsToGoalScopeHint:      "Updated scope hint",
		spine.ClarificationMapsToGoalAcceptanceHint: "Updated acceptance hint",
	}))
	if err != nil {
		t.Fatalf("RecordAnswer() error = %v", err)
	}

	application, updatedGoal, err := service.ApplyAnswer(context.Background(), recorded.ID, applyRequest())
	if err != nil {
		t.Fatalf("ApplyAnswer() error = %v", err)
	}

	if application.AnswerID != recorded.ID {
		t.Fatalf("application answer_id = %q, want %q", application.AnswerID, recorded.ID)
	}
	if application.GoalID != recorded.GoalID {
		t.Fatalf("application goal_id = %q, want %q", application.GoalID, recorded.GoalID)
	}
	if len(application.AppliedMappings) != 3 {
		t.Fatalf("applied mappings length = %d, want 3", len(application.AppliedMappings))
	}
	if updatedGoal.Summary != "Updated summary" {
		t.Fatalf("summary = %q, want updated summary", updatedGoal.Summary)
	}
	if updatedGoal.ScopeHint != "Updated scope hint" {
		t.Fatalf("scope_hint = %q, want updated scope hint", updatedGoal.ScopeHint)
	}
	if updatedGoal.AcceptanceHint != "Updated acceptance hint" {
		t.Fatalf("acceptance_hint = %q, want updated acceptance hint", updatedGoal.AcceptanceHint)
	}

	storedGoal, ok, err := goals.Get(context.Background(), updatedGoal.ID)
	if err != nil {
		t.Fatalf("goals.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("stored goal not found")
	}
	if storedGoal.ScopeHint != "Updated scope hint" {
		t.Fatalf("stored scope_hint = %q, want updated scope hint", storedGoal.ScopeHint)
	}
}

func TestServiceApplyAnswerAppendsApplicationEvents(t *testing.T) {
	service, goals, _, _, events := requestService(t)
	request := createRequest(t, service, goals, spine.GoalReadinessReasonMissingScopeHint)
	recorded, err := service.RecordAnswer(context.Background(), request.ID, answerSubmissionWithValues(request, map[spine.ClarificationMapsTo]string{
		spine.ClarificationMapsToGoalScopeHint: "Updated scope hint",
	}))
	if err != nil {
		t.Fatalf("RecordAnswer() error = %v", err)
	}

	application, _, err := service.ApplyAnswer(context.Background(), recorded.ID, applyRequest())
	if err != nil {
		t.Fatalf("ApplyAnswer() error = %v", err)
	}

	appended := events.Events()
	if len(appended) != 5 {
		t.Fatalf("events length = %d, want 5", len(appended))
	}
	if appended[3].Type != clarification.EventTypeClarificationAnswerApplied {
		t.Fatalf("event[3] type = %q, want %q", appended[3].Type, clarification.EventTypeClarificationAnswerApplied)
	}
	if appended[3].EntityType != clarification.EntityTypeClarificationAnswer {
		t.Fatalf("event[3] entity type = %q, want %q", appended[3].EntityType, clarification.EntityTypeClarificationAnswer)
	}
	if appended[3].EntityID != string(recorded.ID) {
		t.Fatalf("event[3] entity id = %q, want %q", appended[3].EntityID, recorded.ID)
	}
	if appended[4].Type != clarification.EventTypeGoalHintsUpdated {
		t.Fatalf("event[4] type = %q, want %q", appended[4].Type, clarification.EventTypeGoalHintsUpdated)
	}
	if appended[4].EntityType != clarification.EntityTypeGoal {
		t.Fatalf("event[4] entity type = %q, want %q", appended[4].EntityType, clarification.EntityTypeGoal)
	}
	if appended[4].EntityID != string(recorded.GoalID) {
		t.Fatalf("event[4] entity id = %q, want %q", appended[4].EntityID, recorded.GoalID)
	}

	var payload spine.ClarificationAnswerApplicationResult
	if err := json.Unmarshal(appended[3].Payload, &payload); err != nil {
		t.Fatalf("unmarshal application payload: %v", err)
	}
	if payload.AnswerID != application.AnswerID {
		t.Fatalf("payload answer_id = %q, want %q", payload.AnswerID, application.AnswerID)
	}
}

func TestServiceApplyAnswerUsesTransactionRunnerWhenConfigured(t *testing.T) {
	baseService, goals, requests, answers, events := requestService(t)
	request := createRequest(t, baseService, goals, spine.GoalReadinessReasonMissingScopeHint)
	recorded, err := baseService.RecordAnswer(context.Background(), request.ID, answerSubmission(request))
	if err != nil {
		t.Fatalf("RecordAnswer() error = %v", err)
	}
	txRunner := &fakeTransactionRunner{}
	service := clarification.NewService(
		goals,
		requests,
		answers,
		events,
		fixedClock{now: testTime()},
		&sequenceIDs{},
		clarification.WithTransactionRunner(txRunner),
	)

	_, updatedGoal, err := service.ApplyAnswer(context.Background(), recorded.ID, applyRequest())
	if err != nil {
		t.Fatalf("ApplyAnswer() error = %v", err)
	}

	if txRunner.calls != 1 {
		t.Fatalf("TxRunner calls = %d, want 1", txRunner.calls)
	}
	if !answers.markAppliedSawTransaction {
		t.Fatal("clarification answer MarkApplied did not run inside transaction runner")
	}
	if !goals.updateHintsSawTransaction {
		t.Fatal("goal UpdateHints did not run inside transaction runner")
	}
	if events.transactionalAppends != 2 {
		t.Fatalf("transactional event appends = %d, want 2", events.transactionalAppends)
	}
	if updatedGoal.ScopeHint == "" {
		t.Fatal("updated goal scope_hint is empty")
	}
}

func TestServiceApplyAnswerRejectsRepeatedApplication(t *testing.T) {
	service, goals, _, _, events := requestService(t)
	request := createRequest(t, service, goals, spine.GoalReadinessReasonMissingScopeHint)
	recorded, err := service.RecordAnswer(context.Background(), request.ID, answerSubmission(request))
	if err != nil {
		t.Fatalf("RecordAnswer() error = %v", err)
	}

	if _, _, err := service.ApplyAnswer(context.Background(), recorded.ID, applyRequest()); err != nil {
		t.Fatalf("first ApplyAnswer() error = %v", err)
	}
	_, _, err = service.ApplyAnswer(context.Background(), recorded.ID, applyRequest())
	if !errors.Is(err, clarification.ErrAlreadyApplied) {
		t.Fatalf("second ApplyAnswer() error = %v, want ErrAlreadyApplied", err)
	}
	if got := len(events.Events()); got != 5 {
		t.Fatalf("events length = %d, want 5", got)
	}
}

func TestServiceApplyAnswerReturnsGoalNotFoundWhenHintUpdateFindsNoRow(t *testing.T) {
	service, goals, _, _, events := requestService(t)
	request := createRequest(t, service, goals, spine.GoalReadinessReasonMissingScopeHint)
	recorded, err := service.RecordAnswer(context.Background(), request.ID, answerSubmission(request))
	if err != nil {
		t.Fatalf("RecordAnswer() error = %v", err)
	}
	goals.updateHintsMissing = true

	_, _, err = service.ApplyAnswer(context.Background(), recorded.ID, applyRequest())
	if !errors.Is(err, clarification.ErrGoalNotFound) {
		t.Fatalf("ApplyAnswer() error = %v, want ErrGoalNotFound", err)
	}
	if got := len(events.Events()); got != 3 {
		t.Fatalf("events length = %d, want pre-application events only", got)
	}
}

func TestServiceApplyAnswerRejectsUnsupportedIntentOwnerMapping(t *testing.T) {
	service, goals, _, _, events := requestService(t)
	request := createRequest(t, service, goals, spine.GoalReadinessReasonMissingIntentOwner)
	recorded, err := service.RecordAnswer(context.Background(), request.ID, answerSubmissionWithValues(request, map[spine.ClarificationMapsTo]string{
		spine.ClarificationMapsToGoalIntentOwner: "dev_2",
	}))
	if err != nil {
		t.Fatalf("RecordAnswer() error = %v", err)
	}

	_, _, err = service.ApplyAnswer(context.Background(), recorded.ID, applyRequest())
	if !errors.Is(err, clarification.ErrUnsupportedMapping) {
		t.Fatalf("ApplyAnswer() error = %v, want ErrUnsupportedMapping", err)
	}
	if got := len(events.Events()); got != 3 {
		t.Fatalf("events length = %d, want 3", got)
	}
}

func TestServiceApplyAnswerDoesNotChangeAnswerEvidence(t *testing.T) {
	service, goals, _, answers, _ := requestService(t)
	request := createRequest(t, service, goals, spine.GoalReadinessReasonMissingScopeHint)
	recorded, err := service.RecordAnswer(context.Background(), request.ID, answerSubmission(request))
	if err != nil {
		t.Fatalf("RecordAnswer() error = %v", err)
	}
	before, ok, err := answers.Get(context.Background(), recorded.ID)
	if err != nil {
		t.Fatalf("answers.Get() before error = %v", err)
	}
	if !ok {
		t.Fatal("stored answer not found before apply")
	}

	if _, _, err := service.ApplyAnswer(context.Background(), recorded.ID, applyRequest()); err != nil {
		t.Fatalf("ApplyAnswer() error = %v", err)
	}

	after, ok, err := answers.Get(context.Background(), recorded.ID)
	if err != nil {
		t.Fatalf("answers.Get() after error = %v", err)
	}
	if !ok {
		t.Fatal("stored answer not found after apply")
	}
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("answer after apply = %#v, want unchanged %#v", after, before)
	}
}

func TestServiceApplyAnswerDoesNotAppendReadinessOrContractEvents(t *testing.T) {
	service, goals, _, _, events := requestService(t)
	request := createRequest(t, service, goals, spine.GoalReadinessReasonMissingScopeHint)
	recorded, err := service.RecordAnswer(context.Background(), request.ID, answerSubmission(request))
	if err != nil {
		t.Fatalf("RecordAnswer() error = %v", err)
	}

	if _, _, err := service.ApplyAnswer(context.Background(), recorded.ID, applyRequest()); err != nil {
		t.Fatalf("ApplyAnswer() error = %v", err)
	}

	for _, event := range events.Events() {
		switch event.Type {
		case "goal.readiness_recheck_requested", "contract.seed_created", "contract.created", "work_item.created", "gate.decision_written", "proof.created":
			t.Fatalf("unexpected event type %q", event.Type)
		}
	}
}

func requestService(t *testing.T) (*clarification.Service, *fakeGoalStore, *fakeClarificationStore, *fakeClarificationAnswerStore, *fakeEventLog) {
	t.Helper()

	goals := newFakeGoalStore()
	clarifications := newFakeClarificationStore()
	answers := newFakeClarificationAnswerStore()
	events := newFakeEventLog()
	service := clarification.NewService(goals, clarifications, answers, events, fixedClock{now: testTime()}, &sequenceIDs{})
	return service, goals, clarifications, answers, events
}

type fakeGoalStore struct {
	goals                     map[spine.GoalID]spine.Goal
	updateHintsSawTransaction bool
	updateHintsMissing        bool
}

func newFakeGoalStore() *fakeGoalStore {
	return &fakeGoalStore{goals: map[spine.GoalID]spine.Goal{}}
}

func (s *fakeGoalStore) Create(_ context.Context, goal spine.Goal) error {
	s.goals[goal.ID] = cloneGoal(goal)
	return nil
}

func (s *fakeGoalStore) Get(_ context.Context, id spine.GoalID) (spine.Goal, bool, error) {
	goal, ok := s.goals[id]
	return cloneGoal(goal), ok, nil
}

func (s *fakeGoalStore) UpdateHints(ctx context.Context, id spine.GoalID, update spine.GoalHintUpdate) (spine.Goal, bool, error) {
	s.updateHintsSawTransaction = s.updateHintsSawTransaction || sawTransaction(ctx)
	if s.updateHintsMissing {
		return spine.Goal{}, false, nil
	}
	goal, ok := s.goals[id]
	if !ok {
		return spine.Goal{}, false, nil
	}
	if update.Summary != nil {
		goal.Summary = *update.Summary
	}
	if update.ScopeHint != nil {
		goal.ScopeHint = *update.ScopeHint
	}
	if update.AcceptanceHint != nil {
		goal.AcceptanceHint = *update.AcceptanceHint
	}
	if update.IntentOwner != nil {
		goal.IntentOwner = *update.IntentOwner
	}
	s.goals[id] = cloneGoal(goal)
	return cloneGoal(goal), true, nil
}

func cloneGoal(goal spine.Goal) spine.Goal {
	goal.SourceRefs = append([]spine.SourceRef(nil), goal.SourceRefs...)
	goal.LastReadinessReasonCodes = append([]spine.GoalReadinessReasonCode(nil), goal.LastReadinessReasonCodes...)
	return goal
}

type fakeClarificationStore struct {
	requests                  map[spine.ClarificationRequestID]spine.ClarificationRequest
	openByGoal                map[spine.GoalID]spine.ClarificationRequestID
	createSawTransaction      bool
	updateStateSawTransaction bool
	updateStateMissing        bool
}

func newFakeClarificationStore() *fakeClarificationStore {
	return &fakeClarificationStore{
		requests:   map[spine.ClarificationRequestID]spine.ClarificationRequest{},
		openByGoal: map[spine.GoalID]spine.ClarificationRequestID{},
	}
}

func (s *fakeClarificationStore) Create(ctx context.Context, request spine.ClarificationRequest) error {
	s.createSawTransaction = s.createSawTransaction || sawTransaction(ctx)
	s.requests[request.ID] = cloneClarificationRequest(request)
	if request.State == spine.ClarificationRequestStateOpen {
		s.openByGoal[request.GoalID] = request.ID
	}
	return nil
}

func (s *fakeClarificationStore) Get(_ context.Context, id spine.ClarificationRequestID) (spine.ClarificationRequest, bool, error) {
	request, ok := s.requests[id]
	return cloneClarificationRequest(request), ok, nil
}

func (s *fakeClarificationStore) GetOpenByGoalID(_ context.Context, id spine.GoalID) (spine.ClarificationRequest, bool, error) {
	requestID, ok := s.openByGoal[id]
	if !ok {
		return spine.ClarificationRequest{}, false, nil
	}
	request, ok := s.requests[requestID]
	return cloneClarificationRequest(request), ok, nil
}

func (s *fakeClarificationStore) UpdateState(ctx context.Context, id spine.ClarificationRequestID, state spine.ClarificationRequestState) (spine.ClarificationRequest, bool, error) {
	s.updateStateSawTransaction = s.updateStateSawTransaction || sawTransaction(ctx)
	if s.updateStateMissing {
		return spine.ClarificationRequest{}, false, nil
	}
	request, ok := s.requests[id]
	if !ok {
		return spine.ClarificationRequest{}, false, nil
	}
	if request.State == spine.ClarificationRequestStateOpen && state != spine.ClarificationRequestStateOpen {
		delete(s.openByGoal, request.GoalID)
	}
	request.State = state
	s.requests[id] = cloneClarificationRequest(request)
	if state == spine.ClarificationRequestStateOpen {
		s.openByGoal[request.GoalID] = id
	}
	return cloneClarificationRequest(request), true, nil
}

func cloneClarificationRequest(request spine.ClarificationRequest) spine.ClarificationRequest {
	request.ReasonCodes = append([]spine.GoalReadinessReasonCode(nil), request.ReasonCodes...)
	request.Questions = append([]spine.ClarificationQuestion(nil), request.Questions...)
	if request.Target.ActorRef != nil {
		actor := *request.Target.ActorRef
		request.Target.ActorRef = &actor
	}
	return request
}

type fakeClarificationAnswerStore struct {
	answers                   map[spine.ClarificationAnswerID]spine.ClarificationAnswer
	byRequest                 map[spine.ClarificationRequestID]spine.ClarificationAnswerID
	applied                   map[spine.ClarificationAnswerID]bool
	createSawTransaction      bool
	markAppliedSawTransaction bool
}

func newFakeClarificationAnswerStore() *fakeClarificationAnswerStore {
	return &fakeClarificationAnswerStore{
		answers:   map[spine.ClarificationAnswerID]spine.ClarificationAnswer{},
		byRequest: map[spine.ClarificationRequestID]spine.ClarificationAnswerID{},
		applied:   map[spine.ClarificationAnswerID]bool{},
	}
}

func (s *fakeClarificationAnswerStore) Create(ctx context.Context, answer spine.ClarificationAnswer) error {
	s.createSawTransaction = s.createSawTransaction || sawTransaction(ctx)
	s.answers[answer.ID] = cloneClarificationAnswer(answer)
	s.byRequest[answer.RequestID] = answer.ID
	return nil
}

func (s *fakeClarificationAnswerStore) Get(_ context.Context, id spine.ClarificationAnswerID) (spine.ClarificationAnswer, bool, error) {
	answer, ok := s.answers[id]
	return cloneClarificationAnswer(answer), ok, nil
}

func (s *fakeClarificationAnswerStore) GetByRequestID(_ context.Context, id spine.ClarificationRequestID) (spine.ClarificationAnswer, bool, error) {
	answerID, ok := s.byRequest[id]
	if !ok {
		return spine.ClarificationAnswer{}, false, nil
	}
	answer, ok := s.answers[answerID]
	return cloneClarificationAnswer(answer), ok, nil
}

func (s *fakeClarificationAnswerStore) MarkApplied(ctx context.Context, id spine.ClarificationAnswerID, _ spine.ActorRef, _ time.Time) (bool, error) {
	s.markAppliedSawTransaction = s.markAppliedSawTransaction || sawTransaction(ctx)
	if s.applied[id] {
		return false, nil
	}
	s.applied[id] = true
	return true, nil
}

func cloneClarificationAnswer(answer spine.ClarificationAnswer) spine.ClarificationAnswer {
	answer.Answers = append([]spine.ClarificationAnswerItem(nil), answer.Answers...)
	return answer
}

type fakeEventLog struct {
	events               []spine.Event
	transactionalAppends int
}

func newFakeEventLog() *fakeEventLog {
	return &fakeEventLog{}
}

func (l *fakeEventLog) Append(ctx context.Context, event spine.Event) error {
	if sawTransaction(ctx) {
		l.transactionalAppends++
	}
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

func validGoal(reasons ...spine.GoalReadinessReasonCode) spine.Goal {
	return spine.Goal{
		ID:            "goal-1",
		IntakeID:      "intake-1",
		RepoBindingID: "018f0000-0000-7000-8000-000000000004",
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
	answer        int
	event         int
}

type txContextKey struct{}

type fakeTransactionRunner struct {
	calls int
	err   error
}

func (r *fakeTransactionRunner) RunReadCommitted(ctx context.Context, fn func(context.Context) error) error {
	r.calls++
	if r.err != nil {
		return r.err
	}
	return fn(context.WithValue(ctx, txContextKey{}, true))
}

func sawTransaction(ctx context.Context) bool {
	inTx, _ := ctx.Value(txContextKey{}).(bool)
	return inTx
}

func (g *sequenceIDs) NewClarificationRequestID() (spine.ClarificationRequestID, error) {
	g.clarification++
	return spine.ClarificationRequestID(fmt.Sprintf("clarification-%d", g.clarification)), nil
}

func (g *sequenceIDs) NewClarificationQuestionID() (spine.ClarificationQuestionID, error) {
	g.question++
	return spine.ClarificationQuestionID(fmt.Sprintf("question-%d", g.question)), nil
}

func (g *sequenceIDs) NewClarificationAnswerID() (spine.ClarificationAnswerID, error) {
	g.answer++
	return spine.ClarificationAnswerID(fmt.Sprintf("answer-%d", g.answer)), nil
}

func (g *sequenceIDs) NewEventID() (spine.EventID, error) {
	g.event++
	return spine.EventID(fmt.Sprintf("event-%d", g.event)), nil
}

func testTime() time.Time {
	return time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
}

func createRequest(t *testing.T, service *clarification.Service, goals *fakeGoalStore, reasons ...spine.GoalReadinessReasonCode) spine.ClarificationRequest {
	t.Helper()

	createdGoal := validGoal(reasons...)
	if err := goals.Create(context.Background(), createdGoal); err != nil {
		t.Fatalf("Create goal: %v", err)
	}
	request, err := service.CreateRequest(context.Background(), createdGoal.ID)
	if err != nil {
		t.Fatalf("CreateRequest() error = %v", err)
	}
	return request
}

func answerSubmission(request spine.ClarificationRequest) spine.ClarificationAnswerSubmission {
	return answerSubmissionWithValues(request, nil)
}

func answerSubmissionWithValues(request spine.ClarificationRequest, values map[spine.ClarificationMapsTo]string) spine.ClarificationAnswerSubmission {
	answers := make([]spine.ClarificationAnswerItem, 0, len(request.Questions))
	for i, question := range request.Questions {
		value := fmt.Sprintf("Answer %d", i+1)
		if values != nil {
			if mapped, ok := values[question.MapsTo]; ok {
				value = mapped
			}
		}
		answers = append(answers, spine.ClarificationAnswerItem{
			QuestionID: question.ID,
			Value:      value,
		})
	}
	return spine.ClarificationAnswerSubmission{
		Answers:     answers,
		SubmittedBy: spine.ActorRef{Kind: "user", ID: "dev_1"},
	}
}

func applyRequest() spine.ClarificationAnswerApplicationRequest {
	return spine.ClarificationAnswerApplicationRequest{
		AppliedBy: spine.ActorRef{Kind: "user", ID: "dev_1"},
	}
}
