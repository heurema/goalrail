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
	"github.com/heurema/goalrail/apps/server/internal/contractseed"
	"github.com/heurema/goalrail/apps/server/internal/eventlog"
	"github.com/heurema/goalrail/apps/server/internal/goal"
	"github.com/heurema/goalrail/apps/server/internal/httpserver"
	"github.com/heurema/goalrail/apps/server/internal/intake"
	"github.com/heurema/goalrail/apps/server/internal/spine"
	"github.com/heurema/goalrail/apps/server/internal/store"
)

const validIntakeJSON = `{
  "project_id": "018f0000-0000-7000-8000-000000000003",
  "repo_binding_id": "018f0000-0000-7000-8000-000000000004",
  "source": {
    "kind": "codex_skill",
    "external_id": "local-session-1"
  },
  "title": "Refactor CSV export filters",
  "body": "Current code duplicates filter logic. Preserve current behavior.",
  "request_author": {
    "kind": "user",
    "id": "018f0000-0000-7000-8000-000000000001",
    "display_name": "Developer"
  }
}`

type testServerDeps struct {
	router         http.Handler
	intakes        *store.IntakeStore
	goals          *store.GoalStore
	clarifications *store.ClarificationStore
	answers        *store.ClarificationAnswerStore
	contractSeeds  *store.ContractSeedStore
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
	var organizationID string
	decodeRawJSON(t, body["organization_id"], &organizationID)
	if organizationID != "018f0000-0000-7000-8000-000000000002" {
		t.Fatalf("organization_id = %q, want 018f0000-0000-7000-8000-000000000002", organizationID)
	}
	var projectID string
	decodeRawJSON(t, body["project_id"], &projectID)
	if projectID != "018f0000-0000-7000-8000-000000000003" {
		t.Fatalf("project_id = %q, want 018f0000-0000-7000-8000-000000000003", projectID)
	}
	var repoBindingID string
	decodeRawJSON(t, body["repo_binding_id"], &repoBindingID)
	if repoBindingID != "018f0000-0000-7000-8000-000000000004" {
		t.Fatalf("repo_binding_id = %q, want 018f0000-0000-7000-8000-000000000004", repoBindingID)
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

func TestPostIntakeRejectsUnknownRepoBinding(t *testing.T) {
	server := testServerWithResolver(t, fakeProjectContextResolver{ok: false})

	response := doJSON(t, server.router, http.MethodPost, "/v1/intake", validIntakeJSON)
	if response.code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.code, http.StatusBadRequest)
	}

	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	decodeJSON(t, response.body, &body)
	if body.Error.Code != "validation_failed" {
		t.Fatalf("error code = %q, want validation_failed", body.Error.Code)
	}
	if !strings.Contains(body.Error.Message, "repo_binding_id") {
		t.Fatalf("error message = %q, want repo_binding_id", body.Error.Message)
	}
	if got := len(server.events.Events()); got != 0 {
		t.Fatalf("events length = %d, want 0", got)
	}
}

func TestPostIntakeRejectsRepoBindingForDifferentProject(t *testing.T) {
	server := testServerWithResolver(t, fakeProjectContextResolver{
		resolved: spine.ResolvedRepoBindingContext{
			OrganizationID: "018f0000-0000-7000-8000-000000000002",
			ProjectID:      "018f0000-0000-7000-8000-000000000006",
			RepoBindingID:  "018f0000-0000-7000-8000-000000000004",
		},
		ok: true,
	})

	response := doJSON(t, server.router, http.MethodPost, "/v1/intake", validIntakeJSON)
	if response.code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.code, http.StatusBadRequest)
	}
	if got := len(server.events.Events()); got != 0 {
		t.Fatalf("events length = %d, want 0", got)
	}
}

func TestPostIntakeReturnsConfigurationErrorWhenProjectContextUnavailable(t *testing.T) {
	server := testServerWithResolver(t, nil)

	response := doJSON(t, server.router, http.MethodPost, "/v1/intake", validIntakeJSON)
	if response.code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.code, http.StatusServiceUnavailable)
	}

	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeJSON(t, response.body, &body)
	if body.Error.Code != "project_context_unavailable" {
		t.Fatalf("error code = %q, want project_context_unavailable", body.Error.Code)
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
	if record.OrganizationID != "018f0000-0000-7000-8000-000000000002" {
		t.Fatalf("OrganizationID = %q, want %q", record.OrganizationID, "018f0000-0000-7000-8000-000000000002")
	}
	if record.ProjectID != "018f0000-0000-7000-8000-000000000003" {
		t.Fatalf("ProjectID = %q, want %q", record.ProjectID, "018f0000-0000-7000-8000-000000000003")
	}
	if record.RepoBindingID != "018f0000-0000-7000-8000-000000000004" {
		t.Fatalf("RepoBindingID = %q, want %q", record.RepoBindingID, "018f0000-0000-7000-8000-000000000004")
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
			name: "missing project_id",
			body: `{"repo_binding_id":"018f0000-0000-7000-8000-000000000004","source":{"kind":"codex_skill"},"title":"Title","request_author":{"kind":"user","id":"018f0000-0000-7000-8000-000000000001"}}`,
		},
		{
			name: "missing repo_binding_id",
			body: `{"project_id":"018f0000-0000-7000-8000-000000000003","source":{"kind":"codex_skill"},"title":"Title","request_author":{"kind":"user","id":"018f0000-0000-7000-8000-000000000001"}}`,
		},
		{
			name: "invalid project_id",
			body: `{"project_id":"not-a-uuid","repo_binding_id":"018f0000-0000-7000-8000-000000000004","source":{"kind":"codex_skill"},"title":"Title","request_author":{"kind":"user","id":"018f0000-0000-7000-8000-000000000001"}}`,
		},
		{
			name: "invalid repo_binding_id",
			body: `{"project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":"not-a-uuid","source":{"kind":"codex_skill"},"title":"Title","request_author":{"kind":"user","id":"018f0000-0000-7000-8000-000000000001"}}`,
		},
		{
			name: "non uuidv7 project_id",
			body: `{"project_id":"018f0000-0000-4000-8000-000000000003","repo_binding_id":"018f0000-0000-7000-8000-000000000004","source":{"kind":"codex_skill"},"title":"Title","request_author":{"kind":"user","id":"018f0000-0000-7000-8000-000000000001"}}`,
		},
		{
			name: "missing source kind",
			body: `{"project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":"018f0000-0000-7000-8000-000000000004","source":{},"title":"Title","request_author":{"kind":"user","id":"018f0000-0000-7000-8000-000000000001"}}`,
		},
		{
			name: "missing title and body",
			body: `{"project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":"018f0000-0000-7000-8000-000000000004","source":{"kind":"codex_skill"},"request_author":{"kind":"user","id":"018f0000-0000-7000-8000-000000000001"}}`,
		},
		{
			name: "missing request_author kind",
			body: `{"project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":"018f0000-0000-7000-8000-000000000004","source":{"kind":"codex_skill"},"title":"Title","request_author":{"id":"018f0000-0000-7000-8000-000000000001"}}`,
		},
		{
			name: "missing request_author id",
			body: `{"project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":"018f0000-0000-7000-8000-000000000004","source":{"kind":"codex_skill"},"title":"Title","request_author":{"kind":"user"}}`,
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
	body := `{"project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":"018f0000-0000-7000-8000-000000000004","source":{"kind":"codex_skill"},"title":"Title","request_author":{"kind":"user","id":"018f0000-0000-7000-8000-000000000001"},"unexpected":true}`

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
	if created.OrganizationID != "018f0000-0000-7000-8000-000000000002" {
		t.Fatalf("organization_id = %q, want 018f0000-0000-7000-8000-000000000002", created.OrganizationID)
	}
	if created.ProjectID != "018f0000-0000-7000-8000-000000000003" {
		t.Fatalf("project_id = %q, want 018f0000-0000-7000-8000-000000000003", created.ProjectID)
	}
	if created.RepoBindingID != "018f0000-0000-7000-8000-000000000004" {
		t.Fatalf("repo_binding_id = %q, want 018f0000-0000-7000-8000-000000000004", created.RepoBindingID)
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
  "project_id": "018f0000-0000-7000-8000-000000000003",
  "repo_binding_id": "018f0000-0000-7000-8000-000000000004",
  "source": {"kind": "codex_skill"},
  "title": "Refactor CSV export filters",
  "request_author": {"kind": "user", "id": "018f0000-0000-7000-8000-000000000001"}
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
		ID:             "goal-without-reasons",
		IntakeID:       "intake-without-reasons",
		OrganizationID: "018f0000-0000-7000-8000-000000000002",
		ProjectID:      "018f0000-0000-7000-8000-000000000003",
		RepoBindingID:  "018f0000-0000-7000-8000-000000000004",
		Title:          "Refactor CSV export filters",
		Summary:        "Current code duplicates filter logic. Preserve current behavior.",
		SourceRefs: []spine.SourceRef{
			{Kind: "intake", ID: "intake-without-reasons"},
		},
		RequestAuthor: spine.ActorRef{Kind: "user", ID: "018f0000-0000-7000-8000-000000000001"},
		IntentOwner:   spine.ActorRef{Kind: "user", ID: "018f0000-0000-7000-8000-000000000001"},
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
  "submitted_by": {"kind": "user", "id": "018f0000-0000-7000-8000-000000000001"},
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
				return fmt.Sprintf(`{"submitted_by":{"id":"018f0000-0000-7000-8000-000000000001"},"answers":[{"question_id":%q,"value":"Scope"}]}`, request.Questions[0].ID)
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
				return `{"submitted_by":{"kind":"user","id":"018f0000-0000-7000-8000-000000000001"}}`
			},
		},
		{
			name: "unknown question_id",
			body: func(spine.ClarificationRequest) string {
				return `{"submitted_by":{"kind":"user","id":"018f0000-0000-7000-8000-000000000001"},"answers":[{"question_id":"unknown","value":"Scope"}]}`
			},
		},
		{
			name: "duplicate question_id",
			body: func(request spine.ClarificationRequest) string {
				return fmt.Sprintf(`{"submitted_by":{"kind":"user","id":"018f0000-0000-7000-8000-000000000001"},"answers":[{"question_id":%q,"value":"Scope"},{"question_id":%q,"value":"Duplicate"}]}`, request.Questions[0].ID, request.Questions[0].ID)
			},
		},
		{
			name: "missing answer for one question",
			body: func(request spine.ClarificationRequest) string {
				return fmt.Sprintf(`{"submitted_by":{"kind":"user","id":"018f0000-0000-7000-8000-000000000001"},"answers":[{"question_id":%q,"value":"Scope"}]}`, request.Questions[0].ID)
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

func TestPostGoalReadinessExplicitRecheckAfterAppliedAnswersMarksReadyForContractSeed(t *testing.T) {
	server := testServer(t)
	intakeID := createIntake(t, server, validIntakeJSON)
	created := promoteIntake(t, server, intakeID)

	initialReadiness := doJSON(t, server.router, http.MethodPost, "/v1/goals/"+string(created.ID)+"/readiness", "")
	if initialReadiness.code != http.StatusOK {
		t.Fatalf("initial readiness status = %d, want %d: %s", initialReadiness.code, http.StatusOK, initialReadiness.body)
	}

	var initialReadinessBody struct {
		Readiness spine.GoalReadinessResult `json:"readiness"`
		Goal      spine.Goal                `json:"goal"`
	}
	decodeJSON(t, initialReadiness.body, &initialReadinessBody)
	if initialReadinessBody.Readiness.State != spine.GoalStateNeedsClarification {
		t.Fatalf("initial readiness state = %q, want %q", initialReadinessBody.Readiness.State, spine.GoalStateNeedsClarification)
	}
	if !hasReadinessReason(initialReadinessBody.Readiness.ReasonCodes, spine.GoalReadinessReasonMissingScopeHint) {
		t.Fatalf("initial readiness reasons = %#v, want %q", initialReadinessBody.Readiness.ReasonCodes, spine.GoalReadinessReasonMissingScopeHint)
	}
	if !hasReadinessReason(initialReadinessBody.Readiness.ReasonCodes, spine.GoalReadinessReasonMissingAcceptanceHint) {
		t.Fatalf("initial readiness reasons = %#v, want %q", initialReadinessBody.Readiness.ReasonCodes, spine.GoalReadinessReasonMissingAcceptanceHint)
	}

	clarificationResponse := doJSON(t, server.router, http.MethodPost, "/v1/goals/"+string(created.ID)+"/clarification-requests", "")
	if clarificationResponse.code != http.StatusCreated {
		t.Fatalf("clarification request status = %d, want %d: %s", clarificationResponse.code, http.StatusCreated, clarificationResponse.body)
	}

	var request spine.ClarificationRequest
	decodeJSON(t, clarificationResponse.body, &request)
	answerResponse := doJSON(
		t,
		server.router,
		http.MethodPost,
		"/v1/clarification-requests/"+string(request.ID)+"/answers",
		answerSubmissionJSONWithValues(request, map[spine.ClarificationMapsTo]string{
			spine.ClarificationMapsToGoalScopeHint:      "Refactor duplicate CSV export filter logic",
			spine.ClarificationMapsToGoalAcceptanceHint: "Existing CSV export behavior is preserved",
		}),
	)
	if answerResponse.code != http.StatusCreated {
		t.Fatalf("clarification answer status = %d, want %d: %s", answerResponse.code, http.StatusCreated, answerResponse.body)
	}

	var answer spine.ClarificationAnswer
	decodeJSON(t, answerResponse.body, &answer)
	readinessChecksBeforeApply := countEventType(server.events.Events(), goal.EventTypeGoalReadinessChecked)

	applyResponse := doJSON(t, server.router, http.MethodPost, "/v1/clarification-answers/"+string(answer.ID)+"/apply", applyRequestJSON())
	if applyResponse.code != http.StatusOK {
		t.Fatalf("apply status = %d, want %d: %s", applyResponse.code, http.StatusOK, applyResponse.body)
	}
	for _, forbiddenField := range []string{"\"contract_id\"", "\"contract_seed_id\"", "\"work_item_id\"", "\"proof_id\""} {
		if strings.Contains(applyResponse.body, forbiddenField) {
			t.Fatalf("apply response includes forbidden field %s", forbiddenField)
		}
	}

	var applyBody struct {
		Goal spine.Goal `json:"goal"`
	}
	decodeJSON(t, applyResponse.body, &applyBody)
	if applyBody.Goal.ScopeHint != "Refactor duplicate CSV export filter logic" {
		t.Fatalf("scope_hint = %q, want applied value", applyBody.Goal.ScopeHint)
	}
	if applyBody.Goal.AcceptanceHint != "Existing CSV export behavior is preserved" {
		t.Fatalf("acceptance_hint = %q, want applied value", applyBody.Goal.AcceptanceHint)
	}
	if applyBody.Goal.State != spine.GoalStateNeedsClarification {
		t.Fatalf("goal state after apply = %q, want %q before explicit re-check", applyBody.Goal.State, spine.GoalStateNeedsClarification)
	}
	if got := countEventType(server.events.Events(), goal.EventTypeGoalReadinessChecked); got != readinessChecksBeforeApply {
		t.Fatalf("readiness checks after apply = %d, want unchanged %d", got, readinessChecksBeforeApply)
	}
	if got := countEventType(server.events.Events(), goal.EventTypeGoalMarkedReadyForContractSeed); got != 0 {
		t.Fatalf("ready_for_contract_seed events after apply = %d, want 0 before explicit re-check", got)
	}

	recheckResponse := doJSON(t, server.router, http.MethodPost, "/v1/goals/"+string(created.ID)+"/readiness", "")
	if recheckResponse.code != http.StatusOK {
		t.Fatalf("explicit re-check status = %d, want %d: %s", recheckResponse.code, http.StatusOK, recheckResponse.body)
	}
	for _, forbiddenField := range []string{"\"contract_id\"", "\"contract_seed_id\"", "\"work_item_id\"", "\"proof_id\""} {
		if strings.Contains(recheckResponse.body, forbiddenField) {
			t.Fatalf("re-check response includes forbidden field %s", forbiddenField)
		}
	}

	var recheckBody struct {
		Readiness spine.GoalReadinessResult `json:"readiness"`
		Goal      spine.Goal                `json:"goal"`
	}
	decodeJSON(t, recheckResponse.body, &recheckBody)
	if recheckBody.Readiness.State != spine.GoalStateReadyForContractSeed {
		t.Fatalf("re-check readiness state = %q, want %q", recheckBody.Readiness.State, spine.GoalStateReadyForContractSeed)
	}
	if !recheckBody.Readiness.Ready {
		t.Fatal("re-check Ready = false, want true")
	}
	if len(recheckBody.Readiness.ReasonCodes) != 0 {
		t.Fatalf("re-check reason codes = %#v, want empty", recheckBody.Readiness.ReasonCodes)
	}
	if recheckBody.Goal.State != spine.GoalStateReadyForContractSeed {
		t.Fatalf("re-check goal state = %q, want %q", recheckBody.Goal.State, spine.GoalStateReadyForContractSeed)
	}

	stored, ok, err := server.goals.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("goals.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("stored goal not found")
	}
	if stored.State != spine.GoalStateReadyForContractSeed {
		t.Fatalf("stored state = %q, want %q", stored.State, spine.GoalStateReadyForContractSeed)
	}
	if len(stored.LastReadinessReasonCodes) != 0 {
		t.Fatalf("stored readiness reason codes = %#v, want empty", stored.LastReadinessReasonCodes)
	}
	if got := countEventType(server.events.Events(), goal.EventTypeGoalReadinessChecked); got != readinessChecksBeforeApply+1 {
		t.Fatalf("readiness checks after explicit re-check = %d, want %d", got, readinessChecksBeforeApply+1)
	}
	if got := countEventType(server.events.Events(), goal.EventTypeGoalMarkedReadyForContractSeed); got != 1 {
		t.Fatalf("ready_for_contract_seed events after explicit re-check = %d, want 1", got)
	}
	assertNoForbiddenEventTypes(t, server.events.Events())
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
	return testServerWithResolver(t, validProjectContextResolver())
}

func testServerWithResolver(t *testing.T, resolver intake.ProjectContextResolver) testServerDeps {
	t.Helper()

	intakeStore := store.NewIntakeStore()
	goalStore := store.NewGoalStore()
	clarificationStore := store.NewClarificationStore()
	answerStore := store.NewClarificationAnswerStore()
	contractSeedStore := store.NewContractSeedStore()
	events := eventlog.NewEventLog()
	ids := &sequenceIDs{}
	service := intake.NewService(intakeStore, resolver, events, fixedClock{now: testTime()}, ids)
	intakeHandler := httpserver.NewIntakeHandler(service)
	goalService := goal.NewService(intakeStore, goalStore, events, fixedClock{now: testTime()}, ids)
	goalHandler := httpserver.NewGoalHandler(goalService)
	clarificationService := clarification.NewService(goalStore, clarificationStore, answerStore, events, fixedClock{now: testTime()}, ids)
	clarificationHandler := httpserver.NewClarificationHandler(clarificationService)
	contractSeedService := contractseed.NewService(goalStore, contractSeedStore, events, fixedClock{now: testTime()}, ids)
	contractSeedHandler := httpserver.NewContractSeedHandler(contractSeedService)

	return testServerDeps{
		router:         baseHandlers(intakeHandler, goalHandler, clarificationHandler, contractSeedHandler),
		intakes:        intakeStore,
		goals:          goalStore,
		clarifications: clarificationStore,
		answers:        answerStore,
		contractSeeds:  contractSeedStore,
		events:         events,
		idFactory:      ids,
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
	contractSeed  int
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

func (g *sequenceIDs) NewContractSeedID() (spine.ContractSeedID, error) {
	g.contractSeed++
	return spine.ContractSeedID(fmt.Sprintf("contract-seed-%d", g.contractSeed)), nil
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
		ID:             "direct-goal-1",
		IntakeID:       "direct-intake-1",
		OrganizationID: "018f0000-0000-7000-8000-000000000002",
		ProjectID:      "018f0000-0000-7000-8000-000000000003",
		RepoBindingID:  "018f0000-0000-7000-8000-000000000004",
		Title:          "Direct goal",
		Summary:        "Original summary",
		SourceRefs: []spine.SourceRef{
			{Kind: "test", ID: "direct-intake-1"},
		},
		RequestAuthor:            spine.ActorRef{Kind: "user", ID: "018f0000-0000-7000-8000-000000000001"},
		IntentOwner:              spine.ActorRef{Kind: "user", ID: "018f0000-0000-7000-8000-000000000001"},
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
	return fmt.Sprintf(`{"submitted_by":{"kind":"user","id":"018f0000-0000-7000-8000-000000000001"},"answers":[%s]}`, strings.Join(answers, ","))
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

func countEventType(events []spine.Event, eventType string) int {
	count := 0
	for _, event := range events {
		if event.Type == eventType {
			count++
		}
	}
	return count
}

func assertNoForbiddenEventTypes(t *testing.T, events []spine.Event) {
	t.Helper()

	forbidden := map[string]bool{
		"contract.seed_created":  true,
		"contract.draft_created": true,
		"contract.approved":      true,
		"contract.created":       true,
		"work_item.created":      true,
		"run.started":            true,
		"gate.decision_written":  true,
		"proof.created":          true,
	}
	for _, event := range events {
		if forbidden[event.Type] {
			t.Fatalf("forbidden event type appended: %s", event.Type)
		}
	}
}
