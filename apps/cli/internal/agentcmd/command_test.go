package agentcmd

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

	"github.com/heurema/goalrail/apps/cli/internal/exitcode"
	"github.com/heurema/goalrail/apps/cli/internal/projectconfig"
	"github.com/heurema/goalrail/apps/cli/internal/term"
)

func TestInstallCreatesAgentPackFiles(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	writeProjectConfigFixture(t, repoDir)

	output, err := runInstallJSON(t, repoDir, "--format", "json")
	if err != nil {
		t.Fatalf("Run(agent install) error = %v", err)
	}
	if output.Status != "installed" {
		t.Fatalf("status = %q, want installed", output.Status)
	}
	assertFileContains(t, filepath.Join(repoDir, agentGuideRelativePath), "Use the Goalrail CLI as the machine interface")
	assertFileContains(t, filepath.Join(repoDir, agentGuideRelativePath), "This Agent Pack is provider-neutral")
	assertFileContains(t, filepath.Join(repoDir, agentGuideRelativePath), "next_action.available=false")
	assertFileContains(t, filepath.Join(repoDir, agentGuideRelativePath), "next_action.available=true")
	assertFileContains(t, filepath.Join(repoDir, agentCommandsRelativePath), `"version": "goalrail.agent.v0"`)
	assertFileContains(t, filepath.Join(repoDir, agentCommandsRelativePath), `"goalrail work start --title <title> --body-file - --format json"`)
	assertFileContains(t, filepath.Join(repoDir, agentCommandsRelativePath), `"goalrail work continue --goal-id <goal_id> --format json"`)
	assertFileContains(t, filepath.Join(repoDir, agentCommandsRelativePath), `"do_not_call_unavailable_next_actions": true`)
	assertFileContains(t, filepath.Join(repoDir, rootAgentShimRelativePath), ".goalrail/agent/GOALRAIL.md")
	assertProviderFilesMissing(t, repoDir)
}

func TestInstallFailsWithoutProjectConfig(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	_, err := runInstallJSON(t, repoDir, "--format", "json")
	if err == nil {
		t.Fatal("Run(agent install) error = nil, want missing marker")
	}
	if got := exitcode.ForError(err); got != exitcode.Usage {
		t.Fatalf("exit code = %d, want usage", got)
	}
	if !strings.Contains(err.Error(), "run goalrail init first") {
		t.Fatalf("error = %q, want init hint", err.Error())
	}
}

func TestInstallIsIdempotentWhenContentMatches(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	writeProjectConfigFixture(t, repoDir)
	if _, err := runInstallJSON(t, repoDir, "--format", "json"); err != nil {
		t.Fatalf("first Run(agent install) error = %v", err)
	}

	output, err := runInstallJSON(t, repoDir, "--format", "json")
	if err != nil {
		t.Fatalf("second Run(agent install) error = %v", err)
	}
	if output.Status != projectconfig.StatusUnchanged {
		t.Fatalf("status = %q, want unchanged", output.Status)
	}
	if output.Files[agentGuideRelativePath] != projectconfig.StatusUnchanged || output.Files[agentCommandsRelativePath] != projectconfig.StatusUnchanged || output.Files[rootAgentShimRelativePath] != projectconfig.StatusUnchanged {
		t.Fatalf("file statuses = %#v, want unchanged", output.Files)
	}
}

func TestInstallFailsOnChangedFileUnlessForce(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	writeProjectConfigFixture(t, repoDir)
	if _, err := runInstallJSON(t, repoDir, "--format", "json"); err != nil {
		t.Fatalf("first Run(agent install) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, agentGuideRelativePath), []byte("custom\n"), 0o644); err != nil {
		t.Fatalf("mutate agent guide: %v", err)
	}

	_, err := runInstallJSON(t, repoDir, "--format", "json")
	if err == nil {
		t.Fatal("Run(agent install) error = nil, want changed file conflict")
	}
	if got := exitcode.ForError(err); got != exitcode.Validation {
		t.Fatalf("exit code = %d, want validation", got)
	}

	output, err := runInstallJSON(t, repoDir, "--format", "json", "--force")
	if err != nil {
		t.Fatalf("Run(agent install --force) error = %v", err)
	}
	if output.Status != projectconfig.StatusUpdated {
		t.Fatalf("status = %q, want updated", output.Status)
	}
	assertFileContains(t, filepath.Join(repoDir, agentGuideRelativePath), "Goalrail Agent Pack v0")
	assertFileContains(t, filepath.Join(repoDir, rootAgentShimRelativePath), ".goalrail/agent/GOALRAIL.md")
}

func TestInstallSkipsExistingRootAgentsFile(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	writeProjectConfigFixture(t, repoDir)
	customAgents := "custom agent instructions\n"
	if err := os.WriteFile(filepath.Join(repoDir, rootAgentShimRelativePath), []byte(customAgents), 0o644); err != nil {
		t.Fatalf("write custom AGENTS.md: %v", err)
	}

	output, err := runInstallJSON(t, repoDir, "--format", "json")
	if err != nil {
		t.Fatalf("Run(agent install) error = %v", err)
	}
	if output.Files[rootAgentShimRelativePath] != statusSkippedManualPatch {
		t.Fatalf("AGENTS.md status = %q, want skipped", output.Files[rootAgentShimRelativePath])
	}
	assertFileEquals(t, filepath.Join(repoDir, rootAgentShimRelativePath), customAgents)
}

func TestInstallForceDoesNotOverwriteExistingRootAgentsFile(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	writeProjectConfigFixture(t, repoDir)
	if _, err := runInstallJSON(t, repoDir, "--format", "json"); err != nil {
		t.Fatalf("first Run(agent install) error = %v", err)
	}
	customAgents := "custom agent instructions\n"
	if err := os.WriteFile(filepath.Join(repoDir, rootAgentShimRelativePath), []byte(customAgents), 0o644); err != nil {
		t.Fatalf("write custom AGENTS.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, agentCommandsRelativePath), []byte("custom\n"), 0o644); err != nil {
		t.Fatalf("mutate commands: %v", err)
	}

	output, err := runInstallJSON(t, repoDir, "--format", "json", "--force")
	if err != nil {
		t.Fatalf("Run(agent install --force) error = %v", err)
	}
	if output.Files[agentCommandsRelativePath] != projectconfig.StatusUpdated {
		t.Fatalf("commands status = %q, want updated", output.Files[agentCommandsRelativePath])
	}
	if output.Files[rootAgentShimRelativePath] != statusSkippedManualPatch {
		t.Fatalf("AGENTS.md status = %q, want skipped", output.Files[rootAgentShimRelativePath])
	}
	assertFileContains(t, filepath.Join(repoDir, agentCommandsRelativePath), `"do_not_call_unavailable_next_actions": true`)
	assertFileEquals(t, filepath.Join(repoDir, rootAgentShimRelativePath), customAgents)
}

func TestInstallJSONReturnsInstalledPathsAndStatus(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	writeProjectConfigFixture(t, repoDir)

	output, err := runInstallJSON(t, repoDir, "--format", "json")
	if err != nil {
		t.Fatalf("Run(agent install --format json) error = %v", err)
	}
	if len(output.Paths) != 3 || output.Paths[0] != agentGuideRelativePath || output.Paths[1] != agentCommandsRelativePath || output.Paths[2] != rootAgentShimRelativePath {
		t.Fatalf("paths = %#v, want agent pack paths", output.Paths)
	}
	if output.Files[agentGuideRelativePath] != projectconfig.StatusWritten || output.Files[agentCommandsRelativePath] != projectconfig.StatusWritten || output.Files[rootAgentShimRelativePath] != projectconfig.StatusWritten {
		t.Fatalf("file statuses = %#v, want written", output.Files)
	}
}

func TestInstallRejectsSymlinkAgentDirectory(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	writeProjectConfigFixture(t, repoDir)
	outsideDir := t.TempDir()
	requireSymlink(t, outsideDir, filepath.Join(repoDir, agentDirRelativePath))

	_, err := runInstallJSON(t, repoDir, "--format", "json", "--force")
	if err == nil {
		t.Fatal("Run(agent install) error = nil, want symlink validation")
	}
	if got := exitcode.ForError(err); got != exitcode.Validation {
		t.Fatalf("exit code = %d, want validation", got)
	}
	if !strings.Contains(err.Error(), agentDirRelativePath+" must not be a symlink") {
		t.Fatalf("error = %q, want symlink directory error", err.Error())
	}
	if _, err := os.Stat(filepath.Join(outsideDir, "GOALRAIL.md")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("outside GOALRAIL.md stat error = %v, want not exist", err)
	}
}

func TestInstallRejectsSymlinkAgentFileWithForce(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	writeProjectConfigFixture(t, repoDir)
	agentDir := filepath.Join(repoDir, agentDirRelativePath)
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatalf("create agent dir: %v", err)
	}
	outsideFile := filepath.Join(t.TempDir(), "outside.md")
	if err := os.WriteFile(outsideFile, []byte("outside\n"), 0o644); err != nil {
		t.Fatalf("write outside file: %v", err)
	}
	requireSymlink(t, outsideFile, filepath.Join(repoDir, agentGuideRelativePath))

	_, err := runInstallJSON(t, repoDir, "--format", "json", "--force")
	if err == nil {
		t.Fatal("Run(agent install) error = nil, want symlink validation")
	}
	if got := exitcode.ForError(err); got != exitcode.Validation {
		t.Fatalf("exit code = %d, want validation", got)
	}
	if !strings.Contains(err.Error(), agentGuideRelativePath+" must not be a symlink") {
		t.Fatalf("error = %q, want symlink file error", err.Error())
	}
	raw, err := os.ReadFile(outsideFile)
	if err != nil {
		t.Fatalf("read outside file: %v", err)
	}
	if string(raw) != "outside\n" {
		t.Fatalf("outside file = %q, want unchanged", string(raw))
	}
}

func runInstallJSON(t *testing.T, workDir string, args ...string) (InstallOutput, error) {
	t.Helper()

	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), term.New(&stdout, &stderr), workDir, append([]string{"install"}, args...))
	if err != nil {
		return InstallOutput{}, err
	}
	var output InstallOutput
	decoder := json.NewDecoder(&stdout)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&output); err != nil {
		t.Fatalf("decode agent install JSON %q: %v", stdout.String(), err)
	}
	return output, nil
}

func setupGitRepo(t *testing.T) string {
	t.Helper()

	repoDir := t.TempDir()
	runGit(t, repoDir, "init")
	runGit(t, repoDir, "-c", "user.name=Goalrail Test", "-c", "user.email=goalrail@example.test", "commit", "--allow-empty", "-m", "initial")
	return repoDir
}

func writeProjectConfigFixture(t *testing.T, repoDir string) {
	t.Helper()

	configPath := filepath.Join(repoDir, projectconfig.RelativePath)
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("create .goalrail dir: %v", err)
	}
	content := projectconfig.RenderYAML(projectconfig.Config{
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
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write project config: %v", err)
	}
}

func assertFileContains(t *testing.T, path string, want string) {
	t.Helper()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !strings.Contains(string(raw), want) {
		t.Fatalf("%s = %q, want containing %q", path, string(raw), want)
	}
}

func assertFileEquals(t *testing.T, path string, want string) {
	t.Helper()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(raw) != want {
		t.Fatalf("%s = %q, want %q", path, string(raw), want)
	}
}

func assertProviderFilesMissing(t *testing.T, repoDir string) {
	t.Helper()

	for _, relativePath := range []string{".codex", "CLAUDE.md", "GEMINI.md", ".cursor"} {
		_, err := os.Stat(filepath.Join(repoDir, relativePath))
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("%s stat error = %v, want not exist", relativePath, err)
		}
	}
}

func requireGit(t *testing.T) {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}
}

func requireSymlink(t *testing.T, oldname string, newname string) {
	t.Helper()

	if err := os.Symlink(oldname, newname); err != nil {
		t.Skipf("symlink not available: %v", err)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", append([]string{"-C", filepath.Clean(dir)}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, output)
	}
}
