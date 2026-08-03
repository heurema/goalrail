package openspec

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/heurema/goalrail/internal/domain"
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
	if len(pack.Items) != 1 || string(pack.Items[0].Relevance) != "why it matters" ||
		pack.Items[0].VerificationRecipe != "a bounded read-only recipe" {
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

func TestCanonTemplateGapsPlanningArtifactsUseTheBoundReader(t *testing.T) {
	repositoryRoot := adapterRepositoryRoot(t)
	changeDir := canonTemplateGapsChangeDir(t)
	compiled, err := LoadChange(changeDir)
	if err != nil {
		t.Fatalf("load real change through the production reader: %v", err)
	}
	snapshot := compiled.Intent
	if snapshot.ID != "INT-canon-template-gaps-v0" || snapshot.Version != 3 {
		t.Fatalf("unexpected real intent identity: %s version %d", snapshot.ID, snapshot.Version)
	}
	if snapshot.ContextPack == nil || snapshot.ContextPack.Version != 3 {
		t.Fatalf("unexpected real context binding: %+v", snapshot.ContextPack)
	}
	for _, item := range snapshot.ContextPack.Items {
		if item.VerificationRecipe == "" {
			t.Fatalf("real context item %s lost its verification recipe", item.ID)
		}
	}
	rawContextV2, err := os.ReadFile(filepath.Join(changeDir, "context-v2.md"))
	if err != nil {
		t.Fatalf("read retained context v2: %v", err)
	}
	contextV2, err := ReadContext(bytes.NewReader(rawContextV2))
	if err != nil {
		t.Fatalf("parse retained context v2: %v", err)
	}
	rawIntentV2, err := os.ReadFile(filepath.Join(changeDir, "intent-v2.md"))
	if err != nil {
		t.Fatalf("read retained intent v2: %v", err)
	}
	intentV2, err := readIntent(bytes.NewReader(rawIntentV2), &contextV2)
	if err != nil {
		t.Fatalf("parse retained intent v2: %v", err)
	}
	if contextV2.Version != 2 || intentV2.Version != 2 ||
		intentV2.Status != domain.IntentCandidate || snapshot.PreviousVersion != intentV2.Version {
		t.Fatalf("broken retained lineage: context=%d intent=%d status=%s current.previous=%d",
			contextV2.Version, intentV2.Version, intentV2.Status, snapshot.PreviousVersion)
	}

	intentPath := filepath.Join(changeDir, "intent.md")
	rawIntent, err := os.ReadFile(intentPath)
	if err != nil {
		t.Fatalf("read real intent for resolver digest: %v", err)
	}
	digest := sha256.Sum256(rawIntent)
	artifactRef, err := filepath.Rel(repositoryRoot, intentPath)
	if err != nil {
		t.Fatalf("relativize real intent: %v", err)
	}
	if err := (IntentResolver{}).Verify(repositoryRoot, domain.WorkSpecIntentReference{
		ID:          snapshot.ID,
		Version:     snapshot.Version,
		ArtifactRef: filepath.ToSlash(artifactRef),
		Digest:      "sha256:" + hex.EncodeToString(digest[:]),
	}); err != nil {
		t.Fatalf("verify real intent through the production resolver: %v", err)
	}
}

func canonTemplateGapsChangeDir(t *testing.T) string {
	t.Helper()
	repositoryRoot := adapterRepositoryRoot(t)
	active := filepath.Join(repositoryRoot, "openspec", "changes", "canon-template-gaps-v0")
	if _, err := os.Stat(active); err == nil {
		return active
	}
	matches, err := filepath.Glob(filepath.Join(
		repositoryRoot,
		"openspec",
		"changes",
		"archive",
		"*-canon-template-gaps-v0",
	))
	if err != nil || len(matches) != 1 {
		t.Fatalf("locate canon-template-gaps-v0: matches=%v err=%v", matches, err)
	}
	return matches[0]
}

func adapterRepositoryRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test file")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", ".."))
}
