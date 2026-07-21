package domain

import (
	"fmt"
	"testing"
	"time"
)

var canaryTestTime = time.Date(2026, time.July, 21, 12, 0, 0, 0, time.UTC)

func TestCanaryVariantForOrdinalUsesFixedRotation(t *testing.T) {
	var flowCount, baselineCount int
	for ordinal := uint32(1); ordinal <= CanaryAssignmentCount; ordinal++ {
		variant, err := CanaryVariantForOrdinal(ordinal)
		if err != nil {
			t.Fatalf("assign ordinal %d: %v", ordinal, err)
		}
		expected := VariantFlow
		if ordinal%3 == 0 {
			expected = VariantBaseline
		}
		if variant != expected {
			t.Fatalf("ordinal %d variant = %q, want %q", ordinal, variant, expected)
		}
		if variant == VariantFlow {
			flowCount++
		} else {
			baselineCount++
		}
	}
	if flowCount != 10 || baselineCount != 5 {
		t.Fatalf("assignment counts = flow %d baseline %d, want 10 and 5", flowCount, baselineCount)
	}
	for _, ordinal := range []uint32{0, 16} {
		if _, err := CanaryVariantForOrdinal(ordinal); err == nil {
			t.Fatalf("ordinal %d unexpectedly accepted", ordinal)
		}
	}
}

func TestCalculateCanaryReportPassesCompleteEvidence(t *testing.T) {
	observations := passingCanaryObservations(t)
	observations[len(observations)-1].LineageOutcome = CanaryLineageUnresolvedAfterResolution

	report, err := CalculateCanaryReport(CanaryReportInput{Observations: observations})
	if err != nil {
		t.Fatalf("calculate report: %v", err)
	}
	if report.Verdict != CanaryVerdictPass || !report.CompletionReady {
		t.Fatalf("verdict = %q ready=%v, want PASS and ready", report.Verdict, report.CompletionReady)
	}
	if report.Flow.Assigned != 10 || report.Baseline.Assigned != 5 {
		t.Fatalf("unexpected group sizes: flow=%d baseline=%d", report.Flow.Assigned, report.Baseline.Assigned)
	}
	if report.Flow.NonMatchRate.Numerator != 1 || report.Flow.NonMatchRate.Denominator != 10 {
		t.Fatalf("flow rate = %#v, want 1/10", report.Flow.NonMatchRate)
	}
	if report.Baseline.NonMatchRate.Numerator != 2 || report.Baseline.NonMatchRate.Denominator != 5 {
		t.Fatalf("baseline rate = %#v, want 2/5", report.Baseline.NonMatchRate)
	}
	if report.LineageVerified != 14 || report.LineageUnresolved != 1 {
		t.Fatalf("unexpected lineage counts: %#v", report)
	}
	if !report.MedianFlowOverhead.Available || report.MedianFlowOverhead.Value != 10 {
		t.Fatalf("median overhead = %#v, want 10", report.MedianFlowOverhead)
	}
	if !report.PassSignals.all() {
		t.Fatalf("pass signals not all true: %#v", report.PassSignals)
	}
}

func TestCalculateCanaryReportUsesUnequalDeliveredDenominators(t *testing.T) {
	observations := passingCanaryObservations(t)
	setAbandoned(&observations[1], false)
	setAbandoned(&observations[5], false)
	observations[2].Assessment.Outcome = IntentPartial
	for index := range observations {
		if observations[index].Variant == VariantBaseline && observations[index].Ordinal != 3 && observations[index].Assessment != nil {
			observations[index].Assessment.Outcome = IntentMatch
		}
	}

	report, err := CalculateCanaryReport(CanaryReportInput{Observations: observations})
	if err != nil {
		t.Fatalf("calculate report: %v", err)
	}
	if report.Flow.NonMatchRate.Numerator != 1 || report.Flow.NonMatchRate.Denominator != 9 {
		t.Fatalf("flow rate = %#v, want 1/9", report.Flow.NonMatchRate)
	}
	if report.Baseline.NonMatchRate.Numerator != 1 || report.Baseline.NonMatchRate.Denominator != 4 {
		t.Fatalf("baseline rate = %#v, want 1/4", report.Baseline.NonMatchRate)
	}
	if !report.PassSignals.NonMatchRateLower {
		t.Fatal("equal raw non-match counts hid the lower flow rate")
	}
	if report.Flow.Abandoned != 1 || report.Baseline.Abandoned != 1 {
		t.Fatalf("abandonments not reported separately: flow=%d baseline=%d", report.Flow.Abandoned, report.Baseline.Abandoned)
	}
}

func TestCalculateCanaryReportKeepsMissingAssessmentOutOfRate(t *testing.T) {
	observations := passingCanaryObservations(t)
	observations[0].Assessment = nil

	report, err := CalculateCanaryReport(CanaryReportInput{Observations: observations})
	if err != nil {
		t.Fatalf("calculate report: %v", err)
	}
	if report.Verdict != CanaryVerdictPending || report.CompletionReady {
		t.Fatalf("missing assessment verdict = %q ready=%v, want PENDING", report.Verdict, report.CompletionReady)
	}
	if report.Flow.MissingAssessments != 1 || report.Flow.NonMatchRate.Denominator != 9 {
		t.Fatalf("missing assessment was hidden or counted as match: %#v", report.Flow)
	}
}

func TestCalculateCanaryReportCountsWrongButGreenIndependently(t *testing.T) {
	observations := passingCanaryObservations(t)
	observations[0].Assessment.Outcome = IntentMiss
	observations[0].Assessment.ChecksGreen = true

	report, err := CalculateCanaryReport(CanaryReportInput{Observations: observations})
	if err != nil {
		t.Fatalf("calculate report: %v", err)
	}
	if report.Flow.WrongButGreen != 1 || report.Flow.NonMatches != 1 {
		t.Fatalf("wrong-but-green not preserved: %#v", report.Flow)
	}
}

func TestCalculateCanaryReportAppliesEveryHardStop(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*CanaryReportInput)
		assert func(CanaryHardStopSignals) bool
	}{
		{
			name: "wrong join",
			mutate: func(input *CanaryReportInput) {
				input.Observations[0].LineageOutcome = CanaryLineageWrong
			},
			assert: func(signals CanaryHardStopSignals) bool { return signals.WrongJoin },
		},
		{
			name: "two unresolved links",
			mutate: func(input *CanaryReportInput) {
				input.Observations[0].LineageOutcome = CanaryLineageUnresolvedAfterResolution
				input.Observations[1].LineageOutcome = CanaryLineageUnresolvedAfterResolution
			},
			assert: func(signals CanaryHardStopSignals) bool { return signals.UnresolvedLinks },
		},
		{
			name: "evidence rewrite",
			mutate: func(input *CanaryReportInput) {
				input.EvidenceIntegrityViolations = 1
			},
			assert: func(signals CanaryHardStopSignals) bool { return signals.EvidenceIntegrityViolation },
		},
		{
			name: "excessive median overhead",
			mutate: func(input *CanaryReportInput) {
				for index := range input.Observations {
					if input.Observations[index].Variant == VariantFlow {
						minutes := float64(31)
						input.Observations[index].FlowOverheadMinutes = &minutes
					}
				}
			},
			assert: func(signals CanaryHardStopSignals) bool { return signals.ExcessiveOverhead },
		},
		{
			name: "three process-caused flow abandonments",
			mutate: func(input *CanaryReportInput) {
				abandoned := 0
				for index := range input.Observations {
					if input.Observations[index].Variant == VariantFlow && abandoned < 3 {
						setAbandoned(&input.Observations[index], true)
						abandoned++
					}
				}
			},
			assert: func(signals CanaryHardStopSignals) bool { return signals.ProcessCausedAbandonments },
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := CanaryReportInput{Observations: passingCanaryObservations(t)}
			test.mutate(&input)
			report, err := CalculateCanaryReport(input)
			if err != nil {
				t.Fatalf("calculate report: %v", err)
			}
			if report.Verdict != CanaryVerdictStop || !test.assert(report.HardStopSignals) {
				t.Fatalf("hard stop not applied: verdict=%q signals=%#v", report.Verdict, report.HardStopSignals)
			}
		})
	}
}

func TestCalculateCanaryReportReshapesCompletedNonPass(t *testing.T) {
	observations := passingCanaryObservations(t)
	for index := range observations {
		observations[index].MaterialMisunderstandingPrevented = false
		observations[index].Assessment.Outcome = IntentMatch
	}
	observations[0].Assessment.Outcome = IntentPartial
	observations[1].Assessment.Outcome = IntentPartial
	observations[2].Assessment.Outcome = IntentPartial

	report, err := CalculateCanaryReport(CanaryReportInput{Observations: observations})
	if err != nil {
		t.Fatalf("calculate report: %v", err)
	}
	if report.Verdict != CanaryVerdictReshape || !report.NoUsefulMovement {
		t.Fatalf("verdict = %q no_movement=%v, want RESHAPE", report.Verdict, report.NoUsefulMovement)
	}
}

func TestCalculateCanaryReportRejectsMutableAssignmentFacts(t *testing.T) {
	observations := passingCanaryObservations(t)
	observations[0].Variant = VariantBaseline
	if _, err := CalculateCanaryReport(CanaryReportInput{Observations: observations}); err == nil {
		t.Fatal("variant that conflicts with immutable ordinal was accepted")
	}

	observations = passingCanaryObservations(t)
	observations[1].Ordinal = 1
	if _, err := CalculateCanaryReport(CanaryReportInput{Observations: observations}); err == nil {
		t.Fatal("duplicate ordinal was accepted")
	}
}

func passingCanaryObservations(t *testing.T) []CanaryObservation {
	t.Helper()
	observations := make([]CanaryObservation, 0, CanaryAssignmentCount)
	for ordinal := uint32(1); ordinal <= CanaryAssignmentCount; ordinal++ {
		variant, err := CanaryVariantForOrdinal(ordinal)
		if err != nil {
			t.Fatalf("assign ordinal %d: %v", ordinal, err)
		}
		repeat := true
		assessment := &Assessment{
			Outcome:     IntentMatch,
			AssessedBy:  "owner",
			AssessedAt:  canaryTestTime,
			ChecksGreen: true,
			RepeatOptIn: &repeat,
		}
		observation := CanaryObservation{
			Ordinal:        ordinal,
			ChangeID:       ChangeID(fmt.Sprintf("change-%02d", ordinal)),
			Variant:        variant,
			TerminalState:  CanaryStateDelivered,
			LineageOutcome: CanaryLineageVerified,
			Assessment:     assessment,
		}
		if variant == VariantFlow {
			minutes := float64(10)
			observation.FlowOverheadMinutes = &minutes
		}
		observations = append(observations, observation)
	}
	observations[0].Assessment.Outcome = IntentPartial
	observations[0].MaterialMisunderstandingPrevented = true
	observations[2].Assessment.Outcome = IntentPartial
	observations[5].Assessment.Outcome = IntentMiss
	return observations
}

func setAbandoned(observation *CanaryObservation, processCaused bool) {
	observation.TerminalState = CanaryStateAbandoned
	observation.Assessment = nil
	observation.ProcessCausedAbandonment = processCaused
}
