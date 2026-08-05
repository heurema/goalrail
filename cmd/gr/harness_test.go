package main

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

	"github.com/heurema/goalrail/internal/harness"
	projectstate "github.com/heurema/goalrail/internal/project"
)

func TestUpdateReportExposesAPartialMutationWithoutABackup(t *testing.T) {
	report := harness.UpdateReport{Files: []harness.FileOutcome{
		{Path: "unchanged", Action: harness.ActionUnchanged},
		{Path: "created", Action: harness.ActionCreated},
	}}
	if !updateReportHasChanges(report) {
		t.Fatal("a partial creation would be hidden on the update error path")
	}
	if updateReportHasChanges(harness.UpdateReport{Files: []harness.FileOutcome{{Path: "unchanged", Action: harness.ActionUnchanged}}}) {
		t.Fatal("an unchanged preflight was reported as a partial mutation")
	}
}

// scratchRepository is shared by the command integration tests.
func scratchRepository(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, arguments := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "probe@localhost"},
		{"config", "user.name", "probe"},
		{"config", "core.excludesFile", os.DevNull},
	} {
		command := exec.Command("git", append([]string{"-C", root}, arguments...)...)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", arguments, err, output)
		}
	}
	return root
}

func gitCommand(t *testing.T, root string, arguments ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", root}, arguments...)...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", arguments, err, output)
	}
}

func gitOutput(t *testing.T, root string, arguments ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", root}, arguments...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", arguments, err, output)
	}
	return string(output)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runCommand(t *testing.T, arguments ...string) (string, string, error) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	err := run(context.Background(), arguments, strings.NewReader(""), &stdout, &stderr, productionService)
	return stdout.String(), stderr.String(), err
}

func TestInitCommandCreatesPortableProjectContractOnly(t *testing.T) {
	root := scratchRepository(t)
	stdout, _, err := runCommand(t, "init", "--repo", root, "--scaffold", "codex")
	if err != nil {
		t.Fatal(err)
	}
	var report projectstate.InitializeReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatal(err)
	}
	if !report.Managed || report.ProjectID == "" || report.LocallyReady || !report.SetupRequired || report.SharedAdmissionActive {
		t.Fatalf("v1 init report = %#v", report)
	}
	if report.RequestedScaffold != "codex" || report.SetupReason != "GOALRAIL_SETUP_REQUIRED" {
		t.Fatalf("requested local setup was not reported: %#v", report)
	}
	for _, forbidden := range []string{".goalrail/ambient.json", ".codex/settings.json", ".claude/settings.json"} {
		if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(forbidden))); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("init created checkout-local path %s: %v", forbidden, err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, ".goalrail", "project.json")); err != nil {
		t.Fatalf("committed declaration missing: %v", err)
	}

	before, err := os.ReadFile(filepath.Join(root, ".goalrail", "project.json"))
	if err != nil {
		t.Fatal(err)
	}
	stdout, _, err = runCommand(t, "init", "--repo", filepath.Join(root, ".goalrail"), "--scaffold", "codex")
	if err != nil {
		t.Fatal(err)
	}
	var repeat projectstate.InitializeReport
	if err := json.Unmarshal([]byte(stdout), &repeat); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(filepath.Join(root, ".goalrail", "project.json"))
	if !bytes.Equal(before, after) || repeat.ProjectID != report.ProjectID || repeat.CommitRequired {
		t.Fatal("repeated init changed project identity")
	}
}

func TestInitValidatesFlagsAndForeignSchemaBeforeWriting(t *testing.T) {
	root := scratchRepository(t)
	if _, _, err := runCommand(t, "init", "--repo", root, "--scaffold", "clade-code"); err == nil {
		t.Fatal("invalid scaffold was accepted")
	}
	if _, err := os.Stat(filepath.Join(root, ".goalrail")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("invalid scaffold wrote project content")
	}
	writeFile(t, filepath.Join(root, "openspec", "config.yaml"), "schema: owner-flow\nowner: keep\n")
	if _, _, err := runCommand(t, "init", "--repo", root); !errors.Is(err, harness.ErrForeignSchema) {
		t.Fatalf("foreign schema error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".goalrail")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("foreign schema refusal wrote project content")
	}
}

func TestMigrateIsExplicitAndRequiresExactLegacyEvidence(t *testing.T) {
	root := scratchRepository(t)
	if _, _, err := runCommand(t, "migrate", "--repo", root); !errors.Is(err, projectstate.ErrMigrationEvidence) {
		t.Fatalf("migration without evidence error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".goalrail")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("refused migration wrote project content")
	}
	if _, _, err := runCommand(t, "migrate", "--help"); err != nil {
		t.Fatalf("migrate help: %v", err)
	}
}

func TestVersionReportsBinaryAndOverlay(t *testing.T) {
	stdout, _, err := runCommand(t, "version")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, `"version"`) || !strings.Contains(stdout, "sha256:") {
		t.Fatalf("version output omits overlay identity: %s", stdout)
	}
}

func TestUpdateReconcilesBothProjectCanonAndOverlay(t *testing.T) {
	root := scratchRepository(t)
	if _, _, err := runCommand(t, "init", "--repo", root); err != nil {
		t.Fatal(err)
	}
	stdout, _, err := runCommand(t, "update", "--repo", root, "--state-dir", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var report repositoryUpdateReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatal(err)
	}
	if !report.Verified || !report.AlreadyCurrent || !report.Project.Verified || !report.Overlay.Verified {
		t.Fatalf("composite update report = %#v", report)
	}
}

func TestUpdateRefusesUnmanagedRepositoryWithoutWritingHarness(t *testing.T) {
	root := scratchRepository(t)
	writeFile(t, filepath.Join(root, "README.md"), "owner bytes\n")
	before := gitOutput(t, root, "status", "--short", "--untracked-files=all")
	if _, _, err := runCommand(t, "update", "--repo", root, "--state-dir", t.TempDir()); !errors.Is(err, projectstate.ErrClaimNotManaged) {
		t.Fatalf("unmanaged update error = %v", err)
	}
	after := gitOutput(t, root, "status", "--short", "--untracked-files=all")
	if before != after {
		t.Fatalf("unmanaged update changed the worktree:\nbefore=%s\nafter=%s", before, after)
	}
	for _, path := range []string{".goalrail", "openspec"} {
		if _, err := os.Lstat(filepath.Join(root, path)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("unmanaged update created %s: %v", path, err)
		}
	}
}

func TestInitAndMigrateRejectRepositoriesWithoutWorktrees(t *testing.T) {
	bare := filepath.Join(t.TempDir(), "bare.git")
	command := exec.Command("git", "init", "-q", "--bare", bare)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git init --bare: %v\n%s", err, output)
	}
	for _, commandName := range []string{"init", "migrate"} {
		if _, _, err := runCommand(t, commandName, "--repo", bare); !errors.Is(err, projectstate.ErrNoWorktree) {
			t.Fatalf("%s bare repository error = %v", commandName, err)
		}
	}
}
