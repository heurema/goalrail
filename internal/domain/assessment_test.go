package domain

import (
	"errors"
	"testing"
)

func TestValidateAssessmentAgainstBasisDerivesAggregate(t *testing.T) {
	basis := testAssessmentBasis()
	tests := []struct {
		name      string
		outcome   IntentOutcome
		judgments []IntentItemJudgment
	}{
		{name: "match", outcome: IntentMatch, judgments: testItemJudgments(JudgmentAchieved, JudgmentPreserved, JudgmentObserved)},
		{name: "partial outcome", outcome: IntentPartial, judgments: testItemJudgments(JudgmentPartial, JudgmentPreserved, JudgmentObserved)},
		{name: "missing signal", outcome: IntentPartial, judgments: testItemJudgments(JudgmentAchieved, JudgmentPreserved, JudgmentMissing)},
		{name: "missed outcome", outcome: IntentMiss, judgments: testItemJudgments(JudgmentMissed, JudgmentPreserved, JudgmentObserved)},
		{name: "violated non-goal", outcome: IntentMiss, judgments: testItemJudgments(JudgmentAchieved, JudgmentViolated, JudgmentObserved)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := ValidateAssessmentAgainstBasis(Assessment{Outcome: test.outcome, ItemJudgments: test.judgments}, basis); err != nil {
				t.Fatalf("validate assessment: %v", err)
			}
		})
	}
}

func TestValidateAssessmentAgainstBasisRejectsIncompleteDuplicateForeignAndInconsistent(t *testing.T) {
	basis := testAssessmentBasis()
	tests := []struct {
		name       string
		assessment Assessment
	}{
		{name: "incomplete", assessment: Assessment{Outcome: IntentMatch, ItemJudgments: testItemJudgments(JudgmentAchieved, JudgmentPreserved, JudgmentObserved)[:2]}},
		{name: "duplicate", assessment: Assessment{Outcome: IntentMatch, ItemJudgments: []IntentItemJudgment{
			{ItemID: "OUT-1", Category: IntentCategoryDesiredOutcome, Judgment: JudgmentAchieved},
			{ItemID: "OUT-1", Category: IntentCategoryDesiredOutcome, Judgment: JudgmentAchieved},
			{ItemID: "SIG-1", Category: IntentCategorySuccessSignal, Judgment: JudgmentObserved},
		}}},
		{name: "foreign", assessment: Assessment{Outcome: IntentMatch, ItemJudgments: []IntentItemJudgment{
			{ItemID: "OUT-1", Category: IntentCategoryDesiredOutcome, Judgment: JudgmentAchieved},
			{ItemID: "NG-1", Category: IntentCategoryNonGoal, Judgment: JudgmentPreserved},
			{ItemID: "SIG-foreign", Category: IntentCategorySuccessSignal, Judgment: JudgmentObserved},
		}}},
		{name: "inconsistent", assessment: Assessment{Outcome: IntentMatch, ItemJudgments: testItemJudgments(JudgmentMissed, JudgmentPreserved, JudgmentObserved)}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := ValidateAssessmentAgainstBasis(test.assessment, basis); !errors.Is(err, ErrInvalidItemJudgment) {
				t.Fatalf("validation error = %v", err)
			}
		})
	}
}

func testAssessmentBasis() CanaryAssessmentBasis {
	return CanaryAssessmentBasis{
		IntentRef: "openspec:test", IntentID: "intent-test", IntentVersion: 1, Timing: BasisPreExecution,
		DesiredOutcomeIDs: []IntentItemID{"OUT-1"}, NonGoalIDs: []IntentItemID{"NG-1"}, SuccessSignalIDs: []IntentItemID{"SIG-1"},
	}
}

func testItemJudgments(outcome, nonGoal, signal IntentItemJudgmentValue) []IntentItemJudgment {
	return []IntentItemJudgment{
		{ItemID: "OUT-1", Category: IntentCategoryDesiredOutcome, Judgment: outcome},
		{ItemID: "NG-1", Category: IntentCategoryNonGoal, Judgment: nonGoal},
		{ItemID: "SIG-1", Category: IntentCategorySuccessSignal, Judgment: signal},
	}
}
