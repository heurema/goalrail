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
