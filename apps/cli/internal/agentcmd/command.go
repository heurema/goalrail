package agentcmd

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/heurema/goalrail/apps/cli/internal/exitcode"
	"github.com/heurema/goalrail/apps/cli/internal/gitctx"
	"github.com/heurema/goalrail/apps/cli/internal/projectconfig"
	"github.com/heurema/goalrail/apps/cli/internal/term"
)

const (
	agentDirRelativePath      = ".goalrail/agent"
	agentGuideRelativePath    = ".goalrail/agent/GOALRAIL.md"
	agentCommandsRelativePath = ".goalrail/agent/commands.json"
)

type InstallOutput struct {
	Status string            `json:"status"`
	Paths  []string          `json:"paths"`
	Files  map[string]string `json:"files"`
}

func Run(ctx context.Context, out *term.Output, workDir string, args []string) error {
	if len(args) == 0 {
		_, err := fmt.Fprint(out.Stdout, Usage())
		return err
	}

	switch args[0] {
	case "--help", "-h":
		_, err := fmt.Fprint(out.Stdout, Usage())
		return err
	case "install":
		return runInstall(ctx, out, workDir, args[1:])
	default:
		return exitcode.UsageError(fmt.Errorf("unknown agent command %q", args[0]))
	}
}

func Usage() string {
	return "Usage: goalrail agent <command> [options]\n\nCommands:\n  install    install provider-neutral repo-local Goalrail Agent Pack files\n\nRun goalrail agent <command> --help for command usage.\n"
}

func InstallUsage() string {
	return "Usage: goalrail agent install [--format text|json] [--force]\n\nInstalls provider-neutral repo-local Goalrail Agent Pack files under .goalrail/agent for local coding agents. This command does not install Codex, Claude, Gemini, Cursor, or other provider-specific files.\n"
}

func runInstall(ctx context.Context, out *term.Output, workDir string, args []string) error {
	if err := ctx.Err(); err != nil {
		return exitcode.RuntimeError(err)
	}

	flags := flag.NewFlagSet("goalrail agent install", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	formatValue := flags.String("format", string(term.FormatText), "output format: text or json")
	force := flags.Bool("force", false, "overwrite existing Agent Pack files")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, writeErr := fmt.Fprint(out.Stdout, InstallUsage())
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

	discovered, err := gitctx.Discover(ctx, workDir)
	if err != nil {
		if errors.Is(err, gitctx.ErrNotGitRepository) {
			return exitcode.UsageError(errors.New("goalrail agent install requires a Git worktree with .goalrail/project.yml; run goalrail init first"))
		}
		return exitcode.RuntimeError(err)
	}
	if _, ok, err := projectconfig.Read(discovered.GitRoot); err != nil {
		return err
	} else if !ok {
		return exitcode.UsageError(errors.New("missing .goalrail/project.yml; run goalrail init first"))
	}

	files := map[string]string{
		agentGuideRelativePath:    agentGuideContent(),
		agentCommandsRelativePath: commandsJSONContent(),
	}
	paths := []string{agentGuideRelativePath, agentCommandsRelativePath}
	for _, relativePath := range paths {
		if err := preflightAgentFile(discovered.GitRoot, relativePath, files[relativePath], *force); err != nil {
			return err
		}
	}
	statuses := map[string]string{}
	for _, relativePath := range paths {
		status, err := writeAgentFile(discovered.GitRoot, relativePath, files[relativePath], *force)
		if err != nil {
			return err
		}
		statuses[relativePath] = status
	}

	output := InstallOutput{
		Status: aggregateStatus(statuses),
		Paths:  paths,
		Files:  statuses,
	}
	if format == term.FormatJSON {
		return term.WriteJSON(out.Stdout, output)
	}
	_, err = fmt.Fprint(out.Stdout, renderInstallText(output))
	return err
}

func preflightAgentFile(gitRoot, relativePath, content string, force bool) error {
	existing, exists, err := readAgentFile(gitRoot, relativePath)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	if bytes.Equal(existing, []byte(content)) || force {
		return nil
	}
	return exitcode.ValidationError(fmt.Errorf("%s already exists with different content; re-run with --force to overwrite", relativePath))
}

func writeAgentFile(gitRoot, relativePath, content string, force bool) (string, error) {
	path := filepath.Join(gitRoot, relativePath)
	desired := []byte(content)
	existing, exists, err := readAgentFile(gitRoot, relativePath)
	if err != nil {
		return "", err
	}
	if exists {
		if bytes.Equal(existing, desired) {
			return projectconfig.StatusUnchanged, nil
		}
		if !force {
			return "", exitcode.ValidationError(fmt.Errorf("%s already exists with different content; re-run with --force to overwrite", relativePath))
		}
		if err := os.WriteFile(path, desired, 0o644); err != nil {
			return "", exitcode.RuntimeError(fmt.Errorf("write %s: %w", relativePath, err))
		}
		return projectconfig.StatusUpdated, nil
	}
	if err := ensureAgentDirectory(gitRoot); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, desired, 0o644); err != nil {
		return "", exitcode.RuntimeError(fmt.Errorf("write %s: %w", relativePath, err))
	}
	return projectconfig.StatusWritten, nil
}

func readAgentFile(gitRoot, relativePath string) ([]byte, bool, error) {
	if err := validateAgentPath(gitRoot, relativePath); err != nil {
		return nil, false, err
	}
	path := filepath.Join(gitRoot, relativePath)
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, exitcode.RuntimeError(fmt.Errorf("inspect %s: %w", relativePath, err))
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, false, exitcode.ValidationError(fmt.Errorf("%s must not be a symlink", relativePath))
	}
	if info.IsDir() {
		return nil, false, exitcode.ValidationError(fmt.Errorf("%s must be a file", relativePath))
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, false, exitcode.RuntimeError(fmt.Errorf("read %s: %w", relativePath, err))
	}
	return raw, true, nil
}

func validateAgentPath(gitRoot, relativePath string) error {
	for _, dirRelativePath := range []string{".goalrail", agentDirRelativePath} {
		info, err := os.Lstat(filepath.Join(gitRoot, dirRelativePath))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return exitcode.RuntimeError(fmt.Errorf("inspect %s: %w", dirRelativePath, err))
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return exitcode.ValidationError(fmt.Errorf("%s must not be a symlink", dirRelativePath))
		}
		if !info.IsDir() {
			return exitcode.ValidationError(fmt.Errorf("%s must be a directory", dirRelativePath))
		}
	}
	return nil
}

func ensureAgentDirectory(gitRoot string) error {
	if err := validateAgentPath(gitRoot, agentGuideRelativePath); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(gitRoot, agentDirRelativePath), 0o755); err != nil {
		return exitcode.RuntimeError(fmt.Errorf("create %s: %w", agentDirRelativePath, err))
	}
	return nil
}

func aggregateStatus(statuses map[string]string) string {
	hasWritten := false
	hasUpdated := false
	for _, status := range statuses {
		switch status {
		case projectconfig.StatusUpdated:
			hasUpdated = true
		case projectconfig.StatusWritten:
			hasWritten = true
		}
	}
	switch {
	case hasUpdated:
		return "updated"
	case hasWritten:
		return "installed"
	default:
		return projectconfig.StatusUnchanged
	}
}

func renderInstallText(output InstallOutput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Agent Pack %s\n\n", output.Status)
	for _, path := range output.Paths {
		fmt.Fprintf(&b, "%s: %s\n", path, output.Files[path])
	}
	fmt.Fprintf(&b, "\nThis installed only provider-neutral repo-local Goalrail files.\n")
	return b.String()
}

func agentGuideContent() string {
	return `# Goalrail Agent Pack v0

You are in a Goalrail-initialized repository.

Use the Goalrail CLI as the machine interface. Prefer ` + "`--format json`" + ` for commands that support it.

Never invent Goalrail state. The Goalrail server owns canonical Intake, Goal, readiness, clarification, Contract, event, gate, proof, and verification state.

For user requests like "start Goalrail work" or pasted Jira/Linear task text, call:

` + "```bash" + `
goalrail work start --title <title> --body-file - --format json
` + "```" + `

Pass pasted task text through stdin as the work body.

After the command returns, show a concise human summary with:

- ` + "`intake_id`" + `
- ` + "`goal_id`" + `
- ` + "`goal_state`" + `

Do not create branches, run agents, run tests, create proof, or claim verification unless Goalrail returns those states.

This Agent Pack is provider-neutral. It is not a Codex, Claude, Gemini, Cursor, Windsurf, or Gravity adapter, plugin, skill, slash command, or provider setting.
`
}

func commandsJSONContent() string {
	return `{
  "version": "goalrail.agent.v0",
  "source": "goalrail",
  "commands": {
    "start_work": {
      "command": "goalrail work start --title <title> --body-file - --format json",
      "stdin": "task_body",
      "requires": [
        "goalrail_login",
        "goalrail_init",
        "git_worktree"
      ],
      "creates": [
        "IntakeRecord",
        "Goal"
      ],
      "does_not_create": [
        "Contract",
        "WorkItem",
        "Run",
        "Decision",
        "Proof"
      ]
    }
  }
}
`
}
