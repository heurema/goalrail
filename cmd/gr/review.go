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

// reviewModelEnvironment names the reviewer's model.
//
// Stated rather than inherited: a review invoked from a session would otherwise
// run on whatever model that session picked for authoring.
const reviewModelEnvironment = "GOALRAIL_REVIEW_MODEL"

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
	base := set.String("base", "", "ref the branch is reviewed against; omit to use one unambiguous local remote default")
	full := set.Bool("full", false, "review the whole branch even where a previous receipt would make the round incremental")
	refute := set.Bool("refute", false, "run a second fresh session that tries to refute the findings; use when findings exist and you are about to act on them")
	author := set.String("author", "", "authoring agent (codex or claude-code); omit to detect from the environment")
	gate := set.String("gate", "", "command run before the review; a non-zero exit refuses it (default $"+reviewGateEnvironment+")")
	model := set.String("model", "", "reviewer model; passed to whichever provider reviews (default: opus for claude-code, the vendor's own for codex, or $"+reviewModelEnvironment+")")
	effort := set.String("effort", "", "reviewer reasoning effort (default "+review.DefaultEffort+", "+review.FullEffort+" with --full, or $"+reviewEffortEnvironment+")")
	deadline := set.Duration("deadline", 0, "bound on the whole review (default "+review.DefaultDeadline.String()+", "+review.FullDeadline.String()+" with --full)")
	progressBound := set.Duration("progress-bound", 0, "stop a reviewer that produces nothing for this long (default "+review.DefaultProgressBound.String()+"); a stalled reviewer is not a completed review")
	var without integrationNames
	set.Var(&without, "without", "remove one named provider integration from the reviewer's session; repeatable, off by default")
	if err := set.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if set.NArg() != 0 {
		return fmt.Errorf("review accepts no positional arguments")
	}

	// An explicit non-positive bound is a malformed value, not a request for the
	// default: silently starting a 45-minute review on `--deadline=0s` is the
	// opposite of what the caller asked for.
	if isSet(set, "deadline") && *deadline <= 0 {
		return fmt.Errorf("--deadline must be positive; %s asks for no time at all", *deadline)
	}
	// Same rule, same reason: an explicit non-positive bound is a malformed
	// value rather than a request for the default.
	if isSet(set, "progress-bound") && *progressBound <= 0 {
		return fmt.Errorf("--progress-bound must be positive; %s asks for no patience at all", *progressBound)
	}
	if isSet(set, "base") && strings.TrimSpace(*base) == "" {
		return fmt.Errorf("--base must name a non-empty Git ref")
	}

	root, err := filepath.Abs(*repository)
	if err != nil {
		return err
	}
	if err := refuseWhereThereIsNoWorkTree(root); err != nil {
		return err
	}
	// A subdirectory invocation must not mint a second identity: instructions
	// and receipts belong to the worktree root git itself names.
	if toplevel, topErr := review.WorktreeRoot(root); topErr == nil && toplevel != "" {
		root = toplevel
	}

	// Detection improves the choice; its failure is recorded, never fatal. The
	// fresh session provides the independence either way.
	authoring := review.AuthorUnknown
	if resolved, resolveErr := resolveAuthor(*author); resolveErr == nil {
		authoring = resolved
	} else if strings.TrimSpace(*author) != "" {
		// An explicit override that does not parse is a typo, not an unknown.
		return resolveErr
	}

	// Every supported provider is a candidate, and being runnable is what makes
	// one a reviewer. A configuration directory is evidence that a scaffold was
	// used here, not that its command exists now.
	selection, err := review.SelectReviewer(authoring, ambient.SupportedScaffolds(), nil)
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

	chosenModel := *model
	if strings.TrimSpace(chosenModel) == "" {
		chosenModel = os.Getenv(reviewModelEnvironment)
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
		Model:          chosenModel,
		Deadline:       *deadline,
		ProgressBound:  *progressBound,

		// Which integration is hostile is the caller's knowledge of their own
		// machine; Goalrail passes the name through to the provider's own
		// removal syntax and knows none of them.
		WithoutIntegrations: without,
		Author:              authoring,
		Selection:           selection,
		Full:                *full,
		Refute:              *refute,
		Gate:                configured,
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
		Mode:                     result.Receipt.Mode,
		ModeReason:               result.Receipt.ModeReason,
		ReviewedBase:             result.Receipt.ReviewedBase,
		DurationSeconds:          result.Receipt.DurationSeconds,
		Effort:                   result.Receipt.Effort,
		Model:                    result.Receipt.Model,
		UnchangedRounds:          result.Receipt.UnchangedRounds,
		Refutation:               result.Receipt.Refutation,
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
		Notes: append([]string{
			"the report is the reviewer's own text; Goalrail does not read, score, or act on it",
			"this is one review, not a loop: re-run it after fixes, under a ceiling you declared",
		}, stalemateNote(result.Receipt.UnchangedRounds)...),
	})
}

// reviewReport is what one review prints. The report travels with it because
// the caller is the thing that has to read it, and making them open a file to
// find out what the review said would be a step for nothing.
type reviewReport struct {
	Repository               string   `json:"repository"`
	Branch                   string   `json:"branch"`
	Author                   string   `json:"author"`
	Mode                     string   `json:"mode"`
	ModeReason               string   `json:"mode_reason"`
	ReviewedBase             string   `json:"reviewed_base"`
	DurationSeconds          int64    `json:"duration_seconds"`
	Effort                   string   `json:"effort"`
	Model                    string   `json:"model,omitempty"`
	UnchangedRounds          int      `json:"unchanged_rounds"`
	Refutation               string   `json:"refutation,omitempty"`
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
	// Receipts are keyed by the worktree path Git itself reports. filepath.Abs
	// can retain a symlinked spelling (notably /var versus /private/var on
	// macOS), which would make doctor look under a different key and report an
	// existing review as absent.
	if toplevel, err := review.WorktreeRoot(repositoryRoot); err == nil && toplevel != "" {
		repositoryRoot = toplevel
	}
	branch, err := review.CurrentBranch(repositoryRoot)
	if err != nil || branch == "" {
		return harness.ReviewState{}
	}
	state, receipt, err := review.Status(stateRoot, repositoryRoot, branch)
	if err != nil {
		// Precisely the state where stored evidence is unusable — a truncated,
		// malformed or oversized receipt, or a base commit that has been pruned.
		// Reporting nothing here would present "no review recorded" for a branch
		// that has one and cannot read it.
		return harness.ReviewState{
			Branch: branch,
			State:  "unreadable",
			Note: fmt.Sprintf("review: recorded for %s but unreadable (%v); `gr review` records a fresh one",
				branch, err),
		}
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

// stalemateNote states the measurement without acting on it. A loop that keeps
// spending while this climbs is not converging: the previous round's findings
// were not acted on, and the next round reviews the same bytes again.
func stalemateNote(rounds int) []string {
	if rounds == 0 {
		return nil
	}
	return []string{fmt.Sprintf(
		"stalemate: %d consecutive round(s) reviewed the same head with a clean tree — nothing was acted on between them; stop and report the open findings",
		rounds)}
}

// isSet reports whether a flag was named on the command line, which is how an
// explicit zero is told apart from an omitted one.
func isSet(set *flag.FlagSet, name string) bool {
	found := false
	set.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

// integrationNames collects a repeatable --without. A repeatable flag rather
// than one comma-separated value: a separator would have to be excluded from
// every name, and a name is the caller's, not this command's, to constrain
// beyond what the invocation requires.
type integrationNames []string

func (names *integrationNames) String() string { return strings.Join(*names, ",") }

func (names *integrationNames) Set(value string) error {
	*names = append(*names, value)
	return nil
}
