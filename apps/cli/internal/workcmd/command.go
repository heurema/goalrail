package workcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/heurema/goalrail/apps/cli/internal/authsession"
	"github.com/heurema/goalrail/apps/cli/internal/authstore"
	"github.com/heurema/goalrail/apps/cli/internal/exitcode"
	"github.com/heurema/goalrail/apps/cli/internal/gitctx"
	"github.com/heurema/goalrail/apps/cli/internal/projectconfig"
	"github.com/heurema/goalrail/apps/cli/internal/spine"
	"github.com/heurema/goalrail/apps/cli/internal/term"
)

const (
	serverMode         = "server"
	cliSchemaVersion   = "goalrail.cli.v1"
	workStartedMessage = "Work intake started."
	workStartedNote    = "This created an IntakeRecord and promoted it to a Goal on the GoalRail server.\nNo audit, hooks, branch creation, deploy keys, provider integration, runner, gate, proof, or verification were configured."
	maxWorkBodyBytes   = 1 << 20
)

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
	Stdin      io.Reader
}

type workContext struct {
	Config    projectconfig.Config
	Session   authstore.Session
	ServerURL string
	Client    HTTPClient
}

func Run(ctx context.Context, out *term.Output, workDir string, args []string) error {
	return RunWithOptions(ctx, out, workDir, args, Options{})
}

func RunWithOptions(ctx context.Context, out *term.Output, workDir string, args []string, options Options) error {
	if len(args) == 0 {
		_, err := fmt.Fprint(out.Stdout, Usage())
		return err
	}

	switch args[0] {
	case "--help", "-h":
		_, err := fmt.Fprint(out.Stdout, Usage())
		return err
	case "start":
		return runStart(ctx, out, workDir, args[1:], options)
	case "continue":
		return runContinue(ctx, out, workDir, args[1:], options)
	case "answer":
		return runAnswer(ctx, out, workDir, args[1:], options)
	case "plan":
		return runPlan(ctx, out, workDir, args[1:], options)
	case "proposal":
		return runProposal(ctx, out, workDir, args[1:], options)
	case "item":
		return runItem(ctx, out, workDir, args[1:], options)
	case "checkout":
		return runCheckout(ctx, out, workDir, args[1:], options)
	case "execution":
		return runExecution(ctx, out, workDir, args[1:], options)
	default:
		return exitcode.UsageError(fmt.Errorf("unknown work command %q", args[0]))
	}
}

func Usage() string {
	return "Usage: goalrail work <command> [options]\n\nCommands:\n  start      create a server-backed IntakeRecord and Goal from the local project marker\n  continue   reconcile Goal readiness and return the next action\n  answer     submit clarification answers and return the next action\n  plan       create, return, or inspect a server WorkItemPlan\n  proposal   review bridge commands for WorkItemPlanProposal state\n  item       inspect planned WorkItems without starting checkout or execution\n  checkout   prepare runner checkout instructions for planned WorkItems\n  execution  prepare execution jobs from checkout receipts\n\nRun goalrail work <command> --help for command usage.\n"
}

func StartUsage() string {
	return "Usage: goalrail work start --title <title> [--body <body> | --body-file <path|->] [--format text|json]\n\nCreates an IntakeRecord and promotes it to a Goal using the current Git root .goalrail/project.yml marker and the stored goalrail login profile.\n\nUse --body-file - to read the work body from stdin.\n\nThis command does not configure audit, create hooks, create branches, provision deploy keys, connect provider integrations, run workers, gates, proof, or verification.\n"
}

func ContinueUsage() string {
	return "Usage: goalrail work continue --goal-id <goal_id> [--format text|json]\n\nReconciles Goal readiness through the Goalrail server using the current Git root .goalrail/project.yml marker and the stored goalrail login profile.\n\nThis command does not answer clarifications, draft contracts, generate context packs, run workers, gates, proof, or verification.\n"
}

func AnswerUsage() string {
	return "Usage: goalrail work answer --clarification-request-id <id> --answers-file <path|-> [--format text|json]\n\nSubmits structured answers for an open ClarificationRequest through the Goalrail server using the current Git root .goalrail/project.yml marker and the stored goalrail login profile.\n\nUse --answers-file - to read answer JSON from stdin.\n\nThis command does not draft contracts, generate context packs, run workers, gates, proof, or verification.\n"
}

func PlanUsage() string {
	return "Usage: goalrail work plan --contract-id <contract_id> [--format text|json]\n       goalrail work plan status --plan-id <plan_id> [--format text|json]\n\nCreates or returns a server WorkItemPlan for an approved Contract, or inspects an existing plan for submitted proposals, using the current Git root .goalrail/project.yml marker and the stored goalrail login profile.\n\nThis command does not acquire planning leases, submit proposals, create WorkItems, run workers, gates, proof, or verification. Proposal acceptance requires goalrail work proposal accept.\n"
}

func PlanStatusUsage() string {
	return "Usage: goalrail work plan status --plan-id <plan_id> [--format text|json]\n\nReads authenticated WorkItemPlan status and submitted proposal details using the current Git root .goalrail/project.yml marker and the stored goalrail login profile.\n\nThis command does not accept proposals, create WorkItems, run workers, gates, proof, or verification.\n"
}

func ProposalUsage() string {
	return "Usage: goalrail work proposal <command> [options]\n\nCommands:\n  accept   explicitly accept a submitted WorkItemPlanProposal\n\nRun goalrail work proposal <command> --help for command usage.\n"
}

func ProposalAcceptUsage() string {
	return "Usage: goalrail work proposal accept --proposal-id <proposal_id> --confirm-user-acceptance [--format text|json]\n\nAccepts a submitted WorkItemPlanProposal after explicit user acceptance and materializes server-owned WorkItem(planned) records.\n\nThis command does not assign, claim, run, checkout, gate, proof, or verify work.\n"
}

func ItemUsage() string {
	return "Usage: goalrail work item <command> [options]\n\nCommands:\n  show   inspect a materialized WorkItem without starting checkout or execution\n\nRun goalrail work item <command> --help for command usage.\n"
}

func ItemShowUsage() string {
	return "Usage: goalrail work item show --task-id <task_id> [--format text|json]\n\nReads a materialized WorkItem using the current Git root .goalrail/project.yml marker and the stored goalrail login profile.\n\nThis command is read-only. It does not start checkout, prepare execution, create runs, gate, proof, verify, or complete work.\n"
}

func CheckoutUsage() string {
	return "Usage: goalrail work checkout <command> [options]\n\nCommands:\n  prepare   create or return a runner checkout job for a planned WorkItem\n\nRun goalrail work checkout <command> --help for command usage.\n"
}

func CheckoutPrepareUsage() string {
	return "Usage: goalrail work checkout prepare --task-id <task_id> [--format text|json]\n\nCreates or returns a server-owned checkout job and checkout instruction for a planned WorkItem using the current Git root .goalrail/project.yml marker and the stored goalrail login profile.\n\nThis command does not assign, claim, execute commands, create Run, gate, proof, or verify work.\n"
}

func ExecutionUsage() string {
	return "Usage: goalrail work execution <command> [options]\n\nCommands:\n  prepare   create or return an ExecutionJob from a checkout receipt\n\nRun goalrail work execution <command> --help for command usage.\n"
}

func ExecutionPrepareUsage() string {
	return "Usage: goalrail work execution prepare --task-id <task_id> --checkout-receipt-id <checkout_receipt_id> [--format text|json]\n\nCreates or returns a server-owned ExecutionJob for a planned WorkItem and checkout receipt using the current Git root .goalrail/project.yml marker and the stored goalrail login profile.\n\nThis command does not start a Run, execute commands, create execution receipt, gate, proof, or verify work.\n"
}

func runStart(ctx context.Context, out *term.Output, workDir string, args []string, options Options) error {
	if err := ctx.Err(); err != nil {
		return exitcode.RuntimeError(err)
	}

	flags := flag.NewFlagSet("goalrail work start", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	title := flags.String("title", "", "work title")
	body := flags.String("body", "", "work body")
	bodyFile := flags.String("body-file", "", "path to work body file, or - for stdin")
	formatValue := flags.String("format", string(term.FormatText), "output format: text or json")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, writeErr := fmt.Fprint(out.Stdout, StartUsage())
			return writeErr
		}
		return exitcode.UsageError(err)
	}
	if flags.NArg() != 0 {
		return exitcode.UsageError(fmt.Errorf("unexpected arguments: %v", flags.Args()))
	}
	bodySet := false
	bodyFileSet := false
	flags.Visit(func(flag *flag.Flag) {
		switch flag.Name {
		case "body":
			bodySet = true
		case "body-file":
			bodyFileSet = true
		}
	})
	if bodySet && bodyFileSet {
		return exitcode.UsageError(errors.New("--body and --body-file cannot be used together"))
	}

	normalizedTitle := strings.TrimSpace(*title)
	if normalizedTitle == "" {
		return exitcode.UsageError(errors.New("--title is required"))
	}
	format, err := term.ParseFormat(*formatValue)
	if err != nil {
		return exitcode.UsageError(err)
	}
	discovered, err := gitctx.Discover(ctx, workDir)
	if err != nil {
		if errors.Is(err, gitctx.ErrNotGitRepository) {
			return exitcode.UsageError(errors.New("goalrail work start requires a Git worktree with .goalrail/project.yml; run goalrail init first"))
		}
		return exitcode.RuntimeError(err)
	}
	config, ok, err := projectconfig.Read(discovered.GitRoot)
	if err != nil {
		return err
	}
	if !ok {
		return exitcode.UsageError(errors.New("missing .goalrail/project.yml; run goalrail init first"))
	}

	session, serverURL, client, err := loadUsableSession(ctx, options)
	if err != nil {
		return err
	}
	if strings.TrimRight(config.ServerURL, "/") != serverURL {
		return exitcode.ValidationError(errors.New("local .goalrail/project.yml is bound to a different GoalRail server; run goalrail login for that server or re-initialize this repository"))
	}

	profile, err := getCurrentProfile(ctx, client, session)
	if err != nil {
		return err
	}
	if err := validateMarkerMembership(config, profile); err != nil {
		return err
	}
	normalizedBody, err := resolveBody(*body, *bodyFile, bodyFileSet, options.Stdin)
	if err != nil {
		return err
	}
	intake, err := postIntake(ctx, client, session, intakeSubmission{
		ProjectID:     config.ProjectID,
		RepoBindingID: config.RepoBindingID,
		Source: intakeSource{
			Kind:       "goalrail_cli",
			ExternalID: "work start",
		},
		Title: strings.TrimSpace(*title),
		Body:  normalizedBody,
		RequestAuthor: actorRef{
			Kind:        "user",
			ID:          profile.User.ID,
			DisplayName: profile.User.DisplayName,
		},
	})
	if err != nil {
		return err
	}
	goal, err := promoteIntake(ctx, client, session, intake.IntakeID)
	if err != nil {
		return err
	}

	output := spine.WorkStartOutput{
		SchemaVersion:   cliSchemaVersion,
		Mode:            serverMode,
		ServerURL:       serverURL,
		OrganizationID:  intake.OrganizationID,
		ProjectID:       intake.ProjectID,
		RepoBindingID:   intake.RepoBindingID,
		IntakeID:        intake.IntakeID,
		IntakeState:     intake.State,
		GoalID:          goal.ID,
		GoalState:       goal.State,
		Title:           normalizedTitle,
		LocalConfigPath: projectconfig.RelativePath,
		Display: spine.DisplaySummary{
			Summary: "Created an IntakeRecord and promoted it to a Goal. Continue the Goal to reconcile readiness and get the next action.",
		},
		NextAction: spine.NextAction{
			Kind:      "continue_goal",
			Blocking:  false,
			Command:   fmt.Sprintf("goalrail work continue --goal-id %s --format json", goal.ID),
			Available: true,
		},
		Message:              workStartedMessage,
		NextSuggestedCommand: fmt.Sprintf("goalrail work continue --goal-id %s --format json", goal.ID),
	}
	if output.OrganizationID == "" {
		output.OrganizationID = goal.OrganizationID
	}
	if output.ProjectID == "" {
		output.ProjectID = config.ProjectID
	}
	if output.RepoBindingID == "" {
		output.RepoBindingID = config.RepoBindingID
	}

	if format == term.FormatJSON {
		return term.WriteJSON(out.Stdout, output)
	}
	_, err = fmt.Fprint(out.Stdout, renderStartText(output))
	return err
}

func runContinue(ctx context.Context, out *term.Output, workDir string, args []string, options Options) error {
	if err := ctx.Err(); err != nil {
		return exitcode.RuntimeError(err)
	}

	flags := flag.NewFlagSet("goalrail work continue", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	goalID := flags.String("goal-id", "", "Goal ID")
	formatValue := flags.String("format", string(term.FormatText), "output format: text or json")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, writeErr := fmt.Fprint(out.Stdout, ContinueUsage())
			return writeErr
		}
		return exitcode.UsageError(err)
	}
	if flags.NArg() != 0 {
		return exitcode.UsageError(fmt.Errorf("unexpected arguments: %v", flags.Args()))
	}
	normalizedGoalID := strings.TrimSpace(*goalID)
	if normalizedGoalID == "" {
		return exitcode.UsageError(errors.New("--goal-id is required"))
	}
	format, err := term.ParseFormat(*formatValue)
	if err != nil {
		return exitcode.UsageError(err)
	}

	discovered, err := gitctx.Discover(ctx, workDir)
	if err != nil {
		if errors.Is(err, gitctx.ErrNotGitRepository) {
			return exitcode.UsageError(errors.New("goalrail work continue requires a Git worktree with .goalrail/project.yml; run goalrail init first"))
		}
		return exitcode.RuntimeError(err)
	}
	config, ok, err := projectconfig.Read(discovered.GitRoot)
	if err != nil {
		return err
	}
	if !ok {
		return exitcode.UsageError(errors.New("missing .goalrail/project.yml; run goalrail init first"))
	}

	session, serverURL, client, err := loadUsableSession(ctx, options)
	if err != nil {
		return err
	}
	if strings.TrimRight(config.ServerURL, "/") != serverURL {
		return exitcode.ValidationError(errors.New("local .goalrail/project.yml is bound to a different GoalRail server; run goalrail login for that server or re-initialize this repository"))
	}

	profile, err := getCurrentProfile(ctx, client, session)
	if err != nil {
		return err
	}
	if err := validateMarkerMembership(config, profile); err != nil {
		return err
	}

	continuation, err := postGoalContinuation(ctx, client, session, normalizedGoalID)
	if err != nil {
		return err
	}
	output, err := buildContinueOutput(config, serverURL, continuation)
	if err != nil {
		return err
	}
	if format == term.FormatJSON {
		return term.WriteJSON(out.Stdout, output)
	}
	_, err = fmt.Fprint(out.Stdout, renderContinueText(output))
	return err
}

func runAnswer(ctx context.Context, out *term.Output, workDir string, args []string, options Options) error {
	if err := ctx.Err(); err != nil {
		return exitcode.RuntimeError(err)
	}

	flags := flag.NewFlagSet("goalrail work answer", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	requestID := flags.String("clarification-request-id", "", "ClarificationRequest ID")
	answersFile := flags.String("answers-file", "", "path to answer JSON file, or - for stdin")
	formatValue := flags.String("format", string(term.FormatText), "output format: text or json")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, writeErr := fmt.Fprint(out.Stdout, AnswerUsage())
			return writeErr
		}
		return exitcode.UsageError(err)
	}
	if flags.NArg() != 0 {
		return exitcode.UsageError(fmt.Errorf("unexpected arguments: %v", flags.Args()))
	}
	normalizedRequestID := strings.TrimSpace(*requestID)
	if normalizedRequestID == "" {
		return exitcode.UsageError(errors.New("--clarification-request-id is required"))
	}
	if strings.TrimSpace(*answersFile) == "" {
		return exitcode.UsageError(errors.New("--answers-file is required"))
	}
	format, err := term.ParseFormat(*formatValue)
	if err != nil {
		return exitcode.UsageError(err)
	}

	discovered, err := gitctx.Discover(ctx, workDir)
	if err != nil {
		if errors.Is(err, gitctx.ErrNotGitRepository) {
			return exitcode.UsageError(errors.New("goalrail work answer requires a Git worktree with .goalrail/project.yml; run goalrail init first"))
		}
		return exitcode.RuntimeError(err)
	}
	config, ok, err := projectconfig.Read(discovered.GitRoot)
	if err != nil {
		return err
	}
	if !ok {
		return exitcode.UsageError(errors.New("missing .goalrail/project.yml; run goalrail init first"))
	}

	session, serverURL, client, err := loadUsableSession(ctx, options)
	if err != nil {
		return err
	}
	if strings.TrimRight(config.ServerURL, "/") != serverURL {
		return exitcode.ValidationError(errors.New("local .goalrail/project.yml is bound to a different GoalRail server; run goalrail login for that server or re-initialize this repository"))
	}

	profile, err := getCurrentProfile(ctx, client, session)
	if err != nil {
		return err
	}
	if err := validateMarkerMembership(config, profile); err != nil {
		return err
	}

	submission, err := resolveAnswerSubmission(*answersFile, options.Stdin)
	if err != nil {
		return err
	}
	continuation, err := postClarificationContinuation(ctx, client, session, normalizedRequestID, submission)
	if err != nil {
		return err
	}
	continued, err := buildContinueOutput(config, serverURL, continuation)
	if err != nil {
		return err
	}
	output := buildAnswerOutput(continued, normalizedRequestID)
	if format == term.FormatJSON {
		return term.WriteJSON(out.Stdout, output)
	}
	_, err = fmt.Fprint(out.Stdout, renderAnswerText(output))
	return err
}

func runPlan(ctx context.Context, out *term.Output, workDir string, args []string, options Options) error {
	if err := ctx.Err(); err != nil {
		return exitcode.RuntimeError(err)
	}
	if len(args) > 0 && args[0] == "status" {
		return runPlanStatus(ctx, out, workDir, args[1:], options)
	}

	flags := flag.NewFlagSet("goalrail work plan", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	contractID := flags.String("contract-id", "", "Contract ID")
	formatValue := flags.String("format", string(term.FormatText), "output format: text or json")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, writeErr := fmt.Fprint(out.Stdout, PlanUsage())
			return writeErr
		}
		return exitcode.UsageError(err)
	}
	if flags.NArg() != 0 {
		return exitcode.UsageError(fmt.Errorf("unexpected arguments: %v", flags.Args()))
	}
	normalizedContractID := strings.TrimSpace(*contractID)
	if normalizedContractID == "" {
		return exitcode.UsageError(errors.New("--contract-id is required"))
	}
	if err := validateUUIDLike("contract_id", normalizedContractID); err != nil {
		return err
	}
	format, err := term.ParseFormat(*formatValue)
	if err != nil {
		return exitcode.UsageError(err)
	}

	work, err := loadWorkContext(ctx, workDir, options, "work plan")
	if err != nil {
		return err
	}

	plan, err := postWorkPlan(ctx, work.Client, work.Session, normalizedContractID, work.Config)
	if err != nil {
		return err
	}
	if err := validateWorkPlanContext(work.Config, normalizedContractID, plan); err != nil {
		return err
	}

	output := buildPlanOutput(work.Config, work.ServerURL, plan)
	if format == term.FormatJSON {
		return term.WriteJSON(out.Stdout, output)
	}
	_, err = fmt.Fprint(out.Stdout, renderPlanText(output))
	return err
}

func runPlanStatus(ctx context.Context, out *term.Output, workDir string, args []string, options Options) error {
	if err := ctx.Err(); err != nil {
		return exitcode.RuntimeError(err)
	}

	flags := flag.NewFlagSet("goalrail work plan status", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	planID := flags.String("plan-id", "", "WorkItemPlan ID")
	formatValue := flags.String("format", string(term.FormatText), "output format: text or json")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, writeErr := fmt.Fprint(out.Stdout, PlanStatusUsage())
			return writeErr
		}
		return exitcode.UsageError(err)
	}
	if flags.NArg() != 0 {
		return exitcode.UsageError(fmt.Errorf("unexpected arguments: %v", flags.Args()))
	}
	normalizedPlanID := strings.TrimSpace(*planID)
	if normalizedPlanID == "" {
		return exitcode.UsageError(errors.New("--plan-id is required"))
	}
	if err := validateUUIDLike("plan_id", normalizedPlanID); err != nil {
		return err
	}
	format, err := term.ParseFormat(*formatValue)
	if err != nil {
		return exitcode.UsageError(err)
	}

	work, err := loadWorkContext(ctx, workDir, options, "work plan status")
	if err != nil {
		return err
	}

	status, err := postWorkPlanStatus(ctx, work.Client, work.Session, normalizedPlanID, work.Config)
	if err != nil {
		return err
	}
	if err := validateWorkPlanStatusContext(work.Config, normalizedPlanID, status); err != nil {
		return err
	}

	output := buildPlanStatusOutput(work.Config, work.ServerURL, status)
	if format == term.FormatJSON {
		return term.WriteJSON(out.Stdout, output)
	}
	_, err = fmt.Fprint(out.Stdout, renderPlanStatusText(output))
	return err
}

func runProposal(ctx context.Context, out *term.Output, workDir string, args []string, options Options) error {
	if len(args) == 0 {
		_, err := fmt.Fprint(out.Stdout, ProposalUsage())
		return err
	}
	switch args[0] {
	case "--help", "-h":
		_, err := fmt.Fprint(out.Stdout, ProposalUsage())
		return err
	case "accept":
		return runProposalAccept(ctx, out, workDir, args[1:], options)
	default:
		return exitcode.UsageError(fmt.Errorf("unknown work proposal command %q", args[0]))
	}
}

func runProposalAccept(ctx context.Context, out *term.Output, workDir string, args []string, options Options) error {
	if err := ctx.Err(); err != nil {
		return exitcode.RuntimeError(err)
	}

	flags := flag.NewFlagSet("goalrail work proposal accept", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	proposalID := flags.String("proposal-id", "", "WorkItemPlanProposal ID")
	confirm := flags.Bool("confirm-user-acceptance", false, "confirm explicit user acceptance")
	formatValue := flags.String("format", string(term.FormatText), "output format: text or json")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, writeErr := fmt.Fprint(out.Stdout, ProposalAcceptUsage())
			return writeErr
		}
		return exitcode.UsageError(err)
	}
	if flags.NArg() != 0 {
		return exitcode.UsageError(fmt.Errorf("unexpected arguments: %v", flags.Args()))
	}
	normalizedProposalID := strings.TrimSpace(*proposalID)
	if normalizedProposalID == "" {
		return exitcode.UsageError(errors.New("--proposal-id is required"))
	}
	if err := validateUUIDLike("proposal_id", normalizedProposalID); err != nil {
		return err
	}
	if !*confirm {
		return exitcode.UsageError(errors.New("--confirm-user-acceptance is required"))
	}
	format, err := term.ParseFormat(*formatValue)
	if err != nil {
		return exitcode.UsageError(err)
	}

	work, err := loadWorkContext(ctx, workDir, options, "work proposal accept")
	if err != nil {
		return err
	}

	accepted, err := postProposalAcceptance(ctx, work.Client, work.Session, normalizedProposalID, work.Config)
	if err != nil {
		return err
	}
	if err := validateProposalAcceptanceContext(normalizedProposalID, accepted); err != nil {
		return err
	}

	output := buildProposalAcceptOutput(work.Config, work.ServerURL, accepted)
	if format == term.FormatJSON {
		return term.WriteJSON(out.Stdout, output)
	}
	_, err = fmt.Fprint(out.Stdout, renderProposalAcceptText(output))
	return err
}

func runItem(ctx context.Context, out *term.Output, workDir string, args []string, options Options) error {
	if len(args) == 0 {
		_, err := fmt.Fprint(out.Stdout, ItemUsage())
		return err
	}
	switch args[0] {
	case "--help", "-h":
		_, err := fmt.Fprint(out.Stdout, ItemUsage())
		return err
	case "show":
		return runItemShow(ctx, out, workDir, args[1:], options)
	default:
		return exitcode.UsageError(fmt.Errorf("unknown work item command %q", args[0]))
	}
}

func runItemShow(ctx context.Context, out *term.Output, workDir string, args []string, options Options) error {
	if err := ctx.Err(); err != nil {
		return exitcode.RuntimeError(err)
	}

	flags := flag.NewFlagSet("goalrail work item show", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	taskIDValue := flags.String("task-id", "", "work item/task ID")
	formatValue := flags.String("format", string(term.FormatText), "output format: text or json")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, writeErr := fmt.Fprint(out.Stdout, ItemShowUsage())
			return writeErr
		}
		return exitcode.UsageError(err)
	}
	if flags.NArg() != 0 {
		return exitcode.UsageError(fmt.Errorf("unexpected arguments: %v", flags.Args()))
	}
	normalizedTaskID := strings.TrimSpace(*taskIDValue)
	if normalizedTaskID == "" {
		return exitcode.UsageError(errors.New("--task-id is required"))
	}
	if err := validateUUIDLike("task_id", normalizedTaskID); err != nil {
		return err
	}
	normalizedTaskID = strings.ToLower(normalizedTaskID)
	format, err := term.ParseFormat(*formatValue)
	if err != nil {
		return exitcode.UsageError(err)
	}

	work, err := loadWorkContext(ctx, workDir, options, "work item show")
	if err != nil {
		return err
	}
	detail, err := getWorkItemDetail(ctx, work.Client, work.Session, normalizedTaskID, work.Config)
	if err != nil {
		return err
	}
	if err := validateWorkItemDetailContext(work.Config, normalizedTaskID, detail); err != nil {
		return err
	}
	output := buildWorkItemShowOutput(work.Config, work.ServerURL, detail)
	if format == term.FormatJSON {
		return term.WriteJSON(out.Stdout, output)
	}
	_, err = fmt.Fprint(out.Stdout, renderWorkItemShowText(output))
	return err
}

func runCheckout(ctx context.Context, out *term.Output, workDir string, args []string, options Options) error {
	if len(args) == 0 {
		_, err := fmt.Fprint(out.Stdout, CheckoutUsage())
		return err
	}
	switch args[0] {
	case "--help", "-h":
		_, err := fmt.Fprint(out.Stdout, CheckoutUsage())
		return err
	case "prepare":
		return runCheckoutPrepare(ctx, out, workDir, args[1:], options)
	default:
		return exitcode.UsageError(fmt.Errorf("unknown work checkout command %q", args[0]))
	}
}

func runCheckoutPrepare(ctx context.Context, out *term.Output, workDir string, args []string, options Options) error {
	if err := ctx.Err(); err != nil {
		return exitcode.RuntimeError(err)
	}

	flags := flag.NewFlagSet("goalrail work checkout prepare", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	taskID := flags.String("task-id", "", "WorkItem ID")
	formatValue := flags.String("format", string(term.FormatText), "output format: text or json")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, writeErr := fmt.Fprint(out.Stdout, CheckoutPrepareUsage())
			return writeErr
		}
		return exitcode.UsageError(err)
	}
	if flags.NArg() != 0 {
		return exitcode.UsageError(fmt.Errorf("unexpected arguments: %v", flags.Args()))
	}
	normalizedTaskID := strings.TrimSpace(*taskID)
	if normalizedTaskID == "" {
		return exitcode.UsageError(errors.New("--task-id is required"))
	}
	if err := validateUUIDLike("task_id", normalizedTaskID); err != nil {
		return err
	}
	normalizedTaskID = strings.ToLower(normalizedTaskID)
	format, err := term.ParseFormat(*formatValue)
	if err != nil {
		return exitcode.UsageError(err)
	}

	work, err := loadWorkContext(ctx, workDir, options, "work checkout prepare")
	if err != nil {
		return err
	}

	job, err := postCheckoutJob(ctx, work.Client, work.Session, normalizedTaskID, work.Config)
	if err != nil {
		return err
	}
	if err := validateCheckoutJobContext(work.Config, normalizedTaskID, job); err != nil {
		return err
	}

	output := buildCheckoutPrepareOutput(work.Config, work.ServerURL, job)
	if format == term.FormatJSON {
		return term.WriteJSON(out.Stdout, output)
	}
	_, err = fmt.Fprint(out.Stdout, renderCheckoutPrepareText(output))
	return err
}

func runExecution(ctx context.Context, out *term.Output, workDir string, args []string, options Options) error {
	if len(args) == 0 {
		_, err := fmt.Fprint(out.Stdout, ExecutionUsage())
		return err
	}
	switch args[0] {
	case "--help", "-h":
		_, err := fmt.Fprint(out.Stdout, ExecutionUsage())
		return err
	case "prepare":
		return runExecutionPrepare(ctx, out, workDir, args[1:], options)
	default:
		return exitcode.UsageError(fmt.Errorf("unknown work execution command %q", args[0]))
	}
}

func runExecutionPrepare(ctx context.Context, out *term.Output, workDir string, args []string, options Options) error {
	if err := ctx.Err(); err != nil {
		return exitcode.RuntimeError(err)
	}

	flags := flag.NewFlagSet("goalrail work execution prepare", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	taskID := flags.String("task-id", "", "WorkItem ID")
	checkoutReceiptID := flags.String("checkout-receipt-id", "", "CheckoutReceipt ID")
	formatValue := flags.String("format", string(term.FormatText), "output format: text or json")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, writeErr := fmt.Fprint(out.Stdout, ExecutionPrepareUsage())
			return writeErr
		}
		return exitcode.UsageError(err)
	}
	if flags.NArg() != 0 {
		return exitcode.UsageError(fmt.Errorf("unexpected arguments: %v", flags.Args()))
	}
	normalizedTaskID := strings.TrimSpace(*taskID)
	if normalizedTaskID == "" {
		return exitcode.UsageError(errors.New("--task-id is required"))
	}
	if err := validateUUIDLike("task_id", normalizedTaskID); err != nil {
		return err
	}
	normalizedTaskID = strings.ToLower(normalizedTaskID)
	normalizedCheckoutReceiptID := strings.TrimSpace(*checkoutReceiptID)
	if normalizedCheckoutReceiptID == "" {
		return exitcode.UsageError(errors.New("--checkout-receipt-id is required"))
	}
	if err := validateUUIDLike("checkout_receipt_id", normalizedCheckoutReceiptID); err != nil {
		return err
	}
	normalizedCheckoutReceiptID = strings.ToLower(normalizedCheckoutReceiptID)
	format, err := term.ParseFormat(*formatValue)
	if err != nil {
		return exitcode.UsageError(err)
	}

	work, err := loadWorkContext(ctx, workDir, options, "work execution prepare")
	if err != nil {
		return err
	}

	job, err := postExecutionJob(ctx, work.Client, work.Session, normalizedTaskID, normalizedCheckoutReceiptID, work.Config)
	if err != nil {
		return err
	}
	if err := validateExecutionJobContext(work.Config, normalizedTaskID, normalizedCheckoutReceiptID, job); err != nil {
		return err
	}

	output := buildExecutionPrepareOutput(work.Config, work.ServerURL, job)
	if format == term.FormatJSON {
		return term.WriteJSON(out.Stdout, output)
	}
	_, err = fmt.Fprint(out.Stdout, renderExecutionPrepareText(output))
	return err
}

func renderStartText(output spine.WorkStartOutput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Work intake started\n\n")
	fmt.Fprintf(&b, "Server: %s\n", output.ServerURL)
	fmt.Fprintf(&b, "Project: %s\n", output.ProjectID)
	fmt.Fprintf(&b, "Repo binding: %s\n", output.RepoBindingID)
	fmt.Fprintf(&b, "Intake: %s\n", output.IntakeID)
	fmt.Fprintf(&b, "Goal: %s\n", output.GoalID)
	if output.GoalState != "" {
		fmt.Fprintf(&b, "State: %s\n", output.GoalState)
	}
	if output.LocalConfigPath != "" {
		fmt.Fprintf(&b, "Local config: %s\n", output.LocalConfigPath)
	}
	fmt.Fprintf(&b, "\n%s\n\nNext: %s\n", workStartedNote, output.NextSuggestedCommand)
	if output.NextAction.Command != "" && output.NextAction.Available {
		fmt.Fprintf(&b, "Continue: %s\n", output.NextAction.Command)
	}
	if output.NextAction.Command != "" && !output.NextAction.Available {
		fmt.Fprintf(&b, "Planned continuation, not available yet: %s\n", output.NextAction.Command)
	}
	return b.String()
}

func buildAnswerOutput(continued spine.WorkContinueOutput, requestID string) spine.WorkAnswerOutput {
	return spine.WorkAnswerOutput{
		SchemaVersion:          continued.SchemaVersion,
		Mode:                   continued.Mode,
		ServerURL:              continued.ServerURL,
		OrganizationID:         continued.OrganizationID,
		ProjectID:              continued.ProjectID,
		RepoBindingID:          continued.RepoBindingID,
		GoalID:                 continued.GoalID,
		State:                  continued.State,
		ClarificationRequestID: requestID,
		LocalConfigPath:        continued.LocalConfigPath,
		Display:                continued.Display,
		NextAction:             continued.NextAction,
	}
}

func buildContinueOutput(config projectconfig.Config, serverURL string, continuation goalContinuationResponse) (spine.WorkContinueOutput, error) {
	goalID := strings.TrimSpace(continuation.GoalID)
	if goalID == "" {
		return spine.WorkContinueOutput{}, exitcode.RuntimeError(errors.New("goal continuation response did not include goal_id"))
	}
	state := strings.TrimSpace(continuation.State)
	if state == "" {
		return spine.WorkContinueOutput{}, exitcode.RuntimeError(errors.New("goal continuation response did not include state"))
	}

	output := spine.WorkContinueOutput{
		SchemaVersion:   cliSchemaVersion,
		Mode:            serverMode,
		ServerURL:       serverURL,
		OrganizationID:  config.OrganizationID,
		ProjectID:       config.ProjectID,
		RepoBindingID:   config.RepoBindingID,
		GoalID:          goalID,
		State:           state,
		LocalConfigPath: projectconfig.RelativePath,
	}
	if continuation.Goal != nil {
		if strings.TrimSpace(continuation.Goal.OrganizationID) != "" {
			output.OrganizationID = continuation.Goal.OrganizationID
		}
		if strings.TrimSpace(continuation.Goal.ProjectID) != "" {
			output.ProjectID = continuation.Goal.ProjectID
		}
		if strings.TrimSpace(continuation.Goal.RepoBindingID) != "" {
			output.RepoBindingID = continuation.Goal.RepoBindingID
		}
	}

	switch state {
	case "ready_for_contract_seed":
		output.Display = spine.DisplaySummary{
			Summary: "Goal is ready for contract seed. Draft the Contract handle next.",
		}
		output.NextAction = spine.NextAction{
			Kind:      "draft_contract",
			Blocking:  false,
			Command:   fmt.Sprintf("goalrail contract draft --goal-id %s --format json", goalID),
			Available: true,
		}
	case "needs_clarification":
		if continuation.ClarificationRequest == nil {
			return spine.WorkContinueOutput{}, exitcode.RuntimeError(errors.New("goal continuation response did not include clarification_request"))
		}
		if strings.TrimSpace(continuation.ClarificationRequest.ID) == "" {
			return spine.WorkContinueOutput{}, exitcode.RuntimeError(errors.New("goal continuation response did not include clarification_request.id"))
		}
		questions := make([]spine.ClarificationQuestionRef, 0, len(continuation.ClarificationRequest.Questions))
		for _, question := range continuation.ClarificationRequest.Questions {
			questions = append(questions, spine.ClarificationQuestionRef{
				ID:         question.ID,
				Text:       question.Text,
				WhyNeeded:  question.WhyNeeded,
				AnswerType: question.AnswerType,
				MapsTo:     question.MapsTo,
			})
		}
		output.Display = spine.DisplaySummary{
			Summary: "Goal needs clarification. Ask the user the returned questions before continuing.",
		}
		output.NextAction = spine.NextAction{
			Kind:      "ask_user",
			Blocking:  true,
			Available: true,
			RequestID: continuation.ClarificationRequest.ID,
			Questions: questions,
		}
	default:
		output.Display = spine.DisplaySummary{
			Summary: "Goal continuation is blocked in state " + state + ".",
		}
		output.NextAction = spine.NextAction{
			Kind:      "blocked",
			Blocking:  true,
			Available: false,
		}
	}

	return output, nil
}

func renderAnswerText(output spine.WorkAnswerOutput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Work answer submitted\n\n")
	fmt.Fprintf(&b, "Server: %s\n", output.ServerURL)
	fmt.Fprintf(&b, "Project: %s\n", output.ProjectID)
	fmt.Fprintf(&b, "Repo binding: %s\n", output.RepoBindingID)
	fmt.Fprintf(&b, "Goal: %s\n", output.GoalID)
	fmt.Fprintf(&b, "Clarification request: %s\n", output.ClarificationRequestID)
	fmt.Fprintf(&b, "State: %s\n", output.State)
	if output.LocalConfigPath != "" {
		fmt.Fprintf(&b, "Local config: %s\n", output.LocalConfigPath)
	}
	fmt.Fprintf(&b, "\n%s\n", output.Display.Summary)

	switch output.NextAction.Kind {
	case "ask_user":
		fmt.Fprintf(&b, "\nNext clarification request: %s\n", output.NextAction.RequestID)
		for i, question := range output.NextAction.Questions {
			fmt.Fprintf(&b, "%d. %s\n", i+1, question.Text)
		}
	case "draft_contract":
		if output.NextAction.Available {
			fmt.Fprintf(&b, "\nNext: %s\n", output.NextAction.Command)
		} else {
			fmt.Fprintf(&b, "\nNext planned command, not available yet: %s\n", output.NextAction.Command)
			if output.NextAction.PlannedSlice != "" {
				fmt.Fprintf(&b, "Planned slice: %s\n", output.NextAction.PlannedSlice)
			}
		}
	case "blocked":
		fmt.Fprintf(&b, "\nNext action: blocked\n")
	}
	return b.String()
}

func renderPlanText(output spine.WorkPlanOutput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Work planning recorded\n\n")
	fmt.Fprintf(&b, "Server: %s\n", output.ServerURL)
	fmt.Fprintf(&b, "Project: %s\n", output.ProjectID)
	fmt.Fprintf(&b, "Repo binding: %s\n", output.RepoBindingID)
	fmt.Fprintf(&b, "Contract: %s\n", output.ContractID)
	fmt.Fprintf(&b, "Plan: %s\n", output.PlanID)
	fmt.Fprintf(&b, "State: %s\n", output.PlanState)
	if output.LocalConfigPath != "" {
		fmt.Fprintf(&b, "Local config: %s\n", output.LocalConfigPath)
	}
	fmt.Fprintf(&b, "\n%s\n", output.Display.Summary)
	if output.NextAction.Kind != "" {
		if output.NextAction.Available && output.NextAction.Command != "" {
			fmt.Fprintf(&b, "\nNext: %s\n", output.NextAction.Command)
		} else {
			fmt.Fprintf(&b, "\nNext action: %s\n", renderPlanNextActionText(output.NextAction.Kind))
			if output.NextAction.PlannedSlice != "" {
				fmt.Fprintf(&b, "Planned slice: %s\n", output.NextAction.PlannedSlice)
			}
		}
	}
	return b.String()
}

func renderPlanStatusText(output spine.WorkPlanStatusOutput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Work planning status\n\n")
	fmt.Fprintf(&b, "Server: %s\n", output.ServerURL)
	fmt.Fprintf(&b, "Project: %s\n", output.ProjectID)
	fmt.Fprintf(&b, "Repo binding: %s\n", output.RepoBindingID)
	fmt.Fprintf(&b, "Contract: %s\n", output.ContractID)
	fmt.Fprintf(&b, "Plan: %s\n", output.PlanID)
	fmt.Fprintf(&b, "State: %s\n", output.PlanState)
	if output.ProposalID != "" {
		fmt.Fprintf(&b, "Proposal: %s\n", output.ProposalID)
		fmt.Fprintf(&b, "Proposal state: %s\n", output.ProposalState)
		fmt.Fprintf(&b, "Proposed tasks: %d\n", len(output.ProposedTasks))
	}
	if output.LocalConfigPath != "" {
		fmt.Fprintf(&b, "Local config: %s\n", output.LocalConfigPath)
	}
	fmt.Fprintf(&b, "\n%s\n", output.Display.Summary)
	if output.NextAction.Kind != "" {
		if output.NextAction.Available && output.NextAction.Command != "" {
			fmt.Fprintf(&b, "\nNext: %s\n", output.NextAction.Command)
		} else {
			fmt.Fprintf(&b, "\nNext action: %s\n", renderPlanNextActionText(output.NextAction.Kind))
			if output.NextAction.PlannedSlice != "" {
				fmt.Fprintf(&b, "Planned slice: %s\n", output.NextAction.PlannedSlice)
			}
		}
	}
	return b.String()
}

func renderProposalAcceptText(output spine.WorkProposalAcceptOutput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Work proposal accepted\n\n")
	fmt.Fprintf(&b, "Server: %s\n", output.ServerURL)
	fmt.Fprintf(&b, "Project: %s\n", output.ProjectID)
	fmt.Fprintf(&b, "Repo binding: %s\n", output.RepoBindingID)
	fmt.Fprintf(&b, "Contract: %s\n", output.ContractID)
	fmt.Fprintf(&b, "Plan: %s\n", output.PlanID)
	fmt.Fprintf(&b, "Proposal: %s\n", output.ProposalID)
	fmt.Fprintf(&b, "Created planned WorkItems: %d\n", len(output.CreatedTaskIDs))
	if output.LocalConfigPath != "" {
		fmt.Fprintf(&b, "Local config: %s\n", output.LocalConfigPath)
	}
	fmt.Fprintf(&b, "\n%s\n", output.Display.Summary)
	if output.NextAction.Kind != "" {
		if output.NextAction.Available && output.NextAction.Command != "" {
			fmt.Fprintf(&b, "\nNext: %s\n", output.NextAction.Command)
		} else {
			fmt.Fprintf(&b, "\nNext action: %s\n", renderPlanNextActionText(output.NextAction.Kind))
			if output.NextAction.PlannedSlice != "" {
				fmt.Fprintf(&b, "Planned slice: %s\n", output.NextAction.PlannedSlice)
			}
		}
	}
	return b.String()
}

func renderWorkItemShowText(output spine.WorkItemShowOutput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "WorkItem detail\n\n")
	fmt.Fprintf(&b, "Identity / lineage\n")
	fmt.Fprintf(&b, "- WorkItem: %s\n", output.WorkItemID)
	fmt.Fprintf(&b, "- Task: %s\n", output.TaskID)
	if output.GoalID != "" {
		fmt.Fprintf(&b, "- Goal: %s\n", output.GoalID)
	}
	fmt.Fprintf(&b, "- Contract: %s\n", output.ContractID)
	fmt.Fprintf(&b, "- Approved contract: %s\n", output.ApprovedContractID)
	fmt.Fprintf(&b, "- Plan: %s\n", output.PlanID)
	fmt.Fprintf(&b, "- Proposal: %s\n", output.ProposalID)
	fmt.Fprintf(&b, "- Project: %s\n", output.ProjectID)
	fmt.Fprintf(&b, "- Repo binding: %s\n", output.RepoBindingID)
	if output.LocalConfigPath != "" {
		fmt.Fprintf(&b, "- Local config: %s\n", output.LocalConfigPath)
	}
	fmt.Fprintf(&b, "\nStatus\n%s\n", output.Status)
	fmt.Fprintf(&b, "\nTitle\n%s\n", output.Title)
	fmt.Fprintf(&b, "\nSummary\n%s\n", output.Summary)
	renderStringList(&b, "\nScope", output.Scope)
	renderStringList(&b, "\nAcceptance refs", output.AcceptanceRefs)
	renderStringList(&b, "\nProof expectation refs", output.ProofExpectationRefs)
	renderSourceRefs(&b, "\nSource refs", output.SourceRefs)
	if output.OwnerHint != "" {
		fmt.Fprintf(&b, "\nOwner hint\n%s\n", output.OwnerHint)
	}
	if output.OrderIndex != nil {
		fmt.Fprintf(&b, "\nOrder index\n%d\n", *output.OrderIndex)
	}
	fmt.Fprintf(&b, "\nRead-only note\nThis command is read-only. It did not start checkout, prepare execution, create runs, gate, proof, verify, or complete work.\n")
	fmt.Fprintf(&b, "\nNext\n")
	if output.Status == "planned" && output.NextAction.Available && output.NextAction.Command != "" {
		fmt.Fprintf(&b, "Checkout preparation is the next normal stage for planned WorkItems, but it was not run by this command.\n")
		fmt.Fprintf(&b, "Run only if the human/operator chooses: %s\n", output.NextAction.Command)
	} else if output.NextAction.Kind != "" {
		fmt.Fprintf(&b, "Next action: %s\n", output.NextAction.Kind)
	} else {
		fmt.Fprintf(&b, "No next action was returned.\n")
	}
	return b.String()
}

func renderCheckoutPrepareText(output spine.WorkCheckoutPrepareOutput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Checkout job prepared\n\n")
	fmt.Fprintf(&b, "Server: %s\n", output.ServerURL)
	fmt.Fprintf(&b, "Project: %s\n", output.ProjectID)
	fmt.Fprintf(&b, "Repo binding: %s\n", output.RepoBindingID)
	fmt.Fprintf(&b, "Task: %s\n", output.TaskID)
	fmt.Fprintf(&b, "Checkout job: %s\n", output.CheckoutJobID)
	fmt.Fprintf(&b, "State: %s\n", output.CheckoutJobState)
	fmt.Fprintf(&b, "Repository: %s\n", output.Instruction.RepositoryFullName)
	fmt.Fprintf(&b, "Workflow base branch: %s\n", output.Instruction.WorkflowBaseBranch)
	if output.LocalConfigPath != "" {
		fmt.Fprintf(&b, "Local config: %s\n", output.LocalConfigPath)
	}
	fmt.Fprintf(&b, "\n%s\n", output.Display.Summary)
	if output.NextAction.Kind != "" {
		fmt.Fprintf(&b, "\nNext action: %s\n", output.NextAction.Kind)
		if output.NextAction.PlannedSlice != "" {
			fmt.Fprintf(&b, "Planned slice: %s\n", output.NextAction.PlannedSlice)
		}
	}
	return b.String()
}

func renderExecutionPrepareText(output spine.WorkExecutionPrepareOutput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Execution job prepared\n\n")
	fmt.Fprintf(&b, "Server: %s\n", output.ServerURL)
	fmt.Fprintf(&b, "Project: %s\n", output.ProjectID)
	fmt.Fprintf(&b, "Repo binding: %s\n", output.RepoBindingID)
	fmt.Fprintf(&b, "Task: %s\n", output.TaskID)
	fmt.Fprintf(&b, "Checkout receipt: %s\n", output.CheckoutReceiptID)
	fmt.Fprintf(&b, "Execution job: %s\n", output.ExecutionJobID)
	fmt.Fprintf(&b, "State: %s\n", output.ExecutionJobState)
	if output.LocalConfigPath != "" {
		fmt.Fprintf(&b, "Local config: %s\n", output.LocalConfigPath)
	}
	fmt.Fprintf(&b, "\n%s\n", output.Display.Summary)
	if output.NextAction.Kind != "" {
		fmt.Fprintf(&b, "\nNext action: %s\n", output.NextAction.Kind)
		if output.NextAction.PlannedSlice != "" {
			fmt.Fprintf(&b, "Planned slice: %s\n", output.NextAction.PlannedSlice)
		}
	}
	return b.String()
}

func renderStringList(b *strings.Builder, title string, values []string) {
	fmt.Fprintf(b, "%s\n", title)
	if len(values) == 0 {
		fmt.Fprintf(b, "- none\n")
		return
	}
	for _, value := range values {
		fmt.Fprintf(b, "- %s\n", value)
	}
}

func renderSourceRefs(b *strings.Builder, title string, refs []spine.SourceRef) {
	fmt.Fprintf(b, "%s\n", title)
	if len(refs) == 0 {
		fmt.Fprintf(b, "- none\n")
		return
	}
	for _, ref := range refs {
		if ref.Kind == "" {
			fmt.Fprintf(b, "- %s\n", ref.ID)
			continue
		}
		fmt.Fprintf(b, "- %s:%s\n", ref.Kind, ref.ID)
	}
}

func renderContinueText(output spine.WorkContinueOutput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Work continuation\n\n")
	fmt.Fprintf(&b, "Server: %s\n", output.ServerURL)
	fmt.Fprintf(&b, "Project: %s\n", output.ProjectID)
	fmt.Fprintf(&b, "Repo binding: %s\n", output.RepoBindingID)
	fmt.Fprintf(&b, "Goal: %s\n", output.GoalID)
	fmt.Fprintf(&b, "State: %s\n", output.State)
	if output.LocalConfigPath != "" {
		fmt.Fprintf(&b, "Local config: %s\n", output.LocalConfigPath)
	}
	fmt.Fprintf(&b, "\n%s\n", output.Display.Summary)

	switch output.NextAction.Kind {
	case "ask_user":
		fmt.Fprintf(&b, "\nClarification request: %s\n", output.NextAction.RequestID)
		for i, question := range output.NextAction.Questions {
			fmt.Fprintf(&b, "%d. %s\n", i+1, question.Text)
		}
	case "draft_contract":
		if output.NextAction.Available {
			fmt.Fprintf(&b, "\nNext: %s\n", output.NextAction.Command)
		} else {
			fmt.Fprintf(&b, "\nNext planned command: %s\n", output.NextAction.Command)
			if output.NextAction.PlannedSlice != "" {
				fmt.Fprintf(&b, "Planned slice: %s\n", output.NextAction.PlannedSlice)
			}
		}
	case "blocked":
		fmt.Fprintf(&b, "\nNext action: blocked\n")
	}
	return b.String()
}

func buildPlanOutput(config projectconfig.Config, serverURL string, plan workPlanResponse) spine.WorkPlanOutput {
	summary, nextAction := planStateResult(plan.State, plan.ID, "")
	return spine.WorkPlanOutput{
		SchemaVersion:   cliSchemaVersion,
		Mode:            serverMode,
		ServerURL:       serverURL,
		OrganizationID:  config.OrganizationID,
		ProjectID:       config.ProjectID,
		RepoBindingID:   spine.RepoBindingID(config.RepoBindingID),
		ContractID:      plan.ContractID,
		PlanID:          plan.ID,
		PlanState:       plan.State,
		LocalConfigPath: projectconfig.RelativePath,
		Display: spine.DisplaySummary{
			Summary: summary,
		},
		NextAction: nextAction,
	}
}

func buildPlanStatusOutput(config projectconfig.Config, serverURL string, status workPlanStatusResponse) spine.WorkPlanStatusOutput {
	summary, nextAction := planStatusResult(status.Plan.State, status.Plan.ID, status.proposalID())
	output := spine.WorkPlanStatusOutput{
		SchemaVersion:   cliSchemaVersion,
		Mode:            serverMode,
		ServerURL:       serverURL,
		OrganizationID:  config.OrganizationID,
		ProjectID:       config.ProjectID,
		RepoBindingID:   spine.RepoBindingID(config.RepoBindingID),
		ContractID:      status.Plan.ContractID,
		PlanID:          status.Plan.ID,
		PlanState:       status.Plan.State,
		LocalConfigPath: projectconfig.RelativePath,
		Display: spine.DisplaySummary{
			Summary: summary,
		},
		NextAction: nextAction,
	}
	if status.Proposal != nil {
		output.ProposalID = status.Proposal.ID
		output.ProposalState = status.Proposal.State
		output.ProposedTasks = append([]spine.ProposedWorkItem(nil), status.Proposal.ProposedTasks...)
	}
	return output
}

func buildProposalAcceptOutput(config projectconfig.Config, serverURL string, accepted workPlanAcceptanceResponse) spine.WorkProposalAcceptOutput {
	taskIDs := make([]string, 0, len(accepted.CreatedTaskIDs))
	for _, taskID := range accepted.CreatedTaskIDs {
		taskIDs = append(taskIDs, taskID)
	}
	return spine.WorkProposalAcceptOutput{
		SchemaVersion:   cliSchemaVersion,
		Mode:            serverMode,
		ServerURL:       serverURL,
		OrganizationID:  config.OrganizationID,
		ProjectID:       config.ProjectID,
		RepoBindingID:   spine.RepoBindingID(config.RepoBindingID),
		ContractID:      accepted.ContractID,
		PlanID:          accepted.PlanID,
		ProposalID:      accepted.ProposalID,
		ProposalState:   accepted.State,
		CreatedTaskIDs:  taskIDs,
		LocalConfigPath: projectconfig.RelativePath,
		Display: spine.DisplaySummary{
			Summary: fmt.Sprintf("Accepted the planning proposal and created %d planned WorkItem record(s). Execution is not available yet.", len(taskIDs)),
		},
		NextAction: spine.NextAction{
			Kind:      "prepare_checkout",
			Blocking:  false,
			Available: len(taskIDs) > 0,
			Command:   firstCheckoutCommand(taskIDs),
		},
	}
}

func buildWorkItemShowOutput(config projectconfig.Config, serverURL string, detail workItemDetailResponse) spine.WorkItemShowOutput {
	workItemID := detail.WorkItemID
	if strings.TrimSpace(workItemID) == "" {
		workItemID = detail.ID
	}
	taskID := detail.TaskID
	if strings.TrimSpace(taskID) == "" {
		taskID = workItemID
	}
	if strings.TrimSpace(taskID) == "" {
		taskID = detail.ID
	}
	nextAction := spine.NextAction{
		Kind:      detail.NextAction.Kind,
		Blocking:  detail.NextAction.Blocking,
		Command:   detail.NextAction.Command,
		Available: detail.NextAction.Available,
	}
	if nextAction.Kind == "" {
		nextAction = spine.NextAction{
			Kind:      "prepare_checkout",
			Blocking:  false,
			Available: detail.Status == "planned",
			Command:   firstCheckoutCommand([]string{taskID}),
		}
	}
	return spine.WorkItemShowOutput{
		SchemaVersion:        cliSchemaVersion,
		Mode:                 serverMode,
		ServerURL:            serverURL,
		OrganizationID:       config.OrganizationID,
		ProjectID:            config.ProjectID,
		RepoBindingID:        spine.RepoBindingID(config.RepoBindingID),
		WorkItemID:           workItemID,
		TaskID:               taskID,
		GoalID:               detail.GoalID,
		ContractID:           detail.ContractID,
		ApprovedContractID:   detail.ApprovedContractID,
		PlanID:               detail.PlanID,
		ProposalID:           detail.ProposalID,
		Status:               detail.Status,
		Title:                detail.Title,
		Summary:              detail.Summary,
		Scope:                append([]string(nil), detail.Scope...),
		AcceptanceRefs:       append([]string(nil), detail.AcceptanceRefs...),
		ProofExpectationRefs: append([]string(nil), detail.ProofExpectationRefs...),
		SourceRefs:           append([]spine.SourceRef(nil), detail.SourceRefs...),
		OwnerHint:            detail.OwnerHint,
		OrderIndex:           detail.OrderIndex,
		LocalConfigPath:      projectconfig.RelativePath,
		Display: spine.DisplaySummary{
			Summary: "Read WorkItem detail. This command is read-only and did not start checkout or execution.",
		},
		NextAction: nextAction,
	}
}

func buildCheckoutPrepareOutput(config projectconfig.Config, serverURL string, job checkoutJobResponse) spine.WorkCheckoutPrepareOutput {
	return spine.WorkCheckoutPrepareOutput{
		SchemaVersion:    cliSchemaVersion,
		Mode:             serverMode,
		ServerURL:        serverURL,
		OrganizationID:   config.OrganizationID,
		ProjectID:        config.ProjectID,
		RepoBindingID:    spine.RepoBindingID(config.RepoBindingID),
		TaskID:           job.TaskID,
		CheckoutJobID:    job.ID,
		CheckoutJobState: job.State,
		Instruction:      job.Instruction,
		LocalConfigPath:  projectconfig.RelativePath,
		Display: spine.DisplaySummary{
			Summary: "Prepared a runner checkout job and checkout instruction. Execution, Run, gate, and proof are not available in this step.",
		},
		NextAction: spine.NextAction{
			Kind:         "runner_checkout_required",
			Blocking:     true,
			Available:    false,
			PlannedSlice: "H2",
		},
	}
}

func buildExecutionPrepareOutput(config projectconfig.Config, serverURL string, job executionJobResponse) spine.WorkExecutionPrepareOutput {
	return spine.WorkExecutionPrepareOutput{
		SchemaVersion:     cliSchemaVersion,
		Mode:              serverMode,
		ServerURL:         serverURL,
		OrganizationID:    config.OrganizationID,
		ProjectID:         config.ProjectID,
		RepoBindingID:     spine.RepoBindingID(config.RepoBindingID),
		TaskID:            job.TaskID,
		CheckoutReceiptID: job.CheckoutReceiptID,
		ExecutionJobID:    job.ID,
		ExecutionJobState: job.State,
		LocalConfigPath:   projectconfig.RelativePath,
		Display: spine.DisplaySummary{
			Summary: "Execution preparation queued. No Run was created by this command; a runner must lease it to start a Run, and no command was executed.",
		},
		NextAction: spine.NextAction{
			Kind:         "runner_execution_required",
			Blocking:     true,
			Available:    false,
			PlannedSlice: "H2.3",
		},
	}
}

func firstCheckoutCommand(taskIDs []string) string {
	if len(taskIDs) == 0 {
		return ""
	}
	return fmt.Sprintf("goalrail work checkout prepare --task-id %s --format json", taskIDs[0])
}

func planStateResult(state string, planID string, proposalID string) (string, spine.NextAction) {
	switch state {
	case "queued":
		return "A server-owned WorkItemPlan is queued. A planning worker is required to produce a proposal.", spine.NextAction{
			Kind:      "planning_worker_required",
			Blocking:  true,
			Available: false,
		}
	case "leased":
		return "A server-owned WorkItemPlan is already leased. Planning is in progress and no proposal is available yet.", spine.NextAction{
			Kind:      "planning_in_progress",
			Blocking:  true,
			Available: false,
		}
	case "proposal_submitted":
		action := spine.NextAction{
			Kind:      "accept_proposal",
			Blocking:  true,
			Available: true,
		}
		if proposalID != "" {
			action.Command = fmt.Sprintf("goalrail work proposal accept --proposal-id %s --confirm-user-acceptance --format json", proposalID)
			return "A WorkItemPlan proposal is ready for user review and explicit acceptance.", action
		}
		action.Kind = "review_plan_proposal"
		action.Command = fmt.Sprintf("goalrail work plan status --plan-id %s --format json", planID)
		return "A WorkItemPlan proposal has been submitted. Read plan status to review proposal details before accepting.", action
	case "accepted":
		return "This WorkItemPlan has been accepted and planned WorkItems exist server-side. Agent-facing execution is not available yet.", spine.NextAction{
			Kind:         "planned_workitems_ready",
			Blocking:     false,
			Available:    false,
			PlannedSlice: "H",
		}
	default:
		return "The server returned an unsupported WorkItemPlan state. Manual inspection is required before continuing.", spine.NextAction{
			Kind:      "blocked",
			Blocking:  true,
			Available: false,
		}
	}
}

func planStatusResult(state string, planID string, proposalID string) (string, spine.NextAction) {
	return planStateResult(state, planID, proposalID)
}

func renderPlanNextActionText(kind string) string {
	switch kind {
	case "planning_worker_required":
		return "planning worker required; proposal generation is outside this command."
	case "planning_in_progress":
		return "planning is in progress; proposal review is not available yet."
	case "review_plan_proposal":
		return "review plan status to inspect the submitted proposal."
	case "accept_proposal":
		return "review the proposal with the user, then accept only with explicit user acceptance."
	case "planned_workitems_ready":
		return "planned WorkItems exist server-side; execution is a future step and is not available yet."
	case "prepare_checkout":
		return "prepare a checkout job for the first planned WorkItem; execution is not part of checkout preparation."
	case "runner_execution_required":
		return "runner execution start is a future step; no Run exists yet."
	case "blocked":
		return "blocked; inspect the WorkItemPlan state before continuing."
	default:
		return kind
	}
}

func resolveAnswerSubmission(answersFile string, stdin io.Reader) (workAnswerSubmission, error) {
	if answersFile == "" {
		return workAnswerSubmission{}, exitcode.UsageError(errors.New("--answers-file requires a path or -"))
	}

	var raw []byte
	var err error
	if answersFile == "-" {
		if stdin == nil {
			stdin = os.Stdin
		}
		raw, err = readLimitedBody(stdin, "--answers-file - from stdin")
		if err != nil {
			return workAnswerSubmission{}, err
		}
	} else {
		file, err := os.Open(answersFile)
		if err != nil {
			return workAnswerSubmission{}, exitcode.RuntimeError(fmt.Errorf("read --answers-file %s: %w", answersFile, err))
		}
		defer file.Close()
		raw, err = readLimitedBody(file, "--answers-file "+answersFile)
		if err != nil {
			return workAnswerSubmission{}, err
		}
	}

	var submission workAnswerSubmission
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&submission); err != nil {
		return workAnswerSubmission{}, exitcode.ValidationError(fmt.Errorf("decode --answers-file JSON: %w", err))
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return workAnswerSubmission{}, exitcode.ValidationError(errors.New("decode --answers-file JSON: multiple JSON values"))
	}
	return submission, nil
}

func resolveBody(body string, bodyFile string, bodyFileSet bool, stdin io.Reader) (string, error) {
	if !bodyFileSet {
		return strings.TrimSpace(body), nil
	}
	if bodyFile == "" {
		return "", exitcode.UsageError(errors.New("--body-file requires a path or -"))
	}

	var raw []byte
	var err error
	if bodyFile == "-" {
		if stdin == nil {
			stdin = os.Stdin
		}
		raw, err = readLimitedBody(stdin, "--body-file - from stdin")
		if err != nil {
			return "", err
		}
	} else {
		file, err := os.Open(bodyFile)
		if err != nil {
			return "", exitcode.RuntimeError(fmt.Errorf("read --body-file %s: %w", bodyFile, err))
		}
		defer file.Close()
		raw, err = readLimitedBody(file, "--body-file "+bodyFile)
		if err != nil {
			return "", err
		}
	}
	// Keep --body-file consistent with --body: surrounding paste/file
	// whitespace is not part of the intake body.
	return strings.TrimSpace(string(raw)), nil
}

func readLimitedBody(reader io.Reader, source string) ([]byte, error) {
	raw, err := io.ReadAll(io.LimitReader(reader, maxWorkBodyBytes+1))
	if err != nil {
		return nil, exitcode.RuntimeError(fmt.Errorf("read %s: %w", source, err))
	}
	if len(raw) > maxWorkBodyBytes {
		return nil, exitcode.ValidationError(fmt.Errorf("%s exceeds %d bytes", source, maxWorkBodyBytes))
	}
	return raw, nil
}

func loadUsableSession(ctx context.Context, options Options) (authstore.Session, string, HTTPClient, error) {
	store := options.Store
	if store == nil {
		path, err := authstore.DefaultPath()
		if err != nil {
			return authstore.Session{}, "", nil, exitcode.RuntimeError(err)
		}
		store = authstore.NewFileStore(path)
	}
	client := options.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	return authsession.LoadUsable(ctx, authsession.Options{
		Store:  store,
		Client: client,
		Now:    options.Now,
	})
}

func getCurrentProfile(ctx context.Context, client HTTPClient, session authstore.Session) (meResponse, error) {
	serverURL := strings.TrimRight(session.ServerURL, "/")
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, serverURL+"/v1/me", nil)
	if err != nil {
		return meResponse{}, exitcode.RuntimeError(fmt.Errorf("build current user request: %w", err))
	}
	request.Header.Set("Authorization", "Bearer "+session.AccessToken)

	response, err := client.Do(request)
	if err != nil {
		return meResponse{}, exitcode.RuntimeError(fmt.Errorf("load current user from %s: %w", serverURL, err))
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return meResponse{}, mapHTTPError("current user request", response, serverURL)
	}
	var decoded meResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return meResponse{}, exitcode.RuntimeError(fmt.Errorf("decode current user response: %w", err))
	}
	if strings.TrimSpace(decoded.User.ID) == "" {
		return meResponse{}, exitcode.RuntimeError(errors.New("current user response did not include user.id"))
	}
	return decoded, nil
}

func validateMarkerMembership(config projectconfig.Config, profile meResponse) error {
	markerOrganizationID := strings.TrimSpace(config.OrganizationID)
	if markerOrganizationID == "" {
		return exitcode.ValidationError(errors.New("local .goalrail/project.yml is missing organization_id; run goalrail init again"))
	}
	currentOrganizationID := strings.TrimSpace(profile.OrganizationMembership.OrganizationID)
	if currentOrganizationID == "" {
		return exitcode.RuntimeError(errors.New("current user response did not include organization_membership.organization_id"))
	}
	if currentOrganizationID != markerOrganizationID {
		return exitcode.ValidationError(errors.New("local .goalrail/project.yml is bound to a different GoalRail organization; run goalrail login for the active organization or re-initialize this repository"))
	}
	return nil
}

func validateMarkerProjectBinding(config projectconfig.Config) error {
	if strings.TrimSpace(config.ProjectID) == "" {
		return exitcode.ValidationError(errors.New("local .goalrail/project.yml is missing project_id; run goalrail init again"))
	}
	if strings.TrimSpace(config.RepoBindingID) == "" {
		return exitcode.ValidationError(errors.New("local .goalrail/project.yml is missing repo_binding_id; run goalrail init again"))
	}
	return nil
}

func loadWorkContext(ctx context.Context, workDir string, options Options, commandName string) (workContext, error) {
	discovered, err := gitctx.Discover(ctx, workDir)
	if err != nil {
		if errors.Is(err, gitctx.ErrNotGitRepository) {
			return workContext{}, exitcode.UsageError(fmt.Errorf("goalrail %s requires a Git worktree with .goalrail/project.yml; run goalrail init first", commandName))
		}
		return workContext{}, exitcode.RuntimeError(err)
	}
	config, ok, err := projectconfig.Read(discovered.GitRoot)
	if err != nil {
		return workContext{}, err
	}
	if !ok {
		return workContext{}, exitcode.UsageError(errors.New("missing .goalrail/project.yml; run goalrail init first"))
	}
	if err := validateMarkerProjectBinding(config); err != nil {
		return workContext{}, err
	}

	session, serverURL, client, err := loadUsableSession(ctx, options)
	if err != nil {
		return workContext{}, err
	}
	if strings.TrimRight(config.ServerURL, "/") != serverURL {
		return workContext{}, exitcode.ValidationError(errors.New("local .goalrail/project.yml is bound to a different GoalRail server; run goalrail login for that server or re-initialize this repository"))
	}

	profile, err := getCurrentProfile(ctx, client, session)
	if err != nil {
		return workContext{}, err
	}
	if err := validateMarkerMembership(config, profile); err != nil {
		return workContext{}, err
	}

	return workContext{
		Config:    config,
		Session:   session,
		ServerURL: serverURL,
		Client:    client,
	}, nil
}

func validateUUIDLike(field string, value string) error {
	if len(value) != 36 {
		return exitcode.ValidationError(fmt.Errorf("%s must be a UUID", field))
	}
	for i, char := range value {
		switch i {
		case 8, 13, 18, 23:
			if char != '-' {
				return exitcode.ValidationError(fmt.Errorf("%s must be a UUID", field))
			}
		default:
			if !isHex(char) {
				return exitcode.ValidationError(fmt.Errorf("%s must be a UUID", field))
			}
		}
	}
	return nil
}

func isHex(char rune) bool {
	return ('0' <= char && char <= '9') ||
		('a' <= char && char <= 'f') ||
		('A' <= char && char <= 'F')
}

func validateWorkPlanContext(config projectconfig.Config, contractID string, plan workPlanResponse) error {
	if string(plan.ContractID) != contractID {
		return exitcode.ValidationError(errors.New("work plan response contract_id does not match requested Contract"))
	}
	if plan.RepoBindingID != "" && string(plan.RepoBindingID) != config.RepoBindingID {
		return exitcode.ValidationError(errors.New("work plan response repo_binding_id does not match local .goalrail/project.yml; run this command from the repository bound to the Contract"))
	}
	return nil
}

func validateWorkPlanStatusContext(config projectconfig.Config, planID string, status workPlanStatusResponse) error {
	if status.Plan.ID != planID {
		return exitcode.ValidationError(errors.New("work plan status response plan.id does not match requested WorkItemPlan"))
	}
	if status.Plan.RepoBindingID != "" && string(status.Plan.RepoBindingID) != config.RepoBindingID {
		return exitcode.ValidationError(errors.New("work plan status response repo_binding_id does not match local .goalrail/project.yml; run this command from the repository bound to the WorkItemPlan"))
	}
	if status.Proposal != nil {
		if status.Proposal.PlanID != status.Plan.ID {
			return exitcode.ValidationError(errors.New("work plan status response proposal.plan_id does not match WorkItemPlan"))
		}
		if status.Proposal.RepoBindingID != "" && status.Proposal.RepoBindingID != status.Plan.RepoBindingID {
			return exitcode.ValidationError(errors.New("work plan status response proposal repo_binding_id does not match WorkItemPlan"))
		}
	}
	return nil
}

func validateProposalAcceptanceContext(proposalID string, accepted workPlanAcceptanceResponse) error {
	if accepted.ProposalID != proposalID {
		return exitcode.ValidationError(errors.New("proposal acceptance response proposal_id does not match requested WorkItemPlanProposal"))
	}
	if strings.TrimSpace(accepted.PlanID) == "" {
		return exitcode.RuntimeError(errors.New("proposal acceptance response did not include plan_id"))
	}
	if strings.TrimSpace(string(accepted.ContractID)) == "" {
		return exitcode.RuntimeError(errors.New("proposal acceptance response did not include contract_id"))
	}
	return nil
}

func validateWorkItemDetailContext(config projectconfig.Config, taskID string, detail workItemDetailResponse) error {
	responseTaskID := detail.TaskID
	if strings.TrimSpace(responseTaskID) == "" {
		responseTaskID = detail.WorkItemID
	}
	if strings.TrimSpace(responseTaskID) == "" {
		responseTaskID = detail.ID
	}
	if !sameUUIDText(responseTaskID, taskID) {
		return exitcode.ValidationError(errors.New("work item detail response task_id does not match requested WorkItem"))
	}
	if detail.WorkItemID != "" && !sameUUIDText(detail.WorkItemID, taskID) {
		return exitcode.ValidationError(errors.New("work item detail response work_item_id does not match requested WorkItem"))
	}
	if detail.ProjectID != "" && detail.ProjectID != config.ProjectID {
		return exitcode.ValidationError(errors.New("work item detail response project_id does not match local .goalrail/project.yml; run this command from the repository bound to the WorkItem"))
	}
	if detail.RepoBindingID != "" && string(detail.RepoBindingID) != config.RepoBindingID {
		return exitcode.ValidationError(errors.New("work item detail response repo_binding_id does not match local .goalrail/project.yml; run this command from the repository bound to the WorkItem"))
	}
	if strings.TrimSpace(string(detail.ContractID)) == "" {
		return exitcode.RuntimeError(errors.New("work item detail response did not include contract_id"))
	}
	if strings.TrimSpace(detail.PlanID) == "" {
		return exitcode.RuntimeError(errors.New("work item detail response did not include plan_id"))
	}
	if strings.TrimSpace(detail.ProposalID) == "" {
		return exitcode.RuntimeError(errors.New("work item detail response did not include proposal_id"))
	}
	if strings.TrimSpace(detail.Status) == "" {
		return exitcode.RuntimeError(errors.New("work item detail response did not include status"))
	}
	return nil
}

func validateCheckoutJobContext(config projectconfig.Config, taskID string, job checkoutJobResponse) error {
	if !sameUUIDText(job.TaskID, taskID) {
		return exitcode.ValidationError(errors.New("checkout job response task_id does not match requested WorkItem"))
	}
	if job.RepoBindingID != "" && string(job.RepoBindingID) != config.RepoBindingID {
		return exitcode.ValidationError(errors.New("checkout job response repo_binding_id does not match local .goalrail/project.yml; run this command from the repository bound to the WorkItem"))
	}
	if job.Instruction.TaskID != "" && !sameUUIDText(job.Instruction.TaskID, taskID) {
		return exitcode.ValidationError(errors.New("checkout instruction task_id does not match requested WorkItem"))
	}
	if job.Instruction.RepoBindingID != "" && string(job.Instruction.RepoBindingID) != config.RepoBindingID {
		return exitcode.ValidationError(errors.New("checkout instruction repo_binding_id does not match local .goalrail/project.yml"))
	}
	if job.Instruction.RawSourceUploaded {
		return exitcode.ValidationError(errors.New("checkout instruction unexpectedly claims raw source upload"))
	}
	return nil
}

func validateExecutionJobContext(config projectconfig.Config, taskID string, checkoutReceiptID string, job executionJobResponse) error {
	if !sameUUIDText(job.TaskID, taskID) {
		return exitcode.ValidationError(errors.New("execution job response task_id does not match requested WorkItem"))
	}
	if !sameUUIDText(job.CheckoutReceiptID, checkoutReceiptID) {
		return exitcode.ValidationError(errors.New("execution job response checkout_receipt_id does not match requested CheckoutReceipt"))
	}
	if job.RepoBindingID != "" && string(job.RepoBindingID) != config.RepoBindingID {
		return exitcode.ValidationError(errors.New("execution job response repo_binding_id does not match local .goalrail/project.yml; run this command from the repository bound to the WorkItem"))
	}
	return nil
}

func sameUUIDText(left string, right string) bool {
	return strings.EqualFold(strings.TrimSpace(left), strings.TrimSpace(right))
}

func postIntake(ctx context.Context, client HTTPClient, session authstore.Session, payload intakeSubmission) (intakeAcceptedResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return intakeAcceptedResponse{}, exitcode.RuntimeError(fmt.Errorf("encode intake request: %w", err))
	}
	serverURL := strings.TrimRight(session.ServerURL, "/")
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, serverURL+"/v1/intakes", bytes.NewReader(body))
	if err != nil {
		return intakeAcceptedResponse{}, exitcode.RuntimeError(fmt.Errorf("build intake request: %w", err))
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+session.AccessToken)

	response, err := client.Do(request)
	if err != nil {
		return intakeAcceptedResponse{}, exitcode.RuntimeError(fmt.Errorf("create intake on %s: %w", serverURL, err))
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusAccepted {
		return intakeAcceptedResponse{}, mapHTTPError("intake request", response, serverURL)
	}
	var decoded intakeAcceptedResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return intakeAcceptedResponse{}, exitcode.RuntimeError(fmt.Errorf("decode intake response: %w", err))
	}
	if strings.TrimSpace(decoded.IntakeID) == "" {
		return intakeAcceptedResponse{}, exitcode.RuntimeError(errors.New("intake response did not include intake_id"))
	}
	return decoded, nil
}

func promoteIntake(ctx context.Context, client HTTPClient, session authstore.Session, intakeID string) (goalResponse, error) {
	serverURL := strings.TrimRight(session.ServerURL, "/")
	endpoint := serverURL + "/v1/intakes/" + url.PathEscape(intakeID) + "/goals"
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return goalResponse{}, exitcode.RuntimeError(fmt.Errorf("build goal promotion request: %w", err))
	}
	request.Header.Set("Authorization", "Bearer "+session.AccessToken)

	response, err := client.Do(request)
	if err != nil {
		return goalResponse{}, exitcode.RuntimeError(fmt.Errorf("promote intake on %s: %w", serverURL, err))
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		return goalResponse{}, mapHTTPError("goal promotion request", response, serverURL)
	}
	var decoded goalResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return goalResponse{}, exitcode.RuntimeError(fmt.Errorf("decode goal promotion response: %w", err))
	}
	if strings.TrimSpace(decoded.ID) == "" {
		return goalResponse{}, exitcode.RuntimeError(errors.New("goal promotion response did not include id"))
	}
	return decoded, nil
}

func postGoalContinuation(ctx context.Context, client HTTPClient, session authstore.Session, goalID string) (goalContinuationResponse, error) {
	serverURL := strings.TrimRight(session.ServerURL, "/")
	endpoint := serverURL + "/v1/goals/" + url.PathEscape(goalID) + "/continuation"
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return goalContinuationResponse{}, exitcode.RuntimeError(fmt.Errorf("build goal continuation request: %w", err))
	}
	request.Header.Set("Authorization", "Bearer "+session.AccessToken)

	response, err := client.Do(request)
	if err != nil {
		return goalContinuationResponse{}, exitcode.RuntimeError(fmt.Errorf("continue goal on %s: %w", serverURL, err))
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return goalContinuationResponse{}, mapHTTPError("goal continuation request", response, serverURL)
	}
	var decoded goalContinuationResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return goalContinuationResponse{}, exitcode.RuntimeError(fmt.Errorf("decode goal continuation response: %w", err))
	}
	if strings.TrimSpace(decoded.GoalID) == "" {
		return goalContinuationResponse{}, exitcode.RuntimeError(errors.New("goal continuation response did not include goal_id"))
	}
	return decoded, nil
}

func postClarificationContinuation(ctx context.Context, client HTTPClient, session authstore.Session, requestID string, payload workAnswerSubmission) (goalContinuationResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return goalContinuationResponse{}, exitcode.RuntimeError(fmt.Errorf("encode clarification answer request: %w", err))
	}
	serverURL := strings.TrimRight(session.ServerURL, "/")
	endpoint := serverURL + "/v1/clarifications/" + url.PathEscape(requestID) + "/answers/continuation"
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return goalContinuationResponse{}, exitcode.RuntimeError(fmt.Errorf("build clarification answer request: %w", err))
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+session.AccessToken)

	response, err := client.Do(request)
	if err != nil {
		return goalContinuationResponse{}, exitcode.RuntimeError(fmt.Errorf("submit clarification answer on %s: %w", serverURL, err))
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return goalContinuationResponse{}, mapHTTPError("clarification answer request", response, serverURL)
	}
	var decoded goalContinuationResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return goalContinuationResponse{}, exitcode.RuntimeError(fmt.Errorf("decode clarification answer response: %w", err))
	}
	if strings.TrimSpace(decoded.GoalID) == "" {
		return goalContinuationResponse{}, exitcode.RuntimeError(errors.New("clarification answer response did not include goal_id"))
	}
	return decoded, nil
}

func postWorkPlan(ctx context.Context, client HTTPClient, session authstore.Session, contractID string, config projectconfig.Config) (workPlanResponse, error) {
	payload := workPlanCreateRequest{
		ProjectID:     config.ProjectID,
		RepoBindingID: config.RepoBindingID,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return workPlanResponse{}, exitcode.RuntimeError(fmt.Errorf("encode work plan request: %w", err))
	}
	serverURL := strings.TrimRight(session.ServerURL, "/")
	endpoint := serverURL + "/v1/contracts/" + url.PathEscape(contractID) + "/plans"
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return workPlanResponse{}, exitcode.RuntimeError(fmt.Errorf("build work plan request: %w", err))
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+session.AccessToken)

	response, err := client.Do(request)
	if err != nil {
		return workPlanResponse{}, exitcode.RuntimeError(fmt.Errorf("create work plan on %s: %w", serverURL, err))
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated && response.StatusCode != http.StatusOK {
		return workPlanResponse{}, mapHTTPError("work plan request", response, serverURL)
	}
	var decoded workPlanResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return workPlanResponse{}, exitcode.RuntimeError(fmt.Errorf("decode work plan response: %w", err))
	}
	if strings.TrimSpace(decoded.ID) == "" {
		return workPlanResponse{}, exitcode.RuntimeError(errors.New("work plan response did not include id"))
	}
	if strings.TrimSpace(string(decoded.ContractID)) == "" {
		return workPlanResponse{}, exitcode.RuntimeError(errors.New("work plan response did not include contract_id"))
	}
	if strings.TrimSpace(decoded.State) == "" {
		return workPlanResponse{}, exitcode.RuntimeError(errors.New("work plan response did not include state"))
	}
	return decoded, nil
}

func postWorkPlanStatus(ctx context.Context, client HTTPClient, session authstore.Session, planID string, config projectconfig.Config) (workPlanStatusResponse, error) {
	payload := workPlanCreateRequest{
		ProjectID:     config.ProjectID,
		RepoBindingID: config.RepoBindingID,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return workPlanStatusResponse{}, exitcode.RuntimeError(fmt.Errorf("encode work plan status request: %w", err))
	}
	serverURL := strings.TrimRight(session.ServerURL, "/")
	endpoint := serverURL + "/v1/plans/" + url.PathEscape(planID) + "/status"
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return workPlanStatusResponse{}, exitcode.RuntimeError(fmt.Errorf("build work plan status request: %w", err))
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+session.AccessToken)

	response, err := client.Do(request)
	if err != nil {
		return workPlanStatusResponse{}, exitcode.RuntimeError(fmt.Errorf("load work plan status on %s: %w", serverURL, err))
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return workPlanStatusResponse{}, mapHTTPError("work plan status request", response, serverURL)
	}
	var decoded workPlanStatusResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return workPlanStatusResponse{}, exitcode.RuntimeError(fmt.Errorf("decode work plan status response: %w", err))
	}
	if strings.TrimSpace(decoded.Plan.ID) == "" {
		return workPlanStatusResponse{}, exitcode.RuntimeError(errors.New("work plan status response did not include plan.id"))
	}
	if strings.TrimSpace(string(decoded.Plan.ContractID)) == "" {
		return workPlanStatusResponse{}, exitcode.RuntimeError(errors.New("work plan status response did not include plan.contract_id"))
	}
	if strings.TrimSpace(decoded.Plan.State) == "" {
		return workPlanStatusResponse{}, exitcode.RuntimeError(errors.New("work plan status response did not include plan.state"))
	}
	return decoded, nil
}

func postProposalAcceptance(ctx context.Context, client HTTPClient, session authstore.Session, proposalID string, config projectconfig.Config) (workPlanAcceptanceResponse, error) {
	payload := workPlanCreateRequest{
		ProjectID:     config.ProjectID,
		RepoBindingID: config.RepoBindingID,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return workPlanAcceptanceResponse{}, exitcode.RuntimeError(fmt.Errorf("encode proposal acceptance request: %w", err))
	}
	serverURL := strings.TrimRight(session.ServerURL, "/")
	endpoint := serverURL + "/v1/proposals/" + url.PathEscape(proposalID) + "/acceptance"
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return workPlanAcceptanceResponse{}, exitcode.RuntimeError(fmt.Errorf("build proposal acceptance request: %w", err))
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+session.AccessToken)

	response, err := client.Do(request)
	if err != nil {
		return workPlanAcceptanceResponse{}, exitcode.RuntimeError(fmt.Errorf("accept proposal on %s: %w", serverURL, err))
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		return workPlanAcceptanceResponse{}, mapHTTPError("proposal acceptance request", response, serverURL)
	}
	var decoded workPlanAcceptanceResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return workPlanAcceptanceResponse{}, exitcode.RuntimeError(fmt.Errorf("decode proposal acceptance response: %w", err))
	}
	if strings.TrimSpace(decoded.ProposalID) == "" {
		return workPlanAcceptanceResponse{}, exitcode.RuntimeError(errors.New("proposal acceptance response did not include proposal_id"))
	}
	if strings.TrimSpace(decoded.State) == "" {
		return workPlanAcceptanceResponse{}, exitcode.RuntimeError(errors.New("proposal acceptance response did not include state"))
	}
	return decoded, nil
}

func getWorkItemDetail(ctx context.Context, client HTTPClient, session authstore.Session, taskID string, config projectconfig.Config) (workItemDetailResponse, error) {
	serverURL := strings.TrimRight(session.ServerURL, "/")
	values := url.Values{}
	values.Set("project_id", config.ProjectID)
	values.Set("repo_binding_id", config.RepoBindingID)
	endpoint := serverURL + "/v1/tasks/" + url.PathEscape(taskID) + "?" + values.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return workItemDetailResponse{}, exitcode.RuntimeError(fmt.Errorf("build work item detail request: %w", err))
	}
	request.Header.Set("Authorization", "Bearer "+session.AccessToken)

	response, err := client.Do(request)
	if err != nil {
		return workItemDetailResponse{}, exitcode.RuntimeError(fmt.Errorf("load work item detail on %s: %w", serverURL, err))
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return workItemDetailResponse{}, mapHTTPError("work item detail request", response, serverURL)
	}
	var decoded workItemDetailResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return workItemDetailResponse{}, exitcode.RuntimeError(fmt.Errorf("decode work item detail response: %w", err))
	}
	if strings.TrimSpace(decoded.TaskID) == "" && strings.TrimSpace(decoded.WorkItemID) == "" && strings.TrimSpace(decoded.ID) == "" {
		return workItemDetailResponse{}, exitcode.RuntimeError(errors.New("work item detail response did not include task_id"))
	}
	return decoded, nil
}

func postCheckoutJob(ctx context.Context, client HTTPClient, session authstore.Session, taskID string, config projectconfig.Config) (checkoutJobResponse, error) {
	payload := workPlanCreateRequest{
		ProjectID:     config.ProjectID,
		RepoBindingID: config.RepoBindingID,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return checkoutJobResponse{}, exitcode.RuntimeError(fmt.Errorf("encode checkout job request: %w", err))
	}
	serverURL := strings.TrimRight(session.ServerURL, "/")
	endpoint := serverURL + "/v1/tasks/" + url.PathEscape(taskID) + "/checkout-jobs"
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return checkoutJobResponse{}, exitcode.RuntimeError(fmt.Errorf("build checkout job request: %w", err))
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+session.AccessToken)

	response, err := client.Do(request)
	if err != nil {
		return checkoutJobResponse{}, exitcode.RuntimeError(fmt.Errorf("prepare checkout job on %s: %w", serverURL, err))
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated && response.StatusCode != http.StatusOK {
		return checkoutJobResponse{}, mapHTTPError("checkout job request", response, serverURL)
	}
	var decoded checkoutJobResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return checkoutJobResponse{}, exitcode.RuntimeError(fmt.Errorf("decode checkout job response: %w", err))
	}
	if strings.TrimSpace(decoded.ID) == "" {
		return checkoutJobResponse{}, exitcode.RuntimeError(errors.New("checkout job response did not include id"))
	}
	if strings.TrimSpace(decoded.TaskID) == "" {
		return checkoutJobResponse{}, exitcode.RuntimeError(errors.New("checkout job response did not include task_id"))
	}
	if strings.TrimSpace(decoded.State) == "" {
		return checkoutJobResponse{}, exitcode.RuntimeError(errors.New("checkout job response did not include state"))
	}
	if strings.TrimSpace(decoded.Instruction.JobID) == "" {
		return checkoutJobResponse{}, exitcode.RuntimeError(errors.New("checkout job response did not include instruction.job_id"))
	}
	return decoded, nil
}

func postExecutionJob(ctx context.Context, client HTTPClient, session authstore.Session, taskID string, checkoutReceiptID string, config projectconfig.Config) (executionJobResponse, error) {
	payload := executionJobCreateRequest{
		ProjectID:         config.ProjectID,
		RepoBindingID:     config.RepoBindingID,
		CheckoutReceiptID: checkoutReceiptID,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return executionJobResponse{}, exitcode.RuntimeError(fmt.Errorf("encode execution job request: %w", err))
	}
	serverURL := strings.TrimRight(session.ServerURL, "/")
	endpoint := serverURL + "/v1/tasks/" + url.PathEscape(taskID) + "/execution-jobs"
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return executionJobResponse{}, exitcode.RuntimeError(fmt.Errorf("build execution job request: %w", err))
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+session.AccessToken)

	response, err := client.Do(request)
	if err != nil {
		return executionJobResponse{}, exitcode.RuntimeError(fmt.Errorf("prepare execution job on %s: %w", serverURL, err))
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated && response.StatusCode != http.StatusOK {
		return executionJobResponse{}, mapHTTPError("execution job request", response, serverURL)
	}
	var decoded executionJobResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return executionJobResponse{}, exitcode.RuntimeError(fmt.Errorf("decode execution job response: %w", err))
	}
	if strings.TrimSpace(decoded.ID) == "" {
		return executionJobResponse{}, exitcode.RuntimeError(errors.New("execution job response did not include id"))
	}
	if strings.TrimSpace(decoded.TaskID) == "" {
		return executionJobResponse{}, exitcode.RuntimeError(errors.New("execution job response did not include task_id"))
	}
	if strings.TrimSpace(decoded.CheckoutReceiptID) == "" {
		return executionJobResponse{}, exitcode.RuntimeError(errors.New("execution job response did not include checkout_receipt_id"))
	}
	if strings.TrimSpace(decoded.State) == "" {
		return executionJobResponse{}, exitcode.RuntimeError(errors.New("execution job response did not include state"))
	}
	return decoded, nil
}

func mapHTTPError(operation string, response *http.Response, serverURL string) error {
	message := decodeServerErrorMessage(response.Body, response.StatusCode)
	switch response.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return exitcode.UsageError(fmt.Errorf("%s failed: %s; run goalrail login %s", operation, message, serverURL))
	case http.StatusBadRequest:
		return exitcode.ValidationError(fmt.Errorf("%s validation failed: %s", operation, message))
	case http.StatusConflict:
		return exitcode.ValidationError(fmt.Errorf("%s conflict: %s", operation, message))
	default:
		return exitcode.RuntimeError(fmt.Errorf("%s failed: %s", operation, message))
	}
}

type serverErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func decodeServerErrorMessage(body io.Reader, statusCode int) string {
	raw, err := io.ReadAll(io.LimitReader(body, 1<<20))
	if err != nil {
		return fmt.Sprintf("HTTP %d", statusCode)
	}
	var decoded serverErrorResponse
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return fmt.Sprintf("HTTP %d with non-JSON response", statusCode)
	}
	if decoded.Error.Code != "" && decoded.Error.Message != "" {
		return decoded.Error.Code + ": " + decoded.Error.Message
	}
	if decoded.Error.Message != "" {
		return decoded.Error.Message
	}
	if decoded.Error.Code != "" {
		return decoded.Error.Code
	}
	return fmt.Sprintf("HTTP %d", statusCode)
}

type meResponse struct {
	User struct {
		ID          string `json:"id"`
		DisplayName string `json:"display_name"`
	} `json:"user"`
	OrganizationMembership struct {
		OrganizationID string `json:"organization_id"`
		Role           string `json:"role"`
		State          string `json:"state"`
	} `json:"organization_membership"`
}

type intakeSubmission struct {
	ProjectID     string       `json:"project_id"`
	RepoBindingID string       `json:"repo_binding_id"`
	Source        intakeSource `json:"source"`
	Title         string       `json:"title"`
	Body          string       `json:"body"`
	RequestAuthor actorRef     `json:"request_author"`
}

type intakeSource struct {
	Kind       string `json:"kind"`
	ExternalID string `json:"external_id,omitempty"`
	URL        string `json:"url,omitempty"`
}

type actorRef struct {
	Kind        string `json:"kind"`
	ID          string `json:"id"`
	DisplayName string `json:"display_name,omitempty"`
}

type intakeAcceptedResponse struct {
	IntakeID                 string `json:"intake_id"`
	OrganizationID           string `json:"organization_id"`
	ProjectID                string `json:"project_id"`
	RepoBindingID            string `json:"repo_binding_id"`
	State                    string `json:"state"`
	CanonicalContractCreated bool   `json:"canonical_contract_created"`
	Next                     string `json:"next"`
}

type goalResponse struct {
	ID             string `json:"id"`
	IntakeID       string `json:"intake_id"`
	OrganizationID string `json:"organization_id"`
	ProjectID      string `json:"project_id"`
	RepoBindingID  string `json:"repo_binding_id"`
	Title          string `json:"title"`
	Summary        string `json:"summary"`
	State          string `json:"state"`
}

type goalContinuationResponse struct {
	GoalID               string                        `json:"goal_id"`
	State                string                        `json:"state"`
	Readiness            *goalReadinessResponse        `json:"readiness,omitempty"`
	Goal                 *goalResponse                 `json:"goal,omitempty"`
	ClarificationRequest *clarificationRequestResponse `json:"clarification_request,omitempty"`
}

type goalReadinessResponse struct {
	GoalID      string   `json:"goal_id"`
	State       string   `json:"state"`
	Ready       bool     `json:"ready"`
	ReasonCodes []string `json:"reason_codes"`
	Message     string   `json:"message"`
}

type clarificationRequestResponse struct {
	ID          string                          `json:"id"`
	GoalID      string                          `json:"goal_id"`
	ReasonCodes []string                        `json:"reason_codes"`
	Questions   []clarificationQuestionResponse `json:"questions"`
	State       string                          `json:"state"`
}

type clarificationQuestionResponse struct {
	ID         string `json:"id"`
	Text       string `json:"text"`
	WhyNeeded  string `json:"why_needed"`
	AnswerType string `json:"answer_type"`
	MapsTo     string `json:"maps_to"`
}

type workAnswerSubmission struct {
	Answers []workAnswerItem `json:"answers"`
}

type workAnswerItem struct {
	QuestionID string    `json:"question_id"`
	Value      string    `json:"value"`
	ActorRef   *actorRef `json:"actor_ref,omitempty"`
}

type workPlanCreateRequest struct {
	ProjectID     string `json:"project_id"`
	RepoBindingID string `json:"repo_binding_id"`
}

type executionJobCreateRequest struct {
	ProjectID         string `json:"project_id"`
	RepoBindingID     string `json:"repo_binding_id"`
	CheckoutReceiptID string `json:"checkout_receipt_id"`
}

type workPlanResponse struct {
	ID                 string              `json:"id"`
	ContractID         spine.ContractID    `json:"contract_id"`
	ApprovedContractID string              `json:"approved_contract_id"`
	RepoBindingID      spine.RepoBindingID `json:"repo_binding_id"`
	State              string              `json:"state"`
	ApprovedContract   json.RawMessage     `json:"approved_contract,omitempty"`
}

type workPlanStatusResponse struct {
	Plan     workPlanResponse          `json:"plan"`
	Proposal *workPlanProposalResponse `json:"proposal,omitempty"`
}

func (r workPlanStatusResponse) proposalID() string {
	if r.Proposal == nil {
		return ""
	}
	return r.Proposal.ID
}

type workPlanProposalResponse struct {
	ID                 string                   `json:"id"`
	PlanID             string                   `json:"plan_id"`
	ContractID         spine.ContractID         `json:"contract_id"`
	ApprovedContractID string                   `json:"approved_contract_id"`
	RepoBindingID      spine.RepoBindingID      `json:"repo_binding_id"`
	State              string                   `json:"state"`
	ProposedTasks      []spine.ProposedWorkItem `json:"proposed_tasks"`
}

type workPlanAcceptanceResponse struct {
	ProposalID     string           `json:"proposal_id"`
	PlanID         string           `json:"plan_id"`
	ContractID     spine.ContractID `json:"contract_id"`
	State          string           `json:"state"`
	CreatedTaskIDs []string         `json:"created_task_ids"`
}

type workItemNextActionResponse struct {
	Kind      string `json:"kind"`
	Blocking  bool   `json:"blocking"`
	Command   string `json:"command,omitempty"`
	Available bool   `json:"available"`
}

type workItemDetailResponse struct {
	ID                   string                     `json:"id"`
	WorkItemID           string                     `json:"work_item_id"`
	TaskID               string                     `json:"task_id"`
	OrganizationID       string                     `json:"organization_id,omitempty"`
	ProjectID            string                     `json:"project_id,omitempty"`
	GoalID               string                     `json:"goal_id,omitempty"`
	ContractID           spine.ContractID           `json:"contract_id"`
	ApprovedContractID   string                     `json:"approved_contract_id"`
	PlanID               string                     `json:"plan_id"`
	ProposalID           string                     `json:"proposal_id"`
	RepoBindingID        spine.RepoBindingID        `json:"repo_binding_id"`
	Status               string                     `json:"status"`
	Title                string                     `json:"title"`
	Summary              string                     `json:"summary"`
	Scope                []string                   `json:"scope"`
	AcceptanceRefs       []string                   `json:"acceptance_refs"`
	ProofExpectationRefs []string                   `json:"proof_expectation_refs"`
	SourceRefs           []spine.SourceRef          `json:"source_refs,omitempty"`
	OwnerHint            string                     `json:"owner_hint,omitempty"`
	OrderIndex           *int                       `json:"order_index,omitempty"`
	NextAction           workItemNextActionResponse `json:"next_action"`
}

type checkoutJobResponse struct {
	ID                 string                    `json:"id"`
	TaskID             string                    `json:"task_id"`
	ContractID         spine.ContractID          `json:"contract_id"`
	ApprovedContractID string                    `json:"approved_contract_id"`
	PlanID             string                    `json:"plan_id"`
	ProposalID         string                    `json:"proposal_id"`
	RepoBindingID      spine.RepoBindingID       `json:"repo_binding_id"`
	State              string                    `json:"state"`
	Instruction        spine.CheckoutInstruction `json:"instruction"`
}

type executionJobResponse struct {
	ID                 string              `json:"id"`
	TaskID             string              `json:"task_id"`
	ContractID         spine.ContractID    `json:"contract_id"`
	ApprovedContractID string              `json:"approved_contract_id"`
	PlanID             string              `json:"plan_id"`
	ProposalID         string              `json:"proposal_id"`
	RepoBindingID      spine.RepoBindingID `json:"repo_binding_id"`
	CheckoutJobID      string              `json:"checkout_job_id"`
	CheckoutReceiptID  string              `json:"checkout_receipt_id"`
	State              string              `json:"state"`
	ExecutionMode      string              `json:"execution_mode"`
}
