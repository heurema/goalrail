package contract_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/approvedcontract"
	"github.com/heurema/goalrail/apps/server/internal/contract"
	"github.com/heurema/goalrail/apps/server/internal/contractdraft"
	"github.com/heurema/goalrail/apps/server/internal/contractseed"
	"github.com/heurema/goalrail/apps/server/internal/spine"
)

func TestCreateRollsBackSeededContractWhenDraftCreationFails(t *testing.T) {
	ctx := context.Background()
	goalStore := newFakeGoalStore()
	contractStore := newFakeContractStore()
	seedStore := newFakeContractSeedStore()
	draftStore := newFakeContractDraftStore()
	approvedStore := newFakeApprovedContractStore()
	events := newFakeEventLog()
	ids := &sequenceIDs{}

	goal := readyGoal()
	if err := goalStore.Create(ctx, goal); err != nil {
		t.Fatalf("goals.Create() error = %v", err)
	}

	seedService := contractseed.NewService(goalStore, contractStore, seedStore, events, fixedClock{now: testTime()}, ids)
	failingDraftService := &failingDraftService{err: errors.New("draft create failed")}
	approvalService := approvedcontract.NewService(draftStore, contractStore, approvedStore, events, fixedClock{now: testTime()}, ids)
	service := contract.NewService(contractStore, seedService, failingDraftService, approvalService)

	if _, err := service.Create(ctx, spine.ContractCreateRequest{GoalID: goal.ID}); err == nil {
		t.Fatal("Create() error = nil, want draft failure")
	}
	if _, ok, err := contractStore.GetByGoalID(ctx, goal.ID); err != nil {
		t.Fatalf("contracts.GetByGoalID() error = %v", err)
	} else if ok {
		t.Fatal("contract remains after failed facade create")
	}
	if _, ok, err := seedStore.GetByGoalID(ctx, goal.ID); err != nil {
		t.Fatalf("seeds.GetByGoalID() error = %v", err)
	} else if ok {
		t.Fatal("contract seed remains after failed facade create")
	}

	draftService := contractdraft.NewService(seedStore, contractStore, draftStore, events, fixedClock{now: testTime()}, ids)
	service = contract.NewService(contractStore, seedService, draftService, approvalService)
	created, err := service.Create(ctx, spine.ContractCreateRequest{GoalID: goal.ID})
	if err != nil {
		t.Fatalf("retry Create() error = %v", err)
	}
	if created.State != spine.ContractStateDraft {
		t.Fatalf("retry contract state = %q, want %q", created.State, spine.ContractStateDraft)
	}
	if created.CurrentSeedID == nil {
		t.Fatal("retry current_seed_id is nil")
	}
	if created.CurrentDraftID == nil {
		t.Fatal("retry current_draft_id is nil")
	}
	if _, ok, err := seedStore.GetByGoalID(ctx, goal.ID); err != nil {
		t.Fatalf("seeds.GetByGoalID() after retry error = %v", err)
	} else if !ok {
		t.Fatal("contract seed missing after retry")
	}
}

type failingDraftService struct {
	err error
}

func (s *failingDraftService) Create(context.Context, spine.ContractSeedID) (spine.ContractDraft, error) {
	return spine.ContractDraft{}, s.err
}

func (s *failingDraftService) Update(context.Context, spine.ContractDraftID, spine.ContractDraftUpdateRequest) (spine.ContractDraft, error) {
	return spine.ContractDraft{}, errors.New("unexpected update")
}

func (s *failingDraftService) MarkReadyForApproval(context.Context, spine.ContractDraftID, spine.ContractDraftReadyForApprovalRequest) (spine.ContractDraft, error) {
	return spine.ContractDraft{}, errors.New("unexpected mark ready")
}

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

type sequenceIDs struct {
	contract         int
	contractSeed     int
	contractDraft    int
	approvedContract int
	event            int
}

func (g *sequenceIDs) NewContractID() (spine.ContractID, error) {
	g.contract++
	return spine.ContractID(fmt.Sprintf("contract-%d", g.contract)), nil
}

func (g *sequenceIDs) NewContractSeedID() (spine.ContractSeedID, error) {
	g.contractSeed++
	return spine.ContractSeedID(fmt.Sprintf("contract-seed-%d", g.contractSeed)), nil
}

func (g *sequenceIDs) NewContractDraftID() (spine.ContractDraftID, error) {
	g.contractDraft++
	return spine.ContractDraftID(fmt.Sprintf("contract-draft-%d", g.contractDraft)), nil
}

func (g *sequenceIDs) NewApprovedContractID() (spine.ApprovedContractID, error) {
	g.approvedContract++
	return spine.ApprovedContractID(fmt.Sprintf("approved-contract-%d", g.approvedContract)), nil
}

func (g *sequenceIDs) NewEventID() (spine.EventID, error) {
	g.event++
	return spine.EventID(fmt.Sprintf("event-%d", g.event)), nil
}

func readyGoal() spine.Goal {
	return spine.Goal{
		ID:             "goal-1",
		IntakeID:       "intake-1",
		OrganizationID: "018f0000-0000-7000-8000-000000000002",
		ProjectID:      "018f0000-0000-7000-8000-000000000003",
		RepoBindingID:  "018f0000-0000-7000-8000-000000000004",
		Title:          "Refactor CSV export filters",
		Summary:        "Current code duplicates filter logic.",
		ScopeHint:      "Refactor duplicate CSV export filter logic",
		AcceptanceHint: "Existing CSV export behavior is preserved",
		RequestAuthor:  spine.ActorRef{Kind: "user", ID: "dev_1"},
		IntentOwner:    spine.ActorRef{Kind: "user", ID: "dev_1"},
		State:          spine.GoalStateReadyForContractSeed,
		CreatedAt:      testTime(),
	}
}

func testTime() time.Time {
	return time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
}

type fakeGoalStore struct {
	goals map[spine.GoalID]spine.Goal
}

func newFakeGoalStore() *fakeGoalStore {
	return &fakeGoalStore{goals: map[spine.GoalID]spine.Goal{}}
}

func (s *fakeGoalStore) Create(_ context.Context, goal spine.Goal) error {
	s.goals[goal.ID] = goal
	return nil
}

func (s *fakeGoalStore) Get(_ context.Context, id spine.GoalID) (spine.Goal, bool, error) {
	goal, ok := s.goals[id]
	return goal, ok, nil
}

type fakeContractStore struct {
	contracts map[spine.ContractID]spine.Contract
	byGoal    map[spine.GoalID]spine.ContractID
}

func newFakeContractStore() *fakeContractStore {
	return &fakeContractStore{
		contracts: map[spine.ContractID]spine.Contract{},
		byGoal:    map[spine.GoalID]spine.ContractID{},
	}
}

func (s *fakeContractStore) Create(_ context.Context, contract spine.Contract) error {
	s.contracts[contract.ID] = contract
	s.byGoal[contract.GoalID] = contract.ID
	return nil
}

func (s *fakeContractStore) Get(_ context.Context, id spine.ContractID) (spine.Contract, bool, error) {
	contract, ok := s.contracts[id]
	return contract, ok, nil
}

func (s *fakeContractStore) GetByGoalID(_ context.Context, id spine.GoalID) (spine.Contract, bool, error) {
	contractID, ok := s.byGoal[id]
	if !ok {
		return spine.Contract{}, false, nil
	}
	contract, ok := s.contracts[contractID]
	return contract, ok, nil
}

func (s *fakeContractStore) Delete(_ context.Context, id spine.ContractID) error {
	contract, ok := s.contracts[id]
	if !ok {
		return nil
	}
	delete(s.contracts, id)
	if s.byGoal[contract.GoalID] == id {
		delete(s.byGoal, contract.GoalID)
	}
	return nil
}

func (s *fakeContractStore) MarkDraftCreated(_ context.Context, id spine.ContractID, draftID spine.ContractDraftID, updatedAt time.Time) error {
	contract, ok := s.contracts[id]
	if !ok {
		return nil
	}
	contract.State = spine.ContractStateDraft
	contract.CurrentDraftID = &draftID
	contract.UpdatedAt = updatedAt.UTC()
	s.contracts[id] = contract
	return nil
}

func (s *fakeContractStore) MarkReadyForApproval(_ context.Context, id spine.ContractID, updatedAt time.Time) error {
	contract, ok := s.contracts[id]
	if !ok {
		return nil
	}
	contract.State = spine.ContractStateReadyForApproval
	contract.UpdatedAt = updatedAt.UTC()
	s.contracts[id] = contract
	return nil
}

func (s *fakeContractStore) MarkApproved(_ context.Context, id spine.ContractID, approvedID spine.ApprovedContractID, updatedAt time.Time) error {
	contract, ok := s.contracts[id]
	if !ok {
		return nil
	}
	contract.State = spine.ContractStateApproved
	contract.ApprovedSnapshotID = &approvedID
	contract.UpdatedAt = updatedAt.UTC()
	s.contracts[id] = contract
	return nil
}

type fakeContractSeedStore struct {
	seeds  map[spine.ContractSeedID]spine.ContractSeed
	byGoal map[spine.GoalID]spine.ContractSeedID
}

func newFakeContractSeedStore() *fakeContractSeedStore {
	return &fakeContractSeedStore{
		seeds:  map[spine.ContractSeedID]spine.ContractSeed{},
		byGoal: map[spine.GoalID]spine.ContractSeedID{},
	}
}

func (s *fakeContractSeedStore) Create(_ context.Context, seed spine.ContractSeed) error {
	s.seeds[seed.ID] = seed
	s.byGoal[seed.GoalID] = seed.ID
	return nil
}

func (s *fakeContractSeedStore) Get(_ context.Context, id spine.ContractSeedID) (spine.ContractSeed, bool, error) {
	seed, ok := s.seeds[id]
	return seed, ok, nil
}

func (s *fakeContractSeedStore) GetByGoalID(_ context.Context, id spine.GoalID) (spine.ContractSeed, bool, error) {
	seedID, ok := s.byGoal[id]
	if !ok {
		return spine.ContractSeed{}, false, nil
	}
	seed, ok := s.seeds[seedID]
	return seed, ok, nil
}

func (s *fakeContractSeedStore) Delete(_ context.Context, id spine.ContractSeedID) error {
	seed, ok := s.seeds[id]
	if !ok {
		return nil
	}
	delete(s.seeds, id)
	if s.byGoal[seed.GoalID] == id {
		delete(s.byGoal, seed.GoalID)
	}
	return nil
}

type fakeContractDraftStore struct {
	drafts map[spine.ContractDraftID]spine.ContractDraft
	bySeed map[spine.ContractSeedID]spine.ContractDraftID
}

func newFakeContractDraftStore() *fakeContractDraftStore {
	return &fakeContractDraftStore{
		drafts: map[spine.ContractDraftID]spine.ContractDraft{},
		bySeed: map[spine.ContractSeedID]spine.ContractDraftID{},
	}
}

func (s *fakeContractDraftStore) Create(_ context.Context, draft spine.ContractDraft) error {
	s.drafts[draft.ID] = draft
	s.bySeed[draft.ContractSeedID] = draft.ID
	return nil
}

func (s *fakeContractDraftStore) Update(_ context.Context, draft spine.ContractDraft) error {
	s.drafts[draft.ID] = draft
	return nil
}

func (s *fakeContractDraftStore) MarkReadyForApproval(_ context.Context, draft spine.ContractDraft) error {
	draft.State = spine.ContractDraftStateReadyForApproval
	s.drafts[draft.ID] = draft
	return nil
}

func (s *fakeContractDraftStore) Get(_ context.Context, id spine.ContractDraftID) (spine.ContractDraft, bool, error) {
	draft, ok := s.drafts[id]
	return draft, ok, nil
}

func (s *fakeContractDraftStore) GetByContractSeedID(_ context.Context, id spine.ContractSeedID) (spine.ContractDraft, bool, error) {
	draftID, ok := s.bySeed[id]
	if !ok {
		return spine.ContractDraft{}, false, nil
	}
	draft, ok := s.drafts[draftID]
	return draft, ok, nil
}

type fakeApprovedContractStore struct {
	approved map[spine.ApprovedContractID]spine.ApprovedContract
	byDraft  map[spine.ContractDraftID]spine.ApprovedContractID
}

func newFakeApprovedContractStore() *fakeApprovedContractStore {
	return &fakeApprovedContractStore{
		approved: map[spine.ApprovedContractID]spine.ApprovedContract{},
		byDraft:  map[spine.ContractDraftID]spine.ApprovedContractID{},
	}
}

func (s *fakeApprovedContractStore) Create(_ context.Context, approved spine.ApprovedContract) error {
	s.approved[approved.ID] = approved
	s.byDraft[approved.ContractDraftID] = approved.ID
	return nil
}

func (s *fakeApprovedContractStore) Get(_ context.Context, id spine.ApprovedContractID) (spine.ApprovedContract, bool, error) {
	approved, ok := s.approved[id]
	return approved, ok, nil
}

func (s *fakeApprovedContractStore) GetByContractDraftID(_ context.Context, id spine.ContractDraftID) (spine.ApprovedContract, bool, error) {
	approvedID, ok := s.byDraft[id]
	if !ok {
		return spine.ApprovedContract{}, false, nil
	}
	approved, ok := s.approved[approvedID]
	return approved, ok, nil
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
