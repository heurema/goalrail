package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/heurema/goalrail/internal/ambient"
)

// scratchRepository is a git repository with the machine's own ignore
// configuration kept out, so what these tests exercise is the repository's state
// rather than the developer's.
func scratchRepository(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, arguments := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "probe@localhost"},
		{"config", "user.name", "probe"},
		{"config", "core.excludesFile", os.DevNull},
	} {
		command := exec.Command("git", append([]string{"-C", root}, arguments...)...)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", arguments, err, output)
		}
	}
	return root
}

func runCommand(t *testing.T, arguments ...string) (string, string, error) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	err := run(
		context.Background(),
		arguments,
		strings.NewReader(""),
		&stdout,
		&stderr,
		productionService,
	)
	return stdout.String(), stderr.String(), err
}

// TestInitRefusesToRegisterWhereACommitCouldCarryIt pins the consent boundary:
// a registration a repository could supply would install a command in every
// teammate's sessions on one user's decision.
func TestInitRefusesToRegisterWhereACommitCouldCarryIt(t *testing.T) {
	root := scratchRepository(t)
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)

	stdout, _, err := runCommand(t, "init", "--repo", root)
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	var report struct {
		Registration struct {
			Applied bool   `json:"applied"`
			Refused string `json:"refused"`
			Path    string `json:"path"`
		} `json:"registration"`
		Files []struct {
			Action string `json:"action"`
		} `json:"files"`
	}
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatal(err)
	}
	if report.Registration.Applied {
		t.Fatal("a registration was written to a path a commit could carry")
	}
	for _, fragment := range []string{"not ignored", ".gitignore", "--fix-gitignore"} {
		if !strings.Contains(report.Registration.Refused, fragment) {
			t.Errorf("the refusal omits %q: %s", fragment, report.Registration.Refused)
		}
	}
	// The rest of the harness is still installed: the refusal is about one hook,
	// not about the whole act.
	if len(report.Files) == 0 {
		t.Fatal("the overlay was not installed")
	}
	for _, file := range report.Files {
		if file.Action != "created" {
			t.Errorf("overlay file reported %q", file.Action)
		}
	}
	if _, err := os.Stat(report.Registration.Path); err == nil {
		t.Fatal("the settings file was created despite the refusal")
	}

	// And with the flag, the entries are added and the registration proceeds.
	stdout, _, err = runCommand(t, "init", "--repo", root, "--fix-gitignore")
	if err != nil {
		t.Fatalf("init --fix-gitignore: %v", err)
	}
	var fixed struct {
		Ignore       []string `json:"ignore_entries_added"`
		Registration struct {
			Applied bool     `json:"applied"`
			Events  []string `json:"events"`
		} `json:"registration"`
	}
	if err := json.Unmarshal([]byte(stdout), &fixed); err != nil {
		t.Fatal(err)
	}
	if len(fixed.Ignore) != len(ambient.IgnoreEntries()) {
		t.Errorf("ignore entries added = %v", fixed.Ignore)
	}
	if !fixed.Registration.Applied {
		t.Fatal("the registration did not proceed after the entries were added")
	}
	// Retention is registered against the event that fires when a session ends.
	if len(fixed.Registration.Events) != 2 || fixed.Registration.Events[1] != "SessionEnd" {
		t.Errorf("registered events = %v", fixed.Registration.Events)
	}
}

// TestInitNeverWritesUserLevelConfiguration pins the rule that keeps the owner's
// own configuration out of reach, including when an earlier arrangement left a
// registration there.
func TestInitNeverWritesUserLevelConfiguration(t *testing.T) {
	root := scratchRepository(t)
	home := t.TempDir()
	userSettings := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(userSettings), 0o755); err != nil {
		t.Fatal(err)
	}
	existing := `{
  "hooks": {
    "SessionStart": [
      {"matcher": "startup", "hooks": [{"type": "command", "command": "'/old/gr' hook --managed-by=goalrail"}]}
    ]
  }
}`
	if err := os.WriteFile(userSettings, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)

	if _, _, err := runCommand(t, "init", "--repo", root, "--fix-gitignore"); err != nil {
		t.Fatalf("init: %v", err)
	}
	after, err := os.ReadFile(userSettings)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != existing {
		t.Errorf("initialization modified user-level configuration:\n%s", after)
	}

	// The diagnosis names it instead, with the consented command that removes it.
	stdout, _, doctorErr := runCommand(t, "doctor", "--repo", root, "--scaffold", "claude-code", "--json")
	if doctorErr == nil {
		t.Fatal("a duplicated registration was reported as healthy")
	}
	var diagnosis struct {
		Problems    []string `json:"problems"`
		NextActions []string `json:"next_actions"`
	}
	if err := json.Unmarshal([]byte(stdout), &diagnosis); err != nil {
		t.Fatal(err)
	}
	var named bool
	for index, problem := range diagnosis.Problems {
		if strings.Contains(problem, "fires twice") &&
			strings.Contains(diagnosis.NextActions[index], "gr disconnect") {
			named = true
		}
	}
	if !named {
		t.Errorf("the diagnosis does not name the superseded scope: %+v", diagnosis)
	}
}

// TestTheHarnessCommandsNeedNoNodeRuntime pins that a machine without Node still
// gets a standing harness. The absence appears in the diagnosis as a fact.
func TestTheHarnessCommandsNeedNoNodeRuntime(t *testing.T) {
	root := scratchRepository(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("GOALRAIL_STATE_HOME", t.TempDir())

	if _, _, err := runCommand(t, "init", "--repo", root); err != nil {
		t.Fatalf("init: %v", err)
	}
	// Emptying the lookup path removes every external executable, Node included.
	// The commands must still complete.
	t.Setenv("PATH", "")
	if _, _, err := runCommand(t, "init", "--repo", root); err != nil {
		t.Fatalf("init without a lookup path: %v", err)
	}
	if _, _, err := runCommand(t, "update", "--repo", root); err != nil {
		t.Fatalf("update without a lookup path: %v", err)
	}
	stdout, _, err := runCommand(t, "doctor", "--repo", root, "--json")
	if err == nil {
		// Not fatal: whether the harness is otherwise complete is not this test's
		// question. What matters is that the report was produced.
		t.Log("doctor reported a working harness")
	}
	var diagnosis struct {
		Toolchain struct {
			NodePresent bool   `json:"node_present"`
			Note        string `json:"note"`
		} `json:"toolchain"`
		Problems []string `json:"problems"`
	}
	if err := json.Unmarshal([]byte(stdout), &diagnosis); err != nil {
		t.Fatalf("doctor produced no report without a lookup path: %v", err)
	}
	if diagnosis.Toolchain.NodePresent {
		t.Fatal("Node was reported as present with an empty lookup path")
	}
	if !strings.Contains(diagnosis.Toolchain.Note, "harness itself is unaffected") {
		t.Errorf("the note does not separate the harness from the toolchain: %q", diagnosis.Toolchain.Note)
	}
	for _, problem := range diagnosis.Problems {
		if strings.Contains(strings.ToLower(problem), "node") {
			t.Errorf("the absent runtime was reported as a problem: %q", problem)
		}
	}
}

func TestVersionReportsTheBinaryAndTheOverlay(t *testing.T) {
	stdout, _, err := runCommand(t, "version")
	if err != nil {
		t.Fatal(err)
	}
	var reported struct {
		Version string `json:"version"`
		Canon   string `json:"canon"`
	}
	if err := json.Unmarshal([]byte(stdout), &reported); err != nil {
		t.Fatal(err)
	}
	if reported.Version == "" || !strings.HasPrefix(reported.Canon, "sha256:") {
		t.Fatalf("version reported %+v", reported)
	}
}
