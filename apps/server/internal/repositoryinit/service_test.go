package repositoryinit

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/repobinding"
	"github.com/heurema/goalrail/apps/server/internal/spine"
)

func TestInitCreatesProjectAndRepoBinding(t *testing.T) {
	store := newFakeStore()
	events := &fakeEventLog{}
	service := newTestService(store, events)

	result, err := service.Init(context.Background(), validInput("acme/frontend"))
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	if !result.ProjectCreated || !result.RepoBindingCreated {
		t.Fatalf("created flags = %v/%v, want true/true", result.ProjectCreated, result.RepoBindingCreated)
	}
	if result.ProjectSlug != "github-acme-frontend" {
		t.Fatalf("project_slug = %q, want github-acme-frontend", result.ProjectSlug)
	}
	if result.ProjectDisplayName != "acme/frontend" {
		t.Fatalf("project_display_name = %q, want acme/frontend", result.ProjectDisplayName)
	}
	if result.ProjectID == "" || result.RepoBindingID == "" {
		t.Fatalf("ids = %q/%q, want generated project and repo binding ids", result.ProjectID, result.RepoBindingID)
	}
	if got := len(store.createdProjects); got != 1 {
		t.Fatalf("created projects = %d, want 1", got)
	}
	if got := len(store.createdBindings); got != 1 {
		t.Fatalf("created bindings = %d, want 1", got)
	}
	if got := len(events.events); got != 2 {
		t.Fatalf("events = %d, want project.created and repo_binding.initialized", got)
	}
	if events.events[0].Type != EventTypeProjectCreated {
		t.Fatalf("first event = %q, want %q", events.events[0].Type, EventTypeProjectCreated)
	}
	if events.events[1].Type != repobinding.EventTypeInitialized {
		t.Fatalf("second event = %q, want %q", events.events[1].Type, repobinding.EventTypeInitialized)
	}
}

func TestInitAllowsMultipleRepositoriesInSameOrganization(t *testing.T) {
	store := newFakeStore()
	service := newTestService(store, &fakeEventLog{})

	frontend, err := service.Init(context.Background(), validInput("acme/frontend"))
	if err != nil {
		t.Fatalf("frontend Init() error = %v", err)
	}
	backend, err := service.Init(context.Background(), validInput("acme/backend"))
	if err != nil {
		t.Fatalf("backend Init() error = %v", err)
	}

	if frontend.ProjectSlug != "github-acme-frontend" || backend.ProjectSlug != "github-acme-backend" {
		t.Fatalf("project slugs = %q/%q, want distinct repo-backed slugs", frontend.ProjectSlug, backend.ProjectSlug)
	}
	if frontend.ProjectID == backend.ProjectID {
		t.Fatalf("project ids are equal = %q, want one project per repo", frontend.ProjectID)
	}
	if got := len(store.createdBindings); got != 2 {
		t.Fatalf("created bindings = %d, want 2", got)
	}
}

func TestInitReturnsExistingRepositoryContextIdempotently(t *testing.T) {
	store := newFakeStore()
	service := newTestService(store, &fakeEventLog{})

	first, err := service.Init(context.Background(), validInput("acme/frontend"))
	if err != nil {
		t.Fatalf("first Init() error = %v", err)
	}
	second, err := service.Init(context.Background(), validInput("ACME/frontend"))
	if err != nil {
		t.Fatalf("second Init() error = %v", err)
	}

	if second.ProjectCreated || second.RepoBindingCreated {
		t.Fatalf("created flags = %v/%v, want false/false", second.ProjectCreated, second.RepoBindingCreated)
	}
	if second.ProjectID != first.ProjectID || second.RepoBindingID != first.RepoBindingID {
		t.Fatalf("ids = %q/%q, want existing %q/%q", second.ProjectID, second.RepoBindingID, first.ProjectID, first.RepoBindingID)
	}
	if got := len(store.createdProjects); got != 1 {
		t.Fatalf("created projects = %d, want 1", got)
	}
	if got := len(store.createdBindings); got != 1 {
		t.Fatalf("created bindings = %d, want 1", got)
	}
}

func TestInitReusesExistingActiveRepoBindingByOrganizationRepository(t *testing.T) {
	store := newFakeStore()
	project := projectFixture("018f0000-0000-7000-8000-000000000111", "github-acme-frontend", "acme/frontend")
	store.putProject(project)
	binding := repoBindingFixture("018f0000-0000-7000-8000-000000000222", project.ID, "github", "acme/frontend")
	store.putBinding(binding)
	service := newTestService(store, &fakeEventLog{})

	result, err := service.Init(context.Background(), validInput("acme/frontend"))
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	if result.ProjectID != project.ID || result.RepoBindingID != binding.ID {
		t.Fatalf("ids = %q/%q, want existing %q/%q", result.ProjectID, result.RepoBindingID, project.ID, binding.ID)
	}
	if result.ProjectCreated || result.RepoBindingCreated {
		t.Fatalf("created flags = %v/%v, want false/false", result.ProjectCreated, result.RepoBindingCreated)
	}
}

func TestInitReusesEmptyProjectForDerivedSlug(t *testing.T) {
	store := newFakeStore()
	project := projectFixture("018f0000-0000-7000-8000-000000000111", "github-acme-frontend", "Existing Empty Project")
	store.putProject(project)
	service := newTestService(store, &fakeEventLog{})

	result, err := service.Init(context.Background(), validInput("acme/frontend"))
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	if result.ProjectCreated {
		t.Fatal("ProjectCreated = true, want false for existing unbound project")
	}
	if !result.RepoBindingCreated {
		t.Fatal("RepoBindingCreated = false, want true")
	}
	if result.ProjectID != project.ID {
		t.Fatalf("project_id = %q, want existing %q", result.ProjectID, project.ID)
	}
	if got := len(store.createdProjects); got != 0 {
		t.Fatalf("created projects = %d, want 0", got)
	}
	if got := len(store.createdBindings); got != 1 {
		t.Fatalf("created bindings = %d, want 1", got)
	}
}

func TestInitRollsBackProjectAndEventWhenRepoBindingInitFails(t *testing.T) {
	store := newFakeStore()
	events := &fakeEventLog{}
	tx := &fakeTxRunner{
		rollback: func() {
			store.reset()
			events.events = nil
		},
	}
	service := NewService(
		store,
		failingRepoBindingInitializer{err: errors.New("forced repo binding failure")},
		events,
		tx,
		fixedClock{now: testNow()},
		&projectIDs{},
	)

	_, err := service.Init(context.Background(), validInput("acme/frontend"))
	if err == nil {
		t.Fatal("Init() error = nil, want repo binding failure")
	}
	if !strings.Contains(err.Error(), "forced repo binding failure") {
		t.Fatalf("Init() error = %v, want forced repo binding failure", err)
	}
	if tx.calls != 1 {
		t.Fatalf("transaction calls = %d, want 1", tx.calls)
	}
	if got := len(store.createdProjects); got != 0 {
		t.Fatalf("created projects after rollback = %d, want 0", got)
	}
	if got := len(events.events); got != 0 {
		t.Fatalf("events after rollback = %d, want 0", got)
	}
	if _, ok, err := store.GetProjectByOrganizationAndSlug(context.Background(), testOrganizationID, "github-acme-frontend"); err != nil || ok {
		t.Fatalf("project lookup after rollback ok=%v err=%v, want not found", ok, err)
	}
}

func TestInitRejectsViewer(t *testing.T) {
	service := newTestService(newFakeStore(), &fakeEventLog{})
	input := validInput("acme/frontend")
	input.Membership.Role = spine.OrganizationMembershipRoleViewer

	_, err := service.Init(context.Background(), input)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("Init() error = %v, want ErrForbidden", err)
	}
}

func TestInitAllowsOwnerAdminMember(t *testing.T) {
	for _, role := range []spine.OrganizationMembershipRole{
		spine.OrganizationMembershipRoleOwner,
		spine.OrganizationMembershipRoleAdmin,
		spine.OrganizationMembershipRoleMember,
	} {
		t.Run(string(role), func(t *testing.T) {
			service := newTestService(newFakeStore(), &fakeEventLog{})
			input := validInput("acme/frontend")
			input.Membership.Role = role

			if _, err := service.Init(context.Background(), input); err != nil {
				t.Fatalf("Init() error = %v, want nil", err)
			}
		})
	}
}

func TestInitRejectsProjectSlugCollisionWithDifferentRepository(t *testing.T) {
	store := newFakeStore()
	project := projectFixture("018f0000-0000-7000-8000-000000000111", "github-acme-frontend", "acme/other")
	store.putProject(project)
	store.putBinding(repoBindingFixture("018f0000-0000-7000-8000-000000000222", project.ID, "github", "acme/other"))
	service := newTestService(store, &fakeEventLog{})

	_, err := service.Init(context.Background(), validInput("acme/frontend"))
	if !errors.Is(err, ErrProjectSlugConflict) {
		t.Fatalf("Init() error = %v, want ErrProjectSlugConflict", err)
	}
}

func TestInitRejectsInactiveProjectSlug(t *testing.T) {
	store := newFakeStore()
	project := projectFixture("018f0000-0000-7000-8000-000000000111", "github-acme-frontend", "Inactive Project")
	project.State = "inactive"
	store.putProject(project)
	service := newTestService(store, &fakeEventLog{})

	_, err := service.Init(context.Background(), validInput("acme/frontend"))
	if !errors.Is(err, ErrProjectSlugUnavailable) {
		t.Fatalf("Init() error = %v, want ErrProjectSlugUnavailable", err)
	}
	if got := len(store.createdProjects); got != 0 {
		t.Fatalf("created projects = %d, want 0", got)
	}
}

func TestDeriveProjectSlugUsesProviderAndFullName(t *testing.T) {
	tests := map[string]struct {
		provider string
		fullName string
		want     string
	}{
		"github acme frontend": {provider: "github", fullName: "acme/frontend", want: "github-acme-frontend"},
		"github acme backend":  {provider: "github", fullName: "acme/backend", want: "github-acme-backend"},
		"gitlab acme frontend": {provider: "gitlab", fullName: "acme/frontend", want: "gitlab-acme-frontend"},
		"github web app":       {provider: "github", fullName: "web/app", want: "github-web-app"},
		"github mobile app":    {provider: "github", fullName: "mobile/app", want: "github-mobile-app"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := DeriveProjectSlug(tt.provider, tt.fullName); got != tt.want {
				t.Fatalf("DeriveProjectSlug() = %q, want %q", got, tt.want)
			}
		})
	}
}

const (
	testUserID         spine.UserID         = "018f0000-0000-7000-8000-000000000001"
	testOrganizationID spine.OrganizationID = "018f0000-0000-7000-8000-000000000002"
)

func newTestService(store *fakeStore, events *fakeEventLog) *Service {
	tx := &fakeTxRunner{}
	repoService := repobinding.NewService(store, events, tx, fixedClock{now: testNow()}, &repoBindingIDs{})
	return NewService(store, repoService, events, tx, fixedClock{now: testNow()}, &projectIDs{})
}

func validInput(repositoryFullName string) InitInput {
	return InitInput{
		AuthenticatedUserID:         testUserID,
		Membership:                  activeMembership(spine.OrganizationMembershipRoleMember),
		Provider:                    "github",
		RepositoryFullName:          repositoryFullName,
		RepositoryURL:               "git@github.com:" + strings.ToLower(repositoryFullName) + ".git",
		ProviderDefaultBranch:       "main",
		WorkflowBaseBranch:          "main",
		LocalRemoteName:             "origin",
		LocalHeadSHA:                "abc123",
		SuggestedProjectSlug:        "ignored",
		SuggestedProjectDisplayName: "ignored",
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

func projectFixture(id spine.ProjectID, slug string, displayName string) spine.Project {
	return spine.Project{
		ID:              id,
		OrganizationID:  testOrganizationID,
		CreatedByUserID: testUserID,
		Slug:            slug,
		DisplayName:     displayName,
		State:           spine.EntityStateActive,
		CreatedAt:       testNow(),
		UpdatedAt:       testNow(),
	}
}

func repoBindingFixture(id spine.RepoBindingID, projectID spine.ProjectID, provider string, fullName string) spine.RepoBinding {
	return spine.RepoBinding{
		ID:                 id,
		OrganizationID:     testOrganizationID,
		ProjectID:          projectID,
		CreatedByUserID:    testUserID,
		Provider:           provider,
		RepositoryFullName: fullName,
		RepositoryURL:      "git@github.com:" + strings.ToLower(fullName) + ".git",
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
	store := &fakeStore{}
	store.reset()
	return store
}

type fakeStore struct {
	projects          map[spine.ProjectID]spine.Project
	projectsBySlug    map[string]spine.ProjectID
	bindingsByProject map[spine.ProjectID]spine.RepoBinding
	bindingsByRepo    map[string]spine.RepoBinding
	createdProjects   []spine.Project
	createdBindings   []spine.RepoBinding
}

func (s *fakeStore) GetProject(_ context.Context, projectID spine.ProjectID) (spine.Project, bool, error) {
	project, ok := s.projects[projectID]
	return project, ok, nil
}

func (s *fakeStore) GetProjectByOrganizationAndSlug(_ context.Context, organizationID spine.OrganizationID, slug string) (spine.Project, bool, error) {
	projectID, ok := s.projectsBySlug[string(organizationID)+"/"+slug]
	if !ok {
		return spine.Project{}, false, nil
	}
	project, ok := s.projects[projectID]
	return project, ok, nil
}

func (s *fakeStore) GetActiveRepoBindingForProject(_ context.Context, projectID spine.ProjectID) (spine.RepoBinding, bool, error) {
	binding, ok := s.bindingsByProject[projectID]
	return binding, ok, nil
}

func (s *fakeStore) GetActiveRepoBindingByOrganizationAndRepository(_ context.Context, organizationID spine.OrganizationID, provider string, repositoryFullName string) (spine.RepoBinding, bool, error) {
	binding, ok := s.bindingsByRepo[repoKey(organizationID, provider, repositoryFullName)]
	return binding, ok, nil
}

func (s *fakeStore) CreateProject(_ context.Context, project spine.Project) error {
	s.createdProjects = append(s.createdProjects, project)
	s.putProject(project)
	return nil
}

func (s *fakeStore) CreateRepoBinding(_ context.Context, binding spine.RepoBinding) error {
	s.createdBindings = append(s.createdBindings, binding)
	s.putBinding(binding)
	return nil
}

func (s *fakeStore) reset() {
	s.projects = map[spine.ProjectID]spine.Project{}
	s.projectsBySlug = map[string]spine.ProjectID{}
	s.bindingsByProject = map[spine.ProjectID]spine.RepoBinding{}
	s.bindingsByRepo = map[string]spine.RepoBinding{}
	s.createdProjects = nil
	s.createdBindings = nil
}

func (s *fakeStore) putProject(project spine.Project) {
	s.projects[project.ID] = project
	s.projectsBySlug[string(project.OrganizationID)+"/"+project.Slug] = project.ID
}

func (s *fakeStore) putBinding(binding spine.RepoBinding) {
	s.bindingsByProject[binding.ProjectID] = binding
	s.bindingsByRepo[repoKey(binding.OrganizationID, binding.Provider, binding.RepositoryFullName)] = binding
}

func repoKey(organizationID spine.OrganizationID, provider string, repositoryFullName string) string {
	return string(organizationID) + "/" + strings.ToLower(strings.TrimSpace(provider)) + "/" + strings.ToLower(NormalizeRepositoryFullName(repositoryFullName))
}

type fakeEventLog struct {
	events []spine.Event
}

func (l *fakeEventLog) Append(_ context.Context, event spine.Event) error {
	l.events = append(l.events, event)
	return nil
}

type fakeTxRunner struct {
	calls    int
	rollback func()
}

func (r *fakeTxRunner) RunReadCommitted(ctx context.Context, fn func(context.Context) error) error {
	r.calls++
	if err := fn(ctx); err != nil {
		if r.rollback != nil {
			r.rollback()
		}
		return err
	}
	return nil
}

type failingRepoBindingInitializer struct {
	err error
}

func (i failingRepoBindingInitializer) Init(context.Context, repobinding.InitInput) (spine.RepoBindingInitResult, error) {
	return spine.RepoBindingInitResult{}, i.err
}

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

func testNow() time.Time {
	return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC)
}

type projectIDs struct {
	projectSeq int
	eventSeq   int
}

func (g *projectIDs) NewProjectID() (spine.ProjectID, error) {
	g.projectSeq++
	return spine.ProjectID(testUUID(100 + g.projectSeq)), nil
}

func (g *projectIDs) NewEventID() (spine.EventID, error) {
	g.eventSeq++
	return spine.EventID(testUUID(110 + g.eventSeq)), nil
}

type repoBindingIDs struct {
	bindingSeq int
	eventSeq   int
}

func (g *repoBindingIDs) NewRepoBindingID() (spine.RepoBindingID, error) {
	g.bindingSeq++
	return spine.RepoBindingID(testUUID(200 + g.bindingSeq)), nil
}

func (g *repoBindingIDs) NewEventID() (spine.EventID, error) {
	g.eventSeq++
	return spine.EventID(testUUID(210 + g.eventSeq)), nil
}

func testUUID(suffix int) string {
	return fmt.Sprintf("018f0000-0000-7000-8000-%012d", suffix)
}
