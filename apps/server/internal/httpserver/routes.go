package httpserver

import (
	"net/http"
)

// RouteHandlers contains the concrete handlers wired by the app composition root.
type RouteHandlers struct {
	Livez                     http.Handler
	Readyz                    http.Handler
	Version                   http.Handler
	AuthLogin                 http.Handler
	CLILoginPage              http.Handler
	CLILoginSubmit            http.Handler
	AuthCLIExchange           http.Handler
	AuthRefresh               http.Handler
	AuthChangePassword        http.Handler
	AuthLogout                http.Handler
	Me                        http.Handler
	OrganizationUsersList     http.Handler
	OrganizationUsersCreate   http.Handler
	OrganizationUsersPatch    http.Handler
	RepositoryContextInit     http.Handler
	RepositoryContextSnapshot http.Handler
	ProjectRepoBindingInit    http.Handler
	IntakeSubmit              http.Handler
	IntakeGet                 http.Handler
	IntakePromote             http.Handler
	GoalReadiness             http.Handler
	GoalContinuation          http.Handler
	ClarificationContinuation http.Handler
	GoalClarificationRequests http.Handler
	ContractCreate            http.Handler
	ContractGet               http.Handler
	ContractUpdate            http.Handler
	ContractSubmit            http.Handler
	ContractApprove           http.Handler
	ContractPlans             http.Handler
	PlanGet                   http.Handler
	PlanLeases                http.Handler
	PlanLeaseGet              http.Handler
	PlanLeaseRenew            http.Handler
	PlanProposals             http.Handler
	ProposalGet               http.Handler
	ProposalAcceptance        http.Handler
	TaskGet                   http.Handler
	ClarificationAnswers      http.Handler
	ClarificationAnswerApply  http.Handler
}

// NewRouter builds the server router.
func NewRouter(handlers RouteHandlers) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /livez", mustHandler("livez", handlers.Livez))
	mux.Handle("GET /readyz", mustHandler("readyz", handlers.Readyz))
	mux.Handle("GET /version", mustHandler("version", handlers.Version))
	mux.Handle("POST /v1/auth/login", mustHandler("auth login", handlers.AuthLogin))
	mux.Handle("GET /cli/login", mustHandler("CLI login page", handlers.CLILoginPage))
	mux.Handle("POST /cli/login", mustHandler("CLI login submit", handlers.CLILoginSubmit))
	mux.Handle("POST /v1/auth/cli/exchange", mustHandler("auth CLI exchange", handlers.AuthCLIExchange))
	mux.Handle("POST /v1/auth/refresh", mustHandler("auth refresh", handlers.AuthRefresh))
	mux.Handle("POST /v1/auth/change-password", mustHandler("auth change password", handlers.AuthChangePassword))
	mux.Handle("POST /v1/auth/logout", mustHandler("auth logout", handlers.AuthLogout))
	mux.Handle("GET /v1/me", mustHandler("me", handlers.Me))
	mux.Handle("GET /v1/organizations/{organization_id}/users", mustHandler("organization users list", handlers.OrganizationUsersList))
	mux.Handle("POST /v1/organizations/{organization_id}/users", mustHandler("organization users create", handlers.OrganizationUsersCreate))
	mux.Handle("PATCH /v1/organizations/{organization_id}/users/{user_id}", mustHandler("organization users patch", handlers.OrganizationUsersPatch))
	mux.Handle("POST /v1/init/repository-context", mustHandler("repository context init", handlers.RepositoryContextInit))
	mux.Handle("POST /v1/repo-bindings/{repo_binding_id}/context-snapshots", mustHandler("repository context snapshot", handlers.RepositoryContextSnapshot))
	mux.Handle("POST /v1/projects/{project_id}/repo-bindings/init", mustHandler("project repo binding init", handlers.ProjectRepoBindingInit))
	mux.Handle("POST /v1/intakes", mustHandler("intake submit", handlers.IntakeSubmit))
	mux.Handle("GET /v1/intakes/{id}", mustHandler("intake get", handlers.IntakeGet))
	mux.Handle("POST /v1/intakes/{id}/goals", mustHandler("intake promote", handlers.IntakePromote))
	mux.Handle("POST /v1/goals/{id}/readiness", mustHandler("goal readiness", handlers.GoalReadiness))
	mux.Handle("POST /v1/goals/{id}/continuation", mustHandler("goal continuation", handlers.GoalContinuation))
	mux.Handle("POST /v1/clarifications/{id}/answers/continuation", mustHandler("clarification continuation", handlers.ClarificationContinuation))
	mux.Handle("POST /v1/goals/{id}/clarifications", mustHandler("goal clarification requests", handlers.GoalClarificationRequests))
	mux.Handle("POST /v1/contracts", mustHandler("contract create", handlers.ContractCreate))
	mux.Handle("GET /v1/contracts/{id}", mustHandler("contract get", handlers.ContractGet))
	mux.Handle("PATCH /v1/contracts/{id}", mustHandler("contract update", handlers.ContractUpdate))
	mux.Handle("POST /v1/contracts/{id}/submissions", mustHandler("contract submit", handlers.ContractSubmit))
	mux.Handle("POST /v1/contracts/{id}/approvals", mustHandler("contract approve", handlers.ContractApprove))
	mux.Handle("POST /v1/contracts/{id}/plans", mustHandler("contract plans", handlers.ContractPlans))
	mux.Handle("POST /v1/plans/leases", mustHandler("plan leases", handlers.PlanLeases))
	mux.Handle("GET /v1/plans/leases/{id}", mustHandler("plan lease get", handlers.PlanLeaseGet))
	mux.Handle("PATCH /v1/plans/leases/{id}", mustHandler("plan lease renew", handlers.PlanLeaseRenew))
	mux.Handle("GET /v1/plans/{id}", mustHandler("plan get", handlers.PlanGet))
	mux.Handle("POST /v1/plans/{id}/proposals", mustHandler("plan proposals", handlers.PlanProposals))
	mux.Handle("GET /v1/proposals/{id}", mustHandler("proposal get", handlers.ProposalGet))
	mux.Handle("POST /v1/proposals/{id}/acceptance", mustHandler("proposal acceptance", handlers.ProposalAcceptance))
	mux.Handle("GET /v1/tasks/{id}", mustHandler("task get", handlers.TaskGet))
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
