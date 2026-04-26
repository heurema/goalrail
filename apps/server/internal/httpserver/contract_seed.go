package httpserver

import (
	"context"
	"errors"
	"net/http"

	"github.com/heurema/goalrail/apps/server/internal/contractseed"
	"github.com/heurema/goalrail/apps/server/internal/spine"
)

type ContractSeedService interface {
	Create(context.Context, spine.GoalID) (spine.ContractSeed, error)
}

type ContractSeedHandler struct {
	service ContractSeedService
}

func NewContractSeedHandler(service ContractSeedService) *ContractSeedHandler {
	return &ContractSeedHandler{service: service}
}

func (h *ContractSeedHandler) Create(w http.ResponseWriter, r *http.Request) {
	created, err := h.service.Create(r.Context(), spine.GoalID(r.PathValue("id")))
	if err != nil {
		h.respondServiceError(w, err)
		return
	}

	RespondJSON(w, http.StatusCreated, created)
}

func (h *ContractSeedHandler) respondServiceError(w http.ResponseWriter, err error) {
	var validationErr *contractseed.ValidationError
	switch {
	case errors.As(err, &validationErr):
		RespondError(w, http.StatusBadRequest, "validation_failed", validationErr.Error())
	case errors.Is(err, contractseed.ErrGoalNotFound):
		RespondError(w, http.StatusNotFound, "not_found", "goal not found")
	case errors.Is(err, contractseed.ErrInvalidGoalState):
		RespondError(w, http.StatusConflict, "invalid_state", "goal is not ready for contract seed")
	case errors.Is(err, contractseed.ErrAlreadySeeded):
		RespondError(w, http.StatusConflict, "already_seeded", "goal already has contract seed")
	default:
		RespondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
