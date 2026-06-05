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

	"github.com/heurema/goalrail/apps/cli/internal/authsession"
	"github.com/heurema/goalrail/apps/cli/internal/authstore"
	"github.com/heurema/goalrail/apps/cli/internal/exitcode"
	"github.com/heurema/goalrail/apps/cli/internal/projectscan"
	"github.com/heurema/goalrail/apps/cli/internal/spine"
	"github.com/heurema/goalrail/apps/cli/internal/term"
)

const (
	serverMode             = "server"
	nextSuggestedCommand   = "goalrail work start --title <title>"
	dogfoodNextCommand     = "goalrail work start --title \"Dogfood Goalrail on Goalrail\""
	serverRegistrationNote = "This registered repository metadata on the GoalRail server, wrote a non-secret GoalRail repository marker, and ran a local Project Scan.\nNo server clone, audit, hooks, branch creation, deploy keys, provider integration, runner, gate, proof, or verification were configured."
	repositoryContextNote  = "This initialized GoalRail repository context for your existing organization, wrote a non-secret GoalRail repository marker, attempted a metadata-only repository context snapshot, and ran a local Project Scan.\nNo server clone, audit, hooks, branch creation, deploy keys, provider integration, runner, gate, proof, or verification were configured."

	defaultInitHTTPTimeout   = 30 * time.Second
	defaultInitRetryAttempts = 3
	defaultInitRetryBackoff  = 10 * time.Millisecond

	initStepRepoBinding       = "repo_binding"
	initStepRepositoryContext = "repository_context"
	initStepLocalMarker       = "local_marker"
	initStepLocalGitignore    = "local_gitignore"
	initStepContextSnapshot   = "context_snapshot"
	initStepProjectScan       = "project_scan"
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

	session, serverURL, client, err := loadUsableSession(ctx, options)
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
	steps := []spine.InitStepResult{
		initStepOK(initStepRepoBinding, responsePayload.Message),
	}
	retryCommand := repoBindingInitRetryCommand(projectID, draft)
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
		output.LocalConfigPath = projectConfigRelativePath
		output.LocalConfigStatus = string(spine.InitStepStatusError)
		output.LocalConfigMessage = err.Error()
		output.NextCommand = retryCommand
		steps = append(steps,
			initStepError(initStepLocalMarker, "Local Goalrail project marker could not be written: "+err.Error(), retryCommand),
			initStepSkipped(initStepLocalGitignore, "Skipped because local marker write failed."),
			initStepSkipped(initStepProjectScan, "Skipped because local marker write failed."),
		)
		output.Steps = steps
		output.Status = initOverallStatus(steps)
		return writeRepoBindingOutputWithError(out, format, output, err)
	}
	steps = append(steps, initStepOK(initStepLocalMarker, localConfigMessage(configStatus)))
	ignoreStatus, err := ensureProjectConfigGitignore(draft.GitRoot)
	if err != nil {
		output.LocalConfigPath = projectConfigRelativePath
		output.LocalConfigStatus = configStatus
		output.LocalConfigMessage = localConfigMessage(configStatus)
		output.LocalIgnorePath = projectConfigIgnoreRelativePath
		output.LocalIgnoreStatus = string(spine.InitStepStatusError)
		output.NextCommand = retryCommand
		steps = append(steps,
			initStepError(initStepLocalGitignore, "Local Goalrail ignore rules could not be written: "+err.Error(), retryCommand),
			initStepSkipped(initStepProjectScan, "Skipped because local ignore rule write failed."),
		)
		output.Steps = steps
		output.Status = initOverallStatus(steps)
		return writeRepoBindingOutputWithError(out, format, output, err)
	}
	output.LocalConfigPath = projectConfigRelativePath
	output.LocalConfigStatus = configStatus
	output.LocalConfigMessage = localConfigMessage(configStatus)
	output.LocalIgnorePath = projectConfigIgnoreRelativePath
	output.LocalIgnoreStatus = ignoreStatus
	steps = append(steps, initStepOK(initStepLocalGitignore, localIgnoreMessage(ignoreStatus)))
	scanResult := applyRepoBindingProjectScan(ctx, draft.GitRoot, output.RepoBindingID, options, &output)
	steps = append(steps, projectScanInitStep(scanResult))
	output.Steps = steps
	output.Status = initOverallStatus(steps)

	if format == term.FormatJSON {
		return term.WriteJSON(out.Stdout, output)
	}
	_, err = fmt.Fprint(out.Stdout, renderServerText(output, scanResult.Summary))
	return err
}

func runRepositoryContextInit(ctx context.Context, out *term.Output, draft spine.RepoBindingDraft, format term.Format, options Options) error {
	if err := validateServerBackedDraft(draft); err != nil {
		return err
	}

	session, serverURL, client, err := loadUsableSession(ctx, options)
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
	steps := []spine.InitStepResult{
		initStepOK(initStepRepositoryContext, responsePayload.Message),
	}
	retryCommand := repositoryContextInitRetryCommand(draft)
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
		output.LocalConfigPath = projectConfigRelativePath
		output.LocalConfigStatus = string(spine.InitStepStatusError)
		output.LocalConfigMessage = err.Error()
		output.NextCommand = retryCommand
		steps = append(steps,
			initStepError(initStepLocalMarker, "Local Goalrail project marker could not be written: "+err.Error(), retryCommand),
			initStepSkipped(initStepLocalGitignore, "Skipped because local marker write failed."),
			initStepSkipped(initStepContextSnapshot, "Skipped because local marker write failed."),
			initStepSkipped(initStepProjectScan, "Skipped because local marker write failed."),
		)
		output.Steps = steps
		output.Status = initOverallStatus(steps)
		return writeRepositoryContextOutputWithError(out, format, output, err)
	}
	steps = append(steps, initStepOK(initStepLocalMarker, localConfigMessage(configStatus)))
	ignoreStatus, err := ensureProjectConfigGitignore(draft.GitRoot)
	if err != nil {
		output.LocalConfigPath = projectConfigRelativePath
		output.LocalConfigStatus = configStatus
		output.LocalConfigMessage = localConfigMessage(configStatus)
		output.LocalIgnorePath = projectConfigIgnoreRelativePath
		output.LocalIgnoreStatus = string(spine.InitStepStatusError)
		output.NextCommand = retryCommand
		steps = append(steps,
			initStepError(initStepLocalGitignore, "Local Goalrail ignore rules could not be written: "+err.Error(), retryCommand),
			initStepSkipped(initStepContextSnapshot, "Skipped because local ignore rule write failed."),
			initStepSkipped(initStepProjectScan, "Skipped because local ignore rule write failed."),
		)
		output.Steps = steps
		output.Status = initOverallStatus(steps)
		return writeRepositoryContextOutputWithError(out, format, output, err)
	}
	output.LocalConfigPath = projectConfigRelativePath
	output.LocalConfigStatus = configStatus
	output.LocalConfigMessage = localConfigMessage(configStatus)
	output.LocalIgnorePath = projectConfigIgnoreRelativePath
	output.LocalIgnoreStatus = ignoreStatus
	steps = append(steps, initStepOK(initStepLocalGitignore, localIgnoreMessage(ignoreStatus)))

	snapshotStep := applyRepositoryContextSnapshot(ctx, client, session, draft, &output)
	steps = append(steps, snapshotStep)
	scanResult := applyRepositoryContextProjectScan(ctx, draft.GitRoot, output.RepoBindingID, options, &output)
	steps = append(steps, projectScanInitStep(scanResult))
	output.Steps = steps
	output.Status = initOverallStatus(steps)

	if format == term.FormatJSON {
		return term.WriteJSON(out.Stdout, output)
	}
	_, err = fmt.Fprint(out.Stdout, renderRepositoryContextText(output, scanResult.Summary))
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

func loadUsableSession(ctx context.Context, options Options) (authstore.Session, string, HTTPClient, error) {
	store := options.Store
	if store == nil {
		path, err := authstore.DefaultPath()
		if err != nil {
			return authstore.Session{}, "", nil, exitcode.RuntimeError(err)
		}
		store = authstore.NewFileStore(path)
	}
	client := initHTTPClient(options.HTTPClient)
	if client == nil {
		client = http.DefaultClient
	}
	return authsession.LoadUsable(ctx, authsession.Options{
		Store:  store,
		Client: client,
		Now:    options.Now,
	})
}

func initStepOK(name string, message string) spine.InitStepResult {
	return spine.InitStepResult{
		Name:    name,
		Status:  spine.InitStepStatusOK,
		Message: strings.TrimSpace(message),
	}
}

func initStepSkipped(name string, message string) spine.InitStepResult {
	return spine.InitStepResult{
		Name:    name,
		Status:  spine.InitStepStatusSkipped,
		Message: strings.TrimSpace(message),
	}
}

func initStepWarning(name string, message string, retryCommand string) spine.InitStepResult {
	return spine.InitStepResult{
		Name:         name,
		Status:       spine.InitStepStatusWarning,
		Message:      strings.TrimSpace(message),
		Recoverable:  true,
		RetryCommand: strings.TrimSpace(retryCommand),
	}
}

func initStepError(name string, message string, retryCommand string) spine.InitStepResult {
	return spine.InitStepResult{
		Name:         name,
		Status:       spine.InitStepStatusError,
		Message:      strings.TrimSpace(message),
		Recoverable:  true,
		RetryCommand: strings.TrimSpace(retryCommand),
	}
}

func initOverallStatus(steps []spine.InitStepResult) spine.InitOverallStatus {
	hasWarning := false
	for _, step := range steps {
		switch step.Status {
		case spine.InitStepStatusError:
			return spine.InitOverallStatusPartialFailed
		case spine.InitStepStatusWarning:
			hasWarning = true
		}
	}
	if hasWarning {
		return spine.InitOverallStatusSuccessWithWarnings
	}
	return spine.InitOverallStatusSuccess
}

func projectScanInitStep(result initProjectScanResult) spine.InitStepResult {
	if result.Warning != "" || result.Status == projectscan.BaselineStatusError {
		message := "Local Project Scan cache could not be written."
		if result.Warning != "" {
			message = "Local Project Scan cache could not be written: " + result.Warning
		}
		return initStepWarning(initStepProjectScan, message, "goalrail project scan --format json")
	}
	return initStepOK(initStepProjectScan, "Local Project Scan cache written.")
}

func applyRepositoryContextSnapshot(ctx context.Context, client HTTPClient, session authstore.Session, draft spine.RepoBindingDraft, output *spine.RepositoryContextInitOutput) spine.InitStepResult {
	snapshotRequest, err := buildRepositoryContextSnapshot(draft.GitRoot, *output, draft)
	if err != nil {
		output.ContextSnapshotStatus = string(spine.InitStepStatusWarning)
		return initStepWarning(initStepContextSnapshot, "Repository context snapshot was not recorded: collect repository context snapshot: "+err.Error(), repositoryContextInitRetryCommand(draft))
	}
	snapshotResponse, err := postRepositoryContextSnapshot(ctx, client, session, output.RepoBindingID, snapshotRequest)
	if err != nil {
		output.ContextSnapshotStatus = string(spine.InitStepStatusWarning)
		return initStepWarning(initStepContextSnapshot, "Repository context snapshot was not recorded: "+err.Error(), repositoryContextInitRetryCommand(draft))
	}
	if snapshotResponse.Created {
		output.ContextSnapshotStatus = "recorded"
	} else {
		output.ContextSnapshotStatus = "unchanged"
	}
	output.ContextSnapshotID = snapshotResponse.ContextSnapshotID
	output.ContextFingerprint = snapshotResponse.Fingerprint
	if snapshotResponse.Message != "" {
		return initStepOK(initStepContextSnapshot, snapshotResponse.Message)
	}
	return initStepOK(initStepContextSnapshot, "Repository context snapshot "+output.ContextSnapshotStatus+".")
}

func repoBindingInitRetryCommand(projectID string, draft spine.RepoBindingDraft) string {
	parts := []string{"goalrail", "init", "--project", shellQuoteInitArg(projectID)}
	return appendInitContextRetryArgs(parts, draft)
}

func repositoryContextInitRetryCommand(draft spine.RepoBindingDraft) string {
	return appendInitContextRetryArgs([]string{"goalrail", "init"}, draft)
}

func appendInitContextRetryArgs(parts []string, draft spine.RepoBindingDraft) string {
	if strings.TrimSpace(draft.RepoURL) != "" {
		parts = append(parts, "--repo", shellQuoteInitArg(strings.TrimSpace(draft.RepoURL)))
	}
	if strings.TrimSpace(draft.WorkflowBaseBranch) != "" {
		parts = append(parts, "--base", shellQuoteInitArg(strings.TrimSpace(draft.WorkflowBaseBranch)))
	}
	return strings.Join(parts, " ")
}

func shellQuoteInitArg(value string) string {
	if value == "" {
		return "''"
	}
	if isPlainInitShellArg(value) {
		return value
	}
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func isPlainInitShellArg(value string) bool {
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			continue
		}
		switch r {
		case '_', '-', '.', '/', ':', '@', '=', '+', ',', '%':
			continue
		default:
			return false
		}
	}
	return true
}

func writeRepoBindingOutputWithError(out *term.Output, format term.Format, output spine.RepoBindingInitOutput, commandErr error) error {
	var writeErr error
	if format == term.FormatJSON {
		writeErr = term.WriteJSON(out.Stdout, output)
	} else {
		_, writeErr = fmt.Fprint(out.Stdout, renderServerText(output))
	}
	if writeErr != nil {
		return writeErr
	}
	return commandErr
}

func writeRepositoryContextOutputWithError(out *term.Output, format term.Format, output spine.RepositoryContextInitOutput, commandErr error) error {
	var writeErr error
	if format == term.FormatJSON {
		writeErr = term.WriteJSON(out.Stdout, output)
	} else {
		_, writeErr = fmt.Fprint(out.Stdout, renderRepositoryContextText(output))
	}
	if writeErr != nil {
		return writeErr
	}
	return commandErr
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

func renderServerText(output spine.RepoBindingInitOutput, scanSummaries ...initProjectScanSummary) string {
	var b strings.Builder
	title := "Repository binding initialized"
	if !output.Created {
		title = "Repository binding already initialized"
	}
	if output.Status == spine.InitOverallStatusPartialFailed {
		title = "Repository binding partially initialized"
	}
	fmt.Fprintf(&b, "%s\n\n", title)
	writeInitStatusText(&b, output.Status)
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
	writeProjectScanSummaryText(&b, outputProjectScanSummary(output.ProjectScanStatus, output.ProjectScanWarning, scanSummaries...))
	writeInitStepNotices(&b, output.Steps)
	if output.Status == spine.InitOverallStatusPartialFailed {
		fmt.Fprintf(&b, "\nServer binding succeeded, but local init did not complete.\n\nNext: %s\n", humanInitNextCommand(output.NextCommand, output.Status))
	} else {
		fmt.Fprintf(&b, "\n%s\n\nNext: %s\n", serverRegistrationNote, humanInitNextCommand(output.NextCommand, output.Status))
	}
	return b.String()
}

func renderRepositoryContextText(output spine.RepositoryContextInitOutput, scanSummaries ...initProjectScanSummary) string {
	var b strings.Builder
	title := "Repository context initialized"
	if !output.ProjectCreated && !output.RepoBindingCreated {
		title = "Repository context already initialized"
	}
	if output.Status == spine.InitOverallStatusPartialFailed {
		title = "Repository context partially initialized"
	}
	fmt.Fprintf(&b, "%s\n\n", title)
	writeInitStatusText(&b, output.Status)
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
	writeProjectScanSummaryText(&b, outputProjectScanSummary(output.ProjectScanStatus, output.ProjectScanWarning, scanSummaries...))
	writeInitStepNotices(&b, output.Steps)
	if output.Status == spine.InitOverallStatusPartialFailed {
		fmt.Fprintf(&b, "\nServer repository context succeeded, but local init did not complete.\n\nNext: %s\n", humanInitNextCommand(output.NextCommand, output.Status))
	} else {
		fmt.Fprintf(&b, "\n%s\n\nNext: %s\n", repositoryContextNote, humanInitNextCommand(output.NextCommand, output.Status))
	}
	return b.String()
}

func humanInitNextCommand(nextCommand string, status spine.InitOverallStatus) string {
	if status != spine.InitOverallStatusPartialFailed && nextCommand == nextSuggestedCommand {
		return dogfoodNextCommand
	}
	return nextCommand
}

type initProjectScanResult struct {
	Status     string
	BaselineID string
	OverlayID  string
	Freshness  string
	Warning    string
	Summary    initProjectScanSummary
}

func runInitProjectScan(ctx context.Context, gitRoot string, repoBindingID string, options Options) initProjectScanResult {
	cache := projectscan.NewCache(options.ProjectScanCacheRoot)
	baseline, err := projectscan.BuildBaseline(ctx, gitRoot, repoBindingID, projectscan.DefaultBuildOptions())
	if err != nil {
		return initProjectScanResult{
			Status:  projectscan.BaselineStatusError,
			Warning: err.Error(),
			Summary: unavailableProjectScanSummary(err.Error()),
		}
	}
	_, baselineCached, err := cache.LoadLatestBaseline(repoBindingID, baseline.CanonicalRepoRoot)
	if err != nil {
		warning := fmt.Errorf("read project scan cache: %w", err).Error()
		return initProjectScanResult{
			Status:  projectscan.BaselineStatusError,
			Warning: warning,
			Summary: unavailableProjectScanSummary(warning),
		}
	}
	baselineAction := projectScanArtifactCreated
	if baselineCached {
		baselineAction = projectScanArtifactRefreshed
	}
	if err := cache.WriteBaseline(baseline); err != nil {
		return initProjectScanResult{
			Status:     projectscan.BaselineStatusError,
			BaselineID: baseline.RepositoryBaselineProfileID,
			Warning:    err.Error(),
			Summary:    unavailableProjectScanSummary(err.Error()),
		}
	}
	_, overlayCached, _ := cache.LoadCurrentOverlay(repoBindingID, baseline.CanonicalRepoRoot)
	overlayAction := projectScanArtifactCreated
	if overlayCached {
		overlayAction = projectScanArtifactRefreshed
	}
	overlay, rawStatus, err := projectscan.BuildOverlay(ctx, gitRoot, repoBindingID, &baseline, projectscan.OverlayOptions{Now: options.Now})
	if err != nil {
		return initProjectScanResult{
			Status:     projectscan.BaselineStatusError,
			BaselineID: baseline.RepositoryBaselineProfileID,
			Warning:    err.Error(),
			Summary:    unavailableProjectScanSummary(err.Error()),
		}
	}
	if err := cache.WriteOverlay(overlay, rawStatus); err != nil {
		return initProjectScanResult{
			Status:     projectscan.BaselineStatusError,
			BaselineID: baseline.RepositoryBaselineProfileID,
			OverlayID:  overlay.WorkspaceOverlayID,
			Warning:    err.Error(),
			Summary:    unavailableProjectScanSummary(err.Error()),
		}
	}
	freshness := projectscan.EvaluateFreshness(baseline.HeadSHA, &baseline, overlay)
	return initProjectScanResult{
		Status:     baseline.Status,
		BaselineID: baseline.RepositoryBaselineProfileID,
		OverlayID:  overlay.WorkspaceOverlayID,
		Freshness:  freshness.Status,
		Summary:    summarizeInitProjectScan(baselineAction, overlayAction, baseline, overlay, freshness),
	}
}

func applyRepoBindingProjectScan(ctx context.Context, gitRoot string, repoBindingID string, options Options, output *spine.RepoBindingInitOutput) initProjectScanResult {
	result := runInitProjectScan(ctx, gitRoot, repoBindingID, options)
	output.ProjectScanStatus = result.Status
	output.ProjectScanBaselineID = result.BaselineID
	output.ProjectScanOverlayID = result.OverlayID
	output.ProjectScanFreshness = result.Freshness
	output.ProjectScanWarning = result.Warning
	return result
}

func applyRepositoryContextProjectScan(ctx context.Context, gitRoot string, repoBindingID string, options Options, output *spine.RepositoryContextInitOutput) initProjectScanResult {
	result := runInitProjectScan(ctx, gitRoot, repoBindingID, options)
	output.ProjectScanStatus = result.Status
	output.ProjectScanBaselineID = result.BaselineID
	output.ProjectScanOverlayID = result.OverlayID
	output.ProjectScanFreshness = result.Freshness
	output.ProjectScanWarning = result.Warning
	return result
}

func outputProjectScanSummary(status string, warning string, summaries ...initProjectScanSummary) initProjectScanSummary {
	if len(summaries) > 0 && (summaries[0].Baseline != "" || summaries[0].Overlay != "") {
		return summaries[0]
	}
	if status == projectscan.BaselineStatusError || warning != "" {
		return unavailableProjectScanSummary(warning)
	}
	return initProjectScanSummary{}
}

func writeInitStatusText(b *strings.Builder, status spine.InitOverallStatus) {
	if status == "" || status == spine.InitOverallStatusSuccess {
		return
	}
	fmt.Fprintf(b, "Init status: %s\n", status)
}

func writeInitStepNotices(b *strings.Builder, steps []spine.InitStepResult) {
	for _, step := range steps {
		switch step.Status {
		case spine.InitStepStatusWarning:
			if step.Message != "" {
				fmt.Fprintf(b, "Warning: %s\n", step.Message)
			}
			if step.Name == initStepProjectScan {
				continue
			}
			if step.RetryCommand != "" {
				fmt.Fprintf(b, "Recovery hint: retry with `%s`.\n", step.RetryCommand)
			}
		case spine.InitStepStatusError:
			if step.Message != "" {
				fmt.Fprintf(b, "Local recovery needed: %s\n", step.Message)
			}
			if step.RetryCommand != "" {
				fmt.Fprintf(b, "Recovery hint: fix the local file issue, then retry with `%s`.\n", step.RetryCommand)
			}
		}
	}
}

func localConfigMessage(status string) string {
	if status == localConfigStatusUnchanged {
		return "Existing Goalrail project marker found and verified."
	}
	return "Commit .goalrail/project.yml and .goalrail/.gitignore with this repository."
}

func localIgnoreMessage(status string) string {
	switch status {
	case localConfigStatusUnchanged:
		return "Existing .goalrail/.gitignore local state rules found and verified."
	case localConfigStatusUpdated:
		return "Updated .goalrail/.gitignore with Goalrail local state rules."
	default:
		return "Wrote .goalrail/.gitignore with Goalrail local state rules."
	}
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
