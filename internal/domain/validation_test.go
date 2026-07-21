package domain

import (
	"testing"
	"time"
)

func validConfirmedIntent() IntentSnapshot {
	confirmedAt := time.Date(2026, time.July, 21, 9, 0, 0, 0, time.UTC)
	return IntentSnapshot{
		ID:      "INTENT-1",
		Version: 1,
		Status:  IntentConfirmed,
		SourceEvidence: []SourceEvidence{
			{
				ID:        "SE-1",
				Kind:      EvidenceOwnerStatement,
				Statement: "Ship the owner-confirmed result.",
			},
		},
		DesiredOutcomes: []IntentItem{
			{ID: "OUT-1", Statement: "Deliver the intended outcome.", EvidenceRefs: []SourceEvidenceID{"SE-1"}},
		},
		NonGoals: []IntentItem{
			{ID: "NG-1", Statement: "Do not deploy.", EvidenceRefs: []SourceEvidenceID{"SE-1"}},
		},
		SuccessSignals: []IntentItem{
			{ID: "SIG-1", Statement: "The owner confirms a match.", EvidenceRefs: []SourceEvidenceID{"SE-1"}},
		},
		Confirmation: &IntentConfirmation{
			Owner:              "owner",
			ConfirmedAt:        confirmedAt,
			VerificationAction: "reviewed outcomes, non-goals, and success signals",
		},
	}
}

func requireViolation(t *testing.T, err error, code string) {
	t.Helper()
	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
	for _, violation := range validationErr.Violations {
		if violation.Code == code {
			return
		}
	}
	t.Fatalf("missing violation %q in %#v", code, validationErr.Violations)
}

func TestValidateIntentSnapshotAcceptsConfirmedIntent(t *testing.T) {
	if err := ValidateIntentSnapshot(validConfirmedIntent()); err != nil {
		t.Fatalf("expected valid confirmed intent, got %v", err)
	}
}

func TestValidateIntentSnapshotAllowsCandidateWithoutConfirmation(t *testing.T) {
	intent := validConfirmedIntent()
	intent.Status = IntentCandidate
	intent.Confirmation = nil
	intent.Ambiguities = []IntentAmbiguity{
		{ID: "AMB-1", Question: "Which observable result matters?", EvidenceRefs: []SourceEvidenceID{"SE-1"}},
	}

	if err := ValidateIntentSnapshot(intent); err != nil {
		t.Fatalf("expected valid candidate intent, got %v", err)
	}
}

func TestValidateIntentSnapshotRejectsCandidateConfirmation(t *testing.T) {
	intent := validConfirmedIntent()
	intent.Status = IntentCandidate

	err := ValidateIntentSnapshot(intent)
	requireViolation(t, err, "intent.candidate.confirmation_forbidden")
}

func TestValidateIntentSnapshotRejectsPassiveOrAmbiguousConfirmation(t *testing.T) {
	intent := validConfirmedIntent()
	intent.Confirmation.VerificationAction = ""
	intent.Ambiguities = []IntentAmbiguity{
		{ID: "AMB-1", Question: "Which result?", EvidenceRefs: []SourceEvidenceID{"SE-1"}},
	}

	err := ValidateIntentSnapshot(intent)
	requireViolation(t, err, "intent.confirmed.verification_required")
	requireViolation(t, err, "intent.confirmed.ambiguities_unresolved")
}

func TestValidateIntentSnapshotRejectsDuplicateOrUnknownStableIDs(t *testing.T) {
	intent := validConfirmedIntent()
	intent.SuccessSignals[0].ID = "OUT-1"
	intent.SuccessSignals[0].EvidenceRefs = []SourceEvidenceID{"SE-404"}

	err := ValidateIntentSnapshot(intent)
	requireViolation(t, err, "intent.item.id_duplicate")
	requireViolation(t, err, "intent.evidence_refs.unknown")
}

func TestValidateMaterialAmendmentRequiresNewCandidateVersion(t *testing.T) {
	previous := validConfirmedIntent()
	next := validConfirmedIntent()
	next.Version = 2
	next.PreviousVersion = 1
	next.Status = IntentCandidate
	next.Confirmation = nil
	next.DesiredOutcomes[0].Statement = "Deliver the amended intended outcome."

	if err := ValidateIntentAmendment(previous, next, AmendmentMaterial); err != nil {
		t.Fatalf("expected valid material amendment, got %v", err)
	}
}

func TestValidateMaterialAmendmentRejectsReusedConfirmation(t *testing.T) {
	previous := validConfirmedIntent()
	next := validConfirmedIntent()
	next.Version = 2
	next.PreviousVersion = 1

	err := ValidateIntentAmendment(previous, next, AmendmentMaterial)
	requireViolation(t, err, "amendment.material.candidate_required")
	requireViolation(t, err, "amendment.material.reconfirmation_required")
}

func TestValidateMaterialAmendmentRejectsNewIDForUnchangedItem(t *testing.T) {
	previous := validConfirmedIntent()
	next := validConfirmedIntent()
	next.Version = 2
	next.PreviousVersion = 1
	next.Status = IntentCandidate
	next.Confirmation = nil
	next.DesiredOutcomes[0].ID = "OUT-2"

	err := ValidateIntentAmendment(previous, next, AmendmentMaterial)
	requireViolation(t, err, "amendment.intent_item_id_changed")
}

func TestValidateWordingOnlyAmendmentPreservesVersionAndStableIDs(t *testing.T) {
	previous := validConfirmedIntent()
	next := validConfirmedIntent()
	next.DesiredOutcomes[0].Statement = "Deliver the intended outcome"

	if err := ValidateIntentAmendment(previous, next, AmendmentWordingOnly); err != nil {
		t.Fatalf("expected valid wording-only amendment, got %v", err)
	}

	next.DesiredOutcomes[0].ID = "OUT-2"
	err := ValidateIntentAmendment(previous, next, AmendmentWordingOnly)
	requireViolation(t, err, "amendment.wording.intent_ids_changed")
}

func TestValidateWordingOnlyAmendmentAllowsReorderingStableItems(t *testing.T) {
	previous := validConfirmedIntent()
	previous.DesiredOutcomes = append(previous.DesiredOutcomes,
		IntentItem{ID: "OUT-2", Statement: "Keep the result reviewable.", EvidenceRefs: []SourceEvidenceID{"SE-1"}},
	)
	next := validConfirmedIntent()
	next.DesiredOutcomes = []IntentItem{
		previous.DesiredOutcomes[1],
		previous.DesiredOutcomes[0],
	}

	if err := ValidateIntentAmendment(previous, next, AmendmentWordingOnly); err != nil {
		t.Fatalf("stable IDs must survive harmless reordering: %v", err)
	}
}

func TestValidateProposalCoverageRequiresConfirmedIntent(t *testing.T) {
	intent := validConfirmedIntent()
	intent.Status = IntentCandidate
	intent.Confirmation = nil

	err := ValidateProposalCoverage(intent, validProposal())
	requireViolation(t, err, "proposal.intent_not_confirmed")
}

func TestValidateProposalCoverageAcceptsTracedChangesAndPreservedNonGoals(t *testing.T) {
	if err := ValidateProposalCoverage(validConfirmedIntent(), validProposal()); err != nil {
		t.Fatalf("expected valid proposal coverage, got %v", err)
	}
}

func TestValidateProposalCoverageRejectsInventedIntentAndNonGoalConflict(t *testing.T) {
	proposal := validProposal()
	proposal.Changes = append(proposal.Changes,
		ProposalChange{ID: "CHANGE-2", Summary: "Untraced change"},
		ProposalChange{
			ID:                     "CHANGE-3",
			Summary:                "Invented change",
			IntentRefs:             []IntentItemID{"OUT-404"},
			ConflictingNonGoalRefs: []IntentItemID{"NG-1"},
		},
	)
	proposal.PreservedNonGoalRefs = nil

	err := ValidateProposalCoverage(validConfirmedIntent(), proposal)
	requireViolation(t, err, "proposal.change.untraced")
	requireViolation(t, err, "proposal.change.intent_ref_unknown")
	requireViolation(t, err, "proposal.change.non_goal_conflict")
	requireViolation(t, err, "proposal.non_goal_not_preserved")
}

func TestConfirmedIntentNeverGrantsEffectAuthority(t *testing.T) {
	intent := validConfirmedIntent()
	if intent.GrantsEffectAuthority() {
		t.Fatal("confirmed intent must never grant effect authority")
	}
}

func validProposal() Proposal {
	return Proposal{
		Changes: []ProposalChange{
			{
				ID:         "CHANGE-1",
				Summary:    "Deliver the confirmed outcome.",
				IntentRefs: []IntentItemID{"OUT-1", "SIG-1"},
			},
		},
		PreservedNonGoalRefs: []IntentItemID{"NG-1"},
	}
}
