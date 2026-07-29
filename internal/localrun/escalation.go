package localrun

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/heurema/goalrail/internal/boundedio"
	"github.com/heurema/goalrail/internal/domain"
)

// ReservedEscalationPath is the single constant repository-relative path a run
// writes when the work item cannot be completed as specified. It is fixed by
// Goalrail and identical for every WorkSpec: expressing it in the canonical
// WorkSpec would fork goalrail.work-spec/v0, because DecodeWorkSpec rejects
// unknown fields and frozen WorkSpecs are digest-bound.
const ReservedEscalationPath = ".goalrail/blocked.md"

// retainedEscalationName is the run-store location of the retained bytes. The
// receipt references this copy rather than the mutable worktree file, so
// deleting the worktree file after provider observation cannot change the
// recorded outcome.
const retainedEscalationName = "escalation/payload.md"

var ErrEscalationArtifactPresent = errors.New("ESCALATION_ARTIFACT_PRESENT")

// EscalationRecord is the receipt-side evidence that a run produced an
// escalation. It never embeds the payload itself.
type EscalationRecord struct {
	Path        string                    `json:"path"`
	Digest      string                    `json:"digest"`
	RetainedRef string                    `json:"retained_ref"`
	Valid       bool                      `json:"valid"`
	Reason      domain.EvidenceReasonCode `json:"reason,omitempty"`
}

// observationHasEscalation reports whether a worktree observation contains the
// reserved path. Observation includes ignored files, so an ignored directory
// does not hide the artifact.
func observationHasEscalation(observation WorktreeObservation) bool {
	_, present := escalationEntry(observation)
	return present
}

func escalationEntry(observation WorktreeObservation) (WorktreeEntry, bool) {
	for _, entry := range observation.Entries {
		if entry.Path == ReservedEscalationPath {
			return entry, true
		}
	}
	return WorktreeEntry{}, false
}

// escalationArtifactPresent reports whether the reserved path exists in the
// worktree right now, independently of any observation.
//
// Preparation gates the frozen baseline, but a prepared run can be started
// later, and a prepared run is reused rather than re-observed. Without a
// pre-launch check, an artifact created between preparation and launch would be
// attributed to the provider that had not run yet.
func escalationArtifactPresent(repositoryRoot string) (bool, error) {
	artifactPath := filepath.Join(repositoryRoot, filepath.FromSlash(ReservedEscalationPath))
	_, err := os.Lstat(artifactPath)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, fmt.Errorf("inspect reserved escalation path: %w", err)
}

// captureEscalation reads the reserved path from the worktree that produced the
// supplied observation, applies retention hygiene, and retains the exact bytes
// append-only before any terminal receipt is written.
//
// A hygiene failure is recorded rather than raised: the question is not lost,
// the reason is inspectable, and the caller turns the invalid record into a
// failed run. A retention failure is raised, because a receipt must never
// reference bytes that were never retained.
func (service *Service) captureEscalation(
	repositoryRoot string,
	runID domain.RunID,
	observation WorktreeObservation,
) (*EscalationRecord, error) {
	observed, present := escalationEntry(observation)
	if !present {
		return nil, nil
	}
	artifactPath := filepath.Join(repositoryRoot, filepath.FromSlash(ReservedEscalationPath))
	raw, readErr := boundedio.ReadRegularFile(
		artifactPath,
		"escalation artifact",
		domain.MaxEscalationBytes,
	)
	if readErr != nil {
		return &EscalationRecord{
			Path:   ReservedEscalationPath,
			Valid:  false,
			Reason: "ESCALATION_ARTIFACT_UNREADABLE",
		}, nil
	}

	sum := sha256.Sum256(raw)
	digest := "sha256:" + hex.EncodeToString(sum[:])
	// The observation hashed the artifact; this read is a second read. If the
	// two disagree, the file changed between them and the single-snapshot
	// evidence chain does not hold, so the record is recorded as invalid rather
	// than silently binding a receipt to bytes the delta never saw.
	if observed.Digest != "" && observed.Digest != digest {
		return &EscalationRecord{
			Path:   ReservedEscalationPath,
			Valid:  false,
			Reason: "ESCALATION_ARTIFACT_CHANGED",
		}, nil
	}
	retainedRef := runPath(runID, retainedEscalationName)
	if err := service.store.WriteBytesOnce(retainedRef, raw, true); err != nil {
		return nil, fmt.Errorf("retain escalation artifact: %w", err)
	}

	record := &EscalationRecord{
		Path:        ReservedEscalationPath,
		Digest:      digest,
		RetainedRef: retainedRef,
		Valid:       true,
	}
	if err := domain.ValidateEscalationPayload(raw); err != nil {
		record.Valid = false
		record.Reason = "ESCALATION_ARTIFACT_INVALID"
	}
	return record, nil
}

// terminalObservation returns the observation Start took after the provider
// returned, so the delta and the escalation come from one snapshot. A run
// prepared before this behaviour existed has no persisted observation and falls
// back to observing at finish.
func (service *Service) terminalObservation(
	ctx context.Context,
	runID domain.RunID,
	repositoryRoot string,
) (WorktreeObservation, error) {
	var observation WorktreeObservation
	err := service.store.ReadJSON(runPath(runID, "terminal-observation.json"), &observation)
	if err == nil {
		return observation, nil
	}
	if !errors.Is(err, ErrStateNotFound) {
		return WorktreeObservation{}, err
	}
	return service.observer.Observe(ctx, repositoryRoot)
}

// retainedEscalation returns the record Start wrote, if any. The record is
// authoritative for the run: the worktree file may since have been deleted,
// truncated, or replaced.
func (service *Service) retainedEscalation(runID domain.RunID) (*EscalationRecord, error) {
	var record EscalationRecord
	err := service.store.ReadJSON(runPath(runID, "escalation.json"), &record)
	if err == nil {
		return &record, nil
	}
	if errors.Is(err, ErrStateNotFound) {
		return nil, nil
	}
	return nil, err
}

// applyEscalationRules resolves the terminal status once an escalation was
// retained. The rules exist to close specific gaming paths:
//
//   - a hedge — an agent that ships both a patch and a question, taking whichever
//     sticks — yields failed, not blocked;
//   - an explicit check failure keeps failed, so blocked cannot become a channel
//     for converting red checks into a softer outcome;
//   - no combination yields passed while an artifact is retained.
func applyEscalationRules(
	status RunState,
	reasons []domain.EvidenceReasonCode,
	escalation *EscalationRecord,
	delta WorktreeDelta,
	scope []string,
) (RunState, []domain.EvidenceReasonCode) {
	if escalation == nil {
		return status, reasons
	}
	reasons = appendReason(reasons, "ESCALATION_RECORDED")
	if !escalation.Valid {
		reason := escalation.Reason
		if reason == "" {
			reason = "ESCALATION_ARTIFACT_INVALID"
		}
		return StateFailed, appendReason(reasons, reason)
	}
	if delta.HeadChanged || len(delta.ScopeViolations) != 0 {
		return StateFailed, reasons
	}
	if hasInScopeEdits(delta, scope) {
		return StateFailed, appendReason(reasons, "BLOCKED_WITH_EDITS")
	}
	if status == StateFailed {
		return StateFailed, reasons
	}
	return StateBlocked, appendReason(reasons, "ESCALATION_PENDING")
}

// hasInScopeEdits reports whether the run edited anything the WorkSpec declared
// as its scope. The reserved path never counts, including when the declared
// scope is broad enough to contain it.
func hasInScopeEdits(delta WorktreeDelta, scope []string) bool {
	for _, path := range delta.ChangedPaths {
		if path == ReservedEscalationPath {
			continue
		}
		if pathInScope(path, scope) {
			return true
		}
	}
	return false
}
