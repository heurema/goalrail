package authsession

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/heurema/goalrail/apps/cli/internal/authstore"
	"github.com/heurema/goalrail/apps/cli/internal/exitcode"
)

type SessionStore interface {
	Load() (authstore.Session, error)
}

type SavingSessionStore interface {
	SessionStore
	Save(authstore.Session) error
}

type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

type Options struct {
	Store  SessionStore
	Client HTTPClient
	Now    func() time.Time
}

type refreshResponse struct {
	AccessToken          string `json:"access_token"`
	RefreshToken         string `json:"refresh_token"`
	AccessTokenExpiresAt string `json:"access_token_expires_at"`
	TokenType            string `json:"token_type"`
}

type serverErrorResponse struct {
	Error struct {
		Code string `json:"code"`
	} `json:"error"`
}

func LoadUsable(ctx context.Context, options Options) (authstore.Session, string, HTTPClient, error) {
	store := options.Store
	if store == nil {
		path, err := authstore.DefaultPath()
		if err != nil {
			return authstore.Session{}, "", nil, exitcode.RuntimeError(err)
		}
		store = authstore.NewFileStore(path)
	}
	session, err := store.Load()
	if err != nil {
		if errors.Is(err, authstore.ErrSessionNotFound) {
			return authstore.Session{}, "", nil, exitcode.UsageError(errors.New("not logged in; run goalrail login <server_url>"))
		}
		return authstore.Session{}, "", nil, exitcode.RuntimeError(err)
	}
	client := options.Client
	if client == nil {
		client = http.DefaultClient
	}
	now := time.Now
	if options.Now != nil {
		now = options.Now
	}
	manager := &Manager{
		store:   store,
		client:  client,
		session: session,
	}
	if !session.AccessTokenExpiresAt.After(now().UTC()) {
		refreshed, err := manager.refresh(ctx)
		if err != nil {
			return authstore.Session{}, "", nil, err
		}
		session = refreshed
	}
	return session, strings.TrimRight(session.ServerURL, "/"), manager.Client(), nil
}

type Manager struct {
	mu      sync.Mutex
	store   SessionStore
	client  HTTPClient
	session authstore.Session
}

func (m *Manager) Client() HTTPClient {
	return retryingClient{manager: m}
}

func (m *Manager) refresh(ctx context.Context) (authstore.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.refreshLocked(ctx)
}

func (m *Manager) refreshLocked(ctx context.Context) (authstore.Session, error) {
	session := m.session
	serverURL := strings.TrimRight(session.ServerURL, "/")
	if strings.TrimSpace(session.RefreshToken) == "" {
		return authstore.Session{}, exitcode.UsageError(fmt.Errorf("login expired and no refresh token is available; run goalrail login %s", serverURL))
	}
	savingStore, ok := m.store.(SavingSessionStore)
	if !ok {
		return authstore.Session{}, exitcode.UsageError(fmt.Errorf("login expired; run goalrail login %s", serverURL))
	}

	body, err := json.Marshal(map[string]string{"refresh_token": session.RefreshToken})
	if err != nil {
		return authstore.Session{}, exitcode.RuntimeError(fmt.Errorf("encode token refresh request: %w", err))
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, serverURL+"/v1/auth/refresh", bytes.NewReader(body))
	if err != nil {
		return authstore.Session{}, exitcode.RuntimeError(fmt.Errorf("build token refresh request: %w", err))
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := m.client.Do(request)
	if err != nil {
		return authstore.Session{}, exitcode.RuntimeError(fmt.Errorf("refresh access token from %s: %w", serverURL, err))
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return authstore.Session{}, refreshHTTPError(response, serverURL)
	}
	var decoded refreshResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return authstore.Session{}, exitcode.RuntimeError(fmt.Errorf("decode token refresh response: %w", err))
	}
	refreshed, err := refreshedSession(session, decoded)
	if err != nil {
		return authstore.Session{}, err
	}
	if err := savingStore.Save(refreshed); err != nil {
		return authstore.Session{}, exitcode.RuntimeError(fmt.Errorf("save refreshed auth session: %w", err))
	}
	m.session = refreshed
	return refreshed, nil
}

func refreshedSession(existing authstore.Session, response refreshResponse) (authstore.Session, error) {
	accessToken := strings.TrimSpace(response.AccessToken)
	if accessToken == "" {
		return authstore.Session{}, exitcode.RuntimeError(errors.New("token refresh response did not include access_token"))
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(response.AccessTokenExpiresAt))
	if err != nil {
		return authstore.Session{}, exitcode.RuntimeError(fmt.Errorf("parse token refresh expiry: %w", err))
	}
	tokenType := strings.TrimSpace(response.TokenType)
	if tokenType == "" {
		tokenType = existing.TokenType
	}
	refreshToken := strings.TrimSpace(response.RefreshToken)
	if refreshToken == "" {
		refreshToken = existing.RefreshToken
	}
	return authstore.Session{
		ServerURL:            existing.ServerURL,
		AccessToken:          accessToken,
		RefreshToken:         refreshToken,
		AccessTokenExpiresAt: expiresAt.UTC(),
		TokenType:            tokenType,
	}, nil
}

func refreshHTTPError(response *http.Response, serverURL string) error {
	code := decodeServerErrorCode(response.Body)
	detail := fmt.Sprintf("HTTP %d", response.StatusCode)
	if code != "" {
		detail += " " + code
	}
	switch response.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return exitcode.UsageError(fmt.Errorf("refresh access token failed: %s; run goalrail login %s", detail, serverURL))
	default:
		return exitcode.RuntimeError(fmt.Errorf("refresh access token failed: %s", detail))
	}
}

func decodeServerErrorCode(body io.Reader) string {
	raw, err := io.ReadAll(io.LimitReader(body, 1<<20))
	if err != nil {
		return ""
	}
	var decoded serverErrorResponse
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return ""
	}
	return strings.TrimSpace(decoded.Error.Code)
}

type retryingClient struct {
	manager *Manager
}

func (c retryingClient) Do(request *http.Request) (*http.Response, error) {
	prepared, err := c.prepareRequest(request, false)
	if err != nil {
		return nil, err
	}
	response, err := c.manager.client.Do(prepared)
	if err != nil || response == nil || response.StatusCode != http.StatusUnauthorized || !requestAutoRetryAllowed(request) {
		return response, err
	}
	_ = response.Body.Close()

	if _, err := c.manager.refresh(request.Context()); err != nil {
		return nil, err
	}
	retry, err := c.prepareRequest(request, true)
	if err != nil {
		return nil, err
	}
	return c.manager.client.Do(retry)
}

func (c retryingClient) prepareRequest(request *http.Request, rebuildBody bool) (*http.Request, error) {
	prepared := request.Clone(request.Context())
	if rebuildBody && request.Body != nil {
		if request.GetBody == nil {
			return nil, exitcode.RuntimeError(errors.New("authenticated request body cannot be replayed"))
		}
		body, err := request.GetBody()
		if err != nil {
			return nil, exitcode.RuntimeError(fmt.Errorf("rebuild authenticated request body: %w", err))
		}
		prepared.Body = body
	}
	c.manager.mu.Lock()
	token := c.manager.session.AccessToken
	c.manager.mu.Unlock()
	prepared.Header.Set("Authorization", "Bearer "+token)
	return prepared, nil
}

func requestAutoRetryAllowed(request *http.Request) bool {
	switch request.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
	default:
		return false
	}
	return request.Body == nil || request.GetBody != nil
}
