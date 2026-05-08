package httpserver_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/heurema/goalrail/apps/server/internal/actor"
	"github.com/heurema/goalrail/apps/server/internal/auth"
	"github.com/heurema/goalrail/apps/server/internal/contract"
	"github.com/heurema/goalrail/apps/server/internal/httpserver"
	"github.com/heurema/goalrail/apps/server/internal/spine"
)

func TestInvalidJSONResponseShape(t *testing.T) {
	handler := httpserver.NewAuthHandler(fakeHTTPAuthService{})

	response := doJSON(t, http.HandlerFunc(handler.Login), http.MethodPost, "/v1/auth/login", "{")

	assertJSONErrorResponse(t, response, http.StatusBadRequest, "invalid_json", "invalid JSON request body")
}

func TestInternalFallbackResponseShape(t *testing.T) {
	handler := httpserver.NewAuthHandler(fakeHTTPAuthService{loginErr: errors.New("boom")})

	response := doJSON(t, http.HandlerFunc(handler.Login), http.MethodPost, "/v1/auth/login", `{"email":"owner@example.com","password":"temporary-password"}`)

	assertJSONErrorResponse(t, response, http.StatusInternalServerError, "internal_error", "internal server error")
}

func TestContractApproveDerivesActorFromAuthenticatedUser(t *testing.T) {
	service := &capturingContractService{}
	handler := httpserver.NewContractHandler(fakeHTTPAuthService{
		profile: continuationAuthProfile("018f0000-0000-7000-8000-000000000002"),
	}, service)
	request := newContractApproveRequest(t, context.Background(), approveContractJSON())
	recorder := httptest.NewRecorder()

	handler.Approve(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusCreated, recorder.Body.String())
	}
	if !service.approveActorOK {
		t.Fatal("approval actor context was not set")
	}
	want := actor.ActorContext{
		Actor: spine.ActorRef{
			Kind:        "user",
			ID:          "018f0000-0000-7000-8000-000000000001",
			DisplayName: "Developer",
		},
		Source: actor.SourceService,
	}
	if !reflect.DeepEqual(service.approveActor, want) {
		t.Fatalf("approval actor = %#v, want %#v", service.approveActor, want)
	}
	wantInput := spine.ActorRef{
		Kind:        "user",
		ID:          "018f0000-0000-7000-8000-000000000001",
		DisplayName: "Developer",
	}
	if !reflect.DeepEqual(service.approveInput.ApprovedBy, wantInput) {
		t.Fatalf("approved_by input = %#v, want authenticated user", service.approveInput.ApprovedBy)
	}
}

func TestContractApproveRequiresAuthBeforeService(t *testing.T) {
	service := &capturingContractService{}
	handler := httpserver.NewContractHandler(fakeHTTPAuthService{meErr: auth.ErrInvalidToken}, service)
	request := newContractApproveRequest(t, context.Background(), approveContractJSON())
	recorder := httptest.NewRecorder()

	handler.Approve(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusUnauthorized, recorder.Body.String())
	}
	if service.approveCalled {
		t.Fatal("contract approve service was called despite auth failure")
	}
}

func TestContractUpdateRequiresAuthBeforeService(t *testing.T) {
	service := &capturingContractService{}
	handler := httpserver.NewContractHandler(fakeHTTPAuthService{meErr: auth.ErrInvalidToken}, service)
	request := httptest.NewRequest(http.MethodPatch, "/v1/contracts/contract-1", strings.NewReader(updateDraftJSON(`{"title":"Reviewed"}`)))
	request.Header.Set("Content-Type", "application/json")
	request.SetPathValue("id", "contract-1")
	recorder := httptest.NewRecorder()

	handler.UpdateDraft(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusUnauthorized, recorder.Body.String())
	}
	if service.updateCalled {
		t.Fatal("contract update service was called despite auth failure")
	}
}

func newContractApproveRequest(t *testing.T, ctx context.Context, body string) *http.Request {
	t.Helper()

	request := httptest.NewRequest(http.MethodPost, "/v1/contracts/018f0000-0000-7000-8000-000000000009/approvals", strings.NewReader(body)).WithContext(ctx)
	request.Header.Set("Content-Type", "application/json")
	request.SetPathValue("id", "018f0000-0000-7000-8000-000000000009")
	return request
}

func assertJSONErrorResponse(t *testing.T, response routeResponse, status int, code string, message string) {
	t.Helper()

	if response.code != status {
		t.Fatalf("status = %d, want %d: %s", response.code, status, response.body)
	}
	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	decodeJSON(t, response.body, &body)
	if body.Error.Code != code {
		t.Fatalf("error code = %q, want %q", body.Error.Code, code)
	}
	if body.Error.Message != message {
		t.Fatalf("error message = %q, want %q", body.Error.Message, message)
	}
}

type capturingContractService struct {
	approveActor   actor.ActorContext
	approveActorOK bool
	approveCalled  bool
	approveInput   spine.ApproveContractDraftRequest
	updateCalled   bool
}

func (s *capturingContractService) Create(context.Context, spine.ContractCreateRequest, spine.OrganizationMembership) (spine.Contract, bool, error) {
	return spine.Contract{}, false, nil
}

func (s *capturingContractService) Get(context.Context, spine.ContractID) (spine.Contract, error) {
	return spine.Contract{}, nil
}

func (s *capturingContractService) List(context.Context, contract.ListInput) (spine.ContractList, error) {
	return spine.ContractList{}, nil
}

func (s *capturingContractService) CurrentDraft(context.Context, spine.ContractID, spine.OrganizationMembership) (spine.ContractDraft, error) {
	return spine.ContractDraft{}, nil
}

func (s *capturingContractService) UpdateDraft(context.Context, spine.ContractID, spine.ContractDraftUpdateRequest, spine.OrganizationMembership) (spine.Contract, error) {
	s.updateCalled = true
	return spine.Contract{}, nil
}

func (s *capturingContractService) SubmitForApproval(context.Context, spine.ContractID, spine.ContractDraftReadyForApprovalRequest, spine.OrganizationMembership) (spine.Contract, error) {
	return spine.Contract{}, nil
}

func (s *capturingContractService) Approve(ctx context.Context, id spine.ContractID, input spine.ApproveContractDraftRequest, _ spine.OrganizationMembership) (spine.Contract, error) {
	s.approveCalled = true
	s.approveInput = input
	s.approveActor, s.approveActorOK = actor.FromContext(ctx)
	return spine.Contract{
		ID:    id,
		State: spine.ContractStateApproved,
	}, nil
}

var _ httpserver.ContractService = (*capturingContractService)(nil)
