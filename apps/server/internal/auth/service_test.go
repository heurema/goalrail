package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

func TestLoginVerifiesPasswordCreatesSessionAndReturnsTokens(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	store := newFakeAuthStore()
	service := NewService(
		store,
		strongTestJWTSecret,
		WithClock(fixedClock{now: now}),
		WithSessionIDGenerator(func() (spine.UserSessionID, error) {
			return "018f0000-0000-7000-8000-000000000201", nil
		}),
		WithRefreshTokenGenerator(func() (string, error) {
			return "opaque-refresh-token", nil
		}),
		WithPasswordHasher(func(input string) (string, error) {
			return "hash:" + input, nil
		}, func(input string, hash string) (bool, error) {
			return hash == "hash:"+input, nil
		}),
	)

	result, err := service.Login(ctx, LoginInput{
		Email:    " OWNER@EXAMPLE.COM ",
		Password: "temporary-password",
	})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if result.UserID != store.user.ID {
		t.Fatalf("UserID = %q, want %q", result.UserID, store.user.ID)
	}
	if result.AccessToken == "" {
		t.Fatal("AccessToken is empty")
	}
	if result.RefreshToken != "opaque-refresh-token" {
		t.Fatalf("RefreshToken = %q, want generated token", result.RefreshToken)
	}
	if !result.MustChangePassword {
		t.Fatal("MustChangePassword = false, want true")
	}
	if store.lastSession.ID != "018f0000-0000-7000-8000-000000000201" {
		t.Fatalf("session ID = %q, want generated session id", store.lastSession.ID)
	}
	if store.lastSession.RefreshTokenHash == "" || store.lastSession.RefreshTokenHash == "opaque-refresh-token" {
		t.Fatalf("refresh token hash = %q, want hashed opaque token", store.lastSession.RefreshTokenHash)
	}

	claims, err := service.accessTokens.Validate(result.AccessToken, now)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if claims.UserID != store.user.ID || claims.SessionID != store.lastSession.ID {
		t.Fatalf("claims = %#v, want user and session identity only", claims)
	}
}

func TestLoginRejectsWrongPassword(t *testing.T) {
	store := newFakeAuthStore()
	service := newFakeAuthService(store, time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC))

	_, err := service.Login(context.Background(), LoginInput{
		Email:    "owner@example.com",
		Password: "wrong-password",
	})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v, want ErrInvalidCredentials", err)
	}
	if store.lastSession.ID != "" {
		t.Fatalf("session was persisted after wrong password: %#v", store.lastSession)
	}
}

func TestLoginRejectsInactiveUser(t *testing.T) {
	store := newFakeAuthStore()
	store.user.State = "inactive"
	service := newFakeAuthService(store, time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC))

	_, err := service.Login(context.Background(), LoginInput{
		Email:    "owner@example.com",
		Password: "temporary-password",
	})
	if !errors.Is(err, ErrInactiveUser) {
		t.Fatalf("Login() error = %v, want ErrInactiveUser", err)
	}
}

func TestLoginRejectsUserWithoutActiveOrganizationMembership(t *testing.T) {
	store := newFakeAuthStore()
	store.membership.State = "inactive"
	service := newFakeAuthService(store, time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC))

	_, err := service.Login(context.Background(), LoginInput{
		Email:    "owner@example.com",
		Password: "temporary-password",
	})
	if !errors.Is(err, ErrMembershipRequired) {
		t.Fatalf("Login() error = %v, want ErrMembershipRequired", err)
	}
}

func TestLoginRequiresJWTSecretBeforeSessionCreation(t *testing.T) {
	store := newFakeAuthStore()
	service := NewService(
		store,
		"",
		WithClock(fixedClock{now: time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)}),
		WithSessionIDGenerator(func() (spine.UserSessionID, error) {
			return "018f0000-0000-7000-8000-000000000201", nil
		}),
		WithRefreshTokenGenerator(func() (string, error) {
			return "opaque-refresh-token", nil
		}),
		WithPasswordHasher(nil, func(input string, hash string) (bool, error) {
			return hash == "hash:"+input, nil
		}),
	)

	_, err := service.Login(context.Background(), LoginInput{
		Email:    "owner@example.com",
		Password: "temporary-password",
	})
	if !errors.Is(err, ErrJWTSecretMissing) {
		t.Fatalf("Login() error = %v, want ErrJWTSecretMissing", err)
	}
	if store.lastSession.ID != "" {
		t.Fatalf("session was persisted despite missing JWT secret: %#v", store.lastSession)
	}
}

func TestStartCLILoginStoresHashedCodeAndRedirectsWithCodeAndState(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	store := newFakeAuthStore()
	store.credential.MustChangePassword = false
	service := NewService(
		store,
		strongTestJWTSecret,
		WithClock(fixedClock{now: now}),
		WithCLIAuthCodeGenerator(func() (string, error) {
			return "one-time-code", nil
		}),
		WithPasswordHasher(nil, func(input string, hash string) (bool, error) {
			return hash == "hash:"+input, nil
		}),
	)

	result, err := service.StartCLILogin(ctx, CLILoginInput{
		Email:         "owner@example.com",
		Password:      "temporary-password",
		RedirectURI:   "http://127.0.0.1:49152/callback",
		State:         "state-1",
		CodeChallenge: codeChallengeS256("cli-code-verifier"),
	})
	if err != nil {
		t.Fatalf("StartCLILogin() error = %v", err)
	}
	if result.RedirectURI != "http://127.0.0.1:49152/callback?code=one-time-code&state=state-1" {
		t.Fatalf("RedirectURI = %q, want code/state redirect", result.RedirectURI)
	}
	if store.cliCode.CodeHash == "" || store.cliCode.CodeHash == "one-time-code" {
		t.Fatalf("CodeHash = %q, want hashed one-time code", store.cliCode.CodeHash)
	}
	if store.cliCode.UserID != store.user.ID || !store.cliCode.ExpiresAt.Equal(now.Add(5*time.Minute)) {
		t.Fatalf("stored code = %#v, want user and short TTL", store.cliCode)
	}
	if store.cliCode.CodeChallenge != codeChallengeS256("cli-code-verifier") || store.cliCode.CodeChallengeMethod != cliCodeChallengeMethod {
		t.Fatalf("stored code challenge = %q/%q, want S256 challenge", store.cliCode.CodeChallenge, store.cliCode.CodeChallengeMethod)
	}
}

func TestStartCLILoginRejectsMustChangePasswordCredential(t *testing.T) {
	store := newFakeAuthStore()
	service := newFakeAuthService(store, time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC))

	_, err := service.StartCLILogin(context.Background(), CLILoginInput{
		Email:         "owner@example.com",
		Password:      "temporary-password",
		RedirectURI:   "http://127.0.0.1:49152/callback",
		State:         "state-1",
		CodeChallenge: codeChallengeS256("cli-code-verifier"),
	})
	if !errors.Is(err, ErrPasswordChangeRequired) {
		t.Fatalf("StartCLILogin() error = %v, want ErrPasswordChangeRequired", err)
	}
	if store.cliCode.CodeHash != "" {
		t.Fatalf("stored CLI code = %#v, want none", store.cliCode)
	}
}

func TestStartCLILoginRejectsNonLocalhostRedirectTarget(t *testing.T) {
	store := newFakeAuthStore()
	service := newFakeAuthService(store, time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC))

	_, err := service.StartCLILogin(context.Background(), CLILoginInput{
		Email:         "owner@example.com",
		Password:      "temporary-password",
		RedirectURI:   "https://example.com/callback",
		State:         "state-1",
		CodeChallenge: codeChallengeS256("cli-code-verifier"),
	})
	if !errors.Is(err, ErrInvalidRedirectURI) {
		t.Fatalf("StartCLILogin() error = %v, want ErrInvalidRedirectURI", err)
	}
}

func TestExchangeCLIAuthCodeConsumesCodeAndCreatesTokens(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	store := newFakeAuthStore()
	store.cliCode = spine.CLIAuthCode{
		CodeHash:            hashOpaqueToken("one-time-code"),
		UserID:              store.user.ID,
		RedirectURI:         "http://127.0.0.1:49152/callback",
		State:               "state-1",
		CodeChallenge:       codeChallengeS256("cli-code-verifier"),
		CodeChallengeMethod: cliCodeChallengeMethod,
		CreatedAt:           now.Add(-time.Minute),
		ExpiresAt:           now.Add(4 * time.Minute),
	}
	service := NewService(
		store,
		strongTestJWTSecret,
		WithClock(fixedClock{now: now}),
		WithSessionIDGenerator(func() (spine.UserSessionID, error) {
			return "018f0000-0000-7000-8000-000000000202", nil
		}),
		WithRefreshTokenGenerator(func() (string, error) {
			return "cli-refresh-token", nil
		}),
	)

	result, err := service.ExchangeCLIAuthCode(ctx, CLIExchangeInput{Code: "one-time-code", State: "state-1", CodeVerifier: "cli-code-verifier"})
	if err != nil {
		t.Fatalf("ExchangeCLIAuthCode() error = %v", err)
	}
	if result.AccessToken == "" || result.RefreshToken != "cli-refresh-token" || result.TokenType != "Bearer" {
		t.Fatalf("result = %#v, want token response", result)
	}
	if store.cliCode.ConsumedAt == nil || !store.cliCode.ConsumedAt.Equal(now) {
		t.Fatalf("ConsumedAt = %v, want %v", store.cliCode.ConsumedAt, now)
	}
	if store.lastSession.ID != "018f0000-0000-7000-8000-000000000202" {
		t.Fatalf("session ID = %q, want generated session", store.lastSession.ID)
	}
}

func TestExchangeCLIAuthCodeRejectsUnknownExpiredAndUsedCode(t *testing.T) {
	now := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	usedAt := now.Add(-time.Minute)
	tests := []struct {
		name    string
		code    spine.CLIAuthCode
		input   CLIExchangeInput
		wantErr error
	}{
		{name: "unknown", input: CLIExchangeInput{Code: "missing", State: "state-1", CodeVerifier: "cli-code-verifier"}, wantErr: ErrCLIAuthCodeInvalid},
		{name: "expired", code: spine.CLIAuthCode{CodeHash: hashOpaqueToken("expired"), UserID: "018f0000-0000-7000-8000-000000000001", State: "state-1", CodeChallenge: codeChallengeS256("cli-code-verifier"), CodeChallengeMethod: cliCodeChallengeMethod, ExpiresAt: now.Add(-time.Second)}, input: CLIExchangeInput{Code: "expired", State: "state-1", CodeVerifier: "cli-code-verifier"}, wantErr: ErrCLIAuthCodeExpired},
		{name: "used", code: spine.CLIAuthCode{CodeHash: hashOpaqueToken("used"), UserID: "018f0000-0000-7000-8000-000000000001", State: "state-1", CodeChallenge: codeChallengeS256("cli-code-verifier"), CodeChallengeMethod: cliCodeChallengeMethod, ExpiresAt: now.Add(time.Minute), ConsumedAt: &usedAt}, input: CLIExchangeInput{Code: "used", State: "state-1", CodeVerifier: "cli-code-verifier"}, wantErr: ErrCLIAuthCodeUsed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newFakeAuthStore()
			store.cliCode = tt.code
			service := newFakeAuthService(store, now)

			_, err := service.ExchangeCLIAuthCode(context.Background(), tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ExchangeCLIAuthCode() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestExchangeCLIAuthCodeRequiresMatchingVerifier(t *testing.T) {
	now := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name  string
		input CLIExchangeInput
	}{
		{name: "wrong verifier", input: CLIExchangeInput{Code: "one-time-code", State: "state-1", CodeVerifier: "wrong-verifier"}},
		{name: "missing verifier", input: CLIExchangeInput{Code: "one-time-code", State: "state-1"}},
		{name: "code state alone", input: CLIExchangeInput{Code: "one-time-code", State: "state-1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newFakeAuthStore()
			store.cliCode = spine.CLIAuthCode{
				CodeHash:            hashOpaqueToken("one-time-code"),
				UserID:              store.user.ID,
				RedirectURI:         "http://127.0.0.1:49152/callback",
				State:               "state-1",
				CodeChallenge:       codeChallengeS256("cli-code-verifier"),
				CodeChallengeMethod: cliCodeChallengeMethod,
				CreatedAt:           now.Add(-time.Minute),
				ExpiresAt:           now.Add(4 * time.Minute),
			}
			service := newFakeAuthService(store, now)

			_, err := service.ExchangeCLIAuthCode(context.Background(), tt.input)
			if !errors.Is(err, ErrCLIAuthCodeInvalid) {
				t.Fatalf("ExchangeCLIAuthCode() error = %v, want ErrCLIAuthCodeInvalid", err)
			}
			if store.cliCode.ConsumedAt != nil {
				t.Fatalf("ConsumedAt = %v, want nil for rejected verifier", store.cliCode.ConsumedAt)
			}
			if store.lastSession.ID != "" {
				t.Fatalf("session was persisted for rejected verifier: %#v", store.lastSession)
			}
		})
	}
}

func TestRefreshWithValidRefreshTokenReturnsNewAccessToken(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	store := newFakeAuthStore()
	store.session.RefreshTokenHash = hashOpaqueToken("valid-refresh-token")
	service := newFakeAuthService(store, now)

	result, err := service.Refresh(ctx, RefreshInput{RefreshToken: "valid-refresh-token"})
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if result.AccessToken == "" {
		t.Fatal("AccessToken is empty")
	}
	if result.TokenType != "Bearer" {
		t.Fatalf("TokenType = %q, want Bearer", result.TokenType)
	}
	if store.session.LastUsedAt == nil || !store.session.LastUsedAt.Equal(now) {
		t.Fatalf("LastUsedAt = %v, want %v", store.session.LastUsedAt, now)
	}
	if store.session.RefreshTokenHash != hashOpaqueToken("valid-refresh-token") {
		t.Fatalf("refresh token hash changed, want existing refresh token preserved")
	}

	claims, err := service.accessTokens.Validate(result.AccessToken, now)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if claims.UserID != store.user.ID || claims.SessionID != store.session.ID {
		t.Fatalf("claims = %#v, want current user/session identity", claims)
	}
}

func TestRefreshRejectsUnknownRefreshToken(t *testing.T) {
	store := newFakeAuthStore()
	store.session.RefreshTokenHash = hashOpaqueToken("known-refresh-token")
	service := newFakeAuthService(store, time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC))

	_, err := service.Refresh(context.Background(), RefreshInput{RefreshToken: "unknown-refresh-token"})
	if !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("Refresh() error = %v, want ErrSessionInvalid", err)
	}
}

func TestRefreshRejectsRevokedSession(t *testing.T) {
	store := newFakeAuthStore()
	store.session.RefreshTokenHash = hashOpaqueToken("valid-refresh-token")
	store.session.State = spine.UserSessionStateRevoked
	service := newFakeAuthService(store, time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC))

	_, err := service.Refresh(context.Background(), RefreshInput{RefreshToken: "valid-refresh-token"})
	if !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("Refresh() error = %v, want ErrSessionInvalid", err)
	}
}

func TestRefreshRejectsExpiredSession(t *testing.T) {
	now := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	store := newFakeAuthStore()
	store.session.RefreshTokenHash = hashOpaqueToken("valid-refresh-token")
	store.session.ExpiresAt = now.Add(-time.Minute)
	service := newFakeAuthService(store, now)

	_, err := service.Refresh(context.Background(), RefreshInput{RefreshToken: "valid-refresh-token"})
	if !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("Refresh() error = %v, want ErrSessionInvalid", err)
	}
}

func TestRefreshRejectsInactiveUser(t *testing.T) {
	store := newFakeAuthStore()
	store.session.RefreshTokenHash = hashOpaqueToken("valid-refresh-token")
	store.user.State = "inactive"
	service := newFakeAuthService(store, time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC))

	_, err := service.Refresh(context.Background(), RefreshInput{RefreshToken: "valid-refresh-token"})
	if !errors.Is(err, ErrInactiveUser) {
		t.Fatalf("Refresh() error = %v, want ErrInactiveUser", err)
	}
}

func TestRefreshRequiresConfiguredJWTSecretBeforeSessionUpdate(t *testing.T) {
	now := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		secret  string
		wantErr error
	}{
		{name: "missing", secret: "", wantErr: ErrJWTSecretMissing},
		{name: "weak", secret: "0123456789abcdef0123456789abcde", wantErr: ErrJWTSecretWeak},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newFakeAuthStore()
			store.session.RefreshTokenHash = hashOpaqueToken("valid-refresh-token")
			service := NewService(store, tt.secret, WithClock(fixedClock{now: now}))

			_, err := service.Refresh(context.Background(), RefreshInput{RefreshToken: "valid-refresh-token"})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Refresh() error = %v, want %v", err, tt.wantErr)
			}
			if store.session.LastUsedAt != nil {
				t.Fatalf("LastUsedAt = %v, want no session update", store.session.LastUsedAt)
			}
		})
	}
}

func TestLogoutWithBearerTokenRevokesCurrentSession(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	store := newFakeAuthStore()
	service := newFakeAuthService(store, now)
	token := signTestAccessToken(t, service, store, now)

	result, err := service.Logout(ctx, token)
	if err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if !result.Revoked {
		t.Fatalf("Revoked = false, want true")
	}
	if store.session.State != spine.UserSessionStateRevoked {
		t.Fatalf("session state = %q, want revoked", store.session.State)
	}
	if store.session.RevokedAt == nil || !store.session.RevokedAt.Equal(now) {
		t.Fatalf("RevokedAt = %v, want %v", store.session.RevokedAt, now)
	}
}

func TestLogoutRejectsMissingBearerToken(t *testing.T) {
	store := newFakeAuthStore()
	service := newFakeAuthService(store, time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC))

	_, err := service.Logout(context.Background(), "")
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("Logout() error = %v, want ErrInvalidToken", err)
	}
}

func TestLogoutRejectsInvalidOrExpiredAccessToken(t *testing.T) {
	now := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	store := newFakeAuthStore()
	service := newFakeAuthService(store, now)
	expiredToken, err := service.accessTokens.Sign(AccessTokenClaims{
		UserID:    store.user.ID,
		SessionID: store.session.ID,
		IssuedAt:  now.Add(-30 * time.Minute),
		ExpiresAt: now.Add(-time.Minute),
	})
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}

	tests := []struct {
		name        string
		accessToken string
		wantErr     error
	}{
		{name: "invalid", accessToken: "not-a-jwt", wantErr: ErrInvalidToken},
		{name: "expired", accessToken: expiredToken, wantErr: ErrExpiredToken},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Logout(context.Background(), tt.accessToken)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Logout() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestChangePasswordRejectsWrongCurrentPassword(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	store := newFakeAuthStore()
	service := newFakeAuthService(store, now)
	token := signTestAccessToken(t, service, store, now)

	_, err := service.ChangePassword(ctx, token, ChangePasswordInput{
		CurrentPassword: "wrong-password",
		NewPassword:     "new-password",
	})
	if !errors.Is(err, ErrCurrentPassword) {
		t.Fatalf("ChangePassword() error = %v, want ErrCurrentPassword", err)
	}
	if store.credential.PasswordHash != "hash:temporary-password" {
		t.Fatalf("PasswordHash = %q, want unchanged old hash", store.credential.PasswordHash)
	}
}

func TestChangePasswordStoresNewHashClearsMustChangeAndRejectsOldPassword(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	store := newFakeAuthStore()
	service := newFakeAuthService(store, now)
	token := signTestAccessToken(t, service, store, now)

	result, err := service.ChangePassword(ctx, token, ChangePasswordInput{
		CurrentPassword: "temporary-password",
		NewPassword:     "new-password",
	})
	if err != nil {
		t.Fatalf("ChangePassword() error = %v", err)
	}
	if result.MustChangePassword {
		t.Fatal("MustChangePassword = true, want false")
	}
	if store.credential.MustChangePassword {
		t.Fatal("stored MustChangePassword = true, want false")
	}
	if store.credential.PasswordHash != "hash:new-password" {
		t.Fatalf("PasswordHash = %q, want new hash", store.credential.PasswordHash)
	}
	if store.credential.PasswordChangedAt == nil || !store.credential.PasswordChangedAt.Equal(now) {
		t.Fatalf("PasswordChangedAt = %v, want %v", store.credential.PasswordChangedAt, now)
	}

	_, err = service.Login(ctx, LoginInput{
		Email:    "owner@example.com",
		Password: "temporary-password",
	})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login(old password) error = %v, want ErrInvalidCredentials", err)
	}
	_, err = service.Login(ctx, LoginInput{
		Email:    "owner@example.com",
		Password: "new-password",
	})
	if err != nil {
		t.Fatalf("Login(new password) error = %v", err)
	}
}

func TestMeLoadsCurrentMembershipFromStore(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	store := newFakeAuthStore()
	service := NewService(store, strongTestJWTSecret, WithClock(fixedClock{now: now}))
	token := signTestAccessToken(t, service, store, now)

	profile, err := service.Me(ctx, token)
	if err != nil {
		t.Fatalf("Me() error = %v", err)
	}
	if profile.OrganizationMembership.Role != spine.OrganizationMembershipRoleOwner {
		t.Fatalf("Role = %q, want store owner role", profile.OrganizationMembership.Role)
	}
}

func TestAuthenticateAccessTokenRejectsExpiredOrRevokedSession(t *testing.T) {
	now := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		mutate  func(*fakeAuthStore)
		wantErr error
	}{
		{
			name: "expired",
			mutate: func(store *fakeAuthStore) {
				store.session.ExpiresAt = now.Add(-time.Minute)
			},
			wantErr: ErrSessionInvalid,
		},
		{
			name: "revoked",
			mutate: func(store *fakeAuthStore) {
				store.session.State = spine.UserSessionStateRevoked
			},
			wantErr: ErrSessionInvalid,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newFakeAuthStore()
			service := newFakeAuthService(store, now)
			token := signTestAccessToken(t, service, store, now)
			tt.mutate(store)

			_, err := service.AuthenticateAccessToken(context.Background(), token)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("AuthenticateAccessToken() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestAuthenticateAccessTokenRejectsInactiveUserAfterTokenIssuance(t *testing.T) {
	now := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	store := newFakeAuthStore()
	service := newFakeAuthService(store, now)
	token := signTestAccessToken(t, service, store, now)
	store.user.State = "inactive"

	_, err := service.AuthenticateAccessToken(context.Background(), token)
	if !errors.Is(err, ErrInactiveUser) {
		t.Fatalf("AuthenticateAccessToken() error = %v, want ErrInactiveUser", err)
	}
}

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

func newFakeAuthService(store *fakeAuthStore, now time.Time) *Service {
	return NewService(
		store,
		strongTestJWTSecret,
		WithClock(fixedClock{now: now}),
		WithSessionIDGenerator(func() (spine.UserSessionID, error) {
			return "018f0000-0000-7000-8000-000000000201", nil
		}),
		WithRefreshTokenGenerator(func() (string, error) {
			return "opaque-refresh-token", nil
		}),
		WithPasswordHasher(func(input string) (string, error) {
			return "hash:" + input, nil
		}, func(input string, hash string) (bool, error) {
			return hash == "hash:"+input, nil
		}),
	)
}

func signTestAccessToken(t *testing.T, service *Service, store *fakeAuthStore, now time.Time) string {
	t.Helper()
	token, err := service.accessTokens.Sign(AccessTokenClaims{
		UserID:    store.user.ID,
		SessionID: store.session.ID,
		IssuedAt:  now.Add(-time.Minute),
		ExpiresAt: now.Add(15 * time.Minute),
	})
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}
	return token
}

type fakeAuthStore struct {
	user        spine.User
	credential  spine.UserPasswordCredential
	session     spine.UserSession
	cliCode     spine.CLIAuthCode
	membership  spine.OrganizationMembership
	lastSession spine.UserSession
}

func newFakeAuthStore() *fakeAuthStore {
	now := time.Date(2026, 5, 3, 11, 0, 0, 0, time.UTC)
	user := spine.User{
		ID:          "018f0000-0000-7000-8000-000000000001",
		DisplayName: "Owner",
		Email:       "owner@example.com",
		State:       spine.EntityStateActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	return &fakeAuthStore{
		user: user,
		credential: spine.UserPasswordCredential{
			UserID:             user.ID,
			PasswordHash:       "hash:temporary-password",
			MustChangePassword: true,
			CreatedAt:          now,
			UpdatedAt:          now,
		},
		session: spine.UserSession{
			ID:               "018f0000-0000-7000-8000-000000000201",
			UserID:           user.ID,
			RefreshTokenHash: "existing-refresh-token-hash",
			State:            spine.UserSessionStateActive,
			CreatedAt:        now,
			UpdatedAt:        now,
			ExpiresAt:        now.Add(24 * time.Hour),
		},
		membership: spine.OrganizationMembership{
			ID:             "018f0000-0000-7000-8000-000000000301",
			OrganizationID: "018f0000-0000-7000-8000-000000000002",
			UserID:         user.ID,
			Role:           spine.OrganizationMembershipRoleOwner,
			State:          spine.EntityStateActive,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	}
}

func (s *fakeAuthStore) GetUserByEmail(_ context.Context, email string) (spine.User, bool, error) {
	return s.user, email == s.user.Email, nil
}

func (s *fakeAuthStore) GetUser(_ context.Context, userID spine.UserID) (spine.User, bool, error) {
	return s.user, userID == s.user.ID, nil
}

func (s *fakeAuthStore) GetPasswordCredential(_ context.Context, userID spine.UserID) (spine.UserPasswordCredential, bool, error) {
	return s.credential, userID == s.credential.UserID, nil
}

func (s *fakeAuthStore) UpsertPasswordCredential(_ context.Context, credential spine.UserPasswordCredential) error {
	s.credential = credential
	return nil
}

func (s *fakeAuthStore) UpsertSession(_ context.Context, session spine.UserSession) error {
	s.lastSession = session
	s.session = session
	return nil
}

func (s *fakeAuthStore) GetSession(_ context.Context, sessionID spine.UserSessionID) (spine.UserSession, bool, error) {
	return s.session, sessionID == s.session.ID, nil
}

func (s *fakeAuthStore) GetSessionByRefreshTokenHash(_ context.Context, refreshTokenHash string) (spine.UserSession, bool, error) {
	return s.session, refreshTokenHash == s.session.RefreshTokenHash, nil
}

func (s *fakeAuthStore) CreateCLIAuthCode(_ context.Context, code spine.CLIAuthCode) error {
	s.cliCode = code
	return nil
}

func (s *fakeAuthStore) GetCLIAuthCodeByHash(_ context.Context, codeHash string) (spine.CLIAuthCode, bool, error) {
	return s.cliCode, codeHash == s.cliCode.CodeHash, nil
}

func (s *fakeAuthStore) MarkCLIAuthCodeConsumed(_ context.Context, codeHash string, consumedAt time.Time) (bool, error) {
	if codeHash != s.cliCode.CodeHash || s.cliCode.ConsumedAt != nil {
		return false, nil
	}
	s.cliCode.ConsumedAt = &consumedAt
	return true, nil
}

func (s *fakeAuthStore) GetPrimaryOrganizationMembership(_ context.Context, userID spine.UserID) (spine.OrganizationMembership, bool, error) {
	return s.membership, userID == s.membership.UserID, nil
}
