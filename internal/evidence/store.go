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
	ErrRealCanaryNotActivated      = errors.New("real-change canary is not activated")
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
	"langfuse-api":   {},
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
	path            string
	manifestVersion uint32
	mu              *sync.Mutex
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
	return NewStoreForManifest(path, domain.IntentCanaryV0ManifestVersion)
}

// NewStoreForManifest binds one physical append-only chain to one immutable
// manifest version. Different versions must use different paths.
func NewStoreForManifest(path string, manifestVersion uint32) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("%w: path is empty", ErrInvalidEvent)
	}
	if _, err := domain.NewIntentCanaryV0ManifestForVersion(manifestVersion); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidEvent, err)
	}
	absPath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("resolve evidence path: %w", err)
	}
	lock, _ := pathLocks.LoadOrStore(absPath, &sync.Mutex{})
	return &Store{path: absPath, manifestVersion: manifestVersion, mu: lock.(*sync.Mutex)}, nil
}

// ManifestVersion reports the immutable version expected by this chain.
func (s *Store) ManifestVersion() uint32 { return s.manifestVersion }

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
	if err := validateEvent(event, records, false, s.manifestVersion); err != nil {
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
		if err := validateRecord(record, records, s.manifestVersion); err != nil {
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

func validateRecord(record diskRecord, prior []diskRecord, manifestVersion uint32) error {
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
	if err := validateEvent(record.Event, prior, true, manifestVersion); err != nil {
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

func validateEvent(event domain.EvidenceEvent, prior []diskRecord, allowLegacyStart bool, manifestVersion uint32) error {
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
	if effectiveManifestVersion(event) != manifestVersion {
		return fmt.Errorf(
			"%w: event manifest version %d does not match store version %d",
			ErrInvalidEvent,
			effectiveManifestVersion(event),
			manifestVersion,
		)
	}
	if manifestVersion == domain.IntentCanaryV0ManifestVersion2 && event.ManifestVersion == 0 {
		return fmt.Errorf("%w: manifest v2 event must state its version", ErrInvalidEvent)
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
	if event.Admission != nil {
		if err := validateAdmission(*event.Admission); err != nil {
			return err
		}
	}
	if event.Assignment != nil {
		if err := validateAssignment(*event.Assignment, manifestVersion); err != nil {
			return err
		}
	}
	if event.Context != nil {
		if err := validateContextBinding(*event.Context); err != nil {
			return err
		}
	}
	if event.AssessmentBasis != nil {
		if err := validateAssessmentBasis(*event.AssessmentBasis); err != nil {
			return err
		}
	}
	if event.FlowPhase != nil {
		if err := validateFlowPhase(event, *event.FlowPhase); err != nil {
			return err
		}
	}
	if event.CheckSet != nil {
		if err := validateCheckSet(*event.CheckSet); err != nil {
			return err
		}
	}
	if event.Terminal != nil {
		if err := validateTerminal(*event.Terminal); err != nil {
			return err
		}
	}
	if event.Telemetry != nil {
		if err := validateTelemetry(event, *event.Telemetry); err != nil {
			return err
		}
	}
	if event.LineageResolutionAttempts > 1 ||
		(event.Lineage == nil && event.LineageResolutionAttempts != 0) {
		return fmt.Errorf("%w: lineage resolution attempts must be 0 or 1 and require lineage", ErrInvalidEvent)
	}

	switch event.Kind {
	case domain.EventAdmissionDecided:
		if event.Admission == nil || event.ReasonCode == "" {
			return fmt.Errorf("%w: admission event requires a decision and reason", ErrInvalidEvent)
		}
		expectedPayloads := 1
		if event.Admission.Decision == domain.AdmissionEligible {
			expectedPayloads = 2
		}
		if payloadCount(event) != expectedPayloads {
			return fmt.Errorf("%w: admission payload does not match its decision", ErrInvalidEvent)
		}
		if err := validateAdmissionTransition(event, *event.Admission, prior); err != nil {
			return err
		}
	case domain.EventChangeStarted:
		if !allowLegacyStart {
			return fmt.Errorf("%w: new change-start events are replaced by admission decisions", ErrInvalidEvent)
		}
		if event.Assignment == nil || payloadCount(event) != 1 {
			return fmt.Errorf("%w: change-start event requires only assignment payload", ErrInvalidEvent)
		}
		if err := validateAssignmentTransition(event, *event.Assignment, prior); err != nil {
			return err
		}
	case domain.EventContextBound:
		if manifestVersion != domain.IntentCanaryV0ManifestVersion2 || event.Context == nil || payloadCount(event) != 1 {
			return fmt.Errorf("%w: context event is valid only for manifest v2 and requires one context payload", ErrInvalidEvent)
		}
		if err := validateContextTransition(event, prior); err != nil {
			return err
		}
	case domain.EventAssessmentBasisRecorded:
		if manifestVersion != domain.IntentCanaryV0ManifestVersion2 || event.AssessmentBasis == nil || payloadCount(event) != 1 {
			return fmt.Errorf("%w: assessment-basis event is valid only for manifest v2 and requires one basis payload", ErrInvalidEvent)
		}
		if err := validateAssessmentBasisTransition(event, prior); err != nil {
			return err
		}
	case domain.EventFlowPhaseRecorded:
		if manifestVersion != domain.IntentCanaryV0ManifestVersion2 || event.FlowPhase == nil || payloadCount(event) != 1 {
			return fmt.Errorf("%w: flow-phase event is valid only for manifest v2 and requires one phase payload", ErrInvalidEvent)
		}
		if err := validateFlowPhaseTransition(event, prior); err != nil {
			return err
		}
	case domain.EventCheckSetFrozen:
		if event.CheckSet == nil || payloadCount(event) != 1 {
			return fmt.Errorf("%w: check-set event requires only a check-set payload", ErrInvalidEvent)
		}
		if err := validateCheckSetTransition(event, prior); err != nil {
			return err
		}
	case domain.EventLineageRecorded:
		if event.Lineage == nil || payloadCount(event) != 1 {
			return fmt.Errorf("%w: lineage event requires only lineage payload", ErrInvalidEvent)
		}
		if err := validateLineageTransition(event, *event.Lineage, prior); err != nil {
			return err
		}
	case domain.EventTelemetryRecorded:
		if manifestVersion != domain.IntentCanaryV0ManifestVersion2 || event.Telemetry == nil || payloadCount(event) != 1 {
			return fmt.Errorf("%w: telemetry event is valid only for manifest v2 and requires one telemetry payload", ErrInvalidEvent)
		}
		if err := validateTelemetryTransition(event, prior); err != nil {
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
	case domain.EventAdmissionDecided,
		domain.EventChangeStarted,
		domain.EventContextBound,
		domain.EventAssessmentBasisRecorded,
		domain.EventFlowPhaseRecorded,
		domain.EventCheckSetFrozen,
		domain.EventLineageRecorded,
		domain.EventTelemetryRecorded,
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
		event.Admission != nil,
		event.Assignment != nil,
		event.Context != nil,
		event.AssessmentBasis != nil,
		event.FlowPhase != nil,
		event.CheckSet != nil,
		event.Lineage != nil,
		event.Terminal != nil,
		event.Telemetry != nil,
		event.Assessment != nil,
	} {
		if present {
			count++
		}
	}
	return count
}

func validateAdmission(admission domain.CanaryAdmission) error {
	switch admission.Decision {
	case domain.AdmissionEligible, domain.AdmissionExcluded:
	default:
		return fmt.Errorf("%w: unknown admission decision %q", ErrInvalidEvent, admission.Decision)
	}
	if !admission.Synthetic {
		return ErrRealCanaryNotActivated
	}
	return nil
}

func validateAssignment(assignment domain.CanaryAssignment, manifestVersion uint32) error {
	if !assignment.Synthetic {
		return ErrRealCanaryNotActivated
	}
	if assignment.Ordinal == 0 || assignment.Ordinal > domain.CanaryAssignmentCount {
		return fmt.Errorf("%w: assignment ordinal must be within 1..15", ErrInvalidEvent)
	}
	expectedVariant, err := domain.CanaryVariantForOrdinal(assignment.Ordinal)
	if err != nil || assignment.Variant != expectedVariant {
		return fmt.Errorf("%w: assignment variant must match immutable ordinal", ErrInvalidEvent)
	}
	if assignment.ManifestVersion != manifestVersion || assignment.IntentVersion == 0 {
		return fmt.Errorf("%w: assignment requires the store manifest version and a non-zero intent version", ErrInvalidEvent)
	}
	return validateCanonicalID("assignment.run_id", string(assignment.RunID))
}

func effectiveManifestVersion(event domain.EvidenceEvent) uint32 {
	if event.ManifestVersion == 0 {
		return domain.IntentCanaryV0ManifestVersion
	}
	return event.ManifestVersion
}

func validateContextBinding(binding domain.CanaryContextBinding) error {
	if err := validateCanonicalID("context.context_pack_id", string(binding.ContextPackID)); err != nil {
		return err
	}
	if binding.ContextPackVersion == 0 {
		return fmt.Errorf("%w: context pack version must be non-zero", ErrInvalidEvent)
	}
	return nil
}

func validateAssessmentBasis(basis domain.CanaryAssessmentBasis) error {
	if err := domain.ValidateCanaryAssessmentBasis(basis); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidEvent, err)
	}
	if err := validateReference("assessment_basis.intent_ref", string(basis.IntentRef)); err != nil {
		return err
	}
	if err := validateCanonicalID("assessment_basis.intent_id", string(basis.IntentID)); err != nil {
		return err
	}
	if basis.IntentVersion == 0 {
		return fmt.Errorf("%w: assessment basis intent version must be non-zero", ErrInvalidEvent)
	}
	if basis.Timing != domain.BasisPreExecution && basis.Timing != domain.BasisPostDelivery {
		return fmt.Errorf("%w: assessment basis timing is invalid", ErrInvalidEvent)
	}
	if len(basis.DesiredOutcomeIDs) == 0 || len(basis.SuccessSignalIDs) == 0 {
		return fmt.Errorf("%w: assessment basis requires desired outcomes and success signals", ErrInvalidEvent)
	}
	seen := make(map[domain.IntentItemID]struct{})
	for group, ids := range map[string][]domain.IntentItemID{
		"desired_outcome_ids": basis.DesiredOutcomeIDs,
		"non_goal_ids":        basis.NonGoalIDs,
		"success_signal_ids":  basis.SuccessSignalIDs,
	} {
		for index, id := range ids {
			if err := validateCanonicalID(fmt.Sprintf("assessment_basis.%s[%d]", group, index), string(id)); err != nil {
				return err
			}
			if _, duplicate := seen[id]; duplicate {
				return fmt.Errorf("%w: assessment basis item %q is duplicated", ErrInvalidEvent, id)
			}
			seen[id] = struct{}{}
		}
	}
	return nil
}

func validateFlowPhase(event domain.EvidenceEvent, phase domain.CanaryFlowPhase) error {
	if err := validateTimestamp("flow_phase.started_at", phase.StartedAt); err != nil {
		return err
	}
	if err := validateTimestamp("flow_phase.completed_at", phase.CompletedAt); err != nil {
		return err
	}
	if !phase.StartedAt.Before(phase.CompletedAt) {
		return fmt.Errorf("%w: flow phase completion must follow start", ErrInvalidEvent)
	}
	if phase.CompletedAt.After(event.OccurredAt) {
		return fmt.Errorf("%w: flow phase cannot complete after its evidence event", ErrInvalidEvent)
	}
	return nil
}

func validateTelemetry(event domain.EvidenceEvent, telemetry domain.CanaryTelemetry) error {
	if err := validateObservationReference("telemetry.session_lookup", string(telemetry.SessionLookup)); err != nil {
		return err
	}
	if !strings.HasPrefix(string(telemetry.SessionLookup), "langfuse-session:") {
		return fmt.Errorf("%w: telemetry session lookup must use langfuse-session", ErrInvalidEvent)
	}
	seen := make(map[domain.EvidenceReference]struct{}, len(telemetry.TraceIntervals))
	for index, interval := range telemetry.TraceIntervals {
		path := fmt.Sprintf("telemetry.trace_intervals[%d]", index)
		if err := validateObservationReference(path+".reference", string(interval.Reference)); err != nil {
			return err
		}
		if !strings.HasPrefix(string(interval.Reference), "langfuse-trace:") {
			return fmt.Errorf("%w: telemetry interval must use langfuse-trace", ErrInvalidEvent)
		}
		if err := validateTimestamp(path+".started_at", interval.StartedAt); err != nil {
			return err
		}
		if err := validateTimestamp(path+".ended_at", interval.EndedAt); err != nil {
			return err
		}
		if interval.EndedAt.Before(interval.StartedAt) {
			return fmt.Errorf("%w: telemetry interval is reversed", ErrInvalidEvent)
		}
		if _, duplicate := seen[interval.Reference]; duplicate {
			return fmt.Errorf("%w: telemetry trace is duplicated", ErrInvalidEvent)
		}
		seen[interval.Reference] = struct{}{}
	}
	if telemetry.OwnerReview != nil {
		if err := validateReference("telemetry.owner_review.reference", string(telemetry.OwnerReview.Reference)); err != nil {
			return err
		}
		if err := validateTimestamp("telemetry.owner_review.started_at", telemetry.OwnerReview.StartedAt); err != nil {
			return err
		}
		if err := validateTimestamp("telemetry.owner_review.ended_at", telemetry.OwnerReview.EndedAt); err != nil {
			return err
		}
		if telemetry.OwnerReview.EndedAt.Before(telemetry.OwnerReview.StartedAt) {
			return fmt.Errorf("%w: owner-review interval is reversed", ErrInvalidEvent)
		}
	}

	wantRefs := make([]domain.EvidenceReference, 0, len(telemetry.TraceIntervals)+1)
	wantRefs = append(wantRefs, telemetry.SessionLookup)
	for _, interval := range telemetry.TraceIntervals {
		wantRefs = append(wantRefs, interval.Reference)
	}
	if !sameReferenceSet(event.ObservationRefs, wantRefs) {
		return fmt.Errorf("%w: telemetry observation references must match its bounded payload", ErrInvalidEvent)
	}
	switch telemetry.Status {
	case domain.TelemetryAvailable:
		if len(telemetry.TraceIntervals) == 0 ||
			(event.ReasonCode != "" && event.Kind != domain.EventEvidenceCorrected) {
			return fmt.Errorf("%w: available telemetry requires traces and no failure reason", ErrInvalidEvent)
		}
	case domain.TelemetryUnavailable, domain.TelemetryConflict:
		if len(telemetry.TraceIntervals) != 0 || event.ReasonCode == "" {
			return fmt.Errorf("%w: non-available telemetry requires a reason and no trace intervals", ErrInvalidEvent)
		}
	default:
		return fmt.Errorf("%w: unknown telemetry status %q", ErrInvalidEvent, telemetry.Status)
	}
	return nil
}

func validateContextTransition(event domain.EvidenceEvent, prior []diskRecord) error {
	assignment, assignedAt := assignedChange(event, prior)
	if assignment == nil || assignment.Variant != domain.VariantFlow {
		return fmt.Errorf("%w: context binding requires an assigned flow change", ErrInvalidEvent)
	}
	if event.OccurredAt.Before(assignedAt) || terminalExists(event, prior) {
		return fmt.Errorf("%w: context binding must occur after assignment and before terminal evidence", ErrInvalidEvent)
	}
	if latestPayloadEvent(event, prior, func(previous domain.EvidenceEvent) bool { return previous.Context != nil }) != nil {
		return fmt.Errorf("%w: flow change already has a context binding", ErrInvalidEvent)
	}
	return nil
}

func validateAssessmentBasisTransition(event domain.EvidenceEvent, prior []diskRecord) error {
	assignment, assignedAt := assignedChange(event, prior)
	if assignment == nil || event.OccurredAt.Before(assignedAt) {
		return fmt.Errorf("%w: assessment basis requires an assigned change", ErrInvalidEvent)
	}
	if event.AssessmentBasis.IntentVersion != assignment.IntentVersion {
		return fmt.Errorf("%w: assessment basis intent version must match the immutable assignment", ErrInvalidEvent)
	}
	if latestPayloadEvent(event, prior, func(previous domain.EvidenceEvent) bool { return previous.AssessmentBasis != nil }) != nil {
		return fmt.Errorf("%w: assessment basis already exists", ErrInvalidEvent)
	}
	terminal, terminalAt := terminalChange(event, prior)
	switch assignment.Variant {
	case domain.VariantFlow:
		if event.AssessmentBasis.Timing != domain.BasisPreExecution || terminal != nil ||
			latestPayloadEvent(event, prior, func(previous domain.EvidenceEvent) bool { return previous.Context != nil }) == nil {
			return fmt.Errorf("%w: flow basis must be pre-execution after context and before terminal evidence", ErrInvalidEvent)
		}
	case domain.VariantBaseline:
		if event.AssessmentBasis.Timing != domain.BasisPostDelivery || terminal == nil ||
			terminal.State != domain.CanaryStateDelivered || event.OccurredAt.Before(terminalAt) {
			return fmt.Errorf("%w: baseline basis must be visibly post-delivery", ErrInvalidEvent)
		}
	}
	return nil
}

func validateFlowPhaseTransition(event domain.EvidenceEvent, prior []diskRecord) error {
	assignment, assignedAt := assignedChange(event, prior)
	if assignment == nil || assignment.Variant != domain.VariantFlow || terminalExists(event, prior) {
		return fmt.Errorf("%w: flow phase requires an open assigned flow change", ErrInvalidEvent)
	}
	if event.FlowPhase.StartedAt.Before(assignedAt) {
		return fmt.Errorf("%w: flow phase predates assignment", ErrInvalidEvent)
	}
	contextEvent := latestPayloadEvent(event, prior, func(previous domain.EvidenceEvent) bool { return previous.Context != nil })
	basisEvent := latestPayloadEvent(event, prior, func(previous domain.EvidenceEvent) bool { return previous.AssessmentBasis != nil })
	if contextEvent == nil || basisEvent == nil {
		return fmt.Errorf("%w: flow phase requires context and assessment basis", ErrInvalidEvent)
	}
	if event.FlowPhase.StartedAt.Before(contextEvent.OccurredAt) || event.FlowPhase.StartedAt.Before(basisEvent.OccurredAt) {
		return fmt.Errorf("%w: flow phase must start after context and assessment basis", ErrInvalidEvent)
	}
	if latestPayloadEvent(event, prior, func(previous domain.EvidenceEvent) bool { return previous.FlowPhase != nil }) != nil ||
		checkSetEvent(event, prior) != nil {
		return fmt.Errorf("%w: flow phase must be recorded once before checks", ErrInvalidEvent)
	}
	return nil
}

func validateTelemetryTransition(event domain.EvidenceEvent, prior []diskRecord) error {
	assignment, _ := assignedChange(event, prior)
	if assignment == nil {
		return fmt.Errorf("%w: telemetry requires an assigned change", ErrInvalidEvent)
	}
	lineageEvent := latestPayloadEvent(event, prior, func(previous domain.EvidenceEvent) bool {
		return previous.Lineage != nil && previous.Lineage.Status == domain.LineageVerified
	})
	if lineageEvent == nil {
		return fmt.Errorf("%w: telemetry requires verified lineage", ErrInvalidEvent)
	}
	wantLookup := domain.EvidenceReference("langfuse-session:" + string(lineageEvent.Lineage.RootSessionID))
	if event.Telemetry.SessionLookup != wantLookup {
		return fmt.Errorf("%w: telemetry lookup must match verified lineage", ErrInvalidEvent)
	}
	if latestPayloadEvent(event, prior, func(previous domain.EvidenceEvent) bool { return previous.Telemetry != nil }) != nil {
		return fmt.Errorf("%w: telemetry already exists; append a correction", ErrInvalidEvent)
	}
	if assignment.Variant == domain.VariantFlow {
		phaseEvent := latestPayloadEvent(event, prior, func(previous domain.EvidenceEvent) bool { return previous.FlowPhase != nil })
		if phaseEvent == nil {
			if event.Telemetry.Status != domain.TelemetryUnavailable || event.ReasonCode != "flow-phase-missing" {
				return fmt.Errorf("%w: flow telemetry without a phase must record flow-phase-missing as unavailable", ErrInvalidEvent)
			}
			return validateTelemetryMeasurement(*event.Telemetry, *assignment)
		}
		for _, interval := range event.Telemetry.TraceIntervals {
			if interval.StartedAt.Before(phaseEvent.FlowPhase.StartedAt) || interval.StartedAt.After(phaseEvent.FlowPhase.CompletedAt) ||
				interval.EndedAt.After(phaseEvent.FlowPhase.CompletedAt) {
				return fmt.Errorf("%w: flow trace starts outside the recorded phase", ErrInvalidEvent)
			}
		}
		return validateTelemetryMeasurement(*event.Telemetry, *assignment)
	}
	return validateTelemetryMeasurement(*event.Telemetry, *assignment)
}

func validateTelemetryMeasurement(telemetry domain.CanaryTelemetry, assignment domain.CanaryAssignment) error {
	switch assignment.Variant {
	case domain.VariantFlow:
		measurement, err := domain.CalculateCanaryFlowOverhead(domain.CanaryFlowOverheadInput{
			AgentTurns: telemetry.TraceIntervals, OwnerReview: telemetry.OwnerReview, OwnerReviewRequired: true,
		})
		if err != nil || telemetry.FlowOverhead == nil || *telemetry.FlowOverhead != measurement {
			return fmt.Errorf("%w: flow overhead must match the derived timing evidence", ErrInvalidEvent)
		}
		if telemetry.Status == domain.TelemetryAvailable && !measurement.Available {
			return fmt.Errorf("%w: available flow telemetry requires machine and owner-review timing", ErrInvalidEvent)
		}
	case domain.VariantBaseline:
		if telemetry.OwnerReview != nil || telemetry.FlowOverhead != nil {
			return fmt.Errorf("%w: baseline telemetry cannot record flow overhead", ErrInvalidEvent)
		}
	default:
		return fmt.Errorf("%w: unknown telemetry assignment variant", ErrInvalidEvent)
	}
	return nil
}

func terminalExists(event domain.EvidenceEvent, prior []diskRecord) bool {
	terminal, _ := terminalChange(event, prior)
	return terminal != nil
}

func latestPayloadEvent(
	event domain.EvidenceEvent,
	prior []diskRecord,
	matches func(domain.EvidenceEvent) bool,
) *domain.EvidenceEvent {
	var latest *domain.EvidenceEvent
	for index := range prior {
		previous := &prior[index].Event
		if previous.CanaryID == event.CanaryID && previous.ChangeID == event.ChangeID && matches(*previous) {
			latest = previous
		}
	}
	return latest
}

func validateAdmissionTransition(
	event domain.EvidenceEvent,
	admission domain.CanaryAdmission,
	prior []diskRecord,
) error {
	for _, record := range prior {
		previous := record.Event
		if previous.CanaryID != event.CanaryID {
			continue
		}
		if previous.Kind == domain.EventCanaryStopped {
			return fmt.Errorf("%w: canary admissions are stopped", ErrInvalidEvent)
		}
		if previous.ChangeID == event.ChangeID &&
			(previous.Admission != nil || previous.Assignment != nil) {
			return fmt.Errorf("%w: canary change already has an admission or assignment", ErrInvalidEvent)
		}
	}

	switch admission.Decision {
	case domain.AdmissionEligible:
		if event.Assignment == nil {
			return fmt.Errorf("%w: eligible admission requires assignment", ErrInvalidEvent)
		}
		return validateAssignmentTransition(event, *event.Assignment, prior)
	case domain.AdmissionExcluded:
		if event.Assignment != nil {
			return fmt.Errorf("%w: excluded admission cannot carry assignment", ErrInvalidEvent)
		}
		return nil
	default:
		return fmt.Errorf("%w: unknown admission decision %q", ErrInvalidEvent, admission.Decision)
	}
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
		if previous.CanaryID != event.CanaryID {
			continue
		}
		if previous.ChangeID == event.ChangeID &&
			(previous.Admission != nil || previous.Assignment != nil) {
			return fmt.Errorf("%w: canary change already has an admission or assignment", ErrInvalidEvent)
		}
		if previous.Assignment == nil {
			continue
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

func validateCheckSet(checkSet domain.CanaryCheckSet) error {
	if len(checkSet.CheckRefs) == 0 {
		return fmt.Errorf("%w: frozen check set must not be empty", ErrInvalidEvent)
	}
	seen := make(map[domain.EvidenceReference]struct{}, len(checkSet.CheckRefs))
	for index, reference := range checkSet.CheckRefs {
		field := fmt.Sprintf("check_set.check_refs[%d]", index)
		if err := validateCheckReference(field, string(reference)); err != nil {
			return err
		}
		if _, duplicate := seen[reference]; duplicate {
			return fmt.Errorf("%w: duplicate %s", ErrInvalidEvent, field)
		}
		seen[reference] = struct{}{}
	}
	return nil
}

func validateCheckSetTransition(event domain.EvidenceEvent, prior []diskRecord) error {
	assignment, assignedAt := assignedChange(event, prior)
	if assignment == nil {
		return fmt.Errorf("%w: check-set freeze requires an assigned change", ErrInvalidEvent)
	}
	if event.OccurredAt.Before(assignedAt) {
		return fmt.Errorf("%w: check-set event predates assignment", ErrInvalidEvent)
	}
	if assignment.ManifestVersion == domain.IntentCanaryV0ManifestVersion2 &&
		assignment.Variant == domain.VariantFlow &&
		latestPayloadEvent(event, prior, func(previous domain.EvidenceEvent) bool { return previous.FlowPhase != nil }) == nil {
		return fmt.Errorf("%w: manifest v2 flow checks require a completed flow phase", ErrInvalidEvent)
	}
	if checkSetEvent(event, prior) != nil {
		return fmt.Errorf("%w: canary change already has a frozen check set", ErrInvalidEvent)
	}
	terminal, _ := terminalChange(event, prior)
	if terminal != nil {
		return fmt.Errorf("%w: check set must be frozen before terminal evidence", ErrInvalidEvent)
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
		return fmt.Errorf("%w: terminal evidence requires an assigned change", ErrInvalidEvent)
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
	if terminal.State == domain.CanaryStateDelivered {
		frozen := checkSetEvent(event, prior)
		assignedEvent := assignmentEvent(event, prior)
		legacyAssignment := assignedEvent != nil && assignedEvent.Kind == domain.EventChangeStarted
		if (frozen == nil || frozen.CheckSet == nil) && !legacyAssignment {
			return fmt.Errorf("%w: delivery requires a prior frozen check set", ErrInvalidEvent)
		}
		if frozen != nil && event.OccurredAt.Before(frozen.OccurredAt) {
			return fmt.Errorf("%w: terminal evidence predates the effective check set", ErrInvalidEvent)
		}
		if frozen != nil && !sameReferenceSet(terminal.CheckRefs, frozen.CheckSet.CheckRefs) {
			return fmt.Errorf("%w: delivery checks must match the effective frozen set", ErrInvalidEvent)
		}
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
	if assignment.ManifestVersion == domain.IntentCanaryV0ManifestVersion2 {
		basisEvent := latestPayloadEvent(event, prior, func(previous domain.EvidenceEvent) bool { return previous.AssessmentBasis != nil })
		if basisEvent == nil {
			return fmt.Errorf("%w: manifest v2 assessment requires a frozen basis", ErrInvalidEvent)
		}
		if err := domain.ValidateAssessmentAgainstBasis(assessment, *basisEvent.AssessmentBasis); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidEvent, err)
		}
	} else if len(assessment.ItemJudgments) != 0 {
		return fmt.Errorf("%w: manifest v1 assessment cannot record v2 item judgments", ErrInvalidEvent)
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
	assigned := assignmentEvent(event, prior)
	if assigned != nil {
		return assigned.Assignment, assigned.OccurredAt
	}
	return nil, time.Time{}
}

func assignmentEvent(event domain.EvidenceEvent, prior []diskRecord) *domain.EvidenceEvent {
	for index := range prior {
		previous := &prior[index].Event
		if previous.CanaryID == event.CanaryID && previous.ChangeID == event.ChangeID && previous.Assignment != nil {
			return previous
		}
	}
	return nil
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

func checkSetEvent(event domain.EvidenceEvent, prior []diskRecord) *domain.EvidenceEvent {
	var latest *domain.EvidenceEvent
	for index := range prior {
		previous := &prior[index].Event
		if previous.CanaryID != event.CanaryID || previous.ChangeID != event.ChangeID || previous.CheckSet == nil {
			continue
		}
		if previous.Kind == domain.EventCheckSetFrozen || previous.Kind == domain.EventEvidenceCorrected {
			latest = previous
		}
	}
	return latest
}

func sameReferenceSet(left, right []domain.EvidenceReference) bool {
	if len(left) != len(right) {
		return false
	}
	seen := make(map[domain.EvidenceReference]struct{}, len(left))
	for _, reference := range left {
		seen[reference] = struct{}{}
	}
	for _, reference := range right {
		if _, ok := seen[reference]; !ok {
			return false
		}
	}
	return true
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
	if payloadCount(event) != 1 {
		return fmt.Errorf("%w: correction requires exactly one replacement payload", ErrInvalidCorrection)
	}
	switch {
	case target.Assessment != nil && event.Assessment != nil:
		return validateAssessmentCorrection(event, prior)
	case target.CheckSet != nil && event.CheckSet != nil:
		return validateCheckSetCorrection(event, *target, prior)
	case target.Telemetry != nil && event.Telemetry != nil:
		return validateTelemetryCorrection(event, prior)
	default:
		return fmt.Errorf("%w: correction payload must match assessment, check set, or telemetry", ErrInvalidCorrection)
	}
}

func validateTelemetryCorrection(event domain.EvidenceEvent, prior []diskRecord) error {
	latest := latestPayloadEvent(event, prior, func(previous domain.EvidenceEvent) bool {
		return previous.Telemetry != nil
	})
	if latest == nil || latest.ID != event.SupersedesEventID {
		return fmt.Errorf("%w: correction must supersede the latest telemetry event", ErrInvalidCorrection)
	}
	assignment, _ := assignedChange(event, prior)
	lineage := latestPayloadEvent(event, prior, func(previous domain.EvidenceEvent) bool {
		return previous.Lineage != nil && previous.Lineage.Status == domain.LineageVerified
	})
	if assignment == nil || lineage == nil {
		return fmt.Errorf("%w: telemetry correction requires assignment and verified lineage", ErrInvalidCorrection)
	}
	wantLookup := domain.EvidenceReference("langfuse-session:" + string(lineage.Lineage.RootSessionID))
	if event.Telemetry.SessionLookup != wantLookup {
		return fmt.Errorf("%w: corrected telemetry lookup must match verified lineage", ErrInvalidCorrection)
	}
	if assignment.Variant == domain.VariantFlow {
		phase := latestPayloadEvent(event, prior, func(previous domain.EvidenceEvent) bool { return previous.FlowPhase != nil })
		if phase == nil {
			return fmt.Errorf("%w: flow telemetry correction requires a phase", ErrInvalidCorrection)
		}
		for _, interval := range event.Telemetry.TraceIntervals {
			if interval.StartedAt.Before(phase.FlowPhase.StartedAt) || interval.StartedAt.After(phase.FlowPhase.CompletedAt) ||
				interval.EndedAt.After(phase.FlowPhase.CompletedAt) {
				return fmt.Errorf("%w: corrected flow trace starts outside the recorded phase", ErrInvalidCorrection)
			}
		}
	}
	if err := validateTelemetryMeasurement(*event.Telemetry, *assignment); err != nil {
		return fmt.Errorf("%w: corrected telemetry measurement is invalid", ErrInvalidCorrection)
	}
	return nil
}

func validateAssessmentCorrection(event domain.EvidenceEvent, prior []diskRecord) error {
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
		if assignment.ManifestVersion == domain.IntentCanaryV0ManifestVersion2 {
			basisEvent := latestPayloadEvent(event, prior, func(previous domain.EvidenceEvent) bool { return previous.AssessmentBasis != nil })
			if basisEvent == nil {
				return fmt.Errorf("%w: corrected v2 assessment requires a frozen basis", ErrInvalidCorrection)
			}
			if err := domain.ValidateAssessmentAgainstBasis(*event.Assessment, *basisEvent.AssessmentBasis); err != nil {
				return fmt.Errorf("%w: corrected item judgments are invalid: %v", ErrInvalidCorrection, err)
			}
		} else if len(event.Assessment.ItemJudgments) != 0 {
			return fmt.Errorf("%w: manifest v1 assessment cannot record v2 item judgments", ErrInvalidCorrection)
		}
		if event.Assessment.MaterialCorrectionBeforeDelivery != hasMaterialCorrection(event, prior) {
			return fmt.Errorf("%w: corrected material-correction value must match append-only events", ErrInvalidCorrection)
		}
	}
	return nil
}

func validateCheckSetCorrection(
	event domain.EvidenceEvent,
	target domain.EvidenceEvent,
	prior []diskRecord,
) error {
	latest := checkSetEvent(event, prior)
	if latest == nil || latest.ID != event.SupersedesEventID {
		return fmt.Errorf("%w: correction must supersede the latest check set", ErrInvalidCorrection)
	}
	if event.OccurredAt.Before(latest.OccurredAt) {
		return fmt.Errorf("%w: check-set correction predates the effective set", ErrInvalidCorrection)
	}
	terminal, _ := terminalChange(event, prior)
	if terminal != nil {
		return fmt.Errorf("%w: check-set correction must precede terminal evidence", ErrInvalidCorrection)
	}
	retained := make(map[domain.EvidenceReference]struct{}, len(event.CheckSet.CheckRefs))
	for _, reference := range event.CheckSet.CheckRefs {
		retained[reference] = struct{}{}
	}
	for _, reference := range target.CheckSet.CheckRefs {
		if _, ok := retained[reference]; !ok {
			return fmt.Errorf("%w: check-set correction cannot remove %q", ErrInvalidCorrection, reference)
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
