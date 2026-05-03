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
}

type WorkItemHandler struct {
	service WorkItemService
}

func NewWorkItemHandler(service WorkItemService) *WorkItemHandler {
	return &WorkItemHandler{service: service}
}

func (h *WorkItemHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.Get(r.Context(), spine.WorkItemID(r.PathValue("id")))
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
	default:
		respondInternalError(w)
	}
}
