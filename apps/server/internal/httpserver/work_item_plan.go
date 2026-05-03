package httpserver

import (
	"context"
	"errors"
	"net/http"

	"github.com/heurema/goalrail/apps/server/internal/spine"
	"github.com/heurema/goalrail/apps/server/internal/workitemplan"
)

type WorkItemPlanService interface {
	CreatePlan(context.Context, spine.ContractID, spine.WorkItemPlanCreateRequest) (spine.WorkItemPlan, error)
	GetPlan(context.Context, spine.WorkItemPlanID) (spine.WorkItemPlan, error)
	SubmitProposal(context.Context, spine.WorkItemPlanID, spine.WorkItemPlanProposalSubmitRequest) (spine.WorkItemPlanProposal, error)
	GetProposal(context.Context, spine.WorkItemPlanProposalID) (spine.WorkItemPlanProposal, error)
	AcceptProposal(context.Context, spine.WorkItemPlanProposalID, spine.WorkItemPlanAcceptanceRequest) (spine.WorkItemPlanAcceptanceResult, error)
}

type WorkItemPlanHandler struct {
	service WorkItemPlanService
}

func NewWorkItemPlanHandler(service WorkItemPlanService) *WorkItemPlanHandler {
	return &WorkItemPlanHandler{service: service}
}

func (h *WorkItemPlanHandler) CreatePlan(w http.ResponseWriter, r *http.Request) {
	var input spine.WorkItemPlanCreateRequest
	if err := decodeStrictJSON(r.Body, &input); err != nil {
		respondInvalidJSON(w)
		return
	}
	created, err := h.service.CreatePlan(r.Context(), spine.ContractID(r.PathValue("id")), input)
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	RespondJSON(w, http.StatusCreated, created)
}

func (h *WorkItemPlanHandler) GetPlan(w http.ResponseWriter, r *http.Request) {
	plan, err := h.service.GetPlan(r.Context(), spine.WorkItemPlanID(r.PathValue("id")))
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	RespondJSON(w, http.StatusOK, plan)
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
	var input spine.WorkItemPlanAcceptanceRequest
	if err := decodeStrictJSON(r.Body, &input); err != nil {
		respondInvalidJSON(w)
		return
	}
	accepted, err := h.service.AcceptProposal(r.Context(), spine.WorkItemPlanProposalID(r.PathValue("id")), input)
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
	case errors.Is(err, workitemplan.ErrAlreadyProposed):
		RespondError(w, http.StatusConflict, "already_proposed", "plan already has a proposal")
	case errors.Is(err, workitemplan.ErrAlreadyAccepted):
		RespondError(w, http.StatusConflict, "already_accepted", "proposal already accepted")
	default:
		respondInternalError(w)
	}
}
