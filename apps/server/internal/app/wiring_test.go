package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/heurema/goalrail/apps/server/internal/config"
)

func TestHTTPServerWithoutDatabaseKeepsHealthAndVersionAvailable(t *testing.T) {
	server, cleanup, err := newHTTPServer(context.Background(), config.Config{Addr: "127.0.0.1:0"})
	if err != nil {
		t.Fatalf("newHTTPServer() error = %v", err)
	}
	defer cleanup()

	for _, tt := range []struct {
		method string
		path   string
	}{
		{method: http.MethodGet, path: "/livez"},
		{method: http.MethodGet, path: "/readyz"},
		{method: http.MethodGet, path: "/version"},
	} {
		t.Run(tt.path, func(t *testing.T) {
			response := doServerRequest(t, server.Handler, tt.method, tt.path, "")
			if response.code != http.StatusOK {
				t.Fatalf("%s %s status = %d, want %d: %s", tt.method, tt.path, response.code, http.StatusOK, response.body)
			}
		})
	}
}

func TestHTTPServerWithoutDatabaseReturnsUnavailableForProductRoutes(t *testing.T) {
	server, cleanup, err := newHTTPServer(context.Background(), config.Config{Addr: "127.0.0.1:0"})
	if err != nil {
		t.Fatalf("newHTTPServer() error = %v", err)
	}
	defer cleanup()

	for _, tt := range []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "intake", method: http.MethodPost, path: "/v1/intakes", body: `{}`},
		{name: "contract", method: http.MethodPost, path: "/v1/contracts", body: `{}`},
		{name: "auth login", method: http.MethodPost, path: "/v1/auth/login", body: `{}`},
		{name: "auth refresh", method: http.MethodPost, path: "/v1/auth/refresh", body: `{}`},
		{name: "auth logout", method: http.MethodPost, path: "/v1/auth/logout"},
		{name: "me", method: http.MethodGet, path: "/v1/me"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			response := doServerRequest(t, server.Handler, tt.method, tt.path, tt.body)
			if response.code != http.StatusServiceUnavailable {
				t.Fatalf("%s %s status = %d, want %d: %s", tt.method, tt.path, response.code, http.StatusServiceUnavailable, response.body)
			}
			var body struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			if err := json.Unmarshal([]byte(response.body), &body); err != nil {
				t.Fatalf("decode JSON %q: %v", response.body, err)
			}
			if body.Error.Code != "database_not_configured" {
				t.Fatalf("error code = %q, want database_not_configured", body.Error.Code)
			}
		})
	}
}

func TestHTTPServerWithDatabaseConfigAttemptsPostgresPool(t *testing.T) {
	cfg := config.Config{
		Addr: "127.0.0.1:0",
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			Name:     "goalrail",
			User:     "goalrail",
			Password: "secret-password",
			SSLMode:  "invalid sslmode",
		},
	}

	_, _, err := newHTTPServer(context.Background(), cfg)
	if err == nil {
		t.Fatal("newHTTPServer() error = nil, want database config error")
	}
	if !strings.Contains(err.Error(), "parse database config") {
		t.Fatalf("newHTTPServer() error = %v, want database config parse error", err)
	}
	if strings.Contains(err.Error(), "secret-password") {
		t.Fatalf("newHTTPServer() error leaked database password: %v", err)
	}
}

type serverResponse struct {
	code int
	body string
}

func doServerRequest(t *testing.T, handler http.Handler, method string, path string, body string) serverResponse {
	t.Helper()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	handler.ServeHTTP(recorder, request)
	return serverResponse{code: recorder.Code, body: recorder.Body.String()}
}
