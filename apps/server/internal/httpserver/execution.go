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
	AcquireNextLease(context.Context, spine.ExecutionJobLeaseCreateRequest, spine.OrganizationMembership) (spine.ExecutionJobLeaseCreated, bool, error)
	StartRun(context.Context, spine.ExecutionJobID, spine.RunStartRequest, spine.OrganizationMembership) (spine.Run, bool, error)
	SubmitReceipt(context.Context, spine.RunID, spine.ExecutionReceiptSubmitRequest, spine.OrganizationMembership) (spine.ExecutionReceipt, bool, error)
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

func (h *ExecutionHandler) AcquireLease(w http.ResponseWriter, r *http.Request) {
	profile, err := h.authService.Me(r.Context(), bearerToken(r.Header.Get("Authorization")))
	if err != nil {
		respondAuthError(w, err)
		return
	}
	var input spine.ExecutionJobLeaseCreateRequest
	if err := decodeStrictJSON(r.Body, &input); err != nil {
		respondInvalidJSON(w)
		return
	}
	lease, ok, err := h.service.AcquireNextLease(r.Context(), input, profile.OrganizationMembership)
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	if !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	RespondJSON(w, http.StatusCreated, lease)
}

func (h *ExecutionHandler) StartRun(w http.ResponseWriter, r *http.Request) {
	profile, err := h.authService.Me(r.Context(), bearerToken(r.Header.Get("Authorization")))
	if err != nil {
		respondAuthError(w, err)
		return
	}
	var input spine.RunStartRequest
	if err := decodeStrictJSON(r.Body, &input); err != nil {
		respondInvalidJSON(w)
		return
	}
	run, created, err := h.service.StartRun(r.Context(), spine.ExecutionJobID(r.PathValue("id")), input, profile.OrganizationMembership)
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	RespondJSON(w, status, run)
}

func (h *ExecutionHandler) SubmitReceipt(w http.ResponseWriter, r *http.Request) {
	profile, err := h.authService.Me(r.Context(), bearerToken(r.Header.Get("Authorization")))
	if err != nil {
		respondAuthError(w, err)
		return
	}
	var input spine.ExecutionReceiptSubmitRequest
	if err := decodeStrictJSON(r.Body, &input); err != nil {
		respondInvalidJSON(w)
		return
	}
	receipt, created, err := h.service.SubmitReceipt(r.Context(), spine.RunID(r.PathValue("id")), input, profile.OrganizationMembership)
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	RespondJSON(w, status, receipt)
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
	case errors.Is(err, execution.ErrRepoBindingNotFound):
		RespondError(w, http.StatusNotFound, "not_found", "repo binding not found")
	case errors.Is(err, execution.ErrCheckoutReceiptNotFound):
		RespondError(w, http.StatusNotFound, "not_found", "checkout receipt not found")
	case errors.Is(err, execution.ErrCheckoutJobNotFound):
		RespondError(w, http.StatusNotFound, "not_found", "checkout job not found")
	case errors.Is(err, execution.ErrExecutionJobNotFound):
		RespondError(w, http.StatusNotFound, "not_found", "execution job not found")
	case errors.Is(err, execution.ErrRunNotFound):
		RespondError(w, http.StatusNotFound, "not_found", "run not found")
	case errors.Is(err, execution.ErrInvalidWorkItemState):
		RespondError(w, http.StatusConflict, "invalid_state", "work item is not planned")
	case errors.Is(err, execution.ErrInvalidCheckoutState):
		RespondError(w, http.StatusConflict, "invalid_state", "checkout job receipt has not been submitted")
	case errors.Is(err, execution.ErrInvalidExecutionState):
		RespondError(w, http.StatusConflict, "invalid_state", "execution job state does not allow this transition")
	case errors.Is(err, execution.ErrInvalidRunState):
		RespondError(w, http.StatusConflict, "invalid_state", "run state does not allow this transition")
	case errors.Is(err, execution.ErrLeaseExpired):
		RespondError(w, http.StatusConflict, "lease_expired", "execution job lease expired")
	case errors.Is(err, execution.ErrInvalidLease):
		RespondError(w, http.StatusConflict, "invalid_lease", "execution job lease is invalid")
	case errors.Is(err, execution.ErrRunAlreadyStarted):
		RespondError(w, http.StatusConflict, "already_started", "execution job already has a run")
	case errors.Is(err, execution.ErrRawSourceUploaded):
		RespondError(w, http.StatusBadRequest, "validation_failed", "checkout receipt must not upload raw source")
	case errors.Is(err, execution.ErrExecutionRawSourceUploaded):
		RespondError(w, http.StatusBadRequest, "validation_failed", "execution receipt must not upload raw source")
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
