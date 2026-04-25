package health

import (
	"net/http"

	"github.com/heurema/goalrail/apps/server/internal/httpserver"
)

type statusResponse struct {
	Status string `json:"status"`
}

// Handler serves Kubernetes-friendly liveness and readiness checks.
type Handler struct{}

// NewHandler creates a health handler with no external dependencies.
func NewHandler() *Handler {
	return &Handler{}
}

// Livez reports that the HTTP process is alive.
func (h *Handler) Livez(w http.ResponseWriter, r *http.Request) {
	httpserver.RespondJSON(w, http.StatusOK, statusResponse{Status: "ok"})
}

// Readyz reports readiness. It is always ok until real dependencies exist.
func (h *Handler) Readyz(w http.ResponseWriter, r *http.Request) {
	httpserver.RespondJSON(w, http.StatusOK, statusResponse{Status: "ok"})
}
