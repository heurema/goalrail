package httpserver_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/heurema/goalrail/apps/server/internal/health"
	"github.com/heurema/goalrail/apps/server/internal/httpserver"
	"github.com/heurema/goalrail/apps/server/internal/version"
)

func TestLivezReturnsOK(t *testing.T) {
	response := getJSON(t, "/livez")

	if response.code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.code, http.StatusOK)
	}

	var body struct {
		Status string `json:"status"`
	}
	decodeJSON(t, response.body, &body)
	if body.Status != "ok" {
		t.Fatalf("status body = %q, want %q", body.Status, "ok")
	}
}

func TestReadyzReturnsOK(t *testing.T) {
	response := getJSON(t, "/readyz")

	if response.code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.code, http.StatusOK)
	}

	var body struct {
		Status string `json:"status"`
	}
	decodeJSON(t, response.body, &body)
	if body.Status != "ok" {
		t.Fatalf("status body = %q, want %q", body.Status, "ok")
	}
}

func TestVersionReturnsService(t *testing.T) {
	response := getJSON(t, "/version")

	if response.code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.code, http.StatusOK)
	}

	var body struct {
		Service string `json:"service"`
		Version string `json:"version"`
	}
	decodeJSON(t, response.body, &body)
	if body.Service != "goalrail-server" {
		t.Fatalf("service = %q, want %q", body.Service, "goalrail-server")
	}
	if body.Version == "" {
		t.Fatal("version is empty")
	}
}

func TestUnknownRouteReturnsJSONNotFound(t *testing.T) {
	response := getJSON(t, "/missing")

	if response.code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.code, http.StatusNotFound)
	}

	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	decodeJSON(t, response.body, &body)
	if body.Error.Code != "not_found" {
		t.Fatalf("error code = %q, want %q", body.Error.Code, "not_found")
	}
	if body.Error.Message != "not found" {
		t.Fatalf("error message = %q, want %q", body.Error.Message, "not found")
	}
}

type routeResponse struct {
	code        int
	contentType string
	body        string
}

func getJSON(t *testing.T, path string) routeResponse {
	t.Helper()

	return doJSON(t, testServer(t).router, http.MethodGet, path, "")
}

func doJSON(t *testing.T, handler http.Handler, method string, path string, body string) routeResponse {
	t.Helper()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
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

func newRouter(
	livez http.Handler,
	readyz http.Handler,
	versionHandler http.Handler,
	intakeHandler *httpserver.IntakeHandler,
	goalHandler *httpserver.GoalHandler,
	clarificationHandler *httpserver.ClarificationHandler,
	contractSeedHandler *httpserver.ContractSeedHandler,
	contractDraftHandler *httpserver.ContractDraftHandler,
	approvedContractHandler *httpserver.ApprovedContractHandler,
) http.Handler {
	return httpserver.NewRouter(httpserver.RouteHandlers{
		Livez:                     livez,
		Readyz:                    readyz,
		Version:                   versionHandler,
		IntakeSubmit:              http.HandlerFunc(intakeHandler.Submit),
		IntakeGet:                 http.HandlerFunc(intakeHandler.Get),
		IntakePromote:             http.HandlerFunc(goalHandler.PromoteFromIntake),
		GoalReadiness:             http.HandlerFunc(goalHandler.CheckReadiness),
		GoalClarificationRequests: http.HandlerFunc(clarificationHandler.CreateRequest),
		GoalContractSeed:          http.HandlerFunc(contractSeedHandler.Create),
		ContractSeedDraft:         http.HandlerFunc(contractDraftHandler.Create),
		ContractDraftUpdates:      http.HandlerFunc(contractDraftHandler.Update),
		ContractDraftReady:        http.HandlerFunc(contractDraftHandler.MarkReadyForApproval),
		ContractDraftApprove:      http.HandlerFunc(approvedContractHandler.ApproveDraft),
		ClarificationAnswers:      http.HandlerFunc(clarificationHandler.RecordAnswer),
		ClarificationAnswerApply:  http.HandlerFunc(clarificationHandler.ApplyAnswer),
	})
}

func baseHandlers(
	intakeHandler *httpserver.IntakeHandler,
	goalHandler *httpserver.GoalHandler,
	clarificationHandler *httpserver.ClarificationHandler,
	contractSeedHandler *httpserver.ContractSeedHandler,
	contractDraftHandler *httpserver.ContractDraftHandler,
	approvedContractHandler *httpserver.ApprovedContractHandler,
) http.Handler {
	healthHandler := health.NewHandler()
	return newRouter(
		http.HandlerFunc(healthHandler.Livez),
		http.HandlerFunc(healthHandler.Readyz),
		version.NewHandler(),
		intakeHandler,
		goalHandler,
		clarificationHandler,
		contractSeedHandler,
		contractDraftHandler,
		approvedContractHandler,
	)
}

func decodeJSON(t *testing.T, input string, target any) {
	t.Helper()

	if err := json.Unmarshal([]byte(input), target); err != nil {
		t.Fatalf("decode JSON %q: %v", input, err)
	}
}
