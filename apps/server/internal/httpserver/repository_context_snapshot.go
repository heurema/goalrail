package httpserver

import (
	"context"
	"errors"
	"net/http"

	"github.com/heurema/goalrail/apps/server/internal/auth"
	"github.com/heurema/goalrail/apps/server/internal/repositorycontext"
	"github.com/heurema/goalrail/apps/server/internal/spine"
)

type RepositoryContextSnapshotService interface {
	RecordSnapshot(context.Context, repositorycontext.RecordInput) (spine.RepositoryContextSnapshotResult, error)
	GetOrganizationRepositoryContext(context.Context, repositorycontext.ReadOrganizationContextInput) (spine.OrganizationRepositoryContextResult, error)
}

type RepositoryContextSnapshotHandler struct {
	authService AuthService
	service     RepositoryContextSnapshotService
}

func NewRepositoryContextSnapshotHandler(authService AuthService, service RepositoryContextSnapshotService) *RepositoryContextSnapshotHandler {
	return &RepositoryContextSnapshotHandler{authService: authService, service: service}
}

func (h *RepositoryContextSnapshotHandler) Record(w http.ResponseWriter, r *http.Request) {
	profile, err := h.authService.Me(r.Context(), bearerToken(r.Header.Get("Authorization")))
	if err != nil {
		respondAuthError(w, err)
		return
	}

	var request spine.RepositoryContextSnapshotRequest
	if err := decodeStrictJSON(r.Body, &request); err != nil {
		respondInvalidJSON(w)
		return
	}

	result, err := h.service.RecordSnapshot(r.Context(), repositorycontext.RecordInput{
		AuthenticatedUserID: profile.User.ID,
		Membership:          profile.OrganizationMembership,
		RepoBindingID:       spine.RepoBindingID(r.PathValue("repo_binding_id")),
		Snapshot:            request,
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

func (h *RepositoryContextSnapshotHandler) GetOrganizationRepositoryContext(w http.ResponseWriter, r *http.Request) {
	profile, err := h.authService.Me(r.Context(), bearerToken(r.Header.Get("Authorization")))
	if err != nil {
		respondAuthError(w, err)
		return
	}

	result, err := h.service.GetOrganizationRepositoryContext(r.Context(), repositorycontext.ReadOrganizationContextInput{
		AuthenticatedUserID: profile.User.ID,
		Membership:          profile.OrganizationMembership,
		OrganizationID:      spine.OrganizationID(r.PathValue("organization_id")),
	})
	if err != nil {
		h.respondServiceError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, result)
}

func (h *RepositoryContextSnapshotHandler) respondServiceError(w http.ResponseWriter, err error) {
	var validationErr *repositorycontext.ValidationError
	switch {
	case errors.As(err, &validationErr):
		RespondError(w, http.StatusBadRequest, "validation_failed", validationErr.Error())
	case errors.Is(err, repositorycontext.ErrForbidden), errors.Is(err, auth.ErrMembershipRequired):
		RespondError(w, http.StatusForbidden, "forbidden", "user is not allowed to access repository context")
	case errors.Is(err, repositorycontext.ErrOrganizationNotFound):
		RespondError(w, http.StatusNotFound, "not_found", "organization not found")
	case errors.Is(err, repositorycontext.ErrRepoBindingNotFound):
		RespondError(w, http.StatusNotFound, "not_found", "repo binding not found")
	case errors.Is(err, repositorycontext.ErrRepoBindingInactive):
		RespondError(w, http.StatusConflict, "repo_binding_inactive", "repo binding is not active")
	case errors.Is(err, repositorycontext.ErrSnapshotMismatch):
		RespondError(w, http.StatusConflict, "repository_context_snapshot_conflict", "repository context snapshot does not match repo binding")
	default:
		respondInternalError(w)
	}
}
