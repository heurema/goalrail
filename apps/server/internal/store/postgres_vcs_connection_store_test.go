package store

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

func TestPostgresVcsConnectionStoreBuildsPendingSetupCreate(t *testing.T) {
	ctx := context.Background()
	exec := &recordingProjectContextExecer{}
	store := NewPostgresVcsConnectionStoreWithExecutorAndQuerier(exec, nil)
	now := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)

	err := store.CreatePendingSetup(ctx, spine.VcsConnection{
		ID:                  "018f0000-0000-7000-8000-000000000010",
		InstallationID:      "018f0000-0000-7000-8000-000000000006",
		OrganizationID:      "018f0000-0000-7000-8000-000000000002",
		CreatedByUserID:     "018f0000-0000-7000-8000-000000000001",
		ProviderKind:        "gitlab",
		ProviderInstanceURL: "https://gitlab.example.com",
		State:               spine.VcsConnectionStatePendingSetup,
		SetupExpiresAt:      now.Add(30 * time.Minute),
		CreatedAt:           now,
		UpdatedAt:           now,
	})
	if err != nil {
		t.Fatalf("CreatePendingSetup() error = %v", err)
	}
	if len(exec.calls) != 1 {
		t.Fatalf("Exec calls = %d, want 1", len(exec.calls))
	}
	call := exec.calls[0]
	if !strings.Contains(call.sql, "INSERT INTO vcs_connections") {
		t.Fatalf("SQL = %q, want vcs_connections insert", call.sql)
	}
	for _, want := range []string{"provider_kind", "provider_instance_url", "setup_expires_at"} {
		if !strings.Contains(call.sql, want) {
			t.Fatalf("SQL = %q, want %s column", call.sql, want)
		}
	}
	if strings.Contains(call.sql, "token") || strings.Contains(call.sql, "secret") || strings.Contains(call.sql, "credential") {
		t.Fatalf("SQL = %q, must not include credential columns", call.sql)
	}
	if got, want := len(call.args), 10; got != want {
		t.Fatalf("args len = %d, want %d", got, want)
	}
}

func TestPostgresVcsConnectionStoreGetsOrganizationInstallation(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	query := &recordingProjectContextQuerier{
		row: fakeProjectContextRow{
			values: []any{
				"018f0000-0000-7000-8000-000000000002",
				"018f0000-0000-7000-8000-000000000006",
				"default",
				"Default",
				"active",
				now,
				now,
			},
		},
	}
	store := NewPostgresVcsConnectionStoreWithExecutorAndQuerier(&recordingProjectContextExecer{}, query)

	organization, ok, err := store.GetOrganization(ctx, "018f0000-0000-7000-8000-000000000002")
	if err != nil {
		t.Fatalf("GetOrganization() error = %v", err)
	}
	if !ok {
		t.Fatal("GetOrganization() ok = false, want true")
	}
	if organization.InstallationID != "018f0000-0000-7000-8000-000000000006" {
		t.Fatalf("installation_id = %q, want persisted installation", organization.InstallationID)
	}
	if !strings.Contains(query.calls[0].sql, "FROM organizations") {
		t.Fatalf("SQL = %q, want organizations select", query.calls[0].sql)
	}
}

func TestPostgresVcsConnectionStoreGetsVcsConnection(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	query := &recordingProjectContextQuerier{
		row: fakeProjectContextRow{
			values: []any{
				"018f0000-0000-7000-8000-000000000010",
				"018f0000-0000-7000-8000-000000000006",
				"018f0000-0000-7000-8000-000000000002",
				"018f0000-0000-7000-8000-000000000001",
				"gitlab",
				"https://gitlab.example.com",
				"pending_setup",
				now.Add(30 * time.Minute),
				now,
				now,
			},
		},
	}
	store := NewPostgresVcsConnectionStoreWithExecutorAndQuerier(&recordingProjectContextExecer{}, query)

	connection, ok, err := store.GetVcsConnection(ctx, "018f0000-0000-7000-8000-000000000010")
	if err != nil {
		t.Fatalf("GetVcsConnection() error = %v", err)
	}
	if !ok {
		t.Fatal("GetVcsConnection() ok = false, want true")
	}
	if connection.State != spine.VcsConnectionStatePendingSetup {
		t.Fatalf("state = %q, want pending_setup", connection.State)
	}
	if connection.ProviderInstanceURL != "https://gitlab.example.com" {
		t.Fatalf("provider_instance_url = %q, want persisted URL", connection.ProviderInstanceURL)
	}
	if !strings.Contains(query.calls[0].sql, "FROM vcs_connections") {
		t.Fatalf("SQL = %q, want vcs_connections select", query.calls[0].sql)
	}
}
