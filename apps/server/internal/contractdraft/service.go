package contractdraft

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
	EventTypeContractDraftCreated = "contract_draft.created"
	EntityTypeContractDraft       = "ContractDraft"
	SourceRefKindContractSeed     = "contract_seed"
	DefaultProofExpectation       = "Provide evidence that acceptance criteria were checked."
)

var (
	ErrContractSeedNotFound = errors.New("contract seed not found")
	ErrInvalidSeedState     = errors.New("contract seed state is not draftable")
	ErrAlreadyDrafted       = errors.New("contract seed already has contract draft")
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

type SeedReader interface {
	Get(context.Context, spine.ContractSeedID) (spine.ContractSeed, bool, error)
}

type Store interface {
	Create(context.Context, spine.ContractDraft) error
	Get(context.Context, spine.ContractDraftID) (spine.ContractDraft, bool, error)
	GetByContractSeedID(context.Context, spine.ContractSeedID) (spine.ContractDraft, bool, error)
}

type EventLog interface {
	Append(context.Context, spine.Event) error
}

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	NewContractDraftID() (spine.ContractDraftID, error)
	NewEventID() (spine.EventID, error)
}

type Service struct {
	Seeds  SeedReader
	Drafts Store
	Events EventLog
	Clock  Clock
	IDs    IDGenerator
}

func NewService(seeds SeedReader, drafts Store, events EventLog, clock Clock, ids IDGenerator) *Service {
	return &Service{
		Seeds:  seeds,
		Drafts: drafts,
		Events: events,
		Clock:  clock,
		IDs:    ids,
	}
}

func (s *Service) Create(ctx context.Context, seedID spine.ContractSeedID) (spine.ContractDraft, error) {
	if err := s.validateDependencies(); err != nil {
		return spine.ContractDraft{}, err
	}

	seed, ok, err := s.Seeds.Get(ctx, seedID)
	if err != nil {
		return spine.ContractDraft{}, fmt.Errorf("get contract seed: %w", err)
	}
	if !ok {
		return spine.ContractDraft{}, ErrContractSeedNotFound
	}
	if seed.State != spine.ContractSeedStateCreated {
		return spine.ContractDraft{}, fmt.Errorf("%w: %s", ErrInvalidSeedState, seed.State)
	}
	if _, ok, err := s.Drafts.GetByContractSeedID(ctx, seed.ID); err != nil {
		return spine.ContractDraft{}, fmt.Errorf("get contract draft by contract seed id: %w", err)
	} else if ok {
		return spine.ContractDraft{}, ErrAlreadyDrafted
	}
	if err := validateSeedForDraft(seed); err != nil {
		return spine.ContractDraft{}, err
	}

	draftID, err := s.IDs.NewContractDraftID()
	if err != nil {
		return spine.ContractDraft{}, fmt.Errorf("new contract draft id: %w", err)
	}
	now := s.Clock.Now().UTC()
	created := spine.ContractDraft{
		ID:                         draftID,
		ContractSeedID:             seed.ID,
		GoalID:                     seed.GoalID,
		RepoBindingID:              seed.RepoBindingID,
		Title:                      seed.Title,
		IntentSummary:              seed.IntentSummary,
		ProposedScope:              []string{seed.ScopeHint},
		ProposedNonGoals:           []string{},
		ProposedConstraints:        []string{},
		ProposedAcceptanceCriteria: []string{seed.AcceptanceHint},
		ProposedExpectedChecks:     []string{},
		ProposedProofExpectations:  []string{DefaultProofExpectation},
		RiskHints:                  []string{},
		SourceRefs:                 sourceRefsForSeed(seed),
		State:                      spine.ContractDraftStateDraft,
		CreatedAt:                  now,
	}

	event, err := s.contractDraftCreatedEvent(created)
	if err != nil {
		return spine.ContractDraft{}, err
	}
	if err := s.Drafts.Create(ctx, created); err != nil {
		if _, ok, lookupErr := s.Drafts.GetByContractSeedID(ctx, seed.ID); lookupErr != nil {
			return spine.ContractDraft{}, fmt.Errorf("get contract draft by contract seed id after create failure: %w", lookupErr)
		} else if ok {
			return spine.ContractDraft{}, ErrAlreadyDrafted
		}
		return spine.ContractDraft{}, fmt.Errorf("create contract draft: %w", err)
	}
	if err := s.Events.Append(ctx, event); err != nil {
		return spine.ContractDraft{}, fmt.Errorf("append contract draft created event: %w", err)
	}

	return created, nil
}

func validateSeedForDraft(seed spine.ContractSeed) error {
	if strings.TrimSpace(string(seed.GoalID)) == "" {
		return &ValidationError{Field: "goal_id", Message: "is required"}
	}
	if strings.TrimSpace(string(seed.RepoBindingID)) == "" {
		return &ValidationError{Field: "repo_binding_id", Message: "is required"}
	}
	if strings.TrimSpace(seed.Title) == "" {
		return &ValidationError{Field: "title", Message: "is required"}
	}
	if strings.TrimSpace(seed.IntentSummary) == "" {
		return &ValidationError{Field: "intent_summary", Message: "is required"}
	}
	if strings.TrimSpace(seed.IntentOwner.Kind) == "" {
		return &ValidationError{Field: "intent_owner.kind", Message: "is required"}
	}
	if strings.TrimSpace(seed.IntentOwner.ID) == "" {
		return &ValidationError{Field: "intent_owner.id", Message: "is required"}
	}
	if strings.TrimSpace(seed.ScopeHint) == "" {
		return &ValidationError{Field: "scope_hint", Message: "is required"}
	}
	if strings.TrimSpace(seed.AcceptanceHint) == "" {
		return &ValidationError{Field: "acceptance_hint", Message: "is required"}
	}
	return nil
}

func sourceRefsForSeed(seed spine.ContractSeed) []spine.SourceRef {
	refs := make([]spine.SourceRef, 0, len(seed.SourceRefs)+1)
	refs = append(refs, spine.SourceRef{Kind: SourceRefKindContractSeed, ID: string(seed.ID)})
	for _, ref := range seed.SourceRefs {
		if strings.TrimSpace(ref.Kind) == "" || strings.TrimSpace(ref.ID) == "" {
			continue
		}
		refs = append(refs, ref)
	}
	return refs
}

func (s *Service) contractDraftCreatedEvent(created spine.ContractDraft) (spine.Event, error) {
	eventID, err := s.IDs.NewEventID()
	if err != nil {
		return spine.Event{}, fmt.Errorf("new contract draft created event id: %w", err)
	}

	payload, err := json.Marshal(created)
	if err != nil {
		return spine.Event{}, fmt.Errorf("marshal contract draft created event payload: %w", err)
	}

	return spine.Event{
		ID:            eventID,
		Type:          EventTypeContractDraftCreated,
		EntityType:    EntityTypeContractDraft,
		EntityID:      string(created.ID),
		RepoBindingID: created.RepoBindingID,
		Timestamp:     created.CreatedAt,
		Payload:       payload,
	}, nil
}

func (s *Service) validateDependencies() error {
	if s.Seeds == nil {
		return errors.New("contract draft service seed store is nil")
	}
	if s.Drafts == nil {
		return errors.New("contract draft service draft store is nil")
	}
	if s.Events == nil {
		return errors.New("contract draft service event log is nil")
	}
	if s.Clock == nil {
		return errors.New("contract draft service clock is nil")
	}
	if s.IDs == nil {
		return errors.New("contract draft service id generator is nil")
	}
	return nil
}

type SystemClock struct{}

func (SystemClock) Now() time.Time {
	return time.Now()
}

type UUIDGenerator struct{}

func (UUIDGenerator) NewContractDraftID() (spine.ContractDraftID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return spine.ContractDraftID(id.String()), nil
}

func (UUIDGenerator) NewEventID() (spine.EventID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return spine.EventID(id.String()), nil
}
