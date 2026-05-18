package httpserver_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/heurema/goalrail/apps/server/internal/execution"
	"github.com/heurema/goalrail/apps/server/internal/httpserver"
	"github.com/heurema/goalrail/apps/server/internal/spine"
)

func TestCreateRunnerCapabilityReportAcceptsSelfDeclaredUntrustedMetadata(t *testing.T) {
	service := &fakeExecutionCapabilityService{
		report: spine.RunnerCapabilityReport{
			ID:             "018f0000-0000-7000-8000-000000000701",
			RunnerID:       "runner-1",
			OrganizationID: repoBindingProfile().OrganizationMembership.OrganizationID,
			ProjectID:      "018f0000-0000-7000-8000-000000000003",
			RepoBindingID:  "018f0000-0000-7000-8000-000000000004",
			TrustState:     spine.RunnerCapabilityTrustSelfDeclaredUntrusted,
		},
	}
	handler := httpserver.NewExecutionHandler(fakeHTTPAuthService{profile: repoBindingProfile()}, service)
	response := doAuthRequest(t, http.HandlerFunc(handler.CreateRunnerCapabilityReport), http.MethodPost, "/v1/runner-capability-reports", `{
		"runner_id":"runner-1",
		"project_id":"018f0000-0000-7000-8000-000000000003",
		"repo_binding_id":"018f0000-0000-7000-8000-000000000004",
		"network_isolation_declared":false,
		"workspace_write_isolation_declared":false,
		"process_tree_control_declared":false,
		"stdout_stderr_policy_declared":false,
		"artifact_policy_declared":false,
		"trust_state":"self_declared_untrusted"
	}`, "Bearer access-token")

	if response.code != http.StatusCreated {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusCreated, response.body)
	}
	if service.input.RunnerID != "runner-1" || service.input.TrustState != spine.RunnerCapabilityTrustSelfDeclaredUntrusted {
		t.Fatalf("service input = %#v, want self-declared runner report", service.input)
	}
	if service.membership.OrganizationID != repoBindingProfile().OrganizationMembership.OrganizationID {
		t.Fatalf("membership org = %q, want auth profile org", service.membership.OrganizationID)
	}
	var body spine.RunnerCapabilityReport
	decodeJSON(t, response.body, &body)
	if body.ID == "" || body.TrustState != spine.RunnerCapabilityTrustSelfDeclaredUntrusted {
		t.Fatalf("response = %#v, want persisted untrusted report", body)
	}
}

func TestCreateRunnerCapabilityReportRejectsTrustedClaims(t *testing.T) {
	handler := httpserver.NewExecutionHandler(fakeHTTPAuthService{profile: repoBindingProfile()}, &fakeExecutionCapabilityService{
		err: &execution.ValidationError{Field: "trust_state", Message: "must be self_declared_untrusted"},
	})
	response := doAuthRequest(t, http.HandlerFunc(handler.CreateRunnerCapabilityReport), http.MethodPost, "/v1/runner-capability-reports", `{
		"runner_id":"runner-1",
		"project_id":"018f0000-0000-7000-8000-000000000003",
		"repo_binding_id":"018f0000-0000-7000-8000-000000000004",
		"trust_state":"trusted"
	}`, "Bearer access-token")

	if response.code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusBadRequest, response.body)
	}
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeJSON(t, response.body, &body)
	if body.Error.Code != "validation_failed" {
		t.Fatalf("error code = %q, want validation_failed", body.Error.Code)
	}
}

func TestCreateRunnerCapabilityReportRejectsUnknownActiveControlFields(t *testing.T) {
	handler := httpserver.NewExecutionHandler(fakeHTTPAuthService{profile: repoBindingProfile()}, &fakeExecutionCapabilityService{})
	response := doAuthRequest(t, http.HandlerFunc(handler.CreateRunnerCapabilityReport), http.MethodPost, "/v1/runner-capability-reports", `{
		"runner_id":"runner-1",
		"project_id":"018f0000-0000-7000-8000-000000000003",
		"repo_binding_id":"018f0000-0000-7000-8000-000000000004",
		"trust_state":"self_declared_untrusted",
		"network_isolation_active":true
	}`, "Bearer access-token")

	if response.code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d: %s", response.code, http.StatusBadRequest, response.body)
	}
	if !strings.Contains(response.body, "invalid_json") {
		t.Fatalf("response body = %s, want invalid_json for unknown active field", response.body)
	}
}

type fakeExecutionCapabilityService struct {
	report     spine.RunnerCapabilityReport
	input      spine.RunnerCapabilityReportCreateRequest
	membership spine.OrganizationMembership
	err        error
}

func (s *fakeExecutionCapabilityService) CreateOrReturnJob(context.Context, spine.WorkItemID, spine.ExecutionJobCreateRequest, spine.OrganizationMembership) (spine.ExecutionJob, bool, error) {
	return spine.ExecutionJob{}, false, nil
}

func (s *fakeExecutionCapabilityService) AcquireNextLease(context.Context, spine.ExecutionJobLeaseCreateRequest, spine.OrganizationMembership) (spine.ExecutionJobLeaseCreated, bool, error) {
	return spine.ExecutionJobLeaseCreated{}, false, nil
}

func (s *fakeExecutionCapabilityService) StartRun(context.Context, spine.ExecutionJobID, spine.RunStartRequest, spine.OrganizationMembership) (spine.Run, bool, error) {
	return spine.Run{}, false, nil
}

func (s *fakeExecutionCapabilityService) CreateOrReturnCommandPlan(context.Context, spine.RunID, spine.ExecutionCommandPlanCreateRequest, spine.OrganizationMembership) (spine.ExecutionCommandPlan, bool, error) {
	return spine.ExecutionCommandPlan{}, false, nil
}

func (s *fakeExecutionCapabilityService) GetCommandPlan(context.Context, spine.RunID, string, string, spine.OrganizationMembership) (spine.ExecutionCommandPlan, error) {
	return spine.ExecutionCommandPlan{}, nil
}

func (s *fakeExecutionCapabilityService) SubmitReceipt(context.Context, spine.RunID, spine.ExecutionReceiptSubmitRequest, spine.OrganizationMembership) (spine.ExecutionReceipt, bool, error) {
	return spine.ExecutionReceipt{}, false, nil
}

func (s *fakeExecutionCapabilityService) CreateRunnerCapabilityReport(_ context.Context, input spine.RunnerCapabilityReportCreateRequest, membership spine.OrganizationMembership) (spine.RunnerCapabilityReport, error) {
	s.input = input
	s.membership = membership
	if s.err != nil {
		return spine.RunnerCapabilityReport{}, s.err
	}
	return s.report, nil
}
