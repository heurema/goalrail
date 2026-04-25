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
	router    http.Handler
	intakes   *store.IntakeStore
	goals     *store.GoalStore
	events    *eventlog.EventLog
	idFactory *sequenceIDs
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

func testServer(t *testing.T) testServerDeps {
	t.Helper()

	intakeStore := store.NewIntakeStore()
	goalStore := store.NewGoalStore()
	events := eventlog.NewEventLog()
	ids := &sequenceIDs{}
	service := intake.NewService(intakeStore, events, fixedClock{now: testTime()}, ids)
	intakeHandler := httpserver.NewIntakeHandler(service)
	goalService := goal.NewService(intakeStore, goalStore, events, fixedClock{now: testTime()}, ids)
	goalHandler := httpserver.NewGoalHandler(goalService)

	return testServerDeps{
		router:    baseHandlers(intakeHandler, goalHandler),
		intakes:   intakeStore,
		goals:     goalStore,
		events:    events,
		idFactory: ids,
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
	goal   int
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

func (g *sequenceIDs) NewGoalID() (spine.GoalID, error) {
	g.goal++
	return spine.GoalID(fmt.Sprintf("goal-%d", g.goal)), nil
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

func hasReadinessReason(reasons []spine.GoalReadinessReasonCode, want spine.GoalReadinessReasonCode) bool {
	for _, reason := range reasons {
		if reason == want {
			return true
		}
	}
	return false
}
