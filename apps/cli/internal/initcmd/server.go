package initcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/heurema/goalrail/apps/cli/internal/authstore"
	"github.com/heurema/goalrail/apps/cli/internal/exitcode"
	"github.com/heurema/goalrail/apps/cli/internal/spine"
	"github.com/heurema/goalrail/apps/cli/internal/term"
)

const (
	serverMode             = "server"
	nextSuggestedCommand   = "goalrail readiness scan --path ."
	serverRegistrationNote = "This registered repository metadata on the GoalRail server and wrote a non-secret local GoalRail marker.\nNo audit, hooks, branch creation, deploy keys, or verification were configured."
)

type serverErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func runServerBackedInit(ctx context.Context, out *term.Output, draft spine.RepoBindingDraft, projectID string, format term.Format, options Options) error {
	if draft.GitRoot == "" {
		return exitcode.UsageError(errors.New("server-backed init requires a Git root to write .goalrail/project.yml"))
	}
	if draft.WorkflowBaseBranch == "" {
		return exitcode.UsageError(errors.New("workflow_base_branch could not be detected from local origin metadata; server-backed init requires a workflow base branch"))
	}
	if draft.RepositoryFullName == "" {
		return exitcode.UsageError(errors.New("repository_full_name could not be parsed from repo URL"))
	}

	session, err := loadSession(options)
	if err != nil {
		return err
	}
	now := time.Now
	if options.Now != nil {
		now = options.Now
	}
	if !session.AccessTokenExpiresAt.After(now().UTC()) {
		return exitcode.UsageError(fmt.Errorf("login expired; run goalrail login %s", session.ServerURL))
	}
	serverURL := strings.TrimRight(session.ServerURL, "/")
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

	client := options.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	requestPayload := spine.RepoBindingInitRequest{
		Provider:              draft.Provider,
		RepositoryFullName:    draft.RepositoryFullName,
		RepositoryURL:         draft.RepoURL,
		ProviderDefaultBranch: draft.WorkflowBaseBranch,
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
	output.LocalConfigPath = projectConfigRelativePath
	output.LocalConfigStatus = configStatus

	if format == term.FormatJSON {
		return term.WriteJSON(out.Stdout, output)
	}
	_, err = fmt.Fprint(out.Stdout, renderServerText(output))
	return err
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

func postRepoBindingInit(ctx context.Context, client HTTPClient, session authstore.Session, projectID string, payload spine.RepoBindingInitRequest) (spine.RepoBindingInitResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return spine.RepoBindingInitResponse{}, exitcode.RuntimeError(fmt.Errorf("encode repo binding init request: %w", err))
	}
	serverURL := strings.TrimRight(session.ServerURL, "/")
	endpoint := serverURL + "/v1/projects/" + url.PathEscape(projectID) + "/repo-bindings/init"
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return spine.RepoBindingInitResponse{}, exitcode.RuntimeError(fmt.Errorf("build repo binding init request: %w", err))
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+session.AccessToken)

	response, err := client.Do(request)
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
	fmt.Fprintf(&b, "\n%s\n\nNext: %s\n", serverRegistrationNote, output.NextCommand)
	return b.String()
}
