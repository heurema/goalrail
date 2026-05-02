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
	"github.com/heurema/goalrail/apps/server/internal/eventlog"
	"github.com/heurema/goalrail/apps/server/internal/spine"
	"github.com/heurema/goalrail/apps/server/internal/store"
)

func TestCreateRollsBackSeededContractWhenDraftCreationFails(t *testing.T) {
	ctx := context.Background()
	goalStore := store.NewGoalStore()
	contractStore := store.NewContractStore()
	seedStore := store.NewContractSeedStore()
	draftStore := store.NewContractDraftStore()
	approvedStore := store.NewApprovedContractStore()
	events := eventlog.NewEventLog()
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
