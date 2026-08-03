package domain

import (
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	maxContextClaimRunes     = 512
	maxContextRecipeRunes    = 512
	maxContextRelevanceRunes = 320
	maxContextUnknownRunes   = 512
)

var forbiddenContextFragments = []string{
	"authorization:",
	"bearer ",
	"password=",
	"secret_key=",
	"api_key=",
	"-----begin private key-----",
	"-----begin rsa private key-----",
	"github_pat_",
	"ghp_",
	"sk-lf-",
}

// ValidateContextPack enforces the bounded evidence and stop semantics shared
// by every collector/provider. It does not decide whether a claim is true.
func ValidateContextPack(pack ContextPack) error {
	v := &validator{}
	if !IsCanonicalID(string(pack.ID)) {
		v.add("context.id.invalid", "id", "context pack ID must be canonical")
	}
	if pack.Version == 0 {
		v.add("context.version.invalid", "version", "version must be at least 1")
	}
	if pack.Version == 1 && pack.PreviousVersion != 0 {
		v.add("context.previous_version.invalid", "previous_version", "version 1 cannot name a predecessor")
	}
	if pack.Version > 1 && (pack.PreviousVersion == 0 || pack.PreviousVersion >= pack.Version) {
		v.add("context.previous_version.invalid", "previous_version", "later versions must name an earlier predecessor")
	}
	validateUTCTimestamp(v, "started_at", pack.StartedAt)
	validateUTCTimestamp(v, "completed_at", pack.CompletedAt)
	if !pack.StartedAt.IsZero() && !pack.CompletedAt.IsZero() && pack.CompletedAt.Before(pack.StartedAt) {
		v.add("context.time.reversed", "completed_at", "completion cannot precede collection start")
	}

	itemIDs := make(map[ContextItemID]struct{}, len(pack.Items))
	if len(pack.Items) == 0 {
		v.add("context.items.required", "items", "at least one context item is required")
	}
	for index, item := range pack.Items {
		path := fmt.Sprintf("items[%d]", index)
		if !IsCanonicalID(string(item.ID)) {
			v.add("context.item.id_invalid", path+".id", "context item ID must be canonical")
		} else if _, duplicate := itemIDs[item.ID]; duplicate {
			v.add("context.item.id_duplicate", path+".id", "context item IDs must be unique")
		} else {
			itemIDs[item.ID] = struct{}{}
		}
		if item.Kind != ContextRepository && item.Kind != ContextExternal {
			v.add("context.item.kind_invalid", path+".kind", "context item kind must be repository or external")
		}
		validateContextText(v, path+".claim", item.Claim, maxContextClaimRunes)
		// Empty is the legacy six-column representation. New seven-column
		// artifacts are required by their adapter to carry a recipe; once present,
		// it is retained and protected by the same bounded/sensitive-text rules as
		// the other context fields.
		if item.VerificationRecipe != "" {
			validateContextText(v, path+".verification_recipe", item.VerificationRecipe, maxContextRecipeRunes)
		}
		validateContextText(v, path+".relevance", item.Relevance, maxContextRelevanceRunes)
		validateContextSourceRef(v, path+".source_ref", item.SourceRef)
		validateUTCTimestamp(v, path+".observed_at", item.ObservedAt)
		if !item.ObservedAt.IsZero() && !pack.CompletedAt.IsZero() && item.ObservedAt.After(pack.CompletedAt) {
			v.add("context.item.observed_after_completion", path+".observed_at", "context item cannot be observed after pack completion")
		}
	}

	unknownIDs := make(map[ContextUnknownID]struct{}, len(pack.Unknowns))
	for index, unknown := range pack.Unknowns {
		path := fmt.Sprintf("unknowns[%d]", index)
		if !IsCanonicalID(string(unknown.ID)) {
			v.add("context.unknown.id_invalid", path+".id", "context unknown ID must be canonical")
		} else if _, duplicate := unknownIDs[unknown.ID]; duplicate {
			v.add("context.unknown.id_duplicate", path+".id", "context unknown IDs must be unique")
		} else {
			unknownIDs[unknown.ID] = struct{}{}
		}
		validateContextText(v, path+".question", unknown.Question, maxContextUnknownRunes)
		validateContextSourceRefs(v, path+".source_refs", unknown.SourceRefs)
	}

	switch pack.Outcome {
	case ContextSufficient:
		if len(pack.Unknowns) != 0 {
			v.add("context.sufficient.unknowns_forbidden", "unknowns", "sufficient context cannot retain material unknowns")
		}
	case ContextMaterialUnknown, ContextBudgetExhausted:
		if len(pack.Unknowns) == 0 {
			v.add("context.incomplete.unknown_required", "unknowns", "an incomplete collection outcome requires a material unknown")
		}
	default:
		v.add("context.outcome.invalid", "outcome", "outcome must be sufficient, material_unknown, or budget_exhausted")
	}
	return v.result()
}

func validateUTCTimestamp(v *validator, path string, value time.Time) {
	if value.IsZero() || value.Location() != time.UTC {
		v.add("context.timestamp.invalid", path, "timestamp must be a non-zero UTC value")
	}
}

func validateContextText(v *validator, path, value string, maxRunes int) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		v.add("context.text.required", path, "context text is required")
		return
	}
	if value != trimmed || strings.ContainsAny(value, "\r\n") {
		v.add("context.text.not_concise", path, "context text must be one trimmed line")
	}
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) > maxRunes {
		v.add("context.text.too_long", path, "context text exceeds its bounded size")
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			v.add("context.text.control_character", path, "context text cannot contain control characters")
			break
		}
	}
	lower := strings.ToLower(value)
	for _, fragment := range forbiddenContextFragments {
		if strings.Contains(lower, fragment) {
			v.add("context.text.sensitive", path, "context text contains credential or secret-shaped content")
			break
		}
	}
}

func validateContextSourceRef(v *validator, path string, ref EvidenceReference) {
	value := string(ref)
	if !IsEvidenceReference(value) {
		v.add("context.source_ref.invalid", path, "source reference must be bounded")
		return
	}
	lower := strings.ToLower(value)
	for _, fragment := range forbiddenContextFragments {
		if strings.Contains(lower, fragment) {
			v.add("context.source_ref.sensitive", path, "source reference contains credential or secret-shaped content")
			return
		}
	}
}

func validateContextSourceRefs(v *validator, path string, refs []EvidenceReference) {
	seen := make(map[EvidenceReference]struct{}, len(refs))
	for index, ref := range refs {
		itemPath := fmt.Sprintf("%s[%d]", path, index)
		if !IsEvidenceReference(string(ref)) {
			validateContextSourceRef(v, itemPath, ref)
			continue
		}
		validateContextSourceRef(v, itemPath, ref)
		if _, duplicate := seen[ref]; duplicate {
			v.add("context.source_ref.duplicate", itemPath, "source reference is duplicated")
		}
		seen[ref] = struct{}{}
	}
}
