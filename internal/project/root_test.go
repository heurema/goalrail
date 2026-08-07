package project

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveWorktreeRootHandlesPrimaryLinkedSubdirectoryAndIgnoredGitSelection(t *testing.T) {
	ctx := context.Background()
	base := t.TempDir()
	primary := initGitRepository(t, filepath.Join(base, "primary"))
	subdirectory := filepath.Join(primary, "internal", "nested")
	if err := os.MkdirAll(subdirectory, 0o755); err != nil {
		t.Fatal(err)
	}

	alternate := initGitRepository(t, filepath.Join(base, "alternate"))
	t.Setenv("GIT_DIR", filepath.Join(alternate, ".git"))
	t.Setenv("GIT_WORK_TREE", alternate)
	t.Setenv("GIT_INDEX_FILE", filepath.Join(alternate, ".git", "index"))
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "core.worktree")
	t.Setenv("GIT_CONFIG_VALUE_0", alternate)

	for _, start := range []string{primary, subdirectory} {
		root, err := ResolveWorktreeRoot(ctx, start)
		if err != nil {
			t.Fatal(err)
		}
		if root != primary {
			t.Fatalf("caller-selected alternate repository won: got %q, want %q", root, primary)
		}
	}

	linked := filepath.Join(base, "linked")
	runGit(t, primary, "worktree", "add", "--detach", linked, "HEAD")
	root, err := ResolveWorktreeRoot(ctx, linked)
	if err != nil {
		t.Fatal(err)
	}
	physicalLinked, err := filepath.EvalSymlinks(linked)
	if err != nil {
		t.Fatal(err)
	}
	if root != physicalLinked {
		t.Fatalf("linked worktree root mismatch: got %q, want %q", root, physicalLinked)
	}
}

func TestResolveWorktreeRootRejectsBareAndUnrelatedDirectories(t *testing.T) {
	ctx := context.Background()
	base := t.TempDir()
	bare := filepath.Join(base, "bare.git")
	if err := os.MkdirAll(bare, 0o755); err != nil {
		t.Fatal(err)
	}
	runGit(t, bare, "init", "--bare", "--quiet")
	if _, err := ResolveWorktreeRoot(ctx, bare); !errors.Is(err, ErrNoWorktree) {
		t.Fatalf("bare repository error = %v, want ErrNoWorktree", err)
	}

	unrelated := filepath.Join(base, "unrelated")
	if err := os.MkdirAll(unrelated, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveWorktreeRoot(ctx, unrelated); !errors.Is(err, ErrNotRepository) {
		t.Fatalf("unrelated directory error = %v, want ErrNotRepository", err)
	}
}

// Only Git's own verdict means "not a repository".
//
// Every other failure — Git absent above all — used to be reported as one, so a
// caller that branches on the distinction was told a working machine had no
// repository at the path, and a diagnosis would report an ordinary condition
// where its check had actually broken.
func TestOnlyGitsOwnVerdictMeansNotARepository(t *testing.T) {
	outside := t.TempDir()
	if _, err := ResolveWorktreeRoot(context.Background(), outside); !errors.Is(err, ErrNotRepository) {
		t.Fatalf("a directory outside version control gave %v, want ErrNotRepository", err)
	}

	// A PATH with no Git at all: the command cannot run, so nothing said this
	// is not a repository.
	t.Setenv("PATH", t.TempDir())
	_, err := ResolveWorktreeRoot(context.Background(), outside)
	if err == nil {
		t.Fatal("a missing Git resolved a worktree root")
	}
	if errors.Is(err, ErrNotRepository) {
		t.Fatalf("a missing Git was reported as an absent repository: %v", err)
	}
}

// A refusal Git makes after actually running is its own answer, and Git's
// sentence is not part of it.
//
// The distinction is what a caller reports: "this is not a project" is a fact
// about the directory, while "Git declined" is a fact about the attempt, and a
// disputed-ownership refusal — a shared or container-mounted checkout — is the
// second wearing the shape of the first. Carrying Git's own text on the error
// is how a caller ends up printing `fatal:` and a `git config` line to paste,
// so the text stops here.
func TestAGitRefusalIsNeitherAnAbsentRepositoryNorGitsOwnWords(t *testing.T) {
	// Disputed ownership needs a second account to stage; an unreadable format
	// version reaches the same branch — Git ran, and declined for a reason that
	// is not the path being outside a repository — without a privileged test.
	refused := t.TempDir()
	for _, arguments := range [][]string{
		{"init", "-q"},
		{"config", "core.repositoryformatversion", "99"},
	} {
		command := exec.Command("git", append([]string{"-C", refused}, arguments...)...)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", arguments, err, output)
		}
	}

	_, err := ResolveWorktreeRoot(context.Background(), refused)
	if err == nil {
		t.Fatal("a repository Git refuses resolved a worktree root")
	}
	if !errors.Is(err, ErrDiscoveryRefused) {
		t.Fatalf("a refusal gave %v, want ErrDiscoveryRefused", err)
	}
	if errors.Is(err, ErrNotRepository) {
		t.Fatalf("a refusal was reported as an absent repository: %v", err)
	}
	for _, foreign := range []string{"fatal:", "exit status", "git config"} {
		if strings.Contains(err.Error(), foreign) {
			t.Fatalf("the refusal carries Git's own words: %q", err.Error())
		}
	}
}
