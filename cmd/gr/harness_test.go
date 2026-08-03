package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/heurema/goalrail/internal/ambient"
	"github.com/heurema/goalrail/internal/harness"
)

func TestUpdateReportExposesAPartialMutationWithoutABackup(t *testing.T) {
	report := harness.UpdateReport{Files: []harness.FileOutcome{
		{Path: "unchanged", Action: harness.ActionUnchanged},
		{Path: "created", Action: harness.ActionCreated},
	}}
	if !updateReportHasChanges(report) {
		t.Fatal("a partial creation would be hidden on the update error path")
	}
	if updateReportHasChanges(harness.UpdateReport{Files: []harness.FileOutcome{
		{Path: "unchanged", Action: harness.ActionUnchanged},
	}}) {
		t.Fatal("an unchanged preflight was reported as a partial mutation")
	}
}

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

// gitCommand runs one git command in a scratch repository and fails the test
// where it does not succeed.
func gitCommand(t *testing.T, root string, arguments ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", root}, arguments...)...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", arguments, err, output)
	}
}

// writeFile writes one file in a scratch repository, creating its directory.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// cloneIgnoreContent reads this clone's own ignore rule, which is empty where
// none was written.
func cloneIgnoreContent(t *testing.T, root string) string {
	t.Helper()
	path, state := ambient.CloneIgnoreTarget(root)
	if state != ambient.IgnoreTargetWritable {
		t.Fatalf("no clone ignore target in %s: state %v", root, state)
	}
	content, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return string(content)
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

// initReportShape is the part of the initialization report these tests read.
type initReportShape struct {
	Ignore          []string `json:"ignore_entries_added"`
	CloneIgnore     []string `json:"clone_ignore_entries_added"`
	CloneIgnoreFile string   `json:"clone_ignore_file"`
	Registration    struct {
		Applied   bool     `json:"applied"`
		Refused   string   `json:"refused"`
		Path      string   `json:"path"`
		Events    []string `json:"events"`
		ActiveNow bool     `json:"active_now"`
	} `json:"registration"`
	Files []struct {
		Action string `json:"action"`
	} `json:"files"`
	Notices []string `json:"notices"`
}

func decodeInit(t *testing.T, stdout string) initReportShape {
	t.Helper()
	var report initReportShape
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatal(err)
	}
	return report
}

// TestInitMakesTheRegistrationPathUnshareableItself pins the act this change
// exists for: a first run on a repository nobody prepared registers, because
// initialization makes the path unshareable with a rule that belongs to this
// clone alone — and asking the user to edit content their teammates share is not
// the price of installing something only they consented to.
func TestInitMakesTheRegistrationPathUnshareableItself(t *testing.T) {
	root := scratchRepository(t)
	t.Setenv("HOME", homeWithClaudeCode(t))
	// A repository with tracked content, so the comparison below has something to
	// be true about: against a repository with no commits, "no tracked file
	// changed" holds whatever the code does.
	writeFile(t, filepath.Join(root, ".gitignore"), "node_modules/\n")
	gitCommand(t, root, "add", ".gitignore")
	gitCommand(t, root, "commit", "-m", "a shared ignore rule that does not cover either path")
	sharedBefore := readFileContent(t, filepath.Join(root, ".gitignore"))

	stdout, _, err := runCommand(t, "init", "--repo", root)
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	report := decodeInit(t, stdout)
	if !report.Registration.Applied {
		t.Fatalf("the registration was refused on a repository nobody prepared: %s", report.Registration.Refused)
	}
	if !report.Registration.ActiveNow {
		t.Error("the registration applied but the attachment is not active")
	}
	if after := readFileContent(t, filepath.Join(root, ".gitignore")); after != sharedBefore {
		t.Errorf("the shared ignore rules were modified:\n%q\n%q", sharedBefore, after)
	}
	// Both entries, and both in the clone's own rule rather than in shared content.
	if len(report.CloneIgnore) != len(ambient.IgnoreEntries()) {
		t.Errorf("clone ignore entries added = %v, want %v", report.CloneIgnore, ambient.IgnoreEntries())
	}
	if len(report.Ignore) != 0 {
		t.Errorf("shared ignore rules were written without being asked: %v", report.Ignore)
	}
	// Nothing version control would carry to anyone else changed.
	if status := gitOutput(t, root, "status", "--porcelain"); strings.Contains(status, ".gitignore") {
		t.Errorf("a tracked ignore file was modified: %s", status)
	}

	// Re-running adds nothing and changes no byte.
	before := cloneIgnoreContent(t, root)
	stdout, _, err = runCommand(t, "init", "--repo", root)
	if err != nil {
		t.Fatalf("re-init: %v", err)
	}
	if again := decodeInit(t, stdout); len(again.CloneIgnore) != 0 {
		t.Errorf("re-running added entries that were already there: %v", again.CloneIgnore)
	}
	if after := cloneIgnoreContent(t, root); after != before {
		t.Errorf("re-running rewrote the rule:\n%q\n%q", before, after)
	}
}

// TestInitRefusesToRegisterWhereACommitCouldCarryIt pins the consent boundary:
// a registration a repository could supply would install a command in every
// teammate's sessions on one user's decision. Initialization now removes the
// ordinary cause itself, so what remains are the two it cannot: a path already
// tracked, and a rule the repository shares that overrides this clone's.
func TestInitRefusesToRegisterWhereACommitCouldCarryIt(t *testing.T) {
	t.Run("the path is already tracked", func(t *testing.T) {
		root := scratchRepository(t)
		t.Setenv("HOME", homeWithClaudeCode(t))
		writeFile(t, filepath.Join(root, ambient.RepositorySettingsPath), "{}\n")
		gitCommand(t, root, "add", "--force", ambient.RepositorySettingsPath)
		gitCommand(t, root, "commit", "-m", "track the settings path")

		report := decodeInit(t, mustInit(t, root))
		if report.Registration.Applied {
			t.Fatal("a registration was written to a path a commit could carry")
		}
		if !strings.Contains(report.Registration.Refused, "tracked") {
			t.Errorf("the refusal does not name the cause: %s", report.Registration.Refused)
		}
		// The flag cannot repair a tracked file, and must not claim it can.
		if strings.Contains(report.Registration.Refused, "re-run with --fix-gitignore") {
			t.Errorf("the refusal prescribes a remedy that cannot work: %s", report.Registration.Refused)
		}
	})

	t.Run("a shared rule overrides this clone's", func(t *testing.T) {
		root := scratchRepository(t)
		t.Setenv("HOME", homeWithClaudeCode(t))
		writeFile(t, filepath.Join(root, ".gitignore"), "!.claude/**\n")
		gitCommand(t, root, "add", ".gitignore")
		gitCommand(t, root, "commit", "-m", "share a rule that overrides this clone's")

		report := decodeInit(t, mustInit(t, root))
		if report.Registration.Applied {
			t.Fatal("a registration was written to a path a commit could carry")
		}
		for _, fragment := range []string{"not ignored", "--fix-gitignore"} {
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
		// The entry that achieved nothing was taken back out; the one that worked
		// stayed. An ineffective rule is noise the user would have to diagnose.
		rule := cloneIgnoreContent(t, root)
		if strings.Contains(rule, ambient.RepositorySettingsPath) {
			t.Errorf("an entry with no effect was left behind: %q", rule)
		}
		if !strings.Contains(rule, ".goalrail/") {
			t.Errorf("an entry that did take effect was removed: %q", rule)
		}
		if got := report.CloneIgnore; len(got) != 1 || got[0] != ".goalrail/" {
			t.Errorf("clone ignore entries reported = %v", got)
		}

		// And with the flag, the shared rules are amended and the registration
		// proceeds — the one case only shared content can answer.
		fixed := decodeInit(t, mustInit(t, root, "--fix-gitignore"))
		if len(fixed.Ignore) != 1 || fixed.Ignore[0] != ambient.RepositorySettingsPath {
			t.Errorf("shared ignore entries added = %v", fixed.Ignore)
		}
		if !fixed.Registration.Applied {
			t.Fatal("the registration did not proceed after the entries were added")
		}
		// Retention is registered against the event that fires when a session ends.
		if len(fixed.Registration.Events) != 2 || fixed.Registration.Events[1] != "SessionEnd" {
			t.Errorf("registered events = %v", fixed.Registration.Events)
		}
	})
}

// homeWithClaudeCode is a home directory in which the supported repository-scope
// scaffold is detected.
func homeWithClaudeCode(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	return home
}

func mustInit(t *testing.T, root string, arguments ...string) string {
	t.Helper()
	stdout, _, err := runCommand(t, append([]string{"init", "--repo", root}, arguments...)...)
	if err != nil {
		t.Fatalf("init %v: %v", arguments, err)
	}
	return stdout
}

func readFileContent(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func gitOutput(t *testing.T, root string, arguments ...string) string {
	t.Helper()
	output, err := exec.Command("git", append([]string{"-C", root}, arguments...)...).Output()
	if err != nil {
		t.Fatalf("git %v: %v", arguments, err)
	}
	return string(output)
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

	// The marker's exposure is now a notice only where initialization could not
	// cover it itself, which a rule the repository shares is what produces: an
	// ordinary first run covers the marker without saying anything, because there
	// is nothing left to warn about.
	t.Run("marker exposure is a notice", func(t *testing.T) {
		root, home := scratchRepository(t), t.TempDir()
		t.Setenv("HOME", home)
		writeFile(t, filepath.Join(root, ".gitignore"), "!.goalrail/\n")
		gitCommand(t, root, "add", ".gitignore")
		gitCommand(t, root, "commit", "-m", "share a rule that overrides this clone's")

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

	t.Run("an ordinary first run needs no marker notice", func(t *testing.T) {
		root, home := scratchRepository(t), t.TempDir()
		t.Setenv("HOME", home)
		stdout, _, err := runCommand(t, "init", "--repo", root)
		if err != nil {
			t.Fatalf("init: %v", err)
		}
		for _, notice := range decode(t, stdout).Notices {
			if strings.Contains(notice, ambient.MarkerPath) {
				t.Errorf("the marker was covered and still produced a notice: %s", notice)
			}
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

// TestInitValidatesTheScaffoldBeforeWriting answers the external review: a typo
// in --scaffold must produce a usage error against an untouched repository, not
// a half-installed harness followed by one.
func TestInitValidatesTheScaffoldBeforeWriting(t *testing.T) {
	root := scratchRepository(t)
	t.Setenv("HOME", t.TempDir())
	_, _, err := runCommand(t, "init", "--repo", root, "--scaffold", "clade-code")
	if err == nil {
		t.Fatal("a misspelled scaffold was accepted")
	}
	for _, path := range []string{"openspec", ".goalrail", ".claude"} {
		if _, statErr := os.Stat(filepath.Join(root, path)); statErr == nil {
			t.Errorf("%s was written before the flag was validated", path)
		}
	}
}

// TestInitInstallsWithoutAHomeDirectory answers the external review: a
// repository-scope installation must not be blocked by an unresolvable home,
// which only detection and user-scope reporting need.
func TestInitInstallsWithoutAHomeDirectory(t *testing.T) {
	root := scratchRepository(t)
	t.Setenv("HOME", "")
	stdout, _, err := runCommand(t, "init", "--repo", root, "--scaffold", "claude-code", "--fix-gitignore")
	if err != nil {
		t.Fatalf("init without a home directory failed: %v", err)
	}
	var report struct {
		Registration *struct {
			Applied bool `json:"applied"`
		} `json:"registration"`
		Files []struct{ Action string } `json:"files"`
	}
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatal(err)
	}
	if len(report.Files) == 0 {
		t.Fatal("the overlay was not installed")
	}
	if report.Registration == nil || !report.Registration.Applied {
		t.Fatalf("the repository-scope registration was not applied: %+v", report.Registration)
	}
}

// TestInitAdviceForATrackedMarkerNamesTheRealRemedy answers the external review:
// an ignore entry cannot protect a tracked file, so the notice must not
// prescribe the flag that adds one.
func TestInitAdviceForATrackedMarkerNamesTheRealRemedy(t *testing.T) {
	root := scratchRepository(t)
	t.Setenv("HOME", t.TempDir())
	if _, _, err := runCommand(t, "init", "--repo", root); err != nil {
		t.Fatalf("first init: %v", err)
	}
	add := exec.Command("git", "-C", root, "add", "-f", ".goalrail/ambient.json")
	if output, err := add.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, output)
	}
	stdout, _, err := runCommand(t, "init", "--repo", root)
	if err != nil {
		t.Fatalf("second init: %v", err)
	}
	var report struct {
		Notices []string `json:"notices"`
	}
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatal(err)
	}
	var honest bool
	for _, notice := range report.Notices {
		if strings.Contains(notice, "tracked") && !strings.Contains(notice, "--fix-gitignore") {
			honest = true
		}
		if strings.Contains(notice, ambient.MarkerPath) && strings.Contains(notice, "--fix-gitignore") {
			t.Errorf("the notice prescribes a flag that cannot protect a tracked file: %q", notice)
		}
	}
	if !honest {
		t.Errorf("no notice names the tracked-marker remedy: %+v", report.Notices)
	}
}

// TestDoctorReportWriteFailureIsNotAHarnessProblem answers the external review:
// a report that could not be written is a failed check (exit 2), not a failed
// harness (exit 1).
func TestDoctorReportWriteFailureIsNotAHarnessProblem(t *testing.T) {
	root := scratchRepository(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GOALRAIL_STATE_HOME", t.TempDir())
	err := runDoctor([]string{"--repo", root, "--json"}, failingWriter{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("a failed report write returned success")
	}
	var coded interface{ ExitCode() int }
	if !errorsAs(err, &coded) || coded.ExitCode() != 2 {
		t.Fatalf("a failed report write does not exit 2: %v", err)
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("broken pipe")
}

// TestInitReportsAWorkingAttachmentAsActiveOnRepeat answers the archival review:
// the promoted rule that a repeated consented command reports a working
// attachment as active belongs to whichever command owns the registration. For a
// repository-scope scaffold that is initialization, not connection — connection
// there writes nothing and names initialization.
func TestInitReportsAWorkingAttachmentAsActiveOnRepeat(t *testing.T) {
	root, home := scratchRepository(t), t.TempDir()
	t.Setenv("HOME", home)
	if _, _, err := runCommand(t, "init", "--repo", root, "--scaffold", "claude-code", "--fix-gitignore"); err != nil {
		t.Fatalf("first init: %v", err)
	}
	stdout, _, err := runCommand(t, "init", "--repo", root, "--scaffold", "claude-code")
	if err != nil {
		t.Fatalf("repeated init: %v", err)
	}
	var report struct {
		Registration struct {
			ActiveNow bool   `json:"active_now"`
			Applied   bool   `json:"applied"`
			Notice    string `json:"notice"`
		} `json:"registration"`
	}
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatal(err)
	}
	if !report.Registration.ActiveNow {
		t.Error("a repeated initialization did not report the working attachment as active")
	}
	if report.Registration.Applied {
		t.Error("a repeated initialization rewrote a registration that was already current")
	}
	if strings.Contains(strings.ToLower(report.Registration.Notice), "not yet active") {
		t.Errorf("a working attachment was described as inert: %q", report.Registration.Notice)
	}

	// And connection for that scaffold still writes nothing and names the command
	// that owns the registration.
	_, _, connectErr := runCommand(t, "connect", "--scaffold", "claude-code", "--yes")
	if connectErr == nil || !strings.Contains(connectErr.Error(), "gr init") {
		t.Fatalf("connection did not redirect to initialization: %v", connectErr)
	}
}

// TestInitFindsTheCloneRuleWhateverTheLayout pins that the rule is located by
// asking version control rather than by assuming a path. In a linked worktree
// and in a submodule the repository's directory is a regular file, so the
// assumed path names nothing and a joined location would silently write beside
// the repository instead of into it.
func TestInitFindsTheCloneRuleWhateverTheLayout(t *testing.T) {
	t.Run("linked worktree", func(t *testing.T) {
		root := scratchRepository(t)
		t.Setenv("HOME", homeWithClaudeCode(t))
		writeFile(t, filepath.Join(root, "README.md"), "probe\n")
		gitCommand(t, root, "add", "README.md")
		gitCommand(t, root, "commit", "-m", "a commit to branch from")

		linked := filepath.Join(t.TempDir(), "linked")
		gitCommand(t, root, "worktree", "add", "-q", "-b", "probe", linked)
		if info, err := os.Stat(filepath.Join(linked, ".git")); err != nil || info.IsDir() {
			t.Fatalf("the linked worktree does not have the layout this test exists for: %v", err)
		}

		report := decodeInit(t, mustInit(t, linked))
		if !report.Registration.Applied {
			t.Fatalf("the registration was refused in a linked worktree: %s", report.Registration.Refused)
		}
		if len(report.CloneIgnore) != len(ambient.IgnoreEntries()) {
			t.Errorf("clone ignore entries added = %v", report.CloneIgnore)
		}
	})

	t.Run("submodule", func(t *testing.T) {
		inner := scratchRepository(t)
		writeFile(t, filepath.Join(inner, "README.md"), "probe\n")
		gitCommand(t, inner, "add", "README.md")
		gitCommand(t, inner, "commit", "-m", "a commit to embed")

		outer := scratchRepository(t)
		gitCommand(t, outer, "-c", "protocol.file.allow=always", "submodule", "add", "-q", inner, "embedded")
		embedded := filepath.Join(outer, "embedded")
		if info, err := os.Stat(filepath.Join(embedded, ".git")); err != nil || info.IsDir() {
			t.Skipf("this git embeds submodules with a directory, so the layout this test exists for is absent: %v", err)
		}

		t.Setenv("HOME", homeWithClaudeCode(t))
		report := decodeInit(t, mustInit(t, embedded))
		if !report.Registration.Applied {
			t.Fatalf("the registration was refused in a submodule: %s", report.Registration.Refused)
		}
		// Written into the submodule's own rule, not beside it.
		if _, err := os.Stat(filepath.Join(embedded, ".git", "info", "exclude")); err == nil {
			t.Error("the rule was written to the assumed path, which is a file in a submodule")
		}
	})
}

// TestInitLeavesARuleFileTheUserOwnsIntact pins the guard that stops an append
// from concatenating itself onto a line the user never terminated, which would
// leave both their pattern and ours meaning nothing.
func TestInitLeavesARuleFileTheUserOwnsIntact(t *testing.T) {
	root := scratchRepository(t)
	t.Setenv("HOME", homeWithClaudeCode(t))

	rule, state := ambient.CloneIgnoreTarget(root)
	if state != ambient.IgnoreTargetWritable {
		t.Fatalf("no clone ignore target: state %v", state)
	}
	writeFile(t, rule, "# my own notes\nscratch/\nTODO.md")

	if report := decodeInit(t, mustInit(t, root)); !report.Registration.Applied {
		t.Fatalf("the registration was refused: %s", report.Registration.Refused)
	}
	content := cloneIgnoreContent(t, root)
	for _, line := range []string{"# my own notes", "scratch/", "TODO.md"} {
		if !strings.Contains(content, "\n"+line+"\n") && !strings.HasPrefix(content, line+"\n") {
			t.Errorf("the user's line %q did not survive as a line of its own:\n%q", line, content)
		}
	}
	// The user's last line still means what it meant, and so do ours.
	for _, path := range append([]string{"TODO.md"}, ambient.IgnoreEntries()...) {
		if ignored, err := ambient.IgnoreState(root, path); !ignored || err != nil {
			t.Errorf("%s is no longer ignored: %v", path, err)
		}
	}

	before := cloneIgnoreContent(t, root)
	mustInit(t, root)
	if after := cloneIgnoreContent(t, root); after != before {
		t.Errorf("re-running rewrote a file the user owns:\n%q\n%q", before, after)
	}
}

// TestInitRefusesARepositoryWithNoWorkTree pins that nothing Goalrail writes
// lands beside HEAD, objects and refs. A repository with no work tree has
// nowhere for repository content to live, so the refusal comes before the first
// write and the directory is left exactly as it was.
func TestInitRefusesARepositoryWithNoWorkTree(t *testing.T) {
	for _, arguments := range [][]string{nil, {"--fix-gitignore"}} {
		bare := filepath.Join(t.TempDir(), "bare.git")
		gitCommand(t, t.TempDir(), "init", "-q", "--bare", bare)
		before := directoryListing(t, bare)

		_, _, err := runCommand(t, append([]string{"init", "--repo", bare}, arguments...)...)
		if err == nil {
			t.Fatalf("init %v was accepted against a repository with no work tree", arguments)
		}
		if !strings.Contains(err.Error(), "no work tree") {
			t.Errorf("the refusal does not name the cause: %v", err)
		}
		if after := directoryListing(t, bare); after != before {
			t.Errorf("init %v wrote into a repository with no work tree:\n%v\n%v", arguments, before, after)
		}
	}
}

// TestInitSaysWhatBecomesTrueWhereThereIsNoRepository pins the one place the
// same exposure still arrives quietly. Nothing here can be committed, so nothing
// is refused and no rule is written — but `git init` would make both paths
// committable, and that should not be a surprise.
func TestInitSaysWhatBecomesTrueWhereThereIsNoRepository(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HOME", homeWithClaudeCode(t))

	report := decodeInit(t, mustInit(t, root))
	if !report.Registration.Applied {
		t.Fatalf("the registration was refused where nothing can be committed: %s", report.Registration.Refused)
	}
	if len(report.CloneIgnore) != 0 {
		t.Errorf("a rule was written where there is no repository: %v", report.CloneIgnore)
	}
	var told bool
	for _, notice := range report.Notices {
		if strings.Contains(notice, "git init") && strings.Contains(notice, ambient.RepositorySettingsPath) {
			told = true
		}
	}
	if !told {
		t.Errorf("the report does not say what becomes true here: %+v", report.Notices)
	}
	if len(report.Files) == 0 {
		t.Fatal("the harness was not installed")
	}
}

// directoryListing is the sorted names of a directory's entries, for asserting
// that nothing was added to it.
func directoryListing(t *testing.T, directory string) string {
	t.Helper()
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	return strings.Join(names, " ")
}

// TestInitAsksAboutTheRepositoryTheUserNamed pins that the caller's own
// repository selection cannot redirect any of this.
//
// GIT_DIR and GIT_WORK_TREE name a repository directly and override `-C`. A
// shell that exported them would otherwise have the clone's rule written into
// that repository, the check that reads the rule back consult the same one, and
// the answer come out "ignored" about a path the named repository would carry in
// a commit — hooks installed in every teammate's session on one user's consent,
// which is the guarantee this whole path exists to hold.
func TestInitAsksAboutTheRepositoryTheUserNamed(t *testing.T) {
	elsewhere := scratchRepository(t)
	root := scratchRepository(t)
	t.Setenv("HOME", homeWithClaudeCode(t))
	t.Setenv("GIT_DIR", filepath.Join(elsewhere, ".git"))
	t.Setenv("GIT_WORK_TREE", elsewhere)

	report := decodeInit(t, mustInit(t, root))
	if !report.Registration.Applied {
		t.Fatalf("the registration was refused: %s", report.Registration.Refused)
	}
	// Written where the user pointed, and nowhere else.
	if content := cloneIgnoreContent(t, root); !strings.Contains(content, ambient.RepositorySettingsPath) {
		t.Errorf("the named repository's own rule does not carry the entry: %q", content)
	}
	foreign, err := os.ReadFile(filepath.Join(elsewhere, ".git", "info", "exclude"))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	for _, entry := range ambient.IgnoreEntries() {
		if strings.Contains(string(foreign), entry) {
			t.Errorf("%s was written into the repository the environment selected", entry)
		}
	}
	// And the guarantee itself: nothing a commit could carry.
	if ignored, err := ambient.IgnoreState(root, ambient.RepositorySettingsPath); !ignored || err != nil {
		t.Errorf("the registration path is committable in the repository the user named: %v", err)
	}
}

// TestInitAnchorsTheRuleWhereItIsRead pins that a directory below the work tree
// still gets a rule that covers it. This clone's rule is matched from the top of
// the work tree, so an entry carrying a path has to carry the way back down or it
// names something that does not exist.
func TestInitAnchorsTheRuleWhereItIsRead(t *testing.T) {
	root := scratchRepository(t)
	t.Setenv("HOME", homeWithClaudeCode(t))
	nested := filepath.Join(root, "packages", "app")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	report := decodeInit(t, mustInit(t, nested))
	if !report.Registration.Applied {
		t.Fatalf("the registration was refused in a subdirectory: %s", report.Registration.Refused)
	}
	if ignored, err := ambient.IgnoreState(nested, ambient.RepositorySettingsPath); !ignored || err != nil {
		t.Errorf("the subdirectory's registration path is not ignored: %v", err)
	}
	// The remedy the change exists to avoid was not reached for.
	if _, err := os.Stat(filepath.Join(root, ".gitignore")); err == nil {
		t.Error("shared ignore rules were written for a subdirectory installation")
	}
	if status := gitOutput(t, root, "status", "--porcelain"); strings.Contains(status, ".gitignore") {
		t.Errorf("a tracked ignore file was modified: %s", status)
	}
}

// TestInitCompletesWhenTheCloneRuleCannotBeWritten pins that not having made a
// path unshareable is a condition the registration refuses, never one that
// aborts the command. An aborted command would leave the overlay and the marker
// on disk with no report at all, and no remedy named.
func TestInitCompletesWhenTheCloneRuleCannotBeWritten(t *testing.T) {
	root := scratchRepository(t)
	t.Setenv("HOME", homeWithClaudeCode(t))
	rule, state := ambient.CloneIgnoreTarget(root)
	if state != ambient.IgnoreTargetWritable {
		t.Fatalf("target state = %v", state)
	}
	if err := os.MkdirAll(filepath.Dir(rule), 0o755); err != nil {
		t.Fatal(err)
	}
	// The file exists on installations that create it from a template and not on
	// others, so both are denied: the directory to stop it being created, the
	// file to stop it being written.
	if err := os.WriteFile(rule, nil, 0o400); err != nil && !os.IsExist(err) {
		t.Fatal(err)
	}
	if err := os.Chmod(rule, 0o400); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Dir(rule), 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(filepath.Dir(rule), 0o755)
		_ = os.Chmod(rule, 0o644)
	})

	report := decodeInit(t, mustInit(t, root))
	if len(report.Files) == 0 {
		t.Fatal("the harness was not installed")
	}
	if report.Registration.Applied {
		t.Fatal("a registration was written to a path a commit could carry")
	}
	if report.Registration.Refused == "" {
		t.Error("the registration was neither applied nor refused")
	}
	// And the user is told why, rather than left to infer it from the refusal.
	var told bool
	for _, notice := range report.Notices {
		if strings.Contains(notice, "could not be written") {
			told = true
		}
	}
	if !told {
		t.Errorf("the failure to write the rule is not reported: %+v", report.Notices)
	}
}

// TestInitDoesNotFollowASymlinkedCloneRule pins that the write stays inside the
// repository. Following the link would write through to a file the user owns —
// possibly a tracked one — and the take-back path would then remove it.
func TestInitDoesNotFollowASymlinkedCloneRule(t *testing.T) {
	root := scratchRepository(t)
	t.Setenv("HOME", homeWithClaudeCode(t))
	rule, state := ambient.CloneIgnoreTarget(root)
	if state != ambient.IgnoreTargetWritable {
		t.Fatalf("target state = %v", state)
	}
	victim := filepath.Join(t.TempDir(), "shared-excludes")
	writeFile(t, victim, "# somebody else's file\n*.log\n")
	if err := os.MkdirAll(filepath.Dir(rule), 0o755); err != nil {
		t.Fatal(err)
	}
	// Some installations create this file from their own template, so the link
	// replaces whatever is there rather than assuming nothing is.
	if err := os.RemoveAll(rule); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(victim, rule); err != nil {
		t.Fatal(err)
	}

	report := decodeInit(t, mustInit(t, root))
	if len(report.CloneIgnore) != 0 {
		t.Errorf("entries were written through a symbolic link: %v", report.CloneIgnore)
	}
	if content, err := os.ReadFile(victim); err != nil || string(content) != "# somebody else's file\n*.log\n" {
		t.Errorf("the linked file was modified: %q, %v", content, err)
	}
	if report.Registration.Applied {
		t.Fatal("a registration was written to a path a commit could carry")
	}
}

// TestInitRefusesTheRepositoryDirectoryItself pins the boundary at the question
// that covers it: whether a work tree exists. A repository's own directory is
// not bare, and content written there lands beside HEAD, objects and refs just
// the same.
func TestInitRefusesTheRepositoryDirectoryItself(t *testing.T) {
	root := scratchRepository(t)
	inside := filepath.Join(root, ".git")
	before := directoryListing(t, inside)

	if _, _, err := runCommand(t, "init", "--repo", inside); err == nil {
		t.Fatal("init was accepted inside the repository's own directory")
	} else if !strings.Contains(err.Error(), "no work tree") {
		t.Errorf("the refusal does not name the cause: %v", err)
	}
	if after := directoryListing(t, inside); after != before {
		t.Errorf("init wrote into the repository's own directory:\n%v\n%v", before, after)
	}
}

// TestUpdateRefusesARepositoryWithNoWorkTree pins that the boundary is not one
// command's manners: update materializes the same overlay.
func TestUpdateRefusesARepositoryWithNoWorkTree(t *testing.T) {
	bare := filepath.Join(t.TempDir(), "bare.git")
	gitCommand(t, t.TempDir(), "init", "-q", "--bare", bare)
	before := directoryListing(t, bare)

	if _, _, err := runCommand(t, "update", "--repo", bare); err == nil {
		t.Fatal("update was accepted against a repository with no work tree")
	} else if !strings.Contains(err.Error(), "no work tree") {
		t.Errorf("the refusal does not name the cause: %v", err)
	}
	if after := directoryListing(t, bare); after != before {
		t.Errorf("update wrote into a repository with no work tree:\n%v\n%v", before, after)
	}
}

// TestInitNamesTheCauseItActuallyHas pins that the refusal reports a verified
// cause rather than the one that is usually true. Naming a shared rule in a
// repository that has none is the same class of mistake the diagnosis forbids
// everywhere else: claiming more than was verified.
func TestInitNamesTheCauseItActuallyHas(t *testing.T) {
	root := scratchRepository(t)
	t.Setenv("HOME", homeWithClaudeCode(t))
	rule, state := ambient.CloneIgnoreTarget(root)
	if state != ambient.IgnoreTargetWritable {
		t.Fatalf("target state = %v", state)
	}
	// Nothing shared decides here: the clone's own rule is what refuses to cover
	// the path, because a later line in it takes the earlier one back.
	writeFile(t, rule, ambient.RepositorySettingsPath+"\n!"+ambient.RepositorySettingsPath+"\n")

	report := decodeInit(t, mustInit(t, root))
	if report.Registration.Applied {
		t.Fatal("a registration was written to a path a commit could carry")
	}
	if strings.Contains(report.Registration.Refused, ".gitignore` overrides") {
		t.Errorf("the refusal names a shared rule this repository does not have: %s", report.Registration.Refused)
	}
	if !strings.Contains(report.Registration.Refused, "exclude") {
		t.Errorf("the refusal does not name the rule that actually decided: %s", report.Registration.Refused)
	}
}

// TestInitNamesTheFileItWroteInside pins the disclosure. A write the user cannot
// see is one they can neither audit nor undo, and the path is not one they can
// derive: it is elsewhere in a linked worktree and in a submodule.
func TestInitNamesTheFileItWroteInside(t *testing.T) {
	root := scratchRepository(t)
	t.Setenv("HOME", homeWithClaudeCode(t))
	report := decodeInit(t, mustInit(t, root))
	if report.CloneIgnoreFile == "" {
		t.Fatal("the report does not name the file it wrote")
	}
	expected, _ := ambient.CloneIgnoreTarget(root)
	if report.CloneIgnoreFile != expected {
		t.Errorf("the report names %s, the rule is at %s", report.CloneIgnoreFile, expected)
	}
	if _, err := os.Stat(report.CloneIgnoreFile); err != nil {
		t.Errorf("the report names a file that does not exist: %v", err)
	}
}
