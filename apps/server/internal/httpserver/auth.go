package httpserver

import (
	"context"
	"errors"
	"html"
	"net/http"
	"strings"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/auth"
)

type AuthService interface {
	Login(context.Context, auth.LoginInput) (auth.LoginResult, error)
	StartCLILogin(context.Context, auth.CLILoginInput) (auth.CLILoginResult, error)
	ExchangeCLIAuthCode(context.Context, auth.CLIExchangeInput) (auth.CLIExchangeResult, error)
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

type cliExchangeResponse struct {
	UserID               string `json:"user_id"`
	AccessToken          string `json:"access_token"`
	AccessTokenExpiresAt string `json:"access_token_expires_at"`
	TokenType            string `json:"token_type"`
	RefreshToken         string `json:"refresh_token"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var input auth.LoginInput
	if err := decodeStrictJSON(r.Body, &input); err != nil {
		respondInvalidJSON(w)
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

func (h *AuthHandler) CLILoginPage(w http.ResponseWriter, r *http.Request) {
	redirectURI := strings.TrimSpace(r.URL.Query().Get("redirect_uri"))
	state := strings.TrimSpace(r.URL.Query().Get("state"))
	codeChallenge := strings.TrimSpace(r.URL.Query().Get("code_challenge"))
	if err := auth.ValidateLoopbackRedirectURI(redirectURI); err != nil || state == "" || codeChallenge == "" {
		respondHTML(w, http.StatusBadRequest, minimalCLIErrorPage("Invalid CLI login request."))
		return
	}
	respondHTML(w, http.StatusOK, minimalCLILoginPage(redirectURI, state, codeChallenge, ""))
}

func (h *AuthHandler) CLILoginSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		respondHTML(w, http.StatusBadRequest, minimalCLIErrorPage("Invalid CLI login request."))
		return
	}
	result, err := h.service.StartCLILogin(r.Context(), auth.CLILoginInput{
		Email:         r.FormValue("email"),
		Password:      r.FormValue("password"),
		RedirectURI:   r.FormValue("redirect_uri"),
		State:         r.FormValue("state"),
		CodeChallenge: r.FormValue("code_challenge"),
	})
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrInvalidCredentials):
			respondHTML(w, http.StatusUnauthorized, minimalCLILoginPage(r.FormValue("redirect_uri"), r.FormValue("state"), r.FormValue("code_challenge"), "Invalid email or password."))
		case errors.Is(err, auth.ErrPasswordChangeRequired):
			respondHTML(w, http.StatusForbidden, minimalCLILoginPage(r.FormValue("redirect_uri"), r.FormValue("state"), r.FormValue("code_challenge"), "Password change required before CLI login."))
		case errors.Is(err, auth.ErrInvalidRedirectURI), errors.Is(err, auth.ErrStateInvalid), errors.Is(err, auth.ErrCLIAuthCodeInvalid):
			respondHTML(w, http.StatusBadRequest, minimalCLIErrorPage("Invalid CLI login request."))
		default:
			h.respondServiceError(w, err)
		}
		return
	}
	http.Redirect(w, r, result.RedirectURI, http.StatusSeeOther)
}

func (h *AuthHandler) CLIExchange(w http.ResponseWriter, r *http.Request) {
	var input auth.CLIExchangeInput
	if err := decodeStrictJSON(r.Body, &input); err != nil {
		respondInvalidJSON(w)
		return
	}
	result, err := h.service.ExchangeCLIAuthCode(r.Context(), input)
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	RespondJSON(w, http.StatusOK, cliExchangeResponse{
		UserID:               string(result.UserID),
		AccessToken:          result.AccessToken,
		AccessTokenExpiresAt: result.AccessTokenExpiresAt.Format(time.RFC3339Nano),
		TokenType:            result.TokenType,
		RefreshToken:         result.RefreshToken,
	})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var input auth.RefreshInput
	if err := decodeStrictJSON(r.Body, &input); err != nil {
		respondInvalidJSON(w)
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
		respondInvalidJSON(w)
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
	case errors.Is(err, auth.ErrJWTSecretMissing), errors.Is(err, auth.ErrJWTSecretWeak):
		RespondError(w, http.StatusServiceUnavailable, "auth_not_configured", "auth JWT secret is not configured")
	case errors.Is(err, auth.ErrInvalidCredentials):
		RespondError(w, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
	case errors.Is(err, auth.ErrCurrentPassword):
		RespondError(w, http.StatusUnauthorized, "invalid_current_password", "current password is invalid")
	case errors.Is(err, auth.ErrPasswordChangeRequired):
		RespondError(w, http.StatusForbidden, "password_change_required", "password change required before CLI login")
	case errors.Is(err, auth.ErrCLIAuthCodeInvalid), errors.Is(err, auth.ErrCLIAuthCodeExpired), errors.Is(err, auth.ErrCLIAuthCodeUsed), errors.Is(err, auth.ErrStateInvalid):
		RespondError(w, http.StatusUnauthorized, "invalid_cli_code", "CLI authorization code is invalid")
	case errors.Is(err, auth.ErrInvalidToken), errors.Is(err, auth.ErrExpiredToken), errors.Is(err, auth.ErrSessionInvalid):
		RespondError(w, http.StatusUnauthorized, "unauthorized", "valid bearer token is required")
	case errors.Is(err, auth.ErrInactiveUser):
		RespondError(w, http.StatusForbidden, "inactive_user", "user is inactive")
	case errors.Is(err, auth.ErrMembershipRequired):
		RespondError(w, http.StatusForbidden, "membership_required", "active organization membership is required")
	case errors.Is(err, auth.ErrNewPasswordRequired):
		RespondError(w, http.StatusBadRequest, "validation_failed", "new password is required")
	default:
		respondInternalError(w)
	}
}

func respondHTML(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

// /cli/login is a temporary server-rendered CLI auth bridge for
// `goalrail login <server_url>` localhost loopback before a React console login
// exists. It is not the product web console login UI; future React console login
// should replace or front this flow.
func minimalCLILoginPage(redirectURI string, state string, codeChallenge string, message string) string {
	messageHTML := ""
	if strings.TrimSpace(message) != "" {
		messageHTML = `<p role="alert">` + html.EscapeString(message) + `</p>`
	}
	return `<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><title>Goalrail CLI Login</title></head>
<body>
<main>
<h1>Goalrail CLI Login</h1>
` + messageHTML + `
<form method="post" action="/cli/login">
<input type="hidden" name="redirect_uri" value="` + html.EscapeString(redirectURI) + `">
<input type="hidden" name="state" value="` + html.EscapeString(state) + `">
<input type="hidden" name="code_challenge" value="` + html.EscapeString(codeChallenge) + `">
<label>Email <input name="email" type="email" autocomplete="username" required></label>
<label>Password <input name="password" type="password" autocomplete="current-password" required></label>
<button type="submit">Log in</button>
</form>
</main>
</body>
</html>`
}

func minimalCLIErrorPage(message string) string {
	return `<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><title>Goalrail CLI Login</title></head>
<body><main><h1>Goalrail CLI Login</h1><p role="alert">` + html.EscapeString(message) + `</p></main></body>
</html>`
}

func bearerToken(header string) string {
	scheme, token, ok := strings.Cut(strings.TrimSpace(header), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(token) == "" {
		return ""
	}
	return strings.TrimSpace(token)
}
