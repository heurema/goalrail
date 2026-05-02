package httpserver_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/heurema/goalrail/apps/server/internal/contractseed"
	"github.com/heurema/goalrail/apps/server/internal/spine"
)

func TestPostGoalContractSeedReturnsCreatedSeed(t *testing.T) {
	server := testServer(t)
	goal := createReadyForContractSeedGoal(t, server)

	response := doJSON(t, server.router, http.MethodPost, "/v1/goals/"+string(goal.ID)+"/contract-seeds", "")
	if response.code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
	}
	for _, forbiddenField := range []string{"\"contract_id\"", "\"contract_draft_id\"", "\"work_item_id\"", "\"proof_id\""} {
		if strings.Contains(response.body, forbiddenField) {
			t.Fatalf("response includes forbidden field %s", forbiddenField)
		}
	}

	var seed spine.ContractSeed
	decodeJSON(t, response.body, &seed)
	if seed.State != spine.ContractSeedStateCreated {
		t.Fatalf("state = %q, want %q", seed.State, spine.ContractSeedStateCreated)
	}
	if seed.GoalID != goal.ID {
		t.Fatalf("goal_id = %q, want %q", seed.GoalID, goal.ID)
	}
	if seed.RepoBindingID != goal.RepoBindingID {
		t.Fatalf("repo_binding_id = %q, want %q", seed.RepoBindingID, goal.RepoBindingID)
	}
	if seed.Title != goal.Title {
		t.Fatalf("title = %q, want %q", seed.Title, goal.Title)
	}
	if seed.IntentSummary != goal.Summary {
		t.Fatalf("intent_summary = %q, want %q", seed.IntentSummary, goal.Summary)
	}
	if seed.ScopeHint != goal.ScopeHint {
		t.Fatalf("scope_hint = %q, want %q", seed.ScopeHint, goal.ScopeHint)
	}
	if seed.AcceptanceHint != goal.AcceptanceHint {
		t.Fatalf("acceptance_hint = %q, want %q", seed.AcceptanceHint, goal.AcceptanceHint)
	}
	if !hasSourceRef(seed.SourceRefs, "goal", string(goal.ID)) {
		t.Fatalf("source_refs = %#v, want goal ref", seed.SourceRefs)
	}
	if !hasSourceRef(seed.SourceRefs, "intake", string(goal.IntakeID)) {
		t.Fatalf("source_refs = %#v, want intake ref", seed.SourceRefs)
	}

	storedSeed, ok, err := server.contractSeeds.Get(context.Background(), seed.ID)
	if err != nil {
		t.Fatalf("contractSeeds.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("stored contract seed not found")
	}
	if storedSeed.ID != seed.ID {
		t.Fatalf("stored seed id = %q, want %q", storedSeed.ID, seed.ID)
	}

	storedGoal, ok, err := server.goals.Get(context.Background(), goal.ID)
	if err != nil {
		t.Fatalf("goals.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("stored goal not found")
	}
	if storedGoal.State != spine.GoalStateReadyForContractSeed {
		t.Fatalf("goal state = %q, want %q", storedGoal.State, spine.GoalStateReadyForContractSeed)
	}
}

func TestPostGoalContractSeedUnknownGoalReturnsNotFound(t *testing.T) {
	server := testServer(t)

	response := doJSON(t, server.router, http.MethodPost, "/v1/goals/missing/contract-seeds", "")
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

func TestPostGoalContractSeedRejectsGoalNotReady(t *testing.T) {
	server := testServer(t)
	intakeID := createIntake(t, server, validIntakeJSON)
	goal := promoteIntake(t, server, intakeID)

	response := doJSON(t, server.router, http.MethodPost, "/v1/goals/"+string(goal.ID)+"/contract-seeds", "")
	if response.code != http.StatusConflict {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusConflict, response.body)
	}

	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeJSON(t, response.body, &body)
	if body.Error.Code != "invalid_state" {
		t.Fatalf("error code = %q, want invalid_state", body.Error.Code)
	}
}

func TestPostGoalContractSeedRejectsMissingRequiredGoalFields(t *testing.T) {
	server := testServer(t)
	goal := createReadyForContractSeedGoal(t, server)
	mutated := goal
	mutated.ScopeHint = ""
	if _, ok, err := server.goals.UpdateHints(context.Background(), goal.ID, spine.GoalHintUpdate{ScopeHint: &mutated.ScopeHint}); err != nil {
		t.Fatalf("goals.UpdateHints() error = %v", err)
	} else if !ok {
		t.Fatal("goal not found")
	}

	response := doJSON(t, server.router, http.MethodPost, "/v1/goals/"+string(goal.ID)+"/contract-seeds", "")
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
}

func TestPostGoalContractSeedRejectsDuplicateSeed(t *testing.T) {
	server := testServer(t)
	goal := createReadyForContractSeedGoal(t, server)

	first := doJSON(t, server.router, http.MethodPost, "/v1/goals/"+string(goal.ID)+"/contract-seeds", "")
	if first.code != http.StatusCreated {
		t.Fatalf("first status = %d, want %d: %s", first.code, http.StatusCreated, first.body)
	}
	second := doJSON(t, server.router, http.MethodPost, "/v1/goals/"+string(goal.ID)+"/contract-seeds", "")
	if second.code != http.StatusConflict {
		t.Fatalf("second status = %d, want %d: %s", second.code, http.StatusConflict, second.body)
	}

	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeJSON(t, second.body, &body)
	if body.Error.Code != "already_seeded" {
		t.Fatalf("error code = %q, want already_seeded", body.Error.Code)
	}
}

func TestPostGoalContractSeedAppendsEventOnly(t *testing.T) {
	server := testServer(t)
	goal := createReadyForContractSeedGoal(t, server)

	response := doJSON(t, server.router, http.MethodPost, "/v1/goals/"+string(goal.ID)+"/contract-seeds", "")
	if response.code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
	}

	if got := countEventType(server.events.Events(), contractseed.EventTypeContractSeedCreated); got != 1 {
		t.Fatalf("contract_seed.created events = %d, want 1", got)
	}
	assertNoForbiddenEventTypes(t, server.events.Events())
}

func TestPostGoalContractSeedFullIntentPlaneFlow(t *testing.T) {
	server := testServer(t)
	goal := createReadyForContractSeedGoal(t, server)

	response := doJSON(t, server.router, http.MethodPost, "/v1/goals/"+string(goal.ID)+"/contract-seeds", "")
	if response.code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
	}

	var seed spine.ContractSeed
	decodeJSON(t, response.body, &seed)
	if seed.State != spine.ContractSeedStateCreated {
		t.Fatalf("state = %q, want %q", seed.State, spine.ContractSeedStateCreated)
	}
	for _, forbiddenField := range []string{"\"contract_draft_id\"", "\"work_item_id\"", "\"gate_decision_id\"", "\"proof_id\""} {
		if strings.Contains(response.body, forbiddenField) {
			t.Fatalf("response includes forbidden field %s", forbiddenField)
		}
	}
	assertNoForbiddenEventTypes(t, server.events.Events())
}

func createReadyForContractSeedGoal(t *testing.T, server testServerDeps) spine.Goal {
	t.Helper()

	intakeID := createIntake(t, server, validIntakeJSON)
	created := promoteIntake(t, server, intakeID)

	initialReadiness := doJSON(t, server.router, http.MethodPost, "/v1/goals/"+string(created.ID)+"/readiness-checks", "")
	if initialReadiness.code != http.StatusOK {
		t.Fatalf("initial readiness status = %d, want %d: %s", initialReadiness.code, http.StatusOK, initialReadiness.body)
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
	applyResponse := doJSON(t, server.router, http.MethodPost, "/v1/clarification-answers/"+string(answer.ID)+"/applications", applyRequestJSON())
	if applyResponse.code != http.StatusOK {
		t.Fatalf("apply status = %d, want %d: %s", applyResponse.code, http.StatusOK, applyResponse.body)
	}

	recheckResponse := doJSON(t, server.router, http.MethodPost, "/v1/goals/"+string(created.ID)+"/readiness-checks", "")
	if recheckResponse.code != http.StatusOK {
		t.Fatalf("explicit re-check status = %d, want %d: %s", recheckResponse.code, http.StatusOK, recheckResponse.body)
	}

	var recheckBody struct {
		Goal spine.Goal `json:"goal"`
	}
	decodeJSON(t, recheckResponse.body, &recheckBody)
	if recheckBody.Goal.State != spine.GoalStateReadyForContractSeed {
		t.Fatalf("goal state = %q, want %q", recheckBody.Goal.State, spine.GoalStateReadyForContractSeed)
	}
	return recheckBody.Goal
}

func hasSourceRef(refs []spine.SourceRef, kind string, id string) bool {
	for _, ref := range refs {
		if ref.Kind == kind && ref.ID == id {
			return true
		}
	}
	return false
}
