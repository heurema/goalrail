package checkoutrunner

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

const testBearerToken = "runner-api-token"

func TestRunOnceNoWorkExitsCleanly(t *testing.T) {
	t.Parallel()

	var leaseRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/checkout-jobs/leases" {
			assertBearer(t, r)
			leaseRequests.Add(1)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	var logs bytes.Buffer
	if err := Run(context.Background(), Config{
		ServerURL:       server.URL,
		BearerToken:     testBearerToken,
		RunnerID:        "runner-1",
		WorkspaceRef:    "mounted:/workspace/goalrail",
		CommitSHA:       "abc123",
		LeaseTTLSeconds: 900,
		Once:            true,
		LogWriter:       &logs,
	}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if leaseRequests.Load() != 1 {
		t.Fatalf("lease requests = %d, want 1", leaseRequests.Load())
	}
	if !strings.Contains(logs.String(), "no checkout work available") {
		t.Fatalf("logs = %q, want no-work message", logs.String())
	}
}

func TestRunTreatsSleepCancellationAsCleanShutdown(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	var leaseRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/checkout-jobs/leases" {
			assertBearer(t, r)
			leaseRequests.Add(1)
			w.WriteHeader(http.StatusNoContent)
			cancel()
			return
		}
		t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	if err := Run(ctx, Config{
		ServerURL:       server.URL,
		BearerToken:     testBearerToken,
		RunnerID:        "runner-1",
		WorkspaceRef:    "mounted:/workspace/goalrail",
		CommitSHA:       "abc123",
		PollInterval:    time.Hour,
		LeaseTTLSeconds: 900,
	}); err != nil {
		t.Fatalf("Run() error = %v, want clean shutdown", err)
	}
	if leaseRequests.Load() != 1 {
		t.Fatalf("lease requests = %d, want 1", leaseRequests.Load())
	}
}

func TestRunOnceAcquiresLeaseAndSubmitsReceipt(t *testing.T) {
	t.Parallel()

	const secretToken = "secret-checkout-token"
	var leaseRequest checkoutLeaseCreateRequest
	var receiptRequest checkoutReceiptSubmitRequest
	var receiptRequests atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/checkout-jobs/leases":
			assertBearer(t, r)
			decodeStrict(t, r, &leaseRequest)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"job_id":"job-1","task_id":"task-1","state":"leased","runner_id":"runner-1","lease_token":"` + secretToken + `","lease_expires_at":"2026-05-07T13:00:00Z","instruction":{"job_id":"job-1","task_id":"task-1","repo_binding_id":"repo-1","access_mode":"customer_mounted_workspace","provider":"github","repository_full_name":"heurema/goalrail","repository_url":"https://github.com/heurema/goalrail","workflow_base_branch":"main","path_scope":".","source_ref":{"kind":"work_item","id":"task-1"},"raw_source_uploaded":false},"created_at":"2026-05-07T12:45:00Z","updated_at":"2026-05-07T12:45:00Z"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/checkout-jobs/job-1/receipts":
			assertBearer(t, r)
			receiptRequests.Add(1)
			decodeStrict(t, r, &receiptRequest)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"receipt-1","job_id":"job-1","task_id":"task-1","repo_binding_id":"repo-1","runner_id":"runner-1","workspace_ref":"mounted:/workspace/goalrail","commit_sha":"abc123","raw_source_uploaded":false}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	var logs bytes.Buffer
	if err := Run(context.Background(), Config{
		ServerURL:       server.URL,
		BearerToken:     testBearerToken,
		RunnerID:        "runner-1",
		WorkspaceRef:    "mounted:/workspace/goalrail",
		CommitSHA:       "abc123",
		BaselineID:      "baseline-1",
		OverlayID:       "overlay-1",
		LeaseTTLSeconds: 900,
		Once:            true,
		LogWriter:       &logs,
	}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if leaseRequest.RunnerID != "runner-1" || leaseRequest.TTLSeconds != 900 {
		t.Fatalf("lease request = %#v, want runner and ttl", leaseRequest)
	}
	if receiptRequests.Load() != 1 {
		t.Fatalf("receipt requests = %d, want 1", receiptRequests.Load())
	}
	if receiptRequest.LeaseToken != secretToken || receiptRequest.RunnerID != "runner-1" {
		t.Fatalf("receipt lease proof = %#v, want runner lease proof", receiptRequest)
	}
	if receiptRequest.WorkspaceRef != "mounted:/workspace/goalrail" || receiptRequest.CommitSHA != "abc123" {
		t.Fatalf("receipt workspace = %#v, want mounted workspace metadata", receiptRequest)
	}
	if receiptRequest.RawSourceUploaded {
		t.Fatal("receipt request uploaded raw source")
	}
	if strings.Contains(logs.String(), secretToken) {
		t.Fatalf("logs leaked checkout lease token: %q", logs.String())
	}
}

func TestRunOnceLeaseConflictsDoNotRetryStaleReceipt(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name string
		code string
		want StepResult
	}{
		{name: "lease expired", code: "lease_expired", want: StepLeaseExpired},
		{name: "invalid lease", code: "invalid_lease", want: StepInvalidLease},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			const secretToken = "secret-checkout-token"
			var receiptRequests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method == http.MethodPost && r.URL.Path == "/v1/checkout-jobs/leases":
					assertBearer(t, r)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusCreated)
					_, _ = w.Write([]byte(`{"job_id":"job-1","task_id":"task-1","state":"leased","runner_id":"runner-1","lease_token":"` + secretToken + `","lease_expires_at":"2026-05-07T13:00:00Z","instruction":{"job_id":"job-1","task_id":"task-1","repo_binding_id":"repo-1","access_mode":"customer_mounted_workspace","provider":"github","repository_full_name":"heurema/goalrail","repository_url":"https://github.com/heurema/goalrail","workflow_base_branch":"main","path_scope":".","source_ref":{"kind":"work_item","id":"task-1"},"raw_source_uploaded":false},"created_at":"2026-05-07T12:45:00Z","updated_at":"2026-05-07T12:45:00Z"}`))
				case r.Method == http.MethodPost && r.URL.Path == "/v1/checkout-jobs/job-1/receipts":
					assertBearer(t, r)
					receiptRequests.Add(1)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusConflict)
					_, _ = w.Write([]byte(`{"error":{"code":"` + tt.code + `","message":"lease rejected"}}`))
				default:
					t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
				}
			}))
			defer server.Close()

			var logs bytes.Buffer
			runner, err := NewRunner(Config{
				ServerURL:       server.URL,
				BearerToken:     testBearerToken,
				RunnerID:        "runner-1",
				WorkspaceRef:    "mounted:/workspace/goalrail",
				CommitSHA:       "abc123",
				LeaseTTLSeconds: 900,
				Once:            true,
				LogWriter:       &logs,
			})
			if err != nil {
				t.Fatalf("NewRunner() error = %v", err)
			}
			got, err := runner.Step(context.Background())
			if err != nil {
				t.Fatalf("Step() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("Step() = %q, want %q", got, tt.want)
			}
			if receiptRequests.Load() != 1 {
				t.Fatalf("receipt requests = %d, want exactly one stale attempt", receiptRequests.Load())
			}
			if strings.Contains(logs.String(), secretToken) {
				t.Fatalf("logs leaked checkout lease token: %q", logs.String())
			}
		})
	}
}

func TestNewAPIClientUsesBoundedDefaultTimeout(t *testing.T) {
	t.Parallel()

	client, err := newAPIClient("http://goalrail.test", testBearerToken, nil)
	if err != nil {
		t.Fatalf("newAPIClient() error = %v", err)
	}
	if client.client == http.DefaultClient {
		t.Fatal("newAPIClient() reused http.DefaultClient")
	}
	if client.client.Timeout != defaultHTTPClientTimeout {
		t.Fatalf("client timeout = %s, want %s", client.client.Timeout, defaultHTTPClientTimeout)
	}
}

func assertBearer(t *testing.T, r *http.Request) {
	t.Helper()
	if got, want := r.Header.Get("Authorization"), "Bearer "+testBearerToken; got != want {
		t.Fatalf("Authorization = %q, want %q", got, want)
	}
}

func decodeStrict(t *testing.T, r *http.Request, target any) {
	t.Helper()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		t.Fatalf("decode request: %v", err)
	}
}
