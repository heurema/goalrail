package httpserver

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/heurema/goalrail/apps/server/internal/actor"
	"github.com/heurema/goalrail/apps/server/internal/approvedcontract"
	"github.com/heurema/goalrail/apps/server/internal/auth"
	"github.com/heurema/goalrail/apps/server/internal/contract"
	"github.com/heurema/goalrail/apps/server/internal/contractdraft"
	"github.com/heurema/goalrail/apps/server/internal/contractseed"
	"github.com/heurema/goalrail/apps/server/internal/spine"
)

type ContractService interface {
	Create(context.Context, spine.ContractCreateRequest, spine.OrganizationMembership) (spine.Contract, bool, error)
	Get(context.Context, spine.ContractID, spine.OrganizationMembership) (spine.Contract, error)
	List(context.Context, contract.ListInput) (spine.ContractList, error)
	CurrentDraft(context.Context, spine.ContractID, spine.OrganizationMembership) (spine.ContractDraft, error)
	UpdateDraft(context.Context, spine.ContractID, spine.ContractDraftUpdateRequest, spine.OrganizationMembership) (spine.Contract, error)
	SubmitForApproval(context.Context, spine.ContractID, spine.ContractDraftReadyForApprovalRequest, spine.OrganizationMembership) (spine.Contract, error)
	Approve(context.Context, spine.ContractID, spine.ApproveContractDraftRequest, spine.OrganizationMembership) (spine.Contract, error)
}

type ContractHandler struct {
	authService AuthService
	service     ContractService
}

func NewContractHandler(authService AuthService, service ContractService) *ContractHandler {
	return &ContractHandler{authService: authService, service: service}
}

func (h *ContractHandler) Create(w http.ResponseWriter, r *http.Request) {
	profile, err := h.authService.Me(r.Context(), bearerToken(r.Header.Get("Authorization")))
	if err != nil {
		respondAuthError(w, err)
		return
	}

	var input spine.ContractCreateRequest
	if err := decodeStrictJSON(r.Body, &input); err != nil {
		respondInvalidJSON(w)
		return
	}

	created, newlyCreated, err := h.service.Create(r.Context(), input, profile.OrganizationMembership)
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

func (h *ContractHandler) List(w http.ResponseWriter, r *http.Request) {
	profile, err := h.authService.Me(r.Context(), bearerToken(r.Header.Get("Authorization")))
	if err != nil {
		respondAuthError(w, err)
		return
	}

	input, ok := parseContractListQuery(w, r)
	if !ok {
		return
	}
	input.Membership = profile.OrganizationMembership

	result, err := h.service.List(r.Context(), input)
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	RespondJSON(w, http.StatusOK, result)
}

func (h *ContractHandler) Get(w http.ResponseWriter, r *http.Request) {
	profile, err := h.authService.Me(r.Context(), bearerToken(r.Header.Get("Authorization")))
	if err != nil {
		respondAuthError(w, err)
		return
	}

	contract, err := h.service.Get(r.Context(), spine.ContractID(r.PathValue("id")), profile.OrganizationMembership)
	if err != nil {
		h.respondServiceError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, contract)
}

func (h *ContractHandler) CurrentDraft(w http.ResponseWriter, r *http.Request) {
	profile, err := h.authService.Me(r.Context(), bearerToken(r.Header.Get("Authorization")))
	if err != nil {
		respondAuthError(w, err)
		return
	}

	draft, err := h.service.CurrentDraft(r.Context(), spine.ContractID(r.PathValue("id")), profile.OrganizationMembership)
	if err != nil {
		h.respondServiceError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, draft)
}

func (h *ContractHandler) UpdateDraft(w http.ResponseWriter, r *http.Request) {
	profile, err := h.authService.Me(r.Context(), bearerToken(r.Header.Get("Authorization")))
	if err != nil {
		respondAuthError(w, err)
		return
	}

	var input spine.ContractDraftUpdateRequest
	if err := decodeStrictJSON(r.Body, &input); err != nil {
		respondInvalidJSON(w)
		return
	}
	input.UpdatedBy = spine.ActorRef{
		Kind:        "user",
		ID:          string(profile.User.ID),
		DisplayName: profile.User.DisplayName,
	}

	updated, err := h.service.UpdateDraft(r.Context(), spine.ContractID(r.PathValue("id")), input, profile.OrganizationMembership)
	if err != nil {
		h.respondServiceError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, updated)
}

func (h *ContractHandler) SubmitForApproval(w http.ResponseWriter, r *http.Request) {
	profile, err := h.authService.Me(r.Context(), bearerToken(r.Header.Get("Authorization")))
	if err != nil {
		respondAuthError(w, err)
		return
	}

	var input spine.ContractDraftReadyForApprovalRequest
	if err := decodeStrictJSON(r.Body, &input); err != nil {
		respondInvalidJSON(w)
		return
	}
	input.MarkedBy = actorRefForAuthProfile(profile)

	updated, err := h.service.SubmitForApproval(r.Context(), spine.ContractID(r.PathValue("id")), input, profile.OrganizationMembership)
	if err != nil {
		h.respondServiceError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, updated)
}

func (h *ContractHandler) Approve(w http.ResponseWriter, r *http.Request) {
	profile, err := h.authService.Me(r.Context(), bearerToken(r.Header.Get("Authorization")))
	if err != nil {
		respondAuthError(w, err)
		return
	}

	var input spine.ApproveContractDraftRequest
	if err := decodeStrictJSON(r.Body, &input); err != nil {
		respondInvalidJSON(w)
		return
	}
	input.ApprovedBy = actorRefForAuthProfile(profile)

	ctx := actor.WithActor(r.Context(), actor.ActorContext{
		Actor:  input.ApprovedBy,
		Source: actor.SourceService,
	})
	approved, err := h.service.Approve(ctx, spine.ContractID(r.PathValue("id")), input, profile.OrganizationMembership)
	if err != nil {
		h.respondServiceError(w, err)
		return
	}

	RespondJSON(w, http.StatusCreated, approved)
}

func parseContractListQuery(w http.ResponseWriter, r *http.Request) (contract.ListInput, bool) {
	query := r.URL.Query()
	input := contract.ListInput{
		ProjectID:     spine.ProjectID(strings.TrimSpace(query.Get("project_id"))),
		RepoBindingID: spine.RepoBindingID(strings.TrimSpace(query.Get("repo_binding_id"))),
		GoalID:        spine.GoalID(strings.TrimSpace(query.Get("goal_id"))),
		State:         spine.ContractState(strings.TrimSpace(query.Get("state"))),
	}
	if rawLimit := strings.TrimSpace(query.Get("limit")); rawLimit != "" {
		limit, err := strconv.Atoi(rawLimit)
		if err != nil {
			RespondError(w, http.StatusBadRequest, "validation_failed", "limit: must be an integer")
			return contract.ListInput{}, false
		}
		input.Limit = limit
	}
	return input, true
}

func actorRefForAuthProfile(profile auth.Profile) spine.ActorRef {
	return spine.ActorRef{
		Kind:        "user",
		ID:          string(profile.User.ID),
		DisplayName: profile.User.DisplayName,
	}
}

func (h *ContractHandler) respondServiceError(w http.ResponseWriter, err error) {
	var contractValidationErr *contract.ValidationError
	var seedValidationErr *contractseed.ValidationError
	var draftValidationErr *contractdraft.ValidationError
	var approvedValidationErr *approvedcontract.ValidationError
	var unknownFieldErr *contractdraft.UnknownFieldError
	var nonEditableFieldErr *contractdraft.NonEditableFieldError
	var draftCompletenessErr *contractdraft.CompletenessError
	var approvedCompletenessErr *approvedcontract.CompletenessError
	switch {
	case errors.As(err, &contractValidationErr):
		RespondError(w, http.StatusBadRequest, "validation_failed", contractValidationErr.Error())
	case errors.As(err, &seedValidationErr):
		RespondError(w, http.StatusBadRequest, "validation_failed", seedValidationErr.Error())
	case errors.As(err, &draftValidationErr):
		RespondError(w, http.StatusBadRequest, "validation_failed", draftValidationErr.Error())
	case errors.As(err, &approvedValidationErr):
		RespondError(w, http.StatusBadRequest, "validation_failed", approvedValidationErr.Error())
	case errors.As(err, &unknownFieldErr):
		RespondError(w, http.StatusBadRequest, "unknown_field", unknownFieldErr.Error())
	case errors.As(err, &nonEditableFieldErr):
		RespondError(w, http.StatusBadRequest, "non_editable_field", nonEditableFieldErr.Error())
	case errors.As(err, &draftCompletenessErr):
		RespondError(w, http.StatusBadRequest, "validation_failed", draftCompletenessErr.Error())
	case errors.As(err, &approvedCompletenessErr):
		RespondError(w, http.StatusBadRequest, "validation_failed", approvedCompletenessErr.Error())
	case errors.Is(err, contract.ErrContractNotFound):
		RespondError(w, http.StatusNotFound, "not_found", "contract not found")
	case errors.Is(err, contract.ErrGoalNotFound):
		RespondError(w, http.StatusNotFound, "not_found", "goal not found")
	case errors.Is(err, contractseed.ErrGoalNotFound):
		RespondError(w, http.StatusNotFound, "not_found", "goal not found")
	case errors.Is(err, contractdraft.ErrContractSeedNotFound):
		RespondError(w, http.StatusNotFound, "not_found", "contract seed not found")
	case errors.Is(err, contractdraft.ErrContractDraftNotFound):
		RespondError(w, http.StatusNotFound, "not_found", "contract draft not found")
	case errors.Is(err, approvedcontract.ErrContractDraftNotFound):
		RespondError(w, http.StatusNotFound, "not_found", "contract draft not found")
	case errors.Is(err, contractseed.ErrInvalidGoalState):
		RespondError(w, http.StatusConflict, "invalid_state", "goal is not ready for contract")
	case errors.Is(err, contract.ErrInvalidGoalState):
		RespondError(w, http.StatusConflict, "invalid_state", "goal is not ready for contract")
	case errors.Is(err, contract.ErrInvalidContractState):
		RespondError(w, http.StatusConflict, "invalid_state", "contract state does not allow this transition")
	case errors.Is(err, contract.ErrContractCurrentDraftMissing):
		RespondError(w, http.StatusConflict, "invalid_state", "contract current draft is missing")
	case errors.Is(err, contract.ErrContractCurrentDraftMismatch):
		RespondError(w, http.StatusConflict, "invalid_state", "contract current draft does not belong to contract")
	case errors.Is(err, contractdraft.ErrInvalidSeedState):
		RespondError(w, http.StatusConflict, "invalid_state", "contract seed is not ready for contract draft")
	case errors.Is(err, contractdraft.ErrInvalidDraftState):
		RespondError(w, http.StatusConflict, "invalid_state", "contract draft is not in the required state")
	case errors.Is(err, approvedcontract.ErrInvalidDraftState):
		RespondError(w, http.StatusConflict, "invalid_state", "contract draft is not ready for approval")
	case errors.Is(err, contractseed.ErrAlreadySeeded):
		RespondError(w, http.StatusConflict, "already_seeded", "goal already has contract")
	case errors.Is(err, contract.ErrMembershipRequired):
		RespondError(w, http.StatusForbidden, "membership_required", "active organization membership is required")
	case errors.Is(err, contract.ErrOrganizationForbidden):
		RespondError(w, http.StatusForbidden, "forbidden", "user is not allowed to create contract for this goal")
	case errors.Is(err, contract.ErrProjectMismatch):
		RespondError(w, http.StatusConflict, "project_context_mismatch", "request project does not match goal project")
	case errors.Is(err, contract.ErrRepoBindingMismatch):
		RespondError(w, http.StatusConflict, "project_context_mismatch", "request repo binding does not match goal repo binding")
	case errors.Is(err, contractdraft.ErrAlreadyDrafted):
		RespondError(w, http.StatusConflict, "already_drafted", "contract already has draft")
	case errors.Is(err, contract.ErrAlreadyApproved):
		RespondError(w, http.StatusConflict, "already_approved", "contract already approved")
	case errors.Is(err, approvedcontract.ErrAlreadyApproved):
		RespondError(w, http.StatusConflict, "already_approved", "contract draft already approved")
	default:
		respondInternalError(w)
	}
}
