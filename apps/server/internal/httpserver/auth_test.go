package httpserver_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/auth"
	"github.com/heurema/goalrail/apps/server/internal/httpserver"
	"github.com/heurema/goalrail/apps/server/internal/spine"
)

func TestAuthLoginReturnsAccessAndRefreshTokens(t *testing.T) {
	now := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	handler := httpserver.NewAuthHandler(fakeHTTPAuthService{
		loginResult: auth.LoginResult{
			UserID:                "018f0000-0000-7000-8000-000000000001",
			AccessToken:           "access-token",
			AccessTokenExpiresAt:  now.Add(15 * time.Minute),
			TokenType:             "Bearer",
			RefreshToken:          "refresh-token",
			RefreshTokenExpiresAt: now.Add(30 * 24 * time.Hour),
			MustChangePassword:    true,
		},
	})

	response := doAuthRequest(t, http.HandlerFunc(handler.Login), http.MethodPost, "/v1/auth/login", `{"email":"owner@example.com","password":"temporary-password"}`, "")
	if response.code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusOK, response.body)
	}
	var body struct {
		AccessToken        string `json:"access_token"`
		RefreshToken       string `json:"refresh_token"`
		TokenType          string `json:"token_type"`
		MustChangePassword bool   `json:"must_change_password"`
	}
	decodeJSON(t, response.body, &body)
	if body.AccessToken != "access-token" || body.RefreshToken != "refresh-token" || body.TokenType != "Bearer" || !body.MustChangePassword {
		t.Fatalf("login response = %#v, want token response", body)
	}
}

func TestCLILoginPageRendersMinimalForm(t *testing.T) {
	handler := httpserver.NewAuthHandler(fakeHTTPAuthService{})

	response := doAuthRequest(t, http.HandlerFunc(handler.CLILoginPage), http.MethodGet, "/cli/login?redirect_uri=http%3A%2F%2F127.0.0.1%3A49152%2Fcallback&state=state-1&code_challenge=challenge-1", "", "")
	if response.code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusOK, response.body)
	}
	if !strings.Contains(response.body, "Goalrail CLI Login") || !strings.Contains(response.body, `name="redirect_uri"`) {
		t.Fatalf("body = %q, want minimal CLI login form", response.body)
	}
}

func TestCLILoginSubmitRejectsInvalidCredentials(t *testing.T) {
	handler := httpserver.NewAuthHandler(fakeHTTPAuthService{cliLoginErr: auth.ErrInvalidCredentials})

	response := doFormRequest(t, http.HandlerFunc(handler.CLILoginSubmit), "/cli/login", "email=owner%40example.com&password=wrong&redirect_uri=http%3A%2F%2F127.0.0.1%3A49152%2Fcallback&state=state-1&code_challenge=challenge-1")
	if response.code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusUnauthorized, response.body)
	}
	if !strings.Contains(response.body, "Invalid email or password") {
		t.Fatalf("body = %q, want invalid credentials message", response.body)
	}
}

func TestCLILoginSubmitRejectsNonLocalhostRedirectTarget(t *testing.T) {
	handler := httpserver.NewAuthHandler(fakeHTTPAuthService{cliLoginErr: auth.ErrInvalidRedirectURI})

	response := doFormRequest(t, http.HandlerFunc(handler.CLILoginSubmit), "/cli/login", "email=owner%40example.com&password=temporary-password&redirect_uri=https%3A%2F%2Fexample.com%2Fcallback&state=state-1&code_challenge=challenge-1")
	if response.code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusBadRequest, response.body)
	}
}

func TestCLILoginSubmitShowsPasswordChangeRequiredWithoutRedirect(t *testing.T) {
	handler := httpserver.NewAuthHandler(fakeHTTPAuthService{cliLoginErr: auth.ErrPasswordChangeRequired})

	response := doFormRequest(t, http.HandlerFunc(handler.CLILoginSubmit), "/cli/login", "email=owner%40example.com&password=temporary-password&redirect_uri=http%3A%2F%2F127.0.0.1%3A49152%2Fcallback&state=state-1&code_challenge=challenge-1")
	if response.code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusForbidden, response.body)
	}
	if !strings.Contains(response.body, "Password change required before CLI login.") {
		t.Fatalf("body = %q, want password-change-required message", response.body)
	}
	if location := response.header.Get("Location"); location != "" || strings.Contains(response.body, "code=one-time-code") {
		t.Fatalf("Location = %q body = %q, want no code redirect", location, response.body)
	}
}

func TestCLILoginSubmitRedirectsWithCodeAndStateOnly(t *testing.T) {
	handler := httpserver.NewAuthHandler(fakeHTTPAuthService{
		cliLoginResult: auth.CLILoginResult{RedirectURI: "http://127.0.0.1:49152/callback?code=one-time-code&state=state-1"},
	})

	response := doFormRequest(t, http.HandlerFunc(handler.CLILoginSubmit), "/cli/login", "email=owner%40example.com&password=temporary-password&redirect_uri=http%3A%2F%2F127.0.0.1%3A49152%2Fcallback&state=state-1&code_challenge=challenge-1")
	if response.code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusSeeOther, response.body)
	}
	location := response.header.Get("Location")
	if !strings.Contains(location, "code=one-time-code") || !strings.Contains(location, "state=state-1") {
		t.Fatalf("Location = %q, want code and state", location)
	}
	if strings.Contains(location, "access_token") || strings.Contains(location, "refresh_token") || strings.Contains(location, "code_verifier") {
		t.Fatalf("Location = %q, must not include tokens", location)
	}
}

func TestCLIExchangeReturnsTokens(t *testing.T) {
	now := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	handler := httpserver.NewAuthHandler(fakeHTTPAuthService{
		cliExchangeResult: auth.CLIExchangeResult{
			UserID:               "018f0000-0000-7000-8000-000000000001",
			AccessToken:          "access-token",
			AccessTokenExpiresAt: now.Add(15 * time.Minute),
			TokenType:            "Bearer",
			RefreshToken:         "refresh-token",
		},
	})

	response := doAuthRequest(t, http.HandlerFunc(handler.CLIExchange), http.MethodPost, "/v1/auth/cli/exchange", `{"code":"one-time-code","state":"state-1","code_verifier":"cli-code-verifier"}`, "")
	if response.code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusOK, response.body)
	}
	var body map[string]any
	decodeJSON(t, response.body, &body)
	if body["access_token"] != "access-token" || body["refresh_token"] != "refresh-token" || body["token_type"] != "Bearer" {
		t.Fatalf("exchange response = %#v, want token metadata", body)
	}
}

func TestCLIExchangeRejectsInvalidCode(t *testing.T) {
	handler := httpserver.NewAuthHandler(fakeHTTPAuthService{cliExchangeErr: auth.ErrCLIAuthCodeUsed})

	response := doAuthRequest(t, http.HandlerFunc(handler.CLIExchange), http.MethodPost, "/v1/auth/cli/exchange", `{"code":"used","state":"state-1","code_verifier":"cli-code-verifier"}`, "")
	if response.code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusUnauthorized, response.body)
	}
	if !strings.Contains(response.body, "invalid_cli_code") {
		t.Fatalf("body = %q, want invalid_cli_code", response.body)
	}
}

func TestAuthRefreshReturnsNewAccessTokenOnly(t *testing.T) {
	now := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	handler := httpserver.NewAuthHandler(fakeHTTPAuthService{
		refreshResult: auth.RefreshResult{
			UserID:               "018f0000-0000-7000-8000-000000000001",
			AccessToken:          "new-access-token",
			AccessTokenExpiresAt: now.Add(15 * time.Minute),
			TokenType:            "Bearer",
		},
	})

	response := doAuthRequest(t, http.HandlerFunc(handler.Refresh), http.MethodPost, "/v1/auth/refresh", `{"refresh_token":"refresh-token"}`, "")
	if response.code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusOK, response.body)
	}
	var body map[string]any
	decodeJSON(t, response.body, &body)
	if body["access_token"] != "new-access-token" || body["token_type"] != "Bearer" {
		t.Fatalf("refresh response = %#v, want access token response", body)
	}
	if _, ok := body["refresh_token"]; ok {
		t.Fatalf("refresh response included refresh_token: %#v", body)
	}
}

func TestAuthRefreshRejectsUnknownRefreshToken(t *testing.T) {
	handler := httpserver.NewAuthHandler(fakeHTTPAuthService{refreshErr: auth.ErrSessionInvalid})

	response := doAuthRequest(t, http.HandlerFunc(handler.Refresh), http.MethodPost, "/v1/auth/refresh", `{"refresh_token":"unknown"}`, "")
	if response.code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusUnauthorized, response.body)
	}
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeJSON(t, response.body, &body)
	if body.Error.Code != "unauthorized" {
		t.Fatalf("error code = %q, want unauthorized", body.Error.Code)
	}
}

func TestAuthChangePasswordRequiresBearerToken(t *testing.T) {
	handler := httpserver.NewAuthHandler(fakeHTTPAuthService{changePasswordErr: auth.ErrInvalidToken})

	response := doAuthRequest(t, http.HandlerFunc(handler.ChangePassword), http.MethodPost, "/v1/auth/change-password", `{"current_password":"old","new_password":"new"}`, "")
	if response.code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusUnauthorized, response.body)
	}
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeJSON(t, response.body, &body)
	if body.Error.Code != "unauthorized" {
		t.Fatalf("error code = %q, want unauthorized", body.Error.Code)
	}
}

func TestAuthLogoutRevokesCurrentSession(t *testing.T) {
	handler := httpserver.NewAuthHandler(fakeHTTPAuthService{
		logoutResult: auth.LogoutResult{Revoked: true},
	})

	response := doAuthRequest(t, http.HandlerFunc(handler.Logout), http.MethodPost, "/v1/auth/logout", "", "Bearer access-token")
	if response.code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusOK, response.body)
	}
	var body struct {
		Revoked bool `json:"revoked"`
	}
	decodeJSON(t, response.body, &body)
	if !body.Revoked {
		t.Fatalf("Revoked = false, want true")
	}
}

func TestAuthLogoutWithoutBearerTokenReturnsUnauthorized(t *testing.T) {
	handler := httpserver.NewAuthHandler(fakeHTTPAuthService{logoutErr: auth.ErrInvalidToken})

	response := doAuthRequest(t, http.HandlerFunc(handler.Logout), http.MethodPost, "/v1/auth/logout", "", "")
	if response.code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusUnauthorized, response.body)
	}
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeJSON(t, response.body, &body)
	if body.Error.Code != "unauthorized" {
		t.Fatalf("error code = %q, want unauthorized", body.Error.Code)
	}
}

func TestAuthMeReturnsProfile(t *testing.T) {
	handler := httpserver.NewAuthHandler(fakeHTTPAuthService{
		profile: auth.Profile{
			User: spine.User{
				ID:          "018f0000-0000-7000-8000-000000000001",
				DisplayName: "Owner",
				Email:       "owner@example.com",
				State:       spine.EntityStateActive,
			},
			OrganizationMembership: spine.OrganizationMembership{
				ID:             "018f0000-0000-7000-8000-000000000301",
				OrganizationID: "018f0000-0000-7000-8000-000000000002",
				UserID:         "018f0000-0000-7000-8000-000000000001",
				Role:           spine.OrganizationMembershipRoleOwner,
				State:          spine.EntityStateActive,
			},
		},
	})

	response := doAuthRequest(t, http.HandlerFunc(handler.Me), http.MethodGet, "/v1/me", "", "Bearer access-token")
	if response.code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusOK, response.body)
	}
	var body struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
		OrganizationMembership struct {
			Role string `json:"role"`
		} `json:"organization_membership"`
	}
	decodeJSON(t, response.body, &body)
	if body.User.ID != "018f0000-0000-7000-8000-000000000001" || body.OrganizationMembership.Role != "owner" {
		t.Fatalf("profile = %#v, want current user and server-loaded membership", body)
	}
}

func TestAuthMissingJWTSecretReturnsConfigurationError(t *testing.T) {
	handler := httpserver.NewAuthHandler(fakeHTTPAuthService{loginErr: auth.ErrJWTSecretMissing})

	response := doAuthRequest(t, http.HandlerFunc(handler.Login), http.MethodPost, "/v1/auth/login", `{"email":"owner@example.com","password":"temporary-password"}`, "")
	if response.code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusServiceUnavailable, response.body)
	}
	if !strings.Contains(response.body, "auth_not_configured") {
		t.Fatalf("body = %s, want auth_not_configured", response.body)
	}
}

func TestAuthWeakJWTSecretReturnsConfigurationError(t *testing.T) {
	handler := httpserver.NewAuthHandler(fakeHTTPAuthService{loginErr: auth.ErrJWTSecretWeak})

	response := doAuthRequest(t, http.HandlerFunc(handler.Login), http.MethodPost, "/v1/auth/login", `{"email":"owner@example.com","password":"temporary-password"}`, "")
	if response.code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusServiceUnavailable, response.body)
	}
	if !strings.Contains(response.body, "auth_not_configured") {
		t.Fatalf("body = %s, want auth_not_configured", response.body)
	}
}

func TestAuthRefreshMissingOrWeakJWTSecretReturnsConfigurationError(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{name: "missing", err: auth.ErrJWTSecretMissing},
		{name: "weak", err: auth.ErrJWTSecretWeak},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := httpserver.NewAuthHandler(fakeHTTPAuthService{refreshErr: tt.err})

			response := doAuthRequest(t, http.HandlerFunc(handler.Refresh), http.MethodPost, "/v1/auth/refresh", `{"refresh_token":"refresh-token"}`, "")
			if response.code != http.StatusServiceUnavailable {
				t.Fatalf("status = %d, want %d: %s", response.code, http.StatusServiceUnavailable, response.body)
			}
			if !strings.Contains(response.body, "auth_not_configured") {
				t.Fatalf("body = %s, want auth_not_configured", response.body)
			}
		})
	}
}

func TestAuthMeWithoutBearerTokenReturnsUnauthorized(t *testing.T) {
	handler := httpserver.NewAuthHandler(fakeHTTPAuthService{meErr: auth.ErrInvalidToken})

	response := doAuthRequest(t, http.HandlerFunc(handler.Me), http.MethodGet, "/v1/me", "", "")
	if response.code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusUnauthorized, response.body)
	}
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeJSON(t, response.body, &body)
	if body.Error.Code != "unauthorized" {
		t.Fatalf("error code = %q, want unauthorized", body.Error.Code)
	}
}

func TestAuthLogoutInvalidOrExpiredBearerTokenReturnsUnauthorized(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{name: "invalid", err: auth.ErrInvalidToken},
		{name: "expired", err: auth.ErrExpiredToken},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := httpserver.NewAuthHandler(fakeHTTPAuthService{logoutErr: tt.err})

			response := doAuthRequest(t, http.HandlerFunc(handler.Logout), http.MethodPost, "/v1/auth/logout", "", "Bearer access-token")
			if response.code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d: %s", response.code, http.StatusUnauthorized, response.body)
			}
			var body struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			decodeJSON(t, response.body, &body)
			if body.Error.Code != "unauthorized" {
				t.Fatalf("error code = %q, want unauthorized", body.Error.Code)
			}
		})
	}
}

type fakeHTTPAuthService struct {
	loginResult          auth.LoginResult
	loginErr             error
	cliLoginResult       auth.CLILoginResult
	cliLoginErr          error
	cliExchangeResult    auth.CLIExchangeResult
	cliExchangeErr       error
	refreshResult        auth.RefreshResult
	refreshErr           error
	changePasswordResult auth.ChangePasswordResult
	changePasswordErr    error
	logoutResult         auth.LogoutResult
	logoutErr            error
	profile              auth.Profile
	meErr                error
}

func (s fakeHTTPAuthService) Login(context.Context, auth.LoginInput) (auth.LoginResult, error) {
	if s.loginErr != nil {
		return auth.LoginResult{}, s.loginErr
	}
	return s.loginResult, nil
}

func (s fakeHTTPAuthService) StartCLILogin(context.Context, auth.CLILoginInput) (auth.CLILoginResult, error) {
	if s.cliLoginErr != nil {
		return auth.CLILoginResult{}, s.cliLoginErr
	}
	return s.cliLoginResult, nil
}

func (s fakeHTTPAuthService) ExchangeCLIAuthCode(context.Context, auth.CLIExchangeInput) (auth.CLIExchangeResult, error) {
	if s.cliExchangeErr != nil {
		return auth.CLIExchangeResult{}, s.cliExchangeErr
	}
	return s.cliExchangeResult, nil
}

func (s fakeHTTPAuthService) Refresh(context.Context, auth.RefreshInput) (auth.RefreshResult, error) {
	if s.refreshErr != nil {
		return auth.RefreshResult{}, s.refreshErr
	}
	return s.refreshResult, nil
}

func (s fakeHTTPAuthService) ChangePassword(context.Context, string, auth.ChangePasswordInput) (auth.ChangePasswordResult, error) {
	if s.changePasswordErr != nil {
		return auth.ChangePasswordResult{}, s.changePasswordErr
	}
	return s.changePasswordResult, nil
}

func (s fakeHTTPAuthService) Logout(context.Context, string) (auth.LogoutResult, error) {
	if s.logoutErr != nil {
		return auth.LogoutResult{}, s.logoutErr
	}
	return s.logoutResult, nil
}

func (s fakeHTTPAuthService) Me(context.Context, string) (auth.Profile, error) {
	if s.meErr != nil {
		return auth.Profile{}, s.meErr
	}
	return s.profile, nil
}

func doAuthRequest(t *testing.T, handler http.Handler, method string, path string, body string, authorization string) routeResponse {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}
	handler.ServeHTTP(recorder, request)
	return routeResponse{
		code:        recorder.Code,
		contentType: recorder.Header().Get("Content-Type"),
		header:      recorder.Header(),
		body:        recorder.Body.String(),
	}
}

func doFormRequest(t *testing.T, handler http.Handler, path string, body string) routeResponse {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	handler.ServeHTTP(recorder, request)
	return routeResponse{
		code:        recorder.Code,
		contentType: recorder.Header().Get("Content-Type"),
		header:      recorder.Header(),
		body:        recorder.Body.String(),
	}
}
