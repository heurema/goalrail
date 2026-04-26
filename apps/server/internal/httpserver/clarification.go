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
	RecordAnswer(context.Context, spine.ClarificationRequestID, spine.ClarificationAnswerSubmission) (spine.ClarificationAnswer, error)
	ApplyAnswer(context.Context, spine.ClarificationAnswerID, spine.ClarificationAnswerApplicationRequest) (spine.ClarificationAnswerApplicationResult, spine.Goal, error)
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

func (h *ClarificationHandler) RecordAnswer(w http.ResponseWriter, r *http.Request) {
	var submission spine.ClarificationAnswerSubmission
	if err := decodeStrictJSON(r.Body, &submission); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid_json", "invalid JSON request body")
		return
	}

	recorded, err := h.service.RecordAnswer(r.Context(), spine.ClarificationRequestID(r.PathValue("id")), submission)
	if err != nil {
		h.respondServiceError(w, err)
		return
	}

	RespondJSON(w, http.StatusCreated, recorded)
}

func (h *ClarificationHandler) ApplyAnswer(w http.ResponseWriter, r *http.Request) {
	var input spine.ClarificationAnswerApplicationRequest
	if err := decodeStrictJSON(r.Body, &input); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid_json", "invalid JSON request body")
		return
	}

	application, goal, err := h.service.ApplyAnswer(r.Context(), spine.ClarificationAnswerID(r.PathValue("id")), input)
	if err != nil {
		h.respondServiceError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, clarificationAnswerApplicationResponse{
		Application: application,
		Goal:        goal,
	})
}

func (h *ClarificationHandler) respondServiceError(w http.ResponseWriter, err error) {
	var validationErr *clarification.ValidationError
	switch {
	case errors.As(err, &validationErr):
		RespondError(w, http.StatusBadRequest, "validation_failed", validationErr.Error())
	case errors.Is(err, clarification.ErrAnswerNotFound):
		RespondError(w, http.StatusNotFound, "not_found", "clarification answer not found")
	case errors.Is(err, clarification.ErrGoalNotFound):
		RespondError(w, http.StatusNotFound, "not_found", "goal not found")
	case errors.Is(err, clarification.ErrRequestNotFound):
		RespondError(w, http.StatusNotFound, "not_found", "clarification request not found")
	case errors.Is(err, clarification.ErrInvalidGoalState):
		RespondError(w, http.StatusConflict, "invalid_state", "goal must need clarification")
	case errors.Is(err, clarification.ErrInvalidRequestState):
		RespondError(w, http.StatusConflict, "invalid_state", "clarification request is not open")
	case errors.Is(err, clarification.ErrRequestNotAnswered):
		RespondError(w, http.StatusConflict, "invalid_state", "clarification request is not answered")
	case errors.Is(err, clarification.ErrInvalidAnswerState):
		RespondError(w, http.StatusConflict, "invalid_state", "clarification answer is not recorded")
	case errors.Is(err, clarification.ErrAlreadyOpen):
		RespondError(w, http.StatusConflict, "already_open", "clarification request already open")
	case errors.Is(err, clarification.ErrAlreadyAnswered):
		RespondError(w, http.StatusConflict, "already_answered", "clarification request already answered")
	case errors.Is(err, clarification.ErrAlreadyApplied):
		RespondError(w, http.StatusConflict, "already_applied", "clarification answer already applied")
	case errors.Is(err, clarification.ErrUnsupportedMapping):
		RespondError(w, http.StatusBadRequest, "unsupported_mapping", "unsupported clarification answer mapping")
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

type clarificationAnswerApplicationResponse struct {
	Application spine.ClarificationAnswerApplicationResult `json:"application"`
	Goal        spine.Goal                                 `json:"goal"`
}
