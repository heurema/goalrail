package httpserver_test

import (
	"context"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/heurema/goalrail/apps/server/internal/contractdraft"
	"github.com/heurema/goalrail/apps/server/internal/spine"
)

func TestPostContractSeedContractDraftReturnsCreatedDraft(t *testing.T) {
	server := testServer(t)
	seed := createContractSeed(t, server)
	beforeSeed, ok, err := server.contractSeeds.Get(context.Background(), seed.ID)
	if err != nil {
		t.Fatalf("contractSeeds.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("contract seed not found")
	}

	response := doJSON(t, server.router, http.MethodPost, "/v1/contract-seeds/"+string(seed.ID)+"/contract-draft", "")
	if response.code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
	}
	for _, forbiddenField := range []string{"\"approved_contract_id\"", "\"work_item_id\"", "\"proof_id\"", "\"gate_decision_id\""} {
		if strings.Contains(response.body, forbiddenField) {
			t.Fatalf("response includes forbidden field %s", forbiddenField)
		}
	}

	var draft spine.ContractDraft
	decodeJSON(t, response.body, &draft)
	if draft.State != spine.ContractDraftStateDraft {
		t.Fatalf("state = %q, want %q", draft.State, spine.ContractDraftStateDraft)
	}
	if draft.ContractSeedID != seed.ID {
		t.Fatalf("contract_seed_id = %q, want %q", draft.ContractSeedID, seed.ID)
	}
	if draft.GoalID != seed.GoalID {
		t.Fatalf("goal_id = %q, want %q", draft.GoalID, seed.GoalID)
	}
	if draft.RepoBindingID != seed.RepoBindingID {
		t.Fatalf("repo_binding_id = %q, want %q", draft.RepoBindingID, seed.RepoBindingID)
	}
	if draft.Title != seed.Title {
		t.Fatalf("title = %q, want %q", draft.Title, seed.Title)
	}
	if draft.IntentSummary != seed.IntentSummary {
		t.Fatalf("intent_summary = %q, want %q", draft.IntentSummary, seed.IntentSummary)
	}
	if !reflect.DeepEqual(draft.ProposedScope, []string{seed.ScopeHint}) {
		t.Fatalf("proposed_scope = %#v, want seed scope hint", draft.ProposedScope)
	}
	if !reflect.DeepEqual(draft.ProposedAcceptanceCriteria, []string{seed.AcceptanceHint}) {
		t.Fatalf("proposed_acceptance_criteria = %#v, want seed acceptance hint", draft.ProposedAcceptanceCriteria)
	}
	if len(draft.ProposedNonGoals) != 0 {
		t.Fatalf("proposed_non_goals = %#v, want empty", draft.ProposedNonGoals)
	}
	if len(draft.ProposedConstraints) != 0 {
		t.Fatalf("proposed_constraints = %#v, want empty", draft.ProposedConstraints)
	}
	if len(draft.ProposedExpectedChecks) != 0 {
		t.Fatalf("proposed_expected_checks = %#v, want empty", draft.ProposedExpectedChecks)
	}
	if !reflect.DeepEqual(draft.ProposedProofExpectations, []string{contractdraft.DefaultProofExpectation}) {
		t.Fatalf("proposed_proof_expectations = %#v, want default", draft.ProposedProofExpectations)
	}
	if len(draft.RiskHints) != 0 {
		t.Fatalf("risk_hints = %#v, want empty", draft.RiskHints)
	}
	if !hasSourceRef(draft.SourceRefs, "contract_seed", string(seed.ID)) {
		t.Fatalf("source_refs = %#v, want contract_seed ref", draft.SourceRefs)
	}
	if !hasSourceRef(draft.SourceRefs, "goal", string(seed.GoalID)) {
		t.Fatalf("source_refs = %#v, want goal ref", draft.SourceRefs)
	}
	if !hasSourceRef(draft.SourceRefs, "intake", "intake-1") {
		t.Fatalf("source_refs = %#v, want intake ref", draft.SourceRefs)
	}

	storedDraft, ok, err := server.contractDrafts.Get(context.Background(), draft.ID)
	if err != nil {
		t.Fatalf("contractDrafts.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("stored contract draft not found")
	}
	if storedDraft.ID != draft.ID {
		t.Fatalf("stored draft id = %q, want %q", storedDraft.ID, draft.ID)
	}

	afterSeed, ok, err := server.contractSeeds.Get(context.Background(), seed.ID)
	if err != nil {
		t.Fatalf("contractSeeds.Get() after draft error = %v", err)
	}
	if !ok {
		t.Fatal("contract seed not found after draft creation")
	}
	if !reflect.DeepEqual(afterSeed, beforeSeed) {
		t.Fatalf("contract seed mutated: got %#v, want %#v", afterSeed, beforeSeed)
	}
}

func TestPostContractSeedContractDraftUnknownSeedReturnsNotFound(t *testing.T) {
	server := testServer(t)

	response := doJSON(t, server.router, http.MethodPost, "/v1/contract-seeds/missing/contract-draft", "")
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
		t.Fatalf("error code = %q, want not_found", body.Error.Code)
	}
}

func TestPostContractSeedContractDraftRejectsSeedNotCreated(t *testing.T) {
	server := testServer(t)
	seed := validHTTPContractSeed()
	seed.State = "superseded"
	if err := server.contractSeeds.Create(context.Background(), seed); err != nil {
		t.Fatalf("contractSeeds.Create() error = %v", err)
	}

	response := doJSON(t, server.router, http.MethodPost, "/v1/contract-seeds/"+string(seed.ID)+"/contract-draft", "")
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

func TestPostContractSeedContractDraftRejectsMissingRequiredSeedFields(t *testing.T) {
	server := testServer(t)
	seed := validHTTPContractSeed()
	seed.ScopeHint = ""
	if err := server.contractSeeds.Create(context.Background(), seed); err != nil {
		t.Fatalf("contractSeeds.Create() error = %v", err)
	}

	response := doJSON(t, server.router, http.MethodPost, "/v1/contract-seeds/"+string(seed.ID)+"/contract-draft", "")
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

func TestPostContractSeedContractDraftRejectsDuplicateDraft(t *testing.T) {
	server := testServer(t)
	seed := createContractSeed(t, server)

	first := doJSON(t, server.router, http.MethodPost, "/v1/contract-seeds/"+string(seed.ID)+"/contract-draft", "")
	if first.code != http.StatusCreated {
		t.Fatalf("first status = %d, want %d: %s", first.code, http.StatusCreated, first.body)
	}
	second := doJSON(t, server.router, http.MethodPost, "/v1/contract-seeds/"+string(seed.ID)+"/contract-draft", "")
	if second.code != http.StatusConflict {
		t.Fatalf("second status = %d, want %d: %s", second.code, http.StatusConflict, second.body)
	}

	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeJSON(t, second.body, &body)
	if body.Error.Code != "already_drafted" {
		t.Fatalf("error code = %q, want already_drafted", body.Error.Code)
	}
}

func TestPostContractSeedContractDraftAppendsEventOnly(t *testing.T) {
	server := testServer(t)
	seed := createContractSeed(t, server)

	response := doJSON(t, server.router, http.MethodPost, "/v1/contract-seeds/"+string(seed.ID)+"/contract-draft", "")
	if response.code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
	}

	if got := countEventType(server.events.Events(), contractdraft.EventTypeContractDraftCreated); got != 1 {
		t.Fatalf("contract_draft.created events = %d, want 1", got)
	}
	assertNoForbiddenEventTypes(t, server.events.Events())
}

func TestPostContractSeedContractDraftFullFlow(t *testing.T) {
	server := testServer(t)
	seed := createContractSeed(t, server)

	response := doJSON(t, server.router, http.MethodPost, "/v1/contract-seeds/"+string(seed.ID)+"/contract-draft", "")
	if response.code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
	}

	var draft spine.ContractDraft
	decodeJSON(t, response.body, &draft)
	if draft.State != spine.ContractDraftStateDraft {
		t.Fatalf("state = %q, want %q", draft.State, spine.ContractDraftStateDraft)
	}
	for _, forbiddenField := range []string{"\"approved_contract_id\"", "\"work_item_id\"", "\"gate_decision_id\"", "\"proof_id\""} {
		if strings.Contains(response.body, forbiddenField) {
			t.Fatalf("response includes forbidden field %s", forbiddenField)
		}
	}
	assertNoForbiddenEventTypes(t, server.events.Events())
}

func createContractSeed(t *testing.T, server testServerDeps) spine.ContractSeed {
	t.Helper()

	goal := createReadyForContractSeedGoal(t, server)
	response := doJSON(t, server.router, http.MethodPost, "/v1/goals/"+string(goal.ID)+"/contract-seed", "")
	if response.code != http.StatusCreated {
		t.Fatalf("contract seed status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
	}

	var seed spine.ContractSeed
	decodeJSON(t, response.body, &seed)
	return seed
}

func validHTTPContractSeed() spine.ContractSeed {
	return spine.ContractSeed{
		ID:             "contract-seed-1",
		GoalID:         "goal-1",
		RepoBindingID:  "018f0000-0000-7000-8000-000000000004",
		Title:          "Refactor CSV export filters",
		IntentSummary:  "Current code duplicates filter logic. Preserve current behavior.",
		IntentOwner:    spine.ActorRef{Kind: "user", ID: "dev_1"},
		ScopeHint:      "Refactor duplicate CSV export filter logic",
		AcceptanceHint: "Existing CSV export behavior is preserved",
		SourceRefs: []spine.SourceRef{
			{Kind: "goal", ID: "goal-1"},
			{Kind: "intake", ID: "intake-1"},
		},
		State:     spine.ContractSeedStateCreated,
		CreatedAt: testTime(),
	}
}
