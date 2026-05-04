package contractseed

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
	EventTypeContractSeedCreated = "contract_seed.created"
	EntityTypeContractSeed       = "ContractSeed"
	SourceRefKindGoal            = "goal"
)

var (
	ErrGoalNotFound     = errors.New("goal not found")
	ErrInvalidGoalState = errors.New("goal state is not seedable")
	ErrAlreadySeeded    = errors.New("goal already has contract seed")
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

type Store interface {
	Create(context.Context, spine.ContractSeed) error
	Get(context.Context, spine.ContractSeedID) (spine.ContractSeed, bool, error)
	GetByGoalID(context.Context, spine.GoalID) (spine.ContractSeed, bool, error)
}

type ContractStore interface {
	Create(context.Context, spine.Contract) error
	Get(context.Context, spine.ContractID) (spine.Contract, bool, error)
	GetByGoalID(context.Context, spine.GoalID) (spine.Contract, bool, error)
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
	NewContractID() (spine.ContractID, error)
	NewContractSeedID() (spine.ContractSeedID, error)
	NewEventID() (spine.EventID, error)
}

type Service struct {
	Goals     GoalReader
	Contracts ContractStore
	Seeds     Store
	Events    EventLog
	TxRunner  TransactionRunner
	Clock     Clock
	IDs       IDGenerator
}

func NewService(goals GoalReader, contracts ContractStore, seeds Store, events EventLog, txRunner TransactionRunner, clock Clock, ids IDGenerator) *Service {
	return &Service{
		Goals:     goals,
		Contracts: contracts,
		Seeds:     seeds,
		Events:    events,
		TxRunner:  txRunner,
		Clock:     clock,
		IDs:       ids,
	}
}

func (s *Service) Create(ctx context.Context, goalID spine.GoalID) (spine.ContractSeed, error) {
	goal, ok, err := s.Goals.Get(ctx, goalID)
	if err != nil {
		return spine.ContractSeed{}, fmt.Errorf("get goal: %w", err)
	}
	if !ok {
		return spine.ContractSeed{}, ErrGoalNotFound
	}
	if goal.State != spine.GoalStateReadyForContractSeed {
		return spine.ContractSeed{}, fmt.Errorf("%w: %s", ErrInvalidGoalState, goal.State)
	}
	if _, ok, err := s.Seeds.GetByGoalID(ctx, goal.ID); err != nil {
		return spine.ContractSeed{}, fmt.Errorf("get contract seed by goal id: %w", err)
	} else if ok {
		return spine.ContractSeed{}, ErrAlreadySeeded
	}
	if _, ok, err := s.Contracts.GetByGoalID(ctx, goal.ID); err != nil {
		return spine.ContractSeed{}, fmt.Errorf("get contract by goal id: %w", err)
	} else if ok {
		return spine.ContractSeed{}, ErrAlreadySeeded
	}
	if err := validateGoalForSeed(goal); err != nil {
		return spine.ContractSeed{}, err
	}

	contractID, err := s.IDs.NewContractID()
	if err != nil {
		return spine.ContractSeed{}, fmt.Errorf("new contract id: %w", err)
	}
	seedID, err := s.IDs.NewContractSeedID()
	if err != nil {
		return spine.ContractSeed{}, fmt.Errorf("new contract seed id: %w", err)
	}
	now := s.Clock.Now().UTC()
	currentSeedID := seedID
	contract := spine.Contract{
		ID:             contractID,
		OrganizationID: goal.OrganizationID,
		ProjectID:      goal.ProjectID,
		RepoBindingID:  goal.RepoBindingID,
		GoalID:         goal.ID,
		State:          spine.ContractStateSeeded,
		CurrentSeedID:  &currentSeedID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	created := spine.ContractSeed{
		ID:             seedID,
		OrganizationID: goal.OrganizationID,
		ProjectID:      goal.ProjectID,
		ContractID:     contractID,
		GoalID:         goal.ID,
		RepoBindingID:  goal.RepoBindingID,
		Title:          goal.Title,
		IntentSummary:  goal.Summary,
		IntentOwner:    goal.IntentOwner,
		ScopeHint:      goal.ScopeHint,
		AcceptanceHint: goal.AcceptanceHint,
		SourceRefs:     sourceRefsForGoal(goal),
		State:          spine.ContractSeedStateCreated,
		CreatedAt:      now,
	}

	event, err := s.contractSeedCreatedEvent(created, goal)
	if err != nil {
		return spine.ContractSeed{}, err
	}
	if err := s.TxRunner.RunReadCommitted(ctx, func(txCtx context.Context) error {
		if err := s.Contracts.Create(txCtx, contract); err != nil {
			if _, ok, lookupErr := s.Contracts.GetByGoalID(txCtx, goal.ID); lookupErr != nil {
				return fmt.Errorf("get contract by goal id after create failure: %w", lookupErr)
			} else if ok {
				return ErrAlreadySeeded
			}
			return fmt.Errorf("create contract: %w", err)
		}
		if err := s.Seeds.Create(txCtx, created); err != nil {
			if _, ok, lookupErr := s.Seeds.GetByGoalID(txCtx, goal.ID); lookupErr != nil {
				return fmt.Errorf("get contract seed by goal id after create failure: %w", lookupErr)
			} else if ok {
				return ErrAlreadySeeded
			}
			return fmt.Errorf("create contract seed: %w", err)
		}
		if err := s.Events.Append(txCtx, event); err != nil {
			return fmt.Errorf("append contract seed created event: %w", err)
		}
		return nil
	}); err != nil {
		return spine.ContractSeed{}, err
	}

	return created, nil
}

func validateGoalForSeed(goal spine.Goal) error {
	if strings.TrimSpace(string(goal.RepoBindingID)) == "" {
		return &ValidationError{Field: "repo_binding_id", Message: "is required"}
	}
	if strings.TrimSpace(goal.Title) == "" {
		return &ValidationError{Field: "title", Message: "is required"}
	}
	if strings.TrimSpace(goal.Summary) == "" {
		return &ValidationError{Field: "summary", Message: "is required"}
	}
	if strings.TrimSpace(goal.IntentOwner.Kind) == "" {
		return &ValidationError{Field: "intent_owner.kind", Message: "is required"}
	}
	if strings.TrimSpace(goal.IntentOwner.ID) == "" {
		return &ValidationError{Field: "intent_owner.id", Message: "is required"}
	}
	if strings.TrimSpace(goal.ScopeHint) == "" {
		return &ValidationError{Field: "scope_hint", Message: "is required"}
	}
	if strings.TrimSpace(goal.AcceptanceHint) == "" {
		return &ValidationError{Field: "acceptance_hint", Message: "is required"}
	}
	return nil
}

func sourceRefsForGoal(goal spine.Goal) []spine.SourceRef {
	refs := make([]spine.SourceRef, 0, len(goal.SourceRefs)+1)
	refs = append(refs, spine.SourceRef{Kind: SourceRefKindGoal, ID: string(goal.ID)})
	for _, ref := range goal.SourceRefs {
		if strings.TrimSpace(ref.Kind) == "" || strings.TrimSpace(ref.ID) == "" {
			continue
		}
		refs = append(refs, ref)
	}
	return refs
}

func (s *Service) contractSeedCreatedEvent(created spine.ContractSeed, goal spine.Goal) (spine.Event, error) {
	eventID, err := s.IDs.NewEventID()
	if err != nil {
		return spine.Event{}, fmt.Errorf("new contract seed created event id: %w", err)
	}

	payload, err := json.Marshal(created)
	if err != nil {
		return spine.Event{}, fmt.Errorf("marshal contract seed created event payload: %w", err)
	}

	return spine.Event{
		ID:             eventID,
		Type:           EventTypeContractSeedCreated,
		EntityType:     EntityTypeContractSeed,
		EntityID:       string(created.ID),
		OrganizationID: goal.OrganizationID,
		ProjectID:      goal.ProjectID,
		RepoBindingID:  goal.RepoBindingID,
		Timestamp:      created.CreatedAt,
		Payload:        payload,
	}, nil
}

type SystemClock struct{}

func (SystemClock) Now() time.Time {
	return time.Now()
}

type UUIDGenerator struct{}

func (UUIDGenerator) NewContractID() (spine.ContractID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return spine.ContractID(id.String()), nil
}

func (UUIDGenerator) NewContractSeedID() (spine.ContractSeedID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return spine.ContractSeedID(id.String()), nil
}

func (UUIDGenerator) NewEventID() (spine.EventID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return spine.EventID(id.String()), nil
}
