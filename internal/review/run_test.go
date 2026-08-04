package review

import (
	"bytes"
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
	for _, name := range AuthorMarkers() {
		t.Setenv(name, "parent-session")
	}
	t.Setenv("GOALRAIL_ENV_SURVIVES", "yes")
	stubReviewer(t, "codex", `cat >/dev/null; env`)

	result, err := Run(context.Background(), Input{
		RepositoryRoot: root, StateRoot: stateRoot, BaseRef: "main",
		Author: ambient.ScaffoldClaudeCode, Selection: Selection{Reviewer: ambient.ScaffoldCodex, Mode: "cross", Reason: "test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range AuthorMarkers() {
		if strings.Contains(result.Receipt.Report, name+"=") {
			t.Fatalf("the reviewer inherited %s: %q", name, result.Receipt.Report)
		}
	}
	if !strings.Contains(result.Receipt.Report, "GOALRAIL_ENV_SURVIVES=yes") {
		t.Fatalf("the reviewer lost unrelated environment: %q", result.Receipt.Report)
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

func TestRunRefusesAnEmptyThreeDotRangeBeforeSideEffects(t *testing.T) {
	root := repository(t)
	workHead, err := Resolve(root, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	branch(t, root, "work")
	if _, err := git(root, "checkout", "-q", "main"); err != nil {
		t.Fatal(err)
	}
	write(t, root, "base-only.txt", "base advanced\n")
	commit(t, root, "advance base")
	baseHead, err := Resolve(root, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	setRemoteDefault(t, root, "origin", "main", baseHead)
	if _, err := git(root, "checkout", "-q", "work"); err != nil {
		t.Fatal(err)
	}
	if workHead == baseHead {
		t.Fatal("the fixture did not advance the remote default beyond the work head")
	}

	gateMarker := filepath.Join(t.TempDir(), "gate-ran")
	reviewerMarker := filepath.Join(t.TempDir(), "reviewer-ran")
	t.Setenv("GATE_MARKER", gateMarker)
	t.Setenv("REVIEWER_MARKER", reviewerMarker)
	stubReviewer(t, "codex", `touch "$REVIEWER_MARKER"; cat >/dev/null; echo reviewed`)

	result, err := Run(context.Background(), Input{
		RepositoryRoot: root,
		StateRoot:      t.TempDir(),
		Selection: Selection{
			Reviewer: ambient.ScaffoldCodex,
			Mode:     "cross",
			Reason:   "test",
		},
		Gate: `touch "$GATE_MARKER"`,
	})
	if err == nil || !strings.Contains(err.Error(), "nothing to review") {
		t.Fatalf("an empty three-dot range produced %v", err)
	}
	if result.InstructionsMaterialized {
		t.Fatal("an empty range materialized review instructions")
	}
	for _, path := range []string{gateMarker, reviewerMarker, filepath.Join(root, InstructionsPath)} {
		if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
			t.Fatalf("an empty range created %s: %v", path, statErr)
		}
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
	name, arguments, stdin, err := reviewCommand(ambient.ScaffoldCodex, "main...HEAD", DefaultEffort, DefaultModel(ambient.ScaffoldClaudeCode), []byte("look for X"))
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
	_, arguments, stdin, err := reviewCommand(ambient.ScaffoldClaudeCode, "main...HEAD", DefaultEffort, DefaultModel(ambient.ScaffoldClaudeCode), []byte("look for X"))
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
	if _, _, _, err := reviewCommand(ambient.Scaffold("something-else"), "main...HEAD", DefaultEffort, DefaultModel(ambient.ScaffoldClaudeCode), nil); err == nil {
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
	refuteCapture := filepath.Join(t.TempDir(), "refute-stdin")
	t.Setenv("REFUTE_CAPTURE", refuteCapture)
	stubReviewer(t, "codex", `cat >/dev/null; echo "the finding"`)
	stubReviewer(t, "claude", `cat >"$REFUTE_CAPTURE"; echo "REFUTED: not real"`)

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

	reviewedDiff, err := renderCanonicalDiff(context.Background(), root, result.Receipt.ReviewedBase, result.Receipt.HeadCommit)
	if err != nil {
		t.Fatal(err)
	}
	if result.Receipt.ReviewedDiffSHA256 != digest(reviewedDiff) {
		t.Fatal("the reviewed digest does not describe the canonical diff")
	}
	refuteInput, err := os.ReadFile(refuteCapture)
	if err != nil {
		t.Fatal(err)
	}
	for _, exact := range [][]byte{
		[]byte("--- REPORT UNDER CHALLENGE ---\n" + result.Receipt.Report),
		append([]byte("--- DIFF UNDER REVIEW ---\n"), reviewedDiff...),
	} {
		if !bytes.Contains(refuteInput, exact) {
			t.Fatalf("refuter input omitted exact bytes %q:\n%s", exact, refuteInput)
		}
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

// --full changes the reviewed scope, not whether anything moved.
func TestStalemateIsMeasuredEvenOnAFullReview(t *testing.T) {
	root := branchWithWork(t)
	stateRoot := t.TempDir()
	stubReviewer(t, "codex", `cat >/dev/null; echo round`)
	selection := Selection{Reviewer: ambient.ScaffoldCodex, Mode: "cross", Reason: "test"}
	base := Input{RepositoryRoot: root, StateRoot: stateRoot, BaseRef: "main", Selection: selection}

	if _, err := Run(context.Background(), base); err != nil {
		t.Fatal(err)
	}
	full := base
	full.Full = true
	second, err := Run(context.Background(), full)
	if err != nil {
		t.Fatal(err)
	}
	if second.Receipt.UnchangedRounds != 1 {
		t.Fatalf("a full re-review of the same head did not count: %d", second.Receipt.UnchangedRounds)
	}
}

// Discarded work moved between the rounds even though nothing landed, so the
// rounds were not over identical state.
func TestDiscardedWorkDoesNotCountAsAStalemate(t *testing.T) {
	root := branchWithWork(t)
	stateRoot := t.TempDir()
	stubReviewer(t, "codex", `cat >/dev/null; echo round`)
	selection := Selection{Reviewer: ambient.ScaffoldCodex, Mode: "cross", Reason: "test"}
	input := Input{RepositoryRoot: root, StateRoot: stateRoot, BaseRef: "main", Selection: selection}

	// A round taken while the author had work in progress.
	write(t, root, "wip.txt", "half a fix\n")
	dirty, err := Run(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if dirty.Receipt.TreeCleanAtReview == nil || *dirty.Receipt.TreeCleanAtReview {
		t.Fatal("a dirty round recorded a clean tree")
	}

	// The author discards it. Same head, clean now — but state moved.
	if err := os.Remove(filepath.Join(root, "wip.txt")); err != nil {
		t.Fatal(err)
	}
	after, err := Run(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if after.Receipt.UnchangedRounds != 0 {
		t.Fatalf("a round following discarded work counted as a stalemate: %d", after.Receipt.UnchangedRounds)
	}

	// Only now are two rounds genuinely over identical state.
	third, err := Run(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if third.Receipt.UnchangedRounds != 1 {
		t.Fatalf("two rounds over identical state did not count: %d", third.Receipt.UnchangedRounds)
	}
}

// A committed instructions file the author then edits is ordinary work.
func TestEditingACommittedInstructionsFileCountsAsWork(t *testing.T) {
	root := repository(t)
	if _, _, err := EnsureInstructions(root); err != nil {
		t.Fatal(err)
	}
	// Untracked default: not the author's work.
	if clean, err := WorkingTreeClean(root); err != nil || !clean {
		t.Fatalf("the materialized default counted as work: %v (%v)", clean, err)
	}
	commit(t, root, "commit the review instructions")

	// Committed and then edited: that is the author changing the review rules.
	write(t, root, InstructionsPath, "# Mine\n\nlook only at X\n")
	clean, err := WorkingTreeClean(root)
	if err != nil {
		t.Fatal(err)
	}
	if clean {
		t.Fatal("an edit to the committed instructions file was skipped as if Goalrail had made it")
	}
}

// Absent and false are different facts: a receipt written before the tree
// field existed knows nothing about that round, and inferring "dirty" from its
// silence would record an unknown as a measurement.
func TestALegacyReceiptWithoutTreeStateNeverCountsAsAStalemate(t *testing.T) {
	root := branchWithWork(t)
	stateRoot := t.TempDir()
	stubReviewer(t, "codex", `cat >/dev/null; echo round`)
	head, _ := Resolve(root, "HEAD")
	base, _ := Resolve(root, "main")
	diff, _ := DiffDigest(root, base, head)

	// A receipt as an earlier version would have written it: no tree state.
	if _, err := WriteReceipt(stateRoot, Receipt{
		Schema: ReceiptSchema, Repository: root, Branch: "work",
		BaseRef: "main", BaseCommit: base, HeadCommit: head,
		DiffSHA256: diff, ReviewedBase: base, ReviewedDiffSHA256: diff,
	}); err != nil {
		t.Fatal(err)
	}
	result, err := Run(context.Background(), Input{
		RepositoryRoot: root, StateRoot: stateRoot, BaseRef: "main",
		Selection: Selection{Reviewer: ambient.ScaffoldCodex, Mode: "cross", Reason: "test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Receipt.UnchangedRounds != 0 {
		t.Fatalf("a receipt that knew nothing about its tree counted as a stalemate: %d",
			result.Receipt.UnchangedRounds)
	}
	if result.Receipt.TreeCleanAtReview == nil || !*result.Receipt.TreeCleanAtReview {
		t.Fatal("this round measured a clean tree and did not record it")
	}
}

// Codex streams its whole session transcript to stderr — over a megabyte on an
// ordinary large branch — and a refusing bound there killed the first real
// field review. Diagnostics keep a tail; only the report may refuse.
func TestAVerboseReviewerStderrDoesNotKillTheReview(t *testing.T) {
	root := branchWithWork(t)
	stateRoot := t.TempDir()
	stubReviewer(t, "codex", `cat >/dev/null
i=0; while [ $i -lt 3000 ]; do printf '%0512d\n' $i >&2; i=$((i+1)); done
echo "the report"`)

	result, err := Run(context.Background(), Input{
		RepositoryRoot: root, StateRoot: stateRoot, BaseRef: "main",
		Selection: Selection{Reviewer: ambient.ScaffoldCodex, Mode: "cross", Reason: "test"},
	})
	if err != nil {
		t.Fatalf("a verbose stderr failed the review: %v", err)
	}
	if !strings.Contains(result.Receipt.Report, "the report") {
		t.Fatalf("the report was lost: %q", result.Receipt.Report)
	}
}

// And when the reviewer does fail, the error carries a tail, not a transcript.
func TestAFailureDetailIsATailNotATranscript(t *testing.T) {
	root := branchWithWork(t)
	stateRoot := t.TempDir()
	stubReviewer(t, "codex", `cat >/dev/null
i=0; while [ $i -lt 3000 ]; do printf '%0512d\n' $i >&2; i=$((i+1)); done
echo "FINAL-REASON" >&2; exit 3`)

	_, err := Run(context.Background(), Input{
		RepositoryRoot: root, StateRoot: stateRoot, BaseRef: "main",
		Selection: Selection{Reviewer: ambient.ScaffoldCodex, Mode: "cross", Reason: "test"},
	})
	if err == nil {
		t.Fatal("a failing reviewer reported success")
	}
	if len(err.Error()) > 8192 {
		t.Fatalf("the error is a transcript, not a message: %d bytes", len(err.Error()))
	}
	if !strings.Contains(err.Error(), "FINAL-REASON") {
		t.Fatalf("the tail lost the reason: %v", err)
	}
	if !strings.Contains(err.Error(), "exit status 3") {
		t.Fatalf("the process's own exit error was lost: %v", err)
	}
}

// And a reason printed before the noise survives too: vendors disagree about
// which end carries the cause, so both bounded ends do.
func TestAFailureReasonPrintedFirstSurvivesTheNoise(t *testing.T) {
	root := branchWithWork(t)
	stateRoot := t.TempDir()
	stubReviewer(t, "codex", `cat >/dev/null
echo "EARLY-PANIC: real cause" >&2
i=0; while [ $i -lt 3000 ]; do printf '%0512d\n' $i >&2; i=$((i+1)); done
exit 3`)

	_, err := Run(context.Background(), Input{
		RepositoryRoot: root, StateRoot: stateRoot, BaseRef: "main",
		Selection: Selection{Reviewer: ambient.ScaffoldCodex, Mode: "cross", Reason: "test"},
	})
	if err == nil {
		t.Fatal("a failing reviewer reported success")
	}
	if !strings.Contains(err.Error(), "EARLY-PANIC") {
		t.Fatalf("a leading cause was truncated away: %d bytes", len(err.Error()))
	}
}

// A full pass is the thoroughness pass, so it must not inherit the loop's cheap
// defaults — measured: at medium the same range reviewed clean and missed three
// real defects that high found.
func TestAFullPassIsThoroughByDefaultAndTheCallerStillWins(t *testing.T) {
	root := branchWithWork(t)
	selection := Selection{Reviewer: ambient.ScaffoldCodex, Mode: "cross", Reason: "test"}

	// The stub reports the effort it was actually invoked with.
	stubReviewer(t, "codex", `args="$*"; cat >/dev/null; echo "invoked with: $args"`)

	full, err := Run(context.Background(), Input{
		RepositoryRoot: root, StateRoot: t.TempDir(), BaseRef: "main",
		Selection: selection, Full: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(full.Receipt.Report, "model_reasoning_effort="+FullEffort) {
		t.Fatalf("a full pass did not run at %s: %q", FullEffort, full.Receipt.Report)
	}

	incremental, err := Run(context.Background(), Input{
		RepositoryRoot: root, StateRoot: t.TempDir(), BaseRef: "main",
		Selection: selection,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(incremental.Receipt.Report, "model_reasoning_effort="+DefaultEffort) {
		t.Fatalf("an ordinary round did not run at %s: %q", DefaultEffort, incremental.Receipt.Report)
	}

	// An explicit effort always wins, full pass or not.
	stated, err := Run(context.Background(), Input{
		RepositoryRoot: root, StateRoot: t.TempDir(), BaseRef: "main",
		Selection: selection, Full: true, Effort: "low",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stated.Receipt.Report, "model_reasoning_effort=low") {
		t.Fatalf("the caller's effort was overridden: %q", stated.Receipt.Report)
	}
}

// Raising the effort without raising the bound only moves the failure from a
// false clean verdict to a deadline — measured: ultra reached the twenty-minute
// bound and returned nothing.
func TestAFullPassCarriesTheLongerDeadlineAndTheCallerStillWins(t *testing.T) {
	root := branchWithWork(t)
	stubReviewer(t, "codex", `cat >/dev/null; echo r`)
	selection := Selection{Reviewer: ambient.ScaffoldCodex, Mode: "cross", Reason: "test"}

	if FullDeadline <= DefaultDeadline {
		t.Fatalf("a full pass is bounded no better than a loop round: %s vs %s", FullDeadline, DefaultDeadline)
	}
	// The caller's bound wins: a deliberately short one still refuses.
	stubReviewer(t, "codex", `cat >/dev/null; sleep 600 & sleep 600`)
	_, err := Run(context.Background(), Input{
		RepositoryRoot: root, StateRoot: t.TempDir(), BaseRef: "main",
		Selection: selection, Full: true, Deadline: 2 * time.Second,
	})
	if err == nil || !strings.Contains(err.Error(), "deadline") {
		t.Fatalf("an explicit deadline was ignored on a full pass: %v", err)
	}
}

// An expensive timeout must teach something: a bare "did not finish" cannot
// tell working-but-slow from stuck, and the next decision has no evidence.
func TestATimeoutCarriesWhatTheReviewerHadProduced(t *testing.T) {
	root := branchWithWork(t)
	stubReviewer(t, "codex", `cat >/dev/null; echo "PROGRESS-MARKER: reading files" >&2; sleep 600`)

	_, err := Run(context.Background(), Input{
		RepositoryRoot: root, StateRoot: t.TempDir(), BaseRef: "main",
		Selection: Selection{Reviewer: ambient.ScaffoldCodex, Mode: "cross", Reason: "test"},
		Deadline:  2 * time.Second,
	})
	if err == nil || !strings.Contains(err.Error(), "deadline") {
		t.Fatalf("expected a deadline failure, got %v", err)
	}
	if !strings.Contains(err.Error(), "PROGRESS-MARKER") {
		t.Fatalf("the timeout discarded the reviewer's progress: %v", err)
	}
}

// Rendering the reviewed range is part of the advertised deadline too.
func TestTheReviewContextCoversDiffRenderingBeforeSideEffects(t *testing.T) {
	root := branchWithWork(t)
	reviewerMarker := filepath.Join(t.TempDir(), "reviewer-ran")
	t.Setenv("REVIEWER_MARKER", reviewerMarker)
	stubReviewer(t, "codex", `touch "$REVIEWER_MARKER"; cat >/dev/null; echo reviewed`)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := Run(ctx, Input{
		RepositoryRoot: root,
		StateRoot:      t.TempDir(),
		BaseRef:        "main",
		Selection: Selection{
			Reviewer: ambient.ScaffoldCodex,
			Mode:     "cross",
			Reason:   "test",
		},
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled diff rendering produced %v", err)
	}
	if result.InstructionsMaterialized {
		t.Fatal("canceled diff rendering materialized review instructions")
	}
	for _, path := range []string{reviewerMarker, filepath.Join(root, InstructionsPath)} {
		if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
			t.Fatalf("canceled diff rendering created %s: %v", path, statErr)
		}
	}
}

// The bound is advertised for the whole review, and a gate is part of it.
func TestTheDeadlineCoversTheGate(t *testing.T) {
	root := branchWithWork(t)
	stubReviewer(t, "codex", `cat >/dev/null; echo r`)
	started := time.Now()
	_, err := Run(context.Background(), Input{
		RepositoryRoot: root, StateRoot: t.TempDir(), BaseRef: "main",
		Selection: Selection{Reviewer: ambient.ScaffoldCodex, Mode: "cross", Reason: "test"},
		Gate:      "sleep 600", Deadline: 2 * time.Second,
	})
	if err == nil {
		t.Fatal("a blocking gate under a two-second bound reported success")
	}
	if elapsed := time.Since(started); elapsed > 30*time.Second {
		t.Fatalf("the gate ran outside the review's bound: took %s", elapsed)
	}
}

// Silence is observed, never diagnosed: a reviewer may buffer everything until
// it finishes, so no output proves no output — not that anything stopped.
func TestASilentTimeoutStatesTheObservationNotACause(t *testing.T) {
	root := branchWithWork(t)
	stubReviewer(t, "codex", `cat >/dev/null; sleep 600`)
	_, err := Run(context.Background(), Input{
		RepositoryRoot: root, StateRoot: t.TempDir(), BaseRef: "main",
		Selection: Selection{Reviewer: ambient.ScaffoldCodex, Mode: "cross", Reason: "test"},
		Deadline:  2 * time.Second,
	})
	if err == nil {
		t.Fatal("expected a deadline failure")
	}
	for _, invented := range []string{"hang", "hung", "stuck"} {
		if strings.Contains(strings.ToLower(err.Error()), invented) {
			t.Fatalf("silence was diagnosed rather than reported: %v", err)
		}
	}
	if !strings.Contains(err.Error(), "no output before the deadline") {
		t.Fatalf("the observation itself was lost: %v", err)
	}
}

// A timeout may not dump what the ordinary failure path is forbidden to dump.
func TestATimeoutErrorIsBoundedLikeAnyOther(t *testing.T) {
	root := branchWithWork(t)
	stubReviewer(t, "codex", `cat >/dev/null
i=0; while [ $i -lt 400 ]; do printf '%0512d\n' $i >&2; i=$((i+1)); done
sleep 600`)
	_, err := Run(context.Background(), Input{
		RepositoryRoot: root, StateRoot: t.TempDir(), BaseRef: "main",
		Selection: Selection{Reviewer: ambient.ScaffoldCodex, Mode: "cross", Reason: "test"},
		Deadline:  3 * time.Second,
	})
	if err == nil {
		t.Fatal("expected a deadline failure")
	}
	if len(err.Error()) > 8192 {
		t.Fatalf("a timeout dumped a transcript: %d bytes", len(err.Error()))
	}
}

// The bound covers the second gate too: a refute round is part of the review.
// One marker path, created before the input: the earlier version of this test
// used two t.TempDir() calls, so the first gate touched a file the second never
// looked for — it passed whether or not the refute gate was bounded at all.
func TestTheDeadlineCoversTheRefuteGate(t *testing.T) {
	root := branchWithWork(t)
	stubReviewer(t, "codex", `cat >/dev/null; echo "a finding"`)
	stubReviewer(t, "claude", `cat >/dev/null; echo "REFUTED"`)
	marker := filepath.Join(t.TempDir(), "first-gate-ran")

	started := time.Now()
	_, err := Run(context.Background(), Input{
		RepositoryRoot: root, StateRoot: t.TempDir(), BaseRef: "main",
		Selection: Selection{Reviewer: ambient.ScaffoldCodex, Mode: "cross", Reason: "test"},
		Refute:    true,
		// The first invocation passes and leaves the marker; the second finds it
		// and blocks, so only the refute gate is what the deadline must stop.
		Gate:     "if [ -f " + marker + " ]; then sleep 600; fi; touch " + marker,
		Deadline: 3 * time.Second,
	})
	elapsed := time.Since(started)
	if err == nil {
		t.Fatal("a blocking refute gate reported success")
	}
	if !strings.Contains(err.Error(), "deadline") {
		t.Fatalf("the refute gate was not stopped by the deadline: %v", err)
	}
	if elapsed > 40*time.Second {
		t.Fatalf("the refute gate ran outside the review's bound: %s", elapsed)
	}
}

// A gate the deadline killed never refused: reporting a budget denial there
// tells automation a different fact with a different response.
func TestAGateKilledByTheDeadlineIsNotARefusal(t *testing.T) {
	root := branchWithWork(t)
	stubReviewer(t, "codex", `cat >/dev/null; echo r`)
	_, err := Run(context.Background(), Input{
		RepositoryRoot: root, StateRoot: t.TempDir(), BaseRef: "main",
		Selection: Selection{Reviewer: ambient.ScaffoldCodex, Mode: "cross", Reason: "test"},
		Gate:      "sleep 600", Deadline: 2 * time.Second,
	})
	if err == nil {
		t.Fatal("a gate past the deadline reported success")
	}
	if errors.Is(err, ErrGateRefused) {
		t.Fatalf("a timed-out gate was reported as a budget refusal: %v", err)
	}
	if !strings.Contains(err.Error(), "deadline") {
		t.Fatalf("the timeout was not named: %v", err)
	}
}

// A timeout carries whatever each stream had: partial findings live on stdout
// while progress logging lives on stderr, and only one of them surviving
// discards half the evidence the timeout exists to provide.
func TestATimeoutKeepsBothStreams(t *testing.T) {
	root := branchWithWork(t)
	stubReviewer(t, "codex", `cat >/dev/null
echo "PARTIAL-FINDING on stdout"
echo "PROGRESS on stderr" >&2
sleep 600`)
	_, err := Run(context.Background(), Input{
		RepositoryRoot: root, StateRoot: t.TempDir(), BaseRef: "main",
		Selection: Selection{Reviewer: ambient.ScaffoldCodex, Mode: "cross", Reason: "test"},
		Deadline:  3 * time.Second,
	})
	if err == nil {
		t.Fatal("expected a deadline failure")
	}
	for _, expected := range []string{"PARTIAL-FINDING", "PROGRESS"} {
		if !strings.Contains(err.Error(), expected) {
			t.Fatalf("the timeout discarded %s: %v", expected, err)
		}
	}
}

// The reviewer must not be able to write, not merely be asked not to: the
// read-only boundary here lost to `git *` and then to `git diff --output=`
// before it was made structural.
func TestTheCodexReviewerIsInvokedWithoutWritePermission(t *testing.T) {
	_, arguments, _, err := reviewCommand(ambient.ScaffoldCodex, "main...HEAD", DefaultEffort, DefaultModel(ambient.ScaffoldClaudeCode), []byte("x"))
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(arguments, " ")
	if !strings.Contains(joined, "sandbox_mode=read-only") {
		t.Fatalf("the reviewer keeps whatever sandbox the machine configures: %v", arguments)
	}
}

// A gate is routinely a pipeline, so killing the shell alone leaves a
// descendant holding the pipe and Run waiting past the deadline.
func TestTheGateDeadlineBoundsItsWholeProcessTree(t *testing.T) {
	root := branchWithWork(t)
	stubReviewer(t, "codex", `cat >/dev/null; echo r`)
	started := time.Now()
	_, err := Run(context.Background(), Input{
		RepositoryRoot: root, StateRoot: t.TempDir(), BaseRef: "main",
		Selection: Selection{Reviewer: ambient.ScaffoldCodex, Mode: "cross", Reason: "test"},
		Gate:      "sleep 600 & sleep 600",
		Deadline:  2 * time.Second,
	})
	elapsed := time.Since(started)
	if err == nil || !strings.Contains(err.Error(), "deadline") {
		t.Fatalf("a gate whose descendant outlives it was not bounded: %v", err)
	}
	if elapsed > 30*time.Second {
		t.Fatalf("the gate's process tree ran past the bound: %s", elapsed)
	}
}

// A slow budget service and a gate failing for another reason look identical
// without the gate's own output.
func TestATimedOutGateCarriesItsDiagnostics(t *testing.T) {
	root := branchWithWork(t)
	stubReviewer(t, "codex", `cat >/dev/null; echo r`)
	_, err := Run(context.Background(), Input{
		RepositoryRoot: root, StateRoot: t.TempDir(), BaseRef: "main",
		Selection: Selection{Reviewer: ambient.ScaffoldCodex, Mode: "cross", Reason: "test"},
		Gate:      `echo "GATE-PROGRESS: asking the budget service" >&2; sleep 600`,
		Deadline:  3 * time.Second,
	})
	if err == nil {
		t.Fatal("expected a gate deadline failure")
	}
	if !strings.Contains(err.Error(), "GATE-PROGRESS") {
		t.Fatalf("the timed-out gate discarded its own output: %v", err)
	}
}

// A review invoked from a session must not inherit the model that session chose
// for authoring — measured the hard way, as a four-second "out of usage
// credits" on a path that had never run at all.
func TestTheClaudeReviewerIsGivenAModel(t *testing.T) {
	model := DefaultModel(ambient.ScaffoldClaudeCode)
	if model == "" {
		t.Fatal("the Claude reviewer has no stated model, so it inherits the session's")
	}
	_, arguments, _, err := reviewCommand(ambient.ScaffoldClaudeCode, "main...HEAD", DefaultEffort, model, []byte("x"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(arguments, " "), "--model "+model) {
		t.Fatalf("the reviewer inherits the session's model: %v", arguments)
	}

	// An empty model means the vendor's own default, not an empty flag value.
	_, bare, _, err := reviewCommand(ambient.ScaffoldClaudeCode, "main...HEAD", DefaultEffort, "", []byte("x"))
	if err != nil {
		t.Fatal(err)
	}
	for index, argument := range bare {
		if argument == "--model" && index+1 < len(bare) && strings.TrimSpace(bare[index+1]) == "" {
			t.Fatalf("an empty model was passed as a flag value: %v", bare)
		}
	}
}

// A caller's model reaches every provider. The claim that Codex could not take
// one on the command line was asserted without checking and is false — `-c
// model=` is accepted — and that false claim is what let the asymmetry survive
// its own review.
func TestACallerNamedModelReachesCodexToo(t *testing.T) {
	_, arguments, _, err := reviewCommand(ambient.ScaffoldCodex, "main...HEAD", DefaultEffort, "gpt-5.6-sol", []byte("x"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(arguments, " "), "model=gpt-5.6-sol") {
		t.Fatalf("a named model was dropped for the Codex reviewer: %v", arguments)
	}

	// With none named, its own configuration is left alone: no measurement says
	// the configured model is wrong for reviewing, and pinning a vendor model
	// identifier would age into a wrong default nobody revisits.
	if DefaultModel(ambient.ScaffoldCodex) != "" {
		t.Fatal("Codex was given a hardcoded default model")
	}
	_, bare, _, err := reviewCommand(ambient.ScaffoldCodex, "main...HEAD", DefaultEffort, "", []byte("x"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(bare, " "), "-c model=") {
		t.Fatalf("an unnamed model was forced onto Codex: %v", bare)
	}
}

// A model name belongs to one provider. Carrying the reviewer's name to a
// refuter of a different provider fails the second invocation after the first
// has already been paid for — and the refute path had never run live, so this
// would have broken for whoever first enabled it.
func TestTheRefuterGetsItsOwnProvidersModel(t *testing.T) {
	root := branchWithWork(t)
	stateRoot := t.TempDir()
	seen := filepath.Join(t.TempDir(), "codex-args")
	stubReviewer(t, "claude", `cat >/dev/null; echo "a finding"`)
	stubReviewer(t, "codex", `printf '%s' "$*" >> `+seen+`; cat >/dev/null; echo "REFUTED"`)

	if _, err := Run(context.Background(), Input{
		RepositoryRoot: root, StateRoot: stateRoot, BaseRef: "main",
		Selection: Selection{Reviewer: ambient.ScaffoldClaudeCode, Mode: "cross", Reason: "test"},
		Refute:    true,
	}); err != nil {
		t.Fatal(err)
	}
	recorded, err := os.ReadFile(seen)
	if err != nil {
		t.Fatal(err)
	}
	// Claude's default must not reach the Codex refuter.
	if strings.Contains(string(recorded), "model="+DefaultModel(ambient.ScaffoldClaudeCode)) {
		t.Fatalf("the reviewer's provider-specific model was carried to the refuter: %s", recorded)
	}
}

// An explicitly named model reaches the reviewer it was named for.
func TestAnExplicitModelAppliesToTheTargetedReviewer(t *testing.T) {
	root := branchWithWork(t)
	seen := filepath.Join(t.TempDir(), "args")
	stubReviewer(t, "codex", `printf '%s' "$*" > `+seen+`; cat >/dev/null; echo r`)

	if _, err := Run(context.Background(), Input{
		RepositoryRoot: root, StateRoot: t.TempDir(), BaseRef: "main",
		Selection: Selection{Reviewer: ambient.ScaffoldCodex, Mode: "cross", Reason: "test"},
		Model:     "gpt-5.6-sol",
	}); err != nil {
		t.Fatal(err)
	}
	recorded, err := os.ReadFile(seen)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(recorded), "model=gpt-5.6-sol") {
		t.Fatalf("the named model did not reach its reviewer: %s", recorded)
	}
}

// The receipt's contract is to prove how the review ran, and the execution
// parameters are part of how: without them two receipts for the same provider
// and range cannot say which settings produced them.
func TestTheReceiptRecordsTheExecutionParameters(t *testing.T) {
	root := branchWithWork(t)
	stubReviewer(t, "codex", `cat >/dev/null; echo r`)
	result, err := Run(context.Background(), Input{
		RepositoryRoot: root, StateRoot: t.TempDir(), BaseRef: "main",
		Selection: Selection{Reviewer: ambient.ScaffoldCodex, Mode: "cross", Reason: "test"},
		Model:     "gpt-5.6-sol",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Receipt.Effort != DefaultEffort {
		t.Fatalf("the effort was not recorded: %q", result.Receipt.Effort)
	}
	if result.Receipt.Model != "gpt-5.6-sol" {
		t.Fatalf("the model was not recorded: %q", result.Receipt.Model)
	}
}

// The marker means "the provider used its own configuration". On a review with
// no refuter there was no provider to have one, so claiming a refuter model
// would state something that never happened.
func TestAnOrdinaryReviewRecordsNoRefuterModel(t *testing.T) {
	root := branchWithWork(t)
	stubReviewer(t, "codex", `cat >/dev/null; echo r`)
	result, err := Run(context.Background(), Input{
		RepositoryRoot: root, StateRoot: t.TempDir(), BaseRef: "main",
		Selection: Selection{Reviewer: ambient.ScaffoldCodex, Mode: "cross", Reason: "test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Receipt.RefuterModel != "" {
		t.Fatalf("a review with no refuter claimed a refuter model: %q", result.Receipt.RefuterModel)
	}
}

// A gate that exits successfully while a descendant keeps the pipe open returns
// ErrWaitDelay — the shell finished before any deadline, so cancellation never
// fired. Reporting that as a refusal is doubly wrong: the gate returned success,
// and the descendant is still running.
func TestAGateThatLeavesADescendantIsNotARefusal(t *testing.T) {
	root := branchWithWork(t)
	stubReviewer(t, "codex", `cat >/dev/null; echo reviewed`)
	result, err := Run(context.Background(), Input{
		RepositoryRoot: root, StateRoot: t.TempDir(), BaseRef: "main",
		Selection: Selection{Reviewer: ambient.ScaffoldCodex, Mode: "cross", Reason: "test"},
		Gate:      "sleep 600 &",
		Deadline:  60 * time.Second,
	})
	if err != nil {
		t.Fatalf("a gate that exited successfully was treated as a failure: %v", err)
	}
	if !strings.Contains(result.Receipt.Report, "reviewed") {
		t.Fatalf("the review did not run after a passing gate: %q", result.Receipt.Report)
	}
}

// A gate that reports on stdout and then hangs must not be recorded as silent.
func TestATimedOutGateKeepsStdoutToo(t *testing.T) {
	root := branchWithWork(t)
	stubReviewer(t, "codex", `cat >/dev/null; echo r`)
	_, err := Run(context.Background(), Input{
		RepositoryRoot: root, StateRoot: t.TempDir(), BaseRef: "main",
		Selection: Selection{Reviewer: ambient.ScaffoldCodex, Mode: "cross", Reason: "test"},
		Gate:      `echo "GATE-STDOUT: checking"; sleep 600`,
		Deadline:  3 * time.Second,
	})
	if err == nil {
		t.Fatal("expected a gate deadline failure")
	}
	if !strings.Contains(err.Error(), "GATE-STDOUT") {
		t.Fatalf("the gate's stdout was discarded: %v", err)
	}
}

// A descendant that redirects its own stdio holds no pipe, so Run returns nil
// and any cleanup conditioned on a failure never happens — while the context
// watcher has already exited, so nothing later kills it either.
func TestAGateLeavesNoDescendantEvenOnCleanExit(t *testing.T) {
	root := branchWithWork(t)
	stubReviewer(t, "codex", `cat >/dev/null; echo reviewed`)
	marker := filepath.Join(t.TempDir(), "descendant-survived")

	if _, err := Run(context.Background(), Input{
		RepositoryRoot: root, StateRoot: t.TempDir(), BaseRef: "main",
		Selection: Selection{Reviewer: ambient.ScaffoldCodex, Mode: "cross", Reason: "test"},
		Gate:      `(sleep 3; touch ` + marker + `) >/dev/null 2>&1 &`,
		Deadline:  60 * time.Second,
	}); err != nil {
		t.Fatalf("a gate that exited successfully was treated as a failure: %v", err)
	}
	time.Sleep(6 * time.Second)
	if _, statErr := os.Stat(marker); statErr == nil {
		t.Fatal("a backgrounded gate descendant outlived the gate")
	}
}

// Both streams, labelled: a gate writing progress to one and its reason to the
// other loses half its evidence to a fallback that treats them as alternatives.
func TestAFailingGateKeepsBothStreams(t *testing.T) {
	root := branchWithWork(t)
	stubReviewer(t, "codex", `cat >/dev/null; echo r`)
	_, err := Run(context.Background(), Input{
		RepositoryRoot: root, StateRoot: t.TempDir(), BaseRef: "main",
		Selection: Selection{Reviewer: ambient.ScaffoldCodex, Mode: "cross", Reason: "test"},
		Gate:      `echo "GATE-PROGRESS"; echo "GATE-REASON" >&2; exit 1`,
		Deadline:  30 * time.Second,
	})
	if err == nil {
		t.Fatal("a failing gate reported success")
	}
	for _, expected := range []string{"GATE-PROGRESS", "GATE-REASON"} {
		if !strings.Contains(err.Error(), expected) {
			t.Fatalf("the failure lost %s: %v", expected, err)
		}
	}
}

// A failure that belongs to the gate must not be attributed to a reviewer that
// never started: when the deadline expires during the wait grace, the gate
// timeout is the cause.
func TestAGateTimeoutDuringTheGraceIsNotBlamedOnTheReviewer(t *testing.T) {
	root := branchWithWork(t)
	stubReviewer(t, "codex", `cat >/dev/null; echo r`)
	_, err := Run(context.Background(), Input{
		RepositoryRoot: root, StateRoot: t.TempDir(), BaseRef: "main",
		Selection: Selection{Reviewer: ambient.ScaffoldCodex, Mode: "cross", Reason: "test"},
		Gate:      `sleep 600 & exit 0`,
		Deadline:  2 * time.Second,
	})
	if err == nil {
		t.Fatal("expected a failure once the deadline expired")
	}
	if strings.Contains(err.Error(), "reviewer did not finish") {
		t.Fatalf("a gate-owned timeout was blamed on the reviewer: %v", err)
	}
}
