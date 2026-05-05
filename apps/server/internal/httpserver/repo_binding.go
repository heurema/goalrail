package httpserver

import (
	"context"
	"errors"
	"net/http"

	"github.com/heurema/goalrail/apps/server/internal/auth"
	"github.com/heurema/goalrail/apps/server/internal/repobinding"
	"github.com/heurema/goalrail/apps/server/internal/spine"
)

type RepoBindingInitService interface {
	Init(context.Context, repobinding.InitInput) (spine.RepoBindingInitResult, error)
}

type RepoBindingHandler struct {
	authService AuthService
	service     RepoBindingInitService
}

func NewRepoBindingHandler(authService AuthService, service RepoBindingInitService) *RepoBindingHandler {
	return &RepoBindingHandler{authService: authService, service: service}
}

func (h *RepoBindingHandler) Init(w http.ResponseWriter, r *http.Request) {
	profile, err := h.authService.Me(r.Context(), bearerToken(r.Header.Get("Authorization")))
	if err != nil {
		respondAuthError(w, err)
		return
	}

	var request spine.RepoBindingInitRequest
	if err := decodeStrictJSON(r.Body, &request); err != nil {
		respondInvalidJSON(w)
		return
	}

	result, err := h.service.Init(r.Context(), repobinding.InitInput{
		ProjectID:             spine.ProjectID(r.PathValue("project_id")),
		AuthenticatedUserID:   profile.User.ID,
		Membership:            profile.OrganizationMembership,
		Provider:              request.Provider,
		RepositoryFullName:    request.RepositoryFullName,
		RepositoryURL:         request.RepositoryURL,
		ProviderDefaultBranch: request.ProviderDefaultBranch,
		WorkflowBaseBranch:    request.WorkflowBaseBranch,
		LocalRemoteName:       request.LocalRemoteName,
		LocalHeadSHA:          request.LocalHeadSHA,
	})
	if err != nil {
		h.respondServiceError(w, err)
		return
	}

	status := http.StatusOK
	if result.Created {
		status = http.StatusCreated
	}
	RespondJSON(w, status, result)
}

func (h *RepoBindingHandler) respondServiceError(w http.ResponseWriter, err error) {
	var validationErr *repobinding.ValidationError
	switch {
	case errors.As(err, &validationErr):
		RespondError(w, http.StatusBadRequest, "validation_failed", validationErr.Error())
	case errors.Is(err, repobinding.ErrProjectNotFound):
		RespondError(w, http.StatusNotFound, "not_found", "project not found")
	case errors.Is(err, repobinding.ErrForbidden):
		RespondError(w, http.StatusForbidden, "forbidden", "user is not allowed to initialize repo binding for this project")
	case errors.Is(err, repobinding.ErrDifferentRepoBinding):
		RespondError(w, http.StatusConflict, "repo_binding_conflict", "project already has active repo binding for a different repository")
	case errors.Is(err, repobinding.ErrRepositoryAlreadyBound):
		RespondError(w, http.StatusConflict, "repo_binding_conflict", "organization already has active repo binding for this repository")
	default:
		respondInternalError(w)
	}
}

func respondAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, auth.ErrJWTSecretMissing), errors.Is(err, auth.ErrJWTSecretWeak):
		RespondError(w, http.StatusServiceUnavailable, "auth_not_configured", "auth JWT secret is not configured")
	case errors.Is(err, auth.ErrInvalidToken), errors.Is(err, auth.ErrExpiredToken), errors.Is(err, auth.ErrSessionInvalid):
		RespondError(w, http.StatusUnauthorized, "unauthorized", "valid bearer token is required")
	case errors.Is(err, auth.ErrInactiveUser):
		RespondError(w, http.StatusForbidden, "inactive_user", "user is inactive")
	case errors.Is(err, auth.ErrMembershipRequired):
		RespondError(w, http.StatusForbidden, "membership_required", "active organization membership is required")
	default:
		respondInternalError(w)
	}
}
