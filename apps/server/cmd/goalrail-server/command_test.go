package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/spf13/cobra"

	"github.com/heurema/goalrail/apps/server/internal/bootstrapowner"
)

func TestRootCommandRunsServerWithoutArgs(t *testing.T) {
	var serverCalls int
	cmd := newRootCommand(commandActions{
		runServer: func(context.Context) error {
			serverCalls++
			return nil
		},
		migrateUp: func(context.Context) error {
			t.Fatal("migrateUp called")
			return nil
		},
		seedDev: func(context.Context) error {
			t.Fatal("seedDev called")
			return nil
		},
		bootstrapOwner: func(context.Context, bootstrapowner.Input, io.Writer) error {
			t.Fatal("bootstrapOwner called")
			return nil
		},
	})
	stdout, stderr, err := executeCommand(cmd)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if serverCalls != 1 {
		t.Fatalf("server calls = %d, want 1", serverCalls)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

func TestRootCommandRunsMigrateUp(t *testing.T) {
	var migrateCalls int
	cmd := newRootCommand(commandActions{
		runServer: func(context.Context) error {
			t.Fatal("runServer called")
			return nil
		},
		migrateUp: func(context.Context) error {
			migrateCalls++
			return nil
		},
		seedDev: func(context.Context) error {
			t.Fatal("seedDev called")
			return nil
		},
		bootstrapOwner: func(context.Context, bootstrapowner.Input, io.Writer) error {
			t.Fatal("bootstrapOwner called")
			return nil
		},
	})
	if _, _, err := executeCommand(cmd, "migrate", "up"); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if migrateCalls != 1 {
		t.Fatalf("migrate calls = %d, want 1", migrateCalls)
	}
}

func TestRootCommandRunsSeedDev(t *testing.T) {
	var seedCalls int
	cmd := newRootCommand(commandActions{
		runServer: func(context.Context) error {
			t.Fatal("runServer called")
			return nil
		},
		migrateUp: func(context.Context) error {
			t.Fatal("migrateUp called")
			return nil
		},
		seedDev: func(context.Context) error {
			seedCalls++
			return nil
		},
		bootstrapOwner: func(context.Context, bootstrapowner.Input, io.Writer) error {
			t.Fatal("bootstrapOwner called")
			return nil
		},
	})
	if _, _, err := executeCommand(cmd, "seed", "dev"); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if seedCalls != 1 {
		t.Fatalf("seed calls = %d, want 1", seedCalls)
	}
}

func TestRootCommandReturnsUsageErrorForIncompleteParentCommands(t *testing.T) {
	for _, tt := range []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "migrate",
			args:    []string{"migrate"},
			wantErr: "unsupported command [\"migrate\"]",
		},
		{
			name:    "seed",
			args:    []string{"seed"},
			wantErr: "unsupported command [\"seed\"]",
		},
		{
			name:    "bootstrap",
			args:    []string{"bootstrap"},
			wantErr: "unsupported command [\"bootstrap\"]",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newRootCommand(commandActions{
				runServer: func(context.Context) error {
					t.Fatal("runServer called")
					return nil
				},
				migrateUp: func(context.Context) error {
					t.Fatal("migrateUp called")
					return nil
				},
				seedDev: func(context.Context) error {
					t.Fatal("seedDev called")
					return nil
				},
				bootstrapOwner: func(context.Context, bootstrapowner.Input, io.Writer) error {
					t.Fatal("bootstrapOwner called")
					return nil
				},
			})
			stdout, stderr, err := executeCommand(cmd, tt.args...)
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("Execute() error = %v, want %q", err, tt.wantErr)
			}
			if stdout != "" {
				t.Fatalf("stdout = %q, want empty", stdout)
			}
			if stderr != "" {
				t.Fatalf("stderr = %q, want empty", stderr)
			}
		})
	}
}

func TestRootCommandRunsBootstrapOwnerWithFlags(t *testing.T) {
	var gotInput bootstrapowner.Input
	var bootstrapCalls int
	cmd := newRootCommand(commandActions{
		runServer: func(context.Context) error {
			t.Fatal("runServer called")
			return nil
		},
		migrateUp: func(context.Context) error {
			t.Fatal("migrateUp called")
			return nil
		},
		seedDev: func(context.Context) error {
			t.Fatal("seedDev called")
			return nil
		},
		bootstrapOwner: func(_ context.Context, input bootstrapowner.Input, stdout io.Writer) error {
			bootstrapCalls++
			gotInput = input
			_, err := io.WriteString(stdout, "temporary_password=temporary-password\n")
			return err
		},
	})
	stdout, stderr, err := executeCommand(
		cmd,
		"bootstrap",
		"owner",
		"--email",
		"Owner@Example.COM",
		"--display-name",
		"Owner User",
		"--organization-slug",
		"Primary",
		"--organization-name",
		"Primary Org",
		"--public-base-url",
		"https://goalrail.example.com/",
	)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if bootstrapCalls != 1 {
		t.Fatalf("bootstrap calls = %d, want 1", bootstrapCalls)
	}
	if gotInput.Email != "owner@example.com" {
		t.Fatalf("Email = %q, want normalized email", gotInput.Email)
	}
	if gotInput.OrganizationSlug != "primary" {
		t.Fatalf("OrganizationSlug = %q, want normalized slug", gotInput.OrganizationSlug)
	}
	if gotInput.PublicBaseURL != "https://goalrail.example.com" {
		t.Fatalf("PublicBaseURL = %q, want normalized URL", gotInput.PublicBaseURL)
	}
	if stdout != "temporary_password=temporary-password\n" {
		t.Fatalf("stdout = %q, want temporary password line", stdout)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

func TestRootCommandPreservesBootstrapOwnerExistingCredentialStdout(t *testing.T) {
	cmd := newRootCommand(commandActions{
		runServer: func(context.Context) error {
			t.Fatal("runServer called")
			return nil
		},
		migrateUp: func(context.Context) error {
			t.Fatal("migrateUp called")
			return nil
		},
		seedDev: func(context.Context) error {
			t.Fatal("seedDev called")
			return nil
		},
		bootstrapOwner: func(_ context.Context, _ bootstrapowner.Input, stdout io.Writer) error {
			_, err := io.WriteString(stdout, "temporary_password_already_exists=true\n")
			return err
		},
	})
	stdout, _, err := executeCommand(
		cmd,
		"bootstrap",
		"owner",
		"--email",
		"owner@example.com",
		"--display-name",
		"Owner User",
		"--organization-slug",
		"primary",
		"--organization-name",
		"Primary Org",
		"--public-base-url",
		"https://goalrail.example.com",
	)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if stdout != "temporary_password_already_exists=true\n" {
		t.Fatalf("stdout = %q, want existing credential line", stdout)
	}
}

func TestRootCommandValidatesBootstrapOwnerFlagsBeforeAction(t *testing.T) {
	cmd := newRootCommand(commandActions{
		runServer: func(context.Context) error {
			t.Fatal("runServer called")
			return nil
		},
		migrateUp: func(context.Context) error {
			t.Fatal("migrateUp called")
			return nil
		},
		seedDev: func(context.Context) error {
			t.Fatal("seedDev called")
			return nil
		},
		bootstrapOwner: func(context.Context, bootstrapowner.Input, io.Writer) error {
			t.Fatal("bootstrapOwner called")
			return nil
		},
	})
	_, _, err := executeCommand(
		cmd,
		"bootstrap",
		"owner",
		"--email",
		"owner@example.com",
		"--display-name",
		"Owner User",
		"--organization-slug",
		"primary",
		"--organization-name",
		"Primary Org",
	)
	if !errors.Is(err, bootstrapowner.ErrInvalidInput) {
		t.Fatalf("Execute() error = %v, want ErrInvalidInput", err)
	}
}

func executeCommand(cmd *cobra.Command, args ...string) (string, string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetArgs(args)
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	err := cmd.ExecuteContext(context.Background())
	return stdout.String(), stderr.String(), err
}
