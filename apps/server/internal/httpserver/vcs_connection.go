package httpserver

import (
	"context"
	"errors"
	"net/http"

	"github.com/heurema/goalrail/apps/server/internal/auth"
	"github.com/heurema/goalrail/apps/server/internal/spine"
	"github.com/heurema/goalrail/apps/server/internal/vcsconnection"
)

type VcsConnectionService interface {
	CreatePendingSetup(context.Context, vcsconnection.CreateInput) (spine.VcsConnection, error)
	Get(context.Context, vcsconnection.GetInput) (spine.VcsConnection, error)
}

type VcsConnectionHandler struct {
	authService AuthService
	service     VcsConnectionService
}

func NewVcsConnectionHandler(authService AuthService, service VcsConnectionService) *VcsConnectionHandler {
	return &VcsConnectionHandler{authService: authService, service: service}
}

func (h *VcsConnectionHandler) Create(w http.ResponseWriter, r *http.Request) {
	profile, err := h.authService.Me(r.Context(), bearerToken(r.Header.Get("Authorization")))
	if err != nil {
		respondAuthError(w, err)
		return
	}

	var request spine.VcsConnectionCreateRequest
	if err := decodeStrictJSON(r.Body, &request); err != nil {
		respondInvalidJSON(w)
		return
	}

	connection, err := h.service.CreatePendingSetup(r.Context(), vcsconnection.CreateInput{
		AuthenticatedUserID: profile.User.ID,
		Membership:          profile.OrganizationMembership,
		ProviderKind:        request.ProviderKind,
		ProviderInstanceURL: request.ProviderInstanceURL,
	})
	if err != nil {
		h.respondServiceError(w, err)
		return
	}

	RespondJSON(w, http.StatusCreated, connection)
}

func (h *VcsConnectionHandler) Get(w http.ResponseWriter, r *http.Request) {
	profile, err := h.authService.Me(r.Context(), bearerToken(r.Header.Get("Authorization")))
	if err != nil {
		respondAuthError(w, err)
		return
	}

	connection, err := h.service.Get(r.Context(), vcsconnection.GetInput{
		VcsConnectionID: spine.VcsConnectionID(r.PathValue("id")),
		Membership:      profile.OrganizationMembership,
	})
	if err != nil {
		h.respondServiceError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, connection)
}

func (h *VcsConnectionHandler) respondServiceError(w http.ResponseWriter, err error) {
	var validationErr *vcsconnection.ValidationError
	switch {
	case errors.As(err, &validationErr):
		RespondError(w, http.StatusBadRequest, "validation_failed", validationErr.Error())
	case errors.Is(err, vcsconnection.ErrForbidden), errors.Is(err, auth.ErrMembershipRequired):
		RespondError(w, http.StatusForbidden, "forbidden", "user is not allowed to manage VCS connections for this organization")
	case errors.Is(err, vcsconnection.ErrNotFound):
		RespondError(w, http.StatusNotFound, "not_found", "VCS connection not found")
	default:
		respondInternalError(w)
	}
}
