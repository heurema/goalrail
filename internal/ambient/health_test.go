package ambient

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHealthDistinguishesEveryState(t *testing.T) {
	// "Not working" is useless when three causes produce it, so each state must
	// be named and each must carry its own next action.
	t.Run("not connected", func(t *testing.T) {
		home, repo := t.TempDir(), initializedRepository(t)
		state := inspect(t, ScaffoldCodex, home, repo)
		if state.Connected || state.Working {
			t.Fatalf("state = %+v", state)
		}
		mustMention(t, state.NextAction, "gr connect")
	})

	t.Run("connected but repository not initialized", func(t *testing.T) {
		home, repo := connectedHome(t), t.TempDir()
		state := inspect(t, ScaffoldCodex, home, repo)
		if !state.Connected || state.Initialized || state.Working {
			t.Fatalf("state = %+v", state)
		}
		mustMention(t, state.NextAction, "gr init")
	})

	t.Run("connected and initialized but trust pending", func(t *testing.T) {
		home, repo := connectedHome(t), initializedRepository(t)
		state := inspect(t, ScaffoldCodex, home, repo)
		if state.Trust != TrustPending || state.Working {
			t.Fatalf("state = %+v", state)
		}
		// This is the state that otherwise looks exactly like breakage.
		mustMention(t, state.NextAction, "/hooks")
	})

	t.Run("working", func(t *testing.T) {
		home, repo := connectedHome(t), initializedRepository(t)
		grantTrust(t, home)
		state := inspect(t, ScaffoldCodex, home, repo)
		if !state.Working || state.Trust != TrustGranted {
			t.Fatalf("state = %+v", state)
		}
		if state.NextAction != "" {
			t.Fatalf("a working attachment still asked for something: %q", state.NextAction)
		}
	})
}

func TestHealthReportsUnknownRatherThanGuessing(t *testing.T) {
	// No trust record has been observed for this scaffold. Claiming either
	// answer would be invention, and an optimistic guess is the worse one.
	home, repo := t.TempDir(), initializedRepository(t)
	applyConnection(t, ScaffoldClaudeCode, home)
	state := inspect(t, ScaffoldClaudeCode, home, repo)
	if state.Trust != TrustUnknown || state.Working {
		t.Fatalf("state = %+v", state)
	}
	mustMention(t, state.NextAction, "could not be determined")
}

func TestConnectionNoticeTellsTheUserWhatIsMissing(t *testing.T) {
	for _, scaffold := range SupportedScaffolds() {
		notice := ConnectionNotice(scaffold)
		for _, required := range []string{"not yet active", "trust", "does nothing"} {
			if !strings.Contains(strings.ToLower(notice), required) {
				t.Fatalf("%s notice omits %q: %s", scaffold, required, notice)
			}
		}
		// Telling someone a step is required without saying where leaves them
		// exactly as stuck.
		if !strings.Contains(notice, TrustSurface(scaffold)) {
			t.Fatalf("%s notice does not name the surface: %s", scaffold, notice)
		}
	}
}

func TestInspectionNeverWritesAnything(t *testing.T) {
	// The prohibition is on writing trust, and the cheapest proof is that
	// inspection leaves the scaffold configuration byte-identical.
	home, repo := connectedHome(t), initializedRepository(t)
	configPath := filepath.Join(home, ".codex", "config.toml")
	before, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		inspect(t, ScaffoldCodex, home, repo)
	}
	after, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("inspecting attachment health modified the scaffold configuration")
	}
}

func TestNoTrustRecordIsEverWritten(t *testing.T) {
	// Forging trust is reproducible and practised elsewhere, so the absence of
	// a write path is pinned rather than assumed. Trust is standing consent to
	// run a command in every session; manufacturing it is not ours to do.
	home, repo := t.TempDir(), initializedRepository(t)
	configPath := filepath.Join(home, ".codex", "config.toml")

	applyConnection(t, ScaffoldCodex, home)
	inspect(t, ScaffoldCodex, home, repo)
	if _, err := Disconnect(ScaffoldCodex, home); err != nil {
		t.Fatal(err)
	}
	applyConnection(t, ScaffoldCodex, home)

	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"hooks.state", "trusted_hash"} {
		if strings.Contains(string(raw), forbidden) {
			t.Fatalf("Goalrail wrote a scaffold trust record (%q):\n%s", forbidden, raw)
		}
	}
}

func inspect(t *testing.T, scaffold Scaffold, home, repo string) AttachmentState {
	t.Helper()
	state, err := Inspect(scaffold, home, repo)
	if err != nil {
		t.Fatal(err)
	}
	return state
}

func connectedHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	applyConnection(t, ScaffoldCodex, home)
	return home
}

// grantTrust simulates what the scaffold itself writes once the user has
// reviewed the hooks. Tests may write it; Goalrail may not.
func grantTrust(t *testing.T, home string) {
	t.Helper()
	configPath := filepath.Join(home, ".codex", "config.toml")
	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	entry := "\n[hooks.state]\n\n[hooks.state.\"" + configPath +
		":session_start:0:0\"]\ntrusted_hash = \"sha256:" + strings.Repeat("a", 64) + "\"\n"
	if err := os.WriteFile(configPath, append(raw, []byte(entry)...), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustMention(t *testing.T, text, fragment string) {
	t.Helper()
	if !strings.Contains(text, fragment) {
		t.Fatalf("next action %q does not mention %q", text, fragment)
	}
}
