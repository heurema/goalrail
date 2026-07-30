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

// TestInitSelectsTheScaffoldByFlagAndByDetection pins both routes, and that
// installing the harness never depends on which agent environment is present.
func TestInitSelectsTheScaffoldByFlagAndByDetection(t *testing.T) {
	type report struct {
		Files        []struct{ Action string } `json:"files"`
		Registration *struct {
			Scaffold string `json:"scaffold"`
			Applied  bool   `json:"applied"`
		} `json:"registration"`
		Next    []string `json:"next"`
		Notices []string `json:"notices"`
	}
	decode := func(t *testing.T, stdout string) report {
		t.Helper()
		var decoded report
		if err := json.Unmarshal([]byte(stdout), &decoded); err != nil {
			t.Fatal(err)
		}
		return decoded
	}

	t.Run("named explicitly", func(t *testing.T) {
		root, home := scratchRepository(t), t.TempDir()
		t.Setenv("HOME", home)
		// No scaffold configuration exists, so only the flag can select one.
		stdout, _, err := runCommand(t, "init", "--repo", root, "--scaffold", "claude-code", "--fix-gitignore")
		if err != nil {
			t.Fatalf("init: %v", err)
		}
		decoded := decode(t, stdout)
		if decoded.Registration == nil || !decoded.Registration.Applied {
			t.Fatalf("the named scaffold was not registered: %+v", decoded.Registration)
		}
		if decoded.Registration.Scaffold != "claude-code" {
			t.Errorf("registered %q", decoded.Registration.Scaffold)
		}
	})

	t.Run("nothing detected", func(t *testing.T) {
		root, home := scratchRepository(t), t.TempDir()
		t.Setenv("HOME", home)
		stdout, _, err := runCommand(t, "init", "--repo", root)
		if err != nil {
			t.Fatalf("init: %v", err)
		}
		decoded := decode(t, stdout)
		if decoded.Registration != nil {
			t.Fatalf("a registration was written with no scaffold detected: %+v", decoded.Registration)
		}
		if len(decoded.Files) == 0 {
			t.Fatal("the overlay was not installed")
		}
		var named bool
		for _, next := range decoded.Next {
			if strings.Contains(next, "--scaffold") || strings.Contains(next, "gr connect") {
				named = true
			}
		}
		if !named {
			t.Errorf("the report does not name what attaches later: %+v", decoded.Next)
		}
	})

	t.Run("user-scope scaffold names its own command", func(t *testing.T) {
		root, home := scratchRepository(t), t.TempDir()
		t.Setenv("HOME", home)
		stdout, _, err := runCommand(t, "init", "--repo", root, "--scaffold", "codex")
		if err != nil {
			t.Fatalf("init: %v", err)
		}
		decoded := decode(t, stdout)
		if decoded.Registration != nil {
			t.Fatalf("initialization registered a user-scope scaffold: %+v", decoded.Registration)
		}
		var named bool
		for _, next := range decoded.Next {
			if strings.Contains(next, "gr connect --scaffold codex") {
				named = true
			}
		}
		if !named {
			t.Errorf("the report does not name the remaining consented step: %+v", decoded.Next)
		}
	})

	t.Run("marker exposure is a notice", func(t *testing.T) {
		root, home := scratchRepository(t), t.TempDir()
		t.Setenv("HOME", home)
		stdout, _, err := runCommand(t, "init", "--repo", root)
		if err != nil {
			t.Fatalf("init: %v", err)
		}
		decoded := decode(t, stdout)
		var warned bool
		for _, notice := range decoded.Notices {
			if strings.Contains(notice, ambient.MarkerPath) && strings.Contains(notice, "--fix-gitignore") {
				warned = true
			}
		}
		if !warned {
			t.Errorf("an exposed marker produced no notice: %+v", decoded.Notices)
		}
		// And it is a notice rather than a refusal: the harness is installed.
		if len(decoded.Files) == 0 {
			t.Fatal("an exposed marker blocked the installation")
		}
	})
}

// TestDoctorSeparatesHealthyFromNotForAMachine pins the machine-readable contract:
// structured output plus an exit status a script can act on.
func TestDoctorSeparatesHealthyFromNotForAMachine(t *testing.T) {
	root, home := scratchRepository(t), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("GOALRAIL_STATE_HOME", t.TempDir())

	stdout, _, err := runCommand(t, "doctor", "--repo", root, "--json")
	if err == nil {
		t.Fatal("an unharnessed repository exited zero")
	}
	var exit interface{ ExitCode() int }
	if !errorsAs(err, &exit) || exit.ExitCode() == 0 {
		t.Fatalf("the failure carries no distinguishing exit status: %v", err)
	}
	var unharnessed struct {
		Working  bool     `json:"working"`
		Problems []string `json:"problems"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &unharnessed); jsonErr != nil {
		t.Fatalf("--json did not emit a report: %v", jsonErr)
	}
	if unharnessed.Working || len(unharnessed.Problems) == 0 {
		t.Fatalf("report = %+v", unharnessed)
	}

	if _, _, initErr := runCommand(t, "init", "--repo", root, "--scaffold", "claude-code", "--fix-gitignore"); initErr != nil {
		t.Fatalf("init: %v", initErr)
	}
	stdout, _, err = runCommand(t, "doctor", "--repo", root, "--scaffold", "claude-code", "--json")
	if err != nil {
		t.Fatalf("a harnessed repository did not exit zero: %v\n%s", err, stdout)
	}
	var harnessed struct {
		Working bool `json:"working"`
	}
	if jsonErr := json.Unmarshal([]byte(stdout), &harnessed); jsonErr != nil {
		t.Fatal(jsonErr)
	}
	if !harnessed.Working {
		t.Errorf("a harnessed repository was not reported as working:\n%s", stdout)
	}
}

func TestUpdateHelpSaysItDoesNotUpdateTheBinary(t *testing.T) {
	// The word invites the other expectation, and a user who believes they upgraded
	// Goalrail when they did not would misread every later version statement.
	_, stderr, err := runCommand(t, "update", "--help")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr, "does not update the gr binary") {
		t.Errorf("update help does not disclaim the binary: %s", stderr)
	}
	if !strings.Contains(stderr, "no network request") {
		t.Errorf("update help does not say it makes no network request: %s", stderr)
	}
}

// errorsAs is a local shim so the test does not import errors purely for one call.
func errorsAs(err error, target any) bool {
	switch typed := target.(type) {
	case *interface{ ExitCode() int }:
		coded, ok := err.(interface{ ExitCode() int })
		if ok {
			*typed = coded
		}
		return ok
	}
	return false
}
