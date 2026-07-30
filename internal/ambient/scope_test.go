package ambient

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func gitRepository(t *testing.T, ignore string) string {
	t.Helper()
	// Isolate the machine's own configuration. A developer's global ignore file may
	// already cover the registration path — the scaffold's installer adds it there —
	// which is a happy accident for a real user and a false pass for a test that
	// means to exercise an exposed path.
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	root := t.TempDir()
	for _, arguments := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "probe@localhost"},
		{"config", "user.name", "probe"},
		// The excludes file has a built-in default path that no config variable
		// removes, so it is pointed at nothing explicitly.
		{"config", "core.excludesFile", os.DevNull},
	} {
		command := exec.Command("git", append([]string{"-C", root}, arguments...)...)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", arguments, err, output)
		}
	}
	if ignore != "" {
		if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte(ignore), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// TestRegistrationScopeFollowsTheScaffold pins the arrangement: one scaffold
// registers inside the repository, the other at user scope, and the difference is
// a property of the scaffold rather than of a caller's choice.
func TestRegistrationScopeFollowsTheScaffold(t *testing.T) {
	home, repo := t.TempDir(), t.TempDir()

	repositoryScope, err := RegistrationTarget(ScaffoldClaudeCode, home, repo)
	if err != nil {
		t.Fatal(err)
	}
	if repositoryScope.Scope != ScopeRepository {
		t.Fatalf("scope = %q, want %q", repositoryScope.Scope, ScopeRepository)
	}
	expected := filepath.Join(repo, filepath.FromSlash(RepositorySettingsPath))
	if repositoryScope.Path != expected {
		t.Fatalf("path = %q, want %q", repositoryScope.Path, expected)
	}
	// The shareable settings file beside it is never the target: a registration a
	// commit could carry would install hooks in every teammate's sessions.
	if strings.HasSuffix(repositoryScope.Path, string(filepath.Separator)+"settings.json") {
		t.Fatal("the registration targets the shareable settings file")
	}

	userScope, err := RegistrationTarget(ScaffoldCodex, home, repo)
	if err != nil {
		t.Fatal(err)
	}
	if userScope.Scope != ScopeUser {
		t.Fatalf("scope = %q, want %q", userScope.Scope, ScopeUser)
	}
	if !strings.HasPrefix(userScope.Path, home) {
		t.Fatalf("user-scope path %q is not under the home directory", userScope.Path)
	}
}

func TestConnectionRefusesTheRepositoryScopeScaffold(t *testing.T) {
	_, err := PlanConnection(ScaffoldClaudeCode, t.TempDir(), realExecutable(t))
	if err == nil {
		t.Fatal("the user-scope command planned a registration for a repository-scope scaffold")
	}
	if !strings.Contains(err.Error(), "gr init") {
		t.Fatalf("the refusal does not name the command that registers it: %v", err)
	}
}

func TestRetentionIsRegisteredAgainstASessionEndingEvent(t *testing.T) {
	// On this scaffold the stop-like event fires once per turn, so a question left
	// at the reserved path would be retained again on every turn. The event that
	// fires when a session ends is the one that carries the promoted meaning.
	events := managedEvents(ScaffoldClaudeCode)
	if len(events) != 2 || events[0] != sessionStartEvent || events[1] != sessionEndEvent {
		t.Fatalf("registered events = %v", events)
	}
	superseded := SupersededEvents(ScaffoldClaudeCode)
	if len(superseded) != 1 || superseded[0] != stopEvent {
		t.Fatalf("superseded events = %v", superseded)
	}
	// The first scaffold's stop-like event means the end of a session there, so it
	// keeps it and has nothing superseded.
	if codex := managedEvents(ScaffoldCodex); len(codex) != 2 || codex[1] != stopEvent {
		t.Fatalf("codex events = %v", codex)
	}
	if left := SupersededEvents(ScaffoldCodex); len(left) != 0 {
		t.Fatalf("codex superseded events = %v", left)
	}
}

func TestARegistrationOnThePerTurnEventIsRepaired(t *testing.T) {
	home, repo := t.TempDir(), t.TempDir()
	executable := realExecutable(t)
	target, err := RegistrationTarget(ScaffoldClaudeCode, home, repo)
	if err != nil {
		t.Fatal(err)
	}
	// A registration exactly as the earlier arrangement wrote it.
	writeFile(t, target.Path, `{
  "hooks": {
    "SessionStart": [
      {"matcher": "startup", "hooks": [{"type": "command", "command": "'`+executable+`' hook --managed-by=goalrail"}]}
    ],
    "Stop": [
      {"hooks": [{"type": "command", "command": "'`+executable+`' hook --managed-by=goalrail"}]}
    ]
  }
}`)

	plan, err := PlanRegistration(target, executable)
	if err != nil {
		t.Fatal(err)
	}
	if plan.AlreadyPresent {
		t.Fatal("a registration on the per-turn event was reported as already present")
	}
	if !plan.Repair || len(plan.SupersededPresent) != 1 || plan.SupersededPresent[0] != stopEvent {
		t.Fatalf("the plan does not name the superseded event: %+v", plan)
	}
	if _, err := Connect(plan); err != nil {
		t.Fatal(err)
	}

	settings := readJSON(t, target.Path)
	hooks := settings["hooks"].(map[string]any)
	if _, present := hooks[stopEvent]; present {
		t.Error("the per-turn registration survived the repair")
	}
	if !claudeCodeHasGoalrail(settings, sessionEndEvent) {
		t.Error("the session-ending event was not registered")
	}
	if !claudeCodeSessionStartIsScoped(settings) {
		t.Error("the repair widened the session-start occurrence")
	}
	// Exactly one handler per event: a second would fire twice and leave a removal
	// that finds only one of them.
	for _, event := range managedEvents(ScaffoldClaudeCode) {
		count := 0
		forEachManagedCommand(settings, event, func(string) { count++ })
		if count != 1 {
			t.Errorf("%s carries %d managed handlers, want one", event, count)
		}
	}
}

func TestRemovalCoversTheSupersededEvent(t *testing.T) {
	home, repo := t.TempDir(), t.TempDir()
	executable := realExecutable(t)
	target, err := RegistrationTarget(ScaffoldClaudeCode, home, repo)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, target.Path, `{
  "hooks": {
    "Stop": [
      {"hooks": [{"type": "command", "command": "'`+executable+`' hook --managed-by=goalrail"}]}
    ]
  }
}`)
	removed, err := Disconnect(ScaffoldClaudeCode, home, repo)
	if err != nil || !removed {
		t.Fatalf("disconnect removed = %v err = %v", removed, err)
	}
	if _, statErr := os.Stat(target.Path); statErr == nil {
		settings := readJSON(t, target.Path)
		if hooks, present := settings["hooks"]; present {
			t.Errorf("a registration on the superseded event survived disconnection: %v", hooks)
		}
	}
}

// TestDisconnectSpansBothScopes pins the migration path: a registration left in
// the scope this arrangement replaced still fires, so removal must reach it.
func TestDisconnectSpansBothScopes(t *testing.T) {
	home, repo := t.TempDir(), t.TempDir()
	executable := realExecutable(t)

	userScope, present := SupersededTarget(ScaffoldClaudeCode, home)
	if !present {
		t.Fatal("the scaffold reports no superseded scope")
	}
	writeFile(t, userScope.Path, `{
  "hooks": {
    "SessionStart": [
      {"matcher": "startup", "hooks": [{"type": "command", "command": "'`+executable+`' hook --managed-by=goalrail"}]}
    ]
  }
}`)
	repositoryScope, err := RegistrationTarget(ScaffoldClaudeCode, home, repo)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := PlanRegistration(repositoryScope, executable)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Connect(plan); err != nil {
		t.Fatal(err)
	}

	if _, err := Disconnect(ScaffoldClaudeCode, home, repo); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{userScope.Path, repositoryScope.Path} {
		if _, statErr := os.Stat(path); statErr == nil {
			if strings.Contains(readFile(t, path), managedMarker) {
				t.Errorf("a registration survived disconnection at %s", path)
			}
		}
	}
}

func TestDetectionReadsConfigurationPresenceNotEnvironment(t *testing.T) {
	home := t.TempDir()
	if found := DetectScaffolds(home); len(found) != 0 {
		t.Fatalf("an empty home detected %v", found)
	}
	if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	found := DetectScaffolds(home)
	if len(found) != 1 || found[0] != ScaffoldClaudeCode {
		t.Fatalf("detected %v", found)
	}
	if err := os.MkdirAll(filepath.Join(home, ".codex"), 0o755); err != nil {
		t.Fatal(err)
	}
	both := DetectScaffolds(home)
	if len(both) != 2 {
		t.Fatalf("detected %v with both present", both)
	}
	// The session environment says which agent launched gr, not which the user
	// works in; the live-run evidence in this repository shows those variables
	// being unset deliberately.
	t.Setenv("CLAUDECODE", "1")
	t.Setenv("CODEX_SESSION_ID", "abc")
	empty := t.TempDir()
	if found := DetectScaffolds(empty); len(found) != 0 {
		t.Fatalf("detection consulted the session environment: %v", found)
	}
}

func TestIgnoreStateAnswersTheQuestionItCanAnswer(t *testing.T) {
	ignored := gitRepository(t, RepositorySettingsPath+"\n")
	if state, err := IgnoreState(ignored, RepositorySettingsPath); err != nil || !state {
		t.Fatalf("an ignored path reported %v (%v)", state, err)
	}

	exposed := gitRepository(t, "")
	state, err := IgnoreState(exposed, RepositorySettingsPath)
	if state {
		t.Fatal("an unignored path was reported as ignored")
	}
	if err != nil {
		t.Fatalf("an ordinary unignored path produced an error: %v", err)
	}

	// Outside a repository nothing can commit the file, so the question does not
	// arise.
	if state, err := IgnoreState(t.TempDir(), RepositorySettingsPath); err != nil || !state {
		t.Fatalf("a directory outside version control reported %v (%v)", state, err)
	}

	// A tracked file is the dangerous case: an ignore entry would not stop a commit.
	tracked := gitRepository(t, "")
	writeFile(t, filepath.Join(tracked, filepath.FromSlash(RepositorySettingsPath)), "{}\n")
	command := exec.Command("git", "-C", tracked, "add", "-f", RepositorySettingsPath)
	if output, addErr := command.CombinedOutput(); addErr != nil {
		t.Fatalf("git add: %v\n%s", addErr, output)
	}
	state, err = IgnoreState(tracked, RepositorySettingsPath)
	if state {
		t.Fatal("a tracked path was reported as unshareable")
	}
	if err == nil || !strings.Contains(err.Error(), "already tracked") {
		t.Fatalf("the reason is not named: %v", err)
	}
}

// TestIgnoreStateRefusesWhenItCannotCheck pins the fail-closed direction: an
// unverifiable path is not treated as safe, because guessing spends someone
// else's consent.
func TestIgnoreStateRefusesWhenItCannotCheck(t *testing.T) {
	t.Setenv("PATH", "")
	state, err := IgnoreState(t.TempDir(), RepositorySettingsPath)
	if state {
		t.Fatal("an unverifiable path was reported as ignored")
	}
	if err == nil || !strings.Contains(err.Error(), "cannot be established") {
		t.Fatalf("the reason is not named: %v", err)
	}
}

func TestAddIgnoreEntriesIsAdditiveAndIdempotent(t *testing.T) {
	root := gitRepository(t, "node_modules/\n")
	added, err := AddIgnoreEntries(root, IgnoreEntries())
	if err != nil {
		t.Fatal(err)
	}
	if len(added) != len(IgnoreEntries()) {
		t.Fatalf("added %v", added)
	}
	content := readFile(t, filepath.Join(root, ".gitignore"))
	if !strings.HasPrefix(content, "node_modules/\n") {
		t.Errorf("the user's own entries were disturbed:\n%s", content)
	}
	for _, entry := range IgnoreEntries() {
		if !strings.Contains(content, entry) {
			t.Errorf("%s was not added:\n%s", entry, content)
		}
	}
	again, err := AddIgnoreEntries(root, IgnoreEntries())
	if err != nil {
		t.Fatal(err)
	}
	if len(again) != 0 {
		t.Errorf("a repeated call added %v", again)
	}
	if readFile(t, filepath.Join(root, ".gitignore")) != content {
		t.Error("a repeated call modified the file")
	}
}

// TestTheDocumentedAbsentDisclosureStatesItsOwnLimit exercises the third evidence
// state.
//
// No supported scaffold is in it today — one gate was observed to exist and the
// other observed not to — but the branch exists because a third scaffold will
// arrive with documentation and no observation, and a disclosure that quietly
// upgraded documentation to an observation is exactly the drift this discipline
// forbids. The profile is injected rather than the wording asserted in isolation,
// so the branch is reached the way production reaches it.
func TestTheDocumentedAbsentDisclosureStatesItsOwnLimit(t *testing.T) {
	const documented Scaffold = "documented-only"
	profiles[documented] = scaffoldProfile{
		scope:      ScopeRepository,
		events:     []string{sessionStartEvent, sessionEndEvent},
		trust:      TrustGateDocumentedAbsent,
		configHome: ".documented",
	}
	defer delete(profiles, documented)

	notice := strings.ToLower(ConnectionNotice(documented))
	if !strings.Contains(notice, "documents no approval") {
		t.Errorf("the notice does not say what the documentation records: %s", notice)
	}
	if !strings.Contains(notice, "not been observed") {
		t.Errorf("the notice presents documentation as observation: %s", notice)
	}
	if strings.Contains(notice, "live session confirmed") {
		t.Errorf("the notice claims an observation that was not made: %s", notice)
	}

	// And a repair on such a scaffold keeps the same limit rather than asserting a
	// mandatory second review.
	repair := strings.ToLower(RepairNotice(documented, "/old/gr"))
	if strings.Contains(repair, "no longer applies") {
		t.Errorf("the repair notice asserts a gate the documentation denies: %s", repair)
	}
}

// TestRegistrationRefusesAPathThatResolvesOutsideTheRepository closes the
// symlink route the pre-PR review flagged: a repository shipping .claude as a
// link would receive the registration into whatever the link points at,
// including the user's own configuration.
func TestRegistrationRefusesAPathThatResolvesOutsideTheRepository(t *testing.T) {
	executable := realExecutable(t)

	t.Run("linked settings directory", func(t *testing.T) {
		repo, elsewhere := t.TempDir(), t.TempDir()
		if err := os.Symlink(elsewhere, filepath.Join(repo, ".claude")); err != nil {
			t.Fatal(err)
		}
		target, err := RegistrationTarget(ScaffoldClaudeCode, t.TempDir(), repo)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := PlanRegistration(target, executable); err == nil {
			t.Fatal("a settings directory resolving outside the repository was accepted")
		} else if !strings.Contains(err.Error(), "outside the repository") {
			t.Fatalf("the refusal does not name the cause: %v", err)
		}
	})

	t.Run("linked settings file", func(t *testing.T) {
		repo, elsewhere := t.TempDir(), t.TempDir()
		if err := os.MkdirAll(filepath.Join(repo, ".claude"), 0o755); err != nil {
			t.Fatal(err)
		}
		outside := filepath.Join(elsewhere, "settings.local.json")
		writeFile(t, outside, "{}\n")
		if err := os.Symlink(outside, filepath.Join(repo, ".claude", "settings.local.json")); err != nil {
			t.Fatal(err)
		}
		target, err := RegistrationTarget(ScaffoldClaudeCode, t.TempDir(), repo)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := PlanRegistration(target, executable); err == nil {
			t.Fatal("a symlinked settings file was accepted")
		} else if !strings.Contains(err.Error(), "symbolic link") {
			t.Fatalf("the refusal does not name the cause: %v", err)
		}
	})

	t.Run("an ordinary repository is unaffected", func(t *testing.T) {
		repo := t.TempDir()
		target, err := RegistrationTarget(ScaffoldClaudeCode, t.TempDir(), repo)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := PlanRegistration(target, executable); err != nil {
			t.Fatalf("an ordinary repository was refused: %v", err)
		}
	})
}

// TestDisconnectRefusesASymlinkedRepositoryScopePath answers the external
// review: removal edits the same file registration writes, so it honours the
// same containment — a checkout shipping a symlinked settings path must not
// redirect a disconnect into the user's own configuration.
func TestDisconnectRefusesASymlinkedRepositoryScopePath(t *testing.T) {
	home, repo, elsewhere := t.TempDir(), t.TempDir(), t.TempDir()
	executable := realExecutable(t)
	outside := filepath.Join(elsewhere, "settings.local.json")
	writeFile(t, outside, `{
  "hooks": {
    "SessionEnd": [
      {"hooks": [{"type": "command", "command": "'`+executable+`' hook --managed-by=goalrail"}]}
    ]
  }
}`)
	if err := os.MkdirAll(filepath.Join(repo, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(repo, ".claude", "settings.local.json")); err != nil {
		t.Fatal(err)
	}

	before := readFile(t, outside)
	if _, err := Disconnect(ScaffoldClaudeCode, home, repo); err == nil {
		t.Fatal("disconnection followed a symlinked settings path")
	}
	if readFile(t, outside) != before {
		t.Error("the external settings file was modified through the link")
	}
}
