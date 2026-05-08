package projectscan

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBaselineIdentityIncludesBindingRootHeadAndSchema(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupRepo(t)
	writeFile(t, repoDir, "go.mod", "module example.com/repo\n")
	runGit(t, repoDir, "add", "go.mod")
	runGit(t, repoDir, "commit", "-m", "add go module")
	head := gitOutputForTest(t, repoDir, "rev-parse", "--verify", "HEAD")
	root := canonicalPath(t, repoDir)

	baseline, err := BuildBaseline(context.Background(), repoDir, "rb-1", DefaultBuildOptions())
	if err != nil {
		t.Fatalf("BuildBaseline() error = %v", err)
	}

	want := BaselineProfileID("rb-1", root, head, SchemaVersion)
	if baseline.RepositoryBaselineProfileID != want {
		t.Fatalf("baseline id = %q, want %q", baseline.RepositoryBaselineProfileID, want)
	}
	if BaselineProfileID("rb-2", root, head, SchemaVersion) == want {
		t.Fatal("baseline id did not change when repo binding changed")
	}
	if BaselineProfileID("rb-1", root, strings.Repeat("a", 40), SchemaVersion) == want {
		t.Fatal("baseline id did not change when HEAD changed")
	}
	if BaselineProfileID("rb-1", root, head, SchemaVersion+1) == want {
		t.Fatal("baseline id did not change when schema changed")
	}
}

func TestCleanRepoBuildsBaselineAndCleanOverlay(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupRepo(t)
	writeFile(t, repoDir, "go.mod", "module example.com/repo\n")
	writeFile(t, repoDir, "main_test.go", "package main\n")
	writeFile(t, repoDir, ".github/workflows/ci.yml", "name: ci\n")
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "shape")

	baseline, err := BuildBaseline(context.Background(), repoDir, "rb-1", DefaultBuildOptions())
	if err != nil {
		t.Fatalf("BuildBaseline() error = %v", err)
	}
	overlay, _, err := BuildOverlay(context.Background(), repoDir, "rb-1", &baseline, OverlayOptions{Now: fixedNow})
	if err != nil {
		t.Fatalf("BuildOverlay() error = %v", err)
	}

	if baseline.Status != BaselineStatusQuick {
		t.Fatalf("baseline status = %q, want quick", baseline.Status)
	}
	if overlay.State != OverlayStateClean {
		t.Fatalf("overlay state = %q, want clean", overlay.State)
	}
	if got := EvaluateFreshness(baseline.HeadSHA, &baseline, overlay).Status; got != FreshnessFresh {
		t.Fatalf("freshness = %q, want fresh", got)
	}
	if baseline.ReadinessSignals.ProofSurface != "strong" {
		t.Fatalf("proof surface = %q, want strong", baseline.ReadinessSignals.ProofSurface)
	}
}

func TestBuildBaselineUsesSharedRepositoryShapeSignals(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupRepo(t)
	for _, relativePath := range []string{
		"go.mod",
		"package.json",
		"pnpm-lock.yaml",
		"package-lock.json",
		"yarn.lock",
		"bun.lock",
		"Cargo.toml",
		"Cargo.lock",
		"pyproject.toml",
		"requirements.txt",
		"poetry.lock",
		"uv.lock",
		"Gemfile",
		"Gemfile.lock",
		"composer.json",
		"composer.lock",
		"Dockerfile",
		".github/workflows/ci.yml",
	} {
		writeFile(t, repoDir, relativePath, "metadata\n")
	}
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "shape signals")

	baseline, err := BuildBaseline(context.Background(), repoDir, "rb-1", DefaultBuildOptions())
	if err != nil {
		t.Fatalf("BuildBaseline() error = %v", err)
	}

	for _, want := range []string{"go", "node", "rust", "python", "ruby", "php", "docker"} {
		requireContains(t, baseline.Shape.Toolchains, want, "toolchains")
	}
	for _, want := range []string{"pnpm", "npm", "yarn", "bun", "cargo", "poetry", "uv", "pip", "bundler", "composer"} {
		requireContains(t, baseline.Shape.PackageManagers, want, "package managers")
	}
	requireContains(t, baseline.ReadinessSignals.CI, ".github/workflows/ci.yml", "CI readiness")
	requireContains(t, baseline.Shape.Workspaces, ".", "workspaces")
	requireContains(t, baseline.Shape.EntrypointCandidates, "Dockerfile", "entrypoint candidates")
}

func TestDirtyNonCriticalFileDoesNotMakeBaselineStale(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupRepo(t)
	writeFile(t, repoDir, "go.mod", "module example.com/repo\n")
	writeFile(t, repoDir, "docs/note.md", "before\n")
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "shape")
	baseline, err := BuildBaseline(context.Background(), repoDir, "rb-1", DefaultBuildOptions())
	if err != nil {
		t.Fatalf("BuildBaseline() error = %v", err)
	}

	writeFile(t, repoDir, "docs/note.md", "after\n")
	overlay, _, err := BuildOverlay(context.Background(), repoDir, "rb-1", &baseline, OverlayOptions{Now: fixedNow})
	if err != nil {
		t.Fatalf("BuildOverlay() error = %v", err)
	}
	freshness := EvaluateFreshness(baseline.HeadSHA, &baseline, overlay)

	if overlay.State != OverlayStateDirty {
		t.Fatalf("overlay state = %q, want dirty", overlay.State)
	}
	if len(overlay.ScanCriticalChangedPaths) != 0 {
		t.Fatalf("scan critical paths = %#v, want empty", overlay.ScanCriticalChangedPaths)
	}
	if freshness.Status != FreshnessDirtyOverlay {
		t.Fatalf("freshness = %q, want dirty_overlay", freshness.Status)
	}
	if freshness.BaselineRebuildRecommended {
		t.Fatal("baseline rebuild recommended for non-critical dirty overlay")
	}
}

func TestDirtyScanCriticalFileMarksFreshness(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupRepo(t)
	writeFile(t, repoDir, "package.json", `{"scripts":{"test":"node test.js"}}`)
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "node")
	baseline, err := BuildBaseline(context.Background(), repoDir, "rb-1", DefaultBuildOptions())
	if err != nil {
		t.Fatalf("BuildBaseline() error = %v", err)
	}

	writeFile(t, repoDir, "package.json", `{"scripts":{"test":"node test.js","lint":"eslint ."}}`)
	overlay, _, err := BuildOverlay(context.Background(), repoDir, "rb-1", &baseline, OverlayOptions{Now: fixedNow})
	if err != nil {
		t.Fatalf("BuildOverlay() error = %v", err)
	}
	freshness := EvaluateFreshness(baseline.HeadSHA, &baseline, overlay)

	if got := overlay.ScanCriticalChangedPaths; len(got) != 1 || got[0] != "package.json" {
		t.Fatalf("scan critical paths = %#v, want package.json", got)
	}
	if freshness.Status != FreshnessScanCriticalDirty || !freshness.StructuralRescanRecommended {
		t.Fatalf("freshness = %#v, want scan-critical structural rescan", freshness)
	}
}

func TestHeadChangeMakesCachedBaselineStale(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupRepo(t)
	writeFile(t, repoDir, "go.mod", "module example.com/repo\n")
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "shape")
	baseline, err := BuildBaseline(context.Background(), repoDir, "rb-1", DefaultBuildOptions())
	if err != nil {
		t.Fatalf("BuildBaseline() error = %v", err)
	}

	writeFile(t, repoDir, "README.md", "hello\n")
	runGit(t, repoDir, "add", "README.md")
	runGit(t, repoDir, "commit", "-m", "readme")
	facts, err := DiscoverGit(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("DiscoverGit() error = %v", err)
	}
	overlay, _, err := BuildOverlay(context.Background(), repoDir, "rb-1", &baseline, OverlayOptions{Now: fixedNow})
	if err != nil {
		t.Fatalf("BuildOverlay() error = %v", err)
	}
	freshness := EvaluateFreshness(facts.HeadSHA, &baseline, overlay)

	if freshness.Status != FreshnessStaleHead || !freshness.BaselineRebuildRecommended {
		t.Fatalf("freshness = %#v, want stale_head rebuild", freshness)
	}
}

func TestSchemaVersionMismatchMakesBaselineStale(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupRepo(t)
	writeFile(t, repoDir, "go.mod", "module example.com/repo\n")
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "shape")
	baseline, err := BuildBaseline(context.Background(), repoDir, "rb-1", DefaultBuildOptions())
	if err != nil {
		t.Fatalf("BuildBaseline() error = %v", err)
	}
	baseline.SchemaVersion = SchemaVersion + 1
	overlay, _, err := BuildOverlay(context.Background(), repoDir, "rb-1", &baseline, OverlayOptions{Now: fixedNow})
	if err != nil {
		t.Fatalf("BuildOverlay() error = %v", err)
	}

	freshness := EvaluateFreshness(baseline.HeadSHA, &baseline, overlay)
	if freshness.Status != FreshnessSchemaMismatch || !freshness.BaselineRebuildRecommended {
		t.Fatalf("freshness = %#v, want schema_mismatch rebuild", freshness)
	}
}

func TestUnmergedOverlayBlocksBeforeBaselineMaintenance(t *testing.T) {
	t.Parallel()

	overlay := WorkspaceOverlay{
		State:         OverlayStateUnmerged,
		UnmergedPaths: []string{"go.mod"},
	}

	missing := EvaluateFreshness("head", nil, overlay)
	if missing.Status != FreshnessUnmergedBlocking || !missing.BlocksExecutionOrProof {
		t.Fatalf("missing baseline freshness = %#v, want unmerged blocking", missing)
	}

	baseline := RepositoryBaselineProfile{
		RepositoryBaselineProfileID: "baseline-1",
		HeadSHA:                     "old-head",
		SchemaVersion:               SchemaVersion,
	}
	stale := EvaluateFreshness("new-head", &baseline, overlay)
	if stale.Status != FreshnessUnmergedBlocking || !stale.BlocksExecutionOrProof {
		t.Fatalf("stale baseline freshness = %#v, want unmerged blocking", stale)
	}
}

func TestMissingAndCorruptCacheAreMissingBaseline(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupRepo(t)
	writeFile(t, repoDir, "go.mod", "module example.com/repo\n")
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "shape")
	facts, err := DiscoverGit(context.Background(), repoDir)
	if err != nil {
		t.Fatalf("DiscoverGit() error = %v", err)
	}
	cache := NewCache(t.TempDir())
	if _, ok, err := cache.LoadLatestBaseline("rb-1", facts.CanonicalRepoRoot); err != nil || ok {
		t.Fatalf("missing cache LoadLatestBaseline ok=%v err=%v, want false nil", ok, err)
	}
	dir, err := cache.Directory("rb-1", facts.CanonicalRepoRoot)
	if err != nil {
		t.Fatalf("cache directory: %v", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir cache: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, latestBaselineFile), []byte("{bad json"), 0o644); err != nil {
		t.Fatalf("write corrupt cache: %v", err)
	}
	if _, ok, err := cache.LoadLatestBaseline("rb-1", facts.CanonicalRepoRoot); err != nil || ok {
		t.Fatalf("corrupt cache LoadLatestBaseline ok=%v err=%v, want false nil", ok, err)
	}
}

func TestBuildBaselineTruncatesTrackedPathEnumeration(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupRepo(t)
	for i := 0; i < 5; i++ {
		writeFile(t, repoDir, filepath.Join("pkg", "file"+string(rune('a'+i))+".go"), "package pkg\n")
	}
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "many files")

	baseline, err := BuildBaseline(context.Background(), repoDir, "rb-1", BuildOptions{MaxFilesScanned: 2})
	if err != nil {
		t.Fatalf("BuildBaseline() error = %v", err)
	}

	if !baseline.Partiality.Truncated {
		t.Fatal("Partiality.Truncated = false, want true")
	}
	if baseline.Status != BaselineStatusPartial {
		t.Fatalf("baseline status = %q, want partial", baseline.Status)
	}
	if len(baseline.Receipts.Scanned) > 2 {
		t.Fatalf("scanned paths = %d, want <= 2", len(baseline.Receipts.Scanned))
	}
	if !hasSkipReason(baseline.Receipts.Skipped, "scan_budget_file_limit") {
		t.Fatalf("skipped = %#v, want scan_budget_file_limit", baseline.Receipts.Skipped)
	}
}

func TestSubmoduleIndicatorMarksPartialityWithoutRecursion(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupRepo(t)
	writeFile(t, repoDir, ".gitmodules", "[submodule \"vendor/lib\"]\n\tpath = vendor/lib\n\turl = https://example.com/lib.git\n")
	runGit(t, repoDir, "add", ".gitmodules")
	runGit(t, repoDir, "commit", "-m", "submodule marker")

	baseline, err := BuildBaseline(context.Background(), repoDir, "rb-1", DefaultBuildOptions())
	if err != nil {
		t.Fatalf("BuildBaseline() error = %v", err)
	}
	if !baseline.Partiality.SubmodulesPresent {
		t.Fatal("SubmodulesPresent = false, want true")
	}
	if baseline.Status != BaselineStatusPartial {
		t.Fatalf("baseline status = %q, want partial", baseline.Status)
	}
}

func TestSparseCheckoutMarksPartiality(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupRepo(t)
	writeFile(t, repoDir, "go.mod", "module example.com/repo\n")
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "shape")
	runGit(t, repoDir, "config", "core.sparseCheckout", "true")

	baseline, err := BuildBaseline(context.Background(), repoDir, "rb-1", DefaultBuildOptions())
	if err != nil {
		t.Fatalf("BuildBaseline() error = %v", err)
	}
	if !baseline.Partiality.SparseCheckout {
		t.Fatal("SparseCheckout = false, want true")
	}
	if baseline.Status != BaselineStatusPartial {
		t.Fatalf("baseline status = %q, want partial", baseline.Status)
	}
}

func TestCacheWritesBaselineAndOverlay(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupRepo(t)
	writeFile(t, repoDir, "go.mod", "module example.com/repo\n")
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "shape")
	baseline, err := BuildBaseline(context.Background(), repoDir, "rb-1", DefaultBuildOptions())
	if err != nil {
		t.Fatalf("BuildBaseline() error = %v", err)
	}
	overlay, rawStatus, err := BuildOverlay(context.Background(), repoDir, "rb-1", &baseline, OverlayOptions{Now: fixedNow})
	if err != nil {
		t.Fatalf("BuildOverlay() error = %v", err)
	}
	cache := NewCache(t.TempDir())
	if err := cache.WriteBaseline(baseline); err != nil {
		t.Fatalf("WriteBaseline() error = %v", err)
	}
	if err := cache.WriteOverlay(overlay, rawStatus); err != nil {
		t.Fatalf("WriteOverlay() error = %v", err)
	}
	loaded, ok, err := cache.LoadLatestBaseline("rb-1", baseline.CanonicalRepoRoot)
	if err != nil || !ok {
		t.Fatalf("LoadLatestBaseline() ok=%v err=%v, want true nil", ok, err)
	}
	if loaded.RepositoryBaselineProfileID != baseline.RepositoryBaselineProfileID {
		t.Fatalf("loaded baseline = %q, want %q", loaded.RepositoryBaselineProfileID, baseline.RepositoryBaselineProfileID)
	}
	dir, err := cache.Directory("rb-1", baseline.CanonicalRepoRoot)
	if err != nil {
		t.Fatalf("cache directory: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, overlayCurrentFile))
	if err != nil {
		t.Fatalf("read overlay cache: %v", err)
	}
	var decoded WorkspaceOverlay
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("decode overlay cache: %v", err)
	}
	if decoded.WorkspaceOverlayID != overlay.WorkspaceOverlayID {
		t.Fatalf("overlay id = %q, want %q", decoded.WorkspaceOverlayID, overlay.WorkspaceOverlayID)
	}
}

func setupRepo(t *testing.T) string {
	t.Helper()

	repoDir := t.TempDir()
	runGit(t, repoDir, "init")
	runGit(t, repoDir, "config", "user.name", "Goalrail Test")
	runGit(t, repoDir, "config", "user.email", "goalrail@example.test")
	return repoDir
}

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}
}

func runGit(t *testing.T, repoDir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", filepath.Clean(repoDir)}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, output)
	}
}

func gitOutputForTest(t *testing.T, repoDir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", filepath.Clean(repoDir)}, args...)...)
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %s failed: %v", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(output))
}

func writeFile(t *testing.T, repoDir string, relativePath string, content string) {
	t.Helper()
	path := filepath.Join(repoDir, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create parent for %s: %v", relativePath, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", relativePath, err)
	}
}

func canonicalPath(t *testing.T, value string) string {
	t.Helper()
	canonical, err := filepath.EvalSymlinks(value)
	if err != nil {
		t.Fatalf("canonicalize %q: %v", value, err)
	}
	return canonical
}

func fixedNow() time.Time {
	return time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
}

func requireContains(t *testing.T, values []string, want string, label string) {
	t.Helper()
	for _, value := range values {
		if value == want {
			return
		}
	}
	t.Fatalf("%s = %#v, want %q", label, values, want)
}
