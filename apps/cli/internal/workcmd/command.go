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
	default:
		return exitcode.UsageError(fmt.Errorf("unknown work command %q", args[0]))
	}
}

func Usage() string {
	return "Usage: goalrail work <command> [options]\n\nCommands:\n  start      create a server-backed IntakeRecord and Goal from the local project marker\n  continue   reconcile Goal readiness and return the next action\n  answer     submit clarification answers and return the next action\n  plan       create or return a server WorkItemPlan for an approved Contract\n\nRun goalrail work <command> --help for command usage.\n"
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
	return "Usage: goalrail work plan --contract-id <contract_id> [--format text|json]\n\nCreates or returns a server WorkItemPlan for an approved Contract using the current Git root .goalrail/project.yml marker and the stored goalrail login profile.\n\nThis command does not acquire planning leases, submit proposals, accept proposals, create WorkItems, run workers, gates, proof, or verification.\n"
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

	session, serverURL, err := loadUsableSession(options)
	if err != nil {
		return err
	}
	if strings.TrimRight(config.ServerURL, "/") != serverURL {
		return exitcode.ValidationError(errors.New("local .goalrail/project.yml is bound to a different GoalRail server; run goalrail login for that server or re-initialize this repository"))
	}

	client := options.HTTPClient
	if client == nil {
		client = http.DefaultClient
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

	session, serverURL, err := loadUsableSession(options)
	if err != nil {
		return err
	}
	if strings.TrimRight(config.ServerURL, "/") != serverURL {
		return exitcode.ValidationError(errors.New("local .goalrail/project.yml is bound to a different GoalRail server; run goalrail login for that server or re-initialize this repository"))
	}

	client := options.HTTPClient
	if client == nil {
		client = http.DefaultClient
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

	session, serverURL, err := loadUsableSession(options)
	if err != nil {
		return err
	}
	if strings.TrimRight(config.ServerURL, "/") != serverURL {
		return exitcode.ValidationError(errors.New("local .goalrail/project.yml is bound to a different GoalRail server; run goalrail login for that server or re-initialize this repository"))
	}

	client := options.HTTPClient
	if client == nil {
		client = http.DefaultClient
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

	discovered, err := gitctx.Discover(ctx, workDir)
	if err != nil {
		if errors.Is(err, gitctx.ErrNotGitRepository) {
			return exitcode.UsageError(errors.New("goalrail work plan requires a Git worktree with .goalrail/project.yml; run goalrail init first"))
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
	if err := validateMarkerProjectBinding(config); err != nil {
		return err
	}

	session, serverURL, err := loadUsableSession(options)
	if err != nil {
		return err
	}
	if strings.TrimRight(config.ServerURL, "/") != serverURL {
		return exitcode.ValidationError(errors.New("local .goalrail/project.yml is bound to a different GoalRail server; run goalrail login for that server or re-initialize this repository"))
	}

	client := options.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	profile, err := getCurrentProfile(ctx, client, session)
	if err != nil {
		return err
	}
	if err := validateMarkerMembership(config, profile); err != nil {
		return err
	}

	plan, err := postWorkPlan(ctx, client, session, normalizedContractID, config)
	if err != nil {
		return err
	}
	if err := validateWorkPlanContext(config, normalizedContractID, plan); err != nil {
		return err
	}

	output := buildPlanOutput(config, serverURL, plan)
	if format == term.FormatJSON {
		return term.WriteJSON(out.Stdout, output)
	}
	_, err = fmt.Fprint(out.Stdout, renderPlanText(output))
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
		fmt.Fprintf(&b, "\nNext action: %s\n", renderPlanNextActionText(output.NextAction.Kind))
		if output.NextAction.PlannedSlice != "" {
			fmt.Fprintf(&b, "Planned slice: %s\n", output.NextAction.PlannedSlice)
		}
	}
	return b.String()
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
	summary, nextAction := planStateResult(plan.State)
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

func planStateResult(state string) (string, spine.NextAction) {
	switch state {
	case "queued":
		return "A server-owned WorkItemPlan is queued. A planning worker is required to produce a proposal.", spine.NextAction{
			Kind:         "planning_worker_required",
			Blocking:     true,
			Available:    false,
			PlannedSlice: "G2",
		}
	case "leased":
		return "A server-owned WorkItemPlan is already leased. Planning is in progress and no proposal is available yet.", spine.NextAction{
			Kind:         "planning_in_progress",
			Blocking:     true,
			Available:    false,
			PlannedSlice: "G2",
		}
	case "proposal_submitted":
		return "A WorkItemPlan proposal has already been submitted. Proposal review and acceptance are not available in this CLI slice.", spine.NextAction{
			Kind:         "review_plan_proposal",
			Blocking:     true,
			Available:    false,
			PlannedSlice: "G3",
		}
	case "accepted":
		return "This WorkItemPlan has already been accepted and planned WorkItems exist server-side. Agent-facing WorkItem handling is not available in this CLI slice.", spine.NextAction{
			Kind:      "planned_workitems_exist",
			Blocking:  false,
			Available: false,
		}
	default:
		return "The server returned an unsupported WorkItemPlan state. Manual inspection is required before continuing.", spine.NextAction{
			Kind:      "blocked",
			Blocking:  true,
			Available: false,
		}
	}
}

func renderPlanNextActionText(kind string) string {
	switch kind {
	case "planning_worker_required":
		return "planning worker required; proposal generation is not available in this slice."
	case "planning_in_progress":
		return "planning is in progress; proposal review is not available in this slice."
	case "review_plan_proposal":
		return "review plan proposal; proposal acceptance is not available in this slice."
	case "planned_workitems_exist":
		return "planned WorkItems already exist server-side; agent-facing WorkItem handling is not available in this slice."
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

func loadUsableSession(options Options) (authstore.Session, string, error) {
	session, err := loadSession(options)
	if err != nil {
		return authstore.Session{}, "", err
	}
	now := time.Now
	if options.Now != nil {
		now = options.Now
	}
	if !session.AccessTokenExpiresAt.After(now().UTC()) {
		return authstore.Session{}, "", exitcode.UsageError(fmt.Errorf("login expired; run goalrail login %s", session.ServerURL))
	}
	return session, strings.TrimRight(session.ServerURL, "/"), nil
}

func loadSession(options Options) (authstore.Session, error) {
	store := options.Store
	if store == nil {
		path, err := authstore.DefaultPath()
		if err != nil {
			return authstore.Session{}, exitcode.RuntimeError(err)
		}
		store = authstore.NewFileStore(path)
	}
	session, err := store.Load()
	if err != nil {
		if errors.Is(err, authstore.ErrSessionNotFound) {
			return authstore.Session{}, exitcode.UsageError(errors.New("not logged in; run goalrail login <server_url>"))
		}
		return authstore.Session{}, exitcode.RuntimeError(err)
	}
	return session, nil
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

type workPlanResponse struct {
	ID                 string              `json:"id"`
	ContractID         spine.ContractID    `json:"contract_id"`
	ApprovedContractID string              `json:"approved_contract_id"`
	RepoBindingID      spine.RepoBindingID `json:"repo_binding_id"`
	State              string              `json:"state"`
}
