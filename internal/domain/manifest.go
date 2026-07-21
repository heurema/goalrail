package domain

import (
	"fmt"
	"time"
)

const (
	IntentCanaryV0ManifestID      = CanaryID("intent-canary-v0")
	IntentCanaryV0ManifestVersion = uint32(1)

	CanaryEligibilityRuleV1      = EvidenceReference("canary-rule:eligibility-v1")
	CanaryCheckRecordingRuleV1   = EvidenceReference("canary-rule:check-recording-v1")
	CanaryAssessmentTimingRuleV1 = EvidenceReference("canary-rule:assessment-timing-v1")
	CanaryMissingDataRuleV1      = EvidenceReference("canary-rule:missing-data-v1")
	CanaryOverheadRuleV1         = EvidenceReference("canary-rule:overhead-v1")
	CanaryRateFormulaRuleV1      = EvidenceReference("canary-rule:rates-v1")
)

var IntentCanaryV0FrozenAt = time.Date(2026, time.July, 21, 13, 17, 33, 0, time.UTC)

// NewIntentCanaryV0Manifest returns the exact v1 manifest frozen for this
// canary. It does not activate the real-change canary or grant effect authority.
func NewIntentCanaryV0Manifest() (CanaryManifest, error) {
	manifest := CanaryManifest{
		ID:                      IntentCanaryV0ManifestID,
		Version:                 IntentCanaryV0ManifestVersion,
		FrozenAt:                IntentCanaryV0FrozenAt,
		AssignmentCount:         CanaryAssignmentCount,
		VariantRotation:         intentCanaryV0Rotation(),
		EligibilityRuleRef:      CanaryEligibilityRuleV1,
		CheckRecordingRuleRef:   CanaryCheckRecordingRuleV1,
		AssessmentTimingRuleRef: CanaryAssessmentTimingRuleV1,
		MissingDataRuleRef:      CanaryMissingDataRuleV1,
		OverheadRuleRef:         CanaryOverheadRuleV1,
		RateFormulaRuleRef:      CanaryRateFormulaRuleV1,
	}
	if err := ValidateCanaryManifest(manifest); err != nil {
		return CanaryManifest{}, err
	}
	return manifest, nil
}

// ValidateCanaryManifest rejects incomplete or changed v1 measurement rules.
// A material rule change must use a new manifest version instead of mutating v1.
func ValidateCanaryManifest(manifest CanaryManifest) error {
	v := &validator{}
	if manifest.ID != IntentCanaryV0ManifestID {
		v.add("canary.manifest.id_changed", "id", "v1 manifest ID is immutable")
	}
	if manifest.Version != IntentCanaryV0ManifestVersion {
		v.add("canary.manifest.version_changed", "version", "v1 manifest version is immutable")
	}
	if manifest.FrozenAt.IsZero() || manifest.FrozenAt.Location() != time.UTC {
		v.add("canary.manifest.frozen_at_invalid", "frozen_at", "manifest requires a non-zero UTC freeze timestamp")
	}
	if manifest.AssignmentCount != CanaryAssignmentCount {
		v.add("canary.manifest.assignment_count_changed", "assignment_count", "v1 requires exactly 15 assignments")
	}

	expectedRotation := intentCanaryV0Rotation()
	if len(manifest.VariantRotation) != len(expectedRotation) {
		v.add("canary.manifest.rotation_length_changed", "variant_rotation", "v1 requires the complete 15-position rotation")
	} else {
		for index, expected := range expectedRotation {
			if manifest.VariantRotation[index] != expected {
				v.add(
					"canary.manifest.rotation_changed",
					fmt.Sprintf("variant_rotation[%d]", index),
					"variant must match the frozen flow, flow, baseline rotation",
				)
			}
		}
	}

	validateFrozenRuleRef(v, "eligibility_rule_ref", manifest.EligibilityRuleRef, CanaryEligibilityRuleV1)
	validateFrozenRuleRef(v, "check_recording_rule_ref", manifest.CheckRecordingRuleRef, CanaryCheckRecordingRuleV1)
	validateFrozenRuleRef(v, "assessment_timing_rule_ref", manifest.AssessmentTimingRuleRef, CanaryAssessmentTimingRuleV1)
	validateFrozenRuleRef(v, "missing_data_rule_ref", manifest.MissingDataRuleRef, CanaryMissingDataRuleV1)
	validateFrozenRuleRef(v, "overhead_rule_ref", manifest.OverheadRuleRef, CanaryOverheadRuleV1)
	validateFrozenRuleRef(v, "rate_formula_rule_ref", manifest.RateFormulaRuleRef, CanaryRateFormulaRuleV1)
	return v.result()
}

func validateFrozenRuleRef(v *validator, path string, actual, expected EvidenceReference) {
	if !IsEvidenceReference(string(actual)) {
		v.add("canary.manifest.rule_ref_invalid", path, "rule reference must be a bounded scheme:identifier")
		return
	}
	if actual != expected {
		v.add("canary.manifest.rule_ref_changed", path, "v1 rule reference is immutable")
	}
}

func intentCanaryV0Rotation() []CanaryVariant {
	return []CanaryVariant{
		VariantFlow, VariantFlow, VariantBaseline,
		VariantFlow, VariantFlow, VariantBaseline,
		VariantFlow, VariantFlow, VariantBaseline,
		VariantFlow, VariantFlow, VariantBaseline,
		VariantFlow, VariantFlow, VariantBaseline,
	}
}

// CanaryTimingInterval is one source-addressed elapsed interval. Agent turns
// use one interval per distinct root Codex turn trace; nested observations are
// deliberately not summed.
type CanaryTimingInterval struct {
	Reference EvidenceReference `json:"reference"`
	StartedAt time.Time         `json:"started_at"`
	EndedAt   time.Time         `json:"ended_at"`
}

type CanaryFlowOverheadInput struct {
	AgentTurns          []CanaryTimingInterval `json:"agent_turns"`
	OwnerReview         *CanaryTimingInterval  `json:"owner_review,omitempty"`
	OwnerReviewRequired bool                   `json:"owner_review_required"`
}

// CanaryFlowOverhead separates components while exposing one exact total used
// by the report. Partial components may be present when Available is false, but
// TotalMinutes is never imputed from incomplete evidence.
type CanaryFlowOverhead struct {
	AgentTurnCount     uint32  `json:"agent_turn_count"`
	AgentSeconds       float64 `json:"agent_seconds"`
	OwnerReviewSeconds float64 `json:"owner_review_seconds"`
	TotalMinutes       float64 `json:"total_minutes"`
	Available          bool    `json:"available"`
}

// CalculateCanaryFlowOverhead implements canary-rule:overhead-v1:
// sum distinct flow-only root turn envelopes, add the explicit owner-review
// interval, and divide total seconds by 60 without rounding. Missing required
// evidence returns Available=false; malformed or duplicate evidence is invalid.
func CalculateCanaryFlowOverhead(input CanaryFlowOverheadInput) (CanaryFlowOverhead, error) {
	v := &validator{}
	measurement := CanaryFlowOverhead{AgentTurnCount: uint32(len(input.AgentTurns))}
	seen := make(map[EvidenceReference]struct{}, len(input.AgentTurns))
	for index, interval := range input.AgentTurns {
		path := fmt.Sprintf("agent_turns[%d]", index)
		validateTimingInterval(v, path, interval)
		if _, duplicate := seen[interval.Reference]; duplicate {
			v.add("canary.overhead.trace_duplicate", path+".reference", "root turn trace must be counted once")
		} else {
			seen[interval.Reference] = struct{}{}
		}
		if intervalIsValid(interval) {
			measurement.AgentSeconds += interval.EndedAt.Sub(interval.StartedAt).Seconds()
		}
	}

	if input.OwnerReview != nil {
		validateTimingInterval(v, "owner_review", *input.OwnerReview)
		if intervalIsValid(*input.OwnerReview) {
			measurement.OwnerReviewSeconds = input.OwnerReview.EndedAt.Sub(input.OwnerReview.StartedAt).Seconds()
		}
	}
	if err := v.result(); err != nil {
		return CanaryFlowOverhead{}, err
	}
	if len(input.AgentTurns) == 0 || (input.OwnerReviewRequired && input.OwnerReview == nil) {
		return measurement, nil
	}
	measurement.Available = true
	measurement.TotalMinutes = (measurement.AgentSeconds + measurement.OwnerReviewSeconds) / 60
	return measurement, nil
}

func validateTimingInterval(v *validator, path string, interval CanaryTimingInterval) {
	if !IsEvidenceReference(string(interval.Reference)) {
		v.add("canary.overhead.reference_invalid", path+".reference", "timing interval requires a bounded source reference")
	}
	if interval.StartedAt.IsZero() || interval.StartedAt.Location() != time.UTC {
		v.add("canary.overhead.start_invalid", path+".started_at", "start must be a non-zero UTC timestamp")
	}
	if interval.EndedAt.IsZero() || interval.EndedAt.Location() != time.UTC {
		v.add("canary.overhead.end_invalid", path+".ended_at", "end must be a non-zero UTC timestamp")
	}
	if !interval.StartedAt.IsZero() && !interval.EndedAt.IsZero() && interval.EndedAt.Before(interval.StartedAt) {
		v.add("canary.overhead.interval_reversed", path, "end cannot precede start")
	}
}

func intervalIsValid(interval CanaryTimingInterval) bool {
	return IsEvidenceReference(string(interval.Reference)) &&
		!interval.StartedAt.IsZero() && interval.StartedAt.Location() == time.UTC &&
		!interval.EndedAt.IsZero() && interval.EndedAt.Location() == time.UTC &&
		!interval.EndedAt.Before(interval.StartedAt)
}
