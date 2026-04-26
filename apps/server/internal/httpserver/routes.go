package httpserver

import (
	"net/http"
)

// RouteHandlers contains the concrete handlers wired by the app composition root.
type RouteHandlers struct {
	Livez                     http.Handler
	Readyz                    http.Handler
	Version                   http.Handler
	IntakeSubmit              http.Handler
	IntakeGet                 http.Handler
	IntakePromote             http.Handler
	GoalReadiness             http.Handler
	GoalClarificationRequests http.Handler
	ClarificationAnswers      http.Handler
}

// NewRouter builds the server router.
func NewRouter(handlers RouteHandlers) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /livez", mustHandler("livez", handlers.Livez))
	mux.Handle("GET /readyz", mustHandler("readyz", handlers.Readyz))
	mux.Handle("GET /version", mustHandler("version", handlers.Version))
	mux.Handle("POST /v1/intake", mustHandler("intake submit", handlers.IntakeSubmit))
	mux.Handle("GET /v1/intake/{id}", mustHandler("intake get", handlers.IntakeGet))
	mux.Handle("POST /v1/intake/{id}/promote", mustHandler("intake promote", handlers.IntakePromote))
	mux.Handle("POST /v1/goals/{id}/readiness", mustHandler("goal readiness", handlers.GoalReadiness))
	mux.Handle("POST /v1/goals/{id}/clarification-requests", mustHandler("goal clarification requests", handlers.GoalClarificationRequests))
	mux.Handle("POST /v1/clarification-requests/{id}/answers", mustHandler("clarification answers", handlers.ClarificationAnswers))
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
