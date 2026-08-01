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

	report, err := invoke(ctx, input.Reviewer, input.RepositoryRoot, input.BaseRef, instructions)
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

// invoke runs one reviewer through its vendor's own non-interactive interface.
//
// Minimum documented flags on purpose. Neither CLI can be pinned by Goalrail,
// so every flag is a surface that can drift under it; a vendor's refusal is
// surfaced to the caller unchanged rather than translated, because a wrapper
// that smooths over a changed interface converts a loud break into a quiet one.
func invoke(ctx context.Context, reviewer ambient.Scaffold, repositoryRoot, baseRef string, instructions []byte) (string, error) {
	var command *exec.Cmd
	switch reviewer {
	case ambient.ScaffoldCodex:
		// `-` reads the review instructions from stdin, and --base makes the
		// reviewed range the branch's own changes.
		command = exec.CommandContext(ctx, "codex", "review", "--base", baseRef, "-")
		command.Stdin = bytes.NewReader(instructions)
	case ambient.ScaffoldClaudeCode:
		// The general agent gets the same instructions plus the range to read,
		// and only the tools a reviewer needs. It must never edit: a reviewer
		// that can change the work is no longer reviewing it.
		prompt := string(instructions) + "\n\nReview the changes this branch introduces against `" + baseRef + "`.\n" +
			"Use git to read the diff. Do not modify anything.\n"
		command = exec.CommandContext(ctx, "claude", "-p",
			"--output-format", "text",
			"--allowed-tools", "Bash(git *)", "Read", "Grep", "Glob")
		command.Stdin = strings.NewReader(prompt)
	default:
		return "", fmt.Errorf("no review invocation is defined for %q", reviewer)
	}

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
