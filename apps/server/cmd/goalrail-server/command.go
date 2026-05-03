package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/spf13/cobra"

	"github.com/heurema/goalrail/apps/server/internal/app"
	"github.com/heurema/goalrail/apps/server/internal/bootstrapowner"
	"github.com/heurema/goalrail/apps/server/internal/config"
	"github.com/heurema/goalrail/apps/server/internal/postgres"
	"github.com/heurema/goalrail/apps/server/internal/seed"
)

type commandActions struct {
	runServer      func(context.Context) error
	migrateUp      func(context.Context) error
	seedDev        func(context.Context) error
	bootstrapOwner func(context.Context, bootstrapowner.Input, io.Writer) error
}

func productionCommandActions(cfg config.Config, logger *slog.Logger) commandActions {
	return commandActions{
		runServer: func(ctx context.Context) error {
			return app.Run(ctx, cfg, logger)
		},
		migrateUp: func(ctx context.Context) error {
			if err := postgres.MigrateUp(ctx, cfg.Database); err != nil {
				return err
			}
			logger.Info("postgres migrations applied")
			return nil
		},
		seedDev: func(ctx context.Context) error {
			pool, err := postgres.OpenPool(ctx, cfg.Database)
			if err != nil {
				return err
			}
			defer pool.Close()

			if err := seed.RunDevWithPool(ctx, pool, time.Now()); err != nil {
				return err
			}
			logger.Info("dev seed applied")
			return nil
		},
		bootstrapOwner: func(ctx context.Context, input bootstrapowner.Input, stdout io.Writer) error {
			return runBootstrapOwner(ctx, cfg, input, stdout)
		},
	}
}

func newRootCommand(actions commandActions) *cobra.Command {
	root := &cobra.Command{
		Use:           "goalrail-server",
		Short:         "Run the Goalrail server",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return actions.runServer(cmd.Context())
		},
	}

	root.AddCommand(newMigrateCommand(actions))
	root.AddCommand(newSeedCommand(actions))
	root.AddCommand(newBootstrapCommand(actions))
	return root
}

func newMigrateCommand(actions commandActions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Run database migrations",
		RunE: func(*cobra.Command, []string) error {
			return fmt.Errorf("unsupported command %q", []string{"migrate"})
		},
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "up",
		Short: "Apply database migrations",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return actions.migrateUp(cmd.Context())
		},
	})
	return cmd
}

func newSeedCommand(actions commandActions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Seed local development data",
		RunE: func(*cobra.Command, []string) error {
			return fmt.Errorf("unsupported command %q", []string{"seed"})
		},
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "dev",
		Short: "Apply the idempotent development seed",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return actions.seedDev(cmd.Context())
		},
	})
	return cmd
}

func newBootstrapCommand(actions commandActions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bootstrap",
		Short: "Bootstrap self-hosted setup data",
		RunE: func(*cobra.Command, []string) error {
			return fmt.Errorf("unsupported command %q", []string{"bootstrap"})
		},
	}
	cmd.AddCommand(newBootstrapOwnerCommand(actions))
	return cmd
}

func newBootstrapOwnerCommand(actions commandActions) *cobra.Command {
	var input bootstrapowner.Input
	cmd := &cobra.Command{
		Use:   "owner",
		Short: "Bootstrap the first self-hosted owner",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			normalized, err := bootstrapowner.NormalizeInput(input)
			if err != nil {
				return err
			}
			return actions.bootstrapOwner(cmd.Context(), normalized, cmd.OutOrStdout())
		},
	}
	cmd.Flags().StringVar(&input.Email, "email", "", "owner email")
	cmd.Flags().StringVar(&input.DisplayName, "display-name", "", "owner display name")
	cmd.Flags().StringVar(&input.OrganizationSlug, "organization-slug", "", "organization slug")
	cmd.Flags().StringVar(&input.OrganizationName, "organization-name", "", "organization name")
	cmd.Flags().StringVar(&input.PublicBaseURL, "public-base-url", "", "public base URL")
	return cmd
}

func runBootstrapOwner(ctx context.Context, cfg config.Config, input bootstrapowner.Input, stdout io.Writer) error {
	pool, err := postgres.OpenPool(ctx, cfg.Database)
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
