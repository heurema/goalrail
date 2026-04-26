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

func (h *ContractDraftHandler) respondServiceError(w http.ResponseWriter, err error) {
	var validationErr *contractdraft.ValidationError
	switch {
	case errors.As(err, &validationErr):
		RespondError(w, http.StatusBadRequest, "validation_failed", validationErr.Error())
	case errors.Is(err, contractdraft.ErrContractSeedNotFound):
		RespondError(w, http.StatusNotFound, "not_found", "contract seed not found")
	case errors.Is(err, contractdraft.ErrInvalidSeedState):
		RespondError(w, http.StatusConflict, "invalid_state", "contract seed is not ready for contract draft")
	case errors.Is(err, contractdraft.ErrAlreadyDrafted):
		RespondError(w, http.StatusConflict, "already_drafted", "contract seed already has contract draft")
	default:
		RespondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
