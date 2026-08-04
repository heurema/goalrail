// Package ambient attaches Goalrail to sessions the user starts in their own
// scaffold, instead of launching a provider itself.
//
// Two consents scope it. Connection registers persistent session hooks in the
// scaffold's user configuration; initialization marks one repository as
// participating. A hook fires for every session the user starts anywhere, so
// acting outside an initialized repository would monitor unrelated work — the
// marker check is therefore the first act of every invocation.
//
// Everything here is fail-quiet toward the scaffold: an internal error must
// never block, break, or delay an ordinary session. Errors worth keeping are
// recorded in the state root, where the owner can read them later. This is the
// deliberate opposite of the wrapper lifecycle, which refuses to launch a run
// it cannot certify.
package ambient

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

const (
	// MarkerPath is the explicit opt-in for one repository. The `.goalrail`
	// directory itself cannot mean opt-in: it already holds unrelated local
	// content, and the reserved escalation path lives inside it.
	MarkerPath = ".goalrail/ambient.json"

	// MarkerSchema versions the marker so a future shape change is legible
	// rather than silent.
	MarkerSchema = "goalrail.ambient-marker/v0"
)

// Marker is the on-disk opt-in record for one repository.
type Marker struct {
	Schema        string    `json:"schema"`
	InitializedAt time.Time `json:"initialized_at"`
	Adoption      *Adoption `json:"adoption,omitempty"`
}

// Adoption is evidence about one schema replacement. HadRules is stored
// separately from the digest because an absent block has a stable digest too,
// but must never create a standing diagnosis line.
type Adoption struct {
	ReplacedSchema string    `json:"replaced_schema"`
	AdoptedAt      time.Time `json:"adopted_at"`
	RulesDigest    string    `json:"rules_sha256"`
	HadRules       bool      `json:"had_rules"`
}

// AmbientAnnouncement is the exact text an attached session is told.
//
// It is its own constant rather than the launch announcement: that one
// promises the run ends as `blocked`, and an attached session mints no run at
// all. The discipline is identical — name the channel and its conditions, name
// no provider, describe no work item, suggest no conflict, never tell the
// session to look for one.
const AmbientAnnouncement = `This repository accepts one escalation.

If the work item cannot be completed as specified from this repository alone,
write the question to ` + ReservedEscalationPath + ` and change nothing else in
the same act. The question is retained for the owner, who answers it as a new
intent version; the current session does not resume.

The payload format is goalrail.escalation/v0.`

// ReservedEscalationPath mirrors the wrapper's reserved path. Attached
// sessions and wrapped runs use one channel, not two.
const ReservedEscalationPath = ".goalrail/blocked.md"

// ErrAdoptionNotRecorded means an existing valid marker could not be extended
// with additive adoption evidence. The existing marker remains authoritative;
// callers may report this degraded evidence path without treating the harness
// as uninitialized.
var ErrAdoptionNotRecorded = errors.New("adoption record was not written")

// Initialize marks a repository as participating. It is an explicit user act,
// so unlike the hook paths it reports failure loudly.
func Initialize(repositoryRoot string, now func() time.Time) (Marker, bool, error) {
	return InitializeWithAdoption(repositoryRoot, now, nil)
}

// InitializeWithAdoption initializes the marker and, when a schema was
// replaced during this invocation, records that additive evidence even if an
// older marker already existed.
func InitializeWithAdoption(repositoryRoot string, now func() time.Time, adoption *Adoption) (Marker, bool, error) {
	markerPath := filepath.Join(repositoryRoot, filepath.FromSlash(MarkerPath))
	if existing, err := ReadMarker(repositoryRoot); err == nil {
		if adoption == nil {
			return existing, false, nil
		}
		updated := existing
		updated.Adoption = completedAdoption(adoption, now)
		if err := writeMarker(markerPath, updated); err != nil {
			return existing, false, fmt.Errorf("%w: %w", ErrAdoptionNotRecorded, err)
		}
		return updated, false, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return Marker{}, false, err
	}
	if err := os.MkdirAll(filepath.Dir(markerPath), 0o755); err != nil {
		return Marker{}, false, fmt.Errorf("create Goalrail directory: %w", err)
	}
	marker := Marker{Schema: MarkerSchema, InitializedAt: now().UTC()}
	if adoption != nil {
		marker.Adoption = completedAdoption(adoption, now)
	}
	if err := writeMarker(markerPath, marker); err != nil {
		return Marker{}, false, err
	}
	return marker, true, nil
}

func completedAdoption(adoption *Adoption, now func() time.Time) *Adoption {
	completed := *adoption
	if completed.AdoptedAt.IsZero() {
		completed.AdoptedAt = now().UTC()
	} else {
		completed.AdoptedAt = completed.AdoptedAt.UTC()
	}
	return &completed
}

func writeMarker(markerPath string, marker Marker) error {
	return writeMarkerWithRename(markerPath, marker, os.Rename)
}

func writeMarkerWithRename(markerPath string, marker Marker, rename func(string, string) error) error {
	encoded, err := json.MarshalIndent(marker, "", "  ")
	if err != nil {
		return fmt.Errorf("encode ambient marker: %w", err)
	}
	mode := os.FileMode(0o644)
	if existing, err := os.Stat(markerPath); err == nil {
		mode = existing.Mode().Perm()
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("inspect ambient marker mode: %w", err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(markerPath), ".ambient-marker-*")
	if err != nil {
		return fmt.Errorf("create temporary ambient marker: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	closeOnFailure := func() {
		_ = temporary.Close()
	}
	if _, err := temporary.Write(append(encoded, '\n')); err != nil {
		closeOnFailure()
		return fmt.Errorf("write temporary ambient marker: %w", err)
	}
	if err := temporary.Chmod(mode); err != nil {
		closeOnFailure()
		return fmt.Errorf("set temporary ambient marker mode: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		closeOnFailure()
		return fmt.Errorf("sync temporary ambient marker: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary ambient marker: %w", err)
	}
	if err := rename(temporaryPath, markerPath); err != nil {
		return fmt.Errorf("publish ambient marker: %w", err)
	}
	return nil
}

// ReadMarker returns the repository's opt-in record.
func ReadMarker(repositoryRoot string) (Marker, error) {
	raw, err := os.ReadFile(filepath.Join(repositoryRoot, filepath.FromSlash(MarkerPath)))
	if err != nil {
		return Marker{}, err
	}
	var marker Marker
	if err := json.Unmarshal(raw, &marker); err != nil {
		return Marker{}, fmt.Errorf("ambient marker is malformed: %w", err)
	}
	if marker.Schema != MarkerSchema {
		return Marker{}, fmt.Errorf("unsupported ambient marker schema %q", marker.Schema)
	}
	return marker, nil
}

// IsInitialized answers the only question a hook asks before doing anything.
//
// It deliberately reports a plain boolean: a malformed marker, an unreadable
// directory, and an absent marker are all "not ours", because a hook that
// tried to distinguish them would act on repositories it was never given.
func IsInitialized(repositoryRoot string) bool {
	_, err := ReadMarker(repositoryRoot)
	return err == nil
}

// Deinitialize removes the opt-in record, leaving other `.goalrail` content
// untouched.
func Deinitialize(repositoryRoot string) (bool, error) {
	markerPath := filepath.Join(repositoryRoot, filepath.FromSlash(MarkerPath))
	if err := os.Remove(markerPath); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
