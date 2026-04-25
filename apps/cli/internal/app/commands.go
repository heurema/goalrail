package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/heurema/goalrail/apps/cli/internal/cli"
	"github.com/heurema/goalrail/apps/cli/internal/clienv"
	"github.com/heurema/goalrail/apps/cli/internal/contractcmd"
	"github.com/heurema/goalrail/apps/cli/internal/exitcode"
	"github.com/heurema/goalrail/apps/cli/internal/initcmd"
	"github.com/heurema/goalrail/apps/cli/internal/proofcmd"
	"github.com/heurema/goalrail/apps/cli/internal/readinesscmd"
	"github.com/heurema/goalrail/apps/cli/internal/term"
)

func Commands(env clienv.Env, out *term.Output) []cli.Command {
	return []cli.Command{
		{
			Name:    "version",
			Summary: "print the CLI version",
			Run: func(_ context.Context, args []string) error {
				if len(args) > 0 && (args[0] == "--help" || args[0] == "-h") {
					_, err := fmt.Fprint(out.Stdout, "Usage: goalrail version\n")
					return err
				}
				if len(args) > 0 {
					return exitcode.UsageError(errors.New("goalrail version does not accept arguments"))
				}
				_, err := fmt.Fprintln(out.Stdout, Version)
				return err
			},
		},
		{
			Name:    "init",
			Summary: "create a local/demo repo binding draft",
			Run: func(ctx context.Context, args []string) error {
				return initcmd.Run(ctx, out, args)
			},
		},
		{
			Name:    "readiness",
			Summary: "scan local repository readiness evidence",
			Run: func(ctx context.Context, args []string) error {
				return readinesscmd.Run(ctx, out, env.WorkDir, args)
			},
		},
		{
			Name:    "contract",
			Summary: "validate contract JSON files",
			Run: func(ctx context.Context, args []string) error {
				return contractcmd.Run(ctx, out, env.WorkDir, args)
			},
		},
		{
			Name:    "proof",
			Summary: "render proof JSON files",
			Run: func(ctx context.Context, args []string) error {
				return proofcmd.Run(ctx, out, env.WorkDir, args)
			},
		},
	}
}
