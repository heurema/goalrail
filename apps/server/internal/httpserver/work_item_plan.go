package httpserver

import (
	"context"
	"errors"
	"net/http"

	"github.com/heurema/goalrail/apps/server/internal/spine"
	"github.com/heurema/goalrail/apps/server/internal/workitemplan"
)

type WorkItemPlanService interface {
	CreatePlan(context.Context, spine.ContractID, spine.WorkItemPlanCreateRequest, spine.OrganizationMembership) (spine.WorkItemPlan, bool, error)
	GetPlan(context.Context, spine.WorkItemPlanID) (spine.WorkItemPlan, error)
	GetPlanStatus(context.Context, spine.WorkItemPlanID, spine.WorkItemPlanStatusRequest, spine.OrganizationMembership) (spine.WorkItemPlanStatus, error)
	AcquireNextLease(context.Context, spine.WorkItemPlanLeaseCreateRequest) (spine.WorkItemPlanLeaseCreated, bool, error)
	GetLease(context.Context, spine.WorkItemPlanLeaseID) (spine.WorkItemPlanLease, error)
	RenewLease(context.Context, spine.WorkItemPlanLeaseID, spine.WorkItemPlanLeaseRenewRequest) (spine.WorkItemPlanLease, error)
	SubmitProposal(context.Context, spine.WorkItemPlanID, spine.WorkItemPlanProposalSubmitRequest) (spine.WorkItemPlanProposal, error)
	GetProposal(context.Context, spine.WorkItemPlanProposalID) (spine.WorkItemPlanProposal, error)
	AcceptProposal(context.Context, spine.WorkItemPlanProposalID, spine.WorkItemPlanAcceptanceRequest, spine.OrganizationMembership) (spine.WorkItemPlanAcceptanceResult, error)
}

type WorkItemPlanHandler struct {
	authService AuthService
	service     WorkItemPlanService
}

func NewWorkItemPlanHandler(authService AuthService, service WorkItemPlanService) *WorkItemPlanHandler {
	return &WorkItemPlanHandler{authService: authService, service: service}
}

func (h *WorkItemPlanHandler) CreatePlan(w http.ResponseWriter, r *http.Request) {
	profile, err := h.authService.Me(r.Context(), bearerToken(r.Header.Get("Authorization")))
	if err != nil {
		respondAuthError(w, err)
		return
	}

	var input spine.WorkItemPlanCreateRequest
	if err := decodeStrictJSON(r.Body, &input); err != nil {
		respondInvalidJSON(w)
		return
	}
	input.RequestedBy = actorRefForAuthProfile(profile)

	created, newlyCreated, err := h.service.CreatePlan(r.Context(), spine.ContractID(r.PathValue("id")), input, profile.OrganizationMembership)
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	status := http.StatusOK
	if newlyCreated {
		status = http.StatusCreated
	}
	RespondJSON(w, status, created)
}

func (h *WorkItemPlanHandler) GetPlan(w http.ResponseWriter, r *http.Request) {
	plan, err := h.service.GetPlan(r.Context(), spine.WorkItemPlanID(r.PathValue("id")))
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	RespondJSON(w, http.StatusOK, plan)
}

func (h *WorkItemPlanHandler) GetPlanStatus(w http.ResponseWriter, r *http.Request) {
	profile, err := h.authService.Me(r.Context(), bearerToken(r.Header.Get("Authorization")))
	if err != nil {
		respondAuthError(w, err)
		return
	}

	var input spine.WorkItemPlanStatusRequest
	if err := decodeStrictJSON(r.Body, &input); err != nil {
		respondInvalidJSON(w)
		return
	}
	status, err := h.service.GetPlanStatus(r.Context(), spine.WorkItemPlanID(r.PathValue("id")), input, profile.OrganizationMembership)
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	RespondJSON(w, http.StatusOK, status)
}

func (h *WorkItemPlanHandler) AcquireLease(w http.ResponseWriter, r *http.Request) {
	var input spine.WorkItemPlanLeaseCreateRequest
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

func (h *WorkItemPlanHandler) GetLease(w http.ResponseWriter, r *http.Request) {
	lease, err := h.service.GetLease(r.Context(), spine.WorkItemPlanLeaseID(r.PathValue("id")))
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	RespondJSON(w, http.StatusOK, lease)
}

func (h *WorkItemPlanHandler) RenewLease(w http.ResponseWriter, r *http.Request) {
	var input spine.WorkItemPlanLeaseRenewRequest
	if err := decodeStrictJSON(r.Body, &input); err != nil {
		respondInvalidJSON(w)
		return
	}
	lease, err := h.service.RenewLease(r.Context(), spine.WorkItemPlanLeaseID(r.PathValue("id")), input)
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	RespondJSON(w, http.StatusOK, lease)
}

func (h *WorkItemPlanHandler) SubmitProposal(w http.ResponseWriter, r *http.Request) {
	var input spine.WorkItemPlanProposalSubmitRequest
	if err := decodeStrictJSON(r.Body, &input); err != nil {
		respondInvalidJSON(w)
		return
	}
	submitted, err := h.service.SubmitProposal(r.Context(), spine.WorkItemPlanID(r.PathValue("id")), input)
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	RespondJSON(w, http.StatusCreated, submitted)
}

func (h *WorkItemPlanHandler) GetProposal(w http.ResponseWriter, r *http.Request) {
	proposal, err := h.service.GetProposal(r.Context(), spine.WorkItemPlanProposalID(r.PathValue("id")))
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	RespondJSON(w, http.StatusOK, proposal)
}

func (h *WorkItemPlanHandler) AcceptProposal(w http.ResponseWriter, r *http.Request) {
	profile, err := h.authService.Me(r.Context(), bearerToken(r.Header.Get("Authorization")))
	if err != nil {
		respondAuthError(w, err)
		return
	}

	var input spine.WorkItemPlanAcceptanceRequest
	if err := decodeStrictJSON(r.Body, &input); err != nil {
		respondInvalidJSON(w)
		return
	}
	input.AcceptedBy = actorRefForAuthProfile(profile)

	accepted, err := h.service.AcceptProposal(r.Context(), spine.WorkItemPlanProposalID(r.PathValue("id")), input, profile.OrganizationMembership)
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	RespondJSON(w, http.StatusCreated, accepted)
}

func (h *WorkItemPlanHandler) respondServiceError(w http.ResponseWriter, err error) {
	var validationErr *workitemplan.ValidationError
	switch {
	case errors.As(err, &validationErr):
		RespondError(w, http.StatusBadRequest, "validation_failed", validationErr.Error())
	case errors.Is(err, workitemplan.ErrContractNotFound):
		RespondError(w, http.StatusNotFound, "not_found", "contract not found")
	case errors.Is(err, workitemplan.ErrApprovedContractNotFound):
		RespondError(w, http.StatusNotFound, "not_found", "approved contract not found")
	case errors.Is(err, workitemplan.ErrPlanNotFound):
		RespondError(w, http.StatusNotFound, "not_found", "plan not found")
	case errors.Is(err, workitemplan.ErrProposalNotFound):
		RespondError(w, http.StatusNotFound, "not_found", "proposal not found")
	case errors.Is(err, workitemplan.ErrLeaseNotFound):
		RespondError(w, http.StatusNotFound, "not_found", "lease not found")
	case errors.Is(err, workitemplan.ErrInvalidContractState):
		RespondError(w, http.StatusConflict, "invalid_state", "contract is not approved")
	case errors.Is(err, workitemplan.ErrContractMissingApprovedSnapshot):
		RespondError(w, http.StatusConflict, "invalid_state", "contract approved snapshot is missing")
	case errors.Is(err, workitemplan.ErrInvalidApprovedContractState):
		RespondError(w, http.StatusConflict, "invalid_state", "approved contract is not approved")
	case errors.Is(err, workitemplan.ErrInvalidPlanState):
		RespondError(w, http.StatusConflict, "invalid_state", "plan state does not allow this transition")
	case errors.Is(err, workitemplan.ErrInvalidProposalState):
		RespondError(w, http.StatusConflict, "invalid_state", "proposal state does not allow this transition")
	case errors.Is(err, workitemplan.ErrAlreadyPlanned):
		RespondError(w, http.StatusConflict, "already_planned", "contract already has a plan")
	case errors.Is(err, workitemplan.ErrMembershipRequired):
		RespondError(w, http.StatusForbidden, "membership_required", "active organization membership is required")
	case errors.Is(err, workitemplan.ErrOrganizationForbidden):
		RespondError(w, http.StatusForbidden, "forbidden", "user is not allowed to create work item plan for this contract")
	case errors.Is(err, workitemplan.ErrProjectMismatch):
		RespondError(w, http.StatusConflict, "project_context_mismatch", "request project does not match contract project")
	case errors.Is(err, workitemplan.ErrRepoBindingMismatch):
		RespondError(w, http.StatusConflict, "project_context_mismatch", "request repo binding does not match contract repo binding")
	case errors.Is(err, workitemplan.ErrAlreadyProposed):
		RespondError(w, http.StatusConflict, "already_proposed", "plan already has a proposal")
	case errors.Is(err, workitemplan.ErrAlreadyAccepted):
		RespondError(w, http.StatusConflict, "already_accepted", "proposal already accepted")
	case errors.Is(err, workitemplan.ErrLeaseExpired):
		RespondError(w, http.StatusConflict, "lease_expired", "lease expired")
	case errors.Is(err, workitemplan.ErrLeaseCompleted):
		RespondError(w, http.StatusConflict, "lease_completed", "lease completed")
	case errors.Is(err, workitemplan.ErrInvalidLease):
		RespondError(w, http.StatusConflict, "invalid_lease", "lease is invalid")
	default:
		respondInternalError(w)
	}
}
