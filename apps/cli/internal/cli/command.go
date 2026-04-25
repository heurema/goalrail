package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/heurema/goalrail/apps/cli/internal/exitcode"
)

type Command struct {
	Name    string
	Summary string
	Run     func(context.Context, []string) error
}

type Dispatcher struct {
	Binary   string
	Commands []Command
	Stdout   io.Writer
}

func (d Dispatcher) Run(ctx context.Context, args []string) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		_, err := fmt.Fprint(d.Stdout, RootUsage(d.Binary, d.Commands))
		return err
	}

	if args[0] == "help" {
		if len(args) == 1 {
			_, err := fmt.Fprint(d.Stdout, RootUsage(d.Binary, d.Commands))
			return err
		}
		cmd, ok := d.find(args[1])
		if !ok {
			return exitcode.UsageError(fmt.Errorf("unknown command %q", args[1]))
		}
		return cmd.Run(ctx, []string{"--help"})
	}

	cmd, ok := d.find(args[0])
	if !ok {
		return exitcode.UsageError(fmt.Errorf("unknown command %q", args[0]))
	}

	return cmd.Run(ctx, args[1:])
}

func (d Dispatcher) find(name string) (Command, bool) {
	for _, cmd := range d.Commands {
		if cmd.Name == name {
			return cmd, true
		}
	}
	return Command{}, false
}
