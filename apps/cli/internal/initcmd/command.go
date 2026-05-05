package initcmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/heurema/goalrail/apps/cli/internal/authstore"
	"github.com/heurema/goalrail/apps/cli/internal/exitcode"
	"github.com/heurema/goalrail/apps/cli/internal/gitctx"
	"github.com/heurema/goalrail/apps/cli/internal/spine"
	"github.com/heurema/goalrail/apps/cli/internal/term"
)

const localDemoMessage = "Local demo only. Production repository registration belongs server-side; this CLI does not create a server RepoBinding, connect Git apps, queue audits, or provision deploy keys."

type SessionStore interface {
	Load() (authstore.Session, error)
}

type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

type Options struct {
	Store      SessionStore
	HTTPClient HTTPClient
	Now        func() time.Time
}

func Run(ctx context.Context, out *term.Output, workDir string, args []string) error {
	return RunWithOptions(ctx, out, workDir, args, Options{})
}

func RunWithOptions(ctx context.Context, out *term.Output, workDir string, args []string, options Options) error {
	if err := ctx.Err(); err != nil {
		return exitcode.RuntimeError(err)
	}

	flags := flag.NewFlagSet("goalrail init", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	repoURL := flags.String("repo", "", "repository URL")
	projectID := flags.String("project", "", "server Project ID for authenticated RepoBinding init")
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

	format, err := term.ParseFormat(*formatValue)
	if err != nil {
		return exitcode.UsageError(err)
	}

	discovered, discoverErr := gitctx.Discover(ctx, workDir)
	if discoverErr != nil && !errors.Is(discoverErr, gitctx.ErrNotGitRepository) {
		return exitcode.RuntimeError(discoverErr)
	}

	resolvedRepoURL := strings.TrimSpace(*repoURL)
	if resolvedRepoURL == "" {
		resolvedRepoURL = discovered.RemoteURL
	}
	if resolvedRepoURL == "" {
		return exitcode.UsageError(errors.New("missing --repo and no git remote origin was detected"))
	}

	remoteInfo := gitctx.ParseRemoteURL(resolvedRepoURL)
	draft := spine.RepoBindingDraft{
		RepoURL:            resolvedRepoURL,
		Status:             spine.RepoBindingStatusPendingServerKeyProvisioning,
		Message:            localDemoMessage,
		NextCommand:        nextSuggestedCommand,
		GitRoot:            discovered.GitRoot,
		RemoteName:         discovered.RemoteName,
		Provider:           remoteInfo.Provider,
		ProviderHost:       remoteInfo.ProviderHost,
		RepositoryFullName: remoteInfo.RepositoryFullName,
		WorkflowBaseBranch: discovered.WorkflowBaseBranch,
		HeadSHA:            discovered.HeadSHA,
		Warnings:           discovered.Warnings,
	}
	if draft.Warnings == nil {
		draft.Warnings = []string{}
	}

	if strings.TrimSpace(*projectID) != "" {
		return runServerBackedInit(ctx, out, draft, strings.TrimSpace(*projectID), format, options)
	}

	if format == term.FormatJSON {
		return term.WriteJSON(out.Stdout, draft)
	}

	_, err = fmt.Fprint(out.Stdout, renderText(draft))
	return err
}

func Usage() string {
	return "Usage: goalrail init [--repo <repo-url>] [--project <project-id>] [--format text|json]\n\nWithout --project, creates a local/demo repo binding draft. Without --repo, the command reads local Git metadata and remote.origin.url when run inside a Git worktree.\n\nWith --project, performs authenticated server-backed RepoBinding metadata init using the stored goalrail login profile and writes a non-secret .goalrail/project.yml marker in the Git root. It does not configure audit, create hooks, create branches, provision deploy keys, or start verification.\n"
}

func renderText(draft spine.RepoBindingDraft) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Repo binding draft\n")
	fmt.Fprintf(&b, "Repo: %s\n", draft.RepoURL)
	if draft.Provider != "" {
		fmt.Fprintf(&b, "Provider: %s\n", draft.Provider)
	}
	if draft.ProviderHost != "" {
		fmt.Fprintf(&b, "Provider host: %s\n", draft.ProviderHost)
	}
	if draft.RepositoryFullName != "" {
		fmt.Fprintf(&b, "Repository: %s\n", draft.RepositoryFullName)
	}
	if draft.GitRoot != "" {
		fmt.Fprintf(&b, "Git root: %s\n", draft.GitRoot)
	}
	if draft.RemoteName != "" {
		fmt.Fprintf(&b, "Remote: %s\n", draft.RemoteName)
	}
	if draft.WorkflowBaseBranch != "" {
		fmt.Fprintf(&b, "Workflow base branch: %s\n", draft.WorkflowBaseBranch)
	}
	if draft.HeadSHA != "" {
		fmt.Fprintf(&b, "HEAD: %s\n", draft.HeadSHA)
	}
	fmt.Fprintf(&b, "Status: %s\n", draft.Status)
	if len(draft.Warnings) > 0 {
		b.WriteString("\nWarnings:\n")
		for _, warning := range draft.Warnings {
			fmt.Fprintf(&b, "- %s\n", warning)
		}
	}
	fmt.Fprintf(&b, "\n%s\n\nNext: %s\n", draft.Message, draft.NextCommand)
	return b.String()
}
