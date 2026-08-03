package ambient

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/heurema/goalrail/internal/adapters/openspec"
)

func intentArtifact(id, status string, version string) string {
	confirmation := `- **Confirmed by:** pending
- **Confirmed at:** pending
- **Verification action:** pending`
	if status == "confirmed" {
		confirmation = `- **Confirmed by:** owner
- **Confirmed at:** 2026-07-29
- **Verification action:** owner-reviewed-three-groups`
	}
	predecessor := ""
	if version != "1" {
		predecessor = "\n- **Previous version:** 1"
	}
	return `# Intent Snapshot

- **Intent ID:** ` + id + `
- **Version:** ` + version + predecessor + `
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

None.

## Confirmation

` + confirmation + `
`
}

func writeChange(t *testing.T, root, change, id, status, version string) {
	t.Helper()
	directory := filepath.Join(root, "openspec", "changes", change)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(directory, "intent.md"),
		[]byte(intentArtifact(id, status, version)),
		0o644,
	); err != nil {
		t.Fatal(err)
	}
}

func TestBindsExactlyOneConfirmedIntent(t *testing.T) {
	root := t.TempDir()
	writeChange(t, root, "current-change", "INTENT-CURRENT", "confirmed", "2")

	resolution := OpenSpecIntents{}.ActiveConfirmedIntent(root)
	reference := resolution.Reference
	if reference == nil {
		t.Fatalf("no binding, reason %q", resolution.UnboundReason)
	}
	if reference.ID != "INTENT-CURRENT" || reference.Version != 2 {
		t.Fatalf("reference = %+v", reference)
	}
	if !strings.HasPrefix(reference.Digest, "sha256:") || reference.Change != "current-change" {
		t.Fatalf("reference lacks resolvable identity: %+v", reference)
	}
}

func TestArchivedChangesDoNotBind(t *testing.T) {
	// An archived change is finished work; binding a live question to it
	// would point the chain at a decision already made.
	root := t.TempDir()
	directory := filepath.Join(root, "openspec", "changes", "archive", "2026-07-01-old")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(directory, "intent.md"),
		[]byte(intentArtifact("INTENT-OLD", "confirmed", "1")),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	resolution := OpenSpecIntents{}.ActiveConfirmedIntent(root)
	if resolution.Reference != nil {
		t.Fatalf("an archived change bound the question: %+v", resolution.Reference)
	}
	if resolution.UnboundReason != reasonNoChanges {
		t.Fatalf("reason = %q", resolution.UnboundReason)
	}
}

func TestUnconfirmedAndAmbiguousCasesReportWhy(t *testing.T) {
	t.Run("candidate only", func(t *testing.T) {
		root := t.TempDir()
		writeChange(t, root, "current-change", "INTENT-CURRENT", "candidate", "1")
		resolution := OpenSpecIntents{}.ActiveConfirmedIntent(root)
		if resolution.Reference != nil || resolution.UnboundReason != reasonNoConfirmed {
			t.Fatalf("resolution = %+v", resolution)
		}
	})

	t.Run("two confirmed", func(t *testing.T) {
		root := t.TempDir()
		writeChange(t, root, "change-one", "INTENT-ONE", "confirmed", "1")
		writeChange(t, root, "change-two", "INTENT-TWO", "confirmed", "1")
		resolution := OpenSpecIntents{}.ActiveConfirmedIntent(root)
		if resolution.Reference != nil || resolution.UnboundReason != reasonAmbiguous {
			t.Fatalf("resolution = %+v", resolution)
		}
	})

	t.Run("no openspec at all", func(t *testing.T) {
		resolution := OpenSpecIntents{}.ActiveConfirmedIntent(t.TempDir())
		if resolution.Reference != nil || resolution.UnboundReason != reasonNoChanges {
			t.Fatalf("resolution = %+v", resolution)
		}
	})
}

func TestAMalformedIntentDoesNotBlockAValidOne(t *testing.T) {
	root := t.TempDir()
	writeChange(t, root, "good-change", "INTENT-GOOD", "confirmed", "1")
	broken := filepath.Join(root, "openspec", "changes", "broken-change")
	if err := os.MkdirAll(broken, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(broken, "intent.md"), []byte("# not an intent"), 0o644); err != nil {
		t.Fatal(err)
	}

	resolution := OpenSpecIntents{}.ActiveConfirmedIntent(root)
	if resolution.Reference == nil || resolution.Reference.ID != "INTENT-GOOD" {
		t.Fatalf("resolution = %+v", resolution)
	}
}

func TestInvalidConfirmedPairWinsOverAValidIntent(t *testing.T) {
	root := t.TempDir()
	writeChange(t, root, "good-change", "INTENT-GOOD", "confirmed", "1")
	writeInvalidPairChange(t, root, "broken-change", "confirmed")

	resolution := OpenSpecIntents{}.ActiveConfirmedIntent(root)
	if resolution.Reference != nil || resolution.UnboundReason != reasonInvalidConfirmed {
		t.Fatalf("resolution = %+v", resolution)
	}
	if len(resolution.BindingDiagnostics) != 1 {
		t.Fatalf("diagnostics = %+v", resolution.BindingDiagnostics)
	}
	diagnostic := resolution.BindingDiagnostics[0]
	if diagnostic.Change != "broken-change" || diagnostic.Diagnostic.Code != openspec.ArtifactContractInvalid {
		t.Fatalf("diagnostic = %+v", diagnostic)
	}
	if diagnostic.Diagnostic.ContractMode != openspec.ContractModeUnselected {
		t.Fatalf("selector failure mode = %q", diagnostic.Diagnostic.ContractMode)
	}
}

func TestInvalidCandidatePairIsNotPromoted(t *testing.T) {
	root := t.TempDir()
	writeChange(t, root, "good-change", "INTENT-GOOD", "confirmed", "1")
	writeInvalidPairChange(t, root, "candidate-change", "candidate")

	resolution := OpenSpecIntents{}.ActiveConfirmedIntent(root)
	if resolution.Reference == nil || resolution.Reference.ID != "INTENT-GOOD" {
		t.Fatalf("resolution = %+v", resolution)
	}
	if len(resolution.BindingDiagnostics) != 0 {
		t.Fatalf("candidate produced diagnostics: %+v", resolution.BindingDiagnostics)
	}
}

func TestOnlyExactConfirmedStatusIsPromotedAsInvalidEvidence(t *testing.T) {
	root := t.TempDir()
	writeChange(t, root, "good-change", "INTENT-GOOD", "confirmed", "1")
	writeInvalidPairChange(t, root, "decorated-change", "confirmed")
	intentPath := filepath.Join(root, "openspec", "changes", "decorated-change", "intent.md")
	raw, err := os.ReadFile(intentPath)
	if err != nil {
		t.Fatal(err)
	}
	raw = []byte(strings.Replace(string(raw), "- **Status:** confirmed", "- **Status:** **confirmed**", 1))
	if err := os.WriteFile(intentPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	resolution := OpenSpecIntents{}.ActiveConfirmedIntent(root)
	if resolution.Reference == nil || resolution.Reference.ID != "INTENT-GOOD" || len(resolution.BindingDiagnostics) != 0 {
		t.Fatalf("resolution = %+v", resolution)
	}
}

func TestInvalidConfirmedDiagnosticsAreSortedByChange(t *testing.T) {
	root := t.TempDir()
	writeInvalidPairChange(t, root, "z-change", "confirmed")
	writeInvalidPairChange(t, root, "a-change", "confirmed")

	resolution := OpenSpecIntents{}.ActiveConfirmedIntent(root)
	if len(resolution.BindingDiagnostics) != 2 {
		t.Fatalf("diagnostics = %+v", resolution.BindingDiagnostics)
	}
	if resolution.BindingDiagnostics[0].Change != "a-change" || resolution.BindingDiagnostics[1].Change != "z-change" {
		t.Fatalf("diagnostic order = %+v", resolution.BindingDiagnostics)
	}
}

func TestInvalidConfirmedDiagnosticRedactsHostileContractValue(t *testing.T) {
	root := t.TempDir()
	writeInvalidPairChange(t, root, "hostile-change", "confirmed")
	directory := filepath.Join(root, "openspec", "changes", "hostile-change")
	for _, name := range []string{"context.md", "intent.md"} {
		path := filepath.Join(directory, name)
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		updated := strings.ReplaceAll(string(raw), "goalrail-context-intent", "sk-secret")
		if name == "context.md" {
			updated = strings.Replace(
				updated,
				"- **Artifact Contract:** sk-secret\n",
				"- **Artifact Contract:** sk-secret\n- **Artifact Contract Version:** 1\n",
				1,
			)
		}
		if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	resolution := OpenSpecIntents{}.ActiveConfirmedIntent(root)
	if len(resolution.BindingDiagnostics) != 1 {
		t.Fatalf("resolution = %+v", resolution)
	}
	diagnostic := resolution.BindingDiagnostics[0].Diagnostic
	if diagnostic.Observation != "[REDACTED]/1" || strings.Contains(diagnostic.Error(), "sk-secret") {
		t.Fatalf("hostile value escaped redaction: %+v", diagnostic)
	}
}

func writeInvalidPairChange(t *testing.T, root string, change string, status string) {
	t.Helper()
	directory := filepath.Join(root, "openspec", "changes", change)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	intent := strings.Replace(
		intentArtifact("INTENT-"+strings.ToUpper(strings.TrimSuffix(change, "-change")), status, "1"),
		"- **Owner:** owner\n",
		"- **Owner:** owner\n- **Context Pack:** CTXP-broken version 1\n- **Artifact Contract:** goalrail-context-intent\n- **Artifact Contract Version:** 1\n",
		1,
	)
	context := `# Context Pack

- **Context Pack ID:** CTXP-broken
- **Version:** 1
- **Artifact Contract:** goalrail-context-intent
- **Started at:** 2026-07-28T08:00:00Z
- **Completed at:** 2026-07-28T09:00:00Z
- **Outcome:** sufficient

## Context Items

| ID | Kind | Claim | Source | Verification recipe | Observed at | Relevance |
|---|---|---|---|---|---|---|
| CTX-1 | repository | A bounded fact. | repo:README.md | Read README.md. | 2026-07-28T08:30:00Z | It constrains the result. |

## Material Unknowns

None.
`
	if err := os.WriteFile(filepath.Join(directory, "intent.md"), []byte(intent), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "context.md"), []byte(context), 0o644); err != nil {
		t.Fatal(err)
	}
}
