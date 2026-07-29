package ambient

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/heurema/goalrail/internal/boundedio"
	"github.com/heurema/goalrail/internal/domain"
)

const (
	questionRecordSchema = "goalrail.ambient-question/v0"
	archiveDirectory     = "ambient/archive"
	questionDirectory    = "ambient/questions"
)

// QuestionRecord is what an attached session leaves behind. It is not a run
// outcome: an attached session mints no run ID, launch claim, terminal
// receipt, or status. It is a question, retained and attributable.
type QuestionRecord struct {
	Schema      string     `json:"schema"`
	Repository  string     `json:"repository"`
	SessionRef  string     `json:"session_ref,omitempty"`
	Path        string     `json:"path"`
	Digest      string     `json:"digest"`
	RetainedRef string     `json:"retained_ref"`
	RecordedAt  time.Time  `json:"recorded_at"`
	Intent      *IntentRef `json:"intent,omitempty"`
	UnboundWhy  string     `json:"unbound_reason,omitempty"`
	Invalid     string     `json:"invalid_reason,omitempty"`
}

// Store is the append-only state root outside the repository. It is satisfied
// by the local-run store, so attached sessions and wrapped runs retain
// evidence through one mechanism.
type Store interface {
	WriteBytesOnce(relative string, content []byte, allowIdentical bool) error
	WriteJSONOnce(relative string, value any, allowIdentical bool) error
}

// StartSession prepares an initialized repository for a session that is about
// to open, and returns the announcement the session should be told.
//
// A question left by an earlier session is archived first. Attributing it to
// the newly opening session would let a stale question be certified by work it
// never saw — the ambient counterpart of the wrapper's pre-launch gate.
// Archival preserves the evidence rather than judging or discarding it.
func StartSession(
	store Store,
	repositoryRoot string,
	now func() time.Time,
) (announcement string, archived bool, err error) {
	if !IsInitialized(repositoryRoot) {
		return "", false, nil
	}
	archived, err = archiveStaleQuestion(store, repositoryRoot, now)
	if err != nil {
		return "", false, err
	}
	return AmbientAnnouncement, archived, nil
}

func archiveStaleQuestion(
	store Store,
	repositoryRoot string,
	now func() time.Time,
) (bool, error) {
	questionPath := filepath.Join(repositoryRoot, filepath.FromSlash(ReservedEscalationPath))
	raw, err := os.ReadFile(questionPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	digest := digestOf(raw)
	stamp := now().UTC().Format("20060102T150405Z")
	relative := filepath.ToSlash(filepath.Join(
		archiveDirectory,
		fmt.Sprintf("%s-%s.md", stamp, shortDigest(digest)),
	))
	if err := store.WriteBytesOnce(relative, raw, true); err != nil {
		return false, err
	}
	if err := os.Remove(questionPath); err != nil {
		return false, err
	}
	return true, nil
}

// StopSession retains a question the session wrote and binds it to the
// repository's active confirmed intent.
//
// The wrapper's clean-scope rules deliberately do not apply here. An
// interactive session edits legitimately throughout its life and there is no
// frozen scope to violate, so the question is retained without judging the
// worktree and no status is minted.
func StopSession(
	store Store,
	repositoryRoot string,
	sessionRef string,
	resolver IntentResolver,
	now func() time.Time,
) (*QuestionRecord, error) {
	if !IsInitialized(repositoryRoot) {
		return nil, nil
	}
	questionPath := filepath.Join(repositoryRoot, filepath.FromSlash(ReservedEscalationPath))
	if _, err := os.Lstat(questionPath); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	raw, readErr := boundedio.ReadRegularFile(
		questionPath,
		"escalation artifact",
		domain.MaxEscalationBytes,
	)
	if readErr != nil {
		return &QuestionRecord{
			Schema:     questionRecordSchema,
			Repository: repositoryRoot,
			SessionRef: sessionRef,
			Path:       ReservedEscalationPath,
			RecordedAt: now().UTC(),
			Invalid:    "ESCALATION_ARTIFACT_UNREADABLE",
		}, nil
	}

	digest := digestOf(raw)
	retainedRef := filepath.ToSlash(filepath.Join(
		questionDirectory,
		shortDigest(digest),
		"question.md",
	))
	if err := store.WriteBytesOnce(retainedRef, raw, true); err != nil {
		return nil, fmt.Errorf("retain question: %w", err)
	}

	record := QuestionRecord{
		Schema:      questionRecordSchema,
		Repository:  repositoryRoot,
		SessionRef:  sessionRef,
		Path:        ReservedEscalationPath,
		Digest:      digest,
		RetainedRef: retainedRef,
		RecordedAt:  now().UTC(),
	}
	if err := domain.ValidateEscalationPayload(raw); err != nil {
		record.Invalid = "ESCALATION_ARTIFACT_INVALID"
	}

	// Exact-or-explicitly-unbound: a guessed binding would poison the chain
	// the record exists to serve, while an unbound question is still caught,
	// retained, and answerable.
	reference, reason := resolver.ActiveConfirmedIntent(repositoryRoot)
	if reference != nil {
		record.Intent = reference
	} else {
		record.UnboundWhy = reason
	}

	relative := filepath.ToSlash(filepath.Join(
		questionDirectory,
		shortDigest(digest),
		"record.json",
	))
	if err := store.WriteJSONOnce(relative, record, true); err != nil {
		return nil, fmt.Errorf("record question: %w", err)
	}
	return &record, nil
}

func digestOf(content []byte) string {
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func shortDigest(digest string) string {
	trimmed := digest
	if len(trimmed) > len("sha256:")+16 {
		trimmed = trimmed[len("sha256:") : len("sha256:")+16]
	}
	return trimmed
}

// MarshalRecord is a convenience for callers that surface a record to the
// owner without importing encoding/json.
func MarshalRecord(record *QuestionRecord) ([]byte, error) {
	return json.MarshalIndent(record, "", "  ")
}
