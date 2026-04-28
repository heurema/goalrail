package approvedcontract_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/actor"
	"github.com/heurema/goalrail/apps/server/internal/approvedcontract"
	"github.com/heurema/goalrail/apps/server/internal/eventlog"
	"github.com/heurema/goalrail/apps/server/internal/spine"
	"github.com/heurema/goalrail/apps/server/internal/store"
)

func TestServiceApprovesReadyContractDraft(t *testing.T) {
	service, drafts, approvedStore, _ := approvalService(t)
	draft := validReadyDraft()
	if err := drafts.Create(context.Background(), draft); err != nil {
		t.Fatalf("drafts.Create() error = %v", err)
	}

	approved, err := service.ApproveDraft(context.Background(), draft.ID, approveRequest())
	if err != nil {
		t.Fatalf("ApproveDraft() error = %v", err)
	}

	if approved.State != spine.ApprovedContractStateApproved {
		t.Fatalf("state = %q, want %q", approved.State, spine.ApprovedContractStateApproved)
	}
	if approved.ContractDraftID != draft.ID {
		t.Fatalf("contract_draft_id = %q, want %q", approved.ContractDraftID, draft.ID)
	}
	if approved.ContractSeedID != draft.ContractSeedID || approved.GoalID != draft.GoalID || approved.RepoBindingID != draft.RepoBindingID {
		t.Fatalf("source ids = %q/%q/%q, want draft ids", approved.ContractSeedID, approved.GoalID, approved.RepoBindingID)
	}
	if approved.OrganizationID != draft.OrganizationID || approved.ProjectID != draft.ProjectID {
		t.Fatalf("context = %q/%q, want draft context %q/%q", approved.OrganizationID, approved.ProjectID, draft.OrganizationID, draft.ProjectID)
	}
	if approved.Title != draft.Title || approved.IntentSummary != draft.IntentSummary {
		t.Fatalf("title/summary not copied from draft")
	}
	if !reflect.DeepEqual(approved.Scope, draft.ProposedScope) {
		t.Fatalf("scope = %#v, want %#v", approved.Scope, draft.ProposedScope)
	}
	if !reflect.DeepEqual(approved.AcceptanceCriteria, draft.ProposedAcceptanceCriteria) {
		t.Fatalf("acceptance_criteria = %#v, want %#v", approved.AcceptanceCriteria, draft.ProposedAcceptanceCriteria)
	}
	if !reflect.DeepEqual(approved.ProofExpectations, draft.ProposedProofExpectations) {
		t.Fatalf("proof_expectations = %#v, want %#v", approved.ProofExpectations, draft.ProposedProofExpectations)
	}
	if !hasSourceRef(approved.SourceRefs, approvedcontract.SourceRefKindContractDraft, string(draft.ID)) {
		t.Fatalf("source_refs = %#v, want contract_draft ref", approved.SourceRefs)
	}
	if !hasSourceRef(approved.SourceRefs, "contract_seed", string(draft.ContractSeedID)) {
		t.Fatalf("source_refs = %#v, want preserved contract_seed ref", approved.SourceRefs)
	}

	stored, ok, err := approvedStore.GetByContractDraftID(context.Background(), draft.ID)
	if err != nil {
		t.Fatalf("approved.GetByContractDraftID() error = %v", err)
	}
	if !ok {
		t.Fatal("approved.GetByContractDraftID() ok = false, want true")
	}
	if stored.ID != approved.ID {
		t.Fatalf("stored id = %q, want %q", stored.ID, approved.ID)
	}

	storedDraft, ok, err := drafts.Get(context.Background(), draft.ID)
	if err != nil {
		t.Fatalf("drafts.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("drafts.Get() ok = false, want true")
	}
	if !reflect.DeepEqual(storedDraft, draft) {
		t.Fatalf("stored draft mutated: %#v want %#v", storedDraft, draft)
	}
}

func TestServiceAppendsContractApprovedEvent(t *testing.T) {
	service, drafts, _, events := approvalService(t)
	draft := validReadyDraft()
	if err := drafts.Create(context.Background(), draft); err != nil {
		t.Fatalf("drafts.Create() error = %v", err)
	}

	approved, err := service.ApproveDraft(context.Background(), draft.ID, approveRequest())
	if err != nil {
		t.Fatalf("ApproveDraft() error = %v", err)
	}

	appended := events.Events()
	if got := countEventType(appended, approvedcontract.EventTypeContractApproved); got != 1 {
		t.Fatalf("contract.approved events = %d, want 1", got)
	}
	event := appended[len(appended)-1]
	if event.EntityType != approvedcontract.EntityTypeApprovedContract {
		t.Fatalf("entity type = %q, want %q", event.EntityType, approvedcontract.EntityTypeApprovedContract)
	}
	if event.EntityID != string(approved.ID) {
		t.Fatalf("entity id = %q, want %q", event.EntityID, approved.ID)
	}
	if event.OrganizationID != draft.OrganizationID || event.ProjectID != draft.ProjectID || event.RepoBindingID != draft.RepoBindingID {
		t.Fatalf("event context = %q/%q/%q, want draft context %q/%q/%q", event.OrganizationID, event.ProjectID, event.RepoBindingID, draft.OrganizationID, draft.ProjectID, draft.RepoBindingID)
	}

	var payload struct {
		ApprovedContractID spine.ApprovedContractID `json:"approved_contract_id"`
		ContractDraftID    spine.ContractDraftID    `json:"contract_draft_id"`
		ContractSeedID     spine.ContractSeedID     `json:"contract_seed_id"`
		GoalID             spine.GoalID             `json:"goal_id"`
		ApprovedBy         spine.ActorRef           `json:"approved_by"`
		ApprovedAt         time.Time                `json:"approved_at"`
		SourceRefs         []spine.SourceRef        `json:"source_refs"`
		PreviousDraftState spine.ContractDraftState `json:"previous_draft_state"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatalf("unmarshal contract.approved payload: %v", err)
	}
	if payload.ApprovedContractID != approved.ID || payload.ContractDraftID != draft.ID || payload.ContractSeedID != draft.ContractSeedID || payload.GoalID != draft.GoalID {
		t.Fatalf("payload ids = %q/%q/%q/%q, want approved/draft/source ids", payload.ApprovedContractID, payload.ContractDraftID, payload.ContractSeedID, payload.GoalID)
	}
	if payload.ApprovedBy.Kind != "user" || payload.ApprovedBy.ID != "dev_approver" {
		t.Fatalf("approved_by = %#v, want approver", payload.ApprovedBy)
	}
	if !payload.ApprovedAt.Equal(testTime()) {
		t.Fatalf("approved_at = %s, want %s", payload.ApprovedAt, testTime())
	}
	if payload.PreviousDraftState != spine.ContractDraftStateReadyForApproval {
		t.Fatalf("previous_draft_state = %q, want ready_for_approval", payload.PreviousDraftState)
	}
	if !hasSourceRef(payload.SourceRefs, approvedcontract.SourceRefKindContractDraft, string(draft.ID)) {
		t.Fatalf("payload source_refs = %#v, want contract_draft ref", payload.SourceRefs)
	}
}

func TestServiceRejectsDuplicateApproval(t *testing.T) {
	service, drafts, _, events := approvalService(t)
	draft := validReadyDraft()
	if err := drafts.Create(context.Background(), draft); err != nil {
		t.Fatalf("drafts.Create() error = %v", err)
	}
	if _, err := service.ApproveDraft(context.Background(), draft.ID, approveRequest()); err != nil {
		t.Fatalf("first ApproveDraft() error = %v", err)
	}

	_, err := service.ApproveDraft(context.Background(), draft.ID, approveRequest())
	if !errors.Is(err, approvedcontract.ErrAlreadyApproved) {
		t.Fatalf("second ApproveDraft() error = %v, want ErrAlreadyApproved", err)
	}
	if got := countEventType(events.Events(), approvedcontract.EventTypeContractApproved); got != 1 {
		t.Fatalf("contract.approved events = %d, want 1", got)
	}
}

func TestServiceRejectsDraftNotReadyForApproval(t *testing.T) {
	service, drafts, _, events := approvalService(t)
	draft := validReadyDraft()
	draft.State = spine.ContractDraftStateDraft
	if err := drafts.Create(context.Background(), draft); err != nil {
		t.Fatalf("drafts.Create() error = %v", err)
	}

	_, err := service.ApproveDraft(context.Background(), draft.ID, approveRequest())
	if !errors.Is(err, approvedcontract.ErrInvalidDraftState) {
		t.Fatalf("ApproveDraft() error = %v, want ErrInvalidDraftState", err)
	}
	if got := len(events.Events()); got != 0 {
		t.Fatalf("events = %d, want 0", got)
	}
}

func TestServiceValidatesApprovedBy(t *testing.T) {
	tests := []struct {
		name  string
		actor spine.ActorRef
		field string
	}{
		{name: "missing_kind", actor: spine.ActorRef{ID: "dev_approver"}, field: "approved_by.kind"},
		{name: "missing_id", actor: spine.ActorRef{Kind: "user"}, field: "approved_by.id"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, drafts, _, events := approvalService(t)
			draft := validReadyDraft()
			if err := drafts.Create(context.Background(), draft); err != nil {
				t.Fatalf("drafts.Create() error = %v", err)
			}

			_, err := service.ApproveDraft(context.Background(), draft.ID, spine.ApproveContractDraftRequest{ApprovedBy: tt.actor})
			var validationErr *approvedcontract.ValidationError
			if !errors.As(err, &validationErr) {
				t.Fatalf("ApproveDraft() error = %v, want ValidationError", err)
			}
			if validationErr.Field != tt.field {
				t.Fatalf("validation field = %q, want %q", validationErr.Field, tt.field)
			}
			if got := len(events.Events()); got != 0 {
				t.Fatalf("events = %d, want 0", got)
			}
		})
	}
}

func TestServiceUsesActorContextWhenPresent(t *testing.T) {
	service, drafts, _, events := approvalService(t)
	draft := validReadyDraft()
	if err := drafts.Create(context.Background(), draft); err != nil {
		t.Fatalf("drafts.Create() error = %v", err)
	}

	ctxActor := spine.ActorRef{Kind: "user", ID: "ctx_approver", DisplayName: "Context Approver"}
	ctx := actor.WithActor(context.Background(), actor.ActorContext{
		Actor:  ctxActor,
		Source: actor.SourceDevHeader,
	})

	approved, err := service.ApproveDraft(ctx, draft.ID, approveRequest())
	if err != nil {
		t.Fatalf("ApproveDraft() error = %v", err)
	}
	if approved.ApprovedBy != ctxActor {
		t.Fatalf("approved.ApprovedBy = %#v, want ctxActor %#v", approved.ApprovedBy, ctxActor)
	}

	appended := events.Events()
	if got := countEventType(appended, approvedcontract.EventTypeContractApproved); got != 1 {
		t.Fatalf("contract.approved events = %d, want 1", got)
	}
	var payload struct {
		ApprovedBy spine.ActorRef `json:"approved_by"`
	}
	if err := json.Unmarshal(appended[len(appended)-1].Payload, &payload); err != nil {
		t.Fatalf("unmarshal contract.approved payload: %v", err)
	}
	if payload.ApprovedBy != ctxActor {
		t.Fatalf("event approved_by = %#v, want ctxActor %#v", payload.ApprovedBy, ctxActor)
	}
}

func TestServiceFallsBackToPayloadActorWhenContextAbsent(t *testing.T) {
	service, drafts, _, _ := approvalService(t)
	draft := validReadyDraft()
	if err := drafts.Create(context.Background(), draft); err != nil {
		t.Fatalf("drafts.Create() error = %v", err)
	}

	approved, err := service.ApproveDraft(context.Background(), draft.ID, approveRequest())
	if err != nil {
		t.Fatalf("ApproveDraft() error = %v", err)
	}
	want := approveRequest().ApprovedBy
	if approved.ApprovedBy != want {
		t.Fatalf("approved.ApprovedBy = %#v, want payload %#v", approved.ApprovedBy, want)
	}
}

func TestServiceValidatesEffectiveApproverFromContext(t *testing.T) {
	tests := []struct {
		name      string
		ctxActor  spine.ActorRef
		wantField string
	}{
		{name: "ctx_missing_kind", ctxActor: spine.ActorRef{ID: "ctx_approver"}, wantField: "approved_by.kind"},
		{name: "ctx_missing_id", ctxActor: spine.ActorRef{Kind: "user"}, wantField: "approved_by.id"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, drafts, _, events := approvalService(t)
			draft := validReadyDraft()
			if err := drafts.Create(context.Background(), draft); err != nil {
				t.Fatalf("drafts.Create() error = %v", err)
			}

			ctx := actor.WithActor(context.Background(), actor.ActorContext{
				Actor:  tt.ctxActor,
				Source: actor.SourceDevHeader,
			})

			_, err := service.ApproveDraft(ctx, draft.ID, approveRequest())
			var validationErr *approvedcontract.ValidationError
			if !errors.As(err, &validationErr) {
				t.Fatalf("ApproveDraft() error = %v, want ValidationError", err)
			}
			if validationErr.Field != tt.wantField {
				t.Fatalf("validation field = %q, want %q", validationErr.Field, tt.wantField)
			}
			if got := len(events.Events()); got != 0 {
				t.Fatalf("events = %d, want 0", got)
			}
		})
	}
}

func TestServiceRejectsIncompleteDraft(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*spine.ContractDraft)
		reason string
	}{
		{name: "missing_scope", mutate: func(draft *spine.ContractDraft) { draft.ProposedScope = nil }, reason: approvedcontract.ReasonMissingProposedScope},
		{name: "missing_acceptance", mutate: func(draft *spine.ContractDraft) { draft.ProposedAcceptanceCriteria = []string{} }, reason: approvedcontract.ReasonMissingProposedAcceptanceCriteria},
		{name: "missing_proof", mutate: func(draft *spine.ContractDraft) { draft.ProposedProofExpectations = []string{" "} }, reason: approvedcontract.ReasonMissingProposedProofExpectations},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, drafts, _, events := approvalService(t)
			draft := validReadyDraft()
			draft.ID = spine.ContractDraftID("contract-draft-" + tt.name)
			tt.mutate(&draft)
			if err := drafts.Create(context.Background(), draft); err != nil {
				t.Fatalf("drafts.Create() error = %v", err)
			}

			_, err := service.ApproveDraft(context.Background(), draft.ID, approveRequest())
			var completenessErr *approvedcontract.CompletenessError
			if !errors.As(err, &completenessErr) {
				t.Fatalf("ApproveDraft() error = %v, want CompletenessError", err)
			}
			if !hasReason(completenessErr.ReasonCodes, tt.reason) {
				t.Fatalf("reason codes = %#v, want %q", completenessErr.ReasonCodes, tt.reason)
			}
			if got := len(events.Events()); got != 0 {
				t.Fatalf("events = %d, want 0", got)
			}
		})
	}
}

func TestServiceDoesNotAppendWorkGateProofEvents(t *testing.T) {
	service, drafts, _, events := approvalService(t)
	draft := validReadyDraft()
	if err := drafts.Create(context.Background(), draft); err != nil {
		t.Fatalf("drafts.Create() error = %v", err)
	}
	if _, err := service.ApproveDraft(context.Background(), draft.ID, approveRequest()); err != nil {
		t.Fatalf("ApproveDraft() error = %v", err)
	}
	assertNoForbiddenEvents(t, events.Events())
}

func approvalService(t *testing.T) (*approvedcontract.Service, *store.ContractDraftStore, *store.ApprovedContractStore, *eventlog.EventLog) {
	t.Helper()

	drafts := store.NewContractDraftStore()
	approved := store.NewApprovedContractStore()
	events := eventlog.NewEventLog()
	service := approvedcontract.NewService(drafts, approved, events, fixedClock{now: testTime()}, &sequenceIDs{})
	return service, drafts, approved, events
}

func validReadyDraft() spine.ContractDraft {
	return spine.ContractDraft{
		ID:                         "contract-draft-1",
		OrganizationID:             "organization-1",
		ProjectID:                  "project-1",
		ContractSeedID:             "contract-seed-1",
		GoalID:                     "goal-1",
		RepoBindingID:              "repo-binding-1",
		Title:                      "Refactor CSV export filters",
		IntentSummary:              "Current code duplicates filter logic.",
		ProposedScope:              []string{"Refactor duplicate filter logic"},
		ProposedNonGoals:           []string{},
		ProposedConstraints:        []string{},
		ProposedAcceptanceCriteria: []string{"Existing CSV export behavior is preserved"},
		ProposedExpectedChecks:     []string{},
		ProposedProofExpectations:  []string{"Provide evidence that acceptance criteria were checked."},
		RiskHints:                  []string{},
		SourceRefs: []spine.SourceRef{
			{Kind: "contract_seed", ID: "contract-seed-1"},
			{Kind: "goal", ID: "goal-1"},
			{Kind: "intake", ID: "intake-1"},
		},
		State:     spine.ContractDraftStateReadyForApproval,
		CreatedAt: testTime(),
	}
}

func approveRequest() spine.ApproveContractDraftRequest {
	return spine.ApproveContractDraftRequest{
		ApprovedBy: spine.ActorRef{Kind: "user", ID: "dev_approver", DisplayName: "Approver"},
	}
}

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

type sequenceIDs struct {
	approvedContract int
	event            int
}

func (g *sequenceIDs) NewApprovedContractID() (spine.ApprovedContractID, error) {
	g.approvedContract++
	return spine.ApprovedContractID("approved-contract-1"), nil
}

func (g *sequenceIDs) NewEventID() (spine.EventID, error) {
	g.event++
	return spine.EventID("event-1"), nil
}

func testTime() time.Time {
	return time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
}

func hasSourceRef(refs []spine.SourceRef, kind string, id string) bool {
	for _, ref := range refs {
		if ref.Kind == kind && ref.ID == id {
			return true
		}
	}
	return false
}

func hasReason(reasons []string, want string) bool {
	for _, reason := range reasons {
		if reason == want {
			return true
		}
	}
	return false
}

func countEventType(events []spine.Event, eventType string) int {
	count := 0
	for _, event := range events {
		if event.Type == eventType {
			count++
		}
	}
	return count
}

func assertNoForbiddenEvents(t *testing.T, events []spine.Event) {
	t.Helper()

	forbidden := map[string]bool{
		"work_item.created":     true,
		"run.started":           true,
		"gate.decision_written": true,
		"proof.created":         true,
	}
	for _, event := range events {
		if forbidden[event.Type] {
			t.Fatalf("forbidden event type appended: %s", event.Type)
		}
	}
}
