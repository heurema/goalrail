package contract

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

var (
	ErrContractNotFound            = errors.New("contract not found")
	ErrInvalidContractState        = errors.New("contract state is not valid for this transition")
	ErrContractCurrentDraftMissing = errors.New("contract current draft is missing")
	ErrAlreadyApproved             = errors.New("contract already approved")
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

type Store interface {
	Get(context.Context, spine.ContractID) (spine.Contract, bool, error)
}

type SeedCreator interface {
	Create(context.Context, spine.GoalID) (spine.ContractSeed, error)
}

type DraftService interface {
	Create(context.Context, spine.ContractSeedID) (spine.ContractDraft, error)
	Update(context.Context, spine.ContractDraftID, spine.ContractDraftUpdateRequest) (spine.ContractDraft, error)
	MarkReadyForApproval(context.Context, spine.ContractDraftID, spine.ContractDraftReadyForApprovalRequest) (spine.ContractDraft, error)
}

type ApprovalService interface {
	ApproveDraft(context.Context, spine.ContractDraftID, spine.ApproveContractDraftRequest) (spine.ApprovedContract, error)
}

type Service struct {
	Contracts Store
	Seeds     SeedCreator
	Drafts    DraftService
	Approvals ApprovalService
}

func NewService(contracts Store, seeds SeedCreator, drafts DraftService, approvals ApprovalService) *Service {
	return &Service{
		Contracts: contracts,
		Seeds:     seeds,
		Drafts:    drafts,
		Approvals: approvals,
	}
}

func (s *Service) Create(ctx context.Context, input spine.ContractCreateRequest) (spine.Contract, error) {
	if err := s.validateDependencies(); err != nil {
		return spine.Contract{}, err
	}
	if strings.TrimSpace(string(input.GoalID)) == "" {
		return spine.Contract{}, &ValidationError{Field: "goal_id", Message: "is required"}
	}

	seed, err := s.Seeds.Create(ctx, input.GoalID)
	if err != nil {
		return spine.Contract{}, err
	}
	draft, err := s.Drafts.Create(ctx, seed.ID)
	if err != nil {
		return spine.Contract{}, err
	}
	return s.getContract(ctx, draft.ContractID)
}

func (s *Service) Get(ctx context.Context, id spine.ContractID) (spine.Contract, error) {
	if err := s.validateDependencies(); err != nil {
		return spine.Contract{}, err
	}
	return s.getContract(ctx, id)
}

func (s *Service) UpdateDraft(ctx context.Context, id spine.ContractID, input spine.ContractDraftUpdateRequest) (spine.Contract, error) {
	if err := s.validateDependencies(); err != nil {
		return spine.Contract{}, err
	}
	contract, err := s.getContract(ctx, id)
	if err != nil {
		return spine.Contract{}, err
	}
	if contract.State != spine.ContractStateDraft {
		return spine.Contract{}, fmt.Errorf("%w: %s", ErrInvalidContractState, contract.State)
	}
	draftID, err := currentDraftID(contract)
	if err != nil {
		return spine.Contract{}, err
	}
	if _, err := s.Drafts.Update(ctx, draftID, input); err != nil {
		return spine.Contract{}, err
	}
	return s.getContract(ctx, id)
}

func (s *Service) SubmitForApproval(ctx context.Context, id spine.ContractID, input spine.ContractDraftReadyForApprovalRequest) (spine.Contract, error) {
	if err := s.validateDependencies(); err != nil {
		return spine.Contract{}, err
	}
	contract, err := s.getContract(ctx, id)
	if err != nil {
		return spine.Contract{}, err
	}
	if contract.State != spine.ContractStateDraft {
		return spine.Contract{}, fmt.Errorf("%w: %s", ErrInvalidContractState, contract.State)
	}
	draftID, err := currentDraftID(contract)
	if err != nil {
		return spine.Contract{}, err
	}
	if _, err := s.Drafts.MarkReadyForApproval(ctx, draftID, input); err != nil {
		return spine.Contract{}, err
	}
	return s.getContract(ctx, id)
}

func (s *Service) Approve(ctx context.Context, id spine.ContractID, input spine.ApproveContractDraftRequest) (spine.Contract, error) {
	if err := s.validateDependencies(); err != nil {
		return spine.Contract{}, err
	}
	contract, err := s.getContract(ctx, id)
	if err != nil {
		return spine.Contract{}, err
	}
	if contract.State == spine.ContractStateApproved {
		return spine.Contract{}, ErrAlreadyApproved
	}
	if contract.State != spine.ContractStateReadyForApproval {
		return spine.Contract{}, fmt.Errorf("%w: %s", ErrInvalidContractState, contract.State)
	}
	draftID, err := currentDraftID(contract)
	if err != nil {
		return spine.Contract{}, err
	}
	if _, err := s.Approvals.ApproveDraft(ctx, draftID, input); err != nil {
		return spine.Contract{}, err
	}
	return s.getContract(ctx, id)
}

func (s *Service) getContract(ctx context.Context, id spine.ContractID) (spine.Contract, error) {
	contract, ok, err := s.Contracts.Get(ctx, id)
	if err != nil {
		return spine.Contract{}, fmt.Errorf("get contract: %w", err)
	}
	if !ok {
		return spine.Contract{}, ErrContractNotFound
	}
	return contract, nil
}

func currentDraftID(contract spine.Contract) (spine.ContractDraftID, error) {
	if contract.CurrentDraftID == nil || strings.TrimSpace(string(*contract.CurrentDraftID)) == "" {
		return "", ErrContractCurrentDraftMissing
	}
	return *contract.CurrentDraftID, nil
}

func (s *Service) validateDependencies() error {
	if s.Contracts == nil {
		return errors.New("contract service: nil contract store")
	}
	if s.Seeds == nil {
		return errors.New("contract service: nil seed service")
	}
	if s.Drafts == nil {
		return errors.New("contract service: nil draft service")
	}
	if s.Approvals == nil {
		return errors.New("contract service: nil approval service")
	}
	return nil
}
