package app

import (
	"net/http"

	"github.com/heurema/goalrail/apps/server/internal/config"
	"github.com/heurema/goalrail/apps/server/internal/eventlog"
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
	events := eventlog.NewEventLog()
	intakeService := intake.NewService(intakeStore, events, intake.SystemClock{}, intake.UUIDGenerator{})
	intakeHandler := httpserver.NewIntakeHandler(intakeService)

	router := httpserver.NewRouter(httpserver.RouteHandlers{
		Livez:        http.HandlerFunc(healthHandler.Livez),
		Readyz:       http.HandlerFunc(healthHandler.Readyz),
		Version:      versionHandler,
		IntakeSubmit: http.HandlerFunc(intakeHandler.Submit),
		IntakeGet:    http.HandlerFunc(intakeHandler.Get),
	})

	return httpserver.NewServer(cfg.Addr, router)
}
