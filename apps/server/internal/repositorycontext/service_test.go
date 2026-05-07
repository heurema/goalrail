package repositorycontext

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

func TestRecordSnapshotCreatesMetadataOnlyContextSnapshot(t *testing.T) {
	store := newFakeStore()
	binding := repoBindingFixture()
	store.putBinding(binding)
	events := &fakeEventLog{}
	service := newTestService(store, events)

	result, err := service.RecordSnapshot(context.Background(), validInput())
	if err != nil {
		t.Fatalf("RecordSnapshot() error = %v", err)
	}

	if !result.Created {
		t.Fatal("Created = false, want true")
	}
	if result.ContextSnapshotID == "" || result.Fingerprint == "" {
		t.Fatalf("snapshot id/fingerprint = %q/%q, want populated", result.ContextSnapshotID, result.Fingerprint)
	}
	if result.OrganizationID != binding.OrganizationID || result.ProjectID != binding.ProjectID || result.RepoBindingID != binding.ID {
		t.Fatalf("context ids = %#v, want binding context", result)
	}
	if got := len(store.createdSnapshots); got != 1 {
		t.Fatalf("created snapshots = %d, want 1", got)
	}
	if got := len(events.events); got != 1 {
		t.Fatalf("events = %d, want snapshot event", got)
	}
	if events.events[0].Type != EventTypeSnapshotRecorded {
		t.Fatalf("event type = %q, want %q", events.events[0].Type, EventTypeSnapshotRecorded)
	}
	var snapshot spine.RepositoryContextSnapshotRequest
	if err := json.Unmarshal(store.createdSnapshots[0].Snapshot, &snapshot); err != nil {
		t.Fatalf("unmarshal snapshot: %v", err)
	}
	if snapshot.DetectedPaths[0] != ".github/workflows/ci.yml" {
		t.Fatalf("detected paths = %#v, want sorted normalized paths", snapshot.DetectedPaths)
	}
	if bytes.Contains(store.createdSnapshots[0].Snapshot, []byte("access-token")) || bytes.Contains(store.createdSnapshots[0].Snapshot, []byte("refresh-token")) {
		t.Fatalf("snapshot contains token material: %s", store.createdSnapshots[0].Snapshot)
	}
}

func TestRecordSnapshotReturnsExistingFingerprintIdempotently(t *testing.T) {
	store := newFakeStore()
	store.putBinding(repoBindingFixture())
	service := newTestService(store, &fakeEventLog{})

	first, err := service.RecordSnapshot(context.Background(), validInput())
	if err != nil {
		t.Fatalf("first RecordSnapshot() error = %v", err)
	}
	second, err := service.RecordSnapshot(context.Background(), validInput())
	if err != nil {
		t.Fatalf("second RecordSnapshot() error = %v", err)
	}

	if second.Created {
		t.Fatal("second Created = true, want false")
	}
	if second.ContextSnapshotID != first.ContextSnapshotID || second.Fingerprint != first.Fingerprint {
		t.Fatalf("second id/fingerprint = %q/%q, want %q/%q", second.ContextSnapshotID, second.Fingerprint, first.ContextSnapshotID, first.Fingerprint)
	}
	if got := len(store.createdSnapshots); got != 1 {
		t.Fatalf("created snapshots = %d, want 1", got)
	}
}

func TestRecordSnapshotRejectsViewer(t *testing.T) {
	store := newFakeStore()
	store.putBinding(repoBindingFixture())
	service := newTestService(store, &fakeEventLog{})
	input := validInput()
	input.Membership.Role = spine.OrganizationMembershipRoleViewer

	_, err := service.RecordSnapshot(context.Background(), input)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("RecordSnapshot() error = %v, want ErrForbidden", err)
	}
}

func TestRecordSnapshotRejectsOtherOrganizationBinding(t *testing.T) {
	store := newFakeStore()
	store.putBinding(repoBindingFixture())
	service := newTestService(store, &fakeEventLog{})
	input := validInput()
	input.Membership.OrganizationID = "018f0000-0000-7000-8000-000000000999"

	_, err := service.RecordSnapshot(context.Background(), input)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("RecordSnapshot() error = %v, want ErrForbidden", err)
	}
}

func TestRecordSnapshotRejectsMalformedRepoBindingIDBeforeStore(t *testing.T) {
	store := newFakeStore()
	service := newTestService(store, &fakeEventLog{})
	input := validInput()
	input.RepoBindingID = "not-a-uuid"

	_, err := service.RecordSnapshot(context.Background(), input)
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) || validationErr.Field != "repo_binding_id" {
		t.Fatalf("RecordSnapshot() error = %v, want repo_binding_id validation", err)
	}
	if store.getBindingCalls != 0 {
		t.Fatalf("GetRepoBinding calls = %d, want 0 before valid repo_binding_id", store.getBindingCalls)
	}
}

func TestRecordSnapshotRejectsSnapshotMismatch(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(spine.RepositoryContextSnapshotRequest) spine.RepositoryContextSnapshotRequest
	}{
		{
			name: "repository",
			mutate: func(snapshot spine.RepositoryContextSnapshotRequest) spine.RepositoryContextSnapshotRequest {
				snapshot.Repository.FullName = "heurema/other"
				return snapshot
			},
		},
		{
			name: "branch",
			mutate: func(snapshot spine.RepositoryContextSnapshotRequest) spine.RepositoryContextSnapshotRequest {
				snapshot.Repository.WorkflowBaseBranch = "release"
				return snapshot
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newFakeStore()
			store.putBinding(repoBindingFixture())
			service := newTestService(store, &fakeEventLog{})
			input := validInput()
			input.Snapshot = tt.mutate(input.Snapshot)

			_, err := service.RecordSnapshot(context.Background(), input)
			if !errors.Is(err, ErrSnapshotMismatch) {
				t.Fatalf("RecordSnapshot() error = %v, want ErrSnapshotMismatch", err)
			}
			if got := len(store.createdSnapshots); got != 0 {
				t.Fatalf("created snapshots = %d, want 0", got)
			}
		})
	}
}

func TestRecordSnapshotRollsBackEventWhenSnapshotCreateFails(t *testing.T) {
	store := newFakeStore()
	store.putBinding(repoBindingFixture())
	events := &fakeEventLog{}
	tx := &fakeTxRunner{
		rollback: func() {
			store.snapshotsByFingerprint = map[string]spine.RepositoryContextSnapshotRecord{}
			store.createdSnapshots = nil
			events.events = nil
		},
	}
	store.createSnapshotErr = errors.New("forced snapshot failure")
	service := NewService(store, events, tx, fixedClock{now: testNow()}, &snapshotIDs{})

	_, err := service.RecordSnapshot(context.Background(), validInput())
	if err == nil {
		t.Fatal("RecordSnapshot() error = nil, want forced snapshot failure")
	}
	if got := len(events.events); got != 0 {
		t.Fatalf("events after rollback = %d, want 0", got)
	}
}

func TestRecordSnapshotResolvesConcurrentFingerprintRaceIdempotently(t *testing.T) {
	store := newFakeStore()
	store.putBinding(repoBindingFixture())
	input := validInput()
	normalized, err := normalizeInput(input)
	if err != nil {
		t.Fatalf("normalizeInput() error = %v", err)
	}
	snapshotJSON, err := json.Marshal(normalized.Snapshot)
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}
	fingerprint := fingerprintSnapshot(snapshotJSON)
	existing := spine.RepositoryContextSnapshotRecord{
		ID:             "018f0000-0000-7000-8000-000000000301",
		OrganizationID: testOrganizationID,
		ProjectID:      testProjectID,
		RepoBindingID:  testRepoBindingID,
		Source:         "goalrail_cli_init",
		SchemaVersion:  1,
		Fingerprint:    fingerprint,
		Snapshot:       snapshotJSON,
		CreatedAt:      testNow(),
	}
	store.createSnapshotErr = fakeUniqueConstraintError{constraint: "repository_context_snapshots_repo_binding_fingerprint_idx"}
	store.snapshotAfterCreateError = &existing
	service := newTestService(store, &fakeEventLog{})

	result, err := service.RecordSnapshot(context.Background(), input)
	if err != nil {
		t.Fatalf("RecordSnapshot() error = %v", err)
	}

	if result.Created {
		t.Fatal("Created = true, want idempotent existing snapshot")
	}
	if result.ContextSnapshotID != existing.ID {
		t.Fatalf("snapshot id = %q, want existing %q", result.ContextSnapshotID, existing.ID)
	}
}

func TestGetOrganizationRepositoryContextReturnsActiveContextsForAllRoles(t *testing.T) {
	roles := []spine.OrganizationMembershipRole{
		spine.OrganizationMembershipRoleOwner,
		spine.OrganizationMembershipRoleAdmin,
		spine.OrganizationMembershipRoleMember,
		spine.OrganizationMembershipRoleViewer,
	}
	for _, role := range roles {
		t.Run(string(role), func(t *testing.T) {
			store := newFakeStore()
			store.organization = organizationFixture()
			store.organizationOK = true
			store.contexts = []spine.ProjectRepoBindingContext{projectRepoBindingContextFixture()}
			service := newTestService(store, &fakeEventLog{})
			input := validReadInput()
			input.Membership.Role = role

			result, err := service.GetOrganizationRepositoryContext(context.Background(), input)
			if err != nil {
				t.Fatalf("GetOrganizationRepositoryContext() error = %v", err)
			}
			if result.Organization.ID != testOrganizationID {
				t.Fatalf("organization id = %q, want %q", result.Organization.ID, testOrganizationID)
			}
			if got := len(result.Contexts); got != 1 {
				t.Fatalf("contexts = %d, want 1", got)
			}
			if result.Contexts[0].RepoBinding.AccessMode != spine.RepoBindingAccessModeMetadataOnly {
				t.Fatalf("access mode = %q, want metadata_only", result.Contexts[0].RepoBinding.AccessMode)
			}
		})
	}
}

func TestGetOrganizationRepositoryContextReturnsNarrowPublicDTO(t *testing.T) {
	store := newFakeStore()
	store.organization = organizationFixture()
	store.organizationOK = true
	store.contexts = []spine.ProjectRepoBindingContext{projectRepoBindingContextFixture()}
	service := newTestService(store, &fakeEventLog{})

	result, err := service.GetOrganizationRepositoryContext(context.Background(), validReadInput())
	if err != nil {
		t.Fatalf("GetOrganizationRepositoryContext() error = %v", err)
	}
	body, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	if got := result.Organization.ID; got != testOrganizationID {
		t.Fatalf("organization id = %q, want %q", got, testOrganizationID)
	}
	if got := result.Contexts[0].Project.ID; got != testProjectID {
		t.Fatalf("project id = %q, want %q", got, testProjectID)
	}
	if got := result.Contexts[0].RepoBinding.ID; got != testRepoBindingID {
		t.Fatalf("repo binding id = %q, want %q", got, testRepoBindingID)
	}
	for _, forbidden := range []string{
		"installation_id",
		"organization_id",
		"project_id",
		"created_by_user_id",
		"vcs_connection_id",
		"repository_external_id",
		"public_base_url",
		"token",
		"credential",
		"proof",
		"readiness",
	} {
		if bytes.Contains(body, []byte(forbidden)) {
			t.Fatalf("response leaked %q: %s", forbidden, body)
		}
	}
}

func TestGetOrganizationRepositoryContextStripsRepositoryURLUserinfo(t *testing.T) {
	store := newFakeStore()
	store.organization = organizationFixture()
	store.organizationOK = true
	repoContext := projectRepoBindingContextFixture()
	repoContext.RepoBinding.RepositoryURL = "https://token:secret@github.com/heurema/goalrail.git"
	store.contexts = []spine.ProjectRepoBindingContext{repoContext}
	service := newTestService(store, &fakeEventLog{})

	result, err := service.GetOrganizationRepositoryContext(context.Background(), validReadInput())
	if err != nil {
		t.Fatalf("GetOrganizationRepositoryContext() error = %v", err)
	}
	got := result.Contexts[0].RepoBinding.RepositoryURL
	if got != "https://github.com/heurema/goalrail.git" {
		t.Fatalf("repository_url = %q, want userinfo stripped", got)
	}
	body, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	if bytes.Contains(body, []byte("token")) || bytes.Contains(body, []byte("secret")) {
		t.Fatalf("response leaked credentialed repository URL: %s", body)
	}
}

func TestGetOrganizationRepositoryContextRejectsCrossOrganizationRead(t *testing.T) {
	store := newFakeStore()
	store.organization = organizationFixture()
	store.organizationOK = true
	service := newTestService(store, &fakeEventLog{})
	input := validReadInput()
	input.OrganizationID = "018f0000-0000-7000-8000-000000000099"

	_, err := service.GetOrganizationRepositoryContext(context.Background(), input)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("GetOrganizationRepositoryContext() error = %v, want ErrForbidden", err)
	}
	if store.getOrganizationCalls != 0 {
		t.Fatalf("GetOrganization calls = %d, want 0 before authorization", store.getOrganizationCalls)
	}
}

func TestGetOrganizationRepositoryContextReturnsEmptyContexts(t *testing.T) {
	store := newFakeStore()
	store.organization = organizationFixture()
	store.organizationOK = true
	service := newTestService(store, &fakeEventLog{})

	result, err := service.GetOrganizationRepositoryContext(context.Background(), validReadInput())
	if err != nil {
		t.Fatalf("GetOrganizationRepositoryContext() error = %v", err)
	}
	if result.Contexts == nil {
		t.Fatal("contexts = nil, want empty array slice")
	}
	if len(result.Contexts) != 0 {
		t.Fatalf("contexts = %d, want 0", len(result.Contexts))
	}
}

func TestGetOrganizationRepositoryContextAuthorizesCaseInsensitiveUUIDPath(t *testing.T) {
	store := newFakeStore()
	store.organization = organizationFixture()
	store.organizationOK = true
	service := newTestService(store, &fakeEventLog{})
	input := validReadInput()
	input.OrganizationID = spine.OrganizationID(strings.ToUpper(string(testOrganizationID)))

	_, err := service.GetOrganizationRepositoryContext(context.Background(), input)
	if err != nil {
		t.Fatalf("GetOrganizationRepositoryContext() error = %v, want nil", err)
	}
	if store.getOrganizationCalls != 1 {
		t.Fatalf("GetOrganization calls = %d, want 1", store.getOrganizationCalls)
	}
}

func TestGetOrganizationRepositoryContextRejectsUnknownOrganization(t *testing.T) {
	store := newFakeStore()
	service := newTestService(store, &fakeEventLog{})

	_, err := service.GetOrganizationRepositoryContext(context.Background(), validReadInput())
	if !errors.Is(err, ErrOrganizationNotFound) {
		t.Fatalf("GetOrganizationRepositoryContext() error = %v, want ErrOrganizationNotFound", err)
	}
}

const (
	testUserID         spine.UserID         = "018f0000-0000-7000-8000-000000000001"
	testOrganizationID spine.OrganizationID = "018f0000-0000-7000-8000-000000000002"
	testProjectID      spine.ProjectID      = "018f0000-0000-7000-8000-000000000003"
	testRepoBindingID  spine.RepoBindingID  = "018f0000-0000-7000-8000-000000000004"
)

func newTestService(store *fakeStore, events *fakeEventLog) *Service {
	return NewService(store, events, &fakeTxRunner{}, fixedClock{now: testNow()}, &snapshotIDs{})
}

func validInput() RecordInput {
	return RecordInput{
		AuthenticatedUserID: testUserID,
		Membership: spine.OrganizationMembership{
			ID:             "018f0000-0000-7000-8000-000000000005",
			OrganizationID: testOrganizationID,
			UserID:         testUserID,
			Role:           spine.OrganizationMembershipRoleMember,
			State:          spine.EntityStateActive,
		},
		RepoBindingID: testRepoBindingID,
		Snapshot: spine.RepositoryContextSnapshotRequest{
			Source:        "goalrail_cli_init",
			SchemaVersion: 1,
			Repository: spine.RepositoryContextSnapshotRepository{
				Provider:              "github",
				FullName:              "heurema/goalrail",
				URL:                   "git@github.com:heurema/goalrail.git",
				ProviderDefaultBranch: "main",
				WorkflowBaseBranch:    "main",
				RemoteName:            "origin",
				HeadSHA:               "abc123",
			},
			DetectedPaths:           []string{"go.mod", ".github/workflows/ci.yml", "go.mod"},
			DetectedToolchains:      []string{"go"},
			DetectedPackageManagers: []string{"pnpm"},
			WorkspaceCandidates:     []string{"apps/cli"},
		},
	}
}

func validReadInput() ReadOrganizationContextInput {
	return ReadOrganizationContextInput{
		AuthenticatedUserID: testUserID,
		Membership: spine.OrganizationMembership{
			ID:             "018f0000-0000-7000-8000-000000000005",
			OrganizationID: testOrganizationID,
			UserID:         testUserID,
			Role:           spine.OrganizationMembershipRoleViewer,
			State:          spine.EntityStateActive,
		},
		OrganizationID: testOrganizationID,
	}
}

func organizationFixture() spine.Organization {
	return spine.Organization{
		ID:             testOrganizationID,
		InstallationID: "018f0000-0000-7000-8000-000000000006",
		Slug:           "goalrail-dev",
		DisplayName:    "Goalrail Dev",
		State:          spine.EntityStateActive,
		CreatedAt:      testNow(),
		UpdatedAt:      testNow(),
	}
}

func projectRepoBindingContextFixture() spine.ProjectRepoBindingContext {
	return spine.ProjectRepoBindingContext{
		Project: spine.Project{
			ID:              testProjectID,
			OrganizationID:  testOrganizationID,
			CreatedByUserID: testUserID,
			Slug:            "github-heurema-goalrail",
			DisplayName:     "heurema/goalrail",
			State:           spine.EntityStateActive,
			CreatedAt:       testNow(),
			UpdatedAt:       testNow(),
		},
		RepoBinding: repoBindingFixture(),
	}
}

func repoBindingFixture() spine.RepoBinding {
	return spine.RepoBinding{
		ID:                   testRepoBindingID,
		OrganizationID:       testOrganizationID,
		ProjectID:            testProjectID,
		CreatedByUserID:      testUserID,
		VcsConnectionID:      "vcs-connection-internal",
		Provider:             "github",
		RepositoryExternalID: "repo-external-internal",
		RepositoryFullName:   "heurema/goalrail",
		RepositoryURL:        "git@github.com:heurema/goalrail.git",
		DefaultBranch:        "main",
		WorkflowBaseBranch:   "main",
		PathScope:            ".",
		AccessMode:           spine.RepoBindingAccessModeMetadataOnly,
		State:                spine.EntityStateActive,
		CreatedAt:            testNow(),
		UpdatedAt:            testNow(),
	}
}

type fakeStore struct {
	binding                  spine.RepoBinding
	bindingOK                bool
	organization             spine.Organization
	organizationOK           bool
	contexts                 []spine.ProjectRepoBindingContext
	getBindingCalls          int
	getOrganizationCalls     int
	snapshotsByFingerprint   map[string]spine.RepositoryContextSnapshotRecord
	createdSnapshots         []spine.RepositoryContextSnapshotRecord
	createSnapshotErr        error
	snapshotAfterCreateError *spine.RepositoryContextSnapshotRecord
}

func newFakeStore() *fakeStore {
	return &fakeStore{snapshotsByFingerprint: map[string]spine.RepositoryContextSnapshotRecord{}}
}

func (s *fakeStore) GetRepoBinding(_ context.Context, _ spine.RepoBindingID) (spine.RepoBinding, bool, error) {
	s.getBindingCalls++
	return s.binding, s.bindingOK, nil
}

func (s *fakeStore) GetOrganization(_ context.Context, _ spine.OrganizationID) (spine.Organization, bool, error) {
	s.getOrganizationCalls++
	return s.organization, s.organizationOK, nil
}

func (s *fakeStore) ListActiveProjectRepoBindingContexts(_ context.Context, _ spine.OrganizationID) ([]spine.ProjectRepoBindingContext, error) {
	if s.contexts == nil {
		return []spine.ProjectRepoBindingContext{}, nil
	}
	return append([]spine.ProjectRepoBindingContext(nil), s.contexts...), nil
}

func (s *fakeStore) GetRepositoryContextSnapshotByFingerprint(_ context.Context, repoBindingID spine.RepoBindingID, fingerprint string) (spine.RepositoryContextSnapshotRecord, bool, error) {
	record, ok := s.snapshotsByFingerprint[string(repoBindingID)+"/"+fingerprint]
	return record, ok, nil
}

func (s *fakeStore) CreateRepositoryContextSnapshot(_ context.Context, record spine.RepositoryContextSnapshotRecord) error {
	if s.createSnapshotErr != nil {
		if s.snapshotAfterCreateError != nil {
			s.snapshotsByFingerprint[string(s.snapshotAfterCreateError.RepoBindingID)+"/"+s.snapshotAfterCreateError.Fingerprint] = *s.snapshotAfterCreateError
		}
		return s.createSnapshotErr
	}
	s.createdSnapshots = append(s.createdSnapshots, record)
	s.snapshotsByFingerprint[string(record.RepoBindingID)+"/"+record.Fingerprint] = record
	return nil
}

func (s *fakeStore) putBinding(binding spine.RepoBinding) {
	s.binding = binding
	s.bindingOK = true
}

type fakeEventLog struct {
	events []spine.Event
}

func (l *fakeEventLog) Append(_ context.Context, event spine.Event) error {
	l.events = append(l.events, event)
	return nil
}

type fakeTxRunner struct {
	rollback func()
}

func (r *fakeTxRunner) RunReadCommitted(ctx context.Context, fn func(context.Context) error) error {
	if err := fn(ctx); err != nil {
		if r.rollback != nil {
			r.rollback()
		}
		return err
	}
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

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

func testNow() time.Time {
	return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC)
}

type snapshotIDs struct {
	snapshotSeq int
	eventSeq    int
}

func (g *snapshotIDs) NewRepositoryContextSnapshotID() (spine.RepositoryContextSnapshotID, error) {
	g.snapshotSeq++
	return spine.RepositoryContextSnapshotID(testUUID(300 + g.snapshotSeq)), nil
}

func (g *snapshotIDs) NewEventID() (spine.EventID, error) {
	g.eventSeq++
	return spine.EventID(testUUID(310 + g.eventSeq)), nil
}

func testUUID(suffix int) string {
	return fmt.Sprintf("018f0000-0000-7000-8000-%012d", suffix)
}

func TestNormalizeInputSortsListsForStableFingerprint(t *testing.T) {
	input := validInput()
	input.Snapshot.DetectedToolchains = []string{"node", "go", "node", ""}

	normalized, err := normalizeInput(input)
	if err != nil {
		t.Fatalf("normalizeInput() error = %v", err)
	}
	if got := strings.Join(normalized.Snapshot.DetectedToolchains, ","); got != "go,node" {
		t.Fatalf("toolchains = %q, want sorted unique list", got)
	}
}
