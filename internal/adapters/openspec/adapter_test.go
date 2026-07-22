package openspec

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/heurema/goalrail/internal/domain"
)

func TestReadContextParsesBoundedPack(t *testing.T) {
	pack, err := ReadContext(strings.NewReader(minimalContext("sufficient", "None.")))
	if err != nil {
		t.Fatalf("read context: %v", err)
	}
	if pack.ID != "CONTEXT-TEST" || len(pack.Items) != 1 || pack.Items[0].SourceRef != "repo:README.md" {
		t.Fatalf("unexpected context pack: %#v", pack)
	}
}

func TestReadContextRejectsSensitiveClaim(t *testing.T) {
	artifact := strings.Replace(
		minimalContext("sufficient", "None."),
		"The repository defines a bounded flow.",
		"authorization: Bearer example",
		1,
	)
	_, err := ReadContext(strings.NewReader(artifact))
	if err == nil || !strings.Contains(err.Error(), "context.text.sensitive") {
		t.Fatalf("sensitive context error = %v", err)
	}
}

func TestLoadChangeBindsContextBeforeFlowIntent(t *testing.T) {
	changeDir := t.TempDir()
	writeArtifact(t, changeDir, "context.md", minimalContext("sufficient", "None."))
	writeArtifact(t, changeDir, "intent.md", minimalFlowIntent("confirmed", "None.", true))
	writeArtifact(t, changeDir, "proposal.md", minimalProposal("OUT-1", "NG-1"))

	change, err := LoadChange(changeDir)
	if err != nil {
		t.Fatalf("load context-backed change: %v", err)
	}
	if change.Intent.ContextPack == nil || len(change.Intent.DesiredOutcomes[0].ContextRefs) != 1 ||
		change.Intent.DesiredOutcomes[0].ContextRefs[0] != "CTX-1" {
		t.Fatalf("context provenance was not bound: %#v", change.Intent)
	}
}

func TestLoadChangeRequiresMatchingContextPackDeclaration(t *testing.T) {
	tests := []struct {
		name   string
		intent func(string) string
	}{
		{
			name: "missing declaration",
			intent: func(value string) string {
				return strings.Replace(value, "- **Context Pack:** `CONTEXT-TEST` version 1\n", "", 1)
			},
		},
		{
			name: "wrong ID",
			intent: func(value string) string {
				return strings.Replace(value, "`CONTEXT-TEST` version 1", "`CONTEXT-OTHER` version 1", 1)
			},
		},
		{
			name: "wrong version",
			intent: func(value string) string {
				return strings.Replace(value, "`CONTEXT-TEST` version 1", "`CONTEXT-TEST` version 2", 1)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			changeDir := t.TempDir()
			writeArtifact(t, changeDir, "context.md", minimalContext("sufficient", "None."))
			writeArtifact(t, changeDir, "intent.md", test.intent(minimalFlowIntent("confirmed", "None.", true)))
			writeArtifact(t, changeDir, "proposal.md", minimalProposal("OUT-1", "NG-1"))

			_, err := LoadChange(changeDir)
			if !errors.Is(err, ErrMalformedArtifact) || !strings.Contains(err.Error(), "Context Pack") {
				t.Fatalf("Context Pack declaration error = %v, want ErrMalformedArtifact", err)
			}
		})
	}
}

func TestLoadChangeBlocksConfirmedIntentWithMaterialContextUnknown(t *testing.T) {
	changeDir := t.TempDir()
	unknowns := `| ID | Question | Sources |
|---|---|---|
| CTXQ-1 | Does the provider preserve a stable session join? | url:example.com/traces |`
	writeArtifact(t, changeDir, "context.md", minimalContext("material_unknown", unknowns))
	writeArtifact(t, changeDir, "intent.md", minimalFlowIntent("confirmed", "None.", true))

	_, err := LoadChange(changeDir)
	if err == nil || !strings.Contains(err.Error(), "intent.context.not_sufficient") {
		t.Fatalf("material context unknown error = %v", err)
	}
}

func TestLoadChangeRequiresContextForActiveGoalrailIntentChange(t *testing.T) {
	changeDir := t.TempDir()
	writeArtifact(t, changeDir, ".openspec.yaml", "schema: goalrail-intent\n")
	writeArtifact(t, changeDir, "intent.md", minimalFlowIntent("confirmed", "None.", true))
	writeArtifact(t, changeDir, "proposal.md", minimalProposal("OUT-1", "NG-1"))

	_, err := LoadChange(changeDir)
	if !errors.Is(err, ErrContextRequired) {
		t.Fatalf("missing context error = %v, want ErrContextRequired", err)
	}
}

func TestLoadChangeRequiresContextForArchivedV2IntentMetadata(t *testing.T) {
	repositoryRoot := t.TempDir()
	changeDir := filepath.Join(repositoryRoot, "openspec", "changes", "archive", "2026-07-22-v2-change")
	if err := os.MkdirAll(changeDir, 0o700); err != nil {
		t.Fatalf("create archived change: %v", err)
	}
	writeArtifact(t, changeDir, ".openspec.yaml", "schema: goalrail-intent\n")
	writeArtifact(t, changeDir, "intent.md", minimalFlowIntent("confirmed", "None.", true))
	writeArtifact(t, changeDir, "proposal.md", minimalProposal("OUT-1", "NG-1"))

	_, err := LoadChange(changeDir)
	if !errors.Is(err, ErrContextRequired) {
		t.Fatalf("missing archived v2 context error = %v, want ErrContextRequired", err)
	}
}

func TestLoadChangeReadsArchivedConfirmedArtifacts(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate adapter test file")
	}
	repositoryRoot := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
	change, err := LoadChange(filepath.Join(
		repositoryRoot,
		"openspec",
		"changes",
		"archive",
		"2026-07-22-intent-canary-v0",
	))
	if err != nil {
		t.Fatalf("load archived change: %v", err)
	}
	if change.Intent.Status != domain.IntentConfirmed || change.Intent.ID != "INTENT-CANARY-V0" {
		t.Fatalf("unexpected intent identity: %#v", change.Intent)
	}
	if len(change.Intent.SourceEvidence) != 7 || len(change.Intent.DesiredOutcomes) != 5 || len(change.Intent.NonGoals) != 5 || len(change.Intent.SuccessSignals) != 8 {
		t.Fatalf("unexpected intent group counts: evidence=%d outcomes=%d non_goals=%d signals=%d",
			len(change.Intent.SourceEvidence),
			len(change.Intent.DesiredOutcomes),
			len(change.Intent.NonGoals),
			len(change.Intent.SuccessSignals),
		)
	}
	if change.Intent.DesiredOutcomes[0].Statement == change.Intent.SourceEvidence[0].Statement {
		t.Fatal("source evidence and interpretation collapsed into one value")
	}
	if len(change.Proposal.Changes) != 4 || len(change.Proposal.PreservedNonGoalRefs) != 5 {
		t.Fatalf("unexpected proposal coverage: %#v", change.Proposal)
	}
}

func TestLoadChangeReadsArchivedContextEvaluationArtifacts(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate adapter test file")
	}
	repositoryRoot := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
	activeChangePath := filepath.Join(
		repositoryRoot,
		"openspec",
		"changes",
		"intent-context-evaluation-v0",
	)
	if _, err := os.Stat(activeChangePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("context-evaluation change remains active after archive: %v", err)
	}
	change, err := LoadChange(filepath.Join(
		repositoryRoot,
		"openspec",
		"changes",
		"archive",
		"2026-07-22-intent-context-evaluation-v0",
	))
	if err != nil {
		t.Fatalf("load archived context evaluation change: %v", err)
	}
	if change.Intent.ContextPack == nil || change.Intent.ContextPack.ID != "context-intent-context-evaluation-v0" {
		t.Fatalf("unexpected context pack: %#v", change.Intent.ContextPack)
	}
	if change.Intent.Status != domain.IntentConfirmed || len(change.Intent.DesiredOutcomes) != 7 ||
		len(change.Intent.NonGoals) != 6 || len(change.Intent.SuccessSignals) != 10 {
		t.Fatalf("unexpected archived intent: %#v", change.Intent)
	}
}

func TestLoadChangeReadsArchivedConfirmedRuntimeBoundaryArtifacts(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate adapter test file")
	}
	repositoryRoot := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
	activeChangePath := filepath.Join(
		repositoryRoot,
		"openspec",
		"changes",
		"stabilize-canary-runtime-boundary",
	)
	if _, err := os.Stat(activeChangePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("runtime-boundary change remains active after archive: %v", err)
	}
	change, err := LoadChange(filepath.Join(
		repositoryRoot,
		"openspec",
		"changes",
		"archive",
		"2026-07-22-stabilize-canary-runtime-boundary",
	))
	if err != nil {
		t.Fatalf("load runtime-boundary change: %v", err)
	}
	if change.Intent.Status != domain.IntentConfirmed || change.Intent.ID != "STABILIZE-CANARY-RUNTIME-BOUNDARY" {
		t.Fatalf("unexpected intent identity: %#v", change.Intent)
	}
	if len(change.Intent.SourceEvidence) != 5 || len(change.Intent.DesiredOutcomes) != 3 ||
		len(change.Intent.NonGoals) != 4 || len(change.Intent.SuccessSignals) != 5 {
		t.Fatalf("unexpected intent group counts: evidence=%d outcomes=%d non_goals=%d signals=%d",
			len(change.Intent.SourceEvidence),
			len(change.Intent.DesiredOutcomes),
			len(change.Intent.NonGoals),
			len(change.Intent.SuccessSignals),
		)
	}
	if len(change.Proposal.Changes) != 4 || len(change.Proposal.PreservedNonGoalRefs) != 4 {
		t.Fatalf("unexpected proposal coverage: %#v", change.Proposal)
	}
}

func TestLoadChangeBlocksCandidateBeforeReadingProposal(t *testing.T) {
	changeDir := t.TempDir()
	writeArtifact(t, changeDir, "intent.md", minimalIntent("candidate", "None.", false))

	_, err := LoadChange(changeDir)
	if !errors.Is(err, ErrIntentNotConfirmed) {
		t.Fatalf("load candidate error = %v, want ErrIntentNotConfirmed", err)
	}
	if strings.Contains(err.Error(), "proposal") {
		t.Fatalf("candidate gate read downstream proposal: %v", err)
	}
}

func TestLoadChangeRejectsUntracedProposalIntent(t *testing.T) {
	changeDir := t.TempDir()
	writeArtifact(t, changeDir, "intent.md", minimalIntent("confirmed", "None.", true))
	writeArtifact(t, changeDir, "proposal.md", minimalProposal("INVENTED-1", "NG-1"))

	_, err := LoadChange(changeDir)
	var validationErr *domain.ValidationError
	if !errors.As(err, &validationErr) || !strings.Contains(err.Error(), "proposal.change.intent_ref_unknown") {
		t.Fatalf("untraced proposal error = %v, want domain coverage violation", err)
	}
}

func TestLoadChangeRejectsMissingNonGoalCoverage(t *testing.T) {
	changeDir := t.TempDir()
	writeArtifact(t, changeDir, "intent.md", minimalIntent("confirmed", "None.", true))
	writeArtifact(t, changeDir, "proposal.md", minimalProposal("OUT-1", "None"))

	_, err := LoadChange(changeDir)
	if err == nil || !strings.Contains(err.Error(), "proposal.non_goal_not_preserved") {
		t.Fatalf("missing non-goal coverage error = %v", err)
	}
}

func TestReadIntentRejectsConfirmedUnresolvedAmbiguity(t *testing.T) {
	ambiguities := `| ID | Question | Evidence |
|---|---|---|
| AMB-1 | Which outcome applies? | SE-1 |`
	_, err := ReadIntent(strings.NewReader(minimalIntent("confirmed", ambiguities, true)))
	if err == nil || !strings.Contains(err.Error(), "intent.confirmed.ambiguities_unresolved") {
		t.Fatalf("confirmed ambiguity error = %v", err)
	}
}

func TestReadIntentRejectsUnstructuredAmbiguityText(t *testing.T) {
	_, err := ReadIntent(strings.NewReader(minimalIntent("candidate", "Maybe this means something else.", false)))
	if !errors.Is(err, ErrMalformedArtifact) {
		t.Fatalf("unstructured ambiguity error = %v, want ErrMalformedArtifact", err)
	}
}

func TestReadProposalUsesStableContentDerivedChangeIDs(t *testing.T) {
	proposal := minimalProposal("OUT-1", "NG-1")
	first, err := ReadProposal(strings.NewReader(proposal))
	if err != nil {
		t.Fatalf("read first proposal: %v", err)
	}
	second, err := ReadProposal(strings.NewReader(proposal))
	if err != nil {
		t.Fatalf("read second proposal: %v", err)
	}
	if first.Changes[0].ID == "" || first.Changes[0].ID != second.Changes[0].ID {
		t.Fatalf("proposal change ID is not stable: %q versus %q", first.Changes[0].ID, second.Changes[0].ID)
	}
}

func minimalIntent(status, ambiguities string, confirmed bool) string {
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

- **SE-1 — Owner statement:** The owner asked for a bounded result.

## Desired Outcomes

| ID | Confirmed wording | Verification action | Evidence |
|---|---|---|---|
| OUT-1 | Produce the bounded result. | Inspect it. | SE-1 |

## Non-Goals

| ID | Confirmed boundary | Evidence |
|---|---|---|
| NG-1 | Do not publish. | SE-1 |

## Observable Success Signals

| ID | Signal | Measurement | Evidence |
|---|---|---|---|
| SIG-1 | The result is inspectable. | One local artifact exists. | SE-1 |

## Ambiguities and Unknowns

` + ambiguities + `

## Confirmation

` + confirmation + `
`
}

func minimalFlowIntent(status, ambiguities string, confirmed bool) string {
	intent := strings.ReplaceAll(
		minimalIntent(status, ambiguities, confirmed),
		"| SE-1 |",
		"| SE-1, CTX-1 |",
	)
	return strings.Replace(
		intent,
		"- **Owner:** owner",
		"- **Owner:** owner\n- **Context Pack:** `CONTEXT-TEST` version 1",
		1,
	)
}

func minimalContext(outcome, unknowns string) string {
	return `# Context Pack

- **Context Pack ID:** CONTEXT-TEST
- **Version:** 1
- **Previous version:** pending
- **Started at:** 2026-07-20T08:00:00Z
- **Completed at:** 2026-07-20T08:02:00Z
- **Outcome:** ` + outcome + `

## Context Items

| ID | Kind | Claim | Source | Observed at | Relevance |
|---|---|---|---|---|---|
| CTX-1 | repository | The repository defines a bounded flow. | repo:README.md | 2026-07-20T08:01:00Z | This constrains the intended implementation. |

## Material Unknowns

` + unknowns + `
`
}

func minimalProposal(intentRefs, nonGoalRefs string) string {
	return `## Why

Test proposal.

## Intent Coverage

| Proposed change | Intent IDs | Non-goal preserved |
|---|---|---|
| Produce bounded result | ` + intentRefs + ` | ` + nonGoalRefs + ` |
`
}

func writeArtifact(t *testing.T, directory, name, contents string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(directory, name), []byte(contents), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}
