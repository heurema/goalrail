package harness

import (
	"encoding/json"
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

func TestUpdateRefusesAnEditMadeAfterTheBehindFileWasBackedUp(t *testing.T) {
	root, stateRoot := t.TempDir(), t.TempDir()
	if _, err := Materialize(root, false); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	relative := OverlayDirectory + "/templates/design.md"
	target := filepath.Join(root, filepath.FromSlash(relative))
	older := []byte("## older\n")
	if err := os.WriteFile(target, older, 0o644); err != nil {
		t.Fatalf("write older content: %v", err)
	}
	restore := previousCanons
	previousCanons = []Canon{{ID: "sha256:older", Files: []CanonFile{{Path: relative, Digest: Digest(older)}}}}
	defer func() { previousCanons = restore }()

	late := []byte("## my late edit\n")
	report, err := update(
		UpdateInput{RepositoryRoot: root, StateRoot: stateRoot, Now: fixedClock()},
		updateTestHooks{afterBackup: func() {
			if writeErr := os.WriteFile(target, late, 0o644); writeErr != nil {
				t.Fatalf("write late edit: %v", writeErr)
			}
		}},
	)
	if !errors.Is(err, ErrLocalEdits) {
		t.Fatalf("late edit error = %v, want ErrLocalEdits", err)
	}
	if report.Backup == "" {
		t.Fatal("the pre-write refusal lost the existing recovery point")
	}
	content, readErr := os.ReadFile(target)
	if readErr != nil {
		t.Fatalf("read late edit: %v", readErr)
	}
	if string(content) != string(late) {
		t.Fatalf("late edit was overwritten: %q", content)
	}
}

func TestUpdateRefusesAnEditAtThePerFileFinalCheck(t *testing.T) {
	root, stateRoot := t.TempDir(), t.TempDir()
	if _, err := Materialize(root, false); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	relative := OverlayDirectory + "/templates/design.md"
	target := filepath.Join(root, filepath.FromSlash(relative))
	older := []byte("## older\n")
	if err := os.WriteFile(target, older, 0o644); err != nil {
		t.Fatalf("write older content: %v", err)
	}
	restore := previousCanons
	previousCanons = []Canon{{ID: "sha256:older", Files: []CanonFile{{Path: relative, Digest: Digest(older)}}}}
	defer func() { previousCanons = restore }()

	late := []byte("## edited at the final check\n")
	report, err := update(
		UpdateInput{RepositoryRoot: root, StateRoot: stateRoot, Now: fixedClock()},
		updateTestHooks{beforeReplace: func(path string) {
			if path == relative {
				if writeErr := os.WriteFile(target, late, 0o644); writeErr != nil {
					t.Fatalf("write final-check edit: %v", writeErr)
				}
			}
		}},
	)
	if !errors.Is(err, ErrLocalEdits) {
		t.Fatalf("final-check edit error = %v, want ErrLocalEdits", err)
	}
	if report.Backup == "" {
		t.Fatal("the final-check refusal lost the existing recovery point")
	}
	content, readErr := os.ReadFile(target)
	if readErr != nil {
		t.Fatalf("read final-check edit: %v", readErr)
	}
	if string(content) != string(late) {
		t.Fatalf("final-check edit was overwritten: %q", content)
	}
}

func TestUpdateRefusesAFileCreatedAfterItWasObservedMissing(t *testing.T) {
	root, stateRoot := t.TempDir(), t.TempDir()
	if _, err := Materialize(root, false); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	relative := OverlayDirectory + "/templates/tasks.md"
	target := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.Remove(target); err != nil {
		t.Fatalf("remove target: %v", err)
	}
	late := []byte("## created while update ran\n")
	_, err := update(
		UpdateInput{RepositoryRoot: root, StateRoot: stateRoot, Now: fixedClock()},
		updateTestHooks{beforeReplace: func(path string) {
			if path == relative {
				if writeErr := os.WriteFile(target, late, 0o644); writeErr != nil {
					t.Fatalf("create late file: %v", writeErr)
				}
			}
		}},
	)
	if !errors.Is(err, ErrLocalEdits) {
		t.Fatalf("late creation error = %v, want ErrLocalEdits", err)
	}
	content, readErr := os.ReadFile(target)
	if readErr != nil {
		t.Fatalf("read late file: %v", readErr)
	}
	if string(content) != string(late) {
		t.Fatalf("late file was overwritten: %q", content)
	}
}

func TestPartialMissingUpdateReportsTheEarlierCreationWithoutABackup(t *testing.T) {
	root, stateRoot := t.TempDir(), t.TempDir()
	if _, err := Materialize(root, false); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	first := OverlayDirectory + "/templates/context.md"
	second := OverlayDirectory + "/templates/tasks.md"
	for _, path := range []string{first, second} {
		if err := os.Remove(filepath.Join(root, filepath.FromSlash(path))); err != nil {
			t.Fatalf("remove %s: %v", path, err)
		}
	}
	late := []byte("## user-created tasks\n")
	report, err := update(
		UpdateInput{RepositoryRoot: root, StateRoot: stateRoot, Now: fixedClock()},
		updateTestHooks{beforeReplace: func(path string) {
			if path == second {
				if writeErr := os.WriteFile(
					filepath.Join(root, filepath.FromSlash(second)),
					late,
					0o644,
				); writeErr != nil {
					t.Fatalf("create late second file: %v", writeErr)
				}
			}
		}},
	)
	if !errors.Is(err, ErrLocalEdits) {
		t.Fatalf("partial missing error = %v, want ErrLocalEdits", err)
	}
	if report.Backup != "" {
		t.Fatalf("missing files unexpectedly produced backup %s", report.Backup)
	}
	created := false
	for _, outcome := range report.Files {
		if outcome.Path == first && outcome.Action == ActionCreated {
			created = true
		}
	}
	if !created {
		t.Fatalf("partial creation is absent from report: %+v", report.Files)
	}
	content, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(second)))
	if readErr != nil || string(content) != string(late) {
		t.Fatalf("late second file = %q, err=%v", content, readErr)
	}
}

func TestUpdateRefusesAnEditToAFileThatWasCurrentAtInspection(t *testing.T) {
	root, stateRoot := t.TempDir(), t.TempDir()
	if _, err := Materialize(root, false); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	// Keep the update active while the target below starts out current.
	missing := filepath.Join(root, filepath.FromSlash(OverlayDirectory+"/templates/proposal.md"))
	if err := os.Remove(missing); err != nil {
		t.Fatalf("remove companion file: %v", err)
	}
	relative := OverlayDirectory + "/templates/tasks.md"
	target := filepath.Join(root, filepath.FromSlash(relative))
	late := []byte("## edited after the file was observed current\n")
	_, err := update(
		UpdateInput{RepositoryRoot: root, StateRoot: stateRoot, Now: fixedClock()},
		updateTestHooks{afterBackup: func() {
			if writeErr := os.WriteFile(target, late, 0o644); writeErr != nil {
				t.Fatalf("edit current file: %v", writeErr)
			}
		}},
	)
	if !errors.Is(err, ErrLocalEdits) {
		t.Fatalf("late current-file edit error = %v, want ErrLocalEdits", err)
	}
	content, readErr := os.ReadFile(target)
	if readErr != nil {
		t.Fatalf("read late edit: %v", readErr)
	}
	if string(content) != string(late) {
		t.Fatalf("late edit was overwritten: %q", content)
	}
}

func TestExplicitDiscardBacksUpTheLatestConcurrentEdit(t *testing.T) {
	root, stateRoot := t.TempDir(), t.TempDir()
	if _, err := Materialize(root, false); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	relative := OverlayDirectory + "/templates/design.md"
	target := filepath.Join(root, filepath.FromSlash(relative))
	older := []byte("## older\n")
	if err := os.WriteFile(target, older, 0o644); err != nil {
		t.Fatalf("write older content: %v", err)
	}
	restore := previousCanons
	previousCanons = []Canon{{ID: "sha256:older", Files: []CanonFile{{Path: relative, Digest: Digest(older)}}}}
	defer func() { previousCanons = restore }()

	late := []byte("## explicitly discarded late edit\n")
	changed := false
	report, err := update(
		UpdateInput{
			RepositoryRoot:    root,
			StateRoot:         stateRoot,
			DiscardLocalEdits: true,
			Now:               fixedClock(),
		},
		updateTestHooks{beforeReplace: func(path string) {
			if path == relative && !changed {
				changed = true
				if writeErr := os.WriteFile(target, late, 0o644); writeErr != nil {
					t.Fatalf("write late edit: %v", writeErr)
				}
			}
		}},
	)
	if err != nil {
		t.Fatalf("discard late edit: %v", err)
	}
	if len(report.DiscardedLocalEdits) != 1 || report.DiscardedLocalEdits[0] != relative {
		t.Fatalf("discard report = %+v", report.DiscardedLocalEdits)
	}
	recovered, readErr := os.ReadFile(filepath.Join(report.Backup, filepath.FromSlash(relative)))
	if readErr != nil {
		t.Fatalf("read latest backup: %v", readErr)
	}
	if string(recovered) != string(late) {
		t.Fatalf("backup = %q, want latest edit %q", recovered, late)
	}
}

func TestExplicitDiscardKeepsOneRecoverySetAcrossAPartialRetry(t *testing.T) {
	root, stateRoot := t.TempDir(), t.TempDir()
	if _, err := Materialize(root, false); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	first := OverlayDirectory + "/templates/context.md"
	second := OverlayDirectory + "/templates/tasks.md"
	firstOlder := []byte("## older context\n")
	secondOlder := []byte("## older tasks\n")
	for path, content := range map[string][]byte{first: firstOlder, second: secondOlder} {
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(path)), content, 0o644); err != nil {
			t.Fatalf("write older %s: %v", path, err)
		}
	}
	restore := previousCanons
	previousCanons = []Canon{{ID: "sha256:older", Files: []CanonFile{
		{Path: first, Digest: Digest(firstOlder)},
		{Path: second, Digest: Digest(secondOlder)},
	}}}
	defer func() { previousCanons = restore }()

	secondLate := []byte("## latest tasks edit\n")
	changed := false
	report, err := update(
		UpdateInput{
			RepositoryRoot:    root,
			StateRoot:         stateRoot,
			DiscardLocalEdits: true,
			Now:               fixedClock(),
		},
		updateTestHooks{beforeReplace: func(path string) {
			if path == second && !changed {
				changed = true
				if writeErr := os.WriteFile(
					filepath.Join(root, filepath.FromSlash(second)),
					secondLate,
					0o644,
				); writeErr != nil {
					t.Fatalf("write late second edit: %v", writeErr)
				}
			}
		}},
	)
	if err != nil {
		t.Fatalf("partial retry: %v", err)
	}
	for path, want := range map[string][]byte{first: firstOlder, second: secondLate} {
		got, readErr := os.ReadFile(filepath.Join(report.Backup, filepath.FromSlash(path)))
		if readErr != nil {
			t.Fatalf("read cumulative backup %s: %v", path, readErr)
		}
		if string(got) != string(want) {
			t.Fatalf("cumulative backup %s = %q, want %q", path, got, want)
		}
	}
	backups, globErr := filepath.Glob(filepath.Join(stateRoot, filepath.FromSlash(BackupDirectory), "*"))
	if globErr != nil || len(backups) != 1 || backups[0] != report.Backup {
		t.Fatalf("recovery sets = %v, report=%s, err=%v", backups, report.Backup, globErr)
	}
	updated := make(map[string]bool)
	for _, outcome := range report.Files {
		if outcome.Action == ActionUpdated {
			updated[outcome.Path] = true
		}
	}
	if !updated[first] || !updated[second] {
		t.Fatalf("partial retry lost completed outcomes: %+v", report.Files)
	}
	if len(report.DiscardedLocalEdits) != 1 || report.DiscardedLocalEdits[0] != second {
		t.Fatalf("partial retry lost discarded edit: %+v", report.DiscardedLocalEdits)
	}
}

func TestBackupManifestDescribesTheBytesActuallyCopied(t *testing.T) {
	root, stateRoot := t.TempDir(), t.TempDir()
	if _, err := Materialize(root, false); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	relative := OverlayDirectory + "/templates/spec.md"
	target := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.WriteFile(target, []byte("first edit\n"), 0o644); err != nil {
		t.Fatalf("write first edit: %v", err)
	}
	state, err := InspectOverlay(root)
	if err != nil {
		t.Fatalf("inspect first edit: %v", err)
	}
	latest := []byte("latest edit copied by backup\n")
	if err := os.WriteFile(target, latest, 0o644); err != nil {
		t.Fatalf("write latest edit: %v", err)
	}
	directory, err := backupReplaced(root, stateRoot, state, fixedClock(), "")
	if !errors.Is(err, errOverlayChangedDuringUpdate) {
		t.Fatalf("backup stale inspection error = %v, want overlay change", err)
	}
	rawManifest, err := os.ReadFile(filepath.Join(directory, "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var manifest backupManifest
	if err := json.Unmarshal(rawManifest, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	if len(manifest.Files) != 1 || manifest.Files[0].Path != relative ||
		manifest.Files[0].Digest != Digest(latest) || manifest.Files[0].State != FileEdited {
		t.Fatalf("manifest describes stale inspection, not copied bytes: %+v", manifest.Files)
	}
}

func TestExplicitDiscardRetriesAnABADuringBackup(t *testing.T) {
	root, stateRoot := t.TempDir(), t.TempDir()
	if _, err := Materialize(root, false); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	relative := OverlayDirectory + "/templates/spec.md"
	target := filepath.Join(root, filepath.FromSlash(relative))
	first := []byte("E1 local edit\n")
	if err := os.WriteFile(target, first, 0o644); err != nil {
		t.Fatalf("write E1: %v", err)
	}
	temporary := []byte("E2 temporary edit\n")
	report, err := update(
		UpdateInput{
			RepositoryRoot:    root,
			StateRoot:         stateRoot,
			DiscardLocalEdits: true,
			Now:               fixedClock(),
		},
		updateTestHooks{
			beforeBackup: func() {
				if writeErr := os.WriteFile(target, temporary, 0o644); writeErr != nil {
					t.Fatalf("write E2: %v", writeErr)
				}
			},
			afterBackup: func() {
				if writeErr := os.WriteFile(target, first, 0o644); writeErr != nil {
					t.Fatalf("restore E1: %v", writeErr)
				}
			},
		},
	)
	if err != nil {
		t.Fatalf("ABA retry: %v", err)
	}
	recovered, readErr := os.ReadFile(filepath.Join(report.Backup, filepath.FromSlash(relative)))
	if readErr != nil {
		t.Fatalf("read ABA recovery: %v", readErr)
	}
	if string(recovered) != string(first) {
		t.Fatalf("ABA recovery = %q, want overwritten E1 %q", recovered, first)
	}
	if len(report.DiscardedLocalEdits) != 1 || report.DiscardedLocalEdits[0] != relative {
		t.Fatalf("ABA discard report = %+v", report.DiscardedLocalEdits)
	}
	backups, globErr := filepath.Glob(filepath.Join(stateRoot, filepath.FromSlash(BackupDirectory), "*"))
	if globErr != nil || len(backups) != 1 || backups[0] != report.Backup {
		t.Fatalf("ABA recovery sets = %v, report=%s, err=%v", backups, report.Backup, globErr)
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
