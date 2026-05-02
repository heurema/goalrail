package app

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/heurema/goalrail/apps/server/internal/approvedcontract"
	"github.com/heurema/goalrail/apps/server/internal/clarification"
	"github.com/heurema/goalrail/apps/server/internal/config"
	"github.com/heurema/goalrail/apps/server/internal/contract"
	"github.com/heurema/goalrail/apps/server/internal/contractdraft"
	"github.com/heurema/goalrail/apps/server/internal/contractseed"
	"github.com/heurema/goalrail/apps/server/internal/eventlog"
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
	var intakeStore intake.Store = store.NewIntakeStore()
	var goals goalStore = store.NewGoalStore()
	var clarificationStore clarification.Store = store.NewClarificationStore()
	var clarificationAnswerStore clarification.AnswerStore = store.NewClarificationAnswerStore()
	var clarificationOptions []clarification.Option
	var contracts contractStore = store.NewContractStore()
	var contractSeedStore contractseed.Store = store.NewContractSeedStore()
	var contractDraftStore contractdraft.Store = store.NewContractDraftStore()
	var approvedContractStore approvedcontract.Store = store.NewApprovedContractStore()
	var workItemStore workItemStore = store.NewWorkItemStore()
	var workItemPlanStore workitemplan.PlanStore = store.NewWorkItemPlanStore()
	var workItemPlanProposalStore workitemplan.ProposalStore = store.NewWorkItemPlanProposalStore()
	var acceptanceTransaction workitemplan.AcceptanceTransaction
	var events eventAppender = eventlog.NewEventLog()

	var projectContext intake.ProjectContextResolver
	cleanup := func() {}
	if strings.TrimSpace(cfg.DatabaseDSN) != "" {
		pool, err := postgres.OpenPool(ctx, cfg.DatabaseDSN)
		if err != nil {
			return nil, nil, fmt.Errorf("open project context db: %w", err)
		}
		projectContext = store.NewProjectContextStore(pool)
		intakeStore = store.NewPostgresTransactionalIntakeStore(pool)
		goals = store.NewPostgresTransactionalGoalStore(pool)
		clarificationStore = store.NewPostgresClarificationRequestStore(pool)
		clarificationAnswerStore = store.NewPostgresClarificationAnswerStore(pool)
		clarificationTransactions := store.NewPostgresTransactionalClarificationStore(pool)
		clarificationOptions = append(
			clarificationOptions,
			clarification.WithAnswerRecordingTransaction(clarificationTransactions),
			clarification.WithAnswerApplicationTransaction(clarificationTransactions),
		)
		contracts = store.NewPostgresContractStore(pool)
		contractSeedStore = store.NewPostgresTransactionalContractSeedStore(pool)
		contractDraftStore = store.NewPostgresTransactionalContractDraftStore(pool)
		approvedContractStore = store.NewPostgresTransactionalApprovedContractStore(pool)
		workItemStore = store.NewPostgresTransactionalWorkItemStore(pool)
		workItemPlanStore = store.NewPostgresWorkItemPlanStore(pool)
		workItemPlanProposalStore = store.NewPostgresWorkItemPlanProposalStore(pool)
		acceptanceTransaction = store.NewPostgresTransactionalWorkItemPlanStore(pool)
		events = store.NewPostgresEventLog(pool)
		cleanup = pool.Close
	}

	intakeService := intake.NewService(intakeStore, projectContext, events, intake.SystemClock{}, intake.UUIDGenerator{})
	intakeHandler := httpserver.NewIntakeHandler(intakeService)
	goalService := goal.NewService(intakeStore, goals, events, goal.SystemClock{}, goal.UUIDGenerator{})
	goalHandler := httpserver.NewGoalHandler(goalService)
	clarificationService := clarification.NewService(goals, clarificationStore, clarificationAnswerStore, events, clarification.SystemClock{}, clarification.UUIDGenerator{}, clarificationOptions...)
	clarificationHandler := httpserver.NewClarificationHandler(clarificationService)
	contractSeedService := contractseed.NewService(goals, contracts, contractSeedStore, events, contractseed.SystemClock{}, contractseed.UUIDGenerator{})
	contractDraftService := contractdraft.NewService(contractSeedStore, contracts, contractDraftStore, events, contractdraft.SystemClock{}, contractdraft.UUIDGenerator{})
	approvedContractService := approvedcontract.NewService(contractDraftStore, contracts, approvedContractStore, events, approvedcontract.SystemClock{}, approvedcontract.UUIDGenerator{})
	contractOptions := []contract.Option{}
	if runner, ok := contractSeedStore.(contract.TransactionRunner); ok {
		contractOptions = append(contractOptions, contract.WithTransactionRunner(runner))
	}
	contractService := contract.NewService(contracts, contractSeedService, contractDraftService, approvedContractService, contractOptions...)
	contractHandler := httpserver.NewContractHandler(contractService)
	workItemService := workitem.NewService(workItemStore)
	workItemHandler := httpserver.NewWorkItemHandler(workItemService)
	workItemPlanOptions := []workitemplan.Option{}
	if acceptanceTransaction != nil {
		workItemPlanOptions = append(workItemPlanOptions, workitemplan.WithAcceptanceTransaction(acceptanceTransaction))
	}
	workItemPlanService := workitemplan.NewService(contracts, approvedContractStore, workItemPlanStore, workItemPlanProposalStore, workItemStore, events, workitemplan.SystemClock{}, workitemplan.UUIDGenerator{}, workItemPlanOptions...)
	workItemPlanHandler := httpserver.NewWorkItemPlanHandler(workItemPlanService)

	router := httpserver.NewRouter(httpserver.RouteHandlers{
		Livez:                     http.HandlerFunc(healthHandler.Livez),
		Readyz:                    http.HandlerFunc(healthHandler.Readyz),
		Version:                   versionHandler,
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
