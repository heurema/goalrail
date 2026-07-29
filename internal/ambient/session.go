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
	Schema string `json:"schema"`

	// QuestionID is the identifier an answering intent version cites. A
	// background session has no run ID by design, and the answering record
	// must still be writable, so the question carries an identity of its own
	// rather than borrowing a run's.
	QuestionID string `json:"question_id"`

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

// newQuestionID mints a canonical identifier for one occurrence of a question.
//
// Two sessions can write byte-identical questions: the payload deduplicates,
// but the occurrences do not, and each needs its own record. The identifier is
// therefore derived from the occurrence, not from the bytes.
func newQuestionID(digest, sessionRef string, at time.Time) string {
	seed := digest + "|" + sessionRef + "|" + at.UTC().Format(time.RFC3339Nano)
	sum := sha256.Sum256([]byte(seed))
	return "question-" + hex.EncodeToString(sum[:])[:16]
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
	if _, err := os.Lstat(questionPath); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	// The archive read gets the same descriptor-level hygiene as the stop
	// path. An unbounded read here could block session startup on a FIFO,
	// consume arbitrary memory, or follow a symlink and copy bytes from
	// outside the repository into Goalrail state — at exactly the moment a
	// stale artifact exists.
	raw, err := boundedio.ReadRegularFile(
		questionPath,
		"stale escalation artifact",
		domain.MaxEscalationBytes,
	)
	if err != nil {
		// Unreadable stale content still must not reach the new session. Clear
		// the path and record that something unretainable was there.
		if removeErr := os.Remove(questionPath); removeErr != nil {
			return false, removeErr
		}
		return true, store.WriteJSONOnce(
			filepath.ToSlash(filepath.Join(
				archiveDirectory,
				now().UTC().Format("20060102T150405.000000000Z")+"-unreadable.json",
			)),
			map[string]string{"path": ReservedEscalationPath, "reason": "STALE_ARTIFACT_UNREADABLE"},
			true,
		)
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
	recordedAt := now().UTC()
	raw, readErr := boundedio.ReadRegularFile(
		questionPath,
		"escalation artifact",
		domain.MaxEscalationBytes,
	)
	if readErr != nil {
		// An unreadable question is still an event: the session tried to
		// escalate and the attempt must leave evidence, or the owner sees
		// nothing at all.
		record := QuestionRecord{
			Schema:     questionRecordSchema,
			QuestionID: newQuestionID("", sessionRef, recordedAt),
			Repository: repositoryRoot,
			SessionRef: sessionRef,
			Path:       ReservedEscalationPath,
			RecordedAt: recordedAt,
			Invalid:    "ESCALATION_ARTIFACT_UNREADABLE",
		}
		if err := store.WriteJSONOnce(recordPath(record.QuestionID), record, true); err != nil {
			return nil, fmt.Errorf("record unreadable question: %w", err)
		}
		return &record, nil
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
		QuestionID:  newQuestionID(digest, sessionRef, recordedAt),
		Repository:  repositoryRoot,
		SessionRef:  sessionRef,
		Path:        ReservedEscalationPath,
		Digest:      digest,
		RetainedRef: retainedRef,
		RecordedAt:  recordedAt,
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

	// The record is keyed by occurrence, not by payload: two sessions may ask
	// the identical question, and the second must not collide with the first
	// and lose its attribution.
	if err := store.WriteJSONOnce(recordPath(record.QuestionID), record, true); err != nil {
		return nil, fmt.Errorf("record question: %w", err)
	}
	return &record, nil
}

func recordPath(questionID string) string {
	return filepath.ToSlash(filepath.Join(questionDirectory, "records", questionID+".json"))
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
