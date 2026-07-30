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
		if !state.Working || state.Trust != TrustRecorded {
			t.Fatalf("state = %+v", state)
		}
		if state.NextAction != "" {
			t.Fatalf("a working attachment still asked for something: %q", state.NextAction)
		}
	})
}

func TestHealthReportsNoTrustStepWhereNoneWasObserved(t *testing.T) {
	// A live session ran a hook registered in this scaffold's repository-scope
	// settings file with no approval step. Reporting a trust state as pending or
	// undetermined here would invent an obstacle the observation ruled out, and
	// reporting a record would invent a grant nobody made — hence its own state.
	home, repo := t.TempDir(), initializedRepository(t)
	applyConnection(t, ScaffoldClaudeCode, repo, realExecutable(t))
	state := inspect(t, ScaffoldClaudeCode, home, repo)
	if state.Trust != TrustNotRequired {
		t.Fatalf("trust state = %q, want %q: %+v", state.Trust, TrustNotRequired, state)
	}
	if !state.Working {
		t.Fatalf("an attachment needing no approval step is not reported as working: %+v", state)
	}
	if state.NextAction != "" {
		t.Fatalf("a working attachment still asked for something: %q", state.NextAction)
	}
	// What the run did not capture is still named, so a green report does not read
	// as "everything about this scaffold is known".
	if len(state.Unverifiable) == 0 {
		t.Fatal("the report claims no blind spots at all")
	}
}

func TestConnectionNoticeMatchesWhatWasActuallyObserved(t *testing.T) {
	// For the scaffold whose trust gate was observed live, the notice states the
	// requirement outright.
	codex := ConnectionNotice(ScaffoldCodex)
	for _, required := range []string{"not yet active", "trust", "does nothing"} {
		if !strings.Contains(strings.ToLower(codex), required) {
			t.Fatalf("codex notice omits %q: %s", required, codex)
		}
	}

	// For the scaffold observed running a registered hook with no approval step,
	// asserting a mandatory review would send the user hunting for a screen that
	// does not exist. Inventing an obstacle is its own kind of misinformation.
	other := strings.ToLower(ConnectionNotice(ScaffoldClaudeCode))
	if strings.Contains(other, "requires you to review") {
		t.Fatalf("notice asserts a trust gate that was observed not to exist: %s", other)
	}
	if !strings.Contains(other, "no approval step") {
		t.Fatalf("notice does not say the attachment needs no approval: %s", other)
	}
	if !strings.Contains(other, "live session") {
		t.Fatalf("notice states the absence without saying it was observed: %s", other)
	}

	// A scaffold that does gate execution must name where the user acts; one that
	// does not must not send them anywhere at all.
	for _, scaffold := range SupportedScaffolds() {
		notice := ConnectionNotice(scaffold)
		gated := TrustEvidenceOf(scaffold) == TrustGateObserved
		if gated && !strings.Contains(notice, TrustSurface(scaffold)) {
			t.Fatalf("%s notice does not name the surface", scaffold)
		}
		if !gated && strings.Contains(strings.ToLower(notice), "trust them first") {
			t.Fatalf("%s notice invents a review step: %s", scaffold, notice)
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

	applyConnection(t, ScaffoldCodex, home, realExecutable(t))
	inspect(t, ScaffoldCodex, home, repo)
	if _, err := Disconnect(ScaffoldCodex, home, home); err != nil {
		t.Fatal(err)
	}
	applyConnection(t, ScaffoldCodex, home, realExecutable(t))

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
	applyConnection(t, ScaffoldCodex, home, realExecutable(t))
	return home
}

func TestHealthRefusesToReportWorkingWhenTheBinaryIsGone(t *testing.T) {
	// A registration pointing at a missing binary satisfies every configuration
	// check and still cannot run. A green result there is worse than no health
	// command at all.
	home, repo := t.TempDir(), initializedRepository(t)
	executable := realExecutable(t)
	plan, err := PlanConnection(ScaffoldCodex, home, executable)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Connect(plan); err != nil {
		t.Fatal(err)
	}
	grantTrust(t, home)
	if state := inspect(t, ScaffoldCodex, home, repo); !state.Working {
		t.Fatalf("healthy attachment reported as broken: %+v", state)
	}

	if err := os.Remove(executable); err != nil {
		t.Fatal(err)
	}
	state := inspect(t, ScaffoldCodex, home, repo)
	if state.Working {
		t.Fatal("health reported working with the registered binary missing")
	}
	mustMention(t, state.NextAction, "missing")
}

func TestHealthReportsAHalfRemovedRegistration(t *testing.T) {
	// If one stanza is removed while the marker survives, either the
	// announcement or the question retention is silently gone.
	home, repo := connectedHome(t), initializedRepository(t)
	configPath := filepath.Join(home, ".codex", "config.toml")
	grantTrust(t, home)
	raw := readFile(t, configPath)
	writeFile(t, configPath, strings.Replace(raw, "[[hooks.Stop.hooks]]", "", 1))

	state := inspect(t, ScaffoldCodex, home, repo)
	if state.Connected || state.Working {
		t.Fatalf("a half-removed registration reported as connected: %+v", state)
	}
}

func TestHealthSurfacesAnUnreadableConfiguration(t *testing.T) {
	// Reporting "not connected" would recommend a connection that reads the
	// same file and fails identically.
	home, repo := t.TempDir(), initializedRepository(t)
	// The malformed file goes where this scaffold's registration lives.
	writeFile(t, filepath.Join(repo, ".claude", "settings.local.json"), "{not json")
	state := inspect(t, ScaffoldClaudeCode, home, repo)
	if state.ConfigError == "" {
		t.Fatalf("an unreadable configuration was reported as an ordinary state: %+v", state)
	}
	if state.Working {
		t.Fatal("an unreadable configuration reported as working")
	}
	mustMention(t, state.NextAction, "repair")
}

func TestHealthNamesWhatItCouldNotVerify(t *testing.T) {
	// A green result must not read as "everything works" when it means
	// "nothing detected".
	home, repo := connectedHome(t), initializedRepository(t)
	grantTrust(t, home)
	state := inspect(t, ScaffoldCodex, home, repo)
	if len(state.Unverifiable) == 0 {
		t.Fatal("health hid its own blind spot about trust-record freshness")
	}
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
	entry := "\n[hooks.state]\n"
	for _, event := range []string{"session_start", "stop"} {
		entry += "\n[hooks.state.\"" + configPath + ":" + event + ":0:0\"]\n" +
			"trusted_hash = \"sha256:" + strings.Repeat("a", 64) + "\"\n"
	}
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

func TestHealthReportsWorkingAfterTheRepair(t *testing.T) {
	// The whole point of the repair: health detects a moved binary, names it, and
	// the remedy it prescribes now actually restores the attachment. Before this,
	// the third step reported the same failure forever.
	home, repo := t.TempDir(), initializedRepository(t)
	old := realExecutable(t)
	applyConnection(t, ScaffoldCodex, home, old)
	grantTrust(t, home)
	if state := inspect(t, ScaffoldCodex, home, repo); !state.Working {
		t.Fatalf("a healthy attachment reported as broken: %+v", state)
	}

	if err := os.Remove(old); err != nil {
		t.Fatal(err)
	}
	broken := inspect(t, ScaffoldCodex, home, repo)
	if broken.Working {
		t.Fatal("health reported working with the registered binary missing")
	}
	mustMention(t, broken.NextAction, "missing")
	mustMention(t, broken.NextAction, "gr connect")

	// Follow the advice health just gave.
	current := realExecutable(t)
	plan, err := PlanConnection(ScaffoldCodex, home, current)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := Connect(plan)
	if err != nil || !changed {
		t.Fatalf("the prescribed remedy changed = %v err = %v", changed, err)
	}
	// This is confirmed signal SIG-5: health reports working after the repair,
	// having named the missing binary before it.
	//
	// Review disputes the signal itself, and the objection holds: the trust record
	// still in the configuration was made against the command the repair replaced,
	// so "working" here is reported over a hook the scaffold will not run until the
	// user reviews it again. Connection discloses that at the moment of repair, but
	// the knowledge does not survive into a later health check — which is the very
	// command connection tells the user to run.
	//
	// Resolving it means health must learn about the invalidation, which the
	// confirmed non-goal NG-1 excludes from this change. Left as the confirmed
	// contract with the objection recorded, pending an owner decision on a new
	// intent version; it is not a settled question.
	if state := inspect(t, ScaffoldCodex, home, repo); !state.Working {
		t.Fatalf("the attachment did not recover after the prescribed remedy: %+v", state)
	}
}

func TestRepairNoticeMatchesWhatWasActuallyObserved(t *testing.T) {
	// A repair discards the review the previous command was given, so it must say
	// so — under the same discipline as the first-connection notice, because the
	// certainty differs per scaffold.
	codex := RepairNotice(ScaffoldCodex, "/old/gr")
	if !strings.Contains(codex, "/old/gr") {
		t.Fatalf("the notice does not say what was replaced: %s", codex)
	}
	for _, required := range []string{"no longer applies", "does nothing"} {
		if !strings.Contains(strings.ToLower(codex), required) {
			t.Fatalf("codex repair notice omits %q: %s", required, codex)
		}
	}

	// For the scaffold observed to gate nothing, a replaced command needs no second
	// review, and claiming otherwise would invent an obstacle.
	other := strings.ToLower(RepairNotice(ScaffoldClaudeCode, "/old/gr"))
	if strings.Contains(other, "no longer applies") {
		t.Fatalf("repair notice asserts a trust gate that was observed not to exist: %s", other)
	}
	if !strings.Contains(other, "without an approval step") {
		t.Fatalf("repair notice does not say the replacement simply applies: %s", other)
	}

	// The replacement must be named on every scaffold, even when the previous path
	// could not be read, and only a gated scaffold sends the user to a surface.
	for _, scaffold := range SupportedScaffolds() {
		notice := RepairNotice(scaffold, "")
		if !strings.Contains(notice, "was replaced") {
			t.Fatalf("%s repair notice does not say a replacement happened", scaffold)
		}
		if TrustEvidenceOf(scaffold) == TrustGateObserved &&
			!strings.Contains(notice, TrustSurface(scaffold)) {
			t.Fatalf("%s repair notice does not name the surface", scaffold)
		}
	}
}

func TestRepairWritesNoTrustRecord(t *testing.T) {
	// The repair knows the stored record went stale. Knowing that must never
	// become licence to write a fresh one: trust is standing consent to run a
	// command in every session the user starts.
	home := t.TempDir()
	configPath := filepath.Join(home, ".codex", "config.toml")
	old := realExecutable(t)
	applyConnection(t, ScaffoldCodex, home, old)
	if err := os.Remove(old); err != nil {
		t.Fatal(err)
	}
	plan, err := PlanConnection(ScaffoldCodex, home, realExecutable(t))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Connect(plan); err != nil {
		t.Fatal(err)
	}
	raw := readFile(t, configPath)
	for _, forbidden := range []string{"hooks.state", "trusted_hash"} {
		if strings.Contains(raw, forbidden) {
			t.Fatalf("the repair wrote a scaffold trust record (%q):\n%s", forbidden, raw)
		}
	}
}

func TestRepairNoticeDoesNotSendTheUserToASurfaceThatContradictsIt(t *testing.T) {
	// The scaffold's trust record survives a repair and still reads as present, so
	// health reports the repaired attachment as working. A notice that says "review
	// is needed" and then points at that command hands the user two answers and no
	// way to choose between them.
	codex := RepairNotice(ScaffoldCodex, "/old/gr")
	if strings.Contains(codex, "Run `gr doctor` to check") {
		t.Fatalf("the repair notice points at a command that contradicts it: %s", codex)
	}
	if !strings.Contains(codex, "cannot confirm") {
		t.Fatalf("the repair notice hides that health cannot see this step: %s", codex)
	}

	// For the scaffold observed to gate nothing, the diagnosis agrees with the
	// notice — it reports the attachment as working — so the pointer stays accurate.
	other := RepairNotice(ScaffoldClaudeCode, "/old/gr")
	if !strings.Contains(other, "gr doctor") {
		t.Fatalf("the notice withholds an accurate pointer: %s", other)
	}

	// And the first-connection notice keeps its pointer: with no record yet, the
	// diagnosis reports trust as pending and names the review surface.
	if !strings.Contains(ConnectionNotice(ScaffoldCodex), "gr doctor") {
		t.Fatal("the first-connection notice lost an accurate pointer")
	}
}
