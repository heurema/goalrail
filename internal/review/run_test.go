package review

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
		Author: ambient.ScaffoldClaudeCode, Selection: Selection{Reviewer: ambient.ScaffoldCodex, Mode: "cross", Reason: "test"},
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
				Author: ambient.ScaffoldCodex, Selection: Selection{Reviewer: reviewer.scaffold, Mode: "cross", Reason: "test"},
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
		Author: ambient.ScaffoldClaudeCode, Selection: Selection{Reviewer: ambient.ScaffoldCodex, Mode: "cross", Reason: "test"},
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
		Author: ambient.ScaffoldClaudeCode, Selection: Selection{Reviewer: ambient.ScaffoldCodex, Mode: "cross", Reason: "test"},
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
				Author: ambient.ScaffoldClaudeCode, Selection: Selection{Reviewer: ambient.ScaffoldCodex, Mode: "cross", Reason: "test"},
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
		Author: ambient.ScaffoldClaudeCode, Selection: Selection{Reviewer: ambient.ScaffoldCodex, Mode: "cross", Reason: "test"},
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
		Author: ambient.ScaffoldClaudeCode, Selection: Selection{Reviewer: ambient.ScaffoldCodex, Mode: "cross", Reason: "test"},
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
		Author: ambient.ScaffoldClaudeCode, Selection: Selection{Reviewer: ambient.ScaffoldCodex, Mode: "cross", Reason: "test"},
	}); err == nil || !strings.Contains(err.Error(), "detached") {
		t.Fatalf("a detached head produced %v", err)
	}
}

// The first live run of this feature died here: `codex review` documents both
// --base and a custom prompt, and refuses them together. Reading its help was
// not enough, so the shape is pinned rather than trusted.
func TestCodexInvocationNeverCombinesAScopeFlagWithInstructions(t *testing.T) {
	name, arguments, stdin, err := reviewCommand(ambient.ScaffoldCodex, "main...HEAD", DefaultEffort, []byte("look for X"))
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
	_, arguments, stdin, err := reviewCommand(ambient.ScaffoldClaudeCode, "main...HEAD", DefaultEffort, []byte("look for X"))
	if err != nil {
		t.Fatal(err)
	}
	allowed := allowedTools(arguments)
	for _, forbidden := range []string{"Edit", "Write", "NotebookEdit"} {
		for _, tool := range allowed {
			if tool == forbidden {
				t.Fatalf("the reviewer was allowed %s: %v", forbidden, allowed)
			}
		}
	}
	// `git *` admitted reset, checkout, clean, and arbitrary shell through
	// `git -c alias.x='!...'`. Every allowance must name its subcommand.
	for _, tool := range allowed {
		if strings.HasPrefix(tool, "Bash(") && !strings.Contains(tool, ":") {
			t.Fatalf("an unscoped shell allowance survived: %s", tool)
		}
	}
	if !strings.Contains(stdin, "main...HEAD") || !strings.Contains(stdin, "look for X") {
		t.Fatalf("the prompt lost the range or the instructions: %q", stdin)
	}
}

func TestUnknownReviewerHasNoInvocation(t *testing.T) {
	if _, _, _, err := reviewCommand(ambient.Scaffold("something-else"), "main...HEAD", DefaultEffort, nil); err == nil {
		t.Fatal("an unknown reviewer produced an invocation")
	}
}

// allowedTools returns the values of --allowed-tools, which run until the next
// flag.
func allowedTools(arguments []string) []string {
	var allowed []string
	collecting := false
	for _, argument := range arguments {
		if argument == "--allowed-tools" {
			collecting = true
			continue
		}
		if strings.HasPrefix(argument, "--") {
			collecting = false
			continue
		}
		if collecting {
			allowed = append(allowed, argument)
		}
	}
	return allowed
}

// A deadline that only kills the direct child is not a deadline. The reviewers
// are wrappers whose grandchild holds the same pipes, so this stubs that exact
// shape: a script that backgrounds a long sleep inheriting stdout, then exits.
// Without a process-group kill and a wait delay, Run blocks until the sleep is
// done — measured in production as fifty-three minutes against a twenty-minute
// deadline.
func TestTheDeadlineBoundsAReviewerThatOutlivesItsChild(t *testing.T) {
	root := branchWithWork(t)
	stateRoot := t.TempDir()
	stubReviewer(t, "codex", `cat >/dev/null; sleep 600 & sleep 600`)

	started := time.Now()
	_, err := Run(context.Background(), Input{
		RepositoryRoot: root, StateRoot: stateRoot, BaseRef: "main",
		Author: ambient.ScaffoldClaudeCode, Selection: Selection{Reviewer: ambient.ScaffoldCodex, Mode: "cross", Reason: "test"},
		Deadline: 2 * time.Second,
	})
	elapsed := time.Since(started)

	if err == nil {
		t.Fatal("a reviewer past its deadline reported success")
	}
	if !strings.Contains(err.Error(), "deadline") {
		t.Fatalf("the refusal does not name the deadline: %v", err)
	}
	// Generous, and still an order of magnitude below the hang it replaces.
	if elapsed > 30*time.Second {
		t.Fatalf("the deadline did not bound the review: took %s", elapsed)
	}
	if _, found, _ := ReadReceipt(stateRoot, root, "work"); found {
		t.Fatal("a receipt was written for a review that ran out of time")
	}
}

// A round costs what the round is about: the second review's range starts at
// the first receipt's head. The full branch digest still guards staleness.
func TestReReviewIsIncrementalByDefaultAndFullByFlag(t *testing.T) {
	root := branchWithWork(t)
	stateRoot := t.TempDir()
	stubReviewer(t, "codex", `cat >/dev/null; echo round`)
	selection := Selection{Reviewer: ambient.ScaffoldCodex, Mode: "cross", Reason: "test"}

	first, err := Run(context.Background(), Input{
		RepositoryRoot: root, StateRoot: stateRoot, BaseRef: "main", Selection: selection,
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.Receipt.ReviewedBase != first.Receipt.BaseCommit {
		t.Fatalf("the first round did not cover the whole branch: %+v", first.Receipt)
	}

	write(t, root, "more.txt", "more\n")
	commit(t, root, "more work")
	second, err := Run(context.Background(), Input{
		RepositoryRoot: root, StateRoot: stateRoot, BaseRef: "main", Selection: selection,
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.Receipt.ReviewedBase != first.Receipt.HeadCommit {
		t.Fatalf("the second round did not start at the first round's head: %q vs %q",
			second.Receipt.ReviewedBase, first.Receipt.HeadCommit)
	}
	// Staleness still guards the whole branch.
	if state, _, _ := Status(stateRoot, root, "work"); state != StateCurrent {
		t.Fatalf("an incremental receipt does not keep the branch current: %v", state)
	}

	write(t, root, "even-more.txt", "x\n")
	commit(t, root, "even more")
	full, err := Run(context.Background(), Input{
		RepositoryRoot: root, StateRoot: stateRoot, BaseRef: "main", Selection: selection, Full: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if full.Receipt.ReviewedBase != full.Receipt.BaseCommit {
		t.Fatalf("--full did not cover the whole branch: %+v", full.Receipt)
	}
}

// The refuted receipt carries both reports verbatim; nothing is derived.
func TestRefuteStoresBothReportsVerbatim(t *testing.T) {
	root := branchWithWork(t)
	stateRoot := t.TempDir()
	stubReviewer(t, "codex", `cat >/dev/null; echo "the finding"`)
	stubReviewer(t, "claude", `cat >/dev/null; echo "REFUTED: not real"`)

	result, err := Run(context.Background(), Input{
		RepositoryRoot: root, StateRoot: stateRoot, BaseRef: "main",
		Selection: Selection{Reviewer: ambient.ScaffoldCodex, Mode: "cross", Reason: "test"},
		Refute:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Receipt.Report, "the finding") {
		t.Fatalf("the review report was lost: %q", result.Receipt.Report)
	}
	if !strings.Contains(result.Receipt.Refutation, "REFUTED: not real") {
		t.Fatalf("the refutation was lost: %q", result.Receipt.Refutation)
	}
	if result.Receipt.Mode != "refute" || result.Receipt.Refuter == "" {
		t.Fatalf("the mode does not say what happened: %+v", result.Receipt)
	}
	if result.Receipt.RefutationSHA256 != digest([]byte(result.Receipt.Refutation)) {
		t.Fatal("the refutation digest does not describe the stored refutation")
	}
}

// A round that changes neither the commits nor the tree means the author acted
// on nothing; the count is what a loop policy stops on, and Goalrail only
// measures it.
func TestStalemateCountsRoundsThatChangedNothing(t *testing.T) {
	root := branchWithWork(t)
	stateRoot := t.TempDir()
	stubReviewer(t, "codex", `cat >/dev/null; echo round`)
	selection := Selection{Reviewer: ambient.ScaffoldCodex, Mode: "cross", Reason: "test"}
	input := Input{RepositoryRoot: root, StateRoot: stateRoot, BaseRef: "main", Selection: selection}

	first, err := Run(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if first.Receipt.UnchangedRounds != 0 {
		t.Fatalf("the first round reported a stalemate: %d", first.Receipt.UnchangedRounds)
	}

	second, err := Run(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if second.Receipt.UnchangedRounds != 1 {
		t.Fatalf("an unchanged round did not count: %d", second.Receipt.UnchangedRounds)
	}
	third, err := Run(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if third.Receipt.UnchangedRounds != 2 {
		t.Fatalf("consecutive unchanged rounds did not accumulate: %d", third.Receipt.UnchangedRounds)
	}

	// Acting on the findings resets it — that is what convergence looks like.
	write(t, root, "fix.txt", "fixed\n")
	commit(t, root, "act on the findings")
	after, err := Run(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if after.Receipt.UnchangedRounds != 0 {
		t.Fatalf("a round after real work still reported a stalemate: %d", after.Receipt.UnchangedRounds)
	}
}

// An uncommitted fix in progress is work, not a stalemate.
func TestADirtyTreeIsNotAStalemate(t *testing.T) {
	root := branchWithWork(t)
	stateRoot := t.TempDir()
	stubReviewer(t, "codex", `cat >/dev/null; echo round`)
	selection := Selection{Reviewer: ambient.ScaffoldCodex, Mode: "cross", Reason: "test"}
	input := Input{RepositoryRoot: root, StateRoot: stateRoot, BaseRef: "main", Selection: selection}

	if _, err := Run(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	write(t, root, "in-progress.txt", "half a fix\n")
	second, err := Run(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if second.Receipt.UnchangedRounds != 0 {
		t.Fatalf("work in progress was counted as a stalemate: %d", second.Receipt.UnchangedRounds)
	}
}

// Goalrail's own materialized instructions file must not suppress Goalrail's
// own stalemate signal in a repository that has not committed it yet.
func TestTheMaterializedInstructionsFileIsNotTreatedAsWork(t *testing.T) {
	root := repository(t)
	if _, _, err := EnsureInstructions(root); err != nil {
		t.Fatal(err)
	}
	clean, err := WorkingTreeClean(root)
	if err != nil {
		t.Fatal(err)
	}
	if !clean {
		t.Fatal("the materialized instructions file was counted as uncommitted work")
	}
}
