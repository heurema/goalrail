package gitctx

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

var ErrNotGitRepository = errors.New("not inside a git repository")

type Context struct {
	GitRoot            string
	RemoteName         string
	RemoteURL          string
	WorkflowBaseBranch string
	HeadSHA            string
	Warnings           []string
}

func Discover(ctx context.Context, workDir string) (Context, error) {
	if workDir == "" {
		workDir = "."
	}

	gitRoot, err := gitOutput(ctx, workDir, "rev-parse", "--show-toplevel")
	if err != nil {
		return Context{}, ErrNotGitRepository
	}

	out := Context{
		GitRoot:  gitRoot,
		Warnings: []string{},
	}

	if remoteURL, err := gitOutput(ctx, gitRoot, "config", "--get", "remote.origin.url"); err == nil && remoteURL != "" {
		out.RemoteName = "origin"
		out.RemoteURL = remoteURL
	}

	if headSHA, err := gitOutput(ctx, gitRoot, "rev-parse", "--verify", "HEAD"); err == nil {
		out.HeadSHA = headSHA
	}

	if baseBranch, ok := detectWorkflowBaseBranch(ctx, gitRoot); ok {
		out.WorkflowBaseBranch = baseBranch
	} else {
		out.Warnings = append(out.Warnings, "workflow base branch could not be detected from local origin metadata")
	}

	return out, nil
}

func detectWorkflowBaseBranch(ctx context.Context, gitRoot string) (string, bool) {
	originHead, err := gitOutput(ctx, gitRoot, "symbolic-ref", "--quiet", "--short", "refs/remotes/origin/HEAD")
	if err == nil && originHead != "" {
		branch := strings.TrimPrefix(originHead, "origin/")
		if branch != originHead && strings.TrimSpace(branch) != "" {
			return branch, true
		}
	}

	for _, branch := range []string{"main", "master"} {
		if err := gitRun(ctx, gitRoot, "show-ref", "--verify", "--quiet", "refs/remotes/origin/"+branch); err == nil {
			return branch, true
		}
	}
	return "", false
}

func gitOutput(ctx context.Context, workDir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", workDir}, args...)...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func gitRun(ctx context.Context, workDir string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", workDir}, args...)...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return nil
}
