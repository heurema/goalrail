package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/heurema/goalrail/apps/cli/internal/app"
	"github.com/heurema/goalrail/apps/cli/internal/clienv"
	"github.com/heurema/goalrail/apps/cli/internal/exitcode"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	env := clienv.Default()
	if err := app.Run(ctx, env, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitcode.ForError(err))
	}
}
