package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/heurema/goalrail/apps/worker/internal/planningworker"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := planningworker.Config{
		ServerURL:       os.Getenv("GOALRAIL_WORKER_SERVER_URL"),
		WorkerID:        os.Getenv("GOALRAIL_WORKER_ID"),
		PollInterval:    10 * time.Second,
		LeaseTTLSeconds: 900,
		Once:            false,
		LogWriter:       os.Stderr,
	}

	flags := flag.NewFlagSet("goalrail-worker", flag.ExitOnError)
	flags.StringVar(&cfg.ServerURL, "server-url", cfg.ServerURL, "Goalrail API server URL; also configurable with GOALRAIL_WORKER_SERVER_URL")
	flags.StringVar(&cfg.WorkerID, "worker-id", cfg.WorkerID, "worker identity; also configurable with GOALRAIL_WORKER_ID")
	flags.DurationVar(&cfg.PollInterval, "poll-interval", cfg.PollInterval, "poll interval when no planning lease is available")
	flags.IntVar(&cfg.LeaseTTLSeconds, "lease-ttl-seconds", cfg.LeaseTTLSeconds, "requested planning lease TTL in seconds")
	flags.BoolVar(&cfg.Once, "once", cfg.Once, "run one poll iteration and exit")
	if err := flags.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	if err := planningworker.Run(ctx, cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
