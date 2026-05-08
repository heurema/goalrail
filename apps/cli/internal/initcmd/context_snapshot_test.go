package initcmd

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/heurema/goalrail/apps/cli/internal/projectscan"
	"github.com/heurema/goalrail/apps/cli/internal/spine"
)

func TestCollectRepositoryInventoryDetectsWorkspaceManifests(t *testing.T) {
	root := t.TempDir()
	writeInventoryFile(t, root, filepath.Join("apps", "cli", "go.mod"), "module github.com/heurema/goalrail/apps/cli\n")
	writeInventoryFile(t, root, filepath.Join("apps", "server", "go.mod"), "module github.com/heurema/goalrail/apps/server\n")
	writeInventoryFile(t, root, filepath.Join("apps", "web", "package.json"), `{"name":"goalrail-web"}`)
	writeInventoryFile(t, root, filepath.Join("apps", "web", "package-lock.json"), `{"lockfileVersion":3}`)

	inventory, err := collectRepositoryInventory(root)
	if err != nil {
		t.Fatalf("collectRepositoryInventory() error = %v", err)
	}

	for _, want := range []string{"apps/cli/go.mod", "apps/server/go.mod", "apps/web/package.json", "apps/web/package-lock.json"} {
		if !stringSliceContains(inventory.detectedPaths, want) {
			t.Fatalf("detected paths = %#v, want %q", inventory.detectedPaths, want)
		}
	}
	for _, want := range []string{"apps/cli", "apps/server", "apps/web"} {
		if !stringSliceContains(inventory.workspaceCandidates, want) {
			t.Fatalf("workspace candidates = %#v, want %q", inventory.workspaceCandidates, want)
		}
	}
	if !stringSliceContains(inventory.detectedToolchains, "go") || !stringSliceContains(inventory.detectedToolchains, "node") {
		t.Fatalf("toolchains = %#v, want go and node from workspace manifests", inventory.detectedToolchains)
	}
	if !stringSliceContains(inventory.detectedPackageManagers, "npm") {
		t.Fatalf("package managers = %#v, want npm from workspace lockfile", inventory.detectedPackageManagers)
	}
}

func TestRepositoryContextSnapshotAndProjectScanBaselineSharedShapeParity(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir, headSHA := setupRepositoryShapeParityFixture(t)

	snapshot, err := buildRepositoryContextSnapshot(repoDir, repositoryShapeParityOutput(), spine.RepoBindingDraft{
		RemoteName: "origin",
		HeadSHA:    headSHA,
	})
	if err != nil {
		t.Fatalf("buildRepositoryContextSnapshot() error = %v", err)
	}
	baseline, err := projectscan.BuildBaseline(context.Background(), repoDir, "rb-shape-parity", projectscan.DefaultBuildOptions())
	if err != nil {
		t.Fatalf("projectscan.BuildBaseline() error = %v", err)
	}

	if snapshot.Repository.HeadSHA != baseline.HeadSHA {
		t.Fatalf("HEAD SHA = snapshot %q baseline %q, want parity", snapshot.Repository.HeadSHA, baseline.HeadSHA)
	}
	if !sameStrings(snapshot.DetectedToolchains, baseline.Shape.Toolchains) {
		t.Fatalf("toolchains = snapshot %#v baseline %#v, want parity", snapshot.DetectedToolchains, baseline.Shape.Toolchains)
	}
	if !sameStrings(snapshot.DetectedPackageManagers, baseline.Shape.PackageManagers) {
		t.Fatalf("package managers = snapshot %#v baseline %#v, want parity", snapshot.DetectedPackageManagers, baseline.Shape.PackageManagers)
	}
	if !sameStrings(snapshot.WorkspaceCandidates, withoutString(baseline.Shape.Workspaces, ".")) {
		t.Fatalf("workspaces = snapshot %#v baseline %#v, want parity excluding root workspace", snapshot.WorkspaceCandidates, baseline.Shape.Workspaces)
	}

	const workflowPath = ".github/workflows/ci.yml"
	if !stringSliceContains(snapshot.DetectedPaths, workflowPath) {
		t.Fatalf("snapshot detected paths = %#v, want workflow %q", snapshot.DetectedPaths, workflowPath)
	}
	if !stringSliceContains(baseline.ReadinessSignals.CI, workflowPath) {
		t.Fatalf("baseline CI readiness = %#v, want workflow %q", baseline.ReadinessSignals.CI, workflowPath)
	}
}

func TestRepositoryContextSnapshotAndProjectScanPathModelsDocumentedDivergence(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir, headSHA := setupRepositoryShapeParityFixture(t)

	snapshot, err := buildRepositoryContextSnapshot(repoDir, repositoryShapeParityOutput(), spine.RepoBindingDraft{
		RemoteName: "origin",
		HeadSHA:    headSHA,
	})
	if err != nil {
		t.Fatalf("buildRepositoryContextSnapshot() error = %v", err)
	}
	baseline, err := projectscan.BuildBaseline(context.Background(), repoDir, "rb-path-divergence", projectscan.DefaultBuildOptions())
	if err != nil {
		t.Fatalf("projectscan.BuildBaseline() error = %v", err)
	}

	// Snapshot paths are metadata inventory hints; baseline receipts are committed Git file paths.
	if sameStrings(snapshot.DetectedPaths, baseline.Receipts.Scanned) {
		t.Fatalf("snapshot detected paths unexpectedly equal baseline scanned receipts: %#v", snapshot.DetectedPaths)
	}
	if !stringSliceContains(snapshot.DetectedPaths, "apps/api/") {
		t.Fatalf("snapshot detected paths = %#v, want directory marker apps/api/", snapshot.DetectedPaths)
	}
	if stringSliceContains(baseline.Receipts.Scanned, "apps/api/") {
		t.Fatalf("baseline scanned receipts = %#v, want tracked file paths, not directory markers", baseline.Receipts.Scanned)
	}
	if !stringSliceContains(baseline.Receipts.Scanned, "apps/api/go.mod") {
		t.Fatalf("baseline scanned receipts = %#v, want committed manifest file", baseline.Receipts.Scanned)
	}
}

func setupRepositoryShapeParityFixture(t *testing.T) (string, string) {
	t.Helper()

	repoDir := t.TempDir()
	runGit(t, repoDir, "init")
	runGit(t, repoDir, "config", "user.name", "Goalrail Test")
	runGit(t, repoDir, "config", "user.email", "goalrail@example.test")
	runGit(t, repoDir, "remote", "add", "origin", "git@github.com:heurema/goalrail.git")

	writeFile(t, repoDir, "go.mod", "module github.com/heurema/goalrail\n")
	writeFile(t, repoDir, "package.json", `{"name":"goalrail","private":true}`)
	writeFile(t, repoDir, "pnpm-lock.yaml", "lockfileVersion: '9.0'\n")
	writeFile(t, repoDir, "apps/api/go.mod", "module github.com/heurema/goalrail/apps/api\n")
	writeFile(t, repoDir, "packages/ui/package.json", `{"name":"@goalrail/ui","private":true}`)
	writeFile(t, repoDir, ".github/workflows/ci.yml", "name: ci\n")

	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "add repository shape fixture")
	headSHA := strings.TrimSpace(runGitOutput(t, repoDir, "rev-parse", "--verify", "HEAD"))

	return repoDir, headSHA
}

func repositoryShapeParityOutput() spine.RepositoryContextInitOutput {
	return spine.RepositoryContextInitOutput{
		Provider:              "github",
		RepositoryFullName:    "heurema/goalrail",
		RepositoryURL:         "git@github.com:heurema/goalrail.git",
		ProviderDefaultBranch: "main",
		WorkflowBaseBranch:    "main",
	}
}

func sameStrings(left []string, right []string) bool {
	leftCopy := append([]string(nil), left...)
	rightCopy := append([]string(nil), right...)
	slices.Sort(leftCopy)
	slices.Sort(rightCopy)
	return slices.Equal(leftCopy, rightCopy)
}

func withoutString(values []string, excluded string) []string {
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		if value == excluded {
			continue
		}
		filtered = append(filtered, value)
	}
	return filtered
}

func writeInventoryFile(t *testing.T, root string, relative string, content string) {
	t.Helper()

	path := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create %s parent: %v", relative, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", relative, err)
	}
}
