package app

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/heurema/goalrail/apps/cli/internal/clienv"
)

func TestRootCommandVersionUsesCobraArgsAndWriters(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	cmd := NewRootCommand(clienv.Env{WorkDir: "."})
	cmd.SetArgs([]string{"version"})
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext(version) error = %v", err)
	}

	if got := strings.TrimSpace(stdout.String()); got != Version {
		t.Fatalf("stdout = %q, want %q", got, Version)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestRootCommandNestedHelpUsesCobraArgsAndWriters(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	cmd := NewRootCommand(clienv.Env{WorkDir: "."})
	cmd.SetArgs([]string{"readiness", "scan", "--help"})
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext(readiness scan --help) error = %v", err)
	}

	want := "Usage: goalrail readiness scan --path <path> [--format text|json]"
	if got := stdout.String(); !strings.Contains(got, want) {
		t.Fatalf("stdout = %q, want usage containing %q", got, want)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestRootCommandWorkStartHelpUsesCobraArgsAndWriters(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	cmd := NewRootCommand(clienv.Env{WorkDir: "."})
	cmd.SetArgs([]string{"work", "start", "--help"})
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext(work start --help) error = %v", err)
	}

	want := "Usage: goalrail work start --title <title> [--body <body>] [--format text|json]"
	if got := stdout.String(); !strings.Contains(got, want) {
		t.Fatalf("stdout = %q, want usage containing %q", got, want)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestRootCommandProjectStatusHelpUsesCobraArgsAndWriters(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	cmd := NewRootCommand(clienv.Env{WorkDir: "."})
	cmd.SetArgs([]string{"project", "status", "--help"})
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext(project status --help) error = %v", err)
	}

	want := "Usage: goalrail project status [--format text|json]"
	if got := stdout.String(); !strings.Contains(got, want) {
		t.Fatalf("stdout = %q, want usage containing %q", got, want)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestRootCommandLoginRequiresServerURL(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	cmd := NewRootCommand(clienv.Env{WorkDir: "."})
	cmd.SetArgs([]string{"login"})
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("ExecuteContext(login) error = nil, want usage error")
	}
	if !strings.Contains(err.Error(), "missing required server_url") {
		t.Fatalf("error = %v, want missing server_url", err)
	}
}
