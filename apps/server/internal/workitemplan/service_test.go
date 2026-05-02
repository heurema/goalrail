package workitemplan_test

import (
	"context"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/eventlog"
	"github.com/heurema/goalrail/apps/server/internal/spine"
	"github.com/heurema/goalrail/apps/server/internal/store"
	"github.com/heurema/goalrail/apps/server/internal/workitem"
	"github.com/heurema/goalrail/apps/server/internal/workitemplan"
)

func TestServicePlanProposalAcceptanceFlow(t *testing.T) {
	service, _, _, plans, proposals, workItems, events := planningService(t)
	approved := validApprovedContract()

	plan, err := service.CreatePlan(context.Background(), approved.ContractID, spine.WorkItemPlanCreateRequest{
		RequestedBy: spine.ActorRef{Kind: "user", ID: "requester"},
	})
	if err != nil {
		t.Fatalf("CreatePlan() error = %v", err)
	}
	if plan.State != spine.WorkItemPlanStateQueued {
		t.Fatalf("plan state = %q, want queued", plan.State)
	}
	if _, ok, err := workItems.GetByApprovedContractID(context.Background(), approved.ID); err != nil {
		t.Fatalf("work item lookup error = %v", err)
	} else if ok {
		t.Fatal("CreatePlan materialized a WorkItem")
	}

	proposal, err := service.SubmitProposal(context.Background(), plan.ID, validProposalRequest(string(approved.ID)))
	if err != nil {
		t.Fatalf("SubmitProposal() error = %v", err)
	}
	if proposal.State != spine.WorkItemProposalStateSubmitted {
		t.Fatalf("proposal state = %q, want submitted", proposal.State)
	}
	storedPlan, _, _ := plans.Get(context.Background(), plan.ID)
	if storedPlan.State != spine.WorkItemPlanStateProposalSubmitted {
		t.Fatalf("plan state = %q, want proposal_submitted", storedPlan.State)
	}

	accepted, err := service.AcceptProposal(context.Background(), proposal.ID, spine.WorkItemPlanAcceptanceRequest{
		AcceptedBy: spine.ActorRef{Kind: "user", ID: "acceptor"},
	})
	if err != nil {
		t.Fatalf("AcceptProposal() error = %v", err)
	}
	if got := len(accepted.CreatedTaskIDs); got != 1 {
		t.Fatalf("created tasks = %d, want 1", got)
	}
	task, ok, err := workItems.Get(context.Background(), accepted.CreatedTaskIDs[0])
	if err != nil {
		t.Fatalf("workItems.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("accepted task not stored")
	}
	if task.PlanID != plan.ID || task.ProposalID != proposal.ID {
		t.Fatalf("task trace = %q/%q, want %q/%q", task.PlanID, task.ProposalID, plan.ID, proposal.ID)
	}
	storedProposal, _, _ := proposals.Get(context.Background(), proposal.ID)
	if storedProposal.State != spine.WorkItemProposalStateAccepted {
		t.Fatalf("proposal state = %q, want accepted", storedProposal.State)
	}
	if got := countEventType(events.Events(), workitem.EventTypeWorkItemCreated); got != 1 {
		t.Fatalf("work_item.created events = %d, want 1", got)
	}
}

func TestServiceRejectsDuplicatePlanProposalAndAcceptance(t *testing.T) {
	service, _, _, _, _, _, _ := planningService(t)
	approved := validApprovedContract()
	plan, err := service.CreatePlan(context.Background(), approved.ContractID, spine.WorkItemPlanCreateRequest{
		RequestedBy: spine.ActorRef{Kind: "user", ID: "requester"},
	})
	if err != nil {
		t.Fatalf("CreatePlan() error = %v", err)
	}
	if _, err := service.CreatePlan(context.Background(), approved.ContractID, spine.WorkItemPlanCreateRequest{
		RequestedBy: spine.ActorRef{Kind: "user", ID: "requester"},
	}); err != workitemplan.ErrAlreadyPlanned {
		t.Fatalf("duplicate CreatePlan() error = %v, want %v", err, workitemplan.ErrAlreadyPlanned)
	}
	proposal, err := service.SubmitProposal(context.Background(), plan.ID, validProposalRequest(string(approved.ID)))
	if err != nil {
		t.Fatalf("SubmitProposal() error = %v", err)
	}
	if _, err := service.SubmitProposal(context.Background(), plan.ID, validProposalRequest(string(approved.ID))); err != workitemplan.ErrAlreadyProposed {
		t.Fatalf("duplicate SubmitProposal() error = %v, want %v", err, workitemplan.ErrAlreadyProposed)
	}
	if _, err := service.AcceptProposal(context.Background(), proposal.ID, spine.WorkItemPlanAcceptanceRequest{
		AcceptedBy: spine.ActorRef{Kind: "user", ID: "acceptor"},
	}); err != nil {
		t.Fatalf("AcceptProposal() error = %v", err)
	}
	if _, err := service.AcceptProposal(context.Background(), proposal.ID, spine.WorkItemPlanAcceptanceRequest{
		AcceptedBy: spine.ActorRef{Kind: "user", ID: "acceptor"},
	}); err != workitemplan.ErrAlreadyAccepted {
		t.Fatalf("duplicate AcceptProposal() error = %v, want %v", err, workitemplan.ErrAlreadyAccepted)
	}
}

func planningService(t *testing.T) (*workitemplan.Service, *store.ContractStore, *store.ApprovedContractStore, *store.WorkItemPlanStore, *store.WorkItemPlanProposalStore, *store.WorkItemStore, *eventlog.EventLog) {
	t.Helper()
	contracts := store.NewContractStore()
	approvedContracts := store.NewApprovedContractStore()
	plans := store.NewWorkItemPlanStore()
	proposals := store.NewWorkItemPlanProposalStore()
	workItems := store.NewWorkItemStore()
	events := eventlog.NewEventLog()
	approved := validApprovedContract()
	storeContractForApproved(t, contracts, approved)
	if err := approvedContracts.Create(context.Background(), approved); err != nil {
		t.Fatalf("approvedContracts.Create() error = %v", err)
	}
	service := workitemplan.NewService(contracts, approvedContracts, plans, proposals, workItems, events, fixedClock{now: testTime()}, &sequenceIDs{})
	return service, contracts, approvedContracts, plans, proposals, workItems, events
}

func validApprovedContract() spine.ApprovedContract {
	return spine.ApprovedContract{
		ID:                 "approved-contract-1",
		OrganizationID:     "organization-1",
		ProjectID:          "project-1",
		ContractID:         "contract-1",
		ContractDraftID:    "contract-draft-1",
		ContractSeedID:     "contract-seed-1",
		GoalID:             "goal-1",
		RepoBindingID:      "repo-binding-1",
		Title:              "Refactor CSV export filters",
		IntentSummary:      "Current code duplicates filter logic.",
		Scope:              []string{"Refactor duplicate CSV export filter logic"},
		AcceptanceCriteria: []string{"Existing behavior is preserved"},
		ProofExpectations:  []string{"Show checks"},
		ApprovedBy:         spine.ActorRef{Kind: "user", ID: "approver"},
		ApprovedAt:         testTime(),
		State:              spine.ApprovedContractStateApproved,
	}
}

func storeContractForApproved(t *testing.T, contracts *store.ContractStore, approved spine.ApprovedContract) {
	t.Helper()
	seedID := approved.ContractSeedID
	draftID := approved.ContractDraftID
	approvedID := approved.ID
	err := contracts.Create(context.Background(), spine.Contract{
		ID:                 approved.ContractID,
		OrganizationID:     approved.OrganizationID,
		ProjectID:          approved.ProjectID,
		RepoBindingID:      approved.RepoBindingID,
		GoalID:             approved.GoalID,
		State:              spine.ContractStateApproved,
		CurrentSeedID:      &seedID,
		CurrentDraftID:     &draftID,
		ApprovedSnapshotID: &approvedID,
		CreatedAt:          testTime(),
		UpdatedAt:          testTime(),
	})
	if err != nil {
		t.Fatalf("contracts.Create() error = %v", err)
	}
}

func validProposalRequest(approvedContractID string) spine.WorkItemPlanProposalSubmitRequest {
	orderIndex := 0
	return spine.WorkItemPlanProposalSubmitRequest{
		SubmittedBy:        spine.ActorRef{Kind: "worker", ID: "planner-worker-1"},
		Planner:            map[string]any{"kind": "goalrail_worker", "id": "planner-worker-1"},
		SourceSnapshotRefs: []spine.SourceRef{{Kind: "approved_contract", ID: approvedContractID}},
		Rationale:          "One bounded task is enough.",
		ProposedTasks: []spine.ProposedWorkItem{
			{
				Title:                "Refactor CSV export filter builder",
				Summary:              "Extract duplicated filter construction logic.",
				Scope:                []string{"Update export filter construction code"},
				AcceptanceRefs:       []string{"acceptance_criteria[0]"},
				ProofExpectationRefs: []string{"proof_expectations[0]"},
				OrderIndex:           &orderIndex,
				SourceRefs:           []spine.SourceRef{{Kind: "approved_contract", ID: approvedContractID}},
			},
		},
	}
}

func countEventType(events []spine.Event, eventType string) int {
	var count int
	for _, event := range events {
		if event.Type == eventType {
			count++
		}
	}
	return count
}

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

type sequenceIDs struct {
	plan     int
	proposal int
	workItem int
	event    int
}

func (g *sequenceIDs) NewWorkItemPlanID() (spine.WorkItemPlanID, error) {
	g.plan++
	return spine.WorkItemPlanID("plan-1"), nil
}

func (g *sequenceIDs) NewWorkItemPlanProposalID() (spine.WorkItemPlanProposalID, error) {
	g.proposal++
	return spine.WorkItemPlanProposalID("proposal-1"), nil
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
	return time.Date(2026, 5, 2, 12, 0, 0, 0, time.UTC)
}
