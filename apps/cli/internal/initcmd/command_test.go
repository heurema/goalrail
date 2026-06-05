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
	"github.com/heurema/goalrail/apps/cli/internal/projectscan"
	"github.com/heurema/goalrail/apps/cli/internal/spine"
	"github.com/heurema/goalrail/apps/cli/internal/term"
)

func TestRunExplicitRepoOutsideGitJSON(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	draft, err := runInitJSON(t, tempDir, "--local-demo", "--repo", "git@github.com:acme/payments.git", "--format", "json")
	if err != nil {
		t.Fatalf("Run(init --local-demo --repo) error = %v", err)
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

	draft, err := runInitJSON(t, repoDir, "--local-demo", "--format", "json")
	if err != nil {
		t.Fatalf("Run(init --local-demo) error = %v", err)
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

	draft, err := runInitJSON(t, repoDir, "--local-demo", "--format", "json")
	if err != nil {
		t.Fatalf("Run(init --local-demo) error = %v", err)
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

	draft, err := runInitJSON(t, repoDir, "--local-demo", "--format", "json")
	if err != nil {
		t.Fatalf("Run(init --local-demo) error = %v", err)
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

	draft, err := runInitJSON(t, repoDir, "--local-demo", "--format", "json")
	if err != nil {
		t.Fatalf("Run(init --local-demo) error = %v", err)
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

func TestRunRepositoryContextInitSendsExpectedRequestJSON(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir, headSHA := setupGitRepoWithOriginHead(t, "git@github.com:heurema/goalrail.git")
	gitignorePath := filepath.Join(repoDir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte("existing-ignore\n"), 0o644); err != nil {
		t.Fatalf("write .gitignore fixture: %v", err)
	}
	writeFile(t, repoDir, "go.mod", "module github.com/heurema/goalrail\n")
	writeFile(t, repoDir, "package.json", `{"name":"goalrail"}`)
	writeFile(t, repoDir, "pnpm-lock.yaml", "lockfileVersion: '9.0'\n")
	writeFile(t, repoDir, ".github/workflows/ci.yml", "name: ci\n")
	if err := os.MkdirAll(filepath.Join(repoDir, "apps", "cli"), 0o755); err != nil {
		t.Fatalf("create workspace fixture: %v", err)
	}
	var received spine.RepositoryContextInitRequest
	var snapshotReceived spine.RepositoryContextSnapshotRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
			http.Error(w, "bad method", http.StatusMethodNotAllowed)
			return
		}
		if r.Header.Get("Authorization") != "Bearer access-token" {
			t.Errorf("Authorization = %q, want bearer token", r.Header.Get("Authorization"))
			http.Error(w, "bad auth", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/v1/init/repository-context":
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&received); err != nil {
				t.Errorf("decode request body: %v", err)
				http.Error(w, "bad json", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(repositoryContextInitResponseJSON(true, true, "main")))
		case "/v1/repo-bindings/018f0000-0000-7000-8000-000000000004/context-snapshots":
			if _, err := os.Stat(filepath.Join(repoDir, projectConfigRelativePath)); err != nil {
				t.Errorf("local marker before snapshot stat error = %v, want marker written before advisory snapshot", err)
			}
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&snapshotReceived); err != nil {
				t.Errorf("decode snapshot request body: %v", err)
				http.Error(w, "bad json", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(repositoryContextSnapshotResponseJSON(true)))
		default:
			t.Errorf("path = %s, want repository context init or snapshot path", r.URL.Path)
			http.Error(w, "bad path", http.StatusNotFound)
		}
	}))
	defer server.Close()

	output, err := runRepositoryContextJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--format", "json")
	if err != nil {
		t.Fatalf("Run(init) error = %v", err)
	}

	if output.Mode != "server" || output.ServerURL != server.URL {
		t.Fatalf("output mode/server = %q/%q, want server/%q", output.Mode, output.ServerURL, server.URL)
	}
	if output.ProjectSlug != "github-heurema-goalrail" || output.ProjectDisplayName != "heurema/goalrail" {
		t.Fatalf("project output = %q/%q, want github-heurema-goalrail/heurema/goalrail", output.ProjectSlug, output.ProjectDisplayName)
	}
	if output.ProjectCreated != true || output.RepoBindingCreated != true {
		t.Fatalf("created flags = %v/%v, want true/true", output.ProjectCreated, output.RepoBindingCreated)
	}
	if output.RepoBindingID != "018f0000-0000-7000-8000-000000000004" {
		t.Fatalf("repo_binding_id = %q, want server id", output.RepoBindingID)
	}
	if output.LocalConfigPath != projectConfigRelativePath || output.LocalConfigStatus != localConfigStatusWritten {
		t.Fatalf("local config output = %q/%q, want %q/%q", output.LocalConfigPath, output.LocalConfigStatus, projectConfigRelativePath, localConfigStatusWritten)
	}
	if output.LocalIgnorePath != projectConfigIgnoreRelativePath || output.LocalIgnoreStatus != localConfigStatusWritten {
		t.Fatalf("local ignore output = %q/%q, want %q/%q", output.LocalIgnorePath, output.LocalIgnoreStatus, projectConfigIgnoreRelativePath, localConfigStatusWritten)
	}
	if !strings.Contains(output.LocalConfigMessage, "Commit .goalrail/project.yml and .goalrail/.gitignore") {
		t.Fatalf("local config message = %q, want commit hint", output.LocalConfigMessage)
	}
	if output.ContextSnapshotID != "018f0000-0000-7000-8000-000000000301" || output.ContextSnapshotStatus != "recorded" {
		t.Fatalf("context snapshot output = %q/%q, want recorded snapshot", output.ContextSnapshotID, output.ContextSnapshotStatus)
	}
	if output.ProjectScanStatus != "quick" || output.ProjectScanBaselineID == "" || output.ProjectScanOverlayID == "" || output.ProjectScanFreshness != "fresh" {
		t.Fatalf("project scan output = %#v, want quick/fresh with ids", output)
	}
	if output.Status != spine.InitOverallStatusSuccess {
		t.Fatalf("status = %q, want success", output.Status)
	}
	if output.NextCommand != nextSuggestedCommand {
		t.Fatalf("json next_suggested_command = %q, want existing generic command %q", output.NextCommand, nextSuggestedCommand)
	}
	assertInitSteps(t, output.Steps, []wantInitStep{
		{name: initStepRepositoryContext, status: spine.InitStepStatusOK},
		{name: initStepLocalMarker, status: spine.InitStepStatusOK},
		{name: initStepLocalGitignore, status: spine.InitStepStatusOK},
		{name: initStepContextSnapshot, status: spine.InitStepStatusOK},
		{name: initStepProjectScan, status: spine.InitStepStatusOK},
	})
	if received.Provider != "github" || received.RepositoryFullName != "heurema/goalrail" || received.RepositoryURL != "git@github.com:heurema/goalrail.git" {
		t.Fatalf("request repo fields = %#v, want GitHub goalrail repo", received)
	}
	if received.ProviderDefaultBranch != "main" || received.WorkflowBaseBranch != "main" {
		t.Fatalf("request branch fields = %#v, want main/main", received)
	}
	if received.LocalRemoteName != "origin" || received.LocalHeadSHA != headSHA {
		t.Fatalf("request local fields = %#v, want origin/%s", received, headSHA)
	}
	if received.SuggestedProjectSlug != "github-heurema-goalrail" || received.SuggestedProjectDisplayName != "heurema/goalrail" {
		t.Fatalf("request suggested project = %q/%q, want slug/display", received.SuggestedProjectSlug, received.SuggestedProjectDisplayName)
	}
	if snapshotReceived.Source != repositoryContextSnapshotSource || snapshotReceived.SchemaVersion != 1 {
		t.Fatalf("snapshot source/schema = %q/%d, want CLI init v1", snapshotReceived.Source, snapshotReceived.SchemaVersion)
	}
	if snapshotReceived.Repository.Provider != "github" || snapshotReceived.Repository.FullName != "heurema/goalrail" || snapshotReceived.Repository.URL != "git@github.com:heurema/goalrail.git" {
		t.Fatalf("snapshot repository = %#v, want output repository", snapshotReceived.Repository)
	}
	if snapshotReceived.Repository.RemoteName != "origin" || snapshotReceived.Repository.HeadSHA != headSHA {
		t.Fatalf("snapshot local Git fields = %#v, want origin/%s", snapshotReceived.Repository, headSHA)
	}
	for _, want := range []string{".github/workflows/ci.yml", "apps/cli/", "go.mod", "package.json", "pnpm-lock.yaml"} {
		if !stringSliceContains(snapshotReceived.DetectedPaths, want) {
			t.Fatalf("snapshot detected paths = %#v, want %q", snapshotReceived.DetectedPaths, want)
		}
	}
	if !stringSliceContains(snapshotReceived.DetectedToolchains, "go") || !stringSliceContains(snapshotReceived.DetectedToolchains, "node") {
		t.Fatalf("snapshot toolchains = %#v, want go and node", snapshotReceived.DetectedToolchains)
	}
	if !stringSliceContains(snapshotReceived.DetectedPackageManagers, "pnpm") {
		t.Fatalf("snapshot package managers = %#v, want pnpm", snapshotReceived.DetectedPackageManagers)
	}
	if !stringSliceContains(snapshotReceived.WorkspaceCandidates, "apps/cli") {
		t.Fatalf("snapshot workspace candidates = %#v, want apps/cli", snapshotReceived.WorkspaceCandidates)
	}
	config := readProjectConfigFile(t, repoDir)
	wantConfig := expectedProjectConfigYAML(server.URL)
	if config != wantConfig {
		t.Fatalf("project config =\n%s\nwant:\n%s", config, wantConfig)
	}
	if strings.Contains(config, "access-token") || strings.Contains(config, "refresh-token") {
		t.Fatalf("project config contains token material:\n%s", config)
	}
	assertGoalrailLocalIgnoreRules(t, repoDir)
	gitignore, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	if string(gitignore) != "existing-ignore\n" {
		t.Fatalf(".gitignore = %q, want unchanged", string(gitignore))
	}
}

func TestRunRepositoryContextInitTextNamesRepositorySnapshot(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir, _ := setupGitRepoWithOriginHead(t, "git@github.com:heurema/goalrail.git")
	writeFile(t, repoDir, "apps/cli/go.mod", "module github.com/heurema/goalrail/apps/cli\n")
	writeFile(t, repoDir, "apps/cli/main_test.go", "package main\n")
	writeFile(t, repoDir, "apps/web/package.json", `{"name":"goalrail-web","scripts":{"test":"vitest"}}`)
	writeFile(t, repoDir, "apps/web/package-lock.json", `{"lockfileVersion":3}`)
	writeFile(t, repoDir, ".github/workflows/ci.yml", "name: ci\n")
	writeFile(t, repoDir, "AGENTS.md", "# Agent rules\n")
	writeFile(t, repoDir, "CODEOWNERS", "* @goalrail/owners\n")
	runGit(t, repoDir, "add", ".")
	runGit(t, repoDir, "commit", "-m", "add project scan summary fixture")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/init/repository-context":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(repositoryContextInitResponseJSON(true, true, "main")))
		case "/v1/repo-bindings/018f0000-0000-7000-8000-000000000004/context-snapshots":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(repositoryContextSnapshotResponseJSON(true)))
		default:
			t.Errorf("path = %s, want repository context init or snapshot path", r.URL.Path)
			http.Error(w, "bad path", http.StatusNotFound)
		}
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	err := RunWithOptions(context.Background(), term.New(&stdout, &stderr), repoDir, nil, Options{
		Store:                fakeSessionStore{session: validSession(server.URL)},
		Now:                  func() time.Time { return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC) },
		ProjectScanCacheRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("Run(init) error = %v", err)
	}

	got := stdout.String()
	for _, want := range []string{
		"Repository context initialized",
		"Local config: .goalrail/project.yml (written)",
		"Local state ignore rules: .goalrail/.gitignore (written)",
		"Commit .goalrail/project.yml and .goalrail/.gitignore with this repository.",
		"Repository context snapshot: 018f0000-0000-7000-8000-000000000301 (recorded)",
		"Project scan:\n",
		"  baseline: created",
		"  overlay: created",
		"  toolchains: go, node",
		"  package managers: npm",
		"  workspaces: apps/cli, apps/web",
		"  tests: detected",
		"  ci: detected",
		"  agent rules: detected",
		"  codeowners: detected",
		"  partiality: none",
		"  freshness: current_head",
		"This initialized GoalRail repository context for your existing organization, wrote a non-secret GoalRail repository marker, attempted a metadata-only repository context snapshot, and ran a local Project Scan.",
		"No server clone, audit, hooks, branch creation, deploy keys, provider integration, runner, gate, proof, or verification were configured.",
		`Next: goalrail work start --title "Dogfood Goalrail on Goalrail"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("stdout = %q, want %q", got, want)
		}
	}
	if strings.Count(got, "\nNext: ") != 1 {
		t.Fatalf("stdout = %q, want exactly one Next command", got)
	}
	if strings.Contains(got, "Project context snapshot") {
		t.Fatalf("stdout = %q, want repository snapshot wording", got)
	}
}

func TestRunRepositoryContextInitBaseOverrideSplitsProviderDefaultAndWorkflowBase(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir, _ := setupGitRepoWithOriginHead(t, "git@github.com:heurema/goalrail.git")
	var received spine.RepositoryContextInitRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/repo-bindings/018f0000-0000-7000-8000-000000000004/context-snapshots" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(repositoryContextSnapshotResponseJSON(true)))
			return
		}
		if r.URL.Path != "/v1/init/repository-context" {
			t.Errorf("path = %s, want repository context init path", r.URL.Path)
			http.Error(w, "bad path", http.StatusNotFound)
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
		_, _ = w.Write([]byte(repositoryContextInitResponseJSON(true, true, "release/2026-05")))
	}))
	defer server.Close()

	output, err := runRepositoryContextJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--base", "release/2026-05", "--format", "json")
	if err != nil {
		t.Fatalf("Run(init --base) error = %v", err)
	}

	if received.ProviderDefaultBranch != "main" || received.WorkflowBaseBranch != "release/2026-05" {
		t.Fatalf("request branch fields = %#v, want main/release override", received)
	}
	if output.ProviderDefaultBranch != "main" || output.WorkflowBaseBranch != "release/2026-05" {
		t.Fatalf("output branch fields = %#v, want main/release override", output)
	}
	config := readProjectConfigFile(t, repoDir)
	if !strings.Contains(config, `workflow_base_branch: "release/2026-05"`) {
		t.Fatalf("project config =\n%s\nwant release workflow base", config)
	}
}

func TestRunRepositoryContextInitBaseOverrideWorksWithoutDetectedOriginDefault(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir := t.TempDir()
	runGit(t, repoDir, "init")
	runGit(t, repoDir, "-c", "user.name=Goalrail Test", "-c", "user.email=goalrail@example.test", "commit", "--allow-empty", "-m", "initial")
	runGit(t, repoDir, "remote", "add", "origin", "git@github.com:heurema/goalrail.git")
	var received spine.RepositoryContextInitRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/repo-bindings/018f0000-0000-7000-8000-000000000004/context-snapshots" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(repositoryContextSnapshotResponseJSON(true)))
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
		_, _ = w.Write([]byte(`{"organization_id":"018f0000-0000-7000-8000-000000000002","project_id":"018f0000-0000-7000-8000-000000000003","project_slug":"github-heurema-goalrail","project_display_name":"heurema/goalrail","project_created":true,"repo_binding_id":"018f0000-0000-7000-8000-000000000004","repo_binding_created":true,"provider":"github","repository_full_name":"heurema/goalrail","repository_url":"git@github.com:heurema/goalrail.git","provider_default_branch":"release/2026-05","workflow_base_branch":"release/2026-05","state":"active","message":"Repository context initialized."}`))
	}))
	defer server.Close()

	output, err := runRepositoryContextJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--base", "release/2026-05", "--format", "json")
	if err != nil {
		t.Fatalf("Run(init --base without origin default) error = %v", err)
	}

	if received.ProviderDefaultBranch != "" || received.WorkflowBaseBranch != "release/2026-05" {
		t.Fatalf("request branch fields = %#v, want empty provider default/release workflow", received)
	}
	if output.WorkflowBaseBranch != "release/2026-05" {
		t.Fatalf("workflow_base_branch = %q, want release override", output.WorkflowBaseBranch)
	}
}

func TestRunRepositoryContextInitMissingAuthReturnsHelpfulError(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir, _ := setupGitRepoWithOriginHead(t, "git@github.com:heurema/goalrail.git")
	_, err := runRepositoryContextJSON(t, repoDir, fakeSessionStore{err: authstore.ErrSessionNotFound}, "--format", "json")
	if err == nil {
		t.Fatal("Run(init) error = nil, want missing auth")
	}
	if got := exitcode.ForError(err); got != exitcode.Usage {
		t.Fatalf("exit code = %d, want usage", got)
	}
	if !strings.Contains(err.Error(), "not logged in; run goalrail login <server_url>") {
		t.Fatalf("error = %q, want login hint", err.Error())
	}
}

func TestRunRepositoryContextInitExpiredTokenFailsBeforeHTTP(t *testing.T) {
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
	_, err := runRepositoryContextJSON(t, repoDir, fakeSessionStore{session: session}, "--format", "json")
	if err == nil {
		t.Fatal("Run(init) error = nil, want expired login")
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

func TestRunRepositoryContextInitPreflightConflictSkipsServer(t *testing.T) {
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
			name: "repository_full_name",
			mutate: func(config projectConfig) projectConfig {
				config.Repository.FullName = "heurema/other"
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
			var requestCount atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				requestCount.Add(1)
				http.Error(w, "unexpected request", http.StatusInternalServerError)
			}))
			defer server.Close()

			original := writeProjectConfigFixture(t, repoDir, tt.mutate(projectConfigFixture(server.URL)))
			_, err := runRepositoryContextJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--format", "json")
			if err == nil {
				t.Fatal("Run(init) error = nil, want preflight conflict")
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

func TestRunRepositoryContextInitDoesNotPreflightProjectIDBeforeServer(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir, _ := setupGitRepoWithOriginHead(t, "git@github.com:heurema/goalrail.git")
	existing := projectConfigFixture("placeholder")
	existing.ProjectID = "018f0000-0000-7000-8000-000000000099"
	existing.RepoBindingID = "018f0000-0000-7000-8000-000000000098"
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		if strings.HasPrefix(r.URL.Path, "/v1/repo-bindings/") && strings.HasSuffix(r.URL.Path, "/context-snapshots") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(repositoryContextSnapshotResponseJSON(false)))
			return
		}
		if r.URL.Path != "/v1/init/repository-context" {
			t.Errorf("path = %s, want repository context init path", r.URL.Path)
			http.Error(w, "bad path", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"organization_id":"018f0000-0000-7000-8000-000000000002","project_id":"018f0000-0000-7000-8000-000000000099","project_slug":"github-heurema-goalrail","project_display_name":"heurema/goalrail","project_created":false,"repo_binding_id":"018f0000-0000-7000-8000-000000000098","repo_binding_created":false,"provider":"github","repository_full_name":"heurema/goalrail","repository_url":"git@github.com:heurema/goalrail.git","provider_default_branch":"main","workflow_base_branch":"main","state":"active","message":"Repository context already initialized."}`))
	}))
	defer server.Close()
	existing.ServerURL = server.URL
	writeProjectConfigFixture(t, repoDir, existing)

	output, err := runRepositoryContextJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--format", "json")
	if err != nil {
		t.Fatalf("Run(init) error = %v", err)
	}
	if got := requestCount.Load(); got != 2 {
		t.Fatalf("server requests = %d, want init plus snapshot because project_id is post-server for plain init", got)
	}
	if output.LocalConfigStatus != localConfigStatusUnchanged {
		t.Fatalf("local_config_status = %q, want unchanged", output.LocalConfigStatus)
	}
	if output.LocalIgnoreStatus != localConfigStatusWritten {
		t.Fatalf("local_ignore_status = %q, want written when missing", output.LocalIgnoreStatus)
	}
	if !strings.Contains(output.LocalConfigMessage, "Existing Goalrail project marker found and verified") {
		t.Fatalf("local_config_message = %q, want verified marker", output.LocalConfigMessage)
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
	if output.LocalIgnorePath != projectConfigIgnoreRelativePath || output.LocalIgnoreStatus != localConfigStatusWritten {
		t.Fatalf("local ignore output = %q/%q, want %q/%q", output.LocalIgnorePath, output.LocalIgnoreStatus, projectConfigIgnoreRelativePath, localConfigStatusWritten)
	}
	if output.Status != spine.InitOverallStatusSuccess {
		t.Fatalf("status = %q, want success", output.Status)
	}
	assertInitSteps(t, output.Steps, []wantInitStep{
		{name: initStepRepoBinding, status: spine.InitStepStatusOK},
		{name: initStepLocalMarker, status: spine.InitStepStatusOK},
		{name: initStepLocalGitignore, status: spine.InitStepStatusOK},
		{name: initStepProjectScan, status: spine.InitStepStatusOK},
	})
	assertNoInitStep(t, output.Steps, initStepContextSnapshot)
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
	assertGoalrailLocalIgnoreRules(t, repoDir)
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
	if _, err := os.Stat(filepath.Join(repoDir, projectConfigIgnoreRelativePath)); err != nil {
		t.Fatalf("git root local-state gitignore not written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(nestedDir, projectConfigRelativePath)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("nested project config stat error = %v, want not exist", err)
	}
	if _, err := os.Stat(filepath.Join(nestedDir, projectConfigIgnoreRelativePath)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("nested local-state gitignore stat error = %v, want not exist", err)
	}
}

func TestRunServerBackedInitProjectScanFailureReturnsWarningStatus(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir, _ := setupGitRepoWithOriginHead(t, "git@github.com:heurema/goalrail.git")
	projectID := "018f0000-0000-7000-8000-000000000003"
	server := httptest.NewServer(repoBindingInitHandler(t, projectID))
	defer server.Close()
	cacheRootFile := filepath.Join(t.TempDir(), "project-scan-cache-root")
	if err := os.WriteFile(cacheRootFile, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("write cache root file: %v", err)
	}

	var stdout, stderr bytes.Buffer
	err := RunWithOptions(context.Background(), term.New(&stdout, &stderr), repoDir, []string{"--project", projectID, "--format", "json"}, Options{
		Store:                fakeSessionStore{session: validSession(server.URL)},
		Now:                  func() time.Time { return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC) },
		ProjectScanCacheRoot: cacheRootFile,
	})
	if err != nil {
		t.Fatalf("Run(init --project) error = %v", err)
	}

	var output spine.RepoBindingInitOutput
	decoder := json.NewDecoder(&stdout)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&output); err != nil {
		t.Fatalf("decode init server JSON %q: %v", stdout.String(), err)
	}
	if output.Status != spine.InitOverallStatusSuccessWithWarnings {
		t.Fatalf("status = %q, want success_with_warnings", output.Status)
	}
	if output.ProjectScanStatus != "error" || output.ProjectScanWarning == "" {
		t.Fatalf("project scan output = %#v, want non-fatal scan error warning", output)
	}
	step := findInitStep(output.Steps, initStepProjectScan)
	if step == nil || step.Status != spine.InitStepStatusWarning || !step.Recoverable || !strings.Contains(step.RetryCommand, "goalrail project scan") {
		t.Fatalf("project_scan step = %#v, want recoverable warning with project scan retry", step)
	}
	if _, err := os.Stat(filepath.Join(repoDir, projectConfigRelativePath)); err != nil {
		t.Fatalf("local marker stat error = %v, want marker written before scan warning", err)
	}
}

func TestRunServerBackedInitProjectScanFailureTextShowsUnavailableSummary(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir, _ := setupGitRepoWithOriginHead(t, "git@github.com:heurema/goalrail.git")
	projectID := "018f0000-0000-7000-8000-000000000003"
	server := httptest.NewServer(repoBindingInitHandler(t, projectID))
	defer server.Close()
	cacheRootFile := filepath.Join(t.TempDir(), "project-scan-cache-root")
	if err := os.WriteFile(cacheRootFile, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("write cache root file: %v", err)
	}

	var stdout, stderr bytes.Buffer
	err := RunWithOptions(context.Background(), term.New(&stdout, &stderr), repoDir, []string{"--project", projectID}, Options{
		Store:                fakeSessionStore{session: validSession(server.URL)},
		Now:                  func() time.Time { return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC) },
		ProjectScanCacheRoot: cacheRootFile,
	})
	if err != nil {
		t.Fatalf("Run(init --project) error = %v", err)
	}

	got := stdout.String()
	for _, want := range []string{
		"Init status: success_with_warnings",
		"Project scan:\n",
		"  baseline: unavailable",
		"  overlay: unavailable",
		"  tests: unknown",
		"  partiality: not_checked",
		"  freshness: unknown",
		"  warnings:",
		"Warning: Local Project Scan cache could not be written:",
		`Next: goalrail work start --title "Dogfood Goalrail on Goalrail"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("stdout = %q, want %q", got, want)
		}
	}
	if strings.Count(got, "\nNext: ") != 1 {
		t.Fatalf("stdout = %q, want exactly one Next command", got)
	}
	if strings.Contains(got, "goalrail project scan --format json") {
		t.Fatalf("stdout = %q, want Project Scan warning without extra command recommendation", got)
	}
	if _, err := os.Stat(filepath.Join(repoDir, projectConfigRelativePath)); err != nil {
		t.Fatalf("local marker stat error = %v, want marker written before scan warning", err)
	}
}

func TestRunRepositoryContextInitSnapshotFailureAfterMarkerReturnsWarningStatus(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir, _ := setupGitRepoWithOriginHead(t, "git@github.com:heurema/goalrail.git")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/init/repository-context":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(repositoryContextInitResponseJSON(true, true, "release/2026-05")))
		case "/v1/repo-bindings/018f0000-0000-7000-8000-000000000004/context-snapshots":
			if _, err := os.Stat(filepath.Join(repoDir, projectConfigRelativePath)); err != nil {
				t.Errorf("local marker before failed snapshot stat error = %v, want marker written first", err)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":{"code":"temporary_snapshot_error","message":"snapshot unavailable"}}`))
		default:
			t.Errorf("path = %s, want repository context init or snapshot path", r.URL.Path)
			http.Error(w, "bad path", http.StatusNotFound)
		}
	}))
	defer server.Close()

	output, err := runRepositoryContextJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--repo", "git@github.com:heurema/goalrail.git", "--base", "release/2026-05", "--format", "json")
	if err != nil {
		t.Fatalf("Run(init) error = %v", err)
	}
	if output.Status != spine.InitOverallStatusSuccessWithWarnings {
		t.Fatalf("status = %q, want success_with_warnings", output.Status)
	}
	if output.ContextSnapshotStatus != string(spine.InitStepStatusWarning) || output.ContextSnapshotID != "" {
		t.Fatalf("context snapshot output = %#v, want warning without snapshot id", output)
	}
	step := findInitStep(output.Steps, initStepContextSnapshot)
	wantRetry := "goalrail init --repo git@github.com:heurema/goalrail.git --base release/2026-05"
	if step == nil || step.Status != spine.InitStepStatusWarning || !step.Recoverable || step.RetryCommand != wantRetry {
		t.Fatalf("context_snapshot step = %#v, want recoverable warning with init retry", step)
	}
	if output.ProjectScanStatus != "quick" {
		t.Fatalf("project_scan_status = %q, want scan still attempted after snapshot warning", output.ProjectScanStatus)
	}
	if _, err := os.Stat(filepath.Join(repoDir, projectConfigRelativePath)); err != nil {
		t.Fatalf("local marker stat error = %v, want marker present", err)
	}
}

func TestRunRepositoryContextInitMarkerWriteFailureEmitsPartialFailedOutput(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir, _ := setupGitRepoWithOriginHead(t, "git@github.com:heurema/goalrail.git")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/init/repository-context" {
			t.Errorf("path = %s, want repository context init path only", r.URL.Path)
			http.Error(w, "bad path", http.StatusNotFound)
			return
		}
		if err := os.WriteFile(filepath.Join(repoDir, ".goalrail"), []byte("blocks marker directory"), 0o644); err != nil {
			t.Errorf("write blocking .goalrail file: %v", err)
			http.Error(w, "fixture failed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(repositoryContextInitResponseJSON(true, true, "release/2026-05")))
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	err := RunWithOptions(context.Background(), term.New(&stdout, &stderr), repoDir, []string{"--repo", "git@github.com:heurema/goalrail.git", "--base", "release/2026-05", "--format", "json"}, Options{
		Store:                fakeSessionStore{session: validSession(server.URL)},
		Now:                  func() time.Time { return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC) },
		ProjectScanCacheRoot: t.TempDir(),
	})
	if err == nil {
		t.Fatal("Run(init) error = nil, want marker write failure")
	}
	if got := exitcode.ForError(err); got != exitcode.Runtime {
		t.Fatalf("exit code = %d, want runtime", got)
	}

	var output spine.RepositoryContextInitOutput
	decoder := json.NewDecoder(&stdout)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&output); err != nil {
		t.Fatalf("decode partial init JSON %q: %v", stdout.String(), err)
	}
	if output.Status != spine.InitOverallStatusPartialFailed {
		t.Fatalf("status = %q, want partial_failed", output.Status)
	}
	assertInitSteps(t, output.Steps, []wantInitStep{
		{name: initStepRepositoryContext, status: spine.InitStepStatusOK},
		{name: initStepLocalMarker, status: spine.InitStepStatusError},
		{name: initStepLocalGitignore, status: spine.InitStepStatusSkipped},
		{name: initStepContextSnapshot, status: spine.InitStepStatusSkipped},
		{name: initStepProjectScan, status: spine.InitStepStatusSkipped},
	})
	wantRetry := "goalrail init --repo git@github.com:heurema/goalrail.git --base release/2026-05"
	if output.NextCommand != wantRetry {
		t.Fatalf("next_suggested_command = %q, want retry init", output.NextCommand)
	}
	step := findInitStep(output.Steps, initStepLocalMarker)
	if step == nil || step.RetryCommand != wantRetry {
		t.Fatalf("local_marker step = %#v, want retry command %q", step, wantRetry)
	}
}

func TestRunServerBackedInitLocalRecoveryRetryCommandsPreserveProjectRepoAndBase(t *testing.T) {
	t.Parallel()
	requireGit(t)

	projectID := "018f0000-0000-7000-8000-000000000003"
	wantRetry := "goalrail init --project " + projectID + " --repo git@github.com:heurema/goalrail.git --base release/2026-05"
	tests := []struct {
		name             string
		prepare          func(string)
		blockAfterServer bool
		wantStep         string
	}{
		{
			name:             "marker_failure",
			blockAfterServer: true,
			wantStep:         initStepLocalMarker,
		},
		{
			name: "gitignore_failure",
			prepare: func(repoDir string) {
				if err := os.MkdirAll(filepath.Join(repoDir, ".goalrail", ".gitignore"), 0o755); err != nil {
					t.Fatalf("create blocking .goalrail/.gitignore directory: %v", err)
				}
			},
			wantStep: initStepLocalGitignore,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repoDir, _ := setupGitRepoWithOriginHead(t, "git@github.com:heurema/goalrail.git")
			if tt.prepare != nil {
				tt.prepare(repoDir)
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.blockAfterServer {
					if err := os.WriteFile(filepath.Join(repoDir, ".goalrail"), []byte("blocks marker directory"), 0o644); err != nil {
						t.Errorf("write blocking .goalrail file: %v", err)
						http.Error(w, "fixture failed", http.StatusInternalServerError)
						return
					}
				}
				repoBindingInitHandler(t, projectID)(w, r)
			}))
			defer server.Close()

			var stdout, stderr bytes.Buffer
			err := RunWithOptions(context.Background(), term.New(&stdout, &stderr), repoDir, []string{"--project", projectID, "--repo", "git@github.com:heurema/goalrail.git", "--base", "release/2026-05", "--format", "json"}, Options{
				Store:                fakeSessionStore{session: validSession(server.URL)},
				Now:                  func() time.Time { return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC) },
				ProjectScanCacheRoot: t.TempDir(),
			})
			if err == nil {
				t.Fatal("Run(init --project) error = nil, want local recovery failure")
			}
			if got := exitcode.ForError(err); got != exitcode.Runtime {
				t.Fatalf("exit code = %d, want runtime", got)
			}

			var output spine.RepoBindingInitOutput
			decoder := json.NewDecoder(&stdout)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&output); err != nil {
				t.Fatalf("decode partial init JSON %q: %v", stdout.String(), err)
			}
			if output.Status != spine.InitOverallStatusPartialFailed {
				t.Fatalf("status = %q, want partial_failed", output.Status)
			}
			if output.NextCommand != wantRetry {
				t.Fatalf("next_suggested_command = %q, want %q", output.NextCommand, wantRetry)
			}
			step := findInitStep(output.Steps, tt.wantStep)
			if step == nil || step.Status != spine.InitStepStatusError || !step.Recoverable || step.RetryCommand != wantRetry {
				t.Fatalf("%s step = %#v, want recoverable error with retry %q", tt.wantStep, step, wantRetry)
			}
		})
	}
}

func TestInitRetryCommandsPreserveContextAndQuoteUnsafeValues(t *testing.T) {
	t.Parallel()

	if got := repositoryContextInitRetryCommand(spine.RepoBindingDraft{
		RepoURL:            "https://github.com/heurema/goalrail.git",
		WorkflowBaseBranch: "release/2026-05",
	}); got != "goalrail init --repo https://github.com/heurema/goalrail.git --base release/2026-05" {
		t.Fatalf("repository retry command = %q, want ordinary HTTPS args unquoted", got)
	}
	if got := repoBindingInitRetryCommand("018f0000-0000-7000-8000-000000000003", spine.RepoBindingDraft{
		RepoURL:            "git@github.com:heurema/goalrail.git",
		WorkflowBaseBranch: "main",
	}); got != "goalrail init --project 018f0000-0000-7000-8000-000000000003 --repo git@github.com:heurema/goalrail.git --base main" {
		t.Fatalf("repo binding retry command = %q, want ordinary SSH args unquoted", got)
	}
	if got := repositoryContextInitRetryCommand(spine.RepoBindingDraft{
		RepoURL:            "https://example.com/acme/repo with space.git",
		WorkflowBaseBranch: "release/$USER's-branch",
	}); got != "goalrail init --repo 'https://example.com/acme/repo with space.git' --base 'release/$USER'\\''s-branch'" {
		t.Fatalf("quoted repository retry command = %q, want shell-quoted unsafe args", got)
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
	if output.LocalIgnoreStatus != localConfigStatusUnchanged {
		t.Fatalf("local_ignore_status = %q, want unchanged", output.LocalIgnoreStatus)
	}
	if !strings.Contains(output.LocalConfigMessage, "Existing Goalrail project marker found and verified") {
		t.Fatalf("local_config_message = %q, want verified marker", output.LocalConfigMessage)
	}
	assertGoalrailIgnoreBehavior(t, repoDir)
}

func TestRunServerBackedInitUpdatesExistingGoalrailGitignore(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir, _ := setupGitRepoWithOriginHead(t, "git@github.com:heurema/goalrail.git")
	rootGitignorePath := filepath.Join(repoDir, ".gitignore")
	if err := os.WriteFile(rootGitignorePath, []byte("existing-ignore\n"), 0o644); err != nil {
		t.Fatalf("write .gitignore fixture: %v", err)
	}
	writeGoalrailGitignoreFile(t, repoDir, "# existing local state rules\n/local/\n")
	server := httptest.NewServer(repoBindingInitHandler(t, "018f0000-0000-7000-8000-000000000003"))
	defer server.Close()

	output, err := runInitServerJSON(t, repoDir, fakeSessionStore{session: validSession(server.URL)}, "--project", "018f0000-0000-7000-8000-000000000003", "--format", "json")
	if err != nil {
		t.Fatalf("Run(init --project) error = %v", err)
	}

	if output.LocalIgnoreStatus != localConfigStatusUpdated {
		t.Fatalf("local_ignore_status = %q, want updated", output.LocalIgnoreStatus)
	}
	got := readGoalrailGitignoreFile(t, repoDir)
	for _, want := range strings.Split(strings.TrimSpace(renderProjectConfigGitignore()), "\n") {
		if !strings.Contains(got, want+"\n") {
			t.Fatalf(".goalrail/.gitignore = %q, want rule %q", got, want)
		}
	}
	if got := readFile(t, rootGitignorePath); got != "existing-ignore\n" {
		t.Fatalf(".gitignore = %q, want unchanged", got)
	}
	assertGoalrailIgnoreBehavior(t, repoDir)
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
	if output.LocalIgnoreStatus != localConfigStatusWritten {
		t.Fatalf("local_ignore_status = %q, want written when missing", output.LocalIgnoreStatus)
	}
	if got := requestCount.Load(); got != 1 {
		t.Fatalf("server requests = %d, want 1 for matching config", got)
	}
}

func TestRunLocalDemoDoesNotWriteProjectConfig(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir, _ := setupGitRepoWithOriginHead(t, "git@github.com:heurema/goalrail.git")
	if _, err := runInitJSON(t, repoDir, "--local-demo", "--repo", "git@github.com:acme/payments.git", "--format", "json"); err != nil {
		t.Fatalf("Run(init --local-demo --repo) error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(repoDir, projectConfigRelativePath)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("project config stat error = %v, want not exist", err)
	}
	if _, err := os.Stat(filepath.Join(repoDir, projectConfigIgnoreRelativePath)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("local-state gitignore stat error = %v, want not exist", err)
	}
}

func TestRunLocalDemoTextDoesNotPretendProjectScanRan(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir, _ := setupGitRepoWithOriginHead(t, "git@github.com:heurema/goalrail.git")
	var stdout, stderr bytes.Buffer
	err := Run(context.Background(), term.New(&stdout, &stderr), repoDir, []string{"--local-demo"})
	if err != nil {
		t.Fatalf("Run(init --local-demo) error = %v", err)
	}
	got := stdout.String()
	if strings.Contains(got, "Project scan") {
		t.Fatalf("stdout = %q, want no Project Scan claim in local-demo mode", got)
	}
	if !strings.Contains(got, localDemoMessage) {
		t.Fatalf("stdout = %q, want local-demo boundary message", got)
	}
}

func TestProjectScanSummaryFormattingShowsPartialWarnings(t *testing.T) {
	t.Parallel()

	baseline := projectscan.RepositoryBaselineProfile{
		Status: projectscan.BaselineStatusPartial,
		Shape: projectscan.RepositoryShape{
			Toolchains:      []string{"node", "go"},
			PackageManagers: []string{"npm"},
			Workspaces:      []string{"apps/web", "apps/cli"},
		},
		ReadinessSignals: projectscan.ReadinessSignals{
			Tests:      []string{"apps/web/package.json"},
			AgentRules: []string{"AGENTS.md"},
		},
		Partiality: projectscan.Partiality{
			ShallowRepository: true,
			Truncated:         true,
			Reasons:           []string{"scan_budget_truncated"},
		},
	}
	overlay := projectscan.WorkspaceOverlay{State: projectscan.OverlayStateClean}
	freshness := projectscan.FreshnessResult{Status: projectscan.FreshnessPartial}

	var b strings.Builder
	writeProjectScanSummaryText(&b, summarizeInitProjectScan(projectScanArtifactRefreshed, projectScanArtifactRefreshed, baseline, overlay, freshness))
	got := b.String()
	for _, want := range []string{
		"  baseline: refreshed",
		"  overlay: refreshed",
		"  toolchains: go, node",
		"  package managers: npm",
		"  workspaces: apps/cli, apps/web",
		"  tests: detected",
		"  ci: not_detected",
		"  agent rules: detected",
		"  codeowners: not_detected",
		"  partiality: shallow_repo, truncated",
		"  freshness: current_head",
		"  warnings: partial scan: shallow_repo, truncated",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("summary = %q, want %q", got, want)
		}
	}
}

func TestRunLocalDemoIgnoresExistingProjectConfig(t *testing.T) {
	t.Parallel()
	requireGit(t)

	repoDir, _ := setupGitRepoWithOriginHead(t, "git@github.com:heurema/goalrail.git")
	original := writeRawProjectConfigFile(t, repoDir, "not a GoalRail project marker\n")
	if _, err := runInitJSON(t, repoDir, "--local-demo", "--repo", "git@github.com:acme/payments.git", "--format", "json"); err != nil {
		t.Fatalf("Run(init --local-demo --repo) error = %v", err)
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
	if got := stdout.String(); !strings.Contains(got, "Usage: goalrail init [--repo <repo-url>] [--base <branch>] [--project <project-id>] [--local-demo] [--format text|json]") {
		t.Fatalf("stdout = %q, want init usage", got)
	}
	if got := stdout.String(); !strings.Contains(got, "records a metadata-only repository context snapshot") || strings.Contains(got, "project context snapshot") {
		t.Fatalf("stdout = %q, want repository snapshot wording", got)
	}
}

type wantInitStep struct {
	name   string
	status spine.InitStepStatus
}

func assertInitSteps(t *testing.T, steps []spine.InitStepResult, wants []wantInitStep) {
	t.Helper()

	if len(steps) != len(wants) {
		t.Fatalf("steps = %#v, want %d steps", steps, len(wants))
	}
	for i, want := range wants {
		if steps[i].Name != want.name || steps[i].Status != want.status {
			t.Fatalf("steps[%d] = %#v, want %s/%s", i, steps[i], want.name, want.status)
		}
	}
}

func assertNoInitStep(t *testing.T, steps []spine.InitStepResult, name string) {
	t.Helper()

	if step := findInitStep(steps, name); step != nil {
		t.Fatalf("step %q = %#v, want absent", name, step)
	}
}

func findInitStep(steps []spine.InitStepResult, name string) *spine.InitStepResult {
	for i := range steps {
		if steps[i].Name == name {
			return &steps[i]
		}
	}
	return nil
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
		Store:                store,
		Now:                  func() time.Time { return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC) },
		ProjectScanCacheRoot: t.TempDir(),
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

func runRepositoryContextJSON(t *testing.T, workDir string, store fakeSessionStore, args ...string) (spine.RepositoryContextInitOutput, error) {
	t.Helper()

	var stdout, stderr bytes.Buffer
	err := RunWithOptions(context.Background(), term.New(&stdout, &stderr), workDir, args, Options{
		Store:                store,
		Now:                  func() time.Time { return time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC) },
		ProjectScanCacheRoot: t.TempDir(),
	})
	if err != nil {
		return spine.RepositoryContextInitOutput{}, err
	}

	var output spine.RepositoryContextInitOutput
	decoder := json.NewDecoder(&stdout)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&output); err != nil {
		t.Fatalf("decode repository context JSON %q: %v", stdout.String(), err)
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

func repositoryContextInitResponseJSON(projectCreated bool, repoBindingCreated bool, workflowBaseBranch string) string {
	providerDefaultBranch := "main"
	if workflowBaseBranch != "main" {
		providerDefaultBranch = "main"
	}
	return `{"organization_id":"018f0000-0000-7000-8000-000000000002","project_id":"018f0000-0000-7000-8000-000000000003","project_slug":"github-heurema-goalrail","project_display_name":"heurema/goalrail","project_created":` +
		boolJSON(projectCreated) +
		`,"repo_binding_id":"018f0000-0000-7000-8000-000000000004","repo_binding_created":` +
		boolJSON(repoBindingCreated) +
		`,"provider":"github","repository_full_name":"heurema/goalrail","repository_url":"git@github.com:heurema/goalrail.git","provider_default_branch":"` +
		providerDefaultBranch +
		`","workflow_base_branch":"` +
		workflowBaseBranch +
		`","state":"active","message":"Repository context initialized."}`
}

func repositoryContextSnapshotResponseJSON(created bool) string {
	return `{"context_snapshot_id":"018f0000-0000-7000-8000-000000000301","organization_id":"018f0000-0000-7000-8000-000000000002","project_id":"018f0000-0000-7000-8000-000000000003","repo_binding_id":"018f0000-0000-7000-8000-000000000004","source":"goalrail_cli_init","schema_version":1,"fingerprint":"sha256:abc123","created":` +
		boolJSON(created) +
		`,"message":"Repository context snapshot recorded."}`
}

func boolJSON(value bool) string {
	if value {
		return "true"
	}
	return "false"
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

func writeFile(t *testing.T, repoDir string, relative string, content string) {
	t.Helper()

	path := filepath.Join(repoDir, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create parent for %s: %v", relative, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", relative, err)
	}
}

func stringSliceContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
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

func writeGoalrailGitignoreFile(t *testing.T, repoDir string, content string) string {
	t.Helper()

	path := filepath.Join(repoDir, projectConfigIgnoreRelativePath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create .goalrail dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write .goalrail/.gitignore: %v", err)
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

func readGoalrailGitignoreFile(t *testing.T, repoDir string) string {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(repoDir, projectConfigIgnoreRelativePath))
	if err != nil {
		t.Fatalf("read .goalrail/.gitignore: %v", err)
	}
	return string(raw)
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(raw)
}

func assertGoalrailLocalIgnoreRules(t *testing.T, repoDir string) {
	t.Helper()

	if got, want := readGoalrailGitignoreFile(t, repoDir), renderProjectConfigGitignore(); got != want {
		t.Fatalf(".goalrail/.gitignore =\n%s\nwant:\n%s", got, want)
	}
	assertGoalrailIgnoreBehavior(t, repoDir)
}

func assertGoalrailIgnoreBehavior(t *testing.T, repoDir string) {
	t.Helper()

	assertGitIgnoreState(t, repoDir, ".goalrail/project.yml", false)
	assertGitIgnoreState(t, repoDir, ".goalrail/local/current.json", true)
	assertGitIgnoreState(t, repoDir, ".goalrail/cache/current.json", true)
	assertGitIgnoreState(t, repoDir, ".goalrail/state/current.json", true)
	assertGitIgnoreState(t, repoDir, ".goalrail/tmp/current.json", true)
	assertGitIgnoreState(t, repoDir, ".goalrail/project.local.yml", true)
	assertGitIgnoreState(t, repoDir, ".goalrail/project.local.toml", true)
	assertGitIgnoreState(t, repoDir, ".goalrail/project.local.json", true)
}

func assertGitIgnoreState(t *testing.T, repoDir string, relativePath string, wantIgnored bool) {
	t.Helper()

	cmd := exec.Command("git", "-C", filepath.Clean(repoDir), "check-ignore", "--quiet", "--", filepath.ToSlash(relativePath))
	err := cmd.Run()
	gotIgnored := err == nil
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
			t.Fatalf("git check-ignore %s failed: %v", relativePath, err)
		}
	}
	if gotIgnored != wantIgnored {
		t.Fatalf("git ignore state for %s = %v, want %v", relativePath, gotIgnored, wantIgnored)
	}
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
