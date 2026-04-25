package version

import (
	"net/http"

	"github.com/heurema/goalrail/apps/server/internal/httpserver"
)

const (
	// Service is the canonical server service name.
	Service = "goalrail-server"
	// Version is the current development version for the first server skeleton.
	Version = "0.0.0-dev"
)

// Info is the JSON payload returned by the version endpoint.
type Info struct {
	Service string `json:"service"`
	Version string `json:"version"`
}

// Current returns the current server version payload.
func Current() Info {
	return Info{
		Service: Service,
		Version: Version,
	}
}

// Handler serves the version endpoint.
type Handler struct {
	info Info
}

// NewHandler creates a version handler for the current server build.
func NewHandler() *Handler {
	return &Handler{info: Current()}
}

// ServeHTTP writes the current server version payload.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	httpserver.RespondJSON(w, http.StatusOK, h.info)
}
