package gitctx

import (
	"context"
	"os/exec"
	"testing"
)

func TestDetectWorkflowBaseBranchUsesOriginHeadBranchName(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}
	gitRoot := t.TempDir()
	runGit(t, gitRoot, "init", "-q")
	runGit(t, gitRoot, "config", "user.email", "e2e@example.com")
	runGit(t, gitRoot, "config", "user.name", "E2E")
	runGit(t, gitRoot, "commit", "--allow-empty", "-q", "-m", "init")
	runGit(t, gitRoot, "branch", "-M", "trunk")
	runGit(t, gitRoot, "update-ref", "refs/remotes/origin/trunk", "HEAD")
	runGit(t, gitRoot, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/trunk")

	got, ok := detectWorkflowBaseBranch(context.Background(), gitRoot)
	if !ok {
		t.Fatal("detectWorkflowBaseBranch() ok = false, want true")
	}
	if got != "trunk" {
		t.Fatalf("detectWorkflowBaseBranch() = %q, want trunk", got)
	}
}

func runGit(t *testing.T, workDir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", workDir}, args...)...)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
}
