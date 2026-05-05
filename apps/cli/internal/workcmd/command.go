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
	serverMode           = "server"
	nextSuggestedCommand = "goalrail readiness scan --path ."
	workStartedMessage   = "Work intake started."
	workStartedNote      = "This created an IntakeRecord and promoted it to a Goal on the GoalRail server.\nNo audit, hooks, branch creation, deploy keys, provider integration, runner, gate, proof, or verification were configured."
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
	default:
		return exitcode.UsageError(fmt.Errorf("unknown work command %q", args[0]))
	}
}

func Usage() string {
	return "Usage: goalrail work <command> [options]\n\nCommands:\n  start      create a server-backed IntakeRecord and Goal from the local project marker\n\nRun goalrail work <command> --help for command usage.\n"
}

func StartUsage() string {
	return "Usage: goalrail work start --title <title> [--body <body>] [--format text|json]\n\nCreates an IntakeRecord and promotes it to a Goal using the current Git root .goalrail/project.yml marker and the stored goalrail login profile.\n\nThis command does not configure audit, create hooks, create branches, provision deploy keys, connect provider integrations, run workers, gates, proof, or verification.\n"
}

func runStart(ctx context.Context, out *term.Output, workDir string, args []string, options Options) error {
	if err := ctx.Err(); err != nil {
		return exitcode.RuntimeError(err)
	}

	flags := flag.NewFlagSet("goalrail work start", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	title := flags.String("title", "", "work title")
	body := flags.String("body", "", "work body")
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
	intake, err := postIntake(ctx, client, session, intakeSubmission{
		ProjectID:     config.ProjectID,
		RepoBindingID: config.RepoBindingID,
		Source: intakeSource{
			Kind:       "goalrail_cli",
			ExternalID: "work start",
		},
		Title: strings.TrimSpace(*title),
		Body:  strings.TrimSpace(*body),
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
		Mode:                 serverMode,
		ServerURL:            serverURL,
		OrganizationID:       intake.OrganizationID,
		ProjectID:            intake.ProjectID,
		RepoBindingID:        intake.RepoBindingID,
		IntakeID:             intake.IntakeID,
		IntakeState:          intake.State,
		GoalID:               goal.ID,
		GoalState:            goal.State,
		Title:                normalizedTitle,
		LocalConfigPath:      projectconfig.RelativePath,
		Message:              workStartedMessage,
		NextSuggestedCommand: nextSuggestedCommand,
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
	return b.String()
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
