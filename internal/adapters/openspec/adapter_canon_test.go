package openspec

import (
	"strings"
	"testing"
)

// The canon of 2026-08-02 changed the artifact shapes; the reader must accept
// both generations, because artifacts already written keep their shape forever.
// The ceiling round found the seam: templates changed, reader did not, and
// every newly scaffolded change would have failed gr prepare.
func TestTheReaderAcceptsBothArtifactGenerations(t *testing.T) {
	newContext := `# Context Pack

- **Context Pack ID:** CTXP-x
- **Version:** 1
- **Previous version:** pending
- **Started at:** 2026-08-02T09:00:00Z
- **Completed at:** 2026-08-02T10:00:00Z
- **Outcome:** sufficient

## Context Items

| ID | Kind | Claim | Source | Verification recipe | Observed at | Relevance |
|---|---|---|---|---|---|---|
| CTX-1 | repository | a claim | repo:docs/a.md | a bounded read-only recipe | 2026-08-02T10:00:00Z | why it matters |

## Material Unknowns

None.
`
	pack, err := ReadContext(strings.NewReader(newContext))
	if err != nil {
		t.Fatalf("the seven-column canon form is rejected: %v", err)
	}
	if len(pack.Items) != 1 || string(pack.Items[0].Relevance) != "why it matters" {
		t.Fatalf("the recipe column shifted the fields: %+v", pack.Items)
	}

	newIntent := `# Intent Snapshot

- **Intent ID:** INT-x
- **Version:** 1
- **Status:** candidate
- **Context Pack:** CTXP-x v1

## Source Evidence

- **SE-1 — owner:** said so.

## Desired Outcomes

| ID | Outcome | Verification action | Evidence |
|---|---|---|---|
| OUT-1 | it works | run it | SE-1 |

## Non-Goals

| ID | Boundary | Evidence |
|---|---|---|
| NG-1 | not that | SE-1 |

## Observable Success Signals

| ID | Signal | Measurement | Evidence |
|---|---|---|---|
| SIG-1 | green | count | SE-1 |

## Ambiguities and Unknowns

None.

## Confirmation

- **Confirmed by:** pending
- **Confirmed at:** pending
`
	snapshot, err := ReadIntent(strings.NewReader(newIntent))
	if err != nil {
		t.Fatalf("the neutral-heading canon form is rejected: %v", err)
	}
	if len(snapshot.DesiredOutcomes) != 1 || len(snapshot.NonGoals) != 1 {
		t.Fatalf("the neutral headings lost rows: %+v", snapshot)
	}
}
