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
	if view.EventCount != 6 || view.MaterialCorrections != 1 {
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
	if events[4].Kind != domain.EventAssessmentRecorded ||
		events[5].SupersedesEventID != events[4].ID ||
		events[4].Assessment.Outcome != domain.IntentPartial {
		t.Fatalf("assessment history was not preserved: %#v", events[4:])
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

func TestOperatorRejectsRealStartAndPostTerminalMaterialCorrection(t *testing.T) {
	service, store, _ := newTestService(t)
	_, err := service.Start(StartInput{
		EventID:       "event-real-start",
		ChangeID:      "change-real-1",
		RunID:         "run-real-1",
		IntentVersion: 1,
		OccurredAt:    operatorTestTime,
		Actor:         "operator",
		SourceRef:     "request:real-change",
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
		Synthetic:     true,
	})
	if err != nil {
		t.Fatalf("start synthetic %s: %v", changeID, err)
	}
	return receipt
}
