package continuation

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/heurema/goalrail/apps/server/internal/clarification"
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

type ClarificationWorkflow interface {
	ClarificationRequestEnsurer
	Get(context.Context, spine.ClarificationRequestID) (spine.ClarificationRequest, bool, error)
	RecordAnswer(context.Context, spine.ClarificationRequestID, spine.ClarificationAnswerSubmission) (spine.ClarificationAnswer, error)
	ApplyAnswer(context.Context, spine.ClarificationAnswerID, spine.ClarificationAnswerApplicationRequest) (spine.ClarificationAnswerApplicationResult, spine.Goal, error)
}

type Service struct {
	Goals          GoalReader
	Readiness      ReadinessChecker
	Clarifications ClarificationWorkflow
}

func NewService(goals GoalReader, readiness ReadinessChecker, clarifications ClarificationWorkflow) *Service {
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

func (s *Service) AnswerClarification(ctx context.Context, requestID spine.ClarificationRequestID, input spine.WorkAnswerSubmission, membership spine.OrganizationMembership, actor spine.ActorRef) (spine.GoalContinuation, error) {
	if err := validateClarificationRequestID(requestID); err != nil {
		return spine.GoalContinuation{}, err
	}

	request, ok, err := s.Clarifications.Get(ctx, requestID)
	if err != nil {
		return spine.GoalContinuation{}, err
	}
	if !ok {
		return spine.GoalContinuation{}, clarification.ErrRequestNotFound
	}

	current, ok, err := s.Goals.Get(ctx, request.GoalID)
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

	if strings.TrimSpace(actor.Kind) == "" {
		return spine.GoalContinuation{}, &ValidationError{Field: "submitted_by.kind", Message: "is required"}
	}
	if strings.TrimSpace(actor.ID) == "" {
		return spine.GoalContinuation{}, &ValidationError{Field: "submitted_by.id", Message: "is required"}
	}
	if request.State == spine.ClarificationRequestStateOpen {
		if err := validateWorkAnswerSubmissionForApplication(request, input); err != nil {
			return spine.GoalContinuation{}, err
		}
	}

	answer, err := s.Clarifications.RecordAnswer(ctx, request.ID, spine.ClarificationAnswerSubmission{
		Answers:     input.Answers,
		SubmittedBy: actor,
	})
	if err != nil {
		return spine.GoalContinuation{}, err
	}
	if _, _, err := s.Clarifications.ApplyAnswer(ctx, answer.ID, spine.ClarificationAnswerApplicationRequest{AppliedBy: actor}); err != nil {
		return spine.GoalContinuation{}, err
	}

	return s.ReconcileGoal(ctx, request.GoalID, membership)
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

func validateWorkAnswerSubmissionForApplication(request spine.ClarificationRequest, input spine.WorkAnswerSubmission) error {
	if len(input.Answers) == 0 {
		return &ValidationError{Field: "answers", Message: "at least one answer is required"}
	}

	questions := make(map[spine.ClarificationQuestionID]spine.ClarificationQuestion, len(request.Questions))
	for _, question := range request.Questions {
		questions[question.ID] = question
	}

	answered := make(map[spine.ClarificationQuestionID]bool, len(input.Answers))
	for _, answer := range input.Answers {
		question, ok := questions[answer.QuestionID]
		if !ok {
			return &ValidationError{Field: "answers.question_id", Message: "unknown question_id"}
		}
		if answered[answer.QuestionID] {
			return &ValidationError{Field: "answers.question_id", Message: "duplicate question_id"}
		}
		answered[answer.QuestionID] = true

		switch question.MapsTo {
		case spine.ClarificationMapsToGoalSummary, spine.ClarificationMapsToGoalScopeHint, spine.ClarificationMapsToGoalAcceptanceHint:
			if strings.TrimSpace(answer.Value) == "" {
				return &ValidationError{Field: "answers.value", Message: "mapped value is required"}
			}
		case spine.ClarificationMapsToGoalIntentOwner:
			if answer.ActorRef == nil || strings.TrimSpace(answer.ActorRef.Kind) == "" || strings.TrimSpace(answer.ActorRef.ID) == "" {
				return fmt.Errorf("%w: %s requires actor-shaped value", clarification.ErrUnsupportedMapping, question.MapsTo)
			}
		default:
			return fmt.Errorf("%w: %s", clarification.ErrUnsupportedMapping, question.MapsTo)
		}
	}

	for _, question := range request.Questions {
		if !answered[question.ID] {
			return &ValidationError{Field: "answers", Message: "all request questions must be answered"}
		}
	}
	return nil
}

func validateClarificationRequestID(requestID spine.ClarificationRequestID) error {
	text := strings.TrimSpace(string(requestID))
	if text == "" {
		return &ValidationError{Field: "clarification_request_id", Message: "is required"}
	}
	id, err := uuid.Parse(text)
	if err != nil {
		return &ValidationError{Field: "clarification_request_id", Message: "must be a UUID"}
	}
	if id.Version() != 7 {
		return &ValidationError{Field: "clarification_request_id", Message: "must be a UUIDv7"}
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
