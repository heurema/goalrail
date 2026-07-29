package domain

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

var (
	canonicalIDPattern       = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
	evidenceReferencePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{1,31}:[A-Za-z0-9][A-Za-z0-9._/@:+-]{0,223}$`)
)

// ValidationViolation is stable machine-readable validation evidence.
type ValidationViolation struct {
	Code    string
	Path    string
	Message string
}

// ValidationError reports every deterministic violation found in one pass.
type ValidationError struct {
	Violations []ValidationViolation
}

func (e *ValidationError) Error() string {
	parts := make([]string, 0, len(e.Violations))
	for _, violation := range e.Violations {
		parts = append(parts, fmt.Sprintf(
			"%s (%s): %s",
			violation.Code,
			violation.Path,
			violation.Message,
		))
	}
	return strings.Join(parts, "; ")
}

type validator struct {
	violations []ValidationViolation
}

func (v *validator) add(code, path, message string) {
	v.violations = append(v.violations, ValidationViolation{
		Code:    code,
		Path:    path,
		Message: message,
	})
}

func (v *validator) absorb(err error) {
	if err == nil {
		return
	}
	validationErr, ok := err.(*ValidationError)
	if !ok {
		v.add("validation.internal", "", err.Error())
		return
	}
	v.violations = append(v.violations, validationErr.Violations...)
}

func (v *validator) result() error {
	if len(v.violations) == 0 {
		return nil
	}
	return &ValidationError{Violations: append([]ValidationViolation(nil), v.violations...)}
}

func validCanonicalID(value string) bool {
	return IsCanonicalID(value)
}

// IsCanonicalID reports whether value is safe to use as a provider-neutral
// stable identifier in canonical records.
func IsCanonicalID(value string) bool {
	return canonicalIDPattern.MatchString(value)
}

// IsEvidenceReference reports whether value is a bounded provider-neutral
// scheme:identifier reference that is safe to retain.
func IsEvidenceReference(value string) bool {
	return evidenceReferencePattern.MatchString(value) && !hasSecretShapedContent(value)
}

// ValidateIntentSnapshot checks provider-neutral lifecycle and provenance
// invariants. It validates structure but never grants effect authority.
func ValidateIntentSnapshot(snapshot IntentSnapshot) error {
	v := &validator{}

	if !validCanonicalID(string(snapshot.ID)) {
		v.add("intent.id.invalid", "id", "intent ID must be a canonical non-empty ID")
	}
	if snapshot.Version == 0 {
		v.add("intent.version.invalid", "version", "version must be at least 1")
	}
	if snapshot.Version == 1 && snapshot.PreviousVersion != 0 {
		v.add("intent.previous_version.invalid", "previous_version", "version 1 cannot name a predecessor")
	}
	if snapshot.Version > 1 && (snapshot.PreviousVersion == 0 || snapshot.PreviousVersion >= snapshot.Version) {
		v.add("intent.previous_version.invalid", "previous_version", "later versions must name an earlier predecessor")
	}

	evidenceIDs := make(map[SourceEvidenceID]struct{}, len(snapshot.SourceEvidence))
	if len(snapshot.SourceEvidence) == 0 {
		v.add("intent.source_evidence.required", "source_evidence", "at least one source evidence item is required")
	}
	for index, evidence := range snapshot.SourceEvidence {
		path := fmt.Sprintf("source_evidence[%d]", index)
		if !validCanonicalID(string(evidence.ID)) {
			v.add("intent.source_evidence.id_invalid", path+".id", "source evidence ID must be canonical")
		} else if _, exists := evidenceIDs[evidence.ID]; exists {
			v.add("intent.source_evidence.id_duplicate", path+".id", "source evidence IDs must be unique")
		} else {
			evidenceIDs[evidence.ID] = struct{}{}
		}
		if evidence.Kind != EvidenceOwnerStatement && evidence.Kind != EvidenceRepositoryFact {
			v.add("intent.source_evidence.kind_invalid", path+".kind", "source evidence kind is unsupported")
		}
		if strings.TrimSpace(evidence.Statement) == "" && strings.TrimSpace(evidence.Reference) == "" {
			v.add("intent.source_evidence.content_required", path, "source evidence requires a statement or reference")
		}
	}

	if len(snapshot.DesiredOutcomes) == 0 {
		v.add("intent.desired_outcomes.required", "desired_outcomes", "at least one desired outcome is required")
	}
	if len(snapshot.SuccessSignals) == 0 {
		v.add("intent.success_signals.required", "success_signals", "at least one observable success signal is required")
	}

	contextIDs := make(map[ContextItemID]struct{})
	if snapshot.ContextPack != nil {
		v.absorb(ValidateContextPack(*snapshot.ContextPack))
		for _, item := range snapshot.ContextPack.Items {
			contextIDs[item.ID] = struct{}{}
		}
	}
	semanticIDs := make(map[IntentItemID]string)
	validateIntentGroup(v, "desired_outcomes", snapshot.DesiredOutcomes, evidenceIDs, contextIDs, semanticIDs)
	validateIntentGroup(v, "non_goals", snapshot.NonGoals, evidenceIDs, contextIDs, semanticIDs)
	validateIntentGroup(v, "success_signals", snapshot.SuccessSignals, evidenceIDs, contextIDs, semanticIDs)

	ambiguityIDs := make(map[AmbiguityID]struct{}, len(snapshot.Ambiguities))
	for index, ambiguity := range snapshot.Ambiguities {
		path := fmt.Sprintf("ambiguities[%d]", index)
		if !validCanonicalID(string(ambiguity.ID)) {
			v.add("intent.ambiguity.id_invalid", path+".id", "ambiguity ID must be canonical")
		} else if _, exists := ambiguityIDs[ambiguity.ID]; exists {
			v.add("intent.ambiguity.id_duplicate", path+".id", "ambiguity IDs must be unique")
		} else {
			ambiguityIDs[ambiguity.ID] = struct{}{}
		}
		if strings.TrimSpace(ambiguity.Question) == "" {
			v.add("intent.ambiguity.question_required", path+".question", "ambiguity requires a question")
		}
		validateEvidenceRefs(v, path+".evidence_refs", ambiguity.EvidenceRefs, evidenceIDs)
	}

	switch snapshot.Status {
	case IntentCandidate:
		if snapshot.Confirmation != nil {
			v.add("intent.candidate.confirmation_forbidden", "confirmation", "candidate intent cannot carry confirmation")
		}
	case IntentConfirmed:
		if len(snapshot.Ambiguities) > 0 {
			v.add("intent.confirmed.ambiguities_unresolved", "ambiguities", "confirmed intent cannot contain unresolved ambiguities")
		}
		validateConfirmation(v, snapshot.Confirmation)
	default:
		v.add("intent.status.invalid", "status", "status must be candidate or confirmed")
	}

	validateEscalationResolution(v, snapshot.ResolvedEscalation)

	return v.result()
}

// validateEscalationResolution rejects a malformed answering record. An absent
// record is valid and carries no inferred disposition: a version that answers
// nothing is the ordinary case.
func validateEscalationResolution(v *validator, resolution *IntentEscalationResolution) {
	if resolution == nil {
		return
	}
	if !validCanonicalID(string(resolution.RunID)) {
		v.add(
			"intent.resolves.run_invalid",
			"resolves.run_id",
			"resolved run ID must be canonical",
		)
	}
	if !validEscalationDigest(resolution.EscalationDigest) {
		v.add(
			"intent.resolves.digest_invalid",
			"resolves.escalation_digest",
			"resolved escalation digest must be SHA-256",
		)
	}
	switch resolution.Disposition {
	case DispositionAnswered, DispositionSpurious, DispositionWithdrawn:
	default:
		v.add(
			"intent.resolves.disposition_invalid",
			"resolves.disposition",
			"disposition must be answered, spurious, or withdrawn",
		)
	}
}

func validEscalationDigest(value string) bool {
	const prefix = "sha256:"
	if !strings.HasPrefix(value, prefix) || len(value) != len(prefix)+64 {
		return false
	}
	for _, character := range value[len(prefix):] {
		isDigit := character >= '0' && character <= '9'
		isLowerHex := character >= 'a' && character <= 'f'
		if !isDigit && !isLowerHex {
			return false
		}
	}
	return true
}

// ValidateFlowIntentSnapshot applies the context gate required by the v2 flow
// while leaving legacy and baseline Intent Snapshot validation compatible.
func ValidateFlowIntentSnapshot(snapshot IntentSnapshot) error {
	v := &validator{}
	v.absorb(ValidateIntentSnapshot(snapshot))
	if snapshot.ContextPack == nil {
		v.add("intent.context.required", "context_pack", "flow intent requires a Context Pack")
		return v.result()
	}
	pack := snapshot.ContextPack
	if pack.Outcome != ContextSufficient {
		v.add("intent.context.not_sufficient", "context_pack.outcome", "flow intent requires sufficient context")
	}
	if snapshot.Status == IntentConfirmed && snapshot.Confirmation != nil &&
		!pack.CompletedAt.Before(snapshot.Confirmation.ConfirmedAt) {
		v.add("intent.context.completed_after_confirmation", "context_pack.completed_at", "context must complete before intent confirmation")
	}
	return v.result()
}

func validateIntentGroup(
	v *validator,
	group string,
	items []IntentItem,
	evidenceIDs map[SourceEvidenceID]struct{},
	contextIDs map[ContextItemID]struct{},
	semanticIDs map[IntentItemID]string,
) {
	for index, item := range items {
		path := fmt.Sprintf("%s[%d]", group, index)
		if !validCanonicalID(string(item.ID)) {
			v.add("intent.item.id_invalid", path+".id", "intent item ID must be canonical")
		} else if previousGroup, exists := semanticIDs[item.ID]; exists {
			v.add("intent.item.id_duplicate", path+".id", fmt.Sprintf("intent item ID already appears in %s", previousGroup))
		} else {
			semanticIDs[item.ID] = group
		}
		if strings.TrimSpace(item.Statement) == "" {
			v.add("intent.item.statement_required", path+".statement", "intent item requires a statement")
		}
		validateEvidenceRefs(v, path+".evidence_refs", item.EvidenceRefs, evidenceIDs)
		validateContextRefs(v, path+".context_refs", item.ContextRefs, contextIDs)
	}
}

func validateContextRefs(
	v *validator,
	path string,
	references []ContextItemID,
	contextIDs map[ContextItemID]struct{},
) {
	seen := make(map[ContextItemID]struct{}, len(references))
	for index, reference := range references {
		itemPath := fmt.Sprintf("%s[%d]", path, index)
		if _, duplicate := seen[reference]; duplicate {
			v.add("intent.context_refs.duplicate", itemPath, "context item reference is duplicated")
			continue
		}
		seen[reference] = struct{}{}
		if _, exists := contextIDs[reference]; !exists {
			v.add("intent.context_refs.unknown", itemPath, "context item reference does not exist in the bound pack")
		}
	}
}

func validateEvidenceRefs(
	v *validator,
	path string,
	references []SourceEvidenceID,
	evidenceIDs map[SourceEvidenceID]struct{},
) {
	if len(references) == 0 {
		v.add("intent.evidence_refs.required", path, "at least one source evidence reference is required")
		return
	}
	seen := make(map[SourceEvidenceID]struct{}, len(references))
	for index, reference := range references {
		itemPath := fmt.Sprintf("%s[%d]", path, index)
		if _, exists := seen[reference]; exists {
			v.add("intent.evidence_refs.duplicate", itemPath, "source evidence reference is duplicated")
			continue
		}
		seen[reference] = struct{}{}
		if _, exists := evidenceIDs[reference]; !exists {
			v.add("intent.evidence_refs.unknown", itemPath, "source evidence reference does not exist")
		}
	}
}

func validateConfirmation(v *validator, confirmation *IntentConfirmation) {
	if confirmation == nil {
		v.add("intent.confirmed.confirmation_required", "confirmation", "confirmed intent requires an active owner confirmation")
		return
	}
	if strings.TrimSpace(confirmation.Owner) == "" {
		v.add("intent.confirmed.owner_required", "confirmation.owner", "confirmation requires an owner")
	}
	if confirmation.ConfirmedAt.IsZero() {
		v.add("intent.confirmed.timestamp_required", "confirmation.confirmed_at", "confirmation requires a timestamp")
	}
	if strings.TrimSpace(confirmation.VerificationAction) == "" {
		v.add("intent.confirmed.verification_required", "confirmation.verification_action", "confirmation requires an active verification action")
	}
}

// ValidateIntentAmendment checks version linkage and stable semantic identity.
// The caller classifies meaning as material or wording-only; this function
// enforces the corresponding lifecycle transition without guessing semantics.
func ValidateIntentAmendment(previous, next IntentSnapshot, kind AmendmentKind) error {
	v := &validator{}
	v.absorb(ValidateIntentSnapshot(previous))
	v.absorb(ValidateIntentSnapshot(next))

	if previous.ID != next.ID {
		v.add("amendment.intent_id_changed", "id", "an amendment must preserve the intent lineage ID")
	}

	previousGroups := semanticItemGroups(previous)
	for _, group := range intentGroups(next) {
		for index, item := range group.items {
			if previousGroup, exists := previousGroups[item.ID]; exists && previousGroup != group.name {
				v.add(
					"amendment.intent_id_reassigned",
					fmt.Sprintf("%s[%d].id", group.name, index),
					fmt.Sprintf("stable intent item ID moved from %s", previousGroup),
				)
			}
		}
	}
	validateStableIntentItemIDs(v, previous, next)

	switch kind {
	case AmendmentMaterial:
		if previous.Status != IntentConfirmed {
			v.add("amendment.material.predecessor_not_confirmed", "previous.status", "material amendment requires a confirmed predecessor")
		}
		if next.Version != previous.Version+1 {
			v.add("amendment.material.version_invalid", "version", "material amendment must increment version by exactly one")
		}
		if next.PreviousVersion != previous.Version {
			v.add("amendment.material.predecessor_invalid", "previous_version", "material amendment must link to the predecessor version")
		}
		if next.Status != IntentCandidate {
			v.add("amendment.material.candidate_required", "status", "material amendment must return to candidate status")
		}
		if next.Confirmation != nil {
			v.add("amendment.material.reconfirmation_required", "confirmation", "material amendment must clear prior confirmation")
		}
	case AmendmentWordingOnly:
		if next.Version != previous.Version || next.PreviousVersion != previous.PreviousVersion {
			v.add("amendment.wording.version_changed", "version", "wording-only edit must retain version linkage")
		}
		if next.Status != previous.Status {
			v.add("amendment.wording.status_changed", "status", "wording-only edit must retain lifecycle status")
		}
		if !sameConfirmation(previous.Confirmation, next.Confirmation) {
			v.add("amendment.wording.confirmation_changed", "confirmation", "wording-only edit must preserve confirmation")
		}
		validateSameIntentItemIDs(v, previous, next)
		validateWordingOnlyProvenance(v, previous, next)
	default:
		v.add("amendment.kind.invalid", "kind", "amendment kind must be wording_only or material")
	}

	return v.result()
}

type namedIntentGroup struct {
	name  string
	items []IntentItem
}

func intentGroups(snapshot IntentSnapshot) []namedIntentGroup {
	return []namedIntentGroup{
		{name: "desired_outcomes", items: snapshot.DesiredOutcomes},
		{name: "non_goals", items: snapshot.NonGoals},
		{name: "success_signals", items: snapshot.SuccessSignals},
	}
}

func semanticItemGroups(snapshot IntentSnapshot) map[IntentItemID]string {
	groups := make(map[IntentItemID]string)
	for _, group := range intentGroups(snapshot) {
		for _, item := range group.items {
			groups[item.ID] = group.name
		}
	}
	return groups
}

func validateSameIntentItemIDs(v *validator, previous, next IntentSnapshot) {
	previousGroups := intentGroups(previous)
	nextGroups := intentGroups(next)
	for groupIndex, previousGroup := range previousGroups {
		nextGroup := nextGroups[groupIndex]
		if len(previousGroup.items) != len(nextGroup.items) {
			v.add("amendment.wording.intent_ids_changed", previousGroup.name, "wording-only edit must preserve intent item IDs")
			continue
		}
		nextIDs := make(map[IntentItemID]struct{}, len(nextGroup.items))
		for _, item := range nextGroup.items {
			nextIDs[item.ID] = struct{}{}
		}
		for index, item := range previousGroup.items {
			if _, exists := nextIDs[item.ID]; !exists {
				v.add(
					"amendment.wording.intent_ids_changed",
					fmt.Sprintf("%s[%d].id", previousGroup.name, index),
					"wording-only edit must preserve intent item IDs",
				)
			}
		}
	}
}

func validateWordingOnlyProvenance(v *validator, previous, next IntentSnapshot) {
	if !reflect.DeepEqual(previous.ContextPack, next.ContextPack) {
		v.add("amendment.wording.context_pack_changed", "context_pack", "wording-only edit must preserve the Context Pack")
	}
	previousEvidence := make(map[SourceEvidenceID]SourceEvidence, len(previous.SourceEvidence))
	for _, evidence := range previous.SourceEvidence {
		previousEvidence[evidence.ID] = evidence
	}
	if len(previous.SourceEvidence) != len(next.SourceEvidence) {
		v.add("amendment.wording.source_evidence_changed", "source_evidence", "wording-only edit must preserve source evidence")
	} else {
		for index, evidence := range next.SourceEvidence {
			previousValue, exists := previousEvidence[evidence.ID]
			if !exists || previousValue != evidence {
				v.add(
					"amendment.wording.source_evidence_changed",
					fmt.Sprintf("source_evidence[%d]", index),
					"wording-only edit must preserve source evidence",
				)
			}
		}
	}

	previousItems := make(map[IntentItemID]IntentItem)
	for _, group := range intentGroups(previous) {
		for _, item := range group.items {
			previousItems[item.ID] = item
		}
	}
	for _, group := range intentGroups(next) {
		for index, item := range group.items {
			previousItem, exists := previousItems[item.ID]
			if exists {
				if !sameEvidenceReferenceSet(previousItem.EvidenceRefs, item.EvidenceRefs) {
					v.add(
						"amendment.wording.evidence_refs_changed",
						fmt.Sprintf("%s[%d].evidence_refs", group.name, index),
						"wording-only edit must preserve each intent item's evidence references",
					)
				}
				if !sameContextReferenceSet(previousItem.ContextRefs, item.ContextRefs) {
					v.add(
						"amendment.wording.context_refs_changed",
						fmt.Sprintf("%s[%d].context_refs", group.name, index),
						"wording-only edit must preserve each intent item's context references",
					)
				}
			}
		}
	}
}

func sameEvidenceReferenceSet(left, right []SourceEvidenceID) bool {
	if len(left) != len(right) {
		return false
	}
	references := make(map[SourceEvidenceID]uint32, len(left))
	for _, reference := range left {
		references[reference]++
	}
	for _, reference := range right {
		if references[reference] == 0 {
			return false
		}
		references[reference]--
	}
	return true
}

func sameContextReferenceSet(left, right []ContextItemID) bool {
	if len(left) != len(right) {
		return false
	}
	references := make(map[ContextItemID]uint32, len(left))
	for _, reference := range left {
		references[reference]++
	}
	for _, reference := range right {
		if references[reference] == 0 {
			return false
		}
		references[reference]--
	}
	return true
}

func validateStableIntentItemIDs(v *validator, previous, next IntentSnapshot) {
	previousGroups := intentGroups(previous)
	nextGroups := intentGroups(next)
	for groupIndex, previousGroup := range previousGroups {
		nextGroup := nextGroups[groupIndex]
		for nextIndex, nextItem := range nextGroup.items {
			for _, previousItem := range previousGroup.items {
				if strings.TrimSpace(previousItem.Statement) == strings.TrimSpace(nextItem.Statement) &&
					previousItem.ID != nextItem.ID {
					v.add(
						"amendment.intent_item_id_changed",
						fmt.Sprintf("%s[%d].id", nextGroup.name, nextIndex),
						"an unchanged intent item must retain its stable ID",
					)
				}
			}
		}
	}
}

func sameConfirmation(left, right *IntentConfirmation) bool {
	if left == nil || right == nil {
		return left == right
	}
	return left.Owner == right.Owner &&
		left.ConfirmedAt.Equal(right.ConfirmedAt) &&
		left.VerificationAction == right.VerificationAction
}

// ValidateProposalCoverage gates provider-neutral proposal compilation on a
// confirmed intent and rejects untraced changes, invented intent, or non-goal
// conflicts. Passing this validation still grants no effect authority.
func ValidateProposalCoverage(intent IntentSnapshot, proposal Proposal) error {
	v := &validator{}
	v.absorb(ValidateIntentSnapshot(intent))

	if intent.Status != IntentConfirmed {
		v.add("proposal.intent_not_confirmed", "intent.status", "proposal compilation requires confirmed intent")
	}
	if len(proposal.Changes) == 0 {
		v.add("proposal.changes.required", "changes", "proposal requires at least one traced change")
	}

	traceableIntent := make(map[IntentItemID]struct{})
	for _, item := range intent.DesiredOutcomes {
		traceableIntent[item.ID] = struct{}{}
	}
	for _, item := range intent.SuccessSignals {
		traceableIntent[item.ID] = struct{}{}
	}
	nonGoals := make(map[IntentItemID]struct{}, len(intent.NonGoals))
	for _, item := range intent.NonGoals {
		nonGoals[item.ID] = struct{}{}
	}

	changeIDs := make(map[ProposalChangeID]struct{}, len(proposal.Changes))
	for index, change := range proposal.Changes {
		path := fmt.Sprintf("changes[%d]", index)
		if !validCanonicalID(string(change.ID)) {
			v.add("proposal.change.id_invalid", path+".id", "proposal change ID must be canonical")
		} else if _, exists := changeIDs[change.ID]; exists {
			v.add("proposal.change.id_duplicate", path+".id", "proposal change IDs must be unique")
		} else {
			changeIDs[change.ID] = struct{}{}
		}
		if strings.TrimSpace(change.Summary) == "" {
			v.add("proposal.change.summary_required", path+".summary", "proposal change requires a summary")
		}
		if len(change.IntentRefs) == 0 {
			v.add("proposal.change.untraced", path+".intent_refs", "proposal change must trace to confirmed desired outcomes or success signals")
		}
		seenRefs := make(map[IntentItemID]struct{}, len(change.IntentRefs))
		for refIndex, reference := range change.IntentRefs {
			refPath := fmt.Sprintf("%s.intent_refs[%d]", path, refIndex)
			if _, exists := seenRefs[reference]; exists {
				v.add("proposal.change.intent_ref_duplicate", refPath, "intent reference is duplicated")
				continue
			}
			seenRefs[reference] = struct{}{}
			if _, exists := traceableIntent[reference]; !exists {
				v.add("proposal.change.intent_ref_unknown", refPath, "proposal change references unknown or non-traceable intent")
			}
		}
		for conflictIndex, reference := range change.ConflictingNonGoalRefs {
			conflictPath := fmt.Sprintf("%s.conflicting_non_goal_refs[%d]", path, conflictIndex)
			if _, exists := nonGoals[reference]; !exists {
				v.add("proposal.change.non_goal_ref_unknown", conflictPath, "non-goal conflict reference is unknown")
				continue
			}
			v.add("proposal.change.non_goal_conflict", conflictPath, "proposal conflicts with confirmed non-goal")
		}
	}

	preserved := make(map[IntentItemID]struct{}, len(proposal.PreservedNonGoalRefs))
	for index, reference := range proposal.PreservedNonGoalRefs {
		path := fmt.Sprintf("preserved_non_goal_refs[%d]", index)
		if _, exists := preserved[reference]; exists {
			v.add("proposal.non_goal_ref_duplicate", path, "preserved non-goal reference is duplicated")
			continue
		}
		preserved[reference] = struct{}{}
		if _, exists := nonGoals[reference]; !exists {
			v.add("proposal.non_goal_ref_unknown", path, "preserved non-goal reference is unknown")
		}
	}
	for index, item := range intent.NonGoals {
		if _, exists := preserved[item.ID]; !exists {
			v.add(
				"proposal.non_goal_not_preserved",
				fmt.Sprintf("intent.non_goals[%d].id", index),
				"every confirmed non-goal must be explicitly preserved",
			)
		}
	}

	return v.result()
}
