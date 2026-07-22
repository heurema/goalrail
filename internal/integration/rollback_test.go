package integration

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/heurema/goalrail/internal/adapters/codex"
	"github.com/heurema/goalrail/internal/adapters/langfuse"
	"github.com/heurema/goalrail/internal/domain"
	"github.com/heurema/goalrail/internal/evidence"
	"github.com/heurema/goalrail/internal/operator"
)

func TestDisableIsRepositoryLocalAndLeavesUnrelatedAdaptersUsable(t *testing.T) {
	repoRoot := t.TempDir()
	configPath := filepath.Join(repoRoot, ".codex", "langfuse.json")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o750); err != nil {
		t.Fatalf("create local config directory: %v", err)
	}
	configBytes := []byte("{\n  \"tags\": [\"project:goalrail-test\"]\n}\n")
	if err := os.WriteFile(configPath, configBytes, 0o600); err != nil {
		t.Fatalf("write local config sentinel: %v", err)
	}

	storePath := filepath.Join(repoRoot, "canary", "events.jsonl")
	store, err := evidence.NewStore(storePath)
	if err != nil {
		t.Fatalf("create rollback evidence store: %v", err)
	}
	service, err := operator.NewService(store, repoRoot)
	if err != nil {
		t.Fatalf("create rollback operator service: %v", err)
	}
	if _, err := service.Start(operator.StartInput{
		EventID:       "event-rollback-start-v1",
		ChangeID:      "change-rollback-v1",
		RunID:         "run-rollback-v1",
		IntentVersion: 1,
		OccurredAt:    syntheticE2ETime,
		Actor:         "synthetic-operator",
		SourceRef:     "request:synthetic-rollback-v1",
		Reason:        "eligibility-confirmed",
		Synthetic:     true,
	}); err != nil {
		t.Fatalf("start rollback fixture: %v", err)
	}
	beforeStop, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read evidence before rollback: %v", err)
	}
	if _, err := service.Disable(operator.DisableInput{
		EventID:    "event-rollback-stop-v1",
		OccurredAt: syntheticE2ETime.Add(time.Minute),
		Actor:      "synthetic-owner",
		SourceRef:  "review:synthetic-rollback-v1",
		Reason:     "rollback-exercise",
	}); err != nil {
		t.Fatalf("disable rollback fixture: %v", err)
	}
	afterStop, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read evidence after rollback: %v", err)
	}
	if !bytes.HasPrefix(afterStop, beforeStop) || len(afterStop) <= len(beforeStop) {
		t.Fatal("rollback did not preserve prior evidence byte-for-byte")
	}
	if _, err := service.Start(operator.StartInput{
		EventID:       "event-rollback-rejected-v1",
		ChangeID:      "change-rollback-rejected-v1",
		RunID:         "run-rollback-rejected-v1",
		IntentVersion: 1,
		OccurredAt:    syntheticE2ETime.Add(2 * time.Minute),
		Actor:         "synthetic-operator",
		SourceRef:     "request:synthetic-rollback-rejected-v1",
		Reason:        "eligibility-confirmed",
		Synthetic:     true,
	}); !errors.Is(err, operator.ErrCanaryStopped) {
		t.Fatalf("new assignment after rollback error = %v, want ErrCanaryStopped", err)
	}
	afterRejectedStart, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read evidence after rejected assignment: %v", err)
	}
	if !bytes.Equal(afterRejectedStart, afterStop) {
		t.Fatal("rejected assignment changed the stopped evidence chain")
	}
	if err := store.Verify(); err != nil {
		t.Fatalf("verify rollback evidence chain: %v", err)
	}
	report, err := service.Report()
	if err != nil {
		t.Fatalf("report rollback fixture: %v", err)
	}
	if report.Verdict != domain.CanaryVerdictStop || !report.AssignmentsStopped || report.Assigned != 1 {
		t.Fatalf("unexpected rollback report: %#v", report)
	}

	unchangedConfig, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read local config sentinel after rollback: %v", err)
	}
	if !bytes.Equal(unchangedConfig, configBytes) {
		t.Fatal("rollback changed repository-local Langfuse configuration")
	}

	// The stop marker belongs only to the selected evidence store. It does not
	// mutate process environment or disable the provider-neutral adapters used by
	// unrelated Codex and Langfuse work.
	unrelatedRepo := t.TempDir()
	unrelatedContext, err := codex.NewRunContext("unrelated-change", "unrelated-run", unrelatedRepo)
	if err != nil {
		t.Fatalf("create unrelated Codex run context after rollback: %v", err)
	}
	if _, err := unrelatedContext.EnvironmentValue(); err != nil {
		t.Fatalf("encode unrelated Codex run context after rollback: %v", err)
	}
	if _, err := langfuse.BuildObservationReferences(domain.ExecutionLineage{
		Status:         domain.LineageVerified,
		ChangeID:       "unrelated-change",
		RunID:          "unrelated-run",
		RootSessionID:  "unrelated-session",
		IdentitySource: domain.SessionIdentityLifecycleHook,
		ContextDigest:  strings.Repeat("a", 64),
	}, nil); err != nil {
		t.Fatalf("use unrelated Langfuse adapter after rollback: %v", err)
	}
}
