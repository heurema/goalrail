package httpserver_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

func TestAgentPullLoopServerSmokeThroughApprovedContract(t *testing.T) {
	server := testServer(t)

	intakeID := createIntake(t, server, validIntakeJSON)
	created := promoteIntake(t, server, intakeID)

	continuation := doJSON(t, server.router, http.MethodPost, "/v1/goals/"+string(created.ID)+"/continuation", "")
	if continuation.code != http.StatusOK {
		t.Fatalf("continuation status = %d, want %d: %s", continuation.code, http.StatusOK, continuation.body)
	}
	var continued struct {
		GoalID               spine.GoalID               `json:"goal_id"`
		State                spine.GoalState            `json:"state"`
		ClarificationRequest spine.ClarificationRequest `json:"clarification_request"`
	}
	decodeJSON(t, continuation.body, &continued)
	if continued.GoalID != created.ID || continued.State != spine.GoalStateNeedsClarification {
		t.Fatalf("continuation = %#v, want needs_clarification for goal", continued)
	}
	if continued.ClarificationRequest.ID == "" || len(continued.ClarificationRequest.Questions) == 0 {
		t.Fatalf("clarification request = %#v, want open request with questions", continued.ClarificationRequest)
	}

	answer := doJSON(
		t,
		server.router,
		http.MethodPost,
		"/v1/clarifications/"+string(continued.ClarificationRequest.ID)+"/answers/continuation",
		workAnswerSubmissionJSONWithValues(continued.ClarificationRequest, map[spine.ClarificationMapsTo]string{
			spine.ClarificationMapsToGoalScopeHint:      "Refactor duplicate CSV export filter logic",
			spine.ClarificationMapsToGoalAcceptanceHint: "Existing CSV export behavior is preserved",
		}),
	)
	if answer.code != http.StatusOK {
		t.Fatalf("answer continuation status = %d, want %d: %s", answer.code, http.StatusOK, answer.body)
	}
	var answered struct {
		GoalID spine.GoalID    `json:"goal_id"`
		State  spine.GoalState `json:"state"`
	}
	decodeJSON(t, answer.body, &answered)
	if answered.GoalID != created.ID || answered.State != spine.GoalStateReadyForContractSeed {
		t.Fatalf("answer continuation = %#v, want ready_for_contract_seed", answered)
	}

	contractCreate := doJSON(t, server.router, http.MethodPost, "/v1/contracts", contractCreateJSONWithContext(created.ID, created.ProjectID, created.RepoBindingID))
	if contractCreate.code != http.StatusCreated {
		t.Fatalf("contract create status = %d, want %d: %s", contractCreate.code, http.StatusCreated, contractCreate.body)
	}
	var contract spine.Contract
	decodeJSON(t, contractCreate.body, &contract)
	if contract.State != spine.ContractStateDraft {
		t.Fatalf("contract state = %q, want draft", contract.State)
	}

	contractUpdate := doJSON(
		t,
		server.router,
		http.MethodPatch,
		"/v1/contracts/"+string(contract.ID),
		updateDraftJSONWithContext(string(contract.ProjectID), string(contract.RepoBindingID), `{"proposed_scope":["Refactor duplicate CSV export filter logic"],"proposed_acceptance_criteria":["Existing CSV export behavior is preserved"],"proposed_proof_expectations":["CLI and server tests pass"]}`),
	)
	if contractUpdate.code != http.StatusOK {
		t.Fatalf("contract update status = %d, want %d: %s", contractUpdate.code, http.StatusOK, contractUpdate.body)
	}
	decodeJSON(t, contractUpdate.body, &contract)
	if contract.State != spine.ContractStateDraft {
		t.Fatalf("updated contract state = %q, want draft", contract.State)
	}

	submit := doJSON(t, server.router, http.MethodPost, "/v1/contracts/"+string(contract.ID)+"/submissions", readyForApprovalJSONWithContext(string(contract.ProjectID), string(contract.RepoBindingID)))
	if submit.code != http.StatusOK {
		t.Fatalf("contract submit status = %d, want %d: %s", submit.code, http.StatusOK, submit.body)
	}
	decodeJSON(t, submit.body, &contract)
	if contract.State != spine.ContractStateReadyForApproval {
		t.Fatalf("submitted contract state = %q, want ready_for_approval", contract.State)
	}

	approve := doJSON(t, server.router, http.MethodPost, "/v1/contracts/"+string(contract.ID)+"/approvals", approveContractJSONWithContext(string(contract.ProjectID), string(contract.RepoBindingID)))
	if approve.code != http.StatusCreated {
		t.Fatalf("contract approve status = %d, want %d: %s", approve.code, http.StatusCreated, approve.body)
	}
	decodeJSON(t, approve.body, &contract)
	if contract.State != spine.ContractStateApproved || contract.ApprovedSnapshotID == nil {
		t.Fatalf("approved contract = %#v, want approved state with snapshot", contract)
	}
	if len(server.approvedContracts.approved) != 1 {
		t.Fatalf("approved contracts = %d, want 1", len(server.approvedContracts.approved))
	}
	if len(server.workItems.items) != 0 {
		t.Fatalf("work items = %d, want 0 after approval", len(server.workItems.items))
	}
	assertNoPlanningStores(t, server)
	assertNoForbiddenApprovalSideEffects(t, server.events.Events())
}

func contractCreateJSONWithContext(goalID spine.GoalID, projectID spine.ProjectID, repoBindingID spine.RepoBindingID) string {
	return fmt.Sprintf(`{"goal_id":%q,"project_id":%q,"repo_binding_id":%q}`, goalID, projectID, repoBindingID)
}
