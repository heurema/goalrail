package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/heurema/goalrail/apps/server/internal/intake"
	"github.com/heurema/goalrail/apps/server/internal/spine"
)

const intakeNextMessage = "server will validate and may promote intake to goal"

type IntakeService interface {
	Submit(context.Context, spine.IntakeSubmission) (spine.IntakeRecord, error)
	Get(context.Context, spine.IntakeID) (spine.IntakeRecord, error)
}

type IntakeHandler struct {
	service IntakeService
}

func NewIntakeHandler(service IntakeService) *IntakeHandler {
	return &IntakeHandler{service: service}
}

type intakeAcceptedResponse struct {
	IntakeID                 string            `json:"intake_id"`
	OrganizationID           string            `json:"organization_id"`
	ProjectID                string            `json:"project_id"`
	RepoBindingID            string            `json:"repo_binding_id"`
	State                    spine.IntakeState `json:"state"`
	CanonicalContractCreated bool              `json:"canonical_contract_created"`
	Next                     string            `json:"next"`
}

func (h *IntakeHandler) Submit(w http.ResponseWriter, r *http.Request) {
	var submission spine.IntakeSubmission
	if err := decodeStrictJSON(r.Body, &submission); err != nil {
		respondInvalidJSON(w)
		return
	}

	record, err := h.service.Submit(r.Context(), submission)
	if err != nil {
		h.respondServiceError(w, err)
		return
	}

	RespondJSON(w, http.StatusAccepted, intakeAcceptedResponse{
		IntakeID:                 string(record.ID),
		OrganizationID:           string(record.OrganizationID),
		ProjectID:                string(record.ProjectID),
		RepoBindingID:            string(record.RepoBindingID),
		State:                    record.State,
		CanonicalContractCreated: record.CanonicalContractCreated,
		Next:                     intakeNextMessage,
	})
}

func (h *IntakeHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := spine.IntakeID(r.PathValue("id"))
	record, err := h.service.Get(r.Context(), id)
	if err != nil {
		h.respondServiceError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, record)
}

func (h *IntakeHandler) respondServiceError(w http.ResponseWriter, err error) {
	var validationErr *intake.ValidationError
	switch {
	case errors.As(err, &validationErr):
		RespondError(w, http.StatusBadRequest, "validation_failed", validationErr.Error())
	case errors.Is(err, intake.ErrNotFound):
		RespondError(w, http.StatusNotFound, "not_found", "intake record not found")
	case errors.Is(err, intake.ErrProjectContextUnavailable):
		RespondError(w, http.StatusServiceUnavailable, "project_context_unavailable", "project context validation is not configured")
	default:
		respondInternalError(w)
	}
}

func decodeStrictJSON(body io.Reader, target any) error {
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("multiple JSON values")
	}
	return nil
}
