package workcmd

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
	"github.com/heurema/goalrail/apps/cli/internal/projectconfig"
	"github.com/heurema/goalrail/apps/cli/internal/spine"
	"github.com/heurema/goalrail/apps/cli/internal/term"
)

func TestRunStartCreatesIntakeAndPromotesGoal(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	var intakeRequest intakeSubmission
	var meCount, intakeCount, promoteCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer access-token" {
			t.Errorf("Authorization = %q, want bearer token", r.Header.Get("Authorization"))
			http.Error(w, "bad auth", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/v1/me":
			meCount.Add(1)
			if r.Method != http.MethodGet {
				t.Errorf("GET /v1/me method = %s", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"user":{"id":"018f0000-0000-7000-8000-000000000001","display_name":"Developer"},"organization_membership":{"organization_id":"018f0000-0000-7000-8000-000000000002","role":"member"}}`))
		case "/v1/intakes":
			intakeCount.Add(1)
			if r.Method != http.MethodPost {
				t.Errorf("POST /v1/intakes method = %s", r.Method)
			}
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&intakeRequest); err != nil {
				t.Errorf("decode intake request: %v", err)
				http.Error(w, "bad json", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"intake_id":"018f0000-0000-7000-8000-000000000005","organization_id":"018f0000-0000-7000-8000-000000000002","project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":"018f0000-0000-7000-8000-000000000004","state":"received","canonical_contract_created":false,"next":"server will validate and may promote intake to goal"}`))
		case "/v1/intakes/018f0000-0000-7000-8000-000000000005/goals":
			promoteCount.Add(1)
			if r.Method != http.MethodPost {
				t.Errorf("POST /v1/intakes/{id}/goals method = %s", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"018f0000-0000-7000-8000-000000000006","intake_id":"018f0000-0000-7000-8000-000000000005","organization_id":"018f0000-0000-7000-8000-000000000002","project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":"018f0000-0000-7000-8000-000000000004","title":"Refactor CSV export filters","summary":"Preserve current behavior.","state":"created"}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, server.URL)

	output, err := runStartJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--title", "Refactor CSV export filters", "--body", "Preserve current behavior.", "--format", "json")
	if err != nil {
		t.Fatalf("Run(work start) error = %v", err)
	}

	if output.Mode != "server" || output.ServerURL != server.URL {
		t.Fatalf("output mode/server = %q/%q, want server/%q", output.Mode, output.ServerURL, server.URL)
	}
	if output.IntakeID != "018f0000-0000-7000-8000-000000000005" || output.GoalID != "018f0000-0000-7000-8000-000000000006" {
		t.Fatalf("output intake/goal = %q/%q, want server ids", output.IntakeID, output.GoalID)
	}
	if intakeRequest.ProjectID != "018f0000-0000-7000-8000-000000000003" || intakeRequest.RepoBindingID != "018f0000-0000-7000-8000-000000000004" {
		t.Fatalf("intake project context = %#v, want marker IDs", intakeRequest)
	}
	if intakeRequest.RequestAuthor.ID != "018f0000-0000-7000-8000-000000000001" || intakeRequest.RequestAuthor.DisplayName != "Developer" {
		t.Fatalf("request author = %#v, want current profile user", intakeRequest.RequestAuthor)
	}
	if intakeRequest.Source.Kind != "goalrail_cli" {
		t.Fatalf("source kind = %q, want goalrail_cli", intakeRequest.Source.Kind)
	}
	if meCount.Load() != 1 || intakeCount.Load() != 1 || promoteCount.Load() != 1 {
		t.Fatalf("request counts me/intake/promote = %d/%d/%d, want 1/1/1", meCount.Load(), intakeCount.Load(), promoteCount.Load())
	}
	config := readProjectConfigFile(t, repoDir)
	if strings.Contains(config, "access-token") || strings.Contains(config, "refresh-token") {
		t.Fatalf("project config contains token material:\n%s", config)
	}
}

func TestRunStartFromNestedDirectoryReadsGitRootConfig(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	nestedDir := filepath.Join(repoDir, "nested", "deeper")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatalf("create nested dir: %v", err)
	}
	server := newWorkStartFakeServer(t)
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, server.URL)

	output, err := runStartJSON(t, nestedDir, fakeSessionStore{session: validSession(server.URL)}, "--title", "Nested work", "--format", "json")
	if err != nil {
		t.Fatalf("Run(work start from nested dir) error = %v", err)
	}
	if output.ProjectID != "018f0000-0000-7000-8000-000000000003" {
		t.Fatalf("project_id = %q, want marker project", output.ProjectID)
	}
	if _, err := os.Stat(filepath.Join(nestedDir, projectconfig.RelativePath)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("nested project config stat error = %v, want not exist", err)
	}
}

func TestRunStartMissingProjectConfigFailsBeforeHTTP(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestCount.Add(1)
		http.Error(w, "unexpected request", http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := runStartJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--title", "Missing marker", "--format", "json")
	if err == nil {
		t.Fatal("Run(work start) error = nil, want missing marker")
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

func TestRunStartMissingAuthFailsBeforeHTTP(t *testing.T) {
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

	_, err := runStartJSON(t, repoDir, fakeSessionStore{err: authstore.ErrSessionNotFound}, "--title", "Missing login", "--format", "json")
	if err == nil {
		t.Fatal("Run(work start) error = nil, want missing auth")
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

func TestRunStartExpiredTokenFailsBeforeHTTP(t *testing.T) {
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

	session := validSession(server.URL)
	session.AccessTokenExpiresAt = time.Date(2026, 5, 5, 9, 59, 59, 0, time.UTC)
	_, err := runStartJSON(t, repoDir, fakeSessionStore{session: session}, "--title", "Expired login", "--format", "json")
	if err == nil {
		t.Fatal("Run(work start) error = nil, want expired login")
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

func TestRunStartHelpUsage(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	if err := Run(context.Background(), term.New(&stdout, &stderr), t.TempDir(), []string{"start", "--help"}); err != nil {
		t.Fatalf("Run(work start --help) error = %v", err)
	}
	if got := stdout.String(); !strings.Contains(got, "Usage: goalrail work start --title <title>") {
		t.Fatalf("stdout = %q, want work start usage", got)
	}
}

func runStartJSON(t *testing.T, workDir string, store fakeSessionStore, args ...string) (spine.WorkStartOutput, error) {
	t.Helper()

	var stdout, stderr bytes.Buffer
	err := RunWithOptions(context.Background(), term.New(&stdout, &stderr), workDir, append([]string{"start"}, args...), Options{
		Store: store,
		Now:   func() time.Time { return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		return spine.WorkStartOutput{}, err
	}

	var output spine.WorkStartOutput
	decoder := json.NewDecoder(&stdout)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&output); err != nil {
		t.Fatalf("decode work start JSON %q: %v", stdout.String(), err)
	}
	return output, nil
}

func newWorkStartFakeServer(t *testing.T) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer access-token" {
			t.Errorf("Authorization = %q, want bearer token", r.Header.Get("Authorization"))
			http.Error(w, "bad auth", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/v1/me":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"user":{"id":"018f0000-0000-7000-8000-000000000001","display_name":"Developer"},"organization_membership":{"organization_id":"018f0000-0000-7000-8000-000000000002","role":"member"}}`))
		case "/v1/intakes":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"intake_id":"018f0000-0000-7000-8000-000000000005","organization_id":"018f0000-0000-7000-8000-000000000002","project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":"018f0000-0000-7000-8000-000000000004","state":"received","canonical_contract_created":false,"next":"server will validate and may promote intake to goal"}`))
		case "/v1/intakes/018f0000-0000-7000-8000-000000000005/goals":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"018f0000-0000-7000-8000-000000000006","intake_id":"018f0000-0000-7000-8000-000000000005","organization_id":"018f0000-0000-7000-8000-000000000002","project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":"018f0000-0000-7000-8000-000000000004","title":"Nested work","summary":"Nested work","state":"created"}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
}

func writeProjectConfigFixture(t *testing.T, repoDir string, serverURL string) {
	t.Helper()

	configPath := filepath.Join(repoDir, projectconfig.RelativePath)
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("create .goalrail dir: %v", err)
	}
	content := projectconfig.RenderYAML(projectconfig.Config{
		Version:        projectconfig.Version,
		ServerURL:      serverURL,
		OrganizationID: "018f0000-0000-7000-8000-000000000002",
		ProjectID:      "018f0000-0000-7000-8000-000000000003",
		RepoBindingID:  "018f0000-0000-7000-8000-000000000004",
		Repository: projectconfig.Repository{
			Provider:           "github",
			FullName:           "heurema/goalrail",
			URL:                "git@github.com:heurema/goalrail.git",
			WorkflowBaseBranch: "main",
		},
	})
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write project config: %v", err)
	}
}

func readProjectConfigFile(t *testing.T, repoDir string) string {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(repoDir, projectconfig.RelativePath))
	if err != nil {
		t.Fatalf("read project config: %v", err)
	}
	return string(raw)
}

func setupGitRepo(t *testing.T) string {
	t.Helper()

	repoDir := t.TempDir()
	runGit(t, repoDir, "init")
	runGit(t, repoDir, "-c", "user.name=Goalrail Test", "-c", "user.email=goalrail@example.test", "commit", "--allow-empty", "-m", "initial")
	return repoDir
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
