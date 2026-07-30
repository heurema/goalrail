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
