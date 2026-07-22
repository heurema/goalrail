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

func TestLoadChangeReadsConfirmedRuntimeBoundaryArtifacts(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate adapter test file")
	}
	repositoryRoot := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
	change, err := LoadChange(filepath.Join(
		repositoryRoot,
		"openspec",
		"changes",
		"stabilize-canary-runtime-boundary",
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
