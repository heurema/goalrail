package authsession

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/cli/internal/authstore"
	"github.com/heurema/goalrail/apps/cli/internal/exitcode"
)

func TestLoadUsableRefreshesExpiredSessionAndSavesMetadata(t *testing.T) {
	t.Parallel()

	var refreshCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/auth/refresh" {
			t.Errorf("path = %s, want /v1/auth/refresh", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		refreshCount.Add(1)
		if r.Header.Get("Authorization") != "" {
			t.Errorf("Authorization = %q, want empty refresh request auth", r.Header.Get("Authorization"))
		}
		var payload map[string]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode refresh payload: %v", err)
		}
		if payload["refresh_token"] != "stored-refresh-token" {
			t.Errorf("refresh_token = %q, want stored refresh token", payload["refresh_token"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"new-access-token","access_token_expires_at":"2026-05-05T11:30:00Z","token_type":"Bearer"}`))
	}))
	defer server.Close()

	store := &memoryStore{session: authstore.Session{
		ServerURL:            server.URL,
		AccessToken:          "expired-access-token",
		RefreshToken:         "stored-refresh-token",
		AccessTokenExpiresAt: time.Date(2026, 5, 5, 9, 59, 0, 0, time.UTC),
		TokenType:            "Bearer",
	}}

	session, serverURL, _, err := LoadUsable(context.Background(), Options{
		Store: store,
		Now:   func() time.Time { return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("LoadUsable() error = %v", err)
	}
	if serverURL != server.URL {
		t.Fatalf("serverURL = %q, want %q", serverURL, server.URL)
	}
	if session.AccessToken != "new-access-token" || store.saved.AccessToken != "new-access-token" {
		t.Fatalf("access token session/store = %q/%q, want refreshed token", session.AccessToken, store.saved.AccessToken)
	}
	if session.RefreshToken != "stored-refresh-token" {
		t.Fatalf("refresh token = %q, want existing refresh token preserved", session.RefreshToken)
	}
	if refreshCount.Load() != 1 || store.saveCount.Load() != 1 {
		t.Fatalf("refresh/save count = %d/%d, want 1/1", refreshCount.Load(), store.saveCount.Load())
	}
}

func TestLoadUsableWithMetadataReportsValidStoredAccessToken(t *testing.T) {
	t.Parallel()

	store := &memoryStore{session: validSession("https://goalrail.example")}
	session, serverURL, _, reporter, err := LoadUsableWithMetadata(context.Background(), Options{
		Store: store,
		Now:   func() time.Time { return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("LoadUsableWithMetadata() error = %v", err)
	}
	if session.AccessToken != "old-access-token" || serverURL != "https://goalrail.example" {
		t.Fatalf("session/server = %q/%q, want stored session", session.AccessToken, serverURL)
	}
	metadata := reporter.AuthSessionMetadata()
	if metadata.ServerURL != "https://goalrail.example" {
		t.Fatalf("metadata server_url = %q, want session server", metadata.ServerURL)
	}
	if !metadata.UsedStoredAccessToken || metadata.RefreshAttempted || metadata.AccessTokenRefreshed {
		t.Fatalf("metadata = %#v, want stored access token with no refresh", metadata)
	}
	if metadata.Reason != "access_token_valid" {
		t.Fatalf("metadata reason = %q, want access_token_valid", metadata.Reason)
	}
}

func TestLoadUsableWithMetadataReportsInitialRefresh(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/auth/refresh" {
			t.Errorf("path = %s, want /v1/auth/refresh", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"new-access-token","access_token_expires_at":"2026-05-05T11:30:00Z","token_type":"Bearer"}`))
	}))
	defer server.Close()

	store := &memoryStore{session: authstore.Session{
		ServerURL:            server.URL,
		AccessToken:          "expired-access-token",
		RefreshToken:         "stored-refresh-token",
		AccessTokenExpiresAt: time.Date(2026, 5, 5, 9, 59, 0, 0, time.UTC),
		TokenType:            "Bearer",
	}}
	_, _, _, reporter, err := LoadUsableWithMetadata(context.Background(), Options{
		Store: store,
		Now:   func() time.Time { return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("LoadUsableWithMetadata() error = %v", err)
	}
	metadata := reporter.AuthSessionMetadata()
	if metadata.UsedStoredAccessToken || !metadata.RefreshAttempted || !metadata.AccessTokenRefreshed {
		t.Fatalf("metadata = %#v, want attempted successful refresh", metadata)
	}
	if metadata.Reason != "refresh_succeeded" {
		t.Fatalf("metadata reason = %q, want refresh_succeeded", metadata.Reason)
	}
}

func TestRetryingClientRefreshesAndRetriesUnauthorizedRequestOnce(t *testing.T) {
	t.Parallel()

	var refreshCount atomic.Int32
	var contractCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/auth/refresh":
			refreshCount.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"new-access-token","refresh_token":"rotated-refresh-token","access_token_expires_at":"2026-05-05T11:30:00Z","token_type":"Bearer"}`))
		case "/v1/contracts/018f0000-0000-7000-8000-000000000009":
			contractCount.Add(1)
			switch r.Header.Get("Authorization") {
			case "Bearer old-access-token":
				http.Error(w, `{"error":{"code":"unauthorized","message":"expired"}}`, http.StatusUnauthorized)
			case "Bearer new-access-token":
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"id":"018f0000-0000-7000-8000-000000000009","state":"approved"}`))
			default:
				t.Errorf("Authorization = %q, want old or new bearer", r.Header.Get("Authorization"))
				http.Error(w, "bad auth", http.StatusUnauthorized)
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	store := &memoryStore{session: authstore.Session{
		ServerURL:            server.URL,
		AccessToken:          "old-access-token",
		RefreshToken:         "stored-refresh-token",
		AccessTokenExpiresAt: time.Date(2026, 5, 5, 11, 0, 0, 0, time.UTC),
		TokenType:            "Bearer",
	}}
	session, _, client, reporter, err := LoadUsableWithMetadata(context.Background(), Options{
		Store: store,
		Now:   func() time.Time { return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("LoadUsable() error = %v", err)
	}

	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"/v1/contracts/018f0000-0000-7000-8000-000000000009", nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	request.Header.Set("Authorization", "Bearer "+session.AccessToken)
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}
	if refreshCount.Load() != 1 || contractCount.Load() != 2 {
		t.Fatalf("refresh/contract counts = %d/%d, want 1/2", refreshCount.Load(), contractCount.Load())
	}
	if store.saved.AccessToken != "new-access-token" || store.saved.RefreshToken != "rotated-refresh-token" {
		t.Fatalf("saved session = %#v, want refreshed access and rotated refresh token", store.saved)
	}
	metadata := reporter.AuthSessionMetadata()
	if metadata.UsedStoredAccessToken || !metadata.RefreshAttempted || !metadata.AccessTokenRefreshed {
		t.Fatalf("metadata = %#v, want retry refresh metadata", metadata)
	}
}

func TestRetryingClientDoesNotLoopRefreshAttempts(t *testing.T) {
	t.Parallel()

	var refreshCount atomic.Int32
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/auth/refresh":
			refreshCount.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"new-access-token","access_token_expires_at":"2026-05-05T11:30:00Z","token_type":"Bearer"}`))
		case "/v1/contracts/018f0000-0000-7000-8000-000000000009":
			requestCount.Add(1)
			http.Error(w, `{"error":{"code":"unauthorized","message":"still expired"}}`, http.StatusUnauthorized)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	store := &memoryStore{session: validSession(server.URL)}
	_, _, client, err := LoadUsable(context.Background(), Options{
		Store: store,
		Now:   func() time.Time { return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("LoadUsable() error = %v", err)
	}
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"/v1/contracts/018f0000-0000-7000-8000-000000000009", nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want final 401 after one retry", response.StatusCode)
	}
	if refreshCount.Load() != 1 || requestCount.Load() != 2 {
		t.Fatalf("refresh/request counts = %d/%d, want 1/2", refreshCount.Load(), requestCount.Load())
	}
}

func TestRetryingClientDoesNotRetryUnauthorizedPOST(t *testing.T) {
	t.Parallel()

	var refreshCount atomic.Int32
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/auth/refresh":
			refreshCount.Add(1)
			http.Error(w, "unexpected refresh", http.StatusInternalServerError)
		case "/v1/contracts/018f0000-0000-7000-8000-000000000009/submit":
			requestCount.Add(1)
			if r.Method != http.MethodPost {
				t.Errorf("method = %s, want POST", r.Method)
			}
			http.Error(w, `{"error":{"code":"unauthorized","message":"expired"}}`, http.StatusUnauthorized)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	store := &memoryStore{session: validSession(server.URL)}
	_, _, client, err := LoadUsable(context.Background(), Options{
		Store: store,
		Now:   func() time.Time { return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("LoadUsable() error = %v", err)
	}
	request, err := http.NewRequestWithContext(context.Background(), http.MethodPost, server.URL+"/v1/contracts/018f0000-0000-7000-8000-000000000009/submit", strings.NewReader(`{"confirm":true}`))
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want final 401 without retry", response.StatusCode)
	}
	if refreshCount.Load() != 0 || requestCount.Load() != 1 {
		t.Fatalf("refresh/request counts = %d/%d, want 0/1", refreshCount.Load(), requestCount.Load())
	}
}

func TestRefreshFailureDoesNotLeakTokenValues(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"code":"refresh_denied","message":"access=secret-access refresh=secret-refresh"}}`, http.StatusUnauthorized)
	}))
	defer server.Close()

	store := &memoryStore{session: authstore.Session{
		ServerURL:            server.URL,
		AccessToken:          "secret-access",
		RefreshToken:         "secret-refresh",
		AccessTokenExpiresAt: time.Date(2026, 5, 5, 9, 59, 0, 0, time.UTC),
		TokenType:            "Bearer",
	}}
	_, _, _, err := LoadUsable(context.Background(), Options{
		Store: store,
		Now:   func() time.Time { return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC) },
	})
	if err == nil {
		t.Fatal("LoadUsable() error = nil, want refresh failure")
	}
	if got := exitcode.ForError(err); got != exitcode.Usage {
		t.Fatalf("exit code = %d, want usage", got)
	}
	for _, secret := range []string{"secret-access", "secret-refresh"} {
		if strings.Contains(err.Error(), secret) {
			t.Fatalf("error = %q, leaked %q", err.Error(), secret)
		}
	}
	if !strings.Contains(err.Error(), "refresh access token failed") || !strings.Contains(err.Error(), "run goalrail login") {
		t.Fatalf("error = %q, want clear refresh/login guidance", err.Error())
	}
}

func TestMissingSessionKeepsLoginGuidance(t *testing.T) {
	t.Parallel()

	_, _, _, err := LoadUsable(context.Background(), Options{Store: &memoryStore{err: authstore.ErrSessionNotFound}})
	if err == nil {
		t.Fatal("LoadUsable() error = nil, want missing login")
	}
	if !strings.Contains(err.Error(), "not logged in; run goalrail login <server_url>") {
		t.Fatalf("error = %q, want login guidance", err.Error())
	}
}

func validSession(serverURL string) authstore.Session {
	return authstore.Session{
		ServerURL:            serverURL,
		AccessToken:          "old-access-token",
		RefreshToken:         "stored-refresh-token",
		AccessTokenExpiresAt: time.Date(2026, 5, 5, 11, 0, 0, 0, time.UTC),
		TokenType:            "Bearer",
	}
}

type memoryStore struct {
	session   authstore.Session
	saved     authstore.Session
	err       error
	saveCount atomic.Int32
}

func (s *memoryStore) Load() (authstore.Session, error) {
	if s.err != nil {
		return authstore.Session{}, s.err
	}
	return s.session, nil
}

func (s *memoryStore) Save(session authstore.Session) error {
	if session.AccessToken == "" {
		return errors.New("missing access token")
	}
	s.saved = session
	s.session = session
	s.saveCount.Add(1)
	return nil
}
