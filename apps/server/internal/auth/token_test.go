package auth

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

const strongTestJWTSecret = "0123456789abcdef0123456789abcdef"

func TestAccessTokenCarriesOnlyIdentityAndSessionClaims(t *testing.T) {
	now := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	manager := NewAccessTokenManager(strongTestJWTSecret)

	token, err := manager.Sign(AccessTokenClaims{
		UserID:    "018f0000-0000-7000-8000-000000000001",
		SessionID: "018f0000-0000-7000-8000-000000000201",
		IssuedAt:  now,
		ExpiresAt: now.Add(15 * time.Minute),
	})
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}

	claims, err := manager.Validate(token, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if claims.UserID != spine.UserID("018f0000-0000-7000-8000-000000000001") {
		t.Fatalf("UserID = %q, want signed subject", claims.UserID)
	}
	if claims.SessionID != spine.UserSessionID("018f0000-0000-7000-8000-000000000201") {
		t.Fatalf("SessionID = %q, want signed session", claims.SessionID)
	}

	payload := tokenDebugPayload(t, token)
	forbidden := []string{"role", "roles", "organization_role", "permissions"}
	for _, key := range forbidden {
		if _, ok := payload[key]; ok {
			t.Fatalf("payload contains forbidden claim %q: %#v", key, payload)
		}
	}
}

func tokenDebugPayload(t *testing.T, token string) map[string]any {
	t.Helper()
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("token parts = %d, want 3", len(parts))
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return payload
}

func TestAccessTokenRequiresSecret(t *testing.T) {
	tests := []struct {
		name    string
		secret  string
		wantErr error
	}{
		{name: "empty", secret: "", wantErr: ErrJWTSecretMissing},
		{name: "blank", secret: "   ", wantErr: ErrJWTSecretMissing},
		{name: "too_short", secret: "0123456789abcdef0123456789abcde", wantErr: ErrJWTSecretWeak},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewAccessTokenManager(tt.secret)
			_, err := manager.Sign(AccessTokenClaims{
				UserID:    "018f0000-0000-7000-8000-000000000001",
				SessionID: "018f0000-0000-7000-8000-000000000201",
				IssuedAt:  time.Now(),
				ExpiresAt: time.Now().Add(time.Minute),
			})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Sign() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestAccessTokenRejectsExpiredToken(t *testing.T) {
	now := time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
	manager := NewAccessTokenManager(strongTestJWTSecret)
	token, err := manager.Sign(AccessTokenClaims{
		UserID:    "018f0000-0000-7000-8000-000000000001",
		SessionID: "018f0000-0000-7000-8000-000000000201",
		IssuedAt:  now,
		ExpiresAt: now.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("Sign() error = %v", err)
	}
	if _, err := manager.Validate(token, now.Add(2*time.Minute)); !errors.Is(err, ErrExpiredToken) {
		t.Fatalf("Validate() error = %v, want ErrExpiredToken", err)
	}
}
