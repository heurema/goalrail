package harness

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMaterializeInstallsTheOverlayIntoAFreshDirectory(t *testing.T) {
	root := t.TempDir()
	outcomes, err := Materialize(root, false)
	if err != nil {
		t.Fatalf("materialize: %v", err)
	}
	canon, err := CurrentCanon()
	if err != nil {
		t.Fatalf("current canon: %v", err)
	}
	if len(outcomes) != len(canon.Files) {
		t.Fatalf("materialized %d files, expected %d", len(outcomes), len(canon.Files))
	}
	for _, outcome := range outcomes {
		if outcome.Action != ActionCreated {
			t.Errorf("%s reported %q, expected created", outcome.Path, outcome.Action)
		}
	}
	state, err := InspectOverlay(root)
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if !state.Current || !state.Complete || state.Edited || state.Behind {
		t.Fatalf("fresh overlay is not current: %+v", state)
	}
	if len(state.Drifted()) != 0 {
		t.Errorf("fresh overlay reports drift: %+v", state.Drifted())
	}
}

func TestMaterializeIsIdempotent(t *testing.T) {
	root := t.TempDir()
	if _, err := Materialize(root, false); err != nil {
		t.Fatalf("first materialize: %v", err)
	}
	before := snapshotTree(t, root)

	outcomes, err := Materialize(root, false)
	if err != nil {
		t.Fatalf("second materialize: %v", err)
	}
	for _, outcome := range outcomes {
		if outcome.Action != ActionUnchanged {
			t.Errorf("%s reported %q on repeat, expected unchanged", outcome.Path, outcome.Action)
		}
	}
	after := snapshotTree(t, root)
	if len(before) != len(after) {
		t.Fatalf("file count changed: %d then %d", len(before), len(after))
	}
	for path, content := range before {
		if after[path] != content {
			t.Errorf("%s changed on a repeated materialization", path)
		}
	}
}

// TestMaterializeLeavesALocalEditAlone pins the rule that costs the most to get
// wrong: a command that installs files must not be the thing that destroys an
// edit that was never committed.
func TestMaterializeLeavesALocalEditAlone(t *testing.T) {
	root := t.TempDir()
	if _, err := Materialize(root, false); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	edited := filepath.Join(root, filepath.FromSlash(OverlayDirectory), "templates", "intent.md")
	if err := os.WriteFile(edited, []byte("# my own template\n"), 0o644); err != nil {
		t.Fatalf("edit template: %v", err)
	}

	outcomes, err := Materialize(root, false)
	if err != nil {
		t.Fatalf("re-materialize: %v", err)
	}
	var kept bool
	for _, outcome := range outcomes {
		if outcome.Path == OverlayDirectory+"/templates/intent.md" {
			kept = outcome.Action == ActionKept && outcome.State == FileEdited
		}
	}
	if !kept {
		t.Error("an edited template was not reported as kept")
	}
	content, err := os.ReadFile(edited)
	if err != nil {
		t.Fatalf("read edited template: %v", err)
	}
	if string(content) != "# my own template\n" {
		t.Error("materialization overwrote a local edit without being asked")
	}

	state, err := InspectOverlay(root)
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if !state.Edited || state.Current {
		t.Errorf("edited overlay reported as current: %+v", state)
	}
	drifted := state.Drifted()
	if len(drifted) != 1 || drifted[0].Path != OverlayDirectory+"/templates/intent.md" {
		t.Errorf("drift is not reported per file: %+v", drifted)
	}
}

func TestMaterializeOverwritesWhenAsked(t *testing.T) {
	root := t.TempDir()
	if _, err := Materialize(root, false); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	edited := filepath.Join(root, filepath.FromSlash(OverlayDirectory), "templates", "intent.md")
	if err := os.WriteFile(edited, []byte("# my own template\n"), 0o644); err != nil {
		t.Fatalf("edit template: %v", err)
	}

	outcomes, err := Materialize(root, true)
	if err != nil {
		t.Fatalf("materialize with overwrite: %v", err)
	}
	var updated bool
	for _, outcome := range outcomes {
		if outcome.Path == OverlayDirectory+"/templates/intent.md" {
			updated = outcome.Action == ActionUpdated
		}
	}
	if !updated {
		t.Error("an edited template was not reported as updated")
	}
	state, err := InspectOverlay(root)
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if !state.Current {
		t.Errorf("overlay is not current after an overwrite: %+v", state)
	}
}

func TestInspectReportsAMissingOverlay(t *testing.T) {
	root := t.TempDir()
	state, err := InspectOverlay(root)
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if state.Present || state.Complete || state.Current {
		t.Fatalf("empty directory reported as harnessed: %+v", state)
	}
	for _, finding := range state.Files {
		if finding.State != FileMissing {
			t.Errorf("%s reported %q in an empty directory", finding.Path, finding.State)
		}
	}
}

func TestInspectReportsAPartialOverlay(t *testing.T) {
	root := t.TempDir()
	if _, err := Materialize(root, false); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	removed := filepath.Join(root, filepath.FromSlash(OverlayDirectory), "templates", "tasks.md")
	if err := os.Remove(removed); err != nil {
		t.Fatalf("remove template: %v", err)
	}
	state, err := InspectOverlay(root)
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if !state.Present {
		t.Error("a partial overlay is not reported as present")
	}
	if state.Complete || state.Current {
		t.Errorf("a partial overlay reported as complete: %+v", state)
	}
}

// TestBehindIsDistinguishedFromEdited exercises the classification against a
// synthetic earlier canon, because this is the first canon to ship and the state
// is otherwise unreachable until a second one does.
func TestBehindIsDistinguishedFromEdited(t *testing.T) {
	root := t.TempDir()
	if _, err := Materialize(root, false); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	target := OverlayDirectory + "/templates/tasks.md"
	older := []byte("## 1. An older canon's tasks template\n\n- [ ] 1.1 Something\n")
	absolute := filepath.Join(root, filepath.FromSlash(target))
	if err := os.WriteFile(absolute, older, 0o644); err != nil {
		t.Fatalf("write older content: %v", err)
	}

	// Without a history entry the same bytes are indistinguishable from an edit.
	state, err := InspectOverlay(root)
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if !state.Edited {
		t.Fatal("unknown content is not reported as edited")
	}

	restore := previousCanons
	previousCanons = []Canon{{
		ID:    "sha256:older",
		Files: []CanonFile{{Path: target, Digest: Digest(older)}},
	}}
	defer func() { previousCanons = restore }()

	state, err = InspectOverlay(root)
	if err != nil {
		t.Fatalf("inspect with history: %v", err)
	}
	if state.Edited {
		t.Error("content from a known earlier canon is reported as edited")
	}
	if !state.Behind {
		t.Error("content from a known earlier canon is not reported as behind")
	}
	drifted := state.Drifted()
	if len(drifted) != 1 || drifted[0].State != FileBehind {
		t.Errorf("expected exactly one behind file, got %+v", drifted)
	}
}

// TestAnEditWinsOverBehindness pins that a pending upgrade is never read as
// permission to overwrite an edit.
func TestAnEditWinsOverBehindness(t *testing.T) {
	root := t.TempDir()
	if _, err := Materialize(root, false); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	behindPath := OverlayDirectory + "/templates/tasks.md"
	behindContent := []byte("## older canon\n")
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(behindPath)), behindContent, 0o644); err != nil {
		t.Fatalf("write older content: %v", err)
	}
	editedPath := OverlayDirectory + "/templates/design.md"
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(editedPath)), []byte("mine\n"), 0o644); err != nil {
		t.Fatalf("write local edit: %v", err)
	}

	restore := previousCanons
	previousCanons = []Canon{{
		ID:    "sha256:older",
		Files: []CanonFile{{Path: behindPath, Digest: Digest(behindContent)}},
	}}
	defer func() { previousCanons = restore }()

	state, err := InspectOverlay(root)
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if !state.Edited {
		t.Fatal("a local edit is not reported")
	}
	if state.Behind {
		t.Error("behind-ness is reported alongside an edit, which would license an overwrite")
	}
}

// TestASupersededFileIsReportedNotDeleted pins that overlay content the canon
// dropped is named rather than removed: deleting repository content is the user's
// act.
func TestASupersededFileIsReportedNotDeleted(t *testing.T) {
	root := t.TempDir()
	if _, err := Materialize(root, false); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	dropped := OverlayDirectory + "/templates/retired.md"
	absolute := filepath.Join(root, filepath.FromSlash(dropped))
	if err := os.WriteFile(absolute, []byte("retired\n"), 0o644); err != nil {
		t.Fatalf("write retired template: %v", err)
	}

	restore := previousCanons
	previousCanons = []Canon{{
		ID:    "sha256:older",
		Files: []CanonFile{{Path: dropped, Digest: Digest([]byte("retired\n"))}},
	}}
	defer func() { previousCanons = restore }()

	state, err := InspectOverlay(root)
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	var found bool
	for _, finding := range state.Files {
		if finding.Path == dropped {
			found = finding.State == FileSuperseded
		}
	}
	if !found {
		t.Fatalf("a superseded overlay file is not reported: %+v", state.Files)
	}
	if _, err := Materialize(root, true); err != nil {
		t.Fatalf("materialize with overwrite: %v", err)
	}
	if _, statErr := os.Stat(absolute); statErr != nil {
		t.Error("materialization deleted repository content it does not own")
	}
}

// TestMaterializeTouchesNothingOutsideTheOverlay pins that adopting the harness
// disturbs no work in flight.
func TestMaterializeTouchesNothingOutsideTheOverlay(t *testing.T) {
	root := t.TempDir()
	existing := map[string]string{
		"openspec/specs/local-run/spec.md":              "# existing promoted spec\n",
		"openspec/changes/in-flight-v0/intent.md":       "# an intent someone is writing\n",
		"openspec/changes/in-flight-v0/specs/x/spec.md": "## ADDED Requirements\n",
		"README.md": "# the project\n",
	}
	for relative, content := range existing {
		absolute := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(absolute), 0o755); err != nil {
			t.Fatalf("create %s: %v", relative, err)
		}
		if err := os.WriteFile(absolute, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", relative, err)
		}
	}

	if _, err := Materialize(root, true); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	for relative, content := range existing {
		actual, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatalf("read %s: %v", relative, err)
		}
		if string(actual) != content {
			t.Errorf("%s was modified by materialization", relative)
		}
	}
}

func snapshotTree(t *testing.T, root string) map[string]string {
	t.Helper()
	tree := make(map[string]string)
	err := filepath.Walk(root, func(walked string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		content, readErr := os.ReadFile(walked)
		if readErr != nil {
			return readErr
		}
		relative, relErr := filepath.Rel(root, walked)
		if relErr != nil {
			return relErr
		}
		tree[filepath.ToSlash(relative)] = string(content)
		return nil
	})
	if err != nil {
		t.Fatalf("snapshot %s: %v", root, err)
	}
	return tree
}

// TestMaterializeRefusesASymlinkedOverlayPath answers the external review: the
// containment the registration honours applies to every repository-scope write,
// or a checkout shipping a symlinked openspec directory would receive canonical
// content into whatever the link points at.
func TestMaterializeRefusesASymlinkedOverlayPath(t *testing.T) {
	root, elsewhere := t.TempDir(), t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "openspec"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(elsewhere, filepath.Join(root, "openspec", "schemas")); err != nil {
		t.Fatal(err)
	}
	if _, err := Materialize(root, false); err == nil {
		t.Fatal("materialization wrote through a symlinked overlay directory")
	}
	entries, err := os.ReadDir(elsewhere)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("content escaped the repository: %v", entries)
	}
}
