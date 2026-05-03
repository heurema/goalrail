package httpserver

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/auth"
)

type AuthService interface {
	Login(context.Context, auth.LoginInput) (auth.LoginResult, error)
	Refresh(context.Context, auth.RefreshInput) (auth.RefreshResult, error)
	ChangePassword(context.Context, string, auth.ChangePasswordInput) (auth.ChangePasswordResult, error)
	Logout(context.Context, string) (auth.LogoutResult, error)
	Me(context.Context, string) (auth.Profile, error)
}

type AuthHandler struct {
	service AuthService
}

func NewAuthHandler(service AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

type loginResponse struct {
	UserID                string `json:"user_id"`
	AccessToken           string `json:"access_token"`
	AccessTokenExpiresAt  string `json:"access_token_expires_at"`
	TokenType             string `json:"token_type"`
	RefreshToken          string `json:"refresh_token"`
	RefreshTokenExpiresAt string `json:"refresh_token_expires_at"`
	MustChangePassword    bool   `json:"must_change_password"`
}

type changePasswordResponse struct {
	UserID             string `json:"user_id"`
	MustChangePassword bool   `json:"must_change_password"`
	PasswordChangedAt  string `json:"password_changed_at"`
}

type refreshResponse struct {
	UserID               string `json:"user_id"`
	AccessToken          string `json:"access_token"`
	AccessTokenExpiresAt string `json:"access_token_expires_at"`
	TokenType            string `json:"token_type"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var input auth.LoginInput
	if err := decodeStrictJSON(r.Body, &input); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid_json", "invalid JSON request body")
		return
	}
	result, err := h.service.Login(r.Context(), input)
	if err != nil {
		h.respondServiceError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, loginResponse{
		UserID:                string(result.UserID),
		AccessToken:           result.AccessToken,
		AccessTokenExpiresAt:  result.AccessTokenExpiresAt.Format(time.RFC3339Nano),
		TokenType:             result.TokenType,
		RefreshToken:          result.RefreshToken,
		RefreshTokenExpiresAt: result.RefreshTokenExpiresAt.Format(time.RFC3339Nano),
		MustChangePassword:    result.MustChangePassword,
	})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var input auth.RefreshInput
	if err := decodeStrictJSON(r.Body, &input); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid_json", "invalid JSON request body")
		return
	}
	result, err := h.service.Refresh(r.Context(), input)
	if err != nil {
		h.respondServiceError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, refreshResponse{
		UserID:               string(result.UserID),
		AccessToken:          result.AccessToken,
		AccessTokenExpiresAt: result.AccessTokenExpiresAt.Format(time.RFC3339Nano),
		TokenType:            result.TokenType,
	})
}

func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var input auth.ChangePasswordInput
	if err := decodeStrictJSON(r.Body, &input); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid_json", "invalid JSON request body")
		return
	}
	result, err := h.service.ChangePassword(r.Context(), bearerToken(r.Header.Get("Authorization")), input)
	if err != nil {
		h.respondServiceError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, changePasswordResponse{
		UserID:             string(result.UserID),
		MustChangePassword: result.MustChangePassword,
		PasswordChangedAt:  result.PasswordChangedAt.Format(time.RFC3339Nano),
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.Logout(r.Context(), bearerToken(r.Header.Get("Authorization")))
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	RespondJSON(w, http.StatusOK, result)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	profile, err := h.service.Me(r.Context(), bearerToken(r.Header.Get("Authorization")))
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	RespondJSON(w, http.StatusOK, profile)
}

func (h *AuthHandler) respondServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, auth.ErrStoreUnavailable):
		RespondError(w, http.StatusServiceUnavailable, "database_not_configured", "database is not configured")
	case errors.Is(err, auth.ErrJWTSecretMissing), errors.Is(err, auth.ErrJWTSecretWeak):
		RespondError(w, http.StatusServiceUnavailable, "auth_not_configured", "auth JWT secret is not configured")
	case errors.Is(err, auth.ErrInvalidCredentials):
		RespondError(w, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
	case errors.Is(err, auth.ErrCurrentPassword):
		RespondError(w, http.StatusUnauthorized, "invalid_current_password", "current password is invalid")
	case errors.Is(err, auth.ErrInvalidToken), errors.Is(err, auth.ErrExpiredToken), errors.Is(err, auth.ErrSessionInvalid):
		RespondError(w, http.StatusUnauthorized, "unauthorized", "valid bearer token is required")
	case errors.Is(err, auth.ErrInactiveUser):
		RespondError(w, http.StatusForbidden, "inactive_user", "user is inactive")
	case errors.Is(err, auth.ErrMembershipRequired):
		RespondError(w, http.StatusForbidden, "membership_required", "active organization membership is required")
	case errors.Is(err, auth.ErrNewPasswordRequired):
		RespondError(w, http.StatusBadRequest, "validation_failed", "new password is required")
	default:
		RespondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

func bearerToken(header string) string {
	scheme, token, ok := strings.Cut(strings.TrimSpace(header), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(token) == "" {
		return ""
	}
	return strings.TrimSpace(token)
}
