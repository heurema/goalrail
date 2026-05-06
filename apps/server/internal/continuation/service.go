package continuation

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/heurema/goalrail/apps/server/internal/goal"
	"github.com/heurema/goalrail/apps/server/internal/spine"
)

var (
	ErrGoalNotFound          = errors.New("goal not found")
	ErrMembershipRequired    = errors.New("active organization membership is required")
	ErrOrganizationForbidden = errors.New("user is not allowed to continue this goal")
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Field == "" {
		return e.Message
	}
	return e.Field + ": " + e.Message
}

type GoalReader interface {
	Get(context.Context, spine.GoalID) (spine.Goal, bool, error)
}

type ReadinessChecker interface {
	CheckReadiness(context.Context, spine.GoalID) (spine.GoalReadinessResult, spine.Goal, error)
}

type ClarificationRequestEnsurer interface {
	GetOrCreateOpenRequest(context.Context, spine.GoalID) (spine.ClarificationRequest, error)
}

type Service struct {
	Goals          GoalReader
	Readiness      ReadinessChecker
	Clarifications ClarificationRequestEnsurer
}

func NewService(goals GoalReader, readiness ReadinessChecker, clarifications ClarificationRequestEnsurer) *Service {
	return &Service{
		Goals:          goals,
		Readiness:      readiness,
		Clarifications: clarifications,
	}
}

func (s *Service) ReconcileGoal(ctx context.Context, goalID spine.GoalID, membership spine.OrganizationMembership) (spine.GoalContinuation, error) {
	if err := validateGoalID(goalID); err != nil {
		return spine.GoalContinuation{}, err
	}

	current, ok, err := s.Goals.Get(ctx, goalID)
	if err != nil {
		return spine.GoalContinuation{}, fmt.Errorf("get goal: %w", err)
	}
	if !ok {
		return spine.GoalContinuation{}, ErrGoalNotFound
	}
	if err := authorizeGoalContinuation(membership, current); err != nil {
		return spine.GoalContinuation{}, err
	}
	if current.State == spine.GoalStateRejected {
		goalCopy := current
		return spine.GoalContinuation{
			GoalID: current.ID,
			State:  current.State,
			Goal:   &goalCopy,
		}, nil
	}

	readiness, updated, err := s.Readiness.CheckReadiness(ctx, goalID)
	if err != nil {
		if errors.Is(err, goal.ErrGoalNotFound) {
			return spine.GoalContinuation{}, ErrGoalNotFound
		}
		return spine.GoalContinuation{}, err
	}

	result := spine.GoalContinuation{
		GoalID:    updated.ID,
		State:     updated.State,
		Readiness: &readiness,
		Goal:      &updated,
	}
	if updated.State != spine.GoalStateNeedsClarification {
		return result, nil
	}

	request, err := s.Clarifications.GetOrCreateOpenRequest(ctx, updated.ID)
	if err != nil {
		return spine.GoalContinuation{}, err
	}
	result.ClarificationRequest = &request
	return result, nil
}

func authorizeGoalContinuation(membership spine.OrganizationMembership, current spine.Goal) error {
	if membership.State != spine.EntityStateActive || strings.TrimSpace(string(membership.OrganizationID)) == "" {
		return ErrMembershipRequired
	}
	if membership.OrganizationID != current.OrganizationID {
		return ErrOrganizationForbidden
	}
	return nil
}

func validateGoalID(goalID spine.GoalID) error {
	text := strings.TrimSpace(string(goalID))
	if text == "" {
		return &ValidationError{Field: "goal_id", Message: "is required"}
	}
	id, err := uuid.Parse(text)
	if err != nil {
		return &ValidationError{Field: "goal_id", Message: "must be a UUID"}
	}
	if id.Version() != 7 {
		return &ValidationError{Field: "goal_id", Message: "must be a UUIDv7"}
	}
	return nil
}
