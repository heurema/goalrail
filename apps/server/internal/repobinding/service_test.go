package repobinding

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

func TestInitCreatesRepoBindingForExistingProject(t *testing.T) {
	store := newFakeStore()
	events := &fakeEventLog{}
	service := NewService(store, events, &fakeTxRunner{}, fixedClock{now: testNow()}, sequenceIDs{})

	result, err := service.Init(context.Background(), validInput())
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	if !result.Created {
		t.Fatal("Created = false, want true")
	}
	if result.RepoBindingID != "018f0000-0000-7000-8000-000000000004" {
		t.Fatalf("repo_binding_id = %q, want generated id", result.RepoBindingID)
	}
	created := store.created[0]
	if created.OrganizationID != testOrganizationID {
		t.Fatalf("organization_id = %q, want project organization", created.OrganizationID)
	}
	if created.ProjectID != testProjectID {
		t.Fatalf("project_id = %q, want project id", created.ProjectID)
	}
	if created.CreatedByUserID != testUserID {
		t.Fatalf("created_by_user_id = %q, want authenticated user", created.CreatedByUserID)
	}
	if created.AccessMode != spine.RepoBindingAccessModeMetadataOnly {
		t.Fatalf("access_mode = %q, want metadata_only", created.AccessMode)
	}
	if created.WorkflowBaseBranch != "main" {
		t.Fatalf("workflow_base_branch = %q, want main", created.WorkflowBaseBranch)
	}
	if len(events.events) != 1 {
		t.Fatalf("events = %d, want 1", len(events.events))
	}
	if events.events[0].Type != EventTypeInitialized {
		t.Fatalf("event type = %q, want %q", events.events[0].Type, EventTypeInitialized)
	}
	var payload map[string]any
	if err := json.Unmarshal(events.events[0].Payload, &payload); err != nil {
		t.Fatalf("event payload JSON = %v", err)
	}
	if payload["local_remote_name"] != "origin" || payload["local_head_sha"] != "abc123" {
		t.Fatalf("event payload = %#v, want local metadata", payload)
	}
}

func TestInitReturnsExistingRepoBindingIdempotently(t *testing.T) {
	store := newFakeStore()
	existing := existingBinding("github", "Heurema/Goalrail")
	store.activeBinding = &existing
	events := &fakeEventLog{}
	service := NewService(store, events, &fakeTxRunner{}, fixedClock{now: testNow()}, sequenceIDs{})

	input := validInput()
	input.RepositoryFullName = "heurema/goalrail"
	result, err := service.Init(context.Background(), input)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	if result.Created {
		t.Fatal("Created = true, want false for existing binding")
	}
	if result.RepoBindingID != existing.ID {
		t.Fatalf("repo_binding_id = %q, want existing %q", result.RepoBindingID, existing.ID)
	}
	if len(store.created) != 0 {
		t.Fatalf("created bindings = %d, want 0", len(store.created))
	}
	if len(events.events) != 0 {
		t.Fatalf("events = %d, want 0 for idempotent read", len(events.events))
	}
}

func TestInitRejectsDifferentRepositoryForProject(t *testing.T) {
	store := newFakeStore()
	existing := existingBinding("github", "acme/other")
	store.activeBinding = &existing
	service := NewService(store, &fakeEventLog{}, &fakeTxRunner{}, fixedClock{now: testNow()}, sequenceIDs{})

	_, err := service.Init(context.Background(), validInput())
	if !errors.Is(err, ErrDifferentRepoBinding) {
		t.Fatalf("Init() error = %v, want ErrDifferentRepoBinding", err)
	}
}

func TestInitRejectsSameRepositoryAlreadyBoundInOrganization(t *testing.T) {
	store := newFakeStore()
	existing := existingBinding("github", "heurema/goalrail")
	existing.ProjectID = "018f0000-0000-7000-8000-000000000099"
	store.organizationBinding = &existing
	service := NewService(store, &fakeEventLog{}, &fakeTxRunner{}, fixedClock{now: testNow()}, sequenceIDs{})

	_, err := service.Init(context.Background(), validInput())
	if !errors.Is(err, ErrRepositoryAlreadyBound) {
		t.Fatalf("Init() error = %v, want ErrRepositoryAlreadyBound", err)
	}
}

func TestInitAllowsDifferentRepositoryWhenOrganizationHasAnotherRepository(t *testing.T) {
	store := newFakeStore()
	existing := existingBinding("github", "acme/frontend")
	existing.ProjectID = "018f0000-0000-7000-8000-000000000099"
	store.organizationBinding = &existing
	service := NewService(store, &fakeEventLog{}, &fakeTxRunner{}, fixedClock{now: testNow()}, sequenceIDs{})

	input := validInput()
	input.RepositoryFullName = "acme/backend"
	input.RepositoryURL = "git@github.com:acme/backend.git"
	result, err := service.Init(context.Background(), input)
	if err != nil {
		t.Fatalf("Init() error = %v, want nil", err)
	}

	if !result.Created {
		t.Fatal("Created = false, want true for different repository")
	}
	if got := len(store.created); got != 1 {
		t.Fatalf("created bindings = %d, want 1", got)
	}
	if store.created[0].RepositoryFullName != "acme/backend" {
		t.Fatalf("created repository_full_name = %q, want acme/backend", store.created[0].RepositoryFullName)
	}
}

func TestInitResolvesConcurrentSameProjectRepositoryCreateIdempotently(t *testing.T) {
	store := newFakeStore()
	existing := existingBinding("github", "heurema/goalrail")
	store.activeBindingAfterCreateError = &existing
	store.createErr = fakeUniqueConstraintError{constraint: "repo_bindings_one_active_per_project_idx"}
	events := &fakeEventLog{}
	service := NewService(store, events, &fakeTxRunner{}, fixedClock{now: testNow()}, sequenceIDs{})

	result, err := service.Init(context.Background(), validInput())
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	if result.Created {
		t.Fatal("Created = true, want false after concurrent create resolves to existing binding")
	}
	if result.RepoBindingID != existing.ID {
		t.Fatalf("repo_binding_id = %q, want existing %q", result.RepoBindingID, existing.ID)
	}
	if got := len(events.events); got != 0 {
		t.Fatalf("events = %d, want 0 when create loses race", got)
	}
}

func TestInitTranslatesConcurrentOrganizationRepositoryCreateConflict(t *testing.T) {
	store := newFakeStore()
	existing := existingBinding("github", "heurema/goalrail")
	existing.ProjectID = "018f0000-0000-7000-8000-000000000099"
	store.organizationBindingAfterCreateError = &existing
	store.createErr = fakeUniqueConstraintError{constraint: "repo_bindings_one_active_per_org_repository_idx"}
	service := NewService(store, &fakeEventLog{}, &fakeTxRunner{}, fixedClock{now: testNow()}, sequenceIDs{})

	_, err := service.Init(context.Background(), validInput())
	if !errors.Is(err, ErrRepositoryAlreadyBound) {
		t.Fatalf("Init() error = %v, want ErrRepositoryAlreadyBound", err)
	}
}

func TestInitRequiresWorkflowOrProviderDefaultBranch(t *testing.T) {
	service := NewService(newFakeStore(), &fakeEventLog{}, &fakeTxRunner{}, fixedClock{now: testNow()}, sequenceIDs{})
	input := validInput()
	input.WorkflowBaseBranch = ""
	input.ProviderDefaultBranch = ""

	_, err := service.Init(context.Background(), input)
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("Init() error = %v, want validation error", err)
	}
	if validationErr.Field != "workflow_base_branch" {
		t.Fatalf("validation field = %q, want workflow_base_branch", validationErr.Field)
	}
}

func TestInitDefaultsWorkflowBaseBranchFromProviderDefaultBranch(t *testing.T) {
	store := newFakeStore()
	service := NewService(store, &fakeEventLog{}, &fakeTxRunner{}, fixedClock{now: testNow()}, sequenceIDs{})
	input := validInput()
	input.WorkflowBaseBranch = ""
	input.ProviderDefaultBranch = "main"

	result, err := service.Init(context.Background(), input)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if result.WorkflowBaseBranch != "main" {
		t.Fatalf("workflow_base_branch = %q, want main", result.WorkflowBaseBranch)
	}
	if store.created[0].WorkflowBaseBranch != "main" {
		t.Fatalf("stored workflow_base_branch = %q, want main", store.created[0].WorkflowBaseBranch)
	}
}

func TestInitRejectsViewerMembership(t *testing.T) {
	service := NewService(newFakeStore(), &fakeEventLog{}, &fakeTxRunner{}, fixedClock{now: testNow()}, sequenceIDs{})
	input := validInput()
	input.Membership.Role = spine.OrganizationMembershipRoleViewer

	_, err := service.Init(context.Background(), input)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Init() error = %v, want ErrForbidden", err)
	}
}

const (
	testUserID         spine.UserID         = "018f0000-0000-7000-8000-000000000001"
	testOrganizationID spine.OrganizationID = "018f0000-0000-7000-8000-000000000002"
	testProjectID      spine.ProjectID      = "018f0000-0000-7000-8000-000000000003"
)

func validInput() InitInput {
	return InitInput{
		ProjectID:             testProjectID,
		AuthenticatedUserID:   testUserID,
		Membership:            activeMembership(spine.OrganizationMembershipRoleMember),
		Provider:              "github",
		RepositoryFullName:    "heurema/goalrail",
		RepositoryURL:         "git@github.com:heurema/goalrail.git",
		ProviderDefaultBranch: "main",
		WorkflowBaseBranch:    "main",
		LocalRemoteName:       "origin",
		LocalHeadSHA:          "abc123",
	}
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

func existingBinding(provider string, fullName string) spine.RepoBinding {
	return spine.RepoBinding{
		ID:                 "018f0000-0000-7000-8000-000000000099",
		OrganizationID:     testOrganizationID,
		ProjectID:          testProjectID,
		CreatedByUserID:    testUserID,
		Provider:           provider,
		RepositoryFullName: fullName,
		RepositoryURL:      "git@github.com:" + fullName + ".git",
		DefaultBranch:      "main",
		WorkflowBaseBranch: "main",
		PathScope:          ".",
		AccessMode:         spine.RepoBindingAccessModeMetadataOnly,
		State:              spine.EntityStateActive,
		CreatedAt:          testNow(),
		UpdatedAt:          testNow(),
	}
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		project: spine.Project{
			ID:              testProjectID,
			OrganizationID:  testOrganizationID,
			CreatedByUserID: testUserID,
			Slug:            "default",
			DisplayName:     "Default Project",
			State:           spine.EntityStateActive,
			CreatedAt:       testNow(),
			UpdatedAt:       testNow(),
		},
		projectOK: true,
	}
}

type fakeStore struct {
	project                             spine.Project
	projectOK                           bool
	activeBinding                       *spine.RepoBinding
	organizationBinding                 *spine.RepoBinding
	activeBindingAfterCreateError       *spine.RepoBinding
	organizationBindingAfterCreateError *spine.RepoBinding
	createErr                           error
	created                             []spine.RepoBinding
}

func (s *fakeStore) GetProject(context.Context, spine.ProjectID) (spine.Project, bool, error) {
	return s.project, s.projectOK, nil
}

func (s *fakeStore) GetActiveRepoBindingForProject(context.Context, spine.ProjectID) (spine.RepoBinding, bool, error) {
	if s.activeBinding == nil {
		return spine.RepoBinding{}, false, nil
	}
	return *s.activeBinding, true, nil
}

func (s *fakeStore) GetActiveRepoBindingByOrganizationAndRepository(_ context.Context, _ spine.OrganizationID, provider string, repositoryFullName string) (spine.RepoBinding, bool, error) {
	if s.organizationBinding == nil {
		return spine.RepoBinding{}, false, nil
	}
	if !strings.EqualFold(strings.TrimSpace(s.organizationBinding.Provider), strings.TrimSpace(provider)) ||
		!strings.EqualFold(normalizeRepositoryFullName(s.organizationBinding.RepositoryFullName), normalizeRepositoryFullName(repositoryFullName)) {
		return spine.RepoBinding{}, false, nil
	}
	return *s.organizationBinding, true, nil
}

func (s *fakeStore) CreateRepoBinding(_ context.Context, binding spine.RepoBinding) error {
	if s.createErr != nil {
		if s.activeBindingAfterCreateError != nil {
			s.activeBinding = s.activeBindingAfterCreateError
		}
		if s.organizationBindingAfterCreateError != nil {
			s.organizationBinding = s.organizationBindingAfterCreateError
		}
		return s.createErr
	}
	s.created = append(s.created, binding)
	return nil
}

type fakeUniqueConstraintError struct {
	constraint string
}

func (e fakeUniqueConstraintError) Error() string {
	return "unique constraint violation"
}

func (e fakeUniqueConstraintError) ConstraintName() string {
	return e.constraint
}

type fakeEventLog struct {
	events []spine.Event
}

func (l *fakeEventLog) Append(_ context.Context, event spine.Event) error {
	l.events = append(l.events, event)
	return nil
}

type fakeTxRunner struct{}

func (r *fakeTxRunner) RunReadCommitted(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

type sequenceIDs struct{}

func (sequenceIDs) NewRepoBindingID() (spine.RepoBindingID, error) {
	return "018f0000-0000-7000-8000-000000000004", nil
}

func (sequenceIDs) NewEventID() (spine.EventID, error) {
	return "018f0000-0000-7000-8000-000000000006", nil
}

func testNow() time.Time {
	return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC)
}
