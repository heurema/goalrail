package executionrunner

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestStepExitsCleanlyWhenNoExecutionWork(t *testing.T) {
	var leaseRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/execution-jobs/leases" {
			leaseRequests.Add(1)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	var logs bytes.Buffer
	runner := newTestRunner(t, server.URL, &logs)
	result, err := runner.Step(context.Background())
	if err != nil {
		t.Fatalf("Step() error = %v", err)
	}
	if result != StepNoWork {
		t.Fatalf("Step() = %q, want %q", result, StepNoWork)
	}
	if leaseRequests.Load() != 1 {
		t.Fatalf("lease requests = %d, want 1", leaseRequests.Load())
	}
	if !strings.Contains(logs.String(), "no execution work available") {
		t.Fatalf("logs = %q, want no-work message", logs.String())
	}
}

func TestStepAcquiresExecutionLeaseAndStartsRun(t *testing.T) {
	const secretToken = "secret-execution-token"
	var leaseRequest executionLeaseCreateRequest
	var runRequest runStartRequest
	var runRequests atomic.Int32
	var unexpectedRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/execution-jobs/leases":
			decodeStrict(t, r, &leaseRequest)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"lease-1","execution_job_id":"execution-job-1","task_id":"task-1","checkout_receipt_id":"receipt-1","repo_binding_id":"repo-1","runner_id":"runner-1","state":"active","lease_token":"` + secretToken + `","expires_at":"2026-05-07T13:00:00Z","execution_job":{"id":"execution-job-1","task_id":"task-1","repo_binding_id":"repo-1","checkout_receipt_id":"receipt-1","state":"leased"}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/execution-jobs/execution-job-1/runs":
			runRequests.Add(1)
			decodeStrict(t, r, &runRequest)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"run-1","execution_job_id":"execution-job-1","execution_lease_id":"lease-1","task_id":"task-1","runner_id":"runner-1","state":"started"}`))
		default:
			unexpectedRequests.Add(1)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	var logs bytes.Buffer
	runner := newTestRunner(t, server.URL, &logs)
	result, err := runner.Step(context.Background())
	if err != nil {
		t.Fatalf("Step() error = %v", err)
	}
	if result != StepRunStarted {
		t.Fatalf("Step() = %q, want %q", result, StepRunStarted)
	}
	if leaseRequest.ProjectID != "project-1" || leaseRequest.RepoBindingID != "repo-1" || leaseRequest.RunnerID != "runner-1" || leaseRequest.TTLSeconds != 900 {
		t.Fatalf("lease request = %#v, want scoped execution runner request", leaseRequest)
	}
	if runRequests.Load() != 1 {
		t.Fatalf("run requests = %d, want 1", runRequests.Load())
	}
	if runRequest.LeaseID != "lease-1" || runRequest.LeaseToken != secretToken || runRequest.RunnerID != "runner-1" {
		t.Fatalf("run start proof = %#v, want lease proof", runRequest)
	}
	if unexpectedRequests.Load() != 0 {
		t.Fatalf("unexpected requests = %d, want no execution receipt or command-execution calls", unexpectedRequests.Load())
	}
	if strings.Contains(logs.String(), secretToken) {
		t.Fatalf("logs leaked execution lease token: %q", logs.String())
	}
	for _, forbidden := range []string{"execution receipt", "executed command"} {
		if strings.Contains(logs.String(), forbidden) {
			t.Fatalf("logs = %q, want no %q claim", logs.String(), forbidden)
		}
	}
}

func TestStepSubmitsNoCommandExecutionReceipt(t *testing.T) {
	const secretToken = "secret-execution-token"
	var receiptRequest executionReceiptRequest
	var receiptRequests atomic.Int32
	var commandRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/execution-jobs/leases":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"lease-1","execution_job_id":"execution-job-1","task_id":"task-1","checkout_receipt_id":"receipt-1","repo_binding_id":"repo-1","runner_id":"runner-1","state":"active","lease_token":"` + secretToken + `","expires_at":"2026-05-07T13:00:00Z","execution_job":{"id":"execution-job-1","task_id":"task-1","repo_binding_id":"repo-1","checkout_receipt_id":"receipt-1","state":"leased"}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/execution-jobs/execution-job-1/runs":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"run-1","execution_job_id":"execution-job-1","execution_lease_id":"lease-1","task_id":"task-1","checkout_receipt_id":"receipt-1","runner_id":"runner-1","state":"started"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/runs/run-1/receipts":
			receiptRequests.Add(1)
			decodeStrict(t, r, &receiptRequest)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"execution-receipt-1","run_id":"run-1","execution_job_id":"execution-job-1","runner_id":"runner-1","execution_mode":"no_command","process_status":"not_executed"}`))
		default:
			if strings.Contains(r.URL.Path, "commands") || strings.Contains(r.URL.Path, "exec") {
				commandRequests.Add(1)
			}
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	var logs bytes.Buffer
	runner, err := NewRunner(Config{
		ServerURL:       server.URL,
		BearerToken:     "runner-token",
		ProjectID:       "project-1",
		RepoBindingID:   "repo-1",
		RunnerID:        "runner-1",
		WorkspaceRef:    "mounted:/workspace/goalrail",
		CommitSHA:       "abc123",
		BaselineID:      "baseline-1",
		OverlayID:       "overlay-1",
		SubmitReceipt:   true,
		PollInterval:    time.Millisecond,
		LeaseTTLSeconds: 900,
		Once:            true,
		LogWriter:       &logs,
	})
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}
	result, err := runner.Step(context.Background())
	if err != nil {
		t.Fatalf("Step() error = %v", err)
	}
	if result != StepReceiptSubmitted {
		t.Fatalf("Step() = %q, want %q", result, StepReceiptSubmitted)
	}
	if receiptRequests.Load() != 1 {
		t.Fatalf("receipt requests = %d, want 1", receiptRequests.Load())
	}
	if receiptRequest.ExecutionJobID != "execution-job-1" || receiptRequest.LeaseID != "lease-1" || receiptRequest.LeaseToken != secretToken || receiptRequest.RunnerID != "runner-1" {
		t.Fatalf("receipt proof = %#v, want lease-backed run proof", receiptRequest)
	}
	if receiptRequest.ExecutionMode != "no_command" || receiptRequest.ProcessStatus != "not_executed" || receiptRequest.RawSourceUploaded {
		t.Fatalf("receipt mode/status = %#v, want no-command metadata receipt", receiptRequest)
	}
	if len(receiptRequest.ArtifactRefs) != 0 || len(receiptRequest.ChangedPathsSummary) != 0 {
		t.Fatalf("receipt artifact/path claims = %#v/%#v, want empty no-command evidence arrays", receiptRequest.ArtifactRefs, receiptRequest.ChangedPathsSummary)
	}
	if receiptRequest.WorkspaceRef != "mounted:/workspace/goalrail" || receiptRequest.CommitSHA != "abc123" {
		t.Fatalf("receipt workspace metadata = %#v, want configured metadata", receiptRequest)
	}
	if commandRequests.Load() != 0 {
		t.Fatalf("command requests = %d, want no command execution calls", commandRequests.Load())
	}
	if strings.Contains(logs.String(), secretToken) {
		t.Fatalf("logs leaked execution lease token: %q", logs.String())
	}
	for _, forbidden := range []string{"executed command", "gate", "proof"} {
		if strings.Contains(logs.String(), forbidden) {
			t.Fatalf("logs = %q, want no %q claim", logs.String(), forbidden)
		}
	}
}

func TestStepSubmitsBuiltinDiagnosticReceipt(t *testing.T) {
	const secretToken = "secret-execution-token"
	var commandPlanRequest executionCommandPlanRequest
	var receiptRequest executionReceiptRequest
	var commandPlanRequests atomic.Int32
	var receiptRequests atomic.Int32
	var commandExecutionRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/execution-jobs/leases":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"lease-1","execution_job_id":"execution-job-1","task_id":"task-1","checkout_receipt_id":"receipt-1","repo_binding_id":"repo-1","runner_id":"runner-1","state":"active","lease_token":"` + secretToken + `","expires_at":"2026-05-07T13:00:00Z","execution_job":{"id":"execution-job-1","task_id":"task-1","repo_binding_id":"repo-1","checkout_receipt_id":"receipt-1","state":"leased"}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/execution-jobs/execution-job-1/runs":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"run-1","execution_job_id":"execution-job-1","execution_lease_id":"lease-1","task_id":"task-1","checkout_receipt_id":"receipt-1","runner_id":"runner-1","state":"started"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/runs/run-1/command-plans":
			commandPlanRequests.Add(1)
			decodeStrict(t, r, &commandPlanRequest)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"command-plan-1","project_id":"project-1","repo_binding_id":"repo-1","task_id":"task-1","checkout_receipt_id":"receipt-1","execution_job_id":"execution-job-1","run_id":"run-1","command_kind":"builtin_diagnostic","action":"workspace_status","shell_allowed":false,"argv":[],"working_directory":".","path_scope":["."],"timeout_seconds":30,"max_stdout_bytes":0,"max_stderr_bytes":0,"allowed_artifact_kinds":[],"raw_source_upload_allowed":false,"state":"planned"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/runs/run-1/receipts":
			receiptRequests.Add(1)
			decodeStrict(t, r, &receiptRequest)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"execution-receipt-1","run_id":"run-1","execution_job_id":"execution-job-1","runner_id":"runner-1","execution_mode":"builtin_diagnostic","command_plan_id":"command-plan-1","command_kind":"builtin_diagnostic","action":"workspace_status","process_status":"metadata_only"}`))
		default:
			if strings.Contains(r.URL.Path, "shell") || strings.Contains(r.URL.Path, "commands") || strings.Contains(r.URL.Path, "exec") {
				commandExecutionRequests.Add(1)
			}
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	var logs bytes.Buffer
	runner, err := NewRunner(Config{
		ServerURL:         server.URL,
		BearerToken:       "runner-token",
		ProjectID:         "project-1",
		RepoBindingID:     "repo-1",
		RunnerID:          "runner-1",
		WorkspaceRef:      "mounted:/workspace/goalrail",
		CommitSHA:         "abc123",
		BaselineID:        "baseline-1",
		OverlayID:         "overlay-1",
		BuiltinDiagnostic: true,
		PollInterval:      time.Millisecond,
		LeaseTTLSeconds:   900,
		Once:              true,
		LogWriter:         &logs,
	})
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}
	result, err := runner.Step(context.Background())
	if err != nil {
		t.Fatalf("Step() error = %v", err)
	}
	if result != StepReceiptSubmitted {
		t.Fatalf("Step() = %q, want %q", result, StepReceiptSubmitted)
	}
	if commandPlanRequests.Load() != 1 || receiptRequests.Load() != 1 {
		t.Fatalf("command plan/receipt requests = %d/%d, want 1/1", commandPlanRequests.Load(), receiptRequests.Load())
	}
	if commandPlanRequest.ProjectID != "project-1" || commandPlanRequest.RepoBindingID != "repo-1" || commandPlanRequest.CommandKind != "builtin_diagnostic" || commandPlanRequest.Action != "workspace_status" {
		t.Fatalf("command plan request = %#v, want scoped fixed builtin diagnostic request", commandPlanRequest)
	}
	if receiptRequest.ExecutionMode != "builtin_diagnostic" || receiptRequest.CommandPlanID != "command-plan-1" || receiptRequest.CommandKind != "builtin_diagnostic" || receiptRequest.Action != "workspace_status" || receiptRequest.ProcessStatus != "metadata_only" {
		t.Fatalf("diagnostic receipt command metadata = %#v, want fixed builtin diagnostic metadata", receiptRequest)
	}
	if receiptRequest.ExecutionJobID != "execution-job-1" || receiptRequest.LeaseID != "lease-1" || receiptRequest.LeaseToken != secretToken || receiptRequest.RunnerID != "runner-1" {
		t.Fatalf("diagnostic receipt proof = %#v, want lease-backed run proof", receiptRequest)
	}
	if receiptRequest.RawSourceUploaded || len(receiptRequest.ArtifactRefs) != 0 || len(receiptRequest.ChangedPathsSummary) != 0 || receiptRequest.RunnerStartedAt == nil || receiptRequest.RunnerFinishedAt == nil {
		t.Fatalf("diagnostic receipt evidence = %#v, want metadata-only empty evidence", receiptRequest)
	}
	if commandExecutionRequests.Load() != 0 {
		t.Fatalf("command execution requests = %d, want none", commandExecutionRequests.Load())
	}
	if strings.Contains(logs.String(), secretToken) {
		t.Fatalf("logs leaked execution lease token: %q", logs.String())
	}
	for _, forbidden := range []string{"executed command", "shell", "gate", "proof"} {
		if strings.Contains(logs.String(), forbidden) {
			t.Fatalf("logs = %q, want no %q claim", logs.String(), forbidden)
		}
	}
}

func TestStepSubmitsProjectProbeReceipt(t *testing.T) {
	const secretToken = "secret-execution-token"
	workspaceRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspaceRoot, "package.json"), []byte(`{"packageManager":"pnpm@9.0.0","scripts":{"test":"vitest run","test:unit":"vitest run unit","build":"vite build"}}`), 0o600); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspaceRoot, "go.mod"), []byte("module example.com/project\n\ngo 1.23\n"), 0o600); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(workspaceRoot, "nested"), 0o700); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspaceRoot, "nested", "package.json"), []byte(`{"scripts":{"test":"should not be scanned"}}`), 0o600); err != nil {
		t.Fatalf("write nested package.json: %v", err)
	}

	var commandPlanRequest executionCommandPlanRequest
	var receiptRequest executionReceiptRequest
	var commandPlanRequests atomic.Int32
	var receiptRequests atomic.Int32
	var commandExecutionRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/execution-jobs/leases":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"lease-1","execution_job_id":"execution-job-1","task_id":"task-1","checkout_receipt_id":"receipt-1","repo_binding_id":"repo-1","runner_id":"runner-1","state":"active","lease_token":"` + secretToken + `","expires_at":"2026-05-07T13:00:00Z","execution_job":{"id":"execution-job-1","task_id":"task-1","repo_binding_id":"repo-1","checkout_receipt_id":"receipt-1","state":"leased"}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/execution-jobs/execution-job-1/runs":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"run-1","execution_job_id":"execution-job-1","execution_lease_id":"lease-1","task_id":"task-1","checkout_receipt_id":"receipt-1","runner_id":"runner-1","state":"started"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/runs/run-1/command-plans":
			commandPlanRequests.Add(1)
			decodeStrict(t, r, &commandPlanRequest)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"command-plan-1","project_id":"project-1","repo_binding_id":"repo-1","task_id":"task-1","checkout_receipt_id":"receipt-1","execution_job_id":"execution-job-1","run_id":"run-1","command_kind":"project_probe","action":"detect_declared_test_targets","shell_allowed":false,"argv":[],"working_directory":".","path_scope":["."],"timeout_seconds":30,"max_stdout_bytes":0,"max_stderr_bytes":0,"allowed_artifact_kinds":[],"raw_source_upload_allowed":false,"state":"planned"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/runs/run-1/receipts":
			receiptRequests.Add(1)
			decodeStrict(t, r, &receiptRequest)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"execution-receipt-1","run_id":"run-1","execution_job_id":"execution-job-1","runner_id":"runner-1","execution_mode":"project_probe","command_plan_id":"command-plan-1","command_kind":"project_probe","action":"detect_declared_test_targets","process_status":"metadata_only"}`))
		default:
			if strings.Contains(r.URL.Path, "shell") || strings.Contains(r.URL.Path, "commands") || strings.Contains(r.URL.Path, "exec") {
				commandExecutionRequests.Add(1)
			}
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	var logs bytes.Buffer
	runner, err := NewRunner(Config{
		ServerURL:       server.URL,
		BearerToken:     "runner-token",
		ProjectID:       "project-1",
		RepoBindingID:   "repo-1",
		RunnerID:        "runner-1",
		WorkspaceRef:    "mounted:/workspace/goalrail",
		WorkspaceRoot:   workspaceRoot,
		CommitSHA:       "abc123",
		BaselineID:      "baseline-1",
		OverlayID:       "overlay-1",
		ProjectProbe:    true,
		PollInterval:    time.Millisecond,
		LeaseTTLSeconds: 900,
		Once:            true,
		LogWriter:       &logs,
	})
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}
	result, err := runner.Step(context.Background())
	if err != nil {
		t.Fatalf("Step() error = %v", err)
	}
	if result != StepReceiptSubmitted {
		t.Fatalf("Step() = %q, want %q", result, StepReceiptSubmitted)
	}
	if commandPlanRequests.Load() != 1 || receiptRequests.Load() != 1 {
		t.Fatalf("command plan/receipt requests = %d/%d, want 1/1", commandPlanRequests.Load(), receiptRequests.Load())
	}
	if commandPlanRequest.ProjectID != "project-1" || commandPlanRequest.RepoBindingID != "repo-1" || commandPlanRequest.CommandKind != "project_probe" || commandPlanRequest.Action != "detect_declared_test_targets" {
		t.Fatalf("command plan request = %#v, want scoped fixed project probe request", commandPlanRequest)
	}
	if receiptRequest.ExecutionMode != "project_probe" || receiptRequest.CommandPlanID != "command-plan-1" || receiptRequest.CommandKind != "project_probe" || receiptRequest.Action != "detect_declared_test_targets" || receiptRequest.ProcessStatus != "metadata_only" {
		t.Fatalf("project probe receipt command metadata = %#v, want fixed project probe metadata", receiptRequest)
	}
	if receiptRequest.RawSourceUploaded || len(receiptRequest.ArtifactRefs) != 0 || len(receiptRequest.ChangedPathsSummary) != 0 || receiptRequest.RunnerStartedAt == nil || receiptRequest.RunnerFinishedAt == nil {
		t.Fatalf("project probe receipt evidence = %#v, want metadata-only empty evidence", receiptRequest)
	}
	if receiptRequest.ProjectProbeMetadata == nil {
		t.Fatal("project probe metadata = nil, want structured metadata")
	}
	if hasManifest(receiptRequest.ProjectProbeMetadata, "nested/package.json") {
		t.Fatalf("project probe metadata = %#v, must not recursively scan nested manifests", receiptRequest.ProjectProbeMetadata)
	}
	for _, want := range []string{"package.json", "go.mod"} {
		if !hasManifest(receiptRequest.ProjectProbeMetadata, want) {
			t.Fatalf("project probe manifests = %#v, want %s", receiptRequest.ProjectProbeMetadata.DetectedManifests, want)
		}
	}
	for _, want := range []string{"test", "test:unit", "go_package_tests"} {
		if !hasTestTarget(receiptRequest.ProjectProbeMetadata, want) {
			t.Fatalf("project probe test targets = %#v, want %s", receiptRequest.ProjectProbeMetadata.DeclaredTestTargetCandidates, want)
		}
	}
	if hasTestTarget(receiptRequest.ProjectProbeMetadata, "build") {
		t.Fatalf("project probe test targets = %#v, must not include non-test package scripts", receiptRequest.ProjectProbeMetadata.DeclaredTestTargetCandidates)
	}
	metadataPayload, err := json.Marshal(receiptRequest.ProjectProbeMetadata)
	if err != nil {
		t.Fatalf("marshal project probe metadata: %v", err)
	}
	for _, rawBody := range []string{"vitest run", "vite build", "module example.com/project"} {
		if strings.Contains(string(metadataPayload), rawBody) {
			t.Fatalf("project probe metadata contains raw manifest body %q: %s", rawBody, metadataPayload)
		}
	}
	if commandExecutionRequests.Load() != 0 {
		t.Fatalf("command execution requests = %d, want none", commandExecutionRequests.Load())
	}
	if strings.Contains(logs.String(), secretToken) {
		t.Fatalf("logs leaked execution lease token: %q", logs.String())
	}
	for _, forbidden := range []string{"executed command", "shell", "gate", "proof"} {
		if strings.Contains(logs.String(), forbidden) {
			t.Fatalf("logs = %q, want no %q claim", logs.String(), forbidden)
		}
	}
}

func TestStepSubmitsProjectTestReceipt(t *testing.T) {
	const secretToken = "secret-execution-token"
	workspaceRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspaceRoot, "package.json"), []byte(`{"scripts":{"test":"node should-not-run.js"}}`), 0o600); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	fakeBin := t.TempDir()
	executionMarker := filepath.Join(t.TempDir(), "npm-executed")
	if err := os.WriteFile(filepath.Join(fakeBin, "npm"), []byte("#!/bin/sh\nprintf executed > \"$GOALRAIL_TEST_EXECUTION_MARKER\"\nexit 0\n"), 0o700); err != nil {
		t.Fatalf("write fake npm: %v", err)
	}
	t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("GOALRAIL_TEST_EXECUTION_MARKER", executionMarker)

	var receiptRequest executionReceiptRequest
	var getPlanRequests atomic.Int32
	var createPlanRequests atomic.Int32
	var receiptRequests atomic.Int32
	var commandExecutionRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/execution-jobs/leases":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"lease-1","execution_job_id":"execution-job-1","task_id":"task-1","checkout_receipt_id":"receipt-1","repo_binding_id":"repo-1","runner_id":"runner-1","state":"active","lease_token":"` + secretToken + `","expires_at":"2026-05-07T13:00:00Z","execution_job":{"id":"execution-job-1","task_id":"task-1","repo_binding_id":"repo-1","checkout_receipt_id":"receipt-1","state":"leased"}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/execution-jobs/execution-job-1/runs":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"run-1","execution_job_id":"execution-job-1","execution_lease_id":"lease-1","task_id":"task-1","checkout_receipt_id":"receipt-1","runner_id":"runner-1","state":"started"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/runs/run-1/command-plans/project_test/run_declared_test_target":
			getPlanRequests.Add(1)
			w.WriteHeader(http.StatusOK)
			planJSON, err := json.Marshal(projectTestPlanFixture())
			if err != nil {
				t.Fatalf("marshal project test plan: %v", err)
			}
			_, _ = w.Write(planJSON)
		case r.Method == http.MethodPost && r.URL.Path == "/v1/runs/run-1/command-plans":
			createPlanRequests.Add(1)
			http.Error(w, "unexpected command plan create", http.StatusInternalServerError)
		case r.Method == http.MethodPost && r.URL.Path == "/v1/runs/run-1/receipts":
			receiptRequests.Add(1)
			decodeStrict(t, r, &receiptRequest)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"execution-receipt-1","run_id":"run-1","execution_job_id":"execution-job-1","runner_id":"runner-1","execution_mode":"project_test","command_plan_id":"command-plan-1","command_kind":"project_test","action":"run_declared_test_target","process_status":"policy_rejected"}`))
		default:
			if strings.Contains(r.URL.Path, "shell") ||
				strings.Contains(r.URL.Path, "exec") ||
				strings.Contains(r.URL.Path, "stdout") ||
				strings.Contains(r.URL.Path, "stderr") ||
				strings.Contains(r.URL.Path, "artifact") ||
				strings.Contains(r.URL.Path, "source") {
				commandExecutionRequests.Add(1)
			}
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	var logs bytes.Buffer
	runner, err := NewRunner(Config{
		ServerURL:       server.URL,
		BearerToken:     "runner-token",
		ProjectID:       "project-1",
		RepoBindingID:   "repo-1",
		RunnerID:        "runner-1",
		WorkspaceRef:    "mounted:/workspace/goalrail",
		WorkspaceRoot:   workspaceRoot,
		CommitSHA:       "abc123",
		BaselineID:      "baseline-1",
		OverlayID:       "overlay-1",
		ProjectTest:     true,
		PollInterval:    time.Millisecond,
		LeaseTTLSeconds: 900,
		Once:            true,
		LogWriter:       &logs,
	})
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}
	result, err := runner.Step(context.Background())
	if err != nil {
		t.Fatalf("Step() error = %v", err)
	}
	if result != StepReceiptSubmitted {
		t.Fatalf("Step() = %q, want %q", result, StepReceiptSubmitted)
	}
	if getPlanRequests.Load() != 1 || createPlanRequests.Load() != 0 || receiptRequests.Load() != 1 {
		t.Fatalf("get/create/receipt requests = %d/%d/%d, want 1/0/1", getPlanRequests.Load(), createPlanRequests.Load(), receiptRequests.Load())
	}
	if commandExecutionRequests.Load() != 0 {
		t.Fatalf("command/output/artifact/source requests = %d, want none", commandExecutionRequests.Load())
	}
	if receiptRequest.ExecutionMode != "project_test" || receiptRequest.CommandPlanID != "command-plan-1" || receiptRequest.CommandKind != "project_test" || receiptRequest.Action != "run_declared_test_target" || receiptRequest.ProcessStatus != "policy_rejected" {
		t.Fatalf("project test receipt command metadata = %#v, want policy_rejected project_test metadata", receiptRequest)
	}
	if receiptRequest.ExitCode != nil {
		t.Fatalf("project test receipt exit_code = %#v, want nil for policy_rejected", receiptRequest.ExitCode)
	}
	if receiptRequest.ProjectProbeMetadata != nil || receiptRequest.RawSourceUploaded || len(receiptRequest.ArtifactRefs) != 0 || len(receiptRequest.ChangedPathsSummary) != 0 || receiptRequest.RunnerStartedAt == nil || receiptRequest.RunnerFinishedAt == nil {
		t.Fatalf("project test receipt evidence = %#v, want policy rejection without probe metadata/artifacts/raw source", receiptRequest)
	}
	if _, err := os.Stat(executionMarker); !os.IsNotExist(err) {
		t.Fatalf("project test execution marker stat = %v, want no npm/process execution", err)
	}
	for _, forbidden := range []string{secretToken, "gate", "proof", "shell", "executed", "stdout", "stderr", "raw manifest", "secret"} {
		if strings.Contains(logs.String(), forbidden) {
			t.Fatalf("logs = %q, want no %q", logs.String(), forbidden)
		}
	}
}

func TestProjectProbeDetectsOnlyAllowlistedRootManifests(t *testing.T) {
	workspaceRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspaceRoot, "package.json"), []byte(`{"scripts":{"test":"node test.js"}}`), 0o600); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(workspaceRoot, "packages", "app"), 0o700); err != nil {
		t.Fatalf("mkdir nested package: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspaceRoot, "packages", "app", "package.json"), []byte(`{"scripts":{"test":"node nested.js"}}`), 0o600); err != nil {
		t.Fatalf("write nested package.json: %v", err)
	}

	metadata, err := detectDeclaredTestTargets(workspaceRoot, executionCommandPlan{
		WorkingDirectory: ".",
		PathScope:        []string{"."},
	})
	if err != nil {
		t.Fatalf("detectDeclaredTestTargets() error = %v", err)
	}
	if !hasManifest(&metadata, "package.json") {
		t.Fatalf("manifests = %#v, want root package.json", metadata.DetectedManifests)
	}
	if hasManifest(&metadata, "packages/app/package.json") {
		t.Fatalf("manifests = %#v, must not recursively scan nested package.json", metadata.DetectedManifests)
	}
	if !hasTestTarget(&metadata, "test") {
		t.Fatalf("test targets = %#v, want package.json test script name", metadata.DeclaredTestTargetCandidates)
	}
}

func TestProjectProbeModeRequiresWorkspaceRoot(t *testing.T) {
	_, err := NewRunner(Config{
		ServerURL:       "http://127.0.0.1:9999",
		BearerToken:     "runner-token",
		ProjectID:       "project-1",
		RepoBindingID:   "repo-1",
		RunnerID:        "runner-1",
		WorkspaceRef:    "mounted:/workspace/goalrail",
		CommitSHA:       "abc123",
		ProjectProbe:    true,
		PollInterval:    time.Millisecond,
		LeaseTTLSeconds: 900,
		Once:            true,
	})
	if err == nil || !strings.Contains(err.Error(), "workspace root is required") {
		t.Fatalf("NewRunner() error = %v, want workspace root requirement", err)
	}
}

func TestProjectTestModeRequiresWorkspaceRoot(t *testing.T) {
	_, err := NewRunner(Config{
		ServerURL:       "http://127.0.0.1:9999",
		BearerToken:     "runner-token",
		ProjectID:       "project-1",
		RepoBindingID:   "repo-1",
		RunnerID:        "runner-1",
		WorkspaceRef:    "mounted:/workspace/goalrail",
		CommitSHA:       "abc123",
		ProjectTest:     true,
		PollInterval:    time.Millisecond,
		LeaseTTLSeconds: 900,
		Once:            true,
	})
	if err == nil || !strings.Contains(err.Error(), "workspace root is required") {
		t.Fatalf("NewRunner() error = %v, want workspace root requirement", err)
	}
}

func TestStepRejectsUnsafeBuiltinDiagnosticPlan(t *testing.T) {
	const secretToken = "secret-execution-token"
	var receiptRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/execution-jobs/leases":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"lease-1","execution_job_id":"execution-job-1","task_id":"task-1","checkout_receipt_id":"receipt-1","repo_binding_id":"repo-1","runner_id":"runner-1","state":"active","lease_token":"` + secretToken + `","expires_at":"2026-05-07T13:00:00Z","execution_job":{"id":"execution-job-1","task_id":"task-1","repo_binding_id":"repo-1","checkout_receipt_id":"receipt-1","state":"leased"}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/execution-jobs/execution-job-1/runs":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"run-1","execution_job_id":"execution-job-1","execution_lease_id":"lease-1","task_id":"task-1","checkout_receipt_id":"receipt-1","runner_id":"runner-1","state":"started"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/runs/run-1/command-plans":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"command-plan-1","project_id":"project-1","repo_binding_id":"repo-1","task_id":"task-1","checkout_receipt_id":"receipt-1","execution_job_id":"execution-job-1","run_id":"run-1","command_kind":"builtin_diagnostic","action":"workspace_status","shell_allowed":true,"argv":[],"working_directory":".","path_scope":["."],"timeout_seconds":30,"max_stdout_bytes":0,"max_stderr_bytes":0,"allowed_artifact_kinds":[],"raw_source_upload_allowed":false,"state":"planned"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/runs/run-1/receipts":
			receiptRequests.Add(1)
			http.Error(w, "unexpected receipt", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	var logs bytes.Buffer
	runner, err := NewRunner(Config{
		ServerURL:         server.URL,
		BearerToken:       "runner-token",
		ProjectID:         "project-1",
		RepoBindingID:     "repo-1",
		RunnerID:          "runner-1",
		WorkspaceRef:      "mounted:/workspace/goalrail",
		CommitSHA:         "abc123",
		BuiltinDiagnostic: true,
		PollInterval:      time.Millisecond,
		LeaseTTLSeconds:   900,
		Once:              true,
		LogWriter:         &logs,
	})
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}
	_, err = runner.Step(context.Background())
	if err == nil || !strings.Contains(err.Error(), "allows shell") {
		t.Fatalf("Step() error = %v, want shell policy rejection", err)
	}
	if receiptRequests.Load() != 0 {
		t.Fatalf("receipt requests = %d, want 0 after unsafe command plan", receiptRequests.Load())
	}
	if strings.Contains(logs.String(), secretToken) {
		t.Fatalf("logs leaked execution lease token: %q", logs.String())
	}
}

func TestStepRejectsUnsafeProjectProbePlan(t *testing.T) {
	const secretToken = "secret-execution-token"
	basePlan := executionCommandPlan{
		ID:                   "command-plan-1",
		ProjectID:            "project-1",
		RepoBindingID:        "repo-1",
		TaskID:               "task-1",
		CheckoutReceiptID:    "receipt-1",
		ExecutionJobID:       "execution-job-1",
		RunID:                "run-1",
		CommandKind:          "project_probe",
		Action:               "detect_declared_test_targets",
		WorkingDirectory:     ".",
		PathScope:            []string{"."},
		TimeoutSeconds:       30,
		AllowedArtifactKinds: []string{},
		State:                "planned",
	}
	for _, tt := range []struct {
		name    string
		mutate  func(*executionCommandPlan)
		wantErr string
	}{
		{
			name: "shell",
			mutate: func(plan *executionCommandPlan) {
				plan.ShellAllowed = true
			},
			wantErr: "allows shell",
		},
		{
			name: "argv",
			mutate: func(plan *executionCommandPlan) {
				plan.Argv = []string{"npm", "test"}
			},
			wantErr: "argv must be empty",
		},
		{
			name: "working directory",
			mutate: func(plan *executionCommandPlan) {
				plan.WorkingDirectory = "packages/app"
			},
			wantErr: "working directory",
		},
		{
			name: "path scope",
			mutate: func(plan *executionCommandPlan) {
				plan.PathScope = []string{"packages/app"}
			},
			wantErr: "path scope",
		},
		{
			name: "stdout capture",
			mutate: func(plan *executionCommandPlan) {
				plan.MaxStdoutBytes = 1024
			},
			wantErr: "output/timeout policy",
		},
		{
			name: "stderr capture",
			mutate: func(plan *executionCommandPlan) {
				plan.MaxStderrBytes = 1024
			},
			wantErr: "output/timeout policy",
		},
		{
			name: "artifacts",
			mutate: func(plan *executionCommandPlan) {
				plan.AllowedArtifactKinds = []string{"log"}
			},
			wantErr: "allowed artifacts",
		},
		{
			name: "raw source upload",
			mutate: func(plan *executionCommandPlan) {
				plan.RawSourceUploadAllowed = true
			},
			wantErr: "raw source upload",
		},
		{
			name: "wrong kind",
			mutate: func(plan *executionCommandPlan) {
				plan.CommandKind = "builtin_diagnostic"
			},
			wantErr: "unsupported execution command kind",
		},
		{
			name: "wrong action",
			mutate: func(plan *executionCommandPlan) {
				plan.Action = "run_tests"
			},
			wantErr: "unsupported execution command action",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			workspaceRoot := t.TempDir()
			plan := basePlan
			tt.mutate(&plan)
			planJSON, err := json.Marshal(plan)
			if err != nil {
				t.Fatalf("marshal command plan: %v", err)
			}
			var receiptRequests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == http.MethodPost && r.URL.Path == "/v1/execution-jobs/leases":
					w.WriteHeader(http.StatusCreated)
					_, _ = w.Write([]byte(`{"id":"lease-1","execution_job_id":"execution-job-1","task_id":"task-1","checkout_receipt_id":"receipt-1","repo_binding_id":"repo-1","runner_id":"runner-1","state":"active","lease_token":"` + secretToken + `","expires_at":"2026-05-07T13:00:00Z","execution_job":{"id":"execution-job-1","task_id":"task-1","repo_binding_id":"repo-1","checkout_receipt_id":"receipt-1","state":"leased"}}`))
				case r.Method == http.MethodPost && r.URL.Path == "/v1/execution-jobs/execution-job-1/runs":
					w.WriteHeader(http.StatusCreated)
					_, _ = w.Write([]byte(`{"id":"run-1","execution_job_id":"execution-job-1","execution_lease_id":"lease-1","task_id":"task-1","checkout_receipt_id":"receipt-1","runner_id":"runner-1","state":"started"}`))
				case r.Method == http.MethodPost && r.URL.Path == "/v1/runs/run-1/command-plans":
					w.WriteHeader(http.StatusCreated)
					_, _ = w.Write(planJSON)
				case r.Method == http.MethodPost && r.URL.Path == "/v1/runs/run-1/receipts":
					receiptRequests.Add(1)
					http.Error(w, "unexpected receipt", http.StatusInternalServerError)
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()

			var logs bytes.Buffer
			runner, err := NewRunner(Config{
				ServerURL:       server.URL,
				BearerToken:     "runner-token",
				ProjectID:       "project-1",
				RepoBindingID:   "repo-1",
				RunnerID:        "runner-1",
				WorkspaceRef:    "mounted:/workspace/goalrail",
				WorkspaceRoot:   workspaceRoot,
				CommitSHA:       "abc123",
				ProjectProbe:    true,
				PollInterval:    time.Millisecond,
				LeaseTTLSeconds: 900,
				Once:            true,
				LogWriter:       &logs,
			})
			if err != nil {
				t.Fatalf("NewRunner() error = %v", err)
			}
			_, err = runner.Step(context.Background())
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Step() error = %v, want %q policy rejection", err, tt.wantErr)
			}
			if receiptRequests.Load() != 0 {
				t.Fatalf("receipt requests = %d, want 0 after unsafe command plan", receiptRequests.Load())
			}
			if strings.Contains(logs.String(), secretToken) {
				t.Fatalf("logs leaked execution lease token: %q", logs.String())
			}
		})
	}
}

func TestStepRejectsUnsafeProjectTestPlan(t *testing.T) {
	const secretToken = "secret-execution-token"
	for _, tt := range []struct {
		name    string
		mutate  func(*executionCommandPlan)
		wantErr string
	}{
		{
			name: "shell",
			mutate: func(plan *executionCommandPlan) {
				plan.ShellAllowed = true
			},
			wantErr: "allows shell",
		},
		{
			name: "argv",
			mutate: func(plan *executionCommandPlan) {
				plan.Argv = []string{"npm", "test"}
			},
			wantErr: "argv must be empty",
		},
		{
			name: "wrong kind",
			mutate: func(plan *executionCommandPlan) {
				plan.CommandKind = "project_probe"
			},
			wantErr: "unsupported execution command kind",
		},
		{
			name: "wrong action",
			mutate: func(plan *executionCommandPlan) {
				plan.Action = "run_tests"
			},
			wantErr: "unsupported execution command action",
		},
		{
			name: "missing source receipt",
			mutate: func(plan *executionCommandPlan) {
				plan.SourceProjectProbeReceiptID = ""
			},
			wantErr: "source project probe receipt",
		},
		{
			name: "missing selected target",
			mutate: func(plan *executionCommandPlan) {
				plan.SelectedTargetID = ""
			},
			wantErr: "selected target id",
		},
		{
			name: "missing declared target",
			mutate: func(plan *executionCommandPlan) {
				plan.DeclaredTestTarget = nil
			},
			wantErr: "declared test target",
		},
		{
			name: "unsupported target family",
			mutate: func(plan *executionCommandPlan) {
				plan.DeclaredTestTarget.SourceKind = "go_module_manifest"
			},
			wantErr: "unsupported project test target family",
		},
		{
			name: "target mismatch",
			mutate: func(plan *executionCommandPlan) {
				plan.SelectedTargetID = "package.json#package_json_script:test:unit"
			},
			wantErr: "does not match declared target",
		},
		{
			name: "nested source path",
			mutate: func(plan *executionCommandPlan) {
				plan.DeclaredTestTarget.SourcePath = "packages/app/package.json"
				plan.SelectedTargetID = "packages/app/package.json#package_json_script:test"
			},
			wantErr: "outside root package manifest scope",
		},
		{
			name: "working directory",
			mutate: func(plan *executionCommandPlan) {
				plan.WorkingDirectory = "packages/app"
			},
			wantErr: "working directory",
		},
		{
			name: "path scope",
			mutate: func(plan *executionCommandPlan) {
				plan.PathScope = []string{"packages/app"}
			},
			wantErr: "path scope",
		},
		{
			name: "timeout",
			mutate: func(plan *executionCommandPlan) {
				plan.TimeoutSeconds = 0
			},
			wantErr: "output/timeout policy",
		},
		{
			name: "stdout capture",
			mutate: func(plan *executionCommandPlan) {
				plan.MaxStdoutBytes = 1024
			},
			wantErr: "output/timeout policy",
		},
		{
			name: "stderr capture",
			mutate: func(plan *executionCommandPlan) {
				plan.MaxStderrBytes = 1024
			},
			wantErr: "output/timeout policy",
		},
		{
			name: "network",
			mutate: func(plan *executionCommandPlan) {
				plan.NetworkAllowed = true
			},
			wantErr: "allows network",
		},
		{
			name: "workspace writes",
			mutate: func(plan *executionCommandPlan) {
				plan.WorkspaceWriteAllowed = true
			},
			wantErr: "workspace writes",
		},
		{
			name: "scratch writes",
			mutate: func(plan *executionCommandPlan) {
				plan.ScratchWriteAllowed = true
			},
			wantErr: "scratch writes",
		},
		{
			name: "artifacts",
			mutate: func(plan *executionCommandPlan) {
				plan.AllowedArtifactKinds = []string{"junit"}
			},
			wantErr: "allowed artifacts",
		},
		{
			name: "changed paths",
			mutate: func(plan *executionCommandPlan) {
				plan.ChangedPathsAllowed = true
			},
			wantErr: "changed paths",
		},
		{
			name: "raw source upload",
			mutate: func(plan *executionCommandPlan) {
				plan.RawSourceUploadAllowed = true
			},
			wantErr: "raw source upload",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			plan := projectTestPlanFixture()
			tt.mutate(&plan)
			planJSON, err := json.Marshal(plan)
			if err != nil {
				t.Fatalf("marshal command plan: %v", err)
			}
			var receiptRequests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == http.MethodPost && r.URL.Path == "/v1/execution-jobs/leases":
					w.WriteHeader(http.StatusCreated)
					_, _ = w.Write([]byte(`{"id":"lease-1","execution_job_id":"execution-job-1","task_id":"task-1","checkout_receipt_id":"receipt-1","repo_binding_id":"repo-1","runner_id":"runner-1","state":"active","lease_token":"` + secretToken + `","expires_at":"2026-05-07T13:00:00Z","execution_job":{"id":"execution-job-1","task_id":"task-1","repo_binding_id":"repo-1","checkout_receipt_id":"receipt-1","state":"leased"}}`))
				case r.Method == http.MethodPost && r.URL.Path == "/v1/execution-jobs/execution-job-1/runs":
					w.WriteHeader(http.StatusCreated)
					_, _ = w.Write([]byte(`{"id":"run-1","execution_job_id":"execution-job-1","execution_lease_id":"lease-1","task_id":"task-1","checkout_receipt_id":"receipt-1","runner_id":"runner-1","state":"started"}`))
				case r.Method == http.MethodGet && r.URL.Path == "/v1/runs/run-1/command-plans/project_test/run_declared_test_target":
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write(planJSON)
				case r.Method == http.MethodPost && r.URL.Path == "/v1/runs/run-1/receipts":
					receiptRequests.Add(1)
					http.Error(w, "unexpected receipt", http.StatusInternalServerError)
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()

			var logs bytes.Buffer
			runner, err := NewRunner(Config{
				ServerURL:       server.URL,
				BearerToken:     "runner-token",
				ProjectID:       "project-1",
				RepoBindingID:   "repo-1",
				RunnerID:        "runner-1",
				WorkspaceRef:    "mounted:/workspace/goalrail",
				WorkspaceRoot:   t.TempDir(),
				CommitSHA:       "abc123",
				ProjectTest:     true,
				PollInterval:    time.Millisecond,
				LeaseTTLSeconds: 900,
				Once:            true,
				LogWriter:       &logs,
			})
			if err != nil {
				t.Fatalf("NewRunner() error = %v", err)
			}
			_, err = runner.Step(context.Background())
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Step() error = %v, want %q policy rejection", err, tt.wantErr)
			}
			if receiptRequests.Load() != 0 {
				t.Fatalf("receipt requests = %d, want 0 after unsafe command plan", receiptRequests.Load())
			}
			if strings.Contains(logs.String(), secretToken) {
				t.Fatalf("logs leaked execution lease token: %q", logs.String())
			}
		})
	}
}

func TestStepRejectsProjectTestPlanTraceMismatches(t *testing.T) {
	const secretToken = "secret-execution-token"
	for _, tt := range []struct {
		name    string
		mutate  func(*executionCommandPlan)
		wantErr string
	}{
		{
			name: "project scope",
			mutate: func(plan *executionCommandPlan) {
				plan.ProjectID = "project-other"
			},
			wantErr: "project_id",
		},
		{
			name: "repo binding scope",
			mutate: func(plan *executionCommandPlan) {
				plan.RepoBindingID = "repo-other"
			},
			wantErr: "repo_binding_id",
		},
		{
			name: "execution job lineage",
			mutate: func(plan *executionCommandPlan) {
				plan.ExecutionJobID = "execution-job-other"
			},
			wantErr: "job id",
		},
		{
			name: "run lineage",
			mutate: func(plan *executionCommandPlan) {
				plan.RunID = "run-other"
			},
			wantErr: "run id",
		},
		{
			name: "task lineage",
			mutate: func(plan *executionCommandPlan) {
				plan.TaskID = "task-other"
			},
			wantErr: "task id",
		},
		{
			name: "checkout receipt lineage",
			mutate: func(plan *executionCommandPlan) {
				plan.CheckoutReceiptID = "receipt-other"
			},
			wantErr: "checkout receipt id",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			plan := projectTestPlanFixture()
			tt.mutate(&plan)
			planJSON, err := json.Marshal(plan)
			if err != nil {
				t.Fatalf("marshal command plan: %v", err)
			}
			var receiptRequests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == http.MethodPost && r.URL.Path == "/v1/execution-jobs/leases":
					w.WriteHeader(http.StatusCreated)
					_, _ = w.Write([]byte(`{"id":"lease-1","execution_job_id":"execution-job-1","task_id":"task-1","checkout_receipt_id":"receipt-1","repo_binding_id":"repo-1","runner_id":"runner-1","state":"active","lease_token":"` + secretToken + `","expires_at":"2026-05-07T13:00:00Z","execution_job":{"id":"execution-job-1","task_id":"task-1","repo_binding_id":"repo-1","checkout_receipt_id":"receipt-1","state":"leased"}}`))
				case r.Method == http.MethodPost && r.URL.Path == "/v1/execution-jobs/execution-job-1/runs":
					w.WriteHeader(http.StatusCreated)
					_, _ = w.Write([]byte(`{"id":"run-1","execution_job_id":"execution-job-1","execution_lease_id":"lease-1","task_id":"task-1","checkout_receipt_id":"receipt-1","runner_id":"runner-1","state":"started"}`))
				case r.Method == http.MethodGet && r.URL.Path == "/v1/runs/run-1/command-plans/project_test/run_declared_test_target":
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write(planJSON)
				case r.Method == http.MethodPost && r.URL.Path == "/v1/runs/run-1/receipts":
					receiptRequests.Add(1)
					http.Error(w, "unexpected receipt", http.StatusInternalServerError)
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()

			var logs bytes.Buffer
			runner, err := NewRunner(Config{
				ServerURL:       server.URL,
				BearerToken:     "runner-token",
				ProjectID:       "project-1",
				RepoBindingID:   "repo-1",
				RunnerID:        "runner-1",
				WorkspaceRef:    "mounted:/workspace/goalrail",
				WorkspaceRoot:   t.TempDir(),
				CommitSHA:       "abc123",
				ProjectTest:     true,
				PollInterval:    time.Millisecond,
				LeaseTTLSeconds: 900,
				Once:            true,
				LogWriter:       &logs,
			})
			if err != nil {
				t.Fatalf("NewRunner() error = %v", err)
			}
			_, err = runner.Step(context.Background())
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Step() error = %v, want %q trace rejection", err, tt.wantErr)
			}
			if receiptRequests.Load() != 0 {
				t.Fatalf("receipt requests = %d, want 0 after trace mismatch", receiptRequests.Load())
			}
			if strings.Contains(logs.String(), secretToken) {
				t.Fatalf("logs leaked execution lease token: %q", logs.String())
			}
		})
	}
}

func TestProjectTestRunnerHasNoProcessExecutionPrimitives(t *testing.T) {
	for _, file := range []string{"runner.go", "projecttest.go"} {
		source, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		for _, forbidden := range []string{
			`"os/exec"`,
			"exec.Command",
			"CommandContext(",
			".StdoutPipe(",
			".StderrPipe(",
			".CombinedOutput(",
		} {
			if bytes.Contains(source, []byte(forbidden)) {
				t.Fatalf("%s contains forbidden project-test execution primitive %q", file, forbidden)
			}
		}
	}
}

func TestStepCanResumeReceiptAfterTransientSubmitFailure(t *testing.T) {
	const firstSecretToken = "first-secret-execution-token"
	const secondSecretToken = "second-secret-execution-token"
	var leaseRequests atomic.Int32
	var runRequests atomic.Int32
	var receiptRequests atomic.Int32
	var receiptRequest executionReceiptRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/execution-jobs/leases":
			count := leaseRequests.Add(1)
			if count == 1 {
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"id":"lease-1","execution_job_id":"execution-job-1","task_id":"task-1","checkout_receipt_id":"receipt-1","repo_binding_id":"repo-1","runner_id":"runner-1","state":"active","lease_token":"` + firstSecretToken + `","expires_at":"2026-05-07T13:00:00Z","execution_job":{"id":"execution-job-1","task_id":"task-1","repo_binding_id":"repo-1","checkout_receipt_id":"receipt-1","state":"leased"}}`))
				return
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"lease-2","execution_job_id":"execution-job-1","task_id":"task-1","checkout_receipt_id":"receipt-1","repo_binding_id":"repo-1","runner_id":"runner-1","state":"active","lease_token":"` + secondSecretToken + `","expires_at":"2026-05-07T13:00:00Z","execution_job":{"id":"execution-job-1","task_id":"task-1","repo_binding_id":"repo-1","checkout_receipt_id":"receipt-1","state":"run_started"}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/execution-jobs/execution-job-1/runs":
			runRequests.Add(1)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"run-1","execution_job_id":"execution-job-1","execution_lease_id":"lease-1","task_id":"task-1","checkout_receipt_id":"receipt-1","runner_id":"runner-1","state":"started"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/runs/run-1/receipts":
			count := receiptRequests.Add(1)
			if count == 1 {
				w.WriteHeader(http.StatusBadGateway)
				_, _ = w.Write([]byte(`{"error":{"code":"temporary_unavailable","message":"temporary receipt failure"}}`))
				return
			}
			decodeStrict(t, r, &receiptRequest)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"execution-receipt-1","run_id":"run-1","execution_job_id":"execution-job-1","runner_id":"runner-1","execution_mode":"no_command","process_status":"not_executed"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	var logs bytes.Buffer
	runner, err := NewRunner(Config{
		ServerURL:       server.URL,
		BearerToken:     "runner-token",
		ProjectID:       "project-1",
		RepoBindingID:   "repo-1",
		RunnerID:        "runner-1",
		WorkspaceRef:    "mounted:/workspace/goalrail",
		CommitSHA:       "abc123",
		SubmitReceipt:   true,
		PollInterval:    time.Millisecond,
		LeaseTTLSeconds: 900,
		Once:            true,
		LogWriter:       &logs,
	})
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}
	if _, err := runner.Step(context.Background()); err == nil {
		t.Fatal("first Step() error = nil, want transient receipt submission error")
	}
	result, err := runner.Step(context.Background())
	if err != nil {
		t.Fatalf("second Step() error = %v", err)
	}
	if result != StepReceiptSubmitted {
		t.Fatalf("second Step() = %q, want %q", result, StepReceiptSubmitted)
	}
	if leaseRequests.Load() != 2 || runRequests.Load() != 2 || receiptRequests.Load() != 2 {
		t.Fatalf("requests lease/run/receipt = %d/%d/%d, want 2/2/2", leaseRequests.Load(), runRequests.Load(), receiptRequests.Load())
	}
	if receiptRequest.LeaseID != "lease-2" || receiptRequest.LeaseToken != secondSecretToken {
		t.Fatalf("recovered receipt proof = %#v, want fresh lease proof", receiptRequest)
	}
	if len(receiptRequest.ArtifactRefs) != 0 || len(receiptRequest.ChangedPathsSummary) != 0 || receiptRequest.ExecutionMode != "no_command" || receiptRequest.ProcessStatus != "not_executed" {
		t.Fatalf("recovered receipt = %#v, want no-command metadata-only receipt", receiptRequest)
	}
	if strings.Contains(logs.String(), firstSecretToken) || strings.Contains(logs.String(), secondSecretToken) {
		t.Fatalf("logs leaked execution lease token: %q", logs.String())
	}
}

func TestStepDoesNotRetryStaleExecutionLeaseProof(t *testing.T) {
	for _, tt := range []struct {
		name string
		code string
		want StepResult
	}{
		{name: "lease expired", code: "lease_expired", want: StepLeaseExpired},
		{name: "invalid lease", code: "invalid_lease", want: StepInvalidLease},
	} {
		t.Run(tt.name, func(t *testing.T) {
			const secretToken = "secret-execution-token"
			var runRequests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == http.MethodPost && r.URL.Path == "/v1/execution-jobs/leases":
					w.WriteHeader(http.StatusCreated)
					_, _ = w.Write([]byte(`{"id":"lease-1","execution_job_id":"execution-job-1","task_id":"task-1","checkout_receipt_id":"receipt-1","repo_binding_id":"repo-1","runner_id":"runner-1","state":"active","lease_token":"` + secretToken + `","expires_at":"2026-05-07T13:00:00Z","execution_job":{"id":"execution-job-1","task_id":"task-1","repo_binding_id":"repo-1","checkout_receipt_id":"receipt-1","state":"leased"}}`))
				case r.Method == http.MethodPost && r.URL.Path == "/v1/execution-jobs/execution-job-1/runs":
					runRequests.Add(1)
					w.WriteHeader(http.StatusConflict)
					_, _ = w.Write([]byte(`{"error":{"code":"` + tt.code + `","message":"lease rejected"}}`))
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()

			var logs bytes.Buffer
			runner := newTestRunner(t, server.URL, &logs)
			result, err := runner.Step(context.Background())
			if err != nil {
				t.Fatalf("Step() error = %v", err)
			}
			if result != tt.want {
				t.Fatalf("Step() = %q, want %q", result, tt.want)
			}
			if runRequests.Load() != 1 {
				t.Fatalf("run requests = %d, want exactly one stale attempt", runRequests.Load())
			}
			if strings.Contains(logs.String(), secretToken) {
				t.Fatalf("logs leaked execution lease token: %q", logs.String())
			}
		})
	}
}

func newTestRunner(t *testing.T, serverURL string, logs *bytes.Buffer) *Runner {
	t.Helper()
	runner, err := NewRunner(Config{
		ServerURL:       serverURL,
		BearerToken:     "runner-token",
		ProjectID:       "project-1",
		RepoBindingID:   "repo-1",
		RunnerID:        "runner-1",
		PollInterval:    time.Millisecond,
		LeaseTTLSeconds: 900,
		Once:            true,
		LogWriter:       logs,
	})
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}
	return runner
}

func decodeStrict(t *testing.T, r *http.Request, target any) {
	t.Helper()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		t.Fatalf("decode request %s %s: %v", r.Method, r.URL.Path, err)
	}
}

func hasManifest(metadata *projectProbeMetadata, path string) bool {
	for _, manifest := range metadata.DetectedManifests {
		if manifest.Path == path {
			return true
		}
	}
	return false
}

func hasTestTarget(metadata *projectProbeMetadata, name string) bool {
	for _, candidate := range metadata.DeclaredTestTargetCandidates {
		if candidate.Name == name {
			return true
		}
	}
	return false
}

func projectTestPlanFixture() executionCommandPlan {
	return executionCommandPlan{
		ID:                          "command-plan-1",
		ProjectID:                   "project-1",
		RepoBindingID:               "repo-1",
		TaskID:                      "task-1",
		CheckoutReceiptID:           "receipt-1",
		ExecutionJobID:              "execution-job-1",
		RunID:                       "run-1",
		CommandKind:                 "project_test",
		Action:                      "run_declared_test_target",
		SourceProjectProbeReceiptID: "probe-receipt-1",
		SelectedTargetID:            "package.json#package_json_script:test",
		DeclaredTestTarget: &projectProbeTestTargetCandidate{
			Name:       "test",
			SourcePath: "package.json",
			SourceKind: "package_json_script",
		},
		ShellAllowed:           false,
		Argv:                   []string{},
		WorkingDirectory:       ".",
		PathScope:              []string{"."},
		TimeoutSeconds:         120,
		NetworkAllowed:         false,
		WorkspaceWriteAllowed:  false,
		ScratchWriteAllowed:    false,
		MaxStdoutBytes:         0,
		MaxStderrBytes:         0,
		AllowedArtifactKinds:   []string{},
		ChangedPathsAllowed:    false,
		RawSourceUploadAllowed: false,
		State:                  "planned",
	}
}
