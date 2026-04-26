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
	EventTypeClarificationRequested = "clarification.requested"
	EntityTypeClarificationRequest  = "ClarificationRequest"
)

var (
	ErrGoalNotFound             = errors.New("goal not found")
	ErrInvalidGoalState         = errors.New("goal state is not clarification-requestable")
	ErrAlreadyOpen              = errors.New("clarification request already open")
	ErrMissingReadinessReasons  = errors.New("goal has no stored readiness reason codes")
	ErrNoClarificationQuestions = errors.New("no clarification questions available")
	ErrPolicyRejected           = errors.New("policy rejected goals cannot create clarification request")
)

type GoalReader interface {
	Get(context.Context, spine.GoalID) (spine.Goal, bool, error)
}

type Store interface {
	Create(context.Context, spine.ClarificationRequest) error
	GetOpenByGoalID(context.Context, spine.GoalID) (spine.ClarificationRequest, bool, error)
}

type EventLog interface {
	Append(context.Context, spine.Event) error
}

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	NewClarificationRequestID() (spine.ClarificationRequestID, error)
	NewClarificationQuestionID() (spine.ClarificationQuestionID, error)
	NewEventID() (spine.EventID, error)
}

type Service struct {
	Goals  GoalReader
	Store  Store
	Events EventLog
	Clock  Clock
	IDs    IDGenerator
}

func NewService(goals GoalReader, store Store, events EventLog, clock Clock, ids IDGenerator) *Service {
	return &Service{
		Goals:  goals,
		Store:  store,
		Events: events,
		Clock:  clock,
		IDs:    ids,
	}
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
	if err := s.Store.Create(ctx, created); err != nil {
		if errors.Is(err, ErrAlreadyOpen) {
			return spine.ClarificationRequest{}, ErrAlreadyOpen
		}
		return spine.ClarificationRequest{}, fmt.Errorf("create clarification request: %w", err)
	}
	if err := s.Events.Append(ctx, event); err != nil {
		return spine.ClarificationRequest{}, fmt.Errorf("append clarification requested event: %w", err)
	}

	return created, nil
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

func (s *Service) validateDependencies() error {
	if s.Goals == nil {
		return errors.New("clarification service goal reader is nil")
	}
	if s.Store == nil {
		return errors.New("clarification service store is nil")
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

func (UUIDGenerator) NewEventID() (spine.EventID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return spine.EventID(id.String()), nil
}
