package app

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/heurema/goalrail/apps/cli/internal/clienv"
	"github.com/heurema/goalrail/apps/cli/internal/exitcode"
)

func TestRunVersion(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	env := clienv.Env{Stdout: &stdout, Stderr: &bytes.Buffer{}, WorkDir: "."}
	if err := Run(context.Background(), env, []string{"version"}); err != nil {
		t.Fatalf("Run(version) error = %v", err)
	}

	if got := strings.TrimSpace(stdout.String()); got != Version {
		t.Fatalf("version output = %q, want %q", got, Version)
	}
}

func TestRunUnknownCommand(t *testing.T) {
	t.Parallel()

	env := clienv.Env{Stdout: &bytes.Buffer{}, Stderr: &bytes.Buffer{}, WorkDir: "."}
	err := Run(context.Background(), env, []string{"unknown"})
	if err == nil {
		t.Fatal("Run(unknown) error = nil, want usage error")
	}
	if got := exitcode.ForError(err); got != exitcode.Usage {
		t.Fatalf("exit code = %d, want %d", got, exitcode.Usage)
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("error = %q, want unknown command", err.Error())
	}
}
