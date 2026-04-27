package workitem_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/eventlog"
	"github.com/heurema/goalrail/apps/server/internal/spine"
	"github.com/heurema/goalrail/apps/server/internal/store"
	"github.com/heurema/goalrail/apps/server/internal/workitem"
)

func TestServiceCreatesPlannedWorkItemFromApprovedContract(t *testing.T) {
	service, approvedContracts, workItems, _ := planningService(t)
	approved := validApprovedContract()
	if err := approvedContracts.Create(context.Background(), approved); err != nil {
		t.Fatalf("approvedContracts.Create() error = %v", err)
	}

	item, err := service.PlanApprovedContract(context.Background(), approved.ID)
	if err != nil {
		t.Fatalf("PlanApprovedContract() error = %v", err)
	}

	if item.Status != spine.WorkItemStatusPlanned {
		t.Fatalf("status = %q, want %q", item.Status, spine.WorkItemStatusPlanned)
	}
	if item.ApprovedContractID != approved.ID || item.RepoBindingID != approved.RepoBindingID {
		t.Fatalf("ids = %q/%q, want approved/repo ids", item.ApprovedContractID, item.RepoBindingID)
	}
	if item.OrganizationID != approved.OrganizationID || item.ProjectID != approved.ProjectID {
		t.Fatalf("context = %q/%q, want approved context %q/%q", item.OrganizationID, item.ProjectID, approved.OrganizationID, approved.ProjectID)
	}
	if item.Title != approved.Title || item.Summary != approved.IntentSummary {
		t.Fatalf("title/summary not copied from approved contract")
	}
	if !reflect.DeepEqual(item.Scope, approved.Scope) {
		t.Fatalf("scope = %#v, want %#v", item.Scope, approved.Scope)
	}
	if !reflect.DeepEqual(item.AcceptanceRefs, []string{"acceptance_criteria[0]"}) {
		t.Fatalf("acceptance_refs = %#v, want indexed refs", item.AcceptanceRefs)
	}
	if !reflect.DeepEqual(item.ProofExpectationRefs, []string{"proof_expectations[0]"}) {
		t.Fatalf("proof_expectation_refs = %#v, want indexed refs", item.ProofExpectationRefs)
	}
	if item.OwnerHint != "" {
		t.Fatalf("owner_hint = %q, want empty advisory hint", item.OwnerHint)
	}
	if item.OrderIndex != nil {
		t.Fatalf("order_index = %v, want nil", *item.OrderIndex)
	}
	if !hasSourceRef(item.SourceRefs, workitem.SourceRefKindApprovedContract, string(approved.ID)) {
		t.Fatalf("source_refs = %#v, want approved_contract ref", item.SourceRefs)
	}
	if !hasSourceRef(item.SourceRefs, "contract_draft", string(approved.ContractDraftID)) {
		t.Fatalf("source_refs = %#v, want preserved contract_draft ref", item.SourceRefs)
	}

	stored, ok, err := workItems.GetByApprovedContractID(context.Background(), approved.ID)
	if err != nil {
		t.Fatalf("workItems.GetByApprovedContractID() error = %v", err)
	}
	if !ok {
		t.Fatal("workItems.GetByApprovedContractID() ok = false, want true")
	}
	if stored.ID != item.ID {
		t.Fatalf("stored id = %q, want %q", stored.ID, item.ID)
	}

	storedApproved, ok, err := approvedContracts.Get(context.Background(), approved.ID)
	if err != nil {
		t.Fatalf("approvedContracts.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("approvedContracts.Get() ok = false, want true")
	}
	if !reflect.DeepEqual(storedApproved, approved) {
		t.Fatalf("approved contract mutated: %#v want %#v", storedApproved, approved)
	}
}

func TestServiceAppendsWorkItemCreatedEvent(t *testing.T) {
	service, approvedContracts, _, events := planningService(t)
	approved := validApprovedContract()
	if err := approvedContracts.Create(context.Background(), approved); err != nil {
		t.Fatalf("approvedContracts.Create() error = %v", err)
	}

	item, err := service.PlanApprovedContract(context.Background(), approved.ID)
	if err != nil {
		t.Fatalf("PlanApprovedContract() error = %v", err)
	}

	appended := events.Events()
	if got := countEventType(appended, workitem.EventTypeWorkItemCreated); got != 1 {
		t.Fatalf("work_item.created events = %d, want 1", got)
	}
	event := appended[len(appended)-1]
	if event.EntityType != workitem.EntityTypeWorkItem {
		t.Fatalf("entity type = %q, want %q", event.EntityType, workitem.EntityTypeWorkItem)
	}
	if event.EntityID != string(item.ID) {
		t.Fatalf("entity id = %q, want %q", event.EntityID, item.ID)
	}
	if event.OrganizationID != approved.OrganizationID || event.ProjectID != approved.ProjectID || event.RepoBindingID != approved.RepoBindingID {
		t.Fatalf("event context = %q/%q/%q, want approved context %q/%q/%q", event.OrganizationID, event.ProjectID, event.RepoBindingID, approved.OrganizationID, approved.ProjectID, approved.RepoBindingID)
	}

	var payload struct {
		WorkItemID           spine.WorkItemID         `json:"work_item_id"`
		ApprovedContractID   spine.ApprovedContractID `json:"approved_contract_id"`
		RepoBindingID        spine.RepoBindingID      `json:"repo_binding_id"`
		AcceptanceRefs       []string                 `json:"acceptance_refs"`
		ProofExpectationRefs []string                 `json:"proof_expectation_refs"`
		Status               spine.WorkItemStatus     `json:"status"`
		SourceRefs           []spine.SourceRef        `json:"source_refs"`
		CreatedAt            time.Time                `json:"created_at"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatalf("unmarshal work_item.created payload: %v", err)
	}
	if payload.WorkItemID != item.ID || payload.ApprovedContractID != approved.ID || payload.RepoBindingID != approved.RepoBindingID {
		t.Fatalf("payload ids = %q/%q/%q, want item/approved/repo ids", payload.WorkItemID, payload.ApprovedContractID, payload.RepoBindingID)
	}
	if payload.Status != spine.WorkItemStatusPlanned {
		t.Fatalf("payload status = %q, want planned", payload.Status)
	}
	if !payload.CreatedAt.Equal(testTime()) {
		t.Fatalf("created_at = %s, want %s", payload.CreatedAt, testTime())
	}
	if !hasSourceRef(payload.SourceRefs, workitem.SourceRefKindApprovedContract, string(approved.ID)) {
		t.Fatalf("payload source_refs = %#v, want approved_contract ref", payload.SourceRefs)
	}
}

func TestServiceRejectsDuplicatePlanning(t *testing.T) {
	service, approvedContracts, _, events := planningService(t)
	approved := validApprovedContract()
	if err := approvedContracts.Create(context.Background(), approved); err != nil {
		t.Fatalf("approvedContracts.Create() error = %v", err)
	}
	if _, err := service.PlanApprovedContract(context.Background(), approved.ID); err != nil {
		t.Fatalf("first PlanApprovedContract() error = %v", err)
	}

	_, err := service.PlanApprovedContract(context.Background(), approved.ID)
	if !errors.Is(err, workitem.ErrAlreadyPlanned) {
		t.Fatalf("second PlanApprovedContract() error = %v, want ErrAlreadyPlanned", err)
	}
	if got := countEventType(events.Events(), workitem.EventTypeWorkItemCreated); got != 1 {
		t.Fatalf("work_item.created events = %d, want 1", got)
	}
}

func TestServiceRejectsApprovedContractNotApproved(t *testing.T) {
	service, approvedContracts, _, events := planningService(t)
	approved := validApprovedContract()
	approved.State = spine.ApprovedContractState("draft")
	if err := approvedContracts.Create(context.Background(), approved); err != nil {
		t.Fatalf("approvedContracts.Create() error = %v", err)
	}

	_, err := service.PlanApprovedContract(context.Background(), approved.ID)
	if !errors.Is(err, workitem.ErrInvalidApprovedContractState) {
		t.Fatalf("PlanApprovedContract() error = %v, want ErrInvalidApprovedContractState", err)
	}
	if got := len(events.Events()); got != 0 {
		t.Fatalf("events = %d, want 0", got)
	}
}

func TestServiceRejectsIncompleteApprovedContract(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*spine.ApprovedContract)
		reason string
	}{
		{name: "missing_repo_binding", mutate: func(approved *spine.ApprovedContract) { approved.RepoBindingID = "" }, reason: workitem.ReasonMissingRepoBindingID},
		{name: "missing_title", mutate: func(approved *spine.ApprovedContract) { approved.Title = " " }, reason: workitem.ReasonMissingTitle},
		{name: "missing_intent_summary", mutate: func(approved *spine.ApprovedContract) { approved.IntentSummary = "" }, reason: workitem.ReasonMissingIntentSummary},
		{name: "missing_scope", mutate: func(approved *spine.ApprovedContract) { approved.Scope = nil }, reason: workitem.ReasonMissingScope},
		{name: "missing_acceptance", mutate: func(approved *spine.ApprovedContract) { approved.AcceptanceCriteria = []string{} }, reason: workitem.ReasonMissingAcceptanceCriteria},
		{name: "missing_proof", mutate: func(approved *spine.ApprovedContract) { approved.ProofExpectations = []string{" "} }, reason: workitem.ReasonMissingProofExpectations},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, approvedContracts, _, events := planningService(t)
			approved := validApprovedContract()
			approved.ID = spine.ApprovedContractID("approved-contract-" + tt.name)
			approved.ContractDraftID = spine.ContractDraftID("contract-draft-" + tt.name)
			tt.mutate(&approved)
			if err := approvedContracts.Create(context.Background(), approved); err != nil {
				t.Fatalf("approvedContracts.Create() error = %v", err)
			}

			_, err := service.PlanApprovedContract(context.Background(), approved.ID)
			var completenessErr *workitem.CompletenessError
			if !errors.As(err, &completenessErr) {
				t.Fatalf("PlanApprovedContract() error = %v, want CompletenessError", err)
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

func TestServiceDoesNotAppendRunReceiptGateProofEvents(t *testing.T) {
	service, approvedContracts, _, events := planningService(t)
	approved := validApprovedContract()
	if err := approvedContracts.Create(context.Background(), approved); err != nil {
		t.Fatalf("approvedContracts.Create() error = %v", err)
	}
	if _, err := service.PlanApprovedContract(context.Background(), approved.ID); err != nil {
		t.Fatalf("PlanApprovedContract() error = %v", err)
	}
	assertNoForbiddenEvents(t, events.Events())
}

func planningService(t *testing.T) (*workitem.Service, *store.ApprovedContractStore, *store.WorkItemStore, *eventlog.EventLog) {
	t.Helper()

	approvedContracts := store.NewApprovedContractStore()
	workItems := store.NewWorkItemStore()
	events := eventlog.NewEventLog()
	service := workitem.NewService(approvedContracts, workItems, events, fixedClock{now: testTime()}, &sequenceIDs{})
	return service, approvedContracts, workItems, events
}

func validApprovedContract() spine.ApprovedContract {
	return spine.ApprovedContract{
		ID:                 "approved-contract-1",
		OrganizationID:     "organization-1",
		ProjectID:          "project-1",
		ContractDraftID:    "contract-draft-1",
		ContractSeedID:     "contract-seed-1",
		GoalID:             "goal-1",
		RepoBindingID:      "repo-binding-1",
		Title:              "Refactor CSV export filters",
		IntentSummary:      "Current code duplicates filter logic.",
		Scope:              []string{"Refactor duplicate filter logic"},
		NonGoals:           []string{},
		Constraints:        []string{},
		AcceptanceCriteria: []string{"Existing CSV export behavior is preserved"},
		ExpectedChecks:     []string{},
		ProofExpectations:  []string{"Provide evidence that acceptance criteria were checked."},
		RiskHints:          []string{},
		ApprovedBy:         spine.ActorRef{Kind: "user", ID: "dev_approver"},
		ApprovedAt:         testTime(),
		SourceRefs: []spine.SourceRef{
			{Kind: "contract_draft", ID: "contract-draft-1"},
			{Kind: "contract_seed", ID: "contract-seed-1"},
			{Kind: "goal", ID: "goal-1"},
		},
		State: spine.ApprovedContractStateApproved,
	}
}

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

type sequenceIDs struct {
	workItem int
	event    int
}

func (g *sequenceIDs) NewWorkItemID() (spine.WorkItemID, error) {
	g.workItem++
	return spine.WorkItemID("work-item-1"), nil
}

func (g *sequenceIDs) NewEventID() (spine.EventID, error) {
	g.event++
	return spine.EventID("event-1"), nil
}

func testTime() time.Time {
	return time.Date(2026, 4, 27, 12, 0, 0, 0, time.UTC)
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
		"run.started":           true,
		"receipt.submitted":     true,
		"gate.decision_written": true,
		"proof.created":         true,
	}
	for _, event := range events {
		if forbidden[event.Type] {
			t.Fatalf("forbidden event type appended: %s", event.Type)
		}
	}
}
