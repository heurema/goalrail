package review

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// repository builds a scratch checkout with one commit on main and returns its
// root. Everything here needs a real repository, because the digest that makes
// staleness meaningful is git's own rendering of a range.
func repository(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, arguments := range [][]string{
		{"init", "--initial-branch=main"},
		{"config", "user.email", "t@t"},
		{"config", "user.name", "t"},
	} {
		command := exec.Command("git", append([]string{"-C", root}, arguments...)...)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v (%s)", arguments, err, output)
		}
	}
	write(t, root, "README.md", "start\n")
	commit(t, root, "first")
	return root
}

func write(t *testing.T, root, name, contents string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func commit(t *testing.T, root, message string) {
	t.Helper()
	for _, arguments := range [][]string{{"add", "-A"}, {"commit", "-q", "-m", message}} {
		command := exec.Command("git", append([]string{"-C", root}, arguments...)...)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v (%s)", arguments, err, output)
		}
	}
}

func branch(t *testing.T, root, name string) {
	t.Helper()
	command := exec.Command("git", "-C", root, "checkout", "-q", "-b", name)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git checkout -b %s: %v (%s)", name, err, output)
	}
}

func TestDiffDigestDescribesTheRangeAndNothingElse(t *testing.T) {
	root := repository(t)
	base, err := Resolve(root, "main")
	if err != nil {
		t.Fatal(err)
	}
	branch(t, root, "work")
	write(t, root, "added.txt", "one\n")
	commit(t, root, "add one")

	head, err := Resolve(root, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	first, err := DiffDigest(root, base, head)
	if err != nil {
		t.Fatal(err)
	}
	// Reproducible: the same range digests identically when asked again.
	again, err := DiffDigest(root, base, head)
	if err != nil || again != first {
		t.Fatalf("the same range digested differently: %s then %s (%v)", first, again, err)
	}

	// A different range never collides with it.
	write(t, root, "added.txt", "two\n")
	commit(t, root, "change one")
	moved, err := Resolve(root, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	after, err := DiffDigest(root, base, moved)
	if err != nil {
		t.Fatal(err)
	}
	if after == first {
		t.Fatal("a changed range produced the same digest")
	}
}

// The three-dot range is the change the branch introduces since it diverged.
// A base that moved ahead must not drag unrelated work into the digest, or a
// review would go stale because somebody else committed to main.
func TestDiffDigestIgnoresWorkThatLandedOnTheBaseAfterwards(t *testing.T) {
	root := repository(t)
	base, err := Resolve(root, "main")
	if err != nil {
		t.Fatal(err)
	}
	branch(t, root, "work")
	write(t, root, "mine.txt", "mine\n")
	commit(t, root, "mine")
	head, err := Resolve(root, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	before, err := DiffDigest(root, base, head)
	if err != nil {
		t.Fatal(err)
	}

	// Someone else lands work on main.
	if output, err := exec.Command("git", "-C", root, "checkout", "-q", "main").CombinedOutput(); err != nil {
		t.Fatalf("checkout main: %v (%s)", err, output)
	}
	write(t, root, "theirs.txt", "theirs\n")
	commit(t, root, "theirs")
	movedBase, err := Resolve(root, "main")
	if err != nil {
		t.Fatal(err)
	}

	after, err := DiffDigest(root, movedBase, head)
	if err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Fatal("unrelated work on the base changed the branch's digest")
	}
}

func TestReceiptRoundTripsAndStaysOutOfTheRepository(t *testing.T) {
	root := repository(t)
	stateRoot := t.TempDir()
	branch(t, root, "work")
	write(t, root, "added.txt", "one\n")
	commit(t, root, "add one")

	base, _ := Resolve(root, "main")
	head, _ := Resolve(root, "HEAD")
	diff, err := DiffDigest(root, base, head)
	if err != nil {
		t.Fatal(err)
	}
	report := "findings go here\nREVIEW-VERDICT: nothing-material\n"
	written := Receipt{
		Schema:       ReceiptSchema,
		Repository:   root,
		Branch:       "work",
		BaseRef:      "main",
		BaseCommit:   base,
		HeadCommit:   head,
		DiffSHA256:   diff,
		Reviewer:     "codex",
		Author:       "claude-code",
		ReviewedAt:   time.Unix(0, 0).UTC().Format(time.RFC3339),
		Report:       report,
		ReportSHA256: digest([]byte(report)),
	}
	path, err := WriteReceipt(stateRoot, written)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(path, stateRoot) {
		t.Fatalf("the receipt was written outside the state root: %s", path)
	}

	// The repository is untouched: a receipt is evidence about one clone, and a
	// receipt someone else received by pulling would assert a review of a diff
	// they may not have.
	status, err := exec.Command("git", "-C", root, "status", "--porcelain").CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(status)) != "" {
		t.Fatalf("the review left repository changes: %s", status)
	}

	read, found, err := ReadReceipt(stateRoot, root, "work")
	if err != nil || !found {
		t.Fatalf("reading back reported found=%v (%v)", found, err)
	}
	if read.Report != report {
		t.Fatal("the stored report is not byte-identical")
	}
	if read.ReportSHA256 != digest([]byte(read.Report)) {
		t.Fatal("the report digest does not describe the stored report")
	}
	// Recomputing from the receipt's own base and head reproduces its digest.
	recomputed, err := DiffDigest(read.Repository, read.BaseCommit, read.HeadCommit)
	if err != nil || recomputed != read.DiffSHA256 {
		t.Fatalf("the receipt does not describe its own range: %s vs %s (%v)", recomputed, read.DiffSHA256, err)
	}
}

func TestStatusMovesWithTheBranchAndNeedsNobody(t *testing.T) {
	root := repository(t)
	stateRoot := t.TempDir()
	branch(t, root, "work")
	write(t, root, "added.txt", "one\n")
	commit(t, root, "add one")

	state, _, err := Status(stateRoot, root, "work")
	if err != nil || state != StateAbsent {
		t.Fatalf("an unreviewed branch reported %q (%v)", state, err)
	}

	base, _ := Resolve(root, "main")
	head, _ := Resolve(root, "HEAD")
	diff, _ := DiffDigest(root, base, head)
	if _, err := WriteReceipt(stateRoot, Receipt{
		Schema: ReceiptSchema, Repository: root, Branch: "work",
		BaseRef: "main", BaseCommit: base, HeadCommit: head, DiffSHA256: diff,
		Reviewer: "codex", Author: "claude-code",
	}); err != nil {
		t.Fatal(err)
	}

	state, _, err = Status(stateRoot, root, "work")
	if err != nil || state != StateCurrent {
		t.Fatalf("a fresh review reported %q (%v)", state, err)
	}

	// A commit makes it stale by itself. No flag, no dismissal.
	write(t, root, "added.txt", "two\n")
	commit(t, root, "change one")
	state, _, err = Status(stateRoot, root, "work")
	if err != nil || state != StateStale {
		t.Fatalf("a branch that moved reported %q (%v)", state, err)
	}

	// And a fresh review makes it current again, by itself.
	movedHead, _ := Resolve(root, "HEAD")
	movedDiff, _ := DiffDigest(root, base, movedHead)
	if _, err := WriteReceipt(stateRoot, Receipt{
		Schema: ReceiptSchema, Repository: root, Branch: "work",
		BaseRef: "main", BaseCommit: base, HeadCommit: movedHead, DiffSHA256: movedDiff,
		Reviewer: "codex", Author: "claude-code",
	}); err != nil {
		t.Fatal(err)
	}
	state, _, err = Status(stateRoot, root, "work")
	if err != nil || state != StateCurrent {
		t.Fatalf("a re-review reported %q (%v)", state, err)
	}
}

// The same branch name in two clones is two different pieces of work.
func TestReceiptsAreKeyedByRepositoryAndBranchTogether(t *testing.T) {
	stateRoot := t.TempDir()
	first := receiptPath(stateRoot, "/a", "work")
	second := receiptPath(stateRoot, "/b", "work")
	third := receiptPath(stateRoot, "/a", "other")
	if first == second || first == third || second == third {
		t.Fatalf("receipt paths collide: %s %s %s", first, second, third)
	}
}
