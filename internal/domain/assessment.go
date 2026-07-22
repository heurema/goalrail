package domain

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidAssessmentBasis = errors.New("invalid canary assessment basis")
	ErrInvalidItemJudgment    = errors.New("invalid intent item judgment")
)

func ValidateCanaryAssessmentBasis(basis CanaryAssessmentBasis) error {
	if !IsEvidenceReference(string(basis.IntentRef)) || !IsCanonicalID(string(basis.IntentID)) ||
		basis.IntentVersion == 0 ||
		(basis.Timing != BasisPreExecution && basis.Timing != BasisPostDelivery) ||
		len(basis.DesiredOutcomeIDs) == 0 || len(basis.SuccessSignalIDs) == 0 {
		return ErrInvalidAssessmentBasis
	}
	seen := make(map[IntentItemID]struct{})
	for _, group := range [][]IntentItemID{basis.DesiredOutcomeIDs, basis.NonGoalIDs, basis.SuccessSignalIDs} {
		for _, id := range group {
			if !IsCanonicalID(string(id)) {
				return fmt.Errorf("%w: item ID %q is invalid", ErrInvalidAssessmentBasis, id)
			}
			if _, duplicate := seen[id]; duplicate {
				return fmt.Errorf("%w: item ID %q is duplicated", ErrInvalidAssessmentBasis, id)
			}
			seen[id] = struct{}{}
		}
	}
	return nil
}

// ValidateAssessmentAgainstBasis requires one owner judgment for every frozen
// item and verifies that the aggregate outcome is the deterministic projection
// of those judgments.
func ValidateAssessmentAgainstBasis(assessment Assessment, basis CanaryAssessmentBasis) error {
	if err := ValidateCanaryAssessmentBasis(basis); err != nil {
		return err
	}
	expected := make(map[IntentItemID]IntentItemCategory)
	for _, id := range basis.DesiredOutcomeIDs {
		expected[id] = IntentCategoryDesiredOutcome
	}
	for _, id := range basis.NonGoalIDs {
		expected[id] = IntentCategoryNonGoal
	}
	for _, id := range basis.SuccessSignalIDs {
		expected[id] = IntentCategorySuccessSignal
	}
	if len(assessment.ItemJudgments) != len(expected) {
		return fmt.Errorf("%w: expected %d judgments, got %d", ErrInvalidItemJudgment, len(expected), len(assessment.ItemJudgments))
	}

	seen := make(map[IntentItemID]struct{}, len(assessment.ItemJudgments))
	derived := IntentMatch
	for _, judgment := range assessment.ItemJudgments {
		category, known := expected[judgment.ItemID]
		if !known {
			return fmt.Errorf("%w: foreign item %q", ErrInvalidItemJudgment, judgment.ItemID)
		}
		if _, duplicate := seen[judgment.ItemID]; duplicate {
			return fmt.Errorf("%w: duplicate item %q", ErrInvalidItemJudgment, judgment.ItemID)
		}
		seen[judgment.ItemID] = struct{}{}
		if judgment.Category != category {
			return fmt.Errorf("%w: item %q has category %q, want %q", ErrInvalidItemJudgment, judgment.ItemID, judgment.Category, category)
		}
		switch category {
		case IntentCategoryDesiredOutcome:
			switch judgment.Judgment {
			case JudgmentAchieved:
			case JudgmentPartial:
				if derived == IntentMatch {
					derived = IntentPartial
				}
			case JudgmentMissed:
				derived = IntentMiss
			default:
				return invalidJudgmentValue(judgment)
			}
		case IntentCategoryNonGoal:
			switch judgment.Judgment {
			case JudgmentPreserved:
			case JudgmentViolated:
				derived = IntentMiss
			default:
				return invalidJudgmentValue(judgment)
			}
		case IntentCategorySuccessSignal:
			switch judgment.Judgment {
			case JudgmentObserved:
			case JudgmentMissing:
				if derived == IntentMatch {
					derived = IntentPartial
				}
			default:
				return invalidJudgmentValue(judgment)
			}
		}
	}
	if assessment.Outcome != derived {
		return fmt.Errorf("%w: aggregate outcome %q, want %q", ErrInvalidItemJudgment, assessment.Outcome, derived)
	}
	return nil
}

func invalidJudgmentValue(judgment IntentItemJudgment) error {
	return fmt.Errorf(
		"%w: judgment %q is invalid for %s item %q",
		ErrInvalidItemJudgment,
		judgment.Judgment,
		judgment.Category,
		judgment.ItemID,
	)
}
