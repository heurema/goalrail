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

func (s *fakeAuthStore) GetPrimaryOrganizationMembership(_ context.Context, userID spine.UserID) (spine.OrganizationMembership, bool, error) {
	return s.membership, userID == s.membership.UserID, nil
}
