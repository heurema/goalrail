package executionrunner

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
