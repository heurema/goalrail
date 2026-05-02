package httpserver_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/heurema/goalrail/apps/server/internal/actor"
	"github.com/heurema/goalrail/apps/server/internal/approvedcontract"
	"github.com/heurema/goalrail/apps/server/internal/httpserver"
	"github.com/heurema/goalrail/apps/server/internal/spine"
)

func TestPostContractDraftApproveReturnsApprovedContract(t *testing.T) {
	server := testServer(t)
	draft := createReadyContractDraft(t, server)

	response := doJSON(t, server.router, http.MethodPost, "/v1/contract-drafts/"+string(draft.ID)+"/approvals", approveContractJSON())
	if response.code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
	}
	for _, hiddenField := range []string{"\"organization_id\"", "\"project_id\""} {
		if strings.Contains(response.body, hiddenField) {
			t.Fatalf("response includes hidden field %s", hiddenField)
		}
	}
	for _, forbiddenField := range []string{"\"work_item_id\"", "\"proof_id\"", "\"gate_decision_id\""} {
		if strings.Contains(response.body, forbiddenField) {
			t.Fatalf("response includes forbidden field %s", forbiddenField)
		}
	}

	var approved spine.ApprovedContract
	decodeJSON(t, response.body, &approved)
	if approved.State != spine.ApprovedContractStateApproved {
		t.Fatalf("state = %q, want %q", approved.State, spine.ApprovedContractStateApproved)
	}
	if approved.ContractDraftID != draft.ID {
		t.Fatalf("contract_draft_id = %q, want %q", approved.ContractDraftID, draft.ID)
	}
	if approved.Scope[0] != draft.ProposedScope[0] {
		t.Fatalf("scope = %#v, want draft scope %#v", approved.Scope, draft.ProposedScope)
	}
	if approved.ApprovedBy.Kind != "user" || approved.ApprovedBy.ID != "dev_approver" {
		t.Fatalf("approved_by = %#v, want payload approver", approved.ApprovedBy)
	}

	storedDraft, ok, err := server.contractDrafts.Get(context.Background(), draft.ID)
	if err != nil {
		t.Fatalf("contractDrafts.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("contract draft missing after approval")
	}
	if storedDraft.State != spine.ContractDraftStateReadyForApproval {
		t.Fatalf("draft state = %q, want ready_for_approval", storedDraft.State)
	}
	storedApproved, ok, err := server.approvedContracts.GetByContractDraftID(context.Background(), draft.ID)
	if err != nil {
		t.Fatalf("approvedContracts.GetByContractDraftID() error = %v", err)
	}
	if !ok {
		t.Fatal("approved contract missing after approval")
	}
	if storedApproved.ApprovedBy != approved.ApprovedBy {
		t.Fatalf("stored approved_by = %#v, want response approved_by %#v", storedApproved.ApprovedBy, approved.ApprovedBy)
	}
}

func TestApproveDraftHandlerWrapsPayloadActorWhenContextMissing(t *testing.T) {
	payloadActor := spine.ActorRef{
		Kind:        "user",
		ID:          "payload_approver",
		DisplayName: "Payload Approver",
	}
	draftID := spine.ContractDraftID("draft-1")
	called := false
	service := &recordingApprovedContractService{
		approveDraft: func(ctx context.Context, gotDraftID spine.ContractDraftID, input spine.ApproveContractDraftRequest) (spine.ApprovedContract, error) {
			called = true
			if gotDraftID != draftID {
				t.Fatalf("draftID = %q, want %q", gotDraftID, draftID)
			}
			if input.ApprovedBy != payloadActor {
				t.Fatalf("input approved_by = %#v, want payload actor %#v", input.ApprovedBy, payloadActor)
			}
			actorContext, ok := actor.FromContext(ctx)
			if !ok {
				t.Fatal("ActorContext missing")
			}
			if actorContext.Actor != payloadActor {
				t.Fatalf("ActorContext actor = %#v, want payload actor %#v", actorContext.Actor, payloadActor)
			}
			if actorContext.Source != actor.SourcePayloadCompat {
				t.Fatalf("ActorContext source = %q, want %q", actorContext.Source, actor.SourcePayloadCompat)
			}
			return spine.ApprovedContract{
				ID:              "approved-contract-1",
				ContractDraftID: gotDraftID,
				ApprovedBy:      actorContext.Actor,
				State:           spine.ApprovedContractStateApproved,
			}, nil
		},
	}
	handler := httpserver.NewApprovedContractHandler(service)

	response := doApproveDraftHandlerJSON(t, handler, context.Background(), draftID, approveContractJSONWithActor(t, payloadActor))
	if !called {
		t.Fatal("ApproveDraft was not called")
	}
	if response.code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
	}
	var approved spine.ApprovedContract
	decodeJSON(t, response.body, &approved)
	if approved.ApprovedBy != payloadActor {
		t.Fatalf("response approved_by = %#v, want payload actor %#v", approved.ApprovedBy, payloadActor)
	}
}

func TestPostContractDraftApproveAppendsContractApprovedEvent(t *testing.T) {
	server := testServer(t)
	draft := createReadyContractDraft(t, server)

	response := doJSON(t, server.router, http.MethodPost, "/v1/contract-drafts/"+string(draft.ID)+"/approvals", approveContractJSON())
	if response.code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
	}

	if got := countEventType(server.events.Events(), approvedcontract.EventTypeContractApproved); got != 1 {
		t.Fatalf("contract.approved events = %d, want 1", got)
	}
	storedDraft, ok, err := server.contractDrafts.Get(context.Background(), draft.ID)
	if err != nil {
		t.Fatalf("contractDrafts.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("contractDrafts.Get() ok = false, want true")
	}
	event := server.events.Events()[len(server.events.Events())-1]
	if event.OrganizationID != storedDraft.OrganizationID || event.ProjectID != storedDraft.ProjectID || event.RepoBindingID != storedDraft.RepoBindingID {
		t.Fatalf("event context = %q/%q/%q, want stored draft context %q/%q/%q", event.OrganizationID, event.ProjectID, event.RepoBindingID, storedDraft.OrganizationID, storedDraft.ProjectID, storedDraft.RepoBindingID)
	}
	var payload struct {
		ApprovedContractID spine.ApprovedContractID `json:"approved_contract_id"`
		ContractDraftID    spine.ContractDraftID    `json:"contract_draft_id"`
		ApprovedBy         spine.ActorRef           `json:"approved_by"`
		ApprovedAt         string                   `json:"approved_at"`
	}
	decodeJSON(t, string(event.Payload), &payload)
	if payload.ApprovedContractID == "" {
		t.Fatal("approved_contract_id is empty")
	}
	if payload.ContractDraftID != draft.ID {
		t.Fatalf("contract_draft_id = %q, want %q", payload.ContractDraftID, draft.ID)
	}
	if payload.ApprovedBy.Kind != "user" || payload.ApprovedBy.ID != "dev_approver" {
		t.Fatalf("approved_by = %#v, want approver", payload.ApprovedBy)
	}
	if payload.ApprovedAt == "" {
		t.Fatal("approved_at is empty")
	}
	assertNoForbiddenApprovalSideEffects(t, server.events.Events())
}

func TestPostContractDraftApprovePreservesExistingActorContext(t *testing.T) {
	server := testServer(t)
	draft := createReadyContractDraft(t, server)
	contextActor := spine.ActorRef{
		Kind:        "user",
		ID:          "context_approver",
		DisplayName: "Context Approver",
	}
	payloadActor := spine.ActorRef{
		Kind:        "user",
		ID:          "payload_approver",
		DisplayName: "Payload Approver",
	}
	ctx := actor.WithActor(context.Background(), actor.ActorContext{
		Actor:  contextActor,
		Source: actor.SourceService,
	})

	response := doJSONWithContext(
		t,
		server.router,
		http.MethodPost,
		"/v1/contract-drafts/"+string(draft.ID)+"/approvals",
		approveContractJSONWithActor(t, payloadActor),
		ctx,
	)
	if response.code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
	}

	var approved spine.ApprovedContract
	decodeJSON(t, response.body, &approved)
	if approved.ApprovedBy != contextActor {
		t.Fatalf("response approved_by = %#v, want context actor %#v", approved.ApprovedBy, contextActor)
	}
	if approved.ApprovedBy == payloadActor {
		t.Fatalf("response approved_by = payload actor %#v, want context actor", payloadActor)
	}

	storedApproved, ok, err := server.approvedContracts.GetByContractDraftID(context.Background(), draft.ID)
	if err != nil {
		t.Fatalf("approvedContracts.GetByContractDraftID() error = %v", err)
	}
	if !ok {
		t.Fatal("approved contract missing after approval")
	}
	if storedApproved.ApprovedBy != contextActor {
		t.Fatalf("stored approved_by = %#v, want context actor %#v", storedApproved.ApprovedBy, contextActor)
	}

	event := server.events.Events()[len(server.events.Events())-1]
	var payload struct {
		ApprovedBy spine.ActorRef `json:"approved_by"`
	}
	decodeJSON(t, string(event.Payload), &payload)
	if payload.ApprovedBy != contextActor {
		t.Fatalf("event approved_by = %#v, want context actor %#v", payload.ApprovedBy, contextActor)
	}
}

func TestPostContractDraftApproveRejectsDuplicateApproval(t *testing.T) {
	server := testServer(t)
	draft := createReadyContractDraft(t, server)
	first := doJSON(t, server.router, http.MethodPost, "/v1/contract-drafts/"+string(draft.ID)+"/approvals", approveContractJSON())
	if first.code != http.StatusCreated {
		t.Fatalf("first status = %d, want %d: %s", first.code, http.StatusCreated, first.body)
	}

	second := doJSON(t, server.router, http.MethodPost, "/v1/contract-drafts/"+string(draft.ID)+"/approvals", approveContractJSON())
	if second.code != http.StatusConflict {
		t.Fatalf("second status = %d, want %d: %s", second.code, http.StatusConflict, second.body)
	}

	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeJSON(t, second.body, &body)
	if body.Error.Code != "already_approved" {
		t.Fatalf("error code = %q, want already_approved", body.Error.Code)
	}
}

func TestPostContractDraftApproveRejectsDraftState(t *testing.T) {
	server := testServer(t)
	draft := createContractDraft(t, server)

	response := doJSON(t, server.router, http.MethodPost, "/v1/contract-drafts/"+string(draft.ID)+"/approvals", approveContractJSON())
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

func TestPostContractDraftApproveValidatesApprovedBy(t *testing.T) {
	tests := []struct {
		name                string
		body                string
		wantMessageContains string
	}{
		{name: "missing_approved_by", body: `{}`, wantMessageContains: "approved_by.kind"},
		{name: "missing_kind", body: `{"approved_by":{"id":"dev_approver"}}`, wantMessageContains: "approved_by.kind"},
		{name: "missing_id", body: `{"approved_by":{"kind":"user"}}`, wantMessageContains: "approved_by.id"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := testServer(t)
			draft := createReadyContractDraft(t, server)

			response := doJSON(t, server.router, http.MethodPost, "/v1/contract-drafts/"+string(draft.ID)+"/approvals", tt.body)
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
			if !strings.Contains(body.Error.Message, tt.wantMessageContains) {
				t.Fatalf("error message = %q, want %q", body.Error.Message, tt.wantMessageContains)
			}
		})
	}
}

func TestApproveDraftHandlerPassesInvalidPayloadActorToService(t *testing.T) {
	draftID := spine.ContractDraftID("draft-1")
	called := false
	service := &recordingApprovedContractService{
		approveDraft: func(ctx context.Context, gotDraftID spine.ContractDraftID, input spine.ApproveContractDraftRequest) (spine.ApprovedContract, error) {
			called = true
			if gotDraftID != draftID {
				t.Fatalf("draftID = %q, want %q", gotDraftID, draftID)
			}
			if input.ApprovedBy.Kind != "" || input.ApprovedBy.ID != "payload_approver" {
				t.Fatalf("input approved_by = %#v, want invalid payload actor", input.ApprovedBy)
			}
			actorContext, ok := actor.FromContext(ctx)
			if !ok {
				t.Fatal("ActorContext missing")
			}
			if actorContext.Actor != input.ApprovedBy {
				t.Fatalf("ActorContext actor = %#v, want input approved_by %#v", actorContext.Actor, input.ApprovedBy)
			}
			return spine.ApprovedContract{}, &approvedcontract.ValidationError{Field: "approved_by.kind", Message: "is required"}
		},
	}
	handler := httpserver.NewApprovedContractHandler(service)

	response := doApproveDraftHandlerJSON(t, handler, context.Background(), draftID, `{"approved_by":{"id":"payload_approver"}}`)
	if !called {
		t.Fatal("ApproveDraft was not called")
	}
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
	if !strings.Contains(body.Error.Message, "approved_by.kind") {
		t.Fatalf("error message = %q, want approved_by.kind", body.Error.Message)
	}
}

func TestPostContractDraftApproveRejectsIncompleteDraft(t *testing.T) {
	server := testServer(t)
	draft := validHTTPContractDraft()
	draft.ID = "incomplete-ready-draft"
	draft.State = spine.ContractDraftStateReadyForApproval
	draft.ProposedAcceptanceCriteria = []string{}
	if err := server.contractDrafts.Create(context.Background(), draft); err != nil {
		t.Fatalf("contractDrafts.Create() error = %v", err)
	}

	response := doJSON(t, server.router, http.MethodPost, "/v1/contract-drafts/"+string(draft.ID)+"/approvals", approveContractJSON())
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
	if !strings.Contains(body.Error.Message, approvedcontract.ReasonMissingProposedAcceptanceCriteria) {
		t.Fatalf("error message = %q, want missing acceptance reason", body.Error.Message)
	}
}

func createReadyContractDraft(t *testing.T, server testServerDeps) spine.ContractDraft {
	t.Helper()

	draft := createContractDraft(t, server)
	response := doJSON(t, server.router, http.MethodPost, "/v1/contract-drafts/"+string(draft.ID)+"/approval-submissions", readyForApprovalJSON())
	if response.code != http.StatusOK {
		t.Fatalf("ready status = %d, want %d: %s", response.code, http.StatusOK, response.body)
	}
	var ready spine.ContractDraft
	decodeJSON(t, response.body, &ready)
	return ready
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

func approveContractJSONWithActor(t *testing.T, approvedBy spine.ActorRef) string {
	t.Helper()

	encoded, err := json.Marshal(spine.ApproveContractDraftRequest{ApprovedBy: approvedBy})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	return string(encoded)
}

func doJSONWithContext(t *testing.T, handler http.Handler, method string, path string, body string, ctx context.Context) routeResponse {
	t.Helper()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if ctx != nil {
		request = request.WithContext(ctx)
	}
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}

	handler.ServeHTTP(recorder, request)

	contentType := recorder.Header().Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		t.Fatalf("Content-Type = %q, want application/json", contentType)
	}

	return routeResponse{
		code:        recorder.Code,
		contentType: contentType,
		body:        recorder.Body.String(),
	}
}

func doApproveDraftHandlerJSON(t *testing.T, handler *httpserver.ApprovedContractHandler, ctx context.Context, draftID spine.ContractDraftID, body string) routeResponse {
	t.Helper()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/contract-drafts/"+string(draftID)+"/approvals", strings.NewReader(body))
	if ctx != nil {
		request = request.WithContext(ctx)
	}
	request.SetPathValue("id", string(draftID))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}

	handler.ApproveDraft(recorder, request)

	contentType := recorder.Header().Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		t.Fatalf("Content-Type = %q, want application/json", contentType)
	}

	return routeResponse{
		code:        recorder.Code,
		contentType: contentType,
		body:        recorder.Body.String(),
	}
}

type recordingApprovedContractService struct {
	approveDraft func(context.Context, spine.ContractDraftID, spine.ApproveContractDraftRequest) (spine.ApprovedContract, error)
}

func (s *recordingApprovedContractService) ApproveDraft(ctx context.Context, draftID spine.ContractDraftID, input spine.ApproveContractDraftRequest) (spine.ApprovedContract, error) {
	return s.approveDraft(ctx, draftID, input)
}

func assertNoForbiddenApprovalSideEffects(t *testing.T, events []spine.Event) {
	t.Helper()

	forbidden := map[string]bool{
		"work_item.created":     true,
		"run.started":           true,
		"gate.decision_written": true,
		"proof.created":         true,
	}
	for _, event := range events {
		if forbidden[event.Type] {
			t.Fatalf("forbidden event type appended: %s", event.Type)
		}
	}
}

func TestApprovedContractResponseJSONDoesNotExposeContext(t *testing.T) {
	approved := spine.ApprovedContract{
		ID:             "approved-contract-1",
		OrganizationID: "organization-1",
		ProjectID:      "project-1",
	}
	encoded, err := json.Marshal(approved)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if strings.Contains(string(encoded), "organization_id") || strings.Contains(string(encoded), "project_id") {
		t.Fatalf("encoded approved contract exposes internal context: %s", encoded)
	}
}
