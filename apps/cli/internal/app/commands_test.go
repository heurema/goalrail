package app

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/heurema/goalrail/apps/cli/internal/clienv"
	"github.com/heurema/goalrail/apps/cli/internal/projectconfig"
)

func TestRootCommandVersionUsesCobraArgsAndWriters(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	cmd := NewRootCommand(clienv.Env{WorkDir: "."})
	cmd.SetArgs([]string{"version"})
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext(version) error = %v", err)
	}

	if got := strings.TrimSpace(stdout.String()); got != Version {
		t.Fatalf("stdout = %q, want %q", got, Version)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestRootCommandNestedHelpUsesCobraArgsAndWriters(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	cmd := NewRootCommand(clienv.Env{WorkDir: "."})
	cmd.SetArgs([]string{"readiness", "scan", "--help"})
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext(readiness scan --help) error = %v", err)
	}

	want := "Usage: goalrail readiness scan --path <path> [--format text|json]"
	if got := stdout.String(); !strings.Contains(got, want) {
		t.Fatalf("stdout = %q, want usage containing %q", got, want)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestRootCommandWorkStartHelpUsesCobraArgsAndWriters(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	cmd := NewRootCommand(clienv.Env{WorkDir: "."})
	cmd.SetArgs([]string{"work", "start", "--help"})
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext(work start --help) error = %v", err)
	}

	want := "Usage: goalrail work start --title <title> [--body <body> | --body-file <path|->] [--format text|json]"
	if got := stdout.String(); !strings.Contains(got, want) {
		t.Fatalf("stdout = %q, want usage containing %q", got, want)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestRootCommandWorkContinueHelpUsesCobraArgsAndWriters(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	cmd := NewRootCommand(clienv.Env{WorkDir: "."})
	cmd.SetArgs([]string{"work", "continue", "--help"})
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext(work continue --help) error = %v", err)
	}

	want := "Usage: goalrail work continue --goal-id <goal_id> [--format text|json]"
	if got := stdout.String(); !strings.Contains(got, want) {
		t.Fatalf("stdout = %q, want usage containing %q", got, want)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestRootCommandWorkAnswerHelpUsesCobraArgsAndWriters(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	cmd := NewRootCommand(clienv.Env{WorkDir: "."})
	cmd.SetArgs([]string{"work", "answer", "--help"})
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext(work answer --help) error = %v", err)
	}

	want := "Usage: goalrail work answer --clarification-request-id <id> --answers-file <path|-> [--format text|json]"
	if got := stdout.String(); !strings.Contains(got, want) {
		t.Fatalf("stdout = %q, want usage containing %q", got, want)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestRootCommandContractUpdateHelpUsesCobraArgsAndWriters(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	cmd := NewRootCommand(clienv.Env{WorkDir: "."})
	cmd.SetArgs([]string{"contract", "update", "--help"})
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext(contract update --help) error = %v", err)
	}

	want := "Usage: goalrail contract update --contract-id <contract_id> --fields-file <path|-> [--format text|json]"
	if got := stdout.String(); !strings.Contains(got, want) {
		t.Fatalf("stdout = %q, want usage containing %q", got, want)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestRootCommandProjectStatusHelpUsesCobraArgsAndWriters(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	cmd := NewRootCommand(clienv.Env{WorkDir: "."})
	cmd.SetArgs([]string{"project", "status", "--help"})
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext(project status --help) error = %v", err)
	}

	want := "Usage: goalrail project status [--format text|json]"
	if got := stdout.String(); !strings.Contains(got, want) {
		t.Fatalf("stdout = %q, want usage containing %q", got, want)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestRootCommandAgentInstallHelpUsesCobraArgsAndWriters(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	cmd := NewRootCommand(clienv.Env{WorkDir: "."})
	cmd.SetArgs([]string{"agent", "install", "--help"})
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext(agent install --help) error = %v", err)
	}

	want := "Usage: goalrail agent install [--format text|json] [--force]"
	if got := stdout.String(); !strings.Contains(got, want) {
		t.Fatalf("stdout = %q, want usage containing %q", got, want)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestRootCommandAgentInstallCreatesPackFromCobraPath(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	writeProjectConfigFixture(t, repoDir)

	var stdout, stderr bytes.Buffer
	cmd := NewRootCommand(clienv.Env{WorkDir: repoDir})
	cmd.SetArgs([]string{"agent", "install", "--format", "json"})
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext(agent install --format json) error = %v", err)
	}

	var output struct {
		Status string            `json:"status"`
		Paths  []string          `json:"paths"`
		Files  map[string]string `json:"files"`
	}
	decoder := json.NewDecoder(&stdout)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&output); err != nil {
		t.Fatalf("decode stdout %q: %v", stdout.String(), err)
	}
	if output.Status != "installed" {
		t.Fatalf("status = %q, want installed", output.Status)
	}
	for _, relativePath := range []string{".goalrail/agent/GOALRAIL.md", ".goalrail/agent/commands.json"} {
		if _, err := os.Stat(filepath.Join(repoDir, relativePath)); err != nil {
			t.Fatalf("stat %s: %v", relativePath, err)
		}
		if output.Files[relativePath] != projectconfig.StatusWritten {
			t.Fatalf("%s status = %q, want written", relativePath, output.Files[relativePath])
		}
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestRootCommandLoginRequiresServerURL(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	cmd := NewRootCommand(clienv.Env{WorkDir: "."})
	cmd.SetArgs([]string{"login"})
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("ExecuteContext(login) error = nil, want usage error")
	}
	if !strings.Contains(err.Error(), "missing required server_url") {
		t.Fatalf("error = %v, want missing server_url", err)
	}
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

func requireGit(t *testing.T) {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
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
