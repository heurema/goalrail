package approvedcontract

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/heurema/goalrail/apps/server/internal/actor"
	"github.com/heurema/goalrail/apps/server/internal/spine"
)

const (
	EventTypeContractApproved               = "contract.approved"
	EntityTypeApprovedContract              = "ApprovedContract"
	SourceRefKindContractDraft              = "contract_draft"
	ReasonMissingTitle                      = "missing_title"
	ReasonMissingIntentSummary              = "missing_intent_summary"
	ReasonMissingRepoBindingID              = "missing_repo_binding_id"
	ReasonMissingContractSeedID             = "missing_contract_seed_id"
	ReasonMissingGoalID                     = "missing_goal_id"
	ReasonMissingProposedScope              = "missing_proposed_scope"
	ReasonMissingProposedAcceptanceCriteria = "missing_proposed_acceptance_criteria"
	ReasonMissingProposedProofExpectations  = "missing_proposed_proof_expectations"
)

var (
	ErrContractDraftNotFound = errors.New("contract draft not found")
	ErrInvalidDraftState     = errors.New("contract draft is not approvable")
	ErrAlreadyApproved       = errors.New("contract draft already approved")
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

type CompletenessError struct {
	ReasonCodes []string
}

func (e *CompletenessError) Error() string {
	if len(e.ReasonCodes) == 0 {
		return "approval checks failed"
	}
	return "approval checks failed: " + strings.Join(e.ReasonCodes, ",")
}

type DraftReader interface {
	Get(context.Context, spine.ContractDraftID) (spine.ContractDraft, bool, error)
}

type Store interface {
	Create(context.Context, spine.ApprovedContract) error
	Get(context.Context, spine.ApprovedContractID) (spine.ApprovedContract, bool, error)
	GetByContractDraftID(context.Context, spine.ContractDraftID) (spine.ApprovedContract, bool, error)
}

type EventLog interface {
	Append(context.Context, spine.Event) error
}

type transactionalStore interface {
	CreateWithEvent(context.Context, spine.ApprovedContract, spine.Event) error
}

type Clock interface {
	Now() time.Time
}

type IDGenerator interface {
	NewApprovedContractID() (spine.ApprovedContractID, error)
	NewEventID() (spine.EventID, error)
}

type Service struct {
	Drafts   DraftReader
	Approved Store
	Events   EventLog
	Clock    Clock
	IDs      IDGenerator
}

func NewService(drafts DraftReader, approved Store, events EventLog, clock Clock, ids IDGenerator) *Service {
	return &Service{
		Drafts:   drafts,
		Approved: approved,
		Events:   events,
		Clock:    clock,
		IDs:      ids,
	}
}

func (s *Service) ApproveDraft(ctx context.Context, draftID spine.ContractDraftID, input spine.ApproveContractDraftRequest) (spine.ApprovedContract, error) {
	if err := s.validateDependencies(); err != nil {
		return spine.ApprovedContract{}, err
	}
	approvedBy := effectiveApprover(ctx, input)
	if err := validateApprovedBy(approvedBy); err != nil {
		return spine.ApprovedContract{}, err
	}

	draft, ok, err := s.Drafts.Get(ctx, draftID)
	if err != nil {
		return spine.ApprovedContract{}, fmt.Errorf("get contract draft: %w", err)
	}
	if !ok {
		return spine.ApprovedContract{}, ErrContractDraftNotFound
	}
	if draft.State != spine.ContractDraftStateReadyForApproval {
		return spine.ApprovedContract{}, fmt.Errorf("%w: %s", ErrInvalidDraftState, draft.State)
	}
	if _, ok, err := s.Approved.GetByContractDraftID(ctx, draft.ID); err != nil {
		return spine.ApprovedContract{}, fmt.Errorf("get approved contract by contract draft id: %w", err)
	} else if ok {
		return spine.ApprovedContract{}, ErrAlreadyApproved
	}
	if reasonCodes := approvalReasonCodes(draft); len(reasonCodes) > 0 {
		return spine.ApprovedContract{}, &CompletenessError{ReasonCodes: reasonCodes}
	}

	approvedID, err := s.IDs.NewApprovedContractID()
	if err != nil {
		return spine.ApprovedContract{}, fmt.Errorf("new approved contract id: %w", err)
	}
	now := s.Clock.Now().UTC()
	approved := approvedContractFromDraft(approvedID, draft, approvedBy, now)
	event, err := s.contractApprovedEvent(approved, draft.State)
	if err != nil {
		return spine.ApprovedContract{}, err
	}

	if txStore, ok := s.Approved.(transactionalStore); ok {
		if err := txStore.CreateWithEvent(ctx, approved, event); err != nil {
			if _, ok, lookupErr := s.Approved.GetByContractDraftID(ctx, draft.ID); lookupErr != nil {
				return spine.ApprovedContract{}, fmt.Errorf("get approved contract by contract draft id after create failure: %w", lookupErr)
			} else if ok {
				return spine.ApprovedContract{}, ErrAlreadyApproved
			}
			return spine.ApprovedContract{}, fmt.Errorf("create approved contract with event: %w", err)
		}
		return approved, nil
	}
	if err := s.Approved.Create(ctx, approved); err != nil {
		if _, ok, lookupErr := s.Approved.GetByContractDraftID(ctx, draft.ID); lookupErr != nil {
			return spine.ApprovedContract{}, fmt.Errorf("get approved contract by contract draft id after create failure: %w", lookupErr)
		} else if ok {
			return spine.ApprovedContract{}, ErrAlreadyApproved
		}
		return spine.ApprovedContract{}, fmt.Errorf("create approved contract: %w", err)
	}
	if err := s.Events.Append(ctx, event); err != nil {
		return spine.ApprovedContract{}, fmt.Errorf("append contract approved event: %w", err)
	}

	return approved, nil
}

func approvedContractFromDraft(id spine.ApprovedContractID, draft spine.ContractDraft, approvedBy spine.ActorRef, approvedAt time.Time) spine.ApprovedContract {
	return spine.ApprovedContract{
		ID:                 id,
		OrganizationID:     draft.OrganizationID,
		ProjectID:          draft.ProjectID,
		ContractDraftID:    draft.ID,
		ContractSeedID:     draft.ContractSeedID,
		GoalID:             draft.GoalID,
		RepoBindingID:      draft.RepoBindingID,
		Title:              draft.Title,
		IntentSummary:      draft.IntentSummary,
		Scope:              cloneStrings(draft.ProposedScope),
		NonGoals:           cloneStrings(draft.ProposedNonGoals),
		Constraints:        cloneStrings(draft.ProposedConstraints),
		AcceptanceCriteria: cloneStrings(draft.ProposedAcceptanceCriteria),
		ExpectedChecks:     cloneStrings(draft.ProposedExpectedChecks),
		ProofExpectations:  cloneStrings(draft.ProposedProofExpectations),
		RiskHints:          cloneStrings(draft.RiskHints),
		ApprovedBy:         approvedBy,
		ApprovedAt:         approvedAt,
		SourceRefs:         sourceRefsForDraft(draft),
		State:              spine.ApprovedContractStateApproved,
	}
}

// effectiveApprover returns the actor that should be recorded as
// approver for this transition. Per D-0054, when a server-resolved
// ActorContext is present in ctx it is preferred over the legacy
// payload-supplied ApprovedBy field; otherwise the payload field is
// used as prototype compatibility / audit label only.
func effectiveApprover(ctx context.Context, input spine.ApproveContractDraftRequest) spine.ActorRef {
	if ac, ok := actor.FromContext(ctx); ok {
		return ac.Actor
	}
	return input.ApprovedBy
}

func validateApprovedBy(approvedBy spine.ActorRef) error {
	if strings.TrimSpace(approvedBy.Kind) == "" {
		return &ValidationError{Field: "approved_by.kind", Message: "is required"}
	}
	if strings.TrimSpace(approvedBy.ID) == "" {
		return &ValidationError{Field: "approved_by.id", Message: "is required"}
	}
	return nil
}

func approvalReasonCodes(draft spine.ContractDraft) []string {
	var reasonCodes []string
	if strings.TrimSpace(draft.Title) == "" {
		reasonCodes = append(reasonCodes, ReasonMissingTitle)
	}
	if strings.TrimSpace(draft.IntentSummary) == "" {
		reasonCodes = append(reasonCodes, ReasonMissingIntentSummary)
	}
	if strings.TrimSpace(string(draft.RepoBindingID)) == "" {
		reasonCodes = append(reasonCodes, ReasonMissingRepoBindingID)
	}
	if strings.TrimSpace(string(draft.ContractSeedID)) == "" {
		reasonCodes = append(reasonCodes, ReasonMissingContractSeedID)
	}
	if strings.TrimSpace(string(draft.GoalID)) == "" {
		reasonCodes = append(reasonCodes, ReasonMissingGoalID)
	}
	if len(nonBlankStrings(draft.ProposedScope)) == 0 {
		reasonCodes = append(reasonCodes, ReasonMissingProposedScope)
	}
	if len(nonBlankStrings(draft.ProposedAcceptanceCriteria)) == 0 {
		reasonCodes = append(reasonCodes, ReasonMissingProposedAcceptanceCriteria)
	}
	if len(nonBlankStrings(draft.ProposedProofExpectations)) == 0 {
		reasonCodes = append(reasonCodes, ReasonMissingProposedProofExpectations)
	}
	return reasonCodes
}

func nonBlankStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, value)
		}
	}
	return out
}

func sourceRefsForDraft(draft spine.ContractDraft) []spine.SourceRef {
	refs := make([]spine.SourceRef, 0, len(draft.SourceRefs)+1)
	refs = append(refs, spine.SourceRef{Kind: SourceRefKindContractDraft, ID: string(draft.ID)})
	for _, ref := range draft.SourceRefs {
		if strings.TrimSpace(ref.Kind) == "" || strings.TrimSpace(ref.ID) == "" {
			continue
		}
		refs = append(refs, ref)
	}
	return refs
}

func cloneStrings(values []string) []string {
	if values == nil {
		return nil
	}
	return append([]string{}, values...)
}

type contractApprovedPayload struct {
	ApprovedContractID spine.ApprovedContractID `json:"approved_contract_id"`
	ContractDraftID    spine.ContractDraftID    `json:"contract_draft_id"`
	ContractSeedID     spine.ContractSeedID     `json:"contract_seed_id"`
	GoalID             spine.GoalID             `json:"goal_id"`
	ApprovedBy         spine.ActorRef           `json:"approved_by"`
	ApprovedAt         time.Time                `json:"approved_at"`
	SourceRefs         []spine.SourceRef        `json:"source_refs"`
	PreviousDraftState spine.ContractDraftState `json:"previous_draft_state"`
}

func (s *Service) contractApprovedEvent(approved spine.ApprovedContract, previousDraftState spine.ContractDraftState) (spine.Event, error) {
	eventID, err := s.IDs.NewEventID()
	if err != nil {
		return spine.Event{}, fmt.Errorf("new contract approved event id: %w", err)
	}

	payload, err := json.Marshal(contractApprovedPayload{
		ApprovedContractID: approved.ID,
		ContractDraftID:    approved.ContractDraftID,
		ContractSeedID:     approved.ContractSeedID,
		GoalID:             approved.GoalID,
		ApprovedBy:         approved.ApprovedBy,
		ApprovedAt:         approved.ApprovedAt,
		SourceRefs:         append([]spine.SourceRef(nil), approved.SourceRefs...),
		PreviousDraftState: previousDraftState,
	})
	if err != nil {
		return spine.Event{}, fmt.Errorf("marshal contract approved event payload: %w", err)
	}

	return spine.Event{
		ID:             eventID,
		Type:           EventTypeContractApproved,
		EntityType:     EntityTypeApprovedContract,
		EntityID:       string(approved.ID),
		OrganizationID: approved.OrganizationID,
		ProjectID:      approved.ProjectID,
		RepoBindingID:  approved.RepoBindingID,
		Timestamp:      approved.ApprovedAt,
		Payload:        payload,
	}, nil
}

func (s *Service) validateDependencies() error {
	if s.Drafts == nil {
		return errors.New("approved contract service draft store is nil")
	}
	if s.Approved == nil {
		return errors.New("approved contract service approved contract store is nil")
	}
	if s.Events == nil {
		return errors.New("approved contract service event log is nil")
	}
	if s.Clock == nil {
		return errors.New("approved contract service clock is nil")
	}
	if s.IDs == nil {
		return errors.New("approved contract service id generator is nil")
	}
	return nil
}

type SystemClock struct{}

func (SystemClock) Now() time.Time {
	return time.Now()
}

type UUIDGenerator struct{}

func (UUIDGenerator) NewApprovedContractID() (spine.ApprovedContractID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return spine.ApprovedContractID(id.String()), nil
}

func (UUIDGenerator) NewEventID() (spine.EventID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return spine.EventID(id.String()), nil
}
