package harness

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/heurema/goalrail/internal/ambient"
)

// previousCanons records overlay versions that shipped before the embedded one,
// append-only and oldest-first. It exists so a repository's overlay can be told
// apart from a locally edited one: a file matching a version listed here is
// behind, while a file matching nothing known was edited.
//
// The first entry is the canon that shipped from v0.1.1 through v0.1.8,
// recorded the day the canon first changed. Without it, that change would have
// classified every adopter's overlay as edited — their files match what an
// earlier binary materialized, they edited nothing — and `gr update` would have
// demanded --discard-local-edits for a routine upgrade. Found by the
// adversarial exchange before it could happen; the migration test pins this
// exact transition by digest.
var previousCanons = []Canon{
	{
		ID: "sha256:12cf770fb566fd4ae7bbb9d8299064cbbe9d61386c5676850a2d8f329c5ee4ad",
		Files: []CanonFile{
			{Path: "openspec/schemas/goalrail-intent/schema.yaml", Digest: "sha256:b1599076b9ea7b00a6bc0268ce84e06439c0b571def31f6957cfacba252639d8"},
			{Path: "openspec/schemas/goalrail-intent/templates/context.md", Digest: "sha256:6b7ad3dcd86a1a51ba9d5b6d589d302c0cc293fcfee4ebc2d7efb151a134940e"},
			{Path: "openspec/schemas/goalrail-intent/templates/design.md", Digest: "sha256:3653a4ed89b6d1b2f575b75b98096c52744f5ee4a174d76146f6013f044b4cdd"},
			{Path: "openspec/schemas/goalrail-intent/templates/intent.md", Digest: "sha256:11ce3c0af731300bcc785b8ea0a94850359b73607deb966858e52ef39d45c1e5"},
			{Path: "openspec/schemas/goalrail-intent/templates/proposal.md", Digest: "sha256:b23361a7591957c6b9b708ea71ae804e86050909a8119eabf18be42b2fffc63e"},
			{Path: "openspec/schemas/goalrail-intent/templates/spec.md", Digest: "sha256:64332d907be02c4402512504986392e18eb452715d0d17dc22acde2e4578e5dc"},
			{Path: "openspec/schemas/goalrail-intent/templates/tasks.md", Digest: "sha256:b2a6a4c08c15f347a1d8c3e2d43e0c8fb066dc5cc0feb795f47555f176f9c421"},
		},
	},
}

// FileState is what one overlay file is, judged against the canon this binary
// carries and the ones it remembers.
type FileState string

const (
	// FileCurrent matches the embedded canon.
	FileCurrent FileState = "current"

	// FileBehind matches a canon that shipped earlier, so an update can restore
	// it without destroying anything.
	FileBehind FileState = "behind"

	// FileEdited matches no canon this binary knows. It may be a deliberate local
	// change, so nothing overwrites it without being asked.
	FileEdited FileState = "edited"

	// FileMissing is absent from the repository.
	FileMissing FileState = "missing"

	// FileSuperseded exists in the repository and in an earlier canon, but not in
	// the current one: the overlay dropped it.
	FileSuperseded FileState = "superseded"
)

// FileFinding is one overlay file's state, with enough detail to act on.
type FileFinding struct {
	Path  string    `json:"path"`
	State FileState `json:"state"`

	// Digest is what the repository holds, empty when the file is absent.
	Digest string `json:"digest,omitempty"`

	// Expected is what the current canon defines, empty for a superseded file.
	Expected string `json:"expected,omitempty"`

	// MatchedCanon identifies the earlier canon a behind or superseded file
	// matches, so a report can name what the repository is moving from rather
	// than only that it differs.
	MatchedCanon string `json:"matched_canon,omitempty"`
}

// OverlayState answers, for one repository, whether the harness files are there
// and whether they are the ones this binary would install.
type OverlayState struct {
	// Canon identifies the overlay this binary carries.
	Canon string `json:"canon"`

	// Present is true when any canon file exists.
	Present bool `json:"present"`

	// Complete is true when every canon file exists.
	Complete bool `json:"complete"`

	// Current is true when the overlay is complete and every file matches.
	Current bool `json:"current"`

	// Behind is true when the only differences are files a previous canon
	// explains, so an update is a restoration rather than an overwrite.
	Behind bool `json:"behind"`

	// Edited is true when at least one file matches no canon this binary knows.
	// It suppresses an unattended update: a local edit may be deliberate, and a
	// command whose purpose is refreshing files must not destroy the only copy.
	Edited bool `json:"edited"`

	Files []FileFinding `json:"files"`
}

// Drifted returns the findings a report should name, in stable order. A state
// that says only "something differs" leaves the user diffing the overlay by hand,
// which is the work this exists to remove.
func (state OverlayState) Drifted() []FileFinding {
	var drifted []FileFinding
	for _, finding := range state.Files {
		switch finding.State {
		case FileCurrent:
		default:
			drifted = append(drifted, finding)
		}
	}
	return drifted
}

// InspectOverlay classifies a repository's overlay without changing anything.
func InspectOverlay(repositoryRoot string) (OverlayState, error) {
	canon, err := CurrentCanon()
	if err != nil {
		return OverlayState{}, err
	}
	state := OverlayState{Canon: canon.ID}

	for _, file := range canon.Files {
		finding := FileFinding{Path: file.Path, Expected: file.Digest}
		content, readErr := os.ReadFile(filepath.Join(repositoryRoot, filepath.FromSlash(file.Path)))
		switch {
		case errors.Is(readErr, fs.ErrNotExist):
			finding.State = FileMissing
		case readErr != nil:
			return OverlayState{}, fmt.Errorf("read %s: %w", file.Path, readErr)
		default:
			finding.Digest = Digest(content)
			switch matched := matchingPreviousCanon(file.Path, finding.Digest); {
			case finding.Digest == file.Digest:
				finding.State = FileCurrent
			case matched != "":
				finding.State = FileBehind
				finding.MatchedCanon = matched
			default:
				finding.State = FileEdited
			}
			state.Present = true
		}
		state.Files = append(state.Files, finding)
	}

	// A file an earlier canon carried and this one does not is reported rather
	// than ignored: it is overlay content that no longer belongs, and the user
	// cannot tell that from looking at it.
	for _, superseded := range supersededPaths(canon) {
		content, readErr := os.ReadFile(filepath.Join(repositoryRoot, filepath.FromSlash(superseded)))
		if readErr != nil {
			continue
		}
		digest := Digest(content)
		state.Present = true
		state.Files = append(state.Files, FileFinding{
			Path:         superseded,
			State:        FileSuperseded,
			Digest:       digest,
			MatchedCanon: matchingPreviousCanon(superseded, digest),
		})
	}

	sort.Slice(state.Files, func(first, second int) bool {
		return state.Files[first].Path < state.Files[second].Path
	})

	state.Complete = true
	for _, finding := range state.Files {
		switch finding.State {
		case FileMissing:
			state.Complete = false
		case FileEdited:
			state.Edited = true
		case FileBehind, FileSuperseded:
			state.Behind = true
		}
	}
	state.Current = state.Complete && !state.Edited && !state.Behind
	if state.Edited {
		// Drift and behind-ness can coincide. Reporting both would invite an
		// update to treat a pending upgrade as permission to overwrite, so the
		// edit wins as the state that decides what happens next.
		state.Behind = false
	}
	return state, nil
}

// matchingPreviousCanon returns the identity of the earlier canon whose version
// of one path matches the given digest, or an empty string. The newest match
// wins, since previousCanons is oldest-first.
func matchingPreviousCanon(relativePath, digest string) string {
	matched := ""
	for _, previous := range previousCanons {
		if known, present := previous.Digest(relativePath); present && known == digest {
			matched = previous.ID
		}
	}
	return matched
}

// Settled reports whether an update has anything left to do: every canon file is
// present and matches the current canon. A superseded file does not unsettle it —
// that is repository content the canon no longer defines, reported for the user
// to remove, and re-materializing can never change it. Folding it into the
// verdict made an update fail its own verification forever in that state.
func (state OverlayState) Settled() bool {
	for _, finding := range state.Files {
		switch finding.State {
		case FileCurrent, FileSuperseded:
		default:
			return false
		}
	}
	return true
}

// supersededPaths returns paths a previous canon carried that the current one
// does not.
func supersededPaths(current Canon) []string {
	var dropped []string
	seen := make(map[string]bool)
	for _, previous := range previousCanons {
		for _, file := range previous.Files {
			if _, stillCanonical := current.Digest(file.Path); stillCanonical || seen[file.Path] {
				continue
			}
			seen[file.Path] = true
			dropped = append(dropped, file.Path)
		}
	}
	sort.Strings(dropped)
	return dropped
}

// FileAction is what materialization did to one file.
type FileAction string

const (
	ActionCreated   FileAction = "created"
	ActionUpdated   FileAction = "updated"
	ActionUnchanged FileAction = "unchanged"

	// ActionKept means the file differs from the canon and was left alone,
	// because overwriting was not asked for.
	ActionKept FileAction = "kept"
)

// FileOutcome is one file's result, reported so the user knows what changed
// without diffing.
type FileOutcome struct {
	Path   string     `json:"path"`
	Action FileAction `json:"action"`
	State  FileState  `json:"state,omitempty"`
}

// Materialize writes the overlay into a repository.
//
// Overwriting is opt-in. A file that differs from the canon may carry a
// deliberate local edit that was never committed, so the default is to leave it
// and report it; the caller that has asked the user decides otherwise.
func Materialize(repositoryRoot string, overwrite bool) ([]FileOutcome, error) {
	return materializeExpected(repositoryRoot, overwrite, nil, nil)
}

var errOverlayChangedDuringUpdate = errors.New("overlay changed during update")

type overlayChangedDuringUpdateError struct {
	paths []string
}

func (err *overlayChangedDuringUpdateError) Error() string {
	return fmt.Sprintf("%s: %s", errOverlayChangedDuringUpdate, strings.Join(err.paths, ", "))
}

func (err *overlayChangedDuringUpdateError) Unwrap() error {
	return errOverlayChangedDuringUpdate
}

// materializeExpected is the update-specific compare-before-write seam. The
// ordinary Materialize API retains its explicit overwrite behavior, while an
// update can require every replaceable file to still have the digest it backed
// up. A bulk comparison fails before any write, and a second per-file comparison
// narrows the window immediately before each replacement. The latter can detect
// a change after earlier paths were already updated, so callers retain both the
// recovery point and the completed outcomes when reporting that partial update.
func materializeExpected(
	repositoryRoot string,
	overwrite bool,
	expected map[string]string,
	beforeReplace func(string),
) ([]FileOutcome, error) {
	state, err := InspectOverlay(repositoryRoot)
	if err != nil {
		return nil, err
	}
	if len(expected) > 0 {
		var changed []string
		for _, finding := range state.Files {
			digest, tracked := expected[finding.Path]
			if !tracked || finding.State == FileCurrent {
				continue
			}
			if finding.Digest != digest {
				changed = append(changed, finding.Path)
			}
		}
		if len(changed) > 0 {
			return nil, &overlayChangedDuringUpdateError{paths: changed}
		}
	}
	outcomes := make([]FileOutcome, 0, len(state.Files))
	for _, finding := range state.Files {
		if finding.State == FileSuperseded {
			// Removing it is a deletion of repository content, which is the user's
			// act; reporting it is ours.
			outcomes = append(outcomes, FileOutcome{
				Path:   finding.Path,
				Action: ActionKept,
				State:  finding.State,
			})
			continue
		}
		if finding.State == FileCurrent {
			outcomes = append(outcomes, FileOutcome{
				Path:   finding.Path,
				Action: ActionUnchanged,
				State:  finding.State,
			})
			continue
		}
		if finding.State != FileMissing && !overwrite {
			outcomes = append(outcomes, FileOutcome{
				Path:   finding.Path,
				Action: ActionKept,
				State:  finding.State,
			})
			continue
		}
		if beforeReplace != nil {
			beforeReplace(finding.Path)
		}
		// The bulk comparison above prevents an update that already knows the
		// overlay changed from starting any writes. Re-read each path again at the
		// narrowest portable seam before its own replacement so later files do not
		// inherit the time spent writing earlier ones. This is optimistic, not an
		// atomic filesystem compare-and-swap: a non-cooperating writer can still
		// change the path after this read.
		if expectedErr := requireExpectedDigest(repositoryRoot, finding.Path, expected); expectedErr != nil {
			return outcomes, expectedErr
		}
		content, contentErr := Content(finding.Path)
		if contentErr != nil {
			return outcomes, contentErr
		}
		absolute := filepath.Join(repositoryRoot, filepath.FromSlash(finding.Path))
		// The overlay honours the same containment as the registration: a checkout
		// can ship a symlink at any of these paths, and writing through one would
		// land canonical content outside the repository the user named.
		if containErr := ambient.EnsureWriteWithinRepository(repositoryRoot, absolute); containErr != nil {
			return outcomes, containErr
		}
		if mkdirErr := os.MkdirAll(filepath.Dir(absolute), 0o755); mkdirErr != nil {
			return outcomes, fmt.Errorf("create %s: %w", filepath.Dir(finding.Path), mkdirErr)
		}
		if writeErr := os.WriteFile(absolute, content, 0o644); writeErr != nil {
			return outcomes, fmt.Errorf("write %s: %w", finding.Path, writeErr)
		}
		action := ActionUpdated
		if finding.State == FileMissing {
			action = ActionCreated
		}
		outcomes = append(outcomes, FileOutcome{
			Path:   finding.Path,
			Action: action,
			State:  FileCurrent,
		})
	}
	return outcomes, nil
}

func requireExpectedDigest(repositoryRoot, relativePath string, expected map[string]string) error {
	if len(expected) == 0 {
		return nil
	}
	expectedDigest, tracked := expected[relativePath]
	if !tracked {
		return &overlayChangedDuringUpdateError{paths: []string{relativePath}}
	}
	content, err := os.ReadFile(filepath.Join(repositoryRoot, filepath.FromSlash(relativePath)))
	actualDigest := ""
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return fmt.Errorf("recheck %s before update: %w", relativePath, err)
	default:
		actualDigest = Digest(content)
	}
	if actualDigest != expectedDigest {
		return &overlayChangedDuringUpdateError{paths: []string{relativePath}}
	}
	return nil
}
