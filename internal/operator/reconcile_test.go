package operator

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/heurema/goalrail/internal/domain"
)

var reconcileTestTime = time.Date(2026, time.July, 22, 12, 0, 0, 0, time.UTC)

func TestReconcileTraceObservationsGroupsDeduplicatesAndAppliesFlowWindow(t *testing.T) {
	phase := &domain.CanaryFlowPhase{StartedAt: reconcileTestTime, CompletedAt: reconcileTestTime.Add(5 * time.Minute)}
	traceA := domain.EvidenceReference("langfuse-trace:" + strings.Repeat("a", 32))
	traceB := domain.EvidenceReference("langfuse-trace:" + strings.Repeat("b", 32))
	result, err := ReconcileTraceObservations(reconcileLineage(), domain.VariantFlow, phase, []domain.TraceObservation{
		{TraceReference: traceB, SessionID: "session-1", StartedAt: reconcileTestTime.Add(2 * time.Minute), EndedAt: reconcileTestTime.Add(3 * time.Minute)},
		{TraceReference: traceA, SessionID: "session-1", StartedAt: reconcileTestTime.Add(time.Minute), EndedAt: reconcileTestTime.Add(90 * time.Second)},
		{TraceReference: traceA, SessionID: "session-1", StartedAt: reconcileTestTime.Add(70 * time.Second), EndedAt: reconcileTestTime.Add(2 * time.Minute)},
		{TraceReference: domain.EvidenceReference("langfuse-trace:" + strings.Repeat("c", 32)), SessionID: "session-1", StartedAt: reconcileTestTime.Add(6 * time.Minute), EndedAt: reconcileTestTime.Add(7 * time.Minute)},
	})
	if err != nil {
		t.Fatalf("reconcile flow observations: %v", err)
	}
	if result.Reason != "" || result.Telemetry.Status != domain.TelemetryAvailable || len(result.Telemetry.TraceIntervals) != 2 {
		t.Fatalf("unexpected reconciliation: %#v", result)
	}
	if result.Telemetry.TraceIntervals[0].Reference != traceA ||
		!result.Telemetry.TraceIntervals[0].StartedAt.Equal(reconcileTestTime.Add(time.Minute)) ||
		!result.Telemetry.TraceIntervals[0].EndedAt.Equal(reconcileTestTime.Add(2*time.Minute)) {
		t.Fatalf("trace envelope was not grouped deterministically: %#v", result.Telemetry.TraceIntervals)
	}
}

func TestReconcileTraceObservationsKeepsMissingConflictAndMixedExplicit(t *testing.T) {
	phase := &domain.CanaryFlowPhase{StartedAt: reconcileTestTime, CompletedAt: reconcileTestTime.Add(5 * time.Minute)}
	trace := domain.EvidenceReference("langfuse-trace:" + strings.Repeat("a", 32))
	tests := []struct {
		name         string
		phase        *domain.CanaryFlowPhase
		observations []domain.TraceObservation
		status       domain.TelemetryStatus
		reason       domain.EvidenceReasonCode
	}{
		{name: "missing", phase: phase, status: domain.TelemetryUnavailable, reason: ReasonTelemetryMissing},
		{name: "phase missing", status: domain.TelemetryUnavailable, reason: ReasonFlowPhaseMissing},
		{
			name: "session conflict", phase: phase, status: domain.TelemetryConflict, reason: ReasonTelemetrySessionConflict,
			observations: []domain.TraceObservation{{TraceReference: trace, SessionID: "other-session", StartedAt: reconcileTestTime, EndedAt: reconcileTestTime.Add(time.Minute)}},
		},
		{
			name: "mixed trace", phase: phase, status: domain.TelemetryUnavailable, reason: ReasonFlowTraceMixed,
			observations: []domain.TraceObservation{{TraceReference: trace, SessionID: "session-1", StartedAt: reconcileTestTime.Add(4 * time.Minute), EndedAt: reconcileTestTime.Add(6 * time.Minute)}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := ReconcileTraceObservations(reconcileLineage(), domain.VariantFlow, test.phase, test.observations)
			if err != nil {
				t.Fatalf("reconcile: %v", err)
			}
			if result.Telemetry.Status != test.status || result.Reason != test.reason || len(result.Telemetry.TraceIntervals) != 0 {
				t.Fatalf("reconciliation = %#v", result)
			}
		})
	}
}

func TestServiceReconcilesFlowAndDerivesTerminalOverhead(t *testing.T) {
	service, store, _ := newTestServiceV2(t)
	receipt := startSynthetic(t, service, "change-reconcile-flow", "run-reconcile-flow", reconcileTestTime)
	if err := service.RecordContextBinding(ContextBindingInput{
		EventID: "event-context-reconcile-flow", ChangeID: receipt.ChangeID,
		OccurredAt: reconcileTestTime.Add(time.Minute), Actor: "operator",
		SourceRef: "openspec:synthetic-flow", ContextPackID: "context-reconcile-flow", ContextPackVersion: 1,
	}); err != nil {
		t.Fatalf("record context: %v", err)
	}
	if err := service.RecordAssessmentBasis(AssessmentBasisInput{
		EventID: "event-basis-reconcile-flow", ChangeID: receipt.ChangeID,
		OccurredAt: reconcileTestTime.Add(2 * time.Minute), Actor: "owner", SourceRef: "owner-review:flow-basis",
		Basis: domain.CanaryAssessmentBasis{
			IntentRef: "openspec:synthetic-flow", IntentID: "intent-reconcile-flow", IntentVersion: 1,
			Timing: domain.BasisPreExecution, DesiredOutcomeIDs: []domain.IntentItemID{"OUT-1"},
			NonGoalIDs: []domain.IntentItemID{"NG-1"}, SuccessSignalIDs: []domain.IntentItemID{"SIG-1"},
		},
	}); err != nil {
		t.Fatalf("record basis: %v", err)
	}
	phase := domain.CanaryFlowPhase{StartedAt: reconcileTestTime.Add(3 * time.Minute), CompletedAt: reconcileTestTime.Add(5 * time.Minute)}
	if err := service.RecordFlowPhase(FlowPhaseInput{
		EventID: "event-phase-reconcile-flow", ChangeID: receipt.ChangeID, OccurredAt: phase.CompletedAt,
		Actor: "operator", SourceRef: "review:flow-phase", StartedAt: phase.StartedAt, CompletedAt: phase.CompletedAt,
	}); err != nil {
		t.Fatalf("record phase: %v", err)
	}
	lineage := domain.ExecutionLineage{
		Status: domain.LineageVerified, ChangeID: receipt.ChangeID, RunID: receipt.RunID,
		RootSessionID: "session-reconcile-flow", IdentitySource: domain.SessionIdentityLifecycleHook,
		ContextDigest: strings.Repeat("a", 64),
	}
	if err := store.Append(domain.EvidenceEvent{
		ID: "event-lineage-reconcile-flow", CanaryID: domain.IntentCanaryV0ManifestID, ManifestVersion: 2,
		ChangeID: receipt.ChangeID, Kind: domain.EventLineageRecorded, OccurredAt: reconcileTestTime.Add(6 * time.Minute),
		Actor: "goalrail-hook", SourceRef: "codex-hook:lifecycle",
		ObservationRefs: []domain.EvidenceReference{"langfuse-session:session-reconcile-flow"},
		Lineage:         &lineage, LineageResolutionAttempts: 1,
	}); err != nil {
		t.Fatalf("append lineage: %v", err)
	}
	source := &fakeTraceSource{observations: []domain.TraceObservation{{
		TraceReference: "langfuse-trace:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		SessionID:      "session-reconcile-flow", StartedAt: phase.StartedAt.Add(15 * time.Second),
		EndedAt: phase.StartedAt.Add(75 * time.Second),
	}}}
	ownerReview := &domain.CanaryTimingInterval{
		Reference: "review:owner-flow", StartedAt: phase.CompletedAt, EndedAt: phase.CompletedAt.Add(30 * time.Second),
	}
	telemetryReceipt, err := service.ReconcileTelemetry(context.Background(), source, ReconcileTelemetryInput{
		EventID: "event-reconcile-flow", ChangeID: receipt.ChangeID,
		OccurredAt: reconcileTestTime.Add(7 * time.Minute), Actor: "operator",
		SourceRef: "langfuse-api:observations-v2", OwnerReview: ownerReview,
	})
	if err != nil {
		t.Fatalf("reconcile telemetry: %v", err)
	}
	if telemetryReceipt.Telemetry.FlowOverhead == nil || !telemetryReceipt.Telemetry.FlowOverhead.Available ||
		telemetryReceipt.Telemetry.FlowOverhead.AgentSeconds != 60 ||
		telemetryReceipt.Telemetry.FlowOverhead.OwnerReviewSeconds != 30 ||
		telemetryReceipt.Telemetry.FlowOverhead.TotalMinutes != 1.5 {
		t.Fatalf("unexpected derived overhead: %#v", telemetryReceipt.Telemetry.FlowOverhead)
	}
	if !source.query.FromStartTime.Equal(phase.StartedAt) ||
		!source.query.ToStartTime.Equal(phase.CompletedAt.Add(time.Nanosecond)) ||
		source.query.SessionID != lineage.RootSessionID {
		t.Fatalf("unexpected trace query: %#v", source.query)
	}
	if _, err := service.FreezeChecks(FreezeChecksInput{
		EventID: "event-checks-reconcile-flow", ChangeID: receipt.ChangeID,
		OccurredAt: reconcileTestTime.Add(8 * time.Minute), Actor: "operator",
		SourceRef: "review:flow-checks", CheckRefs: []domain.EvidenceReference{"test:go-test"},
	}); err != nil {
		t.Fatalf("freeze checks: %v", err)
	}
	callerTotal := 99.0
	if err := service.Deliver(DeliverInput{
		EventID: "event-deliver-forged-overhead", ChangeID: receipt.ChangeID,
		OccurredAt: reconcileTestTime.Add(9 * time.Minute), Actor: "operator", SourceRef: "review:delivery",
		CheckRefs: []domain.EvidenceReference{"test:go-test"}, ChecksGreen: true, FlowOverheadMinutes: &callerTotal,
	}); err == nil || !strings.Contains(err.Error(), "derives flow overhead") {
		t.Fatalf("caller-authored v2 total error = %v", err)
	}
	if err := service.Deliver(DeliverInput{
		EventID: "event-deliver-reconcile-flow", ChangeID: receipt.ChangeID,
		OccurredAt: reconcileTestTime.Add(9 * time.Minute), Actor: "operator", SourceRef: "review:delivery",
		CheckRefs: []domain.EvidenceReference{"test:go-test"}, ChecksGreen: true,
	}); err != nil {
		t.Fatalf("deliver with derived overhead: %v", err)
	}
	view, err := service.Inspect(receipt.ChangeID)
	if err != nil || view.Terminal == nil || view.Terminal.FlowOverheadMinutes == nil || *view.Terminal.FlowOverheadMinutes != 1.5 {
		t.Fatalf("terminal did not project derived overhead: view=%#v err=%v", view, err)
	}
	if _, err := service.ReconcileTelemetry(context.Background(), source, ReconcileTelemetryInput{
		EventID: "event-reconcile-correction-after-delivery", ChangeID: receipt.ChangeID,
		OccurredAt: reconcileTestTime.Add(9*time.Minute + 30*time.Second), Actor: "operator",
		SourceRef: "langfuse-api:observations-v2", OwnerReview: ownerReview,
		CorrectionReason: "delayed-provider-data",
	}); err != nil {
		t.Fatalf("correct telemetry after delivery: %v", err)
	}
	if err := service.Assess(AssessInput{
		EventID: "event-assess-incomplete-flow", ChangeID: receipt.ChangeID,
		OccurredAt: reconcileTestTime.Add(10 * time.Minute), Owner: "owner", SourceRef: "owner-review:flow-assessment",
		Outcome: domain.IntentMatch, RepeatOptIn: boolPointer(true),
		ItemJudgments: []domain.IntentItemJudgment{{
			ItemID: "OUT-1", Category: domain.IntentCategoryDesiredOutcome, Judgment: domain.JudgmentAchieved,
		}},
	}); err == nil {
		t.Fatal("incomplete v2 assessment was accepted")
	}
	matchingJudgments := []domain.IntentItemJudgment{
		{ItemID: "OUT-1", Category: domain.IntentCategoryDesiredOutcome, Judgment: domain.JudgmentAchieved},
		{ItemID: "NG-1", Category: domain.IntentCategoryNonGoal, Judgment: domain.JudgmentPreserved},
		{ItemID: "SIG-1", Category: domain.IntentCategorySuccessSignal, Judgment: domain.JudgmentObserved},
	}
	if err := service.Assess(AssessInput{
		EventID: "event-assess-reconcile-flow", ChangeID: receipt.ChangeID,
		OccurredAt: reconcileTestTime.Add(10 * time.Minute), Owner: "owner", SourceRef: "owner-review:flow-assessment",
		Outcome: domain.IntentMatch, RepeatOptIn: boolPointer(true), ItemJudgments: matchingJudgments,
	}); err != nil {
		t.Fatalf("record structured assessment: %v", err)
	}
	partialJudgments := append([]domain.IntentItemJudgment(nil), matchingJudgments...)
	partialJudgments[2].Judgment = domain.JudgmentMissing
	if err := service.CorrectAssessment(CorrectAssessmentInput{
		EventID: "event-assess-correction-inconsistent", ChangeID: receipt.ChangeID,
		OccurredAt: reconcileTestTime.Add(11 * time.Minute), Owner: "owner", SourceRef: "owner-review:flow-correction",
		Reason: "owner-reassessment", Outcome: domain.IntentMatch, RepeatOptIn: boolPointer(false),
		ItemJudgments: partialJudgments,
	}); err == nil {
		t.Fatal("inconsistent assessment correction was accepted")
	}
	if err := service.CorrectAssessment(CorrectAssessmentInput{
		EventID: "event-assess-correction-flow", ChangeID: receipt.ChangeID,
		OccurredAt: reconcileTestTime.Add(11 * time.Minute), Owner: "owner", SourceRef: "owner-review:flow-correction",
		Reason: "owner-reassessment", Outcome: domain.IntentPartial, RepeatOptIn: boolPointer(false),
		ItemJudgments: partialJudgments,
	}); err != nil {
		t.Fatalf("correct structured assessment: %v", err)
	}
	view, err = service.Inspect(receipt.ChangeID)
	if err != nil || view.Assessment == nil || view.Assessment.Outcome != domain.IntentPartial ||
		len(view.Assessment.ItemJudgments) != 3 || view.Assessment.ItemJudgments[2].Judgment != domain.JudgmentMissing {
		t.Fatalf("latest structured assessment was not projected: view=%#v err=%v", view, err)
	}
	events, err := store.ReadAll()
	if err != nil {
		t.Fatalf("read append-only assessment history: %v", err)
	}
	assessmentPayloads := 0
	for _, event := range events {
		if event.Assessment != nil {
			assessmentPayloads++
		}
	}
	if assessmentPayloads != 2 {
		t.Fatalf("assessment correction did not retain history: %d payloads", assessmentPayloads)
	}
}

func TestServiceReconcilesBaselineWithoutFlowOverhead(t *testing.T) {
	service, store, _ := newTestServiceV2(t)
	for ordinal, changeID := range []domain.ChangeID{"change-baseline-spacer-1", "change-baseline-spacer-2"} {
		receipt := startSynthetic(t, service, changeID, domain.RunID("run-"+string(changeID)), reconcileTestTime.Add(time.Duration(ordinal)*time.Minute))
		if err := service.Abandon(AbandonInput{
			EventID: domain.EvidenceEventID("event-abandon-" + string(changeID)), ChangeID: receipt.ChangeID,
			OccurredAt: reconcileTestTime.Add(time.Duration(ordinal+1) * time.Minute), Actor: "operator",
			SourceRef: "review:rotation-spacer", Reason: "synthetic-rotation-spacer",
		}); err != nil {
			t.Fatalf("abandon rotation spacer: %v", err)
		}
	}
	receipt := startSynthetic(t, service, "change-reconcile-baseline", "run-reconcile-baseline", reconcileTestTime.Add(3*time.Minute))
	if receipt.Variant != domain.VariantBaseline {
		t.Fatalf("third assignment is not baseline: %#v", receipt)
	}
	lineage := domain.ExecutionLineage{
		Status: domain.LineageVerified, ChangeID: receipt.ChangeID, RunID: receipt.RunID,
		RootSessionID: "session-reconcile-baseline", IdentitySource: domain.SessionIdentityLifecycleHook,
		ContextDigest: strings.Repeat("b", 64),
	}
	if err := store.Append(domain.EvidenceEvent{
		ID: "event-lineage-reconcile-baseline", CanaryID: domain.IntentCanaryV0ManifestID, ManifestVersion: 2,
		ChangeID: receipt.ChangeID, Kind: domain.EventLineageRecorded, OccurredAt: reconcileTestTime.Add(4 * time.Minute),
		Actor: "goalrail-hook", SourceRef: "codex-hook:lifecycle",
		ObservationRefs: []domain.EvidenceReference{"langfuse-session:session-reconcile-baseline"},
		Lineage:         &lineage, LineageResolutionAttempts: 1,
	}); err != nil {
		t.Fatalf("append baseline lineage: %v", err)
	}
	source := &fakeTraceSource{observations: []domain.TraceObservation{{
		TraceReference: "langfuse-trace:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		SessionID:      lineage.RootSessionID, StartedAt: reconcileTestTime.Add(3*time.Minute + 10*time.Second),
		EndedAt: reconcileTestTime.Add(3*time.Minute + 40*time.Second),
	}}}
	reconciledAt := reconcileTestTime.Add(5 * time.Minute)
	reconciled, err := service.ReconcileTelemetry(context.Background(), source, ReconcileTelemetryInput{
		EventID: "event-reconcile-baseline", ChangeID: receipt.ChangeID, OccurredAt: reconciledAt,
		Actor: "operator", SourceRef: "langfuse-api:observations-v2",
	})
	if err != nil {
		t.Fatalf("reconcile baseline: %v", err)
	}
	if reconciled.Telemetry.Status != domain.TelemetryAvailable || reconciled.Telemetry.OwnerReview != nil || reconciled.Telemetry.FlowOverhead != nil {
		t.Fatalf("baseline recorded flow overhead: %#v", reconciled.Telemetry)
	}
	if !source.query.FromStartTime.Equal(reconcileTestTime.Add(3*time.Minute)) || !source.query.ToStartTime.Equal(reconciledAt) {
		t.Fatalf("unexpected baseline query: %#v", source.query)
	}
}

type fakeTraceSource struct {
	observations []domain.TraceObservation
	err          error
	query        domain.TraceObservationQuery
}

func boolPointer(value bool) *bool { return &value }

func (source *fakeTraceSource) ListSessionObservations(
	_ context.Context,
	query domain.TraceObservationQuery,
) ([]domain.TraceObservation, error) {
	source.query = query
	return append([]domain.TraceObservation(nil), source.observations...), source.err
}

func reconcileLineage() domain.ExecutionLineage {
	return domain.ExecutionLineage{
		Status:         domain.LineageVerified,
		ChangeID:       "change-1",
		RunID:          "run-1",
		RootSessionID:  "session-1",
		IdentitySource: domain.SessionIdentityLifecycleHook,
		ContextDigest:  strings.Repeat("a", 64),
	}
}
