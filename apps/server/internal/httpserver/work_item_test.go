package httpserver_test

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/heurema/goalrail/apps/server/internal/approvedcontract"
	"github.com/heurema/goalrail/apps/server/internal/spine"
	"github.com/heurema/goalrail/apps/server/internal/workitem"
)

func TestPostApprovedContractWorkItemsReturnsPlannedWorkItem(t *testing.T) {
	server := testServer(t)
	approved := createApprovedContract(t, server)

	response := doJSON(t, server.router, http.MethodPost, "/v1/approved-contracts/"+string(approved.ID)+"/work-items", "")
	if response.code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
	}
	for _, hiddenField := range []string{"\"organization_id\"", "\"project_id\""} {
		if strings.Contains(response.body, hiddenField) {
			t.Fatalf("response includes hidden field %s", hiddenField)
		}
	}
	for _, forbiddenField := range []string{"\"run_id\"", "\"receipt_id\"", "\"gate_decision_id\"", "\"proof_id\""} {
		if strings.Contains(response.body, forbiddenField) {
			t.Fatalf("response includes forbidden field %s", forbiddenField)
		}
	}

	var item spine.WorkItem
	decodeJSON(t, response.body, &item)
	if item.Status != spine.WorkItemStatusPlanned {
		t.Fatalf("status = %q, want %q", item.Status, spine.WorkItemStatusPlanned)
	}
	if item.ApprovedContractID != approved.ID {
		t.Fatalf("approved_contract_id = %q, want %q", item.ApprovedContractID, approved.ID)
	}
	if item.Title != approved.Title || item.Summary != approved.IntentSummary {
		t.Fatalf("title/summary not copied from approved contract")
	}
	if !reflect.DeepEqual(item.Scope, approved.Scope) {
		t.Fatalf("scope = %#v, want approved scope %#v", item.Scope, approved.Scope)
	}
	if item.OwnerHint != "" {
		t.Fatalf("owner_hint = %q, want empty advisory hint", item.OwnerHint)
	}
	if !hasSourceRef(item.SourceRefs, workitem.SourceRefKindApprovedContract, string(approved.ID)) {
		t.Fatalf("source_refs = %#v, want approved_contract ref", item.SourceRefs)
	}

	stored, ok, err := server.workItems.Get(context.Background(), item.ID)
	if err != nil {
		t.Fatalf("workItems.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("work item not stored")
	}
	if stored.ID != item.ID {
		t.Fatalf("stored id = %q, want %q", stored.ID, item.ID)
	}
}

func TestPostApprovedContractWorkItemsAppendsWorkItemCreatedEvent(t *testing.T) {
	server := testServer(t)
	approved := createApprovedContract(t, server)

	response := doJSON(t, server.router, http.MethodPost, "/v1/approved-contracts/"+string(approved.ID)+"/work-items", "")
	if response.code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
	}

	if got := countEventType(server.events.Events(), workitem.EventTypeWorkItemCreated); got != 1 {
		t.Fatalf("work_item.created events = %d, want 1", got)
	}
	event := server.events.Events()[len(server.events.Events())-1]
	if event.Type != workitem.EventTypeWorkItemCreated {
		t.Fatalf("event type = %q, want work_item.created", event.Type)
	}
	if event.EntityType != workitem.EntityTypeWorkItem {
		t.Fatalf("entity type = %q, want WorkItem", event.EntityType)
	}
	if event.OrganizationID != approved.OrganizationID || event.ProjectID != approved.ProjectID || event.RepoBindingID != approved.RepoBindingID {
		t.Fatalf("event context = %q/%q/%q, want approved context %q/%q/%q", event.OrganizationID, event.ProjectID, event.RepoBindingID, approved.OrganizationID, approved.ProjectID, approved.RepoBindingID)
	}
	var payload struct {
		WorkItemID         spine.WorkItemID         `json:"work_item_id"`
		ApprovedContractID spine.ApprovedContractID `json:"approved_contract_id"`
		RepoBindingID      spine.RepoBindingID      `json:"repo_binding_id"`
		Status             spine.WorkItemStatus     `json:"status"`
		SourceRefs         []spine.SourceRef        `json:"source_refs"`
	}
	decodeJSON(t, string(event.Payload), &payload)
	if payload.WorkItemID == "" {
		t.Fatal("payload work_item_id is empty")
	}
	if payload.ApprovedContractID != approved.ID || payload.RepoBindingID != approved.RepoBindingID {
		t.Fatalf("payload ids = %q/%q, want approved/repo ids", payload.ApprovedContractID, payload.RepoBindingID)
	}
	if payload.Status != spine.WorkItemStatusPlanned {
		t.Fatalf("payload status = %q, want planned", payload.Status)
	}
	if !hasSourceRef(payload.SourceRefs, workitem.SourceRefKindApprovedContract, string(approved.ID)) {
		t.Fatalf("payload source_refs = %#v, want approved_contract ref", payload.SourceRefs)
	}
	assertNoForbiddenWorkItemSideEffects(t, server.events.Events())
}

func TestPostApprovedContractWorkItemsUnknownApprovedContractReturnsNotFound(t *testing.T) {
	server := testServer(t)

	response := doJSON(t, server.router, http.MethodPost, "/v1/approved-contracts/missing/work-items", "")
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

func TestPostApprovedContractWorkItemsRejectsNotApprovedState(t *testing.T) {
	server := testServer(t)
	approved := validHTTPApprovedContract()
	approved.State = spine.ApprovedContractState("draft")
	if err := server.approvedContracts.Create(context.Background(), approved); err != nil {
		t.Fatalf("approvedContracts.Create() error = %v", err)
	}

	response := doJSON(t, server.router, http.MethodPost, "/v1/approved-contracts/"+string(approved.ID)+"/work-items", "")
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

func TestPostApprovedContractWorkItemsRejectsIncompleteApprovedContract(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*spine.ApprovedContract)
		reason string
	}{
		{name: "missing_scope", mutate: func(approved *spine.ApprovedContract) { approved.Scope = nil }, reason: workitem.ReasonMissingScope},
		{name: "missing_acceptance", mutate: func(approved *spine.ApprovedContract) { approved.AcceptanceCriteria = []string{} }, reason: workitem.ReasonMissingAcceptanceCriteria},
		{name: "missing_proof", mutate: func(approved *spine.ApprovedContract) { approved.ProofExpectations = []string{" "} }, reason: workitem.ReasonMissingProofExpectations},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := testServer(t)
			approved := validHTTPApprovedContract()
			approved.ID = spine.ApprovedContractID("approved-contract-" + tt.name)
			approved.ContractDraftID = spine.ContractDraftID("contract-draft-" + tt.name)
			tt.mutate(&approved)
			if err := server.approvedContracts.Create(context.Background(), approved); err != nil {
				t.Fatalf("approvedContracts.Create() error = %v", err)
			}

			response := doJSON(t, server.router, http.MethodPost, "/v1/approved-contracts/"+string(approved.ID)+"/work-items", "")
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
				t.Fatalf("error message = %q, want %q", body.Error.Message, tt.reason)
			}
		})
	}
}

func TestPostApprovedContractWorkItemsRejectsDuplicatePlanning(t *testing.T) {
	server := testServer(t)
	approved := createApprovedContract(t, server)
	first := doJSON(t, server.router, http.MethodPost, "/v1/approved-contracts/"+string(approved.ID)+"/work-items", "")
	if first.code != http.StatusCreated {
		t.Fatalf("first status = %d, want %d: %s", first.code, http.StatusCreated, first.body)
	}

	second := doJSON(t, server.router, http.MethodPost, "/v1/approved-contracts/"+string(approved.ID)+"/work-items", "")
	if second.code != http.StatusConflict {
		t.Fatalf("second status = %d, want %d: %s", second.code, http.StatusConflict, second.body)
	}
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeJSON(t, second.body, &body)
	if body.Error.Code != "already_planned" {
		t.Fatalf("error code = %q, want already_planned", body.Error.Code)
	}
}

func TestPostApprovedContractWorkItemsDoesNotMutateApprovedContract(t *testing.T) {
	server := testServer(t)
	approved := createApprovedContract(t, server)
	before, ok, err := server.approvedContracts.Get(context.Background(), approved.ID)
	if err != nil {
		t.Fatalf("approvedContracts.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("approved contract missing before planning")
	}

	response := doJSON(t, server.router, http.MethodPost, "/v1/approved-contracts/"+string(approved.ID)+"/work-items", "")
	if response.code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
	}

	after, ok, err := server.approvedContracts.Get(context.Background(), approved.ID)
	if err != nil {
		t.Fatalf("approvedContracts.Get() after error = %v", err)
	}
	if !ok {
		t.Fatal("approved contract missing after planning")
	}
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("approved contract mutated: %#v want %#v", after, before)
	}
}

func TestPostApprovedContractWorkItemsRejectsUnknownJSONField(t *testing.T) {
	server := testServer(t)
	approved := createApprovedContract(t, server)

	response := doJSON(t, server.router, http.MethodPost, "/v1/approved-contracts/"+string(approved.ID)+"/work-items", `{"unexpected":true}`)
	if response.code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusBadRequest, response.body)
	}
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeJSON(t, response.body, &body)
	if body.Error.Code != "invalid_json" {
		t.Fatalf("error code = %q, want invalid_json", body.Error.Code)
	}
}

func TestFullFlowCreatesPlannedWorkItemOnly(t *testing.T) {
	server := testServer(t)
	approved := createApprovedContract(t, server)

	response := doJSON(t, server.router, http.MethodPost, "/v1/approved-contracts/"+string(approved.ID)+"/work-items", "")
	if response.code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
	}
	var item spine.WorkItem
	decodeJSON(t, response.body, &item)
	if item.Status != spine.WorkItemStatusPlanned {
		t.Fatalf("status = %q, want planned", item.Status)
	}
	assertNoForbiddenWorkItemSideEffects(t, server.events.Events())
}

func createApprovedContract(t *testing.T, server testServerDeps) spine.ApprovedContract {
	t.Helper()

	draft := createReadyContractDraft(t, server)
	response := doJSON(t, server.router, http.MethodPost, "/v1/contract-drafts/"+string(draft.ID)+"/approve", approveContractJSON())
	if response.code != http.StatusCreated {
		t.Fatalf("approve status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
	}
	var approved spine.ApprovedContract
	decodeJSON(t, response.body, &approved)
	stored, ok, err := server.approvedContracts.Get(context.Background(), approved.ID)
	if err != nil {
		t.Fatalf("approvedContracts.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("approved contract not stored")
	}
	return stored
}

func validHTTPApprovedContract() spine.ApprovedContract {
	return spine.ApprovedContract{
		ID:                 "approved-contract-1",
		OrganizationID:     "018f0000-0000-7000-8000-000000000002",
		ProjectID:          "018f0000-0000-7000-8000-000000000003",
		ContractDraftID:    "contract-draft-1",
		ContractSeedID:     "contract-seed-1",
		GoalID:             "goal-1",
		RepoBindingID:      "018f0000-0000-7000-8000-000000000004",
		Title:              "Refactor CSV export filters",
		IntentSummary:      "Current code duplicates filter logic.",
		Scope:              []string{"Refactor duplicate CSV export filter logic"},
		AcceptanceCriteria: []string{"Existing CSV export behavior is preserved"},
		ProofExpectations:  []string{"Provide evidence that acceptance criteria were checked."},
		ApprovedBy:         spine.ActorRef{Kind: "user", ID: "dev_approver"},
		ApprovedAt:         testTime(),
		SourceRefs: []spine.SourceRef{
			{Kind: approvedcontract.SourceRefKindContractDraft, ID: "contract-draft-1"},
			{Kind: "contract_seed", ID: "contract-seed-1"},
			{Kind: "goal", ID: "goal-1"},
		},
		State: spine.ApprovedContractStateApproved,
	}
}

func assertNoForbiddenWorkItemSideEffects(t *testing.T, events []spine.Event) {
	t.Helper()

	forbidden := map[string]bool{
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

func TestWorkItemResponseJSONDoesNotExposeContext(t *testing.T) {
	item := spine.WorkItem{
		ID:             "work-item-1",
		OrganizationID: "organization-1",
		ProjectID:      "project-1",
	}
	encoded, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if strings.Contains(string(encoded), "organization_id") || strings.Contains(string(encoded), "project_id") {
		t.Fatalf("encoded work item exposes internal context: %s", encoded)
	}
}
