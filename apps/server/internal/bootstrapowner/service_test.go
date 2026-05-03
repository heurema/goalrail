package bootstrapowner

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

func TestBootstrapOwnerCreatesSelfHostedOwnerAndCredential(t *testing.T) {
	ctx := context.Background()
	store := newMemoryStore()
	service := testService(store)

	result, err := service.BootstrapOwner(ctx, Input{
		Email:            " Owner@Example.COM ",
		DisplayName:      " Owner User ",
		OrganizationSlug: "Acme-Team",
		OrganizationName: " Acme Team ",
		PublicBaseURL:    "https://goalrail.example.com/",
	})
	if err != nil {
		t.Fatalf("BootstrapOwner() error = %v", err)
	}

	if result.Installation.Mode != spine.InstallationModeSelfHosted {
		t.Fatalf("installation mode = %q, want self_hosted", result.Installation.Mode)
	}
	if result.Installation.PublicBaseURL != "https://goalrail.example.com" {
		t.Fatalf("public base URL = %q, want normalized URL", result.Installation.PublicBaseURL)
	}
	if result.Organization.InstallationID != result.Installation.ID {
		t.Fatalf("organization installation id = %q, want %q", result.Organization.InstallationID, result.Installation.ID)
	}
	if result.Organization.Slug != "acme-team" {
		t.Fatalf("organization slug = %q, want normalized slug", result.Organization.Slug)
	}
	if result.User.Email != "owner@example.com" {
		t.Fatalf("user email = %q, want normalized email", result.User.Email)
	}
	if result.Membership.Role != spine.OrganizationMembershipRoleOwner {
		t.Fatalf("membership role = %q, want owner", result.Membership.Role)
	}
	if !result.PasswordCredentialCreated {
		t.Fatal("PasswordCredentialCreated = false, want true")
	}
	if result.TemporaryPassword != "temporary-password" {
		t.Fatalf("TemporaryPassword = %q, want generated password", result.TemporaryPassword)
	}
	credential := store.credentials[result.User.ID]
	if credential.PasswordHash != "hash:temporary-password" {
		t.Fatalf("PasswordHash = %q, want hash of temporary password", credential.PasswordHash)
	}
	if !credential.MustChangePassword {
		t.Fatal("MustChangePassword = false, want true")
	}
	if credential.PasswordChangedAt != nil {
		t.Fatalf("PasswordChangedAt = %v, want nil for temporary password", credential.PasswordChangedAt)
	}
}

func TestBootstrapOwnerReusesExistingCredentialWithoutRotating(t *testing.T) {
	ctx := context.Background()
	store := newMemoryStore()
	seedExistingOwner(store, spine.OrganizationMembershipRoleOwner, true)
	passwords := &recordingPasswordGenerator{err: errors.New("should not generate")}
	hasher := &recordingHasher{err: errors.New("should not hash")}
	service := testService(store)
	service.Passwords = passwords
	service.Hasher = hasher

	result, err := service.BootstrapOwner(ctx, Input{
		Email:            "owner@example.com",
		DisplayName:      "Owner User",
		OrganizationSlug: "primary",
		OrganizationName: "Primary Org",
		PublicBaseURL:    "https://goalrail.example.com",
	})
	if err != nil {
		t.Fatalf("BootstrapOwner() error = %v", err)
	}

	if result.PasswordCredentialCreated {
		t.Fatal("PasswordCredentialCreated = true, want false for existing credential")
	}
	if result.TemporaryPassword != "" {
		t.Fatalf("TemporaryPassword = %q, want empty on existing credential", result.TemporaryPassword)
	}
	if got := store.credentials["user-1"].PasswordHash; got != "existing-hash" {
		t.Fatalf("PasswordHash = %q, want existing hash preserved", got)
	}
	if passwords.calls != 0 {
		t.Fatalf("password generator calls = %d, want 0", passwords.calls)
	}
	if hasher.calls != 0 {
		t.Fatalf("hasher calls = %d, want 0", hasher.calls)
	}
	if got := store.memberships["org-1:user-1"].Role; got != spine.OrganizationMembershipRoleOwner {
		t.Fatalf("membership role = %q, want upgraded owner", got)
	}
	if got := store.installations["installation-1"].PublicBaseURL; got != "https://goalrail.example.com" {
		t.Fatalf("public base URL = %q, want provided URL", got)
	}
}

func TestBootstrapOwnerRejectsDifferentEmailWhenOwnerCredentialExists(t *testing.T) {
	ctx := context.Background()
	store := newMemoryStore()
	seedExistingOwner(store, spine.OrganizationMembershipRoleOwner, true)
	service := testService(store)

	_, err := service.BootstrapOwner(ctx, Input{
		Email:            "other@example.com",
		DisplayName:      "Other Owner",
		OrganizationSlug: "primary",
		OrganizationName: "Primary Org",
		PublicBaseURL:    "https://goalrail.example.com",
	})
	if !errors.Is(err, ErrBootstrappedOwnerExists) {
		t.Fatalf("BootstrapOwner() error = %v, want ErrBootstrappedOwnerExists", err)
	}
	if _, exists := store.users["user-1"]; !exists {
		t.Fatal("existing owner user missing")
	}
	if _, exists := store.users["user-2"]; exists {
		t.Fatal("created a second user before rejecting bootstrapped owner conflict")
	}
	if len(store.memberships) != 1 {
		t.Fatalf("memberships = %d, want existing membership only", len(store.memberships))
	}
	if got := store.credentials["user-1"].PasswordHash; got != "existing-hash" {
		t.Fatalf("PasswordHash = %q, want existing hash preserved", got)
	}
}

func TestBootstrapOwnerCreatesCredentialForSameOwnerWithoutCredential(t *testing.T) {
	ctx := context.Background()
	store := newMemoryStore()
	seedExistingOwner(store, spine.OrganizationMembershipRoleOwner, false)
	service := testService(store)

	result, err := service.BootstrapOwner(ctx, Input{
		Email:            "owner@example.com",
		DisplayName:      "Owner User",
		OrganizationSlug: "primary",
		OrganizationName: "Primary Org",
		PublicBaseURL:    "https://goalrail.example.com",
	})
	if err != nil {
		t.Fatalf("BootstrapOwner() error = %v", err)
	}
	if result.User.ID != "user-1" {
		t.Fatalf("User ID = %q, want existing owner user", result.User.ID)
	}
	if !result.PasswordCredentialCreated {
		t.Fatal("PasswordCredentialCreated = false, want true")
	}
	if result.TemporaryPassword != "temporary-password" {
		t.Fatalf("TemporaryPassword = %q, want generated password", result.TemporaryPassword)
	}
	if got := store.credentials["user-1"].PasswordHash; got != "hash:temporary-password" {
		t.Fatalf("PasswordHash = %q, want new credential hash", got)
	}
}

func TestBootstrapOwnerRejectsHTTPForNonLocalhost(t *testing.T) {
	_, err := NormalizeInput(Input{
		Email:            "owner@example.com",
		DisplayName:      "Owner",
		OrganizationSlug: "primary",
		OrganizationName: "Primary",
		PublicBaseURL:    "http://goalrail.example.com",
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("NormalizeInput() error = %v, want ErrInvalidInput", err)
	}
}

func TestBootstrapOwnerAllowsLocalhostHTTPAndPath(t *testing.T) {
	input, err := NormalizeInput(Input{
		Email:            "owner@example.com",
		DisplayName:      "Owner",
		OrganizationSlug: "primary",
		OrganizationName: "Primary",
		PublicBaseURL:    "http://localhost:8080/goalrail/",
	})
	if err != nil {
		t.Fatalf("NormalizeInput() error = %v", err)
	}
	if input.PublicBaseURL != "http://localhost:8080/goalrail" {
		t.Fatalf("PublicBaseURL = %q, want normalized localhost URL with path", input.PublicBaseURL)
	}
}

func testService(store *memoryStore) *Service {
	return &Service{
		Store:     store,
		IDs:       &sequenceIDs{},
		Passwords: &recordingPasswordGenerator{password: "temporary-password"},
		Hasher:    &recordingHasher{},
		Clock:     fixedClock{now: testNow()},
	}
}

func testNow() time.Time {
	return time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
}

func seedExistingOwner(store *memoryStore, role spine.OrganizationMembershipRole, withCredential bool) {
	now := testNow()
	store.installations["installation-1"] = spine.Installation{
		ID:            "installation-1",
		Mode:          spine.InstallationModeSelfHosted,
		PublicBaseURL: "http://localhost:8080",
		State:         spine.EntityStateActive,
		CreatedAt:     now.Add(-time.Hour),
		UpdatedAt:     now.Add(-time.Hour),
	}
	store.organizations["org-1"] = spine.Organization{
		ID:             "org-1",
		InstallationID: "installation-1",
		Slug:           "old-slug",
		DisplayName:    "Old Name",
		State:          spine.EntityStateActive,
		CreatedAt:      now.Add(-time.Hour),
		UpdatedAt:      now.Add(-time.Hour),
	}
	store.users["user-1"] = spine.User{
		ID:          "user-1",
		DisplayName: "Existing Owner",
		Email:       "owner@example.com",
		State:       spine.EntityStateActive,
		CreatedAt:   now.Add(-time.Hour),
		UpdatedAt:   now.Add(-time.Hour),
	}
	store.memberships["org-1:user-1"] = spine.OrganizationMembership{
		ID:             "membership-1",
		OrganizationID: "org-1",
		UserID:         "user-1",
		Role:           role,
		State:          spine.EntityStateActive,
		CreatedAt:      now.Add(-time.Hour),
		UpdatedAt:      now.Add(-time.Hour),
	}
	if withCredential {
		store.credentials["user-1"] = spine.UserPasswordCredential{
			UserID:             "user-1",
			PasswordHash:       "existing-hash",
			MustChangePassword: true,
			CreatedAt:          now.Add(-time.Hour),
			UpdatedAt:          now.Add(-time.Hour),
		}
	}
}

type memoryStore struct {
	installations map[spine.InstallationID]spine.Installation
	organizations map[spine.OrganizationID]spine.Organization
	users         map[spine.UserID]spine.User
	memberships   map[string]spine.OrganizationMembership
	credentials   map[spine.UserID]spine.UserPasswordCredential
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		installations: make(map[spine.InstallationID]spine.Installation),
		organizations: make(map[spine.OrganizationID]spine.Organization),
		users:         make(map[spine.UserID]spine.User),
		memberships:   make(map[string]spine.OrganizationMembership),
		credentials:   make(map[spine.UserID]spine.UserPasswordCredential),
	}
}

func (s *memoryStore) GetSelfHostedInstallation(context.Context) (spine.Installation, bool, error) {
	for _, installation := range s.installations {
		if installation.Mode == spine.InstallationModeSelfHosted {
			return installation, true, nil
		}
	}
	return spine.Installation{}, false, nil
}

func (s *memoryStore) UpsertInstallation(_ context.Context, installation spine.Installation) error {
	s.installations[installation.ID] = installation
	return nil
}

func (s *memoryStore) GetPrimaryOrganization(_ context.Context, installationID spine.InstallationID) (spine.Organization, bool, error) {
	for _, org := range s.organizations {
		if org.InstallationID == installationID {
			return org, true, nil
		}
	}
	return spine.Organization{}, false, nil
}

func (s *memoryStore) UpsertOrganization(_ context.Context, org spine.Organization) error {
	s.organizations[org.ID] = org
	return nil
}

func (s *memoryStore) GetBootstrappedOwner(_ context.Context, organizationID spine.OrganizationID) (spine.User, spine.UserPasswordCredential, bool, error) {
	for _, membership := range s.memberships {
		if membership.OrganizationID != organizationID ||
			membership.Role != spine.OrganizationMembershipRoleOwner ||
			membership.State != spine.EntityStateActive {
			continue
		}
		user, ok := s.users[membership.UserID]
		if !ok || user.State != spine.EntityStateActive {
			continue
		}
		credential, ok := s.credentials[membership.UserID]
		if !ok {
			continue
		}
		return user, credential, true, nil
	}
	return spine.User{}, spine.UserPasswordCredential{}, false, nil
}

func (s *memoryStore) GetUserByEmail(_ context.Context, email string) (spine.User, bool, error) {
	for _, user := range s.users {
		if user.Email == email {
			return user, true, nil
		}
	}
	return spine.User{}, false, nil
}

func (s *memoryStore) UpsertUser(_ context.Context, user spine.User) error {
	s.users[user.ID] = user
	return nil
}

func (s *memoryStore) GetOrganizationMembership(_ context.Context, organizationID spine.OrganizationID, userID spine.UserID) (spine.OrganizationMembership, bool, error) {
	membership, ok := s.memberships[membershipKey(organizationID, userID)]
	return membership, ok, nil
}

func (s *memoryStore) UpsertOrganizationMembership(_ context.Context, membership spine.OrganizationMembership) error {
	s.memberships[membershipKey(membership.OrganizationID, membership.UserID)] = membership
	return nil
}

func (s *memoryStore) GetPasswordCredential(_ context.Context, userID spine.UserID) (spine.UserPasswordCredential, bool, error) {
	credential, ok := s.credentials[userID]
	return credential, ok, nil
}

func (s *memoryStore) CreatePasswordCredential(_ context.Context, credential spine.UserPasswordCredential) error {
	if _, exists := s.credentials[credential.UserID]; exists {
		return ErrExistingPasswordCredential
	}
	s.credentials[credential.UserID] = credential
	return nil
}

func membershipKey(organizationID spine.OrganizationID, userID spine.UserID) string {
	return string(organizationID) + ":" + string(userID)
}

type sequenceIDs struct {
	installation int
	organization int
	user         int
	membership   int
}

func (g *sequenceIDs) NewInstallationID() (spine.InstallationID, error) {
	g.installation++
	return spine.InstallationID("installation-1"), nil
}

func (g *sequenceIDs) NewOrganizationID() (spine.OrganizationID, error) {
	g.organization++
	return spine.OrganizationID("org-1"), nil
}

func (g *sequenceIDs) NewUserID() (spine.UserID, error) {
	g.user++
	return spine.UserID(fmt.Sprintf("user-%d", g.user)), nil
}

func (g *sequenceIDs) NewOrganizationMembershipID() (spine.OrganizationMembershipID, error) {
	g.membership++
	return spine.OrganizationMembershipID("membership-1"), nil
}

type recordingPasswordGenerator struct {
	password string
	err      error
	calls    int
}

func (g *recordingPasswordGenerator) NewPassword() (string, error) {
	g.calls++
	if g.err != nil {
		return "", g.err
	}
	return g.password, nil
}

type recordingHasher struct {
	err   error
	calls int
}

func (h *recordingHasher) HashPassword(value string) (string, error) {
	h.calls++
	if h.err != nil {
		return "", h.err
	}
	return "hash:" + value, nil
}

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}
