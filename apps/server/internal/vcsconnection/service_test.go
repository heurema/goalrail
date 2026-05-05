package vcsconnection

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

func TestCreatePendingSetupSucceedsForOwnerAndAdmin(t *testing.T) {
	for _, role := range []spine.OrganizationMembershipRole{
		spine.OrganizationMembershipRoleOwner,
		spine.OrganizationMembershipRoleAdmin,
	} {
		t.Run(string(role), func(t *testing.T) {
			store := newFakeStore()
			events := &fakeEventLog{}
			tx := &fakeTxRunner{}
			service := NewService(store, events, tx, fixedClock{now: testNow()}, sequenceIDs{})

			connection, err := service.CreatePendingSetup(context.Background(), validCreateInput(role))
			if err != nil {
				t.Fatalf("CreatePendingSetup() error = %v", err)
			}
			if connection.State != spine.VcsConnectionStatePendingSetup {
				t.Fatalf("state = %q, want pending_setup", connection.State)
			}
			if connection.OrganizationID != testOrganizationID {
				t.Fatalf("organization_id = %q, want server-side membership organization", connection.OrganizationID)
			}
			if connection.InstallationID != testInstallationID {
				t.Fatalf("installation_id = %q, want organization installation", connection.InstallationID)
			}
			if connection.CreatedByUserID != testUserID {
				t.Fatalf("created_by_user_id = %q, want authenticated user", connection.CreatedByUserID)
			}
			if !connection.SetupExpiresAt.Equal(testNow().Add(SetupTTL)) {
				t.Fatalf("setup_expires_at = %s, want now + setup TTL", connection.SetupExpiresAt)
			}
			if len(store.created) != 1 {
				t.Fatalf("created connections = %d, want 1", len(store.created))
			}
			if len(events.events) != 1 {
				t.Fatalf("events = %d, want 1", len(events.events))
			}
			if events.events[0].Type != EventTypeSetupStarted {
				t.Fatalf("event type = %q, want %q", events.events[0].Type, EventTypeSetupStarted)
			}
			var payload map[string]any
			if err := json.Unmarshal(events.events[0].Payload, &payload); err != nil {
				t.Fatalf("event payload JSON = %v", err)
			}
			if payload["provider_kind"] != "gitlab" || payload["state"] != "pending_setup" {
				t.Fatalf("event payload = %#v, want non-secret setup metadata", payload)
			}
			if tx.calls != 1 {
				t.Fatalf("transactions = %d, want 1", tx.calls)
			}
			if store.repoBindingMutations != 0 {
				t.Fatalf("repo binding mutations = %d, want 0", store.repoBindingMutations)
			}
		})
	}
}

func TestCreatePendingSetupRejectsViewerOrInactiveMembership(t *testing.T) {
	for _, tt := range []struct {
		name  string
		input CreateInput
	}{
		{name: "viewer", input: validCreateInput(spine.OrganizationMembershipRoleViewer)},
		{name: "member", input: validCreateInput(spine.OrganizationMembershipRoleMember)},
		{name: "inactive", input: inactiveMembershipInput()},
	} {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(newFakeStore(), &fakeEventLog{}, &fakeTxRunner{}, fixedClock{now: testNow()}, sequenceIDs{})
			_, err := service.CreatePendingSetup(context.Background(), tt.input)
			if !errors.Is(err, ErrForbidden) {
				t.Fatalf("CreatePendingSetup() error = %v, want ErrForbidden", err)
			}
		})
	}
}

func TestCreatePendingSetupRejectsMissingOrInactiveOrganization(t *testing.T) {
	for _, tt := range []struct {
		name string
		org  spine.Organization
		ok   bool
	}{
		{name: "missing organization", ok: false},
		{name: "inactive organization", ok: true, org: testOrganization(spine.EntityState("disabled"))},
	} {
		t.Run(tt.name, func(t *testing.T) {
			store := newFakeStore()
			store.organization = tt.org
			store.organizationOK = tt.ok
			service := NewService(store, &fakeEventLog{}, &fakeTxRunner{}, fixedClock{now: testNow()}, sequenceIDs{})

			_, err := service.CreatePendingSetup(context.Background(), validCreateInput(spine.OrganizationMembershipRoleOwner))
			if !errors.Is(err, ErrForbidden) {
				t.Fatalf("CreatePendingSetup() error = %v, want ErrForbidden", err)
			}
		})
	}
}

func TestCreatePendingSetupNormalizesProviderInstanceURL(t *testing.T) {
	store := newFakeStore()
	service := NewService(store, &fakeEventLog{}, &fakeTxRunner{}, fixedClock{now: testNow()}, sequenceIDs{})
	input := validCreateInput(spine.OrganizationMembershipRoleOwner)
	input.ProviderKind = "GitLab"
	input.ProviderInstanceURL = "HTTPS://GITLAB.EXAMPLE.COM///"

	connection, err := service.CreatePendingSetup(context.Background(), input)
	if err != nil {
		t.Fatalf("CreatePendingSetup() error = %v", err)
	}
	if connection.ProviderKind != "gitlab" {
		t.Fatalf("provider_kind = %q, want gitlab", connection.ProviderKind)
	}
	if connection.ProviderInstanceURL != "https://gitlab.example.com" {
		t.Fatalf("provider_instance_url = %q, want normalized URL", connection.ProviderInstanceURL)
	}
}

func TestCreatePendingSetupAllowsLocalhostHTTPProviderInstanceURL(t *testing.T) {
	service := NewService(newFakeStore(), &fakeEventLog{}, &fakeTxRunner{}, fixedClock{now: testNow()}, sequenceIDs{})
	input := validCreateInput(spine.OrganizationMembershipRoleOwner)
	input.ProviderInstanceURL = "http://localhost:8080/gitlab/"

	connection, err := service.CreatePendingSetup(context.Background(), input)
	if err != nil {
		t.Fatalf("CreatePendingSetup() error = %v", err)
	}
	if connection.ProviderInstanceURL != "http://localhost:8080/gitlab" {
		t.Fatalf("provider_instance_url = %q, want localhost http normalized", connection.ProviderInstanceURL)
	}
}

func TestCreatePendingSetupRejectsInvalidProviderKind(t *testing.T) {
	service := NewService(newFakeStore(), &fakeEventLog{}, &fakeTxRunner{}, fixedClock{now: testNow()}, sequenceIDs{})
	input := validCreateInput(spine.OrganizationMembershipRoleOwner)
	input.ProviderKind = "gitlab.com"

	_, err := service.CreatePendingSetup(context.Background(), input)
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("CreatePendingSetup() error = %v, want validation error", err)
	}
	if validationErr.Field != "provider_kind" {
		t.Fatalf("validation field = %q, want provider_kind", validationErr.Field)
	}
}

func TestCreatePendingSetupRejectsInvalidProviderInstanceURL(t *testing.T) {
	for _, raw := range []string{
		"gitlab.example.com",
		"ftp://gitlab.example.com",
		"http://gitlab.example.com",
		"https://token@gitlab.example.com",
		"https://gitlab.example.com?token=secret",
	} {
		t.Run(raw, func(t *testing.T) {
			service := NewService(newFakeStore(), &fakeEventLog{}, &fakeTxRunner{}, fixedClock{now: testNow()}, sequenceIDs{})
			input := validCreateInput(spine.OrganizationMembershipRoleOwner)
			input.ProviderInstanceURL = raw

			_, err := service.CreatePendingSetup(context.Background(), input)
			var validationErr *ValidationError
			if !errors.As(err, &validationErr) {
				t.Fatalf("CreatePendingSetup() error = %v, want validation error", err)
			}
			if validationErr.Field != "provider_instance_url" {
				t.Fatalf("validation field = %q, want provider_instance_url", validationErr.Field)
			}
		})
	}
}

func TestGetEnforcesOrganizationBoundary(t *testing.T) {
	store := newFakeStore()
	connection := testConnection()
	connection.OrganizationID = "018f0000-0000-7000-8000-000000000099"
	store.connections[connection.ID] = connection
	service := NewService(store, &fakeEventLog{}, &fakeTxRunner{}, fixedClock{now: testNow()}, sequenceIDs{})

	_, err := service.Get(context.Background(), GetInput{
		VcsConnectionID: connection.ID,
		Membership:      activeMembership(spine.OrganizationMembershipRoleViewer),
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get() error = %v, want ErrNotFound", err)
	}
}

func TestGetReturnsConnectionForSameOrganizationMembership(t *testing.T) {
	store := newFakeStore()
	connection := testConnection()
	store.connections[connection.ID] = connection
	service := NewService(store, &fakeEventLog{}, &fakeTxRunner{}, fixedClock{now: testNow()}, sequenceIDs{})

	result, err := service.Get(context.Background(), GetInput{
		VcsConnectionID: connection.ID,
		Membership:      activeMembership(spine.OrganizationMembershipRoleViewer),
	})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if result.ID != connection.ID {
		t.Fatalf("id = %q, want %q", result.ID, connection.ID)
	}
}

const (
	testUserID         spine.UserID          = "018f0000-0000-7000-8000-000000000001"
	testInstallationID spine.InstallationID  = "018f0000-0000-7000-8000-000000000006"
	testOrganizationID spine.OrganizationID  = "018f0000-0000-7000-8000-000000000002"
	testConnectionID   spine.VcsConnectionID = "018f0000-0000-7000-8000-000000000010"
	testEventID        spine.EventID         = "018f0000-0000-7000-8000-000000000011"
)

func validCreateInput(role spine.OrganizationMembershipRole) CreateInput {
	return CreateInput{
		AuthenticatedUserID: testUserID,
		Membership:          activeMembership(role),
		ProviderKind:        "gitlab",
		ProviderInstanceURL: "https://gitlab.example.com/",
	}
}

func inactiveMembershipInput() CreateInput {
	input := validCreateInput(spine.OrganizationMembershipRoleOwner)
	input.Membership.State = spine.EntityState("disabled")
	return input
}

func activeMembership(role spine.OrganizationMembershipRole) spine.OrganizationMembership {
	return spine.OrganizationMembership{
		ID:             "018f0000-0000-7000-8000-000000000005",
		OrganizationID: testOrganizationID,
		UserID:         testUserID,
		Role:           role,
		State:          spine.EntityStateActive,
	}
}

func testOrganization(state spine.EntityState) spine.Organization {
	return spine.Organization{
		ID:             testOrganizationID,
		InstallationID: testInstallationID,
		Slug:           "default",
		DisplayName:    "Default",
		State:          state,
		CreatedAt:      testNow(),
		UpdatedAt:      testNow(),
	}
}

func testConnection() spine.VcsConnection {
	return spine.VcsConnection{
		ID:                  testConnectionID,
		InstallationID:      testInstallationID,
		OrganizationID:      testOrganizationID,
		CreatedByUserID:     testUserID,
		ProviderKind:        "gitlab",
		ProviderInstanceURL: "https://gitlab.example.com",
		State:               spine.VcsConnectionStatePendingSetup,
		SetupExpiresAt:      testNow().Add(SetupTTL),
		CreatedAt:           testNow(),
		UpdatedAt:           testNow(),
	}
}

func testNow() time.Time {
	return time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
}

type fakeStore struct {
	organization         spine.Organization
	organizationOK       bool
	created              []spine.VcsConnection
	connections          map[spine.VcsConnectionID]spine.VcsConnection
	repoBindingMutations int
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		organization:   testOrganization(spine.EntityStateActive),
		organizationOK: true,
		connections:    map[spine.VcsConnectionID]spine.VcsConnection{},
	}
}

func (s *fakeStore) GetOrganization(context.Context, spine.OrganizationID) (spine.Organization, bool, error) {
	return s.organization, s.organizationOK, nil
}

func (s *fakeStore) CreatePendingSetup(_ context.Context, connection spine.VcsConnection) error {
	s.created = append(s.created, connection)
	s.connections[connection.ID] = connection
	return nil
}

func (s *fakeStore) GetVcsConnection(_ context.Context, id spine.VcsConnectionID) (spine.VcsConnection, bool, error) {
	connection, ok := s.connections[id]
	return connection, ok, nil
}

type fakeEventLog struct {
	events []spine.Event
}

func (l *fakeEventLog) Append(_ context.Context, event spine.Event) error {
	l.events = append(l.events, event)
	return nil
}

type fakeTxRunner struct {
	calls int
}

func (r *fakeTxRunner) RunReadCommitted(ctx context.Context, fn func(context.Context) error) error {
	r.calls++
	return fn(ctx)
}

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

type sequenceIDs struct{}

func (sequenceIDs) NewVcsConnectionID() (spine.VcsConnectionID, error) {
	return testConnectionID, nil
}

func (sequenceIDs) NewEventID() (spine.EventID, error) {
	return testEventID, nil
}
