package planningworker

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunOnceNoWorkExitsCleanly(t *testing.T) {
	t.Parallel()

	var leaseRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/plans/leases" {
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
		WorkerID:        "planner-worker-1",
		LeaseTTLSeconds: 900,
		Once:            true,
		LogWriter:       &logs,
	}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if leaseRequests.Load() != 1 {
		t.Fatalf("lease requests = %d, want 1", leaseRequests.Load())
	}
	if !strings.Contains(logs.String(), "no planning work available") {
		t.Fatalf("logs = %q, want no-work message", logs.String())
	}
}

func TestRunTreatsSleepCancellationAsCleanShutdown(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	var leaseRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/plans/leases" {
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
		WorkerID:        "planner-worker-1",
		PollInterval:    time.Hour,
		LeaseTTLSeconds: 900,
	}); err != nil {
		t.Fatalf("Run() error = %v, want clean shutdown", err)
	}
	if leaseRequests.Load() != 1 {
		t.Fatalf("lease requests = %d, want 1", leaseRequests.Load())
	}
}

func TestRunOnceAcquiresLeaseAndSubmitsProposal(t *testing.T) {
	t.Parallel()

	const secretToken = "secret-lease-token"
	var leaseRequest leaseCreateRequest
	var proposalRequest proposalSubmitRequest
	var proposalRequests atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/plans/leases":
			decodeStrict(t, r, &leaseRequest)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"lease-1","plan_id":"plan-1","contract_id":"contract-1","approved_contract_id":"approved-1","repo_binding_id":"repo-1","state":"active","lease_token":"` + secretToken + `","expires_at":"2026-05-07T13:00:00Z","created_at":"2026-05-07T12:45:00Z","updated_at":"2026-05-07T12:45:00Z"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/plans/plan-1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"plan-1","contract_id":"contract-1","approved_contract_id":"approved-1","repo_binding_id":"repo-1","state":"leased","current_lease_id":"lease-1","created_at":"2026-05-07T12:44:00Z","updated_at":"2026-05-07T12:45:00Z"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/plans/plan-1/proposals":
			proposalRequests.Add(1)
			decodeStrict(t, r, &proposalRequest)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"proposal-1","plan_id":"plan-1","contract_id":"contract-1","approved_contract_id":"approved-1","repo_binding_id":"repo-1","state":"submitted","submitted_by":{"kind":"worker","id":"planner-worker-1"},"planner":{"kind":"goalrail_worker","mode":"minimal_dev"},"source_snapshot_refs":[{"kind":"approved_contract","id":"approved-1"}],"rationale":"ok","proposed_tasks":[{"title":"Implement approved contract","summary":"Implement","scope":["Implement"],"acceptance_refs":["acceptance_criteria[0]"],"proof_expectation_refs":["proof_expectations[0]"],"source_refs":[{"kind":"approved_contract","id":"approved-1"}]}],"created_at":"2026-05-07T12:46:00Z","updated_at":"2026-05-07T12:46:00Z"}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	var logs bytes.Buffer
	if err := Run(context.Background(), Config{
		ServerURL:       server.URL,
		WorkerID:        "planner-worker-1",
		LeaseTTLSeconds: 900,
		Once:            true,
		LogWriter:       &logs,
	}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if leaseRequest.LeasedBy.Kind != "worker" || leaseRequest.LeasedBy.ID != "planner-worker-1" || leaseRequest.TTLSeconds != 900 {
		t.Fatalf("lease request = %#v, want worker actor and ttl", leaseRequest)
	}
	if proposalRequests.Load() != 1 {
		t.Fatalf("proposal requests = %d, want 1", proposalRequests.Load())
	}
	if proposalRequest.LeaseID != "lease-1" || proposalRequest.LeaseToken != secretToken {
		t.Fatalf("proposal lease proof was not forwarded correctly")
	}
	if proposalRequest.SubmittedBy.Kind != "worker" || proposalRequest.SubmittedBy.ID != "planner-worker-1" {
		t.Fatalf("submitted_by = %#v, want worker actor", proposalRequest.SubmittedBy)
	}
	if len(proposalRequest.ProposedTasks) != 1 {
		t.Fatalf("proposed tasks = %d, want 1", len(proposalRequest.ProposedTasks))
	}
	if len(proposalRequest.ProposedTasks[0].AcceptanceRefs) == 0 || len(proposalRequest.ProposedTasks[0].ProofExpectationRefs) == 0 {
		t.Fatalf("proposal task refs = %#v, want acceptance and proof refs", proposalRequest.ProposedTasks[0])
	}
	if strings.Contains(logs.String(), secretToken) {
		t.Fatalf("logs leaked lease token: %q", logs.String())
	}
}

func TestRunOnceLeaseConflictsDoNotRetryStaleProposal(t *testing.T) {
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

			const secretToken = "secret-lease-token"
			var proposalRequests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method == http.MethodPost && r.URL.Path == "/v1/plans/leases":
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusCreated)
					_, _ = w.Write([]byte(`{"id":"lease-1","plan_id":"plan-1","contract_id":"contract-1","approved_contract_id":"approved-1","repo_binding_id":"repo-1","state":"active","lease_token":"` + secretToken + `","expires_at":"2026-05-07T13:00:00Z","created_at":"2026-05-07T12:45:00Z","updated_at":"2026-05-07T12:45:00Z"}`))
				case r.Method == http.MethodGet && r.URL.Path == "/v1/plans/plan-1":
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"id":"plan-1","contract_id":"contract-1","approved_contract_id":"approved-1","repo_binding_id":"repo-1","state":"leased","created_at":"2026-05-07T12:44:00Z","updated_at":"2026-05-07T12:45:00Z"}`))
				case r.Method == http.MethodPost && r.URL.Path == "/v1/plans/plan-1/proposals":
					proposalRequests.Add(1)
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
				WorkerID:        "planner-worker-1",
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
			if proposalRequests.Load() != 1 {
				t.Fatalf("proposal requests = %d, want exactly one stale attempt", proposalRequests.Load())
			}
			if strings.Contains(logs.String(), secretToken) {
				t.Fatalf("logs leaked lease token: %q", logs.String())
			}
		})
	}
}

func TestRunOnceUnsupportedPlannerInputDoesNotSubmitMalformedProposal(t *testing.T) {
	t.Parallel()

	var proposalRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/plans/leases":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"lease-1","plan_id":"plan-1","contract_id":"contract-1","approved_contract_id":"approved-1","repo_binding_id":"repo-1","state":"active","lease_token":"secret-lease-token","expires_at":"2026-05-07T13:00:00Z","created_at":"2026-05-07T12:45:00Z","updated_at":"2026-05-07T12:45:00Z"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v1/plans/plan-1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"plan-1","contract_id":"contract-1","approved_contract_id":"","repo_binding_id":"repo-1","state":"leased","created_at":"2026-05-07T12:44:00Z","updated_at":"2026-05-07T12:45:00Z"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/plans/plan-1/proposals":
			proposalRequests.Add(1)
			http.Error(w, "malformed proposal should not be submitted", http.StatusInternalServerError)
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	var logs bytes.Buffer
	runner, err := NewRunner(Config{
		ServerURL:       server.URL,
		WorkerID:        "planner-worker-1",
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
	if got != StepUnsupported {
		t.Fatalf("Step() = %q, want %q", got, StepUnsupported)
	}
	if proposalRequests.Load() != 0 {
		t.Fatalf("proposal requests = %d, want 0", proposalRequests.Load())
	}
	if !strings.Contains(logs.String(), "unsupported planner input") {
		t.Fatalf("logs = %q, want unsupported input message", logs.String())
	}
}

func TestWorkerDoesNotImportServerStoresPostgresOrExecution(t *testing.T) {
	t.Parallel()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	moduleRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	forbidden := []string{
		"github.com/heurema/goalrail/apps/server/internal",
		"database/sql",
		"github.com/jackc/pgx",
		"os/exec",
	}
	err := filepath.WalkDir(moduleRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, item := range forbidden {
			if strings.Contains(string(body), item) {
				t.Fatalf("%s contains forbidden worker dependency %q", path, item)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk worker module: %v", err)
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
