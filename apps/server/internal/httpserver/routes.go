package httpserver

import (
	"net/http"
)

// RouteHandlers contains the concrete handlers wired by the app composition root.
type RouteHandlers struct {
	Livez   http.Handler
	Readyz  http.Handler
	Version http.Handler
}

// NewRouter builds the server router with only health and version endpoints.
func NewRouter(handlers RouteHandlers) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /livez", mustHandler("livez", handlers.Livez))
	mux.Handle("GET /readyz", mustHandler("readyz", handlers.Readyz))
	mux.Handle("GET /version", mustHandler("version", handlers.Version))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		RespondError(w, http.StatusNotFound, "not_found", "not found")
	})

	return mux
}

func mustHandler(name string, handler http.Handler) http.Handler {
	if handler == nil {
		panic("httpserver: nil " + name + " handler")
	}
	return handler
}
