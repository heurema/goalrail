package domain

import (
	"fmt"
	"time"
)

const (
	IntentCanaryV0ManifestID       = CanaryID("intent-canary-v0")
	IntentCanaryV0ManifestVersion  = uint32(1)
	IntentCanaryV0ManifestVersion2 = uint32(2)

	CanaryEligibilityRuleV1      = EvidenceReference("canary-rule:eligibility-v1")
	CanaryCheckRecordingRuleV1   = EvidenceReference("canary-rule:check-recording-v1")
	CanaryAssessmentTimingRuleV1 = EvidenceReference("canary-rule:assessment-timing-v1")
	CanaryMissingDataRuleV1      = EvidenceReference("canary-rule:missing-data-v1")
	CanaryOverheadRuleV1         = EvidenceReference("canary-rule:overhead-v1")
	CanaryRateFormulaRuleV1      = EvidenceReference("canary-rule:rates-v1")

	CanaryAssessmentTimingRuleV2 = EvidenceReference("canary-rule:assessment-timing-v2")
	CanaryMissingDataRuleV2      = EvidenceReference("canary-rule:missing-data-v2")
	CanaryOverheadRuleV2         = EvidenceReference("canary-rule:overhead-v2")
	CanaryContextRuleV2          = EvidenceReference("canary-rule:context-v2")
	CanaryTelemetryRuleV2        = EvidenceReference("canary-rule:telemetry-v2")
	CanaryAssessmentBasisRuleV2  = EvidenceReference("canary-rule:assessment-basis-v2")
)

var IntentCanaryV0FrozenAt = time.Date(2026, time.July, 21, 13, 17, 33, 0, time.UTC)
var IntentCanaryV0FrozenAtV2 = time.Date(2026, time.July, 22, 15, 10, 0, 0, time.UTC)

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

// NewIntentCanaryV0ManifestV2 returns the frozen context-and-evaluation
// rehearsal contract. Like v1, it is not an activation grant.
func NewIntentCanaryV0ManifestV2() (CanaryManifest, error) {
	manifest := CanaryManifest{
		ID:                      IntentCanaryV0ManifestID,
		Version:                 IntentCanaryV0ManifestVersion2,
		FrozenAt:                IntentCanaryV0FrozenAtV2,
		AssignmentCount:         CanaryAssignmentCount,
		VariantRotation:         intentCanaryV0Rotation(),
		EligibilityRuleRef:      CanaryEligibilityRuleV1,
		CheckRecordingRuleRef:   CanaryCheckRecordingRuleV1,
		AssessmentTimingRuleRef: CanaryAssessmentTimingRuleV2,
		MissingDataRuleRef:      CanaryMissingDataRuleV2,
		OverheadRuleRef:         CanaryOverheadRuleV2,
		RateFormulaRuleRef:      CanaryRateFormulaRuleV1,
		ContextRuleRef:          CanaryContextRuleV2,
		TelemetryRuleRef:        CanaryTelemetryRuleV2,
		AssessmentBasisRuleRef:  CanaryAssessmentBasisRuleV2,
	}
	if err := ValidateCanaryManifest(manifest); err != nil {
		return CanaryManifest{}, err
	}
	return manifest, nil
}

// NewIntentCanaryV0ManifestForVersion selects one immutable known manifest.
func NewIntentCanaryV0ManifestForVersion(version uint32) (CanaryManifest, error) {
	switch version {
	case IntentCanaryV0ManifestVersion:
		return NewIntentCanaryV0Manifest()
	case IntentCanaryV0ManifestVersion2:
		return NewIntentCanaryV0ManifestV2()
	default:
		return CanaryManifest{}, fmt.Errorf("unsupported intent canary manifest version %d", version)
	}
}

// ValidateCanaryManifest rejects incomplete or changed rules for either known
// immutable manifest version.
func ValidateCanaryManifest(manifest CanaryManifest) error {
	v := &validator{}
	if manifest.ID != IntentCanaryV0ManifestID {
		v.add("canary.manifest.id_changed", "id", "manifest ID is immutable")
	}
	expected, known := expectedManifestRules(manifest.Version)
	if !known {
		v.add("canary.manifest.version_unknown", "version", "manifest version must be 1 or 2")
	}
	if manifest.FrozenAt.IsZero() || manifest.FrozenAt.Location() != time.UTC {
		v.add("canary.manifest.frozen_at_invalid", "frozen_at", "manifest requires a non-zero UTC freeze timestamp")
	} else if known && !manifest.FrozenAt.Equal(expected.frozenAt) {
		v.add("canary.manifest.frozen_at_changed", "frozen_at", "manifest freeze timestamp is immutable")
	}
	if manifest.AssignmentCount != CanaryAssignmentCount {
		v.add("canary.manifest.assignment_count_changed", "assignment_count", "manifest requires exactly 15 assignments")
	}

	expectedRotation := intentCanaryV0Rotation()
	if len(manifest.VariantRotation) != len(expectedRotation) {
		v.add("canary.manifest.rotation_length_changed", "variant_rotation", "manifest requires the complete 15-position rotation")
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

	if known {
		validateFrozenRuleRef(v, "eligibility_rule_ref", manifest.EligibilityRuleRef, expected.eligibility)
		validateFrozenRuleRef(v, "check_recording_rule_ref", manifest.CheckRecordingRuleRef, expected.checkRecording)
		validateFrozenRuleRef(v, "assessment_timing_rule_ref", manifest.AssessmentTimingRuleRef, expected.assessmentTiming)
		validateFrozenRuleRef(v, "missing_data_rule_ref", manifest.MissingDataRuleRef, expected.missingData)
		validateFrozenRuleRef(v, "overhead_rule_ref", manifest.OverheadRuleRef, expected.overhead)
		validateFrozenRuleRef(v, "rate_formula_rule_ref", manifest.RateFormulaRuleRef, expected.rateFormula)
		validateOptionalFrozenRuleRef(v, "context_rule_ref", manifest.ContextRuleRef, expected.context)
		validateOptionalFrozenRuleRef(v, "telemetry_rule_ref", manifest.TelemetryRuleRef, expected.telemetry)
		validateOptionalFrozenRuleRef(v, "assessment_basis_rule_ref", manifest.AssessmentBasisRuleRef, expected.assessmentBasis)
	}
	return v.result()
}

type manifestRules struct {
	frozenAt         time.Time
	eligibility      EvidenceReference
	checkRecording   EvidenceReference
	assessmentTiming EvidenceReference
	missingData      EvidenceReference
	overhead         EvidenceReference
	rateFormula      EvidenceReference
	context          EvidenceReference
	telemetry        EvidenceReference
	assessmentBasis  EvidenceReference
}

func expectedManifestRules(version uint32) (manifestRules, bool) {
	switch version {
	case IntentCanaryV0ManifestVersion:
		return manifestRules{
			frozenAt: IntentCanaryV0FrozenAt, eligibility: CanaryEligibilityRuleV1,
			checkRecording: CanaryCheckRecordingRuleV1, assessmentTiming: CanaryAssessmentTimingRuleV1,
			missingData: CanaryMissingDataRuleV1, overhead: CanaryOverheadRuleV1,
			rateFormula: CanaryRateFormulaRuleV1,
		}, true
	case IntentCanaryV0ManifestVersion2:
		return manifestRules{
			frozenAt: IntentCanaryV0FrozenAtV2, eligibility: CanaryEligibilityRuleV1,
			checkRecording: CanaryCheckRecordingRuleV1, assessmentTiming: CanaryAssessmentTimingRuleV2,
			missingData: CanaryMissingDataRuleV2, overhead: CanaryOverheadRuleV2,
			rateFormula: CanaryRateFormulaRuleV1, context: CanaryContextRuleV2,
			telemetry: CanaryTelemetryRuleV2, assessmentBasis: CanaryAssessmentBasisRuleV2,
		}, true
	default:
		return manifestRules{}, false
	}
}

func validateFrozenRuleRef(v *validator, path string, actual, expected EvidenceReference) {
	if !IsEvidenceReference(string(actual)) {
		v.add("canary.manifest.rule_ref_invalid", path, "rule reference must be a bounded scheme:identifier")
		return
	}
	if actual != expected {
		v.add("canary.manifest.rule_ref_changed", path, "manifest rule reference is immutable")
	}
}

func validateOptionalFrozenRuleRef(v *validator, path string, actual, expected EvidenceReference) {
	if expected == "" {
		if actual != "" {
			v.add("canary.manifest.rule_ref_changed", path, "manifest version does not define this rule")
		}
		return
	}
	validateFrozenRuleRef(v, path, actual, expected)
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
