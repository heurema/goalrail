package app

import (
	"context"

	"github.com/heurema/goalrail/apps/cli/internal/cli"
	"github.com/heurema/goalrail/apps/cli/internal/clienv"
	"github.com/heurema/goalrail/apps/cli/internal/term"
)

func Run(ctx context.Context, env clienv.Env, args []string) error {
	out := term.New(env.Stdout, env.Stderr)
	dispatcher := cli.Dispatcher{
		Binary:   "goalrail",
		Commands: Commands(env, out),
		Stdout:   env.Stdout,
	}
	return dispatcher.Run(ctx, args)
}
