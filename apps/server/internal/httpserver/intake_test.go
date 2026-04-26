package httpserver_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/clarification"
	"github.com/heurema/goalrail/apps/server/internal/eventlog"
	"github.com/heurema/goalrail/apps/server/internal/goal"
	"github.com/heurema/goalrail/apps/server/internal/httpserver"
	"github.com/heurema/goalrail/apps/server/internal/intake"
	"github.com/heurema/goalrail/apps/server/internal/spine"
	"github.com/heurema/goalrail/apps/server/internal/store"
)

const validIntakeJSON = `{
  "repo_binding_id": "repo_demo_1",
  "source": {
    "kind": "codex_skill",
    "external_id": "local-session-1"
  },
  "title": "Refactor CSV export filters",
  "body": "Current code duplicates filter logic. Preserve current behavior.",
  "request_author": {
    "kind": "user",
    "id": "dev_1",
    "display_name": "Developer"
  }
}`

type testServerDeps struct {
	router         http.Handler
	intakes        *store.IntakeStore
	goals          *store.GoalStore
	clarifications *store.ClarificationStore
	answers        *store.ClarificationAnswerStore
	events         *eventlog.EventLog
	idFactory      *sequenceIDs
}

func TestPostIntakeReturnsAccepted(t *testing.T) {
	server := testServer(t)

	response := doJSON(t, server.router, http.MethodPost, "/v1/intake", validIntakeJSON)
	if response.code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", response.code, http.StatusAccepted)
	}

	var body map[string]json.RawMessage
	decodeJSON(t, response.body, &body)

	var canonicalContractCreated bool
	decodeRawJSON(t, body["canonical_contract_created"], &canonicalContractCreated)
	if canonicalContractCreated {
		t.Fatal("canonical_contract_created = true, want false")
	}

	for _, forbiddenField := range []string{"goal_id", "contract_id", "work_item_id"} {
		if _, ok := body[forbiddenField]; ok {
			t.Fatalf("response includes forbidden field %q", forbiddenField)
		}
	}

	var state string
	decodeRawJSON(t, body["state"], &state)
	if state != "received" {
		t.Fatalf("state = %q, want %q", state, "received")
	}
}

func TestGetIntakeReturnsStoredRecord(t *testing.T) {
	server := testServer(t)

	postResponse := doJSON(t, server.router, http.MethodPost, "/v1/intake", validIntakeJSON)
	if postResponse.code != http.StatusAccepted {
		t.Fatalf("POST status = %d, want %d", postResponse.code, http.StatusAccepted)
	}

	var accepted struct {
		IntakeID string `json:"intake_id"`
	}
	decodeJSON(t, postResponse.body, &accepted)

	getResponse := doJSON(t, server.router, http.MethodGet, "/v1/intake/"+accepted.IntakeID, "")
	if getResponse.code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d", getResponse.code, http.StatusOK)
	}

	var record spine.IntakeRecord
	decodeJSON(t, getResponse.body, &record)

	if record.ID != spine.IntakeID(accepted.IntakeID) {
		t.Fatalf("record ID = %q, want %q", record.ID, accepted.IntakeID)
	}
	if record.State != spine.IntakeStateReceived {
		t.Fatalf("state = %q, want %q", record.State, spine.IntakeStateReceived)
	}
	if record.CanonicalContractCreated {
		t.Fatal("CanonicalContractCreated = true, want false")
	}
	if record.RepoBindingID != "repo_demo_1" {
		t.Fatalf("RepoBindingID = %q, want %q", record.RepoBindingID, "repo_demo_1")
	}
	if !reflect.DeepEqual(record.IntentOwner, record.RequestAuthor) {
		t.Fatalf("IntentOwner = %#v, want RequestAuthor %#v", record.IntentOwner, record.RequestAuthor)
	}
}

func TestGetUnknownIntakeReturnsNotFound(t *testing.T) {
	server := testServer(t)

	response := doJSON(t, server.router, http.MethodGet, "/v1/intake/missing", "")
	if response.code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.code, http.StatusNotFound)
	}

	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeJSON(t, response.body, &body)
	if body.Error.Code != "not_found" {
		t.Fatalf("error code = %q, want %q", body.Error.Code, "not_found")
	}
}

func TestPostIntakeValidation(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "missing repo_binding_id",
			body: `{"source":{"kind":"codex_skill"},"title":"Title","request_author":{"kind":"user","id":"dev_1"}}`,
		},
		{
			name: "missing source kind",
			body: `{"repo_binding_id":"repo_demo_1","source":{},"title":"Title","request_author":{"kind":"user","id":"dev_1"}}`,
		},
		{
			name: "missing title and body",
			body: `{"repo_binding_id":"repo_demo_1","source":{"kind":"codex_skill"},"request_author":{"kind":"user","id":"dev_1"}}`,
		},
		{
			name: "missing request_author kind",
			body: `{"repo_binding_id":"repo_demo_1","source":{"kind":"codex_skill"},"title":"Title","request_author":{"id":"dev_1"}}`,
		},
		{
			name: "missing request_author id",
			body: `{"repo_binding_id":"repo_demo_1","source":{"kind":"codex_skill"},"title":"Title","request_author":{"kind":"user"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := testServer(t)

			response := doJSON(t, server.router, http.MethodPost, "/v1/intake", tt.body)
			if response.code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", response.code, http.StatusBadRequest)
			}

			var body struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			decodeJSON(t, response.body, &body)
			if body.Error.Code != "validation_failed" {
				t.Fatalf("error code = %q, want %q", body.Error.Code, "validation_failed")
			}
		})
	}
}

func TestPostIntakeRejectsUnknownJSONField(t *testing.T) {
	server := testServer(t)
	body := `{"repo_binding_id":"repo_demo_1","source":{"kind":"codex_skill"},"title":"Title","request_author":{"kind":"user","id":"dev_1"},"unexpected":true}`

	response := doJSON(t, server.router, http.MethodPost, "/v1/intake", body)
	if response.code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.code, http.StatusBadRequest)
	}

	var responseBody struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeJSON(t, response.body, &responseBody)
	if responseBody.Error.Code != "invalid_json" {
		t.Fatalf("error code = %q, want %q", responseBody.Error.Code, "invalid_json")
	}
}

func TestPostIntakeDefaultsIntentOwnerToRequestAuthor(t *testing.T) {
	server := testServer(t)

	response := doJSON(t, server.router, http.MethodPost, "/v1/intake", validIntakeJSON)
	if response.code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", response.code, http.StatusAccepted)
	}

	record, ok, err := server.intakes.Get(context.Background(), "intake-1")
	if err != nil {
		t.Fatalf("store.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("stored intake not found")
	}
	if !reflect.DeepEqual(record.IntentOwner, record.RequestAuthor) {
		t.Fatalf("IntentOwner = %#v, want RequestAuthor %#v", record.IntentOwner, record.RequestAuthor)
	}
}

func TestPostPromoteIntakeReturnsCreatedGoal(t *testing.T) {
	server := testServer(t)
	intakeID := createIntake(t, server, validIntakeJSON)

	response := doJSON(t, server.router, http.MethodPost, "/v1/intake/"+intakeID+"/promote", "")
	if response.code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.code, http.StatusCreated)
	}

	var body map[string]json.RawMessage
	decodeJSON(t, response.body, &body)
	for _, forbiddenField := range []string{"contract_id", "work_item_id", "proof_id"} {
		if _, ok := body[forbiddenField]; ok {
			t.Fatalf("response includes forbidden field %q", forbiddenField)
		}
	}

	var created spine.Goal
	decodeJSON(t, response.body, &created)
	if created.State != spine.GoalStateCreated {
		t.Fatalf("state = %q, want %q", created.State, spine.GoalStateCreated)
	}
	if created.IntakeID != spine.IntakeID(intakeID) {
		t.Fatalf("intake_id = %q, want %q", created.IntakeID, intakeID)
	}
	if created.Summary != "Current code duplicates filter logic. Preserve current behavior." {
		t.Fatalf("summary = %q, want intake body", created.Summary)
	}
	if !reflect.DeepEqual(created.RequestAuthor, created.IntentOwner) {
		t.Fatalf("IntentOwner = %#v, want RequestAuthor %#v", created.IntentOwner, created.RequestAuthor)
	}
	if len(created.SourceRefs) != 1 {
		t.Fatalf("source_refs length = %d, want 1", len(created.SourceRefs))
	}
	if created.SourceRefs[0] != (spine.SourceRef{Kind: "intake", ID: intakeID}) {
		t.Fatalf("source_refs[0] = %#v, want intake ref", created.SourceRefs[0])
	}

	stored, ok, err := server.goals.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("goals.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("promoted goal not stored")
	}
	if stored.ID != created.ID {
		t.Fatalf("stored goal id = %q, want %q", stored.ID, created.ID)
	}
}

func TestPostPromoteUnknownIntakeReturnsNotFound(t *testing.T) {
	server := testServer(t)

	response := doJSON(t, server.router, http.MethodPost, "/v1/intake/missing/promote", "")
	if response.code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.code, http.StatusNotFound)
	}

	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeJSON(t, response.body, &body)
	if body.Error.Code != "not_found" {
		t.Fatalf("error code = %q, want %q", body.Error.Code, "not_found")
	}
}

func TestPostPromoteIntakeTwiceReturnsConflict(t *testing.T) {
	server := testServer(t)
	intakeID := createIntake(t, server, validIntakeJSON)

	first := doJSON(t, server.router, http.MethodPost, "/v1/intake/"+intakeID+"/promote", "")
	if first.code != http.StatusCreated {
		t.Fatalf("first status = %d, want %d", first.code, http.StatusCreated)
	}

	second := doJSON(t, server.router, http.MethodPost, "/v1/intake/"+intakeID+"/promote", "")
	if second.code != http.StatusConflict {
		t.Fatalf("second status = %d, want %d", second.code, http.StatusConflict)
	}

	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeJSON(t, second.body, &body)
	if body.Error.Code != "already_promoted" {
		t.Fatalf("error code = %q, want %q", body.Error.Code, "already_promoted")
	}

	events := server.events.Events()
	if len(events) != 3 {
		t.Fatalf("events length = %d, want 3", len(events))
	}
}

func TestPostPromoteIntakeUsesTitleAsSummaryWhenBodyIsEmpty(t *testing.T) {
	server := testServer(t)
	intakeID := createIntake(t, server, `{
  "repo_binding_id": "repo_demo_1",
  "source": {"kind": "codex_skill"},
  "title": "Refactor CSV export filters",
  "request_author": {"kind": "user", "id": "dev_1"}
}`)

	response := doJSON(t, server.router, http.MethodPost, "/v1/intake/"+intakeID+"/promote", "")
	if response.code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.code, http.StatusCreated)
	}

	var created spine.Goal
	decodeJSON(t, response.body, &created)
	if created.Summary != created.Title {
		t.Fatalf("summary = %q, want title %q", created.Summary, created.Title)
	}
}

func TestPostGoalReadinessReturnsNeedsClarificationForPromotedGoal(t *testing.T) {
	server := testServer(t)
	intakeID := createIntake(t, server, validIntakeJSON)
	created := promoteIntake(t, server, intakeID)

	response := doJSON(t, server.router, http.MethodPost, "/v1/goals/"+string(created.ID)+"/readiness", "")
	if response.code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.code, http.StatusOK)
	}
	for _, forbiddenField := range []string{"\"contract_id\"", "\"work_item_id\"", "\"proof_id\""} {
		if strings.Contains(response.body, forbiddenField) {
			t.Fatalf("response includes forbidden field %s", forbiddenField)
		}
	}

	var body struct {
		Readiness spine.GoalReadinessResult `json:"readiness"`
		Goal      spine.Goal                `json:"goal"`
	}
	decodeJSON(t, response.body, &body)
	if body.Readiness.GoalID != created.ID {
		t.Fatalf("readiness goal id = %q, want %q", body.Readiness.GoalID, created.ID)
	}
	if body.Readiness.State != spine.GoalStateNeedsClarification {
		t.Fatalf("readiness state = %q, want %q", body.Readiness.State, spine.GoalStateNeedsClarification)
	}
	if body.Readiness.Ready {
		t.Fatal("readiness ready = true, want false")
	}
	if !hasReadinessReason(body.Readiness.ReasonCodes, spine.GoalReadinessReasonMissingScopeHint) {
		t.Fatalf("reason codes = %#v, want %q", body.Readiness.ReasonCodes, spine.GoalReadinessReasonMissingScopeHint)
	}
	if !hasReadinessReason(body.Readiness.ReasonCodes, spine.GoalReadinessReasonMissingAcceptanceHint) {
		t.Fatalf("reason codes = %#v, want %q", body.Readiness.ReasonCodes, spine.GoalReadinessReasonMissingAcceptanceHint)
	}
	if body.Goal.State != spine.GoalStateNeedsClarification {
		t.Fatalf("goal state = %q, want %q", body.Goal.State, spine.GoalStateNeedsClarification)
	}

	stored, ok, err := server.goals.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("goals.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("stored goal not found")
	}
	if stored.State != spine.GoalStateNeedsClarification {
		t.Fatalf("stored goal state = %q, want %q", stored.State, spine.GoalStateNeedsClarification)
	}
}

func TestPostGoalReadinessCanRepeatForPromotedGoal(t *testing.T) {
	server := testServer(t)
	intakeID := createIntake(t, server, validIntakeJSON)
	created := promoteIntake(t, server, intakeID)

	first := doJSON(t, server.router, http.MethodPost, "/v1/goals/"+string(created.ID)+"/readiness", "")
	if first.code != http.StatusOK {
		t.Fatalf("first status = %d, want %d", first.code, http.StatusOK)
	}
	second := doJSON(t, server.router, http.MethodPost, "/v1/goals/"+string(created.ID)+"/readiness", "")
	if second.code != http.StatusOK {
		t.Fatalf("second status = %d, want %d", second.code, http.StatusOK)
	}

	var body struct {
		Readiness spine.GoalReadinessResult `json:"readiness"`
		Goal      spine.Goal                `json:"goal"`
	}
	decodeJSON(t, second.body, &body)
	if body.Readiness.State != spine.GoalStateNeedsClarification {
		t.Fatalf("second readiness state = %q, want %q", body.Readiness.State, spine.GoalStateNeedsClarification)
	}
	if body.Goal.State != spine.GoalStateNeedsClarification {
		t.Fatalf("second goal state = %q, want %q", body.Goal.State, spine.GoalStateNeedsClarification)
	}

	events := server.events.Events()
	if len(events) != 7 {
		t.Fatalf("events length = %d, want 7", len(events))
	}
}

func TestPostGoalReadinessUnknownGoalReturnsNotFound(t *testing.T) {
	server := testServer(t)

	response := doJSON(t, server.router, http.MethodPost, "/v1/goals/missing/readiness", "")
	if response.code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.code, http.StatusNotFound)
	}

	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeJSON(t, response.body, &body)
	if body.Error.Code != "not_found" {
		t.Fatalf("error code = %q, want %q", body.Error.Code, "not_found")
	}
}

func TestPostGoalClarificationRequestsReturnsOpenRequest(t *testing.T) {
	server := testServer(t)
	created := createClarificationReadyGoal(t, server)

	response := doJSON(t, server.router, http.MethodPost, "/v1/goals/"+string(created.ID)+"/clarification-requests", "")
	if response.code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
	}
	for _, forbiddenField := range []string{"\"contract_id\"", "\"work_item_id\"", "\"proof_id\""} {
		if strings.Contains(response.body, forbiddenField) {
			t.Fatalf("response includes forbidden field %s", forbiddenField)
		}
	}

	var request spine.ClarificationRequest
	decodeJSON(t, response.body, &request)
	if request.State != spine.ClarificationRequestStateOpen {
		t.Fatalf("state = %q, want %q", request.State, spine.ClarificationRequestStateOpen)
	}
	if request.GoalID != created.ID {
		t.Fatalf("goal_id = %q, want %q", request.GoalID, created.ID)
	}
	if !hasClarificationQuestion(request.Questions, spine.ClarificationMapsToGoalScopeHint) {
		t.Fatalf("questions = %#v, want scope_hint question", request.Questions)
	}
	if !hasClarificationQuestion(request.Questions, spine.ClarificationMapsToGoalAcceptanceHint) {
		t.Fatalf("questions = %#v, want acceptance_hint question", request.Questions)
	}
	if request.Target.Role != spine.ClarificationTargetRoleIntentOwner {
		t.Fatalf("target role = %q, want %q", request.Target.Role, spine.ClarificationTargetRoleIntentOwner)
	}

	stored, ok, err := server.clarifications.Get(context.Background(), request.ID)
	if err != nil {
		t.Fatalf("clarifications.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("stored clarification request not found")
	}
	if stored.ID != request.ID {
		t.Fatalf("stored request id = %q, want %q", stored.ID, request.ID)
	}
}

func TestPostGoalClarificationRequestsUnknownGoalReturnsNotFound(t *testing.T) {
	server := testServer(t)

	response := doJSON(t, server.router, http.MethodPost, "/v1/goals/missing/clarification-requests", "")
	if response.code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.code, http.StatusNotFound)
	}

	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeJSON(t, response.body, &body)
	if body.Error.Code != "not_found" {
		t.Fatalf("error code = %q, want %q", body.Error.Code, "not_found")
	}
}

func TestPostGoalClarificationRequestsRejectsGoalNotNeedsClarification(t *testing.T) {
	server := testServer(t)
	intakeID := createIntake(t, server, validIntakeJSON)
	created := promoteIntake(t, server, intakeID)

	response := doJSON(t, server.router, http.MethodPost, "/v1/goals/"+string(created.ID)+"/clarification-requests", "")
	if response.code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", response.code, http.StatusConflict)
	}

	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeJSON(t, response.body, &body)
	if body.Error.Code != "invalid_state" {
		t.Fatalf("error code = %q, want %q", body.Error.Code, "invalid_state")
	}
}

func TestPostGoalClarificationRequestsRejectsMissingReadinessReasons(t *testing.T) {
	server := testServer(t)
	created := spine.Goal{
		ID:            "goal-without-reasons",
		IntakeID:      "intake-without-reasons",
		RepoBindingID: "repo_demo_1",
		Title:         "Refactor CSV export filters",
		Summary:       "Current code duplicates filter logic. Preserve current behavior.",
		SourceRefs: []spine.SourceRef{
			{Kind: "intake", ID: "intake-without-reasons"},
		},
		RequestAuthor: spine.ActorRef{Kind: "user", ID: "dev_1"},
		IntentOwner:   spine.ActorRef{Kind: "user", ID: "dev_1"},
		State:         spine.GoalStateNeedsClarification,
		CreatedAt:     testTime(),
	}
	if err := server.goals.Create(context.Background(), created); err != nil {
		t.Fatalf("Create goal: %v", err)
	}

	response := doJSON(t, server.router, http.MethodPost, "/v1/goals/"+string(created.ID)+"/clarification-requests", "")
	if response.code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", response.code, http.StatusConflict)
	}

	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeJSON(t, response.body, &body)
	if body.Error.Code != "missing_readiness_reasons" {
		t.Fatalf("error code = %q, want %q", body.Error.Code, "missing_readiness_reasons")
	}
}

func TestPostGoalClarificationRequestsRejectsDuplicateOpenRequest(t *testing.T) {
	server := testServer(t)
	created := createClarificationReadyGoal(t, server)

	first := doJSON(t, server.router, http.MethodPost, "/v1/goals/"+string(created.ID)+"/clarification-requests", "")
	if first.code != http.StatusCreated {
		t.Fatalf("first status = %d, want %d: %s", first.code, http.StatusCreated, first.body)
	}
	second := doJSON(t, server.router, http.MethodPost, "/v1/goals/"+string(created.ID)+"/clarification-requests", "")
	if second.code != http.StatusConflict {
		t.Fatalf("second status = %d, want %d", second.code, http.StatusConflict)
	}

	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeJSON(t, second.body, &body)
	if body.Error.Code != "already_open" {
		t.Fatalf("error code = %q, want %q", body.Error.Code, "already_open")
	}
}

func TestPostClarificationRequestAnswersReturnsRecordedAnswer(t *testing.T) {
	server := testServer(t)
	request := createClarificationRequest(t, server)
	body := answerSubmissionJSON(request)

	response := doJSON(t, server.router, http.MethodPost, "/v1/clarification-requests/"+string(request.ID)+"/answers", body)
	if response.code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
	}
	for _, forbiddenField := range []string{"\"contract_id\"", "\"work_item_id\"", "\"proof_id\""} {
		if strings.Contains(response.body, forbiddenField) {
			t.Fatalf("response includes forbidden field %s", forbiddenField)
		}
	}

	var answer spine.ClarificationAnswer
	decodeJSON(t, response.body, &answer)
	if answer.State != spine.ClarificationAnswerStateRecorded {
		t.Fatalf("state = %q, want %q", answer.State, spine.ClarificationAnswerStateRecorded)
	}
	if answer.RequestID != request.ID {
		t.Fatalf("request_id = %q, want %q", answer.RequestID, request.ID)
	}
	if answer.GoalID != request.GoalID {
		t.Fatalf("goal_id = %q, want %q", answer.GoalID, request.GoalID)
	}
	if len(answer.Answers) != len(request.Questions) {
		t.Fatalf("answers length = %d, want %d", len(answer.Answers), len(request.Questions))
	}

	storedAnswer, ok, err := server.answers.Get(context.Background(), answer.ID)
	if err != nil {
		t.Fatalf("answers.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("stored clarification answer not found")
	}
	if storedAnswer.ID != answer.ID {
		t.Fatalf("stored answer id = %q, want %q", storedAnswer.ID, answer.ID)
	}

	storedRequest, ok, err := server.clarifications.Get(context.Background(), request.ID)
	if err != nil {
		t.Fatalf("clarifications.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("stored clarification request not found")
	}
	if storedRequest.State != spine.ClarificationRequestStateAnswered {
		t.Fatalf("request state = %q, want %q", storedRequest.State, spine.ClarificationRequestStateAnswered)
	}
}

func TestPostClarificationRequestAnswersUnknownRequestReturnsNotFound(t *testing.T) {
	server := testServer(t)

	response := doJSON(t, server.router, http.MethodPost, "/v1/clarification-requests/missing/answers", `{
  "submitted_by": {"kind": "user", "id": "dev_1"},
  "answers": [{"question_id": "question-1", "value": "Scope"}]
}`)
	if response.code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.code, http.StatusNotFound)
	}

	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeJSON(t, response.body, &body)
	if body.Error.Code != "not_found" {
		t.Fatalf("error code = %q, want %q", body.Error.Code, "not_found")
	}
}

func TestPostClarificationRequestAnswersRejectsAlreadyAnsweredRequest(t *testing.T) {
	server := testServer(t)
	request := createClarificationRequest(t, server)
	body := answerSubmissionJSON(request)

	first := doJSON(t, server.router, http.MethodPost, "/v1/clarification-requests/"+string(request.ID)+"/answers", body)
	if first.code != http.StatusCreated {
		t.Fatalf("first status = %d, want %d: %s", first.code, http.StatusCreated, first.body)
	}
	second := doJSON(t, server.router, http.MethodPost, "/v1/clarification-requests/"+string(request.ID)+"/answers", body)
	if second.code != http.StatusConflict {
		t.Fatalf("second status = %d, want %d", second.code, http.StatusConflict)
	}

	var responseBody struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeJSON(t, second.body, &responseBody)
	if responseBody.Error.Code != "already_answered" {
		t.Fatalf("error code = %q, want %q", responseBody.Error.Code, "already_answered")
	}
}

func TestPostClarificationRequestAnswersValidation(t *testing.T) {
	tests := []struct {
		name string
		body func(spine.ClarificationRequest) string
	}{
		{
			name: "missing submitted_by",
			body: func(request spine.ClarificationRequest) string {
				return fmt.Sprintf(`{"answers":[{"question_id":%q,"value":"Scope"}]}`, request.Questions[0].ID)
			},
		},
		{
			name: "missing submitted_by kind",
			body: func(request spine.ClarificationRequest) string {
				return fmt.Sprintf(`{"submitted_by":{"id":"dev_1"},"answers":[{"question_id":%q,"value":"Scope"}]}`, request.Questions[0].ID)
			},
		},
		{
			name: "missing submitted_by id",
			body: func(request spine.ClarificationRequest) string {
				return fmt.Sprintf(`{"submitted_by":{"kind":"user"},"answers":[{"question_id":%q,"value":"Scope"}]}`, request.Questions[0].ID)
			},
		},
		{
			name: "missing answers",
			body: func(spine.ClarificationRequest) string {
				return `{"submitted_by":{"kind":"user","id":"dev_1"}}`
			},
		},
		{
			name: "unknown question_id",
			body: func(spine.ClarificationRequest) string {
				return `{"submitted_by":{"kind":"user","id":"dev_1"},"answers":[{"question_id":"unknown","value":"Scope"}]}`
			},
		},
		{
			name: "duplicate question_id",
			body: func(request spine.ClarificationRequest) string {
				return fmt.Sprintf(`{"submitted_by":{"kind":"user","id":"dev_1"},"answers":[{"question_id":%q,"value":"Scope"},{"question_id":%q,"value":"Duplicate"}]}`, request.Questions[0].ID, request.Questions[0].ID)
			},
		},
		{
			name: "missing answer for one question",
			body: func(request spine.ClarificationRequest) string {
				return fmt.Sprintf(`{"submitted_by":{"kind":"user","id":"dev_1"},"answers":[{"question_id":%q,"value":"Scope"}]}`, request.Questions[0].ID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := testServer(t)
			request := createClarificationRequest(t, server)

			response := doJSON(t, server.router, http.MethodPost, "/v1/clarification-requests/"+string(request.ID)+"/answers", tt.body(request))
			if response.code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d: %s", response.code, http.StatusBadRequest, response.body)
			}

			var body struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			decodeJSON(t, response.body, &body)
			if body.Error.Code != "validation_failed" {
				t.Fatalf("error code = %q, want %q", body.Error.Code, "validation_failed")
			}
		})
	}
}

func TestPostClarificationAnswersApplyReturnsUpdatedGoal(t *testing.T) {
	server := testServer(t)
	answer := createClarificationAnswerForReasons(t, server, map[spine.ClarificationMapsTo]string{
		spine.ClarificationMapsToGoalScopeHint:      "Updated scope hint",
		spine.ClarificationMapsToGoalAcceptanceHint: "Updated acceptance hint",
	}, spine.GoalReadinessReasonMissingScopeHint, spine.GoalReadinessReasonMissingAcceptanceHint)

	response := doJSON(t, server.router, http.MethodPost, "/v1/clarification-answers/"+string(answer.ID)+"/apply", applyRequestJSON())
	if response.code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusOK, response.body)
	}
	for _, forbiddenField := range []string{"\"contract_id\"", "\"work_item_id\"", "\"proof_id\""} {
		if strings.Contains(response.body, forbiddenField) {
			t.Fatalf("response includes forbidden field %s", forbiddenField)
		}
	}

	var body struct {
		Application spine.ClarificationAnswerApplicationResult `json:"application"`
		Goal        spine.Goal                                 `json:"goal"`
	}
	decodeJSON(t, response.body, &body)
	if body.Application.AnswerID != answer.ID {
		t.Fatalf("application answer_id = %q, want %q", body.Application.AnswerID, answer.ID)
	}
	if body.Goal.ScopeHint != "Updated scope hint" {
		t.Fatalf("scope_hint = %q, want updated scope hint", body.Goal.ScopeHint)
	}
	if body.Goal.AcceptanceHint != "Updated acceptance hint" {
		t.Fatalf("acceptance_hint = %q, want updated acceptance hint", body.Goal.AcceptanceHint)
	}

	stored, ok, err := server.goals.Get(context.Background(), body.Goal.ID)
	if err != nil {
		t.Fatalf("goals.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("stored goal not found")
	}
	if stored.ScopeHint != "Updated scope hint" {
		t.Fatalf("stored scope_hint = %q, want updated scope hint", stored.ScopeHint)
	}
}

func TestPostClarificationAnswersApplyUnknownAnswerReturnsNotFound(t *testing.T) {
	server := testServer(t)

	response := doJSON(t, server.router, http.MethodPost, "/v1/clarification-answers/missing/apply", applyRequestJSON())
	if response.code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.code, http.StatusNotFound)
	}

	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeJSON(t, response.body, &body)
	if body.Error.Code != "not_found" {
		t.Fatalf("error code = %q, want %q", body.Error.Code, "not_found")
	}
}

func TestPostClarificationAnswersApplyValidation(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "missing applied_by",
			body: `{}`,
		},
		{
			name: "missing applied_by kind",
			body: `{"applied_by":{"id":"dev_1"}}`,
		},
		{
			name: "missing applied_by id",
			body: `{"applied_by":{"kind":"user"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := testServer(t)
			answer := createClarificationAnswerForReasons(t, server, map[spine.ClarificationMapsTo]string{
				spine.ClarificationMapsToGoalScopeHint: "Updated scope hint",
			}, spine.GoalReadinessReasonMissingScopeHint)

			response := doJSON(t, server.router, http.MethodPost, "/v1/clarification-answers/"+string(answer.ID)+"/apply", tt.body)
			if response.code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d: %s", response.code, http.StatusBadRequest, response.body)
			}

			var body struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			decodeJSON(t, response.body, &body)
			if body.Error.Code != "validation_failed" {
				t.Fatalf("error code = %q, want %q", body.Error.Code, "validation_failed")
			}
		})
	}
}

func TestPostClarificationAnswersApplyRejectsRepeatedApplication(t *testing.T) {
	server := testServer(t)
	answer := createClarificationAnswerForReasons(t, server, map[spine.ClarificationMapsTo]string{
		spine.ClarificationMapsToGoalScopeHint: "Updated scope hint",
	}, spine.GoalReadinessReasonMissingScopeHint)

	first := doJSON(t, server.router, http.MethodPost, "/v1/clarification-answers/"+string(answer.ID)+"/apply", applyRequestJSON())
	if first.code != http.StatusOK {
		t.Fatalf("first status = %d, want %d: %s", first.code, http.StatusOK, first.body)
	}
	second := doJSON(t, server.router, http.MethodPost, "/v1/clarification-answers/"+string(answer.ID)+"/apply", applyRequestJSON())
	if second.code != http.StatusConflict {
		t.Fatalf("second status = %d, want %d: %s", second.code, http.StatusConflict, second.body)
	}

	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeJSON(t, second.body, &body)
	if body.Error.Code != "already_applied" {
		t.Fatalf("error code = %q, want %q", body.Error.Code, "already_applied")
	}
}

func TestPostClarificationAnswersApplyUpdatesAllowedMappings(t *testing.T) {
	tests := []struct {
		name   string
		reason spine.GoalReadinessReasonCode
		mapsTo spine.ClarificationMapsTo
		value  string
		assert func(t *testing.T, goal spine.Goal, value string)
	}{
		{
			name:   "summary",
			reason: spine.GoalReadinessReasonMissingGoalSummary,
			mapsTo: spine.ClarificationMapsToGoalSummary,
			value:  "Updated summary",
			assert: func(t *testing.T, goal spine.Goal, value string) {
				t.Helper()
				if goal.Summary != value {
					t.Fatalf("summary = %q, want %q", goal.Summary, value)
				}
			},
		},
		{
			name:   "scope hint",
			reason: spine.GoalReadinessReasonMissingScopeHint,
			mapsTo: spine.ClarificationMapsToGoalScopeHint,
			value:  "Updated scope hint",
			assert: func(t *testing.T, goal spine.Goal, value string) {
				t.Helper()
				if goal.ScopeHint != value {
					t.Fatalf("scope_hint = %q, want %q", goal.ScopeHint, value)
				}
			},
		},
		{
			name:   "acceptance hint",
			reason: spine.GoalReadinessReasonMissingAcceptanceHint,
			mapsTo: spine.ClarificationMapsToGoalAcceptanceHint,
			value:  "Updated acceptance hint",
			assert: func(t *testing.T, goal spine.Goal, value string) {
				t.Helper()
				if goal.AcceptanceHint != value {
					t.Fatalf("acceptance_hint = %q, want %q", goal.AcceptanceHint, value)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := testServer(t)
			answer := createClarificationAnswerForReasons(t, server, map[spine.ClarificationMapsTo]string{
				tt.mapsTo: tt.value,
			}, tt.reason)

			response := doJSON(t, server.router, http.MethodPost, "/v1/clarification-answers/"+string(answer.ID)+"/apply", applyRequestJSON())
			if response.code != http.StatusOK {
				t.Fatalf("status = %d, want %d: %s", response.code, http.StatusOK, response.body)
			}

			var body struct {
				Goal spine.Goal `json:"goal"`
			}
			decodeJSON(t, response.body, &body)
			tt.assert(t, body.Goal, tt.value)
		})
	}
}

func TestPostClarificationAnswersApplyRejectsRawTextIntentOwner(t *testing.T) {
	server := testServer(t)
	answer := createClarificationAnswerForReasons(t, server, map[spine.ClarificationMapsTo]string{
		spine.ClarificationMapsToGoalIntentOwner: "dev_2",
	}, spine.GoalReadinessReasonMissingIntentOwner)

	response := doJSON(t, server.router, http.MethodPost, "/v1/clarification-answers/"+string(answer.ID)+"/apply", applyRequestJSON())
	if response.code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusBadRequest, response.body)
	}

	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeJSON(t, response.body, &body)
	if body.Error.Code != "unsupported_mapping" {
		t.Fatalf("error code = %q, want %q", body.Error.Code, "unsupported_mapping")
	}
}

func testServer(t *testing.T) testServerDeps {
	t.Helper()

	intakeStore := store.NewIntakeStore()
	goalStore := store.NewGoalStore()
	clarificationStore := store.NewClarificationStore()
	answerStore := store.NewClarificationAnswerStore()
	events := eventlog.NewEventLog()
	ids := &sequenceIDs{}
	service := intake.NewService(intakeStore, events, fixedClock{now: testTime()}, ids)
	intakeHandler := httpserver.NewIntakeHandler(service)
	goalService := goal.NewService(intakeStore, goalStore, events, fixedClock{now: testTime()}, ids)
	goalHandler := httpserver.NewGoalHandler(goalService)
	clarificationService := clarification.NewService(goalStore, clarificationStore, answerStore, events, fixedClock{now: testTime()}, ids)
	clarificationHandler := httpserver.NewClarificationHandler(clarificationService)

	return testServerDeps{
		router:         baseHandlers(intakeHandler, goalHandler, clarificationHandler),
		intakes:        intakeStore,
		goals:          goalStore,
		clarifications: clarificationStore,
		answers:        answerStore,
		events:         events,
		idFactory:      ids,
	}
}

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

type sequenceIDs struct {
	intake        int
	goal          int
	clarification int
	question      int
	answer        int
	event         int
}

func (g *sequenceIDs) NewIntakeID() (spine.IntakeID, error) {
	g.intake++
	return spine.IntakeID(fmt.Sprintf("intake-%d", g.intake)), nil
}

func (g *sequenceIDs) NewEventID() (spine.EventID, error) {
	g.event++
	return spine.EventID(fmt.Sprintf("event-%d", g.event)), nil
}

func (g *sequenceIDs) NewGoalID() (spine.GoalID, error) {
	g.goal++
	return spine.GoalID(fmt.Sprintf("goal-%d", g.goal)), nil
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

func testTime() time.Time {
	return time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
}

func decodeRawJSON(t *testing.T, input json.RawMessage, target any) {
	t.Helper()

	if len(input) == 0 {
		t.Fatal("missing JSON field")
	}
	if err := json.Unmarshal(input, target); err != nil {
		t.Fatalf("decode raw JSON %q: %v", string(input), err)
	}
}

func createIntake(t *testing.T, server testServerDeps, body string) string {
	t.Helper()

	response := doJSON(t, server.router, http.MethodPost, "/v1/intake", body)
	if response.code != http.StatusAccepted {
		t.Fatalf("POST /v1/intake status = %d, want %d: %s", response.code, http.StatusAccepted, response.body)
	}

	var accepted struct {
		IntakeID string `json:"intake_id"`
	}
	decodeJSON(t, response.body, &accepted)
	return accepted.IntakeID
}

func promoteIntake(t *testing.T, server testServerDeps, intakeID string) spine.Goal {
	t.Helper()

	response := doJSON(t, server.router, http.MethodPost, "/v1/intake/"+intakeID+"/promote", "")
	if response.code != http.StatusCreated {
		t.Fatalf("POST /v1/intake/{id}/promote status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
	}

	var created spine.Goal
	decodeJSON(t, response.body, &created)
	return created
}

func createClarificationReadyGoal(t *testing.T, server testServerDeps) spine.Goal {
	t.Helper()

	intakeID := createIntake(t, server, validIntakeJSON)
	created := promoteIntake(t, server, intakeID)
	response := doJSON(t, server.router, http.MethodPost, "/v1/goals/"+string(created.ID)+"/readiness", "")
	if response.code != http.StatusOK {
		t.Fatalf("POST /v1/goals/{id}/readiness status = %d, want %d: %s", response.code, http.StatusOK, response.body)
	}

	var body struct {
		Goal spine.Goal `json:"goal"`
	}
	decodeJSON(t, response.body, &body)
	return body.Goal
}

func createClarificationRequest(t *testing.T, server testServerDeps) spine.ClarificationRequest {
	t.Helper()

	created := createClarificationReadyGoal(t, server)
	response := doJSON(t, server.router, http.MethodPost, "/v1/goals/"+string(created.ID)+"/clarification-requests", "")
	if response.code != http.StatusCreated {
		t.Fatalf("POST /v1/goals/{id}/clarification-requests status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
	}

	var request spine.ClarificationRequest
	decodeJSON(t, response.body, &request)
	return request
}

func createClarificationRequestForReasons(t *testing.T, server testServerDeps, reasons ...spine.GoalReadinessReasonCode) spine.ClarificationRequest {
	t.Helper()

	created := spine.Goal{
		ID:            "direct-goal-1",
		IntakeID:      "direct-intake-1",
		RepoBindingID: "repo_demo_1",
		Title:         "Direct goal",
		Summary:       "Original summary",
		SourceRefs: []spine.SourceRef{
			{Kind: "test", ID: "direct-intake-1"},
		},
		RequestAuthor:            spine.ActorRef{Kind: "user", ID: "dev_1"},
		IntentOwner:              spine.ActorRef{Kind: "user", ID: "dev_1"},
		State:                    spine.GoalStateNeedsClarification,
		LastReadinessReasonCodes: append([]spine.GoalReadinessReasonCode(nil), reasons...),
		CreatedAt:                testTime(),
	}
	for _, reason := range reasons {
		if reason == spine.GoalReadinessReasonMissingGoalSummary {
			created.Summary = ""
		}
		if reason == spine.GoalReadinessReasonMissingIntentOwner {
			created.IntentOwner = spine.ActorRef{}
		}
	}
	if err := server.goals.Create(context.Background(), created); err != nil {
		t.Fatalf("goals.Create() error = %v", err)
	}

	response := doJSON(t, server.router, http.MethodPost, "/v1/goals/"+string(created.ID)+"/clarification-requests", "")
	if response.code != http.StatusCreated {
		t.Fatalf("POST /v1/goals/{id}/clarification-requests status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
	}

	var request spine.ClarificationRequest
	decodeJSON(t, response.body, &request)
	return request
}

func createClarificationAnswerForReasons(t *testing.T, server testServerDeps, values map[spine.ClarificationMapsTo]string, reasons ...spine.GoalReadinessReasonCode) spine.ClarificationAnswer {
	t.Helper()

	request := createClarificationRequestForReasons(t, server, reasons...)
	response := doJSON(t, server.router, http.MethodPost, "/v1/clarification-requests/"+string(request.ID)+"/answers", answerSubmissionJSONWithValues(request, values))
	if response.code != http.StatusCreated {
		t.Fatalf("POST /v1/clarification-requests/{id}/answers status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
	}

	var answer spine.ClarificationAnswer
	decodeJSON(t, response.body, &answer)
	return answer
}

func answerSubmissionJSON(request spine.ClarificationRequest) string {
	return answerSubmissionJSONWithValues(request, nil)
}

func answerSubmissionJSONWithValues(request spine.ClarificationRequest, values map[spine.ClarificationMapsTo]string) string {
	answers := make([]string, 0, len(request.Questions))
	for i, question := range request.Questions {
		value := fmt.Sprintf("Answer %d", i+1)
		if values != nil {
			if mapped, ok := values[question.MapsTo]; ok {
				value = mapped
			}
		}
		answers = append(answers, fmt.Sprintf(`{"question_id":%q,"value":%q}`, question.ID, value))
	}
	return fmt.Sprintf(`{"submitted_by":{"kind":"user","id":"dev_1"},"answers":[%s]}`, strings.Join(answers, ","))
}

func applyRequestJSON() string {
	return `{"applied_by":{"kind":"user","id":"dev_1","display_name":"Developer"}}`
}

func hasReadinessReason(reasons []spine.GoalReadinessReasonCode, want spine.GoalReadinessReasonCode) bool {
	for _, reason := range reasons {
		if reason == want {
			return true
		}
	}
	return false
}

func hasClarificationQuestion(questions []spine.ClarificationQuestion, want spine.ClarificationMapsTo) bool {
	for _, question := range questions {
		if question.MapsTo == want {
			return true
		}
	}
	return false
}
