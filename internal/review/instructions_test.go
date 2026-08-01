package review

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstructionsMaterializeOnceAndNeverOverwrite(t *testing.T) {
	root := repository(t)

	first, materialized, err := EnsureInstructions(root)
	if err != nil || !materialized {
		t.Fatalf("the default was not materialized: %v (%v)", materialized, err)
	}
	if len(first) == 0 {
		t.Fatal("the materialized default is empty")
	}

	second, materializedAgain, err := EnsureInstructions(root)
	if err != nil || materializedAgain {
		t.Fatalf("materialization repeated: %v (%v)", materializedAgain, err)
	}
	if string(second) != string(first) {
		t.Fatal("the second read returned different bytes")
	}

	// An existing file is the user's, including one they emptied on purpose.
	edited := "# Mine\n\nLook only at the thing I care about.\n"
	if err := os.WriteFile(filepath.Join(root, InstructionsPath), []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	read, materializedOverEdit, err := EnsureInstructions(root)
	if err != nil || materializedOverEdit {
		t.Fatalf("an edited file was overwritten: %v (%v)", materializedOverEdit, err)
	}
	if string(read) != edited {
		t.Fatalf("the edited instructions were not used: %q", read)
	}
}

// Instructions that cannot be committed cannot be shared, and Goalrail's own
// ignore rule keeps `.goalrail/` out of version control.
func TestInstructionsAreCommittable(t *testing.T) {
	if strings.HasPrefix(InstructionsPath, ".goalrail/") {
		t.Fatalf("the instructions live under Goalrail's own ignored directory: %s", InstructionsPath)
	}

	root := repository(t)
	if _, _, err := EnsureInstructions(root); err != nil {
		t.Fatal(err)
	}
	// git itself decides whether the path is committable here, rather than a
	// pattern comparison that would drift from the rule it is imitating.
	output, err := exec.Command("git", "-C", root, "check-ignore", InstructionsPath).CombinedOutput()
	if err == nil {
		t.Fatalf("the instructions path is ignored: %s", output)
	}
	status, err := exec.Command("git", "-C", root, "status", "--porcelain").CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(status), InstructionsPath) {
		t.Fatalf("the instructions did not appear as committable content: %s", status)
	}
}

// The default carries what the findings ratchet has already promoted, rather
// than generic advice: two classes crossed the promotion threshold on the first
// day of counting, and this file is where a promotion lands.
func TestTheDefaultCarriesThePromotedClasses(t *testing.T) {
	for _, expected := range []string{"AGENTS.md", "absence", "REVIEW-VERDICT:"} {
		if !strings.Contains(defaultInstructions, expected) {
			t.Fatalf("the default instructions do not mention %q", expected)
		}
	}
}

func TestInstructionsRefuseToBeWrittenThroughASymlink(t *testing.T) {
	root := repository(t)
	outside := filepath.Join(t.TempDir(), "elsewhere.md")
	if err := os.Symlink(outside, filepath.Join(root, InstructionsPath)); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, _, err := EnsureInstructions(root); err == nil {
		t.Fatal("a symlinked instructions path was written through")
	}
	if _, statErr := os.Stat(outside); statErr == nil {
		t.Fatal("the write landed outside the repository")
	}
}
