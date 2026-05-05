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
	baseBranch := flags.String("base", "", "workflow base branch for init")
	localDemo := flags.Bool("local-demo", false, "create a local/demo repo binding draft without auth or server calls")
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
	if *localDemo && strings.TrimSpace(*projectID) != "" {
		return exitcode.UsageError(errors.New("--local-demo cannot be combined with --project"))
	}
	baseBranchProvided := false
	flags.Visit(func(f *flag.Flag) {
		if f.Name == "base" {
			baseBranchProvided = true
		}
	})

	format, err := term.ParseFormat(*formatValue)
	if err != nil {
		return exitcode.UsageError(err)
	}

	discovered, discoverErr := gitctx.Discover(ctx, workDir)
	if discoverErr != nil && !errors.Is(discoverErr, gitctx.ErrNotGitRepository) {
		return exitcode.RuntimeError(discoverErr)
	}
	providerDefaultBranch := discovered.WorkflowBaseBranch
	workflowBaseBranch := discovered.WorkflowBaseBranch
	warnings := discovered.Warnings
	if baseBranchProvided {
		normalizedBaseBranch, err := normalizeBaseBranch(*baseBranch)
		if err != nil {
			return exitcode.UsageError(err)
		}
		workflowBaseBranch = normalizedBaseBranch
		warnings = warningsForBaseOverride(discovered.Warnings)
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
		RepoURL:               resolvedRepoURL,
		Status:                spine.RepoBindingStatusPendingServerKeyProvisioning,
		Message:               localDemoMessage,
		NextCommand:           nextSuggestedCommand,
		GitRoot:               discovered.GitRoot,
		RemoteName:            discovered.RemoteName,
		Provider:              remoteInfo.Provider,
		ProviderHost:          remoteInfo.ProviderHost,
		RepositoryFullName:    remoteInfo.RepositoryFullName,
		ProviderDefaultBranch: providerDefaultBranch,
		WorkflowBaseBranch:    workflowBaseBranch,
		HeadSHA:               discovered.HeadSHA,
		Warnings:              warnings,
	}
	if draft.Warnings == nil {
		draft.Warnings = []string{}
	}

	if strings.TrimSpace(*projectID) != "" {
		return runServerBackedInit(ctx, out, draft, strings.TrimSpace(*projectID), format, options)
	}

	if !*localDemo {
		return runRepositoryContextInit(ctx, out, draft, format, options)
	}

	if format == term.FormatJSON {
		return term.WriteJSON(out.Stdout, draft)
	}

	_, err = fmt.Fprint(out.Stdout, renderText(draft))
	return err
}

func Usage() string {
	return "Usage: goalrail init [--repo <repo-url>] [--base <branch>] [--project <project-id>] [--local-demo] [--format text|json]\n\nBy default, initializes server-backed repository context using the stored goalrail login profile and writes a non-secret .goalrail/project.yml marker in the Git root. Without --repo, the command reads local Git metadata and remote.origin.url when run inside a Git worktree.\n\nWith --base, sets workflow_base_branch explicitly without creating branches or changing Git state. When local origin default metadata is available, it remains provider_default_branch.\n\nWith --project, uses the low-level Project-scoped RepoBinding init endpoint.\n\nWith --local-demo, creates the old auth-free local/demo repo binding draft and writes no files.\n\nInit does not configure audit, create hooks, create branches, provision deploy keys, connect provider integrations, or start verification.\n"
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
	if draft.ProviderDefaultBranch != "" {
		fmt.Fprintf(&b, "Provider default branch: %s\n", draft.ProviderDefaultBranch)
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

func normalizeBaseBranch(value string) (string, error) {
	branch := strings.TrimSpace(value)
	if branch == "" {
		return "", errors.New("--base requires a branch name")
	}
	if strings.ContainsAny(branch, " \t\r\n") {
		return "", errors.New("--base branch name must not contain whitespace")
	}
	return branch, nil
}

func warningsForBaseOverride(warnings []string) []string {
	if len(warnings) == 0 {
		return warnings
	}
	out := make([]string, 0, len(warnings))
	for _, warning := range warnings {
		if warning == "workflow base branch could not be detected from local origin metadata" {
			out = append(out, "provider default branch could not be detected from local origin metadata")
			continue
		}
		out = append(out, warning)
	}
	return out
}
