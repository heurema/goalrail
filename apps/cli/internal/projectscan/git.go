package projectscan

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var ErrNotGitRepository = errors.New("not inside a git repository")

func DiscoverGit(ctx context.Context, workDir string) (GitFacts, error) {
	if workDir == "" {
		workDir = "."
	}

	root, err := gitOutput(ctx, workDir, "rev-parse", "--show-toplevel")
	if err != nil {
		return GitFacts{}, ErrNotGitRepository
	}
	canonicalRoot, err := canonicalizePath(root)
	if err != nil {
		return GitFacts{}, err
	}

	gitDir, err := gitOutput(ctx, canonicalRoot, "rev-parse", "--git-dir")
	if err != nil {
		return GitFacts{}, fmt.Errorf("git rev-parse --git-dir: %w", err)
	}
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(canonicalRoot, gitDir)
	}
	canonicalGitDir, err := canonicalizePath(gitDir)
	if err != nil {
		return GitFacts{}, err
	}

	headSHA, err := gitOutput(ctx, canonicalRoot, "rev-parse", "--verify", "HEAD")
	if err != nil {
		return GitFacts{}, fmt.Errorf("git rev-parse --verify HEAD: %w", err)
	}

	branch, detached := "", true
	if branchName, err := gitOutput(ctx, canonicalRoot, "symbolic-ref", "--quiet", "--short", "HEAD"); err == nil && branchName != "" {
		branch = branchName
		detached = false
	}

	shallow := false
	if value, err := gitOutput(ctx, canonicalRoot, "rev-parse", "--is-shallow-repository"); err == nil {
		shallow = strings.EqualFold(strings.TrimSpace(value), "true")
	}

	sparse := false
	if value, err := gitOutput(ctx, canonicalRoot, "config", "--bool", "core.sparseCheckout"); err == nil {
		sparse = strings.EqualFold(strings.TrimSpace(value), "true")
	}
	if !sparse {
		if _, err := os.Stat(filepath.Join(canonicalGitDir, "info", "sparse-checkout")); err == nil {
			sparse = true
		}
	}

	submodules := false
	if _, err := os.Stat(filepath.Join(canonicalRoot, ".gitmodules")); err == nil {
		submodules = true
	}

	return GitFacts{
		CanonicalRepoRoot: canonicalRoot,
		CanonicalGitDir:   canonicalGitDir,
		HeadSHA:           headSHA,
		Branch:            branch,
		Detached:          detached,
		ShallowRepository: shallow,
		SparseCheckout:    sparse,
		SubmodulesPresent: submodules,
	}, nil
}

func gitStatusPorcelainV2(ctx context.Context, canonicalRoot string) (string, error) {
	return gitOutput(ctx, canonicalRoot, "--no-optional-locks", "status", "--porcelain=v2", "--branch", "--untracked-files=no", "--ignored=no")
}

func gitTrackedPaths(ctx context.Context, canonicalRoot string) ([]string, error) {
	raw, err := gitOutputBytes(ctx, canonicalRoot, "ls-tree", "-r", "-z", "--name-only", "HEAD")
	if err != nil {
		return nil, fmt.Errorf("git ls-tree HEAD: %w", err)
	}
	parts := strings.Split(string(raw), "\x00")
	paths := make([]string, 0, len(parts))
	for _, part := range parts {
		path := normalizeRelativePath(part)
		if path == "" {
			continue
		}
		paths = append(paths, path)
	}
	sortStrings(paths)
	return paths, nil
}

func gitBlobSize(ctx context.Context, canonicalRoot string, relativePath string) (int, error) {
	out, err := gitOutput(ctx, canonicalRoot, "cat-file", "-s", "HEAD:"+relativePath)
	if err != nil {
		return 0, err
	}
	var size int
	if _, err := fmt.Sscanf(out, "%d", &size); err != nil {
		return 0, err
	}
	return size, nil
}

func gitBlob(ctx context.Context, canonicalRoot string, relativePath string) ([]byte, error) {
	return gitOutputBytes(ctx, canonicalRoot, "show", "HEAD:"+relativePath)
}

func gitOutput(ctx context.Context, workDir string, args ...string) (string, error) {
	raw, err := gitOutputBytes(ctx, workDir, args...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(raw)), nil
}

func gitOutputBytes(ctx context.Context, workDir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", workDir}, args...)...)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return output, nil
}

func canonicalizePath(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	canonical, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", err
	}
	return filepath.Clean(canonical), nil
}
