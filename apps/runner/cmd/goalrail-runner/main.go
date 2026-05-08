package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/heurema/goalrail/apps/runner/internal/checkoutrunner"
	"github.com/heurema/goalrail/apps/runner/internal/executionrunner"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := checkoutrunner.Config{
		ServerURL:       os.Getenv("GOALRAIL_RUNNER_SERVER_URL"),
		BearerToken:     os.Getenv("GOALRAIL_RUNNER_BEARER_TOKEN"),
		ProjectID:       os.Getenv("GOALRAIL_RUNNER_PROJECT_ID"),
		RepoBindingID:   os.Getenv("GOALRAIL_RUNNER_REPO_BINDING_ID"),
		RunnerID:        os.Getenv("GOALRAIL_RUNNER_ID"),
		WorkspaceRef:    os.Getenv("GOALRAIL_RUNNER_WORKSPACE_REF"),
		CommitSHA:       os.Getenv("GOALRAIL_RUNNER_COMMIT_SHA"),
		BaselineID:      os.Getenv("GOALRAIL_RUNNER_BASELINE_ID"),
		OverlayID:       os.Getenv("GOALRAIL_RUNNER_OVERLAY_ID"),
		PollInterval:    10 * time.Second,
		LeaseTTLSeconds: 900,
		Once:            false,
		LogWriter:       os.Stderr,
	}
	mode := strings.TrimSpace(os.Getenv("GOALRAIL_RUNNER_MODE"))
	if mode == "" {
		mode = "checkout"
	}
	if raw := strings.TrimSpace(os.Getenv("GOALRAIL_RUNNER_DIRTY")); raw == "true" || raw == "1" {
		cfg.Dirty = true
	}
	if raw := strings.TrimSpace(os.Getenv("GOALRAIL_RUNNER_PARTIAL")); raw == "true" || raw == "1" {
		cfg.Partial = true
	}

	flags := flag.NewFlagSet("goalrail-runner", flag.ExitOnError)
	flags.StringVar(&mode, "mode", mode, "runner mode: checkout, execution-start, or execution-receipt; also configurable with GOALRAIL_RUNNER_MODE")
	flags.StringVar(&cfg.ServerURL, "server-url", cfg.ServerURL, "Goalrail API server URL; also configurable with GOALRAIL_RUNNER_SERVER_URL")
	flags.StringVar(&cfg.ProjectID, "project-id", cfg.ProjectID, "Project scope for runner leases; also configurable with GOALRAIL_RUNNER_PROJECT_ID")
	flags.StringVar(&cfg.RepoBindingID, "repo-binding-id", cfg.RepoBindingID, "RepoBinding scope for runner leases; also configurable with GOALRAIL_RUNNER_REPO_BINDING_ID")
	flags.StringVar(&cfg.RunnerID, "runner-id", cfg.RunnerID, "runner identity; also configurable with GOALRAIL_RUNNER_ID")
	flags.StringVar(&cfg.WorkspaceRef, "workspace-ref", cfg.WorkspaceRef, "mounted workspace reference; also configurable with GOALRAIL_RUNNER_WORKSPACE_REF")
	flags.StringVar(&cfg.CommitSHA, "commit-sha", cfg.CommitSHA, "workspace commit SHA; also configurable with GOALRAIL_RUNNER_COMMIT_SHA")
	flags.StringVar(&cfg.BaselineID, "baseline-id", cfg.BaselineID, "optional repository baseline id")
	flags.StringVar(&cfg.OverlayID, "overlay-id", cfg.OverlayID, "optional repository overlay id")
	flags.BoolVar(&cfg.Dirty, "dirty", cfg.Dirty, "mark workspace receipt dirty")
	flags.BoolVar(&cfg.Partial, "partial", cfg.Partial, "mark workspace receipt partial")
	flags.DurationVar(&cfg.PollInterval, "poll-interval", cfg.PollInterval, "poll interval when no checkout job is available")
	flags.IntVar(&cfg.LeaseTTLSeconds, "lease-ttl-seconds", cfg.LeaseTTLSeconds, "requested lease TTL in seconds")
	flags.BoolVar(&cfg.Once, "once", cfg.Once, "run one poll iteration and exit")
	if err := flags.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	var err error
	switch strings.TrimSpace(mode) {
	case "checkout":
		err = checkoutrunner.Run(ctx, cfg)
	case "execution-start":
		err = executionrunner.Run(ctx, executionrunner.Config{
			ServerURL:       cfg.ServerURL,
			BearerToken:     cfg.BearerToken,
			ProjectID:       cfg.ProjectID,
			RepoBindingID:   cfg.RepoBindingID,
			RunnerID:        cfg.RunnerID,
			PollInterval:    cfg.PollInterval,
			LeaseTTLSeconds: cfg.LeaseTTLSeconds,
			Once:            cfg.Once,
			LogWriter:       cfg.LogWriter,
		})
	case "execution-receipt":
		err = executionrunner.Run(ctx, executionrunner.Config{
			ServerURL:       cfg.ServerURL,
			BearerToken:     cfg.BearerToken,
			ProjectID:       cfg.ProjectID,
			RepoBindingID:   cfg.RepoBindingID,
			RunnerID:        cfg.RunnerID,
			WorkspaceRef:    cfg.WorkspaceRef,
			CommitSHA:       cfg.CommitSHA,
			BaselineID:      cfg.BaselineID,
			OverlayID:       cfg.OverlayID,
			SubmitReceipt:   true,
			PollInterval:    cfg.PollInterval,
			LeaseTTLSeconds: cfg.LeaseTTLSeconds,
			Once:            cfg.Once,
			LogWriter:       cfg.LogWriter,
		})
	default:
		err = fmt.Errorf("unsupported runner mode %q", mode)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
