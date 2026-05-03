package authstore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileStoreWritesSessionWithRestrictedPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "goalrail", "auth.json")
	store := NewFileStore(path)
	expiresAt := time.Date(2026, 5, 3, 12, 15, 0, 0, time.UTC)

	if err := store.Save(Session{
		ServerURL:            "https://goalrail.example.com",
		AccessToken:          "access-token",
		RefreshToken:         "refresh-token",
		AccessTokenExpiresAt: expiresAt,
		TokenType:            "Bearer",
	}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat auth file: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("auth file permissions = %o, want 0600", got)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read auth file: %v", err)
	}
	var session Session
	if err := json.Unmarshal(raw, &session); err != nil {
		t.Fatalf("decode auth file: %v", err)
	}
	if session.ServerURL != "https://goalrail.example.com" || session.AccessToken != "access-token" || session.RefreshToken != "refresh-token" || session.TokenType != "Bearer" {
		t.Fatalf("session = %#v, want token metadata", session)
	}
	if !session.AccessTokenExpiresAt.Equal(expiresAt) {
		t.Fatalf("AccessTokenExpiresAt = %v, want %v", session.AccessTokenExpiresAt, expiresAt)
	}
}
