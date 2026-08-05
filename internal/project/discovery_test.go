package project

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/heurema/goalrail/internal/boundedio"
	"github.com/heurema/goalrail/internal/domain"
)

func TestInspectSeparatesUnmanagedManagedAndDeclaredInvalidClaims(t *testing.T) {
	ctx := context.Background()
	repository := initGitRepository(t, filepath.Join(t.TempDir(), "repository"))

	before := gitStatus(t, repository)
	unmanaged, err := Inspect(ctx, repository)
	if err != nil {
		t.Fatal(err)
	}
	if unmanaged.State != ClaimUnmanaged {
		t.Fatalf("absent declaration state = %s, want unmanaged", unmanaged.State)
	}
	called := false
	if err := unmanaged.GuardDependentWrite(func() error { called = true; return nil }); !errors.Is(err, ErrClaimNotManaged) {
		t.Fatalf("unmanaged write guard error = %v", err)
	}
	if called || gitStatus(t, repository) != before {
		t.Fatal("unmanaged inspection or guard produced a managed write")
	}
	if _, err := os.Lstat(filepath.Join(repository, ".goalrail")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("unmanaged inspection created a Goalrail directory")
	}

	declarationPath := filepath.Join(repository, filepath.FromSlash(domain.ProjectDeclarationPath))
	if err := os.MkdirAll(filepath.Dir(declarationPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(declarationPath, []byte(`{"schema":`), 0o644); err != nil {
		t.Fatal(err)
	}
	invalid, err := Inspect(ctx, repository)
	if err != nil {
		t.Fatal(err)
	}
	if invalid.State != ClaimDeclaredInvalid || invalid.Reason != ReasonDeclarationMalformed {
		t.Fatalf("malformed declaration = (%s, %s), want declared-invalid/malformed", invalid.State, invalid.Reason)
	}
	called = false
	if err := invalid.GuardDependentWrite(func() error { called = true; return nil }); !errors.Is(err, ErrClaimInvalid) {
		t.Fatalf("invalid write guard error = %v", err)
	}
	if called {
		t.Fatal("declared-invalid guard invoked dependent write")
	}
}

func TestInspectRequiresCanonicalBoundedRegularDeclaration(t *testing.T) {
	ctx := context.Background()
	base := t.TempDir()
	repository := initGitRepository(t, filepath.Join(base, "repository"))
	declaration := projectDeclaration("prj_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	canonical := frozenDeclaration(t, declaration)
	declarationPath := filepath.Join(repository, filepath.FromSlash(domain.ProjectDeclarationPath))
	if err := os.MkdirAll(filepath.Dir(declarationPath), 0o755); err != nil {
		t.Fatal(err)
	}

	pretty, err := json.MarshalIndent(declaration, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(declarationPath, pretty, 0o644); err != nil {
		t.Fatal(err)
	}
	inspection, err := Inspect(ctx, repository)
	if err != nil {
		t.Fatal(err)
	}
	if inspection.State != ClaimDeclaredInvalid || inspection.Reason != ReasonDeclarationNonCanonical {
		t.Fatalf("pretty declaration = (%s, %s), want non-canonical", inspection.State, inspection.Reason)
	}

	if err := os.WriteFile(declarationPath, []byte(strings.Repeat("x", domain.MaxProjectDeclarationBytes+1)), 0o644); err != nil {
		t.Fatal(err)
	}
	inspection, err = Inspect(ctx, repository)
	if err != nil {
		t.Fatal(err)
	}
	if inspection.State != ClaimDeclaredInvalid || inspection.Reason != ReasonDeclarationTooLarge {
		t.Fatalf("oversized declaration = (%s, %s), want too-large", inspection.State, inspection.Reason)
	}

	outside := filepath.Join(base, "outside.json")
	if err := os.WriteFile(outside, canonical, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(declarationPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, declarationPath); err != nil {
		t.Fatal(err)
	}
	inspection, err = Inspect(ctx, repository)
	if err != nil {
		t.Fatal(err)
	}
	if inspection.State != ClaimDeclaredInvalid || inspection.Reason != ReasonDeclarationUnsafePath {
		t.Fatalf("symlink declaration = (%s, %s), want unsafe-path", inspection.State, inspection.Reason)
	}
}

func TestManagedIdentityIgnoresLegacyMarkerAndDetectorIsReadOnly(t *testing.T) {
	ctx := context.Background()
	repository := initGitRepository(t, filepath.Join(t.TempDir(), "repository"))
	writeDeclaration(t, repository, projectDeclaration("prj_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))

	withoutMarker, err := Inspect(ctx, repository)
	if err != nil {
		t.Fatal(err)
	}
	if withoutMarker.State != ClaimManaged {
		t.Fatalf("valid declaration without marker state = %s", withoutMarker.State)
	}
	legacy, err := DetectLegacyV018(ctx, repository)
	if err != nil {
		t.Fatal(err)
	}
	if legacy.Marker != LegacyMarkerAbsent || legacy.Overlay != LegacyOverlayAbsent {
		t.Fatalf("unexpected legacy evidence: %#v", legacy)
	}

	marker := []byte(`{"schema":"goalrail.ambient-marker/v0","initialized_at":"2026-08-04T12:00:00Z"}`)
	markerPath := filepath.Join(repository, filepath.FromSlash(legacyAmbientPath))
	if err := os.WriteFile(markerPath, marker, 0o644); err != nil {
		t.Fatal(err)
	}
	overlayPath := filepath.Join(repository, filepath.FromSlash(legacyV018OverlayPaths[0]))
	if err := os.MkdirAll(filepath.Dir(overlayPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(overlayPath, []byte("schema: fixture\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	legacy, err = DetectLegacyV018(ctx, repository)
	if err != nil {
		t.Fatal(err)
	}
	if legacy.Marker != LegacyMarkerValid || legacy.Overlay != LegacyOverlayInvalid {
		t.Fatalf("legacy evidence = %#v, want valid marker and invalid non-canon overlay", legacy)
	}
	withMarker, err := Inspect(ctx, repository)
	if err != nil {
		t.Fatal(err)
	}
	if withMarker.State != ClaimManaged || withMarker.Declaration.ProjectID != withoutMarker.Declaration.ProjectID {
		t.Fatal("legacy evidence changed committed project identity")
	}
	if err := os.Remove(markerPath); err != nil {
		t.Fatal(err)
	}
	afterRemoval, err := Inspect(ctx, repository)
	if err != nil {
		t.Fatal(err)
	}
	if afterRemoval.State != ClaimManaged || afterRemoval.Declaration.ProjectID != withoutMarker.Declaration.ProjectID {
		t.Fatal("deleting legacy marker made a declared project unmanaged")
	}
}

func TestClaimSubstitutionStopsBeforeFirstDependentWrite(t *testing.T) {
	ctx := context.Background()
	repository := initGitRepository(t, filepath.Join(t.TempDir(), "repository"))
	first := projectDeclaration("prj_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	second := projectDeclaration("prj_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	writeDeclaration(t, repository, first)
	inspection, err := Inspect(ctx, repository)
	if err != nil {
		t.Fatal(err)
	}
	if inspection.State != ClaimManaged {
		t.Fatalf("initial state = %s", inspection.State)
	}

	writeDeclaration(t, repository, second)
	writes := 0
	if err := inspection.GuardDependentWrite(func() error { writes++; return nil }); !errors.Is(err, ErrClaimChanged) {
		t.Fatalf("substitution guard error = %v, want ErrClaimChanged", err)
	}
	if writes != 0 {
		t.Fatal("dependent write ran after declaration substitution")
	}

	fresh, err := Inspect(ctx, repository)
	if err != nil {
		t.Fatal(err)
	}
	if err := fresh.GuardDependentWrite(func() error { writes++; return nil }); err != nil {
		t.Fatal(err)
	}
	if writes != 1 {
		t.Fatalf("validated dependent write count = %d, want 1", writes)
	}

	// Replacing the declaration inode with byte-identical content is still a
	// substitution and must invalidate the older snapshot.
	bytes := frozenDeclaration(t, second)
	temporary := filepath.Join(repository, ".goalrail", "replacement.json")
	if err := os.WriteFile(temporary, bytes, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(temporary, fresh.DeclarationPath); err != nil {
		t.Fatal(err)
	}
	if err := fresh.Revalidate(); !errors.Is(err, ErrClaimChanged) {
		t.Fatalf("byte-identical inode substitution error = %v", err)
	}
}

func TestClaimReadIsBoundToTheOpenedInodeAcrossRenameSwap(t *testing.T) {
	ctx := context.Background()
	repository := initGitRepository(t, filepath.Join(t.TempDir(), "repository"))
	first := projectDeclaration("prj_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	second := projectDeclaration("prj_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	writeDeclaration(t, repository, first)
	declarationPath := filepath.Join(repository, filepath.FromSlash(domain.ProjectDeclarationPath))
	replacementPath := filepath.Join(repository, ".goalrail", "replacement.json")
	if err := os.WriteFile(replacementPath, frozenDeclaration(t, second), 0o644); err != nil {
		t.Fatal(err)
	}

	reader := renameSwapReader(t, replacementPath)
	inspection, err := inspectWithReader(ctx, repository, reader)
	if err != nil {
		t.Fatal(err)
	}
	if inspection.State != ClaimDeclaredInvalid || inspection.Reason != ReasonDeclarationChanged {
		t.Fatalf("rename-swapped inspection = (%s, %s), want declared-invalid/changed", inspection.State, inspection.Reason)
	}

	fresh, err := Inspect(ctx, repository)
	if err != nil {
		t.Fatal(err)
	}
	if fresh.State != ClaimManaged {
		t.Fatalf("restored declaration state = %s", fresh.State)
	}
	if err := fresh.revalidateWithReader(reader); !errors.Is(err, ErrClaimChanged) {
		t.Fatalf("rename-swapped revalidation error = %v", err)
	}
	raw, err := os.ReadFile(declarationPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != string(frozenDeclaration(t, first)) {
		t.Fatal("race fixture did not restore the original declaration")
	}
}

func TestCommittedDeclarationSurvivesClonesAndLinkedWorktreesWithoutInitialization(t *testing.T) {
	ctx := context.Background()
	base := t.TempDir()
	origin := initGitRepository(t, filepath.Join(base, "origin"))
	wantID := domain.ProjectID("prj_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	writeDeclaration(t, origin, projectDeclaration(wantID))
	runGit(t, origin, "add", domain.ProjectDeclarationPath)
	runGit(t, origin, "commit", "--quiet", "-m", "add project declaration")

	cloneOne := filepath.Join(base, "clone-one")
	cloneTwo := filepath.Join(base, "clone-two")
	runGit(t, base, "clone", "--quiet", origin, cloneOne)
	runGit(t, base, "clone", "--quiet", origin, cloneTwo)

	worktrees := make([]string, 0, 3)
	for index := 1; index <= 3; index++ {
		path := filepath.Join(base, "linked-"+string(rune('0'+index)))
		runGit(t, origin, "worktree", "add", "--detach", path, "HEAD")
		worktrees = append(worktrees, path)
	}

	roots := append([]string{origin, cloneOne, cloneTwo}, worktrees...)
	var canonical []byte
	for _, root := range roots {
		inspection, err := Inspect(ctx, root)
		if err != nil {
			t.Fatalf("inspect %s: %v", root, err)
		}
		if inspection.State != ClaimManaged || inspection.Declaration.ProjectID != wantID {
			t.Fatalf("%s resolved (%s, %s), want managed/%s", root, inspection.State, inspection.Declaration.ProjectID, wantID)
		}
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(domain.ProjectDeclarationPath)))
		if err != nil {
			t.Fatal(err)
		}
		if canonical == nil {
			canonical = raw
		} else if string(raw) != string(canonical) {
			t.Fatalf("declaration bytes differ in %s", root)
		}
		if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(legacyAmbientPath))); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("%s required repeated checkout-local initialization", root)
		}
		if status := gitStatus(t, root); status != "" {
			t.Fatalf("inspection changed %s: %q", root, status)
		}
	}
}

func initGitRepository(t *testing.T, root string) string {
	t.Helper()
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	runGit(t, root, "init", "--quiet")
	runGit(t, root, "config", "user.email", "goalrail-test@example.invalid")
	runGit(t, root, "config", "user.name", "Goalrail Test")
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("fixture\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, root, "add", "README.md")
	runGit(t, root, "commit", "--quiet", "-m", "fixture")
	physical, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	return physical
}

func runGit(t *testing.T, directory string, arguments ...string) string {
	t.Helper()
	output, err := gitOutput(context.Background(), directory, arguments...)
	if err != nil {
		t.Fatalf("git %v in %s: %v", arguments, directory, err)
	}
	return strings.TrimSpace(output)
}

func gitStatus(t *testing.T, repository string) string {
	t.Helper()
	return runGit(t, repository, "status", "--porcelain", "--untracked-files=all")
}

func projectDeclaration(projectID domain.ProjectID) domain.ProjectDeclaration {
	return domain.ProjectDeclaration{
		Schema:          domain.ProjectSchemaV1,
		ProjectID:       projectID,
		ContractVersion: domain.GovernanceContractV1,
		Policy: domain.CommittedArtifactReference{
			Schema: domain.PolicySchemaV1,
			Path:   domain.DefaultProjectPolicyPath,
			Digest: domain.SHA256Digest("sha256:" + strings.Repeat("a", 64)),
		},
		Bootstrap: domain.CommittedArtifactReference{
			Schema: "goalrail.bootstrap/v1",
			Path:   domain.DefaultProjectBootstrapPath,
			Digest: domain.SHA256Digest("sha256:" + strings.Repeat("b", 64)),
		},
		SetupProfile: domain.CommittedArtifactReference{
			Schema: domain.SetupProfileSchemaV1,
			Path:   domain.DefaultProjectSetupProfilePath,
			Digest: domain.SHA256Digest("sha256:" + strings.Repeat("c", 64)),
		},
	}
}

func frozenDeclaration(t *testing.T, declaration domain.ProjectDeclaration) []byte {
	t.Helper()
	frozen, err := domain.FreezeProjectDeclaration(declaration)
	if err != nil {
		t.Fatal(err)
	}
	return frozen.CanonicalJSON()
}

func writeDeclaration(t *testing.T, repository string, declaration domain.ProjectDeclaration) {
	t.Helper()
	path := filepath.Join(repository, filepath.FromSlash(domain.ProjectDeclarationPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, frozenDeclaration(t, declaration), 0o644); err != nil {
		t.Fatal(err)
	}
}

func renameSwapReader(t *testing.T, replacementPath string) regularFileReader {
	t.Helper()
	return func(path string, label string, limit int) ([]byte, os.FileInfo, error) {
		backupPath := path + ".checked"
		if err := os.Rename(path, backupPath); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(replacementPath, path); err != nil {
			t.Fatal(err)
		}
		raw, info, readErr := boundedio.ReadRegularFileWithInfo(path, label, limit)
		if err := os.Rename(path, replacementPath); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(backupPath, path); err != nil {
			t.Fatal(err)
		}
		return raw, info, readErr
	}
}
