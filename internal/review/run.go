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

// DefaultDeadline bounds one review.
//
// A bound is required and cannot be per-vendor: `claude` carries no turn limit
// in the version measured here, whatever its documentation elsewhere suggests,
// so a ceiling built out of vendor flags would be a ceiling only where the flag
// happens to exist. A deadline holds for every reviewer and cannot drift with
// an interface.
const DefaultDeadline = 20 * time.Minute

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

	// BaseRef is what the branch is reviewed against.
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

	baseCommit, err := Resolve(input.RepositoryRoot, input.BaseRef)
	if err != nil {
		return Result{}, fmt.Errorf("resolve base ref %q: %w", input.BaseRef, err)
	}
	headCommit, err := Resolve(input.RepositoryRoot, "HEAD")
	if err != nil {
		return Result{}, err
	}
	if baseCommit == headCommit {
		return Result{}, fmt.Errorf("the branch has no changes against %s, so there is nothing to review", input.BaseRef)
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

	instructions, materialized, err := EnsureInstructions(input.RepositoryRoot)
	if err != nil {
		return Result{}, err
	}

	// The gate runs before anything is spawned, because its whole purpose is to
	// stop the spend rather than to report it afterwards.
	if strings.TrimSpace(input.Gate) != "" {
		if gateErr := runGate(ctx, input.RepositoryRoot, input.Gate); gateErr != nil {
			return Result{InstructionsMaterialized: materialized}, gateErr
		}
	}

	effort := strings.TrimSpace(input.Effort)
	if effort == "" {
		effort = DefaultEffort
	}
	deadline := input.Deadline
	if deadline <= 0 {
		deadline = DefaultDeadline
	}
	bounded, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()
	started := now()
	rangeSpec := reviewedBase + "..." + headCommit
	report, err := invoke(bounded, input.Selection.Reviewer, input.RepositoryRoot, rangeSpec, effort, instructions)
	if err != nil {
		// No receipt: one describing a review that did not happen is worse than
		// none, because it reads as done.
		return Result{InstructionsMaterialized: materialized}, err
	}

	mode, modeReason := input.Selection.Mode, input.Selection.Reason
	var refutation, refuter string
	if input.Refute {
		// The refutation is a second paid run: the gate's answer may have
		// changed while the first one spent, so it is asked again.
		if strings.TrimSpace(input.Gate) != "" {
			if gateErr := runGate(ctx, input.RepositoryRoot, input.Gate); gateErr != nil {
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
		refutePrompt := string(instructions) +
			"\n\nYou are refuting a review, not extending it. Below is a reviewer's report on this change.\n" +
			"Try to REFUTE each finding against the actual code; do not add new findings.\n" +
			"For each: state REFUTED or STANDS with the concrete evidence.\n\n--- REPORT UNDER CHALLENGE ---\n" + report
		refutation, refuteErr = invoke(bounded, refuterSelection.Reviewer, input.RepositoryRoot, rangeSpec, effort, []byte(refutePrompt))
		if refuteErr != nil {
			return Result{InstructionsMaterialized: materialized}, refuteErr
		}
		refuter = string(refuterSelection.Reviewer)
		mode = "refute"
		modeReason = input.Selection.Reason + "; refuted by " + refuter + " in a fresh session"
	}

	diffDigest, err := DiffDigest(input.RepositoryRoot, baseCommit, headCommit)
	if err != nil {
		return Result{InstructionsMaterialized: materialized}, err
	}
	reviewedDigest, err := DiffDigest(input.RepositoryRoot, reviewedBase, headCommit)
	if err != nil {
		return Result{InstructionsMaterialized: materialized}, err
	}

	receipt := Receipt{
		Schema:             ReceiptSchema,
		Repository:         input.RepositoryRoot,
		Branch:             branch,
		BaseRef:            input.BaseRef,
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
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = err.Error()
		}
		return fmt.Errorf("%w (%s): %s", ErrGateRefused, gate, detail)
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
func reviewCommand(reviewer ambient.Scaffold, baseRef, effort string, instructions []byte) (name string, arguments []string, stdin string, err error) {
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
		return "codex", []string{
			"-c", "model_reasoning_effort=" + effort,
			"review", "-",
		}, prompt(baseRef, instructions), nil
	case ambient.ScaffoldClaudeCode:
		// Only the tools a reviewer needs, and never an editing one: a reviewer
		// that can change the work is no longer reviewing it.
		//
		// The git allowance is per subcommand rather than `git *`. That wildcard
		// admitted `git reset`, `git checkout` and `git clean`, and `git -c
		// alias.x='!sh -c ...'` would have run an arbitrary shell through it —
		// a read-only contract that the permission it granted did not keep.
		return "claude", []string{
			"-p", "--output-format", "text",
			"--effort", effort,
			"--allowed-tools",
			"Bash(git diff:*)", "Bash(git log:*)", "Bash(git show:*)",
			"Bash(git status:*)", "Bash(git rev-parse:*)", "Bash(git ls-files:*)",
			"Read", "Grep", "Glob",
			"--disallowed-tools", "Edit", "Write", "NotebookEdit",
		}, prompt(baseRef, instructions), nil
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
func invoke(ctx context.Context, reviewer ambient.Scaffold, repositoryRoot, baseRef, effort string, instructions []byte) (string, error) {
	name, arguments, stdin, err := reviewCommand(reviewer, baseRef, effort, instructions)
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
	stdout := &boundedBuffer{limit: receiptBound}
	stderr := &boundedBuffer{limit: 1 << 20}
	command.Stdout = stdout
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return "", fmt.Errorf("the %s reviewer did not finish within the review deadline", reviewer)
		}
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = strings.TrimSpace(stdout.String())
		}
		if detail == "" {
			detail = err.Error()
		}
		return "", fmt.Errorf("the %s reviewer did not complete: %s", reviewer, detail)
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
