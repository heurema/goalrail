package review

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/heurema/goalrail/internal/ambient"
)

// stubReviewer puts an executable of the given name early on PATH, so the real
// invocation path is exercised against a script whose behaviour the test owns.
// Stubbing the binary rather than the call is what keeps the argument
// construction, the environment stripping and the failure handling under test.
func stubReviewer(t *testing.T, name, script string) {
	t.Helper()
	directory := t.TempDir()
	path := filepath.Join(directory, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", directory+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func branchWithWork(t *testing.T) string {
	t.Helper()
	root := repository(t)
	branch(t, root, "work")
	write(t, root, "added.txt", "one\n")
	commit(t, root, "add one")
	return root
}

func TestRunRecordsWhatTheReviewerSaid(t *testing.T) {
	root := branchWithWork(t)
	stateRoot := t.TempDir()
	stubReviewer(t, "codex", `cat >/dev/null; printf 'a finding\nREVIEW-VERDICT: material-findings\n'`)

	result, err := Run(context.Background(), Input{
		RepositoryRoot: root, StateRoot: stateRoot, BaseRef: "main",
		Author: ambient.ScaffoldClaudeCode, Reviewer: ambient.ScaffoldCodex,
	})
	if err != nil {
		t.Fatalf("review failed: %v", err)
	}
	if !strings.Contains(result.Receipt.Report, "a finding") {
		t.Fatalf("the report was not stored verbatim: %q", result.Receipt.Report)
	}
	if result.Receipt.ReportSHA256 != digest([]byte(result.Receipt.Report)) {
		t.Fatal("the report digest does not describe the stored report")
	}
	if !result.InstructionsMaterialized {
		t.Fatal("the default instructions were not materialized")
	}

	// Nothing about the verdict marker is extracted: it is the reviewer's
	// statement, and reading it would be the parsing this package refuses.
	if strings.Contains(strings.ToLower(result.Receipt.Reviewer), "material") {
		t.Fatal("a field was derived from the report's content")
	}
	if _, found, err := ReadReceipt(stateRoot, root, "work"); err != nil || !found {
		t.Fatalf("no receipt was stored: found=%v (%v)", found, err)
	}
}

// A receipt for a review that did not happen is worse than none, because it
// reads as done.
func TestRunWritesNoReceiptWhenTheReviewerFails(t *testing.T) {
	for _, reviewer := range []struct {
		scaffold ambient.Scaffold
		binary   string
	}{
		{ambient.ScaffoldCodex, "codex"},
		{ambient.ScaffoldClaudeCode, "claude"},
	} {
		t.Run(string(reviewer.scaffold), func(t *testing.T) {
			root := branchWithWork(t)
			stateRoot := t.TempDir()
			stubReviewer(t, reviewer.binary, `cat >/dev/null; echo "reviewer exploded" >&2; exit 3`)

			_, err := Run(context.Background(), Input{
				RepositoryRoot: root, StateRoot: stateRoot, BaseRef: "main",
				Author: ambient.ScaffoldCodex, Reviewer: reviewer.scaffold,
			})
			if err == nil {
				t.Fatal("a failing reviewer reported success")
			}
			// The vendor's own message reaches the caller unchanged: a wrapper
			// that translated it would turn a loud break into a quiet one.
			if !strings.Contains(err.Error(), "reviewer exploded") {
				t.Fatalf("the reviewer's own failure was not surfaced: %v", err)
			}
			if _, found, _ := ReadReceipt(stateRoot, root, "work"); found {
				t.Fatal("a receipt was written for a review that did not happen")
			}
		})
	}
}

// An empty report is a failed review wearing a success exit status.
func TestRunRefusesAnEmptyReport(t *testing.T) {
	root := branchWithWork(t)
	stateRoot := t.TempDir()
	stubReviewer(t, "codex", `cat >/dev/null; printf '   \n'`)

	if _, err := Run(context.Background(), Input{
		RepositoryRoot: root, StateRoot: stateRoot, BaseRef: "main",
		Author: ambient.ScaffoldClaudeCode, Reviewer: ambient.ScaffoldCodex,
	}); err == nil {
		t.Fatal("an empty report was accepted")
	}
	if _, found, _ := ReadReceipt(stateRoot, root, "work"); found {
		t.Fatal("a receipt was written for an empty report")
	}
}

func TestGateRefusesBeforeAnythingIsSpawned(t *testing.T) {
	root := branchWithWork(t)
	stateRoot := t.TempDir()
	marker := filepath.Join(t.TempDir(), "spawned")
	stubReviewer(t, "codex", `touch `+marker+`; echo reviewed`)

	_, err := Run(context.Background(), Input{
		RepositoryRoot: root, StateRoot: stateRoot, BaseRef: "main",
		Author: ambient.ScaffoldClaudeCode, Reviewer: ambient.ScaffoldCodex,
		Gate: `echo "budget exhausted" >&2; exit 1`,
	})
	if !errors.Is(err, ErrGateRefused) {
		t.Fatalf("a failing gate produced %v", err)
	}
	if !strings.Contains(err.Error(), "budget exhausted") {
		t.Fatalf("the refusal does not carry the gate's own message: %v", err)
	}
	if _, statErr := os.Stat(marker); statErr == nil {
		t.Fatal("the reviewer was spawned despite the gate refusing")
	}
	if _, found, _ := ReadReceipt(stateRoot, root, "work"); found {
		t.Fatal("a receipt was written despite the gate refusing")
	}
}

func TestGatePermitsAndAbsenceOfAGateIsNotARefusal(t *testing.T) {
	for name, gate := range map[string]string{
		"a passing gate": "exit 0",
		"no gate":        "",
	} {
		t.Run(name, func(t *testing.T) {
			root := branchWithWork(t)
			stateRoot := t.TempDir()
			stubReviewer(t, "codex", `cat >/dev/null; echo reviewed`)

			if _, err := Run(context.Background(), Input{
				RepositoryRoot: root, StateRoot: stateRoot, BaseRef: "main",
				Author: ambient.ScaffoldClaudeCode, Reviewer: ambient.ScaffoldCodex,
				Gate: gate,
			}); err != nil {
				t.Fatalf("%s refused the review: %v", name, err)
			}
		})
	}
}

// A reviewer that believes it is a continuation of the session that wrote the
// code is not an independent reviewer.
func TestTheReviewerDoesNotInheritTheAuthorsSessionIdentity(t *testing.T) {
	root := branchWithWork(t)
	stateRoot := t.TempDir()
	t.Setenv("CLAUDECODE", "1")
	t.Setenv("CLAUDE_CODE_SESSION_ID", "parent-session")
	stubReviewer(t, "codex", `cat >/dev/null; echo "CLAUDECODE=[${CLAUDECODE:-unset}] SESSION=[${CLAUDE_CODE_SESSION_ID:-unset}]"`)

	result, err := Run(context.Background(), Input{
		RepositoryRoot: root, StateRoot: stateRoot, BaseRef: "main",
		Author: ambient.ScaffoldClaudeCode, Reviewer: ambient.ScaffoldCodex,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Receipt.Report, "CLAUDECODE=[unset]") ||
		!strings.Contains(result.Receipt.Report, "SESSION=[unset]") {
		t.Fatalf("the reviewer inherited the author's session: %q", result.Receipt.Report)
	}
}

func TestRunRefusesWhereThereIsNothingToReview(t *testing.T) {
	root := repository(t)
	branch(t, root, "work")
	stateRoot := t.TempDir()
	stubReviewer(t, "codex", `echo reviewed`)

	_, err := Run(context.Background(), Input{
		RepositoryRoot: root, StateRoot: stateRoot, BaseRef: "main",
		Author: ambient.ScaffoldClaudeCode, Reviewer: ambient.ScaffoldCodex,
	})
	if err == nil || !strings.Contains(err.Error(), "nothing to review") {
		t.Fatalf("a branch identical to its base produced %v", err)
	}
}

func TestRunRefusesADetachedHead(t *testing.T) {
	root := branchWithWork(t)
	stateRoot := t.TempDir()
	head, _ := Resolve(root, "HEAD")
	if _, err := git(root, "checkout", "-q", "--detach", head); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(context.Background(), Input{
		RepositoryRoot: root, StateRoot: stateRoot, BaseRef: "main",
		Author: ambient.ScaffoldClaudeCode, Reviewer: ambient.ScaffoldCodex,
	}); err == nil || !strings.Contains(err.Error(), "detached") {
		t.Fatalf("a detached head produced %v", err)
	}
}

// The first live run of this feature died here: `codex review` documents both
// --base and a custom prompt, and refuses them together. Reading its help was
// not enough, so the shape is pinned rather than trusted.
func TestCodexInvocationNeverCombinesAScopeFlagWithInstructions(t *testing.T) {
	name, arguments, stdin, err := reviewCommand(ambient.ScaffoldCodex, "main", []byte("look for X"))
	if err != nil {
		t.Fatal(err)
	}
	if name != "codex" {
		t.Fatalf("invoked %q", name)
	}
	for _, forbidden := range []string{"--base", "--uncommitted", "--commit"} {
		for _, argument := range arguments {
			if argument == forbidden {
				t.Fatalf("%s was passed alongside custom instructions: %v", forbidden, arguments)
			}
		}
	}
	// The range has to survive somewhere, and with no scope flag the only place
	// left is the prose.
	if !strings.Contains(stdin, "main...HEAD") {
		t.Fatalf("the reviewed range is not stated to the reviewer: %q", stdin)
	}
	if !strings.Contains(stdin, "look for X") {
		t.Fatal("the repository's instructions did not reach the reviewer")
	}
}

// A reviewer that can edit is no longer reviewing.
func TestClaudeInvocationCarriesNoEditingTool(t *testing.T) {
	_, arguments, stdin, err := reviewCommand(ambient.ScaffoldClaudeCode, "main", []byte("look for X"))
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(arguments, " ")
	for _, forbidden := range []string{"Edit", "Write", "NotebookEdit"} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf("the reviewer was given %s: %v", forbidden, arguments)
		}
	}
	if !strings.Contains(stdin, "main...HEAD") || !strings.Contains(stdin, "look for X") {
		t.Fatalf("the prompt lost the range or the instructions: %q", stdin)
	}
}

func TestUnknownReviewerHasNoInvocation(t *testing.T) {
	if _, _, _, err := reviewCommand(ambient.Scaffold("something-else"), "main", nil); err == nil {
		t.Fatal("an unknown reviewer produced an invocation")
	}
}
