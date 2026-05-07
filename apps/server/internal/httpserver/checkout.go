package httpserver

import (
	"context"
	"errors"
	"net/http"

	"github.com/heurema/goalrail/apps/server/internal/checkout"
	"github.com/heurema/goalrail/apps/server/internal/spine"
)

type CheckoutService interface {
	CreateOrReturnJob(context.Context, spine.WorkItemID, spine.CheckoutJobCreateRequest, spine.OrganizationMembership) (spine.CheckoutJob, bool, error)
	AcquireNextLease(context.Context, spine.CheckoutJobLeaseCreateRequest) (spine.CheckoutJobLeaseCreated, bool, error)
	SubmitReceipt(context.Context, spine.CheckoutJobID, spine.CheckoutReceiptSubmitRequest) (spine.CheckoutReceipt, error)
}

type CheckoutHandler struct {
	authService AuthService
	service     CheckoutService
}

func NewCheckoutHandler(authService AuthService, service CheckoutService) *CheckoutHandler {
	return &CheckoutHandler{authService: authService, service: service}
}

func (h *CheckoutHandler) CreateJob(w http.ResponseWriter, r *http.Request) {
	profile, err := h.authService.Me(r.Context(), bearerToken(r.Header.Get("Authorization")))
	if err != nil {
		respondAuthError(w, err)
		return
	}
	var input spine.CheckoutJobCreateRequest
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

func (h *CheckoutHandler) AcquireLease(w http.ResponseWriter, r *http.Request) {
	var input spine.CheckoutJobLeaseCreateRequest
	if err := decodeStrictJSON(r.Body, &input); err != nil {
		respondInvalidJSON(w)
		return
	}
	lease, ok, err := h.service.AcquireNextLease(r.Context(), input)
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

func (h *CheckoutHandler) SubmitReceipt(w http.ResponseWriter, r *http.Request) {
	var input spine.CheckoutReceiptSubmitRequest
	if err := decodeStrictJSON(r.Body, &input); err != nil {
		respondInvalidJSON(w)
		return
	}
	receipt, err := h.service.SubmitReceipt(r.Context(), spine.CheckoutJobID(r.PathValue("id")), input)
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	RespondJSON(w, http.StatusCreated, receipt)
}

func (h *CheckoutHandler) respondServiceError(w http.ResponseWriter, err error) {
	var validationErr *checkout.ValidationError
	switch {
	case errors.As(err, &validationErr):
		RespondError(w, http.StatusBadRequest, "validation_failed", validationErr.Error())
	case errors.Is(err, checkout.ErrWorkItemNotFound):
		RespondError(w, http.StatusNotFound, "not_found", "work item not found")
	case errors.Is(err, checkout.ErrRepoBindingNotFound):
		RespondError(w, http.StatusNotFound, "not_found", "repo binding not found")
	case errors.Is(err, checkout.ErrCheckoutJobNotFound):
		RespondError(w, http.StatusNotFound, "not_found", "checkout job not found")
	case errors.Is(err, checkout.ErrInvalidWorkItemState):
		RespondError(w, http.StatusConflict, "invalid_state", "work item is not planned")
	case errors.Is(err, checkout.ErrInvalidCheckoutState):
		RespondError(w, http.StatusConflict, "invalid_state", "checkout job state does not allow this transition")
	case errors.Is(err, checkout.ErrAlreadyReceipted):
		RespondError(w, http.StatusConflict, "already_receipted", "checkout job already has a receipt")
	case errors.Is(err, checkout.ErrLeaseExpired):
		RespondError(w, http.StatusConflict, "lease_expired", "checkout job lease expired")
	case errors.Is(err, checkout.ErrInvalidLease):
		RespondError(w, http.StatusConflict, "invalid_lease", "checkout job lease is invalid")
	case errors.Is(err, checkout.ErrRawSourceUploaded):
		RespondError(w, http.StatusBadRequest, "validation_failed", "checkout receipt must not upload raw source")
	case errors.Is(err, checkout.ErrMembershipRequired):
		RespondError(w, http.StatusForbidden, "membership_required", "active organization membership is required")
	case errors.Is(err, checkout.ErrOrganizationForbidden):
		RespondError(w, http.StatusForbidden, "forbidden", "user is not allowed to prepare checkout for this work item")
	case errors.Is(err, checkout.ErrProjectMismatch):
		RespondError(w, http.StatusConflict, "project_context_mismatch", "request project does not match work item project")
	case errors.Is(err, checkout.ErrRepoBindingMismatch):
		RespondError(w, http.StatusConflict, "project_context_mismatch", "request repo binding does not match work item repo binding")
	default:
		respondInternalError(w)
	}
}
