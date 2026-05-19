package workcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	if output.SchemaVersion != "goalrail.cli.v1" {
		t.Fatalf("schema_version = %q, want goalrail.cli.v1", output.SchemaVersion)
	}
	if output.Display.Summary == "" {
		t.Fatal("display.summary is empty")
	}
	if output.NextAction.Kind != "continue_goal" || output.NextAction.Blocking || !output.NextAction.Available {
		t.Fatalf("next_action = %#v, want available continue_goal", output.NextAction)
	}
	wantCommand := "goalrail work continue --goal-id 018f0000-0000-7000-8000-000000000006 --format json"
	if output.NextAction.Command != wantCommand {
		t.Fatalf("next_action.command = %q, want %q", output.NextAction.Command, wantCommand)
	}
	if output.NextAction.PlannedSlice != "" {
		t.Fatalf("next_action.planned_slice = %q, want empty", output.NextAction.PlannedSlice)
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

func TestRunStartMissingProjectConfigDoesNotReadBodyFileStdin(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	_, err := runStartJSONWithOptions(t, repoDir, fakeSessionStore{session: validSession("http://localhost:8080")}, Options{Stdin: failOnRead{t: t}}, "--title", "Missing marker", "--body-file", "-", "--format", "json")
	if err == nil {
		t.Fatal("Run(work start) error = nil, want missing marker")
	}
	if got := exitcode.ForError(err); got != exitcode.Usage {
		t.Fatalf("exit code = %d, want usage", got)
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

func TestRunStartMissingAuthDoesNotReadBodyFileStdin(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	writeProjectConfigFixture(t, repoDir, "http://localhost:8080")
	_, err := runStartJSONWithOptions(t, repoDir, fakeSessionStore{err: authstore.ErrSessionNotFound}, Options{Stdin: failOnRead{t: t}}, "--title", "Missing login", "--body-file", "-", "--format", "json")
	if err == nil {
		t.Fatal("Run(work start) error = nil, want missing auth")
	}
	if got := exitcode.ForError(err); got != exitcode.Usage {
		t.Fatalf("exit code = %d, want usage", got)
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

func TestRunStartOrganizationMismatchFailsBeforeIntake(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
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
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"user":{"id":"018f0000-0000-7000-8000-000000000001","display_name":"Developer"},"organization_membership":{"organization_id":"018f0000-0000-7000-8000-000000000999","role":"member"}}`))
		case "/v1/intakes":
			intakeCount.Add(1)
			http.Error(w, "unexpected intake request", http.StatusInternalServerError)
		case "/v1/intakes/018f0000-0000-7000-8000-000000000005/goals":
			promoteCount.Add(1)
			http.Error(w, "unexpected promotion request", http.StatusInternalServerError)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, server.URL)

	_, err := runStartJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--title", "Wrong organization", "--format", "json")
	if err == nil {
		t.Fatal("Run(work start) error = nil, want organization mismatch")
	}
	if got := exitcode.ForError(err); got != exitcode.Validation {
		t.Fatalf("exit code = %d, want validation", got)
	}
	if !strings.Contains(err.Error(), "different GoalRail organization") {
		t.Fatalf("error = %q, want organization mismatch", err.Error())
	}
	if meCount.Load() != 1 || intakeCount.Load() != 0 || promoteCount.Load() != 0 {
		t.Fatalf("request counts me/intake/promote = %d/%d/%d, want 1/0/0", meCount.Load(), intakeCount.Load(), promoteCount.Load())
	}
}

func TestRunContinueReadyGoalReturnsDraftContractNextAction(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	var meCount, continueCount atomic.Int32
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
			_, _ = w.Write([]byte(`{"user":{"id":"018f0000-0000-7000-8000-000000000001","display_name":"Developer"},"organization_membership":{"organization_id":"018f0000-0000-7000-8000-000000000002","role":"member"}}`))
		case "/v1/goals/018f0000-0000-7000-8000-000000000006/continuation":
			continueCount.Add(1)
			if r.Method != http.MethodPost {
				t.Errorf("POST /v1/goals/{id}/continuation method = %s", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"goal_id":"018f0000-0000-7000-8000-000000000006","state":"ready_for_contract_seed","readiness":{"goal_id":"018f0000-0000-7000-8000-000000000006","state":"ready_for_contract_seed","ready":true,"reason_codes":[],"message":"goal is ready for contract seed"},"goal":{"id":"018f0000-0000-7000-8000-000000000006","organization_id":"018f0000-0000-7000-8000-000000000002","project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":"018f0000-0000-7000-8000-000000000004","state":"ready_for_contract_seed"}}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, server.URL)

	output, err := runContinueJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--goal-id", "018f0000-0000-7000-8000-000000000006", "--format", "json")
	if err != nil {
		t.Fatalf("Run(work continue) error = %v", err)
	}

	if output.SchemaVersion != "goalrail.cli.v1" {
		t.Fatalf("schema_version = %q, want goalrail.cli.v1", output.SchemaVersion)
	}
	if output.GoalID != "018f0000-0000-7000-8000-000000000006" || output.State != "ready_for_contract_seed" {
		t.Fatalf("goal/state = %q/%q, want ready goal", output.GoalID, output.State)
	}
	if output.Display.Summary == "" {
		t.Fatal("display.summary is empty")
	}
	if output.NextAction.Kind != "draft_contract" || !output.NextAction.Available || output.NextAction.PlannedSlice != "" {
		t.Fatalf("next_action = %#v, want available draft_contract", output.NextAction)
	}
	wantCommand := "goalrail contract draft --goal-id 018f0000-0000-7000-8000-000000000006 --format json"
	if output.NextAction.Command != wantCommand {
		t.Fatalf("next_action.command = %q, want %q", output.NextAction.Command, wantCommand)
	}
	if meCount.Load() != 1 || continueCount.Load() != 1 {
		t.Fatalf("request counts me/continue = %d/%d, want 1/1", meCount.Load(), continueCount.Load())
	}
}

func TestRunContinueIncompleteGoalReturnsAskUserNextAction(t *testing.T) {
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
			_, _ = w.Write([]byte(`{"user":{"id":"018f0000-0000-7000-8000-000000000001","display_name":"Developer"},"organization_membership":{"organization_id":"018f0000-0000-7000-8000-000000000002","role":"member"}}`))
		case "/v1/goals/018f0000-0000-7000-8000-000000000006/continuation":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"goal_id":"018f0000-0000-7000-8000-000000000006","state":"needs_clarification","readiness":{"goal_id":"018f0000-0000-7000-8000-000000000006","state":"needs_clarification","ready":false,"reason_codes":["missing_scope_hint"],"message":"goal needs clarification before contract seed"},"goal":{"id":"018f0000-0000-7000-8000-000000000006","organization_id":"018f0000-0000-7000-8000-000000000002","project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":"018f0000-0000-7000-8000-000000000004","state":"needs_clarification"},"clarification_request":{"id":"018f0000-0000-7000-8000-000000000101","goal_id":"018f0000-0000-7000-8000-000000000006","reason_codes":["missing_scope_hint"],"state":"open","questions":[{"id":"018f0000-0000-7000-8000-000000000102","text":"What is the intended scope at a high level?","why_needed":"A scope hint is required before contract seed readiness.","answer_type":"text","maps_to":"goal.scope_hint"}]}}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, server.URL)

	output, err := runContinueJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--goal-id", "018f0000-0000-7000-8000-000000000006", "--format", "json")
	if err != nil {
		t.Fatalf("Run(work continue) error = %v", err)
	}

	if output.State != "needs_clarification" {
		t.Fatalf("state = %q, want needs_clarification", output.State)
	}
	if output.NextAction.Kind != "ask_user" || !output.NextAction.Available || !output.NextAction.Blocking {
		t.Fatalf("next_action = %#v, want available blocking ask_user", output.NextAction)
	}
	if output.NextAction.RequestID != "018f0000-0000-7000-8000-000000000101" {
		t.Fatalf("request_id = %q, want server request id", output.NextAction.RequestID)
	}
	if len(output.NextAction.Questions) != 1 || output.NextAction.Questions[0].MapsTo != "goal.scope_hint" {
		t.Fatalf("questions = %#v, want scope question", output.NextAction.Questions)
	}
}

func TestRunContinueMissingProjectConfigFailsBeforeHTTP(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestCount.Add(1)
		http.Error(w, "unexpected request", http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := runContinueJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--goal-id", "goal-1", "--format", "json")
	if err == nil {
		t.Fatal("Run(work continue) error = nil, want missing marker")
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

func TestRunContinueExpiredTokenFailsBeforeHTTP(t *testing.T) {
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
	_, err := runContinueJSON(t, repoDir, fakeSessionStore{session: session}, "--goal-id", "goal-1", "--format", "json")
	if err == nil {
		t.Fatal("Run(work continue) error = nil, want expired login")
	}
	if got := exitcode.ForError(err); got != exitcode.Usage {
		t.Fatalf("exit code = %d, want usage", got)
	}
	if got := requestCount.Load(); got != 0 {
		t.Fatalf("server requests = %d, want 0 for expired token", got)
	}
}

func TestRunContinueOrganizationMismatchFailsBeforeContinuation(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	var meCount, continueCount atomic.Int32
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
			_, _ = w.Write([]byte(`{"user":{"id":"018f0000-0000-7000-8000-000000000001","display_name":"Developer"},"organization_membership":{"organization_id":"018f0000-0000-7000-8000-000000000999","role":"member"}}`))
		case "/v1/goals/goal-1/continuation":
			continueCount.Add(1)
			http.Error(w, "unexpected continuation request", http.StatusInternalServerError)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, server.URL)

	_, err := runContinueJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--goal-id", "goal-1", "--format", "json")
	if err == nil {
		t.Fatal("Run(work continue) error = nil, want organization mismatch")
	}
	if got := exitcode.ForError(err); got != exitcode.Validation {
		t.Fatalf("exit code = %d, want validation", got)
	}
	if meCount.Load() != 1 || continueCount.Load() != 0 {
		t.Fatalf("request counts me/continue = %d/%d, want 1/0", meCount.Load(), continueCount.Load())
	}
}

func TestRunContinueTextDoesNotClaimUnavailableRuntimeWork(t *testing.T) {
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
			_, _ = w.Write([]byte(`{"user":{"id":"018f0000-0000-7000-8000-000000000001","display_name":"Developer"},"organization_membership":{"organization_id":"018f0000-0000-7000-8000-000000000002","role":"member"}}`))
		case "/v1/goals/goal-1/continuation":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"goal_id":"goal-1","state":"needs_clarification","clarification_request":{"id":"request-1","goal_id":"goal-1","reason_codes":["missing_scope_hint"],"state":"open","questions":[{"id":"question-1","text":"What is the intended scope at a high level?","why_needed":"A scope hint is required before contract seed readiness.","answer_type":"text","maps_to":"goal.scope_hint"}]}}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, server.URL)

	var stdout, stderr bytes.Buffer
	err := RunWithOptions(context.Background(), term.New(&stdout, &stderr), repoDir, []string{"continue", "--goal-id", "goal-1"}, Options{
		Store: fakeSessionStore{session: validSession(server.URL)},
		Now:   func() time.Time { return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("Run(work continue text) error = %v", err)
	}
	got := stdout.String()
	for _, forbidden := range []string{"created a contract", "runner", "proof"} {
		if strings.Contains(strings.ToLower(got), forbidden) {
			t.Fatalf("stdout = %q, want no %q claim", got, forbidden)
		}
	}
	if !strings.Contains(got, "Clarification request: request-1") {
		t.Fatalf("stdout = %q, want clarification request", got)
	}
}

func TestRunAnswerFileSubmitsStructuredAnswers(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	answersPath := filepath.Join(repoDir, "answers.json")
	if err := os.WriteFile(answersPath, []byte(`{"answers":[{"question_id":"q_scope","value":"Bounded answer bridge"}]}`), 0o644); err != nil {
		t.Fatalf("write answers file: %v", err)
	}

	var answerRequest workAnswerSubmission
	var meCount, answerCount atomic.Int32
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
			_, _ = w.Write([]byte(`{"user":{"id":"018f0000-0000-7000-8000-000000000001","display_name":"Developer"},"organization_membership":{"organization_id":"018f0000-0000-7000-8000-000000000002","role":"member"}}`))
		case "/v1/clarifications/018f0000-0000-7000-8000-000000000101/answers/continuation":
			answerCount.Add(1)
			if r.Method != http.MethodPost {
				t.Errorf("POST /v1/clarifications/{id}/answers/continuation method = %s", r.Method)
			}
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&answerRequest); err != nil {
				t.Errorf("decode answer request: %v", err)
				http.Error(w, "bad json", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"goal_id":"018f0000-0000-7000-8000-000000000006","state":"ready_for_contract_seed","readiness":{"goal_id":"018f0000-0000-7000-8000-000000000006","state":"ready_for_contract_seed","ready":true,"reason_codes":[],"message":"goal is ready for contract seed"},"goal":{"id":"018f0000-0000-7000-8000-000000000006","organization_id":"018f0000-0000-7000-8000-000000000002","project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":"018f0000-0000-7000-8000-000000000004","state":"ready_for_contract_seed"}}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, server.URL)

	output, err := runAnswerJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--clarification-request-id", "018f0000-0000-7000-8000-000000000101", "--answers-file", answersPath, "--format", "json")
	if err != nil {
		t.Fatalf("Run(work answer) error = %v", err)
	}

	if output.SchemaVersion != "goalrail.cli.v1" {
		t.Fatalf("schema_version = %q, want goalrail.cli.v1", output.SchemaVersion)
	}
	if output.ClarificationRequestID != "018f0000-0000-7000-8000-000000000101" {
		t.Fatalf("clarification_request_id = %q, want request id", output.ClarificationRequestID)
	}
	if output.GoalID != "018f0000-0000-7000-8000-000000000006" || output.State != "ready_for_contract_seed" {
		t.Fatalf("goal/state = %q/%q, want ready goal", output.GoalID, output.State)
	}
	if output.Display.Summary == "" {
		t.Fatal("display.summary is empty")
	}
	if output.NextAction.Kind != "draft_contract" || !output.NextAction.Available || output.NextAction.PlannedSlice != "" {
		t.Fatalf("next_action = %#v, want available draft_contract", output.NextAction)
	}
	if len(answerRequest.Answers) != 1 || answerRequest.Answers[0].QuestionID != "q_scope" || answerRequest.Answers[0].Value != "Bounded answer bridge" {
		t.Fatalf("answer request = %#v, want structured answer payload", answerRequest)
	}
	if meCount.Load() != 1 || answerCount.Load() != 1 {
		t.Fatalf("request counts me/answer = %d/%d, want 1/1", meCount.Load(), answerCount.Load())
	}
}

func TestRunAnswerDashReadsStdin(t *testing.T) {
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
			_, _ = w.Write([]byte(`{"user":{"id":"018f0000-0000-7000-8000-000000000001","display_name":"Developer"},"organization_membership":{"organization_id":"018f0000-0000-7000-8000-000000000002","role":"member"}}`))
		case "/v1/clarifications/018f0000-0000-7000-8000-000000000101/answers/continuation":
			var request workAnswerSubmission
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&request); err != nil {
				t.Errorf("decode answer request: %v", err)
				http.Error(w, "bad json", http.StatusBadRequest)
				return
			}
			if len(request.Answers) != 1 || request.Answers[0].Value != "Need one more constraint" {
				t.Errorf("answer request = %#v, want stdin answer", request)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"goal_id":"018f0000-0000-7000-8000-000000000006","state":"needs_clarification","readiness":{"goal_id":"018f0000-0000-7000-8000-000000000006","state":"needs_clarification","ready":false,"reason_codes":["missing_acceptance_hint"],"message":"goal needs clarification before contract seed"},"goal":{"id":"018f0000-0000-7000-8000-000000000006","organization_id":"018f0000-0000-7000-8000-000000000002","project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":"018f0000-0000-7000-8000-000000000004","state":"needs_clarification"},"clarification_request":{"id":"018f0000-0000-7000-8000-000000000201","goal_id":"018f0000-0000-7000-8000-000000000006","reason_codes":["missing_acceptance_hint"],"state":"open","questions":[{"id":"018f0000-0000-7000-8000-000000000202","text":"What outcome would make this goal acceptable?","why_needed":"An acceptance hint is required before contract seed readiness.","answer_type":"text","maps_to":"goal.acceptance_hint"}]}}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, server.URL)

	output, err := runAnswerJSONWithOptions(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, Options{Stdin: strings.NewReader(`{"answers":[{"question_id":"q_scope","value":"Need one more constraint"}]}`)}, "--clarification-request-id", "018f0000-0000-7000-8000-000000000101", "--answers-file", "-", "--format", "json")
	if err != nil {
		t.Fatalf("Run(work answer --answers-file -) error = %v", err)
	}
	if output.NextAction.Kind != "ask_user" || !output.NextAction.Available || !output.NextAction.Blocking {
		t.Fatalf("next_action = %#v, want available ask_user", output.NextAction)
	}
	if output.NextAction.RequestID != "018f0000-0000-7000-8000-000000000201" {
		t.Fatalf("request_id = %q, want next clarification request", output.NextAction.RequestID)
	}
}

func TestRunAnswerMissingProjectConfigFailsBeforeHTTPAndStdin(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestCount.Add(1)
		http.Error(w, "unexpected request", http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := runAnswerJSONWithOptions(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, Options{Stdin: failOnRead{t: t}}, "--clarification-request-id", "018f0000-0000-7000-8000-000000000101", "--answers-file", "-", "--format", "json")
	if err == nil {
		t.Fatal("Run(work answer) error = nil, want missing marker")
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

func TestRunAnswerExpiredTokenFailsBeforeHTTPAndStdin(t *testing.T) {
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
	_, err := runAnswerJSONWithOptions(t, repoDir, fakeSessionStore{session: session}, Options{Stdin: failOnRead{t: t}}, "--clarification-request-id", "018f0000-0000-7000-8000-000000000101", "--answers-file", "-", "--format", "json")
	if err == nil {
		t.Fatal("Run(work answer) error = nil, want expired login")
	}
	if got := exitcode.ForError(err); got != exitcode.Usage {
		t.Fatalf("exit code = %d, want usage", got)
	}
	if got := requestCount.Load(); got != 0 {
		t.Fatalf("server requests = %d, want 0 for expired token", got)
	}
}

func TestRunAnswerOrganizationMismatchFailsBeforeAnswerRequestAndStdin(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	var meCount, answerCount atomic.Int32
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
			_, _ = w.Write([]byte(`{"user":{"id":"018f0000-0000-7000-8000-000000000001","display_name":"Developer"},"organization_membership":{"organization_id":"018f0000-0000-7000-8000-000000000999","role":"member"}}`))
		case "/v1/clarifications/018f0000-0000-7000-8000-000000000101/answers/continuation":
			answerCount.Add(1)
			http.Error(w, "unexpected answer request", http.StatusInternalServerError)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, server.URL)

	_, err := runAnswerJSONWithOptions(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, Options{Stdin: failOnRead{t: t}}, "--clarification-request-id", "018f0000-0000-7000-8000-000000000101", "--answers-file", "-", "--format", "json")
	if err == nil {
		t.Fatal("Run(work answer) error = nil, want organization mismatch")
	}
	if got := exitcode.ForError(err); got != exitcode.Validation {
		t.Fatalf("exit code = %d, want validation", got)
	}
	if meCount.Load() != 1 || answerCount.Load() != 0 {
		t.Fatalf("request counts me/answer = %d/%d, want 1/0", meCount.Load(), answerCount.Load())
	}
}

func TestRunAnswerTextDoesNotClaimUnavailableRuntimeWork(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	answersPath := filepath.Join(repoDir, "answers.json")
	if err := os.WriteFile(answersPath, []byte(`{"answers":[{"question_id":"q_scope","value":"Bounded answer bridge"}]}`), 0o644); err != nil {
		t.Fatalf("write answers file: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer access-token" {
			t.Errorf("Authorization = %q, want bearer token", r.Header.Get("Authorization"))
			http.Error(w, "bad auth", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/v1/me":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"user":{"id":"018f0000-0000-7000-8000-000000000001","display_name":"Developer"},"organization_membership":{"organization_id":"018f0000-0000-7000-8000-000000000002","role":"member"}}`))
		case "/v1/clarifications/018f0000-0000-7000-8000-000000000101/answers/continuation":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"goal_id":"018f0000-0000-7000-8000-000000000006","state":"ready_for_contract_seed","goal":{"id":"018f0000-0000-7000-8000-000000000006","organization_id":"018f0000-0000-7000-8000-000000000002","project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":"018f0000-0000-7000-8000-000000000004","state":"ready_for_contract_seed"}}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, server.URL)

	var stdout, stderr bytes.Buffer
	err := RunWithOptions(context.Background(), term.New(&stdout, &stderr), repoDir, []string{"answer", "--clarification-request-id", "018f0000-0000-7000-8000-000000000101", "--answers-file", answersPath}, Options{
		Store: fakeSessionStore{session: validSession(server.URL)},
		Now:   func() time.Time { return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("Run(work answer text) error = %v", err)
	}
	got := stdout.String()
	for _, forbidden := range []string{"created a contract", "runner", "proof"} {
		if strings.Contains(strings.ToLower(got), forbidden) {
			t.Fatalf("stdout = %q, want no %q claim", got, forbidden)
		}
	}
	if !strings.Contains(got, "Next: goalrail contract draft") {
		t.Fatalf("stdout = %q, want available contract draft next command", got)
	}
}

func TestRunPlanCreatesQueuedWorkItemPlan(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	var meCount, planCount atomic.Int32
	var request workPlanCreateRequest
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
		case "/v1/contracts/018f0000-0000-7000-8000-000000000009/plans":
			planCount.Add(1)
			if r.Method != http.MethodPost {
				t.Errorf("POST /v1/contracts/{id}/plans method = %s", r.Method)
			}
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&request); err != nil {
				t.Errorf("decode work plan request: %v", err)
				http.Error(w, "bad json", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"018f0000-0000-7000-8000-000000000301","contract_id":"018f0000-0000-7000-8000-000000000009","approved_contract_id":"018f0000-0000-7000-8000-000000000010","repo_binding_id":"018f0000-0000-7000-8000-000000000004","state":"queued","requested_by":{"kind":"user","id":"018f0000-0000-7000-8000-000000000001"}}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, server.URL)

	output, err := runPlanJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--contract-id", "018f0000-0000-7000-8000-000000000009", "--format", "json")
	if err != nil {
		t.Fatalf("Run(work plan) error = %v", err)
	}

	if output.SchemaVersion != "goalrail.cli.v1" {
		t.Fatalf("schema_version = %q, want goalrail.cli.v1", output.SchemaVersion)
	}
	if output.ContractID != "018f0000-0000-7000-8000-000000000009" || output.PlanID != "018f0000-0000-7000-8000-000000000301" || output.PlanState != "queued" {
		t.Fatalf("contract/plan/state = %q/%q/%q, want queued plan", output.ContractID, output.PlanID, output.PlanState)
	}
	if output.Display.Summary == "" {
		t.Fatal("display.summary is empty")
	}
	if output.NextAction.Kind != "planning_worker_required" || !output.NextAction.Blocking || output.NextAction.Available {
		t.Fatalf("next_action = %#v, want unavailable blocking planning_worker_required", output.NextAction)
	}
	if request.ProjectID != "018f0000-0000-7000-8000-000000000003" || request.RepoBindingID != "018f0000-0000-7000-8000-000000000004" {
		t.Fatalf("work plan request project/repo = %q/%q, want marker context", request.ProjectID, request.RepoBindingID)
	}
	if meCount.Load() != 1 || planCount.Load() != 1 {
		t.Fatalf("request counts me/plan = %d/%d, want 1/1", meCount.Load(), planCount.Load())
	}
}

func TestRunPlanMapsExistingPlanStatesHonestly(t *testing.T) {
	t.Parallel()
	requireGit(t)

	cases := []struct {
		name         string
		planState    string
		nextKind     string
		blocking     bool
		available    bool
		plannedSlice string
	}{
		{
			name:      "queued",
			planState: "queued",
			nextKind:  "planning_worker_required",
			blocking:  true,
		},
		{
			name:      "leased",
			planState: "leased",
			nextKind:  "planning_in_progress",
			blocking:  true,
		},
		{
			name:      "proposal submitted",
			planState: "proposal_submitted",
			nextKind:  "review_plan_proposal",
			blocking:  true,
			available: true,
		},
		{
			name:         "accepted",
			planState:    "accepted",
			nextKind:     "planned_workitems_ready",
			blocking:     false,
			plannedSlice: "H",
		},
		{
			name:      "unknown",
			planState: "unexpected_state",
			nextKind:  "blocked",
			blocking:  true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

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
				case "/v1/contracts/018f0000-0000-7000-8000-000000000009/plans":
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_, _ = fmt.Fprintf(w, `{"id":"018f0000-0000-7000-8000-000000000301","contract_id":"018f0000-0000-7000-8000-000000000009","approved_contract_id":"018f0000-0000-7000-8000-000000000010","repo_binding_id":"018f0000-0000-7000-8000-000000000004","state":%q}`, tc.planState)
				default:
					t.Errorf("unexpected path %s", r.URL.Path)
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			writeProjectConfigFixture(t, repoDir, server.URL)

			output, err := runPlanJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--contract-id", "018f0000-0000-7000-8000-000000000009", "--format", "json")
			if err != nil {
				t.Fatalf("Run(work plan) error = %v", err)
			}
			if output.PlanState != tc.planState {
				t.Fatalf("plan_state = %q, want %q", output.PlanState, tc.planState)
			}
			if output.NextAction.Kind != tc.nextKind || output.NextAction.Blocking != tc.blocking || output.NextAction.Available != tc.available {
				t.Fatalf("next_action = %#v, want kind=%q blocking=%v available=%v", output.NextAction, tc.nextKind, tc.blocking, tc.available)
			}
			if output.NextAction.PlannedSlice != tc.plannedSlice {
				t.Fatalf("planned_slice = %q, want %q", output.NextAction.PlannedSlice, tc.plannedSlice)
			}
			if tc.planState == "proposal_submitted" && !strings.Contains(output.NextAction.Command, "goalrail work plan status --plan-id") {
				t.Fatalf("next command = %q, want plan status command", output.NextAction.Command)
			}
			if tc.planState != "queued" && strings.Contains(strings.ToLower(output.Display.Summary), "queued") {
				t.Fatalf("display.summary = %q, should not claim queued for state %q", output.Display.Summary, tc.planState)
			}
		})
	}
}

func TestRunPlanPreflightFailuresHappenBeforeHTTP(t *testing.T) {
	t.Parallel()
	requireGit(t)

	t.Run("missing marker", func(t *testing.T) {
		repoDir := setupGitRepo(t)
		var requestCount atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			requestCount.Add(1)
			http.Error(w, "unexpected request", http.StatusInternalServerError)
		}))
		defer server.Close()

		_, err := runPlanJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--contract-id", "018f0000-0000-7000-8000-000000000009", "--format", "json")
		if err == nil {
			t.Fatal("Run(work plan) error = nil, want missing marker")
		}
		if got := exitcode.ForError(err); got != exitcode.Usage {
			t.Fatalf("exit code = %d, want usage", got)
		}
		if got := requestCount.Load(); got != 0 {
			t.Fatalf("server requests = %d, want 0 without marker", got)
		}
	})

	t.Run("damaged marker", func(t *testing.T) {
		repoDir := setupGitRepo(t)
		var requestCount atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			requestCount.Add(1)
			http.Error(w, "unexpected request", http.StatusInternalServerError)
		}))
		defer server.Close()
		writeProjectConfigFixtureWithIDs(t, repoDir, server.URL, "", "018f0000-0000-7000-8000-000000000004")

		_, err := runPlanJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--contract-id", "018f0000-0000-7000-8000-000000000009", "--format", "json")
		if err == nil {
			t.Fatal("Run(work plan) error = nil, want damaged marker")
		}
		if got := exitcode.ForError(err); got != exitcode.Validation {
			t.Fatalf("exit code = %d, want validation", got)
		}
		if !strings.Contains(err.Error(), "missing project_id") {
			t.Fatalf("error = %q, want missing project_id", err.Error())
		}
		if got := requestCount.Load(); got != 0 {
			t.Fatalf("server requests = %d, want 0 with damaged marker", got)
		}
	})

	t.Run("expired login", func(t *testing.T) {
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

		_, err := runPlanJSON(t, repoDir, fakeSessionStore{session: session}, "--contract-id", "018f0000-0000-7000-8000-000000000009", "--format", "json")
		if err == nil {
			t.Fatal("Run(work plan) error = nil, want expired login")
		}
		if got := exitcode.ForError(err); got != exitcode.Usage {
			t.Fatalf("exit code = %d, want usage", got)
		}
		if got := requestCount.Load(); got != 0 {
			t.Fatalf("server requests = %d, want 0 with expired login", got)
		}
	})

	t.Run("malformed contract id", func(t *testing.T) {
		repoDir := setupGitRepo(t)
		var requestCount atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			requestCount.Add(1)
			http.Error(w, "unexpected request", http.StatusInternalServerError)
		}))
		defer server.Close()
		writeProjectConfigFixture(t, repoDir, server.URL)

		_, err := runPlanJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--contract-id", "not-a-uuid", "--format", "json")
		if err == nil {
			t.Fatal("Run(work plan) error = nil, want malformed contract id")
		}
		if got := exitcode.ForError(err); got != exitcode.Validation {
			t.Fatalf("exit code = %d, want validation", got)
		}
		if got := requestCount.Load(); got != 0 {
			t.Fatalf("server requests = %d, want 0 with malformed contract id", got)
		}
	})
}

func TestRunPlanOrganizationMismatchFailsBeforePlanRequest(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	var meCount, planCount atomic.Int32
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
			_, _ = w.Write([]byte(`{"user":{"id":"018f0000-0000-7000-8000-000000000001","display_name":"Developer"},"organization_membership":{"organization_id":"018f0000-0000-7000-8000-000000000999","role":"member","state":"active"}}`))
		case "/v1/contracts/018f0000-0000-7000-8000-000000000009/plans":
			planCount.Add(1)
			http.Error(w, "unexpected plan request", http.StatusInternalServerError)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, server.URL)

	_, err := runPlanJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--contract-id", "018f0000-0000-7000-8000-000000000009", "--format", "json")
	if err == nil {
		t.Fatal("Run(work plan) error = nil, want organization mismatch")
	}
	if got := exitcode.ForError(err); got != exitcode.Validation {
		t.Fatalf("exit code = %d, want validation", got)
	}
	if meCount.Load() != 1 || planCount.Load() != 0 {
		t.Fatalf("request counts me/plan = %d/%d, want 1/0", meCount.Load(), planCount.Load())
	}
}

func TestRunPlanTextDoesNotClaimWorkerProposalOrProof(t *testing.T) {
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
		case "/v1/contracts/018f0000-0000-7000-8000-000000000009/plans":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"018f0000-0000-7000-8000-000000000301","contract_id":"018f0000-0000-7000-8000-000000000009","approved_contract_id":"018f0000-0000-7000-8000-000000000010","repo_binding_id":"018f0000-0000-7000-8000-000000000004","state":"queued"}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, server.URL)

	var stdout, stderr bytes.Buffer
	err := RunWithOptions(context.Background(), term.New(&stdout, &stderr), repoDir, []string{"plan", "--contract-id", "018f0000-0000-7000-8000-000000000009"}, Options{
		Store: fakeSessionStore{session: validSession(server.URL)},
		Now:   func() time.Time { return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("Run(work plan text) error = %v", err)
	}
	got := stdout.String()
	for _, forbidden := range []string{"proposal submitted", "created workitems", "workitems created", "run", "proof", "verified"} {
		if strings.Contains(strings.ToLower(got), forbidden) {
			t.Fatalf("stdout = %q, want no %q claim", got, forbidden)
		}
	}
	if !strings.Contains(got, "planning worker required") {
		t.Fatalf("stdout = %q, want worker-required message", got)
	}
}

func TestRunPlanStatusReturnsSubmittedProposalAndAcceptNextAction(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	var statusRequest workPlanCreateRequest
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
		case "/v1/plans/018f0000-0000-7000-8000-000000000301/status":
			if r.Method != http.MethodPost {
				t.Errorf("POST /v1/plans/{id}/status method = %s", r.Method)
			}
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&statusRequest); err != nil {
				t.Errorf("decode plan status request: %v", err)
				http.Error(w, "bad json", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"plan":{"id":"018f0000-0000-7000-8000-000000000301","contract_id":"018f0000-0000-7000-8000-000000000009","approved_contract_id":"018f0000-0000-7000-8000-000000000010","repo_binding_id":"018f0000-0000-7000-8000-000000000004","state":"proposal_submitted"},"proposal":{"id":"018f0000-0000-7000-8000-000000000302","plan_id":"018f0000-0000-7000-8000-000000000301","contract_id":"018f0000-0000-7000-8000-000000000009","approved_contract_id":"018f0000-0000-7000-8000-000000000010","repo_binding_id":"018f0000-0000-7000-8000-000000000004","state":"submitted","proposed_tasks":[{"title":"Refactor CSV export filters","summary":"Extract duplicated filter construction.","scope":["Update export filter construction"],"acceptance_refs":["acceptance_criteria[0]"],"proof_expectation_refs":["proof_expectations[0]"],"order_index":0}]}}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, server.URL)

	output, err := runPlanStatusJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--plan-id", "018f0000-0000-7000-8000-000000000301", "--format", "json")
	if err != nil {
		t.Fatalf("Run(work plan status) error = %v", err)
	}
	if output.PlanState != "proposal_submitted" || output.ProposalID != "018f0000-0000-7000-8000-000000000302" || len(output.ProposedTasks) != 1 {
		t.Fatalf("plan/proposal/tasks = %q/%q/%d, want submitted proposal", output.PlanState, output.ProposalID, len(output.ProposedTasks))
	}
	if output.NextAction.Kind != "accept_proposal" || !output.NextAction.Available || !strings.Contains(output.NextAction.Command, "--confirm-user-acceptance") {
		t.Fatalf("next_action = %#v, want available explicit proposal accept", output.NextAction)
	}
	if statusRequest.ProjectID != "018f0000-0000-7000-8000-000000000003" || statusRequest.RepoBindingID != "018f0000-0000-7000-8000-000000000004" {
		t.Fatalf("status request project/repo = %q/%q, want marker context", statusRequest.ProjectID, statusRequest.RepoBindingID)
	}
}

func TestRunPlanStatusDoesNotAcceptWithoutProposalID(t *testing.T) {
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
		case "/v1/plans/018f0000-0000-7000-8000-000000000301/status":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"plan":{"id":"018f0000-0000-7000-8000-000000000301","contract_id":"018f0000-0000-7000-8000-000000000009","approved_contract_id":"018f0000-0000-7000-8000-000000000010","repo_binding_id":"018f0000-0000-7000-8000-000000000004","state":"proposal_submitted"}}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, server.URL)

	output, err := runPlanStatusJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--plan-id", "018f0000-0000-7000-8000-000000000301", "--format", "json")
	if err != nil {
		t.Fatalf("Run(work plan status) error = %v", err)
	}
	if output.ProposalID != "" {
		t.Fatalf("proposal_id = %q, want empty when status response has no proposal", output.ProposalID)
	}
	if output.NextAction.Kind == "accept_proposal" || strings.Contains(output.NextAction.Command, "proposal accept") {
		t.Fatalf("next_action = %#v, want no proposal acceptance command without proposal_id", output.NextAction)
	}
	if output.NextAction.Kind != "review_plan_proposal" || !output.NextAction.Available || !strings.Contains(output.NextAction.Command, "work plan status") {
		t.Fatalf("next_action = %#v, want plan status retry/review action", output.NextAction)
	}
}

func TestRunProposalAcceptRequiresConfirmBeforeHTTP(t *testing.T) {
	t.Parallel()

	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestCount.Add(1)
		http.Error(w, "unexpected request", http.StatusInternalServerError)
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	err := RunWithOptions(context.Background(), term.New(&stdout, &stderr), t.TempDir(), []string{"proposal", "accept", "--proposal-id", "018f0000-0000-7000-8000-000000000302", "--format", "json"}, Options{
		Store: fakeSessionStore{session: validSession(server.URL)},
		Now:   func() time.Time { return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC) },
	})
	if err == nil {
		t.Fatal("Run(work proposal accept) error = nil, want missing confirmation")
	}
	if got := exitcode.ForError(err); got != exitcode.Usage {
		t.Fatalf("exit code = %d, want usage", got)
	}
	if got := requestCount.Load(); got != 0 {
		t.Fatalf("server requests = %d, want 0 without confirmation", got)
	}
}

func TestRunProposalAcceptCreatesPlannedWorkItemsNextUnavailable(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	var acceptRequest workPlanCreateRequest
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
		case "/v1/proposals/018f0000-0000-7000-8000-000000000302/acceptance":
			if r.Method != http.MethodPost {
				t.Errorf("POST /v1/proposals/{id}/acceptance method = %s", r.Method)
			}
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&acceptRequest); err != nil {
				t.Errorf("decode proposal acceptance request: %v", err)
				http.Error(w, "bad json", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"proposal_id":"018f0000-0000-7000-8000-000000000302","plan_id":"018f0000-0000-7000-8000-000000000301","contract_id":"018f0000-0000-7000-8000-000000000009","state":"accepted","created_task_ids":["018f0000-0000-7000-8000-000000000401"]}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, server.URL)

	output, err := runProposalAcceptJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--proposal-id", "018f0000-0000-7000-8000-000000000302", "--confirm-user-acceptance", "--format", "json")
	if err != nil {
		t.Fatalf("Run(work proposal accept) error = %v", err)
	}
	if output.ProposalID != "018f0000-0000-7000-8000-000000000302" || len(output.CreatedTaskIDs) != 1 {
		t.Fatalf("proposal/tasks = %q/%d, want accepted proposal with task", output.ProposalID, len(output.CreatedTaskIDs))
	}
	if output.NextAction.Kind != "prepare_checkout" || !output.NextAction.Available || !strings.Contains(output.NextAction.Command, "goalrail work checkout prepare --task-id 018f0000-0000-7000-8000-000000000401") {
		t.Fatalf("next_action = %#v, want available checkout preparation next", output.NextAction)
	}
	if acceptRequest.ProjectID != "018f0000-0000-7000-8000-000000000003" || acceptRequest.RepoBindingID != "018f0000-0000-7000-8000-000000000004" {
		t.Fatalf("accept request project/repo = %q/%q, want marker context", acceptRequest.ProjectID, acceptRequest.RepoBindingID)
	}
}

func TestRenderProposalAcceptTextShowsAvailableCheckoutCommand(t *testing.T) {
	t.Parallel()

	output := spine.WorkProposalAcceptOutput{
		ServerURL:       "https://goalrail.example",
		ProjectID:       "018f0000-0000-7000-8000-000000000003",
		RepoBindingID:   "018f0000-0000-7000-8000-000000000004",
		ContractID:      "018f0000-0000-7000-8000-000000000009",
		PlanID:          "018f0000-0000-7000-8000-000000000301",
		ProposalID:      "018f0000-0000-7000-8000-000000000302",
		ProposalState:   "accepted",
		CreatedTaskIDs:  []string{"018f0000-0000-7000-8000-000000000401"},
		LocalConfigPath: projectconfig.RelativePath,
		Display: spine.DisplaySummary{
			Summary: "Accepted proposal.",
		},
		NextAction: spine.NextAction{
			Kind:      "prepare_checkout",
			Available: true,
			Command:   "goalrail work checkout prepare --task-id 018f0000-0000-7000-8000-000000000401 --format json",
		},
	}

	got := renderProposalAcceptText(output)
	if !strings.Contains(got, "Next: goalrail work checkout prepare --task-id 018f0000-0000-7000-8000-000000000401 --format json") {
		t.Fatalf("renderProposalAcceptText() = %q, want checkout command", got)
	}
	if strings.Contains(got, "Next action: prepare a checkout job") {
		t.Fatalf("renderProposalAcceptText() = %q, want concrete command instead of generic next action", got)
	}
}

func TestRunWorkItemShowJSONReadsTaskDetail(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	var meCount, detailCount atomic.Int32
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
		case "/v1/tasks/018f0000-0000-7000-8000-000000000401":
			detailCount.Add(1)
			if r.Method != http.MethodGet {
				t.Errorf("GET /v1/tasks/{id} method = %s", r.Method)
			}
			if r.URL.Query().Get("project_id") != "018f0000-0000-7000-8000-000000000003" || r.URL.Query().Get("repo_binding_id") != "018f0000-0000-7000-8000-000000000004" {
				t.Errorf("query project/repo = %q/%q, want marker context", r.URL.Query().Get("project_id"), r.URL.Query().Get("repo_binding_id"))
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"018f0000-0000-7000-8000-000000000401","work_item_id":"018f0000-0000-7000-8000-000000000401","task_id":"018f0000-0000-7000-8000-000000000401","project_id":"018f0000-0000-7000-8000-000000000003","goal_id":"018f0000-0000-7000-8000-000000000006","contract_id":"018f0000-0000-7000-8000-000000000009","approved_contract_id":"018f0000-0000-7000-8000-000000000010","plan_id":"018f0000-0000-7000-8000-000000000301","proposal_id":"018f0000-0000-7000-8000-000000000302","repo_binding_id":"018f0000-0000-7000-8000-000000000004","status":"planned","title":"DOGFOOD-005: WorkItem detail visibility","summary":"Add read-only WorkItem detail visibility.","scope":["Add GET task detail surface","Add goalrail work item show"],"acceptance_refs":["acceptance_criteria[0]"],"proof_expectation_refs":["proof_expectations[0]"],"source_refs":[{"kind":"approved_contract","id":"018f0000-0000-7000-8000-000000000010"}],"owner_hint":"cli","order_index":0,"next_action":{"kind":"prepare_checkout","blocking":false,"available":true,"command":"goalrail work checkout prepare --task-id 018f0000-0000-7000-8000-000000000401 --format json"}}`))
		case "/v1/tasks/018f0000-0000-7000-8000-000000000401/checkout-jobs", "/v1/tasks/018f0000-0000-7000-8000-000000000401/execution-jobs":
			t.Errorf("work item show unexpectedly called mutation endpoint %s", r.URL.Path)
			http.NotFound(w, r)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, server.URL)

	output, err := runWorkItemShowJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--task-id", "018F0000-0000-7000-8000-000000000401", "--format", "json")
	if err != nil {
		t.Fatalf("Run(work item show) error = %v", err)
	}
	if output.SchemaVersion != "goalrail.cli.v1" || output.TaskID != "018f0000-0000-7000-8000-000000000401" || output.WorkItemID != output.TaskID {
		t.Fatalf("output identity = %#v, want work item detail envelope", output)
	}
	if output.GoalID != "018f0000-0000-7000-8000-000000000006" || output.ContractID != "018f0000-0000-7000-8000-000000000009" || output.PlanID == "" || output.ProposalID == "" {
		t.Fatalf("output lineage = %#v, want contract/goal/plan/proposal lineage", output)
	}
	if output.Title != "DOGFOOD-005: WorkItem detail visibility" || len(output.Scope) != 2 || len(output.AcceptanceRefs) != 1 || len(output.ProofExpectationRefs) != 1 {
		t.Fatalf("output body = %#v, want WorkItem delivery fields", output)
	}
	if output.NextAction.Kind != "prepare_checkout" || !output.NextAction.Available || !strings.Contains(output.NextAction.Command, "goalrail work checkout prepare --task-id 018f0000-0000-7000-8000-000000000401") {
		t.Fatalf("next_action = %#v, want checkout preparation guidance", output.NextAction)
	}
	if meCount.Load() != 1 || detailCount.Load() != 1 {
		t.Fatalf("request counts me/detail = %d/%d, want 1/1", meCount.Load(), detailCount.Load())
	}
}

func TestRunWorkItemShowTextIncludesReadOnlyNote(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/me":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"user":{"id":"018f0000-0000-7000-8000-000000000001","display_name":"Developer"},"organization_membership":{"organization_id":"018f0000-0000-7000-8000-000000000002","role":"member","state":"active"}}`))
		case "/v1/tasks/018f0000-0000-7000-8000-000000000401":
			if r.Method != http.MethodGet {
				t.Errorf("GET /v1/tasks/{id} method = %s", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"018f0000-0000-7000-8000-000000000401","work_item_id":"018f0000-0000-7000-8000-000000000401","task_id":"018f0000-0000-7000-8000-000000000401","project_id":"018f0000-0000-7000-8000-000000000003","contract_id":"018f0000-0000-7000-8000-000000000009","approved_contract_id":"018f0000-0000-7000-8000-000000000010","plan_id":"018f0000-0000-7000-8000-000000000301","proposal_id":"018f0000-0000-7000-8000-000000000302","repo_binding_id":"018f0000-0000-7000-8000-000000000004","status":"planned","title":"DOGFOOD-005: WorkItem detail visibility","summary":"Add read-only WorkItem detail visibility.","scope":["Add goalrail work item show"],"acceptance_refs":["acceptance_criteria[0]"],"proof_expectation_refs":["proof_expectations[0]"],"next_action":{"kind":"prepare_checkout","blocking":false,"available":true,"command":"goalrail work checkout prepare --task-id 018f0000-0000-7000-8000-000000000401 --format json"}}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, server.URL)

	var stdout, stderr bytes.Buffer
	err := RunWithOptions(context.Background(), term.New(&stdout, &stderr), repoDir, []string{"item", "show", "--task-id", "018f0000-0000-7000-8000-000000000401", "--format", "text"}, Options{
		Store: fakeSessionStore{session: validSession(server.URL)},
		Now:   func() time.Time { return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("Run(work item show text) error = %v", err)
	}
	got := stdout.String()
	for _, want := range []string{
		"WorkItem detail",
		"Identity / lineage",
		"Read-only note",
		"did not start checkout",
		"goalrail work checkout prepare --task-id 018f0000-0000-7000-8000-000000000401 --format json",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("text output missing %q:\n%s", want, got)
		}
	}
}

func TestRunWorkItemShowRequiresTaskID(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	err := RunWithOptions(context.Background(), term.New(&stdout, &stderr), t.TempDir(), []string{"item", "show", "--format", "json"}, Options{
		Store: fakeSessionStore{session: validSession("https://goalrail.example")},
	})
	if err == nil {
		t.Fatal("Run(work item show without task-id) error = nil, want usage error")
	}
	if exitcode.ForError(err) != exitcode.Usage {
		t.Fatalf("exit code = %d, want usage: %v", exitcode.ForError(err), err)
	}
}

func TestRunWorkItemHelpUsage(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	if err := Run(context.Background(), term.New(&stdout, &stderr), t.TempDir(), []string{"item", "--help"}); err != nil {
		t.Fatalf("Run(work item --help) error = %v", err)
	}
	if !strings.Contains(stdout.String(), "show   inspect a materialized WorkItem") {
		t.Fatalf("item help = %q, want show command", stdout.String())
	}
	stdout.Reset()
	if err := Run(context.Background(), term.New(&stdout, &stderr), t.TempDir(), []string{"item", "show", "--help"}); err != nil {
		t.Fatalf("Run(work item show --help) error = %v", err)
	}
	if !strings.Contains(stdout.String(), "read-only") {
		t.Fatalf("item show help = %q, want read-only note", stdout.String())
	}
}

func TestRunCheckoutPrepareCreatesCheckoutJob(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	var meCount, checkoutCount atomic.Int32
	var request workPlanCreateRequest
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
		case "/v1/tasks/018f0000-0000-7000-8000-000000000401/checkout-jobs":
			checkoutCount.Add(1)
			if r.Method != http.MethodPost {
				t.Errorf("POST /v1/tasks/{id}/checkout-jobs method = %s", r.Method)
			}
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&request); err != nil {
				t.Errorf("decode checkout job request: %v", err)
				http.Error(w, "bad json", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"018f0000-0000-7000-8000-000000000501","task_id":"018f0000-0000-7000-8000-000000000401","contract_id":"018f0000-0000-7000-8000-000000000009","approved_contract_id":"018f0000-0000-7000-8000-000000000010","plan_id":"018f0000-0000-7000-8000-000000000301","proposal_id":"018f0000-0000-7000-8000-000000000302","repo_binding_id":"018f0000-0000-7000-8000-000000000004","state":"queued","instruction":{"job_id":"018f0000-0000-7000-8000-000000000501","task_id":"018f0000-0000-7000-8000-000000000401","repo_binding_id":"018f0000-0000-7000-8000-000000000004","access_mode":"customer_mounted_workspace","provider":"github","repository_full_name":"heurema/goalrail","repository_url":"https://github.com/heurema/goalrail","workflow_base_branch":"main","path_scope":".","source_ref":{"kind":"work_item","id":"018f0000-0000-7000-8000-000000000401"},"raw_source_uploaded":false}}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, server.URL)

	output, err := runCheckoutPrepareJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--task-id", "018F0000-0000-7000-8000-000000000401", "--format", "json")
	if err != nil {
		t.Fatalf("Run(work checkout prepare) error = %v", err)
	}
	if output.SchemaVersion != "goalrail.cli.v1" || output.TaskID != "018f0000-0000-7000-8000-000000000401" || output.CheckoutJobID != "018f0000-0000-7000-8000-000000000501" {
		t.Fatalf("output = %#v, want checkout job envelope", output)
	}
	if output.Instruction.RawSourceUploaded || output.Instruction.RepositoryFullName != "heurema/goalrail" {
		t.Fatalf("instruction = %#v, want no raw source and repository metadata", output.Instruction)
	}
	if output.NextAction.Kind != "runner_checkout_required" || !output.NextAction.Blocking || output.NextAction.Available {
		t.Fatalf("next_action = %#v, want unavailable runner checkout requirement", output.NextAction)
	}
	if request.ProjectID != "018f0000-0000-7000-8000-000000000003" || request.RepoBindingID != "018f0000-0000-7000-8000-000000000004" {
		t.Fatalf("checkout request project/repo = %q/%q, want marker context", request.ProjectID, request.RepoBindingID)
	}
	if meCount.Load() != 1 || checkoutCount.Load() != 1 {
		t.Fatalf("request counts me/checkout = %d/%d, want 1/1", meCount.Load(), checkoutCount.Load())
	}
}

func TestRunExecutionPrepareCreatesExecutionJob(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	var request executionJobCreateRequest
	var meCount, executionCount atomic.Int32
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
		case "/v1/tasks/018f0000-0000-7000-8000-000000000401/execution-jobs":
			executionCount.Add(1)
			if r.Method != http.MethodPost {
				t.Errorf("POST /v1/tasks/{id}/execution-jobs method = %s", r.Method)
			}
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&request); err != nil {
				t.Errorf("decode execution job request: %v", err)
				http.Error(w, "bad json", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"018f0000-0000-7000-8000-000000000601","task_id":"018f0000-0000-7000-8000-000000000401","contract_id":"018f0000-0000-7000-8000-000000000009","approved_contract_id":"018f0000-0000-7000-8000-000000000010","plan_id":"018f0000-0000-7000-8000-000000000301","proposal_id":"018f0000-0000-7000-8000-000000000302","repo_binding_id":"018f0000-0000-7000-8000-000000000004","checkout_job_id":"018f0000-0000-7000-8000-000000000501","checkout_receipt_id":"018f0000-0000-7000-8000-000000000502","state":"queued","execution_mode":"prepare_v0"}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, server.URL)

	output, err := runExecutionPrepareJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--task-id", "018F0000-0000-7000-8000-000000000401", "--checkout-receipt-id", "018F0000-0000-7000-8000-000000000502", "--format", "json")
	if err != nil {
		t.Fatalf("Run(work execution prepare) error = %v", err)
	}
	if output.SchemaVersion != "goalrail.cli.v1" || output.TaskID != "018f0000-0000-7000-8000-000000000401" || output.CheckoutReceiptID != "018f0000-0000-7000-8000-000000000502" || output.ExecutionJobID != "018f0000-0000-7000-8000-000000000601" {
		t.Fatalf("output = %#v, want execution job envelope", output)
	}
	if output.ExecutionJobState != "queued" {
		t.Fatalf("execution_job_state = %q, want queued", output.ExecutionJobState)
	}
	if output.NextAction.Kind != "runner_execution_required" || !output.NextAction.Blocking || output.NextAction.Available || output.NextAction.PlannedSlice != "H2.3" {
		t.Fatalf("next_action = %#v, want unavailable runner execution requirement", output.NextAction)
	}
	if request.ProjectID != "018f0000-0000-7000-8000-000000000003" || request.RepoBindingID != "018f0000-0000-7000-8000-000000000004" || request.CheckoutReceiptID != "018f0000-0000-7000-8000-000000000502" {
		t.Fatalf("execution request = %#v, want marker context and checkout receipt", request)
	}
	if meCount.Load() != 1 || executionCount.Load() != 1 {
		t.Fatalf("request counts me/execution = %d/%d, want 1/1", meCount.Load(), executionCount.Load())
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

func TestRunStartTextMakesContinuationPrimaryNextStep(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	server := newWorkStartFakeServer(t)
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, server.URL)

	var stdout, stderr bytes.Buffer
	err := RunWithOptions(context.Background(), term.New(&stdout, &stderr), repoDir, []string{"start", "--title", "Text output"}, Options{
		Store: fakeSessionStore{session: validSession(server.URL)},
		Now:   func() time.Time { return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("Run(work start text) error = %v", err)
	}
	got := stdout.String()
	wantNext := "Next: goalrail work continue --goal-id 018f0000-0000-7000-8000-000000000006 --format json"
	if !strings.Contains(got, wantNext) {
		t.Fatalf("stdout = %q, want continuation next command", got)
	}
	if strings.Contains(got, "not available yet") {
		t.Fatalf("stdout = %q, want available continuation without planned warning", got)
	}
}

func TestRunStartBodyFilePathTrimsLikeBodyFlag(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	bodyPath := filepath.Join(repoDir, "task.txt")
	if err := os.WriteFile(bodyPath, []byte("\nFile body\n\n"), 0o644); err != nil {
		t.Fatalf("write body file: %v", err)
	}
	var intakeRequest intakeSubmission
	server := newWorkStartFakeServerWithIntake(t, &intakeRequest)
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, server.URL)

	if _, err := runStartJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--title", "File body", "--body-file", bodyPath, "--format", "json"); err != nil {
		t.Fatalf("Run(work start --body-file path) error = %v", err)
	}
	if intakeRequest.Body != "File body" {
		t.Fatalf("intake body = %q, want trimmed file body", intakeRequest.Body)
	}
}

func TestRunStartBodyFileDashTrimsLikeBodyFlag(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	var intakeRequest intakeSubmission
	server := newWorkStartFakeServerWithIntake(t, &intakeRequest)
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, server.URL)

	if _, err := runStartJSONWithOptions(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, Options{Stdin: strings.NewReader("\nstdin body\n")}, "--title", "Stdin body", "--body-file", "-", "--format", "json"); err != nil {
		t.Fatalf("Run(work start --body-file -) error = %v", err)
	}
	if intakeRequest.Body != "stdin body" {
		t.Fatalf("intake body = %q, want trimmed stdin body", intakeRequest.Body)
	}
}

func TestRunStartRejectsBodyWithBodyFile(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	server := newWorkStartFakeServer(t)
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, server.URL)

	_, err := runStartJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--title", "Conflict", "--body", "inline", "--body-file", "-", "--format", "json")
	if err == nil {
		t.Fatal("Run(work start --body --body-file) error = nil, want usage error")
	}
	if got := exitcode.ForError(err); got != exitcode.Usage {
		t.Fatalf("exit code = %d, want usage", got)
	}
	if !strings.Contains(err.Error(), "--body and --body-file cannot be used together") {
		t.Fatalf("error = %q, want body conflict", err.Error())
	}
}

func TestRunStartRejectsOversizedBodyFileStdin(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	server := newWorkStartFakeServer(t)
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, server.URL)

	stdin := strings.NewReader(strings.Repeat("x", maxWorkBodyBytes+1))
	_, err := runStartJSONWithOptions(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, Options{Stdin: stdin}, "--title", "Oversized stdin", "--body-file", "-", "--format", "json")
	if err == nil {
		t.Fatal("Run(work start oversized stdin) error = nil, want validation error")
	}
	if got := exitcode.ForError(err); got != exitcode.Validation {
		t.Fatalf("exit code = %d, want validation", got)
	}
	if !strings.Contains(err.Error(), "--body-file - from stdin exceeds") {
		t.Fatalf("error = %q, want oversized stdin error", err.Error())
	}
}

func TestRunStartRejectsOversizedBodyFilePath(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	bodyPath := filepath.Join(repoDir, "large-task.txt")
	if err := os.WriteFile(bodyPath, []byte(strings.Repeat("x", maxWorkBodyBytes+1)), 0o644); err != nil {
		t.Fatalf("write oversized body file: %v", err)
	}
	server := newWorkStartFakeServer(t)
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, server.URL)

	_, err := runStartJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--title", "Oversized file", "--body-file", bodyPath, "--format", "json")
	if err == nil {
		t.Fatal("Run(work start oversized file) error = nil, want validation error")
	}
	if got := exitcode.ForError(err); got != exitcode.Validation {
		t.Fatalf("exit code = %d, want validation", got)
	}
	if !strings.Contains(err.Error(), "--body-file "+bodyPath+" exceeds") {
		t.Fatalf("error = %q, want oversized file error", err.Error())
	}
}

func runStartJSON(t *testing.T, workDir string, store fakeSessionStore, args ...string) (spine.WorkStartOutput, error) {
	t.Helper()

	return runStartJSONWithOptions(t, workDir, store, Options{}, args...)
}

func runStartJSONWithOptions(t *testing.T, workDir string, store fakeSessionStore, extra Options, args ...string) (spine.WorkStartOutput, error) {
	t.Helper()

	var stdout, stderr bytes.Buffer
	extra.Store = store
	extra.Now = func() time.Time { return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC) }
	err := RunWithOptions(context.Background(), term.New(&stdout, &stderr), workDir, append([]string{"start"}, args...), Options{
		Store:      extra.Store,
		HTTPClient: extra.HTTPClient,
		Now:        extra.Now,
		Stdin:      extra.Stdin,
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

func runContinueJSON(t *testing.T, workDir string, store fakeSessionStore, args ...string) (spine.WorkContinueOutput, error) {
	t.Helper()

	var stdout, stderr bytes.Buffer
	err := RunWithOptions(context.Background(), term.New(&stdout, &stderr), workDir, append([]string{"continue"}, args...), Options{
		Store: store,
		Now:   func() time.Time { return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		return spine.WorkContinueOutput{}, err
	}

	var output spine.WorkContinueOutput
	decoder := json.NewDecoder(&stdout)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&output); err != nil {
		t.Fatalf("decode work continue JSON %q: %v", stdout.String(), err)
	}
	return output, nil
}

func runAnswerJSON(t *testing.T, workDir string, store fakeSessionStore, args ...string) (spine.WorkAnswerOutput, error) {
	t.Helper()

	return runAnswerJSONWithOptions(t, workDir, store, Options{}, args...)
}

func runAnswerJSONWithOptions(t *testing.T, workDir string, store fakeSessionStore, extra Options, args ...string) (spine.WorkAnswerOutput, error) {
	t.Helper()

	var stdout, stderr bytes.Buffer
	extra.Store = store
	extra.Now = func() time.Time { return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC) }
	err := RunWithOptions(context.Background(), term.New(&stdout, &stderr), workDir, append([]string{"answer"}, args...), Options{
		Store:      extra.Store,
		HTTPClient: extra.HTTPClient,
		Now:        extra.Now,
		Stdin:      extra.Stdin,
	})
	if err != nil {
		return spine.WorkAnswerOutput{}, err
	}

	var output spine.WorkAnswerOutput
	decoder := json.NewDecoder(&stdout)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&output); err != nil {
		t.Fatalf("decode work answer JSON %q: %v", stdout.String(), err)
	}
	return output, nil
}

func runPlanJSON(t *testing.T, workDir string, store fakeSessionStore, args ...string) (spine.WorkPlanOutput, error) {
	t.Helper()

	var stdout, stderr bytes.Buffer
	err := RunWithOptions(context.Background(), term.New(&stdout, &stderr), workDir, append([]string{"plan"}, args...), Options{
		Store: store,
		Now:   func() time.Time { return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		return spine.WorkPlanOutput{}, err
	}

	var output spine.WorkPlanOutput
	decoder := json.NewDecoder(&stdout)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&output); err != nil {
		t.Fatalf("decode work plan JSON %q: %v", stdout.String(), err)
	}
	return output, nil
}

func runPlanStatusJSON(t *testing.T, workDir string, store fakeSessionStore, args ...string) (spine.WorkPlanStatusOutput, error) {
	t.Helper()

	var stdout, stderr bytes.Buffer
	err := RunWithOptions(context.Background(), term.New(&stdout, &stderr), workDir, append([]string{"plan", "status"}, args...), Options{
		Store: store,
		Now:   func() time.Time { return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		return spine.WorkPlanStatusOutput{}, err
	}

	var output spine.WorkPlanStatusOutput
	decoder := json.NewDecoder(&stdout)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&output); err != nil {
		t.Fatalf("decode work plan status JSON %q: %v", stdout.String(), err)
	}
	return output, nil
}

func runProposalAcceptJSON(t *testing.T, workDir string, store fakeSessionStore, args ...string) (spine.WorkProposalAcceptOutput, error) {
	t.Helper()

	var stdout, stderr bytes.Buffer
	err := RunWithOptions(context.Background(), term.New(&stdout, &stderr), workDir, append([]string{"proposal", "accept"}, args...), Options{
		Store: store,
		Now:   func() time.Time { return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		return spine.WorkProposalAcceptOutput{}, err
	}

	var output spine.WorkProposalAcceptOutput
	decoder := json.NewDecoder(&stdout)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&output); err != nil {
		t.Fatalf("decode work proposal accept JSON %q: %v", stdout.String(), err)
	}
	return output, nil
}

func runWorkItemShowJSON(t *testing.T, workDir string, store fakeSessionStore, args ...string) (spine.WorkItemShowOutput, error) {
	t.Helper()

	var stdout, stderr bytes.Buffer
	err := RunWithOptions(context.Background(), term.New(&stdout, &stderr), workDir, append([]string{"item", "show"}, args...), Options{
		Store: store,
		Now:   func() time.Time { return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		return spine.WorkItemShowOutput{}, err
	}

	var output spine.WorkItemShowOutput
	decoder := json.NewDecoder(&stdout)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&output); err != nil {
		t.Fatalf("decode work item show JSON %q: %v", stdout.String(), err)
	}
	return output, nil
}

func runCheckoutPrepareJSON(t *testing.T, workDir string, store fakeSessionStore, args ...string) (spine.WorkCheckoutPrepareOutput, error) {
	t.Helper()

	var stdout, stderr bytes.Buffer
	err := RunWithOptions(context.Background(), term.New(&stdout, &stderr), workDir, append([]string{"checkout", "prepare"}, args...), Options{
		Store: store,
		Now:   func() time.Time { return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		return spine.WorkCheckoutPrepareOutput{}, err
	}

	var output spine.WorkCheckoutPrepareOutput
	decoder := json.NewDecoder(&stdout)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&output); err != nil {
		t.Fatalf("decode work checkout prepare JSON %q: %v", stdout.String(), err)
	}
	return output, nil
}

func runExecutionPrepareJSON(t *testing.T, workDir string, store fakeSessionStore, args ...string) (spine.WorkExecutionPrepareOutput, error) {
	t.Helper()

	var stdout, stderr bytes.Buffer
	err := RunWithOptions(context.Background(), term.New(&stdout, &stderr), workDir, append([]string{"execution", "prepare"}, args...), Options{
		Store: store,
		Now:   func() time.Time { return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		return spine.WorkExecutionPrepareOutput{}, err
	}

	var output spine.WorkExecutionPrepareOutput
	decoder := json.NewDecoder(&stdout)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&output); err != nil {
		t.Fatalf("decode work execution prepare JSON %q: %v", stdout.String(), err)
	}
	return output, nil
}

func newWorkStartFakeServer(t *testing.T) *httptest.Server {
	t.Helper()

	return newWorkStartFakeServerWithIntake(t, nil)
}

func newWorkStartFakeServerWithIntake(t *testing.T, intakeRequest *intakeSubmission) *httptest.Server {
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
			if intakeRequest != nil {
				decoder := json.NewDecoder(r.Body)
				decoder.DisallowUnknownFields()
				if err := decoder.Decode(intakeRequest); err != nil {
					t.Errorf("decode intake request: %v", err)
					http.Error(w, "bad json", http.StatusBadRequest)
					return
				}
			}
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

	writeProjectConfigFixtureWithIDs(t, repoDir, serverURL, "018f0000-0000-7000-8000-000000000003", "018f0000-0000-7000-8000-000000000004")
}

func writeProjectConfigFixtureWithIDs(t *testing.T, repoDir string, serverURL string, projectID string, repoBindingID string) {
	t.Helper()

	configPath := filepath.Join(repoDir, projectconfig.RelativePath)
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("create .goalrail dir: %v", err)
	}
	content := projectconfig.RenderYAML(projectconfig.Config{
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

type failOnRead struct {
	t *testing.T
}

func (reader failOnRead) Read([]byte) (int, error) {
	reader.t.Helper()
	reader.t.Fatal("stdin was read before command prerequisites were validated")
	return 0, errors.New("unexpected stdin read")
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
