package app

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/heurema/goalrail/apps/server/internal/clarification"
	"github.com/heurema/goalrail/apps/server/internal/config"
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
)

type goalStore interface {
	goal.GoalStore
	clarification.GoalReader
}

type eventAppender interface {
	Append(context.Context, spine.Event) error
}

func newHTTPServer(ctx context.Context, cfg config.Config) (*http.Server, func(), error) {
	healthHandler := health.NewHandler()
	versionHandler := version.NewHandler()
	var intakeStore intake.Store = store.NewIntakeStore()
	var goals goalStore = store.NewGoalStore()
	clarificationStore := store.NewClarificationStore()
	clarificationAnswerStore := store.NewClarificationAnswerStore()
	contractSeedStore := store.NewContractSeedStore()
	contractDraftStore := store.NewContractDraftStore()
	var events eventAppender = eventlog.NewEventLog()

	var projectContext intake.ProjectContextResolver
	cleanup := func() {}
	if strings.TrimSpace(cfg.DatabaseDSN) != "" {
		pool, err := postgres.OpenPool(ctx, cfg.DatabaseDSN)
		if err != nil {
			return nil, nil, fmt.Errorf("open project context db: %w", err)
		}
		projectContext = store.NewProjectContextStore(pool)
		intakeStore = store.NewPostgresIntakeStore(pool)
		goals = store.NewPostgresGoalStore(pool)
		events = store.NewPostgresEventLog(pool)
		cleanup = pool.Close
	}

	intakeService := intake.NewService(intakeStore, projectContext, events, intake.SystemClock{}, intake.UUIDGenerator{})
	intakeHandler := httpserver.NewIntakeHandler(intakeService)
	goalService := goal.NewService(intakeStore, goals, events, goal.SystemClock{}, goal.UUIDGenerator{})
	goalHandler := httpserver.NewGoalHandler(goalService)
	clarificationService := clarification.NewService(goals, clarificationStore, clarificationAnswerStore, events, clarification.SystemClock{}, clarification.UUIDGenerator{})
	clarificationHandler := httpserver.NewClarificationHandler(clarificationService)
	contractSeedService := contractseed.NewService(goals, contractSeedStore, events, contractseed.SystemClock{}, contractseed.UUIDGenerator{})
	contractSeedHandler := httpserver.NewContractSeedHandler(contractSeedService)
	contractDraftService := contractdraft.NewService(contractSeedStore, contractDraftStore, events, contractdraft.SystemClock{}, contractdraft.UUIDGenerator{})
	contractDraftHandler := httpserver.NewContractDraftHandler(contractDraftService)

	router := httpserver.NewRouter(httpserver.RouteHandlers{
		Livez:                     http.HandlerFunc(healthHandler.Livez),
		Readyz:                    http.HandlerFunc(healthHandler.Readyz),
		Version:                   versionHandler,
		IntakeSubmit:              http.HandlerFunc(intakeHandler.Submit),
		IntakeGet:                 http.HandlerFunc(intakeHandler.Get),
		IntakePromote:             http.HandlerFunc(goalHandler.PromoteFromIntake),
		GoalReadiness:             http.HandlerFunc(goalHandler.CheckReadiness),
		GoalClarificationRequests: http.HandlerFunc(clarificationHandler.CreateRequest),
		GoalContractSeed:          http.HandlerFunc(contractSeedHandler.Create),
		ContractSeedDraft:         http.HandlerFunc(contractDraftHandler.Create),
		ClarificationAnswers:      http.HandlerFunc(clarificationHandler.RecordAnswer),
		ClarificationAnswerApply:  http.HandlerFunc(clarificationHandler.ApplyAnswer),
	})

	return httpserver.NewServer(cfg.Addr, router), cleanup, nil
}
