// Package evidence persists the small repository-local evidence record.
// It is an adapter around provider-neutral domain events, not a workflow or
// database boundary.
package evidence

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/heurema/goalrail/internal/domain"
)

const (
	recordSchemaVersion = uint32(1)
	maxRecordBytes      = 64 * 1024
)

var (
	ErrIntegrity                   = errors.New("evidence record integrity violation")
	ErrSensitivePayload            = errors.New("evidence event contains disallowed payload")
	ErrDuplicateEventID            = errors.New("duplicate evidence event ID")
	ErrInvalidCorrection           = errors.New("invalid evidence correction")
	ErrInvalidEvent                = errors.New("invalid evidence event")
	ErrInterprocessLockUnsupported = errors.New("evidence interprocess locking is unsupported")

	digestPattern  = regexp.MustCompile(`^[0-9a-f]{64}$`)
	traceIDPattern = regexp.MustCompile(`^[0-9a-f]{32}$`)

	pathLocks sync.Map
)

var approvedSourceReferenceSchemes = map[string]struct{}{
	"codex-app":      {},
	"codex-hook":     {},
	"git":            {},
	"github":         {},
	"hook":           {},
	"launch-receipt": {},
	"openspec":       {},
	"owner-review":   {},
	"request":        {},
	"review":         {},
}

var approvedCheckReferenceSchemes = map[string]struct{}{
	"check": {},
	"ci":    {},
	"test":  {},
}

var sensitiveFragments = []string{
	"-----begin private key",
	"authorization:",
	"api_key=",
	"apikey=",
	"bearer ",
	"github_pat_",
	"ghp_",
	"password=",
	"passwd=",
	"secret=",
	"sk-",
	"token=",
	"xoxb-",
	"xoxp-",
}

// Store owns one append-only JSONL file. Store instances for the same cleaned
// path share an in-process lock. A sibling coordination file carries an OS
// lock so cooperating processes serialize complete transactions.
type Store struct {
	path string
	mu   *sync.Mutex
}

type diskRecord struct {
	SchemaVersion  uint32               `json:"schema_version"`
	Sequence       uint64               `json:"sequence"`
	PreviousDigest string               `json:"previous_digest"`
	Event          domain.EvidenceEvent `json:"event"`
	Digest         string               `json:"digest"`
}

type digestBody struct {
	SchemaVersion  uint32               `json:"schema_version"`
	Sequence       uint64               `json:"sequence"`
	PreviousDigest string               `json:"previous_digest"`
	Event          domain.EvidenceEvent `json:"event"`
}

// NewStore returns a repository-local evidence store for path.
func NewStore(path string) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("%w: path is empty", ErrInvalidEvent)
	}
	absPath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("resolve evidence path: %w", err)
	}
	lock, _ := pathLocks.LoadOrStore(absPath, &sync.Mutex{})
	return &Store{path: absPath, mu: lock.(*sync.Mutex)}, nil
}

// Append verifies the complete existing chain and appends exactly one event.
// Corrections must point to an existing event; no mutation API is provided.
func (s *Store) Append(event domain.EvidenceEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(s.path), 0o750); err != nil {
		return fmt.Errorf("create evidence directory: %w", err)
	}
	lockFile, err := s.openProcessLock(true)
	if err != nil {
		return err
	}
	defer closeLockedFile(lockFile)
	if _, err := s.inspectEvidencePath(); err != nil {
		return err
	}

	records, err := s.readRecords()
	if err != nil {
		return err
	}
	if err := validateEvent(event, records); err != nil {
		return err
	}

	previousDigest := ""
	if len(records) > 0 {
		previousDigest = records[len(records)-1].Digest
	}
	record := diskRecord{
		SchemaVersion:  recordSchemaVersion,
		Sequence:       uint64(len(records) + 1),
		PreviousDigest: previousDigest,
		Event:          event,
	}
	record.Digest, err = calculateDigest(record)
	if err != nil {
		return fmt.Errorf("encode evidence digest: %w", err)
	}
	line, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("encode evidence record: %w", err)
	}
	if len(line)+1 > maxRecordBytes {
		return fmt.Errorf("%w: encoded event exceeds %d bytes", ErrSensitivePayload, maxRecordBytes)
	}
	line = append(line, '\n')

	file, err := s.openEvidenceForAppend()
	if err != nil {
		return err
	}
	defer file.Close()
	written, err := file.Write(line)
	if err != nil {
		return fmt.Errorf("append evidence record: %w", err)
	}
	if written != len(line) {
		return fmt.Errorf("append evidence record: %w", io.ErrShortWrite)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync evidence record: %w", err)
	}
	return nil
}

// ReadAll verifies the chain and returns events in append order.
func (s *Store) ReadAll() ([]domain.EvidenceEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(s.path), 0o750); err != nil {
		return nil, fmt.Errorf("create evidence directory: %w", err)
	}
	lockFile, err := s.openProcessLock(false)
	if err != nil {
		return nil, err
	}
	defer closeLockedFile(lockFile)
	exists, err := s.inspectEvidencePath()
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}

	records, err := s.readRecords()
	if err != nil {
		return nil, err
	}
	events := make([]domain.EvidenceEvent, 0, len(records))
	for _, record := range records {
		events = append(events, record.Event)
	}
	return events, nil
}

// Verify checks structure, event semantics, sequence, and the complete digest
// chain without returning event contents.
func (s *Store) Verify() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(s.path), 0o750); err != nil {
		return fmt.Errorf("create evidence directory: %w", err)
	}
	lockFile, err := s.openProcessLock(false)
	if err != nil {
		return err
	}
	defer closeLockedFile(lockFile)
	exists, err := s.inspectEvidencePath()
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	_, err = s.readRecords()
	return err
}

func (s *Store) inspectEvidencePath() (bool, error) {
	info, err := os.Lstat(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("inspect evidence path: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return false, fmt.Errorf("%w: evidence path is not a regular file", ErrIntegrity)
	}
	return true, nil
}

func (s *Store) openProcessLock(exclusive bool) (*os.File, error) {
	lockPath := s.path + ".lock"
	if info, err := os.Lstat(lockPath); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return nil, fmt.Errorf("%w: evidence lock path is not a regular file", ErrIntegrity)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("inspect evidence lock path: %w", err)
	}

	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open evidence lock: %w", err)
	}
	if info, statErr := file.Stat(); statErr != nil {
		file.Close()
		return nil, fmt.Errorf("inspect open evidence lock: %w", statErr)
	} else if !info.Mode().IsRegular() {
		file.Close()
		return nil, fmt.Errorf("%w: open evidence lock path is not a regular file", ErrIntegrity)
	}
	if err := lockEvidenceFile(file, exclusive); err != nil {
		file.Close()
		return nil, err
	}
	return file, nil
}

func (s *Store) openEvidenceForAppend() (*os.File, error) {
	if _, err := s.inspectEvidencePath(); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(s.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open evidence record: %w", err)
	}
	if info, statErr := file.Stat(); statErr != nil {
		file.Close()
		return nil, fmt.Errorf("inspect open evidence record: %w", statErr)
	} else if !info.Mode().IsRegular() {
		file.Close()
		return nil, fmt.Errorf("%w: open evidence path is not a regular file", ErrIntegrity)
	}
	return file, nil
}

func closeLockedFile(file *os.File) {
	_ = unlockEvidenceFile(file)
	_ = file.Close()
}

func (s *Store) readRecords() ([]diskRecord, error) {
	file, err := os.Open(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open evidence record: %w", err)
	}
	defer file.Close()
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek evidence record: %w", err)
	}
	if info, statErr := file.Stat(); statErr != nil {
		return nil, fmt.Errorf("inspect evidence record: %w", statErr)
	} else if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%w: evidence path is not a regular file", ErrIntegrity)
	} else if info.Size() > 0 {
		lastByte := []byte{0}
		if _, readErr := file.ReadAt(lastByte, info.Size()-1); readErr != nil {
			return nil, fmt.Errorf("%w: inspect final record delimiter: %v", ErrIntegrity, readErr)
		}
		if lastByte[0] != '\n' {
			return nil, fmt.Errorf("%w: final record is not newline-terminated", ErrIntegrity)
		}
	}

	var records []diskRecord
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 4096), maxRecordBytes)
	for scanner.Scan() {
		lineNumber := len(records) + 1
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			return nil, fmt.Errorf("%w: empty line %d", ErrIntegrity, lineNumber)
		}
		var record diskRecord
		decoder := json.NewDecoder(bytes.NewReader(line))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&record); err != nil {
			return nil, fmt.Errorf("%w: decode line %d: %v", ErrIntegrity, lineNumber, err)
		}
		if err := requireJSONEOF(decoder); err != nil {
			return nil, fmt.Errorf("%w: line %d: %v", ErrIntegrity, lineNumber, err)
		}
		canonicalLine, err := json.Marshal(record)
		if err != nil {
			return nil, fmt.Errorf("%w: canonicalize line %d: %v", ErrIntegrity, lineNumber, err)
		}
		if !bytes.Equal(line, canonicalLine) {
			return nil, fmt.Errorf("%w: line %d is not canonical JSON", ErrIntegrity, lineNumber)
		}
		if err := validateRecord(record, records); err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNumber, err)
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("%w: read evidence record: %v", ErrIntegrity, err)
	}
	return records, nil
}

func requireJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func validateRecord(record diskRecord, prior []diskRecord) error {
	expectedSequence := uint64(len(prior) + 1)
	if record.SchemaVersion != recordSchemaVersion {
		return fmt.Errorf("%w: unsupported schema version %d", ErrIntegrity, record.SchemaVersion)
	}
	if record.Sequence != expectedSequence {
		return fmt.Errorf("%w: sequence is %d, want %d", ErrIntegrity, record.Sequence, expectedSequence)
	}
	expectedPrevious := ""
	if len(prior) > 0 {
		expectedPrevious = prior[len(prior)-1].Digest
	}
	if record.PreviousDigest != expectedPrevious {
		return fmt.Errorf("%w: previous digest mismatch", ErrIntegrity)
	}
	if !digestPattern.MatchString(record.Digest) {
		return fmt.Errorf("%w: malformed digest", ErrIntegrity)
	}
	expectedDigest, err := calculateDigest(record)
	if err != nil {
		return fmt.Errorf("%w: calculate digest: %v", ErrIntegrity, err)
	}
	if record.Digest != expectedDigest {
		return fmt.Errorf("%w: digest mismatch", ErrIntegrity)
	}
	if err := validateEvent(record.Event, prior); err != nil {
		return fmt.Errorf("%w: stored event invalid: %v", ErrIntegrity, err)
	}
	return nil
}

func calculateDigest(record diskRecord) (string, error) {
	payload, err := json.Marshal(digestBody{
		SchemaVersion:  record.SchemaVersion,
		Sequence:       record.Sequence,
		PreviousDigest: record.PreviousDigest,
		Event:          record.Event,
	})
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:]), nil
}

func validateEvent(event domain.EvidenceEvent, prior []diskRecord) error {
	for _, record := range prior {
		if record.Event.ID == event.ID {
			return fmt.Errorf("%w: %s", ErrDuplicateEventID, event.ID)
		}
	}
	if err := validateCanonicalID("event_id", string(event.ID)); err != nil {
		return err
	}
	if err := validateCanonicalID("canary_id", string(event.CanaryID)); err != nil {
		return err
	}
	if event.Kind == domain.EventCanaryStopped {
		if event.ChangeID != "" {
			return fmt.Errorf("%w: canary-stop event must not name a change", ErrInvalidEvent)
		}
	} else if err := validateCanonicalID("change_id", string(event.ChangeID)); err != nil {
		return err
	}
	if err := validateMetadataToken("actor", string(event.Actor), true); err != nil {
		return err
	}
	if err := validateSourceReference("source_ref", string(event.SourceRef)); err != nil {
		return err
	}
	observationRefs := make(map[domain.EvidenceReference]struct{}, len(event.ObservationRefs))
	for index, reference := range event.ObservationRefs {
		field := fmt.Sprintf("observation_refs[%d]", index)
		if err := validateObservationReference(field, string(reference)); err != nil {
			return err
		}
		if _, duplicate := observationRefs[reference]; duplicate {
			return fmt.Errorf("%w: duplicate %s", ErrInvalidEvent, field)
		}
		observationRefs[reference] = struct{}{}
	}
	if err := validateTimestamp("occurred_at", event.OccurredAt); err != nil {
		return err
	}
	if event.ReasonCode != "" {
		if err := validateMetadataToken("reason_code", string(event.ReasonCode), true); err != nil {
			return err
		}
	}
	if event.SupersedesEventID != "" {
		if err := rejectSensitive("supersedes_event_id", string(event.SupersedesEventID)); err != nil {
			return err
		}
		if !domain.IsCanonicalID(string(event.SupersedesEventID)) {
			return fmt.Errorf("%w: superseded event ID must be canonical", ErrInvalidCorrection)
		}
	}
	if err := validateEventKind(event.Kind); err != nil {
		return err
	}
	if event.Lineage != nil {
		if err := validateLineage(event, *event.Lineage); err != nil {
			return err
		}
	}
	if event.Assessment != nil {
		if err := validateAssessment(event, *event.Assessment); err != nil {
			return err
		}
	}
	if event.Assignment != nil {
		if err := validateAssignment(*event.Assignment); err != nil {
			return err
		}
	}
	if event.Terminal != nil {
		if err := validateTerminal(*event.Terminal); err != nil {
			return err
		}
	}
	if event.LineageResolutionAttempts > 1 ||
		(event.Lineage == nil && event.LineageResolutionAttempts != 0) {
		return fmt.Errorf("%w: lineage resolution attempts must be 0 or 1 and require lineage", ErrInvalidEvent)
	}

	switch event.Kind {
	case domain.EventChangeStarted:
		if event.Assignment == nil || payloadCount(event) != 1 {
			return fmt.Errorf("%w: change-start event requires only assignment payload", ErrInvalidEvent)
		}
		if err := validateAssignmentTransition(event, *event.Assignment, prior); err != nil {
			return err
		}
	case domain.EventLineageRecorded:
		if event.Lineage == nil || payloadCount(event) != 1 {
			return fmt.Errorf("%w: lineage event requires only lineage payload", ErrInvalidEvent)
		}
		if err := validateLineageTransition(event, *event.Lineage, prior); err != nil {
			return err
		}
	case domain.EventTerminalStateChanged:
		if event.Terminal == nil || payloadCount(event) != 1 {
			return fmt.Errorf("%w: terminal event requires only terminal payload", ErrInvalidEvent)
		}
		if err := validateTerminalTransition(event, *event.Terminal, prior); err != nil {
			return err
		}
	case domain.EventAssessmentRecorded:
		if event.Assessment == nil || payloadCount(event) != 1 {
			return fmt.Errorf("%w: assessment event requires only assessment payload", ErrInvalidEvent)
		}
		if err := validateAssessmentTransition(event, *event.Assessment, prior); err != nil {
			return err
		}
	case domain.EventMaterialCorrection:
		if payloadCount(event) != 0 || event.ReasonCode == "" {
			return fmt.Errorf("%w: material correction requires only a reason code", ErrInvalidEvent)
		}
		if err := validateMaterialCorrectionTransition(event, prior); err != nil {
			return err
		}
	case domain.EventCanaryStopped:
		if payloadCount(event) != 0 || event.ReasonCode == "" || len(event.ObservationRefs) != 0 {
			return fmt.Errorf("%w: canary-stop event requires only a reason code", ErrInvalidEvent)
		}
		if err := validateCanaryStopTransition(event, prior); err != nil {
			return err
		}
	}

	if event.Kind == domain.EventEvidenceCorrected {
		return validateCorrection(event, prior)
	}
	if event.SupersedesEventID != "" {
		return fmt.Errorf("%w: only correction events may supersede prior events", ErrInvalidCorrection)
	}
	return nil
}

func validateEventKind(kind domain.EvidenceEventKind) error {
	switch kind {
	case domain.EventChangeStarted,
		domain.EventLineageRecorded,
		domain.EventTerminalStateChanged,
		domain.EventAssessmentRecorded,
		domain.EventMaterialCorrection,
		domain.EventEvidenceCorrected,
		domain.EventCanaryStopped:
		return nil
	default:
		return fmt.Errorf("%w: unknown event kind %q", ErrInvalidEvent, kind)
	}
}

func payloadCount(event domain.EvidenceEvent) int {
	count := 0
	for _, present := range []bool{
		event.Assignment != nil,
		event.Lineage != nil,
		event.Terminal != nil,
		event.Assessment != nil,
	} {
		if present {
			count++
		}
	}
	return count
}

func validateAssignment(assignment domain.CanaryAssignment) error {
	if assignment.Ordinal == 0 || assignment.Ordinal > domain.CanaryAssignmentCount {
		return fmt.Errorf("%w: assignment ordinal must be within 1..15", ErrInvalidEvent)
	}
	expectedVariant, err := domain.CanaryVariantForOrdinal(assignment.Ordinal)
	if err != nil || assignment.Variant != expectedVariant {
		return fmt.Errorf("%w: assignment variant must match immutable ordinal", ErrInvalidEvent)
	}
	if assignment.ManifestVersion != domain.IntentCanaryV0ManifestVersion || assignment.IntentVersion == 0 {
		return fmt.Errorf("%w: assignment requires frozen manifest v1 and a non-zero intent version", ErrInvalidEvent)
	}
	return validateCanonicalID("assignment.run_id", string(assignment.RunID))
}

func validateAssignmentTransition(
	event domain.EvidenceEvent,
	assignment domain.CanaryAssignment,
	prior []diskRecord,
) error {
	expectedOrdinal := uint32(1)
	for _, record := range prior {
		previous := record.Event
		if previous.CanaryID == event.CanaryID && previous.Kind == domain.EventCanaryStopped {
			return fmt.Errorf("%w: canary assignments are stopped", ErrInvalidEvent)
		}
		if previous.CanaryID != event.CanaryID || previous.Assignment == nil {
			continue
		}
		if previous.ChangeID == event.ChangeID {
			return fmt.Errorf("%w: canary change is already assigned", ErrInvalidEvent)
		}
		if previous.Assignment.RunID == assignment.RunID {
			return fmt.Errorf("%w: run ID is already assigned", ErrInvalidEvent)
		}
		expectedOrdinal++
	}
	if assignment.Ordinal != expectedOrdinal {
		return fmt.Errorf(
			"%w: assignment ordinal %d must be the next ordinal %d",
			ErrInvalidEvent,
			assignment.Ordinal,
			expectedOrdinal,
		)
	}
	return nil
}

func validateCanaryStopTransition(event domain.EvidenceEvent, prior []diskRecord) error {
	for _, record := range prior {
		if record.Event.CanaryID != event.CanaryID {
			continue
		}
		if record.Event.Kind == domain.EventCanaryStopped {
			return fmt.Errorf("%w: canary assignments are already stopped", ErrInvalidEvent)
		}
		if event.OccurredAt.Before(record.Event.OccurredAt) {
			return fmt.Errorf("%w: canary-stop event precedes existing evidence", ErrInvalidEvent)
		}
	}
	return nil
}

func validateTerminal(terminal domain.CanaryTerminal) error {
	checkRefs := make(map[domain.EvidenceReference]struct{}, len(terminal.CheckRefs))
	for index, reference := range terminal.CheckRefs {
		field := fmt.Sprintf("terminal.check_refs[%d]", index)
		if err := validateCheckReference(field, string(reference)); err != nil {
			return err
		}
		if _, duplicate := checkRefs[reference]; duplicate {
			return fmt.Errorf("%w: duplicate %s", ErrInvalidEvent, field)
		}
		checkRefs[reference] = struct{}{}
	}
	if terminal.FlowOverheadMinutes != nil {
		value := *terminal.FlowOverheadMinutes
		if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
			return fmt.Errorf("%w: flow overhead must be finite and non-negative", ErrInvalidEvent)
		}
	}
	switch terminal.State {
	case domain.CanaryStateDelivered:
		if len(terminal.CheckRefs) == 0 || terminal.AbandonmentReason != "" || terminal.ProcessCausedAbandonment {
			return fmt.Errorf("%w: delivery requires checks and cannot claim abandonment", ErrInvalidEvent)
		}
	case domain.CanaryStateAbandoned:
		if err := validateMetadataToken("terminal.abandonment_reason", string(terminal.AbandonmentReason), true); err != nil {
			return err
		}
		if terminal.ChecksGreen {
			return fmt.Errorf("%w: abandoned change cannot be green", ErrInvalidEvent)
		}
	default:
		return fmt.Errorf("%w: terminal state must be delivered or abandoned", ErrInvalidEvent)
	}
	return nil
}

func validateLineage(event domain.EvidenceEvent, lineage domain.ExecutionLineage) error {
	if lineage.ChangeID != event.ChangeID {
		return fmt.Errorf("%w: lineage must use the event change ID", ErrInvalidEvent)
	}
	if err := validateCanonicalID("run_id", string(lineage.RunID)); err != nil {
		return err
	}
	if !digestPattern.MatchString(lineage.ContextDigest) {
		return fmt.Errorf("%w: lineage context digest must be lowercase SHA-256", ErrInvalidEvent)
	}
	if err := rejectSensitive("context_digest", lineage.ContextDigest); err != nil {
		return err
	}
	switch lineage.Status {
	case domain.LineageVerified:
		if err := validateCanonicalID("root_session_id", string(lineage.RootSessionID)); err != nil {
			return err
		}
		if lineage.IdentitySource != domain.SessionIdentityLifecycleHook && lineage.IdentitySource != domain.SessionIdentityLaunchReceipt {
			return fmt.Errorf("%w: verified lineage requires a provider-authoritative identity source", ErrInvalidEvent)
		}
		if lineage.UnlinkedReasonCode != "" {
			return fmt.Errorf("%w: verified lineage cannot have an unlinked reason", ErrInvalidEvent)
		}
	case domain.LineageUnlinked:
		if lineage.RootSessionID != "" || lineage.IdentitySource != "" {
			return fmt.Errorf("%w: unlinked lineage cannot claim a root session identity", ErrInvalidEvent)
		}
		if err := validateMetadataToken("unlinked_reason_code", string(lineage.UnlinkedReasonCode), true); err != nil {
			return err
		}
	default:
		return fmt.Errorf("%w: unknown lineage status %q", ErrInvalidEvent, lineage.Status)
	}
	return nil
}

func validateLineageTransition(
	event domain.EvidenceEvent,
	lineage domain.ExecutionLineage,
	prior []diskRecord,
) error {
	assignment, _ := assignedChange(event, prior)
	if assignment == nil {
		return nil
	}
	if lineage.RunID != assignment.RunID {
		return fmt.Errorf("%w: lineage run ID must match the immutable assignment", ErrInvalidEvent)
	}
	for _, record := range prior {
		previous := record.Event
		if previous.CanaryID != event.CanaryID || previous.ChangeID != event.ChangeID || previous.Lineage == nil {
			continue
		}
		if previous.Lineage.Status == domain.LineageVerified && lineage.Status == domain.LineageVerified &&
			(previous.Lineage.RootSessionID != lineage.RootSessionID ||
				previous.Lineage.IdentitySource != lineage.IdentitySource) {
			return fmt.Errorf("%w: verified root session identity cannot change", ErrInvalidEvent)
		}
	}
	return nil
}

func validateAssessment(event domain.EvidenceEvent, assessment domain.Assessment) error {
	switch assessment.Outcome {
	case domain.IntentMatch, domain.IntentPartial, domain.IntentMiss:
	default:
		return fmt.Errorf("%w: unknown assessment outcome %q", ErrInvalidEvent, assessment.Outcome)
	}
	if err := validateMetadataToken("assessed_by", string(assessment.AssessedBy), true); err != nil {
		return err
	}
	if err := validateTimestamp("assessed_at", assessment.AssessedAt); err != nil {
		return err
	}
	if assessment.AssessedAt.After(event.OccurredAt) {
		return fmt.Errorf("%w: assessment timestamp cannot be after its evidence event", ErrInvalidEvent)
	}
	if assessment.AssessedBy != event.Actor {
		return fmt.Errorf("%w: assessment actor must be the assessed owner", ErrInvalidEvent)
	}
	return nil
}

func validateTerminalTransition(
	event domain.EvidenceEvent,
	terminal domain.CanaryTerminal,
	prior []diskRecord,
) error {
	assignment, assignedAt := assignedChange(event, prior)
	if assignment == nil {
		return nil
	}
	if event.OccurredAt.Before(assignedAt) {
		return fmt.Errorf("%w: terminal event predates assignment", ErrInvalidEvent)
	}
	for _, record := range prior {
		previous := record.Event
		if previous.CanaryID == event.CanaryID && previous.ChangeID == event.ChangeID && previous.Terminal != nil {
			return fmt.Errorf("%w: canary change already has a terminal event", ErrInvalidEvent)
		}
	}
	if assignment.Variant == domain.VariantBaseline &&
		(terminal.FlowOverheadMinutes != nil || terminal.ProcessCausedAbandonment) {
		return fmt.Errorf("%w: baseline change cannot record flow-only terminal data", ErrInvalidEvent)
	}
	return nil
}

func validateAssessmentTransition(
	event domain.EvidenceEvent,
	assessment domain.Assessment,
	prior []diskRecord,
) error {
	assignment, _ := assignedChange(event, prior)
	if assignment == nil {
		return nil
	}
	terminal, terminalAt := terminalChange(event, prior)
	if terminal == nil || terminal.State != domain.CanaryStateDelivered {
		return fmt.Errorf("%w: assessment requires a delivered canary change", ErrInvalidEvent)
	}
	if event.OccurredAt.Before(terminalAt) || assessment.AssessedAt.Before(terminalAt) {
		return fmt.Errorf("%w: assessment predates delivery", ErrInvalidEvent)
	}
	for _, record := range prior {
		previous := record.Event
		if previous.CanaryID == event.CanaryID && previous.ChangeID == event.ChangeID &&
			previous.Kind == domain.EventAssessmentRecorded {
			return fmt.Errorf("%w: owner assessment already exists; append a correction", ErrInvalidEvent)
		}
	}
	if assessment.MaterialCorrectionBeforeDelivery != hasMaterialCorrection(event, prior) {
		return fmt.Errorf("%w: material-correction assessment must match append-only events", ErrInvalidEvent)
	}
	return validateAssessmentAgainstChange(assessment, *assignment, *terminal)
}

func validateMaterialCorrectionTransition(event domain.EvidenceEvent, prior []diskRecord) error {
	assignment, assignedAt := assignedChange(event, prior)
	if assignment == nil {
		return nil
	}
	if event.OccurredAt.Before(assignedAt) {
		return fmt.Errorf("%w: material correction predates assignment", ErrInvalidEvent)
	}
	terminal, _ := terminalChange(event, prior)
	if terminal != nil {
		return fmt.Errorf("%w: material correction must be recorded before terminal state", ErrInvalidEvent)
	}
	return nil
}

func validateAssessmentAgainstChange(
	assessment domain.Assessment,
	assignment domain.CanaryAssignment,
	terminal domain.CanaryTerminal,
) error {
	if assessment.ChecksGreen != terminal.ChecksGreen {
		return fmt.Errorf("%w: assessment green value must match recorded checks", ErrInvalidEvent)
	}
	if assignment.Variant == domain.VariantFlow && assessment.RepeatOptIn == nil {
		return fmt.Errorf("%w: flow assessment requires explicit repeat opt-in", ErrInvalidEvent)
	}
	if assignment.Variant == domain.VariantBaseline && assessment.RepeatOptIn != nil {
		return fmt.Errorf("%w: baseline assessment cannot record flow repeat opt-in", ErrInvalidEvent)
	}
	return nil
}

func assignedChange(event domain.EvidenceEvent, prior []diskRecord) (*domain.CanaryAssignment, time.Time) {
	for _, record := range prior {
		previous := record.Event
		if previous.CanaryID == event.CanaryID && previous.ChangeID == event.ChangeID && previous.Assignment != nil {
			return previous.Assignment, previous.OccurredAt
		}
	}
	return nil, time.Time{}
}

func terminalChange(event domain.EvidenceEvent, prior []diskRecord) (*domain.CanaryTerminal, time.Time) {
	for _, record := range prior {
		previous := record.Event
		if previous.CanaryID == event.CanaryID && previous.ChangeID == event.ChangeID && previous.Terminal != nil {
			return previous.Terminal, previous.OccurredAt
		}
	}
	return nil, time.Time{}
}

func hasMaterialCorrection(event domain.EvidenceEvent, prior []diskRecord) bool {
	for _, record := range prior {
		previous := record.Event
		if previous.CanaryID == event.CanaryID && previous.ChangeID == event.ChangeID &&
			previous.Kind == domain.EventMaterialCorrection {
			return true
		}
	}
	return false
}

func validateCorrection(event domain.EvidenceEvent, prior []diskRecord) error {
	if event.SupersedesEventID == "" || event.ReasonCode == "" {
		return fmt.Errorf("%w: correction requires superseded event ID and reason code", ErrInvalidCorrection)
	}
	var target *domain.EvidenceEvent
	for i := range prior {
		if prior[i].Event.ID == event.SupersedesEventID {
			target = &prior[i].Event
			break
		}
	}
	if target == nil {
		return fmt.Errorf("%w: superseded event does not exist", ErrInvalidCorrection)
	}
	if target.CanaryID != event.CanaryID || target.ChangeID != event.ChangeID {
		return fmt.Errorf("%w: correction must remain in the same canary change", ErrInvalidCorrection)
	}
	if event.OccurredAt.Before(target.OccurredAt) {
		return fmt.Errorf("%w: correction predates superseded event", ErrInvalidCorrection)
	}
	if target.Assessment == nil || event.Assessment == nil || payloadCount(event) != 1 {
		return fmt.Errorf("%w: v0 corrections may replace only assessment payloads", ErrInvalidCorrection)
	}
	latestAssessmentID := domain.EvidenceEventID("")
	for _, record := range prior {
		previous := record.Event
		if previous.CanaryID != event.CanaryID || previous.ChangeID != event.ChangeID || previous.Assessment == nil {
			continue
		}
		if previous.Kind == domain.EventAssessmentRecorded ||
			(previous.Kind == domain.EventEvidenceCorrected && previous.SupersedesEventID == latestAssessmentID) {
			latestAssessmentID = previous.ID
		}
	}
	if event.SupersedesEventID != latestAssessmentID {
		return fmt.Errorf("%w: correction must supersede the latest owner assessment", ErrInvalidCorrection)
	}
	assignment, _ := assignedChange(event, prior)
	terminal, terminalAt := terminalChange(event, prior)
	if assignment != nil {
		if terminal == nil || event.OccurredAt.Before(terminalAt) || event.Assessment.AssessedAt.Before(terminalAt) {
			return fmt.Errorf("%w: assessment correction requires prior delivery", ErrInvalidCorrection)
		}
		if err := validateAssessmentAgainstChange(*event.Assessment, *assignment, *terminal); err != nil {
			return err
		}
		if event.Assessment.MaterialCorrectionBeforeDelivery != hasMaterialCorrection(event, prior) {
			return fmt.Errorf("%w: corrected material-correction value must match append-only events", ErrInvalidCorrection)
		}
	}
	return nil
}

func validateMetadataToken(field, value string, required bool) error {
	if value == "" && !required {
		return nil
	}
	if err := rejectSensitive(field, value); err != nil {
		return err
	}
	if !domain.IsCanonicalID(value) {
		return fmt.Errorf("%w: %s must be a canonical metadata token", ErrSensitivePayload, field)
	}
	return nil
}

func validateCanonicalID(field, value string) error {
	if err := rejectSensitive(field, value); err != nil {
		return err
	}
	if !domain.IsCanonicalID(value) {
		return fmt.Errorf("%w: %s must be a canonical non-empty ID", ErrInvalidEvent, field)
	}
	return nil
}

func validateReference(field, value string) error {
	if err := rejectSensitive(field, value); err != nil {
		return err
	}
	if !domain.IsEvidenceReference(value) {
		return fmt.Errorf("%w: %s must be a bounded scheme:identifier reference", ErrSensitivePayload, field)
	}
	return nil
}

func validateSourceReference(field, value string) error {
	return validateApprovedMetadataReference(field, value, approvedSourceReferenceSchemes)
}

func validateCheckReference(field, value string) error {
	return validateApprovedMetadataReference(field, value, approvedCheckReferenceSchemes)
}

func validateApprovedMetadataReference(
	field string,
	value string,
	approvedSchemes map[string]struct{},
) error {
	if err := validateReference(field, value); err != nil {
		return err
	}
	scheme, identifier, found := strings.Cut(value, ":")
	if !found {
		return fmt.Errorf("%w: %s is not a reference", ErrSensitivePayload, field)
	}
	if _, approved := approvedSchemes[scheme]; !approved {
		return fmt.Errorf("%w: %s uses unapproved reference scheme %q", ErrSensitivePayload, field, scheme)
	}
	if !domain.IsCanonicalID(identifier) {
		return fmt.Errorf("%w: %s identifier must be bounded metadata, not a path or content", ErrSensitivePayload, field)
	}
	return nil
}

func validateObservationReference(field, value string) error {
	if err := validateReference(field, value); err != nil {
		return err
	}
	scheme, identifier, found := strings.Cut(value, ":")
	if !found {
		return fmt.Errorf("%w: %s is not a reference", ErrSensitivePayload, field)
	}
	switch scheme {
	case "langfuse-session":
		if !domain.IsCanonicalID(identifier) {
			return fmt.Errorf("%w: %s session identity must be canonical metadata", ErrSensitivePayload, field)
		}
	case "langfuse-trace":
		if !traceIDPattern.MatchString(identifier) {
			return fmt.Errorf("%w: %s trace identity must be 32 lowercase hexadecimal characters", ErrSensitivePayload, field)
		}
	default:
		return fmt.Errorf("%w: %s uses unapproved observation scheme %q", ErrSensitivePayload, field, scheme)
	}
	return nil
}

func validateTimestamp(field string, value time.Time) error {
	if value.IsZero() || value.Location() != time.UTC {
		return fmt.Errorf("%w: %s must be a non-zero UTC timestamp", ErrInvalidEvent, field)
	}
	return nil
}

func rejectSensitive(field, value string) error {
	if strings.ContainsAny(value, "\r\n\t ") {
		return fmt.Errorf("%w: %s contains whitespace or raw content", ErrSensitivePayload, field)
	}
	lower := strings.ToLower(value)
	for _, fragment := range sensitiveFragments {
		if strings.Contains(lower, fragment) {
			return fmt.Errorf("%w: %s matches a credential or secret pattern", ErrSensitivePayload, field)
		}
	}
	return nil
}
