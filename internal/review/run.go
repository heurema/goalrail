package review

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/heurema/goalrail/internal/ambient"
)

// cancelGrace is how long a killed reviewer has before its inherited pipes stop
// being waited on. Short: by this point the process group has been signalled,
// and the only thing left to wait for is a descendant that ignored it.
const cancelGrace = 5 * time.Second

// FullEffort and FullDeadline are what a full pass runs at when the caller
// names neither.
//
// A full pass is the thoroughness pass by definition, and the loop's cheap
// defaults are wrong for it. Measured on a 5000-line branch, one range, one set
// of instructions, only the effort changed: medium reviewed clean in 272s and
// missed three real defects, two of them P1; high found them in 565s. A cheap
// full pass is a contradiction that produces a confident wrong verdict, which
// is the most expensive output this command has.
//
// A longer deadline travels with it because the two are one decision: raising
// the effort without raising the bound just moves the failure from a false
// clean verdict to a deadline. Measured at the same time — ultra reached the
// twenty-minute bound and returned nothing at all.
//
// The bound is a safety net, not a plan. Observed full passes at this effort:
// 272s, 500s and 565s, with one run of the very same range exceeding 2700s — an
// order of magnitude apart on identical inputs. Duration is not predictable from
// diff size or effort, so the number is set at roughly three times the observed
// median rather than above the worst tail: at this spread a cheap failure that
// can be retried beats a generous wait that cannot be undone. The first choice
// here was 45 minutes, picked by eye and corrected once the spread was measured.
// Evidence: openspec/changes/pre-pr-review-v0/evidence/effort-experiment-2026-08-01.md
const (
	FullEffort   = "high"
	FullDeadline = 25 * time.Minute
)

// DefaultDeadline bounds one review.
//
// A bound is required and cannot be per-vendor: `claude` carries no turn limit
// in the version measured here, whatever its documentation elsewhere suggests,
// so a ceiling built out of vendor flags would be a ceiling only where the flag
// happens to exist. A deadline holds for every reviewer and cannot drift with
// an interface.
const DefaultDeadline = 20 * time.Minute

// defaultModels is the model each reviewer runs on when the caller names none.
//
// Stated rather than inherited where there is evidence to state something: a
// Claude review invoked from a session inherits that session's model, chosen for
// authoring rather than reviewing, and that inheritance was measured failing —
// a review died in four seconds on "out of usage credits" for a model the
// session happened to be using, on a path that had never once run.
//
// Codex has no entry, and the absence is deliberate rather than a limitation.
// An earlier version of this comment claimed its model could not be named on the
// command line; that was asserted without checking and is false — `-c model=`
// is accepted. What is missing is not the capability but the evidence: nothing
// measured says the configured model is wrong for reviewing, and pinning a
// vendor model identifier here would age into a wrong default nobody revisits.
// The caller can still name one, and it is passed.
var defaultModels = map[ambient.Scaffold]string{
	ambient.ScaffoldClaudeCode: "opus",
}

// DefaultModel reports the model one reviewer runs on absent a caller's choice.
// An empty string means the vendor's own default is left alone.
func DefaultModel(reviewer ambient.Scaffold) string { return defaultModels[reviewer] }

// DefaultEffort is the reasoning effort a review runs at when the caller names
// none.
//
// Moderate on purpose. A review is a step inside a loop that runs again after
// every fix, so its cost is paid on every round; the highest setting turns a
// loop into a batch job. Raising it is one flag away for the change that
// deserves it.
const DefaultEffort = "medium"

// ErrGateRefused reports a configured budget gate that declined the review.
var ErrGateRefused = errors.New("the configured review gate refused")

// Input is one review to perform.
type Input struct {
	RepositoryRoot string
	StateRoot      string

	// BaseRef is what the branch is reviewed against. Empty asks ResolveBase to
	// discover one unambiguous default from local remote-HEAD metadata.
	BaseRef string

	// Author is the provider that wrote the change. Detection fills it; an
	// explicit override replaces it.
	Author ambient.Scaffold

	// Selection is the reviewer together with mode and reason.
	Selection Selection

	// Full forces a whole-branch review even where a previous receipt would
	// make the round incremental.
	Full bool

	// Refute, when set, runs a second fresh session that receives the previous
	// report and tries to refute its findings. The trigger is the caller's:
	// findings exist and they are about to act on them.
	Refute bool

	// Deadline bounds the review. Zero means DefaultDeadline.
	Deadline time.Duration

	// Model is the reviewer's model. Empty means DefaultModel where the provider
	// accepts one on the command line, and the vendor's own default otherwise.
	Model string

	// Effort is the reviewer's reasoning effort. Empty means DefaultEffort:
	// what a review in a loop needs, rather than what the machine's interactive
	// configuration happens to say.
	Effort string

	// Gate is a command run before any reviewer is invoked. Empty means no
	// gate, which is the ordinary case: absence of a gate is a choice the user
	// already made by not making one, and refusing on it would reintroduce the
	// consent step this command exists without.
	Gate string

	Now func() time.Time
}

// Result reports what one review produced.
type Result struct {
	Receipt                  Receipt
	ReceiptPath              string
	InstructionsMaterialized bool
}

// Run performs one review and records it.
//
// It performs exactly one review. The loop that re-runs it after fixes, with
// the round ceiling that loop declared before its first round, belongs to the
// caller: the thing deciding "nothing material remains" has to be something
// that reads the report, and reading the report is precisely what this package
// must never do.
func Run(ctx context.Context, input Input) (Result, error) {
	now := input.Now
	if now == nil {
		now = time.Now
	}

	branch, err := CurrentBranch(input.RepositoryRoot)
	if err != nil {
		return Result{}, err
	}
	if branch == "" {
		return Result{}, fmt.Errorf("HEAD is detached, so there is no branch to file a review against")
	}

	base, err := ResolveBase(input.RepositoryRoot, input.BaseRef)
	if err != nil {
		return Result{}, err
	}
	baseCommit := base.Commit
	headCommit, err := Resolve(input.RepositoryRoot, "HEAD")
	if err != nil {
		return Result{}, err
	}
	if baseCommit == headCommit {
		return Result{}, fmt.Errorf("the branch has no changes against %s, so there is nothing to review", base.Ref)
	}

	// A round costs what the round is about: where a previous receipt exists,
	// the reviewed range starts at its head, and the chain of receipts proves
	// cumulative coverage. Reviewing the whole branch every round was measured
	// into the failure this prevents — each round repaid the full price of all
	// previous ones. The full pass stays one flag away.
	// An unmeasurable tree stays unmeasured: the review still runs, and the
	// receipt says it does not know rather than claiming a dirty tree.
	var treeClean *bool
	if measured, treeErr := WorkingTreeClean(input.RepositoryRoot); treeErr == nil {
		treeClean = &measured
	}

	reviewedBase := baseCommit
	unchangedRounds := 0
	previous, hasPrevious, previousErr := ReadReceipt(input.StateRoot, input.RepositoryRoot, branch)
	if previousErr != nil {
		hasPrevious = false
	}
	if hasPrevious {
		// Stalemate: two rounds over identical state — the same head, and
		// neither round carrying work of the author's. The previous round
		// produced findings nothing was done about, and another spends without
		// converging. Requiring the previous round to have been clean too is
		// what keeps a discarded fix from reading as a stalemate: work moved,
		// even though nothing landed. Measured, never read out of a report;
		// what to do about it is the caller's loop policy.
		//
		// Deliberately outside the incremental branch: --full changes the
		// reviewed scope, not whether anything moved.
		if previous.HeadCommit == headCommit &&
			treeClean != nil && *treeClean &&
			previous.TreeCleanAtReview != nil && *previous.TreeCleanAtReview {
			unchangedRounds = previous.UnchangedRounds + 1
		}
	}
	// Narrowing is only sound against the same base the chain was built on: a
	// receipt taken against another ref proves nothing about the commits this
	// base newly brings into range. A legacy receipt without the reviewed-range
	// digest cannot anchor a chain link that proves its bytes, so it does not
	// narrow either.
	if !input.Full && hasPrevious &&
		previous.ReviewedDiffSHA256 != "" &&
		previous.BaseCommit == baseCommit &&
		previous.HeadCommit != headCommit {
		if _, resolveErr := Resolve(input.RepositoryRoot, previous.HeadCommit); resolveErr == nil {
			reviewedBase = previous.HeadCommit
		}
	}

	// Freeze and render the immutable ranges before creating instructions,
	// running the gate or invoking a provider. The reviewed-range bytes are
	// rendered exactly once and then reused for transport and proof. The full
	// branch remains a separate staleness measurement on incremental rounds.
	fullDiff, err := renderCanonicalDiff(input.RepositoryRoot, baseCommit, headCommit)
	if err != nil {
		return Result{}, err
	}
	reviewedDiff := fullDiff
	if reviewedBase != baseCommit {
		reviewedDiff, err = renderCanonicalDiff(input.RepositoryRoot, reviewedBase, headCommit)
		if err != nil {
			return Result{}, err
		}
	}
	diffDigest := digest(fullDiff)
	reviewedDigest := digest(reviewedDiff)

	instructions, materialized, err := EnsureInstructions(input.RepositoryRoot)
	if err != nil {
		return Result{}, err
	}

	// The caller's word always wins; these are the defaults for a pass nobody
	// parameterized.
	effort := strings.TrimSpace(input.Effort)
	if effort == "" {
		effort = DefaultEffort
		if input.Full {
			effort = FullEffort
		}
	}
	// A model name belongs to one provider and cannot be carried to another, so
	// it is resolved per invocation rather than once. An explicitly named model
	// applies to the reviewer the caller is targeting; a refuter of a different
	// provider falls back to its own default, because passing a name from the
	// other vendor would fail the second, unpaid-for invocation after the first
	// one has already been paid.
	modelFor := func(reviewer ambient.Scaffold) string {
		if named := strings.TrimSpace(input.Model); named != "" && reviewer == input.Selection.Reviewer {
			return named
		}
		return DefaultModel(reviewer)
	}
	deadline := input.Deadline
	if deadline <= 0 {
		deadline = DefaultDeadline
		if input.Full {
			deadline = FullDeadline
		}
	}
	// The bound is created before the gate, because it is advertised as a bound
	// on the whole review and a gate is part of the review. A gate that blocks
	// under an unbounded context makes that promise false.
	bounded, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()

	// The gate runs before anything is spawned, because its whole purpose is to
	// stop the spend rather than to report it afterwards.
	if strings.TrimSpace(input.Gate) != "" {
		if gateErr := runGate(bounded, input.RepositoryRoot, input.Gate); gateErr != nil {
			return Result{InstructionsMaterialized: materialized}, gateErr
		}
	}
	started := now()
	rangeSpec := reviewedBase + "..." + headCommit
	// The Claude reviewer has no shell, so the exact canonical bytes already
	// measured above travel in its prompt. Codex keeps its native range-based
	// custom-instruction invocation and reads the immutable commit range itself.
	invokeInstructions := instructions
	if input.Selection.Reviewer == ambient.ScaffoldClaudeCode {
		invokeInstructions = append(append(append([]byte{}, instructions...),
			[]byte("\n\n--- DIFF UNDER REVIEW ---\n")...), reviewedDiff...)
	}
	report, err := invoke(bounded, input.Selection.Reviewer, input.RepositoryRoot, rangeSpec, effort, modelFor(input.Selection.Reviewer), invokeInstructions)
	if err != nil {
		// No receipt: one describing a review that did not happen is worse than
		// none, because it reads as done.
		return Result{InstructionsMaterialized: materialized}, err
	}

	mode, modeReason := input.Selection.Mode, input.Selection.Reason
	var refutation, refuter, refuterModel string
	if input.Refute {
		// The refutation is a second paid run: the gate's answer may have
		// changed while the first one spent, so it is asked again.
		if strings.TrimSpace(input.Gate) != "" {
			if gateErr := runGate(bounded, input.RepositoryRoot, input.Gate); gateErr != nil {
				return Result{InstructionsMaterialized: materialized}, gateErr
			}
		}
		// The refuter is a fresh session — the other runnable provider where one
		// exists, the same one otherwise. Passing the report verbatim is
		// transport, not parsing; which findings survive is the reader's call.
		refuterSelection, refuteErr := SelectReviewer(input.Selection.Reviewer, ambient.SupportedScaffolds(), nil)
		if refuteErr != nil {
			return Result{InstructionsMaterialized: materialized}, refuteErr
		}
		// The refuter draws from the same committed instructions the reviewer
		// did — the receipt hashes those instructions, and a refuter that never
		// saw the repository's own rules could refute findings those rules
		// require.
		// The whole policy, including the refute directives, comes from the
		// committed file the receipt hashes; the binary adds the verbatim report
		// and the exact canonical diff bytes whose digest the receipt records.
		refutePrompt := append(append(append(append([]byte{}, instructions...),
			[]byte("\n\n--- REPORT UNDER CHALLENGE ---\n")...), []byte(report)...),
			[]byte("\n\n--- DIFF UNDER REVIEW ---\n")...)
		refutePrompt = append(refutePrompt, reviewedDiff...)
		refutation, refuteErr = invoke(bounded, refuterSelection.Reviewer, input.RepositoryRoot, rangeSpec, effort, modelFor(refuterSelection.Reviewer), refutePrompt)
		if refuteErr != nil {
			return Result{InstructionsMaterialized: materialized}, refuteErr
		}
		refuter = string(refuterSelection.Reviewer)
		refuterModel = modelFor(refuterSelection.Reviewer)
		mode = "refute"
		modeReason = input.Selection.Reason + "; refuted by " + refuter + " in a fresh session"
	}

	refuterModelRecorded := ""
	if refuter != "" {
		refuterModelRecorded = recordedModel(refuterModel)
	}

	receipt := Receipt{
		Schema:             ReceiptSchema,
		Repository:         input.RepositoryRoot,
		Branch:             branch,
		BaseRef:            base.Ref,
		BaseCommit:         baseCommit,
		HeadCommit:         headCommit,
		ReviewedBase:       reviewedBase,
		ReviewedDiffSHA256: reviewedDigest,
		DiffSHA256:         diffDigest,
		Mode:               mode,
		ModeReason:         modeReason,
		Reviewer:           string(input.Selection.Reviewer),
		Author:             string(input.Author),
		ReviewedAt:         now().UTC().Format(time.RFC3339),
		Effort:             effort,
		Model:              recordedModel(modelFor(input.Selection.Reviewer)),
		// Only where a refuter actually ran: the marker means "the provider used
		// its own configuration", and on an ordinary review there was no
		// provider to have one.
		RefuterModel:       refuterModelRecorded,
		UnchangedRounds:    unchangedRounds,
		TreeCleanAtReview:  treeClean,
		DurationSeconds:    int64(now().Sub(started) / time.Second),
		Report:             report,
		ReportSHA256:       digest([]byte(report)),
		Refutation:         refutation,
		RefutationSHA256:   refutationDigest(refutation),
		Refuter:            refuter,
		InstructionsSHA256: digest(instructions),
	}
	path, err := WriteReceipt(input.StateRoot, receipt)
	if err != nil {
		return Result{InstructionsMaterialized: materialized}, err
	}
	return Result{Receipt: receipt, ReceiptPath: path, InstructionsMaterialized: materialized}, nil
}

// runGate executes the configured gate command through the user's shell.
//
// A shell on purpose: the gate is the user's own command, frequently a pipeline
// or a script with arguments, and demanding it be a bare executable would make
// the ordinary case awkward enough to skip.
func runGate(ctx context.Context, repositoryRoot, gate string) error {
	shell := os.Getenv("SHELL")
	if strings.TrimSpace(shell) == "" {
		shell = "/bin/sh"
	}
	command := exec.CommandContext(ctx, shell, "-c", gate)
	command.Dir = repositoryRoot
	// The same tree-wide cancellation the reviewers get. A gate is routinely a
	// pipeline or a script, so killing the shell alone leaves a descendant
	// holding the pipe and Run waiting past the deadline — the bound would be
	// advertised and not held, which is the defect this whole path just fixed
	// one layer up.
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Cancel = func() error {
		if command.Process == nil {
			return nil
		}
		return syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
	}
	command.WaitDelay = cancelGrace
	// Both streams: a gate that reports on stdout and then hangs would otherwise
	// be recorded as having produced nothing.
	stdout := &tailBuffer{limit: 64 * 1024}
	stderr := &tailBuffer{limit: 64 * 1024}
	command.Stdout = stdout
	command.Stderr = stderr

	runErr := command.Run()
	// The group is killed unconditionally once the shell is done, not only when
	// waiting produced an error. A descendant that redirects its own stdio holds
	// no pipe, so Run returns nil and any cleanup conditioned on a failure never
	// happens — while the context watcher has already exited, so nothing later
	// will kill it either. The gate is finished; anything still alive in its
	// group is a leak whatever Run returned.
	if command.Process != nil {
		_ = syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
	}
	// A gate that exited successfully while a descendant kept the pipe open
	// returns ErrWaitDelay. Reporting that as a refusal is wrong twice over: the
	// gate returned success, and the descendant was the only thing waiting on.
	// Only while the review still has time. When the deadline expires during the
	// grace period, clearing the error would skip the gate-timeout branch and
	// launch the reviewer on an already-dead context — reporting that the
	// reviewer timed out when it never started, which names the wrong cause for
	// a failure that belongs to the gate.
	if errors.Is(runErr, exec.ErrWaitDelay) && ctx.Err() == nil &&
		command.ProcessState != nil && command.ProcessState.Success() {
		runErr = nil
	}
	if err := runErr; err != nil {
		// Both streams, labelled — the reviewer's timeout path already does
		// this, and a gate that writes progress to one and a reason to the other
		// loses half its evidence to a fallback that treats them as
		// alternatives.
		detail := labelledStreams(stdout.String(), stderr.String())
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			// A gate the deadline killed never returned a verdict. Reporting it
			// as a refusal tells automation the budget denied the review, which
			// is a different fact with a different response.
			if detail == "" {
				detail = "(it emitted no output before the deadline)"
			}
			return fmt.Errorf("the review gate (%s) did not finish within the review deadline; what it had produced:\n%s",
				gate, bounded_(detail))
		}
		if detail == "" {
			detail = err.Error()
		}
		return fmt.Errorf("%w (%s): %s", ErrGateRefused, gate, bounded_(detail))
	}
	return nil
}

// prompt is what a reviewer is told: the repository's instructions, then the
// range to read.
//
// The range travels in the prose rather than in a flag, because `codex review`
// refuses `--base` together with custom instructions — its preset scopes and a
// caller's own instructions are alternatives, not a pair. That was found by
// running it, not by reading its help, which lists both without saying so. The
// instructions are not optional here: they carry what the findings ratchet has
// promoted, so a review without them is a review of the wrong things.
func prompt(rangeSpec string, instructions []byte) string {
	return string(instructions) +
		"\n\nReview exactly the range `" + rangeSpec + "` — not the working tree.\n" +
		"Use git to read that diff. You may read any file for context. Do not modify anything.\n"
}

// refutationDigest digests a refutation, or nothing where there is none.
func refutationDigest(refutation string) string {
	if refutation == "" {
		return ""
	}
	return digest([]byte(refutation))
}

// reviewCommand builds one reviewer's invocation.
//
// Separate from running it so the argument shape is testable without spawning
// anything: the first live run of this feature failed on an argument
// combination its vendor documents as legal, and a construction nobody can
// inspect is one that breaks the same way twice.
func reviewCommand(reviewer ambient.Scaffold, baseRef, effort, model string, instructions []byte) (name string, arguments []string, stdin string, err error) {
	switch reviewer {
	case ambient.ScaffoldCodex:
		// `-` reads the instructions from stdin. No scope flag: it cannot be
		// combined with them.
		//
		// The reasoning effort is stated rather than inherited. A review inside
		// a loop has a different latency budget from the interactive session
		// whose configuration file would otherwise decide it — measured here as
		// a review still running after ten minutes because the machine's config
		// named the highest setting for interactive work. The other reviewer
		// needs no equivalent: its effort arrives in the environment, which is
		// already stripped so the reviewer does not inherit the author's
		// session. A configuration file is out of that reach.
		codexArguments := []string{
			"-c", "model_reasoning_effort=" + effort,
		}
		if strings.TrimSpace(model) != "" {
			// Accepted here as well: the claim that it was not is what let this
			// asymmetry survive its own review.
			codexArguments = append(codexArguments, "-c", "model="+model)
		}
		return "codex", append(codexArguments,
			// A reviewer that can edit the work is no longer reviewing it, and
			// the other reviewer already has no writing tool at all. Asking in
			// the prompt is not the same as removing the permission: the
			// read-only boundary here lost to `git *` and then to
			// `git diff --output=` before it was made structural. This takes
			// the capability away rather than requesting restraint, and it
			// overrides whatever the machine's own sandbox setting says.
			"-c", "sandbox_mode=read-only",
			"review", "-",
		), prompt(baseRef, instructions), nil
	case ambient.ScaffoldClaudeCode:
		// Only the tools a reviewer needs, and never an editing one: a reviewer
		// that can change the work is no longer reviewing it.
		//
		// The git allowance is per subcommand rather than `git *`. That wildcard
		// admitted `git reset`, `git checkout` and `git clean`, and `git -c
		// alias.x='!sh -c ...'` would have run an arbitrary shell through it —
		// a read-only contract that the permission it granted did not keep.
		// No shell at all. The git allowlist was defeated twice — `git *`, then
		// `git diff --output=<path>` through the per-subcommand rule — and a
		// race against git's flag surface is one this boundary keeps losing.
		// The diff travels in the prompt instead; reading files stays allowed.
		claudeArguments := []string{
			"-p", "--output-format", "text",
			"--effort", effort,
		}
		if strings.TrimSpace(model) != "" {
			claudeArguments = append(claudeArguments, "--model", model)
		}
		return "claude", append(claudeArguments,
			"--allowed-tools", "Read", "Grep", "Glob",
			"--disallowed-tools", "Bash", "Edit", "Write", "NotebookEdit",
		), prompt(baseRef, instructions), nil
	default:
		return "", nil, "", fmt.Errorf("no review invocation is defined for %q", reviewer)
	}
}

// invoke runs one reviewer through its vendor's own non-interactive interface.
//
// Minimum documented flags on purpose. Neither CLI can be pinned by Goalrail,
// so every flag is a surface that can drift under it; a vendor's refusal is
// surfaced to the caller unchanged rather than translated, because a wrapper
// that smooths over a changed interface converts a loud break into a quiet one.
func invoke(ctx context.Context, reviewer ambient.Scaffold, repositoryRoot, baseRef, effort, model string, instructions []byte) (string, error) {
	name, arguments, stdin, err := reviewCommand(reviewer, baseRef, effort, model, instructions)
	if err != nil {
		return "", err
	}
	command := exec.CommandContext(ctx, name, arguments...)
	command.Stdin = strings.NewReader(stdin)
	command.Dir = repositoryRoot

	// A deadline that only kills the direct child is not a deadline. The
	// reviewers are wrappers — `codex` is a Node process that launches a native
	// binary — and the grandchild holds the same stdout and stderr, so killing
	// the parent leaves those pipes open and Wait blocks forever. Measured: a
	// review with a twenty-minute deadline was still running at fifty-three
	// minutes with its reviewer already gone.
	//
	// Two things fix it together: the child gets its own process group so the
	// whole tree can be signalled, and WaitDelay stops waiting on inherited
	// pipes shortly after the kill rather than until whoever holds them exits.
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Cancel = func() error {
		if command.Process == nil {
			return nil
		}
		return syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
	}
	command.WaitDelay = cancelGrace
	// The reviewer must not inherit the author's session identity. A spawned
	// agent that reads the parent's markers believes it is a continuation of the
	// session that wrote the code, which is exactly the shared context this
	// review exists to avoid.
	command.Env = strippedEnvironment(os.Environ(), AuthorMarkers())

	// The streams are bounded while being collected: the receipt bound checked
	// at write time cannot protect the running process from a reviewer that
	// floods stdout, and an unbounded buffer turns that into memory exhaustion.
	// Only the report refuses on overflow. stderr is a reviewer's running
	// commentary — Codex streams its whole session transcript there, over a
	// megabyte on an ordinary large branch — and refusing that write killed the
	// first real field review. Diagnostics keep a tail; they never fail a run.
	stdout := &boundedBuffer{limit: receiptBound}
	stderr := &tailBuffer{limit: 64 * 1024}
	command.Stdout = stdout
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			// What the reviewer had produced travels with the timeout. A bare
			// "did not finish" cannot distinguish working-but-slow from stuck,
			// so an expensive failure teaches nothing and the next decision —
			// raise the bound, or stop trying — has no evidence behind it.
			// Both streams, not one instead of the other: a reviewer that logs
			// progress on stderr may still have produced partial findings on
			// stdout, and that is exactly the evidence a timeout exists to
			// carry.
			progress := labelledStreams(stdout.String(), stderr.String())
			if progress == "" {
				// Observed, not diagnosed. A reviewer may buffer everything
				// until it finishes, so silence proves no output was emitted —
				// not that anything stopped. Naming a cause here would state an
				// unknown as a measurement.
				progress = "(it emitted no output before the deadline)"
			}
			return "", fmt.Errorf("the %s reviewer did not finish within the review deadline; what it had produced:\n%s",
				reviewer, bounded_(progress))
		}
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = strings.TrimSpace(stdout.String())
		}
		if detail == "" {
			detail = err.Error()
		}
		// The vendor's message reaches the caller, but a megabyte of session
		// transcript is not a message. Different CLIs put the reason at
		// different ends — a panic leads, an exit summary trails — so both a
		// bounded head and a bounded tail survive, and the process's own exit
		// error is always stated because no truncation can lose it.
		detail = bounded_(detail)
		return "", fmt.Errorf("the %s reviewer did not complete (%v): %s", reviewer, err, detail)
	}
	report := stdout.String()
	if strings.TrimSpace(report) == "" {
		return "", fmt.Errorf("the %s reviewer produced no report", reviewer)
	}
	return report, nil
}

// boundedBuffer collects up to limit bytes and refuses the rest, so a flooding
// child fails the review instead of exhausting memory.
type boundedBuffer struct {
	buffer bytes.Buffer
	limit  int
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	if b.buffer.Len()+len(p) > b.limit {
		return 0, fmt.Errorf("output exceeded the %d-byte bound", b.limit)
	}
	return b.buffer.Write(p)
}

func (b *boundedBuffer) String() string { return b.buffer.String() }

// tailBuffer keeps a bounded head and a bounded tail and never fails a write:
// it exists for diagnostics, and diagnostics must not be able to kill the
// thing they describe. Both ends survive because vendors disagree about where
// the reason goes — a panic leads, an exit summary trails — and a buffer that
// keeps only one end silently discards the other kind of cause.
type tailBuffer struct {
	head   []byte
	tail   []byte
	elided int
	limit  int
}

func (b *tailBuffer) Write(p []byte) (int, error) {
	// The full length is reported whatever was kept: a diagnostic buffer that
	// under-reports a write makes the writer see a broken pipe and turns the
	// diagnostics into the failure again, one layer up.
	written := len(p)
	if len(b.head) < b.limit {
		take := min(b.limit-len(b.head), len(p))
		b.head = append(b.head, p[:take]...)
		p = p[take:]
	}
	if len(p) > 0 {
		b.tail = append(b.tail, p...)
		if len(b.tail) > b.limit {
			b.elided += len(b.tail) - b.limit
			b.tail = b.tail[len(b.tail)-b.limit:]
		}
	}
	return written, nil
}

func (b *tailBuffer) String() string {
	if len(b.tail) == 0 {
		return string(b.head)
	}
	if b.elided == 0 {
		return string(b.head) + string(b.tail)
	}
	return string(b.head) + fmt.Sprintf("\n… [%d bytes elided] …\n", b.elided) + string(b.tail)
}

// recordedModel names what the receipt can prove. An unpinned provider used
// some model from its own configuration; the receipt says that rather than
// leaving a field that reads as "none".
func recordedModel(resolved string) string {
	if strings.TrimSpace(resolved) == "" {
		return "(provider configuration, not pinned by Goalrail)"
	}
	return resolved
}

// bounded_ keeps a message a message. Both failure paths use it, so a timeout
// cannot dump a transcript the ordinary failure path is forbidden from
// dumping — different ends of the same output, one rule.
func bounded_(detail string) string {
	const limit = 4096
	if len(detail) <= limit {
		return detail
	}
	return detail[:1024] +
		fmt.Sprintf("\n… [%d bytes elided] …\n", len(detail)-limit) +
		detail[len(detail)-3072:]
}

// labelledStreams keeps whatever each stream carried, labelled. Both failure
// paths use it: treating the streams as alternatives loses half the evidence
// whenever a process writes progress to one and its reason to the other.
func labelledStreams(stdout, stderr string) string {
	var parts []string
	if out := strings.TrimSpace(stdout); out != "" {
		parts = append(parts, "stdout:\n"+out)
	}
	if errOut := strings.TrimSpace(stderr); errOut != "" {
		parts = append(parts, "stderr:\n"+errOut)
	}
	return strings.Join(parts, "\n")
}
