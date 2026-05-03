package store

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

func TestPostgresAuthStoreUpsertsPasswordCredential(t *testing.T) {
	ctx := context.Background()
	exec := &recordingProjectContextExecer{}
	store := NewPostgresAuthStoreWithExecutorAndQuerier(exec, nil)
	now := time.Date(2026, 5, 3, 10, 0, 0, 0, time.UTC)

	err := store.UpsertPasswordCredential(ctx, spine.UserPasswordCredential{
		UserID:             "018f0000-0000-7000-8000-000000000001",
		PasswordHash:       "$argon2id$v=19$m=19456,t=2,p=1$c2FsdA$a2V5",
		MustChangePassword: true,
		CreatedAt:          now,
		UpdatedAt:          now,
	})
	if err != nil {
		t.Fatalf("UpsertPasswordCredential() error = %v", err)
	}

	if len(exec.calls) != 1 {
		t.Fatalf("Exec calls = %d, want 1", len(exec.calls))
	}
	call := exec.calls[0]
	if !strings.Contains(call.sql, "INSERT INTO user_password_credentials") {
		t.Fatalf("SQL = %q, want user_password_credentials insert", call.sql)
	}
	if !strings.Contains(call.sql, "ON CONFLICT (user_id) DO UPDATE SET") {
		t.Fatalf("SQL = %q, want user_id upsert", call.sql)
	}
	if !strings.Contains(call.sql, "$1") {
		t.Fatalf("SQL = %q, want PostgreSQL placeholders", call.sql)
	}
	if got, want := len(call.args), 6; got != want {
		t.Fatalf("args len = %d, want %d", got, want)
	}
}

func TestPostgresAuthStoreUpsertsUserSession(t *testing.T) {
	ctx := context.Background()
	exec := &recordingProjectContextExecer{}
	store := NewPostgresAuthStoreWithExecutorAndQuerier(exec, nil)
	now := time.Date(2026, 5, 3, 10, 0, 0, 0, time.UTC)
	lastUsedAt := now.Add(time.Minute)

	err := store.UpsertSession(ctx, spine.UserSession{
		ID:               "018f0000-0000-7000-8000-000000000201",
		UserID:           "018f0000-0000-7000-8000-000000000001",
		RefreshTokenHash: "refresh-token-hash",
		State:            spine.UserSessionStateActive,
		CreatedAt:        now,
		UpdatedAt:        now,
		ExpiresAt:        now.Add(24 * time.Hour),
		LastUsedAt:       &lastUsedAt,
	})
	if err != nil {
		t.Fatalf("UpsertSession() error = %v", err)
	}

	if len(exec.calls) != 1 {
		t.Fatalf("Exec calls = %d, want 1", len(exec.calls))
	}
	call := exec.calls[0]
	if !strings.Contains(call.sql, "INSERT INTO user_sessions") {
		t.Fatalf("SQL = %q, want user_sessions insert", call.sql)
	}
	if !strings.Contains(call.sql, "ON CONFLICT (id) DO UPDATE SET") {
		t.Fatalf("SQL = %q, want id upsert", call.sql)
	}
	if got, want := len(call.args), 9; got != want {
		t.Fatalf("args len = %d, want %d", got, want)
	}
}

func TestPostgresAuthStoreGetsPasswordCredential(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 3, 10, 0, 0, 0, time.UTC)
	changedAt := now.Add(time.Hour)
	query := &recordingProjectContextQuerier{
		row: fakeProjectContextRow{
			values: []any{
				"018f0000-0000-7000-8000-000000000001",
				"$argon2id$v=19$m=19456,t=2,p=1$c2FsdA$a2V5",
				false,
				changedAt,
				now,
				now,
			},
		},
	}
	store := NewPostgresAuthStoreWithExecutorAndQuerier(&recordingProjectContextExecer{}, query)

	credential, ok, err := store.GetPasswordCredential(ctx, "018f0000-0000-7000-8000-000000000001")
	if err != nil {
		t.Fatalf("GetPasswordCredential() error = %v", err)
	}
	if !ok {
		t.Fatal("GetPasswordCredential() ok = false, want true")
	}
	if credential.UserID != "018f0000-0000-7000-8000-000000000001" {
		t.Fatalf("UserID = %q, want persisted user id", credential.UserID)
	}
	if credential.PasswordChangedAt == nil || !credential.PasswordChangedAt.Equal(changedAt) {
		t.Fatalf("PasswordChangedAt = %v, want %v", credential.PasswordChangedAt, changedAt)
	}
	if !strings.Contains(query.calls[0].sql, "FROM user_password_credentials") {
		t.Fatalf("SQL = %q, want user_password_credentials select", query.calls[0].sql)
	}
}

func TestPostgresAuthStoreGetsSessionByRefreshTokenHash(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 3, 10, 0, 0, 0, time.UTC)
	query := &recordingProjectContextQuerier{
		row: fakeProjectContextRow{
			values: []any{
				"018f0000-0000-7000-8000-000000000201",
				"018f0000-0000-7000-8000-000000000001",
				"refresh-token-hash",
				string(spine.UserSessionStateActive),
				now,
				now,
				now.Add(24 * time.Hour),
				nil,
				nil,
			},
		},
	}
	store := NewPostgresAuthStoreWithExecutorAndQuerier(&recordingProjectContextExecer{}, query)

	session, ok, err := store.GetSessionByRefreshTokenHash(ctx, "refresh-token-hash")
	if err != nil {
		t.Fatalf("GetSessionByRefreshTokenHash() error = %v", err)
	}
	if !ok {
		t.Fatal("GetSessionByRefreshTokenHash() ok = false, want true")
	}
	if session.ID != "018f0000-0000-7000-8000-000000000201" {
		t.Fatalf("ID = %q, want persisted session id", session.ID)
	}
	if session.State != spine.UserSessionStateActive {
		t.Fatalf("State = %q, want active", session.State)
	}
	if !strings.Contains(query.calls[0].sql, "WHERE refresh_token_hash = $1") {
		t.Fatalf("SQL = %q, want refresh token hash lookup", query.calls[0].sql)
	}
}
