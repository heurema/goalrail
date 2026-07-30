package harness

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/heurema/goalrail/internal/ambient"
)

// diagnoseFor builds a diagnosis with the machine's own environment kept out, so a
// test never depends on whether the developer happens to have a Node runtime or an
// observability endpoint configured.
func diagnoseFor(t *testing.T, root, home string, nodePresent bool, environment map[string]string) Diagnosis {
	t.Helper()
	input := DiagnoseInput{
		RepositoryRoot: root,
		Home:           home,
		StateRoot:      t.TempDir(),
		LookupEnvironment: func(name string) string {
			return environment[name]
		},
		LookPath: func(string) (string, error) {
			if nodePresent {
				return "/usr/local/bin/node", nil
			}
			return "", errors.New("not found")
		},
	}
	diagnosis, err := Diagnose(input)
	if err != nil {
		t.Fatalf("diagnose: %v", err)
	}
	return diagnosis
}

func problemMentioning(diagnosis Diagnosis, fragment string) (string, bool) {
	for index, problem := range diagnosis.Problems {
		if strings.Contains(problem, fragment) {
			action := ""
			if index < len(diagnosis.NextActions) {
				action = diagnosis.NextActions[index]
			}
			return action, true
		}
	}
	return "", false
}

func TestDiagnosisReportsAnUninitializedRepository(t *testing.T) {
	diagnosis := diagnoseFor(t, t.TempDir(), t.TempDir(), true, nil)
	if diagnosis.Working {
		t.Fatal("an empty directory was reported as working")
	}
	action, found := problemMentioning(diagnosis, "not initialized")
	if !found {
		t.Fatalf("the diagnosis does not name the state: %+v", diagnosis.Problems)
	}
	if !strings.Contains(action, "gr init") {
		t.Errorf("next action is %q", action)
	}
	if action, found := problemMentioning(diagnosis, "overlay is not installed"); !found {
		t.Error("a missing overlay is not reported")
	} else if !strings.Contains(action, "gr init") {
		t.Errorf("next action for a missing overlay is %q", action)
	}
}

func TestDiagnosisReportsDriftPerFile(t *testing.T) {
	root := t.TempDir()
	if _, err := Materialize(root, false); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	edited := OverlayDirectory + "/templates/proposal.md"
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(edited)), []byte("mine\n"), 0o644); err != nil {
		t.Fatalf("edit: %v", err)
	}
	diagnosis := diagnoseFor(t, root, t.TempDir(), true, nil)
	if diagnosis.Working {
		t.Fatal("a drifted overlay was reported as working")
	}
	// Naming the file is the point: "something differs" leaves the user diffing
	// seven files by hand, which is the work a diagnosis exists to remove.
	action, found := problemMentioning(diagnosis, edited)
	if !found {
		t.Fatalf("drift is not reported per file: %+v", diagnosis.Problems)
	}
	if !strings.Contains(action, "gr update --discard-local-edits") {
		t.Errorf("next action for an edit is %q", action)
	}
}

// TestDiagnosisIgnoresTheVersionWhenJudgingCurrency pins that no verdict about a
// repository is derived from the binary's version string.
func TestDiagnosisIgnoresTheVersionWhenJudgingCurrency(t *testing.T) {
	root := t.TempDir()
	if _, err := Materialize(root, false); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	diagnosis := diagnoseFor(t, root, t.TempDir(), true, nil)
	if !diagnosis.Overlay.Current {
		t.Fatalf("a freshly installed overlay is not current: %+v", diagnosis.Overlay)
	}
	if diagnosis.Version != Version {
		t.Errorf("version reported as %q", diagnosis.Version)
	}
	// The marker is per clone and git-ignored, so nothing about committed content
	// may depend on it. Removing it must not change the overlay verdict.
	if err := os.RemoveAll(filepath.Join(root, ".goalrail")); err != nil {
		t.Fatalf("remove marker: %v", err)
	}
	after := diagnoseFor(t, root, t.TempDir(), true, nil)
	if !after.Overlay.Current {
		t.Error("the overlay verdict changed when the marker was removed")
	}
}

// TestAnAbsentToolchainIsAFactNotAFault pins that the harness stands without the
// runtime the stock CLI needs.
func TestAnAbsentToolchainIsAFactNotAFault(t *testing.T) {
	root, home := t.TempDir(), t.TempDir()
	prepareWorkingHarness(t, root, home)

	diagnosis := diagnoseFor(t, root, home, false, nil)
	if diagnosis.Toolchain.NodePresent {
		t.Fatal("the toolchain was reported as present")
	}
	if !diagnosis.Working {
		t.Fatalf("an absent Node runtime made the harness unhealthy: %+v", diagnosis.Problems)
	}
	if _, found := problemMentioning(diagnosis, "Node"); found {
		t.Error("the absent runtime was reported as a problem")
	}
	// Its absence has to be legible: what cannot be run is named.
	if !strings.Contains(diagnosis.Toolchain.Note, "validation and archival") {
		t.Errorf("the note does not say what cannot run: %q", diagnosis.Toolchain.Note)
	}
	if !strings.Contains(Describe(diagnosis), "Node runtime") {
		t.Error("the human report omits the toolchain fact")
	}
}

func TestObservabilityAbsenceIsOptionalAndNeverLeaks(t *testing.T) {
	root, home := t.TempDir(), t.TempDir()
	prepareWorkingHarness(t, root, home)

	absent := diagnoseFor(t, root, home, true, nil)
	if !absent.Working {
		t.Fatalf("absent observability made the harness unhealthy: %+v", absent.Problems)
	}
	if absent.Observability.Configured {
		t.Fatal("observability was reported as configured")
	}
	if !strings.Contains(absent.Observability.Note, "optional") {
		t.Errorf("the line does not say it is optional: %q", absent.Observability.Note)
	}
	if _, found := problemMentioning(absent, "observability"); found {
		t.Error("absent observability produced a problem to act on")
	}

	configured := diagnoseFor(t, root, home, true, map[string]string{
		"LANGFUSE_HOST":       "https://langfuse.example",
		"LANGFUSE_PUBLIC_KEY": "pk-not-a-real-key",
		"LANGFUSE_SECRET_KEY": "sk-not-a-real-key",
	})
	if !configured.Observability.Configured {
		t.Fatal("a configured endpoint was not detected")
	}
	if configured.Observability.Endpoint != "https://langfuse.example" {
		t.Errorf("endpoint reported as %q", configured.Observability.Endpoint)
	}
	// The endpoint is the most a report may name.
	rendered := Describe(configured)
	for _, secret := range []string{"pk-not-a-real-key", "sk-not-a-real-key"} {
		if strings.Contains(rendered, secret) {
			t.Error("the report echoed key material")
		}
	}
}

// TestDiagnosisNamesASupersededEventRegistration pins the migration path for the
// arrangement this change replaced.
func TestDiagnosisNamesASupersededEventRegistration(t *testing.T) {
	root, home := t.TempDir(), t.TempDir()
	prepareWorkingHarness(t, root, home)

	settingsPath := filepath.Join(root, filepath.FromSlash(ambient.RepositorySettingsPath))
	raw, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("read settings: %v", err)
	}
	// Move the handler back onto the per-turn event, as an earlier arrangement wrote
	// it.
	if err := os.WriteFile(settingsPath, []byte(strings.Replace(string(raw), `"SessionEnd"`, `"Stop"`, 1)), 0o644); err != nil {
		t.Fatalf("write settings: %v", err)
	}

	diagnosis := diagnoseFor(t, root, home, true, nil)
	action, found := problemMentioning(diagnosis, "once per turn")
	if !found {
		t.Fatalf("a superseded event registration is not reported: %+v", diagnosis.Problems)
	}
	if !strings.Contains(action, "gr init") {
		t.Errorf("next action is %q", action)
	}
	if diagnosis.Working {
		t.Error("a registration on the superseded event was reported as working")
	}
}

// TestEveryNextActionNamesARealCommand pins that advice can always be followed.
func TestEveryNextActionNamesARealCommand(t *testing.T) {
	accepted := map[string]bool{
		"gr init": true, "gr doctor": true, "gr update": true,
		"gr connect": true, "gr disconnect": true, "gr version": true,
		"gr prepare": true, "gr inspect": true, "gr start": true, "gr finish": true,
	}
	// One diagnosis per interesting state, so every action string is exercised.
	roots := []func() (string, string){
		func() (string, string) { return t.TempDir(), t.TempDir() },
		func() (string, string) {
			root, home := t.TempDir(), t.TempDir()
			prepareWorkingHarness(t, root, home)
			target := filepath.Join(root, filepath.FromSlash(OverlayDirectory), "templates", "tasks.md")
			if err := os.WriteFile(target, []byte("mine\n"), 0o644); err != nil {
				t.Fatalf("edit: %v", err)
			}
			return root, home
		},
	}
	for _, build := range roots {
		root, home := build()
		diagnosis := diagnoseFor(t, root, home, true, nil)
		for _, action := range diagnosis.NextActions {
			for _, fragment := range strings.Split(action, "`") {
				trimmed := strings.TrimSpace(fragment)
				if !strings.HasPrefix(trimmed, "gr ") {
					continue
				}
				words := strings.Fields(trimmed)
				if len(words) < 2 {
					t.Errorf("action names no command: %q", action)
					continue
				}
				if !accepted[words[0]+" "+words[1]] {
					t.Errorf("action names a command this tool does not accept: %q", trimmed)
				}
			}
		}
	}
}

// prepareWorkingHarness installs the overlay, the marker, and a repository-scope
// registration naming a runnable executable, which is what a working attachment
// requires.
func prepareWorkingHarness(t *testing.T, root, home string) {
	t.Helper()
	if _, err := Materialize(root, false); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	if _, err := EnsureConfig(root, false); err != nil {
		t.Fatalf("ensure config: %v", err)
	}
	if _, _, err := ambient.Initialize(root, nowForTest); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	executable := filepath.Join(t.TempDir(), "gr")
	if err := os.WriteFile(executable, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write executable: %v", err)
	}
	target, err := ambient.RegistrationTarget(ambient.ScaffoldClaudeCode, home, root)
	if err != nil {
		t.Fatalf("target: %v", err)
	}
	plan, err := ambient.PlanRegistration(target, executable)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if _, err := ambient.Connect(plan); err != nil {
		t.Fatalf("register: %v", err)
	}
}

// nowForTest is the clock the marker is written with. A fixed value keeps the
// marker's contents out of what these tests compare.
func nowForTest() time.Time { return time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC) }

// TestDiagnosisReportsAnOverlayBehindTheCanon exercises the behind verdict at the
// diagnosis level, against a synthetic history: the state is unreachable in the
// field until a second canon ships, and its next action differs from an edit's.
func TestDiagnosisReportsAnOverlayBehindTheCanon(t *testing.T) {
	root, home := t.TempDir(), t.TempDir()
	prepareWorkingHarness(t, root, home)

	relative := OverlayDirectory + "/templates/context.md"
	older := []byte("# an earlier canon's context template\n")
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(relative)), older, 0o644); err != nil {
		t.Fatalf("write older content: %v", err)
	}
	restore := previousCanons
	previousCanons = []Canon{{ID: "sha256:older", Files: []CanonFile{{Path: relative, Digest: Digest(older)}}}}
	defer func() { previousCanons = restore }()

	diagnosis := diagnoseFor(t, root, home, true, nil)
	if !diagnosis.Overlay.Behind {
		t.Fatalf("the overlay is not reported as behind: %+v", diagnosis.Overlay)
	}
	action, found := problemMentioning(diagnosis, "is behind the canon")
	if !found {
		t.Fatalf("the diagnosis does not name the behind file: %+v", diagnosis.Problems)
	}
	// The action must be the plain update, not the one that discards edits: nothing
	// here is a local edit to lose.
	if action != "run `gr update`" {
		t.Errorf("next action for a behind file is %q", action)
	}
}

// TestObservabilityConfigurationNeverEntersTheRepository pins that credentials
// stay outside repository content even when an endpoint is configured.
func TestObservabilityConfigurationNeverEntersTheRepository(t *testing.T) {
	root, home := t.TempDir(), t.TempDir()
	prepareWorkingHarness(t, root, home)
	secrets := map[string]string{
		"LANGFUSE_HOST":       "https://langfuse.example",
		"LANGFUSE_PUBLIC_KEY": "pk-not-a-real-key",
		"LANGFUSE_SECRET_KEY": "sk-not-a-real-key",
	}
	if _, err := Materialize(root, false); err != nil {
		t.Fatalf("materialize: %v", err)
	}
	if _, err := EnsureConfig(root, false); err != nil {
		t.Fatalf("ensure config: %v", err)
	}
	diagnosis := diagnoseFor(t, root, home, true, secrets)
	if !diagnosis.Observability.Configured {
		t.Fatal("a configured endpoint was not detected")
	}

	walkErr := filepath.Walk(root, func(walked string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		content, readErr := os.ReadFile(walked)
		if readErr != nil {
			return readErr
		}
		for name, value := range secrets {
			if name == "LANGFUSE_HOST" {
				continue
			}
			if strings.Contains(string(content), value) {
				t.Errorf("%s carries key material", walked)
			}
		}
		return nil
	})
	if walkErr != nil {
		t.Fatalf("walk: %v", walkErr)
	}
}

// TestObservabilityEndpointNeverEchoesEmbeddedCredentials pins the redaction the
// pre-PR review asked for: a URL carrying userinfo would otherwise print its
// secret in the one line that promises to name the endpoint at most.
func TestObservabilityEndpointNeverEchoesEmbeddedCredentials(t *testing.T) {
	state := InspectObservability("", func(name string) string {
		switch name {
		case "LANGFUSE_HOST":
			return "https://user:embedded-secret@langfuse.example"
		case "LANGFUSE_PUBLIC_KEY":
			return "pk-x"
		case "LANGFUSE_SECRET_KEY":
			return "sk-x"
		}
		return ""
	})
	if !state.Configured {
		t.Fatal("a configured endpoint was not detected")
	}
	if strings.Contains(state.Endpoint, "embedded-secret") || strings.Contains(state.Endpoint, "user") && strings.Contains(state.Endpoint, "@") {
		t.Errorf("the reported endpoint carries credentials: %q", state.Endpoint)
	}
	if !strings.Contains(state.Endpoint, "langfuse.example") {
		t.Errorf("redaction destroyed the endpoint: %q", state.Endpoint)
	}
}
