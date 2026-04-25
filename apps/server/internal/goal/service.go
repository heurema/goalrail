package goal

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
	EventTypeGoalCreated                    = "goal.created"
	EventTypeGoalReadinessChecked           = "goal.readiness_checked"
	EventTypeGoalMarkedNeedsClarification   = "goal.marked_needs_clarification"
	EventTypeGoalMarkedReadyForContractSeed = "goal.marked_ready_for_contract_seed"
	EventTypeGoalRejected                   = "goal.rejected"
	EventTypeIntakePromoted                 = "intake.promoted_to_goal"
	EntityTypeGoal                          = "Goal"
	EntityTypeIntake                        = "IntakeRecord"
	SourceRefKindIntake                     = "intake"
)

var (
	ErrIntakeNotFound     = errors.New("intake record not found")
	ErrGoalNotFound       = errors.New("goal not found")
	ErrInvalidIntakeState = errors.New("intake record state is not promotable")
	ErrInvalidGoalState   = errors.New("goal state is not readiness-checkable")
	ErrAlreadyPromoted    = errors.New("intake record already promoted to goal")
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

type IntakeReader interface {
	Get(context.Context, spine.IntakeID) (spine.IntakeRecord, bool, error)
}

type GoalStore interface {
	Create(context.Context, spine.Goal) error
	Get(context.Context, spine.GoalID) (spine.Goal, bool, error)
	GetByIntakeID(context.Context, spine.IntakeID) (spine.Goal, bool, error)
	UpdateState(context.Context, spine.GoalID, spine.GoalState) (spine.Goal, bool, error)
}

type EventLog interface {
	Append(context.Context, spine.Event) error
}

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	NewGoalID() (spine.GoalID, error)
	NewEventID() (spine.EventID, error)
}

type Service struct {
	Intake IntakeReader
	Goals  GoalStore
	Events EventLog
	Clock  Clock
	IDs    IDGenerator
}

func NewService(intake IntakeReader, goals GoalStore, events EventLog, clock Clock, ids IDGenerator) *Service {
	return &Service{
		Intake: intake,
		Goals:  goals,
		Events: events,
		Clock:  clock,
		IDs:    ids,
	}
}

func (s *Service) PromoteFromIntake(ctx context.Context, intakeID spine.IntakeID) (spine.Goal, error) {
	if err := s.validateDependencies(); err != nil {
		return spine.Goal{}, err
	}

	record, ok, err := s.Intake.Get(ctx, intakeID)
	if err != nil {
		return spine.Goal{}, fmt.Errorf("get intake record: %w", err)
	}
	if !ok {
		return spine.Goal{}, ErrIntakeNotFound
	}
	if record.State != spine.IntakeStateReceived {
		return spine.Goal{}, fmt.Errorf("%w: %s", ErrInvalidIntakeState, record.State)
	}
	if err := ValidateIntakeForPromotion(record); err != nil {
		return spine.Goal{}, err
	}
	if _, ok, err := s.Goals.GetByIntakeID(ctx, record.ID); err != nil {
		return spine.Goal{}, fmt.Errorf("get goal by intake id: %w", err)
	} else if ok {
		return spine.Goal{}, ErrAlreadyPromoted
	}

	now := s.Clock.Now().UTC()
	goalID, err := s.IDs.NewGoalID()
	if err != nil {
		return spine.Goal{}, fmt.Errorf("new goal id: %w", err)
	}

	created := spine.Goal{
		ID:            goalID,
		IntakeID:      record.ID,
		RepoBindingID: record.RepoBindingID,
		Title:         record.Title,
		Summary:       goalSummary(record),
		SourceRefs: []spine.SourceRef{
			{Kind: SourceRefKindIntake, ID: string(record.ID)},
		},
		RequestAuthor: record.RequestAuthor,
		IntentOwner:   record.IntentOwner,
		State:         spine.GoalStateCreated,
		CreatedAt:     now,
	}

	goalCreated, err := s.goalCreatedEvent(created, now)
	if err != nil {
		return spine.Goal{}, err
	}
	intakePromoted, err := s.intakePromotedEvent(record.ID, created.ID, now)
	if err != nil {
		return spine.Goal{}, err
	}

	if err := s.Goals.Create(ctx, created); err != nil {
		return spine.Goal{}, fmt.Errorf("create goal: %w", err)
	}
	if err := s.Events.Append(ctx, goalCreated); err != nil {
		return spine.Goal{}, fmt.Errorf("append goal created event: %w", err)
	}
	if err := s.Events.Append(ctx, intakePromoted); err != nil {
		return spine.Goal{}, fmt.Errorf("append intake promoted event: %w", err)
	}

	return created, nil
}

func (s *Service) CheckReadiness(ctx context.Context, goalID spine.GoalID) (spine.GoalReadinessResult, spine.Goal, error) {
	if err := s.validateDependencies(); err != nil {
		return spine.GoalReadinessResult{}, spine.Goal{}, err
	}

	current, ok, err := s.Goals.Get(ctx, goalID)
	if err != nil {
		return spine.GoalReadinessResult{}, spine.Goal{}, fmt.Errorf("get goal: %w", err)
	}
	if !ok {
		return spine.GoalReadinessResult{}, spine.Goal{}, ErrGoalNotFound
	}
	if !isReadinessCheckable(current.State) {
		return spine.GoalReadinessResult{}, spine.Goal{}, fmt.Errorf("%w: %s", ErrInvalidGoalState, current.State)
	}

	now := s.Clock.Now().UTC()
	result := evaluateReadiness(current, now)
	updated, ok, err := s.Goals.UpdateState(ctx, current.ID, result.State)
	if err != nil {
		return spine.GoalReadinessResult{}, spine.Goal{}, fmt.Errorf("update goal state: %w", err)
	}
	if !ok {
		return spine.GoalReadinessResult{}, spine.Goal{}, ErrGoalNotFound
	}

	readinessChecked, err := s.goalReadinessCheckedEvent(result, current.State, updated.State, now)
	if err != nil {
		return spine.GoalReadinessResult{}, spine.Goal{}, err
	}
	transition, err := s.goalReadinessTransitionEvent(result, current.State, updated.State, now)
	if err != nil {
		return spine.GoalReadinessResult{}, spine.Goal{}, err
	}

	if err := s.Events.Append(ctx, readinessChecked); err != nil {
		return spine.GoalReadinessResult{}, spine.Goal{}, fmt.Errorf("append goal readiness checked event: %w", err)
	}
	if err := s.Events.Append(ctx, transition); err != nil {
		return spine.GoalReadinessResult{}, spine.Goal{}, fmt.Errorf("append goal readiness transition event: %w", err)
	}

	return result, updated, nil
}

func isReadinessCheckable(state spine.GoalState) bool {
	return state == spine.GoalStateCreated ||
		state == spine.GoalStateNeedsClarification ||
		state == spine.GoalStateReadyForContractSeed
}

func evaluateReadiness(goal spine.Goal, checkedAt time.Time) spine.GoalReadinessResult {
	var reasons []spine.GoalReadinessReasonCode
	if strings.TrimSpace(goal.Summary) == "" {
		reasons = append(reasons, spine.GoalReadinessReasonMissingGoalSummary)
	}
	if strings.TrimSpace(goal.IntentOwner.Kind) == "" || strings.TrimSpace(goal.IntentOwner.ID) == "" {
		reasons = append(reasons, spine.GoalReadinessReasonMissingIntentOwner)
	}
	if strings.TrimSpace(goal.ScopeHint) == "" {
		reasons = append(reasons, spine.GoalReadinessReasonMissingScopeHint)
	}
	if strings.TrimSpace(goal.AcceptanceHint) == "" {
		reasons = append(reasons, spine.GoalReadinessReasonMissingAcceptanceHint)
	}

	result := spine.GoalReadinessResult{
		GoalID:      goal.ID,
		State:       spine.GoalStateNeedsClarification,
		Ready:       false,
		ReasonCodes: reasons,
		Message:     "goal needs clarification before contract seed",
		CheckedAt:   checkedAt,
	}
	if len(reasons) == 0 {
		result.State = spine.GoalStateReadyForContractSeed
		result.Ready = true
		result.ReasonCodes = []spine.GoalReadinessReasonCode{}
		result.Message = "goal is ready for contract seed"
	}
	return result
}

func goalSummary(record spine.IntakeRecord) string {
	if strings.TrimSpace(record.Body) != "" {
		return record.Body
	}
	return record.Title
}

func ValidateIntakeForPromotion(record spine.IntakeRecord) error {
	if strings.TrimSpace(string(record.RepoBindingID)) == "" {
		return &ValidationError{Field: "repo_binding_id", Message: "is required"}
	}
	if strings.TrimSpace(record.Title) == "" && strings.TrimSpace(record.Body) == "" {
		return &ValidationError{Field: "title", Message: "title or body is required"}
	}
	if strings.TrimSpace(record.RequestAuthor.Kind) == "" {
		return &ValidationError{Field: "request_author.kind", Message: "is required"}
	}
	if strings.TrimSpace(record.RequestAuthor.ID) == "" {
		return &ValidationError{Field: "request_author.id", Message: "is required"}
	}
	return nil
}

func (s *Service) goalCreatedEvent(created spine.Goal, timestamp time.Time) (spine.Event, error) {
	eventID, err := s.IDs.NewEventID()
	if err != nil {
		return spine.Event{}, fmt.Errorf("new goal created event id: %w", err)
	}

	payload, err := json.Marshal(created)
	if err != nil {
		return spine.Event{}, fmt.Errorf("marshal goal created event payload: %w", err)
	}

	return spine.Event{
		ID:         eventID,
		Type:       EventTypeGoalCreated,
		EntityType: EntityTypeGoal,
		EntityID:   string(created.ID),
		Timestamp:  timestamp,
		Payload:    payload,
	}, nil
}

func (s *Service) intakePromotedEvent(intakeID spine.IntakeID, goalID spine.GoalID, timestamp time.Time) (spine.Event, error) {
	eventID, err := s.IDs.NewEventID()
	if err != nil {
		return spine.Event{}, fmt.Errorf("new intake promoted event id: %w", err)
	}

	payload, err := json.Marshal(intakePromotedPayload{
		IntakeID: intakeID,
		GoalID:   goalID,
	})
	if err != nil {
		return spine.Event{}, fmt.Errorf("marshal intake promoted event payload: %w", err)
	}

	return spine.Event{
		ID:         eventID,
		Type:       EventTypeIntakePromoted,
		EntityType: EntityTypeIntake,
		EntityID:   string(intakeID),
		Timestamp:  timestamp,
		Payload:    payload,
	}, nil
}

func (s *Service) goalReadinessCheckedEvent(result spine.GoalReadinessResult, previousState spine.GoalState, newState spine.GoalState, timestamp time.Time) (spine.Event, error) {
	eventID, err := s.IDs.NewEventID()
	if err != nil {
		return spine.Event{}, fmt.Errorf("new goal readiness checked event id: %w", err)
	}

	payload, err := json.Marshal(goalReadinessPayload{
		Result:        result,
		PreviousState: previousState,
		NewState:      newState,
	})
	if err != nil {
		return spine.Event{}, fmt.Errorf("marshal goal readiness checked event payload: %w", err)
	}

	return spine.Event{
		ID:         eventID,
		Type:       EventTypeGoalReadinessChecked,
		EntityType: EntityTypeGoal,
		EntityID:   string(result.GoalID),
		Timestamp:  timestamp,
		Payload:    payload,
	}, nil
}

func (s *Service) goalReadinessTransitionEvent(result spine.GoalReadinessResult, previousState spine.GoalState, newState spine.GoalState, timestamp time.Time) (spine.Event, error) {
	eventID, err := s.IDs.NewEventID()
	if err != nil {
		return spine.Event{}, fmt.Errorf("new goal readiness transition event id: %w", err)
	}

	eventType, err := transitionEventType(newState)
	if err != nil {
		return spine.Event{}, err
	}

	payload, err := json.Marshal(goalReadinessPayload{
		Result:        result,
		PreviousState: previousState,
		NewState:      newState,
	})
	if err != nil {
		return spine.Event{}, fmt.Errorf("marshal goal readiness transition event payload: %w", err)
	}

	return spine.Event{
		ID:         eventID,
		Type:       eventType,
		EntityType: EntityTypeGoal,
		EntityID:   string(result.GoalID),
		Timestamp:  timestamp,
		Payload:    payload,
	}, nil
}

func transitionEventType(state spine.GoalState) (string, error) {
	switch state {
	case spine.GoalStateNeedsClarification:
		return EventTypeGoalMarkedNeedsClarification, nil
	case spine.GoalStateReadyForContractSeed:
		return EventTypeGoalMarkedReadyForContractSeed, nil
	case spine.GoalStateRejected:
		return EventTypeGoalRejected, nil
	default:
		return "", fmt.Errorf("unsupported readiness transition state: %s", state)
	}
}

type intakePromotedPayload struct {
	IntakeID spine.IntakeID `json:"intake_id"`
	GoalID   spine.GoalID   `json:"goal_id"`
}

type goalReadinessPayload struct {
	Result        spine.GoalReadinessResult `json:"result"`
	PreviousState spine.GoalState           `json:"previous_state"`
	NewState      spine.GoalState           `json:"new_state"`
}

func (s *Service) validateDependencies() error {
	if s.Intake == nil {
		return errors.New("goal service intake reader is nil")
	}
	if s.Goals == nil {
		return errors.New("goal service goal store is nil")
	}
	if s.Events == nil {
		return errors.New("goal service event log is nil")
	}
	if s.Clock == nil {
		return errors.New("goal service clock is nil")
	}
	if s.IDs == nil {
		return errors.New("goal service id generator is nil")
	}
	return nil
}

type SystemClock struct{}

func (SystemClock) Now() time.Time {
	return time.Now()
}

type UUIDGenerator struct{}

func (UUIDGenerator) NewGoalID() (spine.GoalID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return spine.GoalID(id.String()), nil
}

func (UUIDGenerator) NewEventID() (spine.EventID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return spine.EventID(id.String()), nil
}
