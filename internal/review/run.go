package review

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/heurema/goalrail/internal/ambient"
)

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

	// Reviewer is the provider performing the review.
	Reviewer ambient.Scaffold

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
	Receipt                 Receipt
	ReceiptPath             string
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
	report, err := invoke(ctx, input.Reviewer, input.RepositoryRoot, input.BaseRef, effort, instructions)
	if err != nil {
		// No receipt: one describing a review that did not happen is worse than
		// none, because it reads as done.
		return Result{InstructionsMaterialized: materialized}, err
	}

	diffDigest, err := DiffDigest(input.RepositoryRoot, baseCommit, headCommit)
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
		DiffSHA256:         diffDigest,
		Reviewer:           string(input.Reviewer),
		Author:             string(input.Author),
		ReviewedAt:         now().UTC().Format(time.RFC3339),
		Report:             report,
		ReportSHA256:       digest([]byte(report)),
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
func prompt(baseRef string, instructions []byte) string {
	return string(instructions) +
		"\n\nReview the changes this branch introduces against `" + baseRef + "`" +
		" — the three-dot range `" + baseRef + "...HEAD`, not the working tree.\n" +
		"Use git to read the diff. Do not modify anything.\n"
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
		_ = effort
		return "claude", []string{
			"-p", "--output-format", "text",
			"--allowed-tools", "Bash(git *)", "Read", "Grep", "Glob",
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
	// The reviewer must not inherit the author's session identity. A spawned
	// agent that reads the parent's markers believes it is a continuation of the
	// session that wrote the code, which is exactly the shared context this
	// review exists to avoid.
	command.Env = strippedEnvironment(os.Environ(), AuthorMarkers())

	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
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
