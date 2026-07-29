package ambient

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const validQuestion = `---
schema: goalrail.escalation/v0
block_class: conflicting-requirements
question: Which retention period governs when the two documents disagree?
---
`

type recordingStore struct {
	bytes map[string][]byte
	json  map[string]any
	fail  bool
}

func newRecordingStore() *recordingStore {
	return &recordingStore{bytes: map[string][]byte{}, json: map[string]any{}}
}

func (store *recordingStore) WriteBytesOnce(relative string, content []byte, _ bool) error {
	if store.fail {
		return os.ErrPermission
	}
	store.bytes[relative] = append([]byte(nil), content...)
	return nil
}

func (store *recordingStore) WriteJSONOnce(relative string, value any, _ bool) error {
	if store.fail {
		return os.ErrPermission
	}
	store.json[relative] = value
	return nil
}

func (store *recordingStore) wrote() int { return len(store.bytes) + len(store.json) }

type fixedIntents struct {
	reference *IntentRef
	reason    string
}

func (resolver fixedIntents) ActiveConfirmedIntent(string) (*IntentRef, string) {
	return resolver.reference, resolver.reason
}

func fixedClock() func() time.Time {
	return func() time.Time { return time.Date(2026, 7, 29, 17, 0, 0, 0, time.UTC) }
}

func TestUninitializedRepositoryIsInvisible(t *testing.T) {
	// A persistent hook fires for every session the user starts anywhere.
	// Acting outside an initialized repository would monitor unrelated work.
	root := t.TempDir()
	writeQuestion(t, root, validQuestion)
	store := newRecordingStore()

	announcement, archived, err := StartSession(store, root, fixedClock())
	if err != nil {
		t.Fatal(err)
	}
	if announcement != "" || archived {
		t.Fatal("an unconnected repository produced ambient behaviour")
	}
	record, err := StopSession(store, root, "session-one", fixedIntents{}, fixedClock())
	if err != nil {
		t.Fatal(err)
	}
	if record != nil {
		t.Fatal("an unconnected repository produced a question record")
	}
	if store.wrote() != 0 {
		t.Fatalf("an unconnected repository caused %d writes", store.wrote())
	}
	// The question file itself must be left exactly as the user's own work.
	if _, err := os.Stat(questionPath(root)); err != nil {
		t.Fatal("an unconnected repository had its files touched")
	}
}

func TestInitializeIsExplicitAndIdempotent(t *testing.T) {
	root := t.TempDir()
	if IsInitialized(root) {
		t.Fatal("a fresh directory reported itself initialized")
	}
	marker, created, err := Initialize(root, fixedClock())
	if err != nil {
		t.Fatal(err)
	}
	if !created || marker.Schema != MarkerSchema {
		t.Fatalf("marker = %+v created = %v", marker, created)
	}
	if !IsInitialized(root) {
		t.Fatal("initialization did not take effect")
	}
	if _, createdAgain, err := Initialize(root, fixedClock()); err != nil || createdAgain {
		t.Fatalf("second initialization created = %v err = %v", createdAgain, err)
	}
	removed, err := Deinitialize(root)
	if err != nil || !removed {
		t.Fatalf("deinitialize removed = %v err = %v", removed, err)
	}
	if IsInitialized(root) {
		t.Fatal("deinitialization left the repository connected")
	}
}

func TestMalformedMarkerIsTreatedAsNotOurs(t *testing.T) {
	// Distinguishing "malformed" from "absent" would make the hook act on a
	// repository it was never given.
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".goalrail"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(root, filepath.FromSlash(MarkerPath)),
		[]byte("{not json"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
	if IsInitialized(root) {
		t.Fatal("a malformed marker was accepted as opt-in")
	}
}

func TestStartSessionAnnouncesAndArchivesStaleQuestion(t *testing.T) {
	root := initializedRepository(t)
	writeQuestion(t, root, validQuestion)
	store := newRecordingStore()

	announcement, archived, err := StartSession(store, root, fixedClock())
	if err != nil {
		t.Fatal(err)
	}
	if announcement != AmbientAnnouncement {
		t.Fatal("the session was not told the channel exists")
	}
	if !archived {
		t.Fatal("a stale question was not archived")
	}
	// The path must be clear, or the new session inherits someone else's
	// question and gets certified by work it never saw.
	if _, err := os.Stat(questionPath(root)); !os.IsNotExist(err) {
		t.Fatal("the stale question was left in place for the new session")
	}
	found := false
	for name, content := range store.bytes {
		if strings.HasPrefix(name, "ambient/archive/") && string(content) == validQuestion {
			found = true
		}
	}
	if !found {
		t.Fatalf("the archived bytes were not retained: %v", store.bytes)
	}
}

func TestStopSessionRetainsAndBindsTheQuestion(t *testing.T) {
	root := initializedRepository(t)
	writeQuestion(t, root, validQuestion)
	store := newRecordingStore()
	reference := &IntentRef{
		ID:      "intent-example",
		Version: 2,
		Digest:  "sha256:" + strings.Repeat("c", 64),
		Change:  "example-change",
	}

	record, err := StopSession(
		store, root, "session-one", fixedIntents{reference: reference}, fixedClock(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if record == nil || record.Intent == nil {
		t.Fatalf("record = %+v", record)
	}
	if record.Intent.ID != "intent-example" || record.Intent.Version != 2 {
		t.Fatalf("record intent = %+v", record.Intent)
	}
	if record.Invalid != "" || record.UnboundWhy != "" {
		t.Fatalf("a valid bound question reported problems: %+v", record)
	}
	if !strings.HasPrefix(record.Digest, "sha256:") || record.RetainedRef == "" {
		t.Fatalf("record lacks retained evidence: %+v", record)
	}
	if string(store.bytes[record.RetainedRef]) != validQuestion {
		t.Fatal("retained bytes differ from the question")
	}
	if record.SessionRef != "session-one" {
		t.Fatalf("session reference = %q", record.SessionRef)
	}

	// Deleting the file afterwards must not change what was recorded.
	if err := os.Remove(questionPath(root)); err != nil {
		t.Fatal(err)
	}
	if string(store.bytes[record.RetainedRef]) != validQuestion {
		t.Fatal("retained bytes followed the worktree file")
	}
}

func TestStopSessionRecordsWhyItCouldNotBind(t *testing.T) {
	// A guessed binding would poison the chain the record exists to serve.
	for _, reason := range []string{
		reasonNoChanges, reasonNoConfirmed, reasonAmbiguous,
	} {
		t.Run(reason, func(t *testing.T) {
			root := initializedRepository(t)
			writeQuestion(t, root, validQuestion)
			store := newRecordingStore()

			record, err := StopSession(
				store, root, "session-one", fixedIntents{reason: reason}, fixedClock(),
			)
			if err != nil {
				t.Fatal(err)
			}
			if record == nil || record.Intent != nil {
				t.Fatalf("record = %+v, want unbound", record)
			}
			if record.UnboundWhy != reason {
				t.Fatalf("unbound reason = %q, want %q", record.UnboundWhy, reason)
			}
			// Unbound is still retained: an unanswerable question is worse
			// than an unattributed one.
			if store.bytes[record.RetainedRef] == nil {
				t.Fatal("an unbound question was not retained")
			}
		})
	}
}

func TestStopSessionRecordsAnInvalidQuestionWithoutJudgingTheWorktree(t *testing.T) {
	root := initializedRepository(t)
	writeQuestion(t, root, "needs api_key: 8f3ca11b9d0e4c72 to proceed\n")
	// An interactive session edits legitimately; the wrapper's clean-scope
	// rules do not apply here.
	if err := os.WriteFile(filepath.Join(root, "edited.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := newRecordingStore()

	record, err := StopSession(store, root, "session-one", fixedIntents{}, fixedClock())
	if err != nil {
		t.Fatal(err)
	}
	if record == nil || record.Invalid != "ESCALATION_ARTIFACT_INVALID" {
		t.Fatalf("record = %+v", record)
	}
	if store.bytes[record.RetainedRef] == nil {
		t.Fatal("an invalid question was not retained as evidence")
	}
}

func TestStopSessionWithoutAQuestionRecordsNothing(t *testing.T) {
	root := initializedRepository(t)
	store := newRecordingStore()

	record, err := StopSession(store, root, "session-one", fixedIntents{}, fixedClock())
	if err != nil {
		t.Fatal(err)
	}
	if record != nil || store.wrote() != 0 {
		t.Fatalf("an ordinary session left a record: %+v", record)
	}
}

func TestAmbientAnnouncementCarriesNoProviderAndNoTaskHint(t *testing.T) {
	lowered := strings.ToLower(AmbientAnnouncement)
	if !strings.Contains(AmbientAnnouncement, ReservedEscalationPath) {
		t.Fatal("the announcement does not name the reserved path")
	}
	for _, required := range []string{
		"cannot be completed as specified", "goalrail.escalation/v0", "does not resume",
	} {
		if !strings.Contains(AmbientAnnouncement, required) {
			t.Fatalf("the announcement does not state %q", required)
		}
	}
	for _, forbidden := range []string{
		"codex", "claude", "openai", "anthropic", "gpt", "model", "agent", "harness",
		"conflict", "contradict", "ambiguous", "ambiguity", "disagree", "inconsistent",
		"requirement", "document", "policy", "check whether", "look for",
	} {
		if strings.Contains(lowered, forbidden) {
			t.Fatalf("the ambient announcement contains %q", forbidden)
		}
	}
}

func TestRetentionFailureIsReportedToTheCaller(t *testing.T) {
	// The caller converts this into a silent no-op toward the scaffold and a
	// record in the state root; the package itself does not swallow it.
	root := initializedRepository(t)
	writeQuestion(t, root, validQuestion)
	store := newRecordingStore()
	store.fail = true

	if _, err := StopSession(store, root, "session-one", fixedIntents{}, fixedClock()); err == nil {
		t.Fatal("a retention failure was swallowed")
	}
}

func TestQuestionRecordSerialisesWithoutThePayload(t *testing.T) {
	root := initializedRepository(t)
	writeQuestion(t, root, validQuestion)
	store := newRecordingStore()

	record, err := StopSession(store, root, "session-one", fixedIntents{}, fixedClock())
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "Which retention period governs") {
		t.Fatal("the record embedded the question instead of referencing it")
	}
}

func initializedRepository(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if _, _, err := Initialize(root, fixedClock()); err != nil {
		t.Fatal(err)
	}
	return root
}

func questionPath(root string) string {
	return filepath.Join(root, filepath.FromSlash(ReservedEscalationPath))
}

func writeQuestion(t *testing.T, root, payload string) {
	t.Helper()
	path := questionPath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestTwoIdenticalQuestionsKeepSeparateRecords(t *testing.T) {
	// Byte-identical questions from two sessions share retained bytes but not
	// identity: the second record must not collide with the first and lose its
	// attribution.
	root := initializedRepository(t)
	store := newRecordingStore()

	writeQuestion(t, root, validQuestion)
	first, err := StopSession(store, root, "session-one", fixedIntents{}, fixedClock())
	if err != nil {
		t.Fatal(err)
	}
	writeQuestion(t, root, validQuestion)
	later := func() time.Time { return time.Date(2026, 7, 29, 18, 0, 0, 0, time.UTC) }
	second, err := StopSession(store, root, "session-two", fixedIntents{}, later)
	if err != nil {
		t.Fatal(err)
	}
	if first.QuestionID == second.QuestionID {
		t.Fatal("two occurrences of the same question share one identity")
	}
	if first.Digest != second.Digest {
		t.Fatal("identical payloads produced different digests")
	}
	records := 0
	for name := range store.json {
		if strings.Contains(name, "questions/records/") {
			records++
		}
	}
	if records != 2 {
		t.Fatalf("records = %d, want one per occurrence", records)
	}
}

func TestAnUnreadableQuestionStillLeavesARecord(t *testing.T) {
	// The session tried to escalate; the attempt must leave evidence even when
	// the artifact itself cannot be retained.
	root := initializedRepository(t)
	if err := os.MkdirAll(filepath.Join(root, ".goalrail", "blocked.md"), 0o755); err != nil {
		t.Fatal(err) // a directory at the reserved path is unreadable by design
	}
	store := newRecordingStore()

	record, err := StopSession(store, root, "session-one", fixedIntents{}, fixedClock())
	if err != nil {
		t.Fatal(err)
	}
	if record == nil || record.Invalid != "ESCALATION_ARTIFACT_UNREADABLE" {
		t.Fatalf("record = %+v", record)
	}
	if record.QuestionID == "" {
		t.Fatal("an unreadable question has no identity to answer or dismiss")
	}
	if store.json[recordPath(record.QuestionID)] == nil {
		t.Fatal("the unreadable question left no persisted record")
	}
}

func TestAStaleSymlinkIsClearedWithoutFollowingIt(t *testing.T) {
	// Archival gets the same hygiene as retention: following a stale symlink
	// would copy bytes from outside the repository into Goalrail state.
	root := initializedRepository(t)
	outside := filepath.Join(t.TempDir(), "outside.md")
	if err := os.WriteFile(outside, []byte("secret outside content"), 0o644); err != nil {
		t.Fatal(err)
	}
	linkPath := questionPath(root)
	if err := os.MkdirAll(filepath.Dir(linkPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, linkPath); err != nil {
		t.Fatal(err)
	}
	store := newRecordingStore()

	archived, err := archiveStaleQuestion(store, root, fixedClock())
	if err != nil {
		t.Fatal(err)
	}
	if !archived {
		t.Fatal("a stale symlink was left in place for the new session")
	}
	if _, err := os.Lstat(linkPath); !os.IsNotExist(err) {
		t.Fatal("the stale symlink was not cleared")
	}
	for _, content := range store.bytes {
		if strings.Contains(string(content), "secret outside content") {
			t.Fatal("archival followed the symlink and copied outside bytes")
		}
	}
	// The target itself must be untouched.
	if raw, err := os.ReadFile(outside); err != nil || string(raw) != "secret outside content" {
		t.Fatal("archival disturbed the symlink target")
	}
}
