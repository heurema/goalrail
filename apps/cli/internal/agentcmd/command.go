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
	rootAgentShimRelativePath = "AGENTS.md"
	statusSkippedManualPatch  = "skipped_manual_patch_needed"
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
		rootAgentShimRelativePath: rootAgentShimContent(),
	}
	neutralPaths := []string{agentGuideRelativePath, agentCommandsRelativePath}
	paths := []string{agentGuideRelativePath, agentCommandsRelativePath, rootAgentShimRelativePath}
	for _, relativePath := range neutralPaths {
		if err := preflightAgentFile(discovered.GitRoot, relativePath, files[relativePath], *force); err != nil {
			return err
		}
	}
	statuses := map[string]string{}
	for _, relativePath := range neutralPaths {
		status, err := writeAgentFile(discovered.GitRoot, relativePath, files[relativePath], *force)
		if err != nil {
			return err
		}
		statuses[relativePath] = status
	}
	status, err := writeRootAgentShim(discovered.GitRoot, files[rootAgentShimRelativePath])
	if err != nil {
		return err
	}
	statuses[rootAgentShimRelativePath] = status

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

func writeRootAgentShim(gitRoot, content string) (string, error) {
	path := filepath.Join(gitRoot, rootAgentShimRelativePath)
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return "", exitcode.RuntimeError(fmt.Errorf("write %s: %w", rootAgentShimRelativePath, err))
		}
		return projectconfig.StatusWritten, nil
	}
	if err != nil {
		return "", exitcode.RuntimeError(fmt.Errorf("inspect %s: %w", rootAgentShimRelativePath, err))
	}
	if info.Mode()&os.ModeSymlink != 0 || info.IsDir() {
		return statusSkippedManualPatch, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", exitcode.RuntimeError(fmt.Errorf("read %s: %w", rootAgentShimRelativePath, err))
	}
	if bytes.Equal(raw, []byte(content)) {
		return projectconfig.StatusUnchanged, nil
	}
	return statusSkippedManualPatch, nil
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

If a Goalrail JSON response contains ` + "`next_action.available=false`" + `, do not call ` + "`next_action.command`" + `. Treat it as a planned future command and explain the current available next step to the user.

If a Goalrail JSON response contains ` + "`next_action.available=true`" + ` and a ` + "`next_action.command`" + `, you may call that command to continue the current Goalrail flow.

If a Goalrail JSON response contains ` + "`next_action.kind=ask_user`" + `, render the returned questions to the user. Submit answers with ` + "`goalrail work answer`" + ` using question_id-bound structured JSON. Do not submit free-form answers without mapping them to returned ` + "`question_id`" + ` values.

If a Goalrail JSON response contains ` + "`next_action.kind=draft_contract`" + ` and ` + "`next_action.available=true`" + `, call ` + "`goalrail contract draft`" + ` with the returned Goal ID. The command returns a server Contract handle and a local repository receipt. Do not upload raw source or draft contract fields outside returned Goalrail commands.

If a Goalrail JSON response contains ` + "`next_action.kind=update_contract`" + ` and ` + "`next_action.available=true`" + `, read only the local files needed for the draft and submit structured proposed fields with ` + "`goalrail contract update`" + `. Use ` + "`question_id`" + `- and field-bound JSON, include local receipt refs when useful, and do not upload raw source bodies.

If a Goalrail JSON response contains ` + "`next_action.kind=review_contract`" + `, show the changed draft contract fields to the user for review. Do not submit, approve, plan, run, verify, or create proof unless Goalrail returns an available command for that later state.

If the user explicitly accepts the reviewed draft and Goalrail exposes ` + "`goalrail contract submit`" + ` as an available command, submit the Contract for approval. This is not approval.

Only call ` + "`goalrail contract approve --confirm-user-approval`" + ` after the user explicitly approves the submitted Contract. Never infer approval from silence or from a generic continuation request.

If a Goalrail JSON response contains ` + "`next_action.kind=plan_work`" + ` and ` + "`next_action.available=true`" + `, call ` + "`goalrail work plan`" + ` with the returned Contract ID. This only creates or returns a server WorkItemPlan; newly created plans start queued. It does not acquire a lease, produce a proposal, create WorkItems, run code, verify, or create proof.

If a Goalrail JSON response contains ` + "`next_action.kind=review_plan_proposal`" + ` and ` + "`next_action.available=true`" + `, call ` + "`goalrail work plan status`" + ` with the returned Plan ID. Show the proposed tasks to the user.

Only call ` + "`goalrail work proposal accept --confirm-user-acceptance`" + ` after the user explicitly accepts the submitted WorkItemPlanProposal. Never infer plan acceptance from silence or from a generic continuation request.

If a Goalrail JSON response contains ` + "`next_action.kind=prepare_checkout`" + ` and ` + "`next_action.available=true`" + `, call ` + "`goalrail work checkout prepare`" + ` with the returned WorkItem ID. This creates or returns a server-owned checkout job and checkout instruction only. It does not assign, claim, execute commands, create Run, verify, gate, or create proof.

If a Goalrail JSON response contains ` + "`next_action.kind=runner_checkout_required`" + `, explain that a runner process must submit a workspace receipt before execution can be designed. Do not perform checkout by chat, do not run arbitrary commands as proof, and do not claim execution.

If a Goalrail JSON response or runner handoff includes a ` + "`task_id`" + ` and ` + "`checkout_receipt_id`" + `, and the user asks to prepare execution, call ` + "`goalrail work execution prepare`" + `. This creates or returns a server-owned ExecutionJob only. It does not create Run, lease execution, execute commands, create execution receipt, gate, or proof.

If a Goalrail JSON response contains ` + "`next_action.kind=runner_execution_required`" + `, explain that runner execution start is a future slice. Do not run commands by chat and do not claim execution.

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
    },
    "continue_work": {
      "command": "goalrail work continue --goal-id <goal_id> --format json",
      "requires": [
        "goalrail_login",
        "goalrail_init",
        "git_worktree",
        "goal_id"
      ],
      "does_not_create": [
        "Contract",
        "WorkItem",
        "Run",
        "Decision",
        "Proof"
      ]
    },
    "answer_clarification": {
      "command": "goalrail work answer --clarification-request-id <id> --answers-file - --format json",
      "stdin": "structured_answers_json",
      "requires": [
        "goalrail_login",
        "goalrail_init",
        "git_worktree",
        "clarification_request_id"
      ],
      "creates": [
        "ClarificationAnswer"
      ],
      "does_not_create": [
        "Contract",
        "WorkItem",
        "Run",
        "Decision",
        "Proof"
      ]
    },
    "draft_contract": {
      "command": "goalrail contract draft --goal-id <goal_id> --format json",
      "requires": [
        "goalrail_login",
        "goalrail_init",
        "git_worktree",
        "goal_id"
      ],
      "creates": [
        "Contract",
        "ContractSeed",
        "ContractDraft"
      ],
      "returns": [
        "local_repo_receipt"
      ],
      "does_not_create": [
        "WorkItem",
        "Run",
        "Decision",
        "Proof"
      ]
    },
    "update_contract": {
      "command": "goalrail contract update --contract-id <contract_id> --fields-file - --format json",
      "stdin": "structured_contract_fields_json",
      "requires": [
        "goalrail_login",
        "goalrail_init",
        "git_worktree",
        "contract_id"
      ],
      "updates": [
        "ContractDraft"
      ],
      "does_not_create": [
        "WorkItem",
        "Run",
        "Decision",
        "Proof"
      ],
      "does_not_upload": [
        "raw_source"
      ]
    },
    "submit_contract": {
      "command": "goalrail contract submit --contract-id <contract_id> --format json",
      "requires": [
        "goalrail_login",
        "goalrail_init",
        "git_worktree",
        "contract_id",
        "reviewed_contract_draft"
      ],
      "updates": [
        "ContractDraft"
      ],
      "does_not_create": [
        "ApprovedContract",
        "WorkItem",
        "Run",
        "Decision",
        "Proof"
      ]
    },
    "approve_contract": {
      "command": "goalrail contract approve --contract-id <contract_id> --confirm-user-approval --format json",
      "requires": [
        "goalrail_login",
        "goalrail_init",
        "git_worktree",
        "contract_id",
        "explicit_user_approval"
      ],
      "creates": [
        "ApprovedContract"
      ],
      "does_not_create": [
        "WorkItem",
        "Run",
        "Decision",
        "Proof"
      ]
    },
    "plan_work": {
      "command": "goalrail work plan --contract-id <contract_id> --format json",
      "requires": [
        "goalrail_login",
        "goalrail_init",
        "git_worktree",
        "contract_id",
        "approved_contract"
      ],
      "creates": [
        "WorkItemPlan"
      ],
      "does_not_create": [
        "WorkItemPlanLease",
        "WorkItemPlanProposal",
        "WorkItem",
        "Run",
        "Decision",
        "Proof"
      ]
    },
    "plan_status": {
      "command": "goalrail work plan status --plan-id <plan_id> --format json",
      "requires": [
        "goalrail_login",
        "goalrail_init",
        "git_worktree",
        "plan_id"
      ],
      "returns": [
        "WorkItemPlan",
        "WorkItemPlanProposal"
      ],
      "does_not_create": [
        "WorkItem",
        "Run",
        "Decision",
        "Proof"
      ]
    },
    "accept_plan_proposal": {
      "command": "goalrail work proposal accept --proposal-id <proposal_id> --confirm-user-acceptance --format json",
      "requires": [
        "goalrail_login",
        "goalrail_init",
        "git_worktree",
        "proposal_id",
        "explicit_user_acceptance"
      ],
      "creates": [
        "WorkItem"
      ],
      "does_not_create": [
        "Run",
        "Decision",
        "Proof"
      ]
    },
    "prepare_checkout": {
      "command": "goalrail work checkout prepare --task-id <task_id> --format json",
      "requires": [
        "goalrail_login",
        "goalrail_init",
        "git_worktree",
        "task_id",
        "planned_workitem"
      ],
      "creates": [
        "CheckoutJob",
        "CheckoutInstruction"
      ],
      "does_not_create": [
        "Assignment",
        "Claim",
        "Run",
        "Decision",
        "Proof"
      ],
      "does_not_execute": [
        "commands",
        "tests",
        "checkout"
      ]
    },
    "prepare_execution": {
      "command": "goalrail work execution prepare --task-id <task_id> --checkout-receipt-id <checkout_receipt_id> --format json",
      "requires": [
        "goalrail_login",
        "goalrail_init",
        "git_worktree",
        "task_id",
        "checkout_receipt_id",
        "planned_workitem"
      ],
      "creates": [
        "ExecutionJob"
      ],
      "does_not_create": [
        "Run",
        "ExecutionReceipt",
        "Decision",
        "GateDecision",
        "Proof"
      ],
      "does_not_execute": [
        "commands",
        "tests",
        "provider_runtime"
      ]
    }
  },
  "agent_rules": {
    "do_not_call_unavailable_next_actions": true
  }
}
`
}

func rootAgentShimContent() string {
	return `# Goalrail Agent Shim

This repository has a Goalrail Agent Pack.

For Goalrail work, read:

.goalrail/agent/GOALRAIL.md

Use Goalrail CLI with --format json. Do not invent Goalrail state.
`
}
