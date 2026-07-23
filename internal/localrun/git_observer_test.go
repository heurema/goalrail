package localrun

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitObserverTracksOnlyChangesAfterDirtyBaseline(t *testing.T) {
	root, head := testGitRepository(t)
	dirty := filepath.Join(root, "inside.txt")
	if err := os.WriteFile(dirty, []byte("before run\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	observer := GitObserver{}
	resolvedRoot, resolvedHead, err := observer.ResolveRepository(context.Background(), root, head)
	if err != nil {
		t.Fatal(err)
	}
	expectedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	if resolvedRoot != expectedRoot || resolvedHead != head {
		t.Fatalf("resolved %q %q, want %q %q", resolvedRoot, resolvedHead, expectedRoot, head)
	}
	baseline, err := observer.Observe(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	unchanged, err := observer.Observe(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if delta := CompareWorktrees(baseline, unchanged, []string{"inside.txt"}); len(delta.ChangedPaths) != 0 {
		t.Fatalf("pre-existing dirty state was attributed to run: %v", delta.ChangedPaths)
	}

	if err := os.WriteFile(dirty, []byte("after run\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "outside.txt"), []byte("scope violation\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	terminal, err := observer.Observe(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	delta := CompareWorktrees(baseline, terminal, []string{"inside.txt"})
	if strings.Join(delta.ChangedPaths, ",") != "inside.txt,outside.txt" {
		t.Fatalf("changed paths = %v", delta.ChangedPaths)
	}
	if strings.Join(delta.ScopeViolations, ",") != "outside.txt" {
		t.Fatalf("scope violations = %v", delta.ScopeViolations)
	}
}

func TestGitObserverRejectsWrongRevisionAndUnboundedDirtySet(t *testing.T) {
	root, _ := testGitRepository(t)
	observer := GitObserver{MaxPaths: 1}
	if _, _, err := observer.ResolveRepository(context.Background(), root, strings.Repeat("0", 40)); err == nil {
		t.Fatal("expected missing revision rejection")
	}
	if err := os.WriteFile(filepath.Join(root, "one.txt"), []byte("one"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "two.txt"), []byte("two"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := observer.Observe(context.Background(), root); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("expected dirty-set bound, got %v", err)
	}
}

func testGitRepository(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	runGit(t, root, "init", "-q")
	runGit(t, root, "config", "user.name", "Goalrail Test")
	runGit(t, root, "config", "user.email", "goalrail@example.invalid")
	if err := os.WriteFile(filepath.Join(root, "tracked.txt"), []byte("tracked\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, root, "add", "tracked.txt")
	runGit(t, root, "commit", "-qm", "test fixture")
	head := strings.TrimSpace(runGit(t, root, "rev-parse", "HEAD"))
	return root, head
}

func runGit(t *testing.T, root string, arguments ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", root}, arguments...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", arguments, err, output)
	}
	return string(output)
}
