package ambient

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const executablePath = "/usr/local/bin/gr"

func TestConnectionRequiresConsentAndIsPlannedFirst(t *testing.T) {
	// Editing a user's own configuration is consented to as a concrete act:
	// the plan is computed before anything is written.
	home := t.TempDir()
	plan, err := PlanConnection(ScaffoldCodex, home, executablePath)
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

	applyConnection(t, ScaffoldCodex, home)
	after := readFile(t, configPath)
	if !strings.Contains(after, "hooks.SessionStart") || !strings.Contains(after, "hooks.Stop") {
		t.Fatalf("connection did not register both events:\n%s", after)
	}
	if !strings.HasPrefix(after, original) {
		t.Fatal("connection disturbed the user's existing configuration")
	}

	// A second connection must change nothing.
	plan, err := PlanConnection(ScaffoldCodex, home, executablePath)
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

	applyConnection(t, ScaffoldClaudeCode, home)
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

func applyConnection(t *testing.T, scaffold Scaffold, home string) {
	t.Helper()
	plan, err := PlanConnection(scaffold, home, executablePath)
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
	applyConnection(t, ScaffoldClaudeCode, home)
	configPath := filepath.Join(home, ".claude", "settings.json")
	settings := readJSON(t, configPath)
	hooks := settings["hooks"].(map[string]any)
	delete(hooks, "Stop")
	writeFile(t, configPath, marshalJSON(t, settings))

	plan, err := PlanConnection(ScaffoldClaudeCode, home, executablePath)
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
	if _, err := PlanConnection(ScaffoldClaudeCode, home, executablePath); err != nil {
		t.Fatalf("an empty settings file blocked connection planning: %v", err)
	}
}

func TestDisconnectRemovesOnlyOurHandlerFromAMixedGroup(t *testing.T) {
	// A group can mix our handler with a foreign one. Removal must filter the
	// handler, not drop its containing group.
	home := t.TempDir()
	configPath := filepath.Join(home, ".claude", "settings.json")
	applyConnection(t, ScaffoldClaudeCode, home)
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
			applyConnection(t, scaffold, home)
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
