package execution

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

func TestCreateRunnerCapabilityReportPersistsSelfDeclaredUntrustedMetadata(t *testing.T) {
	now := time.Date(2026, 5, 9, 16, 30, 0, 0, time.UTC)
	reports := &fakeRunnerCapabilityReportStore{}
	events := &fakeEventLog{}
	service := &Service{
		RepoBindings:       fakeRepoBindingReader{binding: activeRepoBinding()},
		RunnerCapabilities: reports,
		Events:             events,
		TxRunner:           fakeTransactionRunner{},
		Clock:              fixedClock{now: now},
		IDs:                &runnerCapabilityIDs{},
	}

	report, err := service.CreateRunnerCapabilityReport(context.Background(), spine.RunnerCapabilityReportCreateRequest{
		RunnerID:                        "runner-1",
		ProjectID:                       "018f0000-0000-7000-8000-000000000003",
		RepoBindingID:                   "018f0000-0000-7000-8000-000000000004",
		NetworkIsolationDeclared:        true,
		WorkspaceWriteIsolationDeclared: true,
		ProcessTreeControlDeclared:      true,
		StdoutStderrPolicyDeclared:      true,
		ArtifactPolicyDeclared:          true,
		TrustState:                      spine.RunnerCapabilityTrustSelfDeclaredUntrusted,
	}, activeMembership())
	if err != nil {
		t.Fatalf("CreateRunnerCapabilityReport() error = %v", err)
	}
	if report.ID != "018f0000-0000-7000-8000-000000000701" {
		t.Fatalf("report ID = %q, want generated id", report.ID)
	}
	if report.OrganizationID != activeMembership().OrganizationID || report.ProjectID != "018f0000-0000-7000-8000-000000000003" || report.RepoBindingID != "018f0000-0000-7000-8000-000000000004" {
		t.Fatalf("report scope = %#v, want membership organization plus project/repo scope", report)
	}
	if report.TrustState != spine.RunnerCapabilityTrustSelfDeclaredUntrusted {
		t.Fatalf("trust_state = %q, want self_declared_untrusted", report.TrustState)
	}
	if !report.NetworkIsolationDeclared || !report.WorkspaceWriteIsolationDeclared || !report.ProcessTreeControlDeclared || !report.StdoutStderrPolicyDeclared || !report.ArtifactPolicyDeclared {
		t.Fatalf("declared capability metadata was not preserved: %#v", report)
	}
	if len(reports.created) != 1 || reports.created[0] != report {
		t.Fatalf("persisted reports = %#v, want created report", reports.created)
	}
	if len(events.events) != 1 || events.events[0].Type != EventTypeRunnerCapabilityReported || events.events[0].EntityType != EntityTypeRunnerCapability {
		t.Fatalf("events = %#v, want runner capability event", events.events)
	}
	var payload map[string]string
	if err := json.Unmarshal(events.events[0].Payload, &payload); err != nil {
		t.Fatalf("decode event payload: %v", err)
	}
	if payload["trust_state"] != spine.RunnerCapabilityTrustSelfDeclaredUntrusted || payload["runner_id"] != "runner-1" {
		t.Fatalf("event payload = %#v, want untrusted runner metadata", payload)
	}
}

func TestCreateRunnerCapabilityReportRejectsUnsafeClaims(t *testing.T) {
	base := spine.RunnerCapabilityReportCreateRequest{
		RunnerID:      "runner-1",
		ProjectID:     "018f0000-0000-7000-8000-000000000003",
		RepoBindingID: "018f0000-0000-7000-8000-000000000004",
		TrustState:    spine.RunnerCapabilityTrustSelfDeclaredUntrusted,
	}
	for _, tt := range []struct {
		name    string
		mutate  func(*spine.RunnerCapabilityReportCreateRequest, *spine.OrganizationMembership, *fakeRepoBindingReader)
		wantErr string
	}{
		{
			name: "trusted state",
			mutate: func(input *spine.RunnerCapabilityReportCreateRequest, _ *spine.OrganizationMembership, _ *fakeRepoBindingReader) {
				input.TrustState = "trusted"
			},
			wantErr: "trust_state",
		},
		{
			name: "active network enforcement",
			mutate: func(input *spine.RunnerCapabilityReportCreateRequest, _ *spine.OrganizationMembership, _ *fakeRepoBindingReader) {
				input.NetworkEnforcement = json.RawMessage(`"active"`)
			},
			wantErr: "network_enforcement",
		},
		{
			name: "attestation",
			mutate: func(input *spine.RunnerCapabilityReportCreateRequest, _ *spine.OrganizationMembership, _ *fakeRepoBindingReader) {
				input.Attestation = json.RawMessage(`{"kind":"local"}`)
			},
			wantErr: "attestation",
		},
		{
			name: "missing runner",
			mutate: func(input *spine.RunnerCapabilityReportCreateRequest, _ *spine.OrganizationMembership, _ *fakeRepoBindingReader) {
				input.RunnerID = ""
			},
			wantErr: "runner_id",
		},
		{
			name: "organization mismatch",
			mutate: func(input *spine.RunnerCapabilityReportCreateRequest, _ *spine.OrganizationMembership, _ *fakeRepoBindingReader) {
				input.OrganizationID = "018f0000-0000-7000-8000-000000000099"
			},
			wantErr: "not allowed",
		},
		{
			name: "project mismatch",
			mutate: func(input *spine.RunnerCapabilityReportCreateRequest, _ *spine.OrganizationMembership, reader *fakeRepoBindingReader) {
				reader.binding.ProjectID = "018f0000-0000-7000-8000-000000000099"
			},
			wantErr: "project",
		},
		{
			name: "inactive membership",
			mutate: func(_ *spine.RunnerCapabilityReportCreateRequest, membership *spine.OrganizationMembership, _ *fakeRepoBindingReader) {
				membership.State = spine.EntityStateInactive
			},
			wantErr: "membership",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			input := base
			membership := activeMembership()
			reader := &fakeRepoBindingReader{binding: activeRepoBinding()}
			tt.mutate(&input, &membership, reader)
			reports := &fakeRunnerCapabilityReportStore{}
			service := &Service{
				RepoBindings:       reader,
				RunnerCapabilities: reports,
				Events:             &fakeEventLog{},
				TxRunner:           fakeTransactionRunner{},
				Clock:              fixedClock{now: time.Date(2026, 5, 9, 16, 30, 0, 0, time.UTC)},
				IDs:                &runnerCapabilityIDs{},
			}
			_, err := service.CreateRunnerCapabilityReport(context.Background(), input, membership)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("CreateRunnerCapabilityReport() error = %v, want %q", err, tt.wantErr)
			}
			if len(reports.created) != 0 {
				t.Fatalf("created reports = %#v, want none after rejection", reports.created)
			}
		})
	}
}

type fakeRunnerCapabilityReportStore struct {
	created []spine.RunnerCapabilityReport
}

func (s *fakeRunnerCapabilityReportStore) Create(_ context.Context, report spine.RunnerCapabilityReport) error {
	s.created = append(s.created, report)
	return nil
}

type fakeRepoBindingReader struct {
	binding spine.RepoBinding
	err     error
}

func (r fakeRepoBindingReader) GetRepoBinding(context.Context, spine.RepoBindingID) (spine.RepoBinding, bool, error) {
	if r.err != nil {
		return spine.RepoBinding{}, false, r.err
	}
	if r.binding.ID == "" {
		return spine.RepoBinding{}, false, nil
	}
	return r.binding, true, nil
}

type fakeEventLog struct {
	events []spine.Event
}

func (l *fakeEventLog) Append(_ context.Context, event spine.Event) error {
	l.events = append(l.events, event)
	return nil
}

type fakeTransactionRunner struct{}

func (fakeTransactionRunner) RunReadCommitted(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

type runnerCapabilityIDs struct{}

func (*runnerCapabilityIDs) NewExecutionJobID() (spine.ExecutionJobID, error) {
	return "", errors.New("unexpected execution job id")
}

func (*runnerCapabilityIDs) NewExecutionLeaseID() (spine.ExecutionLeaseID, error) {
	return "", errors.New("unexpected execution lease id")
}

func (*runnerCapabilityIDs) NewRunID() (spine.RunID, error) {
	return "", errors.New("unexpected run id")
}

func (*runnerCapabilityIDs) NewExecutionCommandPlanID() (spine.ExecutionCommandPlanID, error) {
	return "", errors.New("unexpected command plan id")
}

func (*runnerCapabilityIDs) NewExecutionReceiptID() (spine.ExecutionReceiptID, error) {
	return "", errors.New("unexpected execution receipt id")
}

func (*runnerCapabilityIDs) NewRunnerCapabilityReportID() (spine.RunnerCapabilityReportID, error) {
	return "018f0000-0000-7000-8000-000000000701", nil
}

func (*runnerCapabilityIDs) NewEventID() (spine.EventID, error) {
	return "018f0000-0000-7000-8000-000000000702", nil
}

func activeMembership() spine.OrganizationMembership {
	return spine.OrganizationMembership{
		ID:             "018f0000-0000-7000-8000-000000000005",
		OrganizationID: "018f0000-0000-7000-8000-000000000002",
		UserID:         "018f0000-0000-7000-8000-000000000001",
		Role:           spine.OrganizationMembershipRoleMember,
		State:          spine.EntityStateActive,
	}
}

func activeRepoBinding() spine.RepoBinding {
	return spine.RepoBinding{
		ID:             "018f0000-0000-7000-8000-000000000004",
		OrganizationID: "018f0000-0000-7000-8000-000000000002",
		ProjectID:      "018f0000-0000-7000-8000-000000000003",
		State:          spine.EntityStateActive,
	}
}
