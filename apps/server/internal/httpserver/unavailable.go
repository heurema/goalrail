package httpserver

import "net/http"

// DatabaseNotConfiguredHandler returns a stable service-unavailable response for product routes.
func DatabaseNotConfiguredHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		RespondError(w, http.StatusServiceUnavailable, "database_not_configured", "database is not configured")
	})
}
