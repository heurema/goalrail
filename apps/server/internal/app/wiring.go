package app

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/heurema/goalrail/apps/server/internal/clarification"
	"github.com/heurema/goalrail/apps/server/internal/config"
	"github.com/heurema/goalrail/apps/server/internal/eventlog"
	"github.com/heurema/goalrail/apps/server/internal/goal"
	"github.com/heurema/goalrail/apps/server/internal/health"
	"github.com/heurema/goalrail/apps/server/internal/httpserver"
	"github.com/heurema/goalrail/apps/server/internal/intake"
	"github.com/heurema/goalrail/apps/server/internal/postgres"
	"github.com/heurema/goalrail/apps/server/internal/store"
	"github.com/heurema/goalrail/apps/server/internal/version"
)

func newHTTPServer(ctx context.Context, cfg config.Config) (*http.Server, func(), error) {
	healthHandler := health.NewHandler()
	versionHandler := version.NewHandler()
	intakeStore := store.NewIntakeStore()
	goalStore := store.NewGoalStore()
	clarificationStore := store.NewClarificationStore()
	clarificationAnswerStore := store.NewClarificationAnswerStore()
	events := eventlog.NewEventLog()

	var projectContext intake.ProjectContextResolver
	cleanup := func() {}
	if strings.TrimSpace(cfg.DatabaseDSN) != "" {
		pool, err := postgres.OpenPool(ctx, cfg.DatabaseDSN)
		if err != nil {
			return nil, nil, fmt.Errorf("open project context db: %w", err)
		}
		projectContext = store.NewProjectContextStore(pool)
		cleanup = pool.Close
	}

	intakeService := intake.NewService(intakeStore, projectContext, events, intake.SystemClock{}, intake.UUIDGenerator{})
	intakeHandler := httpserver.NewIntakeHandler(intakeService)
	goalService := goal.NewService(intakeStore, goalStore, events, goal.SystemClock{}, goal.UUIDGenerator{})
	goalHandler := httpserver.NewGoalHandler(goalService)
	clarificationService := clarification.NewService(goalStore, clarificationStore, clarificationAnswerStore, events, clarification.SystemClock{}, clarification.UUIDGenerator{})
	clarificationHandler := httpserver.NewClarificationHandler(clarificationService)

	router := httpserver.NewRouter(httpserver.RouteHandlers{
		Livez:                     http.HandlerFunc(healthHandler.Livez),
		Readyz:                    http.HandlerFunc(healthHandler.Readyz),
		Version:                   versionHandler,
		IntakeSubmit:              http.HandlerFunc(intakeHandler.Submit),
		IntakeGet:                 http.HandlerFunc(intakeHandler.Get),
		IntakePromote:             http.HandlerFunc(goalHandler.PromoteFromIntake),
		GoalReadiness:             http.HandlerFunc(goalHandler.CheckReadiness),
		GoalClarificationRequests: http.HandlerFunc(clarificationHandler.CreateRequest),
		ClarificationAnswers:      http.HandlerFunc(clarificationHandler.RecordAnswer),
		ClarificationAnswerApply:  http.HandlerFunc(clarificationHandler.ApplyAnswer),
	})

	return httpserver.NewServer(cfg.Addr, router), cleanup, nil
}
