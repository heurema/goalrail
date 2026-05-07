package initcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/heurema/goalrail/apps/cli/internal/authstore"
	"github.com/heurema/goalrail/apps/cli/internal/exitcode"
	"github.com/heurema/goalrail/apps/cli/internal/projectscan"
	"github.com/heurema/goalrail/apps/cli/internal/spine"
	"github.com/heurema/goalrail/apps/cli/internal/term"
)

const (
	serverMode             = "server"
	nextSuggestedCommand   = "goalrail work start --title <title>"
	serverRegistrationNote = "This registered repository metadata on the GoalRail server, wrote a non-secret GoalRail repository marker, and ran a local Project Scan.\nNo server clone, audit, hooks, branch creation, deploy keys, provider integration, runner, gate, proof, or verification were configured."
	repositoryContextNote  = "This initialized GoalRail repository context for your existing organization, recorded a metadata-only repository context snapshot, and ran a local Project Scan.\nNo server clone, audit, hooks, branch creation, deploy keys, provider integration, runner, gate, proof, or verification were configured."

	defaultInitHTTPTimeout   = 30 * time.Second
	defaultInitRetryAttempts = 3
	defaultInitRetryBackoff  = 10 * time.Millisecond
)

type serverErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func runServerBackedInit(ctx context.Context, out *term.Output, draft spine.RepoBindingDraft, projectID string, format term.Format, options Options) error {
	if err := validateServerBackedDraft(draft); err != nil {
		return err
	}

	session, serverURL, err := loadUsableSession(options)
	if err != nil {
		return err
	}
	if err := preflightProjectConfig(draft.GitRoot, projectConfig{
		ServerURL: serverURL,
		ProjectID: projectID,
		Repository: projectConfigRepository{
			Provider:           draft.Provider,
			FullName:           draft.RepositoryFullName,
			URL:                draft.RepoURL,
			WorkflowBaseBranch: draft.WorkflowBaseBranch,
		},
	}); err != nil {
		return err
	}

	client := initHTTPClient(options.HTTPClient)

	requestPayload := spine.RepoBindingInitRequest{
		Provider:              draft.Provider,
		RepositoryFullName:    draft.RepositoryFullName,
		RepositoryURL:         draft.RepoURL,
		ProviderDefaultBranch: draft.ProviderDefaultBranch,
		WorkflowBaseBranch:    draft.WorkflowBaseBranch,
		LocalRemoteName:       draft.RemoteName,
		LocalHeadSHA:          draft.HeadSHA,
	}
	responsePayload, err := postRepoBindingInit(ctx, client, session, projectID, requestPayload)
	if err != nil {
		return err
	}

	output := spine.RepoBindingInitOutput{
		Mode:                  serverMode,
		ServerURL:             serverURL,
		ProjectID:             responsePayload.ProjectID,
		RepoBindingID:         responsePayload.RepoBindingID,
		OrganizationID:        responsePayload.OrganizationID,
		Provider:              responsePayload.Provider,
		RepositoryFullName:    responsePayload.RepositoryFullName,
		RepositoryURL:         responsePayload.RepositoryURL,
		ProviderDefaultBranch: responsePayload.ProviderDefaultBranch,
		WorkflowBaseBranch:    responsePayload.WorkflowBaseBranch,
		State:                 responsePayload.State,
		Created:               responsePayload.Created,
		Message:               responsePayload.Message,
		NextCommand:           nextSuggestedCommand,
	}
	if output.ProjectID == "" {
		output.ProjectID = projectID
	}
	configStatus, err := writeProjectConfig(draft.GitRoot, projectConfig{
		ServerURL:      output.ServerURL,
		OrganizationID: output.OrganizationID,
		ProjectID:      output.ProjectID,
		RepoBindingID:  output.RepoBindingID,
		Repository: projectConfigRepository{
			Provider:           output.Provider,
			FullName:           output.RepositoryFullName,
			URL:                output.RepositoryURL,
			WorkflowBaseBranch: output.WorkflowBaseBranch,
		},
	})
	if err != nil {
		return err
	}
	ignoreStatus, err := ensureProjectConfigGitignore(draft.GitRoot)
	if err != nil {
		return err
	}
	output.LocalConfigPath = projectConfigRelativePath
	output.LocalConfigStatus = configStatus
	output.LocalConfigMessage = localConfigMessage(configStatus)
	output.LocalIgnorePath = projectConfigIgnoreRelativePath
	output.LocalIgnoreStatus = ignoreStatus
	applyRepoBindingProjectScan(ctx, draft.GitRoot, output.RepoBindingID, options, &output)

	if format == term.FormatJSON {
		return term.WriteJSON(out.Stdout, output)
	}
	_, err = fmt.Fprint(out.Stdout, renderServerText(output))
	return err
}

func runRepositoryContextInit(ctx context.Context, out *term.Output, draft spine.RepoBindingDraft, format term.Format, options Options) error {
	if err := validateServerBackedDraft(draft); err != nil {
		return err
	}

	session, serverURL, err := loadUsableSession(options)
	if err != nil {
		return err
	}
	if err := preflightProjectConfig(draft.GitRoot, projectConfig{
		ServerURL: serverURL,
		Repository: projectConfigRepository{
			Provider:           draft.Provider,
			FullName:           draft.RepositoryFullName,
			URL:                draft.RepoURL,
			WorkflowBaseBranch: draft.WorkflowBaseBranch,
		},
	}); err != nil {
		return err
	}

	client := initHTTPClient(options.HTTPClient)

	requestPayload := spine.RepositoryContextInitRequest{
		Provider:                    draft.Provider,
		RepositoryFullName:          draft.RepositoryFullName,
		RepositoryURL:               draft.RepoURL,
		ProviderDefaultBranch:       draft.ProviderDefaultBranch,
		WorkflowBaseBranch:          draft.WorkflowBaseBranch,
		LocalRemoteName:             draft.RemoteName,
		LocalHeadSHA:                draft.HeadSHA,
		SuggestedProjectSlug:        deriveSuggestedProjectSlug(draft.Provider, draft.RepositoryFullName),
		SuggestedProjectDisplayName: draft.RepositoryFullName,
	}
	responsePayload, err := postRepositoryContextInit(ctx, client, session, requestPayload)
	if err != nil {
		return err
	}

	output := spine.RepositoryContextInitOutput{
		Mode:                  serverMode,
		ServerURL:             serverURL,
		OrganizationID:        responsePayload.OrganizationID,
		ProjectID:             responsePayload.ProjectID,
		ProjectSlug:           responsePayload.ProjectSlug,
		ProjectDisplayName:    responsePayload.ProjectDisplayName,
		ProjectCreated:        responsePayload.ProjectCreated,
		RepoBindingID:         responsePayload.RepoBindingID,
		RepoBindingCreated:    responsePayload.RepoBindingCreated,
		Provider:              responsePayload.Provider,
		RepositoryFullName:    responsePayload.RepositoryFullName,
		RepositoryURL:         responsePayload.RepositoryURL,
		ProviderDefaultBranch: responsePayload.ProviderDefaultBranch,
		WorkflowBaseBranch:    responsePayload.WorkflowBaseBranch,
		State:                 responsePayload.State,
		Message:               responsePayload.Message,
		NextCommand:           nextSuggestedCommand,
	}
	snapshotRequest, err := buildRepositoryContextSnapshot(draft.GitRoot, output, draft)
	if err != nil {
		return exitcode.RuntimeError(fmt.Errorf("collect repository context snapshot: %w", err))
	}
	snapshotResponse, err := postRepositoryContextSnapshot(ctx, client, session, output.RepoBindingID, snapshotRequest)
	if err != nil {
		return err
	}
	if snapshotResponse.Created {
		output.ContextSnapshotStatus = "recorded"
	} else {
		output.ContextSnapshotStatus = "unchanged"
	}
	output.ContextSnapshotID = snapshotResponse.ContextSnapshotID
	output.ContextFingerprint = snapshotResponse.Fingerprint

	configStatus, err := writeProjectConfig(draft.GitRoot, projectConfig{
		ServerURL:      output.ServerURL,
		OrganizationID: output.OrganizationID,
		ProjectID:      output.ProjectID,
		RepoBindingID:  output.RepoBindingID,
		Repository: projectConfigRepository{
			Provider:           output.Provider,
			FullName:           output.RepositoryFullName,
			URL:                output.RepositoryURL,
			WorkflowBaseBranch: output.WorkflowBaseBranch,
		},
	})
	if err != nil {
		return err
	}
	ignoreStatus, err := ensureProjectConfigGitignore(draft.GitRoot)
	if err != nil {
		return err
	}
	output.LocalConfigPath = projectConfigRelativePath
	output.LocalConfigStatus = configStatus
	output.LocalConfigMessage = localConfigMessage(configStatus)
	output.LocalIgnorePath = projectConfigIgnoreRelativePath
	output.LocalIgnoreStatus = ignoreStatus
	applyRepositoryContextProjectScan(ctx, draft.GitRoot, output.RepoBindingID, options, &output)

	if format == term.FormatJSON {
		return term.WriteJSON(out.Stdout, output)
	}
	_, err = fmt.Fprint(out.Stdout, renderRepositoryContextText(output))
	return err
}

func validateServerBackedDraft(draft spine.RepoBindingDraft) error {
	if draft.GitRoot == "" {
		return exitcode.UsageError(errors.New("server-backed init requires a Git root to write .goalrail/project.yml"))
	}
	if draft.WorkflowBaseBranch == "" {
		return exitcode.UsageError(errors.New("workflow_base_branch could not be detected from local origin metadata; server-backed init requires a workflow base branch"))
	}
	if draft.RepositoryFullName == "" {
		return exitcode.UsageError(errors.New("repository_full_name could not be parsed from repo URL"))
	}
	return nil
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

func initHTTPClient(client HTTPClient) HTTPClient {
	if client != nil {
		return client
	}
	return &http.Client{Timeout: defaultInitHTTPTimeout}
}

func doInitRequest(ctx context.Context, client HTTPClient, build func() (*http.Request, error)) (*http.Response, error) {
	for attempt := 1; attempt <= defaultInitRetryAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		request, err := build()
		if err != nil {
			return nil, err
		}
		response, err := client.Do(request)
		if err != nil {
			formattedErr := formatInitHTTPError(ctx, err)
			if attempt == defaultInitRetryAttempts || !shouldRetryInitTransportError(ctx, err) {
				return nil, formattedErr
			}
			if err := waitInitRetryBackoff(ctx, attempt); err != nil {
				return nil, err
			}
			continue
		}
		if !shouldRetryInitStatus(response.StatusCode) || attempt == defaultInitRetryAttempts {
			return response, nil
		}
		closeInitRetryResponse(response)
		if err := waitInitRetryBackoff(ctx, attempt); err != nil {
			return nil, err
		}
	}
	return nil, errors.New("server request could not complete")
}

func formatInitHTTPError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return err
	}
	if errors.Is(err, context.DeadlineExceeded) || isInitHTTPTimeout(err) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("server request timed out or could not complete: %w", err)
	}
	return fmt.Errorf("server request could not complete: %w", err)
}

func shouldRetryInitTransportError(ctx context.Context, err error) bool {
	if err == nil || ctx.Err() != nil {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) && (netErr.Timeout() || netErr.Temporary()) {
		return true
	}
	return errors.Is(err, io.ErrUnexpectedEOF)
}

func isInitHTTPTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func shouldRetryInitStatus(statusCode int) bool {
	return statusCode >= http.StatusInternalServerError && statusCode <= 599
}

func closeInitRetryResponse(response *http.Response) {
	if response == nil || response.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 1<<20))
	_ = response.Body.Close()
}

func waitInitRetryBackoff(ctx context.Context, attempt int) error {
	timer := time.NewTimer(defaultInitRetryBackoff * time.Duration(attempt))
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func postRepoBindingInit(ctx context.Context, client HTTPClient, session authstore.Session, projectID string, payload spine.RepoBindingInitRequest) (spine.RepoBindingInitResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return spine.RepoBindingInitResponse{}, exitcode.RuntimeError(fmt.Errorf("encode repo binding init request: %w", err))
	}
	serverURL := strings.TrimRight(session.ServerURL, "/")
	endpoint := serverURL + "/v1/projects/" + url.PathEscape(projectID) + "/repo-bindings/init"
	response, err := doInitRequest(ctx, client, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("build repo binding init request: %w", err)
		}
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Authorization", "Bearer "+session.AccessToken)
		return request, nil
	})
	if err != nil {
		return spine.RepoBindingInitResponse{}, exitcode.RuntimeError(fmt.Errorf("initialize repo binding on %s: %w", serverURL, err))
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK, http.StatusCreated:
		var decoded spine.RepoBindingInitResponse
		if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
			return spine.RepoBindingInitResponse{}, exitcode.RuntimeError(fmt.Errorf("decode repo binding init response: %w", err))
		}
		return decoded, nil
	case http.StatusUnauthorized, http.StatusForbidden:
		message := decodeServerErrorMessage(response.Body, response.StatusCode)
		return spine.RepoBindingInitResponse{}, exitcode.UsageError(fmt.Errorf("authenticated request failed: %s; run goalrail login %s", message, serverURL))
	case http.StatusBadRequest:
		message := decodeServerErrorMessage(response.Body, response.StatusCode)
		return spine.RepoBindingInitResponse{}, exitcode.ValidationError(fmt.Errorf("server validation failed: %s", message))
	case http.StatusConflict:
		message := decodeServerErrorMessage(response.Body, response.StatusCode)
		return spine.RepoBindingInitResponse{}, exitcode.ValidationError(fmt.Errorf("repo binding conflict: %s", message))
	default:
		message := decodeServerErrorMessage(response.Body, response.StatusCode)
		return spine.RepoBindingInitResponse{}, exitcode.RuntimeError(fmt.Errorf("repo binding init failed: %s", message))
	}
}

func postRepositoryContextInit(ctx context.Context, client HTTPClient, session authstore.Session, payload spine.RepositoryContextInitRequest) (spine.RepositoryContextInitResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return spine.RepositoryContextInitResponse{}, exitcode.RuntimeError(fmt.Errorf("encode repository context init request: %w", err))
	}
	serverURL := strings.TrimRight(session.ServerURL, "/")
	endpoint := serverURL + "/v1/init/repository-context"
	response, err := doInitRequest(ctx, client, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("build repository context init request: %w", err)
		}
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Authorization", "Bearer "+session.AccessToken)
		return request, nil
	})
	if err != nil {
		return spine.RepositoryContextInitResponse{}, exitcode.RuntimeError(fmt.Errorf("initialize repository context on %s: %w", serverURL, err))
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK, http.StatusCreated:
		var decoded spine.RepositoryContextInitResponse
		if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
			return spine.RepositoryContextInitResponse{}, exitcode.RuntimeError(fmt.Errorf("decode repository context init response: %w", err))
		}
		return decoded, nil
	case http.StatusUnauthorized, http.StatusForbidden:
		message := decodeServerErrorMessage(response.Body, response.StatusCode)
		return spine.RepositoryContextInitResponse{}, exitcode.UsageError(fmt.Errorf("authenticated request failed: %s; run goalrail login %s", message, serverURL))
	case http.StatusBadRequest:
		message := decodeServerErrorMessage(response.Body, response.StatusCode)
		return spine.RepositoryContextInitResponse{}, exitcode.ValidationError(fmt.Errorf("server validation failed: %s", message))
	case http.StatusConflict:
		message := decodeServerErrorMessage(response.Body, response.StatusCode)
		return spine.RepositoryContextInitResponse{}, exitcode.ValidationError(fmt.Errorf("repository context conflict: %s", message))
	default:
		message := decodeServerErrorMessage(response.Body, response.StatusCode)
		return spine.RepositoryContextInitResponse{}, exitcode.RuntimeError(fmt.Errorf("repository context init failed: %s", message))
	}
}

func postRepositoryContextSnapshot(ctx context.Context, client HTTPClient, session authstore.Session, repoBindingID string, payload spine.RepositoryContextSnapshotRequest) (spine.RepositoryContextSnapshotResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return spine.RepositoryContextSnapshotResponse{}, exitcode.RuntimeError(fmt.Errorf("encode repository context snapshot request: %w", err))
	}
	serverURL := strings.TrimRight(session.ServerURL, "/")
	endpoint := serverURL + "/v1/repo-bindings/" + url.PathEscape(repoBindingID) + "/context-snapshots"
	response, err := doInitRequest(ctx, client, func() (*http.Request, error) {
		request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("build repository context snapshot request: %w", err)
		}
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Authorization", "Bearer "+session.AccessToken)
		return request, nil
	})
	if err != nil {
		return spine.RepositoryContextSnapshotResponse{}, exitcode.RuntimeError(fmt.Errorf("record repository context snapshot on %s: %w", serverURL, err))
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK, http.StatusCreated:
		var decoded spine.RepositoryContextSnapshotResponse
		if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
			return spine.RepositoryContextSnapshotResponse{}, exitcode.RuntimeError(fmt.Errorf("decode repository context snapshot response: %w", err))
		}
		return decoded, nil
	case http.StatusUnauthorized, http.StatusForbidden:
		message := decodeServerErrorMessage(response.Body, response.StatusCode)
		return spine.RepositoryContextSnapshotResponse{}, exitcode.UsageError(fmt.Errorf("authenticated request failed: %s; run goalrail login %s", message, serverURL))
	case http.StatusBadRequest:
		message := decodeServerErrorMessage(response.Body, response.StatusCode)
		return spine.RepositoryContextSnapshotResponse{}, exitcode.ValidationError(fmt.Errorf("server validation failed: %s", message))
	case http.StatusConflict:
		message := decodeServerErrorMessage(response.Body, response.StatusCode)
		return spine.RepositoryContextSnapshotResponse{}, exitcode.ValidationError(fmt.Errorf("repository context snapshot conflict: %s", message))
	default:
		message := decodeServerErrorMessage(response.Body, response.StatusCode)
		return spine.RepositoryContextSnapshotResponse{}, exitcode.RuntimeError(fmt.Errorf("repository context snapshot failed: %s", message))
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

func renderServerText(output spine.RepoBindingInitOutput) string {
	var b strings.Builder
	title := "Repository binding initialized"
	if !output.Created {
		title = "Repository binding already initialized"
	}
	fmt.Fprintf(&b, "%s\n\n", title)
	fmt.Fprintf(&b, "Server: %s\n", output.ServerURL)
	fmt.Fprintf(&b, "Project: %s\n", output.ProjectID)
	fmt.Fprintf(&b, "Repo binding: %s\n", output.RepoBindingID)
	if output.RepositoryFullName != "" {
		fmt.Fprintf(&b, "Repository: %s\n", output.RepositoryFullName)
	}
	if output.Provider != "" {
		fmt.Fprintf(&b, "Provider: %s\n", output.Provider)
	}
	if output.WorkflowBaseBranch != "" {
		fmt.Fprintf(&b, "Workflow base branch: %s\n", output.WorkflowBaseBranch)
	}
	if output.State != "" {
		fmt.Fprintf(&b, "State: %s\n", output.State)
	}
	if output.LocalConfigPath != "" {
		fmt.Fprintf(&b, "Local config: %s (%s)\n", output.LocalConfigPath, output.LocalConfigStatus)
	}
	writeLocalConfigText(&b, output.LocalConfigMessage, output.LocalIgnorePath, output.LocalIgnoreStatus)
	writeProjectScanText(&b, output.ProjectScanStatus, output.ProjectScanBaselineID, output.ProjectScanFreshness, output.ProjectScanWarning)
	fmt.Fprintf(&b, "\n%s\n\nNext: %s\n", serverRegistrationNote, output.NextCommand)
	return b.String()
}

func renderRepositoryContextText(output spine.RepositoryContextInitOutput) string {
	var b strings.Builder
	title := "Repository context initialized"
	if !output.ProjectCreated && !output.RepoBindingCreated {
		title = "Repository context already initialized"
	}
	fmt.Fprintf(&b, "%s\n\n", title)
	fmt.Fprintf(&b, "Server: %s\n", output.ServerURL)
	fmt.Fprintf(&b, "Organization: %s\n", output.OrganizationID)
	if output.ProjectSlug != "" || output.ProjectDisplayName != "" {
		display := output.ProjectDisplayName
		if display == "" {
			display = output.ProjectSlug
		}
		fmt.Fprintf(&b, "Project: %s (%s)\n", display, output.ProjectID)
	} else {
		fmt.Fprintf(&b, "Project: %s\n", output.ProjectID)
	}
	fmt.Fprintf(&b, "Repo binding: %s\n", output.RepoBindingID)
	if output.RepositoryFullName != "" {
		fmt.Fprintf(&b, "Repository: %s\n", output.RepositoryFullName)
	}
	if output.Provider != "" {
		fmt.Fprintf(&b, "Provider: %s\n", output.Provider)
	}
	if output.WorkflowBaseBranch != "" {
		fmt.Fprintf(&b, "Workflow base branch: %s\n", output.WorkflowBaseBranch)
	}
	if output.State != "" {
		fmt.Fprintf(&b, "State: %s\n", output.State)
	}
	if output.LocalConfigPath != "" {
		fmt.Fprintf(&b, "Local config: %s (%s)\n", output.LocalConfigPath, output.LocalConfigStatus)
	}
	writeLocalConfigText(&b, output.LocalConfigMessage, output.LocalIgnorePath, output.LocalIgnoreStatus)
	if output.ContextSnapshotID != "" {
		fmt.Fprintf(&b, "Repository context snapshot: %s (%s)\n", output.ContextSnapshotID, output.ContextSnapshotStatus)
	}
	writeProjectScanText(&b, output.ProjectScanStatus, output.ProjectScanBaselineID, output.ProjectScanFreshness, output.ProjectScanWarning)
	fmt.Fprintf(&b, "\n%s\n\nNext: %s\n", repositoryContextNote, output.NextCommand)
	return b.String()
}

type initProjectScanResult struct {
	Status     string
	BaselineID string
	OverlayID  string
	Freshness  string
	Warning    string
}

func runInitProjectScan(ctx context.Context, gitRoot string, repoBindingID string, options Options) initProjectScanResult {
	cache := projectscan.NewCache(options.ProjectScanCacheRoot)
	baseline, err := projectscan.BuildBaseline(ctx, gitRoot, repoBindingID, projectscan.DefaultBuildOptions())
	if err != nil {
		return initProjectScanResult{Status: projectscan.BaselineStatusError, Warning: err.Error()}
	}
	if err := cache.WriteBaseline(baseline); err != nil {
		return initProjectScanResult{Status: projectscan.BaselineStatusError, BaselineID: baseline.RepositoryBaselineProfileID, Warning: err.Error()}
	}
	overlay, rawStatus, err := projectscan.BuildOverlay(ctx, gitRoot, repoBindingID, &baseline, projectscan.OverlayOptions{Now: options.Now})
	if err != nil {
		return initProjectScanResult{Status: projectscan.BaselineStatusError, BaselineID: baseline.RepositoryBaselineProfileID, Warning: err.Error()}
	}
	if err := cache.WriteOverlay(overlay, rawStatus); err != nil {
		return initProjectScanResult{Status: projectscan.BaselineStatusError, BaselineID: baseline.RepositoryBaselineProfileID, OverlayID: overlay.WorkspaceOverlayID, Warning: err.Error()}
	}
	freshness := projectscan.EvaluateFreshness(baseline.HeadSHA, &baseline, overlay)
	return initProjectScanResult{
		Status:     baseline.Status,
		BaselineID: baseline.RepositoryBaselineProfileID,
		OverlayID:  overlay.WorkspaceOverlayID,
		Freshness:  freshness.Status,
	}
}

func applyRepoBindingProjectScan(ctx context.Context, gitRoot string, repoBindingID string, options Options, output *spine.RepoBindingInitOutput) {
	result := runInitProjectScan(ctx, gitRoot, repoBindingID, options)
	output.ProjectScanStatus = result.Status
	output.ProjectScanBaselineID = result.BaselineID
	output.ProjectScanOverlayID = result.OverlayID
	output.ProjectScanFreshness = result.Freshness
	output.ProjectScanWarning = result.Warning
}

func applyRepositoryContextProjectScan(ctx context.Context, gitRoot string, repoBindingID string, options Options, output *spine.RepositoryContextInitOutput) {
	result := runInitProjectScan(ctx, gitRoot, repoBindingID, options)
	output.ProjectScanStatus = result.Status
	output.ProjectScanBaselineID = result.BaselineID
	output.ProjectScanOverlayID = result.OverlayID
	output.ProjectScanFreshness = result.Freshness
	output.ProjectScanWarning = result.Warning
}

func writeProjectScanText(b *strings.Builder, status string, baselineID string, freshness string, warning string) {
	if status == "" {
		return
	}
	fmt.Fprintf(b, "Project scan: %s", status)
	if baselineID != "" {
		fmt.Fprintf(b, " (%s)", baselineID)
	}
	if freshness != "" {
		fmt.Fprintf(b, ", freshness: %s", freshness)
	}
	b.WriteByte('\n')
	if warning != "" {
		fmt.Fprintf(b, "Project scan warning: %s\n", warning)
	}
}

func localConfigMessage(status string) string {
	if status == localConfigStatusUnchanged {
		return "Existing Goalrail project marker found and verified."
	}
	return "Commit .goalrail/project.yml and .goalrail/.gitignore with this repository."
}

func writeLocalConfigText(b *strings.Builder, message string, ignorePath string, ignoreStatus string) {
	if ignorePath != "" {
		fmt.Fprintf(b, "Local state ignore rules: %s (%s)\n", ignorePath, ignoreStatus)
	}
	if message != "" {
		fmt.Fprintf(b, "%s\n", message)
	}
}

func deriveSuggestedProjectSlug(provider string, repositoryFullName string) string {
	source := strings.ToLower(strings.TrimSpace(provider) + "-" + strings.Trim(strings.TrimSuffix(strings.TrimSpace(repositoryFullName), ".git"), "/"))
	var b strings.Builder
	lastDash := false
	for _, r := range source {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash && b.Len() > 0 {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
