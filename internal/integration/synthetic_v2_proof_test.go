package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

var syntheticV2Time = time.Date(2026, time.July, 22, 16, 0, 0, 0, time.UTC)

func TestSyntheticV2ProofMatchesRetainedArtifacts(t *testing.T) {
	events, report, validation := buildSyntheticV2Proof(t)
	if os.Getenv("GOALRAIL_PRINT_SYNTHETIC_V2_PROOF") == "1" {
		fmt.Printf("---EVENTS---\n%s---REPORT---\n%s---VALIDATION---\n%s", events, report, validation)
		return
	}
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve integration test path")
	}
	proofRoot := filepath.Join(filepath.Dir(sourceFile), "..", "..", "canary", "intent-canary-v0", "synthetic-v2")
	for name, actual := range map[string][]byte{
		"events.jsonl":    events,
		"report.json":     report,
		"validation.json": validation,
	} {
		expected, err := os.ReadFile(filepath.Join(proofRoot, name))
		if err != nil {
			t.Fatalf("read retained %s: %v", name, err)
		}
		if !bytes.Equal(actual, expected) {
			t.Fatalf("retained %s is stale; regenerate from the deterministic proof", name)
		}
	}
}

func buildSyntheticV2Proof(t *testing.T) ([]byte, []byte, []byte) {
	t.Helper()
	repoRoot := t.TempDir()
	storePath := filepath.Join(repoRoot, "events-v2.jsonl")
	store, err := evidence.NewStoreForManifest(storePath, domain.IntentCanaryV0ManifestVersion2)
	if err != nil {
		t.Fatalf("create proof store: %v", err)
	}
	service, err := operator.NewServiceForManifest(store, repoRoot, domain.IntentCanaryV0ManifestVersion2)
	if err != nil {
		t.Fatalf("create proof service: %v", err)
	}

	spacer := startProofChange(t, service, "change-synthetic-v2-spacer", "run-synthetic-v2-spacer", syntheticV2Time)
	if spacer.Ordinal != 1 || spacer.Variant != domain.VariantFlow {
		t.Fatalf("unexpected spacer assignment: %#v", spacer)
	}
	if err := service.RecordContextBinding(operator.ContextBindingInput{
		EventID: "event-synthetic-v2-spacer-context", ChangeID: spacer.ChangeID,
		OccurredAt: syntheticV2Time.Add(10 * time.Second), Actor: "operator",
		SourceRef: "openspec:synthetic-v2-spacer", ContextPackID: "context-synthetic-v2-spacer", ContextPackVersion: 1,
	}); err != nil {
		t.Fatalf("record spacer context: %v", err)
	}
	spacerBasis := proofBasis(domain.BasisPreExecution, "intent-synthetic-v2-spacer", "openspec:synthetic-v2-spacer")
	if err := service.RecordAssessmentBasis(operator.AssessmentBasisInput{
		EventID: "event-synthetic-v2-spacer-basis", ChangeID: spacer.ChangeID,
		OccurredAt: syntheticV2Time.Add(20 * time.Second), Actor: "owner",
		SourceRef: "owner-review:synthetic-v2-spacer-basis", Basis: spacerBasis,
	}); err != nil {
		t.Fatalf("record spacer basis: %v", err)
	}
	spacerPhase := domain.CanaryFlowPhase{
		StartedAt: syntheticV2Time.Add(30 * time.Second), CompletedAt: syntheticV2Time.Add(40 * time.Second),
	}
	if err := service.RecordFlowPhase(operator.FlowPhaseInput{
		EventID: "event-synthetic-v2-spacer-phase", ChangeID: spacer.ChangeID,
		OccurredAt: spacerPhase.CompletedAt, Actor: "operator", SourceRef: "review:synthetic-v2-spacer-phase",
		StartedAt: spacerPhase.StartedAt, CompletedAt: spacerPhase.CompletedAt,
	}); err != nil {
		t.Fatalf("record spacer phase: %v", err)
	}
	appendProofLineage(t, store, spacer, "session-synthetic-v2-spacer", "event-synthetic-v2-spacer-lineage", syntheticV2Time.Add(45*time.Second), "c")
	spacerSource := &proofTraceSource{observations: []domain.TraceObservation{{
		TraceReference: "langfuse-trace:cccccccccccccccccccccccccccccccc",
		SessionID:      "session-synthetic-v2-spacer", StartedAt: spacerPhase.StartedAt.Add(time.Second),
		EndedAt: spacerPhase.StartedAt.Add(5 * time.Second),
	}}}
	spacerOwnerReview := &domain.CanaryTimingInterval{
		Reference: "review:synthetic-v2-spacer-owner-review", StartedAt: spacerPhase.CompletedAt,
		EndedAt: spacerPhase.CompletedAt.Add(2 * time.Second),
	}
	if _, err := service.ReconcileTelemetry(context.Background(), spacerSource, operator.ReconcileTelemetryInput{
		EventID: "event-synthetic-v2-spacer-telemetry", ChangeID: spacer.ChangeID,
		OccurredAt: syntheticV2Time.Add(50 * time.Second), Actor: "operator",
		SourceRef: "langfuse-api:synthetic-v2-spacer", OwnerReview: spacerOwnerReview,
	}); err != nil {
		t.Fatalf("reconcile spacer telemetry: %v", err)
	}
	if err := service.Abandon(operator.AbandonInput{
		EventID: "event-synthetic-v2-spacer-abandon", ChangeID: spacer.ChangeID,
		OccurredAt: syntheticV2Time.Add(time.Minute), Actor: "operator",
		SourceRef: "review:synthetic-v2-spacer", Reason: "rotation-spacer",
	}); err != nil {
		t.Fatalf("abandon proof spacer: %v", err)
	}

	flow := startProofChange(t, service, "change-synthetic-v2-flow", "run-synthetic-v2-flow", syntheticV2Time.Add(2*time.Minute))
	if flow.Ordinal != 2 || flow.Variant != domain.VariantFlow {
		t.Fatalf("unexpected flow assignment: %#v", flow)
	}
	if err := service.RecordContextBinding(operator.ContextBindingInput{
		EventID: "event-synthetic-v2-flow-context", ChangeID: flow.ChangeID,
		OccurredAt: syntheticV2Time.Add(3 * time.Minute), Actor: "operator",
		SourceRef: "openspec:synthetic-v2-flow", ContextPackID: "context-synthetic-v2-flow", ContextPackVersion: 1,
	}); err != nil {
		t.Fatalf("record flow context: %v", err)
	}
	flowBasis := proofBasis(domain.BasisPreExecution, "intent-synthetic-v2-flow", "openspec:synthetic-v2-flow")
	if err := service.RecordAssessmentBasis(operator.AssessmentBasisInput{
		EventID: "event-synthetic-v2-flow-basis", ChangeID: flow.ChangeID,
		OccurredAt: syntheticV2Time.Add(4 * time.Minute), Actor: "owner",
		SourceRef: "owner-review:synthetic-v2-flow-basis", Basis: flowBasis,
	}); err != nil {
		t.Fatalf("record flow basis: %v", err)
	}
	phase := domain.CanaryFlowPhase{
		StartedAt:   syntheticV2Time.Add(5 * time.Minute),
		CompletedAt: syntheticV2Time.Add(6 * time.Minute),
	}
	if err := service.RecordFlowPhase(operator.FlowPhaseInput{
		EventID: "event-synthetic-v2-flow-phase", ChangeID: flow.ChangeID,
		OccurredAt: phase.CompletedAt, Actor: "operator", SourceRef: "review:synthetic-v2-flow-phase",
		StartedAt: phase.StartedAt, CompletedAt: phase.CompletedAt,
	}); err != nil {
		t.Fatalf("record flow phase: %v", err)
	}
	appendProofLineage(t, store, flow, "session-synthetic-v2-flow", "event-synthetic-v2-flow-lineage", syntheticV2Time.Add(7*time.Minute), "a")
	flowSource := &proofTraceSource{observations: []domain.TraceObservation{{
		TraceReference: "langfuse-trace:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		SessionID:      "session-synthetic-v2-flow", StartedAt: phase.StartedAt.Add(10 * time.Second),
		EndedAt: phase.StartedAt.Add(50 * time.Second),
	}}}
	ownerReview := &domain.CanaryTimingInterval{
		Reference: "review:synthetic-v2-owner-review", StartedAt: phase.CompletedAt,
		EndedAt: phase.CompletedAt.Add(20 * time.Second),
	}
	if _, err := service.ReconcileTelemetry(context.Background(), flowSource, operator.ReconcileTelemetryInput{
		EventID: "event-synthetic-v2-flow-telemetry", ChangeID: flow.ChangeID,
		OccurredAt: syntheticV2Time.Add(8 * time.Minute), Actor: "operator",
		SourceRef: "langfuse-api:synthetic-v2-flow", OwnerReview: ownerReview,
	}); err != nil {
		t.Fatalf("reconcile flow telemetry: %v", err)
	}
	completeProofDelivery(t, service, flow.ChangeID, "flow", syntheticV2Time.Add(9*time.Minute), syntheticV2Time.Add(10*time.Minute))
	if err := service.Assess(operator.AssessInput{
		EventID: "event-synthetic-v2-flow-assessment", ChangeID: flow.ChangeID,
		OccurredAt: syntheticV2Time.Add(11 * time.Minute), Owner: "owner",
		SourceRef: "owner-review:synthetic-v2-flow-assessment", Outcome: domain.IntentMatch,
		RepeatOptIn: proofBool(true), ItemJudgments: proofJudgments(),
	}); err != nil {
		t.Fatalf("assess proof flow: %v", err)
	}

	baseline := startProofChange(t, service, "change-synthetic-v2-baseline", "run-synthetic-v2-baseline", syntheticV2Time.Add(12*time.Minute))
	if baseline.Ordinal != 3 || baseline.Variant != domain.VariantBaseline {
		t.Fatalf("unexpected baseline assignment: %#v", baseline)
	}
	appendProofLineage(t, store, baseline, "session-synthetic-v2-baseline", "event-synthetic-v2-baseline-lineage", syntheticV2Time.Add(13*time.Minute), "b")
	baselineSource := &proofTraceSource{observations: []domain.TraceObservation{{
		TraceReference: "langfuse-trace:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		SessionID:      "session-synthetic-v2-baseline", StartedAt: syntheticV2Time.Add(12*time.Minute + 30*time.Second),
		EndedAt: syntheticV2Time.Add(13 * time.Minute),
	}}}
	if _, err := service.ReconcileTelemetry(context.Background(), baselineSource, operator.ReconcileTelemetryInput{
		EventID: "event-synthetic-v2-baseline-telemetry", ChangeID: baseline.ChangeID,
		OccurredAt: syntheticV2Time.Add(14 * time.Minute), Actor: "operator",
		SourceRef: "langfuse-api:synthetic-v2-baseline",
	}); err != nil {
		t.Fatalf("reconcile baseline telemetry: %v", err)
	}
	completeProofDelivery(t, service, baseline.ChangeID, "baseline", syntheticV2Time.Add(15*time.Minute), syntheticV2Time.Add(16*time.Minute))
	baselineBasis := proofBasis(domain.BasisPostDelivery, "intent-synthetic-v2-baseline", "request:synthetic-v2-baseline")
	if err := service.RecordAssessmentBasis(operator.AssessmentBasisInput{
		EventID: "event-synthetic-v2-baseline-basis", ChangeID: baseline.ChangeID,
		OccurredAt: syntheticV2Time.Add(17 * time.Minute), Actor: "owner",
		SourceRef: "owner-review:synthetic-v2-baseline-basis", Basis: baselineBasis,
	}); err != nil {
		t.Fatalf("record baseline basis: %v", err)
	}
	if err := service.Assess(operator.AssessInput{
		EventID: "event-synthetic-v2-baseline-assessment", ChangeID: baseline.ChangeID,
		OccurredAt: syntheticV2Time.Add(18 * time.Minute), Owner: "owner",
		SourceRef: "owner-review:synthetic-v2-baseline-assessment", Outcome: domain.IntentMatch,
		ItemJudgments: proofJudgments(),
	}); err != nil {
		t.Fatalf("assess proof baseline: %v", err)
	}

	beforeRealAttempt, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read proof before real admission attempt: %v", err)
	}
	_, err = service.Start(operator.StartInput{
		EventID: "event-synthetic-v2-real-attempt", ChangeID: "change-real-v2-attempt", RunID: "run-real-v2-attempt",
		IntentVersion: 1, OccurredAt: syntheticV2Time.Add(19 * time.Minute), Actor: "operator",
		SourceRef: "request:real-v2-attempt", Reason: "eligibility-confirmed",
	})
	if !errors.Is(err, operator.ErrRealCanaryNotActivated) {
		t.Fatalf("real admission error = %v", err)
	}
	afterRealAttempt, err := os.ReadFile(storePath)
	if err != nil || !bytes.Equal(beforeRealAttempt, afterRealAttempt) {
		t.Fatalf("rejected real admission mutated proof: err=%v", err)
	}
	beforeDisable := append([]byte(nil), afterRealAttempt...)
	if _, err := service.Disable(operator.DisableInput{
		EventID: "event-synthetic-v2-disable", OccurredAt: syntheticV2Time.Add(20 * time.Minute),
		Actor: "owner", SourceRef: "review:synthetic-v2-rollback", Reason: "rollback-exercise",
	}); err != nil {
		t.Fatalf("disable proof manifest: %v", err)
	}
	if err := store.Verify(); err != nil {
		t.Fatalf("verify proof chain: %v", err)
	}
	events, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read retained proof chain: %v", err)
	}
	if !bytes.HasPrefix(events, beforeDisable) || len(events) <= len(beforeDisable) {
		t.Fatal("rollback did not preserve the prior chain as an exact prefix")
	}
	finalReport, err := service.Report()
	if err != nil {
		t.Fatalf("derive proof report: %v", err)
	}
	if finalReport.Assigned != 3 || finalReport.Flow.Delivered != 1 || finalReport.Baseline.Delivered != 1 ||
		!finalReport.AssignmentsStopped || finalReport.Verdict != domain.CanaryVerdictStop {
		t.Fatalf("unexpected proof report: %#v", finalReport)
	}
	report := mustIndentedJSON(t, finalReport)
	validation := mustIndentedJSON(t, syntheticProofValidation{
		SchemaVersion: 1, ManifestVersion: 2, EvidenceRecords: countJSONLLines(events),
		CompletedFlowChange: flow.ChangeID, CompletedBaselineChange: baseline.ChangeID,
		RotationSpacerChange: spacer.ChangeID, EvidenceVerified: true, ReportDerived: true,
		RealActivationPerformed: false, RejectedRealAdmissionProved: true,
		ExternalLangfuseRead: false, ExternalLangfuseWrite: false, SyntheticTraceSourceOnly: true,
		RollbackPrefixPreserved: true, StopEventAppended: true,
	})
	return events, report, validation
}

type syntheticProofValidation struct {
	SchemaVersion               uint32          `json:"schema_version"`
	ManifestVersion             uint32          `json:"manifest_version"`
	EvidenceRecords             int             `json:"evidence_records"`
	CompletedFlowChange         domain.ChangeID `json:"completed_flow_change"`
	CompletedBaselineChange     domain.ChangeID `json:"completed_baseline_change"`
	RotationSpacerChange        domain.ChangeID `json:"rotation_spacer_change"`
	EvidenceVerified            bool            `json:"evidence_verified"`
	ReportDerived               bool            `json:"report_derived"`
	RealActivationPerformed     bool            `json:"real_activation_performed"`
	RejectedRealAdmissionProved bool            `json:"rejected_real_admission_proved"`
	ExternalLangfuseRead        bool            `json:"external_langfuse_read"`
	ExternalLangfuseWrite       bool            `json:"external_langfuse_write"`
	SyntheticTraceSourceOnly    bool            `json:"synthetic_trace_source_only"`
	RollbackPrefixPreserved     bool            `json:"rollback_prefix_preserved"`
	StopEventAppended           bool            `json:"stop_event_appended"`
}

func startProofChange(
	t *testing.T,
	service *operator.Service,
	changeID domain.ChangeID,
	runID domain.RunID,
	occurredAt time.Time,
) operator.StartReceipt {
	t.Helper()
	receipt, err := service.Start(operator.StartInput{
		EventID:  domain.EvidenceEventID("event-" + string(changeID) + "-start"),
		ChangeID: changeID, RunID: runID, IntentVersion: 1, OccurredAt: occurredAt,
		Actor: "operator", SourceRef: domain.EvidenceReference("request:" + string(changeID)),
		Reason: "eligibility-confirmed", Synthetic: true,
	})
	if err != nil {
		t.Fatalf("start %s: %v", changeID, err)
	}
	return receipt
}

func appendProofLineage(
	t *testing.T,
	store *evidence.Store,
	receipt operator.StartReceipt,
	sessionID domain.SessionID,
	eventID domain.EvidenceEventID,
	occurredAt time.Time,
	digestCharacter string,
) {
	t.Helper()
	lineage := domain.ExecutionLineage{
		Status: domain.LineageVerified, ChangeID: receipt.ChangeID, RunID: receipt.RunID,
		RootSessionID: sessionID, IdentitySource: domain.SessionIdentityLifecycleHook,
		ContextDigest: strings.Repeat(digestCharacter, 64),
	}
	if err := store.Append(domain.EvidenceEvent{
		ID: eventID, CanaryID: domain.IntentCanaryV0ManifestID, ManifestVersion: 2,
		ChangeID: receipt.ChangeID, Kind: domain.EventLineageRecorded, OccurredAt: occurredAt,
		Actor: "goalrail-hook", SourceRef: "codex-hook:lifecycle",
		ObservationRefs: []domain.EvidenceReference{domain.EvidenceReference("langfuse-session:" + string(sessionID))},
		Lineage:         &lineage, LineageResolutionAttempts: 1,
	}); err != nil {
		t.Fatalf("append %s lineage: %v", receipt.ChangeID, err)
	}
}

func proofBasis(timing domain.AssessmentBasisTiming, intentID domain.IntentID, intentRef domain.EvidenceReference) domain.CanaryAssessmentBasis {
	return domain.CanaryAssessmentBasis{
		IntentRef: intentRef, IntentID: intentID, IntentVersion: 1, Timing: timing,
		DesiredOutcomeIDs: []domain.IntentItemID{"OUT-1"}, NonGoalIDs: []domain.IntentItemID{"NG-1"},
		SuccessSignalIDs: []domain.IntentItemID{"SIG-1"},
	}
}

func completeProofDelivery(
	t *testing.T,
	service *operator.Service,
	changeID domain.ChangeID,
	label string,
	checksAt time.Time,
	deliveredAt time.Time,
) {
	t.Helper()
	checkRef := domain.EvidenceReference("test:synthetic-v2-" + label)
	if _, err := service.FreezeChecks(operator.FreezeChecksInput{
		EventID: domain.EvidenceEventID("event-synthetic-v2-" + label + "-checks"), ChangeID: changeID,
		OccurredAt: checksAt, Actor: "operator", SourceRef: domain.EvidenceReference("review:synthetic-v2-" + label + "-checks"),
		CheckRefs: []domain.EvidenceReference{checkRef},
	}); err != nil {
		t.Fatalf("freeze %s checks: %v", label, err)
	}
	if err := service.Deliver(operator.DeliverInput{
		EventID: domain.EvidenceEventID("event-synthetic-v2-" + label + "-delivery"), ChangeID: changeID,
		OccurredAt: deliveredAt, Actor: "operator", SourceRef: domain.EvidenceReference("review:synthetic-v2-" + label + "-delivery"),
		CheckRefs: []domain.EvidenceReference{checkRef}, ChecksGreen: true,
	}); err != nil {
		t.Fatalf("deliver %s: %v", label, err)
	}
}

func proofJudgments() []domain.IntentItemJudgment {
	return []domain.IntentItemJudgment{
		{ItemID: "OUT-1", Category: domain.IntentCategoryDesiredOutcome, Judgment: domain.JudgmentAchieved},
		{ItemID: "NG-1", Category: domain.IntentCategoryNonGoal, Judgment: domain.JudgmentPreserved},
		{ItemID: "SIG-1", Category: domain.IntentCategorySuccessSignal, Judgment: domain.JudgmentObserved},
	}
}

func proofBool(value bool) *bool { return &value }

type proofTraceSource struct {
	observations []domain.TraceObservation
}

func (source *proofTraceSource) ListSessionObservations(
	_ context.Context,
	_ domain.TraceObservationQuery,
) ([]domain.TraceObservation, error) {
	return append([]domain.TraceObservation(nil), source.observations...), nil
}

func mustIndentedJSON(t *testing.T, value any) []byte {
	t.Helper()
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("encode proof JSON: %v", err)
	}
	return append(encoded, '\n')
}

func countJSONLLines(value []byte) int {
	return bytes.Count(value, []byte{'\n'})
}
