package projectcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/cli/internal/exitcode"
	"github.com/heurema/goalrail/apps/cli/internal/projectconfig"
	"github.com/heurema/goalrail/apps/cli/internal/projectscan"
	"github.com/heurema/goalrail/apps/cli/internal/term"
)

func TestProjectScanRequiresProjectMarker(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupRepo(t)
	writeFile(t, repoDir, "go.mod", "module example.com/repo\n")
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "shape")

	var stdout, stderr bytes.Buffer
	err := RunWithOptions(context.Background(), term.New(&stdout, &stderr), repoDir, []string{"scan", "--format", "json"}, Options{CacheRoot: t.TempDir(), Now: fixedNow})
	if err == nil {
		t.Fatal("Run(project scan) error = nil, want missing marker")
	}
	if got := exitcode.ForError(err); got != exitcode.Usage {
		t.Fatalf("exit code = %d, want usage", got)
	}
	if !strings.Contains(err.Error(), "missing .goalrail/project.yml") {
		t.Fatalf("error = %q, want missing marker", err.Error())
	}
}

func TestProjectScanBuildsBaselineAndOverlayWithoutServer(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupRepo(t)
	writeFile(t, repoDir, "go.mod", "module example.com/repo\n")
	writeFile(t, repoDir, "main_test.go", "package main\n")
	writeProjectConfig(t, repoDir)
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "shape")

	output, err := runProjectJSON(t, repoDir, []string{"scan", "--format", "json"}, t.TempDir())
	if err != nil {
		t.Fatalf("Run(project scan) error = %v", err)
	}

	if output.Baseline == nil {
		t.Fatal("baseline = nil, want baseline")
	}
	if output.Baseline.RepositoryBaselineProfileID == "" || output.Overlay.WorkspaceOverlayID == "" {
		t.Fatalf("baseline/overlay ids = %q/%q, want non-empty", output.Baseline.RepositoryBaselineProfileID, output.Overlay.WorkspaceOverlayID)
	}
	if output.Freshness.Status != projectscan.FreshnessFresh {
		t.Fatalf("freshness = %q, want fresh", output.Freshness.Status)
	}
	if !output.BaselineRebuilt {
		t.Fatal("baseline_rebuilt = false, want true for first scan")
	}
}

func TestProjectStatusDoesNotRebuildStaleBaseline(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupRepo(t)
	writeFile(t, repoDir, "go.mod", "module example.com/repo\n")
	writeProjectConfig(t, repoDir)
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "shape")
	baseline, err := projectscan.BuildBaseline(context.Background(), repoDir, "018f0000-0000-7000-8000-000000000004", projectscan.DefaultBuildOptions())
	if err != nil {
		t.Fatalf("BuildBaseline() error = %v", err)
	}
	cacheRoot := t.TempDir()
	cache := projectscan.NewCache(cacheRoot)
	if err := cache.WriteBaseline(baseline); err != nil {
		t.Fatalf("WriteBaseline() error = %v", err)
	}

	writeFile(t, repoDir, "README.md", "hello\n")
	runGit(t, repoDir, "add", "README.md")
	runGit(t, repoDir, "commit", "-m", "readme")
	newHead := gitOutputForTest(t, repoDir, "rev-parse", "--verify", "HEAD")

	output, err := runProjectJSON(t, repoDir, []string{"status", "--format", "json"}, cacheRoot)
	if err != nil {
		t.Fatalf("Run(project status) error = %v", err)
	}

	if output.Baseline == nil {
		t.Fatal("baseline = nil, want stale cached baseline")
	}
	if output.Baseline.HeadSHA == newHead {
		t.Fatalf("status rebuilt baseline at new head %s", newHead)
	}
	if output.Freshness.Status != projectscan.FreshnessStaleHead {
		t.Fatalf("freshness = %q, want stale_head", output.Freshness.Status)
	}
	if output.BaselineRebuilt {
		t.Fatal("baseline_rebuilt = true, want false for status")
	}
	dir, err := cache.Directory("018f0000-0000-7000-8000-000000000004", output.CanonicalRepoRoot)
	if err != nil {
		t.Fatalf("cache directory: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "baseline-"+newHead[:12]+"-v1.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("new-head baseline stat err = %v, want not exist", err)
	}
}

func TestProjectScanDirtyCriticalTextShowsStructuralRescan(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupRepo(t)
	writeFile(t, repoDir, "package.json", `{"scripts":{"test":"node test.js"}}`)
	writeProjectConfig(t, repoDir)
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "shape")
	if _, err := runProjectJSON(t, repoDir, []string{"scan", "--format", "json"}, t.TempDir()); err != nil {
		t.Fatalf("initial scan error = %v", err)
	}

	writeFile(t, repoDir, "package.json", `{"scripts":{"test":"node test.js","lint":"eslint ."}}`)
	var stdout, stderr bytes.Buffer
	err := RunWithOptions(context.Background(), term.New(&stdout, &stderr), repoDir, []string{"scan"}, Options{CacheRoot: t.TempDir(), Now: fixedNow})
	if err != nil {
		t.Fatalf("Run(project scan text) error = %v", err)
	}
	got := stdout.String()
	if !strings.Contains(got, "Scan-critical changes: package.json") {
		t.Fatalf("stdout = %q, want scan-critical package.json", got)
	}
	if !strings.Contains(got, "Freshness: structural_rescan_recommended") {
		t.Fatalf("stdout = %q, want structural rescan freshness", got)
	}
}

func runProjectJSON(t *testing.T, repoDir string, args []string, cacheRoot string) (Output, error) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	err := RunWithOptions(context.Background(), term.New(&stdout, &stderr), repoDir, args, Options{CacheRoot: cacheRoot, Now: fixedNow})
	if err != nil {
		return Output{}, err
	}
	var output Output
	decoder := json.NewDecoder(&stdout)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&output); err != nil {
		t.Fatalf("decode project JSON %q: %v", stdout.String(), err)
	}
	return output, nil
}

func writeProjectConfig(t *testing.T, repoDir string) {
	t.Helper()
	_, err := projectconfig.Write(repoDir, projectconfig.Config{
		Version:        projectconfig.Version,
		ServerURL:      "https://goalrail.example.test",
		OrganizationID: "018f0000-0000-7000-8000-000000000002",
		ProjectID:      "018f0000-0000-7000-8000-000000000003",
		RepoBindingID:  "018f0000-0000-7000-8000-000000000004",
		Repository: projectconfig.Repository{
			Provider:           "github",
			FullName:           "heurema/goalrail",
			URL:                "git@github.com:heurema/goalrail.git",
			WorkflowBaseBranch: "main",
		},
	})
	if err != nil {
		t.Fatalf("write project config: %v", err)
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

func fixedNow() time.Time {
	return time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
}
