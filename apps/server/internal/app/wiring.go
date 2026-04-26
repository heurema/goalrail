package app

import (
	"net/http"

	"github.com/heurema/goalrail/apps/server/internal/clarification"
	"github.com/heurema/goalrail/apps/server/internal/config"
	"github.com/heurema/goalrail/apps/server/internal/eventlog"
	"github.com/heurema/goalrail/apps/server/internal/goal"
	"github.com/heurema/goalrail/apps/server/internal/health"
	"github.com/heurema/goalrail/apps/server/internal/httpserver"
	"github.com/heurema/goalrail/apps/server/internal/intake"
	"github.com/heurema/goalrail/apps/server/internal/store"
	"github.com/heurema/goalrail/apps/server/internal/version"
)

func newHTTPServer(cfg config.Config) *http.Server {
	healthHandler := health.NewHandler()
	versionHandler := version.NewHandler()
	intakeStore := store.NewIntakeStore()
	goalStore := store.NewGoalStore()
	clarificationStore := store.NewClarificationStore()
	clarificationAnswerStore := store.NewClarificationAnswerStore()
	events := eventlog.NewEventLog()
	intakeService := intake.NewService(intakeStore, events, intake.SystemClock{}, intake.UUIDGenerator{})
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

	return httpserver.NewServer(cfg.Addr, router)
}
