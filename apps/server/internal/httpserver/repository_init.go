package httpserver

import (
	"context"
	"errors"
	"net/http"

	"github.com/heurema/goalrail/apps/server/internal/auth"
	"github.com/heurema/goalrail/apps/server/internal/repobinding"
	"github.com/heurema/goalrail/apps/server/internal/repositoryinit"
	"github.com/heurema/goalrail/apps/server/internal/spine"
)

type RepositoryContextInitService interface {
	Init(context.Context, repositoryinit.InitInput) (spine.RepositoryContextInitResult, error)
}

type RepositoryInitHandler struct {
	authService AuthService
	service     RepositoryContextInitService
}

func NewRepositoryInitHandler(authService AuthService, service RepositoryContextInitService) *RepositoryInitHandler {
	return &RepositoryInitHandler{authService: authService, service: service}
}

func (h *RepositoryInitHandler) Init(w http.ResponseWriter, r *http.Request) {
	profile, err := h.authService.Me(r.Context(), bearerToken(r.Header.Get("Authorization")))
	if err != nil {
		respondAuthError(w, err)
		return
	}

	var request spine.RepositoryContextInitRequest
	if err := decodeStrictJSON(r.Body, &request); err != nil {
		respondInvalidJSON(w)
		return
	}

	result, err := h.service.Init(r.Context(), repositoryinit.InitInput{
		AuthenticatedUserID:         profile.User.ID,
		Membership:                  profile.OrganizationMembership,
		Provider:                    request.Provider,
		RepositoryFullName:          request.RepositoryFullName,
		RepositoryURL:               request.RepositoryURL,
		ProviderDefaultBranch:       request.ProviderDefaultBranch,
		WorkflowBaseBranch:          request.WorkflowBaseBranch,
		LocalRemoteName:             request.LocalRemoteName,
		LocalHeadSHA:                request.LocalHeadSHA,
		SuggestedProjectSlug:        request.SuggestedProjectSlug,
		SuggestedProjectDisplayName: request.SuggestedProjectDisplayName,
	})
	if err != nil {
		h.respondServiceError(w, err)
		return
	}

	status := http.StatusOK
	if result.ProjectCreated || result.RepoBindingCreated {
		status = http.StatusCreated
	}
	RespondJSON(w, status, result)
}

func (h *RepositoryInitHandler) respondServiceError(w http.ResponseWriter, err error) {
	var validationErr *repositoryinit.ValidationError
	switch {
	case errors.As(err, &validationErr):
		RespondError(w, http.StatusBadRequest, "validation_failed", validationErr.Error())
	case errors.Is(err, repositoryinit.ErrMembershipRequired), errors.Is(err, auth.ErrMembershipRequired):
		RespondError(w, http.StatusForbidden, "membership_required", "active organization membership is required; self-hosted servers must be bootstrapped with goalrail-server bootstrap owner")
	case errors.Is(err, repositoryinit.ErrForbidden), errors.Is(err, repobinding.ErrForbidden):
		RespondError(w, http.StatusForbidden, "forbidden", "user is not allowed to initialize repository context")
	case errors.Is(err, repositoryinit.ErrProjectSlugConflict):
		RespondError(w, http.StatusConflict, "project_slug_conflict", "project slug is already bound to a different repository")
	case errors.Is(err, repositoryinit.ErrProjectSlugUnavailable):
		RespondError(w, http.StatusConflict, "project_slug_conflict", "project slug is already used by an inactive project")
	case errors.Is(err, repobinding.ErrDifferentRepoBinding):
		RespondError(w, http.StatusConflict, "repo_binding_conflict", "project already has active repo binding for a different repository")
	case errors.Is(err, repobinding.ErrRepositoryAlreadyBound):
		RespondError(w, http.StatusConflict, "repo_binding_conflict", "organization already has active repo binding for this repository")
	case errors.Is(err, repositoryinit.ErrProjectForBindingNotFound), errors.Is(err, repobinding.ErrProjectNotFound):
		RespondError(w, http.StatusConflict, "repository_context_conflict", "existing repository binding does not resolve to an active project")
	default:
		respondInternalError(w)
	}
}
