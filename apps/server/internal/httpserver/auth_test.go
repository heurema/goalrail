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

type fakeHTTPAuthService struct {
	loginResult          auth.LoginResult
	loginErr             error
	changePasswordResult auth.ChangePasswordResult
	changePasswordErr    error
	profile              auth.Profile
	meErr                error
}

func (s fakeHTTPAuthService) Login(context.Context, auth.LoginInput) (auth.LoginResult, error) {
	if s.loginErr != nil {
		return auth.LoginResult{}, s.loginErr
	}
	return s.loginResult, nil
}

func (s fakeHTTPAuthService) ChangePassword(context.Context, string, auth.ChangePasswordInput) (auth.ChangePasswordResult, error) {
	if s.changePasswordErr != nil {
		return auth.ChangePasswordResult{}, s.changePasswordErr
	}
	return s.changePasswordResult, nil
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
		body:        recorder.Body.String(),
	}
}
