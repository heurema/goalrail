package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/app"
	"github.com/heurema/goalrail/apps/server/internal/config"
	"github.com/heurema/goalrail/apps/server/internal/postgres"
	"github.com/heurema/goalrail/apps/server/internal/seed"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "goalrail-server: load config: %v\n", err)
		os.Exit(1)
	}

	level, err := config.ParseLogLevel(cfg.LogLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "goalrail-server: parse log level: %v\n", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))

	if handled, err := runCommand(ctx, cfg, logger, os.Args[1:]); handled {
		if err != nil {
			logger.Error("goalrail-server command failed", "error", err)
			os.Exit(1)
		}
		return
	}

	if err := app.Run(ctx, cfg, logger); err != nil {
		logger.Error("goalrail-server failed", "error", err)
		os.Exit(1)
	}
}

func runCommand(ctx context.Context, cfg config.Config, logger *slog.Logger, args []string) (bool, error) {
	if len(args) == 0 {
		return false, nil
	}
	if len(args) != 2 {
		return true, fmt.Errorf("unsupported command %q", args)
	}

	switch {
	case args[0] == "migrate" && args[1] == "up":
		if err := postgres.MigrateUp(ctx, cfg.DatabaseDSN); err != nil {
			return true, err
		}
		logger.Info("postgres migrations applied")
		return true, nil
	case args[0] == "seed" && args[1] == "dev":
		pool, err := postgres.OpenPool(ctx, cfg.DatabaseDSN)
		if err != nil {
			return true, err
		}
		defer pool.Close()

		if err := seed.RunDevWithPool(ctx, pool, time.Now()); err != nil {
			return true, err
		}
		logger.Info("dev seed applied")
		return true, nil
	default:
		return true, fmt.Errorf("unsupported command %q", args)
	}
}
