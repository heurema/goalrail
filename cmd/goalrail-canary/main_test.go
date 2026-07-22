package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/heurema/goalrail/internal/domain"
	"github.com/heurema/goalrail/internal/evidence"
	"github.com/heurema/goalrail/internal/operator"
)

func TestCommandRunsSyntheticLifecycleWithoutManualSessionID(t *testing.T) {
	repoRoot := t.TempDir()
	storePath := filepath.Join(repoRoot, "canary", "events.jsonl")
	global := []string{"--repo", repoRoot, "--store", storePath}

	var startOutput bytes.Buffer
	if err := run(append(global,
		"start",
		"--change", "change-cli-1",
		"--intent-version", "1",
		"--actor", "operator",
		"--source", "request:synthetic-cli",
		"--synthetic",
	), bytes.NewReader(nil), &startOutput, &bytes.Buffer{}); err != nil {
		t.Fatalf("start command: %v", err)
	}
	var receipt operator.StartReceipt
	if err := json.Unmarshal(startOutput.Bytes(), &receipt); err != nil {
		t.Fatalf("decode start receipt: %v\n%s", err, startOutput.String())
	}
	if receipt.ChangeID != "change-cli-1" || receipt.RunContextEnv == "" {
		t.Fatalf("unexpected start receipt: %#v", receipt)
	}

	t.Setenv(runContextEnvironment, receipt.RunContextEnv)
	t.Setenv(evidencePathEnvironment, storePath)
	t.Setenv(repoRootEnvironment, repoRoot)
	hook := fmt.Sprintf(
		`{"session_id":"session-cli-1","cwd":%q,"hook_event_name":"SessionStart","source":"startup","transcript_path":"/private/raw-transcript.jsonl","prompt":"do not retain this raw prompt","authorization":"Bearer raw-hook-credential"}`,
		repoRoot,
	)
	if err := run([]string{"bind-hook"}, bytes.NewBufferString(hook), &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("bind-hook command: %v", err)
	}
	if err := run(append(global,
		"correct",
		"--kind", "material",
		"--change", "change-cli-1",
		"--owner", "owner",
		"--source", "review:intent-correction",
		"--reason", "outcome-misunderstood",
	), bytes.NewReader(nil), &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("material correction command: %v", err)
	}
	if err := run(append(global,
		"deliver",
		"--change", "change-cli-1",
		"--actor", "operator",
		"--source", "review:handoff-cli-1",
		"--check-ref", "check:go-test",
		"--green",
		"--overhead-minutes", "4.25",
	), bytes.NewReader(nil), &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("deliver command: %v", err)
	}
	if err := run(append(global,
		"assess",
		"--change", "change-cli-1",
		"--owner", "owner",
		"--source", "review:owner-assessment-cli-1",
		"--outcome", "partial",
		"--repeat-opt-in", "yes",
	), bytes.NewReader(nil), &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("assess command: %v", err)
	}
	if err := run(append(global,
		"correct",
		"--kind", "assessment",
		"--change", "change-cli-1",
		"--owner", "owner",
		"--source", "review:assessment-correction-cli-1",
		"--reason", "owner-outcome-correction",
		"--outcome", "match",
		"--repeat-opt-in", "no",
	), bytes.NewReader(nil), &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("assessment correction command: %v", err)
	}

	var inspectOutput bytes.Buffer
	if err := run(append(global, "inspect", "--change", "change-cli-1"), bytes.NewReader(nil), &inspectOutput, &bytes.Buffer{}); err != nil {
		t.Fatalf("inspect command: %v", err)
	}
	var view operator.ChangeView
	if err := json.Unmarshal(inspectOutput.Bytes(), &view); err != nil {
		t.Fatalf("decode inspect output: %v", err)
	}
	if view.Lineage == nil || view.Lineage.RootSessionID != domain.SessionID("session-cli-1") {
		t.Fatalf("provider hook identity missing from view: %#v", view.Lineage)
	}
	if view.Assessment == nil || view.Assessment.Outcome != domain.IntentMatch ||
		!view.Assessment.MaterialCorrectionBeforeDelivery {
		t.Fatalf("latest explicit assessment missing from view: %#v", view.Assessment)
	}
	var reportOutput bytes.Buffer
	if err := run(append(global, "report"), bytes.NewReader(nil), &reportOutput, &bytes.Buffer{}); err != nil {
		t.Fatalf("report command: %v", err)
	}
	if !strings.Contains(reportOutput.String(), `"completion_ready"`) ||
		strings.Contains(reportOutput.String(), `"CompletionReady"`) {
		t.Fatalf("report JSON does not use stable snake_case fields: %s", reportOutput.String())
	}
	var report domain.CanaryReport
	if err := json.Unmarshal(reportOutput.Bytes(), &report); err != nil {
		t.Fatalf("decode report output: %v", err)
	}
	if report.Verdict != domain.CanaryVerdictPending || report.Assigned != 1 ||
		report.Flow.Assigned != 1 || report.Flow.MaterialPreventions != 1 ||
		report.LineageVerified != 1 || report.Flow.RepeatOptInNo != 1 {
		t.Fatalf("unexpected projected report: %#v", report)
	}
	beforeDisable, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read evidence before disable: %v", err)
	}
	var disableOutput bytes.Buffer
	if err := run(append(global,
		"disable",
		"--actor", "owner",
		"--source", "review:synthetic-rollback-cli-v1",
		"--reason", "rollback-exercise",
	), bytes.NewReader(nil), &disableOutput, &bytes.Buffer{}); err != nil {
		t.Fatalf("disable command: %v", err)
	}
	var disableReceipt operator.DisableReceipt
	if err := json.Unmarshal(disableOutput.Bytes(), &disableReceipt); err != nil {
		t.Fatalf("decode disable receipt: %v", err)
	}
	if !disableReceipt.AssignmentsStopped || disableReceipt.Reason != "rollback-exercise" {
		t.Fatalf("unexpected disable receipt: %#v", disableReceipt)
	}
	afterDisable, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read evidence after disable: %v", err)
	}
	if !bytes.HasPrefix(afterDisable, beforeDisable) || len(afterDisable) <= len(beforeDisable) {
		t.Fatal("disable did not preserve prior evidence as an exact byte prefix")
	}
	err = run(append(global,
		"start",
		"--change", "change-cli-after-stop",
		"--intent-version", "1",
		"--actor", "operator",
		"--source", "request:synthetic-after-stop",
		"--synthetic",
	), bytes.NewReader(nil), &bytes.Buffer{}, &bytes.Buffer{})
	if !errors.Is(err, operator.ErrCanaryStopped) {
		t.Fatalf("start after disable error = %v, want ErrCanaryStopped", err)
	}
	afterRejectedStart, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read evidence after rejected start: %v", err)
	}
	if !bytes.Equal(afterRejectedStart, afterDisable) {
		t.Fatal("rejected start mutated the stopped evidence chain")
	}
	var stoppedReportOutput bytes.Buffer
	if err := run(append(global, "report"), bytes.NewReader(nil), &stoppedReportOutput, &bytes.Buffer{}); err != nil {
		t.Fatalf("stopped report command: %v", err)
	}
	var stoppedReport domain.CanaryReport
	if err := json.Unmarshal(stoppedReportOutput.Bytes(), &stoppedReport); err != nil {
		t.Fatalf("decode stopped report: %v", err)
	}
	if stoppedReport.Verdict != domain.CanaryVerdictStop || !stoppedReport.AssignmentsStopped || stoppedReport.Assigned != 1 {
		t.Fatalf("unexpected stopped report: %#v", stoppedReport)
	}
	store, err := evidence.NewStore(storePath)
	if err != nil {
		t.Fatalf("open evidence store: %v", err)
	}
	if err := store.Verify(); err != nil {
		t.Fatalf("verify CLI evidence: %v", err)
	}
	encodedEvidence, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read serialized CLI evidence: %v", err)
	}
	for _, forbidden := range []string{
		"transcript_path",
		"raw-transcript",
		"do not retain this raw prompt",
		"authorization",
		"raw-hook-credential",
	} {
		if strings.Contains(string(encodedEvidence), forbidden) {
			t.Fatalf("serialized evidence retained forbidden provider payload %q", forbidden)
		}
	}
}

func TestCommandDefaultStoreIsIndependentOfOpenSpecChangeLifecycle(t *testing.T) {
	repoRoot := t.TempDir()
	var output bytes.Buffer
	if err := run([]string{
		"--repo", repoRoot,
		"start",
		"--change", "change-default-store-1",
		"--intent-version", "1",
		"--actor", "operator",
		"--source", "request:default-store",
		"--synthetic",
	}, bytes.NewReader(nil), &output, &bytes.Buffer{}); err != nil {
		t.Fatalf("start with default store: %v", err)
	}

	storePath := filepath.Join(repoRoot, filepath.FromSlash(defaultEvidenceRelativePath))
	store, err := evidence.NewStore(storePath)
	if err != nil {
		t.Fatalf("open default store: %v", err)
	}
	events, err := store.ReadAll()
	if err != nil {
		t.Fatalf("read default store: %v", err)
	}
	if len(events) != 1 || events[0].ChangeID != "change-default-store-1" {
		t.Fatalf("unexpected default store events: %#v", events)
	}
	if _, err := os.Stat(filepath.Join(repoRoot, "openspec")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("default store recreated OpenSpec lifecycle path: %v", err)
	}
}

func TestCurrentOperatorGuidanceMatchesDefaultStore(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source path")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "..", ".."))
	guidancePath := filepath.Join(repoRoot, "canary", "intent-canary-v0", "operator-flow.md")
	contents, err := os.ReadFile(guidancePath)
	if err != nil {
		t.Fatalf("read current operator guidance: %v", err)
	}
	guidance := string(contents)
	if !strings.Contains(guidance, defaultEvidenceRelativePath) {
		t.Fatalf("current operator guidance is missing default path %q", defaultEvidenceRelativePath)
	}
	retiredPath := "openspec/changes/intent-canary-v0/canary/events.jsonl"
	if strings.Contains(guidance, retiredPath) {
		t.Fatalf("current operator guidance retains retired path %q", retiredPath)
	}

	archivedPath := filepath.Join(
		repoRoot,
		"openspec",
		"changes",
		"archive",
		"2026-07-22-intent-canary-v0",
		"canary",
		"operator-flow.md",
	)
	archived, err := os.ReadFile(archivedPath)
	if err != nil {
		t.Fatalf("read archived operator guidance: %v", err)
	}
	if !strings.Contains(string(archived), "Historical planning artifact") ||
		!strings.Contains(string(archived), "canary/intent-canary-v0/operator-flow.md") {
		t.Fatal("archived operator guidance is not clearly linked to the current surface")
	}
}

func TestCommandDefaultStoreRejectsRealAssignmentWithoutEvidence(t *testing.T) {
	repoRoot := t.TempDir()
	err := run([]string{
		"--repo", repoRoot,
		"start",
		"--change", "change-real-default-store-1",
		"--intent-version", "1",
		"--actor", "operator",
		"--source", "request:real-default-store",
	}, bytes.NewReader(nil), &bytes.Buffer{}, &bytes.Buffer{})
	if !errors.Is(err, operator.ErrRealCanaryNotActivated) {
		t.Fatalf("real start error = %v, want ErrRealCanaryNotActivated", err)
	}
	storePath := filepath.Join(repoRoot, filepath.FromSlash(defaultEvidenceRelativePath))
	if _, statErr := os.Stat(storePath); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("rejected real start created repository evidence: %v", statErr)
	}
}

func TestBindCommandsRejectManualIdentityArguments(t *testing.T) {
	repoRoot := t.TempDir()
	stderr := &bytes.Buffer{}
	err := run(
		[]string{"--repo", repoRoot, "bind-hook", "--session-id", "typed-by-person"},
		bytes.NewBufferString(`{"session_id":"provider-session"}`),
		&bytes.Buffer{},
		stderr,
	)
	if err == nil {
		t.Fatal("bind-hook accepted a manual session argument")
	}
}
