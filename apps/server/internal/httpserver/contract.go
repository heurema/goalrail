package httpserver

import (
	"context"
	"errors"
	"net/http"

	"github.com/heurema/goalrail/apps/server/internal/approvedcontract"
	"github.com/heurema/goalrail/apps/server/internal/contract"
	"github.com/heurema/goalrail/apps/server/internal/contractdraft"
	"github.com/heurema/goalrail/apps/server/internal/contractseed"
	"github.com/heurema/goalrail/apps/server/internal/spine"
)

type ContractService interface {
	Create(context.Context, spine.ContractCreateRequest) (spine.Contract, error)
	Get(context.Context, spine.ContractID) (spine.Contract, error)
	UpdateDraft(context.Context, spine.ContractID, spine.ContractDraftUpdateRequest) (spine.Contract, error)
	SubmitForApproval(context.Context, spine.ContractID, spine.ContractDraftReadyForApprovalRequest) (spine.Contract, error)
	Approve(context.Context, spine.ContractID, spine.ApproveContractDraftRequest) (spine.Contract, error)
}

type ContractHandler struct {
	service ContractService
}

func NewContractHandler(service ContractService) *ContractHandler {
	return &ContractHandler{service: service}
}

func (h *ContractHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input spine.ContractCreateRequest
	if err := decodeStrictJSON(r.Body, &input); err != nil {
		respondInvalidJSON(w)
		return
	}

	created, err := h.service.Create(r.Context(), input)
	if err != nil {
		h.respondServiceError(w, err)
		return
	}

	RespondJSON(w, http.StatusCreated, created)
}

func (h *ContractHandler) Get(w http.ResponseWriter, r *http.Request) {
	contract, err := h.service.Get(r.Context(), spine.ContractID(r.PathValue("id")))
	if err != nil {
		h.respondServiceError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, contract)
}

func (h *ContractHandler) UpdateDraft(w http.ResponseWriter, r *http.Request) {
	var input spine.ContractDraftUpdateRequest
	if err := decodeStrictJSON(r.Body, &input); err != nil {
		respondInvalidJSON(w)
		return
	}

	updated, err := h.service.UpdateDraft(r.Context(), spine.ContractID(r.PathValue("id")), input)
	if err != nil {
		h.respondServiceError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, updated)
}

func (h *ContractHandler) SubmitForApproval(w http.ResponseWriter, r *http.Request) {
	var input spine.ContractDraftReadyForApprovalRequest
	if err := decodeStrictJSON(r.Body, &input); err != nil {
		respondInvalidJSON(w)
		return
	}

	updated, err := h.service.SubmitForApproval(r.Context(), spine.ContractID(r.PathValue("id")), input)
	if err != nil {
		h.respondServiceError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, updated)
}

func (h *ContractHandler) Approve(w http.ResponseWriter, r *http.Request) {
	var input spine.ApproveContractDraftRequest
	if err := decodeStrictJSON(r.Body, &input); err != nil {
		respondInvalidJSON(w)
		return
	}

	ctx := contextWithApprovalActor(r.Context(), input.ApprovedBy)
	approved, err := h.service.Approve(ctx, spine.ContractID(r.PathValue("id")), input)
	if err != nil {
		h.respondServiceError(w, err)
		return
	}

	RespondJSON(w, http.StatusCreated, approved)
}

func (h *ContractHandler) respondServiceError(w http.ResponseWriter, err error) {
	var contractValidationErr *contract.ValidationError
	var seedValidationErr *contractseed.ValidationError
	var draftValidationErr *contractdraft.ValidationError
	var approvedValidationErr *approvedcontract.ValidationError
	var unknownFieldErr *contractdraft.UnknownFieldError
	var nonEditableFieldErr *contractdraft.NonEditableFieldError
	var draftCompletenessErr *contractdraft.CompletenessError
	var approvedCompletenessErr *approvedcontract.CompletenessError
	switch {
	case errors.As(err, &contractValidationErr):
		RespondError(w, http.StatusBadRequest, "validation_failed", contractValidationErr.Error())
	case errors.As(err, &seedValidationErr):
		RespondError(w, http.StatusBadRequest, "validation_failed", seedValidationErr.Error())
	case errors.As(err, &draftValidationErr):
		RespondError(w, http.StatusBadRequest, "validation_failed", draftValidationErr.Error())
	case errors.As(err, &approvedValidationErr):
		RespondError(w, http.StatusBadRequest, "validation_failed", approvedValidationErr.Error())
	case errors.As(err, &unknownFieldErr):
		RespondError(w, http.StatusBadRequest, "unknown_field", unknownFieldErr.Error())
	case errors.As(err, &nonEditableFieldErr):
		RespondError(w, http.StatusBadRequest, "non_editable_field", nonEditableFieldErr.Error())
	case errors.As(err, &draftCompletenessErr):
		RespondError(w, http.StatusBadRequest, "validation_failed", draftCompletenessErr.Error())
	case errors.As(err, &approvedCompletenessErr):
		RespondError(w, http.StatusBadRequest, "validation_failed", approvedCompletenessErr.Error())
	case errors.Is(err, contract.ErrContractNotFound):
		RespondError(w, http.StatusNotFound, "not_found", "contract not found")
	case errors.Is(err, contractseed.ErrGoalNotFound):
		RespondError(w, http.StatusNotFound, "not_found", "goal not found")
	case errors.Is(err, contractdraft.ErrContractSeedNotFound):
		RespondError(w, http.StatusNotFound, "not_found", "contract seed not found")
	case errors.Is(err, contractdraft.ErrContractDraftNotFound):
		RespondError(w, http.StatusNotFound, "not_found", "contract draft not found")
	case errors.Is(err, approvedcontract.ErrContractDraftNotFound):
		RespondError(w, http.StatusNotFound, "not_found", "contract draft not found")
	case errors.Is(err, contractseed.ErrInvalidGoalState):
		RespondError(w, http.StatusConflict, "invalid_state", "goal is not ready for contract")
	case errors.Is(err, contract.ErrInvalidContractState):
		RespondError(w, http.StatusConflict, "invalid_state", "contract state does not allow this transition")
	case errors.Is(err, contract.ErrContractCurrentDraftMissing):
		RespondError(w, http.StatusConflict, "invalid_state", "contract current draft is missing")
	case errors.Is(err, contractdraft.ErrInvalidSeedState):
		RespondError(w, http.StatusConflict, "invalid_state", "contract seed is not ready for contract draft")
	case errors.Is(err, contractdraft.ErrInvalidDraftState):
		RespondError(w, http.StatusConflict, "invalid_state", "contract draft is not in the required state")
	case errors.Is(err, approvedcontract.ErrInvalidDraftState):
		RespondError(w, http.StatusConflict, "invalid_state", "contract draft is not ready for approval")
	case errors.Is(err, contractseed.ErrAlreadySeeded):
		RespondError(w, http.StatusConflict, "already_seeded", "goal already has contract")
	case errors.Is(err, contractdraft.ErrAlreadyDrafted):
		RespondError(w, http.StatusConflict, "already_drafted", "contract already has draft")
	case errors.Is(err, contract.ErrAlreadyApproved):
		RespondError(w, http.StatusConflict, "already_approved", "contract already approved")
	case errors.Is(err, approvedcontract.ErrAlreadyApproved):
		RespondError(w, http.StatusConflict, "already_approved", "contract draft already approved")
	default:
		respondInternalError(w)
	}
}
