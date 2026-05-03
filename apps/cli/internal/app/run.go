package app

import (
	"context"
	"errors"
	"strings"

	"github.com/heurema/goalrail/apps/cli/internal/clienv"
	"github.com/heurema/goalrail/apps/cli/internal/exitcode"
)

func Run(ctx context.Context, env clienv.Env, args []string) error {
	cmd := NewRootCommand(env)
	cmd.SetArgs(args)
	if err := cmd.ExecuteContext(ctx); err != nil {
		if isCobraUsageError(err) {
			return exitcode.UsageError(normalizeCobraUsageError(err))
		}
		return err
	}
	return nil
}

func isCobraUsageError(err error) bool {
	message := err.Error()
	return strings.HasPrefix(message, "unknown command ") ||
		strings.HasPrefix(message, "unknown flag: ") ||
		strings.HasPrefix(message, "unknown shorthand flag: ")
}

func normalizeCobraUsageError(err error) error {
	message := err.Error()
	if strings.HasPrefix(message, "unknown command ") {
		if index := strings.Index(message, " for "); index >= 0 {
			return errors.New(message[:index])
		}
	}
	return err
}
