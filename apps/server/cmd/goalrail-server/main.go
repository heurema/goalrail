package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/heurema/goalrail/apps/server/internal/config"
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

	root := newRootCommand(productionCommandActions(cfg, logger))
	root.SetArgs(os.Args[1:])
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	if err := root.ExecuteContext(ctx); err != nil {
		if len(os.Args) > 1 {
			logger.Error("goalrail-server command failed", "error", err)
			os.Exit(1)
		}
		logger.Error("goalrail-server failed", "error", err)
		os.Exit(1)
	}
}
