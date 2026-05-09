package httpserver_test

import (
	"context"
	"encoding/json"
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

func TestGetContractsListsAuthenticatedOrganizationContracts(t *testing.T) {
	server := testServer(t)
	goal := createReadyForContractSeedGoal(t, server)
	response := doAuthRequest(t, server.router, http.MethodPost, "/v1/contracts", createContractJSON(goal.ID), "Bearer access-token")
	if response.code != http.StatusCreated {
		t.Fatalf("contract create status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
	}
	var created spine.Contract
	decodeJSON(t, response.body, &created)
	otherOrg := created
	otherOrg.ID = "018f0000-0000-7000-8000-000000000c99"
	otherOrg.OrganizationID = "018f0000-0000-7000-8000-000000009999"
	otherOrg.GoalID = "018f0000-0000-7000-8000-000000000299"
	if err := server.contracts.Create(context.Background(), otherOrg); err != nil {
		t.Fatalf("other org contract create error = %v", err)
	}

	listResponse := doAuthRequest(t, server.router, http.MethodGet, "/v1/contracts", "", "Bearer access-token")
	if listResponse.code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", listResponse.code, http.StatusOK, listResponse.body)
	}
	for _, hiddenField := range []string{"\"organization_id\"", "\"project_id\""} {
		if strings.Contains(listResponse.body, hiddenField) {
			t.Fatalf("response includes hidden field %s", hiddenField)
		}
	}
	var body spine.ContractList
	decodeJSON(t, listResponse.body, &body)
	if body.Limit != 50 {
		t.Fatalf("limit = %d, want 50", body.Limit)
	}
	if len(body.Contracts) != 1 {
		t.Fatalf("contracts len = %d, want 1: %#v", len(body.Contracts), body.Contracts)
	}
	if body.Contracts[0].ID != created.ID {
		t.Fatalf("contract id = %q, want %q", body.Contracts[0].ID, created.ID)
	}
}

func TestGetContractsAppliesSupportedFilters(t *testing.T) {
	server := testServer(t)
	matchingGoal := createReadyForContractSeedGoal(t, server)
	createResponse := doAuthRequest(t, server.router, http.MethodPost, "/v1/contracts", createContractJSON(matchingGoal.ID), "Bearer access-token")
	if createResponse.code != http.StatusCreated {
		t.Fatalf("contract create status = %d, want %d: %s", createResponse.code, http.StatusCreated, createResponse.body)
	}
	var matching spine.Contract
	decodeJSON(t, createResponse.body, &matching)
	contracts := []spine.Contract{
		contractForList("018f0000-0000-7000-8000-000000000c21", "018f0000-0000-7000-8000-000000000002", "018f0000-0000-7000-8000-000000009903", matching.RepoBindingID, "018f0000-0000-7000-8000-000000009901", spine.ContractStateDraft),
		contractForList("018f0000-0000-7000-8000-000000000c22", "018f0000-0000-7000-8000-000000000002", matchingGoal.ProjectID, "018f0000-0000-7000-8000-000000009904", "018f0000-0000-7000-8000-000000009902", spine.ContractStateDraft),
		contractForList("018f0000-0000-7000-8000-000000000c23", "018f0000-0000-7000-8000-000000000002", matchingGoal.ProjectID, matching.RepoBindingID, "018f0000-0000-7000-8000-000000009905", spine.ContractStateDraft),
		contractForList("018f0000-0000-7000-8000-000000000c24", "018f0000-0000-7000-8000-000000000002", matchingGoal.ProjectID, matching.RepoBindingID, "018f0000-0000-7000-8000-000000009906", spine.ContractStateApproved),
	}
	for _, contract := range contracts {
		if err := server.contracts.Create(context.Background(), contract); err != nil {
			t.Fatalf("contract store create error = %v", err)
		}
	}

	tests := []struct {
		name         string
		query        string
		forbiddenIDs []spine.ContractID
		wantOnlyID   spine.ContractID
	}{
		{name: "project", query: "project_id=" + string(matchingGoal.ProjectID), forbiddenIDs: []spine.ContractID{"018f0000-0000-7000-8000-000000000c21"}},
		{name: "repo binding", query: "repo_binding_id=" + string(matching.RepoBindingID), forbiddenIDs: []spine.ContractID{"018f0000-0000-7000-8000-000000000c22"}},
		{name: "goal", query: "goal_id=" + string(matching.GoalID), forbiddenIDs: []spine.ContractID{"018f0000-0000-7000-8000-000000000c21", "018f0000-0000-7000-8000-000000000c22", "018f0000-0000-7000-8000-000000000c23", "018f0000-0000-7000-8000-000000000c24"}},
		{name: "state", query: "state=draft"},
		{name: "combined", query: "project_id=" + string(matchingGoal.ProjectID) + "&repo_binding_id=" + string(matching.RepoBindingID) + "&goal_id=" + string(matching.GoalID) + "&state=draft", wantOnlyID: matching.ID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := doAuthRequest(t, server.router, http.MethodGet, "/v1/contracts?"+tt.query, "", "Bearer access-token")
			if response.code != http.StatusOK {
				t.Fatalf("status = %d, want %d: %s", response.code, http.StatusOK, response.body)
			}
			var body spine.ContractList
			decodeJSON(t, response.body, &body)
			if len(body.Contracts) == 0 {
				t.Fatal("contracts len = 0, want matches")
			}
			if tt.wantOnlyID != "" && (len(body.Contracts) != 1 || body.Contracts[0].ID != tt.wantOnlyID) {
				t.Fatalf("contracts = %#v, want only %q", body.Contracts, tt.wantOnlyID)
			}
			for _, listed := range body.Contracts {
				for _, forbiddenID := range tt.forbiddenIDs {
					if listed.ID == forbiddenID {
						t.Fatalf("%s filter returned nonmatching contract: %#v", tt.name, listed)
					}
				}
				if tt.name == "state" && listed.State != spine.ContractStateDraft {
					t.Fatalf("state filter returned %q, want draft", listed.State)
				}
			}
		})
	}
}

func TestGetContractsLimitValidationAndReadOnlyBehavior(t *testing.T) {
	server := testServer(t)
	created := createContract(t, server)
	beforeContracts := len(server.contracts.contracts)
	beforeSeeds := len(server.contractSeeds.seeds)
	beforeDrafts := len(server.contractDrafts.drafts)
	beforeApproved := len(server.approvedContracts.approved)
	beforeWorkItems := len(server.workItems.items)
	beforePlans := len(server.workItemPlans.plans)
	beforeRuns := len(server.runs.runs)
	beforeEvents := len(server.events.Events())

	response := doAuthRequest(t, server.router, http.MethodGet, "/v1/contracts?limit=100", "", "Bearer access-token")
	if response.code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusOK, response.body)
	}
	var body spine.ContractList
	decodeJSON(t, response.body, &body)
	if body.Limit != 100 {
		t.Fatalf("limit = %d, want 100", body.Limit)
	}
	if len(body.Contracts) != 1 || body.Contracts[0].ID != created.ID {
		t.Fatalf("contracts = %#v, want created contract", body.Contracts)
	}
	if len(server.contracts.contracts) != beforeContracts || len(server.contractSeeds.seeds) != beforeSeeds || len(server.contractDrafts.drafts) != beforeDrafts || len(server.approvedContracts.approved) != beforeApproved || len(server.workItems.items) != beforeWorkItems || len(server.workItemPlans.plans) != beforePlans || len(server.runs.runs) != beforeRuns || len(server.events.Events()) != beforeEvents {
		t.Fatal("GET /v1/contracts mutated contract lifecycle, planning, run, or event state")
	}

	for _, path := range []string{"/v1/contracts?limit=nope", "/v1/contracts?limit=101", "/v1/contracts?state=blocked", "/v1/contracts?goal_id=not-a-uuid"} {
		t.Run(path, func(t *testing.T) {
			response := doAuthRequest(t, server.router, http.MethodGet, path, "", "Bearer access-token")
			if response.code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d: %s", response.code, http.StatusBadRequest, response.body)
			}
		})
	}
}

func TestGetContractsRequiresBearerAuth(t *testing.T) {
	server := testServerWithContinuationAuth(t, fakeHTTPAuthService{meErr: auth.ErrInvalidToken})

	response := doAuthRequest(t, server.router, http.MethodGet, "/v1/contracts", "", "")
	if response.code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusUnauthorized, response.body)
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
	beforeContracts := len(server.contracts.contracts)
	beforeSeeds := len(server.contractSeeds.seeds)
	beforeDrafts := len(server.contractDrafts.drafts)
	beforeApproved := len(server.approvedContracts.approved)
	beforeWorkItems := len(server.workItems.items)
	beforePlans := len(server.workItemPlans.plans)
	beforePlanLeases := len(server.workItemLeases.leases)
	beforeProposals := len(server.workItemProposals.proposals)
	beforeCheckoutJobs := len(server.checkoutJobs.jobs)
	beforeCheckoutReceipts := len(server.checkoutReceipts.receipts)
	beforeExecutionJobs := len(server.executionJobs.jobs)
	beforeRuns := len(server.runs.runs)
	beforeCommandPlans := len(server.commandPlans.plans)
	beforeExecutionReceipts := len(server.executionReceipts.receipts)
	beforeEvents := len(server.events.Events())
	goalBefore := server.goals.goals[contract.GoalID]

	response := doAuthRequest(t, server.router, http.MethodGet, "/v1/contracts/"+string(contract.ID), "", "Bearer access-token")
	if response.code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusOK, response.body)
	}
	for _, hiddenField := range []string{"\"organization_id\"", "\"project_id\""} {
		if strings.Contains(response.body, hiddenField) {
			t.Fatalf("response includes hidden field %s", hiddenField)
		}
	}
	for _, forbiddenField := range []string{"\"work_item_id\"", "\"run_id\"", "\"receipt_id\"", "\"execution_receipt_id\"", "\"gate_decision_id\"", "\"proof_id\"", "\"readiness\""} {
		if strings.Contains(response.body, forbiddenField) {
			t.Fatalf("response includes forbidden field %s", forbiddenField)
		}
	}
	for _, field := range []string{
		"id",
		"repo_binding_id",
		"goal_id",
		"state",
		"current_seed_id",
		"current_draft_id",
		"created_at",
		"updated_at",
	} {
		if !strings.Contains(response.body, `"`+field+`"`) {
			t.Fatalf("response missing contract field %q: %s", field, response.body)
		}
	}

	var got spine.Contract
	decodeJSON(t, response.body, &got)
	if got.ID != contract.ID || got.State != spine.ContractStateDraft {
		t.Fatalf("contract = %#v, want id %q state draft", got, contract.ID)
	}
	if got.RepoBindingID != contract.RepoBindingID || got.GoalID != contract.GoalID {
		t.Fatalf("contract linkage = repo %q goal %q, want repo %q goal %q", got.RepoBindingID, got.GoalID, contract.RepoBindingID, contract.GoalID)
	}
	if got.CurrentSeedID == nil || got.CurrentDraftID == nil {
		t.Fatalf("contract missing current seed or draft: %#v", got)
	}
	if len(server.contracts.contracts) != beforeContracts ||
		len(server.contractSeeds.seeds) != beforeSeeds ||
		len(server.contractDrafts.drafts) != beforeDrafts ||
		len(server.approvedContracts.approved) != beforeApproved ||
		len(server.workItems.items) != beforeWorkItems ||
		len(server.workItemPlans.plans) != beforePlans ||
		len(server.workItemLeases.leases) != beforePlanLeases ||
		len(server.workItemProposals.proposals) != beforeProposals ||
		len(server.checkoutJobs.jobs) != beforeCheckoutJobs ||
		len(server.checkoutReceipts.receipts) != beforeCheckoutReceipts ||
		len(server.executionJobs.jobs) != beforeExecutionJobs ||
		len(server.runs.runs) != beforeRuns ||
		len(server.commandPlans.plans) != beforeCommandPlans ||
		len(server.executionReceipts.receipts) != beforeExecutionReceipts ||
		len(server.events.Events()) != beforeEvents ||
		!reflect.DeepEqual(server.goals.goals[contract.GoalID], goalBefore) {
		t.Fatal("GET /v1/contracts/{id} mutated contract lifecycle, planning, run, execution, event, or readiness state")
	}
}

func TestGetContractRequiresBearerAuth(t *testing.T) {
	server := testServerWithContinuationAuth(t, fakeHTTPAuthService{meErr: auth.ErrInvalidToken})
	contract := contractForList("018f0000-0000-7000-8000-000000000c30", "018f0000-0000-7000-8000-000000000002", "018f0000-0000-7000-8000-000000000003", "018f0000-0000-7000-8000-000000000004", "018f0000-0000-7000-8000-000000000230", spine.ContractStateDraft)
	if err := server.contracts.Create(context.Background(), contract); err != nil {
		t.Fatalf("contracts.Create() error = %v", err)
	}

	response := doAuthRequest(t, server.router, http.MethodGet, "/v1/contracts/"+string(contract.ID), "", "")
	if response.code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusUnauthorized, response.body)
	}
}

func TestGetContractRejectsInactiveMembership(t *testing.T) {
	profile := continuationAuthProfile("018f0000-0000-7000-8000-000000000002")
	profile.OrganizationMembership.State = spine.EntityStateInactive
	server := testServerWithContinuationAuth(t, fakeHTTPAuthService{profile: profile})
	contract := contractForList("018f0000-0000-7000-8000-000000000c36", "018f0000-0000-7000-8000-000000000002", "018f0000-0000-7000-8000-000000000003", "018f0000-0000-7000-8000-000000000004", "018f0000-0000-7000-8000-000000000236", spine.ContractStateDraft)
	if err := server.contracts.Create(context.Background(), contract); err != nil {
		t.Fatalf("contracts.Create() error = %v", err)
	}

	response := doAuthRequest(t, server.router, http.MethodGet, "/v1/contracts/"+string(contract.ID), "", "Bearer access-token")
	if response.code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusForbidden, response.body)
	}
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeJSON(t, response.body, &body)
	if body.Error.Code != "membership_required" {
		t.Fatalf("error code = %q, want membership_required", body.Error.Code)
	}
}

func TestGetContractRejectsOtherOrganizationContract(t *testing.T) {
	server := testServerWithContinuationAuth(t, fakeHTTPAuthService{
		profile: continuationAuthProfile("018f0000-0000-7000-8000-000000009999"),
	})
	contract := contractForList("018f0000-0000-7000-8000-000000000c37", "018f0000-0000-7000-8000-000000000002", "018f0000-0000-7000-8000-000000000003", "018f0000-0000-7000-8000-000000000004", "018f0000-0000-7000-8000-000000000237", spine.ContractStateDraft)
	if err := server.contracts.Create(context.Background(), contract); err != nil {
		t.Fatalf("contracts.Create() error = %v", err)
	}

	response := doAuthRequest(t, server.router, http.MethodGet, "/v1/contracts/"+string(contract.ID), "", "Bearer access-token")
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
}

func TestGetContractUnknownReturnsNotFound(t *testing.T) {
	server := testServer(t)

	response := doAuthRequest(t, server.router, http.MethodGet, "/v1/contracts/missing", "", "Bearer access-token")
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

func TestGetContractCurrentDraftReturnsAuthenticatedDraft(t *testing.T) {
	server := testServer(t)
	contract := createContract(t, server)
	if contract.CurrentDraftID == nil {
		t.Fatal("current_draft_id is nil")
	}
	beforeContracts := len(server.contracts.contracts)
	beforeSeeds := len(server.contractSeeds.seeds)
	beforeDrafts := len(server.contractDrafts.drafts)
	beforeApproved := len(server.approvedContracts.approved)
	beforeWorkItems := len(server.workItems.items)
	beforePlans := len(server.workItemPlans.plans)
	beforePlanLeases := len(server.workItemLeases.leases)
	beforeProposals := len(server.workItemProposals.proposals)
	beforeCheckoutJobs := len(server.checkoutJobs.jobs)
	beforeCheckoutReceipts := len(server.checkoutReceipts.receipts)
	beforeExecutionJobs := len(server.executionJobs.jobs)
	beforeRuns := len(server.runs.runs)
	beforeCommandPlans := len(server.commandPlans.plans)
	beforeExecutionReceipts := len(server.executionReceipts.receipts)
	beforeEvents := len(server.events.Events())

	response := doAuthRequest(t, server.router, http.MethodGet, "/v1/contracts/"+string(contract.ID)+"/current-draft", "", "Bearer access-token")
	if response.code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusOK, response.body)
	}
	for _, hiddenField := range []string{"\"organization_id\"", "\"project_id\""} {
		if strings.Contains(response.body, hiddenField) {
			t.Fatalf("response includes hidden field %s", hiddenField)
		}
	}
	for _, field := range []string{
		"id",
		"contract_id",
		"contract_seed_id",
		"goal_id",
		"repo_binding_id",
		"title",
		"intent_summary",
		"proposed_scope",
		"proposed_non_goals",
		"proposed_constraints",
		"proposed_acceptance_criteria",
		"proposed_expected_checks",
		"proposed_proof_expectations",
		"risk_hints",
		"source_refs",
		"state",
		"created_at",
	} {
		if !strings.Contains(response.body, `"`+field+`"`) {
			t.Fatalf("response missing draft field %q: %s", field, response.body)
		}
	}

	var draft spine.ContractDraft
	decodeJSON(t, response.body, &draft)
	if draft.ID != *contract.CurrentDraftID {
		t.Fatalf("draft id = %q, want %q", draft.ID, *contract.CurrentDraftID)
	}
	if draft.ContractID != contract.ID || draft.RepoBindingID != contract.RepoBindingID || draft.GoalID != contract.GoalID {
		t.Fatalf("draft linkage = %#v, want contract/repo/goal from %#v", draft, contract)
	}
	if draft.Title == "" || draft.IntentSummary == "" {
		t.Fatalf("draft title/intent summary missing: %#v", draft)
	}
	if len(draft.ProposedScope) == 0 || len(draft.ProposedAcceptanceCriteria) == 0 || len(draft.ProposedProofExpectations) == 0 {
		t.Fatalf("draft body slices incomplete: %#v", draft)
	}
	if len(server.contracts.contracts) != beforeContracts ||
		len(server.contractSeeds.seeds) != beforeSeeds ||
		len(server.contractDrafts.drafts) != beforeDrafts ||
		len(server.approvedContracts.approved) != beforeApproved ||
		len(server.workItems.items) != beforeWorkItems ||
		len(server.workItemPlans.plans) != beforePlans ||
		len(server.workItemLeases.leases) != beforePlanLeases ||
		len(server.workItemProposals.proposals) != beforeProposals ||
		len(server.checkoutJobs.jobs) != beforeCheckoutJobs ||
		len(server.checkoutReceipts.receipts) != beforeCheckoutReceipts ||
		len(server.executionJobs.jobs) != beforeExecutionJobs ||
		len(server.runs.runs) != beforeRuns ||
		len(server.commandPlans.plans) != beforeCommandPlans ||
		len(server.executionReceipts.receipts) != beforeExecutionReceipts ||
		len(server.events.Events()) != beforeEvents {
		t.Fatal("GET /v1/contracts/{id}/current-draft mutated contract lifecycle, planning, run, execution, or event state")
	}
}

func TestGetContractCurrentDraftRequiresBearerAuth(t *testing.T) {
	server := testServerWithContinuationAuth(t, fakeHTTPAuthService{meErr: auth.ErrInvalidToken})
	contract := contractWithCurrentDraft("018f0000-0000-7000-8000-000000000c31", "018f0000-0000-7000-8000-000000000d31", "018f0000-0000-7000-8000-000000000002")
	if err := server.contracts.Create(context.Background(), contract); err != nil {
		t.Fatalf("contracts.Create() error = %v", err)
	}

	response := doAuthRequest(t, server.router, http.MethodGet, "/v1/contracts/"+string(contract.ID)+"/current-draft", "", "")
	if response.code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusUnauthorized, response.body)
	}
}

func TestGetContractCurrentDraftRejectsInactiveMembership(t *testing.T) {
	profile := continuationAuthProfile("018f0000-0000-7000-8000-000000000002")
	profile.OrganizationMembership.State = spine.EntityStateInactive
	server := testServerWithContinuationAuth(t, fakeHTTPAuthService{profile: profile})
	contract := storeContractWithCurrentDraft(t, server, "018f0000-0000-7000-8000-000000000c32", "018f0000-0000-7000-8000-000000000d32", "018f0000-0000-7000-8000-000000000002")

	response := doAuthRequest(t, server.router, http.MethodGet, "/v1/contracts/"+string(contract.ID)+"/current-draft", "", "Bearer access-token")
	if response.code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusForbidden, response.body)
	}
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeJSON(t, response.body, &body)
	if body.Error.Code != "membership_required" {
		t.Fatalf("error code = %q, want membership_required", body.Error.Code)
	}
}

func TestGetContractCurrentDraftRejectsOtherOrganizationContract(t *testing.T) {
	server := testServerWithContinuationAuth(t, fakeHTTPAuthService{
		profile: continuationAuthProfile("018f0000-0000-7000-8000-000000009999"),
	})
	contract := storeContractWithCurrentDraft(t, server, "018f0000-0000-7000-8000-000000000c33", "018f0000-0000-7000-8000-000000000d33", "018f0000-0000-7000-8000-000000000002")

	response := doAuthRequest(t, server.router, http.MethodGet, "/v1/contracts/"+string(contract.ID)+"/current-draft", "", "Bearer access-token")
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
}

func TestGetContractCurrentDraftRejectsMissingCurrentDraft(t *testing.T) {
	server := testServer(t)
	contract := contractForList("018f0000-0000-7000-8000-000000000c34", "018f0000-0000-7000-8000-000000000002", "018f0000-0000-7000-8000-000000000003", "018f0000-0000-7000-8000-000000000004", "018f0000-0000-7000-8000-000000000234", spine.ContractStateDraft)
	if err := server.contracts.Create(context.Background(), contract); err != nil {
		t.Fatalf("contracts.Create() error = %v", err)
	}

	response := doAuthRequest(t, server.router, http.MethodGet, "/v1/contracts/"+string(contract.ID)+"/current-draft", "", "Bearer access-token")
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

func TestGetContractCurrentDraftRejectsMismatchedDraft(t *testing.T) {
	server := testServer(t)
	draftID := spine.ContractDraftID("018f0000-0000-7000-8000-000000000d35")
	contract := contractWithCurrentDraft("018f0000-0000-7000-8000-000000000c35", draftID, "018f0000-0000-7000-8000-000000000002")
	if err := server.contracts.Create(context.Background(), contract); err != nil {
		t.Fatalf("contracts.Create() error = %v", err)
	}
	if err := server.contractDrafts.Create(context.Background(), spine.ContractDraft{
		ID:             draftID,
		OrganizationID: "018f0000-0000-7000-8000-000000009999",
		ProjectID:      contract.ProjectID,
		RepoBindingID:  contract.RepoBindingID,
		ContractID:     "018f0000-0000-7000-8000-000000000c99",
		ContractSeedID: "018f0000-0000-7000-8000-000000000535",
		GoalID:         contract.GoalID,
		Title:          "Mismatched draft",
		IntentSummary:  "This draft is linked to a different contract.",
		State:          spine.ContractDraftStateDraft,
		CreatedAt:      testTime(),
	}); err != nil {
		t.Fatalf("contractDrafts.Create() error = %v", err)
	}

	response := doAuthRequest(t, server.router, http.MethodGet, "/v1/contracts/"+string(contract.ID)+"/current-draft", "", "Bearer access-token")
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

func TestPatchContractUpdatesCurrentDraft(t *testing.T) {
	server := testServer(t)
	contract := createContract(t, server)

	response := doJSON(t, server.router, http.MethodPatch, "/v1/contracts/"+string(contract.ID), updateDraftJSON(`{
		"title": "Reviewed draft title",
			"intent_summary": "Reviewed summary",
			"proposed_scope": ["Reviewed scope"],
			"proposed_acceptance_criteria": ["Reviewed acceptance"],
			"proposed_non_goals": ["Do not change billing UI"],
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
	if got := countEventType(server.events.Events(), contractdraft.EventTypeContractDraftUpdated); got != 1 {
		t.Fatalf("contract_draft.updated events = %d, want 1", got)
	}
	var updatedPayload struct {
		UpdatedBy spine.ActorRef `json:"updated_by"`
	}
	for _, event := range server.events.Events() {
		if event.Type != contractdraft.EventTypeContractDraftUpdated {
			continue
		}
		if err := json.Unmarshal(event.Payload, &updatedPayload); err != nil {
			t.Fatalf("unmarshal contract_draft.updated payload: %v", err)
		}
	}
	if updatedPayload.UpdatedBy.ID != "018f0000-0000-7000-8000-000000000001" {
		t.Fatalf("updated_by.id = %q, want authenticated user id", updatedPayload.UpdatedBy.ID)
	}
	assertNoForbiddenEventTypes(t, server.events.Events())
}

func TestPatchContractRejectsProjectOrRepoBindingMismatchBeforeMutation(t *testing.T) {
	tests := []struct {
		name string
		body func(spine.Contract) string
	}{
		{
			name: "project",
			body: func(contract spine.Contract) string {
				return updateDraftJSONWithContext("018f0000-0000-7000-8000-000000009998", string(contract.RepoBindingID), `{"title":"Reviewed"}`)
			},
		},
		{
			name: "repo binding",
			body: func(contract spine.Contract) string {
				return updateDraftJSONWithContext("018f0000-0000-7000-8000-000000000003", "018f0000-0000-7000-8000-000000009999", `{"title":"Reviewed"}`)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := testServer(t)
			contract := createContract(t, server)
			eventCountBefore := len(server.events.Events())

			response := doJSON(t, server.router, http.MethodPatch, "/v1/contracts/"+string(contract.ID), tt.body(contract))
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
			if got := len(server.events.Events()); got != eventCountBefore {
				t.Fatalf("events = %d, want %d without update mutation", got, eventCountBefore)
			}
		})
	}
}

func TestPatchContractRejectsOrganizationMismatchBeforeMutation(t *testing.T) {
	server := testServerWithContinuationAuth(t, fakeHTTPAuthService{
		profile: continuationAuthProfile("018f0000-0000-7000-8000-000000009999"),
	})
	draftID := spine.ContractDraftID("contract-draft-1")
	contract := spine.Contract{
		ID:             "contract-1",
		OrganizationID: "018f0000-0000-7000-8000-000000000002",
		ProjectID:      "018f0000-0000-7000-8000-000000000003",
		RepoBindingID:  "018f0000-0000-7000-8000-000000000004",
		State:          spine.ContractStateDraft,
		CurrentDraftID: &draftID,
		GoalID:         "018f0000-0000-7000-8000-000000000006",
	}
	if err := server.contracts.Create(context.Background(), contract); err != nil {
		t.Fatalf("contracts.Create() error = %v", err)
	}

	response := doJSON(t, server.router, http.MethodPatch, "/v1/contracts/"+string(contract.ID), updateDraftJSONWithContext(string(contract.ProjectID), string(contract.RepoBindingID), `{"title":"Reviewed"}`))
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
	if got := len(server.events.Events()); got != 0 {
		t.Fatalf("events = %d, want 0 without update mutation", got)
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
	if got := countEventType(server.events.Events(), contractdraft.EventTypeContractDraftMarkedReadyForApproval); got != 1 {
		t.Fatalf("contract_draft.marked_ready_for_approval events = %d, want 1", got)
	}
	var markedPayload struct {
		MarkedBy spine.ActorRef `json:"marked_by"`
	}
	for _, event := range server.events.Events() {
		if event.Type != contractdraft.EventTypeContractDraftMarkedReadyForApproval {
			continue
		}
		if err := json.Unmarshal(event.Payload, &markedPayload); err != nil {
			t.Fatalf("unmarshal contract_draft.marked_ready_for_approval payload: %v", err)
		}
	}
	if markedPayload.MarkedBy.ID != "018f0000-0000-7000-8000-000000000001" {
		t.Fatalf("marked_by.id = %q, want authenticated user id", markedPayload.MarkedBy.ID)
	}
	assertNoForbiddenEventTypes(t, server.events.Events())
}

func TestPostContractSubmissionsRequiresAuthBeforeMutation(t *testing.T) {
	server := testServerWithContinuationAuth(t, fakeHTTPAuthService{meErr: auth.ErrInvalidToken})
	contract := spine.Contract{
		ID:             "contract-1",
		OrganizationID: "018f0000-0000-7000-8000-000000000002",
		ProjectID:      "018f0000-0000-7000-8000-000000000003",
		RepoBindingID:  "018f0000-0000-7000-8000-000000000004",
		State:          spine.ContractStateDraft,
	}
	if err := server.contracts.Create(context.Background(), contract); err != nil {
		t.Fatalf("contracts.Create() error = %v", err)
	}

	response := doJSON(t, server.router, http.MethodPost, "/v1/contracts/"+string(contract.ID)+"/submissions", readyForApprovalJSON())
	if response.code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusUnauthorized, response.body)
	}
	if got := countEventType(server.events.Events(), contractdraft.EventTypeContractDraftMarkedReadyForApproval); got != 0 {
		t.Fatalf("contract_draft.marked_ready_for_approval events = %d, want 0", got)
	}
}

func TestPostContractSubmissionsRejectsOrganizationMismatchBeforeMutation(t *testing.T) {
	server := testServerWithContinuationAuth(t, fakeHTTPAuthService{
		profile: continuationAuthProfile("018f0000-0000-7000-8000-000000009999"),
	})
	draftID := spine.ContractDraftID("contract-draft-1")
	contract := spine.Contract{
		ID:             "contract-1",
		OrganizationID: "018f0000-0000-7000-8000-000000000002",
		ProjectID:      "018f0000-0000-7000-8000-000000000003",
		RepoBindingID:  "018f0000-0000-7000-8000-000000000004",
		State:          spine.ContractStateDraft,
		CurrentDraftID: &draftID,
	}
	if err := server.contracts.Create(context.Background(), contract); err != nil {
		t.Fatalf("contracts.Create() error = %v", err)
	}

	response := doJSON(t, server.router, http.MethodPost, "/v1/contracts/"+string(contract.ID)+"/submissions", readyForApprovalJSONWithContext(string(contract.ProjectID), string(contract.RepoBindingID)))
	if response.code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusForbidden, response.body)
	}
	if got := countEventType(server.events.Events(), contractdraft.EventTypeContractDraftMarkedReadyForApproval); got != 0 {
		t.Fatalf("contract_draft.marked_ready_for_approval events = %d, want 0", got)
	}
}

func TestPostContractSubmissionsRejectsProjectOrRepoBindingMismatchBeforeMutation(t *testing.T) {
	tests := []struct {
		name string
		body func(spine.Contract) string
	}{
		{
			name: "project",
			body: func(contract spine.Contract) string {
				return readyForApprovalJSONWithContext("018f0000-0000-7000-8000-000000009998", string(contract.RepoBindingID))
			},
		},
		{
			name: "repo binding",
			body: func(contract spine.Contract) string {
				return readyForApprovalJSONWithContext(string(contract.ProjectID), "018f0000-0000-7000-8000-000000009999")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := testServer(t)
			contract := createContract(t, server)
			eventCountBefore := len(server.events.Events())

			response := doJSON(t, server.router, http.MethodPost, "/v1/contracts/"+string(contract.ID)+"/submissions", tt.body(contract))
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
			if got := len(server.events.Events()); got != eventCountBefore {
				t.Fatalf("events = %d, want %d without submission mutation", got, eventCountBefore)
			}
		})
	}
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
	if approved.ApprovedBy.ID != "018f0000-0000-7000-8000-000000000001" {
		t.Fatalf("approved_by.id = %q, want authenticated user id", approved.ApprovedBy.ID)
	}
	assertNoPlanningStores(t, server)
	assertNoForbiddenApprovalSideEffects(t, server.events.Events())
}

func TestPostContractApprovalsRequiresAuthBeforeMutation(t *testing.T) {
	server := testServerWithContinuationAuth(t, fakeHTTPAuthService{meErr: auth.ErrInvalidToken})
	contract := spine.Contract{
		ID:             "contract-1",
		OrganizationID: "018f0000-0000-7000-8000-000000000002",
		ProjectID:      "018f0000-0000-7000-8000-000000000003",
		RepoBindingID:  "018f0000-0000-7000-8000-000000000004",
		State:          spine.ContractStateReadyForApproval,
	}
	if err := server.contracts.Create(context.Background(), contract); err != nil {
		t.Fatalf("contracts.Create() error = %v", err)
	}

	response := doJSON(t, server.router, http.MethodPost, "/v1/contracts/"+string(contract.ID)+"/approvals", approveContractJSON())
	if response.code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusUnauthorized, response.body)
	}
	if len(server.approvedContracts.approved) != 0 {
		t.Fatal("approved contract was created despite auth failure")
	}
}

func TestPostContractApprovalsRejectsOrganizationMismatchBeforeMutation(t *testing.T) {
	server := testServerWithContinuationAuth(t, fakeHTTPAuthService{
		profile: continuationAuthProfile("018f0000-0000-7000-8000-000000009999"),
	})
	draftID := spine.ContractDraftID("contract-draft-1")
	contract := spine.Contract{
		ID:             "contract-1",
		OrganizationID: "018f0000-0000-7000-8000-000000000002",
		ProjectID:      "018f0000-0000-7000-8000-000000000003",
		RepoBindingID:  "018f0000-0000-7000-8000-000000000004",
		State:          spine.ContractStateReadyForApproval,
		CurrentDraftID: &draftID,
	}
	if err := server.contracts.Create(context.Background(), contract); err != nil {
		t.Fatalf("contracts.Create() error = %v", err)
	}

	response := doJSON(t, server.router, http.MethodPost, "/v1/contracts/"+string(contract.ID)+"/approvals", approveContractJSONWithContext(string(contract.ProjectID), string(contract.RepoBindingID)))
	if response.code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusForbidden, response.body)
	}
	if len(server.approvedContracts.approved) != 0 {
		t.Fatal("approved contract was created despite org mismatch")
	}
}

func TestPostContractApprovalsRejectsProjectOrRepoBindingMismatchBeforeMutation(t *testing.T) {
	tests := []struct {
		name string
		body func(spine.Contract) string
	}{
		{
			name: "project",
			body: func(contract spine.Contract) string {
				return approveContractJSONWithContext("018f0000-0000-7000-8000-000000009998", string(contract.RepoBindingID))
			},
		},
		{
			name: "repo binding",
			body: func(contract spine.Contract) string {
				return approveContractJSONWithContext(string(contract.ProjectID), "018f0000-0000-7000-8000-000000009999")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := testServer(t)
			contract := submitContractForApproval(t, server, createContract(t, server).ID)
			eventCountBefore := len(server.events.Events())

			response := doJSON(t, server.router, http.MethodPost, "/v1/contracts/"+string(contract.ID)+"/approvals", tt.body(contract))
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
			if got := len(server.events.Events()); got != eventCountBefore {
				t.Fatalf("events = %d, want %d without approval mutation", got, eventCountBefore)
			}
			if len(server.approvedContracts.approved) != 0 {
				t.Fatal("approved contract was created despite context mismatch")
			}
		})
	}
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

func contractForList(id spine.ContractID, organizationID spine.OrganizationID, projectID spine.ProjectID, repoBindingID spine.RepoBindingID, goalID spine.GoalID, state spine.ContractState) spine.Contract {
	return spine.Contract{
		ID:             id,
		OrganizationID: organizationID,
		ProjectID:      projectID,
		RepoBindingID:  repoBindingID,
		GoalID:         goalID,
		State:          state,
		CreatedAt:      testTime(),
		UpdatedAt:      testTime(),
	}
}

func contractWithCurrentDraft(id spine.ContractID, draftID spine.ContractDraftID, organizationID spine.OrganizationID) spine.Contract {
	contract := contractForList(
		id,
		organizationID,
		"018f0000-0000-7000-8000-000000000003",
		"018f0000-0000-7000-8000-000000000004",
		"018f0000-0000-7000-8000-000000000236",
		spine.ContractStateDraft,
	)
	contract.CurrentDraftID = &draftID
	seedID := spine.ContractSeedID("018f0000-0000-7000-8000-000000000536")
	contract.CurrentSeedID = &seedID
	return contract
}

func storeContractWithCurrentDraft(t *testing.T, server testServerDeps, id spine.ContractID, draftID spine.ContractDraftID, organizationID spine.OrganizationID) spine.Contract {
	t.Helper()

	contract := contractWithCurrentDraft(id, draftID, organizationID)
	if err := server.contracts.Create(context.Background(), contract); err != nil {
		t.Fatalf("contracts.Create() error = %v", err)
	}
	if err := server.contractDrafts.Create(context.Background(), spine.ContractDraft{
		ID:                         draftID,
		OrganizationID:             organizationID,
		ProjectID:                  contract.ProjectID,
		RepoBindingID:              contract.RepoBindingID,
		ContractID:                 contract.ID,
		ContractSeedID:             *contract.CurrentSeedID,
		GoalID:                     contract.GoalID,
		Title:                      "Stored current draft",
		IntentSummary:              "Stored current draft body.",
		ProposedScope:              []string{"Keep current draft display read-only."},
		ProposedNonGoals:           []string{"No lifecycle mutation."},
		ProposedConstraints:        []string{"Backend API only."},
		ProposedAcceptanceCriteria: []string{"Current draft body is returned."},
		ProposedExpectedChecks:     []string{"go test ./..."},
		ProposedProofExpectations:  []string{"Validation output is reported."},
		RiskHints:                  []string{"Read-only auth path."},
		SourceRefs:                 []spine.SourceRef{{Kind: "contract_seed", ID: string(*contract.CurrentSeedID)}},
		State:                      spine.ContractDraftStateDraft,
		CreatedAt:                  testTime(),
	}); err != nil {
		t.Fatalf("contractDrafts.Create() error = %v", err)
	}
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

func updateDraftJSONWithContext(projectID string, repoBindingID string, changes string) string {
	return `{
		"project_id": "` + projectID + `",
		"repo_binding_id": "` + repoBindingID + `",
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

func readyForApprovalJSONWithContext(projectID string, repoBindingID string) string {
	return `{
		"project_id": "` + projectID + `",
		"repo_binding_id": "` + repoBindingID + `",
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

func approveContractJSONWithContext(projectID string, repoBindingID string) string {
	return `{
		"project_id": "` + projectID + `",
		"repo_binding_id": "` + repoBindingID + `",
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

func assertNoPlanningStores(t *testing.T, server testServerDeps) {
	t.Helper()

	if len(server.workItemPlans.plans) != 0 {
		t.Fatalf("work item plans = %d, want 0 during contract approval", len(server.workItemPlans.plans))
	}
	if len(server.workItemLeases.leases) != 0 {
		t.Fatalf("work item plan leases = %d, want 0 during contract approval", len(server.workItemLeases.leases))
	}
	if len(server.workItemProposals.proposals) != 0 {
		t.Fatalf("work item plan proposals = %d, want 0 during contract approval", len(server.workItemProposals.proposals))
	}
}
