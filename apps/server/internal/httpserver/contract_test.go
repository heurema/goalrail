package httpserver_test

import (
	"context"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/heurema/goalrail/apps/server/internal/auth"
	"github.com/heurema/goalrail/apps/server/internal/contractdraft"
	"github.com/heurema/goalrail/apps/server/internal/contractseed"
	"github.com/heurema/goalrail/apps/server/internal/spine"
)

func TestPostContractsCreatesDraftContract(t *testing.T) {
	server := testServer(t)
	goal := createReadyForContractSeedGoal(t, server)

	response := doJSON(t, server.router, http.MethodPost, "/v1/contracts", createContractJSON(goal.ID))
	if response.code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
	}
	for _, hiddenField := range []string{"\"organization_id\"", "\"project_id\""} {
		if strings.Contains(response.body, hiddenField) {
			t.Fatalf("response includes hidden field %s", hiddenField)
		}
	}
	for _, forbiddenField := range []string{"\"work_item_id\"", "\"run_id\"", "\"receipt_id\"", "\"gate_decision_id\"", "\"proof_id\""} {
		if strings.Contains(response.body, forbiddenField) {
			t.Fatalf("response includes forbidden field %s", forbiddenField)
		}
	}

	var contract spine.Contract
	decodeJSON(t, response.body, &contract)
	if contract.ID == "" {
		t.Fatal("contract id is empty")
	}
	if contract.State != spine.ContractStateDraft {
		t.Fatalf("state = %q, want %q", contract.State, spine.ContractStateDraft)
	}
	if contract.GoalID != goal.ID {
		t.Fatalf("goal_id = %q, want %q", contract.GoalID, goal.ID)
	}
	if contract.RepoBindingID != goal.RepoBindingID {
		t.Fatalf("repo_binding_id = %q, want %q", contract.RepoBindingID, goal.RepoBindingID)
	}
	if contract.CurrentSeedID == nil {
		t.Fatal("current_seed_id is nil")
	}
	if contract.CurrentDraftID == nil {
		t.Fatal("current_draft_id is nil")
	}
	if contract.ApprovedSnapshotID != nil {
		t.Fatalf("approved_snapshot_id = %v, want nil", contract.ApprovedSnapshotID)
	}

	seed, ok, err := server.contractSeeds.Get(context.Background(), *contract.CurrentSeedID)
	if err != nil {
		t.Fatalf("contractSeeds.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("contract seed not stored")
	}
	if seed.ContractID != contract.ID || seed.GoalID != goal.ID {
		t.Fatalf("seed linkage = %q/%q, want contract %q goal %q", seed.ContractID, seed.GoalID, contract.ID, goal.ID)
	}
	draft, ok, err := server.contractDrafts.Get(context.Background(), *contract.CurrentDraftID)
	if err != nil {
		t.Fatalf("contractDrafts.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("contract draft not stored")
	}
	if draft.ContractID != contract.ID || draft.ContractSeedID != seed.ID {
		t.Fatalf("draft linkage = %q/%q, want contract %q seed %q", draft.ContractID, draft.ContractSeedID, contract.ID, seed.ID)
	}
	if _, ok, err := server.approvedContracts.GetByContractDraftID(context.Background(), draft.ID); err != nil {
		t.Fatalf("approvedContracts.GetByContractDraftID() error = %v", err)
	} else if ok {
		t.Fatal("approved contract was created during contract creation")
	}
	if _, ok, err := server.workItems.GetByApprovedContractID(context.Background(), "approved-contract-1"); err != nil {
		t.Fatalf("workItems.GetByApprovedContractID() error = %v", err)
	} else if ok {
		t.Fatal("work item was created during contract creation")
	}
	if got := countEventType(server.events.Events(), contractseed.EventTypeContractSeedCreated); got != 1 {
		t.Fatalf("contract_seed.created events = %d, want 1", got)
	}
	if got := countEventType(server.events.Events(), contractdraft.EventTypeContractDraftCreated); got != 1 {
		t.Fatalf("contract_draft.created events = %d, want 1", got)
	}
	assertNoForbiddenEventTypes(t, server.events.Events())
}

func TestPostContractsCreateOrReturnsExistingDraft(t *testing.T) {
	server := testServer(t)
	goal := createReadyForContractSeedGoal(t, server)

	first := doJSON(t, server.router, http.MethodPost, "/v1/contracts", createContractJSON(goal.ID))
	if first.code != http.StatusCreated {
		t.Fatalf("first status = %d, want %d: %s", first.code, http.StatusCreated, first.body)
	}
	var firstContract spine.Contract
	decodeJSON(t, first.body, &firstContract)

	second := doJSON(t, server.router, http.MethodPost, "/v1/contracts", createContractJSON(goal.ID))
	if second.code != http.StatusOK {
		t.Fatalf("second status = %d, want %d: %s", second.code, http.StatusOK, second.body)
	}
	var secondContract spine.Contract
	decodeJSON(t, second.body, &secondContract)
	if secondContract.ID != firstContract.ID {
		t.Fatalf("second contract id = %q, want %q", secondContract.ID, firstContract.ID)
	}
	if secondContract.CurrentDraftID == nil || firstContract.CurrentDraftID == nil || *secondContract.CurrentDraftID != *firstContract.CurrentDraftID {
		t.Fatalf("second draft = %v, want %v", secondContract.CurrentDraftID, firstContract.CurrentDraftID)
	}
	if got := len(server.contracts.contracts); got != 1 {
		t.Fatalf("contracts stored = %d, want 1", got)
	}
	if got := len(server.contractSeeds.seeds); got != 1 {
		t.Fatalf("contract seeds stored = %d, want 1", got)
	}
	if got := len(server.contractDrafts.drafts); got != 1 {
		t.Fatalf("contract drafts stored = %d, want 1", got)
	}
	if got := countEventType(server.events.Events(), contractseed.EventTypeContractSeedCreated); got != 1 {
		t.Fatalf("contract_seed.created events = %d, want 1", got)
	}
	if got := countEventType(server.events.Events(), contractdraft.EventTypeContractDraftCreated); got != 1 {
		t.Fatalf("contract_draft.created events = %d, want 1", got)
	}
}

func TestPostContractsRequiresAuthBeforeMutation(t *testing.T) {
	server := testServerWithContinuationAuth(t, fakeHTTPAuthService{meErr: auth.ErrInvalidToken})
	goal := createReadyForContractSeedGoal(t, server)

	response := doJSON(t, server.router, http.MethodPost, "/v1/contracts", createContractJSON(goal.ID))
	if response.code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusUnauthorized, response.body)
	}
	if len(server.contracts.contracts) != 0 || len(server.contractSeeds.seeds) != 0 || len(server.contractDrafts.drafts) != 0 {
		t.Fatal("contract state mutated despite failed auth")
	}
}

func TestPostContractsRejectsOrganizationMismatchBeforeMutation(t *testing.T) {
	server := testServerWithContinuationAuth(t, fakeHTTPAuthService{
		profile: continuationAuthProfile("018f0000-0000-7000-8000-000000009999"),
	})
	goal := createReadyForContractSeedGoal(t, server)

	response := doJSON(t, server.router, http.MethodPost, "/v1/contracts", createContractJSON(goal.ID))
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
	if len(server.contracts.contracts) != 0 || len(server.contractSeeds.seeds) != 0 || len(server.contractDrafts.drafts) != 0 {
		t.Fatal("contract state mutated despite org mismatch")
	}
}

func TestPostContractsMalformedGoalIDReturnsValidationError(t *testing.T) {
	server := testServer(t)

	response := doJSON(t, server.router, http.MethodPost, "/v1/contracts", `{"goal_id":"not-a-uuid"}`)
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
	if len(server.contracts.contracts) != 0 {
		t.Fatal("contract state mutated despite malformed goal_id")
	}
}

func TestPostContractsRejectsNotReadyGoalBeforeMutation(t *testing.T) {
	server := testServer(t)
	goal := spine.Goal{
		ID:             "018f0000-0000-7000-8000-000000000106",
		IntakeID:       "018f0000-0000-7000-8000-000000000105",
		OrganizationID: "018f0000-0000-7000-8000-000000000002",
		ProjectID:      "018f0000-0000-7000-8000-000000000003",
		RepoBindingID:  "018f0000-0000-7000-8000-000000000004",
		Title:          "Incomplete goal",
		Summary:        "Needs more detail.",
		State:          spine.GoalStateNeedsClarification,
		CreatedAt:      testTime(),
	}
	if err := server.goals.Create(context.Background(), goal); err != nil {
		t.Fatalf("goals.Create() error = %v", err)
	}

	response := doJSON(t, server.router, http.MethodPost, "/v1/contracts", createContractJSON(goal.ID))
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
	if len(server.contracts.contracts) != 0 || len(server.contractSeeds.seeds) != 0 || len(server.contractDrafts.drafts) != 0 {
		t.Fatal("contract state mutated despite not-ready goal")
	}
}

func TestPostContractsRejectsRepoBindingMismatchBeforeMutation(t *testing.T) {
	server := testServer(t)
	goal := createReadyForContractSeedGoal(t, server)

	response := doJSON(t, server.router, http.MethodPost, "/v1/contracts", `{
		"goal_id":"`+string(goal.ID)+`",
		"project_id":"`+string(goal.ProjectID)+`",
		"repo_binding_id":"018f0000-0000-7000-8000-000000009999"
	}`)
	if response.code != http.StatusConflict {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusConflict, response.body)
	}
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeJSON(t, response.body, &body)
	if body.Error.Code != "project_context_mismatch" {
		t.Fatalf("error code = %q, want project_context_mismatch", body.Error.Code)
	}
	if len(server.contracts.contracts) != 0 || len(server.contractSeeds.seeds) != 0 || len(server.contractDrafts.drafts) != 0 {
		t.Fatal("contract state mutated despite repo binding mismatch")
	}
}

func TestPostContractsRejectsProjectMismatchBeforeMutation(t *testing.T) {
	server := testServer(t)
	goal := createReadyForContractSeedGoal(t, server)

	response := doJSON(t, server.router, http.MethodPost, "/v1/contracts", `{
		"goal_id":"`+string(goal.ID)+`",
		"project_id":"018f0000-0000-7000-8000-000000009998",
		"repo_binding_id":"`+string(goal.RepoBindingID)+`"
	}`)
	if response.code != http.StatusConflict {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusConflict, response.body)
	}
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeJSON(t, response.body, &body)
	if body.Error.Code != "project_context_mismatch" {
		t.Fatalf("error code = %q, want project_context_mismatch", body.Error.Code)
	}
	if len(server.contracts.contracts) != 0 || len(server.contractSeeds.seeds) != 0 || len(server.contractDrafts.drafts) != 0 {
		t.Fatal("contract state mutated despite project mismatch")
	}
}

func TestGetContractReturnsContractView(t *testing.T) {
	server := testServer(t)
	contract := createContract(t, server)

	response := doJSON(t, server.router, http.MethodGet, "/v1/contracts/"+string(contract.ID), "")
	if response.code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusOK, response.body)
	}

	var got spine.Contract
	decodeJSON(t, response.body, &got)
	if got.ID != contract.ID || got.State != spine.ContractStateDraft {
		t.Fatalf("contract = %#v, want id %q state draft", got, contract.ID)
	}
}

func TestGetContractUnknownReturnsNotFound(t *testing.T) {
	server := testServer(t)

	response := doJSON(t, server.router, http.MethodGet, "/v1/contracts/missing", "")
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

func TestPatchContractUpdatesCurrentDraft(t *testing.T) {
	server := testServer(t)
	contract := createContract(t, server)

	response := doJSON(t, server.router, http.MethodPatch, "/v1/contracts/"+string(contract.ID), updateDraftJSON(`{
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

	var updatedContract spine.Contract
	decodeJSON(t, response.body, &updatedContract)
	if updatedContract.ID != contract.ID || updatedContract.State != spine.ContractStateDraft {
		t.Fatalf("contract = %#v, want same id and draft state", updatedContract)
	}
	if updatedContract.CurrentDraftID == nil {
		t.Fatal("current_draft_id is nil")
	}
	draft, ok, err := server.contractDrafts.Get(context.Background(), *updatedContract.CurrentDraftID)
	if err != nil {
		t.Fatalf("contractDrafts.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("draft not stored")
	}
	if draft.Title != "Reviewed draft title" {
		t.Fatalf("title = %q, want reviewed title", draft.Title)
	}
	if !reflect.DeepEqual(draft.ProposedScope, []string{"Reviewed scope"}) {
		t.Fatalf("proposed_scope = %#v, want reviewed scope", draft.ProposedScope)
	}
	if !reflect.DeepEqual(draft.ProposedAcceptanceCriteria, []string{"Reviewed acceptance"}) {
		t.Fatalf("proposed_acceptance_criteria = %#v, want reviewed acceptance", draft.ProposedAcceptanceCriteria)
	}
}

func TestPatchContractUnknownReturnsNotFound(t *testing.T) {
	server := testServer(t)

	response := doJSON(t, server.router, http.MethodPatch, "/v1/contracts/missing", updateDraftJSON(`{"title":"Reviewed"}`))
	if response.code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusNotFound, response.body)
	}
}

func TestPatchContractRejectsNonDraftState(t *testing.T) {
	server := testServer(t)
	contract := submitContractForApproval(t, server, createContract(t, server).ID)

	response := doJSON(t, server.router, http.MethodPatch, "/v1/contracts/"+string(contract.ID), updateDraftJSON(`{"title":"Reviewed"}`))
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

func TestPostContractSubmissionsMovesContractReadyForApproval(t *testing.T) {
	server := testServer(t)
	contract := createContract(t, server)

	submitted := submitContractForApproval(t, server, contract.ID)
	if submitted.State != spine.ContractStateReadyForApproval {
		t.Fatalf("state = %q, want %q", submitted.State, spine.ContractStateReadyForApproval)
	}
	if submitted.CurrentDraftID == nil {
		t.Fatal("current_draft_id is nil")
	}
	draft, ok, err := server.contractDrafts.Get(context.Background(), *submitted.CurrentDraftID)
	if err != nil {
		t.Fatalf("contractDrafts.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("draft not found")
	}
	if draft.State != spine.ContractDraftStateReadyForApproval {
		t.Fatalf("draft state = %q, want ready_for_approval", draft.State)
	}
	if _, ok, err := server.approvedContracts.GetByContractDraftID(context.Background(), draft.ID); err != nil {
		t.Fatalf("approvedContracts.GetByContractDraftID() error = %v", err)
	} else if ok {
		t.Fatal("approved contract was created during submission")
	}
	assertNoForbiddenEventTypes(t, server.events.Events())
}

func TestPostContractApprovalsCreatesApprovedSnapshot(t *testing.T) {
	server := testServer(t)
	contract := submitContractForApproval(t, server, createContract(t, server).ID)

	response := doJSON(t, server.router, http.MethodPost, "/v1/contracts/"+string(contract.ID)+"/approvals", approveContractJSON())
	if response.code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
	}
	for _, forbiddenField := range []string{"\"work_item_id\"", "\"run_id\"", "\"receipt_id\"", "\"gate_decision_id\"", "\"proof_id\""} {
		if strings.Contains(response.body, forbiddenField) {
			t.Fatalf("response includes forbidden field %s", forbiddenField)
		}
	}

	var approvedContract spine.Contract
	decodeJSON(t, response.body, &approvedContract)
	if approvedContract.State != spine.ContractStateApproved {
		t.Fatalf("state = %q, want approved", approvedContract.State)
	}
	if approvedContract.ApprovedSnapshotID == nil {
		t.Fatal("approved_snapshot_id is nil")
	}
	approved, ok, err := server.approvedContracts.Get(context.Background(), *approvedContract.ApprovedSnapshotID)
	if err != nil {
		t.Fatalf("approvedContracts.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("approved contract not stored")
	}
	if approved.ContractID != approvedContract.ID {
		t.Fatalf("approved contract_id = %q, want %q", approved.ContractID, approvedContract.ID)
	}
	if _, ok, err := server.workItems.GetByApprovedContractID(context.Background(), approved.ID); err != nil {
		t.Fatalf("workItems.GetByApprovedContractID() error = %v", err)
	} else if ok {
		t.Fatal("work item was created during approval")
	}
	assertNoForbiddenApprovalSideEffects(t, server.events.Events())
}

func TestContractLifecycleThroughPlanningFlowUsesPublicContractID(t *testing.T) {
	server := testServer(t)
	approved := createApprovedContract(t, server)

	plan := createPlan(t, server, approved.ContractID)
	if plan.ContractID != approved.ContractID {
		t.Fatalf("plan contract_id = %q, want %q", plan.ContractID, approved.ContractID)
	}
	if plan.ApprovedContractID != approved.ID {
		t.Fatalf("plan approved_contract_id = %q, want %q", plan.ApprovedContractID, approved.ID)
	}
	proposal := submitProposal(t, server, plan.ID, string(approved.ID))
	accepted := acceptProposal(t, server, proposal.ID)
	if len(accepted.CreatedTaskIDs) == 0 {
		t.Fatal("acceptance created no task ids")
	}
	item, ok, err := server.workItems.Get(context.Background(), accepted.CreatedTaskIDs[0])
	if err != nil {
		t.Fatalf("workItems.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("accepted task not stored")
	}
	if item.ContractID != approved.ContractID || item.ApprovedContractID != approved.ID {
		t.Fatalf("work item contract trace = %q/%q, want %q/%q", item.ContractID, item.ApprovedContractID, approved.ContractID, approved.ID)
	}
}

func createContract(t *testing.T, server testServerDeps) spine.Contract {
	t.Helper()

	goal := createReadyForContractSeedGoal(t, server)
	response := doJSON(t, server.router, http.MethodPost, "/v1/contracts", createContractJSON(goal.ID))
	if response.code != http.StatusCreated {
		t.Fatalf("contract create status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
	}
	var contract spine.Contract
	decodeJSON(t, response.body, &contract)
	return contract
}

func submitContractForApproval(t *testing.T, server testServerDeps, contractID spine.ContractID) spine.Contract {
	t.Helper()

	response := doJSON(t, server.router, http.MethodPost, "/v1/contracts/"+string(contractID)+"/submissions", readyForApprovalJSON())
	if response.code != http.StatusOK {
		t.Fatalf("contract submission status = %d, want %d: %s", response.code, http.StatusOK, response.body)
	}
	var contract spine.Contract
	decodeJSON(t, response.body, &contract)
	return contract
}

func approvePublicContract(t *testing.T, server testServerDeps, contractID spine.ContractID) spine.Contract {
	t.Helper()

	response := doJSON(t, server.router, http.MethodPost, "/v1/contracts/"+string(contractID)+"/approvals", approveContractJSON())
	if response.code != http.StatusCreated {
		t.Fatalf("contract approval status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
	}
	var contract spine.Contract
	decodeJSON(t, response.body, &contract)
	return contract
}

func createReadyForContractSeedGoal(t *testing.T, server testServerDeps) spine.Goal {
	t.Helper()

	intakeID := createIntake(t, server, validIntakeJSON)
	created := promoteIntake(t, server, intakeID)

	initialReadiness := doJSON(t, server.router, http.MethodPost, "/v1/goals/"+string(created.ID)+"/readiness", "")
	if initialReadiness.code != http.StatusOK {
		t.Fatalf("initial readiness status = %d, want %d: %s", initialReadiness.code, http.StatusOK, initialReadiness.body)
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
	applyResponse := doJSON(t, server.router, http.MethodPost, "/v1/answers/"+string(answer.ID)+"/applications", applyRequestJSON())
	if applyResponse.code != http.StatusOK {
		t.Fatalf("apply status = %d, want %d: %s", applyResponse.code, http.StatusOK, applyResponse.body)
	}

	recheckResponse := doJSON(t, server.router, http.MethodPost, "/v1/goals/"+string(created.ID)+"/readiness", "")
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

func createContractJSON(goalID spine.GoalID) string {
	return `{"goal_id":"` + string(goalID) + `"}`
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

func approveContractJSON() string {
	return `{
		"approved_by": {
			"kind": "user",
			"id": "dev_approver",
			"display_name": "Approver"
		}
	}`
}

func hasSourceRef(refs []spine.SourceRef, kind string, id string) bool {
	for _, ref := range refs {
		if ref.Kind == kind && ref.ID == id {
			return true
		}
	}
	return false
}

func assertNoForbiddenApprovalSideEffects(t *testing.T, events []spine.Event) {
	t.Helper()

	forbidden := map[string]bool{
		"work_item.created":     true,
		"run.started":           true,
		"receipt.submitted":     true,
		"gate.decision_written": true,
		"proof.created":         true,
	}
	for _, event := range events {
		if forbidden[event.Type] {
			t.Fatalf("forbidden event type appended: %s", event.Type)
		}
	}
}
