package httpserver

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/heurema/goalrail/apps/server/internal/spine"
	"github.com/heurema/goalrail/apps/server/internal/workitem"
)

type WorkItemService interface {
	PlanContract(context.Context, spine.ContractID) (spine.WorkItem, error)
}

type WorkItemHandler struct {
	service WorkItemService
}

func NewWorkItemHandler(service WorkItemService) *WorkItemHandler {
	return &WorkItemHandler{service: service}
}

func (h *WorkItemHandler) PlanContractTasks(w http.ResponseWriter, r *http.Request) {
	if err := validateOptionalEmptyJSON(r.Body); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid_json", "invalid JSON request body")
		return
	}

	item, err := h.service.PlanContract(r.Context(), spine.ContractID(r.PathValue("id")))
	if err != nil {
		h.respondServiceError(w, err)
		return
	}

	RespondJSON(w, http.StatusCreated, item)
}

func (h *WorkItemHandler) respondServiceError(w http.ResponseWriter, err error) {
	var completenessErr *workitem.CompletenessError
	switch {
	case errors.As(err, &completenessErr):
		RespondError(w, http.StatusBadRequest, "validation_failed", completenessErr.Error())
	case errors.Is(err, workitem.ErrContractNotFound):
		RespondError(w, http.StatusNotFound, "not_found", "contract not found")
	case errors.Is(err, workitem.ErrInvalidContractState):
		RespondError(w, http.StatusConflict, "invalid_state", "contract is not approved")
	case errors.Is(err, workitem.ErrContractMissingApprovedSnapshot):
		RespondError(w, http.StatusConflict, "invalid_state", "contract approved snapshot is missing")
	case errors.Is(err, workitem.ErrApprovedContractNotFound):
		RespondError(w, http.StatusNotFound, "not_found", "approved contract not found")
	case errors.Is(err, workitem.ErrInvalidApprovedContractState):
		RespondError(w, http.StatusConflict, "invalid_state", "approved contract is not approved")
	case errors.Is(err, workitem.ErrAlreadyPlanned):
		RespondError(w, http.StatusConflict, "already_planned", "approved contract already planned")
	default:
		RespondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

func validateOptionalEmptyJSON(body io.Reader) error {
	if body == nil {
		return nil
	}
	contents, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	if len(bytes.TrimSpace(contents)) == 0 {
		return nil
	}
	var input struct{}
	return decodeStrictJSON(bytes.NewReader(contents), &input)
}
