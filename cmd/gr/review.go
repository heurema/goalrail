package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/heurema/goalrail/internal/ambient"
	"github.com/heurema/goalrail/internal/harness"
	"github.com/heurema/goalrail/internal/localrun"
	"github.com/heurema/goalrail/internal/review"
)

// reviewGateEnvironment names the command run before any reviewer is invoked.
//
// A command in the user's configuration, never a path or a budget service
// compiled into the binary: whose budget it is and how it is measured belongs to
// one machine, not to everyone who downloads a release.
const reviewGateEnvironment = "GOALRAIL_REVIEW_GATE"

// reviewEffortEnvironment names the reviewer's reasoning effort.
//
// Stated rather than inherited: a review runs again on every round of a loop,
// and the machine's interactive configuration was tuned for a session where one
// answer is waited for once.
const reviewEffortEnvironment = "GOALRAIL_REVIEW_EFFORT"

// defaultBaseRef is what a branch is reviewed against when the caller says
// nothing.
const defaultBaseRef = "main"

func runReview(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	set := flag.NewFlagSet("review", flag.ContinueOnError)
	set.SetOutput(stderr)
	set.Usage = func() {
		fmt.Fprint(stderr, "usage: gr review [flags]\n\n"+
			"Reviews this branch with an agent that did not write it, and records what was reviewed.\n"+
			"Runs one review. The loop that re-runs it after fixes belongs to whoever calls this.\n\n")
		set.PrintDefaults()
	}
	repository := set.String("repo", ".", "repository whose branch to review")
	stateDirectory := set.String("state-dir", "", "Goalrail local state directory")
	base := set.String("base", defaultBaseRef, "ref the branch is reviewed against")
	author := set.String("author", "", "authoring agent (codex or claude-code); omit to detect from the environment")
	gate := set.String("gate", "", "command run before the review; a non-zero exit refuses it (default $"+reviewGateEnvironment+")")
	effort := set.String("effort", "", "reviewer reasoning effort (default "+review.DefaultEffort+", or $"+reviewEffortEnvironment+")")
	if err := set.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if set.NArg() != 0 {
		return fmt.Errorf("review accepts no positional arguments")
	}

	root, err := filepath.Abs(*repository)
	if err != nil {
		return err
	}
	if err := refuseWhereThereIsNoWorkTree(root); err != nil {
		return err
	}

	// Detection first, because getting it wrong hands the work back to its own
	// author and reports success while doing it.
	authoring, err := resolveAuthor(*author)
	if err != nil {
		return err
	}

	home, homeErr := os.UserHomeDir()
	if homeErr != nil {
		return fmt.Errorf("resolve home directory: %w", homeErr)
	}
	reviewer, err := review.SelectReviewer(authoring, ambient.DetectScaffolds(home))
	if err != nil {
		return err
	}

	stateRoot, err := localrun.ResolveStateRoot(*stateDirectory)
	if err != nil {
		return err
	}

	configured := *gate
	if strings.TrimSpace(configured) == "" {
		configured = os.Getenv(reviewGateEnvironment)
	}

	chosenEffort := *effort
	if strings.TrimSpace(chosenEffort) == "" {
		chosenEffort = os.Getenv(reviewEffortEnvironment)
	}

	result, err := review.Run(ctx, review.Input{
		RepositoryRoot: root,
		StateRoot:      stateRoot,
		BaseRef:        *base,
		Effort:         chosenEffort,
		Author:         authoring,
		Reviewer:       reviewer,
		Gate:           configured,
	})
	if err != nil {
		// The instructions file may have been materialized before the refusal.
		// Saying so keeps an unexpected untracked file from looking like damage.
		if result.InstructionsMaterialized {
			fmt.Fprintf(stderr, "note: %s was created with the default review instructions\n", review.InstructionsPath)
		}
		return err
	}

	return writeJSON(stdout, reviewReport{
		Repository:               result.Receipt.Repository,
		Branch:                   result.Receipt.Branch,
		Author:                   result.Receipt.Author,
		Reviewer:                 result.Receipt.Reviewer,
		BaseRef:                  result.Receipt.BaseRef,
		BaseCommit:               result.Receipt.BaseCommit,
		HeadCommit:               result.Receipt.HeadCommit,
		DiffSHA256:               result.Receipt.DiffSHA256,
		ReportSHA256:             result.Receipt.ReportSHA256,
		ReviewedAt:               result.Receipt.ReviewedAt,
		Receipt:                  result.ReceiptPath,
		InstructionsMaterialized: result.InstructionsMaterialized,
		Report:                   result.Receipt.Report,
		Notes: []string{
			"the report is the reviewer's own text; Goalrail does not read, score, or act on it",
			"this is one review, not a loop: re-run it after fixes, under a ceiling you declared",
		},
	})
}

// reviewReport is what one review prints. The report travels with it because
// the caller is the thing that has to read it, and making them open a file to
// find out what the review said would be a step for nothing.
type reviewReport struct {
	Repository               string   `json:"repository"`
	Branch                   string   `json:"branch"`
	Author                   string   `json:"author"`
	Reviewer                 string   `json:"reviewer"`
	BaseRef                  string   `json:"base_ref"`
	BaseCommit               string   `json:"base_commit"`
	HeadCommit               string   `json:"head_commit"`
	DiffSHA256               string   `json:"diff_sha256"`
	ReportSHA256             string   `json:"report_sha256"`
	ReviewedAt               string   `json:"reviewed_at"`
	Receipt                  string   `json:"receipt"`
	InstructionsMaterialized bool     `json:"instructions_materialized"`
	Report                   string   `json:"report"`
	Notes                    []string `json:"notes,omitempty"`
}

// resolveAuthor honours an explicit statement and otherwise reads the
// environment. The override always wins: detection describes the ordinary case,
// and the odd one has to remain expressible.
func resolveAuthor(explicit string) (ambient.Scaffold, error) {
	if strings.TrimSpace(explicit) != "" {
		return parseScaffold(explicit)
	}
	return review.DetectAuthor(os.LookupEnv)
}

// reviewStateFor is what the diagnosis reports about this branch.
//
// Every outcome is a condition rather than a fault, including the ones that
// could not be computed: a repository that is not a checkout, or a detached
// head, has no review state, and reporting that as a problem would make an
// ordinary situation look broken.
func reviewStateFor(repositoryRoot, stateRoot string) harness.ReviewState {
	branch, err := review.CurrentBranch(repositoryRoot)
	if err != nil || branch == "" {
		return harness.ReviewState{}
	}
	state, receipt, err := review.Status(stateRoot, repositoryRoot, branch)
	if err != nil {
		return harness.ReviewState{}
	}
	reported := harness.ReviewState{State: string(state), Branch: branch}
	switch state {
	case review.StateAbsent:
		reported.Note = fmt.Sprintf("review: none for %s (optional; `gr review` runs one)", branch)
	case review.StateCurrent:
		reported.Reviewer, reported.ReviewedAt = receipt.Reviewer, receipt.ReviewedAt
		reported.Note = fmt.Sprintf("review: current for %s (%s, %s)", branch, receipt.Reviewer, receipt.ReviewedAt)
	case review.StateStale:
		reported.Reviewer, reported.ReviewedAt = receipt.Reviewer, receipt.ReviewedAt
		reported.Note = fmt.Sprintf("review: stale for %s (%s reviewed %s; the branch has changed since)",
			branch, receipt.Reviewer, receipt.ReviewedAt)
	}
	return reported
}
