package domain

import (
	"fmt"
	"math"
	"sort"
)

const (
	CanaryAssignmentCount              = uint32(15)
	CanaryMinimumVerifiedLineage       = uint32(13)
	CanaryMaximumMedianOverheadMinutes = float64(30)
	CanaryUnresolvedLinkStopCount      = uint32(2)
	CanaryProcessAbandonmentStopCount  = uint32(3)
)

type CanaryTerminalState string

const (
	CanaryStatePending   CanaryTerminalState = "pending"
	CanaryStateDelivered CanaryTerminalState = "delivered"
	CanaryStateAbandoned CanaryTerminalState = "abandoned"
)

// CanaryLineageOutcome is the canary's assessment of a recorded join. A wrong
// join is distinct from an explicitly unresolved join because either one wrong
// association is a hard stop.
type CanaryLineageOutcome string

const (
	CanaryLineagePending                   CanaryLineageOutcome = "pending"
	CanaryLineageVerified                  CanaryLineageOutcome = "verified"
	CanaryLineageUnresolvedAfterResolution CanaryLineageOutcome = "unresolved_after_resolution"
	CanaryLineageWrong                     CanaryLineageOutcome = "wrong"
)

type CanaryVerdict string

const (
	CanaryVerdictPending CanaryVerdict = "PENDING"
	CanaryVerdictPass    CanaryVerdict = "PASS"
	CanaryVerdictStop    CanaryVerdict = "STOP"
	CanaryVerdictReshape CanaryVerdict = "RESHAPE"
)

// CanaryObservation is one stable assigned change reduced to the facts needed
// for deterministic reporting. Adapters may derive it from append-only events,
// but provider-specific trace or workflow types never enter this boundary.
type CanaryObservation struct {
	Ordinal                           uint32
	ChangeID                          ChangeID
	Variant                           CanaryVariant
	TerminalState                     CanaryTerminalState
	LineageOutcome                    CanaryLineageOutcome
	Assessment                        *Assessment
	MaterialMisunderstandingPrevented bool
	FlowOverheadMinutes               *float64
	ProcessCausedAbandonment          bool
}

type CanaryReportInput struct {
	Observations                []CanaryObservation
	EvidenceIntegrityViolations uint32
	AssignmentsStopped          bool
}

// CanaryRate preserves numerator and denominator so unequal group sizes remain
// visible. Available is false when the denominator is zero.
type CanaryRate struct {
	Numerator   uint32  `json:"numerator"`
	Denominator uint32  `json:"denominator"`
	Value       float64 `json:"value"`
	Available   bool    `json:"available"`
}

type CanaryMeasurement struct {
	Count     uint32  `json:"count"`
	Value     float64 `json:"value"`
	Available bool    `json:"available"`
}

type CanaryVariantReport struct {
	Assigned                    uint32     `json:"assigned"`
	Pending                     uint32     `json:"pending"`
	Delivered                   uint32     `json:"delivered"`
	Assessed                    uint32     `json:"assessed"`
	MissingAssessments          uint32     `json:"missing_assessments"`
	Abandoned                   uint32     `json:"abandoned"`
	NonMatches                  uint32     `json:"non_matches"`
	WrongButGreen               uint32     `json:"wrong_but_green"`
	MaterialPreventions         uint32     `json:"material_preventions"`
	RepeatOptInYes              uint32     `json:"repeat_opt_in_yes"`
	RepeatOptInNo               uint32     `json:"repeat_opt_in_no"`
	MissingRepeatOptIn          uint32     `json:"missing_repeat_opt_in"`
	OverheadMeasurements        uint32     `json:"overhead_measurements"`
	MissingOverheadMeasurements uint32     `json:"missing_overhead_measurements"`
	NonMatchRate                CanaryRate `json:"non_match_rate"`
}

type CanaryPassSignals struct {
	LineageReliable     bool `json:"lineage_reliable"`
	NonMatchRateLower   bool `json:"non_match_rate_lower"`
	PreventionObserved  bool `json:"prevention_observed"`
	OverheadTolerable   bool `json:"overhead_tolerable"`
	RepeatOptInAccepted bool `json:"repeat_opt_in_accepted"`
}

type CanaryHardStopSignals struct {
	WrongJoin                  bool `json:"wrong_join"`
	UnresolvedLinks            bool `json:"unresolved_links"`
	EvidenceIntegrityViolation bool `json:"evidence_integrity_violation"`
	ExcessiveOverhead          bool `json:"excessive_overhead"`
	ProcessCausedAbandonments  bool `json:"process_caused_abandonments"`
}

type CanaryReport struct {
	Verdict                       CanaryVerdict         `json:"verdict"`
	AssignmentsStopped            bool                  `json:"assignments_stopped,omitempty"`
	CompletionReady               bool                  `json:"completion_ready"`
	Assigned                      uint32                `json:"assigned"`
	Terminal                      uint32                `json:"terminal"`
	Flow                          CanaryVariantReport   `json:"flow"`
	Baseline                      CanaryVariantReport   `json:"baseline"`
	LineageVerified               uint32                `json:"lineage_verified"`
	LineagePending                uint32                `json:"lineage_pending"`
	LineageUnresolved             uint32                `json:"lineage_unresolved"`
	WrongJoins                    uint32                `json:"wrong_joins"`
	ProcessCausedFlowAbandonments uint32                `json:"process_caused_flow_abandonments"`
	EvidenceIntegrityViolations   uint32                `json:"evidence_integrity_violations"`
	MedianFlowOverhead            CanaryMeasurement     `json:"median_flow_overhead"`
	PassSignals                   CanaryPassSignals     `json:"pass_signals"`
	HardStopSignals               CanaryHardStopSignals `json:"hard_stop_signals"`
	NoUsefulMovement              bool                  `json:"no_useful_movement"`
}

// CanaryVariantForOrdinal deterministically assigns flow, flow, baseline for
// ordinals 1 through 15. Outcomes and difficulty are deliberately absent.
func CanaryVariantForOrdinal(ordinal uint32) (CanaryVariant, error) {
	if ordinal == 0 || ordinal > CanaryAssignmentCount {
		return "", fmt.Errorf("canary ordinal %d is outside 1..%d", ordinal, CanaryAssignmentCount)
	}
	switch (ordinal - 1) % 3 {
	case 0, 1:
		return VariantFlow, nil
	default:
		return VariantBaseline, nil
	}
}

// CalculateCanaryReport validates immutable assignment facts, calculates all
// rates from their actual denominators, and applies hard-stop rules before
// completion or pass evaluation.
func CalculateCanaryReport(input CanaryReportInput) (CanaryReport, error) {
	observations, err := normalizeCanaryObservations(input.Observations)
	if err != nil {
		return CanaryReport{}, err
	}

	report := CanaryReport{
		Verdict:                     CanaryVerdictPending,
		AssignmentsStopped:          input.AssignmentsStopped,
		Assigned:                    uint32(len(observations)),
		EvidenceIntegrityViolations: input.EvidenceIntegrityViolations,
	}
	flowOverheads := make([]float64, 0, 10)

	for _, observation := range observations {
		variantReport := &report.Flow
		if observation.Variant == VariantBaseline {
			variantReport = &report.Baseline
		}
		variantReport.Assigned++

		switch observation.LineageOutcome {
		case CanaryLineagePending:
			report.LineagePending++
		case CanaryLineageVerified:
			report.LineageVerified++
		case CanaryLineageUnresolvedAfterResolution:
			report.LineageUnresolved++
		case CanaryLineageWrong:
			report.WrongJoins++
		}

		switch observation.TerminalState {
		case CanaryStatePending:
			variantReport.Pending++
		case CanaryStateDelivered:
			report.Terminal++
			variantReport.Delivered++
			if observation.Assessment == nil {
				variantReport.MissingAssessments++
			} else {
				addAssessment(variantReport, *observation.Assessment)
			}
		case CanaryStateAbandoned:
			report.Terminal++
			variantReport.Abandoned++
		}

		if observation.MaterialMisunderstandingPrevented {
			variantReport.MaterialPreventions++
		}
		if observation.ProcessCausedAbandonment {
			report.ProcessCausedFlowAbandonments++
		}
		if observation.FlowOverheadMinutes != nil {
			variantReport.OverheadMeasurements++
			flowOverheads = append(flowOverheads, *observation.FlowOverheadMinutes)
		}
	}

	report.Flow.MissingRepeatOptIn = report.Flow.Assigned - report.Flow.RepeatOptInYes - report.Flow.RepeatOptInNo
	report.Baseline.MissingRepeatOptIn = report.Baseline.Assigned - report.Baseline.RepeatOptInYes - report.Baseline.RepeatOptInNo
	report.Flow.MissingOverheadMeasurements = report.Flow.Assigned - report.Flow.OverheadMeasurements
	report.Flow.NonMatchRate = makeRate(report.Flow.NonMatches, report.Flow.Assessed)
	report.Baseline.NonMatchRate = makeRate(report.Baseline.NonMatches, report.Baseline.Assessed)
	report.MedianFlowOverhead = medianMeasurement(flowOverheads)

	report.HardStopSignals = CanaryHardStopSignals{
		WrongJoin:                  report.WrongJoins > 0,
		UnresolvedLinks:            report.LineageUnresolved >= CanaryUnresolvedLinkStopCount,
		EvidenceIntegrityViolation: report.EvidenceIntegrityViolations > 0,
		ExcessiveOverhead: report.MedianFlowOverhead.Available &&
			report.MedianFlowOverhead.Value > CanaryMaximumMedianOverheadMinutes,
		ProcessCausedAbandonments: report.ProcessCausedFlowAbandonments >= CanaryProcessAbandonmentStopCount,
	}
	if report.AssignmentsStopped || report.HardStopSignals.any() {
		report.Verdict = CanaryVerdictStop
		return report, nil
	}

	report.CompletionReady = report.Assigned == CanaryAssignmentCount &&
		report.Terminal == CanaryAssignmentCount &&
		report.Flow.MissingAssessments == 0 &&
		report.Baseline.MissingAssessments == 0 &&
		report.LineagePending == 0 &&
		report.Flow.MissingOverheadMeasurements == 0

	report.PassSignals = CanaryPassSignals{
		LineageReliable:    report.LineageVerified >= CanaryMinimumVerifiedLineage && report.WrongJoins == 0,
		NonMatchRateLower:  rateLower(report.Flow.NonMatchRate, report.Baseline.NonMatchRate),
		PreventionObserved: report.Flow.MaterialPreventions > 0,
		OverheadTolerable: report.MedianFlowOverhead.Available &&
			report.MedianFlowOverhead.Value <= CanaryMaximumMedianOverheadMinutes,
		RepeatOptInAccepted: report.Flow.Assigned > 0 &&
			report.Flow.RepeatOptInYes*3 >= report.Flow.Assigned*2,
	}
	report.NoUsefulMovement = report.Flow.MaterialPreventions == 0 &&
		report.Flow.NonMatchRate.Available &&
		report.Baseline.NonMatchRate.Available &&
		!report.PassSignals.NonMatchRateLower &&
		report.MedianFlowOverhead.Available &&
		report.MedianFlowOverhead.Value > 0

	if !report.CompletionReady {
		return report, nil
	}
	if report.PassSignals.all() {
		report.Verdict = CanaryVerdictPass
	} else {
		report.Verdict = CanaryVerdictReshape
	}
	return report, nil
}

func normalizeCanaryObservations(observations []CanaryObservation) ([]CanaryObservation, error) {
	normalized := append([]CanaryObservation(nil), observations...)
	sort.Slice(normalized, func(i, j int) bool {
		return normalized[i].Ordinal < normalized[j].Ordinal
	})
	v := &validator{}
	if len(normalized) > int(CanaryAssignmentCount) {
		v.add("canary.assignments.too_many", "observations", "canary accepts at most 15 assignments")
	}
	changeIDs := make(map[ChangeID]struct{}, len(normalized))
	for index, observation := range normalized {
		path := fmt.Sprintf("observations[%d]", index)
		expectedOrdinal := uint32(index + 1)
		if observation.Ordinal != expectedOrdinal {
			v.add("canary.ordinal.non_sequential", path+".ordinal", "ordinals must form one immutable sequence starting at 1")
		}
		if !IsCanonicalID(string(observation.ChangeID)) {
			v.add("canary.change_id.invalid", path+".change_id", "change ID must be canonical")
		} else if _, exists := changeIDs[observation.ChangeID]; exists {
			v.add("canary.change_id.duplicate", path+".change_id", "change ID must be unique")
		}
		changeIDs[observation.ChangeID] = struct{}{}

		expectedVariant, variantErr := CanaryVariantForOrdinal(observation.Ordinal)
		if variantErr != nil {
			v.add("canary.ordinal.invalid", path+".ordinal", variantErr.Error())
		} else if observation.Variant != expectedVariant {
			v.add("canary.variant.mismatch", path+".variant", "variant does not match immutable ordinal")
		}
		validateCanaryObservation(v, path, observation)
	}
	if err := v.result(); err != nil {
		return nil, err
	}
	return normalized, nil
}

func validateCanaryObservation(v *validator, path string, observation CanaryObservation) {
	switch observation.TerminalState {
	case CanaryStatePending, CanaryStateDelivered, CanaryStateAbandoned:
	default:
		v.add("canary.terminal_state.invalid", path+".terminal_state", "terminal state must be pending, delivered, or abandoned")
	}
	switch observation.LineageOutcome {
	case CanaryLineagePending, CanaryLineageVerified, CanaryLineageUnresolvedAfterResolution, CanaryLineageWrong:
	default:
		v.add("canary.lineage_outcome.invalid", path+".lineage_outcome", "lineage outcome is unknown")
	}
	if observation.Assessment != nil {
		if observation.TerminalState != CanaryStateDelivered {
			v.add("canary.assessment.state_mismatch", path+".assessment", "only delivered changes may have an assessment")
		}
		if !IsCanonicalID(string(observation.Assessment.AssessedBy)) {
			v.add("canary.assessment.actor_invalid", path+".assessment.assessed_by", "assessment requires a canonical owner ID")
		}
		if observation.Assessment.AssessedAt.IsZero() {
			v.add("canary.assessment.timestamp_missing", path+".assessment.assessed_at", "assessment requires a timestamp")
		}
		switch observation.Assessment.Outcome {
		case IntentMatch, IntentPartial, IntentMiss:
		default:
			v.add("canary.assessment.outcome_invalid", path+".assessment.outcome", "assessment outcome is unknown")
		}
	}
	if observation.Variant == VariantBaseline && observation.FlowOverheadMinutes != nil {
		v.add("canary.overhead.baseline_forbidden", path+".flow_overhead_minutes", "flow-only overhead cannot be recorded for baseline")
	}
	if observation.FlowOverheadMinutes != nil {
		value := *observation.FlowOverheadMinutes
		if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
			v.add("canary.overhead.invalid", path+".flow_overhead_minutes", "overhead must be a finite non-negative number")
		}
	}
	if observation.ProcessCausedAbandonment &&
		(observation.Variant != VariantFlow || observation.TerminalState != CanaryStateAbandoned) {
		v.add("canary.abandonment.process_mismatch", path+".process_caused_abandonment", "process-caused abandonment requires an abandoned flow change")
	}
}

func addAssessment(report *CanaryVariantReport, assessment Assessment) {
	report.Assessed++
	if assessment.Outcome == IntentPartial || assessment.Outcome == IntentMiss {
		report.NonMatches++
		if assessment.ChecksGreen {
			report.WrongButGreen++
		}
	}
	if assessment.RepeatOptIn == nil {
		return
	}
	if *assessment.RepeatOptIn {
		report.RepeatOptInYes++
	} else {
		report.RepeatOptInNo++
	}
}

func makeRate(numerator, denominator uint32) CanaryRate {
	rate := CanaryRate{Numerator: numerator, Denominator: denominator}
	if denominator == 0 {
		return rate
	}
	rate.Available = true
	rate.Value = float64(numerator) / float64(denominator)
	return rate
}

func rateLower(left, right CanaryRate) bool {
	if !left.Available || !right.Available {
		return false
	}
	return uint64(left.Numerator)*uint64(right.Denominator) <
		uint64(right.Numerator)*uint64(left.Denominator)
}

func medianMeasurement(values []float64) CanaryMeasurement {
	if len(values) == 0 {
		return CanaryMeasurement{}
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	middle := len(sorted) / 2
	median := sorted[middle]
	if len(sorted)%2 == 0 {
		median = (sorted[middle-1] + sorted[middle]) / 2
	}
	return CanaryMeasurement{Count: uint32(len(sorted)), Value: median, Available: true}
}

func (signals CanaryPassSignals) all() bool {
	return signals.LineageReliable &&
		signals.NonMatchRateLower &&
		signals.PreventionObserved &&
		signals.OverheadTolerable &&
		signals.RepeatOptInAccepted
}

func (signals CanaryHardStopSignals) any() bool {
	return signals.WrongJoin ||
		signals.UnresolvedLinks ||
		signals.EvidenceIntegrityViolation ||
		signals.ExcessiveOverhead ||
		signals.ProcessCausedAbandonments
}
