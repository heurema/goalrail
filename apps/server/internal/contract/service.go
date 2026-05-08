package contract

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/heurema/goalrail/apps/server/internal/contractseed"
	"github.com/heurema/goalrail/apps/server/internal/spine"
)

var (
	ErrContractNotFound            = errors.New("contract not found")
	ErrGoalNotFound                = errors.New("goal not found")
	ErrInvalidContractState        = errors.New("contract state is not valid for this transition")
	ErrInvalidGoalState            = errors.New("goal state is not valid for contract creation")
	ErrContractCurrentDraftMissing = errors.New("contract current draft is missing")
	ErrAlreadyApproved             = errors.New("contract already approved")
	ErrMembershipRequired          = errors.New("active organization membership is required")
	ErrOrganizationForbidden       = errors.New("user is not allowed to create contract for this goal")
	ErrProjectMismatch             = errors.New("contract create project expectation does not match goal")
	ErrRepoBindingMismatch         = errors.New("contract create repo binding expectation does not match goal")
)

const (
	defaultListLimit = 50
	maxListLimit     = 100
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
	GetByGoalID(context.Context, spine.GoalID) (spine.Contract, bool, error)
	List(context.Context, spine.ContractListFilter) ([]spine.Contract, error)
}

type GoalReader interface {
	Get(context.Context, spine.GoalID) (spine.Goal, bool, error)
}

type TransactionRunner interface {
	RunReadCommitted(context.Context, func(context.Context) error) error
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
	Goals     GoalReader
	Contracts Store
	Seeds     SeedCreator
	Drafts    DraftService
	Approvals ApprovalService
	TxRunner  TransactionRunner
}

type ListInput struct {
	Membership    spine.OrganizationMembership
	ProjectID     spine.ProjectID
	RepoBindingID spine.RepoBindingID
	GoalID        spine.GoalID
	State         spine.ContractState
	Limit         int
}

func NewService(goals GoalReader, contracts Store, seeds SeedCreator, drafts DraftService, approvals ApprovalService, txRunner TransactionRunner) *Service {
	return &Service{
		Goals:     goals,
		Contracts: contracts,
		Seeds:     seeds,
		Drafts:    drafts,
		Approvals: approvals,
		TxRunner:  txRunner,
	}
}

func (s *Service) List(ctx context.Context, input ListInput) (spine.ContractList, error) {
	if err := authorizeContractRead(input.Membership); err != nil {
		return spine.ContractList{}, err
	}
	limit, err := normalizeListLimit(input.Limit)
	if err != nil {
		return spine.ContractList{}, err
	}
	projectID := spine.ProjectID(strings.TrimSpace(string(input.ProjectID)))
	repoBindingID := spine.RepoBindingID(strings.TrimSpace(string(input.RepoBindingID)))
	goalID := spine.GoalID(strings.TrimSpace(string(input.GoalID)))
	state := spine.ContractState(strings.TrimSpace(string(input.State)))
	if err := validateOptionalUUIDv7Filter("project_id", projectID); err != nil {
		return spine.ContractList{}, err
	}
	if err := validateOptionalUUIDv7Filter("repo_binding_id", repoBindingID); err != nil {
		return spine.ContractList{}, err
	}
	if err := validateOptionalUUIDv7Filter("goal_id", goalID); err != nil {
		return spine.ContractList{}, err
	}
	if err := validateContractState(state); err != nil {
		return spine.ContractList{}, err
	}

	contracts, err := s.Contracts.List(ctx, spine.ContractListFilter{
		OrganizationID: input.Membership.OrganizationID,
		ProjectID:      projectID,
		RepoBindingID:  repoBindingID,
		GoalID:         goalID,
		State:          state,
		Limit:          limit,
	})
	if err != nil {
		return spine.ContractList{}, fmt.Errorf("list contracts: %w", err)
	}
	if contracts == nil {
		contracts = []spine.Contract{}
	}
	return spine.ContractList{Contracts: contracts, Limit: limit}, nil
}

func (s *Service) Create(ctx context.Context, input spine.ContractCreateRequest, membership spine.OrganizationMembership) (spine.Contract, bool, error) {
	if err := validateGoalID(input.GoalID); err != nil {
		return spine.Contract{}, false, err
	}

	goal, ok, err := s.Goals.Get(ctx, input.GoalID)
	if err != nil {
		return spine.Contract{}, false, fmt.Errorf("get goal: %w", err)
	}
	if !ok {
		return spine.Contract{}, false, ErrGoalNotFound
	}
	if err := authorizeGoalContractCreation(membership, goal); err != nil {
		return spine.Contract{}, false, err
	}
	if err := validateExpectedGoalContext(input, goal); err != nil {
		return spine.Contract{}, false, err
	}
	if existing, ok, err := s.Contracts.GetByGoalID(ctx, input.GoalID); err != nil {
		return spine.Contract{}, false, fmt.Errorf("get contract by goal id: %w", err)
	} else if ok {
		return existing, false, nil
	}
	if goal.State != spine.GoalStateReadyForContractSeed {
		return spine.Contract{}, false, fmt.Errorf("%w: %s", ErrInvalidGoalState, goal.State)
	}

	var seed spine.ContractSeed
	var draft spine.ContractDraft
	create := func(createCtx context.Context) error {
		var err error
		seed, err = s.Seeds.Create(createCtx, input.GoalID)
		if err != nil {
			return err
		}
		draft, err = s.Drafts.Create(createCtx, seed.ID)
		if err != nil {
			return err
		}
		return nil
	}

	if err := s.TxRunner.RunReadCommitted(ctx, create); err != nil {
		if errors.Is(err, contractseed.ErrAlreadySeeded) {
			existing, ok, lookupErr := s.Contracts.GetByGoalID(ctx, input.GoalID)
			if lookupErr != nil {
				return spine.Contract{}, false, fmt.Errorf("get existing contract after already seeded: %w", lookupErr)
			}
			if ok {
				return existing, false, nil
			}
		}
		return spine.Contract{}, false, err
	}
	created, err := s.getContract(ctx, draft.ContractID)
	return created, true, err
}

func (s *Service) Get(ctx context.Context, id spine.ContractID) (spine.Contract, error) {
	return s.getContract(ctx, id)
}

func (s *Service) UpdateDraft(ctx context.Context, id spine.ContractID, input spine.ContractDraftUpdateRequest, membership spine.OrganizationMembership) (spine.Contract, error) {
	contract, err := s.getContract(ctx, id)
	if err != nil {
		return spine.Contract{}, err
	}
	if err := authorizeContractMutation(membership, contract); err != nil {
		return spine.Contract{}, err
	}
	if err := validateExpectedContractContext(input.ProjectID, input.RepoBindingID, contract); err != nil {
		return spine.Contract{}, err
	}
	if contract.State != spine.ContractStateDraft {
		return spine.Contract{}, fmt.Errorf("%w: %s", ErrInvalidContractState, contract.State)
	}
	draftID, err := currentDraftID(contract)
	if err != nil {
		return spine.Contract{}, err
	}
	update := func(updateCtx context.Context) error {
		_, err := s.Drafts.Update(updateCtx, draftID, input)
		return err
	}
	if err := s.TxRunner.RunReadCommitted(ctx, update); err != nil {
		return spine.Contract{}, err
	}
	return s.getContract(ctx, id)
}

func (s *Service) SubmitForApproval(ctx context.Context, id spine.ContractID, input spine.ContractDraftReadyForApprovalRequest, membership spine.OrganizationMembership) (spine.Contract, error) {
	contract, err := s.getContract(ctx, id)
	if err != nil {
		return spine.Contract{}, err
	}
	if err := authorizeContractMutation(membership, contract); err != nil {
		return spine.Contract{}, err
	}
	if err := validateExpectedContractContext(input.ProjectID, input.RepoBindingID, contract); err != nil {
		return spine.Contract{}, err
	}
	if contract.State != spine.ContractStateDraft {
		return spine.Contract{}, fmt.Errorf("%w: %s", ErrInvalidContractState, contract.State)
	}
	draftID, err := currentDraftID(contract)
	if err != nil {
		return spine.Contract{}, err
	}
	markReady := func(markReadyCtx context.Context) error {
		_, err := s.Drafts.MarkReadyForApproval(markReadyCtx, draftID, input)
		return err
	}
	if err := s.TxRunner.RunReadCommitted(ctx, markReady); err != nil {
		return spine.Contract{}, err
	}
	return s.getContract(ctx, id)
}

func (s *Service) Approve(ctx context.Context, id spine.ContractID, input spine.ApproveContractDraftRequest, membership spine.OrganizationMembership) (spine.Contract, error) {
	contract, err := s.getContract(ctx, id)
	if err != nil {
		return spine.Contract{}, err
	}
	if err := authorizeContractMutation(membership, contract); err != nil {
		return spine.Contract{}, err
	}
	if err := validateExpectedContractContext(input.ProjectID, input.RepoBindingID, contract); err != nil {
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
	approve := func(approveCtx context.Context) error {
		_, err := s.Approvals.ApproveDraft(approveCtx, draftID, input)
		return err
	}
	if err := s.TxRunner.RunReadCommitted(ctx, approve); err != nil {
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

func authorizeGoalContractCreation(membership spine.OrganizationMembership, goal spine.Goal) error {
	if membership.State != spine.EntityStateActive || strings.TrimSpace(string(membership.OrganizationID)) == "" {
		return ErrMembershipRequired
	}
	if membership.OrganizationID != goal.OrganizationID {
		return ErrOrganizationForbidden
	}
	return nil
}

func authorizeContractMutation(membership spine.OrganizationMembership, contract spine.Contract) error {
	if membership.State != spine.EntityStateActive || strings.TrimSpace(string(membership.OrganizationID)) == "" {
		return ErrMembershipRequired
	}
	if membership.OrganizationID != contract.OrganizationID {
		return ErrOrganizationForbidden
	}
	return nil
}

func authorizeContractRead(membership spine.OrganizationMembership) error {
	if membership.State != spine.EntityStateActive || strings.TrimSpace(string(membership.OrganizationID)) == "" {
		return ErrMembershipRequired
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

func validateOptionalUUIDv7Filter(field string, value any) error {
	text := strings.TrimSpace(fmt.Sprint(value))
	if text == "" {
		return nil
	}
	id, err := uuid.Parse(text)
	if err != nil {
		return &ValidationError{Field: field, Message: "must be a UUID"}
	}
	if id.Version() != 7 {
		return &ValidationError{Field: field, Message: "must be a UUIDv7"}
	}
	return nil
}

func normalizeListLimit(limit int) (int, error) {
	if limit == 0 {
		return defaultListLimit, nil
	}
	if limit < 0 {
		return 0, &ValidationError{Field: "limit", Message: "must be positive"}
	}
	if limit > maxListLimit {
		return 0, &ValidationError{Field: "limit", Message: "must be <= 100"}
	}
	return limit, nil
}

func validateContractState(state spine.ContractState) error {
	switch state {
	case "", spine.ContractStateSeeded, spine.ContractStateDraft, spine.ContractStateReadyForApproval, spine.ContractStateApproved:
		return nil
	default:
		return &ValidationError{Field: "state", Message: "must be a known contract state"}
	}
}

func validateExpectedGoalContext(input spine.ContractCreateRequest, goal spine.Goal) error {
	if strings.TrimSpace(string(input.ProjectID)) != "" && input.ProjectID != goal.ProjectID {
		return ErrProjectMismatch
	}
	if strings.TrimSpace(string(input.RepoBindingID)) != "" && input.RepoBindingID != goal.RepoBindingID {
		return ErrRepoBindingMismatch
	}
	return nil
}

func validateExpectedContractContext(projectID spine.ProjectID, repoBindingID spine.RepoBindingID, contract spine.Contract) error {
	if strings.TrimSpace(string(projectID)) != "" && projectID != contract.ProjectID {
		return ErrProjectMismatch
	}
	if strings.TrimSpace(string(repoBindingID)) != "" && repoBindingID != contract.RepoBindingID {
		return ErrRepoBindingMismatch
	}
	return nil
}
