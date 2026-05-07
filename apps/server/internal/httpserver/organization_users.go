package httpserver

import (
	"context"
	"errors"
	"net/http"

	"github.com/heurema/goalrail/apps/server/internal/auth"
	"github.com/heurema/goalrail/apps/server/internal/spine"
	"github.com/heurema/goalrail/apps/server/internal/usermanagement"
)

type OrganizationUserManagementService interface {
	ListUsers(context.Context, usermanagement.ListUsersInput) (usermanagement.ListUsersResult, error)
	CreateUser(context.Context, usermanagement.CreateUserInput) (usermanagement.CreateUserResult, error)
	PatchUser(context.Context, usermanagement.PatchUserInput) (usermanagement.PatchUserResult, error)
	ResetTemporaryPassword(context.Context, usermanagement.ResetTemporaryPasswordInput) (usermanagement.ResetTemporaryPasswordResult, error)
}

type OrganizationUsersHandler struct {
	authService AuthService
	service     OrganizationUserManagementService
}

func NewOrganizationUsersHandler(authService AuthService, service OrganizationUserManagementService) *OrganizationUsersHandler {
	return &OrganizationUsersHandler{authService: authService, service: service}
}

func (h *OrganizationUsersHandler) List(w http.ResponseWriter, r *http.Request) {
	profile, err := h.authService.Me(r.Context(), bearerToken(r.Header.Get("Authorization")))
	if err != nil {
		respondAuthError(w, err)
		return
	}
	result, err := h.service.ListUsers(r.Context(), usermanagement.ListUsersInput{
		AuthenticatedUserID: profile.User.ID,
		OrganizationID:      spine.OrganizationID(r.PathValue("organization_id")),
	})
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	RespondJSON(w, http.StatusOK, result)
}

func (h *OrganizationUsersHandler) Create(w http.ResponseWriter, r *http.Request) {
	profile, err := h.authService.Me(r.Context(), bearerToken(r.Header.Get("Authorization")))
	if err != nil {
		respondAuthError(w, err)
		return
	}
	var request createOrganizationUserRequest
	if err := decodeStrictJSON(r.Body, &request); err != nil {
		respondInvalidJSON(w)
		return
	}
	result, err := h.service.CreateUser(r.Context(), usermanagement.CreateUserInput{
		AuthenticatedUserID: profile.User.ID,
		OrganizationID:      spine.OrganizationID(r.PathValue("organization_id")),
		Email:               request.Email,
		DisplayName:         request.DisplayName,
		Role:                request.Role,
	})
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	RespondJSON(w, http.StatusCreated, result)
}

func (h *OrganizationUsersHandler) Patch(w http.ResponseWriter, r *http.Request) {
	profile, err := h.authService.Me(r.Context(), bearerToken(r.Header.Get("Authorization")))
	if err != nil {
		respondAuthError(w, err)
		return
	}
	var request patchOrganizationUserRequest
	if err := decodeStrictJSON(r.Body, &request); err != nil {
		respondInvalidJSON(w)
		return
	}
	result, err := h.service.PatchUser(r.Context(), usermanagement.PatchUserInput{
		AuthenticatedUserID: profile.User.ID,
		OrganizationID:      spine.OrganizationID(r.PathValue("organization_id")),
		UserID:              spine.UserID(r.PathValue("user_id")),
		DisplayName:         request.DisplayName,
		Role:                request.Role,
		State:               request.State,
	})
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	RespondJSON(w, http.StatusOK, result)
}

func (h *OrganizationUsersHandler) ResetTemporaryPassword(w http.ResponseWriter, r *http.Request) {
	profile, err := h.authService.Me(r.Context(), bearerToken(r.Header.Get("Authorization")))
	if err != nil {
		respondAuthError(w, err)
		return
	}
	result, err := h.service.ResetTemporaryPassword(r.Context(), usermanagement.ResetTemporaryPasswordInput{
		AuthenticatedUserID: profile.User.ID,
		OrganizationID:      spine.OrganizationID(r.PathValue("organization_id")),
		UserID:              spine.UserID(r.PathValue("user_id")),
	})
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	RespondJSON(w, http.StatusCreated, result)
}

type createOrganizationUserRequest struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
}

type patchOrganizationUserRequest struct {
	DisplayName *string `json:"display_name"`
	Role        *string `json:"role"`
	State       *string `json:"state"`
}

func (h *OrganizationUsersHandler) respondServiceError(w http.ResponseWriter, err error) {
	var validationErr *usermanagement.ValidationError
	switch {
	case errors.As(err, &validationErr):
		RespondError(w, http.StatusBadRequest, "validation_failed", validationErr.Error())
	case errors.Is(err, usermanagement.ErrForbidden):
		RespondError(w, http.StatusForbidden, "forbidden", "user is not allowed to manage users in this organization")
	case errors.Is(err, usermanagement.ErrNotFound):
		RespondError(w, http.StatusNotFound, "not_found", "organization user not found")
	case errors.Is(err, usermanagement.ErrUserExists):
		RespondError(w, http.StatusConflict, "organization_user_exists", "organization user already exists")
	case errors.Is(err, usermanagement.ErrLastActiveOwner):
		RespondError(w, http.StatusConflict, "last_active_owner", "last active owner cannot be disabled or demoted")
	case errors.Is(err, usermanagement.ErrSelfActionForbidden):
		RespondError(w, http.StatusConflict, "self_action_forbidden", "self user-management action is not allowed from this admin surface")
	case errors.Is(err, auth.ErrMembershipRequired):
		RespondError(w, http.StatusForbidden, "membership_required", "active organization membership is required")
	default:
		respondInternalError(w)
	}
}
