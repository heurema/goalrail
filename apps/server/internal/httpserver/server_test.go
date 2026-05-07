package httpserver_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/heurema/goalrail/apps/server/internal/auth"
	"github.com/heurema/goalrail/apps/server/internal/health"
	"github.com/heurema/goalrail/apps/server/internal/httpserver"
	"github.com/heurema/goalrail/apps/server/internal/version"
)

func TestLivezReturnsOK(t *testing.T) {
	response := getJSON(t, "/livez")

	if response.code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.code, http.StatusOK)
	}

	var body struct {
		Status string `json:"status"`
	}
	decodeJSON(t, response.body, &body)
	if body.Status != "ok" {
		t.Fatalf("status body = %q, want %q", body.Status, "ok")
	}
}

func TestReadyzReturnsOK(t *testing.T) {
	response := getJSON(t, "/readyz")

	if response.code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.code, http.StatusOK)
	}

	var body struct {
		Status string `json:"status"`
	}
	decodeJSON(t, response.body, &body)
	if body.Status != "ok" {
		t.Fatalf("status body = %q, want %q", body.Status, "ok")
	}
}

func TestVersionReturnsService(t *testing.T) {
	response := getJSON(t, "/version")

	if response.code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.code, http.StatusOK)
	}

	var body struct {
		Service string `json:"service"`
		Version string `json:"version"`
	}
	decodeJSON(t, response.body, &body)
	if body.Service != "goalrail-server" {
		t.Fatalf("service = %q, want %q", body.Service, "goalrail-server")
	}
	if body.Version == "" {
		t.Fatal("version is empty")
	}
}

func TestUnknownRouteReturnsJSONNotFound(t *testing.T) {
	response := getJSON(t, "/missing")

	if response.code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", response.code, http.StatusNotFound)
	}

	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	decodeJSON(t, response.body, &body)
	if body.Error.Code != "not_found" {
		t.Fatalf("error code = %q, want %q", body.Error.Code, "not_found")
	}
	if body.Error.Message != "not found" {
		t.Fatalf("error message = %q, want %q", body.Error.Message, "not found")
	}
}

func TestPublicV1RouteInventoryUsesResourcePaths(t *testing.T) {
	router := httpserver.NewRouter(httpserver.RouteHandlers{
		Livez:                         probeRoute("livez"),
		Readyz:                        probeRoute("readyz"),
		Version:                       probeRoute("version"),
		AuthLogin:                     probeRoute("auth_login"),
		CLILoginPage:                  probeRoute("cli_login_page"),
		CLILoginSubmit:                probeRoute("cli_login_submit"),
		AuthCLIExchange:               probeRoute("auth_cli_exchange"),
		AuthRefresh:                   probeRoute("auth_refresh"),
		AuthChangePassword:            probeRoute("auth_change_password"),
		AuthLogout:                    probeRoute("auth_logout"),
		Me:                            probeRoute("me"),
		OrganizationUsersList:         probeRoute("organization_users_list"),
		OrganizationUsersCreate:       probeRoute("organization_users_create"),
		OrganizationUsersPatch:        probeRoute("organization_users_patch"),
		OrganizationUsersReset:        probeRoute("organization_users_reset"),
		OrganizationRepositoryContext: probeRoute("organization_repository_context"),
		RepositoryContextInit:         probeRoute("repository_context_init"),
		RepositoryContextSnapshot:     probeRoute("repository_context_snapshot"),
		ProjectRepoBindingInit:        probeRoute("project_repo_binding_init"),
		IntakeSubmit:                  probeRoute("intake_submit"),
		IntakeGet:                     probeRoute("intake_get"),
		IntakePromote:                 probeRoute("intake_promote"),
		GoalReadiness:                 probeRoute("goal_readiness"),
		GoalContinuation:              probeRoute("goal_continuation"),
		ClarificationContinuation:     probeRoute("clarification_continuation"),
		GoalClarificationRequests:     probeRoute("goal_clarification_requests"),
		ContractCreate:                probeRoute("contract_create"),
		ContractGet:                   probeRoute("contract_get"),
		ContractUpdate:                probeRoute("contract_update"),
		ContractSubmit:                probeRoute("contract_submit"),
		ContractApprove:               probeRoute("contract_approve"),
		ContractPlans:                 probeRoute("contract_plans"),
		PlanLeases:                    probeRoute("plan_leases"),
		PlanLeaseGet:                  probeRoute("plan_lease_get"),
		PlanLeaseRenew:                probeRoute("plan_lease_renew"),
		PlanGet:                       probeRoute("plan_get"),
		PlanProposals:                 probeRoute("plan_proposals"),
		ProposalGet:                   probeRoute("proposal_get"),
		ProposalAcceptance:            probeRoute("proposal_acceptance"),
		TaskGet:                       probeRoute("task_get"),
		ClarificationAnswers:          probeRoute("clarification_answers"),
		ClarificationAnswerApply:      probeRoute("clarification_answer_apply"),
	})

	tests := []struct {
		name      string
		method    string
		path      string
		wantRoute string
	}{
		{name: "intake_submit", method: http.MethodPost, path: "/v1/intakes", wantRoute: "intake_submit"},
		{name: "auth_login", method: http.MethodPost, path: "/v1/auth/login", wantRoute: "auth_login"},
		{name: "cli_login_page", method: http.MethodGet, path: "/cli/login", wantRoute: "cli_login_page"},
		{name: "cli_login_submit", method: http.MethodPost, path: "/cli/login", wantRoute: "cli_login_submit"},
		{name: "auth_cli_exchange", method: http.MethodPost, path: "/v1/auth/cli/exchange", wantRoute: "auth_cli_exchange"},
		{name: "auth_refresh", method: http.MethodPost, path: "/v1/auth/refresh", wantRoute: "auth_refresh"},
		{name: "auth_change_password", method: http.MethodPost, path: "/v1/auth/change-password", wantRoute: "auth_change_password"},
		{name: "auth_logout", method: http.MethodPost, path: "/v1/auth/logout", wantRoute: "auth_logout"},
		{name: "me", method: http.MethodGet, path: "/v1/me", wantRoute: "me"},
		{name: "organization_users_list", method: http.MethodGet, path: "/v1/organizations/org-1/users", wantRoute: "organization_users_list"},
		{name: "organization_users_create", method: http.MethodPost, path: "/v1/organizations/org-1/users", wantRoute: "organization_users_create"},
		{name: "organization_users_patch", method: http.MethodPatch, path: "/v1/organizations/org-1/users/user-1", wantRoute: "organization_users_patch"},
		{name: "organization_users_reset", method: http.MethodPost, path: "/v1/organizations/org-1/users/user-1/temporary-password-resets", wantRoute: "organization_users_reset"},
		{name: "organization_repository_context", method: http.MethodGet, path: "/v1/organizations/org-1/repository-context", wantRoute: "organization_repository_context"},
		{name: "repository_context_init", method: http.MethodPost, path: "/v1/init/repository-context", wantRoute: "repository_context_init"},
		{name: "repository_context_snapshot", method: http.MethodPost, path: "/v1/repo-bindings/binding-1/context-snapshots", wantRoute: "repository_context_snapshot"},
		{name: "project_repo_binding_init", method: http.MethodPost, path: "/v1/projects/project-1/repo-bindings/init", wantRoute: "project_repo_binding_init"},
		{name: "intake_get", method: http.MethodGet, path: "/v1/intakes/intake-1", wantRoute: "intake_get"},
		{name: "intake_promote", method: http.MethodPost, path: "/v1/intakes/intake-1/goals", wantRoute: "intake_promote"},
		{name: "goal_readiness", method: http.MethodPost, path: "/v1/goals/goal-1/readiness", wantRoute: "goal_readiness"},
		{name: "goal_continuation", method: http.MethodPost, path: "/v1/goals/goal-1/continuation", wantRoute: "goal_continuation"},
		{name: "clarification_request", method: http.MethodPost, path: "/v1/goals/goal-1/clarifications", wantRoute: "goal_clarification_requests"},
		{name: "clarification_continuation", method: http.MethodPost, path: "/v1/clarifications/request-1/answers/continuation", wantRoute: "clarification_continuation"},
		{name: "clarification_answer", method: http.MethodPost, path: "/v1/clarifications/request-1/answers", wantRoute: "clarification_answers"},
		{name: "clarification_answer_apply", method: http.MethodPost, path: "/v1/answers/answer-1/applications", wantRoute: "clarification_answer_apply"},
		{name: "contract_create", method: http.MethodPost, path: "/v1/contracts", wantRoute: "contract_create"},
		{name: "contract_get", method: http.MethodGet, path: "/v1/contracts/contract-1", wantRoute: "contract_get"},
		{name: "contract_update", method: http.MethodPatch, path: "/v1/contracts/contract-1", wantRoute: "contract_update"},
		{name: "contract_submit", method: http.MethodPost, path: "/v1/contracts/contract-1/submissions", wantRoute: "contract_submit"},
		{name: "contract_approve", method: http.MethodPost, path: "/v1/contracts/contract-1/approvals", wantRoute: "contract_approve"},
		{name: "contract_plans", method: http.MethodPost, path: "/v1/contracts/contract-1/plans", wantRoute: "contract_plans"},
		{name: "plan_leases", method: http.MethodPost, path: "/v1/plans/leases", wantRoute: "plan_leases"},
		{name: "plan_lease_get", method: http.MethodGet, path: "/v1/plans/leases/lease-1", wantRoute: "plan_lease_get"},
		{name: "plan_lease_renew", method: http.MethodPatch, path: "/v1/plans/leases/lease-1", wantRoute: "plan_lease_renew"},
		{name: "plan_get", method: http.MethodGet, path: "/v1/plans/plan-1", wantRoute: "plan_get"},
		{name: "plan_proposals", method: http.MethodPost, path: "/v1/plans/plan-1/proposals", wantRoute: "plan_proposals"},
		{name: "proposal_get", method: http.MethodGet, path: "/v1/proposals/proposal-1", wantRoute: "proposal_get"},
		{name: "proposal_acceptance", method: http.MethodPost, path: "/v1/proposals/proposal-1/acceptance", wantRoute: "proposal_acceptance"},
		{name: "task_get", method: http.MethodGet, path: "/v1/tasks/task-1", wantRoute: "task_get"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := doJSON(t, router, tt.method, tt.path, "")
			if response.code != http.StatusOK {
				t.Fatalf("%s %s status = %d, want %d: %s", tt.method, tt.path, response.code, http.StatusOK, response.body)
			}

			var body struct {
				Route string `json:"route"`
			}
			decodeJSON(t, response.body, &body)
			if body.Route != tt.wantRoute {
				t.Fatalf("route = %q, want %q", body.Route, tt.wantRoute)
			}
		})
	}
}

func TestPublicV1OldVerbStyleRoutesAreNotRegistered(t *testing.T) {
	router := httpserver.NewRouter(httpserver.RouteHandlers{
		Livez:                         probeRoute("livez"),
		Readyz:                        probeRoute("readyz"),
		Version:                       probeRoute("version"),
		AuthLogin:                     probeRoute("auth_login"),
		CLILoginPage:                  probeRoute("cli_login_page"),
		CLILoginSubmit:                probeRoute("cli_login_submit"),
		AuthCLIExchange:               probeRoute("auth_cli_exchange"),
		AuthRefresh:                   probeRoute("auth_refresh"),
		AuthChangePassword:            probeRoute("auth_change_password"),
		AuthLogout:                    probeRoute("auth_logout"),
		Me:                            probeRoute("me"),
		OrganizationUsersList:         probeRoute("organization_users_list"),
		OrganizationUsersCreate:       probeRoute("organization_users_create"),
		OrganizationUsersPatch:        probeRoute("organization_users_patch"),
		OrganizationUsersReset:        probeRoute("organization_users_reset"),
		OrganizationRepositoryContext: probeRoute("organization_repository_context"),
		RepositoryContextInit:         probeRoute("repository_context_init"),
		RepositoryContextSnapshot:     probeRoute("repository_context_snapshot"),
		ProjectRepoBindingInit:        probeRoute("project_repo_binding_init"),
		IntakeSubmit:                  probeRoute("intake_submit"),
		IntakeGet:                     probeRoute("intake_get"),
		IntakePromote:                 probeRoute("intake_promote"),
		GoalReadiness:                 probeRoute("goal_readiness"),
		GoalContinuation:              probeRoute("goal_continuation"),
		ClarificationContinuation:     probeRoute("clarification_continuation"),
		GoalClarificationRequests:     probeRoute("goal_clarification_requests"),
		ContractCreate:                probeRoute("contract_create"),
		ContractGet:                   probeRoute("contract_get"),
		ContractUpdate:                probeRoute("contract_update"),
		ContractSubmit:                probeRoute("contract_submit"),
		ContractApprove:               probeRoute("contract_approve"),
		ContractPlans:                 probeRoute("contract_plans"),
		PlanLeases:                    probeRoute("plan_leases"),
		PlanLeaseGet:                  probeRoute("plan_lease_get"),
		PlanLeaseRenew:                probeRoute("plan_lease_renew"),
		PlanGet:                       probeRoute("plan_get"),
		PlanProposals:                 probeRoute("plan_proposals"),
		ProposalGet:                   probeRoute("proposal_get"),
		ProposalAcceptance:            probeRoute("proposal_acceptance"),
		TaskGet:                       probeRoute("task_get"),
		ClarificationAnswers:          probeRoute("clarification_answers"),
		ClarificationAnswerApply:      probeRoute("clarification_answer_apply"),
	})

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{name: "intake_submit", method: http.MethodPost, path: "/v1/intake"},
		{name: "intake_get", method: http.MethodGet, path: "/v1/intake/intake-1"},
		{name: "intake_promote", method: http.MethodPost, path: "/v1/intake/intake-1/promote"},
		{name: "clarification_answer_apply", method: http.MethodPost, path: "/v1/clarification-answers/answer-1/apply"},
		{name: "contract_seed", method: http.MethodPost, path: "/v1/goals/goal-1/contract-seed"},
		{name: "contract_draft", method: http.MethodPost, path: "/v1/contract-seeds/seed-1/contract-draft"},
		{name: "contract_draft_updates", method: http.MethodPost, path: "/v1/contract-drafts/draft-1/updates"},
		{name: "contract_draft_ready", method: http.MethodPost, path: "/v1/contract-drafts/draft-1/ready-for-approval"},
		{name: "contract_draft_approve", method: http.MethodPost, path: "/v1/contract-drafts/draft-1/approve"},
		{name: "intermediate_intake_promote", method: http.MethodPost, path: "/v1/intakes/intake-1/promotions"},
		{name: "intermediate_goal_readiness", method: http.MethodPost, path: "/v1/goals/goal-1/readiness-checks"},
		{name: "intermediate_clarification_request", method: http.MethodPost, path: "/v1/goals/goal-1/clarification-requests"},
		{name: "intermediate_clarification_answer", method: http.MethodPost, path: "/v1/clarification-requests/request-1/answers"},
		{name: "intermediate_clarification_answer_apply", method: http.MethodPost, path: "/v1/clarification-answers/answer-1/applications"},
		{name: "intermediate_contract_draft_ready", method: http.MethodPost, path: "/v1/contract-drafts/draft-1/approval-submissions"},
		{name: "intermediate_work_item_plan", method: http.MethodPost, path: "/v1/approved-contracts/contract-1/work-items"},
		{name: "transitional_contract_seed", method: http.MethodPost, path: "/v1/goals/goal-1/contract-seeds"},
		{name: "transitional_contract_draft", method: http.MethodPost, path: "/v1/contract-seeds/seed-1/contract-drafts"},
		{name: "transitional_contract_draft_update", method: http.MethodPatch, path: "/v1/contract-drafts/draft-1"},
		{name: "transitional_contract_draft_submit", method: http.MethodPost, path: "/v1/contract-drafts/draft-1/submissions"},
		{name: "transitional_contract_draft_approve", method: http.MethodPost, path: "/v1/contract-drafts/draft-1/approvals"},
		{name: "removed_direct_task_creation", method: http.MethodPost, path: "/v1/contracts/contract-1/tasks"},
		{name: "plan_list", method: http.MethodGet, path: "/v1/plans"},
		{name: "proposal_list", method: http.MethodGet, path: "/v1/proposals"},
		{name: "task_list", method: http.MethodGet, path: "/v1/tasks"},
		{name: "public_registration", method: http.MethodPost, path: "/v1/auth/register"},
		{name: "admin_user_creation", method: http.MethodPost, path: "/v1/users"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := doJSON(t, router, tt.method, tt.path, "")
			if response.code != http.StatusNotFound {
				t.Fatalf("%s %s status = %d, want %d: %s", tt.method, tt.path, response.code, http.StatusNotFound, response.body)
			}
		})
	}
}

func probeRoute(route string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpserver.RespondJSON(w, http.StatusOK, map[string]string{"route": route})
	})
}

type stubAuthService struct{}

func (stubAuthService) Login(context.Context, auth.LoginInput) (auth.LoginResult, error) {
	return auth.LoginResult{}, auth.ErrInvalidCredentials
}

func (stubAuthService) StartCLILogin(context.Context, auth.CLILoginInput) (auth.CLILoginResult, error) {
	return auth.CLILoginResult{}, auth.ErrInvalidCredentials
}

func (stubAuthService) ExchangeCLIAuthCode(context.Context, auth.CLIExchangeInput) (auth.CLIExchangeResult, error) {
	return auth.CLIExchangeResult{}, auth.ErrCLIAuthCodeInvalid
}

func (stubAuthService) Refresh(context.Context, auth.RefreshInput) (auth.RefreshResult, error) {
	return auth.RefreshResult{}, auth.ErrSessionInvalid
}

func (stubAuthService) ChangePassword(context.Context, string, auth.ChangePasswordInput) (auth.ChangePasswordResult, error) {
	return auth.ChangePasswordResult{}, auth.ErrInvalidToken
}

func (stubAuthService) Logout(context.Context, string) (auth.LogoutResult, error) {
	return auth.LogoutResult{}, auth.ErrInvalidToken
}

func (stubAuthService) Me(context.Context, string) (auth.Profile, error) {
	return auth.Profile{}, auth.ErrInvalidToken
}

type routeResponse struct {
	code        int
	contentType string
	header      http.Header
	body        string
}

func getJSON(t *testing.T, path string) routeResponse {
	t.Helper()

	return doJSON(t, testServer(t).router, http.MethodGet, path, "")
}

func doJSON(t *testing.T, handler http.Handler, method string, path string, body string) routeResponse {
	t.Helper()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}

	handler.ServeHTTP(recorder, request)

	contentType := recorder.Header().Get("Content-Type")
	if recorder.Code != http.StatusNoContent && !strings.HasPrefix(contentType, "application/json") {
		t.Fatalf("Content-Type = %q, want application/json", contentType)
	}

	return routeResponse{
		code:        recorder.Code,
		contentType: contentType,
		header:      recorder.Header(),
		body:        recorder.Body.String(),
	}
}

func newRouter(
	livez http.Handler,
	readyz http.Handler,
	versionHandler http.Handler,
	authHandler *httpserver.AuthHandler,
	intakeHandler *httpserver.IntakeHandler,
	goalHandler *httpserver.GoalHandler,
	clarificationHandler *httpserver.ClarificationHandler,
	continuationHandler *httpserver.ContinuationHandler,
	contractHandler *httpserver.ContractHandler,
	workItemHandler *httpserver.WorkItemHandler,
	workItemPlanHandler *httpserver.WorkItemPlanHandler,
) http.Handler {
	return httpserver.NewRouter(httpserver.RouteHandlers{
		Livez:                         livez,
		Readyz:                        readyz,
		Version:                       versionHandler,
		AuthLogin:                     http.HandlerFunc(authHandler.Login),
		CLILoginPage:                  http.HandlerFunc(authHandler.CLILoginPage),
		CLILoginSubmit:                http.HandlerFunc(authHandler.CLILoginSubmit),
		AuthCLIExchange:               http.HandlerFunc(authHandler.CLIExchange),
		AuthRefresh:                   http.HandlerFunc(authHandler.Refresh),
		AuthChangePassword:            http.HandlerFunc(authHandler.ChangePassword),
		AuthLogout:                    http.HandlerFunc(authHandler.Logout),
		Me:                            http.HandlerFunc(authHandler.Me),
		OrganizationUsersList:         probeRoute("organization_users_list"),
		OrganizationUsersCreate:       probeRoute("organization_users_create"),
		OrganizationUsersPatch:        probeRoute("organization_users_patch"),
		OrganizationUsersReset:        probeRoute("organization_users_reset"),
		OrganizationRepositoryContext: probeRoute("organization_repository_context"),
		RepositoryContextInit:         probeRoute("repository_context_init"),
		RepositoryContextSnapshot:     probeRoute("repository_context_snapshot"),
		ProjectRepoBindingInit:        probeRoute("project_repo_binding_init"),
		IntakeSubmit:                  http.HandlerFunc(intakeHandler.Submit),
		IntakeGet:                     http.HandlerFunc(intakeHandler.Get),
		IntakePromote:                 http.HandlerFunc(goalHandler.PromoteFromIntake),
		GoalReadiness:                 http.HandlerFunc(goalHandler.CheckReadiness),
		GoalContinuation:              http.HandlerFunc(continuationHandler.ReconcileGoal),
		ClarificationContinuation:     http.HandlerFunc(continuationHandler.AnswerClarification),
		GoalClarificationRequests:     http.HandlerFunc(clarificationHandler.CreateRequest),
		ContractCreate:                http.HandlerFunc(contractHandler.Create),
		ContractGet:                   http.HandlerFunc(contractHandler.Get),
		ContractUpdate:                http.HandlerFunc(contractHandler.UpdateDraft),
		ContractSubmit:                http.HandlerFunc(contractHandler.SubmitForApproval),
		ContractApprove:               http.HandlerFunc(contractHandler.Approve),
		ContractPlans:                 http.HandlerFunc(workItemPlanHandler.CreatePlan),
		PlanGet:                       http.HandlerFunc(workItemPlanHandler.GetPlan),
		PlanLeases:                    http.HandlerFunc(workItemPlanHandler.AcquireLease),
		PlanLeaseGet:                  http.HandlerFunc(workItemPlanHandler.GetLease),
		PlanLeaseRenew:                http.HandlerFunc(workItemPlanHandler.RenewLease),
		PlanProposals:                 http.HandlerFunc(workItemPlanHandler.SubmitProposal),
		ProposalGet:                   http.HandlerFunc(workItemPlanHandler.GetProposal),
		ProposalAcceptance:            http.HandlerFunc(workItemPlanHandler.AcceptProposal),
		TaskGet:                       http.HandlerFunc(workItemHandler.GetTask),
		ClarificationAnswers:          http.HandlerFunc(clarificationHandler.RecordAnswer),
		ClarificationAnswerApply:      http.HandlerFunc(clarificationHandler.ApplyAnswer),
	})
}

func baseHandlers(
	intakeHandler *httpserver.IntakeHandler,
	goalHandler *httpserver.GoalHandler,
	clarificationHandler *httpserver.ClarificationHandler,
	continuationHandler *httpserver.ContinuationHandler,
	contractHandler *httpserver.ContractHandler,
	workItemHandler *httpserver.WorkItemHandler,
	workItemPlanHandler *httpserver.WorkItemPlanHandler,
) http.Handler {
	healthHandler := health.NewHandler()
	authHandler := httpserver.NewAuthHandler(stubAuthService{})
	return newRouter(
		http.HandlerFunc(healthHandler.Livez),
		http.HandlerFunc(healthHandler.Readyz),
		version.NewHandler(),
		authHandler,
		intakeHandler,
		goalHandler,
		clarificationHandler,
		continuationHandler,
		contractHandler,
		workItemHandler,
		workItemPlanHandler,
	)
}

func decodeJSON(t *testing.T, input string, target any) {
	t.Helper()

	if err := json.Unmarshal([]byte(input), target); err != nil {
		t.Fatalf("decode JSON %q: %v", input, err)
	}
}
