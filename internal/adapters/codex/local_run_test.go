package codex

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/heurema/goalrail/internal/domain"
	"github.com/heurema/goalrail/internal/localrun"
)

type recordingLauncher struct {
	request ProcessRequest
	result  ProcessResult
	calls   int
	lock    sync.Mutex
}

func (launcher *recordingLauncher) Launch(
	_ context.Context,
	request ProcessRequest,
) ProcessResult {
	launcher.lock.Lock()
	defer launcher.lock.Unlock()
	launcher.calls++
	launcher.request = request
	return launcher.result
}

func TestLocalRunContextRoundTripsAndRejectsMutableFields(t *testing.T) {
	runContext := testLocalRunContext(t)
	encoded, err := runContext.EnvironmentValue()
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeLocalRunContext([]byte(encoded))
	if err != nil {
		t.Fatal(err)
	}
	if decoded.WorkSpecDigest() != runContext.WorkSpecDigest() ||
		decoded.RunID() != runContext.RunID() ||
		decoded.RepoRoot() != runContext.RepoRoot() {
		t.Fatalf("decoded context = %+v", decoded)
	}
	withUnknown := strings.TrimSuffix(encoded, "}") + `,"provider":"codex"}`
	if _, err := DecodeLocalRunContext([]byte(withUnknown)); err == nil {
		t.Fatal("expected unknown context field rejection")
	}
}

func TestLocalRunLifecycleHookUsesProviderIdentityAndKeepsConflictSticky(t *testing.T) {
	runContext := testLocalRunContext(t)
	encoded, err := runContext.EnvironmentValue()
	if err != nil {
		t.Fatal(err)
	}
	first, err := BindLocalRunLifecycleHook(
		runContext,
		encoded,
		localRunHook(t, runContext.RepoRoot(), "session-one"),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !first.Verified() || first.Lineage.RootSessionID != "session-one" {
		t.Fatalf("first correlation = %+v", first)
	}
	conflict, err := BindLocalRunLifecycleHook(
		runContext,
		encoded,
		localRunHook(t, runContext.RepoRoot(), "session-two"),
		&first,
	)
	if err != nil {
		t.Fatal(err)
	}
	if conflict.Verified() || conflict.Lineage.UnlinkedReasonCode != ReasonSessionConflict {
		t.Fatalf("conflict correlation = %+v", conflict)
	}
}

func TestLocalRunAdapterIsInjectedAndRetainsNoRawPayload(t *testing.T) {
	frozen := testFrozenWorkSpec(t)
	hook := localRunHook(t, frozen.Spec().Repository.Root, "session-root")
	launcher := &recordingLauncher{result: ProcessResult{
		Outcome:       localrun.ProviderCompleted,
		ExitCode:      0,
		LifecycleHook: hook,
	}}
	adapter, err := NewLocalRunAdapter(launcher, []string{"PATH=/usr/bin"})
	if err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	observation := adapter.Launch(context.Background(), localrun.LaunchRequest{
		RunID:     "run-one",
		WorkSpec:  frozen,
		Arguments: []string{"--sandbox", "workspace-write"},
		Stdout:    &stdout,
	})
	if observation.Outcome != localrun.ProviderCompleted ||
		observation.IdentityStatus != localrun.IdentityVerified ||
		observation.RootSessionRef != "session-root" {
		t.Fatalf("observation = %+v", observation)
	}
	if launcher.calls != 1 {
		t.Fatalf("launcher calls = %d", launcher.calls)
	}
	if strings.Join(launcher.request.Arguments, " ") != "--sandbox workspace-write" {
		t.Fatalf("arguments changed: %v", launcher.request.Arguments)
	}
	if launcher.request.Directory != frozen.Spec().Repository.Root {
		t.Fatalf("directory = %q", launcher.request.Directory)
	}
	if len(launcher.request.Environment) != 2 ||
		!strings.HasPrefix(launcher.request.Environment[1], LocalRunContextEnvironment+"=") {
		t.Fatalf("environment = %v", launcher.request.Environment)
	}
	encoded, err := json.Marshal(observation)
	if err != nil {
		t.Fatal(err)
	}
	for _, prohibited := range []string{"workspace-write", "Implement one bounded task", "SessionStart"} {
		if bytes.Contains(encoded, []byte(prohibited)) {
			t.Fatalf("observation retained %q", prohibited)
		}
	}
}

func TestLocalRunAdapterRejectsCredentialEnvironmentAndMissingIdentity(t *testing.T) {
	launcher := &recordingLauncher{result: ProcessResult{
		Outcome: localrun.ProviderCompleted,
	}}
	if _, err := NewLocalRunAdapter(launcher, []string{"API_KEY=secret"}); err == nil {
		t.Fatal("expected credential-shaped environment rejection")
	}
	adapter, err := NewLocalRunAdapter(launcher, nil)
	if err != nil {
		t.Fatal(err)
	}
	observation := adapter.Launch(context.Background(), localrun.LaunchRequest{
		RunID:    "run-one",
		WorkSpec: testFrozenWorkSpec(t),
	})
	if observation.IdentityStatus != localrun.IdentityUnlinked ||
		observation.Reason != ReasonInvalidHookInput {
		t.Fatalf("missing identity observation = %+v", observation)
	}
}

func testLocalRunContext(t *testing.T) LocalRunContext {
	t.Helper()
	root, err := filepath.Abs(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	runContext, err := NewLocalRunContext(
		"work-dogfood",
		domain.WorkSpecDigest("sha256:"+strings.Repeat("a", 64)),
		"run-one",
		root,
	)
	if err != nil {
		t.Fatal(err)
	}
	return runContext
}

func testFrozenWorkSpec(t *testing.T) domain.FrozenWorkSpec {
	t.Helper()
	root, err := filepath.Abs(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	frozen, err := domain.FreezeWorkSpec(domain.WorkSpec{
		Schema:  domain.WorkSpecSchemaV0,
		ID:      "work-dogfood",
		Version: 1,
		Repository: domain.WorkSpecRepository{
			Root:         root,
			BaseRevision: strings.Repeat("a", 40),
		},
		Intent: domain.WorkSpecIntentReference{
			ID:          "intent-dogfood",
			Version:     3,
			ArtifactRef: "intent.md",
			Digest:      "sha256:" + strings.Repeat("b", 64),
		},
		Task:  "Implement one bounded task.",
		Paths: []string{"internal/localrun"},
		Checks: []domain.WorkSpecCheck{{
			ID:   "test",
			Argv: []string{"go", "test", "./..."},
		}},
		StopConditions: []domain.WorkSpecStopCondition{{
			ID:          "receipt",
			Description: "Stop after receipt.",
		}},
		Posture: domain.PostureTrustedLocalProviderEnforcedV0,
	})
	if err != nil {
		t.Fatal(err)
	}
	return frozen
}

func localRunHook(t *testing.T, root, sessionID string) []byte {
	t.Helper()
	raw, err := json.Marshal(map[string]any{
		"session_id":      sessionID,
		"cwd":             root,
		"hook_event_name": "SessionStart",
		"source":          "startup",
	})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
