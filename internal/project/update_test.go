package project

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/heurema/goalrail/internal/domain"
)

func TestProjectUpdateRequiresExplicitDiscardForManagedBlockDrift(t *testing.T) {
	repository := initializedProject(t)
	path := filepath.Join(repository, AgentsRootPath)
	editedBlock := RenderManagedBlock([]byte("locally edited Goalrail block"))
	before := append([]byte("owner before\n"), editedBlock...)
	before = append(before, []byte("owner after\n")...)
	writeTestFile(t, path, before)

	input := CanonUpdateInput{RepositoryRoot: repository, StateRoot: t.TempDir(), Now: fixedProjectUpdateClock}
	if _, err := UpdateProjectCanon(context.Background(), input); !errors.Is(err, ErrProjectLocalEdits) {
		t.Fatalf("managed-block drift error = %v", err)
	}
	if got, err := os.ReadFile(path); err != nil || !bytes.Equal(got, before) {
		t.Fatal("refused update changed owner instruction bytes")
	}

	input.DiscardLocalEdits = true
	report, err := UpdateProjectCanon(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	files, err := RenderProjectCanon(report.ProjectID)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(after, []byte("owner before\n")) || !bytes.HasSuffix(after, []byte("owner after\n")) ||
		!bytes.Contains(after, contentFor(files, AgentsSnippetPath)) {
		t.Fatalf("explicit block discard did not preserve owner bytes:\n%s", after)
	}
	if report.Backup == "" || !containsString(report.DiscardedLocalEdits, AgentsRootPath) || !report.Verified {
		t.Fatalf("discard report = %#v", report)
	}
	recovered, err := os.ReadFile(filepath.Join(report.Backup, AgentsRootPath))
	if err != nil || !bytes.Equal(recovered, before) {
		t.Fatalf("managed-block recovery bytes = %q, %v", recovered, err)
	}
}

func TestProjectUpdateMovesKnownPreviousFilesAndBlocks(t *testing.T) {
	repository := initializedProject(t)
	previousDescriptor := []byte(`{"schema":"goalrail.planning-adapter/v0","compiler":"previous"}` + "\n")
	writeTestFile(t, filepath.Join(repository, filepath.FromSlash(PlanningAdapterDescriptorPath)), previousDescriptor)
	previousBlock := RenderManagedBlock([]byte("previous retained adapter"))
	owner := append([]byte("owner prefix\n"), previousBlock...)
	owner = append(owner, []byte("owner suffix\n")...)
	writeTestFile(t, filepath.Join(repository, AgentsRootPath), owner)

	report, err := updateProjectCanon(context.Background(), CanonUpdateInput{
		RepositoryRoot: repository,
		StateRoot:      t.TempDir(),
		Now:            fixedProjectUpdateClock,
	}, canonUpdateTestHooks{
		retainedFiles:  []RenderedFile{{Path: PlanningAdapterDescriptorPath, Content: previousDescriptor}},
		retainedBlocks: [][]byte{previousBlock},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Verified || report.AlreadyCurrent {
		t.Fatalf("previous-canon report = %#v", report)
	}
	updated, err := os.ReadFile(filepath.Join(repository, AgentsRootPath))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(updated, []byte("owner prefix\n")) || !bytes.HasSuffix(updated, []byte("owner suffix\n")) {
		t.Fatal("known-block transition changed owner-authored bytes")
	}
	if outcomeAction(report.Files, PlanningAdapterDescriptorPath) != CanonUpdateUpdated ||
		outcomeAction(report.ManagedBlocks, AgentsRootPath) != CanonUpdateUpdated {
		t.Fatalf("known transitions were not reported: %#v", report)
	}
}

func TestProjectUpdatePreservesDeclarationBoundOwnerPolicyBytes(t *testing.T) {
	repository := initializedProject(t)
	inspection, err := Inspect(context.Background(), repository)
	if err != nil {
		t.Fatal(err)
	}
	artifacts, err := InspectGoverningArtifacts(inspection)
	if err != nil {
		t.Fatal(err)
	}
	policy := artifacts.PolicyValue
	policy.Rules[0].RequiredEvidenceKinds = append(policy.Rules[0].RequiredEvidenceKinds, "review_index")
	frozenPolicy, err := domain.FreezeProjectPolicy(policy)
	if err != nil {
		t.Fatal(err)
	}
	policyPath := filepath.Join(repository, filepath.FromSlash(domain.DefaultProjectPolicyPath))
	writeTestFile(t, policyPath, frozenPolicy.CanonicalJSON())
	declaration := inspection.Declaration
	declaration.Policy.Digest = frozenPolicy.Digest()
	frozenDeclaration, err := domain.FreezeProjectDeclaration(declaration)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(repository, filepath.FromSlash(domain.ProjectDeclarationPath)), frozenDeclaration.CanonicalJSON())
	before, err := os.ReadFile(policyPath)
	if err != nil {
		t.Fatal(err)
	}

	report, err := UpdateProjectCanon(context.Background(), CanonUpdateInput{
		RepositoryRoot: repository, StateRoot: t.TempDir(), Now: fixedProjectUpdateClock,
	})
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(policyPath)
	if err != nil || !bytes.Equal(after, before) {
		t.Fatal("canon update rewrote repository-owned policy bytes")
	}
	if report.ProjectID != declaration.ProjectID || !report.Verified {
		t.Fatalf("owner-policy update report = %#v", report)
	}
}

func TestProjectUpdateStopsOnJustInTimeRace(t *testing.T) {
	repository := initializedProject(t)
	path := filepath.Join(repository, filepath.FromSlash(PlanningAdapterDescriptorPath))
	previous := []byte("retained previous descriptor\n")
	concurrent := []byte("concurrent owner edit\n")
	writeTestFile(t, path, previous)
	report, err := updateProjectCanon(context.Background(), CanonUpdateInput{
		RepositoryRoot: repository,
		StateRoot:      t.TempDir(),
		Now:            fixedProjectUpdateClock,
	}, canonUpdateTestHooks{
		retainedFiles: []RenderedFile{{Path: PlanningAdapterDescriptorPath, Content: previous}},
		afterBackup: func() {
			writeTestFile(t, path, concurrent)
		},
	})
	if !errors.Is(err, errProjectChangedOnUpdate) {
		t.Fatalf("race error = %v", err)
	}
	if got, readErr := os.ReadFile(path); readErr != nil || !bytes.Equal(got, concurrent) {
		t.Fatal("just-in-time refusal overwrote the concurrent edit")
	}
	if report.Backup == "" {
		t.Fatal("race report lost the recovery path")
	}
}

func TestProjectUpdateKeepsOneCumulativeRecoverySetAcrossDiscardRetry(t *testing.T) {
	repository := initializedProject(t)
	bootstrapPath := filepath.Join(repository, filepath.FromSlash(BootstrapPath))
	agentsPath := filepath.Join(repository, AgentsRootPath)
	bootstrapEdit := []byte("owner-edited bootstrap\n")
	agentsInitial := RenderManagedBlock([]byte("first local block edit"))
	agentsLate := RenderManagedBlock([]byte("late local block edit"))
	writeTestFile(t, bootstrapPath, bootstrapEdit)
	writeTestFile(t, agentsPath, agentsInitial)
	mutated := false

	report, err := updateProjectCanon(context.Background(), CanonUpdateInput{
		RepositoryRoot:    repository,
		StateRoot:         t.TempDir(),
		DiscardLocalEdits: true,
		Now:               fixedProjectUpdateClock,
	}, canonUpdateTestHooks{
		beforeReplace: func(path string) {
			if path == BootstrapPath && !mutated {
				mutated = true
				writeTestFile(t, agentsPath, agentsLate)
			}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Verified || report.Backup == "" {
		t.Fatalf("discard retry report = %#v", report)
	}
	for path, want := range map[string][]byte{
		BootstrapPath:  bootstrapEdit,
		AgentsRootPath: agentsLate,
	} {
		got, readErr := os.ReadFile(filepath.Join(report.Backup, filepath.FromSlash(path)))
		if readErr != nil || !bytes.Equal(got, want) {
			t.Fatalf("cumulative recovery %s = %q, want %q (%v)", path, got, want, readErr)
		}
	}
	entries, err := filepath.Glob(filepath.Join(filepath.Dir(report.Backup), "*"))
	if err != nil || len(entries) != 1 {
		t.Fatalf("recovery sets = %v, %v", entries, err)
	}
}

func initializedProject(t *testing.T) string {
	t.Helper()
	repository := initGitRepository(t, filepath.Join(t.TempDir(), "repository"))
	if _, err := Initialize(context.Background(), repository, InitializeOptions{}); err != nil {
		t.Fatal(err)
	}
	return repository
}

func fixedProjectUpdateClock() time.Time {
	return time.Date(2026, time.August, 4, 12, 0, 0, 0, time.UTC)
}

func outcomeAction(outcomes []CanonUpdateOutcome, path string) CanonUpdateAction {
	for _, outcome := range outcomes {
		if outcome.Path == path {
			return outcome.Action
		}
	}
	return ""
}

func containsString(values []string, value string) bool {
	for _, existing := range values {
		if existing == value {
			return true
		}
	}
	return false
}
