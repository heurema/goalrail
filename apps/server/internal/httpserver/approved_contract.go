package httpserver

import (
	"context"
	"errors"
	"net/http"

	"github.com/heurema/goalrail/apps/server/internal/approvedcontract"
	"github.com/heurema/goalrail/apps/server/internal/spine"
)

type ApprovedContractService interface {
	ApproveDraft(context.Context, spine.ContractDraftID, spine.ApproveContractDraftRequest) (spine.ApprovedContract, error)
}

type ApprovedContractHandler struct {
	service ApprovedContractService
}

func NewApprovedContractHandler(service ApprovedContractService) *ApprovedContractHandler {
	return &ApprovedContractHandler{service: service}
}

func (h *ApprovedContractHandler) ApproveDraft(w http.ResponseWriter, r *http.Request) {
	var input spine.ApproveContractDraftRequest
	if err := decodeStrictJSON(r.Body, &input); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid_json", "invalid JSON request body")
		return
	}

	approved, err := h.service.ApproveDraft(r.Context(), spine.ContractDraftID(r.PathValue("id")), input)
	if err != nil {
		h.respondServiceError(w, err)
		return
	}

	RespondJSON(w, http.StatusCreated, approved)
}

func (h *ApprovedContractHandler) respondServiceError(w http.ResponseWriter, err error) {
	var validationErr *approvedcontract.ValidationError
	var completenessErr *approvedcontract.CompletenessError
	switch {
	case errors.As(err, &validationErr):
		RespondError(w, http.StatusBadRequest, "validation_failed", validationErr.Error())
	case errors.As(err, &completenessErr):
		RespondError(w, http.StatusBadRequest, "validation_failed", completenessErr.Error())
	case errors.Is(err, approvedcontract.ErrContractDraftNotFound):
		RespondError(w, http.StatusNotFound, "not_found", "contract draft not found")
	case errors.Is(err, approvedcontract.ErrInvalidDraftState):
		RespondError(w, http.StatusConflict, "invalid_state", "contract draft is not ready for approval")
	case errors.Is(err, approvedcontract.ErrAlreadyApproved):
		RespondError(w, http.StatusConflict, "already_approved", "contract draft already approved")
	default:
		RespondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
