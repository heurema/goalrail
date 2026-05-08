package contract_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/approvedcontract"
	"github.com/heurema/goalrail/apps/server/internal/contract"
	"github.com/heurema/goalrail/apps/server/internal/contractdraft"
	"github.com/heurema/goalrail/apps/server/internal/contractseed"
	"github.com/heurema/goalrail/apps/server/internal/spine"
)

func TestCreateUsesTransactionRunnerRollbackWhenDraftCreationFails(t *testing.T) {
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

	txRunner := &fakeTransactionRunner{rollback: func() {
		contractStore.contracts = map[spine.ContractID]spine.Contract{}
		contractStore.byGoal = map[spine.GoalID]spine.ContractID{}
		seedStore.seeds = map[spine.ContractSeedID]spine.ContractSeed{}
		seedStore.byGoal = map[spine.GoalID]spine.ContractSeedID{}
	}}
	seedService := contractseed.NewService(goalStore, contractStore, seedStore, events, txRunner, fixedClock{now: testTime()}, ids)
	failingDraftService := &failingDraftService{err: errors.New("draft create failed")}
	approvalService := approvedcontract.NewService(draftStore, contractStore, approvedStore, events, txRunner, fixedClock{now: testTime()}, ids)
	service := contract.NewService(goalStore, contractStore, seedService, failingDraftService, approvalService, txRunner)

	if _, _, err := service.Create(ctx, spine.ContractCreateRequest{GoalID: goal.ID}, activeMembership(goal.OrganizationID)); err == nil {
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

	draftService := contractdraft.NewService(seedStore, contractStore, draftStore, events, txRunner, fixedClock{now: testTime()}, ids)
	service = contract.NewService(goalStore, contractStore, seedService, draftService, approvalService, &fakeTransactionRunner{})
	created, createdNew, err := service.Create(ctx, spine.ContractCreateRequest{GoalID: goal.ID}, activeMembership(goal.OrganizationID))
	if err != nil {
		t.Fatalf("retry Create() error = %v", err)
	}
	if !createdNew {
		t.Fatal("retry Create() createdNew = false, want true")
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

func TestCreateUsesRequiredTransactionRunner(t *testing.T) {
	ctx := context.Background()
	goalStore := newFakeGoalStore()
	contractStore := newFakeContractStore()
	seedStore := newFakeContractSeedStore()
	draftStore := newFakeContractDraftStore()
	approvedStore := newFakeApprovedContractStore()
	events := newFakeEventLog()
	ids := &sequenceIDs{}
	txRunner := &fakeTransactionRunner{}

	goal := readyGoal()
	if err := goalStore.Create(ctx, goal); err != nil {
		t.Fatalf("goals.Create() error = %v", err)
	}

	seedService := contractseed.NewService(goalStore, contractStore, seedStore, events, txRunner, fixedClock{now: testTime()}, ids)
	draftService := contractdraft.NewService(seedStore, contractStore, draftStore, events, txRunner, fixedClock{now: testTime()}, ids)
	approvalService := approvedcontract.NewService(draftStore, contractStore, approvedStore, events, txRunner, fixedClock{now: testTime()}, ids)
	service := contract.NewService(goalStore, contractStore, seedService, draftService, approvalService, txRunner)

	created, createdNew, err := service.Create(ctx, spine.ContractCreateRequest{GoalID: goal.ID}, activeMembership(goal.OrganizationID))
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if !createdNew {
		t.Fatal("Create() createdNew = false, want true")
	}
	if txRunner.calls != 3 {
		t.Fatalf("TxRunner calls = %d, want 3", txRunner.calls)
	}
	if !contractStore.createSawTransaction {
		t.Fatal("contract store Create did not run inside transaction runner")
	}
	if !seedStore.createSawTransaction {
		t.Fatal("seed store Create did not run inside transaction runner")
	}
	if !draftStore.createSawTransaction {
		t.Fatal("draft store Create did not run inside transaction runner")
	}
	if !contractStore.markDraftCreatedSawTransaction {
		t.Fatal("contract store MarkDraftCreated did not run inside transaction runner")
	}
	if events.transactionalAppends != 2 {
		t.Fatalf("transactional event appends = %d, want 2", events.transactionalAppends)
	}
	if created.State != spine.ContractStateDraft {
		t.Fatalf("contract state = %q, want %q", created.State, spine.ContractStateDraft)
	}
	if created.CurrentSeedID == nil {
		t.Fatal("current_seed_id is nil")
	}
	if created.CurrentDraftID == nil {
		t.Fatal("current_draft_id is nil")
	}
}

func TestCreateReturnsExistingContractForGoal(t *testing.T) {
	ctx := context.Background()
	goalStore := newFakeGoalStore()
	contractStore := newFakeContractStore()
	seedStore := newFakeContractSeedStore()
	draftStore := newFakeContractDraftStore()
	approvedStore := newFakeApprovedContractStore()
	events := newFakeEventLog()
	ids := &sequenceIDs{}
	txRunner := &fakeTransactionRunner{}

	goal := readyGoal()
	if err := goalStore.Create(ctx, goal); err != nil {
		t.Fatalf("goals.Create() error = %v", err)
	}
	seedService := contractseed.NewService(goalStore, contractStore, seedStore, events, txRunner, fixedClock{now: testTime()}, ids)
	draftService := contractdraft.NewService(seedStore, contractStore, draftStore, events, txRunner, fixedClock{now: testTime()}, ids)
	approvalService := approvedcontract.NewService(draftStore, contractStore, approvedStore, events, txRunner, fixedClock{now: testTime()}, ids)
	service := contract.NewService(goalStore, contractStore, seedService, draftService, approvalService, txRunner)

	first, createdNew, err := service.Create(ctx, spine.ContractCreateRequest{GoalID: goal.ID}, activeMembership(goal.OrganizationID))
	if err != nil {
		t.Fatalf("first Create() error = %v", err)
	}
	if !createdNew {
		t.Fatal("first Create() createdNew = false, want true")
	}
	second, createdNew, err := service.Create(ctx, spine.ContractCreateRequest{GoalID: goal.ID}, activeMembership(goal.OrganizationID))
	if err != nil {
		t.Fatalf("second Create() error = %v", err)
	}
	if createdNew {
		t.Fatal("second Create() createdNew = true, want false")
	}
	if second.ID != first.ID || second.CurrentDraftID == nil || *second.CurrentDraftID != *first.CurrentDraftID {
		t.Fatalf("second contract = %#v, want same draft handle as %#v", second, first)
	}
	if len(contractStore.contracts) != 1 || len(seedStore.seeds) != 1 || len(draftStore.drafts) != 1 {
		t.Fatalf("stored counts contracts/seeds/drafts = %d/%d/%d, want 1/1/1", len(contractStore.contracts), len(seedStore.seeds), len(draftStore.drafts))
	}
	if got := countEventType(events.Events(), contractseed.EventTypeContractSeedCreated); got != 1 {
		t.Fatalf("contract_seed.created events = %d, want 1", got)
	}
	if got := countEventType(events.Events(), contractdraft.EventTypeContractDraftCreated); got != 1 {
		t.Fatalf("contract_draft.created events = %d, want 1", got)
	}
}

func TestListScopesContractsToActiveMembershipOrganization(t *testing.T) {
	ctx := context.Background()
	contractStore := newFakeContractStore()
	service := newContractServiceWithStore(contractStore)
	orgID := spine.OrganizationID("018f0000-0000-7000-8000-000000000002")
	if err := contractStore.Create(ctx, storedContract("018f0000-0000-7000-8000-000000000c01", orgID, "018f0000-0000-7000-8000-000000000201", spine.ContractStateDraft)); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := contractStore.Create(ctx, storedContract("018f0000-0000-7000-8000-000000000c02", "018f0000-0000-7000-8000-000000009999", "018f0000-0000-7000-8000-000000000202", spine.ContractStateApproved)); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	result, err := service.List(ctx, contract.ListInput{Membership: activeMembership(orgID)})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(result.Contracts) != 1 {
		t.Fatalf("contracts len = %d, want 1: %#v", len(result.Contracts), result.Contracts)
	}
	if result.Contracts[0].OrganizationID != orgID {
		t.Fatalf("contract organization = %q, want %q", result.Contracts[0].OrganizationID, orgID)
	}
	if result.Limit != 50 {
		t.Fatalf("limit = %d, want default 50", result.Limit)
	}
}

func TestListAppliesFiltersAndMaxLimit(t *testing.T) {
	ctx := context.Background()
	contractStore := newFakeContractStore()
	service := newContractServiceWithStore(contractStore)
	orgID := spine.OrganizationID("018f0000-0000-7000-8000-000000000002")
	matching := storedContract("018f0000-0000-7000-8000-000000000c01", orgID, "018f0000-0000-7000-8000-000000000201", spine.ContractStateReadyForApproval)
	if err := contractStore.Create(ctx, matching); err != nil {
		t.Fatalf("Create() matching error = %v", err)
	}
	for _, other := range []spine.Contract{
		storedContract("018f0000-0000-7000-8000-000000000c02", orgID, "018f0000-0000-7000-8000-000000000202", spine.ContractStateReadyForApproval),
		storedContract("018f0000-0000-7000-8000-000000000c03", orgID, "018f0000-0000-7000-8000-000000000201", spine.ContractStateDraft),
	} {
		if err := contractStore.Create(ctx, other); err != nil {
			t.Fatalf("Create() other error = %v", err)
		}
	}

	result, err := service.List(ctx, contract.ListInput{
		Membership:    activeMembership(orgID),
		ProjectID:     matching.ProjectID,
		RepoBindingID: matching.RepoBindingID,
		GoalID:        matching.GoalID,
		State:         spine.ContractStateReadyForApproval,
		Limit:         100,
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(result.Contracts) != 1 || result.Contracts[0].ID != matching.ID {
		t.Fatalf("contracts = %#v, want matching contract only", result.Contracts)
	}
	if result.Limit != 100 || contractStore.lastListFilter.Limit != 100 {
		t.Fatalf("limit result/filter = %d/%d, want 100/100", result.Limit, contractStore.lastListFilter.Limit)
	}
}

func TestListAllowsActiveReaderRoles(t *testing.T) {
	ctx := context.Background()
	contractStore := newFakeContractStore()
	service := newContractServiceWithStore(contractStore)
	for _, role := range []spine.OrganizationMembershipRole{
		spine.OrganizationMembershipRoleViewer,
		spine.OrganizationMembershipRoleMember,
		spine.OrganizationMembershipRoleAdmin,
		spine.OrganizationMembershipRoleOwner,
	} {
		t.Run(string(role), func(t *testing.T) {
			membership := activeMembership("018f0000-0000-7000-8000-000000000002")
			membership.Role = role
			if _, err := service.List(ctx, contract.ListInput{Membership: membership}); err != nil {
				t.Fatalf("List() error = %v, want nil", err)
			}
		})
	}
}

func TestListRejectsInactiveMembershipAndInvalidFilters(t *testing.T) {
	ctx := context.Background()
	service := newContractServiceWithStore(newFakeContractStore())
	tests := []struct {
		name  string
		input contract.ListInput
	}{
		{
			name: "inactive membership",
			input: contract.ListInput{Membership: spine.OrganizationMembership{
				OrganizationID: "018f0000-0000-7000-8000-000000000002",
				State:          spine.EntityStateInactive,
			}},
		},
		{name: "bad project id", input: contract.ListInput{Membership: activeMembership("018f0000-0000-7000-8000-000000000002"), ProjectID: "not-a-uuid"}},
		{name: "bad repo binding id", input: contract.ListInput{Membership: activeMembership("018f0000-0000-7000-8000-000000000002"), RepoBindingID: "not-a-uuid"}},
		{name: "bad goal id", input: contract.ListInput{Membership: activeMembership("018f0000-0000-7000-8000-000000000002"), GoalID: "not-a-uuid"}},
		{name: "invalid state", input: contract.ListInput{Membership: activeMembership("018f0000-0000-7000-8000-000000000002"), State: "blocked"}},
		{name: "negative limit", input: contract.ListInput{Membership: activeMembership("018f0000-0000-7000-8000-000000000002"), Limit: -1}},
		{name: "too large limit", input: contract.ListInput{Membership: activeMembership("018f0000-0000-7000-8000-000000000002"), Limit: 101}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := service.List(ctx, tt.input); err == nil {
				t.Fatal("List() error = nil, want validation or membership error")
			}
		})
	}
}

func TestCreateRejectsOrganizationMismatchBeforeMutation(t *testing.T) {
	ctx := context.Background()
	goalStore := newFakeGoalStore()
	contractStore := newFakeContractStore()
	seedStore := newFakeContractSeedStore()
	draftStore := newFakeContractDraftStore()
	approvedStore := newFakeApprovedContractStore()
	events := newFakeEventLog()
	ids := &sequenceIDs{}
	txRunner := &fakeTransactionRunner{}

	goal := readyGoal()
	if err := goalStore.Create(ctx, goal); err != nil {
		t.Fatalf("goals.Create() error = %v", err)
	}
	seedService := contractseed.NewService(goalStore, contractStore, seedStore, events, txRunner, fixedClock{now: testTime()}, ids)
	draftService := contractdraft.NewService(seedStore, contractStore, draftStore, events, txRunner, fixedClock{now: testTime()}, ids)
	approvalService := approvedcontract.NewService(draftStore, contractStore, approvedStore, events, txRunner, fixedClock{now: testTime()}, ids)
	service := contract.NewService(goalStore, contractStore, seedService, draftService, approvalService, txRunner)

	_, _, err := service.Create(ctx, spine.ContractCreateRequest{GoalID: goal.ID}, activeMembership("018f0000-0000-7000-8000-000000009999"))
	if !errors.Is(err, contract.ErrOrganizationForbidden) {
		t.Fatalf("Create() error = %v, want ErrOrganizationForbidden", err)
	}
	if len(contractStore.contracts) != 0 || len(seedStore.seeds) != 0 || len(draftStore.drafts) != 0 || len(events.Events()) != 0 {
		t.Fatalf("mutation happened despite org mismatch")
	}
}

func TestCreateRejectsExpectedProjectOrRepoBindingMismatchBeforeMutation(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name  string
		input func(spine.Goal) spine.ContractCreateRequest
		want  error
	}{
		{
			name: "project",
			input: func(goal spine.Goal) spine.ContractCreateRequest {
				return spine.ContractCreateRequest{
					GoalID:        goal.ID,
					ProjectID:     "018f0000-0000-7000-8000-000000009998",
					RepoBindingID: goal.RepoBindingID,
				}
			},
			want: contract.ErrProjectMismatch,
		},
		{
			name: "repo binding",
			input: func(goal spine.Goal) spine.ContractCreateRequest {
				return spine.ContractCreateRequest{
					GoalID:        goal.ID,
					ProjectID:     goal.ProjectID,
					RepoBindingID: "018f0000-0000-7000-8000-000000009999",
				}
			},
			want: contract.ErrRepoBindingMismatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			goalStore := newFakeGoalStore()
			contractStore := newFakeContractStore()
			seedStore := newFakeContractSeedStore()
			draftStore := newFakeContractDraftStore()
			approvedStore := newFakeApprovedContractStore()
			events := newFakeEventLog()
			ids := &sequenceIDs{}
			txRunner := &fakeTransactionRunner{}

			goal := readyGoal()
			if err := goalStore.Create(ctx, goal); err != nil {
				t.Fatalf("goals.Create() error = %v", err)
			}
			seedService := contractseed.NewService(goalStore, contractStore, seedStore, events, txRunner, fixedClock{now: testTime()}, ids)
			draftService := contractdraft.NewService(seedStore, contractStore, draftStore, events, txRunner, fixedClock{now: testTime()}, ids)
			approvalService := approvedcontract.NewService(draftStore, contractStore, approvedStore, events, txRunner, fixedClock{now: testTime()}, ids)
			service := contract.NewService(goalStore, contractStore, seedService, draftService, approvalService, txRunner)

			_, _, err := service.Create(ctx, tt.input(goal), activeMembership(goal.OrganizationID))
			if !errors.Is(err, tt.want) {
				t.Fatalf("Create() error = %v, want %v", err, tt.want)
			}
			if len(contractStore.contracts) != 0 || len(seedStore.seeds) != 0 || len(draftStore.drafts) != 0 || len(events.Events()) != 0 {
				t.Fatalf("mutation happened despite expected context mismatch")
			}
		})
	}
}

func TestUpdateDraftRejectsOrganizationMismatchBeforeMutation(t *testing.T) {
	ctx := context.Background()
	goalStore := newFakeGoalStore()
	contractStore := newFakeContractStore()
	seedStore := newFakeContractSeedStore()
	draftStore := newFakeContractDraftStore()
	approvedStore := newFakeApprovedContractStore()
	events := newFakeEventLog()
	ids := &sequenceIDs{}
	txRunner := &fakeTransactionRunner{}

	goal := readyGoal()
	if err := goalStore.Create(ctx, goal); err != nil {
		t.Fatalf("goals.Create() error = %v", err)
	}
	seedService := contractseed.NewService(goalStore, contractStore, seedStore, events, txRunner, fixedClock{now: testTime()}, ids)
	draftService := contractdraft.NewService(seedStore, contractStore, draftStore, events, txRunner, fixedClock{now: testTime()}, ids)
	approvalService := approvedcontract.NewService(draftStore, contractStore, approvedStore, events, txRunner, fixedClock{now: testTime()}, ids)
	service := contract.NewService(goalStore, contractStore, seedService, draftService, approvalService, txRunner)

	created, _, err := service.Create(ctx, spine.ContractCreateRequest{GoalID: goal.ID}, activeMembership(goal.OrganizationID))
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	eventCountBefore := len(events.Events())
	_, err = service.UpdateDraft(ctx, created.ID, draftUpdateRequest(t, `{"title": "Reviewed draft title"}`), activeMembership("018f0000-0000-7000-8000-000000009999"))
	if !errors.Is(err, contract.ErrOrganizationForbidden) {
		t.Fatalf("UpdateDraft() error = %v, want ErrOrganizationForbidden", err)
	}
	if got := len(events.Events()); got != eventCountBefore {
		t.Fatalf("events = %d, want %d without update mutation", got, eventCountBefore)
	}
	if draftStore.updateSawTransaction {
		t.Fatal("draft update ran despite organization mismatch")
	}
}

func TestUpdateDraftRejectsExpectedProjectOrRepoBindingMismatchBeforeMutation(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name  string
		input func(spine.Goal) spine.ContractDraftUpdateRequest
		want  error
	}{
		{
			name: "project",
			input: func(goal spine.Goal) spine.ContractDraftUpdateRequest {
				request := draftUpdateRequest(t, `{"title": "Reviewed draft title"}`)
				request.ProjectID = "018f0000-0000-7000-8000-000000009998"
				request.RepoBindingID = goal.RepoBindingID
				return request
			},
			want: contract.ErrProjectMismatch,
		},
		{
			name: "repo binding",
			input: func(goal spine.Goal) spine.ContractDraftUpdateRequest {
				request := draftUpdateRequest(t, `{"title": "Reviewed draft title"}`)
				request.ProjectID = goal.ProjectID
				request.RepoBindingID = "018f0000-0000-7000-8000-000000009999"
				return request
			},
			want: contract.ErrRepoBindingMismatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			goalStore := newFakeGoalStore()
			contractStore := newFakeContractStore()
			seedStore := newFakeContractSeedStore()
			draftStore := newFakeContractDraftStore()
			approvedStore := newFakeApprovedContractStore()
			events := newFakeEventLog()
			ids := &sequenceIDs{}
			txRunner := &fakeTransactionRunner{}

			goal := readyGoal()
			if err := goalStore.Create(ctx, goal); err != nil {
				t.Fatalf("goals.Create() error = %v", err)
			}
			seedService := contractseed.NewService(goalStore, contractStore, seedStore, events, txRunner, fixedClock{now: testTime()}, ids)
			draftService := contractdraft.NewService(seedStore, contractStore, draftStore, events, txRunner, fixedClock{now: testTime()}, ids)
			approvalService := approvedcontract.NewService(draftStore, contractStore, approvedStore, events, txRunner, fixedClock{now: testTime()}, ids)
			service := contract.NewService(goalStore, contractStore, seedService, draftService, approvalService, txRunner)

			created, _, err := service.Create(ctx, spine.ContractCreateRequest{GoalID: goal.ID}, activeMembership(goal.OrganizationID))
			if err != nil {
				t.Fatalf("Create() error = %v", err)
			}
			eventCountBefore := len(events.Events())
			_, err = service.UpdateDraft(ctx, created.ID, tt.input(goal), activeMembership(goal.OrganizationID))
			if !errors.Is(err, tt.want) {
				t.Fatalf("UpdateDraft() error = %v, want %v", err, tt.want)
			}
			if got := len(events.Events()); got != eventCountBefore {
				t.Fatalf("events = %d, want %d without update mutation", got, eventCountBefore)
			}
			if draftStore.updateSawTransaction {
				t.Fatal("draft update ran despite expected context mismatch")
			}
		})
	}
}

func TestUpdateDraftUsesRequiredTransactionRunner(t *testing.T) {
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

	subServiceTxRunner := &fakeTransactionRunner{}
	seedService := contractseed.NewService(goalStore, contractStore, seedStore, events, subServiceTxRunner, fixedClock{now: testTime()}, ids)
	draftService := contractdraft.NewService(seedStore, contractStore, draftStore, events, subServiceTxRunner, fixedClock{now: testTime()}, ids)
	approvalService := approvedcontract.NewService(draftStore, contractStore, approvedStore, events, subServiceTxRunner, fixedClock{now: testTime()}, ids)
	createService := contract.NewService(goalStore, contractStore, seedService, draftService, approvalService, &fakeTransactionRunner{})
	created, _, err := createService.Create(ctx, spine.ContractCreateRequest{GoalID: goal.ID}, activeMembership(goal.OrganizationID))
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	transactionalAppendsBeforeUpdate := events.transactionalAppends

	txRunner := &fakeTransactionRunner{}
	service := contract.NewService(goalStore, contractStore, seedService, draftService, approvalService, txRunner)
	updated, err := service.UpdateDraft(ctx, created.ID, draftUpdateRequest(t, `{"title": "Reviewed draft title"}`), activeMembership(goal.OrganizationID))
	if err != nil {
		t.Fatalf("UpdateDraft() error = %v", err)
	}
	if txRunner.calls != 1 {
		t.Fatalf("TxRunner calls = %d, want 1", txRunner.calls)
	}
	if !draftStore.updateSawTransaction {
		t.Fatal("draft store Update did not run inside transaction runner")
	}
	if got := events.transactionalAppends - transactionalAppendsBeforeUpdate; got != 1 {
		t.Fatalf("transactional event appends = %d, want 1", got)
	}
	if updated.State != spine.ContractStateDraft {
		t.Fatalf("contract state = %q, want %q", updated.State, spine.ContractStateDraft)
	}
	if updated.CurrentDraftID == nil {
		t.Fatal("current_draft_id is nil")
	}
	draft, ok, err := draftStore.Get(ctx, *updated.CurrentDraftID)
	if err != nil {
		t.Fatalf("drafts.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("updated draft not found")
	}
	if draft.Title != "Reviewed draft title" {
		t.Fatalf("draft title = %q, want reviewed title", draft.Title)
	}
	if got := countEventType(events.Events(), contractdraft.EventTypeContractDraftUpdated); got != 1 {
		t.Fatalf("contract_draft.updated events = %d, want 1", got)
	}
}

func TestSubmitForApprovalUsesRequiredTransactionRunner(t *testing.T) {
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

	subServiceTxRunner := &fakeTransactionRunner{}
	seedService := contractseed.NewService(goalStore, contractStore, seedStore, events, subServiceTxRunner, fixedClock{now: testTime()}, ids)
	draftService := contractdraft.NewService(seedStore, contractStore, draftStore, events, subServiceTxRunner, fixedClock{now: testTime()}, ids)
	approvalService := approvedcontract.NewService(draftStore, contractStore, approvedStore, events, subServiceTxRunner, fixedClock{now: testTime()}, ids)
	createService := contract.NewService(goalStore, contractStore, seedService, draftService, approvalService, &fakeTransactionRunner{})
	created, _, err := createService.Create(ctx, spine.ContractCreateRequest{GoalID: goal.ID}, activeMembership(goal.OrganizationID))
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	transactionalAppendsBeforeSubmit := events.transactionalAppends

	txRunner := &fakeTransactionRunner{}
	service := contract.NewService(goalStore, contractStore, seedService, draftService, approvalService, txRunner)
	updated, err := service.SubmitForApproval(ctx, created.ID, draftReadyForApprovalRequest(), activeMembership(goal.OrganizationID))
	if err != nil {
		t.Fatalf("SubmitForApproval() error = %v", err)
	}
	if txRunner.calls != 1 {
		t.Fatalf("TxRunner calls = %d, want 1", txRunner.calls)
	}
	if !draftStore.markReadyForApprovalSawTransaction {
		t.Fatal("draft store MarkReadyForApproval did not run inside transaction runner")
	}
	if !contractStore.markReadyForApprovalSawTransaction {
		t.Fatal("contract store MarkReadyForApproval did not run inside transaction runner")
	}
	if got := events.transactionalAppends - transactionalAppendsBeforeSubmit; got != 1 {
		t.Fatalf("transactional event appends = %d, want 1", got)
	}
	if updated.State != spine.ContractStateReadyForApproval {
		t.Fatalf("contract state = %q, want %q", updated.State, spine.ContractStateReadyForApproval)
	}
	if updated.CurrentDraftID == nil {
		t.Fatal("current_draft_id is nil")
	}
	draft, ok, err := draftStore.Get(ctx, *updated.CurrentDraftID)
	if err != nil {
		t.Fatalf("drafts.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("updated draft not found")
	}
	if draft.State != spine.ContractDraftStateReadyForApproval {
		t.Fatalf("draft state = %q, want %q", draft.State, spine.ContractDraftStateReadyForApproval)
	}
	if got := countEventType(events.Events(), contractdraft.EventTypeContractDraftMarkedReadyForApproval); got != 1 {
		t.Fatalf("contract_draft.marked_ready_for_approval events = %d, want 1", got)
	}
}

func TestApproveUsesRequiredTransactionRunner(t *testing.T) {
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

	subServiceTxRunner := &fakeTransactionRunner{}
	seedService := contractseed.NewService(goalStore, contractStore, seedStore, events, subServiceTxRunner, fixedClock{now: testTime()}, ids)
	draftService := contractdraft.NewService(seedStore, contractStore, draftStore, events, subServiceTxRunner, fixedClock{now: testTime()}, ids)
	approvalService := approvedcontract.NewService(draftStore, contractStore, approvedStore, events, subServiceTxRunner, fixedClock{now: testTime()}, ids)
	createService := contract.NewService(goalStore, contractStore, seedService, draftService, approvalService, &fakeTransactionRunner{})
	created, _, err := createService.Create(ctx, spine.ContractCreateRequest{GoalID: goal.ID}, activeMembership(goal.OrganizationID))
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	submitted, err := createService.SubmitForApproval(ctx, created.ID, draftReadyForApprovalRequest(), activeMembership(goal.OrganizationID))
	if err != nil {
		t.Fatalf("SubmitForApproval() error = %v", err)
	}
	transactionalAppendsBeforeApprove := events.transactionalAppends

	txRunner := &fakeTransactionRunner{}
	service := contract.NewService(goalStore, contractStore, seedService, draftService, approvalService, txRunner)
	approved, err := service.Approve(ctx, submitted.ID, approveRequest(), activeMembership(goal.OrganizationID))
	if err != nil {
		t.Fatalf("Approve() error = %v", err)
	}
	if txRunner.calls != 1 {
		t.Fatalf("TxRunner calls = %d, want 1", txRunner.calls)
	}
	if !approvedStore.createSawTransaction {
		t.Fatal("approved contract store Create did not run inside transaction runner")
	}
	if !contractStore.markApprovedSawTransaction {
		t.Fatal("contract store MarkApproved did not run inside transaction runner")
	}
	if got := events.transactionalAppends - transactionalAppendsBeforeApprove; got != 1 {
		t.Fatalf("transactional event appends = %d, want 1", got)
	}
	if approved.State != spine.ContractStateApproved {
		t.Fatalf("contract state = %q, want %q", approved.State, spine.ContractStateApproved)
	}
	if approved.ApprovedSnapshotID == nil {
		t.Fatal("approved_snapshot_id is nil")
	}
	snapshot, ok, err := approvedStore.Get(ctx, *approved.ApprovedSnapshotID)
	if err != nil {
		t.Fatalf("approved.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("approved snapshot not found")
	}
	if snapshot.ContractID != approved.ID {
		t.Fatalf("approved snapshot contract_id = %q, want %q", snapshot.ContractID, approved.ID)
	}
	if got := countEventType(events.Events(), approvedcontract.EventTypeContractApproved); got != 1 {
		t.Fatalf("contract.approved events = %d, want 1", got)
	}
}

func TestContractLifecycleTransitionsUseRequiredTransactionRunner(t *testing.T) {
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

	subServiceTxRunner := &fakeTransactionRunner{}
	seedService := contractseed.NewService(goalStore, contractStore, seedStore, events, subServiceTxRunner, fixedClock{now: testTime()}, ids)
	draftService := contractdraft.NewService(seedStore, contractStore, draftStore, events, subServiceTxRunner, fixedClock{now: testTime()}, ids)
	approvalService := approvedcontract.NewService(draftStore, contractStore, approvedStore, events, subServiceTxRunner, fixedClock{now: testTime()}, ids)
	service := contract.NewService(goalStore, contractStore, seedService, draftService, approvalService, &fakeTransactionRunner{})

	created, _, err := service.Create(ctx, spine.ContractCreateRequest{GoalID: goal.ID}, activeMembership(goal.OrganizationID))
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	updated, err := service.UpdateDraft(ctx, created.ID, draftUpdateRequest(t, `{"title": "Reviewed draft title"}`), activeMembership(goal.OrganizationID))
	if err != nil {
		t.Fatalf("UpdateDraft() error = %v", err)
	}
	if updated.State != spine.ContractStateDraft {
		t.Fatalf("updated contract state = %q, want %q", updated.State, spine.ContractStateDraft)
	}
	submitted, err := service.SubmitForApproval(ctx, created.ID, draftReadyForApprovalRequest(), activeMembership(goal.OrganizationID))
	if err != nil {
		t.Fatalf("SubmitForApproval() error = %v", err)
	}
	if submitted.State != spine.ContractStateReadyForApproval {
		t.Fatalf("submitted contract state = %q, want %q", submitted.State, spine.ContractStateReadyForApproval)
	}
	approved, err := service.Approve(ctx, created.ID, approveRequest(), activeMembership(goal.OrganizationID))
	if err != nil {
		t.Fatalf("Approve() error = %v", err)
	}
	if approved.State != spine.ContractStateApproved {
		t.Fatalf("approved contract state = %q, want %q", approved.State, spine.ContractStateApproved)
	}
	if !draftStore.updateSawTransaction {
		t.Fatal("draft store Update did not see transaction context")
	}
	if !draftStore.markReadyForApprovalSawTransaction {
		t.Fatal("draft store MarkReadyForApproval did not see transaction context")
	}
	if !contractStore.markReadyForApprovalSawTransaction {
		t.Fatal("contract store MarkReadyForApproval did not see transaction context")
	}
	if !approvedStore.createSawTransaction {
		t.Fatal("approved contract store Create did not see transaction context")
	}
	if !contractStore.markApprovedSawTransaction {
		t.Fatal("contract store MarkApproved did not see transaction context")
	}
	if events.transactionalAppends != 5 {
		t.Fatalf("transactional event appends = %d, want 5", events.transactionalAppends)
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

type txContextKey struct{}

type fakeTransactionRunner struct {
	calls    int
	rollback func()
}

func (r *fakeTransactionRunner) RunReadCommitted(ctx context.Context, fn func(context.Context) error) error {
	r.calls++
	err := fn(context.WithValue(ctx, txContextKey{}, true))
	if err != nil && r.rollback != nil {
		r.rollback()
	}
	return err
}

func sawTransaction(ctx context.Context) bool {
	inTx, _ := ctx.Value(txContextKey{}).(bool)
	return inTx
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
		ID:             "018f0000-0000-7000-8000-000000000101",
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

func activeMembership(organizationID spine.OrganizationID) spine.OrganizationMembership {
	return spine.OrganizationMembership{
		ID:             "018f0000-0000-7000-8000-000000000011",
		OrganizationID: organizationID,
		UserID:         "018f0000-0000-7000-8000-000000000001",
		Role:           spine.OrganizationMembershipRoleMember,
		State:          spine.EntityStateActive,
		CreatedAt:      testTime(),
		UpdatedAt:      testTime(),
	}
}

func testTime() time.Time {
	return time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
}

func newContractServiceWithStore(contractStore *fakeContractStore) *contract.Service {
	goalStore := newFakeGoalStore()
	seedStore := newFakeContractSeedStore()
	draftStore := newFakeContractDraftStore()
	approvedStore := newFakeApprovedContractStore()
	events := newFakeEventLog()
	ids := &sequenceIDs{}
	txRunner := &fakeTransactionRunner{}
	seedService := contractseed.NewService(goalStore, contractStore, seedStore, events, txRunner, fixedClock{now: testTime()}, ids)
	draftService := contractdraft.NewService(seedStore, contractStore, draftStore, events, txRunner, fixedClock{now: testTime()}, ids)
	approvalService := approvedcontract.NewService(draftStore, contractStore, approvedStore, events, txRunner, fixedClock{now: testTime()}, ids)
	return contract.NewService(goalStore, contractStore, seedService, draftService, approvalService, txRunner)
}

func storedContract(id spine.ContractID, organizationID spine.OrganizationID, goalID spine.GoalID, state spine.ContractState) spine.Contract {
	return spine.Contract{
		ID:             id,
		OrganizationID: organizationID,
		ProjectID:      "018f0000-0000-7000-8000-000000000003",
		RepoBindingID:  "018f0000-0000-7000-8000-000000000004",
		GoalID:         goalID,
		State:          state,
		CreatedAt:      testTime(),
		UpdatedAt:      testTime().Add(time.Duration(len(id)) * time.Minute),
	}
}

func cloneContract(contract spine.Contract) spine.Contract {
	if contract.CurrentSeedID != nil {
		value := *contract.CurrentSeedID
		contract.CurrentSeedID = &value
	}
	if contract.CurrentDraftID != nil {
		value := *contract.CurrentDraftID
		contract.CurrentDraftID = &value
	}
	if contract.ApprovedSnapshotID != nil {
		value := *contract.ApprovedSnapshotID
		contract.ApprovedSnapshotID = &value
	}
	return contract
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
	contracts                          map[spine.ContractID]spine.Contract
	byGoal                             map[spine.GoalID]spine.ContractID
	lastListFilter                     spine.ContractListFilter
	createSawTransaction               bool
	markDraftCreatedSawTransaction     bool
	markReadyForApprovalSawTransaction bool
	markApprovedSawTransaction         bool
}

func newFakeContractStore() *fakeContractStore {
	return &fakeContractStore{
		contracts: map[spine.ContractID]spine.Contract{},
		byGoal:    map[spine.GoalID]spine.ContractID{},
	}
}

func (s *fakeContractStore) Create(ctx context.Context, contract spine.Contract) error {
	s.createSawTransaction = s.createSawTransaction || sawTransaction(ctx)
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

func (s *fakeContractStore) List(_ context.Context, filter spine.ContractListFilter) ([]spine.Contract, error) {
	s.lastListFilter = filter
	contracts := make([]spine.Contract, 0, len(s.contracts))
	for _, contract := range s.contracts {
		if filter.OrganizationID != "" && contract.OrganizationID != filter.OrganizationID {
			continue
		}
		if filter.ProjectID != "" && contract.ProjectID != filter.ProjectID {
			continue
		}
		if filter.RepoBindingID != "" && contract.RepoBindingID != filter.RepoBindingID {
			continue
		}
		if filter.GoalID != "" && contract.GoalID != filter.GoalID {
			continue
		}
		if filter.State != "" && contract.State != filter.State {
			continue
		}
		contracts = append(contracts, cloneContract(contract))
	}
	sort.Slice(contracts, func(i, j int) bool {
		if !contracts[i].UpdatedAt.Equal(contracts[j].UpdatedAt) {
			return contracts[i].UpdatedAt.After(contracts[j].UpdatedAt)
		}
		if !contracts[i].CreatedAt.Equal(contracts[j].CreatedAt) {
			return contracts[i].CreatedAt.After(contracts[j].CreatedAt)
		}
		return contracts[i].ID > contracts[j].ID
	})
	if filter.Limit > 0 && len(contracts) > filter.Limit {
		contracts = contracts[:filter.Limit]
	}
	return contracts, nil
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

func (s *fakeContractStore) MarkDraftCreated(ctx context.Context, id spine.ContractID, draftID spine.ContractDraftID, updatedAt time.Time) error {
	s.markDraftCreatedSawTransaction = s.markDraftCreatedSawTransaction || sawTransaction(ctx)
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

func (s *fakeContractStore) MarkReadyForApproval(ctx context.Context, id spine.ContractID, updatedAt time.Time) error {
	s.markReadyForApprovalSawTransaction = s.markReadyForApprovalSawTransaction || sawTransaction(ctx)
	contract, ok := s.contracts[id]
	if !ok {
		return nil
	}
	contract.State = spine.ContractStateReadyForApproval
	contract.UpdatedAt = updatedAt.UTC()
	s.contracts[id] = contract
	return nil
}

func (s *fakeContractStore) MarkApproved(ctx context.Context, id spine.ContractID, approvedID spine.ApprovedContractID, updatedAt time.Time) error {
	s.markApprovedSawTransaction = s.markApprovedSawTransaction || sawTransaction(ctx)
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
	seeds                map[spine.ContractSeedID]spine.ContractSeed
	byGoal               map[spine.GoalID]spine.ContractSeedID
	createSawTransaction bool
}

func newFakeContractSeedStore() *fakeContractSeedStore {
	return &fakeContractSeedStore{
		seeds:  map[spine.ContractSeedID]spine.ContractSeed{},
		byGoal: map[spine.GoalID]spine.ContractSeedID{},
	}
}

func (s *fakeContractSeedStore) Create(ctx context.Context, seed spine.ContractSeed) error {
	s.createSawTransaction = s.createSawTransaction || sawTransaction(ctx)
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
	drafts                             map[spine.ContractDraftID]spine.ContractDraft
	bySeed                             map[spine.ContractSeedID]spine.ContractDraftID
	createSawTransaction               bool
	updateSawTransaction               bool
	markReadyForApprovalSawTransaction bool
}

func newFakeContractDraftStore() *fakeContractDraftStore {
	return &fakeContractDraftStore{
		drafts: map[spine.ContractDraftID]spine.ContractDraft{},
		bySeed: map[spine.ContractSeedID]spine.ContractDraftID{},
	}
}

func (s *fakeContractDraftStore) Create(ctx context.Context, draft spine.ContractDraft) error {
	s.createSawTransaction = s.createSawTransaction || sawTransaction(ctx)
	s.drafts[draft.ID] = draft
	s.bySeed[draft.ContractSeedID] = draft.ID
	return nil
}

func (s *fakeContractDraftStore) Update(ctx context.Context, draft spine.ContractDraft) error {
	s.updateSawTransaction = s.updateSawTransaction || sawTransaction(ctx)
	s.drafts[draft.ID] = draft
	return nil
}

func (s *fakeContractDraftStore) MarkReadyForApproval(ctx context.Context, draft spine.ContractDraft) error {
	s.markReadyForApprovalSawTransaction = s.markReadyForApprovalSawTransaction || sawTransaction(ctx)
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
	approved             map[spine.ApprovedContractID]spine.ApprovedContract
	byDraft              map[spine.ContractDraftID]spine.ApprovedContractID
	createSawTransaction bool
}

func newFakeApprovedContractStore() *fakeApprovedContractStore {
	return &fakeApprovedContractStore{
		approved: map[spine.ApprovedContractID]spine.ApprovedContract{},
		byDraft:  map[spine.ContractDraftID]spine.ApprovedContractID{},
	}
}

func (s *fakeApprovedContractStore) Create(ctx context.Context, approved spine.ApprovedContract) error {
	s.createSawTransaction = s.createSawTransaction || sawTransaction(ctx)
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
	events               []spine.Event
	transactionalAppends int
}

func newFakeEventLog() *fakeEventLog {
	return &fakeEventLog{}
}

func (l *fakeEventLog) Append(ctx context.Context, event spine.Event) error {
	if sawTransaction(ctx) {
		l.transactionalAppends++
	}
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

func draftUpdateRequest(t *testing.T, changesJSON string) spine.ContractDraftUpdateRequest {
	t.Helper()

	var changes map[string]json.RawMessage
	if err := json.Unmarshal([]byte(changesJSON), &changes); err != nil {
		t.Fatalf("unmarshal update changes: %v", err)
	}
	return spine.ContractDraftUpdateRequest{
		UpdatedBy: spine.ActorRef{Kind: "user", ID: "dev_1", DisplayName: "Developer"},
		Changes:   changes,
	}
}

func draftReadyForApprovalRequest() spine.ContractDraftReadyForApprovalRequest {
	return spine.ContractDraftReadyForApprovalRequest{
		MarkedBy: spine.ActorRef{Kind: "user", ID: "dev_1", DisplayName: "Developer"},
	}
}

func approveRequest() spine.ApproveContractDraftRequest {
	return spine.ApproveContractDraftRequest{
		ApprovedBy: spine.ActorRef{Kind: "user", ID: "approver_1", DisplayName: "Approver"},
	}
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
