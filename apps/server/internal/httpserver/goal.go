package httpserver

import (
	"context"
	"errors"
	"net/http"

	"github.com/heurema/goalrail/apps/server/internal/goal"
	"github.com/heurema/goalrail/apps/server/internal/spine"
)

type GoalPromotionService interface {
	PromoteFromIntake(context.Context, spine.IntakeID) (spine.Goal, error)
}

type GoalHandler struct {
	service GoalPromotionService
}

func NewGoalHandler(service GoalPromotionService) *GoalHandler {
	return &GoalHandler{service: service}
}

func (h *GoalHandler) PromoteFromIntake(w http.ResponseWriter, r *http.Request) {
	created, err := h.service.PromoteFromIntake(r.Context(), spine.IntakeID(r.PathValue("id")))
	if err != nil {
		h.respondServiceError(w, err)
		return
	}

	RespondJSON(w, http.StatusCreated, created)
}

func (h *GoalHandler) respondServiceError(w http.ResponseWriter, err error) {
	var validationErr *goal.ValidationError
	switch {
	case errors.As(err, &validationErr):
		RespondError(w, http.StatusBadRequest, "validation_failed", validationErr.Error())
	case errors.Is(err, goal.ErrIntakeNotFound):
		RespondError(w, http.StatusNotFound, "not_found", "intake record not found")
	case errors.Is(err, goal.ErrInvalidIntakeState):
		RespondError(w, http.StatusConflict, "invalid_state", "intake record state is not promotable")
	case errors.Is(err, goal.ErrAlreadyPromoted):
		RespondError(w, http.StatusConflict, "already_promoted", "intake record already promoted to goal")
	default:
		RespondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
