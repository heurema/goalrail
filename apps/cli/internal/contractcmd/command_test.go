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
	if output.NextAction.Kind != "update_contract" || !output.NextAction.Available || output.NextAction.PlannedSlice != "" {
		t.Fatalf("next_action = %#v, want available update_contract", output.NextAction)
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

func TestBuildDraftOutputDoesNotOfferUpdateForNonDraftContract(t *testing.T) {
	t.Parallel()

	output := buildDraftOutput(projectconfig.Config{
		ServerURL:      "https://goalrail.example",
		OrganizationID: "018f0000-0000-7000-8000-000000000002",
		ProjectID:      "018f0000-0000-7000-8000-000000000003",
		RepoBindingID:  "018f0000-0000-7000-8000-000000000004",
	}, "https://goalrail.example", "018f0000-0000-7000-8000-000000000006", contractDraftResponse{
		ID:            "018f0000-0000-7000-8000-000000000009",
		RepoBindingID: "018f0000-0000-7000-8000-000000000004",
		GoalID:        "018f0000-0000-7000-8000-000000000006",
		State:         spine.ContractState("ready_for_approval"),
	}, spine.LocalRepoReceipt{})

	if output.NextAction.Kind != "update_contract" || output.NextAction.Available || output.NextAction.Command != "" {
		t.Fatalf("next_action = %#v, want unavailable update_contract without command", output.NextAction)
	}
	got := renderDraftText(output)
	if strings.Contains(got, "goalrail contract update") {
		t.Fatalf("draft text = %q, should not offer contract update command for non-draft contract", got)
	}
	if !strings.Contains(got, "Contract update is only available while the Contract is draft.") {
		t.Fatalf("draft text = %q, want non-draft update warning", got)
	}
}

func TestRunShowReturnsContractAndCurrentDraftJSON(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	var meCount, contractCount, draftCount atomic.Int32
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
			_, _ = w.Write([]byte(`{"user":{"id":"018f0000-0000-7000-8000-000000000001","display_name":"Developer"},"organization_membership":{"organization_id":"018f0000-0000-7000-8000-000000000002","role":"member","state":"active"}}`))
		case "/v1/contracts/018f0000-0000-7000-8000-000000000009":
			contractCount.Add(1)
			if r.Method != http.MethodGet {
				t.Errorf("GET /v1/contracts/{id} method = %s", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"018f0000-0000-7000-8000-000000000009","repo_binding_id":"018f0000-0000-7000-8000-000000000004","goal_id":"018f0000-0000-7000-8000-000000000006","state":"ready_for_approval","current_seed_id":"018f0000-0000-7000-8000-000000000007","current_draft_id":"018f0000-0000-7000-8000-000000000008"}`))
		case "/v1/contracts/018f0000-0000-7000-8000-000000000009/current-draft":
			draftCount.Add(1)
			if r.Method != http.MethodGet {
				t.Errorf("GET /v1/contracts/{id}/current-draft method = %s", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"018f0000-0000-7000-8000-000000000008","contract_id":"018f0000-0000-7000-8000-000000000009","contract_seed_id":"018f0000-0000-7000-8000-000000000007","repo_binding_id":"018f0000-0000-7000-8000-000000000004","goal_id":"018f0000-0000-7000-8000-000000000006","state":"ready_for_approval","title":"Review contract from CLI","intent_summary":"Show a contract before approval.","proposed_scope":["Add goalrail contract show","Support text and JSON output"],"proposed_non_goals":["Do not approve"],"proposed_constraints":["Read-only"],"proposed_acceptance_criteria":["Developer can inspect before approval"],"proposed_expected_checks":["go test ./..."],"proposed_proof_expectations":["PR shows command output"],"risk_hints":["approval-ux"]}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, server.URL)

	output, err := runShowJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--contract-id", "018f0000-0000-7000-8000-000000000009", "--format", "json")
	if err != nil {
		t.Fatalf("Run(contract show) error = %v", err)
	}

	if output.SchemaVersion != "goalrail.cli.v1" || output.Mode != "server" {
		t.Fatalf("schema/mode = %q/%q, want goalrail.cli.v1/server", output.SchemaVersion, output.Mode)
	}
	if output.ContractID != "018f0000-0000-7000-8000-000000000009" || output.ContractState != spine.ContractStateReadyForApproval {
		t.Fatalf("contract = %q/%q, want ready contract", output.ContractID, output.ContractState)
	}
	if output.GoalID != "018f0000-0000-7000-8000-000000000006" || output.RepoBindingID != "018f0000-0000-7000-8000-000000000004" {
		t.Fatalf("goal/repo = %q/%q, want aggregate context", output.GoalID, output.RepoBindingID)
	}
	if output.CurrentSeedID != "018f0000-0000-7000-8000-000000000007" || output.CurrentDraftID != "018f0000-0000-7000-8000-000000000008" {
		t.Fatalf("seed/draft = %q/%q, want current ids", output.CurrentSeedID, output.CurrentDraftID)
	}
	if output.CurrentDraft == nil {
		t.Fatal("current_draft = nil, want draft body")
	}
	if output.CurrentDraft.Title != "Review contract from CLI" || output.CurrentDraft.IntentSummary != "Show a contract before approval." {
		t.Fatalf("draft title/intent = %q/%q, want server draft fields", output.CurrentDraft.Title, output.CurrentDraft.IntentSummary)
	}
	if !equalStrings(output.CurrentDraft.ProposedScope, []string{"Add goalrail contract show", "Support text and JSON output"}) {
		t.Fatalf("draft scope = %#v, want server scope", output.CurrentDraft.ProposedScope)
	}
	if output.NextAction.Kind != "approve_contract" || !output.NextAction.Blocking || !output.NextAction.Available {
		t.Fatalf("next_action = %#v, want blocking approve_contract", output.NextAction)
	}
	if output.NextAction.Command != "goalrail contract approve --contract-id 018f0000-0000-7000-8000-000000000009 --confirm-user-approval --format json" {
		t.Fatalf("next_action.command = %q, want approve command", output.NextAction.Command)
	}
	if meCount.Load() != 1 || contractCount.Load() != 1 || draftCount.Load() != 1 {
		t.Fatalf("request counts me/contract/draft = %d/%d/%d, want 1/1/1", meCount.Load(), contractCount.Load(), draftCount.Load())
	}
}

func TestRunShowRefreshesExpiredAccessTokenBeforeRequests(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	var refreshCount, meCount, contractCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/auth/refresh":
			refreshCount.Add(1)
			if r.Method != http.MethodPost {
				t.Errorf("refresh method = %s, want POST", r.Method)
			}
			if r.Header.Get("Authorization") != "" {
				t.Errorf("refresh Authorization = %q, want empty", r.Header.Get("Authorization"))
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"refreshed-access-token","access_token_expires_at":"2026-05-05T11:30:00Z","token_type":"Bearer"}`))
		case "/v1/me":
			meCount.Add(1)
			if r.Header.Get("Authorization") != "Bearer refreshed-access-token" {
				t.Errorf("me Authorization = %q, want refreshed bearer", r.Header.Get("Authorization"))
				http.Error(w, "bad auth", http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"user":{"id":"018f0000-0000-7000-8000-000000000001","display_name":"Developer"},"organization_membership":{"organization_id":"018f0000-0000-7000-8000-000000000002","role":"member","state":"active"}}`))
		case "/v1/contracts/018f0000-0000-7000-8000-000000000009":
			contractCount.Add(1)
			if r.Header.Get("Authorization") != "Bearer refreshed-access-token" {
				t.Errorf("contract Authorization = %q, want refreshed bearer", r.Header.Get("Authorization"))
				http.Error(w, "bad auth", http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"018f0000-0000-7000-8000-000000000009","repo_binding_id":"018f0000-0000-7000-8000-000000000004","goal_id":"018f0000-0000-7000-8000-000000000006","state":"approved"}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, server.URL)

	store := &fakeSavingSessionStore{session: authstore.Session{
		ServerURL:            server.URL,
		AccessToken:          "expired-access-token",
		RefreshToken:         "stored-refresh-token",
		AccessTokenExpiresAt: time.Date(2026, 5, 5, 9, 59, 0, 0, time.UTC),
		TokenType:            "Bearer",
	}}
	output, err := runShowJSON(t, repoDir, store, "--contract-id", "018f0000-0000-7000-8000-000000000009", "--format", "json")
	if err != nil {
		t.Fatalf("Run(contract show) error = %v", err)
	}
	if output.ContractID != "018f0000-0000-7000-8000-000000000009" || output.ContractState != spine.ContractStateApproved {
		t.Fatalf("contract output = %q/%q, want approved contract", output.ContractID, output.ContractState)
	}
	if refreshCount.Load() != 1 || meCount.Load() != 1 || contractCount.Load() != 1 || store.saveCount.Load() != 1 {
		t.Fatalf("counts refresh/me/contract/save = %d/%d/%d/%d, want 1/1/1/1", refreshCount.Load(), meCount.Load(), contractCount.Load(), store.saveCount.Load())
	}
	if store.saved.AccessToken != "refreshed-access-token" || store.saved.RefreshToken != "stored-refresh-token" {
		t.Fatalf("saved session = %#v, want refreshed access token and preserved refresh token", store.saved)
	}
}

func TestRunShowTextRendersReviewAndApprovalNote(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/me":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"user":{"id":"018f0000-0000-7000-8000-000000000001","display_name":"Developer"},"organization_membership":{"organization_id":"018f0000-0000-7000-8000-000000000002","role":"member","state":"active"}}`))
		case "/v1/contracts/018f0000-0000-7000-8000-000000000009":
			if r.Method != http.MethodGet {
				t.Errorf("contract method = %s, want GET", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"018f0000-0000-7000-8000-000000000009","repo_binding_id":"018f0000-0000-7000-8000-000000000004","goal_id":"018f0000-0000-7000-8000-000000000006","state":"ready_for_approval","current_draft_id":"018f0000-0000-7000-8000-000000000008"}`))
		case "/v1/contracts/018f0000-0000-7000-8000-000000000009/current-draft":
			if r.Method != http.MethodGet {
				t.Errorf("draft method = %s, want GET", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"018f0000-0000-7000-8000-000000000008","contract_id":"018f0000-0000-7000-8000-000000000009","repo_binding_id":"018f0000-0000-7000-8000-000000000004","state":"ready_for_approval","title":"Review contract from CLI","intent_summary":"Show a contract before approval.","proposed_scope":["Add goalrail contract show"],"proposed_non_goals":["Do not approve"],"proposed_constraints":["Read-only"],"proposed_acceptance_criteria":["Developer can inspect before approval"],"proposed_expected_checks":["go test ./..."],"proposed_proof_expectations":["PR shows command output"],"risk_hints":["approval-ux"]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, server.URL)

	var stdout, stderr bytes.Buffer
	err := RunWithOptions(context.Background(), term.New(&stdout, &stderr), repoDir, []string{"show", "--contract-id", "018f0000-0000-7000-8000-000000000009"}, Options{
		Store: fakeSessionStore{session: validSession(server.URL)},
		Now:   func() time.Time { return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("Run(contract show text) error = %v", err)
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	got := stdout.String()
	for _, want := range []string{
		"Contract review",
		"Contract identity",
		"Title\nReview contract from CLI",
		"Proposed scope\n- Add goalrail contract show",
		"Proposed non-goals\n- Do not approve",
		"This command is read-only and does not approve or mutate the Contract.",
		"Human approval is required before running: goalrail contract approve --contract-id 018f0000-0000-7000-8000-000000000009 --confirm-user-approval",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("stdout = %q, want containing %q", got, want)
		}
	}
}

func TestRunShowRequiresContractID(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	err := RunWithOptions(context.Background(), term.New(&stdout, &stderr), t.TempDir(), []string{"show", "--format", "json"}, Options{})
	if err == nil {
		t.Fatal("Run(contract show) error = nil, want missing contract id")
	}
	if got := exitcode.ForError(err); got != exitcode.Usage {
		t.Fatalf("exit code = %d, want usage", got)
	}
	if !strings.Contains(err.Error(), "--contract-id is required") {
		t.Fatalf("error = %q, want contract id hint", err.Error())
	}
}

func TestRunShowHandlesMissingCurrentDraftID(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	var draftCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/me":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"user":{"id":"018f0000-0000-7000-8000-000000000001","display_name":"Developer"},"organization_membership":{"organization_id":"018f0000-0000-7000-8000-000000000002","role":"member","state":"active"}}`))
		case "/v1/contracts/018f0000-0000-7000-8000-000000000009":
			if r.Method != http.MethodGet {
				t.Errorf("contract method = %s, want GET", r.Method)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"018f0000-0000-7000-8000-000000000009","repo_binding_id":"018f0000-0000-7000-8000-000000000004","goal_id":"018f0000-0000-7000-8000-000000000006","state":"seeded","current_seed_id":"018f0000-0000-7000-8000-000000000007"}`))
		case "/v1/contracts/018f0000-0000-7000-8000-000000000009/current-draft":
			draftCount.Add(1)
			http.Error(w, "unexpected current draft request", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, server.URL)

	output, err := runShowJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--contract-id", "018f0000-0000-7000-8000-000000000009", "--format", "json")
	if err != nil {
		t.Fatalf("Run(contract show) error = %v", err)
	}
	if output.CurrentDraft != nil {
		t.Fatalf("current_draft = %#v, want nil without current_draft_id", output.CurrentDraft)
	}
	if output.CurrentSeedID != "018f0000-0000-7000-8000-000000000007" || output.CurrentDraftID != "" {
		t.Fatalf("seed/draft ids = %q/%q, want seed only", output.CurrentSeedID, output.CurrentDraftID)
	}
	if draftCount.Load() != 0 {
		t.Fatalf("current draft requests = %d, want 0 without current_draft_id", draftCount.Load())
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
	if !strings.Contains(got, "Next: goalrail contract update") {
		t.Fatalf("stdout = %q, want available update_contract next command", got)
	}
	for _, forbidden := range []string{"workitem", "runner", "proof", "verified"} {
		if strings.Contains(strings.ToLower(got), forbidden) {
			t.Fatalf("stdout = %q, want no %q claim", got, forbidden)
		}
	}
}

func TestRunUpdateSubmitsStructuredFieldsFromFile(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	fieldsPath := filepath.Join(repoDir, "fields.json")
	if err := os.WriteFile(fieldsPath, []byte(`{
		"proposed_scope": ["Refactor CSV export filters"],
		"proposed_non_goals": ["Do not change billing UI"],
		"proposed_acceptance_criteria": ["CSV export behavior is preserved"],
		"proposed_verification": ["go test ./..."],
		"context_refs": [
			{
				"kind": "local_repo_receipt",
				"id": "receipt-1",
				"baseline_id": "baseline-1",
				"overlay_id": "overlay-1"
			}
		],
		"unknowns": ["Confirm rollout window"]
	}`), 0o644); err != nil {
		t.Fatalf("write fields file: %v", err)
	}

	var meCount, updateCount atomic.Int32
	var request contractUpdateRequest
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
		case "/v1/contracts/018f0000-0000-7000-8000-000000000009":
			updateCount.Add(1)
			if r.Method != http.MethodPatch {
				t.Errorf("PATCH /v1/contracts/{id} method = %s", r.Method)
			}
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&request); err != nil {
				t.Errorf("decode contract update request: %v", err)
				http.Error(w, "bad json", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"018f0000-0000-7000-8000-000000000009","repo_binding_id":"018f0000-0000-7000-8000-000000000004","goal_id":"018f0000-0000-7000-8000-000000000006","state":"draft","current_seed_id":"018f0000-0000-7000-8000-000000000007","current_draft_id":"018f0000-0000-7000-8000-000000000008"}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, server.URL)

	output, err := runUpdateJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "", "--contract-id", "018f0000-0000-7000-8000-000000000009", "--fields-file", fieldsPath, "--format", "json")
	if err != nil {
		t.Fatalf("Run(contract update) error = %v", err)
	}

	if output.SchemaVersion != "goalrail.cli.v1" {
		t.Fatalf("schema_version = %q, want goalrail.cli.v1", output.SchemaVersion)
	}
	if output.ContractID != "018f0000-0000-7000-8000-000000000009" || output.ContractState != "draft" {
		t.Fatalf("contract/state = %q/%q, want updated draft", output.ContractID, output.ContractState)
	}
	wantChanged := []string{"proposed_acceptance_criteria", "proposed_expected_checks", "proposed_non_goals", "proposed_scope"}
	if !equalStrings(output.ChangedFields, wantChanged) {
		t.Fatalf("changed_fields = %#v, want %#v", output.ChangedFields, wantChanged)
	}
	if output.Display.Summary == "" {
		t.Fatal("display.summary is empty")
	}
	if output.NextAction.Kind != "review_contract" || !output.NextAction.Available || !output.NextAction.Blocking {
		t.Fatalf("next_action = %#v, want available blocking review_contract", output.NextAction)
	}
	if request.ProjectID != "018f0000-0000-7000-8000-000000000003" || request.RepoBindingID != "018f0000-0000-7000-8000-000000000004" {
		t.Fatalf("update request project/repo = %q/%q, want local marker context", request.ProjectID, request.RepoBindingID)
	}
	if request.UpdatedBy.Kind != "user" || request.UpdatedBy.ID != "018f0000-0000-7000-8000-000000000001" {
		t.Fatalf("updated_by = %#v, want current user actor", request.UpdatedBy)
	}
	if _, ok := request.Changes["proposed_verification"]; ok {
		t.Fatal("request used proposed_verification instead of canonical proposed_expected_checks")
	}
	if got := decodeRawStringSlice(t, request.Changes["proposed_expected_checks"]); !equalStrings(got, []string{"go test ./..."}) {
		t.Fatalf("proposed_expected_checks = %#v, want mapped verification", got)
	}
	if len(request.ContextRefs) != 1 || request.ContextRefs[0].BaselineID != "baseline-1" || request.ContextRefs[0].OverlayID != "overlay-1" {
		t.Fatalf("context_refs = %#v, want structured local repo receipt ref", request.ContextRefs)
	}
	if !equalStrings(request.Unknowns, []string{"Confirm rollout window"}) {
		t.Fatalf("unknowns = %#v, want structured unknowns", request.Unknowns)
	}
	if meCount.Load() != 1 || updateCount.Load() != 1 {
		t.Fatalf("request counts me/update = %d/%d, want 1/1", meCount.Load(), updateCount.Load())
	}
}

func TestRunUpdateReadsFieldsFromStdin(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	var updateCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/me":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"user":{"id":"018f0000-0000-7000-8000-000000000001","display_name":"Developer"},"organization_membership":{"organization_id":"018f0000-0000-7000-8000-000000000002","role":"member","state":"active"}}`))
		case "/v1/contracts/018f0000-0000-7000-8000-000000000009":
			updateCount.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"018f0000-0000-7000-8000-000000000009","repo_binding_id":"018f0000-0000-7000-8000-000000000004","state":"draft"}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, server.URL)

	output, err := runUpdateJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, `{"proposed_non_goals":["Do not change billing UI"]}`, "--contract-id", "018f0000-0000-7000-8000-000000000009", "--fields-file", "-", "--format", "json")
	if err != nil {
		t.Fatalf("Run(contract update stdin) error = %v", err)
	}
	if !equalStrings(output.ChangedFields, []string{"proposed_non_goals"}) {
		t.Fatalf("changed_fields = %#v, want proposed_non_goals", output.ChangedFields)
	}
	if updateCount.Load() != 1 {
		t.Fatalf("update requests = %d, want 1", updateCount.Load())
	}
}

func TestRunUpdateRejectsMalformedAndEmptyFieldsBeforeHTTP(t *testing.T) {
	t.Parallel()
	requireGit(t)

	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "malformed", body: `{"proposed_scope":`, want: "parse contract update fields JSON"},
		{name: "empty", body: `{}`, want: "at least one proposed field"},
		{name: "context only", body: `{"context_refs":[{"kind":"local_repo_receipt","id":"receipt-1"}]}`, want: "at least one editable proposed field"},
		{name: "empty array", body: `{"proposed_scope":[]}`, want: "proposed_scope must include at least one value"},
		{name: "blank string", body: `{"title":" "}`, want: "title must not be blank"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoDir := setupGitRepo(t)
			fieldsPath := filepath.Join(repoDir, "fields.json")
			if err := os.WriteFile(fieldsPath, []byte(tt.body), 0o644); err != nil {
				t.Fatalf("write fields file: %v", err)
			}
			var requestCount atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				requestCount.Add(1)
				http.Error(w, "unexpected request", http.StatusInternalServerError)
			}))
			defer server.Close()
			writeProjectConfigFixture(t, repoDir, server.URL)

			_, err := runUpdateJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "", "--contract-id", "018f0000-0000-7000-8000-000000000009", "--fields-file", fieldsPath, "--format", "json")
			if err == nil {
				t.Fatalf("Run(contract update) error = nil, want %s", tt.want)
			}
			if got := exitcode.ForError(err); got != exitcode.Validation {
				t.Fatalf("exit code = %d, want validation", got)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want %q", err.Error(), tt.want)
			}
			if got := requestCount.Load(); got != 0 {
				t.Fatalf("server requests = %d, want 0", got)
			}
		})
	}
}

func TestRunUpdateMissingProjectConfigFailsBeforeHTTP(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	fieldsPath := filepath.Join(repoDir, "fields.json")
	if err := os.WriteFile(fieldsPath, []byte(`{"proposed_scope":["Scope"]}`), 0o644); err != nil {
		t.Fatalf("write fields file: %v", err)
	}
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestCount.Add(1)
		http.Error(w, "unexpected request", http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := runUpdateJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "", "--contract-id", "018f0000-0000-7000-8000-000000000009", "--fields-file", fieldsPath, "--format", "json")
	if err == nil {
		t.Fatal("Run(contract update) error = nil, want missing marker")
	}
	if got := exitcode.ForError(err); got != exitcode.Usage {
		t.Fatalf("exit code = %d, want usage", got)
	}
	if got := requestCount.Load(); got != 0 {
		t.Fatalf("server requests = %d, want 0 without marker", got)
	}
}

func TestRunUpdateMissingProjectBindingFailsBeforeHTTP(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	fieldsPath := filepath.Join(repoDir, "fields.json")
	if err := os.WriteFile(fieldsPath, []byte(`{"proposed_scope":["Scope"]}`), 0o644); err != nil {
		t.Fatalf("write fields file: %v", err)
	}
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestCount.Add(1)
		http.Error(w, "unexpected request", http.StatusInternalServerError)
	}))
	defer server.Close()
	writeProjectConfigFixtureWithIDs(t, repoDir, server.URL, "", "018f0000-0000-7000-8000-000000000004")

	_, err := runUpdateJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "", "--contract-id", "018f0000-0000-7000-8000-000000000009", "--fields-file", fieldsPath, "--format", "json")
	if err == nil {
		t.Fatal("Run(contract update) error = nil, want missing project binding")
	}
	if got := exitcode.ForError(err); got != exitcode.Validation {
		t.Fatalf("exit code = %d, want validation", got)
	}
	if !strings.Contains(err.Error(), "missing project_id") {
		t.Fatalf("error = %q, want missing project_id", err.Error())
	}
	if got := requestCount.Load(); got != 0 {
		t.Fatalf("server requests = %d, want 0 without complete marker binding", got)
	}
}

func TestRunUpdateMissingAuthFailsBeforeHTTP(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	fieldsPath := filepath.Join(repoDir, "fields.json")
	if err := os.WriteFile(fieldsPath, []byte(`{"proposed_scope":["Scope"]}`), 0o644); err != nil {
		t.Fatalf("write fields file: %v", err)
	}
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestCount.Add(1)
		http.Error(w, "unexpected request", http.StatusInternalServerError)
	}))
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, server.URL)

	_, err := runUpdateJSON(t, repoDir, fakeSessionStore{err: authstore.ErrSessionNotFound}, "", "--contract-id", "018f0000-0000-7000-8000-000000000009", "--fields-file", fieldsPath, "--format", "json")
	if err == nil {
		t.Fatal("Run(contract update) error = nil, want missing auth")
	}
	if got := exitcode.ForError(err); got != exitcode.Usage {
		t.Fatalf("exit code = %d, want usage", got)
	}
	if got := requestCount.Load(); got != 0 {
		t.Fatalf("server requests = %d, want 0 without auth", got)
	}
}

func TestRunUpdateOrganizationMismatchFailsBeforeMutation(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	fieldsPath := filepath.Join(repoDir, "fields.json")
	if err := os.WriteFile(fieldsPath, []byte(`{"proposed_scope":["Scope"]}`), 0o644); err != nil {
		t.Fatalf("write fields file: %v", err)
	}
	var meCount, updateCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/me":
			meCount.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"user":{"id":"018f0000-0000-7000-8000-000000000001","display_name":"Developer"},"organization_membership":{"organization_id":"018f0000-0000-7000-8000-000000009999","role":"member","state":"active"}}`))
		case "/v1/contracts/018f0000-0000-7000-8000-000000000009":
			updateCount.Add(1)
			http.Error(w, "unexpected mutation", http.StatusInternalServerError)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, server.URL)

	_, err := runUpdateJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "", "--contract-id", "018f0000-0000-7000-8000-000000000009", "--fields-file", fieldsPath, "--format", "json")
	if err == nil {
		t.Fatal("Run(contract update) error = nil, want organization mismatch")
	}
	if got := exitcode.ForError(err); got != exitcode.Validation {
		t.Fatalf("exit code = %d, want validation", got)
	}
	if meCount.Load() != 1 || updateCount.Load() != 0 {
		t.Fatalf("request counts me/update = %d/%d, want 1/0", meCount.Load(), updateCount.Load())
	}
}

func TestRunUpdateMalformedContractIDFailsBeforeHTTP(t *testing.T) {
	t.Parallel()

	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestCount.Add(1)
		http.Error(w, "unexpected request", http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := runUpdateJSON(t, t.TempDir(), fakeSessionStore{session: validSession(server.URL)}, `{"proposed_scope":["Scope"]}`, "--contract-id", "not-a-uuid", "--fields-file", "-", "--format", "json")
	if err == nil {
		t.Fatal("Run(contract update) error = nil, want malformed contract_id")
	}
	if got := exitcode.ForError(err); got != exitcode.Validation {
		t.Fatalf("exit code = %d, want validation", got)
	}
	if !strings.Contains(err.Error(), "contract_id must be a UUID") {
		t.Fatalf("error = %q, want malformed contract_id", err.Error())
	}
	if got := requestCount.Load(); got != 0 {
		t.Fatalf("server requests = %d, want 0", got)
	}
}

func TestRunUpdateTextIsHumanSafe(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	fieldsPath := filepath.Join(repoDir, "fields.json")
	if err := os.WriteFile(fieldsPath, []byte(`{"proposed_non_goals":["Do not change billing UI"]}`), 0o644); err != nil {
		t.Fatalf("write fields file: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/me":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"user":{"id":"018f0000-0000-7000-8000-000000000001","display_name":"Developer"},"organization_membership":{"organization_id":"018f0000-0000-7000-8000-000000000002","role":"member","state":"active"}}`))
		case "/v1/contracts/018f0000-0000-7000-8000-000000000009":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"018f0000-0000-7000-8000-000000000009","repo_binding_id":"018f0000-0000-7000-8000-000000000004","state":"draft"}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, server.URL)

	var stdout, stderr bytes.Buffer
	err := RunWithOptions(context.Background(), term.New(&stdout, &stderr), repoDir, []string{"update", "--contract-id", "018f0000-0000-7000-8000-000000000009", "--fields-file", fieldsPath}, Options{
		Store: fakeSessionStore{session: validSession(server.URL)},
		Now:   func() time.Time { return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("Run(contract update text) error = %v", err)
	}
	got := stdout.String()
	if !strings.Contains(got, "Next: review the draft contract with the user.") {
		t.Fatalf("stdout = %q, want review next action", got)
	}
	for _, forbidden := range []string{"approved", "workitem", "runner", "verified"} {
		if strings.Contains(strings.ToLower(got), forbidden) {
			t.Fatalf("stdout = %q, want no %q claim", got, forbidden)
		}
	}
}

func TestRunSubmitMovesContractToApprovalReview(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	var meCount, submitCount atomic.Int32
	var request contractTransitionRequest
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
		case "/v1/contracts/018f0000-0000-7000-8000-000000000009/submissions":
			submitCount.Add(1)
			if r.Method != http.MethodPost {
				t.Errorf("POST /v1/contracts/{id}/submissions method = %s", r.Method)
			}
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&request); err != nil {
				t.Errorf("decode contract submit request: %v", err)
				http.Error(w, "bad json", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"018f0000-0000-7000-8000-000000000009","repo_binding_id":"018f0000-0000-7000-8000-000000000004","state":"ready_for_approval"}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, server.URL)

	output, err := runSubmitJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--contract-id", "018f0000-0000-7000-8000-000000000009", "--format", "json")
	if err != nil {
		t.Fatalf("Run(contract submit) error = %v", err)
	}

	if output.SchemaVersion != "goalrail.cli.v1" {
		t.Fatalf("schema_version = %q, want goalrail.cli.v1", output.SchemaVersion)
	}
	if output.ContractID != "018f0000-0000-7000-8000-000000000009" || output.ContractState != "ready_for_approval" {
		t.Fatalf("contract/state = %q/%q, want ready_for_approval", output.ContractID, output.ContractState)
	}
	if output.NextAction.Kind != "approve_contract" || !output.NextAction.Available || !output.NextAction.Blocking {
		t.Fatalf("next_action = %#v, want available blocking approve_contract", output.NextAction)
	}
	wantCommand := "goalrail contract approve --contract-id 018f0000-0000-7000-8000-000000000009 --confirm-user-approval --format json"
	if output.NextAction.Command != wantCommand {
		t.Fatalf("next_action.command = %q, want %q", output.NextAction.Command, wantCommand)
	}
	if request.ProjectID != "018f0000-0000-7000-8000-000000000003" || request.RepoBindingID != "018f0000-0000-7000-8000-000000000004" {
		t.Fatalf("submit request project/repo = %q/%q, want local marker context", request.ProjectID, request.RepoBindingID)
	}
	if meCount.Load() != 1 || submitCount.Load() != 1 {
		t.Fatalf("request counts me/submit = %d/%d, want 1/1", meCount.Load(), submitCount.Load())
	}
}

func TestRunApproveRequiresExplicitConfirmationBeforeHTTP(t *testing.T) {
	t.Parallel()

	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestCount.Add(1)
		http.Error(w, "unexpected request", http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := runApproveJSON(t, t.TempDir(), fakeSessionStore{session: validSession(server.URL)}, "--contract-id", "018f0000-0000-7000-8000-000000000009", "--format", "json")
	if err == nil {
		t.Fatal("Run(contract approve) error = nil, want explicit confirmation error")
	}
	if got := exitcode.ForError(err); got != exitcode.Usage {
		t.Fatalf("exit code = %d, want usage", got)
	}
	if !strings.Contains(err.Error(), "--confirm-user-approval") {
		t.Fatalf("error = %q, want confirmation flag hint", err.Error())
	}
	if got := requestCount.Load(); got != 0 {
		t.Fatalf("server requests = %d, want 0 without confirmation", got)
	}
}

func TestRunApproveCreatesApprovedSnapshotAndExposesWorkPlan(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	var meCount, approveCount atomic.Int32
	var request contractTransitionRequest
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
		case "/v1/contracts/018f0000-0000-7000-8000-000000000009/approvals":
			approveCount.Add(1)
			if r.Method != http.MethodPost {
				t.Errorf("POST /v1/contracts/{id}/approvals method = %s", r.Method)
			}
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&request); err != nil {
				t.Errorf("decode contract approve request: %v", err)
				http.Error(w, "bad json", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"018f0000-0000-7000-8000-000000000009","repo_binding_id":"018f0000-0000-7000-8000-000000000004","state":"approved","approved_snapshot_id":"018f0000-0000-7000-8000-000000000010"}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, server.URL)

	output, err := runApproveJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--contract-id", "018f0000-0000-7000-8000-000000000009", "--confirm-user-approval", "--format", "json")
	if err != nil {
		t.Fatalf("Run(contract approve) error = %v", err)
	}

	if output.ContractID != "018f0000-0000-7000-8000-000000000009" || output.ContractState != "approved" {
		t.Fatalf("contract/state = %q/%q, want approved", output.ContractID, output.ContractState)
	}
	if output.NextAction.Kind != "plan_work" || !output.NextAction.Available || output.NextAction.PlannedSlice != "" {
		t.Fatalf("next_action = %#v, want available plan_work", output.NextAction)
	}
	wantCommand := "goalrail work plan --contract-id 018f0000-0000-7000-8000-000000000009 --format json"
	if output.NextAction.Command != wantCommand {
		t.Fatalf("next_action.command = %q, want %q", output.NextAction.Command, wantCommand)
	}
	if request.ProjectID != "018f0000-0000-7000-8000-000000000003" || request.RepoBindingID != "018f0000-0000-7000-8000-000000000004" {
		t.Fatalf("approve request project/repo = %q/%q, want local marker context", request.ProjectID, request.RepoBindingID)
	}
	if meCount.Load() != 1 || approveCount.Load() != 1 {
		t.Fatalf("request counts me/approve = %d/%d, want 1/1", meCount.Load(), approveCount.Load())
	}
}

func TestRunSubmitMissingProjectConfigFailsBeforeHTTP(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requestCount.Add(1)
		http.Error(w, "unexpected request", http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := runSubmitJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--contract-id", "018f0000-0000-7000-8000-000000000009", "--format", "json")
	if err == nil {
		t.Fatal("Run(contract submit) error = nil, want missing marker")
	}
	if got := exitcode.ForError(err); got != exitcode.Usage {
		t.Fatalf("exit code = %d, want usage", got)
	}
	if got := requestCount.Load(); got != 0 {
		t.Fatalf("server requests = %d, want 0 without marker", got)
	}
}

func TestRunSubmitOrganizationMismatchFailsBeforeMutation(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	var meCount, submitCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/me":
			meCount.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"user":{"id":"018f0000-0000-7000-8000-000000000001","display_name":"Developer"},"organization_membership":{"organization_id":"018f0000-0000-7000-8000-000000009999","role":"member","state":"active"}}`))
		case "/v1/contracts/018f0000-0000-7000-8000-000000000009/submissions":
			submitCount.Add(1)
			http.Error(w, "unexpected mutation", http.StatusInternalServerError)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, server.URL)

	_, err := runSubmitJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--contract-id", "018f0000-0000-7000-8000-000000000009", "--format", "json")
	if err == nil {
		t.Fatal("Run(contract submit) error = nil, want organization mismatch")
	}
	if got := exitcode.ForError(err); got != exitcode.Validation {
		t.Fatalf("exit code = %d, want validation", got)
	}
	if meCount.Load() != 1 || submitCount.Load() != 0 {
		t.Fatalf("request counts me/submit = %d/%d, want 1/0", meCount.Load(), submitCount.Load())
	}
}

func TestRunApproveTextIsHumanSafe(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := setupGitRepo(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/me":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"user":{"id":"018f0000-0000-7000-8000-000000000001","display_name":"Developer"},"organization_membership":{"organization_id":"018f0000-0000-7000-8000-000000000002","role":"member","state":"active"}}`))
		case "/v1/contracts/018f0000-0000-7000-8000-000000000009/approvals":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"018f0000-0000-7000-8000-000000000009","repo_binding_id":"018f0000-0000-7000-8000-000000000004","state":"approved"}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	writeProjectConfigFixture(t, repoDir, server.URL)

	var stdout, stderr bytes.Buffer
	err := RunWithOptions(context.Background(), term.New(&stdout, &stderr), repoDir, []string{"approve", "--contract-id", "018f0000-0000-7000-8000-000000000009", "--confirm-user-approval"}, Options{
		Store: fakeSessionStore{session: validSession(server.URL)},
		Now:   func() time.Time { return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("Run(contract approve text) error = %v", err)
	}
	got := stdout.String()
	if !strings.Contains(got, "Next: goalrail work plan") {
		t.Fatalf("stdout = %q, want available work plan next command", got)
	}
	for _, forbidden := range []string{"runner", "verified", "proof"} {
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

func runShowJSON(t *testing.T, repoDir string, store SessionStore, args ...string) (spine.ContractShowOutput, error) {
	t.Helper()

	var stdout, stderr bytes.Buffer
	err := RunWithOptions(context.Background(), term.New(&stdout, &stderr), repoDir, append([]string{"show"}, args...), Options{
		Store: store,
		Now:   func() time.Time { return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		return spine.ContractShowOutput{}, err
	}
	var output spine.ContractShowOutput
	decoder := json.NewDecoder(&stdout)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&output); err != nil {
		t.Fatalf("decode contract show JSON %q: %v", stdout.String(), err)
	}
	return output, nil
}

func runUpdateJSON(t *testing.T, repoDir string, store fakeSessionStore, stdin string, args ...string) (spine.ContractUpdateOutput, error) {
	t.Helper()

	var stdout, stderr bytes.Buffer
	options := Options{
		Store: store,
		Now:   func() time.Time { return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC) },
	}
	if stdin != "" {
		options.Stdin = strings.NewReader(stdin)
	}
	err := RunWithOptions(context.Background(), term.New(&stdout, &stderr), repoDir, append([]string{"update"}, args...), options)
	if err != nil {
		return spine.ContractUpdateOutput{}, err
	}
	var output spine.ContractUpdateOutput
	decoder := json.NewDecoder(&stdout)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&output); err != nil {
		t.Fatalf("decode contract update JSON %q: %v", stdout.String(), err)
	}
	return output, nil
}

func runSubmitJSON(t *testing.T, repoDir string, store fakeSessionStore, args ...string) (spine.ContractTransitionOutput, error) {
	t.Helper()

	var stdout, stderr bytes.Buffer
	err := RunWithOptions(context.Background(), term.New(&stdout, &stderr), repoDir, append([]string{"submit"}, args...), Options{
		Store: store,
		Now:   func() time.Time { return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		return spine.ContractTransitionOutput{}, err
	}
	var output spine.ContractTransitionOutput
	decoder := json.NewDecoder(&stdout)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&output); err != nil {
		t.Fatalf("decode contract submit JSON %q: %v", stdout.String(), err)
	}
	return output, nil
}

func runApproveJSON(t *testing.T, repoDir string, store fakeSessionStore, args ...string) (spine.ContractTransitionOutput, error) {
	t.Helper()

	var stdout, stderr bytes.Buffer
	err := RunWithOptions(context.Background(), term.New(&stdout, &stderr), repoDir, append([]string{"approve"}, args...), Options{
		Store: store,
		Now:   func() time.Time { return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		return spine.ContractTransitionOutput{}, err
	}
	var output spine.ContractTransitionOutput
	decoder := json.NewDecoder(&stdout)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&output); err != nil {
		t.Fatalf("decode contract approve JSON %q: %v", stdout.String(), err)
	}
	return output, nil
}

func decodeRawStringSlice(t *testing.T, raw json.RawMessage) []string {
	t.Helper()

	var values []string
	if err := json.Unmarshal(raw, &values); err != nil {
		t.Fatalf("decode raw string slice: %v", err)
	}
	return values
}

func equalStrings(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
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

type fakeSavingSessionStore struct {
	session   authstore.Session
	saved     authstore.Session
	saveCount atomic.Int32
}

func (s *fakeSavingSessionStore) Load() (authstore.Session, error) {
	return s.session, nil
}

func (s *fakeSavingSessionStore) Save(session authstore.Session) error {
	s.saved = session
	s.session = session
	s.saveCount.Add(1)
	return nil
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
