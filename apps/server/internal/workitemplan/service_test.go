package workitemplan_test

import (
	"context"
	"github.com/heurema/goalrail/apps/server/internal/spine"
	"github.com/heurema/goalrail/apps/server/internal/workitem"
	"github.com/heurema/goalrail/apps/server/internal/workitemplan"
	"testing"
	"time"
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

func planningService(t *testing.T) (*workitemplan.Service, *fakeContractStore, *fakeApprovedContractStore, *fakeWorkItemPlanStore, *fakeWorkItemPlanProposalStore, *fakeWorkItemStore, *fakeEventLog) {
	t.Helper()
	contracts := newFakeContractStore()
	approvedContracts := newFakeApprovedContractStore()
	plans := newFakeWorkItemPlanStore()
	proposals := newFakeWorkItemPlanProposalStore()
	workItems := newFakeWorkItemStore()
	events := newFakeEventLog()
	approved := validApprovedContract()
	storeContractForApproved(t, contracts, approved)
	if err := approvedContracts.Create(context.Background(), approved); err != nil {
		t.Fatalf("approvedContracts.Create() error = %v", err)
	}
	service := workitemplan.NewService(contracts, approvedContracts, plans, proposals, workItems, events, fixedClock{now: testTime()}, &sequenceIDs{})
	return service, contracts, approvedContracts, plans, proposals, workItems, events
}

type fakeContractStore struct {
	contracts map[spine.ContractID]spine.Contract
}

func newFakeContractStore() *fakeContractStore {
	return &fakeContractStore{contracts: map[spine.ContractID]spine.Contract{}}
}

func (s *fakeContractStore) Create(_ context.Context, contract spine.Contract) error {
	s.contracts[contract.ID] = contract
	return nil
}

func (s *fakeContractStore) Get(_ context.Context, id spine.ContractID) (spine.Contract, bool, error) {
	contract, ok := s.contracts[id]
	return contract, ok, nil
}

type fakeApprovedContractStore struct {
	approved map[spine.ApprovedContractID]spine.ApprovedContract
}

func newFakeApprovedContractStore() *fakeApprovedContractStore {
	return &fakeApprovedContractStore{approved: map[spine.ApprovedContractID]spine.ApprovedContract{}}
}

func (s *fakeApprovedContractStore) Create(_ context.Context, approved spine.ApprovedContract) error {
	s.approved[approved.ID] = approved
	return nil
}

func (s *fakeApprovedContractStore) Get(_ context.Context, id spine.ApprovedContractID) (spine.ApprovedContract, bool, error) {
	approved, ok := s.approved[id]
	return approved, ok, nil
}

type fakeWorkItemPlanStore struct {
	plans      map[spine.WorkItemPlanID]spine.WorkItemPlan
	byContract map[spine.ContractID]spine.WorkItemPlanID
}

func newFakeWorkItemPlanStore() *fakeWorkItemPlanStore {
	return &fakeWorkItemPlanStore{
		plans:      map[spine.WorkItemPlanID]spine.WorkItemPlan{},
		byContract: map[spine.ContractID]spine.WorkItemPlanID{},
	}
}

func (s *fakeWorkItemPlanStore) Create(_ context.Context, plan spine.WorkItemPlan) error {
	s.plans[plan.ID] = plan
	s.byContract[plan.ContractID] = plan.ID
	return nil
}

func (s *fakeWorkItemPlanStore) Get(_ context.Context, id spine.WorkItemPlanID) (spine.WorkItemPlan, bool, error) {
	plan, ok := s.plans[id]
	return plan, ok, nil
}

func (s *fakeWorkItemPlanStore) GetByContractID(_ context.Context, id spine.ContractID) (spine.WorkItemPlan, bool, error) {
	planID, ok := s.byContract[id]
	if !ok {
		return spine.WorkItemPlan{}, false, nil
	}
	plan, ok := s.plans[planID]
	return plan, ok, nil
}

func (s *fakeWorkItemPlanStore) MarkProposalSubmitted(_ context.Context, id spine.WorkItemPlanID, updatedAt time.Time) error {
	plan := s.plans[id]
	plan.State = spine.WorkItemPlanStateProposalSubmitted
	plan.UpdatedAt = updatedAt.UTC()
	s.plans[id] = plan
	return nil
}

func (s *fakeWorkItemPlanStore) MarkAccepted(_ context.Context, id spine.WorkItemPlanID, updatedAt time.Time) error {
	plan := s.plans[id]
	plan.State = spine.WorkItemPlanStateAccepted
	plan.UpdatedAt = updatedAt.UTC()
	s.plans[id] = plan
	return nil
}

type fakeWorkItemPlanProposalStore struct {
	proposals map[spine.WorkItemPlanProposalID]spine.WorkItemPlanProposal
	byPlan    map[spine.WorkItemPlanID]spine.WorkItemPlanProposalID
}

func newFakeWorkItemPlanProposalStore() *fakeWorkItemPlanProposalStore {
	return &fakeWorkItemPlanProposalStore{
		proposals: map[spine.WorkItemPlanProposalID]spine.WorkItemPlanProposal{},
		byPlan:    map[spine.WorkItemPlanID]spine.WorkItemPlanProposalID{},
	}
}

func (s *fakeWorkItemPlanProposalStore) Create(_ context.Context, proposal spine.WorkItemPlanProposal) error {
	s.proposals[proposal.ID] = proposal
	s.byPlan[proposal.PlanID] = proposal.ID
	return nil
}

func (s *fakeWorkItemPlanProposalStore) Get(_ context.Context, id spine.WorkItemPlanProposalID) (spine.WorkItemPlanProposal, bool, error) {
	proposal, ok := s.proposals[id]
	return proposal, ok, nil
}

func (s *fakeWorkItemPlanProposalStore) GetByPlanID(_ context.Context, id spine.WorkItemPlanID) (spine.WorkItemPlanProposal, bool, error) {
	proposalID, ok := s.byPlan[id]
	if !ok {
		return spine.WorkItemPlanProposal{}, false, nil
	}
	proposal, ok := s.proposals[proposalID]
	return proposal, ok, nil
}

func (s *fakeWorkItemPlanProposalStore) MarkAccepted(_ context.Context, id spine.WorkItemPlanProposalID, acceptedBy spine.ActorRef, acceptedAt time.Time) error {
	proposal := s.proposals[id]
	proposal.State = spine.WorkItemProposalStateAccepted
	proposal.AcceptedBy = &acceptedBy
	acceptedAt = acceptedAt.UTC()
	proposal.AcceptedAt = &acceptedAt
	proposal.UpdatedAt = acceptedAt
	s.proposals[id] = proposal
	return nil
}

type fakeWorkItemStore struct {
	items              map[spine.WorkItemID]spine.WorkItem
	byApprovedContract map[spine.ApprovedContractID][]spine.WorkItemID
}

func newFakeWorkItemStore() *fakeWorkItemStore {
	return &fakeWorkItemStore{
		items:              map[spine.WorkItemID]spine.WorkItem{},
		byApprovedContract: map[spine.ApprovedContractID][]spine.WorkItemID{},
	}
}

func (s *fakeWorkItemStore) Create(_ context.Context, item spine.WorkItem) error {
	s.items[item.ID] = item
	s.byApprovedContract[item.ApprovedContractID] = append(s.byApprovedContract[item.ApprovedContractID], item.ID)
	return nil
}

func (s *fakeWorkItemStore) Get(_ context.Context, id spine.WorkItemID) (spine.WorkItem, bool, error) {
	item, ok := s.items[id]
	return item, ok, nil
}

func (s *fakeWorkItemStore) GetByApprovedContractID(_ context.Context, id spine.ApprovedContractID) (spine.WorkItem, bool, error) {
	itemIDs := s.byApprovedContract[id]
	if len(itemIDs) == 0 {
		return spine.WorkItem{}, false, nil
	}
	item, ok := s.items[itemIDs[0]]
	return item, ok, nil
}

type fakeEventLog struct {
	events []spine.Event
}

func newFakeEventLog() *fakeEventLog {
	return &fakeEventLog{}
}

func (l *fakeEventLog) Append(_ context.Context, event spine.Event) error {
	l.events = append(l.events, cloneEvent(event))
	return nil
}

func (l *fakeEventLog) Events() []spine.Event {
	events := make([]spine.Event, len(l.events))
	for i, event := range l.events {
		events[i] = cloneEvent(event)
	}
	return events
}

func cloneEvent(event spine.Event) spine.Event {
	event.Payload = append([]byte(nil), event.Payload...)
	return event
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

func storeContractForApproved(t *testing.T, contracts *fakeContractStore, approved spine.ApprovedContract) {
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
