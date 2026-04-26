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

func TestPostContractDraftUpdatesReturnsUpdatedDraft(t *testing.T) {
	server := testServer(t)
	draft := createContractDraft(t, server)

	response := doJSON(t, server.router, http.MethodPost, "/v1/contract-drafts/"+string(draft.ID)+"/updates", updateDraftJSON(`{
		"title": "Reviewed draft title",
		"intent_summary": "Reviewed summary",
		"proposed_scope": ["Reviewed scope"],
		"proposed_acceptance_criteria": ["Reviewed acceptance"],
		"proposed_non_goals": [],
		"proposed_constraints": ["No schema changes"],
		"proposed_expected_checks": ["go test ./..."],
		"proposed_proof_expectations": ["Attach test output"],
		"risk_hints": ["Low risk"]
	}`))
	if response.code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusOK, response.body)
	}
	for _, forbiddenField := range []string{"\"approved_contract_id\"", "\"work_item_id\"", "\"proof_id\"", "\"gate_decision_id\""} {
		if strings.Contains(response.body, forbiddenField) {
			t.Fatalf("response includes forbidden field %s", forbiddenField)
		}
	}

	var updated spine.ContractDraft
	decodeJSON(t, response.body, &updated)
	if updated.State != spine.ContractDraftStateDraft {
		t.Fatalf("state = %q, want %q", updated.State, spine.ContractDraftStateDraft)
	}
	if updated.Title != "Reviewed draft title" {
		t.Fatalf("title = %q, want reviewed title", updated.Title)
	}
	if updated.IntentSummary != "Reviewed summary" {
		t.Fatalf("intent_summary = %q, want reviewed summary", updated.IntentSummary)
	}
	if !reflect.DeepEqual(updated.ProposedScope, []string{"Reviewed scope"}) {
		t.Fatalf("proposed_scope = %#v", updated.ProposedScope)
	}
	if !reflect.DeepEqual(updated.ProposedAcceptanceCriteria, []string{"Reviewed acceptance"}) {
		t.Fatalf("proposed_acceptance_criteria = %#v", updated.ProposedAcceptanceCriteria)
	}
	if !reflect.DeepEqual(updated.ProposedNonGoals, []string{}) {
		t.Fatalf("proposed_non_goals = %#v, want empty slice", updated.ProposedNonGoals)
	}
	if !reflect.DeepEqual(updated.ProposedConstraints, []string{"No schema changes"}) {
		t.Fatalf("proposed_constraints = %#v", updated.ProposedConstraints)
	}
	if !reflect.DeepEqual(updated.ProposedExpectedChecks, []string{"go test ./..."}) {
		t.Fatalf("proposed_expected_checks = %#v", updated.ProposedExpectedChecks)
	}
	if !reflect.DeepEqual(updated.ProposedProofExpectations, []string{"Attach test output"}) {
		t.Fatalf("proposed_proof_expectations = %#v", updated.ProposedProofExpectations)
	}
	if !reflect.DeepEqual(updated.RiskHints, []string{"Low risk"}) {
		t.Fatalf("risk_hints = %#v", updated.RiskHints)
	}
	if updated.ContractSeedID != draft.ContractSeedID || updated.GoalID != draft.GoalID || updated.RepoBindingID != draft.RepoBindingID {
		t.Fatalf("identity fields changed: got %#v, want unchanged from %#v", updated, draft)
	}
	if !reflect.DeepEqual(updated.SourceRefs, draft.SourceRefs) {
		t.Fatalf("source_refs = %#v, want unchanged %#v", updated.SourceRefs, draft.SourceRefs)
	}
	if updated.CreatedAt != draft.CreatedAt {
		t.Fatalf("created_at = %s, want %s", updated.CreatedAt, draft.CreatedAt)
	}
}

func TestPostContractDraftUpdatesAllowsEmptyCoreDraftArrays(t *testing.T) {
	server := testServer(t)
	draft := createContractDraft(t, server)

	response := doJSON(t, server.router, http.MethodPost, "/v1/contract-drafts/"+string(draft.ID)+"/updates", updateDraftJSON(`{
		"proposed_scope": [],
		"proposed_acceptance_criteria": [],
		"proposed_proof_expectations": []
	}`))
	if response.code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusOK, response.body)
	}

	var updated spine.ContractDraft
	decodeJSON(t, response.body, &updated)
	if updated.ProposedScope == nil || len(updated.ProposedScope) != 0 {
		t.Fatalf("proposed_scope = %#v, want explicit empty slice", updated.ProposedScope)
	}
	if updated.ProposedAcceptanceCriteria == nil || len(updated.ProposedAcceptanceCriteria) != 0 {
		t.Fatalf("proposed_acceptance_criteria = %#v, want explicit empty slice", updated.ProposedAcceptanceCriteria)
	}
	if updated.ProposedProofExpectations == nil || len(updated.ProposedProofExpectations) != 0 {
		t.Fatalf("proposed_proof_expectations = %#v, want explicit empty slice", updated.ProposedProofExpectations)
	}
}

func TestPostContractDraftUpdatesUnknownDraftReturnsNotFound(t *testing.T) {
	server := testServer(t)

	response := doJSON(t, server.router, http.MethodPost, "/v1/contract-drafts/missing/updates", updateDraftJSON(`{"title": "Reviewed"}`))
	if response.code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusNotFound, response.body)
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

func TestPostContractDraftUpdatesRejectsDraftNotDraft(t *testing.T) {
	server := testServer(t)
	draft := validHTTPContractDraft()
	draft.State = "ready_for_approval"
	if err := server.contractDrafts.Create(context.Background(), draft); err != nil {
		t.Fatalf("contractDrafts.Create() error = %v", err)
	}

	response := doJSON(t, server.router, http.MethodPost, "/v1/contract-drafts/"+string(draft.ID)+"/updates", updateDraftJSON(`{"title": "Reviewed"}`))
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

func TestPostContractDraftUpdatesValidatesUpdatedBy(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "missing_updated_by", body: `{"changes":{"title":"Reviewed"}}`},
		{name: "missing_kind", body: `{"updated_by":{"id":"dev_1"},"changes":{"title":"Reviewed"}}`},
		{name: "missing_id", body: `{"updated_by":{"kind":"user"},"changes":{"title":"Reviewed"}}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := testServer(t)
			draft := createContractDraft(t, server)

			response := doJSON(t, server.router, http.MethodPost, "/v1/contract-drafts/"+string(draft.ID)+"/updates", tt.body)
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
		})
	}
}

func TestPostContractDraftUpdatesRejectsEmptyChanges(t *testing.T) {
	server := testServer(t)
	draft := createContractDraft(t, server)

	response := doJSON(t, server.router, http.MethodPost, "/v1/contract-drafts/"+string(draft.ID)+"/updates", updateDraftJSON(`{}`))
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

func TestPostContractDraftUpdatesRejectsUnknownFieldInChanges(t *testing.T) {
	server := testServer(t)
	draft := createContractDraft(t, server)

	response := doJSON(t, server.router, http.MethodPost, "/v1/contract-drafts/"+string(draft.ID)+"/updates", updateDraftJSON(`{"unexpected": "value"}`))
	if response.code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusBadRequest, response.body)
	}

	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeJSON(t, response.body, &body)
	if body.Error.Code != "unknown_field" {
		t.Fatalf("error code = %q, want unknown_field", body.Error.Code)
	}
}

func TestPostContractDraftUpdatesRejectsNonEditableFieldInChanges(t *testing.T) {
	server := testServer(t)
	draft := createContractDraft(t, server)

	response := doJSON(t, server.router, http.MethodPost, "/v1/contract-drafts/"+string(draft.ID)+"/updates", updateDraftJSON(`{"state": "ready_for_approval"}`))
	if response.code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusBadRequest, response.body)
	}

	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeJSON(t, response.body, &body)
	if body.Error.Code != "non_editable_field" {
		t.Fatalf("error code = %q, want non_editable_field", body.Error.Code)
	}
}

func TestPostContractDraftUpdatesAppendsEventOnly(t *testing.T) {
	server := testServer(t)
	draft := createContractDraft(t, server)

	response := doJSON(t, server.router, http.MethodPost, "/v1/contract-drafts/"+string(draft.ID)+"/updates", updateDraftJSON(`{"proposed_scope": ["Reviewed scope"]}`))
	if response.code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusOK, response.body)
	}

	if got := countEventType(server.events.Events(), contractdraft.EventTypeContractDraftUpdated); got != 1 {
		t.Fatalf("contract_draft.updated events = %d, want 1", got)
	}
	assertNoForbiddenEventTypes(t, server.events.Events())
}

func TestPostContractDraftUpdatesFullFlow(t *testing.T) {
	server := testServer(t)
	draft := createContractDraft(t, server)

	response := doJSON(t, server.router, http.MethodPost, "/v1/contract-drafts/"+string(draft.ID)+"/updates", updateDraftJSON(`{
		"proposed_scope": ["Reviewed scope"],
		"proposed_acceptance_criteria": ["Reviewed acceptance"]
	}`))
	if response.code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusOK, response.body)
	}

	var updated spine.ContractDraft
	decodeJSON(t, response.body, &updated)
	if updated.State != spine.ContractDraftStateDraft {
		t.Fatalf("state = %q, want %q", updated.State, spine.ContractDraftStateDraft)
	}
	if !reflect.DeepEqual(updated.ProposedScope, []string{"Reviewed scope"}) {
		t.Fatalf("proposed_scope = %#v", updated.ProposedScope)
	}
	if !reflect.DeepEqual(updated.ProposedAcceptanceCriteria, []string{"Reviewed acceptance"}) {
		t.Fatalf("proposed_acceptance_criteria = %#v", updated.ProposedAcceptanceCriteria)
	}
	for _, forbiddenField := range []string{"\"approved_contract_id\"", "\"work_item_id\"", "\"gate_decision_id\"", "\"proof_id\""} {
		if strings.Contains(response.body, forbiddenField) {
			t.Fatalf("response includes forbidden field %s", forbiddenField)
		}
	}
	assertNoForbiddenEventTypes(t, server.events.Events())
}

func TestPostContractDraftReadyForApprovalReturnsReadyDraft(t *testing.T) {
	server := testServer(t)
	draft := createContractDraft(t, server)
	beforeStored, ok, err := server.contractDrafts.Get(context.Background(), draft.ID)
	if err != nil {
		t.Fatalf("contractDrafts.Get() before ready error = %v", err)
	}
	if !ok {
		t.Fatal("stored draft before ready not found")
	}

	response := doJSON(t, server.router, http.MethodPost, "/v1/contract-drafts/"+string(draft.ID)+"/ready-for-approval", readyForApprovalJSON())
	if response.code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusOK, response.body)
	}
	for _, forbiddenField := range []string{"\"approved_contract_id\"", "\"work_item_id\"", "\"proof_id\"", "\"gate_decision_id\""} {
		if strings.Contains(response.body, forbiddenField) {
			t.Fatalf("response includes forbidden field %s", forbiddenField)
		}
	}

	var updated spine.ContractDraft
	decodeJSON(t, response.body, &updated)
	if updated.State != spine.ContractDraftStateReadyForApproval {
		t.Fatalf("state = %q, want %q", updated.State, spine.ContractDraftStateReadyForApproval)
	}
	expected := draft
	expected.State = spine.ContractDraftStateReadyForApproval
	if !reflect.DeepEqual(updated, expected) {
		t.Fatalf("updated draft = %#v, want only state changed %#v", updated, expected)
	}

	stored, ok, err := server.contractDrafts.Get(context.Background(), draft.ID)
	if err != nil {
		t.Fatalf("contractDrafts.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("stored draft not found")
	}
	expectedStored := beforeStored
	expectedStored.State = spine.ContractDraftStateReadyForApproval
	if !reflect.DeepEqual(stored, expectedStored) {
		t.Fatalf("stored draft = %#v, want %#v", stored, expectedStored)
	}
}

func TestPostContractDraftReadyForApprovalUnknownDraftReturnsNotFound(t *testing.T) {
	server := testServer(t)

	response := doJSON(t, server.router, http.MethodPost, "/v1/contract-drafts/missing/ready-for-approval", readyForApprovalJSON())
	if response.code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusNotFound, response.body)
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

func TestPostContractDraftReadyForApprovalRejectsDraftNotDraft(t *testing.T) {
	server := testServer(t)
	draft := validHTTPContractDraft()
	draft.State = spine.ContractDraftStateReadyForApproval
	if err := server.contractDrafts.Create(context.Background(), draft); err != nil {
		t.Fatalf("contractDrafts.Create() error = %v", err)
	}

	response := doJSON(t, server.router, http.MethodPost, "/v1/contract-drafts/"+string(draft.ID)+"/ready-for-approval", readyForApprovalJSON())
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

func TestPostContractDraftReadyForApprovalValidatesMarkedBy(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "missing_marked_by", body: `{}`},
		{name: "missing_kind", body: `{"marked_by":{"id":"dev_1"}}`},
		{name: "missing_id", body: `{"marked_by":{"kind":"user"}}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := testServer(t)
			draft := createContractDraft(t, server)

			response := doJSON(t, server.router, http.MethodPost, "/v1/contract-drafts/"+string(draft.ID)+"/ready-for-approval", tt.body)
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
		})
	}
}

func TestPostContractDraftReadyForApprovalRejectsIncompleteDraft(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*spine.ContractDraft)
		reason string
	}{
		{name: "missing_proposed_scope", mutate: func(draft *spine.ContractDraft) { draft.ProposedScope = nil }, reason: contractdraft.ReasonMissingProposedScope},
		{name: "missing_proposed_acceptance_criteria", mutate: func(draft *spine.ContractDraft) { draft.ProposedAcceptanceCriteria = []string{} }, reason: contractdraft.ReasonMissingProposedAcceptanceCriteria},
		{name: "missing_proposed_proof_expectations", mutate: func(draft *spine.ContractDraft) { draft.ProposedProofExpectations = []string{" "} }, reason: contractdraft.ReasonMissingProposedProofExpectations},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := testServer(t)
			draft := validHTTPContractDraft()
			draft.ID = spine.ContractDraftID("contract-draft-" + tt.name)
			draft.ContractSeedID = spine.ContractSeedID("contract-seed-" + tt.name)
			tt.mutate(&draft)
			if err := server.contractDrafts.Create(context.Background(), draft); err != nil {
				t.Fatalf("contractDrafts.Create() error = %v", err)
			}

			response := doJSON(t, server.router, http.MethodPost, "/v1/contract-drafts/"+string(draft.ID)+"/ready-for-approval", readyForApprovalJSON())
			if response.code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d: %s", response.code, http.StatusBadRequest, response.body)
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
			if !strings.Contains(body.Error.Message, tt.reason) {
				t.Fatalf("error message = %q, want reason %q", body.Error.Message, tt.reason)
			}
		})
	}
}

func TestPostContractDraftReadyForApprovalAppendsEventOnly(t *testing.T) {
	server := testServer(t)
	draft := createContractDraft(t, server)

	response := doJSON(t, server.router, http.MethodPost, "/v1/contract-drafts/"+string(draft.ID)+"/ready-for-approval", readyForApprovalJSON())
	if response.code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusOK, response.body)
	}

	if got := countEventType(server.events.Events(), contractdraft.EventTypeContractDraftMarkedReadyForApproval); got != 1 {
		t.Fatalf("contract_draft.marked_ready_for_approval events = %d, want 1", got)
	}
	stored, ok, err := server.contractDrafts.Get(context.Background(), draft.ID)
	if err != nil {
		t.Fatalf("contractDrafts.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("contractDrafts.Get() ok = false, want true")
	}
	event := server.events.Events()[len(server.events.Events())-1]
	if event.OrganizationID != stored.OrganizationID || event.ProjectID != stored.ProjectID || event.RepoBindingID != stored.RepoBindingID {
		t.Fatalf("event context = %q/%q/%q, want stored draft context %q/%q/%q", event.OrganizationID, event.ProjectID, event.RepoBindingID, stored.OrganizationID, stored.ProjectID, stored.RepoBindingID)
	}
	var payload struct {
		ContractDraftID spine.ContractDraftID    `json:"contract_draft_id"`
		ContractSeedID  spine.ContractSeedID     `json:"contract_seed_id"`
		GoalID          spine.GoalID             `json:"goal_id"`
		MarkedBy        spine.ActorRef           `json:"marked_by"`
		PreviousState   spine.ContractDraftState `json:"previous_state"`
		NewState        spine.ContractDraftState `json:"new_state"`
	}
	decodeJSON(t, string(event.Payload), &payload)
	if payload.ContractDraftID != draft.ID || payload.ContractSeedID != draft.ContractSeedID || payload.GoalID != draft.GoalID {
		t.Fatalf("payload identity = %q/%q/%q, want draft identity %q/%q/%q", payload.ContractDraftID, payload.ContractSeedID, payload.GoalID, draft.ID, draft.ContractSeedID, draft.GoalID)
	}
	if payload.MarkedBy.Kind != "user" || payload.MarkedBy.ID != "dev_1" {
		t.Fatalf("marked_by = %#v, want audit user", payload.MarkedBy)
	}
	if payload.PreviousState != spine.ContractDraftStateDraft || payload.NewState != spine.ContractDraftStateReadyForApproval {
		t.Fatalf("states = %q -> %q, want draft -> ready_for_approval", payload.PreviousState, payload.NewState)
	}
	assertNoForbiddenEventTypes(t, server.events.Events())
}

func TestPostContractDraftReadyForApprovalFullFlow(t *testing.T) {
	server := testServer(t)
	draft := createContractDraft(t, server)

	update := doJSON(t, server.router, http.MethodPost, "/v1/contract-drafts/"+string(draft.ID)+"/updates", updateDraftJSON(`{
		"proposed_scope": ["Reviewed scope"],
		"proposed_acceptance_criteria": ["Reviewed acceptance"],
		"proposed_proof_expectations": ["Attach reviewed proof expectations"]
	}`))
	if update.code != http.StatusOK {
		t.Fatalf("update status = %d, want %d: %s", update.code, http.StatusOK, update.body)
	}

	response := doJSON(t, server.router, http.MethodPost, "/v1/contract-drafts/"+string(draft.ID)+"/ready-for-approval", readyForApprovalJSON())
	if response.code != http.StatusOK {
		t.Fatalf("ready status = %d, want %d: %s", response.code, http.StatusOK, response.body)
	}

	var updated spine.ContractDraft
	decodeJSON(t, response.body, &updated)
	if updated.State != spine.ContractDraftStateReadyForApproval {
		t.Fatalf("state = %q, want %q", updated.State, spine.ContractDraftStateReadyForApproval)
	}
	if !reflect.DeepEqual(updated.ProposedScope, []string{"Reviewed scope"}) {
		t.Fatalf("proposed_scope = %#v", updated.ProposedScope)
	}
	if !reflect.DeepEqual(updated.ProposedAcceptanceCriteria, []string{"Reviewed acceptance"}) {
		t.Fatalf("proposed_acceptance_criteria = %#v", updated.ProposedAcceptanceCriteria)
	}
	if !reflect.DeepEqual(updated.ProposedProofExpectations, []string{"Attach reviewed proof expectations"}) {
		t.Fatalf("proposed_proof_expectations = %#v", updated.ProposedProofExpectations)
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

func createContractDraft(t *testing.T, server testServerDeps) spine.ContractDraft {
	t.Helper()

	seed := createContractSeed(t, server)
	response := doJSON(t, server.router, http.MethodPost, "/v1/contract-seeds/"+string(seed.ID)+"/contract-draft", "")
	if response.code != http.StatusCreated {
		t.Fatalf("contract draft status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
	}

	var draft spine.ContractDraft
	decodeJSON(t, response.body, &draft)
	return draft
}

func updateDraftJSON(changes string) string {
	return `{
		"updated_by": {
			"kind": "user",
			"id": "dev_1",
			"display_name": "Developer"
		},
		"changes": ` + changes + `
	}`
}

func readyForApprovalJSON() string {
	return `{
		"marked_by": {
			"kind": "user",
			"id": "dev_1",
			"display_name": "Developer"
		}
	}`
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

func validHTTPContractDraft() spine.ContractDraft {
	return spine.ContractDraft{
		ID:                         "contract-draft-1",
		ContractSeedID:             "contract-seed-1",
		GoalID:                     "goal-1",
		RepoBindingID:              "018f0000-0000-7000-8000-000000000004",
		Title:                      "Refactor CSV export filters",
		IntentSummary:              "Current code duplicates filter logic.",
		ProposedScope:              []string{"Refactor duplicate CSV export filter logic"},
		ProposedNonGoals:           []string{},
		ProposedConstraints:        []string{},
		ProposedAcceptanceCriteria: []string{"Existing CSV export behavior is preserved"},
		ProposedExpectedChecks:     []string{},
		ProposedProofExpectations:  []string{contractdraft.DefaultProofExpectation},
		RiskHints:                  []string{},
		SourceRefs: []spine.SourceRef{
			{Kind: "contract_seed", ID: "contract-seed-1"},
			{Kind: "goal", ID: "goal-1"},
			{Kind: "intake", ID: "intake-1"},
		},
		State:     spine.ContractDraftStateDraft,
		CreatedAt: testTime(),
	}
}
