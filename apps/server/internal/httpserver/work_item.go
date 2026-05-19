package httpserver

import (
	"context"
	"errors"
	"net/http"

	"github.com/heurema/goalrail/apps/server/internal/spine"
	"github.com/heurema/goalrail/apps/server/internal/workitem"
)

type WorkItemService interface {
	Get(context.Context, spine.WorkItemID) (spine.WorkItem, error)
	GetDetail(context.Context, spine.WorkItemID, spine.WorkItemDetailRequest, spine.OrganizationMembership) (spine.WorkItemDetail, error)
}

type WorkItemHandler struct {
	authService AuthService
	service     WorkItemService
}

func NewWorkItemHandler(authService AuthService, service WorkItemService) *WorkItemHandler {
	return &WorkItemHandler{authService: authService, service: service}
}

func (h *WorkItemHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	profile, err := h.authService.Me(r.Context(), bearerToken(r.Header.Get("Authorization")))
	if err != nil {
		respondAuthError(w, err)
		return
	}
	query := r.URL.Query()
	input := spine.WorkItemDetailRequest{
		ProjectID:     spine.ProjectID(query.Get("project_id")),
		RepoBindingID: spine.RepoBindingID(query.Get("repo_binding_id")),
	}
	item, err := h.service.GetDetail(r.Context(), spine.WorkItemID(r.PathValue("id")), input, profile.OrganizationMembership)
	if err != nil {
		h.respondServiceError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, item)
}

func (h *WorkItemHandler) respondServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, workitem.ErrWorkItemNotFound):
		RespondError(w, http.StatusNotFound, "not_found", "task not found")
	case errors.Is(err, workitem.ErrMembershipRequired):
		RespondError(w, http.StatusForbidden, "membership_required", "active organization membership is required")
	case errors.Is(err, workitem.ErrOrganizationForbidden):
		RespondError(w, http.StatusForbidden, "forbidden", "user is not allowed to read this work item")
	case errors.Is(err, workitem.ErrProjectMismatch):
		RespondError(w, http.StatusConflict, "project_context_mismatch", "request project does not match work item project")
	case errors.Is(err, workitem.ErrRepoBindingMismatch):
		RespondError(w, http.StatusConflict, "project_context_mismatch", "request repo binding does not match work item repo binding")
	default:
		respondInternalError(w)
	}
}
