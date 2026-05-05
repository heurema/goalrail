package initcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	"github.com/heurema/goalrail/apps/cli/internal/spine"
	"github.com/heurema/goalrail/apps/cli/internal/term"
)

func TestRunExplicitRepoOutsideGitJSON(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	draft, err := runInitJSON(t, tempDir, "--repo", "git@github.com:acme/payments.git", "--format", "json")
	if err != nil {
		t.Fatalf("Run(init --repo) error = %v", err)
	}

	if draft.RepoURL != "git@github.com:acme/payments.git" {
		t.Fatalf("RepoURL = %q, want explicit repo", draft.RepoURL)
	}
	if draft.Provider != "github" {
		t.Fatalf("Provider = %q, want github", draft.Provider)
	}
	if draft.ProviderHost != "github.com" {
		t.Fatalf("ProviderHost = %q, want github.com", draft.ProviderHost)
	}
	if draft.RepositoryFullName != "acme/payments" {
		t.Fatalf("RepositoryFullName = %q, want acme/payments", draft.RepositoryFullName)
	}
	if draft.GitRoot != "" {
		t.Fatalf("GitRoot = %q, want empty outside git", draft.GitRoot)
	}
	if len(draft.Warnings) != 0 {
		t.Fatalf("Warnings = %v, want empty outside git", draft.Warnings)
	}
}

func TestRunDiscoversOriginInsideGitRepoJSON(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := t.TempDir()
	runGit(t, repoDir, "init")
	runGit(t, repoDir, "remote", "add", "origin", "https://github.com/acme/payments.git")

	draft, err := runInitJSON(t, repoDir, "--format", "json")
	if err != nil {
		t.Fatalf("Run(init) error = %v", err)
	}

	if draft.RepoURL != "https://github.com/acme/payments.git" {
		t.Fatalf("RepoURL = %q, want origin URL", draft.RepoURL)
	}
	if draft.GitRoot != canonicalPath(t, repoDir) {
		t.Fatalf("GitRoot = %q, want %q", draft.GitRoot, canonicalPath(t, repoDir))
	}
	if draft.RemoteName != "origin" {
		t.Fatalf("RemoteName = %q, want origin", draft.RemoteName)
	}
	if draft.Provider != "github" {
		t.Fatalf("Provider = %q, want github", draft.Provider)
	}
	if draft.RepositoryFullName != "acme/payments" {
		t.Fatalf("RepositoryFullName = %q, want acme/payments", draft.RepositoryFullName)
	}
}

func TestRunDetectsOriginHeadMainAsWorkflowBaseBranch(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := t.TempDir()
	runGit(t, repoDir, "init")
	runGit(t, repoDir, "-c", "user.name=Goalrail Test", "-c", "user.email=goalrail@example.test", "commit", "--allow-empty", "-m", "initial")
	headSHA := strings.TrimSpace(runGitOutput(t, repoDir, "rev-parse", "--verify", "HEAD"))
	runGit(t, repoDir, "remote", "add", "origin", "git@github.com:acme/payments.git")
	runGit(t, repoDir, "update-ref", "refs/remotes/origin/main", "HEAD")
	runGit(t, repoDir, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")

	draft, err := runInitJSON(t, repoDir, "--format", "json")
	if err != nil {
		t.Fatalf("Run(init) error = %v", err)
	}

	if draft.WorkflowBaseBranch != "main" {
		t.Fatalf("WorkflowBaseBranch = %q, want main", draft.WorkflowBaseBranch)
	}
	if draft.HeadSHA != headSHA {
		t.Fatalf("HeadSHA = %q, want %q", draft.HeadSHA, headSHA)
	}
	if len(draft.Warnings) != 0 {
		t.Fatalf("Warnings = %v, want empty with origin/HEAD", draft.Warnings)
	}
}

func TestRunFallsBackToOriginMainRemoteRef(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := t.TempDir()
	runGit(t, repoDir, "init")
	runGit(t, repoDir, "-c", "user.name=Goalrail Test", "-c", "user.email=goalrail@example.test", "commit", "--allow-empty", "-m", "initial")
	runGit(t, repoDir, "remote", "add", "origin", "https://github.com/acme/payments.git")
	runGit(t, repoDir, "update-ref", "refs/remotes/origin/main", "HEAD")

	draft, err := runInitJSON(t, repoDir, "--format", "json")
	if err != nil {
		t.Fatalf("Run(init) error = %v", err)
	}

	if draft.WorkflowBaseBranch != "main" {
		t.Fatalf("WorkflowBaseBranch = %q, want main", draft.WorkflowBaseBranch)
	}
}

func TestRunDoesNotUseCurrentLocalBranchAsWorkflowBase(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := t.TempDir()
	runGit(t, repoDir, "init")
	runGit(t, repoDir, "checkout", "-b", "main")
	runGit(t, repoDir, "-c", "user.name=Goalrail Test", "-c", "user.email=goalrail@example.test", "commit", "--allow-empty", "-m", "initial")
	runGit(t, repoDir, "remote", "add", "origin", "https://github.com/acme/payments.git")

	draft, err := runInitJSON(t, repoDir, "--format", "json")
	if err != nil {
		t.Fatalf("Run(init) error = %v", err)
	}

	if draft.WorkflowBaseBranch != "" {
		t.Fatalf("WorkflowBaseBranch = %q, want empty without origin metadata", draft.WorkflowBaseBranch)
	}
	if len(draft.Warnings) == 0 {
		t.Fatal("Warnings = empty, want missing workflow base warning")
	}
}

func TestRunNoRepoAndNoGitContextReturnsUsageError(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), term.New(&stdout, &stderr), t.TempDir(), []string{"--format", "json"})
	if err == nil {
		t.Fatal("Run(init) error = nil, want usage error")
	}
	if got := exitcode.ForError(err); got != exitcode.Usage {
		t.Fatalf("exit code = %d, want %d", got, exitcode.Usage)
	}
	if !strings.Contains(err.Error(), "missing --repo and no git remote origin was detected") {
		t.Fatalf("error = %q, want helpful missing repo message", err.Error())
	}
}

func TestRunServerBackedInitSendsExpectedRequestJSON(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir, headSHA := setupGitRepoWithOriginHead(t, "git@github.com:heurema/goalrail.git")
	gitignorePath := filepath.Join(repoDir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte("existing-ignore\n"), 0o644); err != nil {
		t.Fatalf("write .gitignore fixture: %v", err)
	}
	projectID := "018f0000-0000-7000-8000-000000000003"
	var received spine.RepoBindingInitRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
			http.Error(w, "bad method", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/v1/projects/"+projectID+"/repo-bindings/init" {
			t.Errorf("path = %s, want project repo binding init path", r.URL.Path)
			http.Error(w, "bad path", http.StatusNotFound)
			return
		}
		if r.Header.Get("Authorization") != "Bearer access-token" {
			t.Errorf("Authorization = %q, want bearer token", r.Header.Get("Authorization"))
			http.Error(w, "bad auth", http.StatusUnauthorized)
			return
		}
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&received); err != nil {
			t.Errorf("decode request body: %v", err)
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"repo_binding_id":"018f0000-0000-7000-8000-000000000004","project_id":"018f0000-0000-7000-8000-000000000003","organization_id":"018f0000-0000-7000-8000-000000000002","provider":"github","repository_full_name":"heurema/goalrail","repository_url":"git@github.com:heurema/goalrail.git","provider_default_branch":"main","workflow_base_branch":"main","state":"active","created":true,"message":"Repository binding initialized."}`))
	}))
	defer server.Close()

	output, err := runInitServerJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--project", projectID, "--format", "json")
	if err != nil {
		t.Fatalf("Run(init --project) error = %v", err)
	}

	if output.Mode != "server" || output.ServerURL != server.URL {
		t.Fatalf("output mode/server = %q/%q, want server/%q", output.Mode, output.ServerURL, server.URL)
	}
	if output.RepoBindingID != "018f0000-0000-7000-8000-000000000004" {
		t.Fatalf("repo_binding_id = %q, want server id", output.RepoBindingID)
	}
	if output.LocalConfigPath != projectConfigRelativePath || output.LocalConfigStatus != localConfigStatusWritten {
		t.Fatalf("local config output = %q/%q, want %q/%q", output.LocalConfigPath, output.LocalConfigStatus, projectConfigRelativePath, localConfigStatusWritten)
	}
	if received.Provider != "github" || received.RepositoryFullName != "heurema/goalrail" || received.RepositoryURL != "git@github.com:heurema/goalrail.git" {
		t.Fatalf("request repo fields = %#v, want GitHub goalrail repo", received)
	}
	if received.ProviderDefaultBranch != "main" || received.WorkflowBaseBranch != "main" {
		t.Fatalf("request branch fields = %#v, want main/main", received)
	}
	if received.LocalRemoteName != "origin" || received.LocalHeadSHA != headSHA {
		t.Fatalf("request local fields = %#v, want origin/%s", received, headSHA)
	}
	config := readProjectConfigFile(t, repoDir)
	wantConfig := expectedProjectConfigYAML(server.URL)
	if config != wantConfig {
		t.Fatalf("project config =\n%s\nwant:\n%s", config, wantConfig)
	}
	if strings.Contains(config, "access-token") || strings.Contains(config, "refresh-token") {
		t.Fatalf("project config contains token material:\n%s", config)
	}
	gitignore, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	if string(gitignore) != "existing-ignore\n" {
		t.Fatalf(".gitignore = %q, want unchanged", string(gitignore))
	}
}

func TestRunServerBackedInitFromNestedDirectoryWritesGitRootConfig(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir, _ := setupGitRepoWithOriginHead(t, "git@github.com:heurema/goalrail.git")
	nestedDir := filepath.Join(repoDir, "nested", "deeper")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatalf("create nested dir: %v", err)
	}
	server := httptest.NewServer(repoBindingInitHandler(t, "018f0000-0000-7000-8000-000000000003"))
	defer server.Close()

	output, err := runInitServerJSON(t, nestedDir, fakeSessionStore{session: validSession(server.URL)}, "--project", "018f0000-0000-7000-8000-000000000003", "--format", "json")
	if err != nil {
		t.Fatalf("Run(init --project from nested dir) error = %v", err)
	}

	if output.LocalConfigStatus != localConfigStatusWritten {
		t.Fatalf("local_config_status = %q, want written", output.LocalConfigStatus)
	}
	if _, err := os.Stat(filepath.Join(repoDir, projectConfigRelativePath)); err != nil {
		t.Fatalf("git root project config not written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(nestedDir, projectConfigRelativePath)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("nested project config stat error = %v, want not exist", err)
	}
}

func TestRunServerBackedInitIdenticalConfigReportsUnchanged(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir, _ := setupGitRepoWithOriginHead(t, "git@github.com:heurema/goalrail.git")
	server := httptest.NewServer(repoBindingInitHandler(t, "018f0000-0000-7000-8000-000000000003"))
	defer server.Close()

	if _, err := runInitServerJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--project", "018f0000-0000-7000-8000-000000000003", "--format", "json"); err != nil {
		t.Fatalf("first Run(init --project) error = %v", err)
	}
	output, err := runInitServerJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--project", "018f0000-0000-7000-8000-000000000003", "--format", "json")
	if err != nil {
		t.Fatalf("second Run(init --project) error = %v", err)
	}
	if output.LocalConfigStatus != localConfigStatusUnchanged {
		t.Fatalf("local_config_status = %q, want unchanged", output.LocalConfigStatus)
	}
}

func TestRunServerBackedInitDifferentExistingFullConfigFails(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir, _ := setupGitRepoWithOriginHead(t, "git@github.com:heurema/goalrail.git")
	projectID := "018f0000-0000-7000-8000-000000000003"
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		repoBindingInitHandler(t, projectID)(w, r)
	}))
	defer server.Close()
	existing := projectConfigFixture(server.URL)
	existing.RepoBindingID = "018f0000-0000-7000-8000-000000000999"
	original := writeProjectConfigFixture(t, repoDir, existing)

	_, err := runInitServerJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--project", projectID, "--format", "json")
	if err == nil {
		t.Fatal("Run(init --project) error = nil, want config conflict")
	}
	if got := exitcode.ForError(err); got != exitcode.Validation {
		t.Fatalf("exit code = %d, want validation", got)
	}
	if !strings.Contains(err.Error(), "local .goalrail/project.yml already exists with different content") {
		t.Fatalf("error = %q, want config conflict", err.Error())
	}
	if got := requestCount.Load(); got != 1 {
		t.Fatalf("server requests = %d, want 1 for post-response full-content conflict", got)
	}
	if got := readProjectConfigFile(t, repoDir); got != original {
		t.Fatalf("project config overwritten = %q, want original %q", got, original)
	}
}

func TestRunServerBackedInitPreflightConflictSkipsServer(t *testing.T) {
	t.Parallel()
	requireGit(t)

	tests := []struct {
		name   string
		mutate func(projectConfig) projectConfig
	}{
		{
			name: "server_url",
			mutate: func(config projectConfig) projectConfig {
				config.ServerURL = "https://other.example.com"
				return config
			},
		},
		{
			name: "project_id",
			mutate: func(config projectConfig) projectConfig {
				config.ProjectID = "018f0000-0000-7000-8000-000000000999"
				return config
			},
		},
		{
			name: "repository_full_name",
			mutate: func(config projectConfig) projectConfig {
				config.Repository.FullName = "heurema/other"
				return config
			},
		},
		{
			name: "repository_url",
			mutate: func(config projectConfig) projectConfig {
				config.Repository.URL = "https://github.com/heurema/goalrail.git"
				return config
			},
		},
		{
			name: "workflow_base_branch",
			mutate: func(config projectConfig) projectConfig {
				config.Repository.WorkflowBaseBranch = "release"
				return config
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repoDir, _ := setupGitRepoWithOriginHead(t, "git@github.com:heurema/goalrail.git")
			projectID := "018f0000-0000-7000-8000-000000000003"
			var requestCount atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				requestCount.Add(1)
				http.Error(w, "unexpected request", http.StatusInternalServerError)
			}))
			defer server.Close()

			original := writeProjectConfigFixture(t, repoDir, tt.mutate(projectConfigFixture(server.URL)))
			_, err := runInitServerJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--project", projectID, "--format", "json")
			if err == nil {
				t.Fatal("Run(init --project) error = nil, want preflight conflict")
			}
			if got := exitcode.ForError(err); got != exitcode.Validation {
				t.Fatalf("exit code = %d, want validation", got)
			}
			if !strings.Contains(err.Error(), projectConfigConflictMessage) {
				t.Fatalf("error = %q, want preflight conflict", err.Error())
			}
			if got := requestCount.Load(); got != 0 {
				t.Fatalf("server requests = %d, want 0 for preflight conflict", got)
			}
			if got := readProjectConfigFile(t, repoDir); got != original {
				t.Fatalf("project config overwritten = %q, want original %q", got, original)
			}
		})
	}
}

func TestRunServerBackedInitUnparseableProjectConfigSkipsServer(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir, _ := setupGitRepoWithOriginHead(t, "git@github.com:heurema/goalrail.git")
	original := writeRawProjectConfigFile(t, repoDir, "not a GoalRail project marker\n")
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestCount.Add(1)
		http.Error(w, "unexpected request", http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := runInitServerJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--project", "018f0000-0000-7000-8000-000000000003", "--format", "json")
	if err == nil {
		t.Fatal("Run(init --project) error = nil, want unparseable config")
	}
	if got := exitcode.ForError(err); got != exitcode.Validation {
		t.Fatalf("exit code = %d, want validation", got)
	}
	if !strings.Contains(err.Error(), projectConfigUnparseableMessage) {
		t.Fatalf("error = %q, want unparseable config", err.Error())
	}
	if got := requestCount.Load(); got != 0 {
		t.Fatalf("server requests = %d, want 0 for unparseable config", got)
	}
	if got := readProjectConfigFile(t, repoDir); got != original {
		t.Fatalf("project config overwritten = %q, want original %q", got, original)
	}
}

func TestRunServerBackedInitMatchingExistingConfigContinuesAndReportsUnchanged(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir, _ := setupGitRepoWithOriginHead(t, "git@github.com:heurema/goalrail.git")
	projectID := "018f0000-0000-7000-8000-000000000003"
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		repoBindingInitHandler(t, projectID)(w, r)
	}))
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, projectConfigFixture(server.URL))

	output, err := runInitServerJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--project", projectID, "--format", "json")
	if err != nil {
		t.Fatalf("Run(init --project) error = %v", err)
	}
	if output.LocalConfigStatus != localConfigStatusUnchanged {
		t.Fatalf("local_config_status = %q, want unchanged", output.LocalConfigStatus)
	}
	if got := requestCount.Load(); got != 1 {
		t.Fatalf("server requests = %d, want 1 for matching config", got)
	}
}

func TestRunLocalDemoDoesNotWriteProjectConfig(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir, _ := setupGitRepoWithOriginHead(t, "git@github.com:heurema/goalrail.git")
	if _, err := runInitJSON(t, repoDir, "--repo", "git@github.com:acme/payments.git", "--format", "json"); err != nil {
		t.Fatalf("Run(init --repo) error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(repoDir, projectConfigRelativePath)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("project config stat error = %v, want not exist", err)
	}
}

func TestRunLocalDemoIgnoresExistingProjectConfig(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir, _ := setupGitRepoWithOriginHead(t, "git@github.com:heurema/goalrail.git")
	original := writeRawProjectConfigFile(t, repoDir, "not a GoalRail project marker\n")
	if _, err := runInitJSON(t, repoDir, "--repo", "git@github.com:acme/payments.git", "--format", "json"); err != nil {
		t.Fatalf("Run(init --repo) error = %v", err)
	}
	if got := readProjectConfigFile(t, repoDir); got != original {
		t.Fatalf("project config changed = %q, want original %q", got, original)
	}
}

func TestRunServerBackedInitMissingAuthReturnsHelpfulError(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir, _ := setupGitRepoWithOriginHead(t, "git@github.com:heurema/goalrail.git")
	_, err := runInitServerJSON(t, repoDir, fakeSessionStore{err: authstore.ErrSessionNotFound}, "--project", "018f0000-0000-7000-8000-000000000003", "--format", "json")
	if err == nil {
		t.Fatal("Run(init --project) error = nil, want missing auth")
	}
	if got := exitcode.ForError(err); got != exitcode.Usage {
		t.Fatalf("exit code = %d, want usage", got)
	}
	if !strings.Contains(err.Error(), "not logged in; run goalrail login <server_url>") {
		t.Fatalf("error = %q, want login hint", err.Error())
	}
}

func TestRunServerBackedInitRequiresGitRoot(t *testing.T) {
	t.Parallel()

	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestCount.Add(1)
		http.Error(w, "unexpected request", http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := runInitServerJSON(t, t.TempDir(), fakeSessionStore{session: validSession(server.URL)}, "--repo", "git@github.com:heurema/goalrail.git", "--project", "018f0000-0000-7000-8000-000000000003", "--format", "json")
	if err == nil {
		t.Fatal("Run(init --project) error = nil, want missing git root")
	}
	if got := exitcode.ForError(err); got != exitcode.Usage {
		t.Fatalf("exit code = %d, want usage", got)
	}
	if !strings.Contains(err.Error(), "server-backed init requires a Git root") {
		t.Fatalf("error = %q, want Git root error", err.Error())
	}
	if got := requestCount.Load(); got != 0 {
		t.Fatalf("server requests = %d, want 0 without Git root", got)
	}
}

func TestRunServerBackedInitExpiredTokenFailsBeforeHTTP(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir, _ := setupGitRepoWithOriginHead(t, "git@github.com:heurema/goalrail.git")
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestCount.Add(1)
		http.Error(w, "unexpected request", http.StatusInternalServerError)
	}))
	defer server.Close()

	session := validSession(server.URL)
	session.AccessTokenExpiresAt = time.Date(2026, 5, 5, 9, 59, 59, 0, time.UTC)
	_, err := runInitServerJSON(t, repoDir, fakeSessionStore{session: session}, "--project", "018f0000-0000-7000-8000-000000000003", "--format", "json")
	if err == nil {
		t.Fatal("Run(init --project) error = nil, want expired login")
	}
	if got := exitcode.ForError(err); got != exitcode.Usage {
		t.Fatalf("exit code = %d, want usage", got)
	}
	if !strings.Contains(err.Error(), "login expired; run goalrail login "+server.URL) {
		t.Fatalf("error = %q, want expired login hint", err.Error())
	}
	if got := requestCount.Load(); got != 0 {
		t.Fatalf("server requests = %d, want 0 for expired token", got)
	}
}

func TestRunServerBackedInitMapsConflictError(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir, _ := setupGitRepoWithOriginHead(t, "git@github.com:heurema/goalrail.git")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error":{"code":"repo_binding_conflict","message":"project already has active repo binding for a different repository"}}`))
	}))
	defer server.Close()

	_, err := runInitServerJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--project", "018f0000-0000-7000-8000-000000000003", "--format", "json")
	if err == nil {
		t.Fatal("Run(init --project) error = nil, want conflict")
	}
	if got := exitcode.ForError(err); got != exitcode.Validation {
		t.Fatalf("exit code = %d, want validation", got)
	}
	if !strings.Contains(err.Error(), "repo_binding_conflict") {
		t.Fatalf("error = %q, want server conflict code", err.Error())
	}
}

func TestRunHelpUsage(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	if err := Run(context.Background(), term.New(&stdout, &stderr), t.TempDir(), []string{"--help"}); err != nil {
		t.Fatalf("Run(init --help) error = %v", err)
	}
	if got := stdout.String(); !strings.Contains(got, "Usage: goalrail init [--repo <repo-url>] [--project <project-id>] [--format text|json]") {
		t.Fatalf("stdout = %q, want init usage", got)
	}
}

func runInitJSON(t *testing.T, workDir string, args ...string) (spine.RepoBindingDraft, error) {
	t.Helper()

	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), term.New(&stdout, &stderr), workDir, args)
	if err != nil {
		return spine.RepoBindingDraft{}, err
	}

	var draft spine.RepoBindingDraft
	decoder := json.NewDecoder(&stdout)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&draft); err != nil {
		t.Fatalf("decode init JSON %q: %v", stdout.String(), err)
	}
	return draft, nil
}

func runInitServerJSON(t *testing.T, workDir string, store fakeSessionStore, args ...string) (spine.RepoBindingInitOutput, error) {
	t.Helper()

	var stdout, stderr bytes.Buffer
	err := RunWithOptions(context.Background(), term.New(&stdout, &stderr), workDir, args, Options{
		Store: store,
		Now:   func() time.Time { return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		return spine.RepoBindingInitOutput{}, err
	}

	var output spine.RepoBindingInitOutput
	decoder := json.NewDecoder(&stdout)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&output); err != nil {
		t.Fatalf("decode init server JSON %q: %v", stdout.String(), err)
	}
	return output, nil
}

func repoBindingInitHandler(t *testing.T, projectID string) http.HandlerFunc {
	t.Helper()

	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/projects/"+projectID+"/repo-bindings/init" {
			t.Errorf("path = %s, want project repo binding init path", r.URL.Path)
			http.Error(w, "bad path", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"repo_binding_id":"018f0000-0000-7000-8000-000000000004","project_id":"018f0000-0000-7000-8000-000000000003","organization_id":"018f0000-0000-7000-8000-000000000002","provider":"github","repository_full_name":"heurema/goalrail","repository_url":"git@github.com:heurema/goalrail.git","provider_default_branch":"main","workflow_base_branch":"main","state":"active","created":true,"message":"Repository binding initialized."}`))
	}
}

func projectConfigFixture(serverURL string) projectConfig {
	return projectConfig{
		Version:        projectConfigVersion,
		ServerURL:      serverURL,
		OrganizationID: "018f0000-0000-7000-8000-000000000002",
		ProjectID:      "018f0000-0000-7000-8000-000000000003",
		RepoBindingID:  "018f0000-0000-7000-8000-000000000004",
		Repository: projectConfigRepository{
			Provider:           "github",
			FullName:           "heurema/goalrail",
			URL:                "git@github.com:heurema/goalrail.git",
			WorkflowBaseBranch: "main",
		},
	}
}

func writeProjectConfigFixture(t *testing.T, repoDir string, config projectConfig) string {
	t.Helper()

	return writeRawProjectConfigFile(t, repoDir, renderProjectConfigYAML(config))
}

func writeRawProjectConfigFile(t *testing.T, repoDir string, content string) string {
	t.Helper()

	configPath := filepath.Join(repoDir, projectConfigRelativePath)
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("create .goalrail dir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write project config: %v", err)
	}
	return content
}

func readProjectConfigFile(t *testing.T, repoDir string) string {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(repoDir, projectConfigRelativePath))
	if err != nil {
		t.Fatalf("read project config: %v", err)
	}
	return string(raw)
}

func expectedProjectConfigYAML(serverURL string) string {
	return `version: 1
server_url: "` + serverURL + `"
organization_id: "018f0000-0000-7000-8000-000000000002"
project_id: "018f0000-0000-7000-8000-000000000003"
repo_binding_id: "018f0000-0000-7000-8000-000000000004"

repository:
  provider: "github"
  full_name: "heurema/goalrail"
  url: "git@github.com:heurema/goalrail.git"
  workflow_base_branch: "main"
`
}

func setupGitRepoWithOriginHead(t *testing.T, remoteURL string) (string, string) {
	t.Helper()

	repoDir := t.TempDir()
	runGit(t, repoDir, "init")
	runGit(t, repoDir, "-c", "user.name=Goalrail Test", "-c", "user.email=goalrail@example.test", "commit", "--allow-empty", "-m", "initial")
	headSHA := strings.TrimSpace(runGitOutput(t, repoDir, "rev-parse", "--verify", "HEAD"))
	runGit(t, repoDir, "remote", "add", "origin", remoteURL)
	runGit(t, repoDir, "update-ref", "refs/remotes/origin/main", "HEAD")
	runGit(t, repoDir, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")
	return repoDir, headSHA
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

	cmd := exec.Command("git", append([]string{"-C", filepath.Clean(dir)}, args...)...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, output)
	}
}

func runGitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", append([]string{"-C", filepath.Clean(dir)}, args...)...)
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %s failed: %v", strings.Join(args, " "), err)
	}
	return string(output)
}

func canonicalPath(t *testing.T, path string) string {
	t.Helper()

	canonical, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatalf("canonicalize %q: %v", path, err)
	}
	return canonical
}
