package httpserver_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/heurema/goalrail/apps/server/internal/health"
	"github.com/heurema/goalrail/apps/server/internal/httpserver"
)

func TestCORSDisabledKeepsOrdinaryResponseWithoutCORSHeaders(t *testing.T) {
	healthHandler := health.NewHandler()
	handler := httpserver.WithCORS(http.HandlerFunc(healthHandler.Livez), nil)

	response := doRaw(t, handler, http.MethodGet, "/livez", "https://goalrail.dev")

	if response.code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusOK, response.body)
	}
	if !strings.Contains(response.body, `"status":"ok"`) {
		t.Fatalf("body = %q, want livez health response", response.body)
	}
	if got := response.header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want empty", got)
	}
	if got := response.header.Get("Access-Control-Allow-Methods"); got != "" {
		t.Fatalf("Access-Control-Allow-Methods = %q, want empty", got)
	}
	if got := response.header.Get("Access-Control-Allow-Headers"); got != "" {
		t.Fatalf("Access-Control-Allow-Headers = %q, want empty", got)
	}
}

func TestCORSAllowedOriginEchoesOriginAndVary(t *testing.T) {
	handler := httpserver.WithCORS(probeRoute("me"), []string{"https://goalrail.dev"})

	response := doRaw(t, handler, http.MethodGet, "/v1/me", "https://goalrail.dev")

	if response.code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusOK, response.body)
	}
	if got := response.header.Get("Access-Control-Allow-Origin"); got != "https://goalrail.dev" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want exact origin", got)
	}
	if got := response.header.Get("Vary"); got != "Origin" {
		t.Fatalf("Vary = %q, want Origin", got)
	}
	if got := response.header.Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("Access-Control-Allow-Credentials = %q, want empty", got)
	}
}

func TestCORSDisallowedOriginDoesNotEmitAllowOrigin(t *testing.T) {
	handler := httpserver.WithCORS(probeRoute("me"), []string{"https://goalrail.dev"})

	response := doRaw(t, handler, http.MethodGet, "/v1/me", "https://evil.example")

	if response.code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusOK, response.body)
	}
	if got := response.header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want empty", got)
	}
}

func TestCORSAllowedPreflightReturnsNoContentWithMethodsAndHeaders(t *testing.T) {
	handler := httpserver.WithCORS(probeRoute("auth_login"), []string{"https://goalrail.dev"})

	response := doRaw(t, handler, http.MethodOptions, "/v1/auth/login", "https://goalrail.dev")

	if response.code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusNoContent, response.body)
	}
	if response.body != "" {
		t.Fatalf("body = %q, want empty", response.body)
	}
	if got := response.header.Get("Access-Control-Allow-Origin"); got != "https://goalrail.dev" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want exact origin", got)
	}
	assertHeaderContains(t, response.header.Get("Access-Control-Allow-Methods"), http.MethodGet)
	assertHeaderContains(t, response.header.Get("Access-Control-Allow-Methods"), http.MethodPost)
	assertHeaderContains(t, response.header.Get("Access-Control-Allow-Methods"), http.MethodPatch)
	assertHeaderContains(t, response.header.Get("Access-Control-Allow-Methods"), http.MethodOptions)
	assertHeaderContains(t, response.header.Get("Access-Control-Allow-Headers"), "Authorization")
	assertHeaderContains(t, response.header.Get("Access-Control-Allow-Headers"), "Content-Type")
	if got := response.header.Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("Access-Control-Allow-Credentials = %q, want empty", got)
	}
}

func assertHeaderContains(t *testing.T, header string, want string) {
	t.Helper()

	for _, part := range strings.Split(header, ",") {
		if strings.EqualFold(strings.TrimSpace(part), want) {
			return
		}
	}
	t.Fatalf("header %q does not contain %q", header, want)
}

func doRaw(t *testing.T, handler http.Handler, method string, path string, origin string) routeResponse {
	t.Helper()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, nil)
	if origin != "" {
		request.Header.Set("Origin", origin)
	}

	handler.ServeHTTP(recorder, request)

	return routeResponse{
		code:        recorder.Code,
		contentType: recorder.Header().Get("Content-Type"),
		header:      recorder.Header(),
		body:        recorder.Body.String(),
	}
}
