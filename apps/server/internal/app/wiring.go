package app

import (
	"context"
	"fmt"
	"net/http"

	"github.com/heurema/goalrail/apps/server/internal/approvedcontract"
	"github.com/heurema/goalrail/apps/server/internal/auth"
	"github.com/heurema/goalrail/apps/server/internal/clarification"
	"github.com/heurema/goalrail/apps/server/internal/config"
	"github.com/heurema/goalrail/apps/server/internal/contract"
	"github.com/heurema/goalrail/apps/server/internal/contractdraft"
	"github.com/heurema/goalrail/apps/server/internal/contractseed"
	"github.com/heurema/goalrail/apps/server/internal/goal"
	"github.com/heurema/goalrail/apps/server/internal/health"
	"github.com/heurema/goalrail/apps/server/internal/httpserver"
	"github.com/heurema/goalrail/apps/server/internal/intake"
	"github.com/heurema/goalrail/apps/server/internal/postgres"
	"github.com/heurema/goalrail/apps/server/internal/spine"
	"github.com/heurema/goalrail/apps/server/internal/store"
	"github.com/heurema/goalrail/apps/server/internal/version"
	"github.com/heurema/goalrail/apps/server/internal/workitem"
	"github.com/heurema/goalrail/apps/server/internal/workitemplan"
)

type goalStore interface {
	goal.GoalStore
	clarification.GoalReader
}

type eventAppender interface {
	Append(context.Context, spine.Event) error
}

type contractStore interface {
	contractseed.ContractStore
	contractdraft.ContractStore
	approvedcontract.ContractStore
	workitemplan.ContractReader
}

type workItemStore interface {
	workitem.Store
	workitemplan.WorkItemStore
}

func newHTTPServer(ctx context.Context, cfg config.Config) (*http.Server, func(), error) {
	healthHandler := health.NewHandler()
	versionHandler := version.NewHandler()

	if !cfg.DatabaseConfigured() {
		router := httpserver.NewRouter(databaseUnavailableRouteHandlers(healthHandler, versionHandler))
		return httpserver.NewServer(cfg.Addr, router), func() {}, nil
	}

	pool, err := postgres.OpenPool(ctx, cfg.Database)
	if err != nil {
		return nil, nil, fmt.Errorf("open project context db: %w", err)
	}
	cleanup := pool.Close

	projectContext := store.NewProjectContextStore(pool)
	intakeStore := store.NewPostgresTransactionalIntakeStore(pool)
	goals := store.NewPostgresTransactionalGoalStore(pool)
	clarificationStore := store.NewPostgresClarificationRequestStore(pool)
	clarificationAnswerStore := store.NewPostgresClarificationAnswerStore(pool)
	contracts := store.NewPostgresContractStore(pool)
	contractSeedStore := store.NewPostgresContractSeedStore(pool)
	contractDraftStore := store.NewPostgresContractDraftStore(pool)
	approvedContractStore := store.NewPostgresApprovedContractStore(pool)
	workItemStore := store.NewPostgresTransactionalWorkItemStore(pool)
	workItemPlanStore := store.NewPostgresWorkItemPlanStore(pool)
	workItemPlanProposalStore := store.NewPostgresWorkItemPlanProposalStore(pool)
	events := store.NewPostgresEventLog(pool)
	authStore := store.NewPostgresAuthStore(pool)
	txRunner := store.NewPostgresTransactionRunner(pool)
	clarificationOptions := []clarification.Option{clarification.WithTransactionRunner(txRunner)}

	intakeService := intake.NewService(intakeStore, projectContext, events, intake.SystemClock{}, intake.UUIDGenerator{})
	intakeHandler := httpserver.NewIntakeHandler(intakeService)
	goalService := goal.NewService(intakeStore, goals, events, goal.SystemClock{}, goal.UUIDGenerator{})
	goalHandler := httpserver.NewGoalHandler(goalService)
	clarificationService := clarification.NewService(goals, clarificationStore, clarificationAnswerStore, events, clarification.SystemClock{}, clarification.UUIDGenerator{}, clarificationOptions...)
	clarificationHandler := httpserver.NewClarificationHandler(clarificationService)
	contractSeedService := contractseed.NewService(goals, contracts, contractSeedStore, events, contractseed.SystemClock{}, contractseed.UUIDGenerator{})
	contractDraftService := contractdraft.NewService(contractSeedStore, contracts, contractDraftStore, events, contractdraft.SystemClock{}, contractdraft.UUIDGenerator{})
	approvedContractService := approvedcontract.NewService(contractDraftStore, contracts, approvedContractStore, events, approvedcontract.SystemClock{}, approvedcontract.UUIDGenerator{})
	contractOptions := []contract.Option{contract.WithTransactionRunner(txRunner)}
	contractService := contract.NewService(contracts, contractSeedService, contractDraftService, approvedContractService, contractOptions...)
	contractHandler := httpserver.NewContractHandler(contractService)
	workItemService := workitem.NewService(workItemStore)
	workItemHandler := httpserver.NewWorkItemHandler(workItemService)
	workItemPlanOptions := []workitemplan.Option{workitemplan.WithTransactionRunner(txRunner)}
	workItemPlanService := workitemplan.NewService(contracts, approvedContractStore, workItemPlanStore, workItemPlanProposalStore, workItemStore, events, workitemplan.SystemClock{}, workitemplan.UUIDGenerator{}, workItemPlanOptions...)
	workItemPlanHandler := httpserver.NewWorkItemPlanHandler(workItemPlanService)
	authService := auth.NewService(authStore, cfg.AuthJWTSecret)
	authHandler := httpserver.NewAuthHandler(authService)

	router := httpserver.NewRouter(httpserver.RouteHandlers{
		Livez:                     http.HandlerFunc(healthHandler.Livez),
		Readyz:                    http.HandlerFunc(healthHandler.Readyz),
		Version:                   versionHandler,
		AuthLogin:                 http.HandlerFunc(authHandler.Login),
		CLILoginPage:              http.HandlerFunc(authHandler.CLILoginPage),
		CLILoginSubmit:            http.HandlerFunc(authHandler.CLILoginSubmit),
		AuthCLIExchange:           http.HandlerFunc(authHandler.CLIExchange),
		AuthRefresh:               http.HandlerFunc(authHandler.Refresh),
		AuthChangePassword:        http.HandlerFunc(authHandler.ChangePassword),
		AuthLogout:                http.HandlerFunc(authHandler.Logout),
		Me:                        http.HandlerFunc(authHandler.Me),
		IntakeSubmit:              http.HandlerFunc(intakeHandler.Submit),
		IntakeGet:                 http.HandlerFunc(intakeHandler.Get),
		IntakePromote:             http.HandlerFunc(goalHandler.PromoteFromIntake),
		GoalReadiness:             http.HandlerFunc(goalHandler.CheckReadiness),
		GoalClarificationRequests: http.HandlerFunc(clarificationHandler.CreateRequest),
		ContractCreate:            http.HandlerFunc(contractHandler.Create),
		ContractGet:               http.HandlerFunc(contractHandler.Get),
		ContractUpdate:            http.HandlerFunc(contractHandler.UpdateDraft),
		ContractSubmit:            http.HandlerFunc(contractHandler.SubmitForApproval),
		ContractApprove:           http.HandlerFunc(contractHandler.Approve),
		ContractPlans:             http.HandlerFunc(workItemPlanHandler.CreatePlan),
		PlanGet:                   http.HandlerFunc(workItemPlanHandler.GetPlan),
		PlanProposals:             http.HandlerFunc(workItemPlanHandler.SubmitProposal),
		ProposalGet:               http.HandlerFunc(workItemPlanHandler.GetProposal),
		ProposalAcceptance:        http.HandlerFunc(workItemPlanHandler.AcceptProposal),
		TaskGet:                   http.HandlerFunc(workItemHandler.GetTask),
		ClarificationAnswers:      http.HandlerFunc(clarificationHandler.RecordAnswer),
		ClarificationAnswerApply:  http.HandlerFunc(clarificationHandler.ApplyAnswer),
	})

	return httpserver.NewServer(cfg.Addr, router), cleanup, nil
}

func databaseUnavailableRouteHandlers(healthHandler *health.Handler, versionHandler http.Handler) httpserver.RouteHandlers {
	unavailable := httpserver.DatabaseNotConfiguredHandler()
	return httpserver.RouteHandlers{
		Livez:                     http.HandlerFunc(healthHandler.Livez),
		Readyz:                    http.HandlerFunc(healthHandler.Readyz),
		Version:                   versionHandler,
		AuthLogin:                 unavailable,
		CLILoginPage:              unavailable,
		CLILoginSubmit:            unavailable,
		AuthCLIExchange:           unavailable,
		AuthRefresh:               unavailable,
		AuthChangePassword:        unavailable,
		AuthLogout:                unavailable,
		Me:                        unavailable,
		IntakeSubmit:              unavailable,
		IntakeGet:                 unavailable,
		IntakePromote:             unavailable,
		GoalReadiness:             unavailable,
		GoalClarificationRequests: unavailable,
		ContractCreate:            unavailable,
		ContractGet:               unavailable,
		ContractUpdate:            unavailable,
		ContractSubmit:            unavailable,
		ContractApprove:           unavailable,
		ContractPlans:             unavailable,
		PlanGet:                   unavailable,
		PlanProposals:             unavailable,
		ProposalGet:               unavailable,
		ProposalAcceptance:        unavailable,
		TaskGet:                   unavailable,
		ClarificationAnswers:      unavailable,
		ClarificationAnswerApply:  unavailable,
	}
}
