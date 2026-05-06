package app

import (
	"errors"
	"fmt"
	"strings"

	"github.com/heurema/goalrail/apps/cli/internal/clienv"
	"github.com/heurema/goalrail/apps/cli/internal/contractcmd"
	"github.com/heurema/goalrail/apps/cli/internal/exitcode"
	"github.com/heurema/goalrail/apps/cli/internal/initcmd"
	"github.com/heurema/goalrail/apps/cli/internal/logincmd"
	"github.com/heurema/goalrail/apps/cli/internal/projectcmd"
	"github.com/heurema/goalrail/apps/cli/internal/proofcmd"
	"github.com/heurema/goalrail/apps/cli/internal/readinesscmd"
	"github.com/heurema/goalrail/apps/cli/internal/term"
	"github.com/heurema/goalrail/apps/cli/internal/workcmd"
	"github.com/spf13/cobra"
)

type commandSummary struct {
	name    string
	summary string
}

var rootCommands = []commandSummary{
	{name: "version", summary: "print the CLI version"},
	{name: "login", summary: "authenticate to a Goalrail server"},
	{name: "init", summary: "initialize repository metadata"},
	{name: "project", summary: "scan and report local project freshness"},
	{name: "work", summary: "start server-backed work from a local project marker"},
	{name: "readiness", summary: "scan local repository readiness evidence"},
	{name: "contract", summary: "validate contract JSON files"},
	{name: "proof", summary: "render proof JSON files"},
}

// NewRootCommand builds the Cobra command tree for tests and process execution.
func NewRootCommand(env clienv.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "goalrail",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprint(cmd.OutOrStdout(), RootUsage())
			return err
		},
	}
	cmd.SetOut(env.Stdout)
	cmd.SetErr(env.Stderr)
	cmd.CompletionOptions.DisableDefaultCmd = true
	cmd.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		_, _ = fmt.Fprint(cmd.OutOrStdout(), helpFor(cmd))
	})
	cmd.SetUsageFunc(func(cmd *cobra.Command) error {
		_, err := fmt.Fprint(cmd.OutOrStdout(), helpFor(cmd))
		return err
	})

	cmd.AddCommand(
		newVersionCommand(),
		newLoginCommand(),
		newInitCommand(env),
		newProjectCommand(env),
		newWorkCommand(env),
		newReadinessCommand(env),
		newContractCommand(env),
		newProofCommand(env),
	)
	cmd.SetHelpCommand(newHelpCommand(cmd))
	return cmd
}

func RootUsage() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Usage: goalrail <command> [options]\n\n")
	b.WriteString("Goalrail CLI foundation for local repository context, server-backed intake, and local readiness evidence. This CLI does not implement hosted execution, gate, or proof generation.\n\n")
	b.WriteString("Commands:\n")
	for _, cmd := range rootCommands {
		fmt.Fprintf(&b, "  %-10s %s\n", cmd.name, cmd.summary)
	}
	fmt.Fprintf(&b, "\nRun goalrail <command> --help for command usage.\n")
	return b.String()
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:                "version",
		Short:              "print the CLI version",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 && (args[0] == "--help" || args[0] == "-h") {
				_, err := fmt.Fprint(cmd.OutOrStdout(), versionUsage())
				return err
			}
			if len(args) > 0 {
				return exitcode.UsageError(errors.New("goalrail version does not accept arguments"))
			}
			_, err := fmt.Fprintln(cmd.OutOrStdout(), Version)
			return err
		},
	}
}

func newLoginCommand() *cobra.Command {
	return &cobra.Command{
		Use:                "login <server_url>",
		Short:              "authenticate to a Goalrail server",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return logincmd.Run(cmd.Context(), outputFor(cmd), args, logincmd.Options{})
		},
	}
}

func newInitCommand(env clienv.Env) *cobra.Command {
	return &cobra.Command{
		Use:                "init",
		Short:              "initialize repository metadata",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return initcmd.Run(cmd.Context(), outputFor(cmd), env.WorkDir, args)
		},
	}
}

func newProjectCommand(env clienv.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:                "project",
		Short:              "scan and report local project freshness",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return projectcmd.Run(cmd.Context(), outputFor(cmd), env.WorkDir, args)
		},
	}
	cmd.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		_, _ = fmt.Fprint(cmd.OutOrStdout(), projectcmd.Usage())
	})
	cmd.SetUsageFunc(func(cmd *cobra.Command) error {
		_, err := fmt.Fprint(cmd.OutOrStdout(), projectcmd.Usage())
		return err
	})
	cmd.AddCommand(&cobra.Command{
		Use:                "scan",
		Short:              "build or refresh local Project Scan evidence",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return projectcmd.Run(cmd.Context(), outputFor(cmd), env.WorkDir, append([]string{"scan"}, args...))
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:                "status",
		Short:              "report local Project Scan freshness",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return projectcmd.Run(cmd.Context(), outputFor(cmd), env.WorkDir, append([]string{"status"}, args...))
		},
	})
	return cmd
}

func newWorkCommand(env clienv.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:                "work",
		Short:              "start server-backed work from a local project marker",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return workcmd.Run(cmd.Context(), outputFor(cmd), env.WorkDir, args)
		},
	}
	cmd.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		_, _ = fmt.Fprint(cmd.OutOrStdout(), workcmd.Usage())
	})
	cmd.SetUsageFunc(func(cmd *cobra.Command) error {
		_, err := fmt.Fprint(cmd.OutOrStdout(), workcmd.Usage())
		return err
	})
	cmd.AddCommand(&cobra.Command{
		Use:                "start",
		Short:              "create a server-backed IntakeRecord and Goal",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return workcmd.Run(cmd.Context(), outputFor(cmd), env.WorkDir, append([]string{"start"}, args...))
		},
	})
	return cmd
}

func newReadinessCommand(env clienv.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:                "readiness",
		Short:              "scan local repository readiness evidence",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 && (args[0] == "--help" || args[0] == "-h") {
				_, err := fmt.Fprint(cmd.OutOrStdout(), readinesscmd.Usage())
				return err
			}
			if len(args) > 0 {
				return exitcode.UsageError(fmt.Errorf("unknown readiness command %q", args[0]))
			}
			_, err := fmt.Fprint(cmd.OutOrStdout(), readinesscmd.Usage())
			return err
		},
	}
	cmd.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		_, _ = fmt.Fprint(cmd.OutOrStdout(), readinesscmd.Usage())
	})
	cmd.SetUsageFunc(func(cmd *cobra.Command) error {
		_, err := fmt.Fprint(cmd.OutOrStdout(), readinesscmd.Usage())
		return err
	})
	cmd.AddCommand(&cobra.Command{
		Use:                "scan",
		Short:              "scan local repository readiness evidence",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return readinesscmd.Run(cmd.Context(), outputFor(cmd), env.WorkDir, append([]string{"scan"}, args...))
		},
	})
	return cmd
}

func newContractCommand(env clienv.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:                "contract",
		Short:              "validate contract JSON files",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 && (args[0] == "--help" || args[0] == "-h") {
				_, err := fmt.Fprint(cmd.OutOrStdout(), contractcmd.Usage())
				return err
			}
			if len(args) > 0 {
				return exitcode.UsageError(fmt.Errorf("unknown contract command %q", args[0]))
			}
			_, err := fmt.Fprint(cmd.OutOrStdout(), contractcmd.Usage())
			return err
		},
	}
	cmd.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		_, _ = fmt.Fprint(cmd.OutOrStdout(), contractcmd.Usage())
	})
	cmd.SetUsageFunc(func(cmd *cobra.Command) error {
		_, err := fmt.Fprint(cmd.OutOrStdout(), contractcmd.Usage())
		return err
	})
	cmd.AddCommand(&cobra.Command{
		Use:                "validate",
		Short:              "validate a contract JSON file",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return contractcmd.Run(cmd.Context(), outputFor(cmd), env.WorkDir, append([]string{"validate"}, args...))
		},
	})
	return cmd
}

func newProofCommand(env clienv.Env) *cobra.Command {
	cmd := &cobra.Command{
		Use:                "proof",
		Short:              "render proof JSON files",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 && (args[0] == "--help" || args[0] == "-h") {
				_, err := fmt.Fprint(cmd.OutOrStdout(), proofcmd.Usage())
				return err
			}
			if len(args) > 0 {
				return exitcode.UsageError(fmt.Errorf("unknown proof command %q", args[0]))
			}
			_, err := fmt.Fprint(cmd.OutOrStdout(), proofcmd.Usage())
			return err
		},
	}
	cmd.SetHelpFunc(func(cmd *cobra.Command, _ []string) {
		_, _ = fmt.Fprint(cmd.OutOrStdout(), proofcmd.Usage())
	})
	cmd.SetUsageFunc(func(cmd *cobra.Command) error {
		_, err := fmt.Fprint(cmd.OutOrStdout(), proofcmd.Usage())
		return err
	})
	cmd.AddCommand(&cobra.Command{
		Use:                "show",
		Short:              "render a proof JSON file",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return proofcmd.Run(cmd.Context(), outputFor(cmd), env.WorkDir, append([]string{"show"}, args...))
		},
	})
	return cmd
}

func newHelpCommand(root *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:                "help [command]",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				_, err := fmt.Fprint(cmd.OutOrStdout(), RootUsage())
				return err
			}
			target := firstLevelCommand(root, args[0])
			if target == nil {
				return exitcode.UsageError(fmt.Errorf("unknown command %q", args[0]))
			}
			_, err := fmt.Fprint(cmd.OutOrStdout(), helpFor(target))
			return err
		},
	}
}

func firstLevelCommand(root *cobra.Command, name string) *cobra.Command {
	for _, cmd := range root.Commands() {
		if cmd.Name() == name {
			return cmd
		}
	}
	return nil
}

func helpFor(cmd *cobra.Command) string {
	switch cmd.CommandPath() {
	case "goalrail readiness":
		return readinesscmd.Usage()
	case "goalrail readiness scan":
		return readinesscmd.ScanUsage()
	case "goalrail project":
		return projectcmd.Usage()
	case "goalrail project scan":
		return projectcmd.ScanUsage()
	case "goalrail project status":
		return projectcmd.StatusUsage()
	case "goalrail contract":
		return contractcmd.Usage()
	case "goalrail contract validate":
		return contractcmd.ValidateUsage()
	case "goalrail proof":
		return proofcmd.Usage()
	case "goalrail proof show":
		return proofcmd.ShowUsage()
	case "goalrail init":
		return initcmd.Usage()
	case "goalrail work":
		return workcmd.Usage()
	case "goalrail work start":
		return workcmd.StartUsage()
	case "goalrail login":
		return logincmd.Usage()
	case "goalrail version":
		return versionUsage()
	default:
		return RootUsage()
	}
}

func outputFor(cmd *cobra.Command) *term.Output {
	return term.New(cmd.OutOrStdout(), cmd.ErrOrStderr())
}

func versionUsage() string {
	return "Usage: goalrail version\n"
}
