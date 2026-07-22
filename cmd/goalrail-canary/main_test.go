package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

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
		"--reason", "eligibility-confirmed",
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
		"freeze-checks",
		"--change", "change-cli-1",
		"--actor", "operator",
		"--source", "request:checks-cli-1",
		"--check-ref", "check:go-test",
	), bytes.NewReader(nil), &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("freeze checks command: %v", err)
	}
	if err := run(append(global,
		"correct",
		"--kind", "checks",
		"--change", "change-cli-1",
		"--actor", "operator",
		"--source", "review:checks-correction-cli-1",
		"--reason", "missing-ci",
		"--check-ref", "check:go-test",
		"--check-ref", "ci:required",
	), bytes.NewReader(nil), &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("correct checks command: %v", err)
	}
	if err := run(append(global,
		"deliver",
		"--change", "change-cli-1",
		"--actor", "operator",
		"--source", "review:handoff-cli-1",
		"--check-ref", "check:go-test",
		"--check-ref", "ci:required",
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
	if view.CheckSet == nil || len(view.CheckSet.CheckRefs) != 2 {
		t.Fatalf("effective frozen checks missing from view: %#v", view.CheckSet)
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
		"--reason", "eligibility-confirmed",
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

func TestCommandExclusionDoesNotExposeVariantOrConsumeOrdinal(t *testing.T) {
	repoRoot := t.TempDir()
	storePath := filepath.Join(repoRoot, "events.jsonl")
	global := []string{"--repo", repoRoot, "--store", storePath}
	var excludedOutput bytes.Buffer
	if err := run(append(global,
		"exclude",
		"--change", "change-cli-excluded-1",
		"--actor", "operator",
		"--source", "request:excluded-cli-1",
		"--reason", "manual-task",
		"--synthetic",
	), bytes.NewReader(nil), &excludedOutput, &bytes.Buffer{}); err != nil {
		t.Fatalf("exclude command: %v", err)
	}
	for _, forbidden := range []string{"ordinal", "variant", "run_id"} {
		if strings.Contains(excludedOutput.String(), forbidden) {
			t.Fatalf("exclusion receipt disclosed %s: %s", forbidden, excludedOutput.String())
		}
	}
	var excludedViewOutput bytes.Buffer
	if err := run(append(global, "inspect", "--change", "change-cli-excluded-1"), bytes.NewReader(nil), &excludedViewOutput, &bytes.Buffer{}); err != nil {
		t.Fatalf("inspect exclusion: %v", err)
	}
	var excludedView operator.ChangeView
	if err := json.Unmarshal(excludedViewOutput.Bytes(), &excludedView); err != nil {
		t.Fatalf("decode exclusion view: %v", err)
	}
	if excludedView.Admission == nil || excludedView.Admission.Decision != domain.AdmissionExcluded || excludedView.Assignment != nil {
		t.Fatalf("unexpected exclusion view: %#v", excludedView)
	}

	start := func(change string) operator.StartReceipt {
		t.Helper()
		var output bytes.Buffer
		if err := run(append(global,
			"start",
			"--change", change,
			"--intent-version", "1",
			"--actor", "operator",
			"--source", "request:"+change,
			"--reason", "eligibility-confirmed",
			"--synthetic",
		), bytes.NewReader(nil), &output, &bytes.Buffer{}); err != nil {
			t.Fatalf("start %s: %v", change, err)
		}
		var receipt operator.StartReceipt
		if err := json.Unmarshal(output.Bytes(), &receipt); err != nil {
			t.Fatalf("decode %s receipt: %v", change, err)
		}
		return receipt
	}
	first := start("change-cli-eligible-1")
	second := start("change-cli-eligible-2")
	if first.Ordinal != 1 || second.Ordinal != 2 {
		t.Fatalf("exclusion consumed ordinal: first=%#v second=%#v", first, second)
	}
	var reportOutput bytes.Buffer
	if err := run(append(global, "report"), bytes.NewReader(nil), &reportOutput, &bytes.Buffer{}); err != nil {
		t.Fatalf("report admissions: %v", err)
	}
	var report domain.CanaryReport
	if err := json.Unmarshal(reportOutput.Bytes(), &report); err != nil {
		t.Fatalf("decode admissions report: %v", err)
	}
	if report.Assigned != 2 || report.Excluded != 1 || report.ExclusionReasons["manual-task"] != 1 {
		t.Fatalf("unexpected admissions report: %#v", report)
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
		"--reason", "eligibility-confirmed",
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
	if len(events) != 1 || events[0].ChangeID != "change-default-store-1" ||
		events[0].Admission == nil || events[0].Admission.Decision != domain.AdmissionEligible {
		t.Fatalf("unexpected default store events: %#v", events)
	}
	if _, err := os.Stat(filepath.Join(repoRoot, "openspec")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("default store recreated OpenSpec lifecycle path: %v", err)
	}
}

func TestCommandManifestV2UsesExplicitVersionAndSeparateDefaultStore(t *testing.T) {
	repoRoot := t.TempDir()
	var output bytes.Buffer
	if err := run([]string{
		"--repo", repoRoot,
		"--manifest-version", "2",
		"start",
		"--change", "change-v2-store-1",
		"--intent-version", "1",
		"--actor", "operator",
		"--source", "request:v2-store",
		"--reason", "eligibility-confirmed",
		"--synthetic",
	}, bytes.NewReader(nil), &output, &bytes.Buffer{}); err != nil {
		t.Fatalf("start v2 with default store: %v", err)
	}
	var receipt operator.StartReceipt
	if err := json.Unmarshal(output.Bytes(), &receipt); err != nil {
		t.Fatalf("decode v2 receipt: %v", err)
	}
	if receipt.ManifestVersion != 2 {
		t.Fatalf("v2 receipt manifest version = %d", receipt.ManifestVersion)
	}

	v2Path := filepath.Join(repoRoot, filepath.FromSlash(defaultEvidenceV2RelativePath))
	store, err := evidence.NewStoreForManifest(v2Path, 2)
	if err != nil {
		t.Fatalf("open v2 store: %v", err)
	}
	events, err := store.ReadAll()
	if err != nil {
		t.Fatalf("read v2 store: %v", err)
	}
	if len(events) != 1 || events[0].ManifestVersion != 2 ||
		events[0].Assignment == nil || events[0].Assignment.ManifestVersion != 2 {
		t.Fatalf("unexpected v2 evidence: %#v", events)
	}
	if _, err := os.Stat(filepath.Join(repoRoot, filepath.FromSlash(defaultEvidenceRelativePath))); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("v2 command touched v1 evidence path: %v", err)
	}
}

func TestCommandRecordsV2ContextBasisAndPhase(t *testing.T) {
	repoRoot := t.TempDir()
	storePath := filepath.Join(repoRoot, "events-v2.jsonl")
	store, err := evidence.NewStoreForManifest(storePath, 2)
	if err != nil {
		t.Fatalf("create v2 store: %v", err)
	}
	service, err := operator.NewServiceForManifest(store, repoRoot, 2)
	if err != nil {
		t.Fatalf("create v2 service: %v", err)
	}
	receipt, err := service.Start(operator.StartInput{
		EventID: "event-cli-v2-setup-start", ChangeID: "change-cli-v2-setup", RunID: "run-cli-v2-setup",
		IntentVersion: 1, OccurredAt: time.Now().UTC().Add(-time.Minute), Actor: "operator",
		SourceRef: "request:cli-v2-setup", Reason: "eligibility-confirmed", Synthetic: true,
	})
	if err != nil {
		t.Fatalf("start v2 setup: %v", err)
	}
	global := []string{"--repo", repoRoot, "--store", storePath, "--manifest-version", "2"}
	if err := run(append(global,
		"bind-context", "--change", string(receipt.ChangeID), "--actor", "operator",
		"--source", "openspec:cli-v2-setup", "--context-pack", "context-cli-v2-setup", "--context-version", "1",
	), bytes.NewReader(nil), &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("bind v2 context: %v", err)
	}
	if err := run(append(global,
		"record-basis", "--change", string(receipt.ChangeID), "--actor", "owner",
		"--source", "owner-review:cli-v2-basis", "--intent-ref", "openspec:cli-v2-setup",
		"--intent-id", "intent-cli-v2-setup", "--intent-version", "1", "--timing", "pre_execution",
		"--desired-outcome-id", "OUT-1", "--non-goal-id", "NG-1", "--success-signal-id", "SIG-1",
	), bytes.NewReader(nil), &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("record v2 basis: %v", err)
	}
	events, err := store.ReadAll()
	if err != nil {
		t.Fatalf("read setup events: %v", err)
	}
	var basisAt time.Time
	for _, event := range events {
		if event.AssessmentBasis != nil {
			basisAt = event.OccurredAt
		}
	}
	if basisAt.IsZero() {
		t.Fatal("basis command did not append evidence")
	}
	phaseStart, phaseEnd := basisAt.Add(time.Nanosecond), basisAt.Add(2*time.Nanosecond)
	if err := run(append(global,
		"record-phase", "--change", string(receipt.ChangeID), "--actor", "operator",
		"--source", "review:cli-v2-phase", "--started-at", phaseStart.Format(time.RFC3339Nano),
		"--completed-at", phaseEnd.Format(time.RFC3339Nano),
	), bytes.NewReader(nil), &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("record v2 phase: %v", err)
	}
	var inspect bytes.Buffer
	if err := run(append(global, "inspect", "--change", string(receipt.ChangeID)), bytes.NewReader(nil), &inspect, &bytes.Buffer{}); err != nil {
		t.Fatalf("inspect v2 setup: %v", err)
	}
	var view operator.ChangeView
	if err := json.Unmarshal(inspect.Bytes(), &view); err != nil {
		t.Fatalf("decode v2 setup: %v", err)
	}
	if view.Context == nil || view.AssessmentBasis == nil || view.FlowPhase == nil ||
		view.AssessmentBasis.Timing != domain.BasisPreExecution {
		t.Fatalf("v2 setup commands did not project artifacts: %#v", view)
	}
}

func TestCommandReconcilesV2TelemetryWithoutPersistingCredentials(t *testing.T) {
	repoRoot := t.TempDir()
	storePath := filepath.Join(repoRoot, "events-v2.jsonl")
	store, err := evidence.NewStoreForManifest(storePath, 2)
	if err != nil {
		t.Fatalf("create v2 store: %v", err)
	}
	service, err := operator.NewServiceForManifest(store, repoRoot, 2)
	if err != nil {
		t.Fatalf("create v2 service: %v", err)
	}
	baseTime := time.Now().UTC().Add(-10 * time.Minute).Truncate(time.Second)
	receipt, err := service.Start(operator.StartInput{
		EventID: "event-cli-reconcile-start", ChangeID: "change-cli-reconcile", RunID: "run-cli-reconcile",
		IntentVersion: 1, OccurredAt: baseTime, Actor: "operator", SourceRef: "request:cli-reconcile",
		Reason: "eligibility-confirmed", Synthetic: true,
	})
	if err != nil {
		t.Fatalf("start v2 flow: %v", err)
	}
	if err := service.RecordContextBinding(operator.ContextBindingInput{
		EventID: "event-cli-reconcile-context", ChangeID: receipt.ChangeID, OccurredAt: baseTime.Add(time.Minute),
		Actor: "operator", SourceRef: "openspec:cli-reconcile", ContextPackID: "context-cli-reconcile", ContextPackVersion: 1,
	}); err != nil {
		t.Fatalf("record context: %v", err)
	}
	if err := service.RecordAssessmentBasis(operator.AssessmentBasisInput{
		EventID: "event-cli-reconcile-basis", ChangeID: receipt.ChangeID, OccurredAt: baseTime.Add(2 * time.Minute),
		Actor: "owner", SourceRef: "owner-review:cli-reconcile-basis",
		Basis: domain.CanaryAssessmentBasis{
			IntentRef: "openspec:cli-reconcile", IntentID: "intent-cli-reconcile", IntentVersion: 1,
			Timing: domain.BasisPreExecution, DesiredOutcomeIDs: []domain.IntentItemID{"OUT-1"},
			NonGoalIDs: []domain.IntentItemID{"NG-1"}, SuccessSignalIDs: []domain.IntentItemID{"SIG-1"},
		},
	}); err != nil {
		t.Fatalf("record basis: %v", err)
	}
	phaseStart, phaseEnd := baseTime.Add(3*time.Minute), baseTime.Add(5*time.Minute)
	if err := service.RecordFlowPhase(operator.FlowPhaseInput{
		EventID: "event-cli-reconcile-phase", ChangeID: receipt.ChangeID, OccurredAt: phaseEnd,
		Actor: "operator", SourceRef: "review:cli-reconcile-phase", StartedAt: phaseStart, CompletedAt: phaseEnd,
	}); err != nil {
		t.Fatalf("record phase: %v", err)
	}
	lineage := domain.ExecutionLineage{
		Status: domain.LineageVerified, ChangeID: receipt.ChangeID, RunID: receipt.RunID,
		RootSessionID: "session-cli-reconcile", IdentitySource: domain.SessionIdentityLifecycleHook,
		ContextDigest: strings.Repeat("c", 64),
	}
	if err := store.Append(domain.EvidenceEvent{
		ID: "event-cli-reconcile-lineage", CanaryID: domain.IntentCanaryV0ManifestID, ManifestVersion: 2,
		ChangeID: receipt.ChangeID, Kind: domain.EventLineageRecorded, OccurredAt: baseTime.Add(6 * time.Minute),
		Actor: "goalrail-hook", SourceRef: "codex-hook:lifecycle",
		ObservationRefs: []domain.EvidenceReference{"langfuse-session:session-cli-reconcile"},
		Lineage:         &lineage, LineageResolutionAttempts: 1,
	}); err != nil {
		t.Fatalf("append lineage: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		username, password, ok := request.BasicAuth()
		if !ok || username != "public-cli-test" || password != "secret-cli-test" {
			t.Errorf("unexpected Langfuse authentication")
		}
		fmt.Fprintf(writer, `{"data":[{"id":"observation-cli-1","traceId":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","startTime":%q,"endTime":%q,"projectId":"project-cli","parentObservationId":null,"type":"SPAN","name":"Codex Turn","level":"DEFAULT","statusMessage":null,"version":null,"environment":"development","bookmarked":false,"public":false,"userId":null,"sessionId":"session-cli-reconcile"}],"meta":{"cursor":null}}`, phaseStart.Add(10*time.Second).Format(time.RFC3339Nano), phaseStart.Add(70*time.Second).Format(time.RFC3339Nano))
	}))
	defer server.Close()
	t.Setenv(langfuseBaseURLEnvironment, server.URL)
	t.Setenv(langfusePublicKeyEnvironment, "public-cli-test")
	t.Setenv(langfuseSecretKeyEnvironment, "secret-cli-test")
	ownerStart, ownerEnd := phaseEnd, phaseEnd.Add(30*time.Second)
	var output bytes.Buffer
	if err := run([]string{
		"--repo", repoRoot, "--store", storePath, "--manifest-version", "2", "reconcile",
		"--change", string(receipt.ChangeID), "--actor", "operator", "--source", "langfuse-api:observations-v2",
		"--owner-review-ref", "review:cli-owner-review",
		"--owner-review-start", ownerStart.Format(time.RFC3339Nano),
		"--owner-review-end", ownerEnd.Format(time.RFC3339Nano),
	}, bytes.NewReader(nil), &output, &bytes.Buffer{}); err != nil {
		t.Fatalf("reconcile command: %v", err)
	}
	var reconciled operator.ReconcileTelemetryReceipt
	if err := json.Unmarshal(output.Bytes(), &reconciled); err != nil {
		t.Fatalf("decode reconcile receipt: %v", err)
	}
	if reconciled.Telemetry.FlowOverhead == nil || reconciled.Telemetry.FlowOverhead.TotalMinutes != 1.5 {
		t.Fatalf("unexpected CLI reconciliation: %#v", reconciled)
	}
	global := []string{"--repo", repoRoot, "--store", storePath, "--manifest-version", "2"}
	if err := run(append(global,
		"freeze-checks", "--change", string(receipt.ChangeID), "--actor", "operator",
		"--source", "review:cli-v2-checks", "--check-ref", "test:go-test",
	), bytes.NewReader(nil), &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("freeze v2 checks: %v", err)
	}
	if err := run(append(global,
		"deliver", "--change", string(receipt.ChangeID), "--actor", "operator",
		"--source", "review:cli-v2-delivery", "--check-ref", "test:go-test", "--green",
	), bytes.NewReader(nil), &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("deliver v2 flow: %v", err)
	}
	if err := run(append(global,
		"assess", "--change", string(receipt.ChangeID), "--owner", "owner",
		"--source", "owner-review:cli-v2-assessment", "--outcome", "match", "--repeat-opt-in", "yes",
		"--desired-outcome", "OUT-1=achieved", "--non-goal", "NG-1=preserved",
		"--success-signal", "SIG-1=observed",
	), bytes.NewReader(nil), &bytes.Buffer{}, &bytes.Buffer{}); err != nil {
		t.Fatalf("assess v2 flow: %v", err)
	}
	var inspect bytes.Buffer
	if err := run(append(global, "inspect", "--change", string(receipt.ChangeID)), bytes.NewReader(nil), &inspect, &bytes.Buffer{}); err != nil {
		t.Fatalf("inspect v2 flow: %v", err)
	}
	var view operator.ChangeView
	if err := json.Unmarshal(inspect.Bytes(), &view); err != nil {
		t.Fatalf("decode v2 view: %v", err)
	}
	if view.Assessment == nil || len(view.Assessment.ItemJudgments) != 3 || view.Assessment.Outcome != domain.IntentMatch {
		t.Fatalf("CLI did not project structured assessment: %#v", view.Assessment)
	}
	serialized, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read v2 evidence: %v", err)
	}
	for _, secret := range []string{"public-cli-test", "secret-cli-test"} {
		if bytes.Contains(serialized, []byte(secret)) {
			t.Fatalf("evidence persisted credential %q", secret)
		}
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
		"--reason", "eligibility-confirmed",
	}, bytes.NewReader(nil), &bytes.Buffer{}, &bytes.Buffer{})
	if !errors.Is(err, operator.ErrRealCanaryNotActivated) {
		t.Fatalf("real start error = %v, want ErrRealCanaryNotActivated", err)
	}
	err = run([]string{
		"--repo", repoRoot,
		"exclude",
		"--change", "change-real-excluded-default-store-1",
		"--actor", "operator",
		"--source", "request:real-excluded-default-store",
		"--reason", "manual-task",
	}, bytes.NewReader(nil), &bytes.Buffer{}, &bytes.Buffer{})
	if !errors.Is(err, operator.ErrRealCanaryNotActivated) {
		t.Fatalf("real exclusion error = %v, want ErrRealCanaryNotActivated", err)
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
