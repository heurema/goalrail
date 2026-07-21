package evidence

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/heurema/goalrail/internal/domain"
)

var testTime = time.Date(2026, time.July, 21, 12, 0, 0, 0, time.UTC)

const crossProcessHelperEnv = "GOALRAIL_TEST_EVIDENCE_HELPER"

func TestStoreSerializesConcurrentProcesses(t *testing.T) {
	const processCount = 8

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	dir := t.TempDir()
	evidencePath := filepath.Join(dir, "events.jsonl")
	gatePath := filepath.Join(dir, "start")
	type childProcess struct {
		command *exec.Cmd
		output  *bytes.Buffer
	}
	children := make([]childProcess, 0, processCount)
	readyPaths := make([]string, 0, processCount)
	for index := 1; index <= processCount; index++ {
		suffix := strconv.Itoa(index)
		readyPath := filepath.Join(dir, "ready-"+suffix)
		output := &bytes.Buffer{}
		command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestStoreConcurrentAppendHelper$")
		command.Env = append(os.Environ(),
			crossProcessHelperEnv+"=1",
			"GOALRAIL_TEST_EVIDENCE_PATH="+evidencePath,
			"GOALRAIL_TEST_EVIDENCE_GATE="+gatePath,
			"GOALRAIL_TEST_EVIDENCE_READY="+readyPath,
			"GOALRAIL_TEST_EVIDENCE_INDEX="+suffix,
		)
		command.Stdout = output
		command.Stderr = output
		if err := command.Start(); err != nil {
			t.Fatalf("start helper %d: %v", index, err)
		}
		children = append(children, childProcess{command: command, output: output})
		readyPaths = append(readyPaths, readyPath)
	}

	waitForReadyFiles(t, ctx, readyPaths)
	if err := os.WriteFile(gatePath, []byte("start\n"), 0o600); err != nil {
		t.Fatalf("release helpers: %v", err)
	}
	for index, child := range children {
		if err := child.command.Wait(); err != nil {
			t.Fatalf("helper %d: %v\n%s", index+1, err, child.output.String())
		}
	}
	if err := ctx.Err(); err != nil {
		t.Fatalf("concurrent append deadline: %v", err)
	}

	store, err := NewStore(evidencePath)
	if err != nil {
		t.Fatalf("open concurrent store: %v", err)
	}
	if err := store.Verify(); err != nil {
		t.Fatalf("verify concurrent store: %v", err)
	}
	events, err := store.ReadAll()
	if err != nil {
		t.Fatalf("read concurrent store: %v", err)
	}
	if len(events) != processCount {
		t.Fatalf("event count = %d, want %d", len(events), processCount)
	}
	seen := make(map[domain.EvidenceEventID]struct{}, processCount)
	for _, event := range events {
		if _, duplicate := seen[event.ID]; duplicate {
			t.Fatalf("duplicate event %q", event.ID)
		}
		seen[event.ID] = struct{}{}
	}
	for index := 1; index <= processCount; index++ {
		id := domain.EvidenceEventID("event-" + strconv.Itoa(index))
		if _, exists := seen[id]; !exists {
			t.Fatalf("missing event %q", id)
		}
	}
}

func TestStoreReadersWaitForInProgressFirstAppend(t *testing.T) {
	templateStore := newTestStore(t)
	event := validLineageEvent("event-committed")
	if err := templateStore.Append(event); err != nil {
		t.Fatalf("append template event: %v", err)
	}
	templateBytes := readStoreFile(t, templateStore)

	operations := []struct {
		name string
		run  func(*Store) error
	}{
		{
			name: "read all",
			run: func(store *Store) error {
				events, err := store.ReadAll()
				if err != nil {
					return err
				}
				if len(events) != 1 || events[0].ID != event.ID {
					return fmt.Errorf("events = %#v, want committed event %q", events, event.ID)
				}
				return nil
			},
		},
		{name: "verify", run: func(store *Store) error { return store.Verify() }},
	}
	for _, operation := range operations {
		t.Run(operation.name, func(t *testing.T) {
			dir := t.TempDir()
			evidencePath := filepath.Join(dir, "events.jsonl")
			store, err := NewStore(evidencePath)
			if err != nil {
				t.Fatalf("open target store: %v", err)
			}
			writerLock, err := store.openProcessLock(true)
			if err != nil {
				t.Fatalf("acquire writer lock: %v", err)
			}
			lockReleased := false
			t.Cleanup(func() {
				if !lockReleased {
					closeLockedFile(writerLock)
				}
			})

			started := make(chan struct{})
			done := make(chan error, 1)
			go func() {
				close(started)
				done <- operation.run(store)
			}()
			<-started
			select {
			case err := <-done:
				t.Fatalf("reader returned before first append committed: %v", err)
			case <-time.After(100 * time.Millisecond):
			}

			if err := os.WriteFile(evidencePath, templateBytes, 0o600); err != nil {
				t.Fatalf("commit first append: %v", err)
			}
			closeLockedFile(writerLock)
			lockReleased = true
			select {
			case err := <-done:
				if err != nil {
					t.Fatalf("reader after first append: %v", err)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("reader did not resume after first append committed")
			}
		})
	}
}

func TestStoreConcurrentAppendHelper(t *testing.T) {
	if os.Getenv(crossProcessHelperEnv) != "1" {
		return
	}
	index, err := strconv.Atoi(os.Getenv("GOALRAIL_TEST_EVIDENCE_INDEX"))
	if err != nil || index < 1 {
		t.Fatalf("invalid helper index: %q", os.Getenv("GOALRAIL_TEST_EVIDENCE_INDEX"))
	}
	readyPath := os.Getenv("GOALRAIL_TEST_EVIDENCE_READY")
	if err := os.WriteFile(readyPath, []byte("ready\n"), 0o600); err != nil {
		t.Fatalf("write ready marker: %v", err)
	}
	gatePath := os.Getenv("GOALRAIL_TEST_EVIDENCE_GATE")
	deadline := time.NewTimer(10 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		if _, err := os.Stat(gatePath); err == nil {
			break
		} else if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("inspect start marker: %v", err)
		}
		select {
		case <-deadline.C:
			t.Fatal("start marker deadline exceeded")
		case <-ticker.C:
		}
	}

	suffix := strconv.Itoa(index)
	store, err := NewStore(os.Getenv("GOALRAIL_TEST_EVIDENCE_PATH"))
	if err != nil {
		t.Fatalf("open helper store: %v", err)
	}
	event := validAssignmentEvent(
		domain.EvidenceEventID("event-"+suffix),
		domain.ChangeID("change-"+suffix),
		domain.RunID("run-"+suffix),
		1,
	)
	event.CanaryID = domain.CanaryID("canary-" + suffix)
	if err := store.Append(event); err != nil {
		t.Fatalf("append helper event: %v", err)
	}
}

func waitForReadyFiles(t *testing.T, ctx context.Context, paths []string) {
	t.Helper()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		allReady := true
		for _, path := range paths {
			if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
				allReady = false
				break
			} else if err != nil {
				t.Fatalf("inspect ready marker: %v", err)
			}
		}
		if allReady {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("ready marker deadline: %v", ctx.Err())
		case <-ticker.C:
		}
	}
}

func TestStoreAppendsAndReadsVerifiedEvents(t *testing.T) {
	store := newTestStore(t)
	first := validLineageEvent("event-1")
	first.ObservationRefs = []domain.EvidenceReference{
		"langfuse-session:session-1",
		"langfuse-trace:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
	second := validAssessmentEvent("event-2")

	if err := store.Append(first); err != nil {
		t.Fatalf("append first event: %v", err)
	}
	if err := store.Append(second); err != nil {
		t.Fatalf("append second event: %v", err)
	}
	events, err := store.ReadAll()
	if err != nil {
		t.Fatalf("read events: %v", err)
	}
	if len(events) != 2 || events[0].ID != first.ID || events[1].ID != second.ID {
		t.Fatalf("unexpected event order: %#v", events)
	}
	if len(events[0].ObservationRefs) != 2 || events[0].ObservationRefs[1] != first.ObservationRefs[1] {
		t.Fatalf("observation references were not preserved: %#v", events[0].ObservationRefs)
	}
	if err := store.Verify(); err != nil {
		t.Fatalf("verify store: %v", err)
	}
}

func TestStoreRejectsSymlinkEvidencePathForEveryOperation(t *testing.T) {
	dir := t.TempDir()
	targetPath := filepath.Join(dir, "target.jsonl")
	if err := os.WriteFile(targetPath, nil, 0o600); err != nil {
		t.Fatalf("write symlink target: %v", err)
	}
	linkPath := filepath.Join(dir, "events.jsonl")
	if err := os.Symlink(targetPath, linkPath); err != nil {
		t.Fatalf("create evidence symlink: %v", err)
	}
	store, err := NewStore(linkPath)
	if err != nil {
		t.Fatalf("new symlink store: %v", err)
	}
	operations := []struct {
		name string
		run  func() error
	}{
		{name: "append", run: func() error { return store.Append(validAssignmentEvent("event-1", "change-1", "run-1", 1)) }},
		{name: "read", run: func() error { _, err := store.ReadAll(); return err }},
		{name: "verify", run: store.Verify},
	}
	for _, operation := range operations {
		t.Run(operation.name, func(t *testing.T) {
			if err := operation.run(); !errors.Is(err, ErrIntegrity) {
				t.Fatalf("error = %v, want ErrIntegrity", err)
			}
		})
	}
}

func TestStoreRejectsDuplicateWithoutChangingFile(t *testing.T) {
	store := newTestStore(t)
	event := validLineageEvent("event-1")
	if err := store.Append(event); err != nil {
		t.Fatalf("append event: %v", err)
	}
	before := readStoreFile(t, store)

	err := store.Append(event)
	if !errors.Is(err, ErrDuplicateEventID) {
		t.Fatalf("append duplicate error = %v, want ErrDuplicateEventID", err)
	}
	after := readStoreFile(t, store)
	if !bytes.Equal(before, after) {
		t.Fatal("duplicate append changed prior evidence")
	}
}

func TestCorrectionAppendsLinkAndPreservesOriginalBytes(t *testing.T) {
	store := newTestStore(t)
	original := validAssessmentEvent("event-1")
	if err := store.Append(original); err != nil {
		t.Fatalf("append original: %v", err)
	}
	before := readStoreFile(t, store)

	correctedAssessment := *original.Assessment
	correctedAssessment.Outcome = domain.IntentPartial
	correction := original
	correction.ID = "event-2"
	correction.Kind = domain.EventEvidenceCorrected
	correction.OccurredAt = testTime.Add(time.Minute)
	correction.Assessment = &correctedAssessment
	correction.ReasonCode = "owner-correction"
	correction.SupersedesEventID = original.ID
	if err := store.Append(correction); err != nil {
		t.Fatalf("append correction: %v", err)
	}
	after := readStoreFile(t, store)
	if !bytes.HasPrefix(after, before) {
		t.Fatal("correction replaced or modified original bytes")
	}

	events, err := store.ReadAll()
	if err != nil {
		t.Fatalf("read corrected record: %v", err)
	}
	if len(events) != 2 || events[0].Assessment.Outcome != domain.IntentMatch {
		t.Fatalf("original event was not preserved: %#v", events)
	}
	if events[1].SupersedesEventID != events[0].ID || events[1].Assessment.Outcome != domain.IntentPartial {
		t.Fatalf("correction link or payload missing: %#v", events[1])
	}
}

func TestStoreRejectsInvalidCorrectionLinks(t *testing.T) {
	store := newTestStore(t)
	original := validAssessmentEvent("event-1")
	if err := store.Append(original); err != nil {
		t.Fatalf("append original: %v", err)
	}

	correction := original
	correction.ID = "event-2"
	correction.Kind = domain.EventEvidenceCorrected
	correction.OccurredAt = testTime.Add(time.Minute)
	correction.ReasonCode = "owner-correction"
	correction.SupersedesEventID = "missing-event"
	err := store.Append(correction)
	if !errors.Is(err, ErrInvalidCorrection) {
		t.Fatalf("append correction error = %v, want ErrInvalidCorrection", err)
	}

	correction.SupersedesEventID = original.ID
	correction.ChangeID = "another-change"
	err = store.Append(correction)
	if !errors.Is(err, ErrInvalidCorrection) {
		t.Fatalf("cross-change correction error = %v, want ErrInvalidCorrection", err)
	}
}

func TestStoreDetectsPriorEventRewrite(t *testing.T) {
	store := newTestStore(t)
	if err := store.Append(validLineageEvent("event-1")); err != nil {
		t.Fatalf("append first event: %v", err)
	}
	if err := store.Append(validAssessmentEvent("event-2")); err != nil {
		t.Fatalf("append second event: %v", err)
	}

	contents := readStoreFile(t, store)
	tampered := bytes.Replace(contents, []byte(`"actor":"owner"`), []byte(`"actor":"other"`), 1)
	if bytes.Equal(contents, tampered) {
		t.Fatal("test fixture did not find actor field to tamper")
	}
	if err := os.WriteFile(store.path, tampered, 0o600); err != nil {
		t.Fatalf("tamper evidence file: %v", err)
	}

	if err := store.Verify(); !errors.Is(err, ErrIntegrity) {
		t.Fatalf("verify tampered record error = %v, want ErrIntegrity", err)
	}
	if err := store.Append(validAssessmentEvent("event-3")); !errors.Is(err, ErrIntegrity) {
		t.Fatalf("append after tamper error = %v, want ErrIntegrity", err)
	}
}

func TestStoreRejectsSensitiveOrRawPayload(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*domain.EvidenceEvent)
	}{
		{
			name: "credential-shaped actor",
			mutate: func(event *domain.EvidenceEvent) {
				event.Actor = "sk-live-secret"
			},
		},
		{
			name: "credential-shaped run ID",
			mutate: func(event *domain.EvidenceEvent) {
				event.Lineage.RunID = "ghp_example"
			},
		},
		{
			name: "raw source content",
			mutate: func(event *domain.EvidenceEvent) {
				event.SourceRef = "source:this is raw prompt content"
			},
		},
		{
			name: "raw observation content",
			mutate: func(event *domain.EvidenceEvent) {
				event.ObservationRefs = []domain.EvidenceReference{"langfuse-trace:raw trace output"}
			},
		},
		{
			name: "free-form correction reason",
			mutate: func(event *domain.EvidenceEvent) {
				event.ReasonCode = "contains raw prose"
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := newTestStore(t)
			event := validLineageEvent("event-1")
			test.mutate(&event)
			err := store.Append(event)
			if !errors.Is(err, ErrSensitivePayload) {
				t.Fatalf("append error = %v, want ErrSensitivePayload", err)
			}
			if _, statErr := os.Stat(store.path); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("rejected event created evidence file: %v", statErr)
			}
		})
	}
}

func TestStoreAllowsOnlyApprovedRepositoryReferenceKinds(t *testing.T) {
	lineageCases := []struct {
		name   string
		mutate func(*domain.EvidenceEvent)
	}{
		{
			name: "prompt-shaped source scheme",
			mutate: func(event *domain.EvidenceEvent) {
				event.SourceRef = "prompt:delete-everything"
			},
		},
		{
			name: "transcript path disguised as review source",
			mutate: func(event *domain.EvidenceEvent) {
				event.SourceRef = "review:private/raw-transcript.jsonl"
			},
		},
		{
			name: "unapproved observation provider",
			mutate: func(event *domain.EvidenceEvent) {
				event.ObservationRefs = []domain.EvidenceReference{
					"otel-trace:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				}
			},
		},
		{
			name: "path-shaped session lookup",
			mutate: func(event *domain.EvidenceEvent) {
				event.ObservationRefs = []domain.EvidenceReference{
					"langfuse-session:private/raw-transcript",
				}
			},
		},
		{
			name: "noncanonical trace identity",
			mutate: func(event *domain.EvidenceEvent) {
				event.ObservationRefs = []domain.EvidenceReference{
					"langfuse-trace:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
				}
			},
		},
	}
	for _, test := range lineageCases {
		t.Run(test.name, func(t *testing.T) {
			store := newTestStore(t)
			event := validLineageEvent("event-1")
			test.mutate(&event)
			if err := store.Append(event); !errors.Is(err, ErrSensitivePayload) {
				t.Fatalf("append error = %v, want ErrSensitivePayload", err)
			}
		})
	}

	t.Run("raw content cannot masquerade as check reference", func(t *testing.T) {
		store := newTestStore(t)
		assignment := validAssignmentEvent("event-start-1", "change-1", "run-1", 1)
		if err := store.Append(assignment); err != nil {
			t.Fatalf("append assignment: %v", err)
		}
		terminal := domain.EvidenceEvent{
			ID:         "event-terminal-1",
			CanaryID:   assignment.CanaryID,
			ChangeID:   assignment.ChangeID,
			Kind:       domain.EventTerminalStateChanged,
			OccurredAt: testTime.Add(time.Minute),
			Actor:      "operator",
			SourceRef:  "review:handoff-1",
			Terminal: &domain.CanaryTerminal{
				State:       domain.CanaryStateDelivered,
				CheckRefs:   []domain.EvidenceReference{"source:copied-code-content"},
				ChecksGreen: true,
			},
		}
		if err := store.Append(terminal); !errors.Is(err, ErrSensitivePayload) {
			t.Fatalf("append error = %v, want ErrSensitivePayload", err)
		}
	})
}

func TestStoreRejectsDuplicateObservationReferences(t *testing.T) {
	store := newTestStore(t)
	event := validLineageEvent("event-1")
	event.ObservationRefs = []domain.EvidenceReference{
		"langfuse-session:session-1",
		"langfuse-session:session-1",
	}

	err := store.Append(event)
	if !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("append error = %v, want ErrInvalidEvent", err)
	}
}

func TestStoreRejectsUnknownStoredFields(t *testing.T) {
	store := newTestStore(t)
	if err := store.Append(validLineageEvent("event-1")); err != nil {
		t.Fatalf("append event: %v", err)
	}
	contents := readStoreFile(t, store)
	tampered := bytes.Replace(contents, []byte(`"event":{`), []byte(`"event":{"prompt":"raw-secret",`), 1)
	if err := os.WriteFile(store.path, tampered, 0o600); err != nil {
		t.Fatalf("inject unknown field: %v", err)
	}
	if err := store.Verify(); !errors.Is(err, ErrIntegrity) || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("verify unknown field error = %v, want strict integrity error", err)
	}
}

func TestStoreRejectsTruncatedFinalRecord(t *testing.T) {
	store := newTestStore(t)
	if err := store.Append(validLineageEvent("event-1")); err != nil {
		t.Fatalf("append event: %v", err)
	}
	contents := readStoreFile(t, store)
	if err := os.WriteFile(store.path, bytes.TrimSuffix(contents, []byte{'\n'}), 0o600); err != nil {
		t.Fatalf("truncate final delimiter: %v", err)
	}
	if err := store.Verify(); !errors.Is(err, ErrIntegrity) {
		t.Fatalf("verify truncated record error = %v, want ErrIntegrity", err)
	}
}

func TestStoreEnforcesAssignedCanaryLifecycleTransitions(t *testing.T) {
	store := newTestStore(t)
	assignment := validAssignmentEvent("event-start-1", "change-1", "run-1", 1)
	if err := store.Append(assignment); err != nil {
		t.Fatalf("append assignment: %v", err)
	}

	duplicateOrdinal := validAssignmentEvent("event-start-2", "change-2", "run-2", 1)
	if err := store.Append(duplicateOrdinal); !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("duplicate ordinal error = %v, want ErrInvalidEvent", err)
	}

	wrongRun := validLineageEvent("event-lineage-wrong-run")
	wrongRun.CanaryID = assignment.CanaryID
	wrongRun.ChangeID = assignment.ChangeID
	wrongRun.Lineage.ChangeID = assignment.ChangeID
	wrongRun.Lineage.RunID = "run-other"
	wrongRun.OccurredAt = testTime.Add(time.Minute)
	if err := store.Append(wrongRun); !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("wrong lineage run error = %v, want ErrInvalidEvent", err)
	}

	terminal := domain.EvidenceEvent{
		ID:         "event-terminal-1",
		CanaryID:   assignment.CanaryID,
		ChangeID:   assignment.ChangeID,
		Kind:       domain.EventTerminalStateChanged,
		OccurredAt: testTime.Add(2 * time.Minute),
		Actor:      "operator",
		SourceRef:  "review:handoff-1",
		Terminal: &domain.CanaryTerminal{
			State:       domain.CanaryStateDelivered,
			CheckRefs:   []domain.EvidenceReference{"check:go-test"},
			ChecksGreen: true,
		},
	}
	if err := store.Append(terminal); err != nil {
		t.Fatalf("append terminal: %v", err)
	}

	repeat := true
	assessment := validAssessmentEvent("event-assessment-1")
	assessment.CanaryID = assignment.CanaryID
	assessment.ChangeID = assignment.ChangeID
	assessment.OccurredAt = testTime.Add(3 * time.Minute)
	assessment.Assessment.AssessedAt = assessment.OccurredAt
	assessment.Assessment.RepeatOptIn = &repeat
	assessment.Assessment.ChecksGreen = false
	if err := store.Append(assessment); !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("mismatched check state error = %v, want ErrInvalidEvent", err)
	}

	assessment.Assessment.ChecksGreen = true
	if err := store.Append(assessment); err != nil {
		t.Fatalf("append valid assessment: %v", err)
	}
	secondAssessment := assessment
	secondAssessment.ID = "event-assessment-2"
	secondAssessment.OccurredAt = testTime.Add(4 * time.Minute)
	secondAssessment.Assessment = &domain.Assessment{
		Outcome:     domain.IntentPartial,
		AssessedBy:  "owner",
		AssessedAt:  secondAssessment.OccurredAt,
		ChecksGreen: true,
		RepeatOptIn: &repeat,
	}
	if err := store.Append(secondAssessment); !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("second assessment error = %v, want ErrInvalidEvent", err)
	}
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := NewStore(filepath.Join(t.TempDir(), "evidence.jsonl"))
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	return store
}

func validLineageEvent(id domain.EvidenceEventID) domain.EvidenceEvent {
	return domain.EvidenceEvent{
		ID:         id,
		CanaryID:   "canary-v0",
		ChangeID:   "change-1",
		Kind:       domain.EventLineageRecorded,
		OccurredAt: testTime,
		Actor:      "goalrail-adapter",
		SourceRef:  "hook:session-start",
		Lineage: &domain.ExecutionLineage{
			Status:         domain.LineageVerified,
			ChangeID:       "change-1",
			RunID:          "run-1",
			RootSessionID:  "session-1",
			IdentitySource: domain.SessionIdentityLifecycleHook,
			ContextDigest:  strings.Repeat("a", 64),
		},
	}
}

func validAssessmentEvent(id domain.EvidenceEventID) domain.EvidenceEvent {
	repeat := true
	return domain.EvidenceEvent{
		ID:         id,
		CanaryID:   "canary-v0",
		ChangeID:   "change-1",
		Kind:       domain.EventAssessmentRecorded,
		OccurredAt: testTime,
		Actor:      "owner",
		SourceRef:  "review:owner-assessment",
		Assessment: &domain.Assessment{
			Outcome:     domain.IntentMatch,
			AssessedBy:  "owner",
			AssessedAt:  testTime,
			ChecksGreen: true,
			RepeatOptIn: &repeat,
		},
	}
}

func TestStoreStopMarkerBlocksDirectAssignmentAppend(t *testing.T) {
	store := newTestStore(t)
	if err := store.Append(validAssignmentEvent(
		"event-assignment-before-stop",
		"change-before-stop",
		"run-before-stop",
		1,
	)); err != nil {
		t.Fatalf("append assignment before stop: %v", err)
	}
	if err := store.Append(domain.EvidenceEvent{
		ID:         "event-canary-stop",
		CanaryID:   domain.IntentCanaryV0ManifestID,
		Kind:       domain.EventCanaryStopped,
		OccurredAt: testTime.Add(time.Minute),
		Actor:      "owner",
		SourceRef:  "review:rollback-test",
		ReasonCode: "rollback-exercise",
	}); err != nil {
		t.Fatalf("append canary stop: %v", err)
	}
	if err := store.Append(validAssignmentEvent(
		"event-assignment-after-stop",
		"change-after-stop",
		"run-after-stop",
		2,
	)); !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("direct assignment after stop error = %v, want ErrInvalidEvent", err)
	}
	events, err := store.ReadAll()
	if err != nil {
		t.Fatalf("read stopped event chain: %v", err)
	}
	if len(events) != 2 || events[1].Kind != domain.EventCanaryStopped {
		t.Fatalf("rejected append changed stopped chain: %#v", events)
	}
}

func validAssignmentEvent(
	id domain.EvidenceEventID,
	changeID domain.ChangeID,
	runID domain.RunID,
	ordinal uint32,
) domain.EvidenceEvent {
	variant, err := domain.CanaryVariantForOrdinal(ordinal)
	if err != nil {
		panic(err)
	}
	return domain.EvidenceEvent{
		ID:         id,
		CanaryID:   domain.IntentCanaryV0ManifestID,
		ChangeID:   changeID,
		Kind:       domain.EventChangeStarted,
		OccurredAt: testTime,
		Actor:      "operator",
		SourceRef:  "request:synthetic-change",
		Assignment: &domain.CanaryAssignment{
			Ordinal:         ordinal,
			Variant:         variant,
			ManifestVersion: domain.IntentCanaryV0ManifestVersion,
			IntentVersion:   1,
			RunID:           runID,
			Synthetic:       true,
		},
	}
}

func readStoreFile(t *testing.T, store *Store) []byte {
	t.Helper()
	contents, err := os.ReadFile(store.path)
	if err != nil {
		t.Fatalf("read store file: %v", err)
	}
	return contents
}
