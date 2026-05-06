package httpserver

import (
	"context"
	"errors"
	"net/http"

	"github.com/heurema/goalrail/apps/server/internal/clarification"
	"github.com/heurema/goalrail/apps/server/internal/continuation"
	"github.com/heurema/goalrail/apps/server/internal/goal"
	"github.com/heurema/goalrail/apps/server/internal/spine"
)

type ContinuationService interface {
	ReconcileGoal(context.Context, spine.GoalID, spine.OrganizationMembership) (spine.GoalContinuation, error)
}

type ContinuationHandler struct {
	authService AuthService
	service     ContinuationService
}

func NewContinuationHandler(authService AuthService, service ContinuationService) *ContinuationHandler {
	return &ContinuationHandler{authService: authService, service: service}
}

func (h *ContinuationHandler) ReconcileGoal(w http.ResponseWriter, r *http.Request) {
	profile, err := h.authService.Me(r.Context(), bearerToken(r.Header.Get("Authorization")))
	if err != nil {
		respondAuthError(w, err)
		return
	}

	result, err := h.service.ReconcileGoal(r.Context(), spine.GoalID(r.PathValue("id")), profile.OrganizationMembership)
	if err != nil {
		h.respondServiceError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, result)
}

func (h *ContinuationHandler) respondServiceError(w http.ResponseWriter, err error) {
	var continuationValidationErr *continuation.ValidationError
	var goalValidationErr *goal.ValidationError
	var clarificationValidationErr *clarification.ValidationError
	switch {
	case errors.As(err, &continuationValidationErr), errors.As(err, &goalValidationErr), errors.As(err, &clarificationValidationErr):
		RespondError(w, http.StatusBadRequest, "validation_failed", err.Error())
	case errors.Is(err, continuation.ErrGoalNotFound), errors.Is(err, clarification.ErrGoalNotFound):
		RespondError(w, http.StatusNotFound, "not_found", "goal not found")
	case errors.Is(err, continuation.ErrMembershipRequired):
		RespondError(w, http.StatusForbidden, "membership_required", "active organization membership is required")
	case errors.Is(err, continuation.ErrOrganizationForbidden):
		RespondError(w, http.StatusForbidden, "forbidden", "user is not allowed to continue this goal")
	case errors.Is(err, goal.ErrInvalidGoalState):
		RespondError(w, http.StatusConflict, "invalid_state", "goal state is not readiness-checkable")
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
		respondInternalError(w)
	}
}
