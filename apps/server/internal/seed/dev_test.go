package seed

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/heurema/goalrail/apps/server/internal/spine"
)

func TestRunDevUsesDeterministicProjectContextRecords(t *testing.T) {
	ctx := context.Background()
	store := &recordingSeedStore{}
	now := time.Date(2026, 4, 26, 10, 0, 0, 0, time.UTC)

	if err := RunDev(ctx, store, now); err != nil {
		t.Fatalf("RunDev() error = %v", err)
	}

	if got, want := store.user.ID, DevUserID; got != want {
		t.Fatalf("user ID = %q, want %q", got, want)
	}
	if got, want := store.organization.ID, DevOrganizationID; got != want {
		t.Fatalf("organization ID = %q, want %q", got, want)
	}
	if got, want := store.membership.Role, spine.OrganizationMembershipRoleOwner; got != want {
		t.Fatalf("membership role = %q, want %q", got, want)
	}
	if got, want := store.project.ID, DevProjectID; got != want {
		t.Fatalf("project ID = %q, want %q", got, want)
	}
	if got, want := store.repoBinding.ID, DevRepoBindingID; got != want {
		t.Fatalf("repo binding ID = %q, want %q", got, want)
	}
	for name, id := range map[string]string{
		"user":                    string(DevUserID),
		"organization":            string(DevOrganizationID),
		"organization_membership": string(DevOrganizationMembershipID),
		"project":                 string(DevProjectID),
		"repo_binding":            string(DevRepoBindingID),
	} {
		assertUUIDv7(t, name, id)
	}
	if got, want := store.project.CreatedByUserID, DevUserID; got != want {
		t.Fatalf("project created_by_user_id = %q, want %q", got, want)
	}
	if got, want := store.repoBinding.CreatedByUserID, DevUserID; got != want {
		t.Fatalf("repo binding created_by_user_id = %q, want %q", got, want)
	}
	if got, want := store.repoBinding.AccessMode, spine.RepoBindingAccessModeMetadataOnly; got != want {
		t.Fatalf("repo binding access mode = %q, want %q", got, want)
	}
	if got, want := store.calls, []string{"user", "organization", "membership", "project", "repo_binding"}; !equalStrings(got, want) {
		t.Fatalf("calls = %v, want %v", got, want)
	}
}

func assertUUIDv7(t *testing.T, name string, value string) {
	t.Helper()

	id, err := uuid.Parse(value)
	if err != nil {
		t.Fatalf("%s ID parse UUID: %v", name, err)
	}
	if id.Version() != 7 {
		t.Fatalf("%s ID version = %d, want 7", name, id.Version())
	}
}

type recordingSeedStore struct {
	calls        []string
	user         spine.User
	organization spine.Organization
	membership   spine.OrganizationMembership
	project      spine.Project
	repoBinding  spine.RepoBinding
}

func (s *recordingSeedStore) UpsertUser(_ context.Context, user spine.User) error {
	s.calls = append(s.calls, "user")
	s.user = user
	return nil
}

func (s *recordingSeedStore) UpsertOrganization(_ context.Context, org spine.Organization) error {
	s.calls = append(s.calls, "organization")
	s.organization = org
	return nil
}

func (s *recordingSeedStore) UpsertOrganizationMembership(_ context.Context, membership spine.OrganizationMembership) error {
	s.calls = append(s.calls, "membership")
	s.membership = membership
	return nil
}

func (s *recordingSeedStore) UpsertProject(_ context.Context, project spine.Project) error {
	s.calls = append(s.calls, "project")
	s.project = project
	return nil
}

func (s *recordingSeedStore) UpsertRepoBinding(_ context.Context, binding spine.RepoBinding) error {
	s.calls = append(s.calls, "repo_binding")
	s.repoBinding = binding
	return nil
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
