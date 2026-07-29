package domain

import (
	"strings"
	"testing"
	"time"
)

func TestIntentWithoutAnEscalationRecordStaysValid(t *testing.T) {
	snapshot := answeringIntentFixture()
	snapshot.ResolvedEscalation = nil
	if err := ValidateIntentSnapshot(snapshot); err != nil {
		t.Fatalf("a version that answers nothing was rejected: %v", err)
	}
}

func TestIntentEscalationRecordAcceptsEachDisposition(t *testing.T) {
	for _, disposition := range []IntentDisposition{
		DispositionAnswered,
		DispositionSpurious,
		DispositionWithdrawn,
	} {
		t.Run(string(disposition), func(t *testing.T) {
			snapshot := answeringIntentFixture()
			snapshot.ResolvedEscalation.Disposition = disposition
			if err := ValidateIntentSnapshot(snapshot); err != nil {
				t.Fatalf("disposition %q was rejected: %v", disposition, err)
			}
		})
	}
}

func TestIntentEscalationRecordRejectsMalformedValues(t *testing.T) {
	for name, mutate := range map[string]func(*IntentEscalationResolution){
		"run id not canonical": func(resolution *IntentEscalationResolution) {
			resolution.ResolvedID = "Run One"
		},
		"empty run id": func(resolution *IntentEscalationResolution) {
			resolution.ResolvedID = ""
		},
		"digest not sha-256": func(resolution *IntentEscalationResolution) {
			resolution.EscalationDigest = "md5:" + strings.Repeat("a", 32)
		},
		"digest truncated": func(resolution *IntentEscalationResolution) {
			resolution.EscalationDigest = "sha256:" + strings.Repeat("a", 63)
		},
		"digest not hex": func(resolution *IntentEscalationResolution) {
			resolution.EscalationDigest = "sha256:" + strings.Repeat("z", 64)
		},
		"unknown disposition": func(resolution *IntentEscalationResolution) {
			resolution.Disposition = "acknowledged"
		},
		"empty disposition": func(resolution *IntentEscalationResolution) {
			resolution.Disposition = ""
		},
	} {
		t.Run(name, func(t *testing.T) {
			snapshot := answeringIntentFixture()
			mutate(snapshot.ResolvedEscalation)
			if err := ValidateIntentSnapshot(snapshot); err == nil {
				t.Fatal("a malformed escalation record was silently accepted")
			}
		})
	}
}

func TestWordingOnlyAmendmentCannotAcquireAnEscalationRecord(t *testing.T) {
	// Claiming to answer a blocked run is a material act. Allowing it through a
	// wording-only edit would let an already confirmed version acquire a
	// disposition without returning to candidate and being confirmed again.
	previous := answeringIntentFixture()
	previous.ResolvedEscalation = nil

	for name, mutate := range map[string]func(*IntentSnapshot){
		"record added": func(next *IntentSnapshot) {
			next.ResolvedEscalation = &IntentEscalationResolution{
				ResolvedID:       "run-blocked-one",
				EscalationDigest: "sha256:" + strings.Repeat("c", 64),
				Disposition:      DispositionAnswered,
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			next := previous
			mutate(&next)
			if err := ValidateIntentAmendment(previous, next, AmendmentWordingOnly); err == nil {
				t.Fatal("a wording-only edit acquired an escalation record")
			}
		})
	}

	withRecord := answeringIntentFixture()
	changed := withRecord
	changed.ResolvedEscalation = &IntentEscalationResolution{
		ResolvedID:       withRecord.ResolvedEscalation.ResolvedID,
		EscalationDigest: withRecord.ResolvedEscalation.EscalationDigest,
		Disposition:      DispositionSpurious,
	}
	if err := ValidateIntentAmendment(withRecord, changed, AmendmentWordingOnly); err == nil {
		t.Fatal("a wording-only edit changed a recorded disposition")
	}

	unchanged := withRecord
	if err := ValidateIntentAmendment(withRecord, unchanged, AmendmentWordingOnly); err != nil {
		t.Fatalf("a wording-only edit preserving the record was rejected: %v", err)
	}
}

func TestAnsweringIntentGrantsNoEffectAuthority(t *testing.T) {
	snapshot := answeringIntentFixture()
	if snapshot.GrantsEffectAuthority() {
		t.Fatal("recording a resolved escalation granted effect authority")
	}
}

func answeringIntentFixture() IntentSnapshot {
	confirmedAt := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	return IntentSnapshot{
		ID:      "intent-answering",
		Version: 1,
		Status:  IntentConfirmed,
		SourceEvidence: []SourceEvidence{{
			ID:        "se-1",
			Kind:      EvidenceOwnerStatement,
			Statement: "Answer the blocked run.",
			Reference: "owner:blocked-answer",
		}},
		DesiredOutcomes: []IntentItem{{
			ID:           "out-1",
			Statement:    "Record which reading of the requirement governs.",
			EvidenceRefs: []SourceEvidenceID{"se-1"},
		}},
		NonGoals: []IntentItem{{
			ID:           "ng-1",
			Statement:    "Do not resume the blocked run.",
			EvidenceRefs: []SourceEvidenceID{"se-1"},
		}},
		SuccessSignals: []IntentItem{{
			ID:           "sig-1",
			Statement:    "The chain resolves from the receipt to this version.",
			EvidenceRefs: []SourceEvidenceID{"se-1"},
		}},
		Confirmation: &IntentConfirmation{
			Owner:              "repository owner",
			ConfirmedAt:        confirmedAt,
			VerificationAction: "Owner reviewed and confirmed the answering version.",
		},
		ResolvedEscalation: &IntentEscalationResolution{
			ResolvedID:       "run-blocked-one",
			EscalationDigest: "sha256:" + strings.Repeat("c", 64),
			Disposition:      DispositionAnswered,
		},
	}
}
