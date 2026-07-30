package harness

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func fixedClock() func() time.Time {
	return func() time.Time { return time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC) }
}

func TestUpdateReportsAnAlreadyCurrentOverlay(t *testing.T) {
	root := t.TempDir()
	if _, err := Materialize(root, false); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	report, err := Update(UpdateInput{RepositoryRoot: root, StateRoot: t.TempDir(), Now: fixedClock()})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if !report.AlreadyCurrent {
		t.Error("a current overlay was not reported as already current")
	}
	if report.Backup != "" || len(report.Files) != 0 {
		t.Errorf("an update with nothing to do still reported work: %+v", report)
	}
	if !report.Verified {
		t.Error("an already-current overlay was not verified")
	}
}

// TestUpdateRefusesOverALocalEdit pins the rule that protects work the user never
// committed. A command whose purpose is refreshing files must not be the thing
// that destroys the only copy of an edit.
func TestUpdateRefusesOverALocalEdit(t *testing.T) {
	root, state := t.TempDir(), t.TempDir()
	if _, err := Materialize(root, false); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	target := filepath.Join(root, filepath.FromSlash(OverlayDirectory), "templates", "spec.md")
	if err := os.WriteFile(target, []byte("mine\n"), 0o644); err != nil {
		t.Fatalf("edit: %v", err)
	}

	_, err := Update(UpdateInput{RepositoryRoot: root, StateRoot: state, Now: fixedClock()})
	if !errors.Is(err, ErrLocalEdits) {
		t.Fatalf("expected a refusal over local edits, got %v", err)
	}
	if err == nil || !contains(err.Error(), "templates/spec.md") {
		t.Errorf("the refusal does not name the file: %v", err)
	}
	content, readErr := os.ReadFile(target)
	if readErr != nil {
		t.Fatalf("read: %v", readErr)
	}
	if string(content) != "mine\n" {
		t.Error("a refused update still overwrote the edit")
	}
	if entries, _ := os.ReadDir(state); len(entries) != 0 {
		t.Error("a refused update left state behind")
	}
}

func TestUpdateDiscardsOnlyWhenAskedAndSaysSo(t *testing.T) {
	root, state := t.TempDir(), t.TempDir()
	if _, err := Materialize(root, false); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	relative := OverlayDirectory + "/templates/spec.md"
	target := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.WriteFile(target, []byte("mine\n"), 0o644); err != nil {
		t.Fatalf("edit: %v", err)
	}

	report, err := Update(UpdateInput{
		RepositoryRoot:    root,
		StateRoot:         state,
		DiscardLocalEdits: true,
		Now:               fixedClock(),
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if len(report.DiscardedLocalEdits) != 1 || report.DiscardedLocalEdits[0] != relative {
		t.Errorf("the report does not record what was discarded: %+v", report.DiscardedLocalEdits)
	}
	if !report.Verified {
		t.Error("the update was not verified against the canon")
	}

	// Recovery must not lean on version control: the edit may never have been
	// committed, and the state root is where a user can still get it back.
	if report.Backup == "" {
		t.Fatal("the update kept no recoverable copy")
	}
	recovered, readErr := os.ReadFile(filepath.Join(report.Backup, filepath.FromSlash(relative)))
	if readErr != nil {
		t.Fatalf("read the backup: %v", readErr)
	}
	if string(recovered) != "mine\n" {
		t.Errorf("the backup does not hold the replaced content: %q", recovered)
	}

	state2, err := InspectOverlay(root)
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if !state2.Current {
		t.Errorf("the overlay is not current after the update: %+v", state2)
	}
}

// TestUpdateRestoresAFileFromAnEarlierCanon exercises the behind path against a
// synthetic history, which is the only way to reach it before a second canon
// ships.
func TestUpdateRestoresAFileFromAnEarlierCanon(t *testing.T) {
	root, state := t.TempDir(), t.TempDir()
	if _, err := Materialize(root, false); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	relative := OverlayDirectory + "/templates/design.md"
	older := []byte("## an earlier canon's design template\n")
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(relative)), older, 0o644); err != nil {
		t.Fatalf("write older content: %v", err)
	}
	restore := previousCanons
	previousCanons = []Canon{{ID: "sha256:older", Files: []CanonFile{{Path: relative, Digest: Digest(older)}}}}
	defer func() { previousCanons = restore }()

	report, err := Update(UpdateInput{RepositoryRoot: root, StateRoot: state, Now: fixedClock()})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if len(report.DiscardedLocalEdits) != 0 {
		t.Error("restoring a known earlier version was reported as discarding an edit")
	}
	var updated bool
	for _, file := range report.Files {
		if file.Path == relative && file.Action == ActionUpdated {
			updated = true
		}
	}
	if !updated {
		t.Errorf("the behind file was not updated: %+v", report.Files)
	}
	if !report.Verified {
		t.Error("the update was not verified")
	}
}

// TestUpdateStopsOnDriftEvenWhenSomethingIsAlsoBehind pins that a pending upgrade
// is never read as permission to overwrite.
func TestUpdateStopsOnDriftEvenWhenSomethingIsAlsoBehind(t *testing.T) {
	root, state := t.TempDir(), t.TempDir()
	if _, err := Materialize(root, false); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	behind := OverlayDirectory + "/templates/design.md"
	older := []byte("## older\n")
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(behind)), older, 0o644); err != nil {
		t.Fatalf("write older: %v", err)
	}
	edited := OverlayDirectory + "/templates/spec.md"
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(edited)), []byte("mine\n"), 0o644); err != nil {
		t.Fatalf("write edit: %v", err)
	}
	restore := previousCanons
	previousCanons = []Canon{{ID: "sha256:older", Files: []CanonFile{{Path: behind, Digest: Digest(older)}}}}
	defer func() { previousCanons = restore }()

	if _, err := Update(UpdateInput{RepositoryRoot: root, StateRoot: state, Now: fixedClock()}); !errors.Is(err, ErrLocalEdits) {
		t.Fatalf("expected the edit to stop the update, got %v", err)
	}
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(behind)))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(content) != string(older) {
		t.Error("the behind file was updated while the edit blocked the run")
	}
}

func TestUpdateSaysItDoesNotUpdateTheBinary(t *testing.T) {
	root := t.TempDir()
	if _, err := Materialize(root, false); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	report, err := Update(UpdateInput{RepositoryRoot: root, StateRoot: t.TempDir(), Now: fixedClock()})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	var says bool
	for _, note := range report.Notes {
		if contains(note, "not the gr binary") {
			says = true
		}
	}
	if !says {
		t.Errorf("the report does not say the binary is untouched: %+v", report.Notes)
	}
}

func contains(text, fragment string) bool {
	return len(text) >= len(fragment) && (func() bool {
		for index := 0; index+len(fragment) <= len(text); index++ {
			if text[index:index+len(fragment)] == fragment {
				return true
			}
		}
		return false
	})()
}

// TestUpdateConvergesWithASupersededFilePresent pins the fix for a review
// finding: folding a superseded file into the verification verdict made every
// update fail forever in a state re-materializing cannot change.
func TestUpdateConvergesWithASupersededFilePresent(t *testing.T) {
	root, state := t.TempDir(), t.TempDir()
	if _, err := Materialize(root, false); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	dropped := OverlayDirectory + "/templates/retired.md"
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(dropped)), []byte("retired\n"), 0o644); err != nil {
		t.Fatalf("write retired template: %v", err)
	}
	restore := previousCanons
	previousCanons = []Canon{{ID: "sha256:older", Files: []CanonFile{{Path: dropped, Digest: Digest([]byte("retired\n"))}}}}
	defer func() { previousCanons = restore }()

	report, err := Update(UpdateInput{RepositoryRoot: root, StateRoot: state, Now: fixedClock()})
	if err != nil {
		t.Fatalf("an update with only a superseded file present failed: %v", err)
	}
	if !report.AlreadyCurrent || !report.Verified {
		t.Fatalf("report = %+v", report)
	}
	var noted bool
	for _, note := range report.Notes {
		if contains(note, dropped) {
			noted = true
		}
	}
	if !noted {
		t.Errorf("the kept superseded file is not named: %+v", report.Notes)
	}
	// And with a behind file alongside, the update proceeds and still converges.
	behind := OverlayDirectory + "/templates/tasks.md"
	older := []byte("## older\n")
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(behind)), older, 0o644); err != nil {
		t.Fatalf("write older content: %v", err)
	}
	previousCanons = []Canon{{ID: "sha256:older", Files: []CanonFile{
		{Path: dropped, Digest: Digest([]byte("retired\n"))},
		{Path: behind, Digest: Digest(older)},
	}}}
	report, err = Update(UpdateInput{RepositoryRoot: root, StateRoot: state, Now: fixedClock()})
	if err != nil {
		t.Fatalf("update alongside a superseded file failed: %v", err)
	}
	if !report.Verified || report.AlreadyCurrent {
		t.Fatalf("report = %+v", report)
	}
}

// TestUpdateNamesTheCanonItMovedFrom pins the from/to reporting the delta
// requires: a report that says only that the repository moved leaves the user
// unable to say what it moved from.
func TestUpdateNamesTheCanonItMovedFrom(t *testing.T) {
	root, state := t.TempDir(), t.TempDir()
	if _, err := Materialize(root, false); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	relative := OverlayDirectory + "/templates/design.md"
	older := []byte("## an earlier canon's design template\n")
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(relative)), older, 0o644); err != nil {
		t.Fatalf("write older content: %v", err)
	}
	restore := previousCanons
	previousCanons = []Canon{{ID: "sha256:older", Files: []CanonFile{{Path: relative, Digest: Digest(older)}}}}
	defer func() { previousCanons = restore }()

	report, err := Update(UpdateInput{RepositoryRoot: root, StateRoot: state, Now: fixedClock()})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if len(report.MovedFrom) != 1 || report.MovedFrom[0] != "sha256:older" {
		t.Errorf("the report does not name the canon it moved from: %+v", report.MovedFrom)
	}
	if report.Canon == "" || report.Canon == "sha256:older" {
		t.Errorf("the report does not name the canon it moved to: %q", report.Canon)
	}
	// The backup carries a manifest, so a directory found months later answers
	// its own questions.
	manifest, readErr := os.ReadFile(filepath.Join(report.Backup, "manifest.json"))
	if readErr != nil {
		t.Fatalf("read manifest: %v", readErr)
	}
	for _, expected := range []string{"sha256:older", report.Canon, relative} {
		if !contains(string(manifest), expected) {
			t.Errorf("the manifest omits %q", expected)
		}
	}
}

// TestBackupDirectoriesDoNotCollide answers the external review: two updates
// inside one clock second must not share a backup directory, or the second
// silently overwrites the first's recovery point.
func TestBackupDirectoriesDoNotCollide(t *testing.T) {
	state := t.TempDir()
	makeDrifted := func() string {
		root := t.TempDir()
		if _, err := Materialize(root, false); err != nil {
			t.Fatalf("materialize: %v", err)
		}
		target := filepath.Join(root, filepath.FromSlash(OverlayDirectory), "templates", "spec.md")
		if err := os.WriteFile(target, []byte("mine\n"), 0o644); err != nil {
			t.Fatalf("edit: %v", err)
		}
		return root
	}
	first, err := Update(UpdateInput{RepositoryRoot: makeDrifted(), StateRoot: state, DiscardLocalEdits: true, Now: fixedClock()})
	if err != nil {
		t.Fatalf("first update: %v", err)
	}
	second, err := Update(UpdateInput{RepositoryRoot: makeDrifted(), StateRoot: state, DiscardLocalEdits: true, Now: fixedClock()})
	if err != nil {
		t.Fatalf("second update: %v", err)
	}
	if first.Backup == second.Backup {
		t.Fatalf("two updates in one second shared a backup directory: %s", first.Backup)
	}
	for _, backup := range []string{first.Backup, second.Backup} {
		if _, err := os.Stat(filepath.Join(backup, "manifest.json")); err != nil {
			t.Errorf("backup %s lost its manifest: %v", backup, err)
		}
	}
}

// TestUpdateKeepsTheReportWhenMaterializationFails answers the external review:
// a write failure after the backup was made must not discard the report, because
// the report is the only thing naming the backup.
func TestUpdateKeepsTheReportWhenMaterializationFails(t *testing.T) {
	root, state := t.TempDir(), t.TempDir()
	if _, err := Materialize(root, false); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	target := filepath.Join(root, filepath.FromSlash(OverlayDirectory), "templates", "spec.md")
	if err := os.WriteFile(target, []byte("mine\n"), 0o644); err != nil {
		t.Fatalf("edit: %v", err)
	}
	// A read-only file makes the overwrite fail after the backup succeeded.
	if err := os.Chmod(target, 0o400); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	defer os.Chmod(target, 0o644)

	report, err := Update(UpdateInput{RepositoryRoot: root, StateRoot: state, DiscardLocalEdits: true, Now: fixedClock()})
	if err == nil {
		t.Fatal("an unwritable overlay file did not fail the update")
	}
	if report.Backup == "" {
		t.Fatal("the failed update discarded the report carrying the backup path")
	}
	recovered, readErr := os.ReadFile(filepath.Join(report.Backup, filepath.FromSlash(OverlayDirectory), "templates", "spec.md"))
	if readErr != nil {
		t.Fatalf("the backup is unreadable: %v", readErr)
	}
	if string(recovered) != "mine\n" {
		t.Errorf("the backup does not hold the replaced content: %q", recovered)
	}
}
