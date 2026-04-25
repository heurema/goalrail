package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/config"
)

const shutdownTimeout = 10 * time.Second

// Run starts the HTTP server and blocks until the context is canceled or the server fails.
func Run(ctx context.Context, cfg config.Config, logger *slog.Logger) error {
	if ctx == nil {
		return errors.New("context must not be nil")
	}
	if logger == nil {
		logger = slog.Default()
	}

	server := newHTTPServer(cfg)
	serverErrors := make(chan error, 1)

	go func() {
		logger.Info("http server starting", "addr", cfg.Addr)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("http server: %w", err)
	case <-ctx.Done():
	}

	logger.Info("http server shutdown started", "timeout", shutdownTimeout.String())
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	shutdownErr := server.Shutdown(shutdownCtx)
	serverErr := <-serverErrors

	if shutdownErr != nil {
		return fmt.Errorf("http server shutdown: %w", shutdownErr)
	}
	if serverErr != nil && !errors.Is(serverErr, http.ErrServerClosed) {
		return fmt.Errorf("http server: %w", serverErr)
	}

	logger.Info("http server stopped")
	return nil
}
