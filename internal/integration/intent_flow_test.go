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
	"github.com/heurema/goalrail/internal/adapters/openspec"
	"github.com/heurema/goalrail/internal/domain"
	"github.com/heurema/goalrail/internal/evidence"
)

var integrationTime = time.Date(2026, time.July, 21, 12, 0, 0, 0, time.UTC)

func TestConfirmedIntentFlowPersistsCorrelatedEvidenceAndIndependentAssessment(t *testing.T) {
	repoRoot := t.TempDir()
	changeDir := filepath.Join(repoRoot, "openspec", "changes", "change-1")
	writeArtifact(t, changeDir, "intent.md", intentArtifact("candidate", false))

	if _, err := openspec.LoadChange(changeDir); !errors.Is(err, openspec.ErrIntentNotConfirmed) {
		t.Fatalf("load candidate error = %v, want ErrIntentNotConfirmed", err)
	} else if strings.Contains(err.Error(), "proposal") {
		t.Fatalf("candidate gate read a downstream proposal: %v", err)
	}

	writeArtifact(t, changeDir, "intent.md", intentArtifact("confirmed", true))
	writeArtifact(t, changeDir, "proposal.md", proposalArtifact())
	compiled, err := openspec.LoadChange(changeDir)
	if err != nil {
		t.Fatalf("load confirmed change: %v", err)
	}
	if compiled.Intent.GrantsEffectAuthority() {
		t.Fatal("confirmed intent granted effect authority")
	}
	if len(compiled.Proposal.Changes) != 1 {
		t.Fatalf("compiled proposal changes = %d, want 1", len(compiled.Proposal.Changes))
	}

	runContext, err := codex.NewRunContext("change-1", "run-1", repoRoot)
	if err != nil {
		t.Fatalf("create run context: %v", err)
	}
	correlation, err := codex.BindLaunchReceipt(
		runContext,
		[]byte(`{"threadId":"session-1"}`),
		nil,
	)
	if err != nil {
		t.Fatalf("bind launch receipt: %v", err)
	}
	if correlation.Lineage.Status != domain.LineageVerified ||
		correlation.Lineage.ChangeID != "change-1" ||
		correlation.Lineage.RunID != "run-1" ||
		correlation.Lineage.RootSessionID != "session-1" {
		t.Fatalf("unexpected verified lineage: %#v", correlation.Lineage)
	}

	observationRefs, err := langfuse.BuildObservationReferences(
		correlation.Lineage,
		[]langfuse.ObservationIdentity{
			{
				TraceID:   strings.Repeat("a", 32),
				SessionID: correlation.Lineage.RootSessionID,
			},
		},
	)
	if err != nil {
		t.Fatalf("build observation references: %v", err)
	}
	if !observationRefs.HasTraceEvidence() {
		t.Fatal("exact Langfuse trace identity was not preserved")
	}

	evidencePath := filepath.Join(changeDir, "evidence", "events.jsonl")
	store, err := evidence.NewStore(evidencePath)
	if err != nil {
		t.Fatalf("create evidence store: %v", err)
	}
	lineage := correlation.Lineage
	lineageEvent := domain.EvidenceEvent{
		ID:              "event-lineage-1",
		CanaryID:        "canary-v0",
		ChangeID:        "change-1",
		Kind:            domain.EventLineageRecorded,
		OccurredAt:      integrationTime,
		Actor:           "goalrail-adapter",
		SourceRef:       "launch-receipt:create-thread",
		ObservationRefs: observationRefs.EvidenceReferences(),
		Lineage:         &lineage,
	}
	if err := store.Append(lineageEvent); err != nil {
		t.Fatalf("append lineage evidence: %v", err)
	}
	lineageBytes := readEvidence(t, evidencePath)

	assessmentEvent := domain.EvidenceEvent{
		ID:         "event-assessment-1",
		CanaryID:   "canary-v0",
		ChangeID:   "change-1",
		Kind:       domain.EventAssessmentRecorded,
		OccurredAt: integrationTime.Add(time.Minute),
		Actor:      "owner",
		SourceRef:  "owner-review:terminal",
		Assessment: &domain.Assessment{
			Outcome:     domain.IntentMiss,
			AssessedBy:  "owner",
			AssessedAt:  integrationTime.Add(time.Minute),
			ChecksGreen: true,
		},
	}
	if err := store.Append(assessmentEvent); err != nil {
		t.Fatalf("append assessment evidence: %v", err)
	}
	if completeBytes := readEvidence(t, evidencePath); !bytes.HasPrefix(completeBytes, lineageBytes) {
		t.Fatal("assessment append rewrote prior lineage evidence")
	}

	reopened, err := evidence.NewStore(evidencePath)
	if err != nil {
		t.Fatalf("reopen evidence store: %v", err)
	}
	events, err := reopened.ReadAll()
	if err != nil {
		t.Fatalf("read verified evidence chain: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("evidence event count = %d, want 2", len(events))
	}
	if len(events[0].ObservationRefs) != 2 ||
		events[0].ObservationRefs[0] != "langfuse-session:session-1" ||
		events[0].ObservationRefs[1] != domain.EvidenceReference("langfuse-trace:"+strings.Repeat("a", 32)) {
		t.Fatalf("unexpected observation references: %#v", events[0].ObservationRefs)
	}
	assessment := events[1].Assessment
	if assessment == nil || !assessment.ChecksGreen || assessment.Outcome != domain.IntentMiss {
		t.Fatalf("green checks replaced independent intent outcome: %#v", assessment)
	}
	if err := reopened.Verify(); err != nil {
		t.Fatalf("verify reopened evidence chain: %v", err)
	}
}

func TestCorrelatedFlowRemainsUsableWithoutLangfuseTraceEvidence(t *testing.T) {
	repoRoot := t.TempDir()
	runContext, err := codex.NewRunContext("change-2", "run-2", repoRoot)
	if err != nil {
		t.Fatalf("create run context: %v", err)
	}
	correlation, err := codex.BindLaunchReceipt(
		runContext,
		[]byte(`{"threadId":"session-2"}`),
		nil,
	)
	if err != nil {
		t.Fatalf("bind launch receipt: %v", err)
	}
	references, err := langfuse.BuildObservationReferences(correlation.Lineage, nil)
	if err != nil {
		t.Fatalf("build references without traces: %v", err)
	}
	if references.HasTraceEvidence() {
		t.Fatal("missing Langfuse trace was reported as evidence")
	}

	store, err := evidence.NewStore(filepath.Join(repoRoot, "evidence", "events.jsonl"))
	if err != nil {
		t.Fatalf("create evidence store: %v", err)
	}
	lineage := correlation.Lineage
	event := domain.EvidenceEvent{
		ID:              "event-lineage-2",
		CanaryID:        "canary-v0",
		ChangeID:        "change-2",
		Kind:            domain.EventLineageRecorded,
		OccurredAt:      integrationTime,
		Actor:           "goalrail-adapter",
		SourceRef:       "launch-receipt:create-thread",
		ObservationRefs: references.EvidenceReferences(),
		Lineage:         &lineage,
	}
	if err := store.Append(event); err != nil {
		t.Fatalf("append evidence without Langfuse trace: %v", err)
	}
	events, err := store.ReadAll()
	if err != nil {
		t.Fatalf("read evidence without Langfuse trace: %v", err)
	}
	if len(events) != 1 || len(events[0].ObservationRefs) != 1 ||
		events[0].ObservationRefs[0] != "langfuse-session:session-2" {
		t.Fatalf("unexpected missing-trace evidence: %#v", events)
	}
}

func writeArtifact(t *testing.T, changeDir, name, contents string) {
	t.Helper()
	if err := os.MkdirAll(changeDir, 0o750); err != nil {
		t.Fatalf("create change directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(changeDir, name), []byte(contents), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func readEvidence(t *testing.T, path string) []byte {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read evidence: %v", err)
	}
	return contents
}

func intentArtifact(status string, confirmed bool) string {
	confirmation := `- **Confirmed by:** pending
- **Confirmed at:** pending
- **Verification action:** pending`
	if confirmed {
		confirmation = `- **Confirmed by:** owner
- **Confirmed at:** 2026-07-21
- **Verification action:** owner-reviewed-three-groups`
	}
	return `# Intent Snapshot

- **Intent ID:** INTENT-TEST
- **Version:** 1
- **Status:** ` + status + `
- **Owner:** owner

## Source Evidence

- **SE-1 — Owner statement:** The owner asked for a bounded local result.

## Desired Outcomes

| ID | Confirmed wording | Verification action | Evidence |
|---|---|---|---|
| OUT-1 | Produce the bounded local result. | Inspect it. | SE-1 |

## Non-Goals

| ID | Confirmed boundary | Evidence |
|---|---|---|
| NG-1 | Do not publish. | SE-1 |

## Observable Success Signals

| ID | Signal | Measurement | Evidence |
|---|---|---|---|
| SIG-1 | The result is inspectable. | One local evidence chain exists. | SE-1 |

## Ambiguities and Unknowns

None.

## Confirmation

` + confirmation + `
`
}

func proposalArtifact() string {
	return `## Why

Exercise the bounded intent-first flow.

## Intent Coverage

| Proposed change | Intent IDs | Non-goal preserved |
|---|---|---|
| Produce bounded local result | OUT-1, SIG-1 | NG-1 |
`
}
