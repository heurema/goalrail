package httpserver

import (
	"context"
	"errors"
	"net/http"

	"github.com/heurema/goalrail/apps/server/internal/execution"
	"github.com/heurema/goalrail/apps/server/internal/spine"
)

type ExecutionService interface {
	CreateOrReturnJob(context.Context, spine.WorkItemID, spine.ExecutionJobCreateRequest, spine.OrganizationMembership) (spine.ExecutionJob, bool, error)
}

type ExecutionHandler struct {
	authService AuthService
	service     ExecutionService
}

func NewExecutionHandler(authService AuthService, service ExecutionService) *ExecutionHandler {
	return &ExecutionHandler{authService: authService, service: service}
}

func (h *ExecutionHandler) CreateJob(w http.ResponseWriter, r *http.Request) {
	profile, err := h.authService.Me(r.Context(), bearerToken(r.Header.Get("Authorization")))
	if err != nil {
		respondAuthError(w, err)
		return
	}
	var input spine.ExecutionJobCreateRequest
	if err := decodeStrictJSON(r.Body, &input); err != nil {
		respondInvalidJSON(w)
		return
	}
	input.RequestedBy = actorRefForAuthProfile(profile)
	job, created, err := h.service.CreateOrReturnJob(r.Context(), spine.WorkItemID(r.PathValue("id")), input, profile.OrganizationMembership)
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	RespondJSON(w, status, job)
}

func (h *ExecutionHandler) respondServiceError(w http.ResponseWriter, err error) {
	var validationErr *execution.ValidationError
	var malformedID spine.MalformedIDError
	switch {
	case errors.As(err, &validationErr):
		RespondError(w, http.StatusBadRequest, "validation_failed", validationErr.Error())
	case errors.As(err, &malformedID):
		RespondError(w, http.StatusBadRequest, "validation_failed", malformedID.Error())
	case errors.Is(err, execution.ErrWorkItemNotFound):
		RespondError(w, http.StatusNotFound, "not_found", "work item not found")
	case errors.Is(err, execution.ErrCheckoutReceiptNotFound):
		RespondError(w, http.StatusNotFound, "not_found", "checkout receipt not found")
	case errors.Is(err, execution.ErrCheckoutJobNotFound):
		RespondError(w, http.StatusNotFound, "not_found", "checkout job not found")
	case errors.Is(err, execution.ErrInvalidWorkItemState):
		RespondError(w, http.StatusConflict, "invalid_state", "work item is not planned")
	case errors.Is(err, execution.ErrInvalidCheckoutState):
		RespondError(w, http.StatusConflict, "invalid_state", "checkout job receipt has not been submitted")
	case errors.Is(err, execution.ErrRawSourceUploaded):
		RespondError(w, http.StatusBadRequest, "validation_failed", "checkout receipt must not upload raw source")
	case errors.Is(err, execution.ErrMembershipRequired):
		RespondError(w, http.StatusForbidden, "membership_required", "active organization membership is required")
	case errors.Is(err, execution.ErrOrganizationForbidden):
		RespondError(w, http.StatusForbidden, "forbidden", "user is not allowed to prepare execution for this work item")
	case errors.Is(err, execution.ErrProjectMismatch):
		RespondError(w, http.StatusConflict, "project_context_mismatch", "request project does not match work item project")
	case errors.Is(err, execution.ErrRepoBindingMismatch):
		RespondError(w, http.StatusConflict, "project_context_mismatch", "request repo binding does not match work item repo binding")
	case errors.Is(err, execution.ErrCheckoutReceiptMismatch):
		RespondError(w, http.StatusConflict, "checkout_receipt_mismatch", "checkout receipt does not match work item")
	default:
		respondInternalError(w)
	}
}
