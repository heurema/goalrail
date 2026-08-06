package review

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/heurema/goalrail/internal/boundedio"
)

// ReceiptSchema is the schema identifier a stored receipt carries.
const ReceiptSchema = "goalrail.review-receipt.v1"

// receiptBound caps a stored receipt. A review report is prose; anything past
// this is not a report.
const receiptBound = 8 * 1024 * 1024

// Receipt is what one completed review leaves behind.
//
// Every field is measured. None is derived by reading the report: the report is
// stored as the bytes the reviewer emitted, and deciding which findings are
// real belongs to whoever reads them. A tool that parsed review prose to
// populate a "severity" or a "count" would be committing the defect class this
// whole feature exists to catch.
type Receipt struct {
	Schema string `json:"schema"`

	Repository string `json:"repository"`
	Branch     string `json:"branch"`

	// BaseRef is the selected human-readable ref: either the caller's explicit
	// value or the one unambiguous local remote default. BaseCommit is what that
	// resolved to at review time. A branch name moves, a commit does not, and the
	// digest below is only reproducible from the commit.
	BaseRef    string `json:"base_ref"`
	BaseCommit string `json:"base_commit"`
	HeadCommit string `json:"head_commit"`

	// DiffSHA256 digests the canonical diff of the reviewed range. It is what
	// makes "this review still describes this branch" a computation rather than
	// a belief.
	DiffSHA256 string `json:"diff_sha256"`

	// ReviewedBase..HeadCommit is the range this round actually covered. On an
	// incremental round it starts at the previous receipt's head, and the chain
	// of receipts is what proves cumulative coverage; DiffSHA256 above always
	// digests the full branch so staleness stays one comparison.
	ReviewedBase string `json:"reviewed_base"`

	// ReviewedDiffSHA256 digests the range this round actually covered, so the
	// receipt proves the exact bytes reviewed even when the round was
	// incremental; DiffSHA256 stays the full-branch digest staleness compares.
	ReviewedDiffSHA256 string `json:"reviewed_diff_sha256"`

	// Mode is fresh, cross, or refute; ModeReason states why in one sentence,
	// because a cross that quietly became fresh is an unprovable claim.
	Mode       string `json:"mode"`
	ModeReason string `json:"mode_reason"`

	Reviewer   string `json:"reviewer"`
	Author     string `json:"author"`
	ReviewedAt string `json:"reviewed_at"`

	// UnchangedRounds counts consecutive rounds that reviewed the same head with
	// a clean working tree — the author acted on nothing between them. It is a
	// measurement, not a reading of any report: a loop that keeps spending while
	// this climbs is not converging, and the caller's policy decides where to
	// stop. Zero means work moved since the previous round.
	UnchangedRounds int `json:"unchanged_rounds"`

	// TreeCleanAtReview records whether the author's tree was clean when this
	// round ran. Without it the count cannot tell two rounds over identical
	// state from a round that followed discarded work, and a stop signal that
	// cannot be audited is one nobody should stop on.
	//
	// A pointer because absent and false are different facts. A receipt written
	// before this field existed knows nothing about that round's tree, and a
	// failed `git status` measured nothing — collapsing either into `false`
	// would record an unknown as a measurement, which is the one thing a
	// receipt must never do. Nil never counts toward a stalemate.
	TreeCleanAtReview *bool `json:"tree_clean_at_review,omitempty"`

	// Effort and Model are the execution parameters the paid invocation actually
	// ran with. Where Goalrail pinned no model, Model records that the provider
	// used its own configuration rather than being left empty: an empty field
	// reads as "no model", and the provider certainly used one. What that
	// configuration named is outside what this receipt can prove, and saying so
	// is better than implying completeness. Without them two receipts for the same provider and range cannot
	// establish which settings produced them, and the receipt's whole contract is
	// to prove how the review ran. Model is empty where the provider was left on
	// its own configuration; RefuterModel likewise, and it can differ because a
	// model name belongs to one provider.
	Effort       string `json:"effort"`
	Model        string `json:"model,omitempty"`
	RefuterModel string `json:"refuter_model,omitempty"`

	// WithoutIntegrations records what the reviewer was run without. A review
	// with an integration removed is a different review from one with it, and a
	// consumer that cannot tell them apart cannot judge either.
	WithoutIntegrations []string `json:"without_integrations,omitempty"`

	// DurationSeconds is measured wall time of the reviewer invocation(s), the
	// number effort and deadline defaults get tuned against.
	DurationSeconds int64 `json:"duration_seconds"`

	// Report is verbatim, and ReportSHA256 digests exactly those bytes. On a
	// refute round Refutation carries the second report the same way; which
	// findings survived is the reader's judgement, never a field.
	Report           string `json:"report"`
	ReportSHA256     string `json:"report_sha256"`
	Refutation       string `json:"refutation,omitempty"`
	RefutationSHA256 string `json:"refutation_sha256,omitempty"`
	Refuter          string `json:"refuter,omitempty"`

	// InstructionsSHA256 records which instructions produced this review, so a
	// receipt from before a ratchet promotion is distinguishable from one after.
	InstructionsSHA256 string `json:"instructions_sha256"`
}

func digest(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}

// git runs one git command in a repository and returns its trimmed output.
//
// Repository selectors are stripped from the child environment for the same
// reason every other git call in this project does it: an exported GIT_DIR or
// GIT_WORK_TREE would silently redirect the command into a different
// repository, and a digest taken there would describe someone else's changes.
func git(repositoryRoot string, arguments ...string) (string, error) {
	return gitContext(context.Background(), repositoryRoot, arguments...)
}

func gitContext(ctx context.Context, repositoryRoot string, arguments ...string) (string, error) {
	command := exec.CommandContext(ctx, "git", append([]string{"-C", repositoryRoot}, arguments...)...)
	// Repository selectors are stripped, and the reader's global and system
	// configuration is taken out of the answer. The repository's own .git/config
	// survives that, so the diff-affecting keys are additionally overridden on
	// the command line — including diff.orderFile, whose "empty" form is
	// /dev/null, a readable empty file, because an empty *value* is a path git
	// fails to open.
	command.Env = append(
		strippedEnvironment(os.Environ(), []string{"GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE", "GIT_OBJECT_DIRECTORY"}),
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_SYSTEM=/dev/null",
	)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		if contextErr := ctx.Err(); contextErr != nil {
			return "", fmt.Errorf("git %s: %w", strings.Join(arguments, " "), contextErr)
		}
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(arguments, " "), detail)
	}
	return strings.TrimRight(stdout.String(), "\n"), nil
}

// strippedEnvironment returns an environment with the named variables removed.
func strippedEnvironment(environment []string, remove []string) []string {
	drop := make(map[string]struct{}, len(remove))
	for _, name := range remove {
		drop[name] = struct{}{}
	}
	kept := make([]string, 0, len(environment))
	for _, entry := range environment {
		name, _, found := strings.Cut(entry, "=")
		if !found {
			kept = append(kept, entry)
			continue
		}
		if _, unwanted := drop[name]; unwanted {
			continue
		}
		kept = append(kept, entry)
	}
	return kept
}

// canonicalDiffArguments renders the reviewed range identically on any machine.
//
// The three-dot range is the change the branch introduces since it diverged,
// not the difference between two tips — a base that moved ahead would otherwise
// make unrelated work appear in the digest. External diff drivers, colour and
// textconv are all refused because each of them makes the same range render
// differently depending on the reader's configuration, which would turn
// staleness into a property of whose machine asked.
func canonicalDiffArguments(baseCommit, headCommit string) []string {
	return []string{
		// Attributes come from the digested head itself, not the checkout's
		// moving HEAD: recomputing a historical range must not change with what
		// is checked out today. Known residual: .git/info/attributes still
		// outranks every configurable source; a clone that edits it diverges,
		// and there is no git switch to exclude it — documented rather than
		// papered over.
		// Flags alone do not make a diff canonical: the repository's own
		// configuration still decides context lines, the algorithm, prefixes and
		// ordering. Left to it, changing `diff.context` after a review makes the
		// same commits digest differently and reports an untouched branch as
		// stale — the staleness signal would then describe the reader's
		// configuration rather than the branch.
		"-c", "diff.context=3",
		"-c", "diff.algorithm=myers",
		"-c", "diff.noprefix=false",
		"-c", "diff.mnemonicPrefix=false",
		"-c", "diff.srcPrefix=a/",
		"-c", "diff.dstPrefix=b/",
		"-c", "diff.relative=false",
		"-c", "diff.indentHeuristic=true",
		"-c", "diff.orderFile=/dev/null",
		// Working-tree attributes steer a commit-range diff too: an uncommitted
		// .gitattributes "binary" rule flips a textual patch to "Binary files
		// differ" and a stable branch reads as stale. Both attribute sources are
		// pointed at empty files for the digest.
		"-c", "core.attributesFile=/dev/null",
		"-c", "attr.tree=" + headCommit,
		"-c", "core.abbrev=40",
		"-c", "diff.ignoreSubmodules=none",
		"diff", "--no-ext-diff", "--no-color", "--no-textconv",
		"--full-index", "--no-renames", "--ignore-submodules=none",
		baseCommit + "..." + headCommit,
	}
}

// renderCanonicalDiff returns the exact bytes that define a reviewed range.
// Callers that both transport and digest a diff must keep this single rendering:
// rendering twice would make the receipt a claim about bytes that need not be
// the bytes a provider actually received.
func renderCanonicalDiff(ctx context.Context, repositoryRoot, baseCommit, headCommit string) ([]byte, error) {
	rendered, err := gitContext(ctx, repositoryRoot, canonicalDiffArguments(baseCommit, headCommit)...)
	if err != nil {
		return nil, err
	}
	return []byte(rendered), nil
}

// DiffDigest measures the reviewed range.
func DiffDigest(repositoryRoot, baseCommit, headCommit string) (string, error) {
	rendered, err := renderCanonicalDiff(context.Background(), repositoryRoot, baseCommit, headCommit)
	if err != nil {
		return "", err
	}
	return digest(rendered), nil
}

// Resolve reports the commit a ref names.
func Resolve(repositoryRoot, ref string) (string, error) {
	return git(repositoryRoot, "rev-parse", "--verify", ref+"^{commit}")
}

// CurrentBranch reports the checked-out branch, or an empty string on a
// detached head. A detached head has no branch a receipt could be filed under,
// and saying so is better than filing one under a name that means nothing.
func CurrentBranch(repositoryRoot string) (string, error) {
	name, err := git(repositoryRoot, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	if name == "HEAD" {
		return "", nil
	}
	return name, nil
}

// receiptPath locates one branch's receipt inside the state root.
//
// Keyed by repository path and branch together, because the same branch name in
// two clones is two different pieces of work. The receipt lives in per-clone
// state and never in the repository: it is evidence about one clone's branch at
// one moment, and a receipt someone else received by pulling would assert a
// review of a diff they may not have.
func branchKey(repositoryRoot, branch string) string {
	return digest([]byte(repositoryRoot + "\x00" + branch))
}

// receiptPath locates one round's receipt: one file per reviewed head, so a
// later round never erases the evidence of an earlier one. The chain is what
// lets an incremental receipt prove the rounds before it happened.
func receiptPath(stateRoot, repositoryRoot, branch, headCommit, reportDigest, settings string) string {
	suffix := headCommit
	if settings != "" {
		// Two rounds at one head with identical report bytes but different
		// settings are different rounds; keying without the settings let the
		// second erase the first, which is the very distinguishability the
		// recorded parameters were added for.
		suffix += "-" + digest([]byte(settings))[:8]
	}
	if len(reportDigest) >= 12 {
		// Rounds at the same head are distinct rounds; keying by head alone let
		// a repeated round erase the evidence the stalemate count refers to.
		suffix += "-" + reportDigest[:12]
	}
	return filepath.Join(stateRoot, "reviews", branchKey(repositoryRoot, branch)+"-"+suffix+".json")
}

// latestPath points at the branch's most recent receipt.
func latestPath(stateRoot, repositoryRoot, branch string) string {
	return filepath.Join(stateRoot, "reviews", branchKey(repositoryRoot, branch)+".latest")
}

// WriteReceipt stores one round's receipt and moves the branch pointer to it.
// Earlier rounds' files stay: overwriting them would erase the only proof that
// the start of the branch was ever reviewed.
func WriteReceipt(stateRoot string, receipt Receipt) (string, error) {
	path := receiptPath(stateRoot, receipt.Repository, receipt.Branch, receipt.HeadCommit, receipt.ReportSHA256, receipt.Effort+"\x00"+receipt.Model+"\x00"+receipt.Refuter+"\x00"+receipt.RefuterModel+
		"\x00"+strings.Join(receipt.WithoutIntegrations, ","))
	// The same protection the rest of the local state store uses. A receipt
	// holds a verbatim review of private source; leaving it readable by every
	// account on a shared machine would make this the one piece of Goalrail
	// state that leaks.
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", fmt.Errorf("create review state directory: %w", err)
	}
	if err := os.Chmod(filepath.Dir(path), 0o700); err != nil {
		return "", fmt.Errorf("protect review state directory: %w", err)
	}
	encoded, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode review receipt: %w", err)
	}
	// Refuse before writing what no reader could ever load. A receipt that
	// stores successfully and then fails every read reports a review as done
	// while leaving nothing usable behind.
	if len(encoded)+1 > receiptBound {
		return "", fmt.Errorf("the review report is larger than the %d-byte receipt bound, so it could never be read back", receiptBound)
	}
	if err := os.WriteFile(path, append(encoded, '\n'), 0o600); err != nil {
		return "", fmt.Errorf("write review receipt: %w", err)
	}
	if err := os.WriteFile(latestPath(stateRoot, receipt.Repository, receipt.Branch),
		[]byte(filepath.Base(path)+"\n"), 0o600); err != nil {
		return "", fmt.Errorf("write review pointer: %w", err)
	}
	return path, nil
}

// ReadReceipt returns one branch's receipt, reporting absence as a miss rather
// than an error. A branch that was never reviewed is an ordinary state.
func ReadReceipt(stateRoot, repositoryRoot, branch string) (Receipt, bool, error) {
	pointer, err := boundedio.ReadRegularFile(latestPath(stateRoot, repositoryRoot, branch), "review pointer", 4096)
	if errors.Is(err, fs.ErrNotExist) {
		return Receipt{}, false, nil
	}
	if err != nil {
		return Receipt{}, false, err
	}
	name := strings.TrimSpace(string(pointer))
	if name != filepath.Base(name) || name == "" {
		return Receipt{}, false, fmt.Errorf("review pointer names an invalid receipt %q", name)
	}
	path := filepath.Join(stateRoot, "reviews", name)
	contents, err := boundedio.ReadRegularFile(path, "review receipt", receiptBound)
	if errors.Is(err, fs.ErrNotExist) {
		// A pointer to a receipt that is gone is corrupted evidence, not an
		// unreviewed branch: reporting absence here would present "never
		// reviewed" for a branch whose record was destroyed.
		return Receipt{}, false, fmt.Errorf("the review pointer names %s, which does not exist", name)
	}
	if errors.Is(err, fs.ErrNotExist) {
		return Receipt{}, false, nil
	}
	if err != nil {
		return Receipt{}, false, err
	}
	var receipt Receipt
	if unmarshalErr := json.Unmarshal(contents, &receipt); unmarshalErr != nil {
		return Receipt{}, false, fmt.Errorf("read review receipt: %w", unmarshalErr)
	}
	if receipt.Schema != ReceiptSchema {
		return Receipt{}, false, fmt.Errorf("review receipt has schema %q, not %q", receipt.Schema, ReceiptSchema)
	}
	// Corrupted or edited evidence must read as unreadable, not as current: a
	// receipt is only evidence while its own bindings hold.
	if receipt.Repository != repositoryRoot || receipt.Branch != branch {
		return Receipt{}, false, fmt.Errorf("review receipt describes %s@%s, not this branch", receipt.Repository, receipt.Branch)
	}
	if receipt.ReportSHA256 != digest([]byte(receipt.Report)) {
		return Receipt{}, false, fmt.Errorf("review receipt report does not match its digest")
	}
	if receipt.Refutation != "" && receipt.RefutationSHA256 != digest([]byte(receipt.Refutation)) {
		return Receipt{}, false, fmt.Errorf("review receipt refutation does not match its digest")
	}
	return receipt, true, nil
}

// State is what a receipt says about the branch it was filed for.
type State string

const (
	// StateAbsent means the branch has never been reviewed.
	StateAbsent State = "absent"
	// StateCurrent means the reviewed diff is the branch's present diff.
	StateCurrent State = "current"
	// StateStale means work has landed since the review.
	StateStale State = "stale"
)

// Status reports the review state of one branch by recomputing the digest.
//
// Nothing is remembered and nothing is acknowledged: committing further work
// makes a review stale by itself, and a fresh review makes it current by
// itself. A condition that has to be dismissed by hand is one nobody reads.
func Status(stateRoot, repositoryRoot, branch string) (State, Receipt, error) {
	receipt, found, err := ReadReceipt(stateRoot, repositoryRoot, branch)
	if err != nil {
		return "", Receipt{}, err
	}
	if !found {
		return StateAbsent, Receipt{}, nil
	}
	head, err := Resolve(repositoryRoot, "HEAD")
	if err != nil {
		return "", Receipt{}, err
	}
	present, err := DiffDigest(repositoryRoot, receipt.BaseCommit, head)
	if err != nil {
		return "", Receipt{}, err
	}
	if present == receipt.DiffSHA256 {
		return StateCurrent, receipt, nil
	}
	return StateStale, receipt, nil
}

// WorkingTreeClean reports whether the repository has no uncommitted or
// untracked changes. It is half of the stalemate measurement: a round that
// changed neither the commits nor the tree is a round the author acted on
// nothing.
func WorkingTreeClean(repositoryRoot string) (bool, error) {
	status, err := git(repositoryRoot, "status", "--porcelain", "--untracked-files=all")
	if err != nil {
		return false, err
	}
	for _, line := range strings.Split(strings.TrimSpace(status), "\n") {
		entry := strings.TrimSpace(line)
		if entry == "" {
			continue
		}
		// Only the *untracked* instructions file Goalrail materialized is not
		// the author's work. Once committed, an edit or deletion of it is
		// ordinary work and must count: skipping it unconditionally would let a
		// real change to the review rules read as a stalemate.
		if strings.HasPrefix(entry, "?? ") && strings.TrimSpace(entry[3:]) == InstructionsPath {
			continue
		}
		return false, nil
	}
	return true, nil
}

// WorktreeRoot names the repository root git recognizes for a path, so an
// invocation from a subdirectory does not key state to the subdirectory.
func WorktreeRoot(path string) (string, error) {
	return git(path, "rev-parse", "--show-toplevel")
}
