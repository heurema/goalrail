package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/heurema/goalrail/internal/ambient"
)

// runHookIn drives the ambient entry point the way a scaffold would: from the
// session's working directory, with the event payload on stdin.
func runHookIn(t *testing.T, directory, payload string) (string, error) {
	t.Helper()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(directory); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })

	var stdout bytes.Buffer
	hookErr := run(
		context.Background(),
		[]string{"hook"},
		strings.NewReader(payload),
		&stdout,
		&bytes.Buffer{},
		productionService,
	)
	return stdout.String(), hookErr
}

func TestHookIsSilentInAnUninitializedDirectory(t *testing.T) {
	// The hook fires for every session the user starts anywhere. Everywhere
	// but an initialized repository it must do nothing at all.
	directory := t.TempDir()
	t.Setenv("GOALRAIL_STATE_HOME", t.TempDir())

	output, err := runHookIn(t, directory, `{"hook_event_name":"SessionStart","source":"startup"}`)
	if err != nil {
		t.Fatalf("the hook reported an error into an ordinary session: %v", err)
	}
	if output != "" {
		t.Fatalf("the hook spoke in an unconnected directory: %q", output)
	}
}

func TestHookAnnouncesInAnInitializedDirectory(t *testing.T) {
	directory := t.TempDir()
	t.Setenv("GOALRAIL_STATE_HOME", t.TempDir())
	if _, _, err := ambient.Initialize(directory, nowUTC); err != nil {
		t.Fatal(err)
	}

	output, err := runHookIn(t, directory, `{"hook_event_name":"SessionStart","source":"startup"}`)
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		HookSpecificOutput struct {
			HookEventName     string `json:"hookEventName"`
			AdditionalContext string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal([]byte(output), &decoded); err != nil {
		t.Fatalf("hook output does not parse: %v (%q)", err, output)
	}
	if decoded.HookSpecificOutput.AdditionalContext != ambient.AmbientAnnouncement {
		t.Fatal("the session was not told the channel exists")
	}
}

func TestHookStaysSilentOnRecurringSessionEvents(t *testing.T) {
	directory := t.TempDir()
	t.Setenv("GOALRAIL_STATE_HOME", t.TempDir())
	if _, _, err := ambient.Initialize(directory, nowUTC); err != nil {
		t.Fatal(err)
	}

	for _, source := range []string{"resume", "clear", "compact"} {
		output, err := runHookIn(
			t, directory,
			`{"hook_event_name":"SessionStart","source":"`+source+`"}`,
		)
		if err != nil {
			t.Fatal(err)
		}
		if output != "" {
			t.Fatalf("source %q repeated the announcement: %q", source, output)
		}
	}
}

func TestHookRetainsAQuestionAtSessionStop(t *testing.T) {
	directory := t.TempDir()
	stateHome := t.TempDir()
	t.Setenv("GOALRAIL_STATE_HOME", stateHome)
	if _, _, err := ambient.Initialize(directory, nowUTC); err != nil {
		t.Fatal(err)
	}
	question := filepath.Join(directory, filepath.FromSlash(ambient.ReservedEscalationPath))
	if err := os.WriteFile(question, []byte("Which document governs here?\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := runHookIn(t, directory, `{"hook_event_name":"Stop","session_id":"session-one"}`); err != nil {
		t.Fatal(err)
	}
	if !containsRetainedQuestion(t, stateHome) {
		t.Fatal("the question was not retained outside the repository")
	}
}

func TestHookNeverFailsIntoTheSession(t *testing.T) {
	// Fail-quiet: whatever is malformed, missing, or broken, an ordinary
	// session must proceed as if Goalrail were not installed.
	directory := t.TempDir()
	t.Setenv("GOALRAIL_STATE_HOME", t.TempDir())
	if _, _, err := ambient.Initialize(directory, nowUTC); err != nil {
		t.Fatal(err)
	}

	for name, payload := range map[string]string{
		"empty":            "",
		"not json":         "{{{",
		"unknown event":    `{"hook_event_name":"Wat"}`,
		"missing event":    `{"session_id":"x"}`,
		"stop unprompted":  `{"hook_event_name":"Stop"}`,
		"truncated object": `{"hook_event_name":`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := runHookIn(t, directory, payload); err != nil {
				t.Fatalf("the hook failed into the session: %v", err)
			}
		})
	}
}

func TestInitCommandIsExplicitAndReportsItself(t *testing.T) {
	directory := t.TempDir()
	var stdout bytes.Buffer
	if err := run(
		context.Background(),
		[]string{"init", "--repo", directory},
		strings.NewReader(""),
		&stdout,
		&bytes.Buffer{},
		productionService,
	); err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Created bool   `json:"created"`
		Marker  string `json:"marker"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if !decoded.Created || decoded.Marker != ambient.MarkerPath {
		t.Fatalf("init reported %+v", decoded)
	}
	if !ambient.IsInitialized(directory) {
		t.Fatal("init did not initialize the repository")
	}
}

func TestConnectWithoutConsentChangesNothing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	var stdout bytes.Buffer
	if err := run(
		context.Background(),
		[]string{"connect", "--scaffold", "codex"},
		strings.NewReader(""),
		&stdout,
		&bytes.Buffer{},
		productionService,
	); err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Applied bool `json:"applied"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Applied {
		t.Fatal("connect applied without consent")
	}
	if _, err := os.Stat(filepath.Join(home, ".codex", "config.toml")); !os.IsNotExist(err) {
		t.Fatal("connect wrote to the scaffold configuration without consent")
	}
}

func containsRetainedQuestion(t *testing.T, stateHome string) bool {
	t.Helper()
	found := false
	_ = filepath.Walk(stateHome, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		if strings.Contains(filepath.ToSlash(path), "ambient/questions/") &&
			strings.HasSuffix(path, "question.md") {
			found = true
		}
		return nil
	})
	return found
}

func TestHelpPresentsBackgroundSurfaceAndHidesTheHook(t *testing.T) {
	var stdout bytes.Buffer
	if err := run(
		context.Background(),
		[]string{"help"},
		strings.NewReader(""),
		&stdout,
		&bytes.Buffer{},
		productionService,
	); err != nil {
		t.Fatal(err)
	}
	text := stdout.String()
	for _, required := range []string{"connect", "disconnect", "init", "no Goalrail command"} {
		if !strings.Contains(text, required) {
			t.Fatalf("help does not mention %q:\n%s", required, text)
		}
	}
	// The hook is invoked by the scaffold, never by a person. Advertising it
	// would invite manual runs of a fail-quiet path whose silence reads as
	// breakage.
	if strings.Contains(text, "hook") {
		t.Fatalf("help advertises the scaffold-invoked hook:\n%s", text)
	}
}

// tattlingReader fails the test if anything reads from it. It stands in for a
// hook payload in a repository the user never connected: prompts, transcript
// paths, and authorization fields that must not be observed.
type tattlingReader struct{ t *testing.T }

func (reader tattlingReader) Read([]byte) (int, error) {
	reader.t.Fatal("the hook read the session payload before the initialization check")
	return 0, nil
}

func TestHookDoesNotReadThePayloadInAnUninitializedDirectory(t *testing.T) {
	directory := t.TempDir()
	t.Setenv("GOALRAIL_STATE_HOME", t.TempDir())
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(directory); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })

	if err := run(
		context.Background(),
		[]string{"hook"},
		tattlingReader{t: t},
		&bytes.Buffer{},
		&bytes.Buffer{},
		productionService,
	); err != nil {
		t.Fatal(err)
	}
}

func TestConnectOutputDisclosesTheTrustStep(t *testing.T) {
	// Without this the user connects, works, observes nothing, and reasonably
	// concludes the product is broken. That happened during live verification.
	home := t.TempDir()
	t.Setenv("HOME", home)
	var stdout bytes.Buffer
	if err := run(
		context.Background(),
		[]string{"connect", "--scaffold", "codex", "--yes"},
		strings.NewReader(""),
		&stdout,
		&bytes.Buffer{},
		productionService,
	); err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Applied      bool   `json:"applied"`
		Notice       string `json:"notice"`
		TrustSurface string `json:"trust_surface"`
		ActiveNow    bool   `json:"active_now"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if !decoded.Applied || decoded.ActiveNow {
		t.Fatalf("connect reported %+v", decoded)
	}
	for _, required := range []string{"not yet active", "trust", "does nothing"} {
		if !strings.Contains(strings.ToLower(decoded.Notice), required) {
			t.Fatalf("notice omits %q: %s", required, decoded.Notice)
		}
	}
	if decoded.TrustSurface == "" {
		t.Fatal("connect did not name where the user grants trust")
	}
}

func TestHealthCommandReportsWhatIsMissing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	directory := t.TempDir()
	var stdout bytes.Buffer
	if err := run(
		context.Background(),
		[]string{"health", "--scaffold", "codex", "--repo", directory},
		strings.NewReader(""),
		&stdout,
		&bytes.Buffer{},
		productionService,
	); err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Connected  bool   `json:"connected"`
		Working    bool   `json:"working"`
		NextAction string `json:"next_action"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Connected || decoded.Working {
		t.Fatalf("health reported %+v on an unconnected scaffold", decoded)
	}
	if !strings.Contains(decoded.NextAction, "gr connect") {
		t.Fatalf("health did not name the next action: %q", decoded.NextAction)
	}
}

func TestHelpPresentsHealthAndTheTrustStep(t *testing.T) {
	var stdout bytes.Buffer
	if err := run(
		context.Background(),
		[]string{"help"},
		strings.NewReader(""),
		&stdout,
		&bytes.Buffer{},
		productionService,
	); err != nil {
		t.Fatal(err)
	}
	text := stdout.String()
	for _, required := range []string{"health", "trust", "nothing runs until"} {
		if !strings.Contains(strings.ToLower(text), required) {
			t.Fatalf("help omits %q:\n%s", required, text)
		}
	}
}
