package openspec

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestConformPairSelectsContractV1BeforeParsing(t *testing.T) {
	pair, err := ConformPair(
		[]byte(pairContext(true, true)),
		[]byte(pairIntent(true, "CTXP-test version 1", "Outcome", "Boundary", "SE-1, CTX-1")),
		"openspec/changes/test/context.md",
		"openspec/changes/test/intent.md",
	)
	if err != nil {
		t.Fatalf("conform contract v1 pair: %v", err)
	}
	if pair.Selection.Mode != ContractModeV1 || pair.Selection.Version != 1 {
		t.Fatalf("selection = %+v", pair.Selection)
	}
	if pair.Context.Items[0].VerificationRecipe != "Inspect the repository fact." {
		t.Fatalf("verification recipe was not preserved: %+v", pair.Context.Items[0])
	}
	if pair.Intent.ContextPack == nil || pair.Intent.ContextPack.ID != pair.Context.ID {
		t.Fatalf("intent is not bound to parsed context: %+v", pair.Intent.ContextPack)
	}
}

func TestConformPairLegacyBindingSpellingsHaveEqualSemantics(t *testing.T) {
	context := []byte(pairContext(false, true))
	withVersion, err := ConformPair(
		context,
		[]byte(pairIntent(false, "CTXP-test version 1", "Outcome", "Boundary", "SE-1, CTX-1")),
		"context.md",
		"intent.md",
	)
	if err != nil {
		t.Fatalf("conform version spelling: %v", err)
	}
	withV, err := ConformPair(
		context,
		[]byte(pairIntent(false, "CTXP-test v1", "Outcome", "Boundary", "SE-1, CTX-1")),
		"context.md",
		"intent.md",
	)
	if err != nil {
		t.Fatalf("conform v spelling: %v", err)
	}
	if withVersion.Selection.Mode != ContractModeLegacyV0 || withV.Selection.Mode != ContractModeLegacyV0 {
		t.Fatalf("unexpected selections: %+v %+v", withVersion.Selection, withV.Selection)
	}
	if !reflect.DeepEqual(withVersion.Context, withV.Context) || !reflect.DeepEqual(withVersion.Intent, withV.Intent) {
		t.Fatalf("legacy spellings changed semantics:\nversion=%+v\nv=%+v", withVersion, withV)
	}
}

func TestConformPairLegacyPinsSixColumnsAndHistoricalHeadings(t *testing.T) {
	pair, err := ConformPair(
		[]byte(pairContext(false, false)),
		[]byte(pairIntent(false, "CTXP-test v1", "Confirmed wording", "Confirmed boundary", "SE-1, CTX-1")),
		"context.md",
		"intent.md",
	)
	if err != nil {
		t.Fatalf("conform pinned historical pair: %v", err)
	}
	if pair.Context.Items[0].VerificationRecipe != "" {
		t.Fatalf("six-column input invented a recipe: %+v", pair.Context.Items[0])
	}
	if pair.Intent.DesiredOutcomes[0].Statement != "Produce the bounded result." ||
		pair.Intent.NonGoals[0].Statement != "Do not publish." {
		t.Fatalf("historical headings lost semantics: %+v", pair.Intent)
	}
}

func TestConformPairRejectsDeclaredContractWithoutLegacyFallback(t *testing.T) {
	tests := []struct {
		name     string
		context  string
		intent   string
		wantCode ArtifactDiagnosticCode
	}{
		{
			name:     "partial declaration",
			context:  strings.Replace(pairContext(true, true), "- **Artifact Contract Version:** 1\n", "", 1),
			intent:   pairIntent(true, "CTXP-test version 1", "Outcome", "Boundary", "SE-1, CTX-1"),
			wantCode: ArtifactContractInvalid,
		},
		{
			name:     "unknown version",
			context:  strings.Replace(pairContext(true, true), "Artifact Contract Version:** 1", "Artifact Contract Version:** 2", 1),
			intent:   strings.Replace(pairIntent(true, "CTXP-test version 1", "Outcome", "Boundary", "SE-1, CTX-1"), "Artifact Contract Version:** 1", "Artifact Contract Version:** 2", 1),
			wantCode: ArtifactContractUnsupported,
		},
		{
			name:     "mismatch",
			context:  pairContext(true, true),
			intent:   strings.Replace(pairIntent(true, "CTXP-test version 1", "Outcome", "Boundary", "SE-1, CTX-1"), "Artifact Contract Version:** 1", "Artifact Contract Version:** 2", 1),
			wantCode: ArtifactContractMismatch,
		},
		{
			name:     "v spelling under v1",
			context:  pairContext(true, true),
			intent:   pairIntent(true, "CTXP-test v1", "Outcome", "Boundary", "SE-1, CTX-1"),
			wantCode: ArtifactContextBindingInvalid,
		},
		{
			name: "duplicate declaration",
			context: strings.Replace(
				pairContext(true, true),
				"- **Artifact Contract:** goalrail-context-intent\n",
				"- **Artifact Contract:** goalrail-context-intent\n- **Artifact Contract:** goalrail-context-intent\n",
				1,
			),
			intent:   pairIntent(true, "CTXP-test version 1", "Outcome", "Boundary", "SE-1, CTX-1"),
			wantCode: ArtifactContractInvalid,
		},
		{
			name:    "empty identifier",
			context: strings.Replace(pairContext(true, true), "Artifact Contract:** goalrail-context-intent", "Artifact Contract:** ", 1),
			intent: strings.Replace(
				pairIntent(true, "CTXP-test version 1", "Outcome", "Boundary", "SE-1, CTX-1"),
				"Artifact Contract:** goalrail-context-intent",
				"Artifact Contract:** ",
				1,
			),
			wantCode: ArtifactContractInvalid,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ConformPair([]byte(test.context), []byte(test.intent), "context.md", "intent.md")
			var diagnostic *ArtifactDiagnostic
			if !errors.As(err, &diagnostic) {
				t.Fatalf("error is not an ArtifactDiagnostic: %v", err)
			}
			if diagnostic.Code != test.wantCode {
				t.Fatalf("code = %s, want %s: %v", diagnostic.Code, test.wantCode, err)
			}
			if test.wantCode != ArtifactContextBindingInvalid && diagnostic.ContractMode != ContractModeUnselected {
				t.Fatalf("selector failure mode = %q", diagnostic.ContractMode)
			}
		})
	}
}

func TestConformPairDoesNotTreatUnlistedLegacyMetadataOrHeadingsAsFallback(t *testing.T) {
	legacyContext := strings.Replace(
		pairContext(false, true),
		"- **Outcome:** sufficient\n",
		"- **Outcome:** sufficient\n- **Artifact Contrakt:** goalrail-context-intent\n",
		1,
	)
	_, err := ConformPair(
		[]byte(legacyContext),
		[]byte(pairIntent(false, "CTXP-test v1", "Outcome", "Boundary", "SE-1, CTX-1")),
		"context.md",
		"intent.md",
	)
	var diagnostic *ArtifactDiagnostic
	if !errors.As(err, &diagnostic) || diagnostic.Code != ArtifactFormatInvalid || diagnostic.ContractMode != ContractModeLegacyV0 {
		t.Fatalf("misspelled contract field was accepted as legacy: diagnostic=%+v err=%v", diagnostic, err)
	}

	_, err = ConformPair(
		[]byte(pairContext(true, true)),
		[]byte(pairIntent(true, "CTXP-test version 1", "Confirmed wording", "Confirmed boundary", "SE-1, CTX-1")),
		"context.md",
		"intent.md",
	)
	if !errors.As(err, &diagnostic) || diagnostic.Code != ArtifactFormatInvalid || diagnostic.ContractMode != ContractModeV1 {
		t.Fatalf("contract v1 retried historical headings: diagnostic=%+v err=%v", diagnostic, err)
	}
}

func TestConformPairReturnsBindingAndReferenceDiagnostics(t *testing.T) {
	tests := []struct {
		name     string
		binding  string
		evidence string
		wantCode ArtifactDiagnosticCode
	}{
		{name: "identity mismatch", binding: "CTXP-other version 1", evidence: "SE-1, CTX-1", wantCode: ArtifactContextBindingInvalid},
		{name: "version mismatch", binding: "CTXP-test version 2", evidence: "SE-1, CTX-1", wantCode: ArtifactContextBindingInvalid},
		{name: "missing reference", binding: "CTXP-test version 1", evidence: "SE-1, CTX-404", wantCode: ArtifactContextReferenceMissing},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ConformPair(
				[]byte(pairContext(true, true)),
				[]byte(pairIntent(true, test.binding, "Outcome", "Boundary", test.evidence)),
				"context.md",
				"intent.md",
			)
			var diagnostic *ArtifactDiagnostic
			if !errors.As(err, &diagnostic) || diagnostic.Code != test.wantCode {
				t.Fatalf("diagnostic = %+v, err = %v", diagnostic, err)
			}
		})
	}
}

func TestArtifactDiagnosticRenderingIsBoundedRedactedAndDeterministic(t *testing.T) {
	diagnostic := newArtifactDiagnostic(
		ArtifactFormatInvalid,
		"openspec/changes/test/intent.md",
		ArtifactKindIntent,
		ContractModeV1,
		"Status",
		"Bearer sk-secret\n"+strings.Repeat("x", 200),
		"candidate or confirmed",
		"replace the status value",
		ErrMalformedArtifact,
	)
	if diagnostic.Observation != "[REDACTED]" {
		t.Fatalf("observation = %q", diagnostic.Observation)
	}
	first, second := diagnostic.Error(), diagnostic.Error()
	if first != second {
		t.Fatalf("rendering changed: %q != %q", first, second)
	}
	if strings.Contains(first, "sk-secret") || strings.Contains(first, "\n") {
		t.Fatalf("rendering leaked hostile input: %q", first)
	}
	if !errors.Is(diagnostic, ErrMalformedArtifact) {
		t.Fatal("diagnostic did not preserve its sentinel cause")
	}

	escaped := sanitizeDiagnosticObservation("a\nb\t" + strings.Repeat("z", 100))
	if strings.ContainsRune(escaped, '\n') || len([]rune(escaped)) > 80 {
		t.Fatalf("observation is not escaped and bounded: %q", escaped)
	}
}

func TestConformPairRejectsUnsafeLogicalPathWithoutEchoingIt(t *testing.T) {
	for _, unsafePath := range []string{"/Users/private/context.md", "../private/context.md", `..\private\context.md`} {
		_, err := ConformPair([]byte("x"), []byte("x"), unsafePath, "intent.md")
		var diagnostic *ArtifactDiagnostic
		if !errors.As(err, &diagnostic) {
			t.Fatalf("error is not diagnostic for %q: %v", unsafePath, err)
		}
		if diagnostic.Path != "context.md" || strings.Contains(diagnostic.Error(), "private") {
			t.Fatalf("unsafe path escaped into diagnostic: %+v", diagnostic)
		}
	}
}

func pairContext(contract bool, sevenColumns bool) string {
	contractRows := ""
	if contract {
		contractRows = "- **Artifact Contract:** goalrail-context-intent\n- **Artifact Contract Version:** 1\n"
	}
	header := "| ID | Kind | Claim | Source | Verification recipe | Observed at | Relevance |\n|---|---|---|---|---|---|---|\n| CTX-1 | repository | The repository defines a bounded flow. | repo:README.md | Inspect the repository fact. | 2026-08-03T07:00:00Z | This constrains the outcome. |"
	if !sevenColumns {
		header = "| ID | Kind | Claim | Source | Observed at | Relevance |\n|---|---|---|---|---|---|\n| CTX-1 | repository | The repository defines a bounded flow. | repo:README.md | 2026-08-03T07:00:00Z | This constrains the outcome. |"
	}
	return `# Context Pack

- **Context Pack ID:** CTXP-test
- **Version:** 1
- **Previous version:** pending
` + contractRows + `- **Started at:** 2026-08-03T06:00:00Z
- **Completed at:** 2026-08-03T07:00:00Z
- **Outcome:** sufficient

## Context Items

` + header + `

## Material Unknowns

None.
`
}

func pairIntent(contract bool, binding string, outcomeHeader string, boundaryHeader string, evidence string) string {
	contractRows := ""
	if contract {
		contractRows = "- **Artifact Contract:** goalrail-context-intent\n- **Artifact Contract Version:** 1\n"
	}
	return `# Intent Snapshot

- **Intent ID:** INT-test
- **Version:** 1
- **Previous version:** pending
- **Status:** confirmed
- **Owner:** owner
- **Context Pack:** ` + binding + `
` + contractRows + `- **Run references:** local test

## Source Evidence

- **SE-1 — owner:** The owner requested a bounded result.

## Desired Outcomes

| ID | ` + outcomeHeader + ` | Verification action | Evidence |
|---|---|---|---|
| OUT-1 | Produce the bounded result. | Inspect it. | ` + evidence + ` |

## Non-Goals

| ID | ` + boundaryHeader + ` | Evidence |
|---|---|---|
| NG-1 | Do not publish. | SE-1, CTX-1 |

## Observable Success Signals

| ID | Signal | Measurement | Evidence |
|---|---|---|---|
| SIG-1 | The result is inspectable. | One local result exists. | SE-1, CTX-1 |

## Ambiguities and Unknowns

None.

## Confirmation

- **Confirmed by:** owner
- **Confirmed at:** 2026-08-03T08:00:00Z
- **Verification action:** The owner reviewed the three semantic groups.
`
}
