package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/heurema/goalrail/apps/server/internal/approvedcontract"
	"github.com/heurema/goalrail/apps/server/internal/auth"
	"github.com/heurema/goalrail/apps/server/internal/checkout"
	"github.com/heurema/goalrail/apps/server/internal/clarification"
	"github.com/heurema/goalrail/apps/server/internal/config"
	"github.com/heurema/goalrail/apps/server/internal/continuation"
	"github.com/heurema/goalrail/apps/server/internal/contract"
	"github.com/heurema/goalrail/apps/server/internal/contractdraft"
	"github.com/heurema/goalrail/apps/server/internal/contractseed"
	"github.com/heurema/goalrail/apps/server/internal/execution"
	"github.com/heurema/goalrail/apps/server/internal/goal"
	"github.com/heurema/goalrail/apps/server/internal/health"
	"github.com/heurema/goalrail/apps/server/internal/httpserver"
	"github.com/heurema/goalrail/apps/server/internal/intake"
	"github.com/heurema/goalrail/apps/server/internal/postgres"
	"github.com/heurema/goalrail/apps/server/internal/qualificationfeed"
	"github.com/heurema/goalrail/apps/server/internal/repobinding"
	"github.com/heurema/goalrail/apps/server/internal/repositorycontext"
	"github.com/heurema/goalrail/apps/server/internal/repositoryinit"
	"github.com/heurema/goalrail/apps/server/internal/spine"
	"github.com/heurema/goalrail/apps/server/internal/store"
	"github.com/heurema/goalrail/apps/server/internal/usermanagement"
	"github.com/heurema/goalrail/apps/server/internal/version"
	"github.com/heurema/goalrail/apps/server/internal/workitem"
	"github.com/heurema/goalrail/apps/server/internal/workitemplan"
)

type postgresStores struct {
	projectContext        *store.ProjectContextStore
	intakes               *store.PostgresIntakeStore
	goals                 *store.PostgresGoalStore
	clarificationRequests *store.PostgresClarificationRequestStore
	clarificationAnswers  *store.PostgresClarificationAnswerStore
	qualificationFeed     *store.PostgresQualificationFeedStore
	contracts             *store.PostgresContractStore
	contractSeeds         *store.PostgresContractSeedStore
	contractDrafts        *store.PostgresContractDraftStore
	approvedContracts     *store.PostgresApprovedContractStore
	workItems             *store.PostgresWorkItemStore
	workItemPlans         *store.PostgresWorkItemPlanStore
	workItemPlanLeases    *store.PostgresWorkItemPlanLeaseStore
	workItemProposals     *store.PostgresWorkItemPlanProposalStore
	checkoutJobs          *store.PostgresCheckoutJobStore
	checkoutReceipts      *store.PostgresCheckoutReceiptStore
	executionJobs         *store.PostgresExecutionJobStore
	runs                  *store.PostgresRunStore
	executionCommandPlans *store.PostgresExecutionCommandPlanStore
	executionReceipts     *store.PostgresExecutionReceiptStore
	runnerCapabilities    *store.PostgresRunnerCapabilityReportStore
	events                *store.PostgresEventLog
	auth                  *store.PostgresAuthStore
	userManagement        *store.PostgresUserManagementStore
}

func newPostgresStores(pool *pgxpool.Pool) postgresStores {
	return postgresStores{
		projectContext:        store.NewProjectContextStore(pool),
		intakes:               store.NewPostgresIntakeStore(pool),
		goals:                 store.NewPostgresGoalStore(pool),
		clarificationRequests: store.NewPostgresClarificationRequestStore(pool),
		clarificationAnswers:  store.NewPostgresClarificationAnswerStore(pool),
		qualificationFeed:     store.NewPostgresQualificationFeedStore(pool),
		contracts:             store.NewPostgresContractStore(pool),
		contractSeeds:         store.NewPostgresContractSeedStore(pool),
		contractDrafts:        store.NewPostgresContractDraftStore(pool),
		approvedContracts:     store.NewPostgresApprovedContractStore(pool),
		workItems:             store.NewPostgresWorkItemStore(pool),
		workItemPlans:         store.NewPostgresWorkItemPlanStore(pool),
		workItemPlanLeases:    store.NewPostgresWorkItemPlanLeaseStore(pool),
		workItemProposals:     store.NewPostgresWorkItemPlanProposalStore(pool),
		checkoutJobs:          store.NewPostgresCheckoutJobStore(pool),
		checkoutReceipts:      store.NewPostgresCheckoutReceiptStore(pool),
		executionJobs:         store.NewPostgresExecutionJobStore(pool),
		runs:                  store.NewPostgresRunStore(pool),
		executionCommandPlans: store.NewPostgresExecutionCommandPlanStore(pool),
		executionReceipts:     store.NewPostgresExecutionReceiptStore(pool),
		runnerCapabilities:    store.NewPostgresRunnerCapabilityReportStore(pool),
		events:                store.NewPostgresEventLog(pool),
		auth:                  store.NewPostgresAuthStore(pool),
		userManagement:        store.NewPostgresUserManagementStore(pool),
	}
}

type appServices struct {
	intake            *intake.Service
	goal              *goal.Service
	clarification     *clarification.Service
	continuation      *continuation.Service
	qualificationFeed *qualificationfeed.Service
	contract          *contract.Service
	workItem          *workitem.Service
	workItemPlan      *workitemplan.Service
	checkout          *checkout.Service
	execution         *execution.Service
	repoBinding       *repobinding.Service
	repositoryInit    *repositoryinit.Service
	repositoryContext *repositorycontext.Service
	auth              *auth.Service
	userManagement    *usermanagement.Service
}

func newAppServices(stores postgresStores, txRunner *store.PostgresTransactionRunner, authJWTSecret string) appServices {
	contractSeedService := contractseed.NewService(stores.goals, stores.contracts, stores.contractSeeds, stores.events, txRunner, contractseed.SystemClock{}, contractseed.UUIDGenerator{})
	contractDraftService := contractdraft.NewService(stores.contractSeeds, stores.contracts, stores.contractDrafts, stores.events, txRunner, contractdraft.SystemClock{}, contractdraft.UUIDGenerator{})
	approvedContractService := approvedcontract.NewService(stores.contractDrafts, stores.contracts, stores.approvedContracts, stores.events, txRunner, approvedcontract.SystemClock{}, approvedcontract.UUIDGenerator{})

	repoBindingService := repobinding.NewService(stores.projectContext, stores.events, txRunner, repobinding.SystemClock{}, repobinding.UUIDGenerator{})

	goalService := goal.NewService(stores.intakes, stores.goals, stores.events, txRunner, goal.SystemClock{}, goal.UUIDGenerator{})
	clarificationRequests := clarificationRequestStoreAdapter{store: stores.clarificationRequests}
	clarificationService := clarification.NewService(stores.goals, clarificationRequests, stores.clarificationAnswers, stores.events, txRunner, clarification.SystemClock{}, clarification.UUIDGenerator{})
	executionService := execution.NewService(stores.workItems, stores.projectContext, stores.checkoutReceipts, stores.checkoutJobs, stores.executionJobs, stores.runs, stores.executionCommandPlans, stores.executionReceipts, stores.events, txRunner, execution.SystemClock{}, execution.UUIDGenerator{})
	executionService.RunnerCapabilities = stores.runnerCapabilities

	return appServices{
		intake:            intake.NewService(stores.intakes, stores.projectContext, stores.events, txRunner, intake.SystemClock{}, intake.UUIDGenerator{}),
		goal:              goalService,
		clarification:     clarificationService,
		continuation:      continuation.NewService(stores.goals, goalService, clarificationService),
		qualificationFeed: qualificationfeed.NewService(stores.qualificationFeed),
		contract:          contract.NewService(stores.goals, stores.contracts, contractSeedService, contractDraftService, stores.contractDrafts, approvedContractService, txRunner),
		workItem:          workitem.NewService(stores.workItems, stores.contracts),
		workItemPlan:      workitemplan.NewService(stores.contracts, stores.approvedContracts, stores.workItemPlans, stores.workItemPlanLeases, stores.workItemProposals, stores.workItems, stores.events, txRunner, workitemplan.SystemClock{}, workitemplan.UUIDGenerator{}),
		checkout:          checkout.NewService(stores.workItems, stores.projectContext, stores.checkoutJobs, stores.checkoutReceipts, stores.events, txRunner, checkout.SystemClock{}, checkout.UUIDGenerator{}),
		execution:         executionService,
		repoBinding:       repoBindingService,
		repositoryInit:    repositoryinit.NewService(stores.projectContext, repoBindingService, stores.events, txRunner, repositoryinit.SystemClock{}, repositoryinit.UUIDGenerator{}),
		repositoryContext: repositorycontext.NewService(stores.projectContext, stores.events, txRunner, repositorycontext.SystemClock{}, repositorycontext.UUIDGenerator{}),
		auth:              auth.NewService(stores.auth, authJWTSecret),
		userManagement:    usermanagement.NewService(stores.userManagement, txRunner),
	}
}

type appHandlers struct {
	intake            *httpserver.IntakeHandler
	goal              *httpserver.GoalHandler
	clarification     *httpserver.ClarificationHandler
	continuation      *httpserver.ContinuationHandler
	qualificationFeed *httpserver.QualificationFeedHandler
	contract          *httpserver.ContractHandler
	workItem          *httpserver.WorkItemHandler
	workItemPlan      *httpserver.WorkItemPlanHandler
	checkout          *httpserver.CheckoutHandler
	execution         *httpserver.ExecutionHandler
	repoBinding       *httpserver.RepoBindingHandler
	repositoryInit    *httpserver.RepositoryInitHandler
	repositoryContext *httpserver.RepositoryContextSnapshotHandler
	auth              *httpserver.AuthHandler
	userManagement    *httpserver.OrganizationUsersHandler
}

func newAppHandlers(services appServices) appHandlers {
	return appHandlers{
		intake:            httpserver.NewIntakeHandler(services.intake),
		goal:              httpserver.NewGoalHandler(services.goal),
		clarification:     httpserver.NewClarificationHandler(services.clarification),
		continuation:      httpserver.NewContinuationHandler(services.auth, services.continuation),
		qualificationFeed: httpserver.NewQualificationFeedHandler(services.auth, services.qualificationFeed),
		contract:          httpserver.NewContractHandler(services.auth, services.contract),
		workItem:          httpserver.NewWorkItemHandler(services.auth, services.workItem),
		workItemPlan:      httpserver.NewWorkItemPlanHandler(services.auth, services.workItemPlan),
		checkout:          httpserver.NewCheckoutHandler(services.auth, services.checkout),
		execution:         httpserver.NewExecutionHandler(services.auth, services.execution),
		repoBinding:       httpserver.NewRepoBindingHandler(services.auth, services.repoBinding),
		repositoryInit:    httpserver.NewRepositoryInitHandler(services.auth, services.repositoryInit),
		repositoryContext: httpserver.NewRepositoryContextSnapshotHandler(services.auth, services.repositoryContext),
		auth:              httpserver.NewAuthHandler(services.auth),
		userManagement:    httpserver.NewOrganizationUsersHandler(services.auth, services.userManagement),
	}
}

type clarificationRequestStoreAdapter struct {
	store *store.PostgresClarificationRequestStore
}

func (s clarificationRequestStoreAdapter) Create(ctx context.Context, request spine.ClarificationRequest) error {
	err := s.store.Create(ctx, request)
	if errors.Is(err, store.ErrClarificationRequestAlreadyOpen) {
		return clarification.ErrAlreadyOpen
	}
	return err
}

func (s clarificationRequestStoreAdapter) Get(ctx context.Context, id spine.ClarificationRequestID) (spine.ClarificationRequest, bool, error) {
	return s.store.Get(ctx, id)
}

func (s clarificationRequestStoreAdapter) GetOpenByGoalID(ctx context.Context, goalID spine.GoalID) (spine.ClarificationRequest, bool, error) {
	return s.store.GetOpenByGoalID(ctx, goalID)
}

func (s clarificationRequestStoreAdapter) UpdateState(ctx context.Context, id spine.ClarificationRequestID, state spine.ClarificationRequestState) (spine.ClarificationRequest, bool, error) {
	request, ok, err := s.store.UpdateState(ctx, id, state)
	if errors.Is(err, store.ErrClarificationRequestAlreadyOpen) {
		return request, ok, clarification.ErrAlreadyOpen
	}
	return request, ok, err
}

func (h appHandlers) routeHandlers(healthHandler *health.Handler, versionHandler http.Handler) httpserver.RouteHandlers {
	return httpserver.RouteHandlers{
		Livez:                         http.HandlerFunc(healthHandler.Livez),
		Readyz:                        http.HandlerFunc(healthHandler.Readyz),
		Version:                       versionHandler,
		AuthLogin:                     http.HandlerFunc(h.auth.Login),
		CLILoginPage:                  http.HandlerFunc(h.auth.CLILoginPage),
		CLILoginSubmit:                http.HandlerFunc(h.auth.CLILoginSubmit),
		AuthCLIExchange:               http.HandlerFunc(h.auth.CLIExchange),
		AuthRefresh:                   http.HandlerFunc(h.auth.Refresh),
		AuthChangePassword:            http.HandlerFunc(h.auth.ChangePassword),
		AuthLogout:                    http.HandlerFunc(h.auth.Logout),
		Me:                            http.HandlerFunc(h.auth.Me),
		OrganizationUsersList:         http.HandlerFunc(h.userManagement.List),
		OrganizationUsersCreate:       http.HandlerFunc(h.userManagement.Create),
		OrganizationUsersPatch:        http.HandlerFunc(h.userManagement.Patch),
		OrganizationUsersReset:        http.HandlerFunc(h.userManagement.ResetTemporaryPassword),
		OrganizationRepositoryContext: http.HandlerFunc(h.repositoryContext.GetOrganizationRepositoryContext),
		RepositoryContextInit:         http.HandlerFunc(h.repositoryInit.Init),
		RepositoryContextSnapshot:     http.HandlerFunc(h.repositoryContext.Record),
		ProjectRepoBindingInit:        http.HandlerFunc(h.repoBinding.Init),
		IntakeSubmit:                  http.HandlerFunc(h.intake.Submit),
		IntakeGet:                     http.HandlerFunc(h.intake.Get),
		IntakePromote:                 http.HandlerFunc(h.goal.PromoteFromIntake),
		GoalReadiness:                 http.HandlerFunc(h.goal.CheckReadiness),
		GoalContinuation:              http.HandlerFunc(h.continuation.ReconcileGoal),
		ClarificationContinuation:     http.HandlerFunc(h.continuation.AnswerClarification),
		QualificationFeed:             http.HandlerFunc(h.qualificationFeed.List),
		GoalClarificationRequests:     http.HandlerFunc(h.clarification.CreateRequest),
		ContractCreate:                http.HandlerFunc(h.contract.Create),
		ContractList:                  http.HandlerFunc(h.contract.List),
		ContractGet:                   http.HandlerFunc(h.contract.Get),
		ContractCurrentDraft:          http.HandlerFunc(h.contract.CurrentDraft),
		ContractUpdate:                http.HandlerFunc(h.contract.UpdateDraft),
		ContractSubmit:                http.HandlerFunc(h.contract.SubmitForApproval),
		ContractApprove:               http.HandlerFunc(h.contract.Approve),
		ContractPlans:                 http.HandlerFunc(h.workItemPlan.CreatePlan),
		PlanLeases:                    http.HandlerFunc(h.workItemPlan.AcquireLease),
		PlanLeaseGet:                  http.HandlerFunc(h.workItemPlan.GetLease),
		PlanLeaseRenew:                http.HandlerFunc(h.workItemPlan.RenewLease),
		PlanGet:                       http.HandlerFunc(h.workItemPlan.GetPlan),
		PlanStatus:                    http.HandlerFunc(h.workItemPlan.GetPlanStatus),
		PlanProposals:                 http.HandlerFunc(h.workItemPlan.SubmitProposal),
		ProposalGet:                   http.HandlerFunc(h.workItemPlan.GetProposal),
		ProposalAcceptance:            http.HandlerFunc(h.workItemPlan.AcceptProposal),
		TaskGet:                       http.HandlerFunc(h.workItem.GetTask),
		TaskCheckoutJobs:              http.HandlerFunc(h.checkout.CreateJob),
		TaskExecutionJobs:             http.HandlerFunc(h.execution.CreateJob),
		CheckoutJobGet:                http.HandlerFunc(h.checkout.InspectJob),
		CheckoutJobLeases:             http.HandlerFunc(h.checkout.AcquireLease),
		CheckoutJobReceipts:           http.HandlerFunc(h.checkout.SubmitReceipt),
		ExecutionJobLeases:            http.HandlerFunc(h.execution.AcquireLease),
		ExecutionJobRuns:              http.HandlerFunc(h.execution.StartRun),
		RunCommandPlans:               http.HandlerFunc(h.execution.CreateCommandPlan),
		RunCommandPlan:                http.HandlerFunc(h.execution.GetCommandPlan),
		RunReceipts:                   http.HandlerFunc(h.execution.SubmitReceipt),
		RunnerCapabilityReports:       http.HandlerFunc(h.execution.CreateRunnerCapabilityReport),
		ClarificationAnswers:          http.HandlerFunc(h.clarification.RecordAnswer),
		ClarificationAnswerApply:      http.HandlerFunc(h.clarification.ApplyAnswer),
	}
}

func newHTTPServer(ctx context.Context, cfg config.Config) (*http.Server, func(), error) {
	healthHandler := health.NewHandler()
	versionHandler := version.NewHandler()

	if !cfg.DatabaseConfigured() {
		router := httpserver.WithCORS(httpserver.NewRouter(databaseUnavailableRouteHandlers(healthHandler, versionHandler)), cfg.CORS.AllowedOrigins)
		return httpserver.NewServer(cfg.Addr, router), func() {}, nil
	}

	pool, err := postgres.OpenPool(ctx, cfg.Database)
	if err != nil {
		return nil, nil, fmt.Errorf("open project context db: %w", err)
	}
	cleanup := pool.Close

	stores := newPostgresStores(pool)
	txRunner := store.NewPostgresTransactionRunner(pool)
	services := newAppServices(stores, txRunner, cfg.AuthJWTSecret)
	handlers := newAppHandlers(services)

	router := httpserver.WithCORS(httpserver.NewRouter(handlers.routeHandlers(healthHandler, versionHandler)), cfg.CORS.AllowedOrigins)

	return httpserver.NewServer(cfg.Addr, router), cleanup, nil
}

func databaseUnavailableRouteHandlers(healthHandler *health.Handler, versionHandler http.Handler) httpserver.RouteHandlers {
	unavailable := httpserver.DatabaseNotConfiguredHandler()
	return httpserver.RouteHandlers{
		Livez:                         http.HandlerFunc(healthHandler.Livez),
		Readyz:                        http.HandlerFunc(healthHandler.Readyz),
		Version:                       versionHandler,
		AuthLogin:                     unavailable,
		CLILoginPage:                  unavailable,
		CLILoginSubmit:                unavailable,
		AuthCLIExchange:               unavailable,
		AuthRefresh:                   unavailable,
		AuthChangePassword:            unavailable,
		AuthLogout:                    unavailable,
		Me:                            unavailable,
		OrganizationUsersList:         unavailable,
		OrganizationUsersCreate:       unavailable,
		OrganizationUsersPatch:        unavailable,
		OrganizationUsersReset:        unavailable,
		OrganizationRepositoryContext: unavailable,
		RepositoryContextInit:         unavailable,
		RepositoryContextSnapshot:     unavailable,
		ProjectRepoBindingInit:        unavailable,
		IntakeSubmit:                  unavailable,
		IntakeGet:                     unavailable,
		IntakePromote:                 unavailable,
		GoalReadiness:                 unavailable,
		GoalContinuation:              unavailable,
		ClarificationContinuation:     unavailable,
		QualificationFeed:             unavailable,
		GoalClarificationRequests:     unavailable,
		ContractCreate:                unavailable,
		ContractList:                  unavailable,
		ContractGet:                   unavailable,
		ContractCurrentDraft:          unavailable,
		ContractUpdate:                unavailable,
		ContractSubmit:                unavailable,
		ContractApprove:               unavailable,
		ContractPlans:                 unavailable,
		PlanLeases:                    unavailable,
		PlanLeaseGet:                  unavailable,
		PlanLeaseRenew:                unavailable,
		PlanGet:                       unavailable,
		PlanStatus:                    unavailable,
		PlanProposals:                 unavailable,
		ProposalGet:                   unavailable,
		ProposalAcceptance:            unavailable,
		TaskGet:                       unavailable,
		TaskCheckoutJobs:              unavailable,
		TaskExecutionJobs:             unavailable,
		CheckoutJobGet:                unavailable,
		CheckoutJobLeases:             unavailable,
		CheckoutJobReceipts:           unavailable,
		ExecutionJobLeases:            unavailable,
		ExecutionJobRuns:              unavailable,
		RunCommandPlans:               unavailable,
		RunCommandPlan:                unavailable,
		RunReceipts:                   unavailable,
		RunnerCapabilityReports:       unavailable,
		ClarificationAnswers:          unavailable,
		ClarificationAnswerApply:      unavailable,
	}
}
