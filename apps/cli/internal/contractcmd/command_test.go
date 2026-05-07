package contractcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/cli/internal/authstore"
	"github.com/heurema/goalrail/apps/cli/internal/exitcode"
	"github.com/heurema/goalrail/apps/cli/internal/projectconfig"
	"github.com/heurema/goalrail/apps/cli/internal/spine"
	"github.com/heurema/goalrail/apps/cli/internal/term"
)

func TestRunDraftCreatesContractAndReturnsLocalRepoReceipt(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)

	var meCount, contractCount atomic.Int32
	var request contractCreateRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer access-token" {
			t.Errorf("Authorization = %q, want bearer token", r.Header.Get("Authorization"))
			http.Error(w, "bad auth", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/v1/me":
			meCount.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"user":{"id":"018f0000-0000-7000-8000-000000000001","display_name":"Developer"},"organization_membership":{"organization_id":"018f0000-0000-7000-8000-000000000002","role":"member","state":"active"}}`))
		case "/v1/contracts":
			contractCount.Add(1)
			if r.Method != http.MethodPost {
				t.Errorf("POST /v1/contracts method = %s", r.Method)
			}
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&request); err != nil {
				t.Errorf("decode contract request: %v", err)
				http.Error(w, "bad json", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"018f0000-0000-7000-8000-000000000009","repo_binding_id":"018f0000-0000-7000-8000-000000000004","goal_id":"018f0000-0000-7000-8000-000000000006","state":"draft","current_seed_id":"018f0000-0000-7000-8000-000000000007","current_draft_id":"018f0000-0000-7000-8000-000000000008"}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, server.URL)

	output, err := runDraftJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--goal-id", "018f0000-0000-7000-8000-000000000006", "--format", "json")
	if err != nil {
		t.Fatalf("Run(contract draft) error = %v", err)
	}

	if output.SchemaVersion != "goalrail.cli.v1" {
		t.Fatalf("schema_version = %q, want goalrail.cli.v1", output.SchemaVersion)
	}
	if output.GoalID != "018f0000-0000-7000-8000-000000000006" || output.ContractID != "018f0000-0000-7000-8000-000000000009" {
		t.Fatalf("goal/contract = %q/%q, want server ids", output.GoalID, output.ContractID)
	}
	if output.ContractState != "draft" {
		t.Fatalf("contract_state = %q, want draft", output.ContractState)
	}
	if output.Display.Summary == "" {
		t.Fatal("display.summary is empty")
	}
	if output.LocalRepoReceipt.RepoBindingID != "018f0000-0000-7000-8000-000000000004" {
		t.Fatalf("receipt repo_binding_id = %q, want marker repo binding", output.LocalRepoReceipt.RepoBindingID)
	}
	if output.LocalRepoReceipt.HeadSHA == "" || output.LocalRepoReceipt.BaselineID == "" || output.LocalRepoReceipt.OverlayID == "" {
		t.Fatalf("receipt ids = %#v, want head/baseline/overlay", output.LocalRepoReceipt)
	}
	if output.LocalRepoReceipt.Dirty || output.LocalRepoReceipt.Partial {
		t.Fatalf("receipt dirty/partial = %t/%t, want clean/full", output.LocalRepoReceipt.Dirty, output.LocalRepoReceipt.Partial)
	}
	if !output.LocalRepoReceipt.BaselineRebuilt {
		t.Fatal("receipt baseline_rebuilt = false, want true on first draft")
	}
	if output.LocalRepoReceipt.RawSourceUploaded {
		t.Fatal("receipt raw_source_uploaded = true, want false")
	}
	if output.NextAction.Kind != "update_contract" || output.NextAction.Available || output.NextAction.PlannedSlice != "E" {
		t.Fatalf("next_action = %#v, want unavailable update_contract planned for E", output.NextAction)
	}
	wantCommand := "goalrail contract update --contract-id 018f0000-0000-7000-8000-000000000009 --fields-file - --format json"
	if output.NextAction.Command != wantCommand {
		t.Fatalf("next_action.command = %q, want %q", output.NextAction.Command, wantCommand)
	}
	if request.GoalID != "018f0000-0000-7000-8000-000000000006" {
		t.Fatalf("contract request goal_id = %q, want goal id", request.GoalID)
	}
	if request.ProjectID != "018f0000-0000-7000-8000-000000000003" || request.RepoBindingID != "018f0000-0000-7000-8000-000000000004" {
		t.Fatalf("contract request project/repo = %q/%q, want local marker context", request.ProjectID, request.RepoBindingID)
	}
	if meCount.Load() != 1 || contractCount.Load() != 1 {
		t.Fatalf("request counts me/contracts = %d/%d, want 1/1", meCount.Load(), contractCount.Load())
	}
}

func TestRunDraftMissingProjectConfigFailsBeforeHTTP(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestCount.Add(1)
		http.Error(w, "unexpected request", http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := runDraftJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--goal-id", "018f0000-0000-7000-8000-000000000006", "--format", "json")
	if err == nil {
		t.Fatal("Run(contract draft) error = nil, want missing marker")
	}
	if got := exitcode.ForError(err); got != exitcode.Usage {
		t.Fatalf("exit code = %d, want usage", got)
	}
	if !strings.Contains(err.Error(), "missing .goalrail/project.yml") {
		t.Fatalf("error = %q, want missing marker hint", err.Error())
	}
	if got := requestCount.Load(); got != 0 {
		t.Fatalf("server requests = %d, want 0 without marker", got)
	}
}

func TestRunDraftMissingProjectBindingFailsBeforeHTTP(t *testing.T) {
	t.Parallel()
	requireGit(t)

	tests := []struct {
		name          string
		projectID     string
		repoBindingID string
		want          string
	}{
		{
			name:          "project id",
			projectID:     "",
			repoBindingID: "018f0000-0000-7000-8000-000000000004",
			want:          "missing project_id",
		},
		{
			name:          "repo binding id",
			projectID:     "018f0000-0000-7000-8000-000000000003",
			repoBindingID: "",
			want:          "missing repo_binding_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoDir := setupGitRepo(t)
			var requestCount atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				requestCount.Add(1)
				http.Error(w, "unexpected request", http.StatusInternalServerError)
			}))
			defer server.Close()
			writeProjectConfigFixtureWithIDs(t, repoDir, server.URL, tt.projectID, tt.repoBindingID)

			_, err := runDraftJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--goal-id", "018f0000-0000-7000-8000-000000000006", "--format", "json")
			if err == nil {
				t.Fatalf("Run(contract draft) error = nil, want %s", tt.want)
			}
			if got := exitcode.ForError(err); got != exitcode.Validation {
				t.Fatalf("exit code = %d, want validation", got)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want %q", err.Error(), tt.want)
			}
			if got := requestCount.Load(); got != 0 {
				t.Fatalf("server requests = %d, want 0 without complete marker binding", got)
			}
		})
	}
}

func TestRunDraftMissingAuthFailsBeforeHTTP(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestCount.Add(1)
		http.Error(w, "unexpected request", http.StatusInternalServerError)
	}))
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, server.URL)

	_, err := runDraftJSON(t, repoDir, fakeSessionStore{err: authstore.ErrSessionNotFound}, "--goal-id", "018f0000-0000-7000-8000-000000000006", "--format", "json")
	if err == nil {
		t.Fatal("Run(contract draft) error = nil, want missing auth")
	}
	if got := exitcode.ForError(err); got != exitcode.Usage {
		t.Fatalf("exit code = %d, want usage", got)
	}
	if !strings.Contains(err.Error(), "not logged in; run goalrail login <server_url>") {
		t.Fatalf("error = %q, want login hint", err.Error())
	}
	if got := requestCount.Load(); got != 0 {
		t.Fatalf("server requests = %d, want 0 without auth", got)
	}
}

func TestRunDraftOrganizationMismatchFailsBeforeMutation(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	var meCount, contractCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer access-token" {
			t.Errorf("Authorization = %q, want bearer token", r.Header.Get("Authorization"))
			http.Error(w, "bad auth", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/v1/me":
			meCount.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"user":{"id":"018f0000-0000-7000-8000-000000000001","display_name":"Developer"},"organization_membership":{"organization_id":"018f0000-0000-7000-8000-000000009999","role":"member","state":"active"}}`))
		case "/v1/contracts":
			contractCount.Add(1)
			http.Error(w, "unexpected mutation", http.StatusInternalServerError)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, server.URL)

	_, err := runDraftJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--goal-id", "018f0000-0000-7000-8000-000000000006", "--format", "json")
	if err == nil {
		t.Fatal("Run(contract draft) error = nil, want organization mismatch")
	}
	if got := exitcode.ForError(err); got != exitcode.Validation {
		t.Fatalf("exit code = %d, want validation", got)
	}
	if !strings.Contains(err.Error(), "different GoalRail organization") {
		t.Fatalf("error = %q, want organization mismatch", err.Error())
	}
	if meCount.Load() != 1 || contractCount.Load() != 0 {
		t.Fatalf("request counts me/contracts = %d/%d, want 1/0", meCount.Load(), contractCount.Load())
	}
}

func TestRunDraftMalformedGoalIDMapsValidationError(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer access-token" {
			t.Errorf("Authorization = %q, want bearer token", r.Header.Get("Authorization"))
			http.Error(w, "bad auth", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/v1/me":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"user":{"id":"018f0000-0000-7000-8000-000000000001","display_name":"Developer"},"organization_membership":{"organization_id":"018f0000-0000-7000-8000-000000000002","role":"member","state":"active"}}`))
		case "/v1/contracts":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":{"code":"validation_failed","message":"goal_id: must be a UUID"}}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, server.URL)

	_, err := runDraftJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--goal-id", "not-a-uuid", "--format", "json")
	if err == nil {
		t.Fatal("Run(contract draft) error = nil, want validation error")
	}
	if got := exitcode.ForError(err); got != exitcode.Validation {
		t.Fatalf("exit code = %d, want validation", got)
	}
	if !strings.Contains(err.Error(), "contract draft request validation failed") {
		t.Fatalf("error = %q, want validation mapping", err.Error())
	}
}

func TestRunDraftRefreshesLocalReceiptBeforeMutation(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	var meCount, contractCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer access-token" {
			t.Errorf("Authorization = %q, want bearer token", r.Header.Get("Authorization"))
			http.Error(w, "bad auth", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/v1/me":
			meCount.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"user":{"id":"018f0000-0000-7000-8000-000000000001","display_name":"Developer"},"organization_membership":{"organization_id":"018f0000-0000-7000-8000-000000000002","role":"member","state":"active"}}`))
		case "/v1/contracts":
			contractCount.Add(1)
			http.Error(w, "unexpected mutation", http.StatusInternalServerError)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, server.URL)
	cacheRoot := filepath.Join(repoDir, "cache-root-file")
	if err := os.WriteFile(cacheRoot, []byte("not a directory\n"), 0o644); err != nil {
		t.Fatalf("write cache root file: %v", err)
	}

	var stdout, stderr bytes.Buffer
	err := RunWithOptions(context.Background(), term.New(&stdout, &stderr), repoDir, []string{"draft", "--goal-id", "018f0000-0000-7000-8000-000000000006", "--format", "json"}, Options{
		Store:     fakeSessionStore{session: validSession(server.URL)},
		CacheRoot: cacheRoot,
		Now:       func() time.Time { return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC) },
	})
	if err == nil {
		t.Fatal("Run(contract draft) error = nil, want local receipt cache failure")
	}
	if got := exitcode.ForError(err); got != exitcode.Runtime {
		t.Fatalf("exit code = %d, want runtime", got)
	}
	if meCount.Load() != 1 || contractCount.Load() != 0 {
		t.Fatalf("request counts me/contracts = %d/%d, want 1/0", meCount.Load(), contractCount.Load())
	}
}

func TestRunDraftRejectsContractRepoBindingMismatchResponse(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer access-token" {
			t.Errorf("Authorization = %q, want bearer token", r.Header.Get("Authorization"))
			http.Error(w, "bad auth", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/v1/me":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"user":{"id":"018f0000-0000-7000-8000-000000000001","display_name":"Developer"},"organization_membership":{"organization_id":"018f0000-0000-7000-8000-000000000002","role":"member","state":"active"}}`))
		case "/v1/contracts":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"018f0000-0000-7000-8000-000000000009","repo_binding_id":"018f0000-0000-7000-8000-000000009999","goal_id":"018f0000-0000-7000-8000-000000000006","state":"draft","current_seed_id":"018f0000-0000-7000-8000-000000000007","current_draft_id":"018f0000-0000-7000-8000-000000000008"}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, server.URL)

	_, err := runDraftJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--goal-id", "018f0000-0000-7000-8000-000000000006", "--format", "json")
	if err == nil {
		t.Fatal("Run(contract draft) error = nil, want response repo binding mismatch")
	}
	if got := exitcode.ForError(err); got != exitcode.Validation {
		t.Fatalf("exit code = %d, want validation", got)
	}
	if !strings.Contains(err.Error(), "repo_binding_id does not match") {
		t.Fatalf("error = %q, want repo binding mismatch", err.Error())
	}
}

func TestRunDraftTextIsHumanSafe(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer access-token" {
			t.Errorf("Authorization = %q, want bearer token", r.Header.Get("Authorization"))
			http.Error(w, "bad auth", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/v1/me":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"user":{"id":"018f0000-0000-7000-8000-000000000001","display_name":"Developer"},"organization_membership":{"organization_id":"018f0000-0000-7000-8000-000000000002","role":"member","state":"active"}}`))
		case "/v1/contracts":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"018f0000-0000-7000-8000-000000000009","repo_binding_id":"018f0000-0000-7000-8000-000000000004","goal_id":"018f0000-0000-7000-8000-000000000006","state":"draft","current_seed_id":"018f0000-0000-7000-8000-000000000007","current_draft_id":"018f0000-0000-7000-8000-000000000008"}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, server.URL)

	var stdout, stderr bytes.Buffer
	err := RunWithOptions(context.Background(), term.New(&stdout, &stderr), repoDir, []string{"draft", "--goal-id", "018f0000-0000-7000-8000-000000000006"}, Options{
		Store:     fakeSessionStore{session: validSession(server.URL)},
		CacheRoot: t.TempDir(),
		Now:       func() time.Time { return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("Run(contract draft text) error = %v", err)
	}
	got := stdout.String()
	if !strings.Contains(got, "Next planned command, not available yet") {
		t.Fatalf("stdout = %q, want unavailable update_contract warning", got)
	}
	for _, forbidden := range []string{"workitem", "runner", "proof", "verified"} {
		if strings.Contains(strings.ToLower(got), forbidden) {
			t.Fatalf("stdout = %q, want no %q claim", got, forbidden)
		}
	}
}

func runDraftJSON(t *testing.T, repoDir string, store fakeSessionStore, args ...string) (spine.ContractDraftOutput, error) {
	t.Helper()

	var stdout, stderr bytes.Buffer
	err := RunWithOptions(context.Background(), term.New(&stdout, &stderr), repoDir, append([]string{"draft"}, args...), Options{
		Store:     store,
		CacheRoot: t.TempDir(),
		Now:       func() time.Time { return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		return spine.ContractDraftOutput{}, err
	}
	var output spine.ContractDraftOutput
	decoder := json.NewDecoder(&stdout)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&output); err != nil {
		t.Fatalf("decode contract draft JSON %q: %v", stdout.String(), err)
	}
	return output, nil
}

func setupGitRepo(t *testing.T) string {
	t.Helper()

	repoDir := t.TempDir()
	runGit(t, repoDir, "init")
	runGit(t, repoDir, "-c", "user.name=Goalrail Test", "-c", "user.email=goalrail@example.test", "commit", "--allow-empty", "-m", "initial")
	return repoDir
}

func writeProjectConfigFixture(t *testing.T, repoDir string, serverURL string) {
	t.Helper()
	writeProjectConfigFixtureWithIDs(t, repoDir, serverURL, "018f0000-0000-7000-8000-000000000003", "018f0000-0000-7000-8000-000000000004")
}

func writeProjectConfigFixtureWithIDs(t *testing.T, repoDir string, serverURL string, projectID string, repoBindingID string) {
	t.Helper()

	_, err := projectconfig.Write(repoDir, projectconfig.Config{
		Version:        projectconfig.Version,
		ServerURL:      serverURL,
		OrganizationID: "018f0000-0000-7000-8000-000000000002",
		ProjectID:      projectID,
		RepoBindingID:  repoBindingID,
		Repository: projectconfig.Repository{
			Provider:           "github",
			FullName:           "heurema/goalrail",
			URL:                "git@github.com:heurema/goalrail.git",
			WorkflowBaseBranch: "main",
		},
	})
	if err != nil {
		t.Fatalf("write project config: %v", err)
	}
}

func validSession(serverURL string) authstore.Session {
	return authstore.Session{
		ServerURL:            serverURL,
		AccessToken:          "access-token",
		RefreshToken:         "refresh-token",
		AccessTokenExpiresAt: time.Date(2026, 5, 5, 11, 0, 0, 0, time.UTC),
		TokenType:            "Bearer",
	}
}

type fakeSessionStore struct {
	session authstore.Session
	err     error
}

func (s fakeSessionStore) Load() (authstore.Session, error) {
	if s.err != nil {
		return authstore.Session{}, s.err
	}
	return s.session, nil
}

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(output))
	}
}
