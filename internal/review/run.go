package review

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
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
// Evidence: openspec/changes/archive/2026-08-04-pre-pr-review-v0/evidence/effort-experiment-2026-08-01.md
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

// DefaultProgressBound bounds silence rather than total time.
//
// A deadline bounds what a review costs; it does not bound what it wastes. A
// reviewer that stops entirely pays the whole deadline and returns nothing,
// and from outside the process it looks exactly like one still thinking. Two
// full passes on one branch were measured that way — 25 minutes, then 50 —
// each recording no reviewer activity whatever for the remainder after a single
// blocking call to a machine-configured integration. Doubling the deadline
// diagnosed nothing, which is the point: the failure is silence, so silence is
// what has to be measured.
//
// Five minutes is deliberately far above any healthy quiet period observed. In
// the same sessions, a working reviewer emitted something every few seconds and
// the largest gap between its actions was 16 seconds. The margin is generous
// because the cost of the two errors is not symmetric: stopping a working
// reviewer wastes one round, while failing to stop a stopped one wastes the
// whole budget and returns nothing. The caller can name a different bound.
const DefaultProgressBound = 5 * time.Minute

// ErrReviewerStalled reports a reviewer that produced nothing for longer than
// the progress bound and was stopped. It is not a deadline overrun and not a
// review: a stalled reviewer reviewed nothing, and no receipt describes it.
var ErrReviewerStalled = errors.New("the reviewer stopped producing output")

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

	// ProgressBound stops a reviewer that produces nothing for this long. Zero
	// means DefaultProgressBound, reduced where a short deadline would leave it
	// no room to fire.
	ProgressBound time.Duration

	// WithoutIntegrations names provider integrations to remove from the
	// reviewer's session. Goalrail renders each name into the provider's own
	// removal syntax and knows none of them: which integration is hostile is the
	// caller's knowledge of their own machine.
	WithoutIntegrations []string

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

	deadline := input.Deadline
	if deadline <= 0 {
		deadline = DefaultDeadline
		if input.Full {
			deadline = FullDeadline
		}
	}
	progressBound, err := resolveProgressBound(input.ProgressBound, deadline)
	if err != nil {
		return Result{}, err
	}
	// Isolation is refused before anything is spent, and for every provider this
	// review could invoke rather than only the first. A refute round is a second
	// paid invocation; discovering there that its provider cannot express the
	// removal would refuse after the reviewer had already been paid for.
	if len(input.WithoutIntegrations) > 0 {
		if err := validateIntegrations(input.WithoutIntegrations); err != nil {
			return Result{}, err
		}
		providers := []ambient.Scaffold{input.Selection.Reviewer}
		if input.Refute {
			if refuterSelection, refuteErr := SelectReviewer(input.Selection.Reviewer, ambient.SupportedScaffolds(), nil); refuteErr == nil {
				providers = append(providers, refuterSelection.Reviewer)
			}
		}
		for _, provider := range providers {
			if !supportsIntegrationRemoval(provider) {
				return Result{}, fmt.Errorf(
					"the %s reviewer's interface removes either all integrations or none, so it cannot remove one by name; "+
						"run without isolation, or with a reviewer whose interface can express it", provider)
			}
		}
	}

	// Canonical rendering is review work too. Start the advertised deadline
	// before it and pass the bound into Git, so a large diff cannot spend past
	// the caller's limit before the gate or reviewer even starts.
	bounded, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()

	// Freeze and render the immutable ranges before creating instructions,
	// running the gate or invoking a provider. The reviewed-range bytes are
	// rendered exactly once and then reused for transport and proof. The full
	// branch remains a separate staleness measurement on incremental rounds.
	fullDiff, err := renderCanonicalDiff(bounded, input.RepositoryRoot, baseCommit, headCommit)
	if err != nil {
		return Result{}, err
	}
	if len(fullDiff) == 0 {
		return Result{}, fmt.Errorf("the branch has no changes against %s, so there is nothing to review", base.Ref)
	}
	reviewedDiff := fullDiff
	if reviewedBase != baseCommit {
		reviewedDiff, err = renderCanonicalDiff(bounded, input.RepositoryRoot, reviewedBase, headCommit)
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
	report, err := invoke(bounded, invocation{
		reviewer: input.Selection.Reviewer, repositoryRoot: input.RepositoryRoot, baseRef: rangeSpec,
		effort: effort, model: modelFor(input.Selection.Reviewer), instructions: invokeInstructions,
		without: input.WithoutIntegrations, progressBound: progressBound,
	})
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
		refutation, refuteErr = invoke(bounded, invocation{
			reviewer: refuterSelection.Reviewer, repositoryRoot: input.RepositoryRoot, baseRef: rangeSpec,
			effort: effort, model: modelFor(refuterSelection.Reviewer), instructions: refutePrompt,
			without: input.WithoutIntegrations, progressBound: progressBound,
		})
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
func reviewCommand(reviewer ambient.Scaffold, baseRef, effort, model string, instructions []byte, without []string) (name string, arguments []string, stdin string, err error) {
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
		// The caller names an integration; this renders it into the provider's
		// own documented syntax and adds nothing else. A name is all that
		// crosses the boundary, so nothing a caller supplies can reach the
		// sandbox, the model, the effort or the recorded identity below.
		for _, integration := range without {
			codexArguments = append(codexArguments, "-c", "mcp_servers."+integration+".enabled=false")
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
		if len(without) > 0 {
			// Its interface offers all-or-nothing, so honouring "remove this one"
			// would mean removing more than was asked. Refusing says so; doing
			// nothing would read as isolation that happened.
			return "", nil, "", fmt.Errorf(
				"the claude-code reviewer's interface removes either all integrations or none, so it cannot remove one by name")
		}
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
// invocation is one reviewer run. It is a struct because the argument list had
// outgrown the point where positional strings stay readable, and two of them are
// now boundaries rather than settings.
type invocation struct {
	reviewer       ambient.Scaffold
	repositoryRoot string
	baseRef        string
	effort         string
	model          string
	instructions   []byte
	without        []string
	progressBound  time.Duration
}

func invoke(ctx context.Context, inv invocation) (string, error) {
	reviewer := inv.reviewer
	name, arguments, stdin, err := reviewCommand(reviewer, inv.baseRef, inv.effort, inv.model, inv.instructions, inv.without)
	if err != nil {
		return "", err
	}
	// A second cancel sits under the deadline so a stalled reviewer is killed by
	// the same machinery, and the two causes stay distinguishable afterwards:
	// the parent context still reports the deadline, and only the watchdog sets
	// the stall flag.
	runCtx, stopForStall := context.WithCancel(ctx)
	defer stopForStall()
	command := exec.CommandContext(runCtx, name, arguments...)
	command.Stdin = strings.NewReader(stdin)
	command.Dir = inv.repositoryRoot

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
	// Arrival time is the liveness signal. Content is never inspected for it:
	// what a reviewer says is the reader's business, and whether it is saying
	// anything at all is the only thing observable from out here.
	activity := &atomic.Int64{}
	activity.Store(time.Now().UnixNano())
	command.Stdout = &activityWriter{inner: stdout, last: activity}
	command.Stderr = &activityWriter{inner: stderr, last: activity}

	if err := command.Start(); err != nil {
		return "", fmt.Errorf("the %s reviewer did not start: %w", reviewer, err)
	}
	stalled := watchProgress(runCtx, activity, inv.progressBound, stopForStall)
	waitErr := command.Wait()
	stopForStall()

	if waitErr != nil {
		if stalled.Load() {
			// The tail travels with the stall for the same reason it travels
			// with a timeout, and more urgently: reconstructing what a stopped
			// reviewer last did otherwise means reading provider session files,
			// which is exactly the work this saves.
			progress := labelledStreams(stdout.String(), stderr.String())
			if progress == "" {
				progress = "(it emitted no output at all)"
			}
			return "", fmt.Errorf("%w: the %s reviewer produced nothing for %s and was stopped; what it had produced:\n%s",
				ErrReviewerStalled, reviewer, inv.progressBound, bounded_(progress))
		}
		err := waitErr
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

// watchProgress stops a reviewer that has gone silent, and reports which cause
// fired. It watches arrival times rather than content: a reviewer streaming
// anything at all is working, and one streaming nothing cannot be told apart
// from a stopped one by any signal available outside its process.
func watchProgress(ctx context.Context, activity *atomic.Int64, bound time.Duration, stop func()) *atomic.Bool {
	stalled := &atomic.Bool{}
	// A tenth of the bound is often enough to be prompt and rare enough to cost
	// nothing, clamped so a very small or very large bound stays sane.
	interval := bound / 10
	if interval < 100*time.Millisecond {
		interval = 100 * time.Millisecond
	}
	if interval > 10*time.Second {
		interval = 10 * time.Second
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if time.Since(time.Unix(0, activity.Load())) < bound {
					continue
				}
				// Order matters: the flag is set before the kill, so the wait
				// that the kill releases always sees the cause already recorded.
				stalled.Store(true)
				stop()
				return
			}
		}
	}()
	return stalled
}

// activityWriter records when output last arrived and passes the bytes on
// unchanged. It never inspects, buffers, or delays them: a liveness probe that
// can alter what it observes is a second failure mode.
type activityWriter struct {
	inner io.Writer
	last  *atomic.Int64
}

func (w *activityWriter) Write(p []byte) (int, error) {
	w.last.Store(time.Now().UnixNano())
	return w.inner.Write(p)
}

// resolveProgressBound settles the silence bound against the deadline it sits
// under.
//
// A caller's explicit bound at or past the deadline can never fire, so it is
// refused rather than accepted as a bound that does nothing. A defaulted bound
// is fitted instead of refused: a caller who shortens the deadline is not asking
// about silence at all, and refusing their review over a default they never
// named would be this command inventing a policy.
func resolveProgressBound(requested, deadline time.Duration) (time.Duration, error) {
	if requested > 0 {
		if requested >= deadline {
			return 0, fmt.Errorf(
				"the progress bound (%s) must be shorter than the review deadline (%s), or it can never fire",
				requested, deadline)
		}
		return requested, nil
	}
	if DefaultProgressBound < deadline {
		return DefaultProgressBound, nil
	}
	return deadline / 2, nil
}

// validateIntegrations checks the only caller input that reaches a reviewer's
// invocation.
//
// Bounded identifiers, refused before anything is spent. The rendering around
// them is fixed, so this is not a filter protecting a boundary — it is what
// keeps a name a name, and it fails loudly rather than passing something
// structural into an argument list.
func validateIntegrations(names []string) error {
	const maxIntegrationName = 64
	for _, name := range names {
		if name == "" {
			return fmt.Errorf("an integration to remove must be named; an empty name removes nothing")
		}
		if len(name) > maxIntegrationName {
			return fmt.Errorf("the integration name %q exceeds %d characters", bounded_(name), maxIntegrationName)
		}
		for index, character := range name {
			switch {
			case character >= 'a' && character <= 'z',
				character >= 'A' && character <= 'Z',
				character >= '0' && character <= '9':
			case character == '-' || character == '_' || character == '.':
				if index == 0 {
					return fmt.Errorf("the integration name %q must start with a letter or digit", bounded_(name))
				}
			default:
				return fmt.Errorf(
					"the integration name %q may contain only letters, digits, and the separators - _ .", bounded_(name))
			}
		}
	}
	return nil
}

// supportsIntegrationRemoval reports whether a provider's own interface can
// remove one named integration. It is a statement about vendor interfaces, not
// about any integration: Goalrail still knows none of them by name.
func supportsIntegrationRemoval(reviewer ambient.Scaffold) bool {
	return reviewer == ambient.ScaffoldCodex
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
