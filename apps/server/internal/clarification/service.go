package clarification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/heurema/goalrail/apps/server/internal/spine"
)

const (
	EventTypeClarificationRequested       = "clarification.requested"
	EventTypeClarificationAnswerRecorded  = "clarification.answer_recorded"
	EventTypeClarificationRequestAnswered = "clarification.request_answered"
	EventTypeClarificationAnswerApplied   = "clarification.answer_applied_to_goal"
	EventTypeGoalHintsUpdated             = "goal.hints_updated"
	EntityTypeClarificationRequest        = "ClarificationRequest"
	EntityTypeClarificationAnswer         = "ClarificationAnswer"
	EntityTypeGoal                        = "Goal"
)

var (
	ErrGoalNotFound             = errors.New("goal not found")
	ErrInvalidGoalState         = errors.New("goal state is not clarification-requestable")
	ErrAlreadyOpen              = errors.New("clarification request already open")
	ErrRequestNotFound          = errors.New("clarification request not found")
	ErrInvalidRequestState      = errors.New("clarification request state is not answerable")
	ErrRequestNotAnswered       = errors.New("clarification request is not answered")
	ErrAnswerNotFound           = errors.New("clarification answer not found")
	ErrInvalidAnswerState       = errors.New("clarification answer state is not applicable")
	ErrAlreadyAnswered          = errors.New("clarification request already answered")
	ErrAlreadyApplied           = errors.New("clarification answer already applied")
	ErrUnsupportedMapping       = errors.New("unsupported clarification answer mapping")
	ErrMissingReadinessReasons  = errors.New("goal has no stored readiness reason codes")
	ErrNoClarificationQuestions = errors.New("no clarification questions available")
	ErrPolicyRejected           = errors.New("policy rejected goals cannot create clarification request")
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
	UpdateHints(context.Context, spine.GoalID, spine.GoalHintUpdate) (spine.Goal, bool, error)
}

type Store interface {
	Create(context.Context, spine.ClarificationRequest) error
	Get(context.Context, spine.ClarificationRequestID) (spine.ClarificationRequest, bool, error)
	GetOpenByGoalID(context.Context, spine.GoalID) (spine.ClarificationRequest, bool, error)
	UpdateState(context.Context, spine.ClarificationRequestID, spine.ClarificationRequestState) (spine.ClarificationRequest, bool, error)
}

type AnswerStore interface {
	Create(context.Context, spine.ClarificationAnswer) error
	Get(context.Context, spine.ClarificationAnswerID) (spine.ClarificationAnswer, bool, error)
	GetByRequestID(context.Context, spine.ClarificationRequestID) (spine.ClarificationAnswer, bool, error)
	MarkApplied(context.Context, spine.ClarificationAnswerID, spine.ActorRef, time.Time) (bool, error)
}

type EventLog interface {
	Append(context.Context, spine.Event) error
}

type TransactionRunner interface {
	RunReadCommitted(context.Context, func(context.Context) error) error
}

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	NewClarificationRequestID() (spine.ClarificationRequestID, error)
	NewClarificationQuestionID() (spine.ClarificationQuestionID, error)
	NewClarificationAnswerID() (spine.ClarificationAnswerID, error)
	NewEventID() (spine.EventID, error)
}

type Service struct {
	Goals   GoalReader
	Store   Store
	Answers AnswerStore
	Events  EventLog
	Clock   Clock
	IDs     IDGenerator

	TxRunner TransactionRunner
}

type Option func(*Service)

func WithTransactionRunner(runner TransactionRunner) Option {
	return func(s *Service) {
		s.TxRunner = runner
	}
}

func NewService(goals GoalReader, store Store, answers AnswerStore, events EventLog, clock Clock, ids IDGenerator, options ...Option) *Service {
	service := &Service{
		Goals:   goals,
		Store:   store,
		Answers: answers,
		Events:  events,
		Clock:   clock,
		IDs:     ids,
	}
	for _, option := range options {
		option(service)
	}
	return service
}

func (s *Service) CreateRequest(ctx context.Context, goalID spine.GoalID) (spine.ClarificationRequest, error) {
	if err := s.validateDependencies(); err != nil {
		return spine.ClarificationRequest{}, err
	}

	goal, ok, err := s.Goals.Get(ctx, goalID)
	if err != nil {
		return spine.ClarificationRequest{}, fmt.Errorf("get goal: %w", err)
	}
	if !ok {
		return spine.ClarificationRequest{}, ErrGoalNotFound
	}
	if goal.State != spine.GoalStateNeedsClarification {
		return spine.ClarificationRequest{}, fmt.Errorf("%w: %s", ErrInvalidGoalState, goal.State)
	}
	if _, ok, err := s.Store.GetOpenByGoalID(ctx, goal.ID); err != nil {
		return spine.ClarificationRequest{}, fmt.Errorf("get open clarification request: %w", err)
	} else if ok {
		return spine.ClarificationRequest{}, ErrAlreadyOpen
	}

	requestID, err := s.IDs.NewClarificationRequestID()
	if err != nil {
		return spine.ClarificationRequest{}, fmt.Errorf("new clarification request id: %w", err)
	}
	questions, err := s.questionsForReasons(goal.LastReadinessReasonCodes)
	if err != nil {
		return spine.ClarificationRequest{}, err
	}

	created := spine.ClarificationRequest{
		ID:          requestID,
		GoalID:      goal.ID,
		ReasonCodes: cloneReasonCodes(goal.LastReadinessReasonCodes),
		Questions:   questions,
		Target:      targetForGoal(goal),
		State:       spine.ClarificationRequestStateOpen,
		CreatedAt:   s.Clock.Now().UTC(),
	}

	event, err := s.clarificationRequestedEvent(created)
	if err != nil {
		return spine.ClarificationRequest{}, err
	}

	create := func(createCtx context.Context) error {
		if err := s.Store.Create(createCtx, created); err != nil {
			if errors.Is(err, ErrAlreadyOpen) {
				return ErrAlreadyOpen
			}
			return fmt.Errorf("create clarification request: %w", err)
		}
		if err := s.Events.Append(createCtx, event); err != nil {
			return fmt.Errorf("append clarification requested event: %w", err)
		}
		return nil
	}
	if s.TxRunner != nil {
		if err := s.TxRunner.RunReadCommitted(ctx, create); err != nil {
			if errors.Is(err, ErrAlreadyOpen) {
				return spine.ClarificationRequest{}, ErrAlreadyOpen
			}
			return spine.ClarificationRequest{}, fmt.Errorf("create clarification request transaction: %w", err)
		}
	} else if err := create(ctx); err != nil {
		if errors.Is(err, ErrAlreadyOpen) {
			return spine.ClarificationRequest{}, ErrAlreadyOpen
		}
		return spine.ClarificationRequest{}, err
	}

	return created, nil
}

func (s *Service) RecordAnswer(ctx context.Context, requestID spine.ClarificationRequestID, input spine.ClarificationAnswerSubmission) (spine.ClarificationAnswer, error) {
	if err := s.validateDependencies(); err != nil {
		return spine.ClarificationAnswer{}, err
	}

	request, ok, err := s.Store.Get(ctx, requestID)
	if err != nil {
		return spine.ClarificationAnswer{}, fmt.Errorf("get clarification request: %w", err)
	}
	if !ok {
		return spine.ClarificationAnswer{}, ErrRequestNotFound
	}
	if request.State == spine.ClarificationRequestStateAnswered {
		return spine.ClarificationAnswer{}, ErrAlreadyAnswered
	}
	if request.State != spine.ClarificationRequestStateOpen {
		return spine.ClarificationAnswer{}, fmt.Errorf("%w: %s", ErrInvalidRequestState, request.State)
	}
	if _, ok, err := s.Answers.GetByRequestID(ctx, request.ID); err != nil {
		return spine.ClarificationAnswer{}, fmt.Errorf("get clarification answer by request id: %w", err)
	} else if ok {
		return spine.ClarificationAnswer{}, ErrAlreadyAnswered
	}

	if err := validateAnswerSubmission(request, input); err != nil {
		return spine.ClarificationAnswer{}, err
	}

	answerID, err := s.IDs.NewClarificationAnswerID()
	if err != nil {
		return spine.ClarificationAnswer{}, fmt.Errorf("new clarification answer id: %w", err)
	}
	now := s.Clock.Now().UTC()
	recorded := spine.ClarificationAnswer{
		ID:          answerID,
		RequestID:   request.ID,
		GoalID:      request.GoalID,
		Answers:     cloneAnswerItems(input.Answers),
		SubmittedBy: input.SubmittedBy,
		State:       spine.ClarificationAnswerStateRecorded,
		CreatedAt:   now,
	}

	answerRecorded, err := s.clarificationAnswerRecordedEvent(recorded)
	if err != nil {
		return spine.ClarificationAnswer{}, err
	}
	requestAnswered, err := s.clarificationRequestAnsweredEvent(request.ID, recorded.ID, request.State, spine.ClarificationRequestStateAnswered, now)
	if err != nil {
		return spine.ClarificationAnswer{}, err
	}

	events := []spine.Event{answerRecorded, requestAnswered}
	record := func(recordCtx context.Context) error {
		if err := s.Answers.Create(recordCtx, recorded); err != nil {
			return fmt.Errorf("create clarification answer: %w", err)
		}
		if _, ok, err := s.Store.UpdateState(recordCtx, request.ID, spine.ClarificationRequestStateAnswered); err != nil {
			return fmt.Errorf("update clarification request state: %w", err)
		} else if !ok {
			return ErrRequestNotFound
		}
		for _, event := range events {
			if err := s.Events.Append(recordCtx, event); err != nil {
				return fmt.Errorf("append clarification answer event: %w", err)
			}
		}
		return nil
	}
	if s.TxRunner != nil {
		if err := s.TxRunner.RunReadCommitted(ctx, record); err != nil {
			if errors.Is(err, ErrRequestNotFound) {
				return spine.ClarificationAnswer{}, ErrRequestNotFound
			}
			return spine.ClarificationAnswer{}, fmt.Errorf("record clarification answer transaction: %w", err)
		}
	} else if err := record(ctx); err != nil {
		return spine.ClarificationAnswer{}, err
	}

	return recorded, nil
}

func (s *Service) ApplyAnswer(ctx context.Context, answerID spine.ClarificationAnswerID, input spine.ClarificationAnswerApplicationRequest) (spine.ClarificationAnswerApplicationResult, spine.Goal, error) {
	if err := s.validateDependencies(); err != nil {
		return spine.ClarificationAnswerApplicationResult{}, spine.Goal{}, err
	}
	if strings.TrimSpace(input.AppliedBy.Kind) == "" {
		return spine.ClarificationAnswerApplicationResult{}, spine.Goal{}, &ValidationError{Field: "applied_by.kind", Message: "is required"}
	}
	if strings.TrimSpace(input.AppliedBy.ID) == "" {
		return spine.ClarificationAnswerApplicationResult{}, spine.Goal{}, &ValidationError{Field: "applied_by.id", Message: "is required"}
	}

	answer, ok, err := s.Answers.Get(ctx, answerID)
	if err != nil {
		return spine.ClarificationAnswerApplicationResult{}, spine.Goal{}, fmt.Errorf("get clarification answer: %w", err)
	}
	if !ok {
		return spine.ClarificationAnswerApplicationResult{}, spine.Goal{}, ErrAnswerNotFound
	}
	if answer.State != spine.ClarificationAnswerStateRecorded {
		return spine.ClarificationAnswerApplicationResult{}, spine.Goal{}, fmt.Errorf("%w: %s", ErrInvalidAnswerState, answer.State)
	}

	request, ok, err := s.Store.Get(ctx, answer.RequestID)
	if err != nil {
		return spine.ClarificationAnswerApplicationResult{}, spine.Goal{}, fmt.Errorf("get clarification request: %w", err)
	}
	if !ok {
		return spine.ClarificationAnswerApplicationResult{}, spine.Goal{}, ErrRequestNotFound
	}
	if request.State != spine.ClarificationRequestStateAnswered {
		return spine.ClarificationAnswerApplicationResult{}, spine.Goal{}, fmt.Errorf("%w: %s", ErrRequestNotAnswered, request.State)
	}

	goal, ok, err := s.Goals.Get(ctx, answer.GoalID)
	if err != nil {
		return spine.ClarificationAnswerApplicationResult{}, spine.Goal{}, fmt.Errorf("get goal: %w", err)
	}
	if !ok {
		return spine.ClarificationAnswerApplicationResult{}, spine.Goal{}, ErrGoalNotFound
	}

	update, mappings, err := applicationUpdate(goal, request, answer)
	if err != nil {
		return spine.ClarificationAnswerApplicationResult{}, spine.Goal{}, err
	}

	appliedAt := s.Clock.Now().UTC()
	result := spine.ClarificationAnswerApplicationResult{
		AnswerID:        answer.ID,
		GoalID:          goal.ID,
		AppliedBy:       input.AppliedBy,
		AppliedMappings: mappings,
		AppliedAt:       appliedAt,
	}

	answerApplied, err := s.clarificationAnswerAppliedEvent(result)
	if err != nil {
		return spine.ClarificationAnswerApplicationResult{}, spine.Goal{}, err
	}
	goalHintsUpdated, err := s.goalHintsUpdatedEvent(result)
	if err != nil {
		return spine.ClarificationAnswerApplicationResult{}, spine.Goal{}, err
	}
	events := []spine.Event{answerApplied, goalHintsUpdated}

	var updatedGoal spine.Goal
	marked := false
	goalOK := false
	apply := func(applyCtx context.Context) error {
		var err error
		marked, err = s.Answers.MarkApplied(applyCtx, answer.ID, input.AppliedBy, appliedAt)
		if err != nil {
			return fmt.Errorf("mark clarification answer applied: %w", err)
		}
		if !marked {
			return nil
		}

		updatedGoal, goalOK, err = s.Goals.UpdateHints(applyCtx, goal.ID, update)
		if err != nil {
			return fmt.Errorf("update goal hints: %w", err)
		}
		if !goalOK {
			return ErrGoalNotFound
		}

		for _, event := range events {
			if err := s.Events.Append(applyCtx, event); err != nil {
				return fmt.Errorf("append clarification answer application event: %w", err)
			}
		}
		return nil
	}
	if s.TxRunner != nil {
		if err := s.TxRunner.RunReadCommitted(ctx, apply); err != nil {
			if errors.Is(err, ErrGoalNotFound) {
				return spine.ClarificationAnswerApplicationResult{}, spine.Goal{}, ErrGoalNotFound
			}
			return spine.ClarificationAnswerApplicationResult{}, spine.Goal{}, fmt.Errorf("apply clarification answer transaction: %w", err)
		}
	} else if err := apply(ctx); err != nil {
		return spine.ClarificationAnswerApplicationResult{}, spine.Goal{}, err
	}
	if !marked {
		return spine.ClarificationAnswerApplicationResult{}, spine.Goal{}, ErrAlreadyApplied
	}
	if !goalOK {
		return spine.ClarificationAnswerApplicationResult{}, spine.Goal{}, ErrGoalNotFound
	}

	return result, updatedGoal, nil
}

func (s *Service) questionsForReasons(reasons []spine.GoalReadinessReasonCode) ([]spine.ClarificationQuestion, error) {
	if len(reasons) == 0 {
		return nil, ErrMissingReadinessReasons
	}

	seen := make(map[spine.GoalReadinessReasonCode]bool, len(reasons))
	questions := make([]spine.ClarificationQuestion, 0, len(reasons))
	for _, reason := range reasons {
		if seen[reason] {
			continue
		}
		seen[reason] = true
		if reason == spine.GoalReadinessReasonPolicyRejected {
			return nil, ErrPolicyRejected
		}

		spec, ok := questionSpec(reason)
		if !ok {
			continue
		}
		questionID, err := s.IDs.NewClarificationQuestionID()
		if err != nil {
			return nil, fmt.Errorf("new clarification question id: %w", err)
		}
		questions = append(questions, spine.ClarificationQuestion{
			ID:         questionID,
			Text:       spec.text,
			WhyNeeded:  spec.whyNeeded,
			AnswerType: spine.ClarificationAnswerTypeText,
			MapsTo:     spec.mapsTo,
		})
	}
	if len(questions) == 0 {
		return nil, ErrNoClarificationQuestions
	}
	return questions, nil
}

func validateAnswerSubmission(request spine.ClarificationRequest, input spine.ClarificationAnswerSubmission) error {
	if strings.TrimSpace(input.SubmittedBy.Kind) == "" {
		return &ValidationError{Field: "submitted_by.kind", Message: "is required"}
	}
	if strings.TrimSpace(input.SubmittedBy.ID) == "" {
		return &ValidationError{Field: "submitted_by.id", Message: "is required"}
	}
	if len(input.Answers) == 0 {
		return &ValidationError{Field: "answers", Message: "at least one answer is required"}
	}

	questions := make(map[spine.ClarificationQuestionID]bool, len(request.Questions))
	for _, question := range request.Questions {
		questions[question.ID] = true
	}

	answered := make(map[spine.ClarificationQuestionID]bool, len(input.Answers))
	for _, answer := range input.Answers {
		if !questions[answer.QuestionID] {
			return &ValidationError{Field: "answers.question_id", Message: "unknown question_id"}
		}
		if answered[answer.QuestionID] {
			return &ValidationError{Field: "answers.question_id", Message: "duplicate question_id"}
		}
		answered[answer.QuestionID] = true
	}

	for _, question := range request.Questions {
		if !answered[question.ID] {
			return &ValidationError{Field: "answers", Message: "all request questions must be answered"}
		}
	}
	return nil
}

func applicationUpdate(goal spine.Goal, request spine.ClarificationRequest, answer spine.ClarificationAnswer) (spine.GoalHintUpdate, []spine.ClarificationAnswerAppliedMapping, error) {
	questions := make(map[spine.ClarificationQuestionID]spine.ClarificationQuestion, len(request.Questions))
	for _, question := range request.Questions {
		questions[question.ID] = question
	}

	var update spine.GoalHintUpdate
	mappings := make([]spine.ClarificationAnswerAppliedMapping, 0, len(answer.Answers))
	for _, item := range answer.Answers {
		question, ok := questions[item.QuestionID]
		if !ok {
			return spine.GoalHintUpdate{}, nil, &ValidationError{Field: "answers.question_id", Message: "unknown question_id"}
		}
		value := strings.TrimSpace(item.Value)
		if value == "" {
			return spine.GoalHintUpdate{}, nil, &ValidationError{Field: "answers.value", Message: "mapped value is required"}
		}

		mapping := spine.ClarificationAnswerAppliedMapping{
			QuestionID: item.QuestionID,
			MapsTo:     question.MapsTo,
			NewValue:   value,
		}
		switch question.MapsTo {
		case spine.ClarificationMapsToGoalSummary:
			mapping.OldValue = goal.Summary
			update.Summary = stringPtr(value)
		case spine.ClarificationMapsToGoalScopeHint:
			mapping.OldValue = goal.ScopeHint
			update.ScopeHint = stringPtr(value)
		case spine.ClarificationMapsToGoalAcceptanceHint:
			mapping.OldValue = goal.AcceptanceHint
			update.AcceptanceHint = stringPtr(value)
		case spine.ClarificationMapsToGoalIntentOwner:
			return spine.GoalHintUpdate{}, nil, fmt.Errorf("%w: %s requires actor-shaped value", ErrUnsupportedMapping, question.MapsTo)
		default:
			return spine.GoalHintUpdate{}, nil, fmt.Errorf("%w: %s", ErrUnsupportedMapping, question.MapsTo)
		}
		mappings = append(mappings, mapping)
	}
	if len(mappings) == 0 {
		return spine.GoalHintUpdate{}, nil, &ValidationError{Field: "answers", Message: "at least one applicable answer is required"}
	}

	return update, mappings, nil
}

func questionSpec(reason spine.GoalReadinessReasonCode) (clarificationQuestionSpec, bool) {
	switch reason {
	case spine.GoalReadinessReasonMissingGoalSummary:
		return clarificationQuestionSpec{
			text:      "What is the goal summary?",
			whyNeeded: "Goal summary is required before contract seed readiness.",
			mapsTo:    spine.ClarificationMapsToGoalSummary,
		}, true
	case spine.GoalReadinessReasonMissingIntentOwner:
		return clarificationQuestionSpec{
			text:      "Who owns the intent for this goal?",
			whyNeeded: "Intent owner is required before contract seed readiness.",
			mapsTo:    spine.ClarificationMapsToGoalIntentOwner,
		}, true
	case spine.GoalReadinessReasonMissingScopeHint:
		return clarificationQuestionSpec{
			text:      "What is the intended scope at a high level?",
			whyNeeded: "A scope hint is required before contract seed readiness.",
			mapsTo:    spine.ClarificationMapsToGoalScopeHint,
		}, true
	case spine.GoalReadinessReasonMissingAcceptanceHint:
		return clarificationQuestionSpec{
			text:      "What outcome would make this goal acceptable?",
			whyNeeded: "An acceptance hint is required before contract seed readiness.",
			mapsTo:    spine.ClarificationMapsToGoalAcceptanceHint,
		}, true
	default:
		return clarificationQuestionSpec{}, false
	}
}

func targetForGoal(goal spine.Goal) spine.ClarificationTarget {
	if actorRefPresent(goal.IntentOwner) {
		actor := goal.IntentOwner
		return spine.ClarificationTarget{
			Role:     spine.ClarificationTargetRoleIntentOwner,
			ActorRef: &actor,
		}
	}
	if actorRefPresent(goal.RequestAuthor) {
		actor := goal.RequestAuthor
		return spine.ClarificationTarget{
			Role:     spine.ClarificationTargetRoleRequestAuthor,
			ActorRef: &actor,
		}
	}
	return spine.ClarificationTarget{Role: spine.ClarificationTargetRoleRequestAuthor}
}

func actorRefPresent(actor spine.ActorRef) bool {
	return strings.TrimSpace(actor.Kind) != "" && strings.TrimSpace(actor.ID) != ""
}

func (s *Service) clarificationRequestedEvent(created spine.ClarificationRequest) (spine.Event, error) {
	eventID, err := s.IDs.NewEventID()
	if err != nil {
		return spine.Event{}, fmt.Errorf("new clarification requested event id: %w", err)
	}

	payload, err := json.Marshal(created)
	if err != nil {
		return spine.Event{}, fmt.Errorf("marshal clarification requested event payload: %w", err)
	}

	return spine.Event{
		ID:         eventID,
		Type:       EventTypeClarificationRequested,
		EntityType: EntityTypeClarificationRequest,
		EntityID:   string(created.ID),
		Timestamp:  created.CreatedAt,
		Payload:    payload,
	}, nil
}

func (s *Service) clarificationAnswerRecordedEvent(recorded spine.ClarificationAnswer) (spine.Event, error) {
	eventID, err := s.IDs.NewEventID()
	if err != nil {
		return spine.Event{}, fmt.Errorf("new clarification answer recorded event id: %w", err)
	}

	payload, err := json.Marshal(recorded)
	if err != nil {
		return spine.Event{}, fmt.Errorf("marshal clarification answer recorded event payload: %w", err)
	}

	return spine.Event{
		ID:         eventID,
		Type:       EventTypeClarificationAnswerRecorded,
		EntityType: EntityTypeClarificationAnswer,
		EntityID:   string(recorded.ID),
		Timestamp:  recorded.CreatedAt,
		Payload:    payload,
	}, nil
}

func (s *Service) clarificationRequestAnsweredEvent(requestID spine.ClarificationRequestID, answerID spine.ClarificationAnswerID, previousState spine.ClarificationRequestState, newState spine.ClarificationRequestState, timestamp time.Time) (spine.Event, error) {
	eventID, err := s.IDs.NewEventID()
	if err != nil {
		return spine.Event{}, fmt.Errorf("new clarification request answered event id: %w", err)
	}

	payload, err := json.Marshal(clarificationRequestAnsweredPayload{
		RequestID:     requestID,
		AnswerID:      answerID,
		PreviousState: previousState,
		NewState:      newState,
	})
	if err != nil {
		return spine.Event{}, fmt.Errorf("marshal clarification request answered event payload: %w", err)
	}

	return spine.Event{
		ID:         eventID,
		Type:       EventTypeClarificationRequestAnswered,
		EntityType: EntityTypeClarificationRequest,
		EntityID:   string(requestID),
		Timestamp:  timestamp,
		Payload:    payload,
	}, nil
}

func (s *Service) clarificationAnswerAppliedEvent(result spine.ClarificationAnswerApplicationResult) (spine.Event, error) {
	eventID, err := s.IDs.NewEventID()
	if err != nil {
		return spine.Event{}, fmt.Errorf("new clarification answer applied event id: %w", err)
	}

	payload, err := json.Marshal(result)
	if err != nil {
		return spine.Event{}, fmt.Errorf("marshal clarification answer applied event payload: %w", err)
	}

	return spine.Event{
		ID:         eventID,
		Type:       EventTypeClarificationAnswerApplied,
		EntityType: EntityTypeClarificationAnswer,
		EntityID:   string(result.AnswerID),
		Timestamp:  result.AppliedAt,
		Payload:    payload,
	}, nil
}

func (s *Service) goalHintsUpdatedEvent(result spine.ClarificationAnswerApplicationResult) (spine.Event, error) {
	eventID, err := s.IDs.NewEventID()
	if err != nil {
		return spine.Event{}, fmt.Errorf("new goal hints updated event id: %w", err)
	}

	payload, err := json.Marshal(goalHintsUpdatedPayload{
		AnswerID:        result.AnswerID,
		GoalID:          result.GoalID,
		AppliedBy:       result.AppliedBy,
		AppliedMappings: result.AppliedMappings,
		AppliedAt:       result.AppliedAt,
	})
	if err != nil {
		return spine.Event{}, fmt.Errorf("marshal goal hints updated event payload: %w", err)
	}

	return spine.Event{
		ID:         eventID,
		Type:       EventTypeGoalHintsUpdated,
		EntityType: EntityTypeGoal,
		EntityID:   string(result.GoalID),
		Timestamp:  result.AppliedAt,
		Payload:    payload,
	}, nil
}

type clarificationRequestAnsweredPayload struct {
	RequestID     spine.ClarificationRequestID    `json:"request_id"`
	AnswerID      spine.ClarificationAnswerID     `json:"answer_id"`
	PreviousState spine.ClarificationRequestState `json:"previous_state"`
	NewState      spine.ClarificationRequestState `json:"new_state"`
}

type goalHintsUpdatedPayload struct {
	AnswerID        spine.ClarificationAnswerID               `json:"answer_id"`
	GoalID          spine.GoalID                              `json:"goal_id"`
	AppliedBy       spine.ActorRef                            `json:"applied_by"`
	AppliedMappings []spine.ClarificationAnswerAppliedMapping `json:"applied_mappings"`
	AppliedAt       time.Time                                 `json:"applied_at"`
}

func (s *Service) validateDependencies() error {
	if s.Goals == nil {
		return errors.New("clarification service goal reader is nil")
	}
	if s.Store == nil {
		return errors.New("clarification service store is nil")
	}
	if s.Answers == nil {
		return errors.New("clarification service answer store is nil")
	}
	if s.Events == nil {
		return errors.New("clarification service event log is nil")
	}
	if s.Clock == nil {
		return errors.New("clarification service clock is nil")
	}
	if s.IDs == nil {
		return errors.New("clarification service id generator is nil")
	}
	return nil
}

func cloneReasonCodes(reasons []spine.GoalReadinessReasonCode) []spine.GoalReadinessReasonCode {
	if reasons == nil {
		return nil
	}
	return append([]spine.GoalReadinessReasonCode(nil), reasons...)
}

func cloneAnswerItems(answers []spine.ClarificationAnswerItem) []spine.ClarificationAnswerItem {
	if answers == nil {
		return nil
	}
	return append([]spine.ClarificationAnswerItem(nil), answers...)
}

func stringPtr(value string) *string {
	return &value
}

type clarificationQuestionSpec struct {
	text      string
	whyNeeded string
	mapsTo    spine.ClarificationMapsTo
}

type SystemClock struct{}

func (SystemClock) Now() time.Time {
	return time.Now()
}

type UUIDGenerator struct{}

func (UUIDGenerator) NewClarificationRequestID() (spine.ClarificationRequestID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return spine.ClarificationRequestID(id.String()), nil
}

func (UUIDGenerator) NewClarificationQuestionID() (spine.ClarificationQuestionID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return spine.ClarificationQuestionID(id.String()), nil
}

func (UUIDGenerator) NewClarificationAnswerID() (spine.ClarificationAnswerID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return spine.ClarificationAnswerID(id.String()), nil
}

func (UUIDGenerator) NewEventID() (spine.EventID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return spine.EventID(id.String()), nil
}
