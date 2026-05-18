package contractcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/heurema/goalrail/apps/cli/internal/authsession"
	"github.com/heurema/goalrail/apps/cli/internal/authstore"
	"github.com/heurema/goalrail/apps/cli/internal/contract"
	"github.com/heurema/goalrail/apps/cli/internal/exitcode"
	"github.com/heurema/goalrail/apps/cli/internal/projectconfig"
	"github.com/heurema/goalrail/apps/cli/internal/projectscan"
	"github.com/heurema/goalrail/apps/cli/internal/spine"
	"github.com/heurema/goalrail/apps/cli/internal/term"
)

const (
	serverMode             = "server"
	cliSchemaVersion       = "goalrail.cli.v1"
	maxContractUpdateBytes = 1 << 20
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
	CacheRoot  string
	Now        func() time.Time
	Stdin      io.Reader
}

func Run(ctx context.Context, out *term.Output, workDir string, args []string) error {
	return RunWithOptions(ctx, out, workDir, args, Options{})
}

func RunWithOptions(ctx context.Context, out *term.Output, workDir string, args []string, options Options) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		_, err := fmt.Fprint(out.Stdout, Usage())
		return err
	}

	switch args[0] {
	case "validate":
		return runValidate(ctx, out, workDir, args[1:])
	case "show":
		return runShow(ctx, out, workDir, args[1:], options)
	case "draft":
		return runDraft(ctx, out, workDir, args[1:], options)
	case "update":
		return runUpdate(ctx, out, workDir, args[1:], options)
	case "submit":
		return runSubmit(ctx, out, workDir, args[1:], options)
	case "approve":
		return runApprove(ctx, out, workDir, args[1:], options)
	default:
		return exitcode.UsageError(fmt.Errorf("unknown contract command %q", args[0]))
	}
}

func runShow(ctx context.Context, out *term.Output, workDir string, args []string, options Options) error {
	if err := ctx.Err(); err != nil {
		return exitcode.RuntimeError(err)
	}

	flags := flag.NewFlagSet("goalrail contract show", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	contractID := flags.String("contract-id", "", "Contract ID")
	formatValue := flags.String("format", string(term.FormatText), "output format: text or json")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, writeErr := fmt.Fprint(out.Stdout, ShowUsage())
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

	commandContext, err := loadContractCommandContext(ctx, workDir, "show", options)
	if err != nil {
		return err
	}
	aggregate, err := getContractAggregate(ctx, commandContext.Client, commandContext.Session, normalizedContractID)
	if err != nil {
		return err
	}
	if err := validateContractAggregateContext(commandContext.Config, normalizedContractID, aggregate); err != nil {
		return err
	}

	var draft *contractDraftBodyResponse
	if strings.TrimSpace(aggregate.CurrentDraftID) != "" {
		loadedDraft, err := getContractCurrentDraft(ctx, commandContext.Client, commandContext.Session, normalizedContractID)
		if err != nil {
			return err
		}
		if err := validateContractDraftBodyContext(commandContext.Config, normalizedContractID, aggregate.CurrentDraftID, loadedDraft); err != nil {
			return err
		}
		draft = &loadedDraft
	}

	output := buildShowOutput(commandContext.Config, commandContext.ServerURL, aggregate, draft)
	if format == term.FormatJSON {
		return term.WriteJSON(out.Stdout, output)
	}
	_, err = fmt.Fprint(out.Stdout, renderShowText(output))
	return err
}

func runValidate(ctx context.Context, out *term.Output, workDir string, args []string) error {
	if err := ctx.Err(); err != nil {
		return exitcode.RuntimeError(err)
	}

	flags := flag.NewFlagSet("goalrail contract validate", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	fileValue := flags.String("file", "", "contract JSON file")
	formatValue := flags.String("format", string(term.FormatText), "output format: text or json")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, writeErr := fmt.Fprint(out.Stdout, ValidateUsage())
			return writeErr
		}
		return exitcode.UsageError(err)
	}

	if flags.NArg() != 0 {
		return exitcode.UsageError(fmt.Errorf("unexpected arguments: %v", flags.Args()))
	}
	if *fileValue == "" {
		return exitcode.UsageError(errors.New("missing required --file <contract.json>"))
	}

	format, err := term.ParseFormat(*formatValue)
	if err != nil {
		return exitcode.UsageError(err)
	}

	filePath := resolvePath(workDir, *fileValue)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return exitcode.RuntimeError(fmt.Errorf("read contract file %q: %w", *fileValue, err))
	}

	var parsed spine.Contract
	if err := json.Unmarshal(data, &parsed); err != nil {
		return exitcode.RuntimeError(fmt.Errorf("parse contract file %q: %w", *fileValue, err))
	}

	report := contract.Validate(parsed)
	if format == term.FormatJSON {
		if err := term.WriteJSON(out.Stdout, report); err != nil {
			return err
		}
	} else if err := writeText(out.Stdout, report); err != nil {
		return err
	}

	if !report.Valid {
		return exitcode.ValidationError(contract.ErrValidationFailed)
	}
	return nil
}

func runDraft(ctx context.Context, out *term.Output, workDir string, args []string, options Options) error {
	if err := ctx.Err(); err != nil {
		return exitcode.RuntimeError(err)
	}

	flags := flag.NewFlagSet("goalrail contract draft", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	goalID := flags.String("goal-id", "", "Goal ID")
	formatValue := flags.String("format", string(term.FormatText), "output format: text or json")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, writeErr := fmt.Fprint(out.Stdout, DraftUsage())
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

	facts, err := projectscan.DiscoverGit(ctx, workDir)
	if err != nil {
		if errors.Is(err, projectscan.ErrNotGitRepository) {
			return exitcode.UsageError(errors.New("goalrail contract draft requires a Git worktree with .goalrail/project.yml; run goalrail init first"))
		}
		return exitcode.RuntimeError(err)
	}
	config, ok, err := projectconfig.Read(facts.CanonicalRepoRoot)
	if err != nil {
		return err
	}
	if !ok {
		return exitcode.UsageError(errors.New("missing .goalrail/project.yml; run goalrail init first"))
	}
	if err := validateMarkerProjectBinding(config); err != nil {
		return err
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

	receipt, err := refreshLocalRepoReceipt(ctx, facts, config, options)
	if err != nil {
		return err
	}
	contractDraft, err := postContractDraft(ctx, client, session, normalizedGoalID, config)
	if err != nil {
		return err
	}
	if err := validateContractDraftContext(config, normalizedGoalID, contractDraft); err != nil {
		return err
	}

	output := buildDraftOutput(config, serverURL, normalizedGoalID, contractDraft, receipt)
	if format == term.FormatJSON {
		return term.WriteJSON(out.Stdout, output)
	}
	_, err = fmt.Fprint(out.Stdout, renderDraftText(output))
	return err
}

func runUpdate(ctx context.Context, out *term.Output, workDir string, args []string, options Options) error {
	if err := ctx.Err(); err != nil {
		return exitcode.RuntimeError(err)
	}

	flags := flag.NewFlagSet("goalrail contract update", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	contractID := flags.String("contract-id", "", "Contract ID")
	fieldsFile := flags.String("fields-file", "", "structured contract fields JSON file or - for stdin")
	formatValue := flags.String("format", string(term.FormatText), "output format: text or json")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, writeErr := fmt.Fprint(out.Stdout, UpdateUsage())
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
	if strings.TrimSpace(*fieldsFile) == "" {
		return exitcode.UsageError(errors.New("--fields-file is required"))
	}
	format, err := term.ParseFormat(*formatValue)
	if err != nil {
		return exitcode.UsageError(err)
	}

	facts, err := projectscan.DiscoverGit(ctx, workDir)
	if err != nil {
		if errors.Is(err, projectscan.ErrNotGitRepository) {
			return exitcode.UsageError(errors.New("goalrail contract update requires a Git worktree with .goalrail/project.yml; run goalrail init first"))
		}
		return exitcode.RuntimeError(err)
	}
	config, ok, err := projectconfig.Read(facts.CanonicalRepoRoot)
	if err != nil {
		return err
	}
	if !ok {
		return exitcode.UsageError(errors.New("missing .goalrail/project.yml; run goalrail init first"))
	}
	if err := validateMarkerProjectBinding(config); err != nil {
		return err
	}

	session, serverURL, client, err := loadUsableSession(ctx, options)
	if err != nil {
		return err
	}
	if strings.TrimRight(config.ServerURL, "/") != serverURL {
		return exitcode.ValidationError(errors.New("local .goalrail/project.yml is bound to a different GoalRail server; run goalrail login for that server or re-initialize this repository"))
	}

	update, err := readContractUpdateRequest(workDir, *fieldsFile, options)
	if err != nil {
		return err
	}
	update.ProjectID = config.ProjectID
	update.RepoBindingID = config.RepoBindingID

	profile, err := getCurrentProfile(ctx, client, session)
	if err != nil {
		return err
	}
	if err := validateMarkerMembership(config, profile); err != nil {
		return err
	}
	update.UpdatedBy = actorRefForProfile(profile)

	contractResponse, err := patchContractUpdate(ctx, client, session, normalizedContractID, update)
	if err != nil {
		return err
	}
	if err := validateContractUpdateContext(config, normalizedContractID, contractResponse); err != nil {
		return err
	}

	output := buildUpdateOutput(config, serverURL, contractResponse, update.changedFields)
	if format == term.FormatJSON {
		return term.WriteJSON(out.Stdout, output)
	}
	_, err = fmt.Fprint(out.Stdout, renderUpdateText(output))
	return err
}

func runSubmit(ctx context.Context, out *term.Output, workDir string, args []string, options Options) error {
	if err := ctx.Err(); err != nil {
		return exitcode.RuntimeError(err)
	}

	flags := flag.NewFlagSet("goalrail contract submit", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	contractID := flags.String("contract-id", "", "Contract ID")
	formatValue := flags.String("format", string(term.FormatText), "output format: text or json")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, writeErr := fmt.Fprint(out.Stdout, SubmitUsage())
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

	commandContext, err := loadContractCommandContext(ctx, workDir, "submit", options)
	if err != nil {
		return err
	}
	contractResponse, err := postContractSubmit(ctx, commandContext.Client, commandContext.Session, normalizedContractID, commandContext.Config)
	if err != nil {
		return err
	}
	if err := validateContractUpdateContext(commandContext.Config, normalizedContractID, contractResponse); err != nil {
		return err
	}

	output := buildSubmitOutput(commandContext.Config, commandContext.ServerURL, contractResponse)
	if format == term.FormatJSON {
		return term.WriteJSON(out.Stdout, output)
	}
	_, err = fmt.Fprint(out.Stdout, renderSubmitText(output))
	return err
}

func runApprove(ctx context.Context, out *term.Output, workDir string, args []string, options Options) error {
	if err := ctx.Err(); err != nil {
		return exitcode.RuntimeError(err)
	}

	flags := flag.NewFlagSet("goalrail contract approve", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	contractID := flags.String("contract-id", "", "Contract ID")
	confirmUserApproval := flags.Bool("confirm-user-approval", false, "confirm the user explicitly approved this Contract")
	formatValue := flags.String("format", string(term.FormatText), "output format: text or json")

	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, writeErr := fmt.Fprint(out.Stdout, ApproveUsage())
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
	if !*confirmUserApproval {
		return exitcode.UsageError(errors.New("goalrail contract approve requires --confirm-user-approval after explicit user approval"))
	}
	format, err := term.ParseFormat(*formatValue)
	if err != nil {
		return exitcode.UsageError(err)
	}

	commandContext, err := loadContractCommandContext(ctx, workDir, "approve", options)
	if err != nil {
		return err
	}
	contractResponse, err := postContractApprove(ctx, commandContext.Client, commandContext.Session, normalizedContractID, commandContext.Config)
	if err != nil {
		return err
	}
	if err := validateContractUpdateContext(commandContext.Config, normalizedContractID, contractResponse); err != nil {
		return err
	}

	output := buildApproveOutput(commandContext.Config, commandContext.ServerURL, contractResponse)
	if format == term.FormatJSON {
		return term.WriteJSON(out.Stdout, output)
	}
	_, err = fmt.Fprint(out.Stdout, renderApproveText(output))
	return err
}

type contractCommandContext struct {
	Config    projectconfig.Config
	Session   authstore.Session
	ServerURL string
	Client    HTTPClient
	Profile   meResponse
}

func loadContractCommandContext(ctx context.Context, workDir string, command string, options Options) (contractCommandContext, error) {
	facts, err := projectscan.DiscoverGit(ctx, workDir)
	if err != nil {
		if errors.Is(err, projectscan.ErrNotGitRepository) {
			return contractCommandContext{}, exitcode.UsageError(fmt.Errorf("goalrail contract %s requires a Git worktree with .goalrail/project.yml; run goalrail init first", command))
		}
		return contractCommandContext{}, exitcode.RuntimeError(err)
	}
	config, ok, err := projectconfig.Read(facts.CanonicalRepoRoot)
	if err != nil {
		return contractCommandContext{}, err
	}
	if !ok {
		return contractCommandContext{}, exitcode.UsageError(errors.New("missing .goalrail/project.yml; run goalrail init first"))
	}
	if err := validateMarkerProjectBinding(config); err != nil {
		return contractCommandContext{}, err
	}

	session, serverURL, client, err := loadUsableSession(ctx, options)
	if err != nil {
		return contractCommandContext{}, err
	}
	if strings.TrimRight(config.ServerURL, "/") != serverURL {
		return contractCommandContext{}, exitcode.ValidationError(errors.New("local .goalrail/project.yml is bound to a different GoalRail server; run goalrail login for that server or re-initialize this repository"))
	}

	profile, err := getCurrentProfile(ctx, client, session)
	if err != nil {
		return contractCommandContext{}, err
	}
	if err := validateMarkerMembership(config, profile); err != nil {
		return contractCommandContext{}, err
	}

	return contractCommandContext{
		Config:    config,
		Session:   session,
		ServerURL: serverURL,
		Client:    client,
		Profile:   profile,
	}, nil
}

func resolvePath(workDir, value string) string {
	if filepath.IsAbs(value) {
		return value
	}
	if workDir == "" {
		workDir = "."
	}
	return filepath.Join(workDir, value)
}

func writeText(w io.Writer, report spine.ContractValidationReport) error {
	var b strings.Builder
	fmt.Fprintf(&b, "Contract validation: %s\n", report.ContractID)
	if report.Valid {
		b.WriteString("Status: valid\n")
	} else {
		b.WriteString("Status: invalid\n\nFindings:\n")
		for _, finding := range report.Findings {
			fmt.Fprintf(&b, "- [%s] %s: %s\n", finding.Severity, finding.Field, finding.Message)
		}
	}

	_, err := fmt.Fprint(w, b.String())
	return err
}

func Usage() string {
	return "Usage: goalrail contract <command> [options]\n\nCommands:\n  show        inspect a server Contract and current draft without mutating state\n  draft       create or return a server Contract draft handle for a ready Goal\n  update      update proposed fields on a server ContractDraft\n  submit      submit a draft Contract for explicit user approval\n  approve     approve a submitted Contract after explicit user confirmation\n  validate    validate a contract JSON file\n\nRun goalrail contract <command> --help for command usage.\n"
}

func ValidateUsage() string {
	return "Usage: goalrail contract validate --file <contract.json> [--format text|json]\n\nValidates the minimum contract fields needed before approval or execution.\n"
}

func ShowUsage() string {
	return "Usage: goalrail contract show --contract-id <contract_id> [--format text|json]\n\nInspects a server Contract and its current draft body, when available, using read-only authenticated API requests. The command validates the Git-root .goalrail/project.yml marker plus login and Organization marker. It does not create, update, submit, approve, plan, run workers, gates, proof, or verification.\n"
}

func DraftUsage() string {
	return "Usage: goalrail contract draft --goal-id <goal_id> [--format text|json]\n\nCreates or returns a server Contract draft handle for a ready Goal using the current Git root .goalrail/project.yml marker and the stored goalrail login profile. It refreshes local Project Scan evidence and returns a local repository receipt. It does not upload raw source bodies, update contract fields, create WorkItems, run workers, gates, proof, or verification.\n"
}

func UpdateUsage() string {
	return "Usage: goalrail contract update --contract-id <contract_id> --fields-file <path|-> [--format text|json]\n\nUpdates proposed fields on a server ContractDraft using structured JSON from a file or stdin. The command reads the Git-root .goalrail/project.yml marker, validates the stored login profile and Organization marker, sends project/repo expectations, and returns changed fields plus the next review action. It does not upload raw source bodies, submit or approve contracts, create WorkItems, run workers, gates, proof, or verification.\n"
}

func SubmitUsage() string {
	return "Usage: goalrail contract submit --contract-id <contract_id> [--format text|json]\n\nSubmits the current server ContractDraft for explicit user approval. The command reads the Git-root .goalrail/project.yml marker, validates the stored login profile and Organization marker, sends project/repo expectations, and returns the approval next action. It does not approve contracts, create WorkItems, run workers, gates, proof, or verification.\n"
}

func ApproveUsage() string {
	return "Usage: goalrail contract approve --contract-id <contract_id> --confirm-user-approval [--format text|json]\n\nApproves a submitted Contract only after explicit user approval. The command requires --confirm-user-approval, validates the Git-root .goalrail/project.yml marker plus login and Organization marker, and sends project/repo expectations. It does not create WorkItems, run workers, gates, proof, or verification.\n"
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

func refreshLocalRepoReceipt(ctx context.Context, facts projectscan.GitFacts, config projectconfig.Config, options Options) (spine.LocalRepoReceipt, error) {
	if strings.TrimSpace(config.RepoBindingID) == "" {
		return spine.LocalRepoReceipt{}, exitcode.ValidationError(errors.New("local .goalrail/project.yml is missing repo_binding_id; run goalrail init again"))
	}

	cache := projectscan.NewCache(options.CacheRoot)
	baseline, ok, err := cache.LoadLatestBaseline(config.RepoBindingID, facts.CanonicalRepoRoot)
	if err != nil {
		return spine.LocalRepoReceipt{}, exitcode.RuntimeError(fmt.Errorf("read project scan cache: %w", err))
	}

	needsRebuild := !ok || baseline.SchemaVersion != projectscan.SchemaVersion || baseline.HeadSHA != facts.HeadSHA
	rebuilt := false
	if needsRebuild {
		built, err := projectscan.BuildBaseline(ctx, facts.CanonicalRepoRoot, config.RepoBindingID, projectscan.DefaultBuildOptions())
		if err != nil {
			return spine.LocalRepoReceipt{}, exitcode.RuntimeError(fmt.Errorf("build project baseline: %w", err))
		}
		if err := cache.WriteBaseline(built); err != nil {
			return spine.LocalRepoReceipt{}, exitcode.RuntimeError(fmt.Errorf("write project baseline cache: %w", err))
		}
		baseline = built
		ok = true
		rebuilt = true
	}

	var baselinePtr *projectscan.RepositoryBaselineProfile
	if ok {
		baselinePtr = &baseline
	}
	overlay, rawStatus, err := projectscan.BuildOverlay(ctx, facts.CanonicalRepoRoot, config.RepoBindingID, baselinePtr, projectscan.OverlayOptions{Now: options.Now})
	if err != nil {
		return spine.LocalRepoReceipt{}, exitcode.RuntimeError(fmt.Errorf("refresh project overlay: %w", err))
	}
	if err := cache.WriteOverlay(overlay, rawStatus); err != nil {
		return spine.LocalRepoReceipt{}, exitcode.RuntimeError(fmt.Errorf("write project overlay cache: %w", err))
	}

	freshness := projectscan.EvaluateFreshness(facts.HeadSHA, baselinePtr, overlay)
	receipt := spine.LocalRepoReceipt{
		RepoBindingID:            spine.RepoBindingID(config.RepoBindingID),
		HeadSHA:                  facts.HeadSHA,
		OverlayID:                overlay.WorkspaceOverlayID,
		OverlayState:             overlay.State,
		Freshness:                freshness.Status,
		Dirty:                    overlayIsDirty(overlay),
		Partial:                  projectEvidenceIsPartial(baselinePtr, overlay, freshness),
		RawSourceUploaded:        false,
		BaselineRebuilt:          rebuilt,
		PartialReasons:           projectEvidencePartialReasons(baselinePtr, overlay),
		ScanCriticalChangedPaths: overlay.ScanCriticalChangedPaths,
		UnmergedPaths:            overlay.UnmergedPaths,
	}
	if baselinePtr != nil {
		receipt.BaselineID = baselinePtr.RepositoryBaselineProfileID
	}
	return receipt, nil
}

func overlayIsDirty(overlay projectscan.WorkspaceOverlay) bool {
	return overlay.State == projectscan.OverlayStateDirty ||
		overlay.State == projectscan.OverlayStateUnmerged ||
		len(overlay.ChangedPaths) > 0 ||
		len(overlay.ScanCriticalChangedPaths) > 0 ||
		len(overlay.UnmergedPaths) > 0
}

func projectEvidenceIsPartial(baseline *projectscan.RepositoryBaselineProfile, overlay projectscan.WorkspaceOverlay, freshness projectscan.FreshnessResult) bool {
	return freshness.Status == projectscan.FreshnessPartial ||
		overlay.State == projectscan.OverlayStatePartial ||
		len(overlay.PartialityReasons) > 0 ||
		(baseline != nil && (baseline.Partiality.SparseCheckout ||
			baseline.Partiality.ShallowRepository ||
			baseline.Partiality.SubmodulesPresent ||
			baseline.Partiality.Truncated ||
			len(baseline.Partiality.Reasons) > 0))
}

func projectEvidencePartialReasons(baseline *projectscan.RepositoryBaselineProfile, overlay projectscan.WorkspaceOverlay) []string {
	reasons := []string{}
	if baseline != nil {
		reasons = append(reasons, baseline.Partiality.Reasons...)
	}
	reasons = append(reasons, overlay.PartialityReasons...)
	return uniqueStrings(reasons)
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	result := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func getContractAggregate(ctx context.Context, client HTTPClient, session authstore.Session, contractID string) (contractAggregateResponse, error) {
	serverURL := strings.TrimRight(session.ServerURL, "/")
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, serverURL+"/v1/contracts/"+contractID, nil)
	if err != nil {
		return contractAggregateResponse{}, exitcode.RuntimeError(fmt.Errorf("build contract show request: %w", err))
	}
	request.Header.Set("Authorization", "Bearer "+session.AccessToken)

	response, err := client.Do(request)
	if err != nil {
		return contractAggregateResponse{}, exitcode.RuntimeError(fmt.Errorf("load contract from %s: %w", serverURL, err))
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return contractAggregateResponse{}, mapHTTPError("contract show request", response, serverURL)
	}
	var decoded contractAggregateResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return contractAggregateResponse{}, exitcode.RuntimeError(fmt.Errorf("decode contract show response: %w", err))
	}
	if strings.TrimSpace(string(decoded.ID)) == "" {
		return contractAggregateResponse{}, exitcode.RuntimeError(errors.New("contract show response did not include id"))
	}
	if strings.TrimSpace(string(decoded.State)) == "" {
		return contractAggregateResponse{}, exitcode.RuntimeError(errors.New("contract show response did not include state"))
	}
	return decoded, nil
}

func getContractCurrentDraft(ctx context.Context, client HTTPClient, session authstore.Session, contractID string) (contractDraftBodyResponse, error) {
	serverURL := strings.TrimRight(session.ServerURL, "/")
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, serverURL+"/v1/contracts/"+contractID+"/current-draft", nil)
	if err != nil {
		return contractDraftBodyResponse{}, exitcode.RuntimeError(fmt.Errorf("build contract current draft request: %w", err))
	}
	request.Header.Set("Authorization", "Bearer "+session.AccessToken)

	response, err := client.Do(request)
	if err != nil {
		return contractDraftBodyResponse{}, exitcode.RuntimeError(fmt.Errorf("load contract current draft from %s: %w", serverURL, err))
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return contractDraftBodyResponse{}, mapHTTPError("contract current draft request", response, serverURL)
	}
	var decoded contractDraftBodyResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return contractDraftBodyResponse{}, exitcode.RuntimeError(fmt.Errorf("decode contract current draft response: %w", err))
	}
	if strings.TrimSpace(decoded.ID) == "" {
		return contractDraftBodyResponse{}, exitcode.RuntimeError(errors.New("contract current draft response did not include id"))
	}
	if strings.TrimSpace(decoded.State) == "" {
		return contractDraftBodyResponse{}, exitcode.RuntimeError(errors.New("contract current draft response did not include state"))
	}
	return decoded, nil
}

func postContractDraft(ctx context.Context, client HTTPClient, session authstore.Session, goalID string, config projectconfig.Config) (contractDraftResponse, error) {
	payload := contractCreateRequest{
		GoalID:        goalID,
		ProjectID:     config.ProjectID,
		RepoBindingID: config.RepoBindingID,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return contractDraftResponse{}, exitcode.RuntimeError(fmt.Errorf("encode contract draft request: %w", err))
	}
	serverURL := strings.TrimRight(session.ServerURL, "/")
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, serverURL+"/v1/contracts", bytes.NewReader(body))
	if err != nil {
		return contractDraftResponse{}, exitcode.RuntimeError(fmt.Errorf("build contract draft request: %w", err))
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+session.AccessToken)

	response, err := client.Do(request)
	if err != nil {
		return contractDraftResponse{}, exitcode.RuntimeError(fmt.Errorf("create contract draft on %s: %w", serverURL, err))
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated && response.StatusCode != http.StatusOK {
		return contractDraftResponse{}, mapHTTPError("contract draft request", response, serverURL)
	}
	var decoded contractDraftResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return contractDraftResponse{}, exitcode.RuntimeError(fmt.Errorf("decode contract draft response: %w", err))
	}
	if strings.TrimSpace(string(decoded.ID)) == "" {
		return contractDraftResponse{}, exitcode.RuntimeError(errors.New("contract draft response did not include id"))
	}
	if strings.TrimSpace(string(decoded.State)) == "" {
		return contractDraftResponse{}, exitcode.RuntimeError(errors.New("contract draft response did not include state"))
	}
	return decoded, nil
}

func readContractUpdateRequest(workDir string, fieldsFile string, options Options) (contractUpdateRequest, error) {
	data, err := readContractUpdateData(workDir, fieldsFile, options.Stdin)
	if err != nil {
		return contractUpdateRequest{}, err
	}
	return decodeContractUpdateRequest(data)
}

func readContractUpdateData(workDir string, fieldsFile string, stdin io.Reader) ([]byte, error) {
	fieldsFile = strings.TrimSpace(fieldsFile)
	if fieldsFile == "-" {
		if stdin == nil {
			stdin = os.Stdin
		}
		data, err := io.ReadAll(io.LimitReader(stdin, maxContractUpdateBytes+1))
		if err != nil {
			return nil, exitcode.RuntimeError(fmt.Errorf("read contract update fields from stdin: %w", err))
		}
		if int64(len(data)) > maxContractUpdateBytes {
			return nil, exitcode.ValidationError(errors.New("contract update fields JSON is too large"))
		}
		return data, nil
	}

	filePath := resolvePath(workDir, fieldsFile)
	file, err := os.Open(filePath)
	if err != nil {
		return nil, exitcode.RuntimeError(fmt.Errorf("read contract update fields file %q: %w", fieldsFile, err))
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxContractUpdateBytes+1))
	if err != nil {
		return nil, exitcode.RuntimeError(fmt.Errorf("read contract update fields file %q: %w", fieldsFile, err))
	}
	if int64(len(data)) > maxContractUpdateBytes {
		return nil, exitcode.ValidationError(errors.New("contract update fields JSON is too large"))
	}
	return data, nil
}

func decodeContractUpdateRequest(data []byte) (contractUpdateRequest, error) {
	var raw map[string]json.RawMessage
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&raw); err != nil {
		return contractUpdateRequest{}, exitcode.ValidationError(fmt.Errorf("parse contract update fields JSON: %w", err))
	}
	if err := ensureSingleJSONValue(decoder); err != nil {
		return contractUpdateRequest{}, err
	}
	if len(raw) == 0 {
		return contractUpdateRequest{}, exitcode.ValidationError(errors.New("contract update fields must include at least one proposed field"))
	}

	request := contractUpdateRequest{
		Changes: map[string]json.RawMessage{},
	}
	for field, value := range raw {
		switch field {
		case "context_refs":
			refs, err := decodeContextRefs(value)
			if err != nil {
				return contractUpdateRequest{}, err
			}
			request.ContextRefs = refs
		case "unknowns":
			unknowns, err := decodeUnknowns(value)
			if err != nil {
				return contractUpdateRequest{}, err
			}
			request.Unknowns = unknowns
		case "proposed_verification":
			if _, exists := raw["proposed_expected_checks"]; exists {
				return contractUpdateRequest{}, exitcode.ValidationError(errors.New("proposed_verification cannot be combined with proposed_expected_checks"))
			}
			normalized, err := normalizeStringSliceField(field, value)
			if err != nil {
				return contractUpdateRequest{}, err
			}
			request.Changes["proposed_expected_checks"] = normalized
		default:
			kind, ok := editableContractUpdateFields[field]
			if !ok {
				return contractUpdateRequest{}, exitcode.ValidationError(fmt.Errorf("unknown contract update field %q", field))
			}
			normalized, err := normalizeContractUpdateField(field, kind, value)
			if err != nil {
				return contractUpdateRequest{}, err
			}
			request.Changes[field] = normalized
		}
	}

	if len(request.Changes) == 0 {
		return contractUpdateRequest{}, exitcode.ValidationError(errors.New("contract update fields must include at least one editable proposed field"))
	}
	request.changedFields = sortedMapKeys(request.Changes)
	return request, nil
}

func ensureSingleJSONValue(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return exitcode.ValidationError(errors.New("contract update fields JSON must contain one object"))
		}
		return exitcode.ValidationError(fmt.Errorf("parse contract update fields JSON: %w", err))
	}
	return nil
}

func normalizeContractUpdateField(field string, kind contractUpdateFieldKind, raw json.RawMessage) (json.RawMessage, error) {
	switch kind {
	case contractUpdateFieldString:
		return normalizeStringField(field, raw)
	case contractUpdateFieldStringSlice:
		return normalizeStringSliceField(field, raw)
	default:
		return nil, exitcode.ValidationError(fmt.Errorf("unsupported contract update field %q", field))
	}
}

func isJSONNull(raw json.RawMessage) bool {
	return strings.TrimSpace(string(raw)) == "null"
}

func normalizeStringField(field string, raw json.RawMessage) (json.RawMessage, error) {
	if isJSONNull(raw) {
		return nil, exitcode.ValidationError(fmt.Errorf("%s must not be null", field))
	}
	var value string
	if err := decodeStrictRaw(raw, &value); err != nil {
		return nil, exitcode.ValidationError(fmt.Errorf("%s must be a string", field))
	}
	if strings.TrimSpace(value) == "" {
		return nil, exitcode.ValidationError(fmt.Errorf("%s must not be blank", field))
	}
	normalized, err := json.Marshal(value)
	if err != nil {
		return nil, exitcode.RuntimeError(fmt.Errorf("encode %s: %w", field, err))
	}
	return normalized, nil
}

func normalizeStringSliceField(field string, raw json.RawMessage) (json.RawMessage, error) {
	if isJSONNull(raw) {
		return nil, exitcode.ValidationError(fmt.Errorf("%s must not be null", field))
	}
	var values []string
	if err := decodeStrictRaw(raw, &values); err != nil {
		return nil, exitcode.ValidationError(fmt.Errorf("%s must be an array of strings", field))
	}
	if len(values) == 0 {
		return nil, exitcode.ValidationError(fmt.Errorf("%s must include at least one value", field))
	}
	for i, value := range values {
		if strings.TrimSpace(value) == "" {
			return nil, exitcode.ValidationError(fmt.Errorf("%s[%d] must not be blank", field, i))
		}
	}
	normalized, err := json.Marshal(values)
	if err != nil {
		return nil, exitcode.RuntimeError(fmt.Errorf("encode %s: %w", field, err))
	}
	return normalized, nil
}

func decodeContextRefs(raw json.RawMessage) ([]contractUpdateContextRef, error) {
	if isJSONNull(raw) {
		return nil, exitcode.ValidationError(errors.New("context_refs must not be null"))
	}
	var refs []contractUpdateContextRef
	if err := decodeStrictRaw(raw, &refs); err != nil {
		return nil, exitcode.ValidationError(errors.New("context_refs must be an array of structured refs"))
	}
	for i, ref := range refs {
		if strings.TrimSpace(ref.Kind) == "" {
			return nil, exitcode.ValidationError(fmt.Errorf("context_refs[%d].kind is required", i))
		}
		if strings.TrimSpace(ref.ID) == "" {
			return nil, exitcode.ValidationError(fmt.Errorf("context_refs[%d].id is required", i))
		}
	}
	return refs, nil
}

func decodeUnknowns(raw json.RawMessage) ([]string, error) {
	if isJSONNull(raw) {
		return nil, exitcode.ValidationError(errors.New("unknowns must not be null"))
	}
	var unknowns []string
	if err := decodeStrictRaw(raw, &unknowns); err != nil {
		return nil, exitcode.ValidationError(errors.New("unknowns must be an array of strings"))
	}
	for i, value := range unknowns {
		if strings.TrimSpace(value) == "" {
			return nil, exitcode.ValidationError(fmt.Errorf("unknowns[%d] must not be blank", i))
		}
	}
	return unknowns, nil
}

func decodeStrictRaw(raw json.RawMessage, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("extra JSON value")
		}
		return err
	}
	return nil
}

func sortedMapKeys(values map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func actorRefForProfile(profile meResponse) contractUpdateActorRef {
	return contractUpdateActorRef{
		Kind:        "user",
		ID:          profile.User.ID,
		DisplayName: profile.User.DisplayName,
	}
}

func patchContractUpdate(ctx context.Context, client HTTPClient, session authstore.Session, contractID string, update contractUpdateRequest) (contractDraftResponse, error) {
	body, err := json.Marshal(update)
	if err != nil {
		return contractDraftResponse{}, exitcode.RuntimeError(fmt.Errorf("encode contract update request: %w", err))
	}
	serverURL := strings.TrimRight(session.ServerURL, "/")
	request, err := http.NewRequestWithContext(ctx, http.MethodPatch, serverURL+"/v1/contracts/"+contractID, bytes.NewReader(body))
	if err != nil {
		return contractDraftResponse{}, exitcode.RuntimeError(fmt.Errorf("build contract update request: %w", err))
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+session.AccessToken)

	response, err := client.Do(request)
	if err != nil {
		return contractDraftResponse{}, exitcode.RuntimeError(fmt.Errorf("update contract draft on %s: %w", serverURL, err))
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return contractDraftResponse{}, mapHTTPError("contract update request", response, serverURL)
	}
	var decoded contractDraftResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return contractDraftResponse{}, exitcode.RuntimeError(fmt.Errorf("decode contract update response: %w", err))
	}
	if strings.TrimSpace(string(decoded.ID)) == "" {
		return contractDraftResponse{}, exitcode.RuntimeError(errors.New("contract update response did not include id"))
	}
	if strings.TrimSpace(string(decoded.State)) == "" {
		return contractDraftResponse{}, exitcode.RuntimeError(errors.New("contract update response did not include state"))
	}
	return decoded, nil
}

func postContractSubmit(ctx context.Context, client HTTPClient, session authstore.Session, contractID string, config projectconfig.Config) (contractDraftResponse, error) {
	payload := contractTransitionRequest{
		ProjectID:     config.ProjectID,
		RepoBindingID: config.RepoBindingID,
	}
	return postContractTransition(ctx, client, session, contractID, "submissions", payload, http.StatusOK, "contract submit request")
}

func postContractApprove(ctx context.Context, client HTTPClient, session authstore.Session, contractID string, config projectconfig.Config) (contractDraftResponse, error) {
	payload := contractTransitionRequest{
		ProjectID:     config.ProjectID,
		RepoBindingID: config.RepoBindingID,
	}
	return postContractTransition(ctx, client, session, contractID, "approvals", payload, http.StatusCreated, "contract approve request")
}

func postContractTransition(ctx context.Context, client HTTPClient, session authstore.Session, contractID string, path string, payload contractTransitionRequest, successStatus int, operation string) (contractDraftResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return contractDraftResponse{}, exitcode.RuntimeError(fmt.Errorf("encode %s: %w", operation, err))
	}
	serverURL := strings.TrimRight(session.ServerURL, "/")
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, serverURL+"/v1/contracts/"+contractID+"/"+path, bytes.NewReader(body))
	if err != nil {
		return contractDraftResponse{}, exitcode.RuntimeError(fmt.Errorf("build %s: %w", operation, err))
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+session.AccessToken)

	response, err := client.Do(request)
	if err != nil {
		return contractDraftResponse{}, exitcode.RuntimeError(fmt.Errorf("run %s on %s: %w", operation, serverURL, err))
	}
	defer response.Body.Close()

	if response.StatusCode != successStatus {
		return contractDraftResponse{}, mapHTTPError(operation, response, serverURL)
	}
	var decoded contractDraftResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return contractDraftResponse{}, exitcode.RuntimeError(fmt.Errorf("decode %s response: %w", operation, err))
	}
	if strings.TrimSpace(string(decoded.ID)) == "" {
		return contractDraftResponse{}, exitcode.RuntimeError(fmt.Errorf("%s response did not include id", operation))
	}
	if strings.TrimSpace(string(decoded.State)) == "" {
		return contractDraftResponse{}, exitcode.RuntimeError(fmt.Errorf("%s response did not include state", operation))
	}
	return decoded, nil
}

func validateContractDraftContext(config projectconfig.Config, goalID string, contractDraft contractDraftResponse) error {
	if contractDraft.GoalID != "" && contractDraft.GoalID != goalID {
		return exitcode.ValidationError(errors.New("contract draft response goal_id does not match requested Goal"))
	}
	if contractDraft.RepoBindingID != "" && string(contractDraft.RepoBindingID) != config.RepoBindingID {
		return exitcode.ValidationError(errors.New("contract draft response repo_binding_id does not match local .goalrail/project.yml; run this command from the repository bound to the Goal"))
	}
	return nil
}

func validateContractUpdateContext(config projectconfig.Config, contractID string, contractResponse contractDraftResponse) error {
	if string(contractResponse.ID) != contractID {
		return exitcode.ValidationError(errors.New("contract update response id does not match requested Contract"))
	}
	if contractResponse.RepoBindingID != "" && string(contractResponse.RepoBindingID) != config.RepoBindingID {
		return exitcode.ValidationError(errors.New("contract update response repo_binding_id does not match local .goalrail/project.yml; run this command from the repository bound to the Contract"))
	}
	return nil
}

func validateContractAggregateContext(config projectconfig.Config, contractID string, contractResponse contractAggregateResponse) error {
	if string(contractResponse.ID) != contractID {
		return exitcode.ValidationError(errors.New("contract show response id does not match requested Contract"))
	}
	if contractResponse.RepoBindingID != "" && string(contractResponse.RepoBindingID) != config.RepoBindingID {
		return exitcode.ValidationError(errors.New("contract show response repo_binding_id does not match local .goalrail/project.yml; run this command from the repository bound to the Contract"))
	}
	return nil
}

func validateContractDraftBodyContext(config projectconfig.Config, contractID string, currentDraftID string, draft contractDraftBodyResponse) error {
	if draft.ID != currentDraftID {
		return exitcode.ValidationError(errors.New("contract current draft response id does not match public Contract current_draft_id"))
	}
	if draft.ContractID != "" && draft.ContractID != contractID {
		return exitcode.ValidationError(errors.New("contract current draft response contract_id does not match requested Contract"))
	}
	if draft.RepoBindingID != "" && string(draft.RepoBindingID) != config.RepoBindingID {
		return exitcode.ValidationError(errors.New("contract current draft response repo_binding_id does not match local .goalrail/project.yml; run this command from the repository bound to the Contract"))
	}
	return nil
}

func buildShowOutput(config projectconfig.Config, serverURL string, contractResponse contractAggregateResponse, draft *contractDraftBodyResponse) spine.ContractShowOutput {
	nextAction := contractShowNextAction(contractResponse)
	summary := fmt.Sprintf("Contract %s is %s.", contractResponse.ID, contractResponse.State)
	if draft != nil {
		summary = fmt.Sprintf("Contract %s is %s and current draft review fields are loaded.", contractResponse.ID, contractResponse.State)
	}
	output := spine.ContractShowOutput{
		SchemaVersion:   cliSchemaVersion,
		Mode:            serverMode,
		ServerURL:       serverURL,
		OrganizationID:  config.OrganizationID,
		ProjectID:       config.ProjectID,
		RepoBindingID:   contractResponse.RepoBindingID,
		GoalID:          contractResponse.GoalID,
		ContractID:      contractResponse.ID,
		ContractState:   contractResponse.State,
		CurrentSeedID:   contractResponse.CurrentSeedID,
		CurrentDraftID:  contractResponse.CurrentDraftID,
		LocalConfigPath: projectconfig.RelativePath,
		Display: spine.DisplaySummary{
			Summary: summary,
		},
		NextAction: nextAction,
	}
	if output.RepoBindingID == "" {
		output.RepoBindingID = spine.RepoBindingID(config.RepoBindingID)
	}
	if draft != nil {
		output.CurrentDraft = &spine.ContractShowDraft{
			ID:                         draft.ID,
			State:                      draft.State,
			Title:                      draft.Title,
			IntentSummary:              draft.IntentSummary,
			ProposedScope:              append([]string{}, draft.ProposedScope...),
			ProposedNonGoals:           append([]string{}, draft.ProposedNonGoals...),
			ProposedConstraints:        append([]string{}, draft.ProposedConstraints...),
			ProposedAcceptanceCriteria: append([]string{}, draft.ProposedAcceptanceCriteria...),
			ProposedExpectedChecks:     append([]string{}, draft.ProposedExpectedChecks...),
			ProposedProofExpectations:  append([]string{}, draft.ProposedProofExpectations...),
			RiskHints:                  append([]string{}, draft.RiskHints...),
		}
	}
	return output
}

func contractShowNextAction(contractResponse contractAggregateResponse) spine.NextAction {
	switch contractResponse.State {
	case spine.ContractStateDraft:
		return spine.NextAction{
			Kind:      "review_or_update_contract",
			Blocking:  true,
			Available: true,
			Command:   fmt.Sprintf("goalrail contract update --contract-id %s --fields-file - --format json", contractResponse.ID),
		}
	case spine.ContractStateReadyForApproval:
		return spine.NextAction{
			Kind:      "approve_contract",
			Blocking:  true,
			Available: true,
			Command:   fmt.Sprintf("goalrail contract approve --contract-id %s --confirm-user-approval --format json", contractResponse.ID),
		}
	case spine.ContractStateApproved:
		return spine.NextAction{
			Kind:      "plan_work",
			Blocking:  false,
			Available: true,
			Command:   fmt.Sprintf("goalrail work plan --contract-id %s --format json", contractResponse.ID),
		}
	default:
		return spine.NextAction{
			Kind:      "review_contract",
			Blocking:  true,
			Available: false,
		}
	}
}

func buildDraftOutput(config projectconfig.Config, serverURL string, goalID string, contractDraft contractDraftResponse, receipt spine.LocalRepoReceipt) spine.ContractDraftOutput {
	summary := fmt.Sprintf("Found a Contract handle in state %s. Contract update is only available while the Contract is draft.", contractDraft.State)
	nextAction := spine.NextAction{
		Kind:      "update_contract",
		Blocking:  false,
		Available: false,
	}
	if contractDraft.State == spine.ContractStateDraft {
		summary = "Created or found a draft Contract handle. Local repository receipt is attached; update proposed contract fields next."
		nextAction.Available = true
		nextAction.Command = fmt.Sprintf("goalrail contract update --contract-id %s --fields-file - --format json", contractDraft.ID)
	}
	return spine.ContractDraftOutput{
		SchemaVersion:    cliSchemaVersion,
		Mode:             serverMode,
		ServerURL:        serverURL,
		OrganizationID:   config.OrganizationID,
		ProjectID:        config.ProjectID,
		RepoBindingID:    spine.RepoBindingID(config.RepoBindingID),
		GoalID:           goalID,
		ContractID:       contractDraft.ID,
		ContractState:    contractDraft.State,
		LocalRepoReceipt: receipt,
		LocalConfigPath:  projectconfig.RelativePath,
		Display: spine.DisplaySummary{
			Summary: summary,
		},
		NextAction: nextAction,
	}
}

func buildUpdateOutput(config projectconfig.Config, serverURL string, contractResponse contractDraftResponse, changedFields []string) spine.ContractUpdateOutput {
	return spine.ContractUpdateOutput{
		SchemaVersion:   cliSchemaVersion,
		Mode:            serverMode,
		ServerURL:       serverURL,
		OrganizationID:  config.OrganizationID,
		ProjectID:       config.ProjectID,
		RepoBindingID:   spine.RepoBindingID(config.RepoBindingID),
		ContractID:      contractResponse.ID,
		ContractState:   contractResponse.State,
		ChangedFields:   append([]string{}, changedFields...),
		LocalConfigPath: projectconfig.RelativePath,
		Display: spine.DisplaySummary{
			Summary: "Updated proposed ContractDraft fields. Review the draft contract next.",
		},
		NextAction: spine.NextAction{
			Kind:      "review_contract",
			Blocking:  true,
			Available: true,
		},
	}
}

func buildSubmitOutput(config projectconfig.Config, serverURL string, contractResponse contractDraftResponse) spine.ContractTransitionOutput {
	return spine.ContractTransitionOutput{
		SchemaVersion:   cliSchemaVersion,
		Mode:            serverMode,
		ServerURL:       serverURL,
		OrganizationID:  config.OrganizationID,
		ProjectID:       config.ProjectID,
		RepoBindingID:   spine.RepoBindingID(config.RepoBindingID),
		ContractID:      contractResponse.ID,
		ContractState:   contractResponse.State,
		LocalConfigPath: projectconfig.RelativePath,
		Display: spine.DisplaySummary{
			Summary: "Submitted Contract for explicit user approval. Ask the user to approve before calling approve.",
		},
		NextAction: spine.NextAction{
			Kind:      "approve_contract",
			Blocking:  true,
			Available: true,
			Command:   fmt.Sprintf("goalrail contract approve --contract-id %s --confirm-user-approval --format json", contractResponse.ID),
		},
	}
}

func buildApproveOutput(config projectconfig.Config, serverURL string, contractResponse contractDraftResponse) spine.ContractTransitionOutput {
	return spine.ContractTransitionOutput{
		SchemaVersion:   cliSchemaVersion,
		Mode:            serverMode,
		ServerURL:       serverURL,
		OrganizationID:  config.OrganizationID,
		ProjectID:       config.ProjectID,
		RepoBindingID:   spine.RepoBindingID(config.RepoBindingID),
		ContractID:      contractResponse.ID,
		ContractState:   contractResponse.State,
		LocalConfigPath: projectconfig.RelativePath,
		Display: spine.DisplaySummary{
			Summary: "Approved Contract snapshot created. Create the queued WorkItemPlan next.",
		},
		NextAction: spine.NextAction{
			Kind:      "plan_work",
			Blocking:  false,
			Available: true,
			Command:   fmt.Sprintf("goalrail work plan --contract-id %s --format json", contractResponse.ID),
		},
	}
}

func renderShowText(output spine.ContractShowOutput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Contract review\n\n")
	fmt.Fprintf(&b, "Contract identity\n")
	fmt.Fprintf(&b, "- contract_id: %s\n", output.ContractID)
	fmt.Fprintf(&b, "- state: %s\n", output.ContractState)
	fmt.Fprintf(&b, "- goal_id: %s\n", output.GoalID)
	fmt.Fprintf(&b, "- repo_binding_id: %s\n", output.RepoBindingID)
	if output.CurrentSeedID != "" {
		fmt.Fprintf(&b, "- current_seed_id: %s\n", output.CurrentSeedID)
	}
	if output.CurrentDraftID != "" {
		fmt.Fprintf(&b, "- current_draft_id: %s\n", output.CurrentDraftID)
	}
	if output.LocalConfigPath != "" {
		fmt.Fprintf(&b, "- local_config: %s\n", output.LocalConfigPath)
	}
	b.WriteString("\n")

	if output.CurrentDraft == nil {
		b.WriteString("Current draft\n")
		b.WriteString("No current draft body is available for this Contract.\n")
	} else {
		writeShowStringSection(&b, "Title", output.CurrentDraft.Title)
		writeShowStringSection(&b, "Intent summary", output.CurrentDraft.IntentSummary)
		writeShowListSection(&b, "Proposed scope", output.CurrentDraft.ProposedScope)
		writeShowListSection(&b, "Proposed non-goals", output.CurrentDraft.ProposedNonGoals)
		writeShowListSection(&b, "Proposed constraints", output.CurrentDraft.ProposedConstraints)
		writeShowListSection(&b, "Proposed acceptance criteria", output.CurrentDraft.ProposedAcceptanceCriteria)
		writeShowListSection(&b, "Proposed expected checks", output.CurrentDraft.ProposedExpectedChecks)
		writeShowListSection(&b, "Proposed proof expectations", output.CurrentDraft.ProposedProofExpectations)
		writeShowListSection(&b, "Risk hints", output.CurrentDraft.RiskHints)
	}

	b.WriteString("\nApproval note\n")
	b.WriteString("This command is read-only and does not approve or mutate the Contract.\n")
	b.WriteString("\nNext\n")
	switch output.ContractState {
	case spine.ContractStateReadyForApproval:
		fmt.Fprintf(&b, "Human approval is required before running: goalrail contract approve --contract-id %s --confirm-user-approval\n", output.ContractID)
	case spine.ContractStateDraft:
		fmt.Fprintf(&b, "Review the draft, update it if needed, then submit with: goalrail contract submit --contract-id %s --format json\n", output.ContractID)
	case spine.ContractStateApproved:
		fmt.Fprintf(&b, "The Contract is already approved. The next normal stage is planning: goalrail work plan --contract-id %s --format json\n", output.ContractID)
	default:
		fmt.Fprintf(&b, "Review current Contract state before choosing the next lifecycle command.\n")
	}
	return b.String()
}

func writeShowStringSection(b *strings.Builder, title string, value string) {
	fmt.Fprintf(b, "%s\n", title)
	value = strings.TrimSpace(value)
	if value == "" {
		b.WriteString("(empty)\n\n")
		return
	}
	fmt.Fprintf(b, "%s\n\n", value)
}

func writeShowListSection(b *strings.Builder, title string, values []string) {
	fmt.Fprintf(b, "%s\n", title)
	if len(values) == 0 {
		b.WriteString("- (none)\n\n")
		return
	}
	for _, value := range values {
		fmt.Fprintf(b, "- %s\n", value)
	}
	b.WriteString("\n")
}

func renderDraftText(output spine.ContractDraftOutput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Contract draft handle\n\n")
	fmt.Fprintf(&b, "Server: %s\n", output.ServerURL)
	fmt.Fprintf(&b, "Project: %s\n", output.ProjectID)
	fmt.Fprintf(&b, "Repo binding: %s\n", output.RepoBindingID)
	fmt.Fprintf(&b, "Goal: %s\n", output.GoalID)
	fmt.Fprintf(&b, "Contract: %s\n", output.ContractID)
	fmt.Fprintf(&b, "State: %s\n", output.ContractState)
	fmt.Fprintf(&b, "HEAD: %s\n", output.LocalRepoReceipt.HeadSHA)
	if output.LocalRepoReceipt.BaselineID != "" {
		fmt.Fprintf(&b, "Baseline: %s\n", output.LocalRepoReceipt.BaselineID)
	}
	fmt.Fprintf(&b, "Overlay: %s\n", output.LocalRepoReceipt.OverlayID)
	fmt.Fprintf(&b, "Freshness: %s\n", output.LocalRepoReceipt.Freshness)
	fmt.Fprintf(&b, "Dirty: %t\n", output.LocalRepoReceipt.Dirty)
	fmt.Fprintf(&b, "Partial: %t\n", output.LocalRepoReceipt.Partial)
	if output.LocalConfigPath != "" {
		fmt.Fprintf(&b, "Local config: %s\n", output.LocalConfigPath)
	}
	fmt.Fprintf(&b, "\n%s\n", output.Display.Summary)
	if output.NextAction.Command != "" && output.NextAction.Available {
		fmt.Fprintf(&b, "\nNext: %s\n", output.NextAction.Command)
	}
	if output.NextAction.Command != "" && !output.NextAction.Available {
		fmt.Fprintf(&b, "\nNext planned command, not available yet: %s\n", output.NextAction.Command)
	}
	if output.NextAction.PlannedSlice != "" {
		fmt.Fprintf(&b, "Planned slice: %s\n", output.NextAction.PlannedSlice)
	}
	return b.String()
}

func renderUpdateText(output spine.ContractUpdateOutput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Contract draft updated\n\n")
	fmt.Fprintf(&b, "Server: %s\n", output.ServerURL)
	fmt.Fprintf(&b, "Project: %s\n", output.ProjectID)
	fmt.Fprintf(&b, "Repo binding: %s\n", output.RepoBindingID)
	fmt.Fprintf(&b, "Contract: %s\n", output.ContractID)
	fmt.Fprintf(&b, "State: %s\n", output.ContractState)
	if len(output.ChangedFields) > 0 {
		b.WriteString("Changed fields:\n")
		for _, field := range output.ChangedFields {
			fmt.Fprintf(&b, "- %s\n", field)
		}
	}
	if output.LocalConfigPath != "" {
		fmt.Fprintf(&b, "Local config: %s\n", output.LocalConfigPath)
	}
	fmt.Fprintf(&b, "\n%s\n", output.Display.Summary)
	b.WriteString("\nNext: review the draft contract with the user.\n")
	return b.String()
}

func renderSubmitText(output spine.ContractTransitionOutput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Contract submitted for approval\n\n")
	fmt.Fprintf(&b, "Server: %s\n", output.ServerURL)
	fmt.Fprintf(&b, "Project: %s\n", output.ProjectID)
	fmt.Fprintf(&b, "Repo binding: %s\n", output.RepoBindingID)
	fmt.Fprintf(&b, "Contract: %s\n", output.ContractID)
	fmt.Fprintf(&b, "State: %s\n", output.ContractState)
	if output.LocalConfigPath != "" {
		fmt.Fprintf(&b, "Local config: %s\n", output.LocalConfigPath)
	}
	fmt.Fprintf(&b, "\n%s\n", output.Display.Summary)
	if output.NextAction.Command != "" && output.NextAction.Available {
		fmt.Fprintf(&b, "\nNext: %s\n", output.NextAction.Command)
	}
	return b.String()
}

func renderApproveText(output spine.ContractTransitionOutput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Contract approved\n\n")
	fmt.Fprintf(&b, "Server: %s\n", output.ServerURL)
	fmt.Fprintf(&b, "Project: %s\n", output.ProjectID)
	fmt.Fprintf(&b, "Repo binding: %s\n", output.RepoBindingID)
	fmt.Fprintf(&b, "Contract: %s\n", output.ContractID)
	fmt.Fprintf(&b, "State: %s\n", output.ContractState)
	if output.LocalConfigPath != "" {
		fmt.Fprintf(&b, "Local config: %s\n", output.LocalConfigPath)
	}
	fmt.Fprintf(&b, "\n%s\n", output.Display.Summary)
	if output.NextAction.Command != "" && output.NextAction.Available {
		fmt.Fprintf(&b, "\nNext: %s\n", output.NextAction.Command)
	}
	if output.NextAction.Command != "" && !output.NextAction.Available {
		fmt.Fprintf(&b, "\nNext planned command, not available yet: %s\n", output.NextAction.Command)
		if output.NextAction.PlannedSlice != "" {
			fmt.Fprintf(&b, "Planned slice: %s\n", output.NextAction.PlannedSlice)
		}
	}
	return b.String()
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

type serverErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type contractCreateRequest struct {
	GoalID        string `json:"goal_id"`
	ProjectID     string `json:"project_id"`
	RepoBindingID string `json:"repo_binding_id"`
}

type contractUpdateRequest struct {
	ProjectID     string                     `json:"project_id"`
	RepoBindingID string                     `json:"repo_binding_id"`
	UpdatedBy     contractUpdateActorRef     `json:"updated_by"`
	Changes       map[string]json.RawMessage `json:"changes"`
	ContextRefs   []contractUpdateContextRef `json:"context_refs,omitempty"`
	Unknowns      []string                   `json:"unknowns,omitempty"`
	changedFields []string
}

type contractTransitionRequest struct {
	ProjectID     string `json:"project_id"`
	RepoBindingID string `json:"repo_binding_id"`
}

type contractAggregateResponse struct {
	ID             spine.ContractID    `json:"id"`
	RepoBindingID  spine.RepoBindingID `json:"repo_binding_id"`
	GoalID         string              `json:"goal_id"`
	State          spine.ContractState `json:"state"`
	CurrentSeedID  string              `json:"current_seed_id,omitempty"`
	CurrentDraftID string              `json:"current_draft_id,omitempty"`
}

type contractDraftBodyResponse struct {
	ID                         string              `json:"id"`
	ContractID                 string              `json:"contract_id"`
	ContractSeedID             string              `json:"contract_seed_id"`
	GoalID                     string              `json:"goal_id"`
	RepoBindingID              spine.RepoBindingID `json:"repo_binding_id"`
	Title                      string              `json:"title"`
	IntentSummary              string              `json:"intent_summary"`
	ProposedScope              []string            `json:"proposed_scope"`
	ProposedNonGoals           []string            `json:"proposed_non_goals"`
	ProposedConstraints        []string            `json:"proposed_constraints"`
	ProposedAcceptanceCriteria []string            `json:"proposed_acceptance_criteria"`
	ProposedExpectedChecks     []string            `json:"proposed_expected_checks"`
	ProposedProofExpectations  []string            `json:"proposed_proof_expectations"`
	RiskHints                  []string            `json:"risk_hints"`
	State                      string              `json:"state"`
}

type contractUpdateActorRef struct {
	Kind        string `json:"kind"`
	ID          string `json:"id"`
	DisplayName string `json:"display_name,omitempty"`
}

type contractUpdateContextRef struct {
	Kind       string `json:"kind"`
	ID         string `json:"id"`
	BaselineID string `json:"baseline_id,omitempty"`
	OverlayID  string `json:"overlay_id,omitempty"`
}

type contractUpdateFieldKind string

const (
	contractUpdateFieldString      contractUpdateFieldKind = "string"
	contractUpdateFieldStringSlice contractUpdateFieldKind = "string_slice"
)

var editableContractUpdateFields = map[string]contractUpdateFieldKind{
	"title":                        contractUpdateFieldString,
	"intent_summary":               contractUpdateFieldString,
	"proposed_scope":               contractUpdateFieldStringSlice,
	"proposed_non_goals":           contractUpdateFieldStringSlice,
	"proposed_constraints":         contractUpdateFieldStringSlice,
	"proposed_acceptance_criteria": contractUpdateFieldStringSlice,
	"proposed_expected_checks":     contractUpdateFieldStringSlice,
	"proposed_proof_expectations":  contractUpdateFieldStringSlice,
	"risk_hints":                   contractUpdateFieldStringSlice,
}

type contractDraftResponse struct {
	ID             spine.ContractID    `json:"id"`
	RepoBindingID  spine.RepoBindingID `json:"repo_binding_id"`
	GoalID         string              `json:"goal_id"`
	State          spine.ContractState `json:"state"`
	CurrentSeedID  string              `json:"current_seed_id,omitempty"`
	CurrentDraftID string              `json:"current_draft_id,omitempty"`
}
