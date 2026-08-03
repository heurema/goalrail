package harness

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
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

const maxUpdateWriteAttempts = 3

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

	// MovedFrom names the earlier canons the replaced files matched, so the
	// report says what the repository moved from and not only that it moved.
	// Empty when the replaced content matched no canon this binary knows.
	MovedFrom []string `json:"moved_from,omitempty"`

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
	return update(input, updateTestHooks{})
}

type updateTestHooks struct {
	beforeBackup  func()
	afterBackup   func()
	beforeReplace func(string)
}

// update accepts unexported hooks so the inspect, backup, and per-file replace
// boundaries are deterministic in tests. Production callers always use Update
// and therefore cannot inject behavior into the write path.
func update(input UpdateInput, hooks updateTestHooks) (UpdateReport, error) {
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
	for _, finding := range state.Files {
		if finding.State == FileSuperseded {
			report.Notes = append(report.Notes,
				finding.Path+" is overlay content the canon no longer defines; "+
					"an update never deletes repository content, so removing it stays your act")
		}
	}
	// Settled, not Current: a superseded file is unreachable by re-materializing,
	// and treating it as unfinished business would make every subsequent update
	// report failure against a state it cannot change.
	if state.Settled() {
		report.AlreadyCurrent, report.Verified = true, true
		return report, nil
	}

	for attempt := 1; attempt <= maxUpdateWriteAttempts; attempt++ {
		if attempt > 1 {
			state, err = InspectOverlay(input.RepositoryRoot)
			if err != nil {
				return report, err
			}
		}
		edited := editedPaths(state)
		if len(edited) > 0 && !input.DiscardLocalEdits {
			// Drift and behind-ness can coincide; the edit wins. A pending upgrade is
			// not permission to overwrite.
			return UpdateReport{}, fmt.Errorf("%w: %s; keep them, or re-run with --discard-local-edits",
				ErrLocalEdits, strings.Join(edited, ", "))
		}
		for _, finding := range state.Files {
			if finding.MatchedCanon != "" && finding.State == FileBehind {
				report.MovedFrom = appendUnique(report.MovedFrom, finding.MatchedCanon)
			}
		}
		if hooks.beforeBackup != nil {
			hooks.beforeBackup()
			hooks.beforeBackup = nil
		}

		backup, backupErr := backupReplaced(
			input.RepositoryRoot,
			input.StateRoot,
			state,
			now,
			report.Backup,
		)
		if backup != "" {
			report.Backup = backup
		}
		if hooks.afterBackup != nil {
			hooks.afterBackup()
			hooks.afterBackup = nil
		}
		if errors.Is(backupErr, errOverlayChangedDuringUpdate) {
			if !input.DiscardLocalEdits {
				return report, fmt.Errorf(
					"%w: %s; changed while backup was read, keep them or re-run with --discard-local-edits",
					ErrLocalEdits,
					strings.TrimPrefix(backupErr.Error(), errOverlayChangedDuringUpdate.Error()+": "),
				)
			}
			if attempt < maxUpdateWriteAttempts {
				continue
			}
			return report, fmt.Errorf(
				"overlay kept changing during %d update attempts: %w",
				maxUpdateWriteAttempts,
				backupErr,
			)
		}
		if backupErr != nil {
			return report, backupErr
		}

		outcomes, materializeErr := materializeExpected(
			input.RepositoryRoot,
			true,
			expectedUpdateDigests(state),
			hooks.beforeReplace,
		)
		report.Files = mergeFileOutcomes(report.Files, outcomes)
		if input.DiscardLocalEdits {
			report.DiscardedLocalEdits = recordDiscardedEdits(
				report.DiscardedLocalEdits,
				state,
				outcomes,
			)
		}
		if errors.Is(materializeErr, errOverlayChangedDuringUpdate) {
			if !input.DiscardLocalEdits {
				return report, fmt.Errorf(
					"%w: %s; changed after inspection, keep them or re-run with --discard-local-edits",
					ErrLocalEdits,
					strings.TrimPrefix(materializeErr.Error(), errOverlayChangedDuringUpdate.Error()+": "),
				)
			}
			if attempt < maxUpdateWriteAttempts {
				// Re-inspect after any completed earlier replacements and back up the
				// latest changed bytes before honoring the explicit discard on a
				// bounded retry.
				continue
			}
			return report, fmt.Errorf(
				"overlay kept changing during %d update attempts: %w",
				maxUpdateWriteAttempts,
				materializeErr,
			)
		}
		if materializeErr != nil {
			// The backup exists and earlier files may already be rewritten. The
			// report is the only thing carrying the backup path, so it survives the
			// error instead of being discarded at the moment it is needed most.
			return report, materializeErr
		}
		break
	}

	// Verify by comparing digests rather than by trusting the writes. A report
	// that says "updated" because no error was returned is the same class of
	// claim this package exists to remove.
	after, err := InspectOverlay(input.RepositoryRoot)
	if err != nil {
		return report, err
	}
	report.Verified = after.Settled()
	if !report.Verified {
		return report, errors.New("the overlay does not match the canon after the update")
	}
	return report, nil
}

func editedPaths(state OverlayState) []string {
	var edited []string
	for _, finding := range state.Files {
		if finding.State == FileEdited {
			edited = append(edited, finding.Path)
		}
	}
	return edited
}

func expectedUpdateDigests(state OverlayState) map[string]string {
	expected := make(map[string]string)
	for _, finding := range state.Files {
		if finding.State == FileSuperseded {
			continue
		}
		// Track every canon-defined path, including files that were current at
		// inspection time. A current file can become edited before Materialize
		// re-inspects it; omitting it here would turn that late edit into a newly
		// overwriteable FileEdited finding without a corresponding backup.
		expected[finding.Path] = finding.Digest
	}
	return expected
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func mergeFileOutcomes(existing, observed []FileOutcome) []FileOutcome {
	positions := make(map[string]int, len(existing)+len(observed))
	merged := append([]FileOutcome(nil), existing...)
	for index, outcome := range merged {
		positions[outcome.Path] = index
	}
	for _, outcome := range observed {
		index, found := positions[outcome.Path]
		if !found {
			positions[outcome.Path] = len(merged)
			merged = append(merged, outcome)
			continue
		}
		if fileActionChanged(outcome.Action) && !fileActionChanged(merged[index].Action) {
			merged[index] = outcome
		}
	}
	return merged
}

func fileActionChanged(action FileAction) bool {
	return action == ActionCreated || action == ActionUpdated
}

func recordDiscardedEdits(
	existing []string,
	state OverlayState,
	outcomes []FileOutcome,
) []string {
	states := make(map[string]FileState, len(state.Files))
	for _, finding := range state.Files {
		states[finding.Path] = finding.State
	}
	for _, outcome := range outcomes {
		if outcome.Action == ActionUpdated && states[outcome.Path] == FileEdited {
			existing = appendUnique(existing, outcome.Path)
		}
	}
	return existing
}

type backupManifest struct {
	Canon string        `json:"replaced_toward_canon"`
	Files []FileFinding `json:"files"`
}

// backupReplaced copies every file the update is about to replace into one
// cumulative recovery directory. A bounded retry refreshes the entries that
// changed instead of orphaning an earlier partial attempt's recovery point.
func backupReplaced(
	repositoryRoot, stateRoot string,
	state OverlayState,
	now func() time.Time,
	existingDirectory string,
) (string, error) {
	var replaced []FileFinding
	for _, finding := range state.Files {
		switch finding.State {
		case FileBehind, FileEdited:
			replaced = append(replaced, finding)
		}
	}
	if len(replaced) == 0 {
		return existingDirectory, nil
	}
	if strings.TrimSpace(stateRoot) == "" {
		return "", errors.New("an update needs a state root to keep the replaced files in")
	}
	directory := existingDirectory
	manifest := backupManifest{}
	if directory == "" {
		base := filepath.Join(stateRoot, filepath.FromSlash(BackupDirectory),
			now().UTC().Format("20060102T150405Z"))
		if err := os.MkdirAll(filepath.Dir(base), 0o700); err != nil {
			return "", fmt.Errorf("create backup directory: %w", err)
		}
		// Exclusive creation, with a suffix on collision: two updates inside one
		// second must not share a directory, or the second silently overwrites the
		// first's recovery point.
		directory = base
		for attempt := 2; ; attempt++ {
			mkdirErr := os.Mkdir(directory, 0o700)
			if mkdirErr == nil {
				break
			}
			if !errors.Is(mkdirErr, fs.ErrExist) {
				return "", fmt.Errorf("create backup directory: %w", mkdirErr)
			}
			directory = fmt.Sprintf("%s-%d", base, attempt)
		}
	} else {
		raw, err := os.ReadFile(filepath.Join(directory, "manifest.json"))
		if err != nil {
			return directory, fmt.Errorf("read cumulative backup manifest: %w", err)
		}
		if err := json.Unmarshal(raw, &manifest); err != nil {
			return directory, fmt.Errorf("decode cumulative backup manifest: %w", err)
		}
	}

	actual := make([]FileFinding, 0, len(replaced))
	var changed []string
	for _, finding := range replaced {
		inspectedDigest := finding.Digest
		content, err := os.ReadFile(filepath.Join(repositoryRoot, filepath.FromSlash(finding.Path)))
		if err != nil {
			return directory, fmt.Errorf("read %s: %w", finding.Path, err)
		}
		finding = findingForBackupBytes(finding, Digest(content))
		if finding.Digest != inspectedDigest {
			changed = append(changed, finding.Path)
		}
		destination := filepath.Join(directory, filepath.FromSlash(finding.Path))
		if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
			return directory, fmt.Errorf("create backup directory: %w", err)
		}
		if err := os.WriteFile(destination, content, 0o600); err != nil {
			return directory, fmt.Errorf("write backup of %s: %w", finding.Path, err)
		}
		actual = append(actual, finding)
	}
	manifest.Files = mergeBackupFindings(manifest.Files, actual)
	if canon, canonErr := CurrentCanon(); canonErr == nil {
		manifest.Canon = canon.ID
	}
	encoded, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return directory, err
	}
	if err := os.WriteFile(filepath.Join(directory, "manifest.json"), append(encoded, '\n'), 0o600); err != nil {
		return directory, fmt.Errorf("write backup manifest: %w", err)
	}
	if len(changed) > 0 {
		return directory, &overlayChangedDuringUpdateError{paths: changed}
	}
	return directory, nil
}

func findingForBackupBytes(finding FileFinding, digest string) FileFinding {
	finding.Digest = digest
	finding.MatchedCanon = ""
	switch matched := matchingPreviousCanon(finding.Path, digest); {
	case digest == finding.Expected:
		finding.State = FileCurrent
	case matched != "":
		finding.State = FileBehind
		finding.MatchedCanon = matched
	default:
		finding.State = FileEdited
	}
	return finding
}

func mergeBackupFindings(existing, observed []FileFinding) []FileFinding {
	positions := make(map[string]int, len(existing)+len(observed))
	merged := append([]FileFinding(nil), existing...)
	for index, finding := range merged {
		positions[finding.Path] = index
	}
	for _, finding := range observed {
		if index, found := positions[finding.Path]; found {
			merged[index] = finding
			continue
		}
		positions[finding.Path] = len(merged)
		merged = append(merged, finding)
	}
	sort.Slice(merged, func(first, second int) bool {
		return merged[first].Path < merged[second].Path
	})
	return merged
}
