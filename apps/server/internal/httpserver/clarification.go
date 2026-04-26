package httpserver

import (
	"context"
	"errors"
	"net/http"

	"github.com/heurema/goalrail/apps/server/internal/clarification"
	"github.com/heurema/goalrail/apps/server/internal/spine"
)

type ClarificationService interface {
	CreateRequest(context.Context, spine.GoalID) (spine.ClarificationRequest, error)
}

type ClarificationHandler struct {
	service ClarificationService
}

func NewClarificationHandler(service ClarificationService) *ClarificationHandler {
	return &ClarificationHandler{service: service}
}

func (h *ClarificationHandler) CreateRequest(w http.ResponseWriter, r *http.Request) {
	created, err := h.service.CreateRequest(r.Context(), spine.GoalID(r.PathValue("id")))
	if err != nil {
		h.respondServiceError(w, err)
		return
	}

	RespondJSON(w, http.StatusCreated, created)
}

func (h *ClarificationHandler) respondServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, clarification.ErrGoalNotFound):
		RespondError(w, http.StatusNotFound, "not_found", "goal not found")
	case errors.Is(err, clarification.ErrInvalidGoalState):
		RespondError(w, http.StatusConflict, "invalid_state", "goal must need clarification")
	case errors.Is(err, clarification.ErrAlreadyOpen):
		RespondError(w, http.StatusConflict, "already_open", "clarification request already open")
	case errors.Is(err, clarification.ErrMissingReadinessReasons):
		RespondError(w, http.StatusConflict, "missing_readiness_reasons", "goal has no stored readiness reason codes")
	case errors.Is(err, clarification.ErrNoClarificationQuestions):
		RespondError(w, http.StatusConflict, "invalid_state", "goal has no clarification questions")
	case errors.Is(err, clarification.ErrPolicyRejected):
		RespondError(w, http.StatusConflict, "invalid_state", "policy rejected goals cannot create clarification request")
	default:
		RespondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
