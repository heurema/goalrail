package app

import (
	"net/http"

	"github.com/heurema/goalrail/apps/server/internal/config"
	"github.com/heurema/goalrail/apps/server/internal/health"
	"github.com/heurema/goalrail/apps/server/internal/httpserver"
	"github.com/heurema/goalrail/apps/server/internal/version"
)

func newHTTPServer(cfg config.Config) *http.Server {
	healthHandler := health.NewHandler()
	versionHandler := version.NewHandler()

	router := httpserver.NewRouter(httpserver.RouteHandlers{
		Livez:   http.HandlerFunc(healthHandler.Livez),
		Readyz:  http.HandlerFunc(healthHandler.Readyz),
		Version: versionHandler,
	})

	return httpserver.NewServer(cfg.Addr, router)
}
