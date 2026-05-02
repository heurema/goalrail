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
	ContractCreate            http.Handler
	ContractGet               http.Handler
	ContractUpdate            http.Handler
	ContractSubmit            http.Handler
	ContractApprove           http.Handler
	ContractTasks             http.Handler
	ClarificationAnswers      http.Handler
	ClarificationAnswerApply  http.Handler
}

// NewRouter builds the server router.
func NewRouter(handlers RouteHandlers) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /livez", mustHandler("livez", handlers.Livez))
	mux.Handle("GET /readyz", mustHandler("readyz", handlers.Readyz))
	mux.Handle("GET /version", mustHandler("version", handlers.Version))
	mux.Handle("POST /v1/intakes", mustHandler("intake submit", handlers.IntakeSubmit))
	mux.Handle("GET /v1/intakes/{id}", mustHandler("intake get", handlers.IntakeGet))
	mux.Handle("POST /v1/intakes/{id}/goals", mustHandler("intake promote", handlers.IntakePromote))
	mux.Handle("POST /v1/goals/{id}/readiness", mustHandler("goal readiness", handlers.GoalReadiness))
	mux.Handle("POST /v1/goals/{id}/clarifications", mustHandler("goal clarification requests", handlers.GoalClarificationRequests))
	mux.Handle("POST /v1/contracts", mustHandler("contract create", handlers.ContractCreate))
	mux.Handle("GET /v1/contracts/{id}", mustHandler("contract get", handlers.ContractGet))
	mux.Handle("PATCH /v1/contracts/{id}", mustHandler("contract update", handlers.ContractUpdate))
	mux.Handle("POST /v1/contracts/{id}/submissions", mustHandler("contract submit", handlers.ContractSubmit))
	mux.Handle("POST /v1/contracts/{id}/approvals", mustHandler("contract approve", handlers.ContractApprove))
	mux.Handle("POST /v1/contracts/{id}/tasks", mustHandler("contract tasks", handlers.ContractTasks))
	mux.Handle("POST /v1/clarifications/{id}/answers", mustHandler("clarification answers", handlers.ClarificationAnswers))
	mux.Handle("POST /v1/answers/{id}/applications", mustHandler("clarification answer apply", handlers.ClarificationAnswerApply))
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
