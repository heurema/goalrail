package operator

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/heurema/goalrail/internal/adapters/codex"
	"github.com/heurema/goalrail/internal/domain"
	"github.com/heurema/goalrail/internal/evidence"
)

var operatorTestTime = time.Date(2026, time.July, 21, 14, 0, 0, 0, time.UTC)

func TestSyntheticFlowLifecycleKeepsOwnerAssessmentExplicitAndAppendOnly(t *testing.T) {
	service, store, repoRoot := newTestService(t)
	receipt := startSynthetic(t, service, "change-flow-1", "run-flow-1", operatorTestTime)
	if receipt.Ordinal != 1 || receipt.Variant != domain.VariantFlow {
		t.Fatalf("unexpected first assignment: %#v", receipt)
	}

	hook := []byte(fmt.Sprintf(
		`{"session_id":"session-flow-1","cwd":%q,"hook_event_name":"SessionStart","source":"startup"}`,
		repoRoot,
	))
	correlation, err := service.BindLifecycleHook(HookInput{
		EventID:        "event-lineage-1",
		OccurredAt:     operatorTestTime.Add(time.Minute),
		Actor:          "goalrail-hook",
		SourceRef:      "codex-hook:lifecycle",
		EncodedContext: receipt.RunContextEnv,
		RawHook:        hook,
	})
	if err != nil {
		t.Fatalf("bind lifecycle hook: %v", err)
	}
	if correlation.Lineage.Status != domain.LineageVerified ||
		correlation.Lineage.RootSessionID != "session-flow-1" {
		t.Fatalf("unexpected correlation: %#v", correlation)
	}

	if err := service.RecordMaterialCorrection(MaterialCorrectionInput{
		EventID:    "event-material-1",
		ChangeID:   receipt.ChangeID,
		OccurredAt: operatorTestTime.Add(2 * time.Minute),
		Owner:      "owner",
		SourceRef:  "review:intent-correction",
		Reason:     "non-goal-misunderstood",
	}); err != nil {
		t.Fatalf("record material correction: %v", err)
	}
	freezeSyntheticChecks(t, service, receipt.ChangeID, operatorTestTime.Add(150*time.Second), "check:go-test")
	overhead := 12.5
	if err := service.Deliver(DeliverInput{
		EventID:             "event-delivery-1",
		ChangeID:            receipt.ChangeID,
		OccurredAt:          operatorTestTime.Add(3 * time.Minute),
		Actor:               "operator",
		SourceRef:           "review:handoff-1",
		CheckRefs:           []domain.EvidenceReference{"check:go-test"},
		ChecksGreen:         true,
		FlowOverheadMinutes: &overhead,
	}); err != nil {
		t.Fatalf("deliver flow change: %v", err)
	}
	repeatYes := true
	if err := service.Assess(AssessInput{
		EventID:     "event-assessment-1",
		ChangeID:    receipt.ChangeID,
		OccurredAt:  operatorTestTime.Add(4 * time.Minute),
		Owner:       "owner",
		SourceRef:   "review:owner-assessment-1",
		Outcome:     domain.IntentPartial,
		RepeatOptIn: &repeatYes,
	}); err != nil {
		t.Fatalf("assess flow change: %v", err)
	}

	repeatNo := false
	if err := service.CorrectAssessment(CorrectAssessmentInput{
		EventID:     "event-assessment-correction-1",
		ChangeID:    receipt.ChangeID,
		OccurredAt:  operatorTestTime.Add(5 * time.Minute),
		Owner:       "owner",
		SourceRef:   "review:owner-assessment-correction-1",
		Reason:      "owner-outcome-correction",
		Outcome:     domain.IntentMatch,
		RepeatOptIn: &repeatNo,
	}); err != nil {
		t.Fatalf("correct owner assessment: %v", err)
	}

	view, err := service.Inspect(receipt.ChangeID)
	if err != nil {
		t.Fatalf("inspect change: %v", err)
	}
	if view.EventCount != 7 || view.MaterialCorrections != 1 {
		t.Fatalf("unexpected event projection: %#v", view)
	}
	if view.Assessment == nil || view.Assessment.Outcome != domain.IntentMatch ||
		view.Assessment.ChecksGreen != true ||
		view.Assessment.MaterialCorrectionBeforeDelivery != true ||
		view.Assessment.RepeatOptIn == nil || *view.Assessment.RepeatOptIn {
		t.Fatalf("latest owner assessment was not projected: %#v", view.Assessment)
	}
	if view.Lineage == nil || view.Lineage.RunID != receipt.RunID {
		t.Fatalf("lineage does not match assignment: %#v", view.Lineage)
	}
	events, err := store.ReadAll()
	if err != nil {
		t.Fatalf("read append-only events: %v", err)
	}
	if events[5].Kind != domain.EventAssessmentRecorded ||
		events[6].SupersedesEventID != events[5].ID ||
		events[5].Assessment.Outcome != domain.IntentPartial {
		t.Fatalf("assessment history was not preserved: %#v", events[5:])
	}
	if err := store.Verify(); err != nil {
		t.Fatalf("verify evidence chain: %v", err)
	}
}

func TestLaunchReceiptAndBaselineRulesNeedNoSessionFlag(t *testing.T) {
	service, store, _ := newTestService(t)
	startSynthetic(t, service, "change-flow-1", "run-flow-1", operatorTestTime)
	startSynthetic(t, service, "change-flow-2", "run-flow-2", operatorTestTime.Add(time.Second))
	baseline := startSynthetic(t, service, "change-baseline-3", "run-baseline-3", operatorTestTime.Add(2*time.Second))
	if baseline.Ordinal != 3 || baseline.Variant != domain.VariantBaseline {
		t.Fatalf("unexpected baseline assignment: %#v", baseline)
	}

	correlation, err := service.BindLaunchReceipt(LaunchReceiptInput{
		EventID:        "event-launch-lineage-3",
		OccurredAt:     operatorTestTime.Add(time.Minute),
		Actor:          "goalrail-launcher",
		SourceRef:      "codex-app:create-thread",
		EncodedContext: baseline.RunContextEnv,
		RawReceipt:     []byte(`{"threadId":"thread-baseline-3","title":"ignored"}`),
	})
	if err != nil {
		t.Fatalf("bind launch receipt: %v", err)
	}
	if correlation.Lineage.RootSessionID != "thread-baseline-3" ||
		correlation.Lineage.IdentitySource != domain.SessionIdentityLaunchReceipt {
		t.Fatalf("unexpected launch correlation: %#v", correlation)
	}
	freezeSyntheticChecks(t, service, baseline.ChangeID, operatorTestTime.Add(90*time.Second), "check:go-test")

	overhead := 1.0
	err = service.Deliver(DeliverInput{
		EventID:             "event-invalid-baseline-delivery",
		ChangeID:            baseline.ChangeID,
		OccurredAt:          operatorTestTime.Add(2 * time.Minute),
		Actor:               "operator",
		SourceRef:           "review:handoff-3",
		CheckRefs:           []domain.EvidenceReference{"check:go-test"},
		ChecksGreen:         true,
		FlowOverheadMinutes: &overhead,
	})
	if err == nil {
		t.Fatal("baseline delivery accepted flow overhead")
	}
	if err := service.Deliver(DeliverInput{
		EventID:     "event-baseline-delivery-3",
		ChangeID:    baseline.ChangeID,
		OccurredAt:  operatorTestTime.Add(2 * time.Minute),
		Actor:       "operator",
		SourceRef:   "review:handoff-3",
		CheckRefs:   []domain.EvidenceReference{"check:go-test"},
		ChecksGreen: true,
	}); err != nil {
		t.Fatalf("deliver baseline: %v", err)
	}
	repeat := true
	if err := service.Assess(AssessInput{
		EventID:     "event-invalid-baseline-assessment",
		ChangeID:    baseline.ChangeID,
		OccurredAt:  operatorTestTime.Add(3 * time.Minute),
		Owner:       "owner",
		SourceRef:   "review:owner-assessment-3",
		Outcome:     domain.IntentMatch,
		RepeatOptIn: &repeat,
	}); !errors.Is(err, ErrAssessmentState) {
		t.Fatalf("baseline repeat opt-in error = %v, want ErrAssessmentState", err)
	}
	if err := service.Assess(AssessInput{
		EventID:    "event-baseline-assessment-3",
		ChangeID:   baseline.ChangeID,
		OccurredAt: operatorTestTime.Add(3 * time.Minute),
		Owner:      "owner",
		SourceRef:  "review:owner-assessment-3",
		Outcome:    domain.IntentMatch,
	}); err != nil {
		t.Fatalf("assess baseline: %v", err)
	}
	if err := store.Verify(); err != nil {
		t.Fatalf("verify baseline evidence: %v", err)
	}
}

func TestLaunchReceiptRepositoryEvidenceKeepsOnlyApprovedIdentity(t *testing.T) {
	repoRoot := t.TempDir()
	storePath := filepath.Join(repoRoot, "events.jsonl")
	store, err := evidence.NewStore(storePath)
	if err != nil {
		t.Fatalf("create evidence store: %v", err)
	}
	service, err := NewService(store, repoRoot)
	if err != nil {
		t.Fatalf("create operator service: %v", err)
	}
	receipt := startSynthetic(t, service, "change-app-privacy-1", "run-app-privacy-1", operatorTestTime)
	result, err := service.BindLaunchReceipt(LaunchReceiptInput{
		EventID:        "event-app-privacy-lineage-1",
		OccurredAt:     operatorTestTime.Add(time.Minute),
		Actor:          "goalrail-launcher",
		SourceRef:      "codex-app:create-thread",
		EncodedContext: receipt.RunContextEnv,
		RawReceipt: []byte(
			`{"threadId":"thread-app-privacy-1","title":"private source content",` +
				`"prompt":"do not retain launch prompt","transcript_path":"/private/app.jsonl",` +
				`"token":"raw-launch-credential"}`,
		),
	})
	if err != nil {
		t.Fatalf("bind launch receipt: %v", err)
	}
	if result.Lineage.RootSessionID != "thread-app-privacy-1" {
		t.Fatalf("approved provider identity missing: %#v", result.Lineage)
	}
	encodedEvidence, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read serialized launch evidence: %v", err)
	}
	if !strings.Contains(string(encodedEvidence), "thread-app-privacy-1") {
		t.Fatal("serialized evidence omitted approved provider identity")
	}
	for _, forbidden := range []string{
		"private source content",
		"do not retain launch prompt",
		"transcript_path",
		"/private/app.jsonl",
		"raw-launch-credential",
	} {
		if strings.Contains(string(encodedEvidence), forbidden) {
			t.Fatalf("serialized evidence retained forbidden launch payload %q", forbidden)
		}
	}
}

func TestWordingOnlyFixtureDoesNotBecomeMaterialPrevention(t *testing.T) {
	service, _, _ := newTestService(t)
	receipt := startSynthetic(t, service, "change-wording-only-1", "run-wording-only-1", operatorTestTime)
	freezeSyntheticChecks(t, service, receipt.ChangeID, operatorTestTime.Add(30*time.Second), "check:go-test")
	overhead := 3.0
	if err := service.Deliver(DeliverInput{
		EventID:             "event-wording-delivery-1",
		ChangeID:            receipt.ChangeID,
		OccurredAt:          operatorTestTime.Add(time.Minute),
		Actor:               "operator",
		SourceRef:           "review:wording-handoff-1",
		CheckRefs:           []domain.EvidenceReference{"check:go-test"},
		ChecksGreen:         true,
		FlowOverheadMinutes: &overhead,
	}); err != nil {
		t.Fatalf("deliver wording-only fixture: %v", err)
	}
	repeat := true
	if err := service.Assess(AssessInput{
		EventID:     "event-wording-assessment-1",
		ChangeID:    receipt.ChangeID,
		OccurredAt:  operatorTestTime.Add(2 * time.Minute),
		Owner:       "owner",
		SourceRef:   "review:wording-assessment-1",
		Outcome:     domain.IntentMatch,
		RepeatOptIn: &repeat,
	}); err != nil {
		t.Fatalf("assess wording-only fixture: %v", err)
	}
	view, err := service.Inspect(receipt.ChangeID)
	if err != nil {
		t.Fatalf("inspect wording-only fixture: %v", err)
	}
	if view.MaterialCorrections != 0 || view.Assessment == nil ||
		view.Assessment.MaterialCorrectionBeforeDelivery {
		t.Fatalf("wording-only fixture counted as material prevention: %#v", view)
	}
}

func TestOperatorExclusionsStayOutsideOrdinalAndReportDenominators(t *testing.T) {
	service, store, _ := newTestService(t)
	excluded, err := service.Exclude(ExcludeInput{
		EventID:    "event-excluded-manual-1",
		ChangeID:   "change-excluded-manual-1",
		OccurredAt: operatorTestTime,
		Actor:      "operator",
		SourceRef:  "request:manual-task",
		Reason:     "manual-task",
		Synthetic:  true,
	})
	if err != nil {
		t.Fatalf("exclude candidate: %v", err)
	}
	if excluded.Decision != domain.AdmissionExcluded || excluded.Reason != "manual-task" {
		t.Fatalf("unexpected exclusion receipt: %#v", excluded)
	}
	if _, err := service.Start(StartInput{
		EventID:       "event-conflicting-eligible-1",
		ChangeID:      excluded.ChangeID,
		RunID:         "run-conflicting-eligible-1",
		IntentVersion: 1,
		OccurredAt:    operatorTestTime.Add(time.Second),
		Actor:         "operator",
		SourceRef:     "request:conflicting-eligible",
		Reason:        "eligibility-confirmed",
		Synthetic:     true,
	}); !errors.Is(err, ErrChangeAlreadyStarted) {
		t.Fatalf("conflicting decision error = %v, want ErrChangeAlreadyStarted", err)
	}

	first := startSynthetic(t, service, "change-eligible-after-exclusion-1", "run-eligible-after-exclusion-1", operatorTestTime.Add(2*time.Second))
	second := startSynthetic(t, service, "change-eligible-after-exclusion-2", "run-eligible-after-exclusion-2", operatorTestTime.Add(3*time.Second))
	if first.Ordinal != 1 || second.Ordinal != 2 {
		t.Fatalf("exclusion consumed ordinal: first=%#v second=%#v", first, second)
	}
	view, err := service.Inspect(excluded.ChangeID)
	if err != nil {
		t.Fatalf("inspect exclusion: %v", err)
	}
	if view.Admission == nil || view.Admission.Decision != domain.AdmissionExcluded ||
		view.AdmissionReason != "manual-task" || view.Assignment != nil {
		t.Fatalf("exclusion projection is incomplete: %#v", view)
	}
	report, err := service.Report()
	if err != nil {
		t.Fatalf("report admissions: %v", err)
	}
	if report.Assigned != 2 || report.Excluded != 1 || report.ExclusionReasons["manual-task"] != 1 ||
		report.Flow.Assigned+report.Baseline.Assigned != 2 {
		t.Fatalf("admission report changed denominators: %#v", report)
	}
	if err := store.Verify(); err != nil {
		t.Fatalf("verify admission evidence: %v", err)
	}
}

func TestOperatorProjectsEffectiveChecksAndRejectsLateCorrection(t *testing.T) {
	service, store, _ := newTestService(t)
	receipt := startSynthetic(t, service, "change-check-correction-1", "run-check-correction-1", operatorTestTime)
	frozen := freezeSyntheticChecks(t, service, receipt.ChangeID, operatorTestTime.Add(time.Minute), "test:unit")
	corrected, err := service.CorrectChecks(CorrectChecksInput{
		EventID:    "event-check-correction-1",
		ChangeID:   receipt.ChangeID,
		OccurredAt: operatorTestTime.Add(2 * time.Minute),
		Actor:      "operator",
		SourceRef:  "review:check-correction-1",
		Reason:     "missing-ci",
		CheckRefs:  []domain.EvidenceReference{"test:unit", "ci:required"},
	})
	if err != nil {
		t.Fatalf("correct checks: %v", err)
	}
	if corrected.EvidenceEventID == frozen.EvidenceEventID || len(corrected.CheckRefs) != 2 {
		t.Fatalf("unexpected check correction receipt: %#v", corrected)
	}
	view, err := service.Inspect(receipt.ChangeID)
	if err != nil {
		t.Fatalf("inspect corrected checks: %v", err)
	}
	if view.CheckSet == nil || len(view.CheckSet.CheckRefs) != 2 || view.CheckSetEventID != corrected.EvidenceEventID {
		t.Fatalf("effective check set not projected: %#v", view)
	}
	if err := service.Deliver(DeliverInput{
		EventID:     "event-delivery-check-mismatch-1",
		ChangeID:    receipt.ChangeID,
		OccurredAt:  operatorTestTime.Add(3 * time.Minute),
		Actor:       "operator",
		SourceRef:   "review:check-mismatch-1",
		CheckRefs:   []domain.EvidenceReference{"test:unit"},
		ChecksGreen: true,
	}); !errors.Is(err, evidence.ErrInvalidEvent) {
		t.Fatalf("mismatched delivery error = %v, want evidence.ErrInvalidEvent", err)
	}
	if err := service.Deliver(DeliverInput{
		EventID:     "event-delivery-checks-1",
		ChangeID:    receipt.ChangeID,
		OccurredAt:  operatorTestTime.Add(3 * time.Minute),
		Actor:       "operator",
		SourceRef:   "review:checks-handoff-1",
		CheckRefs:   []domain.EvidenceReference{"ci:required", "test:unit"},
		ChecksGreen: true,
	}); err != nil {
		t.Fatalf("deliver effective checks: %v", err)
	}
	if _, err := service.CorrectChecks(CorrectChecksInput{
		EventID:    "event-check-correction-late-1",
		ChangeID:   receipt.ChangeID,
		OccurredAt: operatorTestTime.Add(4 * time.Minute),
		Actor:      "operator",
		SourceRef:  "review:check-correction-late-1",
		Reason:     "late-change",
		CheckRefs:  []domain.EvidenceReference{"test:unit", "ci:required", "check:manual"},
	}); !errors.Is(err, ErrChangeAlreadyTerminal) {
		t.Fatalf("late check correction error = %v, want ErrChangeAlreadyTerminal", err)
	}
	if err := store.Verify(); err != nil {
		t.Fatalf("verify corrected check evidence: %v", err)
	}
}

func TestOperatorRejectsRealStartAndPostTerminalMaterialCorrection(t *testing.T) {
	service, store, _ := newTestService(t)
	if _, err := service.Exclude(ExcludeInput{
		EventID:    "event-real-exclusion",
		ChangeID:   "change-real-exclusion",
		OccurredAt: operatorTestTime,
		Actor:      "operator",
		SourceRef:  "request:real-exclusion",
		Reason:     "manual-task",
	}); !errors.Is(err, ErrRealCanaryNotActivated) {
		t.Fatalf("real exclusion error = %v, want ErrRealCanaryNotActivated", err)
	}
	_, err := service.Start(StartInput{
		EventID:       "event-real-start",
		ChangeID:      "change-real-1",
		RunID:         "run-real-1",
		IntentVersion: 1,
		OccurredAt:    operatorTestTime,
		Actor:         "operator",
		SourceRef:     "request:real-change",
		Reason:        "eligibility-confirmed",
	})
	if !errors.Is(err, ErrRealCanaryNotActivated) {
		t.Fatalf("real start error = %v, want ErrRealCanaryNotActivated", err)
	}
	if events, readErr := store.ReadAll(); readErr != nil || len(events) != 0 {
		t.Fatalf("rejected real start wrote evidence: events=%#v err=%v", events, readErr)
	}

	receipt := startSynthetic(t, service, "change-abandoned-1", "run-abandoned-1", operatorTestTime)
	overhead := 2.0
	if err := service.Abandon(AbandonInput{
		EventID:             "event-abandoned-1",
		ChangeID:            receipt.ChangeID,
		OccurredAt:          operatorTestTime.Add(time.Minute),
		Actor:               "operator",
		SourceRef:           "review:abandonment-1",
		Reason:              "flow-too-costly",
		ProcessCaused:       true,
		FlowOverheadMinutes: &overhead,
	}); err != nil {
		t.Fatalf("abandon flow change: %v", err)
	}
	view, err := service.Inspect(receipt.ChangeID)
	if err != nil {
		t.Fatalf("inspect abandoned change: %v", err)
	}
	if view.Terminal == nil || view.Terminal.State != domain.CanaryStateAbandoned ||
		!view.Terminal.ProcessCausedAbandonment {
		t.Fatalf("process-caused abandonment fixture missing: %#v", view.Terminal)
	}
	if err := service.RecordMaterialCorrection(MaterialCorrectionInput{
		EventID:    "event-late-material-1",
		ChangeID:   receipt.ChangeID,
		OccurredAt: operatorTestTime.Add(2 * time.Minute),
		Owner:      "owner",
		SourceRef:  "review:late-correction",
		Reason:     "late-correction",
	}); !errors.Is(err, ErrChangeAlreadyTerminal) {
		t.Fatalf("post-terminal correction error = %v, want ErrChangeAlreadyTerminal", err)
	}
}

func TestDisableStopsNewAssignmentsAndPreservesExistingEvidence(t *testing.T) {
	service, store, repoRoot := newTestService(t)
	receipt := startSynthetic(t, service, "change-before-stop-1", "run-before-stop-1", operatorTestTime)
	evidencePath := filepath.Join(repoRoot, "events.jsonl")
	beforeStop, err := os.ReadFile(evidencePath)
	if err != nil {
		t.Fatalf("read evidence before stop: %v", err)
	}

	disableReceipt, err := service.Disable(DisableInput{
		EventID:    "event-canary-stop-1",
		OccurredAt: operatorTestTime.Add(time.Minute),
		Actor:      "owner",
		SourceRef:  "review:synthetic-rollback-v1",
		Reason:     "rollback-exercise",
	})
	if err != nil {
		t.Fatalf("disable assignments: %v", err)
	}
	if !disableReceipt.AssignmentsStopped || disableReceipt.ManifestVersion != 1 {
		t.Fatalf("unexpected disable receipt: %#v", disableReceipt)
	}
	afterStop, err := os.ReadFile(evidencePath)
	if err != nil {
		t.Fatalf("read evidence after stop: %v", err)
	}
	if !strings.HasPrefix(string(afterStop), string(beforeStop)) || len(afterStop) <= len(beforeStop) {
		t.Fatal("stop did not append after the byte-identical prior evidence")
	}

	_, err = service.Start(StartInput{
		EventID:       "event-start-after-stop",
		ChangeID:      "change-after-stop-2",
		RunID:         "run-after-stop-2",
		IntentVersion: 1,
		OccurredAt:    operatorTestTime.Add(2 * time.Minute),
		Actor:         "operator",
		SourceRef:     "request:synthetic-after-stop",
		Reason:        "eligibility-confirmed",
		Synthetic:     true,
	})
	if !errors.Is(err, ErrCanaryStopped) {
		t.Fatalf("start after stop error = %v, want ErrCanaryStopped", err)
	}
	afterRejectedStart, err := os.ReadFile(evidencePath)
	if err != nil {
		t.Fatalf("read evidence after rejected start: %v", err)
	}
	if string(afterRejectedStart) != string(afterStop) {
		t.Fatal("rejected assignment mutated the stopped evidence chain")
	}

	// Stopping assignments does not discard or freeze evidence for work that was
	// already assigned.
	freezeSyntheticChecks(t, service, receipt.ChangeID, operatorTestTime.Add(150*time.Second), "test:rollback-v1")
	overhead := 1.5
	if err := service.Deliver(DeliverInput{
		EventID:             "event-deliver-after-stop-1",
		ChangeID:            receipt.ChangeID,
		OccurredAt:          operatorTestTime.Add(3 * time.Minute),
		Actor:               "operator",
		SourceRef:           "review:handoff-after-stop-1",
		CheckRefs:           []domain.EvidenceReference{"test:rollback-v1"},
		ChecksGreen:         true,
		FlowOverheadMinutes: &overhead,
	}); err != nil {
		t.Fatalf("record terminal evidence after stop: %v", err)
	}
	view, err := service.Inspect(receipt.ChangeID)
	if err != nil {
		t.Fatalf("inspect pre-stop assignment: %v", err)
	}
	if view.Assignment == nil || view.Terminal == nil || view.Terminal.State != domain.CanaryStateDelivered {
		t.Fatalf("pre-stop evidence is not readable: %#v", view)
	}
	report, err := service.Report()
	if err != nil {
		t.Fatalf("report stopped canary: %v", err)
	}
	if report.Verdict != domain.CanaryVerdictStop || !report.AssignmentsStopped || report.Assigned != 1 {
		t.Fatalf("unexpected stopped report: %#v", report)
	}
	if _, err := service.Disable(DisableInput{
		EventID:    "event-canary-stop-2",
		OccurredAt: operatorTestTime.Add(4 * time.Minute),
		Actor:      "owner",
		SourceRef:  "review:synthetic-rollback-v1",
		Reason:     "duplicate-stop",
	}); !errors.Is(err, ErrCanaryAlreadyStopped) {
		t.Fatalf("second disable error = %v, want ErrCanaryAlreadyStopped", err)
	}
	if err := store.Verify(); err != nil {
		t.Fatalf("verify stopped evidence chain: %v", err)
	}
}

func TestSessionConflictProjectionStopsImmediately(t *testing.T) {
	conflictOutcome := lineageOutcome(ChangeView{
		Lineage: &domain.ExecutionLineage{
			Status:             domain.LineageUnlinked,
			UnlinkedReasonCode: codex.ReasonSessionConflict,
		},
		LineageResolutionAttempts: 1,
	})
	report, err := domain.CalculateCanaryReport(domain.CanaryReportInput{
		Observations: []domain.CanaryObservation{{
			Ordinal:        1,
			ChangeID:       "change-session-conflict",
			Variant:        domain.VariantFlow,
			TerminalState:  domain.CanaryStatePending,
			LineageOutcome: conflictOutcome,
		}},
	})
	if err != nil {
		t.Fatalf("calculate conflict report: %v", err)
	}
	if conflictOutcome != domain.CanaryLineageWrong || report.WrongJoins != 1 ||
		!report.HardStopSignals.WrongJoin || report.Verdict != domain.CanaryVerdictStop {
		t.Fatalf("session conflict did not stop: outcome=%q report=%#v", conflictOutcome, report)
	}

	unresolvedOutcome := lineageOutcome(ChangeView{
		Lineage: &domain.ExecutionLineage{
			Status:             domain.LineageUnlinked,
			UnlinkedReasonCode: codex.ReasonResolutionExhausted,
		},
		LineageResolutionAttempts: 1,
	})
	report, err = domain.CalculateCanaryReport(domain.CanaryReportInput{
		Observations: []domain.CanaryObservation{{
			Ordinal:        1,
			ChangeID:       "change-unresolved",
			Variant:        domain.VariantFlow,
			TerminalState:  domain.CanaryStatePending,
			LineageOutcome: unresolvedOutcome,
		}},
	})
	if err != nil {
		t.Fatalf("calculate unresolved report: %v", err)
	}
	if unresolvedOutcome != domain.CanaryLineageUnresolvedAfterResolution ||
		report.LineageUnresolved != 1 || report.WrongJoins != 0 ||
		report.HardStopSignals.UnresolvedLinks || report.Verdict == domain.CanaryVerdictStop {
		t.Fatalf("ordinary unresolved lineage changed threshold: outcome=%q report=%#v", unresolvedOutcome, report)
	}
}

func TestSessionConflictProjectionSurvivesResolutionExhaustion(t *testing.T) {
	service, _, repoRoot := newTestService(t)
	receipt := startSynthetic(t, service, "change-session-conflict-history", "run-session-conflict-history", operatorTestTime)
	bind := func(eventID domain.EvidenceEventID, sessionID string, occurredAt time.Time) codex.CorrelationResult {
		t.Helper()
		hook := []byte(fmt.Sprintf(
			`{"session_id":%q,"cwd":%q,"hook_event_name":"SessionStart","source":"startup"}`,
			sessionID,
			repoRoot,
		))
		result, err := service.BindLifecycleHook(HookInput{
			EventID:        eventID,
			OccurredAt:     occurredAt,
			Actor:          "goalrail-hook",
			SourceRef:      "codex-hook:lifecycle",
			EncodedContext: receipt.RunContextEnv,
			RawHook:        hook,
		})
		if err != nil {
			t.Fatalf("bind lifecycle hook %s: %v", eventID, err)
		}
		return result
	}

	verified := bind("event-lineage-verified", "session-original", operatorTestTime.Add(time.Minute))
	if !verified.Verified() {
		t.Fatalf("initial lineage was not verified: %#v", verified)
	}
	conflict := bind("event-lineage-conflict", "session-replaced", operatorTestTime.Add(2*time.Minute))
	if conflict.Lineage.UnlinkedReasonCode != codex.ReasonSessionConflict {
		t.Fatalf("conflicting lineage reason = %q", conflict.Lineage.UnlinkedReasonCode)
	}
	exhausted := bind("event-lineage-exhausted", "session-replaced", operatorTestTime.Add(3*time.Minute))
	if exhausted.Lineage.UnlinkedReasonCode != codex.ReasonResolutionExhausted {
		t.Fatalf("retry lineage reason = %q", exhausted.Lineage.UnlinkedReasonCode)
	}

	view, err := service.Inspect(receipt.ChangeID)
	if err != nil {
		t.Fatalf("inspect conflict history: %v", err)
	}
	if view.Lineage == nil || view.Lineage.UnlinkedReasonCode != codex.ReasonResolutionExhausted ||
		lineageOutcome(view) != domain.CanaryLineageWrong {
		t.Fatalf("wrong join was not retained across retry: %#v", view)
	}
	report, err := service.Report()
	if err != nil {
		t.Fatalf("report conflict history: %v", err)
	}
	if report.WrongJoins != 1 || !report.HardStopSignals.WrongJoin || report.Verdict != domain.CanaryVerdictStop {
		t.Fatalf("session conflict history did not stop: %#v", report)
	}
}

func TestManifestV2ProjectsContextBasisPhaseAndTelemetryAppendOnly(t *testing.T) {
	service, store, _ := newTestServiceV2(t)
	receipt := startSynthetic(t, service, "change-v2-flow", "run-v2-flow", operatorTestTime)
	if receipt.ManifestVersion != 2 || receipt.Variant != domain.VariantFlow {
		t.Fatalf("unexpected v2 assignment: %#v", receipt)
	}

	contextInput := ContextBindingInput{
		EventID:            "event-context-v2-flow",
		ChangeID:           receipt.ChangeID,
		OccurredAt:         operatorTestTime.Add(time.Minute),
		Actor:              "operator",
		SourceRef:          "openspec:intent-context-evaluation-v0",
		ContextPackID:      "context-v2-flow",
		ContextPackVersion: 1,
	}
	if err := service.RecordContextBinding(contextInput); err != nil {
		t.Fatalf("record v2 context: %v", err)
	}
	contextInput.EventID = "event-context-v2-duplicate"
	contextInput.OccurredAt = contextInput.OccurredAt.Add(time.Second)
	if err := service.RecordContextBinding(contextInput); err == nil || !strings.Contains(err.Error(), "already has a context binding") {
		t.Fatalf("duplicate context error = %v", err)
	}

	basis := domain.CanaryAssessmentBasis{
		IntentRef:         "openspec:intent-context-evaluation-v0",
		IntentID:          "intent-context-evaluation-v0",
		IntentVersion:     1,
		Timing:            domain.BasisPreExecution,
		DesiredOutcomeIDs: []domain.IntentItemID{"OUT-1"},
		NonGoalIDs:        []domain.IntentItemID{"NG-1"},
		SuccessSignalIDs:  []domain.IntentItemID{"SIG-1"},
	}
	wrongBasis := basis
	wrongBasis.IntentVersion = 2
	if err := service.RecordAssessmentBasis(AssessmentBasisInput{
		EventID:    "event-basis-v2-wrong-intent-version",
		ChangeID:   receipt.ChangeID,
		OccurredAt: operatorTestTime.Add(90 * time.Second),
		Actor:      "owner",
		SourceRef:  "owner-review:basis-v2-wrong-version",
		Basis:      wrongBasis,
	}); err == nil || !strings.Contains(err.Error(), "intent version must match") {
		t.Fatalf("mismatched assessment-basis intent version error = %v", err)
	}
	if err := service.RecordAssessmentBasis(AssessmentBasisInput{
		EventID:    "event-basis-v2-flow",
		ChangeID:   receipt.ChangeID,
		OccurredAt: operatorTestTime.Add(2 * time.Minute),
		Actor:      "owner",
		SourceRef:  "owner-review:basis-v2-flow",
		Basis:      basis,
	}); err != nil {
		t.Fatalf("record v2 basis: %v", err)
	}
	phaseStart := operatorTestTime.Add(3 * time.Minute)
	phaseEnd := operatorTestTime.Add(5 * time.Minute)
	if err := service.RecordFlowPhase(FlowPhaseInput{
		EventID:     "event-phase-v2-flow",
		ChangeID:    receipt.ChangeID,
		OccurredAt:  phaseEnd,
		Actor:       "operator",
		SourceRef:   "review:flow-phase-v2",
		StartedAt:   phaseStart,
		CompletedAt: phaseEnd,
	}); err != nil {
		t.Fatalf("record v2 flow phase: %v", err)
	}

	lineage := domain.ExecutionLineage{
		Status:         domain.LineageVerified,
		ChangeID:       receipt.ChangeID,
		RunID:          receipt.RunID,
		RootSessionID:  "session-v2-flow",
		IdentitySource: domain.SessionIdentityLifecycleHook,
		ContextDigest:  strings.Repeat("a", 64),
	}
	if err := store.Append(domain.EvidenceEvent{
		ID:                        "event-lineage-v2-flow",
		CanaryID:                  domain.IntentCanaryV0ManifestID,
		ManifestVersion:           2,
		ChangeID:                  receipt.ChangeID,
		Kind:                      domain.EventLineageRecorded,
		OccurredAt:                operatorTestTime.Add(6 * time.Minute),
		Actor:                     "goalrail-hook",
		SourceRef:                 "codex-hook:lifecycle",
		ObservationRefs:           []domain.EvidenceReference{"langfuse-session:session-v2-flow"},
		Lineage:                   &lineage,
		LineageResolutionAttempts: 1,
	}); err != nil {
		t.Fatalf("append v2 lineage: %v", err)
	}

	traceA := domain.CanaryTimingInterval{
		Reference: "langfuse-trace:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		StartedAt: phaseStart.Add(10 * time.Second),
		EndedAt:   phaseStart.Add(40 * time.Second),
	}
	ownerReview := domain.CanaryTimingInterval{
		Reference: "owner-review:change-v2-flow",
		StartedAt: phaseEnd,
		EndedAt:   phaseEnd.Add(30 * time.Second),
	}
	firstOverhead, err := domain.CalculateCanaryFlowOverhead(domain.CanaryFlowOverheadInput{
		AgentTurns: []domain.CanaryTimingInterval{traceA}, OwnerReview: &ownerReview, OwnerReviewRequired: true,
	})
	if err != nil {
		t.Fatalf("calculate first v2 overhead: %v", err)
	}
	if err := service.RecordTelemetryEvidence(TelemetryEvidenceInput{
		EventID:    "event-telemetry-v2-flow",
		ChangeID:   receipt.ChangeID,
		OccurredAt: operatorTestTime.Add(7 * time.Minute),
		Actor:      "operator",
		SourceRef:  "review:telemetry-v2-flow",
		Telemetry: domain.CanaryTelemetry{
			Status:         domain.TelemetryAvailable,
			SessionLookup:  "langfuse-session:session-v2-flow",
			TraceIntervals: []domain.CanaryTimingInterval{traceA},
			OwnerReview:    &ownerReview,
			FlowOverhead:   &firstOverhead,
		},
	}); err != nil {
		t.Fatalf("record v2 telemetry: %v", err)
	}
	traceB := domain.CanaryTimingInterval{
		Reference: "langfuse-trace:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		StartedAt: phaseStart.Add(50 * time.Second),
		EndedAt:   phaseStart.Add(80 * time.Second),
	}
	correctedOverhead, err := domain.CalculateCanaryFlowOverhead(domain.CanaryFlowOverheadInput{
		AgentTurns: []domain.CanaryTimingInterval{traceA, traceB}, OwnerReview: &ownerReview, OwnerReviewRequired: true,
	})
	if err != nil {
		t.Fatalf("calculate corrected v2 overhead: %v", err)
	}
	if err := service.RecordTelemetryEvidence(TelemetryEvidenceInput{
		EventID:          "event-telemetry-v2-correction",
		ChangeID:         receipt.ChangeID,
		OccurredAt:       operatorTestTime.Add(8 * time.Minute),
		Actor:            "operator",
		SourceRef:        "review:telemetry-v2-correction",
		CorrectionReason: "delayed-provider-data",
		Telemetry: domain.CanaryTelemetry{
			Status:         domain.TelemetryAvailable,
			SessionLookup:  "langfuse-session:session-v2-flow",
			TraceIntervals: []domain.CanaryTimingInterval{traceA, traceB},
			OwnerReview:    &ownerReview,
			FlowOverhead:   &correctedOverhead,
		},
	}); err != nil {
		t.Fatalf("correct v2 telemetry: %v", err)
	}

	view, err := service.Inspect(receipt.ChangeID)
	if err != nil {
		t.Fatalf("inspect v2 flow: %v", err)
	}
	if view.ManifestVersion != 2 || view.Context == nil || view.AssessmentBasis == nil ||
		view.FlowPhase == nil || view.Telemetry == nil || len(view.Telemetry.TraceIntervals) != 2 ||
		view.EventCount != 7 {
		t.Fatalf("unexpected v2 projection: %#v", view)
	}
}

func newTestService(t *testing.T) (*Service, *evidence.Store, string) {
	t.Helper()
	repoRoot := t.TempDir()
	store, err := evidence.NewStore(repoRoot + "/events.jsonl")
	if err != nil {
		t.Fatalf("create evidence store: %v", err)
	}
	service, err := NewService(store, repoRoot)
	if err != nil {
		t.Fatalf("create operator service: %v", err)
	}
	return service, store, repoRoot
}

func newTestServiceV2(t *testing.T) (*Service, *evidence.Store, string) {
	t.Helper()
	repoRoot := t.TempDir()
	store, err := evidence.NewStoreForManifest(filepath.Join(repoRoot, "events-v2.jsonl"), 2)
	if err != nil {
		t.Fatalf("create v2 evidence store: %v", err)
	}
	service, err := NewServiceForManifest(store, repoRoot, 2)
	if err != nil {
		t.Fatalf("create v2 operator service: %v", err)
	}
	return service, store, repoRoot
}

func startSynthetic(
	t *testing.T,
	service *Service,
	changeID domain.ChangeID,
	runID domain.RunID,
	occurredAt time.Time,
) StartReceipt {
	t.Helper()
	receipt, err := service.Start(StartInput{
		EventID:       domain.EvidenceEventID("event-start-" + string(changeID)),
		ChangeID:      changeID,
		RunID:         runID,
		IntentVersion: 1,
		OccurredAt:    occurredAt,
		Actor:         "operator",
		SourceRef:     "request:synthetic-change",
		Reason:        "eligibility-confirmed",
		Synthetic:     true,
	})
	if err != nil {
		t.Fatalf("start synthetic %s: %v", changeID, err)
	}
	return receipt
}

func freezeSyntheticChecks(
	t *testing.T,
	service *Service,
	changeID domain.ChangeID,
	occurredAt time.Time,
	checkRefs ...domain.EvidenceReference,
) FreezeChecksReceipt {
	t.Helper()
	receipt, err := service.FreezeChecks(FreezeChecksInput{
		EventID:    domain.EvidenceEventID("event-checks-" + string(changeID)),
		ChangeID:   changeID,
		OccurredAt: occurredAt,
		Actor:      "operator",
		SourceRef:  "request:synthetic-checks",
		CheckRefs:  checkRefs,
	})
	if err != nil {
		t.Fatalf("freeze synthetic checks %s: %v", changeID, err)
	}
	return receipt
}
