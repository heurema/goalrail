package initcmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/heurema/goalrail/apps/cli/internal/exitcode"
	"github.com/heurema/goalrail/apps/cli/internal/spine"
	"github.com/heurema/goalrail/apps/cli/internal/term"
)

func Run(ctx context.Context, out *term.Output, args []string) error {
	if err := ctx.Err(); err != nil {
		return exitcode.RuntimeError(err)
	}

	flags := flag.NewFlagSet("goalrail init", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	repoURL := flags.String("repo", "", "repository URL")
	formatValue := flags.String("format", string(term.FormatText), "output format: text or json")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, writeErr := fmt.Fprint(out.Stdout, Usage())
			return writeErr
		}
		return exitcode.UsageError(err)
	}

	if flags.NArg() != 0 {
		return exitcode.UsageError(fmt.Errorf("unexpected arguments: %v", flags.Args()))
	}
	if *repoURL == "" {
		return exitcode.UsageError(errors.New("missing required --repo <repo-url>"))
	}

	format, err := term.ParseFormat(*formatValue)
	if err != nil {
		return exitcode.UsageError(err)
	}

	draft := spine.RepoBindingDraft{
		RepoURL:     *repoURL,
		Status:      spine.RepoBindingStatusPendingServerKeyProvisioning,
		Message:     "Local demo only. Production deploy key provisioning belongs server-side; this CLI does not generate or store private SSH keys.",
		NextCommand: "goalrail readiness scan --path .",
	}

	if format == term.FormatJSON {
		return term.WriteJSON(out.Stdout, draft)
	}

	_, err = fmt.Fprintf(out.Stdout, "Repo binding draft\nRepo: %s\nStatus: %s\n\n%s\n\nNext: %s\n", draft.RepoURL, draft.Status, draft.Message, draft.NextCommand)
	return err
}

func Usage() string {
	return "Usage: goalrail init --repo <repo-url> [--format text|json]\n\nCreates a local/demo repo binding draft. Production deploy key provisioning belongs server-side.\n"
}
