package workitemplan_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/spine"
	"github.com/heurema/goalrail/apps/server/internal/workitem"
	"github.com/heurema/goalrail/apps/server/internal/workitemplan"
)

func TestServicePlanProposalAcceptanceFlow(t *testing.T) {
	service, _, _, plans, leases, proposals, workItems, events := planningService(t)
	approved := validApprovedContract()

	plan, _, err := service.CreatePlan(context.Background(), approved.ContractID, spine.WorkItemPlanCreateRequest{
		RequestedBy: spine.ActorRef{Kind: "user", ID: "requester"},
	}, activeMembership(approved.OrganizationID))
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

	lease := acquireLease(t, service)
	proposal, err := service.SubmitProposal(context.Background(), plan.ID, validProposalRequest(string(approved.ID), lease))
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
	storedLease, _, _ := leases.Get(context.Background(), lease.ID)
	if storedLease.State != spine.WorkItemPlanLeaseStateCompleted {
		t.Fatalf("lease state = %q, want completed", storedLease.State)
	}

	accepted, err := service.AcceptProposal(context.Background(), proposal.ID, spine.WorkItemPlanAcceptanceRequest{
		AcceptedBy: spine.ActorRef{Kind: "user", ID: "acceptor"},
	}, activeMembership(approved.OrganizationID))
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

func TestServiceGetPlanIncludesApprovedContractProjection(t *testing.T) {
	service, _, _, _, _, _, _, _ := planningService(t)
	approved := validApprovedContract()
	plan, _, err := service.CreatePlan(context.Background(), approved.ContractID, spine.WorkItemPlanCreateRequest{
		RequestedBy: spine.ActorRef{Kind: "user", ID: "requester"},
	}, activeMembership(approved.OrganizationID))
	if err != nil {
		t.Fatalf("CreatePlan() error = %v", err)
	}

	got, err := service.GetPlan(context.Background(), plan.ID)
	if err != nil {
		t.Fatalf("GetPlan() error = %v", err)
	}
	if got.ApprovedContract == nil {
		t.Fatal("ApprovedContract projection is nil")
	}
	if got.ApprovedContract.ID != approved.ID || got.ApprovedContract.ContractID != approved.ContractID || got.ApprovedContract.RepoBindingID != approved.RepoBindingID {
		t.Fatalf("approved projection ids = %#v, want approved contract ids", got.ApprovedContract)
	}
	if got.ApprovedContract.Title != approved.Title || got.ApprovedContract.IntentSummary != approved.IntentSummary {
		t.Fatalf("approved projection text = %#v, want title/intent from approved Contract", got.ApprovedContract)
	}
	if len(got.ApprovedContract.Scope) != len(approved.Scope) || got.ApprovedContract.Scope[0] != approved.Scope[0] {
		t.Fatalf("approved projection scope = %#v, want %#v", got.ApprovedContract.Scope, approved.Scope)
	}
}

func TestServiceRejectsDuplicatePlanProposalAndAcceptance(t *testing.T) {
	service, _, _, _, _, _, _, _ := planningService(t)
	approved := validApprovedContract()
	plan, _, err := service.CreatePlan(context.Background(), approved.ContractID, spine.WorkItemPlanCreateRequest{
		RequestedBy: spine.ActorRef{Kind: "user", ID: "requester"},
	}, activeMembership(approved.OrganizationID))
	if err != nil {
		t.Fatalf("CreatePlan() error = %v", err)
	}
	existing, newlyCreated, err := service.CreatePlan(context.Background(), approved.ContractID, spine.WorkItemPlanCreateRequest{
		RequestedBy: spine.ActorRef{Kind: "user", ID: "requester"},
	}, activeMembership(approved.OrganizationID))
	if err != nil {
		t.Fatalf("duplicate CreatePlan() error = %v, want nil", err)
	}
	if newlyCreated || existing.ID != plan.ID {
		t.Fatalf("duplicate CreatePlan() = %#v newlyCreated=%t, want existing plan %q", existing, newlyCreated, plan.ID)
	}
	lease := acquireLease(t, service)
	proposal, err := service.SubmitProposal(context.Background(), plan.ID, validProposalRequest(string(approved.ID), lease))
	if err != nil {
		t.Fatalf("SubmitProposal() error = %v", err)
	}
	if _, err := service.SubmitProposal(context.Background(), plan.ID, validProposalRequest(string(approved.ID), lease)); err != workitemplan.ErrAlreadyProposed {
		t.Fatalf("duplicate SubmitProposal() error = %v, want %v", err, workitemplan.ErrAlreadyProposed)
	}
	if _, err := service.AcceptProposal(context.Background(), proposal.ID, spine.WorkItemPlanAcceptanceRequest{
		AcceptedBy: spine.ActorRef{Kind: "user", ID: "acceptor"},
	}, activeMembership(approved.OrganizationID)); err != nil {
		t.Fatalf("AcceptProposal() error = %v", err)
	}
	if _, err := service.AcceptProposal(context.Background(), proposal.ID, spine.WorkItemPlanAcceptanceRequest{
		AcceptedBy: spine.ActorRef{Kind: "user", ID: "acceptor"},
	}, activeMembership(approved.OrganizationID)); err != workitemplan.ErrAlreadyAccepted {
		t.Fatalf("duplicate AcceptProposal() error = %v, want %v", err, workitemplan.ErrAlreadyAccepted)
	}
}

func TestServiceSubmitProposalUsesRequiredTransactionRunner(t *testing.T) {
	txRunner := newFakeTransactionRunner()
	service, _, _, plans, _, proposals, _, _ := planningService(t, txRunner)
	approved := validApprovedContract()
	outerCtx := context.WithValue(context.Background(), txContextKey{}, "outer")
	plan, _, err := service.CreatePlan(context.Background(), approved.ContractID, spine.WorkItemPlanCreateRequest{
		RequestedBy: spine.ActorRef{Kind: "user", ID: "requester"},
	}, activeMembership(approved.OrganizationID))
	if err != nil {
		t.Fatalf("CreatePlan() error = %v", err)
	}

	lease := acquireLease(t, service)
	txRunner.calls = 0
	proposal, err := service.SubmitProposal(outerCtx, plan.ID, validProposalRequest(string(approved.ID), lease))
	if err != nil {
		t.Fatalf("SubmitProposal() error = %v", err)
	}

	if txRunner.calls != 1 {
		t.Fatalf("TxRunner calls = %d, want 1", txRunner.calls)
	}
	if proposals.createCtx != txRunner.txCtx {
		t.Fatal("Proposals.Create did not receive transaction context")
	}
	if plans.markProposalSubmittedCtx != txRunner.txCtx {
		t.Fatal("Plans.MarkProposalSubmitted did not receive transaction context")
	}
	if proposals.createCtx == outerCtx || plans.markProposalSubmittedCtx == outerCtx {
		t.Fatal("proposal submission writes used outer context")
	}
	storedPlan, _, _ := plans.Get(context.Background(), plan.ID)
	if storedPlan.State != spine.WorkItemPlanStateProposalSubmitted {
		t.Fatalf("plan state = %q, want proposal_submitted", storedPlan.State)
	}
	storedProposal, _, _ := proposals.Get(context.Background(), proposal.ID)
	if storedProposal.State != spine.WorkItemProposalStateSubmitted {
		t.Fatalf("proposal state = %q, want submitted", storedProposal.State)
	}
}

func TestServiceRenewLeaseMissReturnsSpecificLeaseConflict(t *testing.T) {
	service, _, _, _, leases, _, _, _ := planningService(t)
	approved := validApprovedContract()
	if _, _, err := service.CreatePlan(context.Background(), approved.ContractID, spine.WorkItemPlanCreateRequest{
		RequestedBy: spine.ActorRef{Kind: "user", ID: "requester"},
	}, activeMembership(approved.OrganizationID)); err != nil {
		t.Fatalf("CreatePlan() error = %v", err)
	}
	lease := acquireLease(t, service)

	leases.renewMiss = true
	leases.renewMissMode = spine.WorkItemPlanLeaseStateCompleted
	_, err := service.RenewLease(context.Background(), lease.ID, spine.WorkItemPlanLeaseRenewRequest{
		LeaseToken: lease.LeaseToken,
	})
	if !errors.Is(err, workitemplan.ErrLeaseCompleted) {
		t.Fatalf("RenewLease() error = %v, want %v", err, workitemplan.ErrLeaseCompleted)
	}
}

func TestServiceSubmitProposalLeaseCompletionMissReturnsLeaseConflict(t *testing.T) {
	service, _, _, _, leases, _, _, _ := planningService(t)
	approved := validApprovedContract()
	plan, _, err := service.CreatePlan(context.Background(), approved.ContractID, spine.WorkItemPlanCreateRequest{
		RequestedBy: spine.ActorRef{Kind: "user", ID: "requester"},
	}, activeMembership(approved.OrganizationID))
	if err != nil {
		t.Fatalf("CreatePlan() error = %v", err)
	}
	lease := acquireLease(t, service)

	leases.markCompletedMiss = true
	leases.markCompletedMissMode = spine.WorkItemPlanLeaseStateExpired
	_, err = service.SubmitProposal(context.Background(), plan.ID, validProposalRequest(string(approved.ID), lease))
	if !errors.Is(err, workitemplan.ErrLeaseExpired) {
		t.Fatalf("SubmitProposal() error = %v, want %v", err, workitemplan.ErrLeaseExpired)
	}
}

func TestServiceSubmitProposalFailedCreateDoesNotRunPostFailureDuplicateLookup(t *testing.T) {
	txRunner := newFakeTransactionRunner()
	service, _, _, _, _, proposals, _, _ := planningService(t, txRunner)
	approved := validApprovedContract()
	createErr := errors.New("create failed")
	proposals.createErr = createErr
	plan, _, err := service.CreatePlan(context.Background(), approved.ContractID, spine.WorkItemPlanCreateRequest{
		RequestedBy: spine.ActorRef{Kind: "user", ID: "requester"},
	}, activeMembership(approved.OrganizationID))
	if err != nil {
		t.Fatalf("CreatePlan() error = %v", err)
	}

	lease := acquireLease(t, service)
	_, err = service.SubmitProposal(txRunner.txCtx, plan.ID, validProposalRequest(string(approved.ID), lease))
	if !errors.Is(err, createErr) {
		t.Fatalf("SubmitProposal() error = %v, want original create error", err)
	}
	if got := len(proposals.getByPlanCtxs); got != 1 {
		t.Fatalf("Proposals.GetByPlanID calls = %d, want preflight only", got)
	}
}

func TestServiceAcceptProposalUsesRequiredTransactionRunner(t *testing.T) {
	txRunner := newFakeTransactionRunner()
	service, _, _, plans, _, proposals, workItems, events := planningService(t, txRunner)
	approved := validApprovedContract()

	plan, _, err := service.CreatePlan(context.Background(), approved.ContractID, spine.WorkItemPlanCreateRequest{
		RequestedBy: spine.ActorRef{Kind: "user", ID: "requester"},
	}, activeMembership(approved.OrganizationID))
	if err != nil {
		t.Fatalf("CreatePlan() error = %v", err)
	}
	lease := acquireLease(t, service)
	proposal, err := service.SubmitProposal(context.Background(), plan.ID, validProposalRequest(string(approved.ID), lease))
	if err != nil {
		t.Fatalf("SubmitProposal() error = %v", err)
	}
	txRunner.calls = 0

	accepted, err := service.AcceptProposal(context.Background(), proposal.ID, spine.WorkItemPlanAcceptanceRequest{
		AcceptedBy: spine.ActorRef{Kind: "user", ID: "acceptor"},
	}, activeMembership(approved.OrganizationID))
	if err != nil {
		t.Fatalf("AcceptProposal() error = %v", err)
	}
	if txRunner.calls != 1 {
		t.Fatalf("TxRunner calls = %d, want 1", txRunner.calls)
	}
	if got := len(accepted.CreatedTaskIDs); got != 1 {
		t.Fatalf("created tasks = %d, want 1", got)
	}
	if _, ok, err := workItems.Get(context.Background(), accepted.CreatedTaskIDs[0]); err != nil {
		t.Fatalf("workItems.Get() error = %v", err)
	} else if !ok {
		t.Fatal("accepted task not stored")
	}
	storedProposal, _, _ := proposals.Get(context.Background(), proposal.ID)
	if storedProposal.State != spine.WorkItemProposalStateAccepted {
		t.Fatalf("proposal state = %q, want accepted", storedProposal.State)
	}
	storedPlan, _, _ := plans.Get(context.Background(), plan.ID)
	if storedPlan.State != spine.WorkItemPlanStateAccepted {
		t.Fatalf("plan state = %q, want accepted", storedPlan.State)
	}
	if got := countEventType(events.Events(), workitem.EventTypeWorkItemCreated); got != 1 {
		t.Fatalf("work_item.created events = %d, want 1", got)
	}
}

func planningService(t *testing.T, runners ...workitemplan.TransactionRunner) (*workitemplan.Service, *fakeContractStore, *fakeApprovedContractStore, *fakeWorkItemPlanStore, *fakeLeaseStore, *fakeWorkItemPlanProposalStore, *fakeWorkItemStore, *fakeEventLog) {
	t.Helper()
	txRunner := workitemplan.TransactionRunner(newFakeTransactionRunner())
	if len(runners) > 0 {
		txRunner = runners[0]
	}
	contracts := newFakeContractStore()
	approvedContracts := newFakeApprovedContractStore()
	plans := newFakeWorkItemPlanStore()
	leases := newFakeLeaseStore(plans)
	proposals := newFakeWorkItemPlanProposalStore()
	workItems := newFakeWorkItemStore()
	events := newFakeEventLog()
	approved := validApprovedContract()
	storeContractForApproved(t, contracts, approved)
	if err := approvedContracts.Create(context.Background(), approved); err != nil {
		t.Fatalf("approvedContracts.Create() error = %v", err)
	}
	service := workitemplan.NewService(contracts, approvedContracts, plans, leases, proposals, workItems, events, txRunner, fixedClock{now: testTime()}, &sequenceIDs{})
	return service, contracts, approvedContracts, plans, leases, proposals, workItems, events
}

type fakeTransactionRunner struct {
	calls int
	txCtx context.Context
}

type txContextKey struct{}

func newFakeTransactionRunner() *fakeTransactionRunner {
	return &fakeTransactionRunner{txCtx: context.WithValue(context.Background(), txContextKey{}, "tx")}
}

func (r *fakeTransactionRunner) RunReadCommitted(_ context.Context, fn func(context.Context) error) error {
	r.calls++
	return fn(r.txCtx)
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
	plans                    map[spine.WorkItemPlanID]spine.WorkItemPlan
	byContract               map[spine.ContractID]spine.WorkItemPlanID
	markProposalSubmittedCtx context.Context
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

func (s *fakeWorkItemPlanStore) MarkProposalSubmitted(ctx context.Context, id spine.WorkItemPlanID, updatedAt time.Time) error {
	s.markProposalSubmittedCtx = ctx
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

type fakeLeaseStore struct {
	plans                 *fakeWorkItemPlanStore
	leases                map[spine.WorkItemPlanLeaseID]spine.WorkItemPlanLease
	renewMiss             bool
	renewMissMode         spine.WorkItemPlanLeaseState
	markCompletedMiss     bool
	markCompletedMissMode spine.WorkItemPlanLeaseState
}

func newFakeLeaseStore(plans *fakeWorkItemPlanStore) *fakeLeaseStore {
	return &fakeLeaseStore{
		plans:  plans,
		leases: map[spine.WorkItemPlanLeaseID]spine.WorkItemPlanLease{},
	}
}

func (s *fakeLeaseStore) AcquireNextLease(_ context.Context, input workitemplan.LeaseAcquireInput) (spine.WorkItemPlanLease, bool, error) {
	var selected spine.WorkItemPlan
	found := false
	for _, plan := range s.plans.plans {
		if plan.State == spine.WorkItemPlanStateQueued || (plan.State == spine.WorkItemPlanStateLeased && plan.LeaseExpiresAt != nil && !plan.LeaseExpiresAt.After(input.CreatedAt)) {
			if !found || plan.CreatedAt.Before(selected.CreatedAt) || (plan.CreatedAt.Equal(selected.CreatedAt) && plan.ID < selected.ID) {
				selected = plan
				found = true
			}
		}
	}
	if !found {
		return spine.WorkItemPlanLease{}, false, nil
	}
	if selected.State == spine.WorkItemPlanStateLeased && selected.CurrentLeaseID != nil {
		previous := s.leases[*selected.CurrentLeaseID]
		previous.State = spine.WorkItemPlanLeaseStateExpired
		previous.UpdatedAt = input.UpdatedAt
		s.leases[previous.ID] = previous
	}
	lease := spine.WorkItemPlanLease{
		ID:                 input.ID,
		PlanID:             selected.ID,
		ContractID:         selected.ContractID,
		ApprovedContractID: selected.ApprovedContractID,
		RepoBindingID:      selected.RepoBindingID,
		LeasedBy:           input.LeasedBy,
		State:              spine.WorkItemPlanLeaseStateActive,
		LeaseTokenHash:     input.LeaseTokenHash,
		ExpiresAt:          input.ExpiresAt,
		CreatedAt:          input.CreatedAt,
		UpdatedAt:          input.UpdatedAt,
	}
	s.leases[lease.ID] = lease
	selected.State = spine.WorkItemPlanStateLeased
	selected.CurrentLeaseID = &lease.ID
	selected.LeasedBy = &lease.LeasedBy
	selected.LeaseExpiresAt = &lease.ExpiresAt
	selected.UpdatedAt = input.UpdatedAt
	s.plans.plans[selected.ID] = selected
	return lease, true, nil
}

func (s *fakeLeaseStore) Get(_ context.Context, id spine.WorkItemPlanLeaseID) (spine.WorkItemPlanLease, bool, error) {
	lease, ok := s.leases[id]
	return lease, ok, nil
}

func (s *fakeLeaseStore) Renew(_ context.Context, id spine.WorkItemPlanLeaseID, tokenHash string, expiresAt time.Time, updatedAt time.Time) (spine.WorkItemPlanLease, bool, error) {
	lease, ok := s.leases[id]
	if !ok || lease.LeaseTokenHash != tokenHash || lease.State != spine.WorkItemPlanLeaseStateActive || !lease.ExpiresAt.After(updatedAt) {
		return spine.WorkItemPlanLease{}, false, nil
	}
	if s.renewMiss {
		if s.renewMissMode != "" {
			lease.State = s.renewMissMode
			s.leases[id] = lease
		}
		return spine.WorkItemPlanLease{}, false, nil
	}
	lease.ExpiresAt = expiresAt
	lease.UpdatedAt = updatedAt
	s.leases[id] = lease
	plan := s.plans.plans[lease.PlanID]
	plan.LeaseExpiresAt = &lease.ExpiresAt
	plan.UpdatedAt = updatedAt
	s.plans.plans[plan.ID] = plan
	return lease, true, nil
}

func (s *fakeLeaseStore) MarkCompleted(_ context.Context, id spine.WorkItemPlanLeaseID, tokenHash string, completedAt time.Time) (bool, error) {
	lease, ok := s.leases[id]
	if !ok || lease.LeaseTokenHash != tokenHash || lease.State != spine.WorkItemPlanLeaseStateActive || !lease.ExpiresAt.After(completedAt) {
		return false, nil
	}
	if s.markCompletedMiss {
		if s.markCompletedMissMode != "" {
			lease.State = s.markCompletedMissMode
			s.leases[id] = lease
		}
		return false, nil
	}
	lease.State = spine.WorkItemPlanLeaseStateCompleted
	lease.UpdatedAt = completedAt
	s.leases[id] = lease
	return true, nil
}

type fakeWorkItemPlanProposalStore struct {
	proposals     map[spine.WorkItemPlanProposalID]spine.WorkItemPlanProposal
	byPlan        map[spine.WorkItemPlanID]spine.WorkItemPlanProposalID
	createCtx     context.Context
	createErr     error
	getByPlanCtxs []context.Context
}

func newFakeWorkItemPlanProposalStore() *fakeWorkItemPlanProposalStore {
	return &fakeWorkItemPlanProposalStore{
		proposals: map[spine.WorkItemPlanProposalID]spine.WorkItemPlanProposal{},
		byPlan:    map[spine.WorkItemPlanID]spine.WorkItemPlanProposalID{},
	}
}

func (s *fakeWorkItemPlanProposalStore) Create(ctx context.Context, proposal spine.WorkItemPlanProposal) error {
	s.createCtx = ctx
	if s.createErr != nil {
		return s.createErr
	}
	s.proposals[proposal.ID] = proposal
	s.byPlan[proposal.PlanID] = proposal.ID
	return nil
}

func (s *fakeWorkItemPlanProposalStore) Get(_ context.Context, id spine.WorkItemPlanProposalID) (spine.WorkItemPlanProposal, bool, error) {
	proposal, ok := s.proposals[id]
	return proposal, ok, nil
}

func (s *fakeWorkItemPlanProposalStore) GetByPlanID(ctx context.Context, id spine.WorkItemPlanID) (spine.WorkItemPlanProposal, bool, error) {
	s.getByPlanCtxs = append(s.getByPlanCtxs, ctx)
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

func activeMembership(organizationID spine.OrganizationID) spine.OrganizationMembership {
	return spine.OrganizationMembership{
		ID:             "membership-1",
		OrganizationID: organizationID,
		UserID:         "requester",
		Role:           spine.OrganizationMembershipRoleMember,
		State:          spine.EntityStateActive,
		CreatedAt:      testTime(),
		UpdatedAt:      testTime(),
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

func acquireLease(t *testing.T, service *workitemplan.Service) spine.WorkItemPlanLeaseCreated {
	t.Helper()
	lease, ok, err := service.AcquireNextLease(context.Background(), spine.WorkItemPlanLeaseCreateRequest{
		LeasedBy: spine.ActorRef{Kind: "worker", ID: "planner-worker-1"},
	})
	if err != nil {
		t.Fatalf("AcquireNextLease() error = %v", err)
	}
	if !ok {
		t.Fatal("AcquireNextLease() ok = false, want true")
	}
	return lease
}

func validProposalRequest(approvedContractID string, lease spine.WorkItemPlanLeaseCreated) spine.WorkItemPlanProposalSubmitRequest {
	orderIndex := 0
	return spine.WorkItemPlanProposalSubmitRequest{
		LeaseID:            lease.ID,
		LeaseToken:         lease.LeaseToken,
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
	lease    int
	proposal int
	workItem int
	event    int
}

func (g *sequenceIDs) NewWorkItemPlanID() (spine.WorkItemPlanID, error) {
	g.plan++
	return spine.WorkItemPlanID("plan-1"), nil
}

func (g *sequenceIDs) NewWorkItemPlanLeaseID() (spine.WorkItemPlanLeaseID, error) {
	g.lease++
	return spine.WorkItemPlanLeaseID("lease-1"), nil
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
