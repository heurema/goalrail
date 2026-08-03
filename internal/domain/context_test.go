package domain

import (
	"strings"
	"testing"
	"time"
)

var contextTestTime = time.Date(2026, time.July, 22, 12, 0, 0, 0, time.UTC)

func validContextPack() ContextPack {
	return ContextPack{
		ID:          "CONTEXT-1",
		Version:     1,
		StartedAt:   contextTestTime,
		CompletedAt: contextTestTime.Add(time.Minute),
		Outcome:     ContextSufficient,
		Items: []ContextItem{
			{
				ID:                 "CTX-1",
				Kind:               ContextRepository,
				Claim:              "The current canary manifest is frozen and not activated.",
				SourceRef:          "repo:canary/intent-canary-v0/manifest-v1.md",
				VerificationRecipe: "Read the manifest and expect activation to be false.",
				ObservedAt:         contextTestTime.Add(30 * time.Second),
				Relevance:          "A material flow change requires a new manifest version.",
			},
		},
	}
}

func TestValidateContextPackRejectsUnsafeVerificationRecipes(t *testing.T) {
	tests := []struct {
		name   string
		recipe string
		code   string
	}{
		{name: "multiline", recipe: "first line\nsecond line", code: "context.text.not_concise"},
		{name: "secret shaped", recipe: "password=example", code: "context.text.sensitive"},
		{name: "unbounded", recipe: strings.Repeat("x", maxContextRecipeRunes+1), code: "context.text.too_long"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pack := validContextPack()
			pack.Items[0].VerificationRecipe = test.recipe
			err := ValidateContextPack(pack)
			if err == nil || !strings.Contains(err.Error(), test.code) {
				t.Fatalf("verification recipe error = %v, want %s", err, test.code)
			}
		})
	}
}

func TestValidateContextPackAcceptsBoundedCompletionOutcomes(t *testing.T) {
	tests := []struct {
		name    string
		outcome ContextCollectionOutcome
		unknown []ContextUnknown
	}{
		{name: "sufficient", outcome: ContextSufficient},
		{
			name:    "material unknown",
			outcome: ContextMaterialUnknown,
			unknown: []ContextUnknown{{ID: "CTXQ-1", Question: "Does the child trace share the root session?", SourceRefs: []EvidenceReference{"probe:langfuse-child-session"}}},
		},
		{
			name:    "budget exhausted",
			outcome: ContextBudgetExhausted,
			unknown: []ContextUnknown{{ID: "CTXQ-1", Question: "Does the provider expose a stable child trace join?"}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pack := validContextPack()
			pack.Outcome = test.outcome
			pack.Unknowns = test.unknown
			if err := ValidateContextPack(pack); err != nil {
				t.Fatalf("validate context pack: %v", err)
			}
		})
	}
}

func TestValidateContextPackRejectsInconsistentStopOutcome(t *testing.T) {
	pack := validContextPack()
	pack.Unknowns = []ContextUnknown{{ID: "CTXQ-1", Question: "A material question remains."}}
	if err := ValidateContextPack(pack); err == nil || !strings.Contains(err.Error(), "context.sufficient.unknowns_forbidden") {
		t.Fatalf("sufficient pack error = %v", err)
	}

	pack = validContextPack()
	pack.Outcome = ContextMaterialUnknown
	if err := ValidateContextPack(pack); err == nil || !strings.Contains(err.Error(), "context.incomplete.unknown_required") {
		t.Fatalf("unresolved pack error = %v", err)
	}
}

func TestValidateContextPackRejectsRawOrSensitivePayloads(t *testing.T) {
	tests := []struct {
		name  string
		value string
		code  string
	}{
		{name: "multiline", value: "first line\nsecond line", code: "context.text.not_concise"},
		{name: "secret shaped", value: "authorization: Bearer example", code: "context.text.sensitive"},
		{name: "unbounded", value: strings.Repeat("x", maxContextClaimRunes+1), code: "context.text.too_long"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pack := validContextPack()
			pack.Items[0].Claim = test.value
			err := ValidateContextPack(pack)
			if err == nil || !strings.Contains(err.Error(), test.code) {
				t.Fatalf("context error = %v, want %s", err, test.code)
			}
		})
	}
}

func TestValidateContextPackRejectsSecretShapedSourceReferences(t *testing.T) {
	itemPack := validContextPack()
	itemPack.Items[0].SourceRef = "url:ghp_example"
	if err := ValidateContextPack(itemPack); err == nil || !strings.Contains(err.Error(), "context.source_ref.sensitive") {
		t.Fatalf("sensitive item source reference error = %v", err)
	}

	unknownPack := validContextPack()
	unknownPack.Outcome = ContextMaterialUnknown
	unknownPack.Unknowns = []ContextUnknown{{
		ID: "CTXQ-1", Question: "Does the provider expose a stable join?", SourceRefs: []EvidenceReference{"url:sk-lf-example"},
	}}
	if err := ValidateContextPack(unknownPack); err == nil || !strings.Contains(err.Error(), "context.source_ref.sensitive") {
		t.Fatalf("sensitive unknown source reference error = %v", err)
	}
}

func TestValidateFlowIntentSnapshotRequiresSufficientPriorContext(t *testing.T) {
	snapshot := validConfirmedIntent()
	if err := ValidateFlowIntentSnapshot(snapshot); err == nil || !strings.Contains(err.Error(), "intent.context.required") {
		t.Fatalf("missing context error = %v", err)
	}

	pack := validContextPack()
	pack.StartedAt = snapshot.Confirmation.ConfirmedAt.Add(-2 * time.Minute)
	pack.CompletedAt = snapshot.Confirmation.ConfirmedAt.Add(-time.Minute)
	pack.Items[0].ObservedAt = snapshot.Confirmation.ConfirmedAt.Add(-90 * time.Second)
	snapshot.ContextPack = &pack
	snapshot.DesiredOutcomes[0].ContextRefs = []ContextItemID{"CTX-1"}
	if err := ValidateFlowIntentSnapshot(snapshot); err != nil {
		t.Fatalf("validate flow intent: %v", err)
	}

	late := pack
	late.CompletedAt = snapshot.Confirmation.ConfirmedAt.Add(time.Minute)
	snapshot.ContextPack = &late
	if err := ValidateFlowIntentSnapshot(snapshot); err == nil || !strings.Contains(err.Error(), "intent.context.completed_after_confirmation") {
		t.Fatalf("late context error = %v", err)
	}

	unresolved := pack
	unresolved.Outcome = ContextMaterialUnknown
	unresolved.Unknowns = []ContextUnknown{{ID: "CTXQ-1", Question: "A material question remains."}}
	snapshot.ContextPack = &unresolved
	if err := ValidateFlowIntentSnapshot(snapshot); err == nil || !strings.Contains(err.Error(), "intent.context.not_sufficient") {
		t.Fatalf("unresolved context error = %v", err)
	}
}

func TestValidateIntentSnapshotRejectsUnknownOrChangedContextProvenance(t *testing.T) {
	snapshot := validConfirmedIntent()
	pack := validContextPack()
	pack.CompletedAt = snapshot.Confirmation.ConfirmedAt.Add(-time.Minute)
	snapshot.ContextPack = &pack
	snapshot.DesiredOutcomes[0].ContextRefs = []ContextItemID{"CTX-MISSING"}
	if err := ValidateIntentSnapshot(snapshot); err == nil || !strings.Contains(err.Error(), "intent.context_refs.unknown") {
		t.Fatalf("unknown context ref error = %v", err)
	}

	snapshot.DesiredOutcomes[0].ContextRefs = []ContextItemID{"CTX-1"}
	next := snapshot
	next.DesiredOutcomes = append([]IntentItem(nil), snapshot.DesiredOutcomes...)
	next.DesiredOutcomes[0].ContextRefs = nil
	if err := ValidateIntentAmendment(snapshot, next, AmendmentWordingOnly); err == nil ||
		!strings.Contains(err.Error(), "amendment.wording.context_refs_changed") {
		t.Fatalf("changed context provenance error = %v", err)
	}
}
