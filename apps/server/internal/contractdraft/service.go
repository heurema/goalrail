package contractdraft

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/heurema/goalrail/apps/server/internal/spine"
)

const (
	EventTypeContractDraftCreated = "contract_draft.created"
	EventTypeContractDraftUpdated = "contract_draft.updated"
	EntityTypeContractDraft       = "ContractDraft"
	SourceRefKindContractSeed     = "contract_seed"
	DefaultProofExpectation       = "Provide evidence that acceptance criteria were checked."
)

var (
	ErrContractSeedNotFound  = errors.New("contract seed not found")
	ErrContractDraftNotFound = errors.New("contract draft not found")
	ErrInvalidSeedState      = errors.New("contract seed state is not draftable")
	ErrInvalidDraftState     = errors.New("contract draft state is not updateable")
	ErrAlreadyDrafted        = errors.New("contract seed already has contract draft")
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

type UnknownFieldError struct {
	Field string
}

func (e *UnknownFieldError) Error() string {
	return "unknown field: " + e.Field
}

type NonEditableFieldError struct {
	Field string
}

func (e *NonEditableFieldError) Error() string {
	return "non-editable field: " + e.Field
}

type SeedReader interface {
	Get(context.Context, spine.ContractSeedID) (spine.ContractSeed, bool, error)
}

type Store interface {
	Create(context.Context, spine.ContractDraft) error
	Update(context.Context, spine.ContractDraft) error
	Get(context.Context, spine.ContractDraftID) (spine.ContractDraft, bool, error)
	GetByContractSeedID(context.Context, spine.ContractSeedID) (spine.ContractDraft, bool, error)
}

type EventLog interface {
	Append(context.Context, spine.Event) error
}

type transactionalDraftStore interface {
	CreateWithEvent(context.Context, spine.ContractDraft, spine.Event) error
}

type transactionalDraftUpdateStore interface {
	UpdateWithEvent(context.Context, spine.ContractDraft, spine.Event) error
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
		OrganizationID:             seed.OrganizationID,
		ProjectID:                  seed.ProjectID,
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
	if txDrafts, ok := s.Drafts.(transactionalDraftStore); ok {
		if err := txDrafts.CreateWithEvent(ctx, created, event); err != nil {
			if _, ok, lookupErr := s.Drafts.GetByContractSeedID(ctx, seed.ID); lookupErr != nil {
				return spine.ContractDraft{}, fmt.Errorf("get contract draft by contract seed id after create failure: %w", lookupErr)
			} else if ok {
				return spine.ContractDraft{}, ErrAlreadyDrafted
			}
			return spine.ContractDraft{}, fmt.Errorf("create contract draft with event: %w", err)
		}
		return created, nil
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

func (s *Service) Update(ctx context.Context, draftID spine.ContractDraftID, input spine.ContractDraftUpdateRequest) (spine.ContractDraft, error) {
	if err := s.validateDependencies(); err != nil {
		return spine.ContractDraft{}, err
	}

	draft, ok, err := s.Drafts.Get(ctx, draftID)
	if err != nil {
		return spine.ContractDraft{}, fmt.Errorf("get contract draft: %w", err)
	}
	if !ok {
		return spine.ContractDraft{}, ErrContractDraftNotFound
	}
	if draft.State != spine.ContractDraftStateDraft {
		return spine.ContractDraft{}, fmt.Errorf("%w: %s", ErrInvalidDraftState, draft.State)
	}
	if err := validateUpdatedBy(input.UpdatedBy); err != nil {
		return spine.ContractDraft{}, err
	}
	if len(input.Changes) == 0 {
		return spine.ContractDraft{}, &ValidationError{Field: "changes", Message: "must include at least one editable field"}
	}

	updated := draft
	previousValues := make(map[string]any, len(input.Changes))
	newValues := make(map[string]any, len(input.Changes))
	changedFields := make([]string, 0, len(input.Changes))
	for field, raw := range input.Changes {
		if isNonEditableField(field) {
			return spine.ContractDraft{}, &NonEditableFieldError{Field: field}
		}
		if !isEditableField(field) {
			return spine.ContractDraft{}, &UnknownFieldError{Field: field}
		}
		if isJSONNull(raw) {
			return spine.ContractDraft{}, &ValidationError{Field: "changes." + field, Message: "must not be null"}
		}

		switch field {
		case "title":
			value, err := decodeStringChange(field, raw)
			if err != nil {
				return spine.ContractDraft{}, err
			}
			previousValues[field] = draft.Title
			newValues[field] = value
			updated.Title = value
		case "intent_summary":
			value, err := decodeStringChange(field, raw)
			if err != nil {
				return spine.ContractDraft{}, err
			}
			previousValues[field] = draft.IntentSummary
			newValues[field] = value
			updated.IntentSummary = value
		case "proposed_scope":
			value, err := decodeStringSliceChange(field, raw, true)
			if err != nil {
				return spine.ContractDraft{}, err
			}
			previousValues[field] = cloneStrings(draft.ProposedScope)
			newValues[field] = cloneStrings(value)
			updated.ProposedScope = value
		case "proposed_non_goals":
			value, err := decodeStringSliceChange(field, raw, false)
			if err != nil {
				return spine.ContractDraft{}, err
			}
			previousValues[field] = cloneStrings(draft.ProposedNonGoals)
			newValues[field] = cloneStrings(value)
			updated.ProposedNonGoals = value
		case "proposed_constraints":
			value, err := decodeStringSliceChange(field, raw, false)
			if err != nil {
				return spine.ContractDraft{}, err
			}
			previousValues[field] = cloneStrings(draft.ProposedConstraints)
			newValues[field] = cloneStrings(value)
			updated.ProposedConstraints = value
		case "proposed_acceptance_criteria":
			value, err := decodeStringSliceChange(field, raw, true)
			if err != nil {
				return spine.ContractDraft{}, err
			}
			previousValues[field] = cloneStrings(draft.ProposedAcceptanceCriteria)
			newValues[field] = cloneStrings(value)
			updated.ProposedAcceptanceCriteria = value
		case "proposed_expected_checks":
			value, err := decodeStringSliceChange(field, raw, false)
			if err != nil {
				return spine.ContractDraft{}, err
			}
			previousValues[field] = cloneStrings(draft.ProposedExpectedChecks)
			newValues[field] = cloneStrings(value)
			updated.ProposedExpectedChecks = value
		case "proposed_proof_expectations":
			value, err := decodeStringSliceChange(field, raw, true)
			if err != nil {
				return spine.ContractDraft{}, err
			}
			previousValues[field] = cloneStrings(draft.ProposedProofExpectations)
			newValues[field] = cloneStrings(value)
			updated.ProposedProofExpectations = value
		case "risk_hints":
			value, err := decodeStringSliceChange(field, raw, false)
			if err != nil {
				return spine.ContractDraft{}, err
			}
			previousValues[field] = cloneStrings(draft.RiskHints)
			newValues[field] = cloneStrings(value)
			updated.RiskHints = value
		}
		changedFields = append(changedFields, field)
	}
	sort.Strings(changedFields)
	updated.State = spine.ContractDraftStateDraft

	now := s.Clock.Now().UTC()
	event, err := s.contractDraftUpdatedEvent(updated, changedFields, input.UpdatedBy, previousValues, newValues, now)
	if err != nil {
		return spine.ContractDraft{}, err
	}
	if txDrafts, ok := s.Drafts.(transactionalDraftUpdateStore); ok {
		if err := txDrafts.UpdateWithEvent(ctx, updated, event); err != nil {
			return spine.ContractDraft{}, fmt.Errorf("update contract draft with event: %w", err)
		}
		return updated, nil
	}
	if err := s.Drafts.Update(ctx, updated); err != nil {
		return spine.ContractDraft{}, fmt.Errorf("update contract draft: %w", err)
	}
	if err := s.Events.Append(ctx, event); err != nil {
		return spine.ContractDraft{}, fmt.Errorf("append contract draft updated event: %w", err)
	}

	return updated, nil
}

func validateUpdatedBy(updatedBy spine.ActorRef) error {
	if strings.TrimSpace(updatedBy.Kind) == "" {
		return &ValidationError{Field: "updated_by.kind", Message: "is required"}
	}
	if strings.TrimSpace(updatedBy.ID) == "" {
		return &ValidationError{Field: "updated_by.id", Message: "is required"}
	}
	return nil
}

func isEditableField(field string) bool {
	switch field {
	case "title",
		"intent_summary",
		"proposed_scope",
		"proposed_non_goals",
		"proposed_constraints",
		"proposed_acceptance_criteria",
		"proposed_expected_checks",
		"proposed_proof_expectations",
		"risk_hints":
		return true
	default:
		return false
	}
}

func isNonEditableField(field string) bool {
	switch field {
	case "id",
		"contract_seed_id",
		"goal_id",
		"repo_binding_id",
		"source_refs",
		"created_at",
		"state":
		return true
	default:
		return false
	}
}

func isJSONNull(raw json.RawMessage) bool {
	return strings.TrimSpace(string(raw)) == "null"
}

func decodeStringChange(field string, raw json.RawMessage) (string, error) {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", &ValidationError{Field: "changes." + field, Message: "must be a string"}
	}
	return value, nil
}

func decodeStringSliceChange(field string, raw json.RawMessage, requireNonEmpty bool) ([]string, error) {
	var value []string
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, &ValidationError{Field: "changes." + field, Message: "must be an array of strings"}
	}
	if value == nil {
		value = []string{}
	}
	if requireNonEmpty && len(value) == 0 {
		return nil, &ValidationError{Field: "changes." + field, Message: "must include at least one item"}
	}
	return cloneStrings(value), nil
}

func cloneStrings(values []string) []string {
	if values == nil {
		return nil
	}
	return append([]string{}, values...)
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
		ID:             eventID,
		Type:           EventTypeContractDraftCreated,
		EntityType:     EntityTypeContractDraft,
		EntityID:       string(created.ID),
		OrganizationID: created.OrganizationID,
		ProjectID:      created.ProjectID,
		RepoBindingID:  created.RepoBindingID,
		Timestamp:      created.CreatedAt,
		Payload:        payload,
	}, nil
}

type contractDraftUpdatedPayload struct {
	ContractDraftID spine.ContractDraftID `json:"contract_draft_id"`
	ChangedFields   []string              `json:"changed_fields"`
	UpdatedBy       spine.ActorRef        `json:"updated_by"`
	PreviousValues  map[string]any        `json:"previous_values"`
	NewValues       map[string]any        `json:"new_values"`
	UpdatedAt       time.Time             `json:"updated_at"`
}

func (s *Service) contractDraftUpdatedEvent(updated spine.ContractDraft, changedFields []string, updatedBy spine.ActorRef, previousValues map[string]any, newValues map[string]any, updatedAt time.Time) (spine.Event, error) {
	eventID, err := s.IDs.NewEventID()
	if err != nil {
		return spine.Event{}, fmt.Errorf("new contract draft updated event id: %w", err)
	}

	payload, err := json.Marshal(contractDraftUpdatedPayload{
		ContractDraftID: updated.ID,
		ChangedFields:   append([]string{}, changedFields...),
		UpdatedBy:       updatedBy,
		PreviousValues:  previousValues,
		NewValues:       newValues,
		UpdatedAt:       updatedAt,
	})
	if err != nil {
		return spine.Event{}, fmt.Errorf("marshal contract draft updated event payload: %w", err)
	}

	return spine.Event{
		ID:            eventID,
		Type:          EventTypeContractDraftUpdated,
		EntityType:    EntityTypeContractDraft,
		EntityID:      string(updated.ID),
		RepoBindingID: updated.RepoBindingID,
		Timestamp:     updatedAt,
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
