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
	"strings"
	"time"

	"github.com/heurema/goalrail/apps/cli/internal/authstore"
	"github.com/heurema/goalrail/apps/cli/internal/contract"
	"github.com/heurema/goalrail/apps/cli/internal/exitcode"
	"github.com/heurema/goalrail/apps/cli/internal/projectconfig"
	"github.com/heurema/goalrail/apps/cli/internal/projectscan"
	"github.com/heurema/goalrail/apps/cli/internal/spine"
	"github.com/heurema/goalrail/apps/cli/internal/term"
)

const (
	serverMode       = "server"
	cliSchemaVersion = "goalrail.cli.v1"
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
	case "draft":
		return runDraft(ctx, out, workDir, args[1:], options)
	default:
		return exitcode.UsageError(fmt.Errorf("unknown contract command %q", args[0]))
	}
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
	return "Usage: goalrail contract <command> [options]\n\nCommands:\n  draft       create or return a server Contract draft handle for a ready Goal\n  validate    validate a contract JSON file\n\nRun goalrail contract <command> --help for command usage.\n"
}

func ValidateUsage() string {
	return "Usage: goalrail contract validate --file <contract.json> [--format text|json]\n\nValidates the minimum contract fields needed before approval or execution.\n"
}

func DraftUsage() string {
	return "Usage: goalrail contract draft --goal-id <goal_id> [--format text|json]\n\nCreates or returns a server Contract draft handle for a ready Goal using the current Git root .goalrail/project.yml marker and the stored goalrail login profile. It refreshes local Project Scan evidence and returns a local repository receipt. It does not upload raw source bodies, update contract fields, create WorkItems, run workers, gates, proof, or verification.\n"
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

func validateContractDraftContext(config projectconfig.Config, goalID string, contractDraft contractDraftResponse) error {
	if contractDraft.GoalID != "" && contractDraft.GoalID != goalID {
		return exitcode.ValidationError(errors.New("contract draft response goal_id does not match requested Goal"))
	}
	if contractDraft.RepoBindingID != "" && string(contractDraft.RepoBindingID) != config.RepoBindingID {
		return exitcode.ValidationError(errors.New("contract draft response repo_binding_id does not match local .goalrail/project.yml; run this command from the repository bound to the Goal"))
	}
	return nil
}

func buildDraftOutput(config projectconfig.Config, serverURL string, goalID string, contractDraft contractDraftResponse, receipt spine.LocalRepoReceipt) spine.ContractDraftOutput {
	command := fmt.Sprintf("goalrail contract update --contract-id %s --fields-file - --format json", contractDraft.ID)
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
			Summary: "Created or found a draft Contract handle. Local repository receipt is attached; contract field updates are a later slice.",
		},
		NextAction: spine.NextAction{
			Kind:         "update_contract",
			Blocking:     false,
			Command:      command,
			Available:    false,
			PlannedSlice: "E",
		},
	}
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
	if output.NextAction.Command != "" {
		fmt.Fprintf(&b, "\nNext planned command, not available yet: %s\n", output.NextAction.Command)
	}
	if output.NextAction.PlannedSlice != "" {
		fmt.Fprintf(&b, "Planned slice: %s\n", output.NextAction.PlannedSlice)
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

type contractDraftResponse struct {
	ID             spine.ContractID    `json:"id"`
	RepoBindingID  spine.RepoBindingID `json:"repo_binding_id"`
	GoalID         string              `json:"goal_id"`
	State          spine.ContractState `json:"state"`
	CurrentSeedID  string              `json:"current_seed_id,omitempty"`
	CurrentDraftID string              `json:"current_draft_id,omitempty"`
}
