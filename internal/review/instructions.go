package review

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/heurema/goalrail/internal/ambient"
	"github.com/heurema/goalrail/internal/boundedio"
)

// InstructionsPath is the repository file the reviewer's instructions are read
// from.
//
// Committed on purpose, and deliberately not under `.goalrail/`: Goalrail's own
// ignore rule keeps that directory out of version control, and instructions
// that cannot be committed cannot be shared or reviewed. What a reviewer is
// told to look for is the accumulated record of what has previously been
// missed, so it belongs in the same pull requests as the code it protects.
const InstructionsPath = ".goalrail-review.md"

// instructionsBound is generous for prose and small enough that a planted
// enormous file cannot be read into memory unbounded.
const instructionsBound = 256 * 1024

// defaultInstructions is materialized once, into a repository that has none.
//
// It carries what the findings ratchet has already promoted rather than generic
// advice: the two classes that reached the promotion threshold on the first day
// of counting were a diff contradicting the repository's own AGENTS.md, and
// claims of absence shipped without the search that would falsify them.
const defaultInstructions = `# Review instructions

You are reviewing a change you did not write, in a repository you must read
before judging. Report only defects you can point at.

## Read first

- The repository's own ` + "`AGENTS.md`" + ` (and any subtree ` + "`AGENTS.md`" + `). A change may
  satisfy every test and still contradict the contract that governs the
  repository. Check the diff against it explicitly.
- The files the diff touches, and their callers.

## Look for

- **Claims without checks.** Any assertion of absence, count, or consistency —
  "there is no X", "nothing does Y", "N of these exist" — that ships without the
  command, test, or search that verifies it. Absence claims are the expensive
  kind: a reader acts on them by building the thing.
- **Unenumerated input space.** For anything that reads a file, a format, or an
  environment: what happens when it is absent, empty, written in another legal
  syntax, or edited by the user? A reader that guesses confidently is worse than
  one that refuses.
- **Reused code applied to a new purpose.** A helper that was correct for its
  original caller may be wrong here. Say what breaks, concretely.
- **Boundaries asserted in one artifact but absent from the one above it.**
- **Correctness, then everything else.** A defect that changes behaviour ranks
  above a defect of taste. Do not report style.

## Report

For each finding: the file and line, what is wrong, and the concrete input or
state that makes it wrong. No summaries of what the change does — the author
knows.

End your report with exactly one of these lines, alone on the last line:

    REVIEW-VERDICT: material-findings
    REVIEW-VERDICT: nothing-material

That line is your statement, not a verdict anyone else computes.

## Refute round

When this file is handed to you together with a prior report, you are refuting
that review, not extending it. Try to REFUTE each finding against the actual
code; do not add new findings. For each: state REFUTED or STANDS with the
concrete evidence.
`

// EnsureInstructions returns the reviewer instructions for one repository,
// materializing the default when the repository has none.
//
// Materialization never overwrites: an existing file is the user's, including
// one they emptied on purpose. The write goes through the same containment
// every repository-scope write does, because a checkout can ship a symlink at
// this path and redirect it into the user's own files.
func EnsureInstructions(repositoryRoot string) (contents []byte, materialized bool, err error) {
	absolute := filepath.Join(repositoryRoot, InstructionsPath)
	if containErr := ambient.EnsureWriteWithinRepository(repositoryRoot, absolute); containErr != nil {
		return nil, false, containErr
	}

	contents, readErr := boundedio.ReadRegularFile(absolute, "review instructions", instructionsBound)
	if readErr == nil {
		return contents, false, nil
	}
	if !errors.Is(readErr, fs.ErrNotExist) {
		return nil, false, readErr
	}

	if writeErr := os.WriteFile(absolute, []byte(defaultInstructions), 0o644); writeErr != nil {
		return nil, false, fmt.Errorf("write %s: %w", InstructionsPath, writeErr)
	}
	// Instructions the repository ignores never reach collaborators, and the
	// ratchet's promotions would stay local while looking shared.
	if _, err := git(repositoryRoot, "check-ignore", "-q", InstructionsPath); err == nil {
		return nil, true, fmt.Errorf("%s is ignored by this repository's rules; review instructions must be committable — adjust the ignore rules", InstructionsPath)
	}
	return []byte(defaultInstructions), true, nil
}
