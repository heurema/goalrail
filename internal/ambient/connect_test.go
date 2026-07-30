package ambient

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// realExecutable is a file that actually exists and is runnable: health now
// verifies the registered binary, so a fictional path would fail for the wrong
// reason.
func realExecutable(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "gr")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestConnectionRequiresConsentAndIsPlannedFirst(t *testing.T) {
	// Editing a user's own configuration is consented to as a concrete act:
	// the plan is computed before anything is written.
	home := t.TempDir()
	plan, err := PlanConnection(ScaffoldCodex, home, realExecutable(t))
	if err != nil {
		t.Fatal(err)
	}
	if plan.AlreadyPresent {
		t.Fatal("a fresh configuration reported an existing connection")
	}
	if _, err := os.Stat(plan.ConfigPath); !os.IsNotExist(err) {
		t.Fatal("planning wrote to the scaffold configuration")
	}
}

func TestPlanRejectsARelativeExecutable(t *testing.T) {
	if _, err := PlanConnection(ScaffoldCodex, t.TempDir(), "gr"); err == nil {
		t.Fatal("a relative executable was accepted for a persistent hook")
	}
}

func TestCodexConnectionIsIdempotentAndResidueFree(t *testing.T) {
	home := t.TempDir()
	configPath := filepath.Join(home, ".codex", "config.toml")
	original := "model = \"gpt-5.6-sol\"\nsandbox_mode = \"workspace-write\"\n"
	writeFile(t, configPath, original)

	// One executable across both connections: idempotency is a claim about the
	// same binary, and a fresh path each time would exercise repair instead.
	executable := realExecutable(t)
	applyConnection(t, ScaffoldCodex, home, executable)
	after := readFile(t, configPath)
	if !strings.Contains(after, "hooks.SessionStart") || !strings.Contains(after, "hooks.Stop") {
		t.Fatalf("connection did not register both events:\n%s", after)
	}
	if !strings.HasPrefix(after, original) {
		t.Fatal("connection disturbed the user's existing configuration")
	}

	// A second connection must change nothing.
	plan, err := PlanConnection(ScaffoldCodex, home, executable)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.AlreadyPresent {
		t.Fatal("an existing connection was not detected")
	}
	changed, err := Connect(plan)
	if err != nil || changed {
		t.Fatalf("second connection changed = %v err = %v", changed, err)
	}
	if readFile(t, configPath) != after {
		t.Fatal("a repeated connection modified the configuration")
	}

	removed, err := Disconnect(ScaffoldCodex, home)
	if err != nil || !removed {
		t.Fatalf("disconnect removed = %v err = %v", removed, err)
	}
	// Residue-free: the user's file returns to exactly what it was.
	if restored := readFile(t, configPath); restored != original {
		t.Fatalf("disconnection left residue:\n%q\nwant:\n%q", restored, original)
	}
	if again, err := Disconnect(ScaffoldCodex, home); err != nil || again {
		t.Fatalf("repeated disconnect removed = %v err = %v", again, err)
	}
}

func TestClaudeCodeConnectionPreservesForeignHooks(t *testing.T) {
	home := t.TempDir()
	configPath := filepath.Join(home, ".claude", "settings.json")
	writeFile(t, configPath, `{
  "theme": "dark",
  "hooks": {
    "SessionStart": [
      {"hooks": [{"type": "command", "command": "/usr/bin/other-tool"}]}
    ]
  }
}`)

	applyConnection(t, ScaffoldClaudeCode, home, realExecutable(t))
	settings := readJSON(t, configPath)
	if settings["theme"] != "dark" {
		t.Fatal("connection disturbed unrelated settings")
	}
	hooks := settings["hooks"].(map[string]any)
	starts := hooks["SessionStart"].([]any)
	if len(starts) != 2 {
		t.Fatalf("SessionStart groups = %d, want the foreign one plus ours", len(starts))
	}
	if _, ok := hooks["Stop"]; !ok {
		t.Fatal("Stop was not registered")
	}

	removed, err := Disconnect(ScaffoldClaudeCode, home)
	if err != nil || !removed {
		t.Fatalf("disconnect removed = %v err = %v", removed, err)
	}
	settings = readJSON(t, configPath)
	hooks = settings["hooks"].(map[string]any)
	starts = hooks["SessionStart"].([]any)
	if len(starts) != 1 {
		t.Fatalf("SessionStart groups after disconnect = %d, want the foreign one", len(starts))
	}
	// Removing our own entries must never remove someone else's.
	group := starts[0].(map[string]any)
	handler := group["hooks"].([]any)[0].(map[string]any)
	if handler["command"] != "/usr/bin/other-tool" {
		t.Fatal("disconnection removed a foreign hook")
	}
	if _, ok := hooks["Stop"]; ok {
		t.Fatal("our Stop registration survived disconnection")
	}
}

func TestDisconnectRefusesAnUnterminatedManagedBlock(t *testing.T) {
	// Guessing the extent of a half-written block could delete user content.
	home := t.TempDir()
	configPath := filepath.Join(home, ".codex", "config.toml")
	writeFile(t, configPath, "model = \"x\"\n"+blockBegin+"\n[[hooks.SessionStart]]\n")
	if _, err := Disconnect(ScaffoldCodex, home); err == nil {
		t.Fatal("an unterminated managed block was removed by guesswork")
	}
}

func TestDisconnectOnAnUntouchedConfigurationDoesNothing(t *testing.T) {
	home := t.TempDir()
	for _, scaffold := range SupportedScaffolds() {
		removed, err := Disconnect(scaffold, home)
		if err != nil || removed {
			t.Fatalf("%s: removed = %v err = %v", scaffold, removed, err)
		}
	}
}

// applyConnection connects from a named executable. Naming it is not incidental:
// the helper used to mint a fresh path on every call, so a test that connected
// and then planned again compared two different binaries without meaning to.
func applyConnection(t *testing.T, scaffold Scaffold, home, executable string) {
	t.Helper()
	plan, err := PlanConnection(scaffold, home, executable)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := Connect(plan)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("connection reported no change on a fresh configuration")
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func readJSON(t *testing.T, path string) map[string]any {
	t.Helper()
	var settings map[string]any
	if err := json.Unmarshal([]byte(readFile(t, path)), &settings); err != nil {
		t.Fatal(err)
	}
	return settings
}

func TestConnectRepairsAPartialClaudeRegistration(t *testing.T) {
	// A hand-removed Stop registration must read as "not connected" so the
	// consented command can restore it; otherwise questions are never retained
	// at session stop and nothing reports why.
	home := t.TempDir()
	executable := realExecutable(t)
	applyConnection(t, ScaffoldClaudeCode, home, executable)
	configPath := filepath.Join(home, ".claude", "settings.json")
	settings := readJSON(t, configPath)
	hooks := settings["hooks"].(map[string]any)
	delete(hooks, "Stop")
	writeFile(t, configPath, marshalJSON(t, settings))

	// The same executable throughout, so the missing event is the only reason
	// the registration reads as incomplete.
	plan, err := PlanConnection(ScaffoldClaudeCode, home, executable)
	if err != nil {
		t.Fatal(err)
	}
	if plan.AlreadyPresent {
		t.Fatal("a partial registration was reported as complete")
	}
	if _, err := Connect(plan); err != nil {
		t.Fatal(err)
	}
	settings = readJSON(t, configPath)
	hooks = settings["hooks"].(map[string]any)
	if _, ok := hooks["Stop"]; !ok {
		t.Fatal("reconnection did not restore the missing Stop registration")
	}
	if starts := hooks["SessionStart"].([]any); len(starts) != 1 {
		t.Fatalf("reconnection duplicated SessionStart: %d groups", len(starts))
	}
}

func TestConnectTreatsAnEmptySettingsFileAsEmpty(t *testing.T) {
	home := t.TempDir()
	writeFile(t, filepath.Join(home, ".claude", "settings.json"), "")
	if _, err := PlanConnection(ScaffoldClaudeCode, home, realExecutable(t)); err != nil {
		t.Fatalf("an empty settings file blocked connection planning: %v", err)
	}
}

func TestDisconnectRemovesOnlyOurHandlerFromAMixedGroup(t *testing.T) {
	// A group can mix our handler with a foreign one. Removal must filter the
	// handler, not drop its containing group.
	home := t.TempDir()
	configPath := filepath.Join(home, ".claude", "settings.json")
	applyConnection(t, ScaffoldClaudeCode, home, realExecutable(t))
	settings := readJSON(t, configPath)
	hooks := settings["hooks"].(map[string]any)
	group := hooks["SessionStart"].([]any)[0].(map[string]any)
	group["hooks"] = append(group["hooks"].([]any), map[string]any{
		"type": "command", "command": "/usr/bin/foreign-tool",
	})
	writeFile(t, configPath, marshalJSON(t, settings))

	if _, err := Disconnect(ScaffoldClaudeCode, home); err != nil {
		t.Fatal(err)
	}
	settings = readJSON(t, configPath)
	hooks = settings["hooks"].(map[string]any)
	starts := hooks["SessionStart"].([]any)
	if len(starts) != 1 {
		t.Fatalf("the mixed group was dropped entirely: %v", starts)
	}
	handlers := starts[0].(map[string]any)["hooks"].([]any)
	if len(handlers) != 1 ||
		handlers[0].(map[string]any)["command"] != "/usr/bin/foreign-tool" {
		t.Fatalf("the foreign handler did not survive: %v", handlers)
	}
}

func TestDisconnectIgnoresALookalikeCommand(t *testing.T) {
	// Another tool that happens to invoke an executable named gr must not be
	// treated as ours: removal keys on the managed marker, not the name.
	home := t.TempDir()
	configPath := filepath.Join(home, ".claude", "settings.json")
	writeFile(t, configPath, `{
  "hooks": {
    "SessionStart": [
      {"hooks": [{"type": "command", "command": "/opt/other/gr hook"}]}
    ]
  }
}`)
	removed, err := Disconnect(ScaffoldClaudeCode, home)
	if err != nil {
		t.Fatal(err)
	}
	if removed {
		t.Fatal("a lookalike command was removed as if it were ours")
	}
}

func TestDisconnectRemovesAConfigurationFileConnectionCreated(t *testing.T) {
	// When connection created the file, an empty leftover is residue: the
	// filesystem must return to its pre-connection state.
	for _, scaffold := range SupportedScaffolds() {
		t.Run(string(scaffold), func(t *testing.T) {
			home := t.TempDir()
			applyConnection(t, scaffold, home, realExecutable(t))
			configPath, err := ConfigPath(scaffold, home)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(configPath); err != nil {
				t.Fatal("connection did not create the configuration")
			}
			removed, err := Disconnect(scaffold, home)
			if err != nil || !removed {
				t.Fatalf("disconnect removed = %v err = %v", removed, err)
			}
			if _, err := os.Stat(configPath); !os.IsNotExist(err) {
				t.Fatal("a configuration file created by connection was left behind")
			}
		})
	}
}

func marshalJSON(t *testing.T, value any) string {
	t.Helper()
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}

// claudeCodeSessionStartGroups returns the raw session-start group list.
func claudeCodeSessionStartGroups(t *testing.T, configPath string) []any {
	t.Helper()
	settings := readJSON(t, configPath)
	hooks, ok := settings["hooks"].(map[string]any)
	if !ok {
		t.Fatal("settings carry no hooks object")
	}
	groups, _ := hooks["SessionStart"].([]any)
	return groups
}

func TestClaudeCodeRegistersOnlyTheOpeningOccurrence(t *testing.T) {
	// An omitted matcher means every occurrence on this scaffold, which would
	// repeat the announcement on resumption, clearing, compaction, and forking.
	// The transport here has no occurrence to inspect, so the registration is
	// the only place the single-delivery rule can hold.
	home := t.TempDir()
	configPath := filepath.Join(home, ".claude", "settings.json")
	applyConnection(t, ScaffoldClaudeCode, home, realExecutable(t))

	groups := claudeCodeSessionStartGroups(t, configPath)
	if len(groups) != 1 {
		t.Fatalf("session-start groups = %d, want exactly one", len(groups))
	}
	group := groups[0].(map[string]any)
	if group["matcher"] != openingSessionMatcher {
		t.Fatalf("matcher = %v, want %q", group["matcher"], openingSessionMatcher)
	}
	for _, recurring := range []string{"resume", "clear", "compact", "fork"} {
		if group["matcher"] == recurring {
			t.Fatalf("registered against a recurring occurrence %q", recurring)
		}
	}
}

func TestClaudeCodeRepairsAnUnscopedRegistration(t *testing.T) {
	// A user who connected with an earlier version carries a registration that
	// fires on every occurrence. Treating it as "already present" would leave
	// them permanently unable to repair it.
	home := t.TempDir()
	configPath := filepath.Join(home, ".claude", "settings.json")
	writeFile(t, configPath, `{
  "hooks": {
    "SessionStart": [
      {"hooks": [{"type": "command", "command": "'/old/gr' hook --managed-by=goalrail"}]}
    ],
    "Stop": [
      {"hooks": [{"type": "command", "command": "'/old/gr' hook --managed-by=goalrail"}]}
    ]
  }
}`)

	plan, err := PlanConnection(ScaffoldClaudeCode, home, realExecutable(t))
	if err != nil {
		t.Fatal(err)
	}
	if plan.AlreadyPresent {
		t.Fatal("an unscoped registration was reported as already present")
	}
	if _, err := Connect(plan); err != nil {
		t.Fatal(err)
	}

	groups := claudeCodeSessionStartGroups(t, configPath)
	if len(groups) != 1 {
		t.Fatalf("session-start groups after repair = %d, want one", len(groups))
	}
	group := groups[0].(map[string]any)
	if group["matcher"] != openingSessionMatcher {
		t.Fatalf("repair left matcher = %v", group["matcher"])
	}
	// The unscoped session-start handler is gone, and so is the stale executable
	// path it carried — that second defect was recorded as issue #25 while this
	// test was first written, and is now repaired, so nothing named /old/gr may
	// survive anywhere in the configuration.
	for _, group := range groups {
		if strings.Contains(fmt.Sprint(group), "/old/gr") {
			t.Fatal("the stale unscoped session-start handler survived the repair")
		}
	}
	if strings.Contains(readFile(t, configPath), "/old/gr") {
		t.Fatalf("a handler still names the old executable:\n%s", readFile(t, configPath))
	}
}

func TestClaudeCodeScopedRegistrationIsIdempotent(t *testing.T) {
	home := t.TempDir()
	configPath := filepath.Join(home, ".claude", "settings.json")
	executable := realExecutable(t)
	applyConnection(t, ScaffoldClaudeCode, home, executable)
	before := readFile(t, configPath)

	plan, err := PlanConnection(ScaffoldClaudeCode, home, executable)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.AlreadyPresent {
		t.Fatal("a correctly scoped registration was not recognised")
	}
	if plan.Repair {
		t.Fatal("a registration naming the current executable was reported as stale")
	}
	changed, err := Connect(plan)
	if err != nil || changed {
		t.Fatalf("second connection changed = %v err = %v", changed, err)
	}
	if readFile(t, configPath) != before {
		t.Fatal("a repeated connection rewrote the configuration")
	}
}

func TestClaudeCodeRemovalSpansTheScopedRegistration(t *testing.T) {
	// Removal matches on the managed marker rather than position, so it must
	// widen with the registration and still leave foreign entries alone.
	home := t.TempDir()
	configPath := filepath.Join(home, ".claude", "settings.json")
	original := `{
  "theme": "dark",
  "hooks": {
    "SessionStart": [
      {"matcher": "resume", "hooks": [{"type": "command", "command": "/usr/bin/other-tool"}]}
    ]
  }
}`
	writeFile(t, configPath, original)
	applyConnection(t, ScaffoldClaudeCode, home, realExecutable(t))

	removed, err := Disconnect(ScaffoldClaudeCode, home)
	if err != nil || !removed {
		t.Fatalf("disconnect removed = %v err = %v", removed, err)
	}
	after := readJSON(t, configPath)
	if after["theme"] != "dark" {
		t.Fatal("disconnection disturbed unrelated settings")
	}
	hooks := after["hooks"].(map[string]any)
	groups, _ := hooks["SessionStart"].([]any)
	if len(groups) != 1 {
		t.Fatalf("session-start groups after disconnect = %d, want the foreign one", len(groups))
	}
	foreign := groups[0].(map[string]any)
	handler := foreign["hooks"].([]any)[0].(map[string]any)
	if handler["command"] != "/usr/bin/other-tool" {
		t.Fatal("disconnection removed a foreign handler")
	}
	if foreign["matcher"] != "resume" {
		t.Fatal("disconnection altered a foreign matcher")
	}
	if strings.Contains(readFile(t, configPath), managedMarker) {
		t.Fatal("a managed handler survived disconnection")
	}
}

// seedStaleRegistration connects from a throwaway executable and then removes
// it, producing exactly the state issue #25 describes: a registration that is
// recognisably ours and cannot run.
func seedStaleRegistration(t *testing.T, scaffold Scaffold, home string) string {
	t.Helper()
	old := realExecutable(t)
	applyConnection(t, scaffold, home, old)
	if err := os.Remove(old); err != nil {
		t.Fatal(err)
	}
	return old
}

func TestConnectRepairsAStaleExecutable(t *testing.T) {
	// Health detects a registration whose binary has moved and tells the user to
	// re-run connection. Before this, connection found its own marker, called the
	// attachment present, and wrote nothing — so the only remedy the tool offers
	// was guaranteed to do nothing.
	for _, scaffold := range SupportedScaffolds() {
		t.Run(string(scaffold), func(t *testing.T) {
			home := t.TempDir()
			configPath, err := ConfigPath(scaffold, home)
			if err != nil {
				t.Fatal(err)
			}
			old := seedStaleRegistration(t, scaffold, home)

			current := realExecutable(t)
			plan, err := PlanConnection(scaffold, home, current)
			if err != nil {
				t.Fatal(err)
			}
			if plan.AlreadyPresent {
				t.Fatal("a registration naming a moved executable was reported as present")
			}
			if !plan.Repair {
				t.Fatal("the plan did not report a repair")
			}
			if plan.RegisteredExecutable != old {
				t.Fatalf("plan named %q as the stale executable, want %q",
					plan.RegisteredExecutable, old)
			}

			changed, err := Connect(plan)
			if err != nil || !changed {
				t.Fatalf("repair changed = %v err = %v", changed, err)
			}

			after := readFile(t, configPath)
			if strings.Contains(after, old) {
				t.Fatalf("the stale executable survived the repair:\n%s", after)
			}
			if !strings.Contains(after, current) {
				t.Fatalf("the repair did not register the current executable:\n%s", after)
			}
			// Replaced, not accompanied: a second registration would fire every
			// hook twice and leave a removal that finds only one of them.
			if count := strings.Count(after, managedMarker); count != len(managedEvents()) {
				t.Fatalf("managed handlers = %d, want %d:\n%s",
					count, len(managedEvents()), after)
			}
		})
	}
}

func TestRepairIsNotTriggeredForTheCurrentExecutable(t *testing.T) {
	// The repeat that must stay free. Rewriting a registration that already names
	// the current binary would cost the user a review step for nothing.
	for _, scaffold := range SupportedScaffolds() {
		t.Run(string(scaffold), func(t *testing.T) {
			home := t.TempDir()
			configPath, err := ConfigPath(scaffold, home)
			if err != nil {
				t.Fatal(err)
			}
			executable := realExecutable(t)
			applyConnection(t, scaffold, home, executable)
			before := readFile(t, configPath)

			plan, err := PlanConnection(scaffold, home, executable)
			if err != nil {
				t.Fatal(err)
			}
			if plan.Repair || !plan.AlreadyPresent {
				t.Fatalf("plan = %+v on an unchanged registration", plan)
			}
			if _, err := Connect(plan); err != nil {
				t.Fatal(err)
			}
			if readFile(t, configPath) != before {
				t.Fatal("a repeated connection rewrote a registration that was already current")
			}
		})
	}
}

func TestRepairPreservesAForeignHandlerForTheSameEvent(t *testing.T) {
	// A repair may replace only what the connection added. Dropping a foreign
	// handler would silently disable another tool.
	home := t.TempDir()
	configPath := filepath.Join(home, ".claude", "settings.json")
	writeFile(t, configPath, `{
  "hooks": {
    "SessionStart": [
      {"matcher": "resume", "hooks": [{"type": "command", "command": "/usr/bin/other-tool"}]},
      {"matcher": "startup", "hooks": [{"type": "command", "command": "'/old/gr' hook --managed-by=goalrail"}]}
    ],
    "Stop": [
      {"hooks": [{"type": "command", "command": "'/old/gr' hook --managed-by=goalrail"}]}
    ]
  }
}`)

	current := realExecutable(t)
	plan, err := PlanConnection(ScaffoldClaudeCode, home, current)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Repair {
		t.Fatal("a stale registration beside a foreign handler was not reported as a repair")
	}
	if _, err := Connect(plan); err != nil {
		t.Fatal(err)
	}

	groups := claudeCodeSessionStartGroups(t, configPath)
	foreign := 0
	for _, group := range groups {
		asMap := group.(map[string]any)
		handler := asMap["hooks"].([]any)[0].(map[string]any)
		if handler["command"] == "/usr/bin/other-tool" {
			foreign++
			if asMap["matcher"] != "resume" {
				t.Fatalf("the repair altered a foreign occurrence: %v", asMap["matcher"])
			}
		}
	}
	if foreign != 1 {
		t.Fatalf("the foreign handler did not survive the repair: %v", groups)
	}
	if strings.Contains(readFile(t, configPath), "/old/gr") {
		t.Fatal("the stale handler survived the repair")
	}
}

func TestRepairLeavesAnEventThatIsAlreadyCurrent(t *testing.T) {
	// Reconciliation is per event. An event that is already correct must keep its
	// exact bytes, and with them whatever review the user has given it — the
	// sentinel key below is something our writer never produces, so its survival
	// proves the event was not rewritten.
	home := t.TempDir()
	configPath := filepath.Join(home, ".claude", "settings.json")
	current := realExecutable(t)
	writeFile(t, configPath, `{
  "hooks": {
    "SessionStart": [
      {"matcher": "startup", "hooks": [{"type": "command", "command": "'/old/gr' hook --managed-by=goalrail"}]}
    ],
    "Stop": [
      {"sentinel": "untouched", "hooks": [{"type": "command", "command": "'`+current+`' hook --managed-by=goalrail"}]}
    ]
  }
}`)

	plan, err := PlanConnection(ScaffoldClaudeCode, home, current)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Repair {
		t.Fatal("a stale session-start registration was not reported as a repair")
	}
	if _, err := Connect(plan); err != nil {
		t.Fatal(err)
	}

	settings := readJSON(t, configPath)
	hooks := settings["hooks"].(map[string]any)
	stops := hooks["Stop"].([]any)
	if len(stops) != 1 {
		t.Fatalf("the current stop registration was duplicated: %v", stops)
	}
	if stops[0].(map[string]any)["sentinel"] != "untouched" {
		t.Fatalf("the stop event was rewritten although it was already current: %v", stops[0])
	}
	if strings.Contains(readFile(t, configPath), "/old/gr") {
		t.Fatal("the stale session-start handler survived the repair")
	}
}

func TestRepairKeepsSessionStartScoped(t *testing.T) {
	// A registration can be both stale and unscoped — an earlier version, an
	// older binary. The repair must fix both without widening the occurrence.
	home := t.TempDir()
	configPath := filepath.Join(home, ".claude", "settings.json")
	writeFile(t, configPath, `{
  "hooks": {
    "SessionStart": [
      {"hooks": [{"type": "command", "command": "'/old/gr' hook --managed-by=goalrail"}]}
    ],
    "Stop": [
      {"hooks": [{"type": "command", "command": "'/old/gr' hook --managed-by=goalrail"}]}
    ]
  }
}`)

	current := realExecutable(t)
	plan, err := PlanConnection(ScaffoldClaudeCode, home, current)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Connect(plan); err != nil {
		t.Fatal(err)
	}

	groups := claudeCodeSessionStartGroups(t, configPath)
	if len(groups) != 1 {
		t.Fatalf("session-start groups after repair = %d, want one", len(groups))
	}
	if matcher := groups[0].(map[string]any)["matcher"]; matcher != openingSessionMatcher {
		t.Fatalf("the repair left matcher = %v, want %q", matcher, openingSessionMatcher)
	}
	if strings.Contains(readFile(t, configPath), "/old/gr") {
		t.Fatal("the stale handler survived the repair")
	}
}

func TestRepairRefusesAnUnterminatedManagedBlock(t *testing.T) {
	// The repair writes by removing first, and removal refuses to guess the
	// extent of a half-written block rather than risk deleting user content. A
	// reported error beats the silent no-op this state produced before.
	home := t.TempDir()
	configPath := filepath.Join(home, ".codex", "config.toml")
	writeFile(t, configPath, "model = \"x\"\n"+blockBegin+"\n"+
		"[[hooks.SessionStart.hooks]]\ncommand = \"'/old/gr' hook "+managedMarker+"\"\n"+
		"[[hooks.Stop.hooks]]\ncommand = \"'/old/gr' hook "+managedMarker+"\"\n")

	plan, err := PlanConnection(ScaffoldCodex, home, realExecutable(t))
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Repair {
		t.Fatal("a stale registration in an unterminated block was not reported as a repair")
	}
	if _, err := Connect(plan); err == nil {
		t.Fatal("the repair wrote beside a block whose extent is unknown")
	}
}

func TestRepairIgnoresALookalikeCommand(t *testing.T) {
	// Another tool invoking an executable named gr must not make our attachment
	// look stale: staleness is read from managed handlers only.
	home := t.TempDir()
	configPath := filepath.Join(home, ".claude", "settings.json")
	writeFile(t, configPath, `{
  "hooks": {
    "SessionStart": [
      {"hooks": [{"type": "command", "command": "'/opt/other/gr' hook"}]}
    ]
  }
}`)
	plan, err := PlanConnection(ScaffoldClaudeCode, home, realExecutable(t))
	if err != nil {
		t.Fatal(err)
	}
	if plan.Repair || plan.RegisteredExecutable != "" {
		t.Fatalf("a foreign command was read as our stale registration: %+v", plan)
	}
}

func TestConnectRepairsAPartialCodexBlockWithoutDuplicatingIt(t *testing.T) {
	// A managed block missing one of its stanzas already reads as not connected,
	// which is what health reports. Writing beside it would leave two blocks —
	// every hook firing twice — and a removal that finds only the first, so
	// disconnection would leave residue that still looks like an attachment.
	// Removing before writing closes that path as well as the stale one.
	home := t.TempDir()
	configPath := filepath.Join(home, ".codex", "config.toml")
	// Existing user content, so the repair's effect on it is observable too.
	original := "model = \"gpt-5.6-sol\"\n"
	writeFile(t, configPath, original)
	executable := realExecutable(t)
	applyConnection(t, ScaffoldCodex, home, executable)
	writeFile(t, configPath,
		strings.Replace(readFile(t, configPath), "[[hooks.Stop.hooks]]", "", 1))

	plan, err := PlanConnection(ScaffoldCodex, home, executable)
	if err != nil {
		t.Fatal(err)
	}
	if plan.AlreadyPresent {
		t.Fatal("a block missing a stanza was reported as a complete registration")
	}
	if _, err := Connect(plan); err != nil {
		t.Fatal(err)
	}

	after := readFile(t, configPath)
	if count := strings.Count(after, blockBegin); count != 1 {
		t.Fatalf("managed blocks after the repair = %d, want one:\n%s", count, after)
	}
	if count := strings.Count(after, managedMarker); count != len(managedEvents()) {
		t.Fatalf("managed handlers = %d, want %d:\n%s", count, len(managedEvents()), after)
	}
	if !strings.HasPrefix(after, original) {
		t.Fatalf("the repair disturbed the user's own configuration:\n%s", after)
	}
	// Removal must still find the whole thing, and leave the user's file as it was.
	removed, err := Disconnect(ScaffoldCodex, home)
	if err != nil || !removed {
		t.Fatalf("disconnect removed = %v err = %v", removed, err)
	}
	if restored := readFile(t, configPath); restored != original {
		t.Fatalf("disconnection after a repair left residue:\n%q\nwant:\n%q", restored, original)
	}
}

func TestRegistrationSurvivesAnApostropheInThePath(t *testing.T) {
	// shellQuote encodes an apostrophe in the path as '"'"'. Reading the text
	// between the last two apostrophes then lands inside that sequence and
	// returns only the tail, so a user whose home directory holds an apostrophe
	// would have every connection call their current registration stale, rewrite
	// it, and invalidate the review they had just given.
	for _, scaffold := range SupportedScaffolds() {
		t.Run(string(scaffold), func(t *testing.T) {
			home := t.TempDir()
			configPath, err := ConfigPath(scaffold, home)
			if err != nil {
				t.Fatal(err)
			}
			awkward := filepath.Join(t.TempDir(), "o'brien")
			if err := os.MkdirAll(awkward, 0o755); err != nil {
				t.Fatal(err)
			}
			executable := filepath.Join(awkward, "gr")
			if err := os.WriteFile(executable, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
				t.Fatal(err)
			}

			applyConnection(t, scaffold, home, executable)
			before := readFile(t, configPath)

			plan, err := PlanConnection(scaffold, home, executable)
			if err != nil {
				t.Fatal(err)
			}
			if plan.Repair {
				t.Fatalf("a registration naming %q was read as stale (%q)",
					executable, plan.RegisteredExecutable)
			}
			if !plan.AlreadyPresent {
				t.Fatalf("plan = %+v for an unchanged registration", plan)
			}
			if _, err := Connect(plan); err != nil {
				t.Fatal(err)
			}
			if readFile(t, configPath) != before {
				t.Fatal("a repeated connection rewrote a registration it should have kept")
			}
		})
	}
}

func TestStalenessIgnoresACommandOutsideTheManagedBlock(t *testing.T) {
	// A hand-kept copy of an older stanza — a backup, or residue from an earlier
	// corrupt state — sits outside the markers and is not a registration this
	// command owns. Counting it would report every repeated connection as a
	// repair and demand a review each time, while rewriting the block leaves that
	// stanza exactly where it was, so the loop would never converge.
	//
	// A commented-out copy is filtered earlier, by reading the file's own
	// assignments rather than its text; this fixture is uncommented so the
	// scoping to the managed block is what the assertion rests on.
	home := t.TempDir()
	configPath := filepath.Join(home, ".codex", "config.toml")
	executable := realExecutable(t)
	applyConnection(t, ScaffoldCodex, home, executable)
	writeFile(t, configPath, readFile(t, configPath)+
		"\n[[hooks.SessionStart.hooks]]\ntype = \"command\"\n"+
		"command = \"'/old/gr' hook "+managedMarker+"\"\n")
	before := readFile(t, configPath)

	plan, err := PlanConnection(ScaffoldCodex, home, executable)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Repair {
		t.Fatalf("a stanza outside the block was read as ours, naming %q",
			plan.RegisteredExecutable)
	}
	if _, err := Connect(plan); err != nil {
		t.Fatal(err)
	}
	if readFile(t, configPath) != before {
		t.Fatal("a repeated connection rewrote the configuration over a foreign stanza")
	}
}

func TestStalenessIgnoresACommentedOutCommand(t *testing.T) {
	// Reading the file's assignments rather than scanning it as text: a commented
	// copy of an old command carries the marker but registers nothing.
	home := t.TempDir()
	configPath := filepath.Join(home, ".codex", "config.toml")
	executable := realExecutable(t)
	applyConnection(t, ScaffoldCodex, home, executable)
	writeFile(t, configPath, readFile(t, configPath)+
		"\n# command = \"'/old/gr' hook "+managedMarker+"\"\n")

	plan, err := PlanConnection(ScaffoldCodex, home, executable)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Repair {
		t.Fatalf("a commented command was read as a registration naming %q",
			plan.RegisteredExecutable)
	}
}

func TestRepairRemovesEveryManagedBlock(t *testing.T) {
	// Reachable from the write path this change replaces: a partial block caused
	// reconnection to append a complete one, leaving two. Removal handles one
	// block per call, so repairing by removing once and appending would leave the
	// duplication exactly as it was — hooks still firing twice, and a later
	// disconnect leaving a registration behind.
	home := t.TempDir()
	configPath := filepath.Join(home, ".codex", "config.toml")
	original := "model = \"gpt-5.6-sol\"\n"
	writeFile(t, configPath, original)
	applyConnection(t, ScaffoldCodex, home, realExecutable(t))
	registered := strings.TrimPrefix(readFile(t, configPath), original)
	writeFile(t, configPath, original+registered+registered)
	if count := strings.Count(readFile(t, configPath), blockBegin); count != 2 {
		t.Fatalf("the fixture did not produce two blocks: %d", count)
	}

	current := realExecutable(t)
	plan, err := PlanConnection(ScaffoldCodex, home, current)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Connect(plan); err != nil {
		t.Fatal(err)
	}

	after := readFile(t, configPath)
	if count := strings.Count(after, blockBegin); count != 1 {
		t.Fatalf("managed blocks after the repair = %d, want one:\n%s", count, after)
	}
	if count := strings.Count(after, managedMarker); count != len(managedEvents()) {
		t.Fatalf("managed handlers = %d, want %d:\n%s", count, len(managedEvents()), after)
	}
	// And removal must now leave nothing behind.
	removed, err := Disconnect(ScaffoldCodex, home)
	if err != nil || !removed {
		t.Fatalf("disconnect removed = %v err = %v", removed, err)
	}
	if restored := readFile(t, configPath); restored != original {
		t.Fatalf("disconnection left residue:\n%q\nwant:\n%q", restored, original)
	}
}
