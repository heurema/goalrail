package httpserver

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/heurema/goalrail/apps/server/internal/qualificationfeed"
	"github.com/heurema/goalrail/apps/server/internal/spine"
)

type QualificationFeedService interface {
	List(context.Context, qualificationfeed.ListInput) (spine.QualificationFeed, error)
}

type QualificationFeedHandler struct {
	authService AuthService
	service     QualificationFeedService
}

func NewQualificationFeedHandler(authService AuthService, service QualificationFeedService) *QualificationFeedHandler {
	return &QualificationFeedHandler{authService: authService, service: service}
}

func (h *QualificationFeedHandler) List(w http.ResponseWriter, r *http.Request) {
	profile, err := h.authService.Me(r.Context(), bearerToken(r.Header.Get("Authorization")))
	if err != nil {
		respondAuthError(w, err)
		return
	}

	input, ok := parseQualificationFeedQuery(w, r)
	if !ok {
		return
	}
	input.Membership = profile.OrganizationMembership

	feed, err := h.service.List(r.Context(), input)
	if err != nil {
		h.respondServiceError(w, err)
		return
	}
	RespondJSON(w, http.StatusOK, feed)
}

func parseQualificationFeedQuery(w http.ResponseWriter, r *http.Request) (qualificationfeed.ListInput, bool) {
	query := r.URL.Query()
	state := strings.TrimSpace(query.Get("state"))
	if state == "" {
		state = strings.TrimSpace(query.Get("goal_state"))
	}
	input := qualificationfeed.ListInput{
		ProjectID:     spine.ProjectID(strings.TrimSpace(query.Get("project_id"))),
		RepoBindingID: spine.RepoBindingID(strings.TrimSpace(query.Get("repo_binding_id"))),
		GoalState:     spine.GoalState(state),
	}
	if rawLimit := strings.TrimSpace(query.Get("limit")); rawLimit != "" {
		limit, err := strconv.Atoi(rawLimit)
		if err != nil {
			RespondError(w, http.StatusBadRequest, "validation_failed", "limit: must be an integer")
			return qualificationfeed.ListInput{}, false
		}
		input.Limit = limit
	}
	return input, true
}

func (h *QualificationFeedHandler) respondServiceError(w http.ResponseWriter, err error) {
	var validationErr *qualificationfeed.ValidationError
	var malformedID spine.MalformedIDError
	switch {
	case errors.As(err, &validationErr):
		RespondError(w, http.StatusBadRequest, "validation_failed", validationErr.Error())
	case errors.As(err, &malformedID):
		RespondError(w, http.StatusBadRequest, "validation_failed", malformedID.Error())
	case errors.Is(err, qualificationfeed.ErrMembershipRequired):
		RespondError(w, http.StatusForbidden, "membership_required", "active organization membership is required")
	default:
		respondInternalError(w)
	}
}
