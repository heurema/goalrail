package httpserver

import (
	"context"
	"errors"
	"net/http"

	"github.com/heurema/goalrail/apps/server/internal/contractdraft"
	"github.com/heurema/goalrail/apps/server/internal/spine"
)

type ContractDraftService interface {
	Create(context.Context, spine.ContractSeedID) (spine.ContractDraft, error)
	Update(context.Context, spine.ContractDraftID, spine.ContractDraftUpdateRequest) (spine.ContractDraft, error)
	MarkReadyForApproval(context.Context, spine.ContractDraftID, spine.ContractDraftReadyForApprovalRequest) (spine.ContractDraft, error)
}

type ContractDraftHandler struct {
	service ContractDraftService
}

func NewContractDraftHandler(service ContractDraftService) *ContractDraftHandler {
	return &ContractDraftHandler{service: service}
}

func (h *ContractDraftHandler) Create(w http.ResponseWriter, r *http.Request) {
	created, err := h.service.Create(r.Context(), spine.ContractSeedID(r.PathValue("id")))
	if err != nil {
		h.respondServiceError(w, err)
		return
	}

	RespondJSON(w, http.StatusCreated, created)
}

func (h *ContractDraftHandler) Update(w http.ResponseWriter, r *http.Request) {
	var input spine.ContractDraftUpdateRequest
	if err := decodeStrictJSON(r.Body, &input); err != nil {
		respondInvalidJSON(w)
		return
	}

	updated, err := h.service.Update(r.Context(), spine.ContractDraftID(r.PathValue("id")), input)
	if err != nil {
		h.respondServiceError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, updated)
}

func (h *ContractDraftHandler) MarkReadyForApproval(w http.ResponseWriter, r *http.Request) {
	var input spine.ContractDraftReadyForApprovalRequest
	if err := decodeStrictJSON(r.Body, &input); err != nil {
		respondInvalidJSON(w)
		return
	}

	updated, err := h.service.MarkReadyForApproval(r.Context(), spine.ContractDraftID(r.PathValue("id")), input)
	if err != nil {
		h.respondServiceError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, updated)
}

func (h *ContractDraftHandler) respondServiceError(w http.ResponseWriter, err error) {
	var validationErr *contractdraft.ValidationError
	var unknownFieldErr *contractdraft.UnknownFieldError
	var nonEditableFieldErr *contractdraft.NonEditableFieldError
	var completenessErr *contractdraft.CompletenessError
	switch {
	case errors.As(err, &validationErr):
		RespondError(w, http.StatusBadRequest, "validation_failed", validationErr.Error())
	case errors.As(err, &unknownFieldErr):
		RespondError(w, http.StatusBadRequest, "unknown_field", unknownFieldErr.Error())
	case errors.As(err, &nonEditableFieldErr):
		RespondError(w, http.StatusBadRequest, "non_editable_field", nonEditableFieldErr.Error())
	case errors.As(err, &completenessErr):
		RespondError(w, http.StatusBadRequest, "validation_failed", completenessErr.Error())
	case errors.Is(err, contractdraft.ErrContractSeedNotFound):
		RespondError(w, http.StatusNotFound, "not_found", "contract seed not found")
	case errors.Is(err, contractdraft.ErrContractDraftNotFound):
		RespondError(w, http.StatusNotFound, "not_found", "contract draft not found")
	case errors.Is(err, contractdraft.ErrInvalidSeedState):
		RespondError(w, http.StatusConflict, "invalid_state", "contract seed is not ready for contract draft")
	case errors.Is(err, contractdraft.ErrInvalidDraftState):
		RespondError(w, http.StatusConflict, "invalid_state", "contract draft is not in draft state")
	case errors.Is(err, contractdraft.ErrAlreadyDrafted):
		RespondError(w, http.StatusConflict, "already_drafted", "contract seed already has contract draft")
	default:
		respondInternalError(w)
	}
}
