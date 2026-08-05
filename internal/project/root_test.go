package project

import (
	"context"
	"errors"
	"os"
	"path/filepath"
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
