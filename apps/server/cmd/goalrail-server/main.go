package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/app"
	"github.com/heurema/goalrail/apps/server/internal/bootstrapowner"
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

	if handled, err := runCommand(ctx, cfg, logger, os.Args[1:], os.Stdout); handled {
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

func runCommand(ctx context.Context, cfg config.Config, logger *slog.Logger, args []string, stdout io.Writer) (bool, error) {
	if len(args) == 0 {
		return false, nil
	}
	if len(args) < 2 {
		return true, fmt.Errorf("unsupported command %q", args)
	}

	switch {
	case len(args) == 2 && args[0] == "migrate" && args[1] == "up":
		if err := postgres.MigrateUp(ctx, cfg.DatabaseDSN); err != nil {
			return true, err
		}
		logger.Info("postgres migrations applied")
		return true, nil
	case len(args) == 2 && args[0] == "seed" && args[1] == "dev":
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
	case args[0] == "bootstrap" && args[1] == "owner":
		return true, runBootstrapOwnerCommand(ctx, cfg, args[2:], stdout)
	default:
		return true, fmt.Errorf("unsupported command %q", args)
	}
}

func runBootstrapOwnerCommand(ctx context.Context, cfg config.Config, args []string, stdout io.Writer) error {
	input, err := parseBootstrapOwnerFlags(args)
	if err != nil {
		return err
	}

	pool, err := postgres.OpenPool(ctx, cfg.DatabaseDSN)
	if err != nil {
		return err
	}
	defer pool.Close()

	result, err := bootstrapowner.RunWithPool(ctx, pool, input)
	if err != nil {
		return err
	}
	if result.PasswordCredentialCreated {
		fmt.Fprintf(stdout, "temporary_password=%s\n", result.TemporaryPassword)
		return nil
	}
	fmt.Fprintln(stdout, "temporary_password_already_exists=true")
	return nil
}

func parseBootstrapOwnerFlags(args []string) (bootstrapowner.Input, error) {
	var input bootstrapowner.Input
	flags := flag.NewFlagSet("goalrail-server bootstrap owner", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&input.Email, "email", "", "owner email")
	flags.StringVar(&input.DisplayName, "display-name", "", "owner display name")
	flags.StringVar(&input.OrganizationSlug, "organization-slug", "", "organization slug")
	flags.StringVar(&input.OrganizationName, "organization-name", "", "organization name")
	flags.StringVar(&input.PublicBaseURL, "public-base-url", "", "public base URL")
	if err := flags.Parse(args); err != nil {
		return bootstrapowner.Input{}, err
	}
	if flags.NArg() != 0 {
		return bootstrapowner.Input{}, fmt.Errorf("unexpected bootstrap owner arguments: %v", flags.Args())
	}
	return bootstrapowner.NormalizeInput(input)
}
