package harness

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ErrLocalEdits reports overlay files that differ from every canon this binary
// knows. A user who edited a template did it deliberately, and a command whose
// purpose is refreshing files must not be the thing that destroys the only copy
// of that edit.
var ErrLocalEdits = errors.New("the overlay carries local edits")

// BackupDirectory is where replaced files are copied, inside the state root
// rather than the repository. Recovery does not lean on version control: the
// files an update replaces may never have been committed.
const BackupDirectory = "harness/backups"

// UpdateInput is what an update needs, with the clock injectable so a backup path
// is reproducible in a test.
type UpdateInput struct {
	RepositoryRoot string
	StateRoot      string

	// DiscardLocalEdits proceeds over files that match no known canon. The report
	// says that it did, so the loss is recorded at the moment it happens rather
	// than discovered later.
	DiscardLocalEdits bool

	Now func() time.Time
}

// UpdateReport is what an update tells the user it did.
type UpdateReport struct {
	Repository string `json:"repository"`

	// Canon is the canon the repository now matches, and Version the binary that
	// carried it.
	Canon   string `json:"canon"`
	Version string `json:"version"`

	// AlreadyCurrent reports an update that had nothing to do.
	AlreadyCurrent bool `json:"already_current"`

	Files []FileOutcome `json:"files"`

	// Backup names where the replaced files were copied, empty when nothing was
	// replaced.
	Backup string `json:"backup,omitempty"`

	// DiscardedLocalEdits names files whose local content was overwritten on
	// explicit instruction.
	DiscardedLocalEdits []string `json:"discarded_local_edits,omitempty"`

	// Verified reports that the result was compared against the canon rather than
	// assumed from a successful write.
	Verified bool `json:"verified"`

	Notes []string `json:"notes,omitempty"`
}

// Update brings a repository's overlay to the canon this binary carries.
//
// It runs no Node, makes no network request, and consults no release channel: the
// canon is already inside the binary. Updating the binary itself is a different
// act, and the command says so rather than letting the word imply it.
func Update(input UpdateInput) (UpdateReport, error) {
	now := input.Now
	if now == nil {
		now = time.Now
	}
	canon, err := CurrentCanon()
	if err != nil {
		return UpdateReport{}, err
	}
	report := UpdateReport{
		Repository: input.RepositoryRoot,
		Canon:      canon.ID,
		Version:    Version,
		Notes: []string{
			"this updates the harness in this repository, not the gr binary",
		},
	}

	state, err := InspectOverlay(input.RepositoryRoot)
	if err != nil {
		return UpdateReport{}, err
	}
	if state.Current {
		report.AlreadyCurrent, report.Verified = true, true
		return report, nil
	}

	var edited []string
	for _, finding := range state.Files {
		if finding.State == FileEdited {
			edited = append(edited, finding.Path)
		}
	}
	if len(edited) > 0 && !input.DiscardLocalEdits {
		// Drift and behind-ness can coincide; the edit wins. A pending upgrade is
		// not permission to overwrite.
		return UpdateReport{}, fmt.Errorf("%w: %s; keep them, or re-run with --discard-local-edits",
			ErrLocalEdits, strings.Join(edited, ", "))
	}

	backup, err := backupReplaced(input.RepositoryRoot, input.StateRoot, state, now)
	if err != nil {
		return UpdateReport{}, err
	}
	report.Backup = backup

	outcomes, err := Materialize(input.RepositoryRoot, true)
	if err != nil {
		return UpdateReport{}, err
	}
	report.Files = outcomes
	if input.DiscardLocalEdits {
		report.DiscardedLocalEdits = edited
	}

	// Verify by comparing digests rather than by trusting the writes. A report
	// that says "updated" because no error was returned is the same class of
	// claim this package exists to remove.
	after, err := InspectOverlay(input.RepositoryRoot)
	if err != nil {
		return UpdateReport{}, err
	}
	report.Verified = after.Current
	if !after.Current {
		return report, errors.New("the overlay does not match the canon after the update")
	}
	return report, nil
}

// backupReplaced copies every file the update is about to replace into a
// timestamped directory under the state root, and returns its path.
func backupReplaced(
	repositoryRoot, stateRoot string,
	state OverlayState,
	now func() time.Time,
) (string, error) {
	var replaced []FileFinding
	for _, finding := range state.Files {
		switch finding.State {
		case FileBehind, FileEdited:
			replaced = append(replaced, finding)
		}
	}
	if len(replaced) == 0 {
		return "", nil
	}
	if strings.TrimSpace(stateRoot) == "" {
		return "", errors.New("an update needs a state root to keep the replaced files in")
	}
	directory := filepath.Join(stateRoot, filepath.FromSlash(BackupDirectory),
		now().UTC().Format("20060102T150405Z"))
	for _, finding := range replaced {
		content, err := os.ReadFile(filepath.Join(repositoryRoot, filepath.FromSlash(finding.Path)))
		if err != nil {
			return "", fmt.Errorf("read %s: %w", finding.Path, err)
		}
		destination := filepath.Join(directory, filepath.FromSlash(finding.Path))
		if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
			return "", fmt.Errorf("create backup directory: %w", err)
		}
		if err := os.WriteFile(destination, content, 0o600); err != nil {
			return "", fmt.Errorf("write backup of %s: %w", finding.Path, err)
		}
	}
	return directory, nil
}
