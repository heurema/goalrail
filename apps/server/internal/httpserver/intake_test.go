package httpserver_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/approvedcontract"
	"github.com/heurema/goalrail/apps/server/internal/auth"
	"github.com/heurema/goalrail/apps/server/internal/checkout"
	"github.com/heurema/goalrail/apps/server/internal/clarification"
	"github.com/heurema/goalrail/apps/server/internal/continuation"
	contractsvc "github.com/heurema/goalrail/apps/server/internal/contract"
	"github.com/heurema/goalrail/apps/server/internal/contractdraft"
	"github.com/heurema/goalrail/apps/server/internal/contractseed"
	"github.com/heurema/goalrail/apps/server/internal/execution"
	"github.com/heurema/goalrail/apps/server/internal/goal"
	"github.com/heurema/goalrail/apps/server/internal/httpserver"
	"github.com/heurema/goalrail/apps/server/internal/intake"
	"github.com/heurema/goalrail/apps/server/internal/spine"
	"github.com/heurema/goalrail/apps/server/internal/workitem"
	"github.com/heurema/goalrail/apps/server/internal/workitemplan"
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
	router            http.Handler
	intakes           *fakeIntakeStore
	goals             *fakeGoalStore
	clarifications    *fakeClarificationStore
	answers           *fakeClarificationAnswerStore
	contracts         *fakeContractStore
	contractSeeds     *fakeContractSeedStore
	contractDrafts    *fakeContractDraftStore
	approvedContracts *fakeApprovedContractStore
	workItems         *fakeWorkItemStore
	workItemPlans     *fakeWorkItemPlanStore
	workItemLeases    *fakeWorkItemPlanLeaseStore
	workItemProposals *fakeWorkItemPlanProposalStore
	repoBindings      *fakeRepoBindingStore
	checkoutJobs      *fakeCheckoutJobStore
	checkoutReceipts  *fakeCheckoutReceiptStore
	executionJobs     *fakeExecutionJobStore
	runs              *fakeRunStore
	commandPlans      *fakeExecutionCommandPlanStore
	executionReceipts *fakeExecutionReceiptStore
	events            *fakeEventLog
	idFactory         *sequenceIDs
}

func TestPostIntakeReturnsAccepted(t *testing.T) {
	server := testServer(t)

	response := doJSON(t, server.router, http.MethodPost, "/v1/intakes", validIntakeJSON)
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

	response := doJSON(t, server.router, http.MethodPost, "/v1/intakes", validIntakeJSON)
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

	response := doJSON(t, server.router, http.MethodPost, "/v1/intakes", validIntakeJSON)
	if response.code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.code, http.StatusBadRequest)
	}
	if got := len(server.events.Events()); got != 0 {
		t.Fatalf("events length = %d, want 0", got)
	}
}

func TestGetIntakeReturnsStoredRecord(t *testing.T) {
	server := testServer(t)

	postResponse := doJSON(t, server.router, http.MethodPost, "/v1/intakes", validIntakeJSON)
	if postResponse.code != http.StatusAccepted {
		t.Fatalf("POST status = %d, want %d", postResponse.code, http.StatusAccepted)
	}

	var accepted struct {
		IntakeID string `json:"intake_id"`
	}
	decodeJSON(t, postResponse.body, &accepted)

	getResponse := doJSON(t, server.router, http.MethodGet, "/v1/intakes/"+accepted.IntakeID, "")
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

	response := doJSON(t, server.router, http.MethodGet, "/v1/intakes/missing", "")
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

			response := doJSON(t, server.router, http.MethodPost, "/v1/intakes", tt.body)
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

	response := doJSON(t, server.router, http.MethodPost, "/v1/intakes", body)
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

	response := doJSON(t, server.router, http.MethodPost, "/v1/intakes", validIntakeJSON)
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

	response := doJSON(t, server.router, http.MethodPost, "/v1/intakes/"+intakeID+"/goals", "")
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

	response := doJSON(t, server.router, http.MethodPost, "/v1/intakes/missing/goals", "")
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

	first := doJSON(t, server.router, http.MethodPost, "/v1/intakes/"+intakeID+"/goals", "")
	if first.code != http.StatusCreated {
		t.Fatalf("first status = %d, want %d", first.code, http.StatusCreated)
	}

	second := doJSON(t, server.router, http.MethodPost, "/v1/intakes/"+intakeID+"/goals", "")
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

	response := doJSON(t, server.router, http.MethodPost, "/v1/intakes/"+intakeID+"/goals", "")
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

func TestPostGoalContinuationReturnsReadyForContractSeed(t *testing.T) {
	server := testServer(t)
	created := createReadyEnoughGoal(t, server)

	response := doJSON(t, server.router, http.MethodPost, "/v1/goals/"+string(created.ID)+"/continuation", "")
	if response.code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusOK, response.body)
	}
	for _, forbiddenField := range []string{"\"contract_id\"", "\"work_item_id\"", "\"proof_id\""} {
		if strings.Contains(response.body, forbiddenField) {
			t.Fatalf("response includes forbidden field %s", forbiddenField)
		}
	}

	var body spine.GoalContinuation
	decodeJSON(t, response.body, &body)
	if body.GoalID != created.ID {
		t.Fatalf("goal_id = %q, want %q", body.GoalID, created.ID)
	}
	if body.State != spine.GoalStateReadyForContractSeed {
		t.Fatalf("state = %q, want %q", body.State, spine.GoalStateReadyForContractSeed)
	}
	if body.Readiness == nil || !body.Readiness.Ready {
		t.Fatalf("readiness = %#v, want ready result", body.Readiness)
	}
	if body.ClarificationRequest != nil {
		t.Fatalf("clarification_request = %#v, want nil", body.ClarificationRequest)
	}
	if len(server.clarifications.requests) != 0 {
		t.Fatalf("clarification requests = %d, want 0", len(server.clarifications.requests))
	}
	assertNoContractTaskSideEffects(t, server)
}

func TestPostGoalContinuationReturnsOpenClarificationForIncompleteGoal(t *testing.T) {
	server := testServer(t)
	intakeID := createIntake(t, server, validIntakeJSON)
	created := promoteIntake(t, server, intakeID)

	response := doJSON(t, server.router, http.MethodPost, "/v1/goals/"+string(created.ID)+"/continuation", "")
	if response.code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusOK, response.body)
	}

	var body spine.GoalContinuation
	decodeJSON(t, response.body, &body)
	if body.State != spine.GoalStateNeedsClarification {
		t.Fatalf("state = %q, want %q", body.State, spine.GoalStateNeedsClarification)
	}
	if body.Readiness == nil || body.Readiness.Ready {
		t.Fatalf("readiness = %#v, want not ready result", body.Readiness)
	}
	if body.ClarificationRequest == nil {
		t.Fatal("clarification_request = nil, want open request")
	}
	if body.ClarificationRequest.GoalID != created.ID {
		t.Fatalf("clarification goal_id = %q, want %q", body.ClarificationRequest.GoalID, created.ID)
	}
	if body.ClarificationRequest.State != spine.ClarificationRequestStateOpen {
		t.Fatalf("clarification state = %q, want %q", body.ClarificationRequest.State, spine.ClarificationRequestStateOpen)
	}
	if !hasClarificationQuestion(body.ClarificationRequest.Questions, spine.ClarificationMapsToGoalScopeHint) {
		t.Fatalf("questions = %#v, want scope_hint question", body.ClarificationRequest.Questions)
	}
	if !hasClarificationQuestion(body.ClarificationRequest.Questions, spine.ClarificationMapsToGoalAcceptanceHint) {
		t.Fatalf("questions = %#v, want acceptance_hint question", body.ClarificationRequest.Questions)
	}
	assertNoContractTaskSideEffects(t, server)
}

func TestPostGoalContinuationReusesOpenClarificationRequest(t *testing.T) {
	server := testServer(t)
	intakeID := createIntake(t, server, validIntakeJSON)
	created := promoteIntake(t, server, intakeID)

	first := doJSON(t, server.router, http.MethodPost, "/v1/goals/"+string(created.ID)+"/continuation", "")
	if first.code != http.StatusOK {
		t.Fatalf("first status = %d, want %d: %s", first.code, http.StatusOK, first.body)
	}
	second := doJSON(t, server.router, http.MethodPost, "/v1/goals/"+string(created.ID)+"/continuation", "")
	if second.code != http.StatusOK {
		t.Fatalf("second status = %d, want %d: %s", second.code, http.StatusOK, second.body)
	}
	third := doJSON(t, server.router, http.MethodPost, "/v1/goals/"+string(created.ID)+"/continuation", "")
	if third.code != http.StatusOK {
		t.Fatalf("third status = %d, want %d: %s", third.code, http.StatusOK, third.body)
	}

	var firstBody spine.GoalContinuation
	var secondBody spine.GoalContinuation
	var thirdBody spine.GoalContinuation
	decodeJSON(t, first.body, &firstBody)
	decodeJSON(t, second.body, &secondBody)
	decodeJSON(t, third.body, &thirdBody)
	if firstBody.ClarificationRequest == nil || secondBody.ClarificationRequest == nil || thirdBody.ClarificationRequest == nil {
		t.Fatalf("clarification requests = %#v / %#v / %#v, want all present", firstBody.ClarificationRequest, secondBody.ClarificationRequest, thirdBody.ClarificationRequest)
	}
	if firstBody.ClarificationRequest.ID != secondBody.ClarificationRequest.ID || firstBody.ClarificationRequest.ID != thirdBody.ClarificationRequest.ID {
		t.Fatalf("clarification request ids = %q/%q/%q, want same", firstBody.ClarificationRequest.ID, secondBody.ClarificationRequest.ID, thirdBody.ClarificationRequest.ID)
	}
	if len(server.clarifications.requests) != 1 {
		t.Fatalf("clarification requests = %d, want 1", len(server.clarifications.requests))
	}
}

func TestPostGoalContinuationInvalidGoalIDReturnsValidationError(t *testing.T) {
	server := testServer(t)

	response := doJSON(t, server.router, http.MethodPost, "/v1/goals/not-a-uuid/continuation", "")
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
		t.Fatalf("error code = %q, want validation_failed", body.Error.Code)
	}
	if len(server.clarifications.requests) != 0 {
		t.Fatalf("clarification requests = %d, want 0", len(server.clarifications.requests))
	}
	assertNoContractTaskSideEffects(t, server)
}

func TestPostGoalContinuationRequiresAuthentication(t *testing.T) {
	server := testServerWithContinuationAuth(t, fakeHTTPAuthService{meErr: auth.ErrInvalidToken})
	created := createReadyEnoughGoal(t, server)

	response := doJSON(t, server.router, http.MethodPost, "/v1/goals/"+string(created.ID)+"/continuation", "")
	if response.code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusUnauthorized, response.body)
	}

	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeJSON(t, response.body, &body)
	if body.Error.Code != "unauthorized" {
		t.Fatalf("error code = %q, want unauthorized", body.Error.Code)
	}

	stored, ok, err := server.goals.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("goals.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("goal missing after unauthorized continuation")
	}
	if stored.State != spine.GoalStateCreated {
		t.Fatalf("stored state = %q, want %q", stored.State, spine.GoalStateCreated)
	}
	if len(server.clarifications.requests) != 0 {
		t.Fatalf("clarification requests = %d, want 0", len(server.clarifications.requests))
	}
	assertNoContractTaskSideEffects(t, server)
}

func TestPostGoalContinuationRejectsOrganizationMismatchBeforeMutation(t *testing.T) {
	server := testServerWithContinuationAuth(t, fakeHTTPAuthService{
		profile: continuationAuthProfile("018f0000-0000-7000-8000-000000009999"),
	})
	created := createReadyEnoughGoal(t, server)

	response := doJSON(t, server.router, http.MethodPost, "/v1/goals/"+string(created.ID)+"/continuation", "")
	if response.code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusForbidden, response.body)
	}

	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeJSON(t, response.body, &body)
	if body.Error.Code != "forbidden" {
		t.Fatalf("error code = %q, want forbidden", body.Error.Code)
	}

	stored, ok, err := server.goals.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("goals.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("goal missing after forbidden continuation")
	}
	if stored.State != spine.GoalStateCreated {
		t.Fatalf("stored state = %q, want %q", stored.State, spine.GoalStateCreated)
	}
	if len(server.clarifications.requests) != 0 {
		t.Fatalf("clarification requests = %d, want 0", len(server.clarifications.requests))
	}
	assertNoContractTaskSideEffects(t, server)
}

func TestPostGoalContinuationReturnsRejectedGoalAsBlockedState(t *testing.T) {
	server := testServer(t)
	created := createReadyEnoughGoal(t, server)
	created.State = spine.GoalStateRejected
	if err := server.goals.Create(context.Background(), created); err != nil {
		t.Fatalf("goals.Create() error = %v", err)
	}

	response := doJSON(t, server.router, http.MethodPost, "/v1/goals/"+string(created.ID)+"/continuation", "")
	if response.code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusOK, response.body)
	}

	var body spine.GoalContinuation
	decodeJSON(t, response.body, &body)
	if body.State != spine.GoalStateRejected {
		t.Fatalf("state = %q, want %q", body.State, spine.GoalStateRejected)
	}
	if body.Readiness != nil {
		t.Fatalf("readiness = %#v, want nil", body.Readiness)
	}
	if body.ClarificationRequest != nil {
		t.Fatalf("clarification_request = %#v, want nil", body.ClarificationRequest)
	}
}

func TestPostClarificationAnswersContinuationReturnsReadyNextAction(t *testing.T) {
	server := testServer(t)
	request := createClarificationRequest(t, server)

	response := doJSON(t, server.router, http.MethodPost, "/v1/clarifications/"+string(request.ID)+"/answers/continuation", workAnswerSubmissionJSONWithValues(request, map[spine.ClarificationMapsTo]string{
		spine.ClarificationMapsToGoalScopeHint:      "CSV export filter duplicate handling",
		spine.ClarificationMapsToGoalAcceptanceHint: "Existing CSV export behavior is preserved",
	}))
	if response.code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusOK, response.body)
	}
	for _, forbiddenField := range []string{"\"contract_id\"", "\"work_item_id\"", "\"proof_id\""} {
		if strings.Contains(response.body, forbiddenField) {
			t.Fatalf("response includes forbidden field %s", forbiddenField)
		}
	}

	var body spine.GoalContinuation
	decodeJSON(t, response.body, &body)
	if body.GoalID != request.GoalID {
		t.Fatalf("goal_id = %q, want %q", body.GoalID, request.GoalID)
	}
	if body.State != spine.GoalStateReadyForContractSeed {
		t.Fatalf("state = %q, want %q", body.State, spine.GoalStateReadyForContractSeed)
	}
	if body.Readiness == nil || !body.Readiness.Ready {
		t.Fatalf("readiness = %#v, want ready result", body.Readiness)
	}
	if body.ClarificationRequest != nil {
		t.Fatalf("clarification_request = %#v, want nil", body.ClarificationRequest)
	}

	storedGoal, ok, err := server.goals.Get(context.Background(), request.GoalID)
	if err != nil {
		t.Fatalf("goals.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("stored goal not found")
	}
	if storedGoal.ScopeHint != "CSV export filter duplicate handling" {
		t.Fatalf("scope_hint = %q, want applied answer", storedGoal.ScopeHint)
	}
	if storedGoal.AcceptanceHint != "Existing CSV export behavior is preserved" {
		t.Fatalf("acceptance_hint = %q, want applied answer", storedGoal.AcceptanceHint)
	}
	if len(server.answers.answers) != 1 {
		t.Fatalf("answers = %d, want 1", len(server.answers.answers))
	}
	for answerID := range server.answers.answers {
		if !server.answers.applied[answerID] {
			t.Fatalf("answer %q was not marked applied", answerID)
		}
	}
	assertNoContractTaskSideEffects(t, server)
}

func TestPostClarificationAnswersContinuationReturnsAskUserWhenStillIncomplete(t *testing.T) {
	server := testServer(t)
	request := createClarificationRequestForReasons(t, server, spine.GoalReadinessReasonMissingScopeHint)

	response := doJSON(t, server.router, http.MethodPost, "/v1/clarifications/"+string(request.ID)+"/answers/continuation", workAnswerSubmissionJSONWithValues(request, map[spine.ClarificationMapsTo]string{
		spine.ClarificationMapsToGoalScopeHint: "Bounded answer bridge only",
	}))
	if response.code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusOK, response.body)
	}

	var body spine.GoalContinuation
	decodeJSON(t, response.body, &body)
	if body.State != spine.GoalStateNeedsClarification {
		t.Fatalf("state = %q, want %q", body.State, spine.GoalStateNeedsClarification)
	}
	if body.Readiness == nil || body.Readiness.Ready {
		t.Fatalf("readiness = %#v, want not ready result", body.Readiness)
	}
	if body.ClarificationRequest == nil {
		t.Fatal("clarification_request = nil, want next open request")
	}
	if body.ClarificationRequest.ID == request.ID {
		t.Fatalf("clarification_request.id = %q, want a new open request after answered request", body.ClarificationRequest.ID)
	}
	if body.ClarificationRequest.State != spine.ClarificationRequestStateOpen {
		t.Fatalf("clarification_request.state = %q, want open", body.ClarificationRequest.State)
	}
	if !hasClarificationQuestion(body.ClarificationRequest.Questions, spine.ClarificationMapsToGoalAcceptanceHint) {
		t.Fatalf("questions = %#v, want acceptance_hint question", body.ClarificationRequest.Questions)
	}
	assertNoContractTaskSideEffects(t, server)
}

func TestPostClarificationAnswersContinuationRejectsAlreadyAnsweredRequest(t *testing.T) {
	server := testServer(t)
	request := createClarificationRequest(t, server)
	body := workAnswerSubmissionJSON(request)

	first := doJSON(t, server.router, http.MethodPost, "/v1/clarifications/"+string(request.ID)+"/answers/continuation", body)
	if first.code != http.StatusOK {
		t.Fatalf("first status = %d, want %d: %s", first.code, http.StatusOK, first.body)
	}

	second := doJSON(t, server.router, http.MethodPost, "/v1/clarifications/"+string(request.ID)+"/answers/continuation", body)
	if second.code != http.StatusConflict {
		t.Fatalf("second status = %d, want %d: %s", second.code, http.StatusConflict, second.body)
	}

	var responseBody struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeJSON(t, second.body, &responseBody)
	if responseBody.Error.Code != "already_answered" {
		t.Fatalf("error code = %q, want already_answered", responseBody.Error.Code)
	}
	if len(server.answers.answers) != 1 {
		t.Fatalf("answers = %d, want 1", len(server.answers.answers))
	}
	assertNoContractTaskSideEffects(t, server)
}

func TestPostClarificationAnswersContinuationInvalidRequestIDReturnsValidationError(t *testing.T) {
	server := testServer(t)

	response := doJSON(t, server.router, http.MethodPost, "/v1/clarifications/not-a-uuid/answers/continuation", `{"answers":[]}`)
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
		t.Fatalf("error code = %q, want validation_failed", body.Error.Code)
	}
	if len(server.answers.answers) != 0 {
		t.Fatalf("answers = %d, want 0", len(server.answers.answers))
	}
	assertNoContractTaskSideEffects(t, server)
}

func TestPostClarificationAnswersContinuationRejectsUnsupportedMappingBeforeMutation(t *testing.T) {
	server := testServer(t)
	request := createClarificationRequestForReasons(t, server, spine.GoalReadinessReasonMissingIntentOwner)

	response := doJSON(t, server.router, http.MethodPost, "/v1/clarifications/"+string(request.ID)+"/answers/continuation", workAnswerSubmissionJSONWithValues(request, map[spine.ClarificationMapsTo]string{
		spine.ClarificationMapsToGoalIntentOwner: "dev_2",
	}))
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
		t.Fatalf("error code = %q, want unsupported_mapping", body.Error.Code)
	}
	storedRequest, ok, err := server.clarifications.Get(context.Background(), request.ID)
	if err != nil {
		t.Fatalf("clarifications.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("clarification request missing after unsupported mapping")
	}
	if storedRequest.State != spine.ClarificationRequestStateOpen {
		t.Fatalf("request state = %q, want open", storedRequest.State)
	}
	if len(server.answers.answers) != 0 {
		t.Fatalf("answers = %d, want 0", len(server.answers.answers))
	}
	assertNoContractTaskSideEffects(t, server)
}

func TestPostClarificationAnswersContinuationAppliesActorShapedIntentOwner(t *testing.T) {
	server := testServer(t)
	request := createClarificationRequestForReasons(t, server, spine.GoalReadinessReasonMissingIntentOwner)

	response := doJSON(t, server.router, http.MethodPost, "/v1/clarifications/"+string(request.ID)+"/answers/continuation", fmt.Sprintf(`{"answers":[{"question_id":%q,"value":"","actor_ref":{"kind":"user","id":"018f0000-0000-7000-8000-000000000099","display_name":"Dev 2"}}]}`, request.Questions[0].ID))
	if response.code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusOK, response.body)
	}

	storedGoal, ok, err := server.goals.Get(context.Background(), request.GoalID)
	if err != nil {
		t.Fatalf("goals.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("stored goal not found")
	}
	if storedGoal.IntentOwner.ID != "018f0000-0000-7000-8000-000000000099" || storedGoal.IntentOwner.DisplayName != "Dev 2" {
		t.Fatalf("intent_owner = %#v, want actor-shaped answer", storedGoal.IntentOwner)
	}
	if len(server.answers.answers) != 1 {
		t.Fatalf("answers = %d, want 1", len(server.answers.answers))
	}
	assertNoContractTaskSideEffects(t, server)
}

func TestPostClarificationAnswersContinuationRequiresAuthenticationBeforeMutation(t *testing.T) {
	server := testServerWithContinuationAuth(t, fakeHTTPAuthService{meErr: auth.ErrInvalidToken})
	request := createClarificationRequest(t, server)

	response := doJSON(t, server.router, http.MethodPost, "/v1/clarifications/"+string(request.ID)+"/answers/continuation", workAnswerSubmissionJSON(request))
	if response.code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusUnauthorized, response.body)
	}

	storedRequest, ok, err := server.clarifications.Get(context.Background(), request.ID)
	if err != nil {
		t.Fatalf("clarifications.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("clarification request missing after unauthorized answer")
	}
	if storedRequest.State != spine.ClarificationRequestStateOpen {
		t.Fatalf("request state = %q, want open", storedRequest.State)
	}
	if len(server.answers.answers) != 0 {
		t.Fatalf("answers = %d, want 0", len(server.answers.answers))
	}
	assertNoContractTaskSideEffects(t, server)
}

func TestPostClarificationAnswersContinuationRejectsOrganizationMismatchBeforeMutation(t *testing.T) {
	server := testServerWithContinuationAuth(t, fakeHTTPAuthService{
		profile: continuationAuthProfile("018f0000-0000-7000-8000-000000009999"),
	})
	request := createClarificationRequest(t, server)

	response := doJSON(t, server.router, http.MethodPost, "/v1/clarifications/"+string(request.ID)+"/answers/continuation", workAnswerSubmissionJSON(request))
	if response.code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusForbidden, response.body)
	}

	storedRequest, ok, err := server.clarifications.Get(context.Background(), request.ID)
	if err != nil {
		t.Fatalf("clarifications.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("clarification request missing after forbidden answer")
	}
	if storedRequest.State != spine.ClarificationRequestStateOpen {
		t.Fatalf("request state = %q, want open", storedRequest.State)
	}
	if len(server.answers.answers) != 0 {
		t.Fatalf("answers = %d, want 0", len(server.answers.answers))
	}
	assertNoContractTaskSideEffects(t, server)
}

func TestPostGoalClarificationRequestsReturnsOpenRequest(t *testing.T) {
	server := testServer(t)
	created := createClarificationReadyGoal(t, server)

	response := doJSON(t, server.router, http.MethodPost, "/v1/goals/"+string(created.ID)+"/clarifications", "")
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

	response := doJSON(t, server.router, http.MethodPost, "/v1/goals/missing/clarifications", "")
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

	response := doJSON(t, server.router, http.MethodPost, "/v1/goals/"+string(created.ID)+"/clarifications", "")
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

	response := doJSON(t, server.router, http.MethodPost, "/v1/goals/"+string(created.ID)+"/clarifications", "")
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

	first := doJSON(t, server.router, http.MethodPost, "/v1/goals/"+string(created.ID)+"/clarifications", "")
	if first.code != http.StatusCreated {
		t.Fatalf("first status = %d, want %d: %s", first.code, http.StatusCreated, first.body)
	}
	second := doJSON(t, server.router, http.MethodPost, "/v1/goals/"+string(created.ID)+"/clarifications", "")
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

	response := doJSON(t, server.router, http.MethodPost, "/v1/clarifications/"+string(request.ID)+"/answers", body)
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

	response := doJSON(t, server.router, http.MethodPost, "/v1/clarifications/missing/answers", `{
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

	first := doJSON(t, server.router, http.MethodPost, "/v1/clarifications/"+string(request.ID)+"/answers", body)
	if first.code != http.StatusCreated {
		t.Fatalf("first status = %d, want %d: %s", first.code, http.StatusCreated, first.body)
	}
	second := doJSON(t, server.router, http.MethodPost, "/v1/clarifications/"+string(request.ID)+"/answers", body)
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

			response := doJSON(t, server.router, http.MethodPost, "/v1/clarifications/"+string(request.ID)+"/answers", tt.body(request))
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

	response := doJSON(t, server.router, http.MethodPost, "/v1/answers/"+string(answer.ID)+"/applications", applyRequestJSON())
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

	clarificationResponse := doJSON(t, server.router, http.MethodPost, "/v1/goals/"+string(created.ID)+"/clarifications", "")
	if clarificationResponse.code != http.StatusCreated {
		t.Fatalf("clarification request status = %d, want %d: %s", clarificationResponse.code, http.StatusCreated, clarificationResponse.body)
	}

	var request spine.ClarificationRequest
	decodeJSON(t, clarificationResponse.body, &request)
	answerResponse := doJSON(
		t,
		server.router,
		http.MethodPost,
		"/v1/clarifications/"+string(request.ID)+"/answers",
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

	applyResponse := doJSON(t, server.router, http.MethodPost, "/v1/answers/"+string(answer.ID)+"/applications", applyRequestJSON())
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

	response := doJSON(t, server.router, http.MethodPost, "/v1/answers/missing/applications", applyRequestJSON())
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

			response := doJSON(t, server.router, http.MethodPost, "/v1/answers/"+string(answer.ID)+"/applications", tt.body)
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

	first := doJSON(t, server.router, http.MethodPost, "/v1/answers/"+string(answer.ID)+"/applications", applyRequestJSON())
	if first.code != http.StatusOK {
		t.Fatalf("first status = %d, want %d: %s", first.code, http.StatusOK, first.body)
	}
	second := doJSON(t, server.router, http.MethodPost, "/v1/answers/"+string(answer.ID)+"/applications", applyRequestJSON())
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

			response := doJSON(t, server.router, http.MethodPost, "/v1/answers/"+string(answer.ID)+"/applications", applyRequestJSON())
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

	response := doJSON(t, server.router, http.MethodPost, "/v1/answers/"+string(answer.ID)+"/applications", applyRequestJSON())
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
	return testServerWithResolverAndContinuationAuth(t, resolver, fakeHTTPAuthService{
		profile: continuationAuthProfile("018f0000-0000-7000-8000-000000000002"),
	})
}

func testServerWithContinuationAuth(t *testing.T, authService httpserver.AuthService) testServerDeps {
	t.Helper()
	return testServerWithResolverAndContinuationAuth(t, validProjectContextResolver(), authService)
}

func testServerWithResolverAndContinuationAuth(t *testing.T, resolver intake.ProjectContextResolver, authService httpserver.AuthService) testServerDeps {
	t.Helper()

	intakeStore := newFakeIntakeStore()
	goalStore := newFakeGoalStore()
	clarificationStore := newFakeClarificationStore()
	answerStore := newFakeClarificationAnswerStore()
	contractStore := newFakeContractStore()
	contractSeedStore := newFakeContractSeedStore()
	contractDraftStore := newFakeContractDraftStore()
	approvedContractStore := newFakeApprovedContractStore()
	workItemStore := newFakeWorkItemStore()
	workItemPlanStore := newFakeWorkItemPlanStore()
	workItemLeaseStore := newFakeWorkItemPlanLeaseStore(workItemPlanStore)
	workItemPlanProposalStore := newFakeWorkItemPlanProposalStore()
	repoBindingStore := newFakeRepoBindingStore()
	checkoutJobStore := newFakeCheckoutJobStore()
	checkoutReceiptStore := newFakeCheckoutReceiptStore()
	executionJobStore := newFakeExecutionJobStore()
	runStore := newFakeRunStore()
	commandPlanStore := newFakeExecutionCommandPlanStore()
	executionReceiptStore := newFakeExecutionReceiptStore()
	events := newFakeEventLog()
	ids := &sequenceIDs{}
	txRunner := &fakeTransactionRunner{}
	service := intake.NewService(intakeStore, resolver, events, txRunner, fixedClock{now: testTime()}, ids)
	intakeHandler := httpserver.NewIntakeHandler(service)
	goalService := goal.NewService(intakeStore, goalStore, events, txRunner, fixedClock{now: testTime()}, ids)
	goalHandler := httpserver.NewGoalHandler(goalService)
	clarificationService := clarification.NewService(goalStore, clarificationStore, answerStore, events, txRunner, fixedClock{now: testTime()}, ids)
	clarificationHandler := httpserver.NewClarificationHandler(clarificationService)
	continuationService := continuation.NewService(goalStore, goalService, clarificationService)
	continuationHandler := httpserver.NewContinuationHandler(authService, continuationService)
	contractSeedService := contractseed.NewService(goalStore, contractStore, contractSeedStore, events, txRunner, fixedClock{now: testTime()}, ids)
	contractDraftService := contractdraft.NewService(contractSeedStore, contractStore, contractDraftStore, events, txRunner, fixedClock{now: testTime()}, ids)
	approvedContractService := approvedcontract.NewService(contractDraftStore, contractStore, approvedContractStore, events, txRunner, fixedClock{now: testTime()}, ids)
	contractService := contractsvc.NewService(goalStore, contractStore, contractSeedService, contractDraftService, contractDraftStore, approvedContractService, txRunner)
	contractHandler := httpserver.NewContractHandler(authService, contractService)
	workItemService := workitem.NewService(workItemStore)
	workItemHandler := httpserver.NewWorkItemHandler(workItemService)
	workItemPlanService := workitemplan.NewService(contractStore, approvedContractStore, workItemPlanStore, workItemLeaseStore, workItemPlanProposalStore, workItemStore, events, txRunner, fixedClock{now: testTime()}, ids)
	workItemPlanHandler := httpserver.NewWorkItemPlanHandler(authService, workItemPlanService)
	checkoutService := checkout.NewService(workItemStore, repoBindingStore, checkoutJobStore, checkoutReceiptStore, events, txRunner, fixedClock{now: testTime()}, ids)
	checkoutHandler := httpserver.NewCheckoutHandler(authService, checkoutService)
	executionService := execution.NewService(workItemStore, repoBindingStore, checkoutReceiptStore, checkoutJobStore, executionJobStore, runStore, commandPlanStore, executionReceiptStore, events, txRunner, fixedClock{now: testTime()}, ids)
	executionHandler := httpserver.NewExecutionHandler(authService, executionService)

	return testServerDeps{
		router:            baseHandlers(intakeHandler, goalHandler, clarificationHandler, continuationHandler, contractHandler, workItemHandler, workItemPlanHandler, checkoutHandler, executionHandler),
		intakes:           intakeStore,
		goals:             goalStore,
		clarifications:    clarificationStore,
		answers:           answerStore,
		contracts:         contractStore,
		contractSeeds:     contractSeedStore,
		contractDrafts:    contractDraftStore,
		approvedContracts: approvedContractStore,
		workItems:         workItemStore,
		workItemPlans:     workItemPlanStore,
		workItemLeases:    workItemLeaseStore,
		workItemProposals: workItemPlanProposalStore,
		repoBindings:      repoBindingStore,
		checkoutJobs:      checkoutJobStore,
		checkoutReceipts:  checkoutReceiptStore,
		executionJobs:     executionJobStore,
		runs:              runStore,
		commandPlans:      commandPlanStore,
		executionReceipts: executionReceiptStore,
		events:            events,
		idFactory:         ids,
	}
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

type fakeGoalStore struct {
	goals    map[spine.GoalID]spine.Goal
	byIntake map[spine.IntakeID]spine.GoalID
}

func newFakeGoalStore() *fakeGoalStore {
	return &fakeGoalStore{
		goals:    map[spine.GoalID]spine.Goal{},
		byIntake: map[spine.IntakeID]spine.GoalID{},
	}
}

func (s *fakeGoalStore) Create(_ context.Context, goal spine.Goal) error {
	s.goals[goal.ID] = cloneGoal(goal)
	s.byIntake[goal.IntakeID] = goal.ID
	return nil
}

func (s *fakeGoalStore) Get(_ context.Context, id spine.GoalID) (spine.Goal, bool, error) {
	goal, ok := s.goals[id]
	return cloneGoal(goal), ok, nil
}

func (s *fakeGoalStore) GetByIntakeID(_ context.Context, id spine.IntakeID) (spine.Goal, bool, error) {
	goalID, ok := s.byIntake[id]
	if !ok {
		return spine.Goal{}, false, nil
	}
	goal, ok := s.goals[goalID]
	return cloneGoal(goal), ok, nil
}

func (s *fakeGoalStore) UpdateReadiness(_ context.Context, id spine.GoalID, state spine.GoalState, reasonCodes []spine.GoalReadinessReasonCode) (spine.Goal, bool, error) {
	goal, ok := s.goals[id]
	if !ok {
		return spine.Goal{}, false, nil
	}
	goal.State = state
	goal.LastReadinessReasonCodes = append([]spine.GoalReadinessReasonCode(nil), reasonCodes...)
	s.goals[id] = cloneGoal(goal)
	return cloneGoal(goal), true, nil
}

func (s *fakeGoalStore) UpdateHints(_ context.Context, id spine.GoalID, update spine.GoalHintUpdate) (spine.Goal, bool, error) {
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
	requests   map[spine.ClarificationRequestID]spine.ClarificationRequest
	openByGoal map[spine.GoalID]spine.ClarificationRequestID
}

func newFakeClarificationStore() *fakeClarificationStore {
	return &fakeClarificationStore{
		requests:   map[spine.ClarificationRequestID]spine.ClarificationRequest{},
		openByGoal: map[spine.GoalID]spine.ClarificationRequestID{},
	}
}

func (s *fakeClarificationStore) Create(_ context.Context, request spine.ClarificationRequest) error {
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

func (s *fakeClarificationStore) UpdateState(_ context.Context, id spine.ClarificationRequestID, state spine.ClarificationRequestState) (spine.ClarificationRequest, bool, error) {
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
	answers   map[spine.ClarificationAnswerID]spine.ClarificationAnswer
	byRequest map[spine.ClarificationRequestID]spine.ClarificationAnswerID
	applied   map[spine.ClarificationAnswerID]bool
}

func newFakeClarificationAnswerStore() *fakeClarificationAnswerStore {
	return &fakeClarificationAnswerStore{
		answers:   map[spine.ClarificationAnswerID]spine.ClarificationAnswer{},
		byRequest: map[spine.ClarificationRequestID]spine.ClarificationAnswerID{},
		applied:   map[spine.ClarificationAnswerID]bool{},
	}
}

func (s *fakeClarificationAnswerStore) Create(_ context.Context, answer spine.ClarificationAnswer) error {
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

func (s *fakeClarificationAnswerStore) MarkApplied(_ context.Context, id spine.ClarificationAnswerID, _ spine.ActorRef, _ time.Time) (bool, error) {
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

type fakeContractStore struct {
	contracts map[spine.ContractID]spine.Contract
	byGoal    map[spine.GoalID]spine.ContractID
}

func newFakeContractStore() *fakeContractStore {
	return &fakeContractStore{
		contracts: map[spine.ContractID]spine.Contract{},
		byGoal:    map[spine.GoalID]spine.ContractID{},
	}
}

func (s *fakeContractStore) Create(_ context.Context, contract spine.Contract) error {
	s.contracts[contract.ID] = contract
	s.byGoal[contract.GoalID] = contract.ID
	return nil
}

func (s *fakeContractStore) Get(_ context.Context, id spine.ContractID) (spine.Contract, bool, error) {
	contract, ok := s.contracts[id]
	return contract, ok, nil
}

func (s *fakeContractStore) GetByGoalID(_ context.Context, id spine.GoalID) (spine.Contract, bool, error) {
	contractID, ok := s.byGoal[id]
	if !ok {
		return spine.Contract{}, false, nil
	}
	contract, ok := s.contracts[contractID]
	return contract, ok, nil
}

func (s *fakeContractStore) List(_ context.Context, filter spine.ContractListFilter) ([]spine.Contract, error) {
	contracts := make([]spine.Contract, 0, len(s.contracts))
	for _, contract := range s.contracts {
		if filter.OrganizationID != "" && contract.OrganizationID != filter.OrganizationID {
			continue
		}
		if filter.ProjectID != "" && contract.ProjectID != filter.ProjectID {
			continue
		}
		if filter.RepoBindingID != "" && contract.RepoBindingID != filter.RepoBindingID {
			continue
		}
		if filter.GoalID != "" && contract.GoalID != filter.GoalID {
			continue
		}
		if filter.State != "" && contract.State != filter.State {
			continue
		}
		contracts = append(contracts, cloneContract(contract))
	}
	sort.Slice(contracts, func(i, j int) bool {
		if !contracts[i].UpdatedAt.Equal(contracts[j].UpdatedAt) {
			return contracts[i].UpdatedAt.After(contracts[j].UpdatedAt)
		}
		if !contracts[i].CreatedAt.Equal(contracts[j].CreatedAt) {
			return contracts[i].CreatedAt.After(contracts[j].CreatedAt)
		}
		return contracts[i].ID > contracts[j].ID
	})
	if filter.Limit > 0 && len(contracts) > filter.Limit {
		contracts = contracts[:filter.Limit]
	}
	return contracts, nil
}

func cloneContract(contract spine.Contract) spine.Contract {
	if contract.CurrentSeedID != nil {
		value := *contract.CurrentSeedID
		contract.CurrentSeedID = &value
	}
	if contract.CurrentDraftID != nil {
		value := *contract.CurrentDraftID
		contract.CurrentDraftID = &value
	}
	if contract.ApprovedSnapshotID != nil {
		value := *contract.ApprovedSnapshotID
		contract.ApprovedSnapshotID = &value
	}
	return contract
}

func (s *fakeContractStore) Delete(_ context.Context, id spine.ContractID) error {
	contract, ok := s.contracts[id]
	if !ok {
		return nil
	}
	delete(s.contracts, id)
	if s.byGoal[contract.GoalID] == id {
		delete(s.byGoal, contract.GoalID)
	}
	return nil
}

func (s *fakeContractStore) MarkDraftCreated(_ context.Context, id spine.ContractID, draftID spine.ContractDraftID, updatedAt time.Time) error {
	contract, ok := s.contracts[id]
	if !ok {
		return nil
	}
	contract.State = spine.ContractStateDraft
	contract.CurrentDraftID = &draftID
	contract.UpdatedAt = updatedAt.UTC()
	s.contracts[id] = contract
	return nil
}

func (s *fakeContractStore) MarkReadyForApproval(_ context.Context, id spine.ContractID, updatedAt time.Time) error {
	contract, ok := s.contracts[id]
	if !ok {
		return nil
	}
	contract.State = spine.ContractStateReadyForApproval
	contract.UpdatedAt = updatedAt.UTC()
	s.contracts[id] = contract
	return nil
}

func (s *fakeContractStore) MarkApproved(_ context.Context, id spine.ContractID, approvedID spine.ApprovedContractID, updatedAt time.Time) error {
	contract, ok := s.contracts[id]
	if !ok {
		return nil
	}
	contract.State = spine.ContractStateApproved
	contract.ApprovedSnapshotID = &approvedID
	contract.UpdatedAt = updatedAt.UTC()
	s.contracts[id] = contract
	return nil
}

type fakeContractSeedStore struct {
	seeds  map[spine.ContractSeedID]spine.ContractSeed
	byGoal map[spine.GoalID]spine.ContractSeedID
}

func newFakeContractSeedStore() *fakeContractSeedStore {
	return &fakeContractSeedStore{
		seeds:  map[spine.ContractSeedID]spine.ContractSeed{},
		byGoal: map[spine.GoalID]spine.ContractSeedID{},
	}
}

func (s *fakeContractSeedStore) Create(_ context.Context, seed spine.ContractSeed) error {
	s.seeds[seed.ID] = seed
	s.byGoal[seed.GoalID] = seed.ID
	return nil
}

func (s *fakeContractSeedStore) Get(_ context.Context, id spine.ContractSeedID) (spine.ContractSeed, bool, error) {
	seed, ok := s.seeds[id]
	return seed, ok, nil
}

func (s *fakeContractSeedStore) GetByGoalID(_ context.Context, id spine.GoalID) (spine.ContractSeed, bool, error) {
	seedID, ok := s.byGoal[id]
	if !ok {
		return spine.ContractSeed{}, false, nil
	}
	seed, ok := s.seeds[seedID]
	return seed, ok, nil
}

func (s *fakeContractSeedStore) Delete(_ context.Context, id spine.ContractSeedID) error {
	seed, ok := s.seeds[id]
	if !ok {
		return nil
	}
	delete(s.seeds, id)
	if s.byGoal[seed.GoalID] == id {
		delete(s.byGoal, seed.GoalID)
	}
	return nil
}

type fakeContractDraftStore struct {
	drafts map[spine.ContractDraftID]spine.ContractDraft
	bySeed map[spine.ContractSeedID]spine.ContractDraftID
}

func newFakeContractDraftStore() *fakeContractDraftStore {
	return &fakeContractDraftStore{
		drafts: map[spine.ContractDraftID]spine.ContractDraft{},
		bySeed: map[spine.ContractSeedID]spine.ContractDraftID{},
	}
}

func (s *fakeContractDraftStore) Create(_ context.Context, draft spine.ContractDraft) error {
	s.drafts[draft.ID] = draft
	s.bySeed[draft.ContractSeedID] = draft.ID
	return nil
}

func (s *fakeContractDraftStore) Update(_ context.Context, draft spine.ContractDraft) error {
	existing, ok := s.drafts[draft.ID]
	if !ok {
		return nil
	}
	draft.ContractID = existing.ContractID
	draft.ContractSeedID = existing.ContractSeedID
	draft.GoalID = existing.GoalID
	draft.RepoBindingID = existing.RepoBindingID
	draft.State = existing.State
	draft.CreatedAt = existing.CreatedAt
	s.drafts[draft.ID] = draft
	return nil
}

func (s *fakeContractDraftStore) MarkReadyForApproval(_ context.Context, draft spine.ContractDraft) error {
	existing, ok := s.drafts[draft.ID]
	if !ok {
		return nil
	}
	existing.State = spine.ContractDraftStateReadyForApproval
	s.drafts[draft.ID] = existing
	return nil
}

func (s *fakeContractDraftStore) Get(_ context.Context, id spine.ContractDraftID) (spine.ContractDraft, bool, error) {
	draft, ok := s.drafts[id]
	return draft, ok, nil
}

func (s *fakeContractDraftStore) GetByContractSeedID(_ context.Context, id spine.ContractSeedID) (spine.ContractDraft, bool, error) {
	draftID, ok := s.bySeed[id]
	if !ok {
		return spine.ContractDraft{}, false, nil
	}
	draft, ok := s.drafts[draftID]
	return draft, ok, nil
}

type fakeApprovedContractStore struct {
	approved map[spine.ApprovedContractID]spine.ApprovedContract
	byDraft  map[spine.ContractDraftID]spine.ApprovedContractID
}

func newFakeApprovedContractStore() *fakeApprovedContractStore {
	return &fakeApprovedContractStore{
		approved: map[spine.ApprovedContractID]spine.ApprovedContract{},
		byDraft:  map[spine.ContractDraftID]spine.ApprovedContractID{},
	}
}

func (s *fakeApprovedContractStore) Create(_ context.Context, approved spine.ApprovedContract) error {
	s.approved[approved.ID] = approved
	s.byDraft[approved.ContractDraftID] = approved.ID
	return nil
}

func (s *fakeApprovedContractStore) Get(_ context.Context, id spine.ApprovedContractID) (spine.ApprovedContract, bool, error) {
	approved, ok := s.approved[id]
	return approved, ok, nil
}

func (s *fakeApprovedContractStore) GetByContractDraftID(_ context.Context, id spine.ContractDraftID) (spine.ApprovedContract, bool, error) {
	approvedID, ok := s.byDraft[id]
	if !ok {
		return spine.ApprovedContract{}, false, nil
	}
	approved, ok := s.approved[approvedID]
	return approved, ok, nil
}

type fakeWorkItemStore struct {
	items              map[spine.WorkItemID]spine.WorkItem
	byApprovedContract map[spine.ApprovedContractID][]spine.WorkItemID
}

func newFakeWorkItemStore() *fakeWorkItemStore {
	return &fakeWorkItemStore{
		items:              map[spine.WorkItemID]spine.WorkItem{},
		byApprovedContract: map[spine.ApprovedContractID][]spine.WorkItemID{},
	}
}

func (s *fakeWorkItemStore) Create(_ context.Context, item spine.WorkItem) error {
	s.items[item.ID] = item
	s.byApprovedContract[item.ApprovedContractID] = append(s.byApprovedContract[item.ApprovedContractID], item.ID)
	return nil
}

func (s *fakeWorkItemStore) Get(_ context.Context, id spine.WorkItemID) (spine.WorkItem, bool, error) {
	item, ok := s.items[id]
	return item, ok, nil
}

func (s *fakeWorkItemStore) GetByApprovedContractID(_ context.Context, id spine.ApprovedContractID) (spine.WorkItem, bool, error) {
	itemIDs := s.byApprovedContract[id]
	if len(itemIDs) == 0 {
		return spine.WorkItem{}, false, nil
	}
	item, ok := s.items[itemIDs[0]]
	return item, ok, nil
}

type fakeWorkItemPlanStore struct {
	plans      map[spine.WorkItemPlanID]spine.WorkItemPlan
	byContract map[spine.ContractID]spine.WorkItemPlanID
}

func newFakeWorkItemPlanStore() *fakeWorkItemPlanStore {
	return &fakeWorkItemPlanStore{
		plans:      map[spine.WorkItemPlanID]spine.WorkItemPlan{},
		byContract: map[spine.ContractID]spine.WorkItemPlanID{},
	}
}

func (s *fakeWorkItemPlanStore) Create(_ context.Context, plan spine.WorkItemPlan) error {
	s.plans[plan.ID] = plan
	s.byContract[plan.ContractID] = plan.ID
	return nil
}

func (s *fakeWorkItemPlanStore) Get(_ context.Context, id spine.WorkItemPlanID) (spine.WorkItemPlan, bool, error) {
	plan, ok := s.plans[id]
	return plan, ok, nil
}

func (s *fakeWorkItemPlanStore) GetByContractID(_ context.Context, id spine.ContractID) (spine.WorkItemPlan, bool, error) {
	planID, ok := s.byContract[id]
	if !ok {
		return spine.WorkItemPlan{}, false, nil
	}
	plan, ok := s.plans[planID]
	return plan, ok, nil
}

func (s *fakeWorkItemPlanStore) MarkProposalSubmitted(_ context.Context, id spine.WorkItemPlanID, updatedAt time.Time) error {
	plan := s.plans[id]
	plan.State = spine.WorkItemPlanStateProposalSubmitted
	plan.UpdatedAt = updatedAt.UTC()
	s.plans[id] = plan
	return nil
}

func (s *fakeWorkItemPlanStore) MarkAccepted(_ context.Context, id spine.WorkItemPlanID, updatedAt time.Time) error {
	plan := s.plans[id]
	plan.State = spine.WorkItemPlanStateAccepted
	plan.UpdatedAt = updatedAt.UTC()
	s.plans[id] = plan
	return nil
}

type fakeWorkItemPlanLeaseStore struct {
	plans  *fakeWorkItemPlanStore
	leases map[spine.WorkItemPlanLeaseID]spine.WorkItemPlanLease
}

func newFakeWorkItemPlanLeaseStore(plans *fakeWorkItemPlanStore) *fakeWorkItemPlanLeaseStore {
	return &fakeWorkItemPlanLeaseStore{
		plans:  plans,
		leases: map[spine.WorkItemPlanLeaseID]spine.WorkItemPlanLease{},
	}
}

func (s *fakeWorkItemPlanLeaseStore) AcquireNextLease(_ context.Context, input workitemplan.LeaseAcquireInput) (spine.WorkItemPlanLease, bool, error) {
	var selected spine.WorkItemPlan
	found := false
	for _, plan := range s.plans.plans {
		if plan.State == spine.WorkItemPlanStateQueued || (plan.State == spine.WorkItemPlanStateLeased && plan.LeaseExpiresAt != nil && !plan.LeaseExpiresAt.After(input.CreatedAt)) {
			if !found || plan.CreatedAt.Before(selected.CreatedAt) || (plan.CreatedAt.Equal(selected.CreatedAt) && plan.ID < selected.ID) {
				selected = plan
				found = true
			}
		}
	}
	if !found {
		return spine.WorkItemPlanLease{}, false, nil
	}
	if selected.State == spine.WorkItemPlanStateLeased && selected.CurrentLeaseID != nil {
		previous := s.leases[*selected.CurrentLeaseID]
		previous.State = spine.WorkItemPlanLeaseStateExpired
		previous.UpdatedAt = input.UpdatedAt
		s.leases[previous.ID] = previous
	}
	lease := spine.WorkItemPlanLease{
		ID:                 input.ID,
		PlanID:             selected.ID,
		ContractID:         selected.ContractID,
		ApprovedContractID: selected.ApprovedContractID,
		RepoBindingID:      selected.RepoBindingID,
		LeasedBy:           input.LeasedBy,
		State:              spine.WorkItemPlanLeaseStateActive,
		LeaseTokenHash:     input.LeaseTokenHash,
		ExpiresAt:          input.ExpiresAt,
		CreatedAt:          input.CreatedAt,
		UpdatedAt:          input.UpdatedAt,
	}
	s.leases[lease.ID] = lease
	selected.State = spine.WorkItemPlanStateLeased
	selected.CurrentLeaseID = &lease.ID
	selected.LeasedBy = &lease.LeasedBy
	selected.LeaseExpiresAt = &lease.ExpiresAt
	selected.UpdatedAt = input.UpdatedAt
	s.plans.plans[selected.ID] = selected
	return lease, true, nil
}

func (s *fakeWorkItemPlanLeaseStore) Get(_ context.Context, id spine.WorkItemPlanLeaseID) (spine.WorkItemPlanLease, bool, error) {
	lease, ok := s.leases[id]
	return lease, ok, nil
}

func (s *fakeWorkItemPlanLeaseStore) Renew(_ context.Context, id spine.WorkItemPlanLeaseID, tokenHash string, expiresAt time.Time, updatedAt time.Time) (spine.WorkItemPlanLease, bool, error) {
	lease, ok := s.leases[id]
	if !ok || lease.LeaseTokenHash != tokenHash || lease.State != spine.WorkItemPlanLeaseStateActive || !lease.ExpiresAt.After(updatedAt) {
		return spine.WorkItemPlanLease{}, false, nil
	}
	lease.ExpiresAt = expiresAt
	lease.UpdatedAt = updatedAt
	s.leases[id] = lease
	plan := s.plans.plans[lease.PlanID]
	plan.LeaseExpiresAt = &lease.ExpiresAt
	plan.UpdatedAt = updatedAt
	s.plans.plans[plan.ID] = plan
	return lease, true, nil
}

func (s *fakeWorkItemPlanLeaseStore) MarkCompleted(_ context.Context, id spine.WorkItemPlanLeaseID, tokenHash string, completedAt time.Time) (bool, error) {
	lease, ok := s.leases[id]
	if !ok || lease.LeaseTokenHash != tokenHash || lease.State != spine.WorkItemPlanLeaseStateActive || !lease.ExpiresAt.After(completedAt) {
		return false, nil
	}
	lease.State = spine.WorkItemPlanLeaseStateCompleted
	lease.UpdatedAt = completedAt
	s.leases[id] = lease
	return true, nil
}

type fakeWorkItemPlanProposalStore struct {
	proposals map[spine.WorkItemPlanProposalID]spine.WorkItemPlanProposal
	byPlan    map[spine.WorkItemPlanID]spine.WorkItemPlanProposalID
}

func newFakeWorkItemPlanProposalStore() *fakeWorkItemPlanProposalStore {
	return &fakeWorkItemPlanProposalStore{
		proposals: map[spine.WorkItemPlanProposalID]spine.WorkItemPlanProposal{},
		byPlan:    map[spine.WorkItemPlanID]spine.WorkItemPlanProposalID{},
	}
}

func (s *fakeWorkItemPlanProposalStore) Create(_ context.Context, proposal spine.WorkItemPlanProposal) error {
	s.proposals[proposal.ID] = proposal
	s.byPlan[proposal.PlanID] = proposal.ID
	return nil
}

func (s *fakeWorkItemPlanProposalStore) Get(_ context.Context, id spine.WorkItemPlanProposalID) (spine.WorkItemPlanProposal, bool, error) {
	proposal, ok := s.proposals[id]
	return proposal, ok, nil
}

func (s *fakeWorkItemPlanProposalStore) GetByPlanID(_ context.Context, id spine.WorkItemPlanID) (spine.WorkItemPlanProposal, bool, error) {
	proposalID, ok := s.byPlan[id]
	if !ok {
		return spine.WorkItemPlanProposal{}, false, nil
	}
	proposal, ok := s.proposals[proposalID]
	return proposal, ok, nil
}

func (s *fakeWorkItemPlanProposalStore) MarkAccepted(_ context.Context, id spine.WorkItemPlanProposalID, acceptedBy spine.ActorRef, acceptedAt time.Time) error {
	proposal := s.proposals[id]
	proposal.State = spine.WorkItemProposalStateAccepted
	proposal.AcceptedBy = &acceptedBy
	acceptedAt = acceptedAt.UTC()
	proposal.AcceptedAt = &acceptedAt
	proposal.UpdatedAt = acceptedAt
	s.proposals[id] = proposal
	return nil
}

type fakeRepoBindingStore struct {
	bindings map[spine.RepoBindingID]spine.RepoBinding
}

func newFakeRepoBindingStore() *fakeRepoBindingStore {
	binding := spine.RepoBinding{
		ID:                 "018f0000-0000-7000-8000-000000000004",
		OrganizationID:     "018f0000-0000-7000-8000-000000000002",
		ProjectID:          "018f0000-0000-7000-8000-000000000003",
		CreatedByUserID:    "018f0000-0000-7000-8000-000000000001",
		Provider:           "github",
		RepositoryFullName: "heurema/goalrail",
		RepositoryURL:      "https://github.com/heurema/goalrail",
		DefaultBranch:      "main",
		WorkflowBaseBranch: "main",
		PathScope:          ".",
		AccessMode:         spine.RepoBindingAccessModeCustomerMountedWorkspace,
		State:              spine.EntityStateActive,
		CreatedAt:          testTime(),
		UpdatedAt:          testTime(),
	}
	return &fakeRepoBindingStore{bindings: map[spine.RepoBindingID]spine.RepoBinding{binding.ID: binding}}
}

func (s *fakeRepoBindingStore) GetRepoBinding(_ context.Context, id spine.RepoBindingID) (spine.RepoBinding, bool, error) {
	binding, ok := s.bindings[id]
	return binding, ok, nil
}

type fakeCheckoutJobStore struct {
	jobs   map[spine.CheckoutJobID]spine.CheckoutJob
	byTask map[spine.WorkItemID]spine.CheckoutJobID
}

func newFakeCheckoutJobStore() *fakeCheckoutJobStore {
	return &fakeCheckoutJobStore{
		jobs:   map[spine.CheckoutJobID]spine.CheckoutJob{},
		byTask: map[spine.WorkItemID]spine.CheckoutJobID{},
	}
}

func (s *fakeCheckoutJobStore) Create(_ context.Context, job spine.CheckoutJob) error {
	s.jobs[job.ID] = job
	s.byTask[job.TaskID] = job.ID
	return nil
}

func (s *fakeCheckoutJobStore) Get(_ context.Context, id spine.CheckoutJobID) (spine.CheckoutJob, bool, error) {
	if id == "malformed-id" {
		return spine.CheckoutJob{}, false, spine.MalformedIDError{Field: "checkout job id", Reason: "must be uuid"}
	}
	job, ok := s.jobs[id]
	return job, ok, nil
}

func (s *fakeCheckoutJobStore) GetByTaskID(_ context.Context, id spine.WorkItemID) (spine.CheckoutJob, bool, error) {
	jobID, ok := s.byTask[id]
	if !ok {
		return spine.CheckoutJob{}, false, nil
	}
	job, ok := s.jobs[jobID]
	return job, ok, nil
}

func (s *fakeCheckoutJobStore) AcquireNextLease(_ context.Context, input checkout.JobLeaseInput) (spine.CheckoutJob, bool, error) {
	var selected spine.CheckoutJob
	found := false
	for _, job := range s.jobs {
		if input.OrganizationID != "" && job.OrganizationID != input.OrganizationID {
			continue
		}
		if input.ProjectID != "" && job.ProjectID != input.ProjectID {
			continue
		}
		if input.RepoBindingID != "" && job.RepoBindingID != input.RepoBindingID {
			continue
		}
		if job.State == spine.CheckoutJobStateQueued || (job.State == spine.CheckoutJobStateLeased && job.LeaseExpiresAt != nil && !job.LeaseExpiresAt.After(input.UpdatedAt)) {
			if !found || job.CreatedAt.Before(selected.CreatedAt) || (job.CreatedAt.Equal(selected.CreatedAt) && job.ID < selected.ID) {
				selected = job
				found = true
			}
		}
	}
	if !found {
		return spine.CheckoutJob{}, false, nil
	}
	selected.State = spine.CheckoutJobStateLeased
	selected.CurrentRunnerID = input.RunnerID
	selected.LeaseTokenHash = input.LeaseTokenHash
	selected.LeaseExpiresAt = &input.LeaseExpiresAt
	selected.UpdatedAt = input.UpdatedAt
	s.jobs[selected.ID] = selected
	return selected, true, nil
}

func (s *fakeCheckoutJobStore) MarkReceiptSubmitted(_ context.Context, id spine.CheckoutJobID, runnerID string, tokenHash string, updatedAt time.Time) (bool, error) {
	job, ok := s.jobs[id]
	if !ok || job.State != spine.CheckoutJobStateLeased || job.CurrentRunnerID != runnerID || job.LeaseTokenHash != tokenHash || job.LeaseExpiresAt == nil || !job.LeaseExpiresAt.After(updatedAt) {
		return false, nil
	}
	job.State = spine.CheckoutJobStateReceiptSubmitted
	job.UpdatedAt = updatedAt
	s.jobs[id] = job
	return true, nil
}

type fakeCheckoutReceiptStore struct {
	receipts map[spine.CheckoutReceiptID]spine.CheckoutReceipt
	byJob    map[spine.CheckoutJobID]spine.CheckoutReceiptID
}

func newFakeCheckoutReceiptStore() *fakeCheckoutReceiptStore {
	return &fakeCheckoutReceiptStore{
		receipts: map[spine.CheckoutReceiptID]spine.CheckoutReceipt{},
		byJob:    map[spine.CheckoutJobID]spine.CheckoutReceiptID{},
	}
}

func (s *fakeCheckoutReceiptStore) Create(_ context.Context, receipt spine.CheckoutReceipt) error {
	s.receipts[receipt.ID] = receipt
	s.byJob[receipt.JobID] = receipt.ID
	return nil
}

func (s *fakeCheckoutReceiptStore) Get(_ context.Context, id spine.CheckoutReceiptID) (spine.CheckoutReceipt, bool, error) {
	receipt, ok := s.receipts[id]
	return receipt, ok, nil
}

type fakeExecutionJobStore struct {
	jobs  map[spine.ExecutionJobID]spine.ExecutionJob
	byKey map[string]spine.ExecutionJobID
}

func newFakeExecutionJobStore() *fakeExecutionJobStore {
	return &fakeExecutionJobStore{
		jobs:  map[spine.ExecutionJobID]spine.ExecutionJob{},
		byKey: map[string]spine.ExecutionJobID{},
	}
}

func (s *fakeExecutionJobStore) Create(_ context.Context, job spine.ExecutionJob) error {
	key := executionJobKey(job.TaskID, job.CheckoutReceiptID)
	if _, ok := s.byKey[key]; ok {
		return fmt.Errorf("execution job already prepared")
	}
	s.jobs[job.ID] = job
	s.byKey[key] = job.ID
	return nil
}

func (s *fakeExecutionJobStore) Get(_ context.Context, id spine.ExecutionJobID) (spine.ExecutionJob, bool, error) {
	job, ok := s.jobs[id]
	return job, ok, nil
}

func (s *fakeExecutionJobStore) GetByTaskAndCheckoutReceipt(_ context.Context, taskID spine.WorkItemID, receiptID spine.CheckoutReceiptID) (spine.ExecutionJob, bool, error) {
	jobID, ok := s.byKey[executionJobKey(taskID, receiptID)]
	if !ok {
		return spine.ExecutionJob{}, false, nil
	}
	job, ok := s.jobs[jobID]
	return job, ok, nil
}

func (s *fakeExecutionJobStore) AcquireNextLease(_ context.Context, input execution.JobLeaseInput) (spine.ExecutionLease, spine.ExecutionJob, bool, error) {
	var selected spine.ExecutionJob
	found := false
	for _, job := range s.jobs {
		if input.OrganizationID != "" && job.OrganizationID != input.OrganizationID {
			continue
		}
		if input.ProjectID != "" && job.ProjectID != input.ProjectID {
			continue
		}
		if input.RepoBindingID != "" && job.RepoBindingID != input.RepoBindingID {
			continue
		}
		if job.State == spine.ExecutionJobStateQueued || ((job.State == spine.ExecutionJobStateLeased || job.State == spine.ExecutionJobStateRunStarted) && job.LeaseExpiresAt != nil && !job.LeaseExpiresAt.After(input.UpdatedAt)) {
			if !found || job.CreatedAt.Before(selected.CreatedAt) || (job.CreatedAt.Equal(selected.CreatedAt) && job.ID < selected.ID) {
				selected = job
				found = true
			}
		}
	}
	if !found {
		return spine.ExecutionLease{}, spine.ExecutionJob{}, false, nil
	}
	lease := spine.ExecutionLease{
		ID:                input.ID,
		ExecutionJobID:    selected.ID,
		TaskID:            selected.TaskID,
		CheckoutReceiptID: selected.CheckoutReceiptID,
		RepoBindingID:     selected.RepoBindingID,
		RunnerID:          input.RunnerID,
		State:             spine.ExecutionLeaseStateActive,
		LeaseTokenHash:    input.LeaseTokenHash,
		ExpiresAt:         input.ExpiresAt,
		CreatedAt:         input.CreatedAt,
		UpdatedAt:         input.UpdatedAt,
	}
	if selected.State != spine.ExecutionJobStateRunStarted {
		selected.State = spine.ExecutionJobStateLeased
	}
	selected.CurrentLeaseID = &lease.ID
	selected.CurrentRunnerID = input.RunnerID
	selected.LeaseTokenHash = input.LeaseTokenHash
	selected.LeaseExpiresAt = &input.ExpiresAt
	selected.UpdatedAt = input.UpdatedAt
	s.jobs[selected.ID] = selected
	return lease, selected, true, nil
}

func (s *fakeExecutionJobStore) MarkRunStarted(_ context.Context, id spine.ExecutionJobID, leaseID spine.ExecutionLeaseID, runnerID string, tokenHash string, updatedAt time.Time) (bool, error) {
	job, ok := s.jobs[id]
	if !ok || job.State != spine.ExecutionJobStateLeased || job.CurrentLeaseID == nil || *job.CurrentLeaseID != leaseID || job.CurrentRunnerID != runnerID || job.LeaseTokenHash != tokenHash || job.LeaseExpiresAt == nil || !job.LeaseExpiresAt.After(updatedAt) {
		return false, nil
	}
	job.State = spine.ExecutionJobStateRunStarted
	job.UpdatedAt = updatedAt
	s.jobs[id] = job
	return true, nil
}

func (s *fakeExecutionJobStore) MarkReceiptSubmitted(_ context.Context, id spine.ExecutionJobID, updatedAt time.Time) (bool, error) {
	job, ok := s.jobs[id]
	if !ok || job.State != spine.ExecutionJobStateRunStarted {
		return false, nil
	}
	job.State = spine.ExecutionJobStateReceiptSubmitted
	job.UpdatedAt = updatedAt
	s.jobs[id] = job
	return true, nil
}

func executionJobKey(taskID spine.WorkItemID, receiptID spine.CheckoutReceiptID) string {
	return string(taskID) + "\x00" + string(receiptID)
}

type fakeRunStore struct {
	runs  map[spine.RunID]spine.Run
	byJob map[spine.ExecutionJobID]spine.RunID
}

func newFakeRunStore() *fakeRunStore {
	return &fakeRunStore{
		runs:  map[spine.RunID]spine.Run{},
		byJob: map[spine.ExecutionJobID]spine.RunID{},
	}
}

func (s *fakeRunStore) Create(_ context.Context, run spine.Run) error {
	if _, ok := s.byJob[run.ExecutionJobID]; ok {
		return fmt.Errorf("run already started")
	}
	s.runs[run.ID] = run
	s.byJob[run.ExecutionJobID] = run.ID
	return nil
}

func (s *fakeRunStore) GetByExecutionLease(_ context.Context, leaseID spine.ExecutionLeaseID) (spine.Run, bool, error) {
	for _, run := range s.runs {
		if run.ExecutionLeaseID == leaseID {
			return run, true, nil
		}
	}
	return spine.Run{}, false, nil
}

func (s *fakeRunStore) GetByExecutionJob(_ context.Context, jobID spine.ExecutionJobID) (spine.Run, bool, error) {
	runID, ok := s.byJob[jobID]
	if !ok {
		return spine.Run{}, false, nil
	}
	run, ok := s.runs[runID]
	return run, ok, nil
}

func (s *fakeRunStore) Get(_ context.Context, id spine.RunID) (spine.Run, bool, error) {
	run, ok := s.runs[id]
	return run, ok, nil
}

func (s *fakeRunStore) MarkReceiptSubmitted(_ context.Context, id spine.RunID, finishedAt time.Time, updatedAt time.Time) (bool, error) {
	run, ok := s.runs[id]
	if !ok || run.State != spine.RunStateStarted {
		return false, nil
	}
	finished := finishedAt.UTC()
	run.State = spine.RunStateReceiptSubmitted
	run.FinishedAt = &finished
	run.UpdatedAt = updatedAt
	s.runs[id] = run
	return true, nil
}

type fakeExecutionCommandPlanStore struct {
	plans map[spine.ExecutionCommandPlanID]spine.ExecutionCommandPlan
	byKey map[string]spine.ExecutionCommandPlanID
}

func newFakeExecutionCommandPlanStore() *fakeExecutionCommandPlanStore {
	return &fakeExecutionCommandPlanStore{
		plans: map[spine.ExecutionCommandPlanID]spine.ExecutionCommandPlan{},
		byKey: map[string]spine.ExecutionCommandPlanID{},
	}
}

func (s *fakeExecutionCommandPlanStore) Create(_ context.Context, plan spine.ExecutionCommandPlan) error {
	key := executionCommandPlanKey(plan.RunID, plan.CommandKind, plan.Action)
	if _, ok := s.byKey[key]; ok {
		return fmt.Errorf("execution command plan already planned")
	}
	s.plans[plan.ID] = plan
	s.byKey[key] = plan.ID
	return nil
}

func (s *fakeExecutionCommandPlanStore) Get(_ context.Context, id spine.ExecutionCommandPlanID) (spine.ExecutionCommandPlan, bool, error) {
	plan, ok := s.plans[id]
	return plan, ok, nil
}

func (s *fakeExecutionCommandPlanStore) GetByRunAndAction(_ context.Context, runID spine.RunID, kind string, action string) (spine.ExecutionCommandPlan, bool, error) {
	planID, ok := s.byKey[executionCommandPlanKey(runID, kind, action)]
	if !ok {
		return spine.ExecutionCommandPlan{}, false, nil
	}
	plan, ok := s.plans[planID]
	return plan, ok, nil
}

func executionCommandPlanKey(runID spine.RunID, kind string, action string) string {
	return string(runID) + "\x00" + kind + "\x00" + action
}

type fakeExecutionReceiptStore struct {
	receipts map[spine.ExecutionReceiptID]spine.ExecutionReceipt
	byRun    map[spine.RunID]spine.ExecutionReceiptID
}

func newFakeExecutionReceiptStore() *fakeExecutionReceiptStore {
	return &fakeExecutionReceiptStore{
		receipts: map[spine.ExecutionReceiptID]spine.ExecutionReceipt{},
		byRun:    map[spine.RunID]spine.ExecutionReceiptID{},
	}
}

func (s *fakeExecutionReceiptStore) Create(_ context.Context, receipt spine.ExecutionReceipt) error {
	if _, ok := s.byRun[receipt.RunID]; ok {
		return fmt.Errorf("execution receipt already submitted")
	}
	s.receipts[receipt.ID] = receipt
	s.byRun[receipt.RunID] = receipt.ID
	return nil
}

func (s *fakeExecutionReceiptStore) Get(_ context.Context, id spine.ExecutionReceiptID) (spine.ExecutionReceipt, bool, error) {
	receipt, ok := s.receipts[id]
	return receipt, ok, nil
}

func (s *fakeExecutionReceiptStore) GetByRun(_ context.Context, runID spine.RunID) (spine.ExecutionReceipt, bool, error) {
	receiptID, ok := s.byRun[runID]
	if !ok {
		return spine.ExecutionReceipt{}, false, nil
	}
	receipt, ok := s.receipts[receiptID]
	return receipt, ok, nil
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

func continuationAuthProfile(organizationID spine.OrganizationID) auth.Profile {
	return auth.Profile{
		User: spine.User{
			ID:          "018f0000-0000-7000-8000-000000000001",
			DisplayName: "Developer",
			State:       spine.EntityStateActive,
			CreatedAt:   testTime(),
			UpdatedAt:   testTime(),
		},
		OrganizationMembership: spine.OrganizationMembership{
			ID:             "018f0000-0000-7000-8000-000000000011",
			OrganizationID: organizationID,
			UserID:         "018f0000-0000-7000-8000-000000000001",
			Role:           spine.OrganizationMembershipRoleMember,
			State:          spine.EntityStateActive,
			CreatedAt:      testTime(),
			UpdatedAt:      testTime(),
		},
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

type fakeTransactionRunner struct{}

func (r *fakeTransactionRunner) RunReadCommitted(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type sequenceIDs struct {
	intake            int
	goal              int
	clarification     int
	question          int
	answer            int
	contract          int
	contractSeed      int
	contractDraft     int
	approvedContract  int
	workItem          int
	workItemPlan      int
	workItemPlanLease int
	workItemProposal  int
	checkoutJob       int
	checkoutReceipt   int
	executionJob      int
	executionLease    int
	run               int
	commandPlan       int
	executionReceipt  int
	event             int
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
	return spine.GoalID(fmt.Sprintf("018f0000-0000-7000-8000-%012d", g.goal)), nil
}

func (g *sequenceIDs) NewClarificationRequestID() (spine.ClarificationRequestID, error) {
	g.clarification++
	return spine.ClarificationRequestID(fmt.Sprintf("018f0000-0000-7000-8100-%012d", g.clarification)), nil
}

func (g *sequenceIDs) NewClarificationQuestionID() (spine.ClarificationQuestionID, error) {
	g.question++
	return spine.ClarificationQuestionID(fmt.Sprintf("018f0000-0000-7000-8200-%012d", g.question)), nil
}

func (g *sequenceIDs) NewClarificationAnswerID() (spine.ClarificationAnswerID, error) {
	g.answer++
	return spine.ClarificationAnswerID(fmt.Sprintf("018f0000-0000-7000-8300-%012d", g.answer)), nil
}

func (g *sequenceIDs) NewContractID() (spine.ContractID, error) {
	g.contract++
	return spine.ContractID(fmt.Sprintf("contract-%d", g.contract)), nil
}

func (g *sequenceIDs) NewContractSeedID() (spine.ContractSeedID, error) {
	g.contractSeed++
	return spine.ContractSeedID(fmt.Sprintf("contract-seed-%d", g.contractSeed)), nil
}

func (g *sequenceIDs) NewContractDraftID() (spine.ContractDraftID, error) {
	g.contractDraft++
	return spine.ContractDraftID(fmt.Sprintf("contract-draft-%d", g.contractDraft)), nil
}

func (g *sequenceIDs) NewApprovedContractID() (spine.ApprovedContractID, error) {
	g.approvedContract++
	return spine.ApprovedContractID(fmt.Sprintf("approved-contract-%d", g.approvedContract)), nil
}

func (g *sequenceIDs) NewWorkItemID() (spine.WorkItemID, error) {
	g.workItem++
	return spine.WorkItemID(fmt.Sprintf("work-item-%d", g.workItem)), nil
}

func (g *sequenceIDs) NewWorkItemPlanID() (spine.WorkItemPlanID, error) {
	g.workItemPlan++
	return spine.WorkItemPlanID(fmt.Sprintf("plan-%d", g.workItemPlan)), nil
}

func (g *sequenceIDs) NewWorkItemPlanLeaseID() (spine.WorkItemPlanLeaseID, error) {
	g.workItemPlanLease++
	return spine.WorkItemPlanLeaseID(fmt.Sprintf("lease-%d", g.workItemPlanLease)), nil
}

func (g *sequenceIDs) NewWorkItemPlanProposalID() (spine.WorkItemPlanProposalID, error) {
	g.workItemProposal++
	return spine.WorkItemPlanProposalID(fmt.Sprintf("proposal-%d", g.workItemProposal)), nil
}

func (g *sequenceIDs) NewCheckoutJobID() (spine.CheckoutJobID, error) {
	g.checkoutJob++
	return spine.CheckoutJobID(fmt.Sprintf("checkout-job-%d", g.checkoutJob)), nil
}

func (g *sequenceIDs) NewCheckoutReceiptID() (spine.CheckoutReceiptID, error) {
	g.checkoutReceipt++
	return spine.CheckoutReceiptID(fmt.Sprintf("checkout-receipt-%d", g.checkoutReceipt)), nil
}

func (g *sequenceIDs) NewExecutionJobID() (spine.ExecutionJobID, error) {
	g.executionJob++
	return spine.ExecutionJobID(fmt.Sprintf("execution-job-%d", g.executionJob)), nil
}

func (g *sequenceIDs) NewExecutionLeaseID() (spine.ExecutionLeaseID, error) {
	g.executionLease++
	return spine.ExecutionLeaseID(fmt.Sprintf("execution-lease-%d", g.executionLease)), nil
}

func (g *sequenceIDs) NewRunID() (spine.RunID, error) {
	g.run++
	return spine.RunID(fmt.Sprintf("run-%d", g.run)), nil
}

func (g *sequenceIDs) NewExecutionCommandPlanID() (spine.ExecutionCommandPlanID, error) {
	g.commandPlan++
	return spine.ExecutionCommandPlanID(fmt.Sprintf("execution-command-plan-%d", g.commandPlan)), nil
}

func (g *sequenceIDs) NewExecutionReceiptID() (spine.ExecutionReceiptID, error) {
	g.executionReceipt++
	return spine.ExecutionReceiptID(fmt.Sprintf("018f0000-0000-7000-8004-%012d", g.executionReceipt)), nil
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

	response := doJSON(t, server.router, http.MethodPost, "/v1/intakes", body)
	if response.code != http.StatusAccepted {
		t.Fatalf("POST /v1/intakes status = %d, want %d: %s", response.code, http.StatusAccepted, response.body)
	}

	var accepted struct {
		IntakeID string `json:"intake_id"`
	}
	decodeJSON(t, response.body, &accepted)
	return accepted.IntakeID
}

func promoteIntake(t *testing.T, server testServerDeps, intakeID string) spine.Goal {
	t.Helper()

	response := doJSON(t, server.router, http.MethodPost, "/v1/intakes/"+intakeID+"/goals", "")
	if response.code != http.StatusCreated {
		t.Fatalf("POST /v1/intakes/{id}/goals status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
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

func createReadyEnoughGoal(t *testing.T, server testServerDeps) spine.Goal {
	t.Helper()

	created := spine.Goal{
		ID:             "018f0000-0000-7000-8000-000000000101",
		IntakeID:       "direct-ready-intake-1",
		OrganizationID: "018f0000-0000-7000-8000-000000000002",
		ProjectID:      "018f0000-0000-7000-8000-000000000003",
		RepoBindingID:  "018f0000-0000-7000-8000-000000000004",
		Title:          "Ready goal",
		Summary:        "Refactor duplicate CSV export filter logic",
		ScopeHint:      "CSV export filter duplicate handling",
		AcceptanceHint: "Existing CSV export behavior is preserved",
		SourceRefs: []spine.SourceRef{
			{Kind: "test", ID: "direct-ready-intake-1"},
		},
		RequestAuthor: spine.ActorRef{Kind: "user", ID: "018f0000-0000-7000-8000-000000000001"},
		IntentOwner:   spine.ActorRef{Kind: "user", ID: "018f0000-0000-7000-8000-000000000001"},
		State:         spine.GoalStateCreated,
		CreatedAt:     testTime(),
	}
	if err := server.goals.Create(context.Background(), created); err != nil {
		t.Fatalf("goals.Create() error = %v", err)
	}
	return created
}

func assertNoContractTaskSideEffects(t *testing.T, server testServerDeps) {
	t.Helper()
	if len(server.contracts.contracts) != 0 {
		t.Fatalf("contracts = %d, want 0", len(server.contracts.contracts))
	}
	if len(server.contractSeeds.seeds) != 0 {
		t.Fatalf("contract seeds = %d, want 0", len(server.contractSeeds.seeds))
	}
	if len(server.contractDrafts.drafts) != 0 {
		t.Fatalf("contract drafts = %d, want 0", len(server.contractDrafts.drafts))
	}
	if len(server.approvedContracts.approved) != 0 {
		t.Fatalf("approved contracts = %d, want 0", len(server.approvedContracts.approved))
	}
	if len(server.workItems.items) != 0 {
		t.Fatalf("work items = %d, want 0", len(server.workItems.items))
	}
	if len(server.workItemPlans.plans) != 0 {
		t.Fatalf("work item plans = %d, want 0", len(server.workItemPlans.plans))
	}
	if len(server.workItemProposals.proposals) != 0 {
		t.Fatalf("work item proposals = %d, want 0", len(server.workItemProposals.proposals))
	}
}

func createClarificationRequest(t *testing.T, server testServerDeps) spine.ClarificationRequest {
	t.Helper()

	created := createClarificationReadyGoal(t, server)
	response := doJSON(t, server.router, http.MethodPost, "/v1/goals/"+string(created.ID)+"/clarifications", "")
	if response.code != http.StatusCreated {
		t.Fatalf("POST /v1/goals/{id}/clarifications status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
	}

	var request spine.ClarificationRequest
	decodeJSON(t, response.body, &request)
	return request
}

func createClarificationRequestForReasons(t *testing.T, server testServerDeps, reasons ...spine.GoalReadinessReasonCode) spine.ClarificationRequest {
	t.Helper()

	created := spine.Goal{
		ID:             "018f0000-0000-7000-8000-000000000201",
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

	response := doJSON(t, server.router, http.MethodPost, "/v1/goals/"+string(created.ID)+"/clarifications", "")
	if response.code != http.StatusCreated {
		t.Fatalf("POST /v1/goals/{id}/clarifications status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
	}

	var request spine.ClarificationRequest
	decodeJSON(t, response.body, &request)
	return request
}

func createClarificationAnswerForReasons(t *testing.T, server testServerDeps, values map[spine.ClarificationMapsTo]string, reasons ...spine.GoalReadinessReasonCode) spine.ClarificationAnswer {
	t.Helper()

	request := createClarificationRequestForReasons(t, server, reasons...)
	response := doJSON(t, server.router, http.MethodPost, "/v1/clarifications/"+string(request.ID)+"/answers", answerSubmissionJSONWithValues(request, values))
	if response.code != http.StatusCreated {
		t.Fatalf("POST /v1/clarifications/{id}/answers status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
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

func workAnswerSubmissionJSON(request spine.ClarificationRequest) string {
	return workAnswerSubmissionJSONWithValues(request, nil)
}

func workAnswerSubmissionJSONWithValues(request spine.ClarificationRequest, values map[spine.ClarificationMapsTo]string) string {
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
	return fmt.Sprintf(`{"answers":[%s]}`, strings.Join(answers, ","))
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
