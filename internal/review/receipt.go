package review

import (
	"bytes"
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

	// BaseRef is what the branch was reviewed against, as the caller named it;
	// BaseCommit is what that resolved to at review time. A branch name moves,
	// a commit does not, and the digest below is only reproducible from the
	// commit.
	BaseRef    string `json:"base_ref"`
	BaseCommit string `json:"base_commit"`
	HeadCommit string `json:"head_commit"`

	// DiffSHA256 digests the canonical diff of the reviewed range. It is what
	// makes "this review still describes this branch" a computation rather than
	// a belief.
	DiffSHA256 string `json:"diff_sha256"`

	Reviewer   string `json:"reviewer"`
	Author     string `json:"author"`
	ReviewedAt string `json:"reviewed_at"`

	// Report is verbatim, and ReportSHA256 digests exactly those bytes.
	Report       string `json:"report"`
	ReportSHA256 string `json:"report_sha256"`

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
	command := exec.Command("git", append([]string{"-C", repositoryRoot}, arguments...)...)
	command.Env = strippedEnvironment(os.Environ(), []string{"GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE", "GIT_OBJECT_DIRECTORY"})
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
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
		"diff", "--no-ext-diff", "--no-color", "--no-textconv",
		"--full-index", "--no-renames",
		baseCommit + "..." + headCommit,
	}
}

// DiffDigest measures the reviewed range.
func DiffDigest(repositoryRoot, baseCommit, headCommit string) (string, error) {
	rendered, err := git(repositoryRoot, canonicalDiffArguments(baseCommit, headCommit)...)
	if err != nil {
		return "", err
	}
	return digest([]byte(rendered)), nil
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
func receiptPath(stateRoot, repositoryRoot, branch string) string {
	key := digest([]byte(repositoryRoot + "\x00" + branch))
	return filepath.Join(stateRoot, "reviews", key+".json")
}

// WriteReceipt stores one receipt.
func WriteReceipt(stateRoot string, receipt Receipt) (string, error) {
	path := receiptPath(stateRoot, receipt.Repository, receipt.Branch)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create review state directory: %w", err)
	}
	encoded, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode review receipt: %w", err)
	}
	if err := os.WriteFile(path, append(encoded, '\n'), 0o644); err != nil {
		return "", fmt.Errorf("write review receipt: %w", err)
	}
	return path, nil
}

// ReadReceipt returns one branch's receipt, reporting absence as a miss rather
// than an error. A branch that was never reviewed is an ordinary state.
func ReadReceipt(stateRoot, repositoryRoot, branch string) (Receipt, bool, error) {
	path := receiptPath(stateRoot, repositoryRoot, branch)
	contents, err := boundedio.ReadRegularFile(path, "review receipt", receiptBound)
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
