package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/heurema/goalrail/internal/domain"
	"github.com/heurema/goalrail/internal/evidence"
	"github.com/heurema/goalrail/internal/operator"
)

var syntheticE2ETime = time.Date(2026, time.July, 21, 14, 10, 0, 0, time.UTC)

type syntheticValidationEvidence struct {
	SchemaVersion      uint32                         `json:"schema_version"`
	FixtureID          string                         `json:"fixture_id"`
	Synthetic          bool                           `json:"synthetic"`
	ManifestID         domain.CanaryID                `json:"manifest_id"`
	ManifestVersion    uint32                         `json:"manifest_version"`
	ChangeID           domain.ChangeID                `json:"change_id"`
	RunID              domain.RunID                   `json:"run_id"`
	AdmissionDecision  domain.AdmissionDecision       `json:"admission_decision"`
	FrozenCheckRefs    []domain.EvidenceReference     `json:"frozen_check_refs"`
	CheckRefs          []domain.EvidenceReference     `json:"check_refs"`
	ChecksGreen        bool                           `json:"checks_green"`
	FlowOverheadInput  domain.CanaryFlowOverheadInput `json:"flow_overhead_input"`
	FlowOverhead       domain.CanaryFlowOverhead      `json:"flow_overhead"`
	EventChainVerified bool                           `json:"event_chain_verified"`
	LineageStatus      domain.LineageStatus           `json:"lineage_status"`
	RootSessionID      domain.SessionID               `json:"root_session_id"`
	AssessmentOutcome  domain.IntentOutcome           `json:"assessment_outcome"`
	ReportVerdict      domain.CanaryVerdict           `json:"report_verdict"`
	RealCanaryStarted  bool                           `json:"real_canary_started"`
}

func TestSyntheticEndToEndEvidenceMatchesPreservedArtifacts(t *testing.T) {
	const (
		fixtureRepoRoot = "/workspace/goalrail"
		fixtureChangeID = domain.ChangeID("change-synthetic-e2e-v1")
		fixtureRunID    = domain.RunID("run-synthetic-e2e-v1")
		fixtureSession  = domain.SessionID("session-synthetic-e2e-v1")
	)
	storePath := filepath.Join(t.TempDir(), "events.jsonl")
	store, err := evidence.NewStore(storePath)
	if err != nil {
		t.Fatalf("create synthetic evidence store: %v", err)
	}
	service, err := operator.NewService(store, fixtureRepoRoot)
	if err != nil {
		t.Fatalf("create synthetic operator service: %v", err)
	}
	receipt, err := service.Start(operator.StartInput{
		EventID:       "event-synthetic-start-v1",
		ChangeID:      fixtureChangeID,
		RunID:         fixtureRunID,
		IntentVersion: 1,
		OccurredAt:    syntheticE2ETime,
		Actor:         "synthetic-operator",
		SourceRef:     "request:synthetic-e2e-v1",
		Reason:        "eligibility-confirmed",
		Synthetic:     true,
	})
	if err != nil {
		t.Fatalf("start synthetic change: %v", err)
	}
	hook := []byte(fmt.Sprintf(
		`{"session_id":%q,"cwd":%q,"hook_event_name":"SessionStart","source":"startup"}`,
		fixtureSession,
		fixtureRepoRoot,
	))
	correlation, err := service.BindLifecycleHook(operator.HookInput{
		EventID:        "event-synthetic-lineage-v1",
		OccurredAt:     syntheticE2ETime.Add(time.Minute),
		Actor:          "goalrail-hook",
		SourceRef:      "codex-hook:synthetic-e2e-v1",
		EncodedContext: receipt.RunContextEnv,
		RawHook:        hook,
	})
	if err != nil {
		t.Fatalf("bind synthetic lifecycle hook: %v", err)
	}
	overheadInput := domain.CanaryFlowOverheadInput{
		AgentTurns: []domain.CanaryTimingInterval{
			{
				Reference: "langfuse-trace:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				StartedAt: syntheticE2ETime.Add(10 * time.Second),
				EndedAt:   syntheticE2ETime.Add(20 * time.Second),
			},
		},
		OwnerReview: &domain.CanaryTimingInterval{
			Reference: "owner-review:synthetic-e2e-v1",
			StartedAt: syntheticE2ETime.Add(20 * time.Second),
			EndedAt:   syntheticE2ETime.Add(25 * time.Second),
		},
		OwnerReviewRequired: true,
	}
	overheadMeasurement, err := domain.CalculateCanaryFlowOverhead(overheadInput)
	if err != nil {
		t.Fatalf("calculate synthetic overhead: %v", err)
	}
	if !overheadMeasurement.Available || overheadMeasurement.TotalMinutes != 0.25 {
		t.Fatalf("unexpected synthetic overhead: %#v", overheadMeasurement)
	}
	overhead := overheadMeasurement.TotalMinutes
	checkRefs := []domain.EvidenceReference{"test:synthetic-e2e-v1"}
	if _, err := service.FreezeChecks(operator.FreezeChecksInput{
		EventID:    "event-synthetic-checks-v1",
		ChangeID:   fixtureChangeID,
		OccurredAt: syntheticE2ETime.Add(90 * time.Second),
		Actor:      "synthetic-operator",
		SourceRef:  "request:synthetic-checks-v1",
		CheckRefs:  checkRefs,
	}); err != nil {
		t.Fatalf("freeze synthetic checks: %v", err)
	}
	if err := service.Deliver(operator.DeliverInput{
		EventID:             "event-synthetic-delivery-v1",
		ChangeID:            fixtureChangeID,
		OccurredAt:          syntheticE2ETime.Add(2 * time.Minute),
		Actor:               "synthetic-operator",
		SourceRef:           "review:synthetic-handoff-v1",
		CheckRefs:           checkRefs,
		ChecksGreen:         true,
		FlowOverheadMinutes: &overhead,
	}); err != nil {
		t.Fatalf("deliver synthetic change: %v", err)
	}
	repeat := true
	if err := service.Assess(operator.AssessInput{
		EventID:     "event-synthetic-assessment-v1",
		ChangeID:    fixtureChangeID,
		OccurredAt:  syntheticE2ETime.Add(3 * time.Minute),
		Owner:       "synthetic-owner",
		SourceRef:   "owner-review:synthetic-assessment-v1",
		Outcome:     domain.IntentMatch,
		RepeatOptIn: &repeat,
	}); err != nil {
		t.Fatalf("assess synthetic change: %v", err)
	}
	if err := store.Verify(); err != nil {
		t.Fatalf("verify synthetic evidence chain: %v", err)
	}
	view, err := service.Inspect(fixtureChangeID)
	if err != nil {
		t.Fatalf("inspect synthetic change: %v", err)
	}
	report, err := service.Report()
	if err != nil {
		t.Fatalf("report synthetic change: %v", err)
	}
	if report.Verdict != domain.CanaryVerdictPending || report.Assigned != 1 ||
		report.LineageVerified != 1 || view.Assessment == nil || view.Admission == nil ||
		view.Admission.Decision != domain.AdmissionEligible || view.CheckSet == nil {
		t.Fatalf("unexpected synthetic projection: view=%#v report=%#v", view, report)
	}

	eventBytes, err := os.ReadFile(storePath)
	if err != nil {
		t.Fatalf("read synthetic evidence: %v", err)
	}
	reportBytes := marshalSyntheticArtifact(t, report)
	validationBytes := marshalSyntheticArtifact(t, syntheticValidationEvidence{
		SchemaVersion:      1,
		FixtureID:          "synthetic-e2e-v1",
		Synthetic:          view.Assignment.Synthetic,
		ManifestID:         receipt.CanaryID,
		ManifestVersion:    view.Assignment.ManifestVersion,
		ChangeID:           receipt.ChangeID,
		RunID:              receipt.RunID,
		AdmissionDecision:  view.Admission.Decision,
		FrozenCheckRefs:    append([]domain.EvidenceReference(nil), view.CheckSet.CheckRefs...),
		CheckRefs:          append([]domain.EvidenceReference(nil), view.Terminal.CheckRefs...),
		ChecksGreen:        view.Terminal.ChecksGreen,
		FlowOverheadInput:  overheadInput,
		FlowOverhead:       overheadMeasurement,
		EventChainVerified: true,
		LineageStatus:      correlation.Lineage.Status,
		RootSessionID:      correlation.Lineage.RootSessionID,
		AssessmentOutcome:  view.Assessment.Outcome,
		ReportVerdict:      report.Verdict,
		RealCanaryStarted:  false,
	})

	fixtureRoot := filepath.Join("testdata", "synthetic-e2e-v1")
	compareSyntheticArtifact(t, filepath.Join(fixtureRoot, "events.jsonl"), eventBytes)
	compareSyntheticArtifact(t, filepath.Join(fixtureRoot, "report.json"), reportBytes)
	compareSyntheticArtifact(t, filepath.Join(fixtureRoot, "validation.json"), validationBytes)
}

func marshalSyntheticArtifact(t *testing.T, value any) []byte {
	t.Helper()
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("encode synthetic artifact: %v", err)
	}
	return append(encoded, '\n')
}

func compareSyntheticArtifact(t *testing.T, path string, actual []byte) {
	t.Helper()
	expected, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("read preserved artifact %s: %v\n--- actual ---\n%s--- end actual ---", path, err, actual)
		return
	}
	if !bytes.Equal(expected, actual) {
		t.Errorf("preserved artifact %s does not match synthetic run\n--- actual ---\n%s--- end actual ---", path, actual)
	}
}
