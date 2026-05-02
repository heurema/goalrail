package contractseed_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/contractseed"
	"github.com/heurema/goalrail/apps/server/internal/eventlog"
	"github.com/heurema/goalrail/apps/server/internal/spine"
	"github.com/heurema/goalrail/apps/server/internal/store"
)

func TestServiceCreatesContractSeedSnapshotFromGoal(t *testing.T) {
	service, goals, contracts, seeds, _ := seedService(t)
	goal := validSeedableGoal()
	if err := goals.Create(context.Background(), goal); err != nil {
		t.Fatalf("Create goal: %v", err)
	}

	created, err := service.Create(context.Background(), goal.ID)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if created.State != spine.ContractSeedStateCreated {
		t.Fatalf("state = %q, want %q", created.State, spine.ContractSeedStateCreated)
	}
	if created.ContractID == "" {
		t.Fatal("contract_id is empty")
	}
	if created.GoalID != goal.ID {
		t.Fatalf("goal_id = %q, want %q", created.GoalID, goal.ID)
	}
	if created.RepoBindingID != goal.RepoBindingID {
		t.Fatalf("repo_binding_id = %q, want %q", created.RepoBindingID, goal.RepoBindingID)
	}
	if created.OrganizationID != goal.OrganizationID {
		t.Fatalf("organization_id = %q, want %q", created.OrganizationID, goal.OrganizationID)
	}
	if created.ProjectID != goal.ProjectID {
		t.Fatalf("project_id = %q, want %q", created.ProjectID, goal.ProjectID)
	}
	if created.Title != goal.Title {
		t.Fatalf("title = %q, want %q", created.Title, goal.Title)
	}
	if created.IntentSummary != goal.Summary {
		t.Fatalf("intent_summary = %q, want %q", created.IntentSummary, goal.Summary)
	}
	if !reflect.DeepEqual(created.IntentOwner, goal.IntentOwner) {
		t.Fatalf("intent_owner = %#v, want %#v", created.IntentOwner, goal.IntentOwner)
	}
	if created.ScopeHint != goal.ScopeHint {
		t.Fatalf("scope_hint = %q, want %q", created.ScopeHint, goal.ScopeHint)
	}
	if created.AcceptanceHint != goal.AcceptanceHint {
		t.Fatalf("acceptance_hint = %q, want %q", created.AcceptanceHint, goal.AcceptanceHint)
	}
	if !hasSourceRef(created.SourceRefs, "goal", string(goal.ID)) {
		t.Fatalf("source_refs = %#v, want goal ref", created.SourceRefs)
	}
	if !hasSourceRef(created.SourceRefs, "intake", string(goal.IntakeID)) {
		t.Fatalf("source_refs = %#v, want intake ref", created.SourceRefs)
	}

	stored, ok, err := seeds.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("seeds.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("stored seed not found")
	}
	if stored.ID != created.ID {
		t.Fatalf("stored seed id = %q, want %q", stored.ID, created.ID)
	}
	contract, ok, err := contracts.Get(context.Background(), created.ContractID)
	if err != nil {
		t.Fatalf("contracts.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("contract not stored")
	}
	if contract.State != spine.ContractStateSeeded {
		t.Fatalf("contract state = %q, want %q", contract.State, spine.ContractStateSeeded)
	}
	if contract.CurrentSeedID == nil || *contract.CurrentSeedID != created.ID {
		t.Fatalf("contract current_seed_id = %v, want %q", contract.CurrentSeedID, created.ID)
	}
	if contract.GoalID != goal.ID {
		t.Fatalf("contract goal_id = %q, want %q", contract.GoalID, goal.ID)
	}
}

func TestServiceAppendsContractSeedCreatedEvent(t *testing.T) {
	service, goals, _, _, events := seedService(t)
	goal := validSeedableGoal()
	if err := goals.Create(context.Background(), goal); err != nil {
		t.Fatalf("Create goal: %v", err)
	}

	created, err := service.Create(context.Background(), goal.ID)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	appended := events.Events()
	if len(appended) != 1 {
		t.Fatalf("events length = %d, want 1", len(appended))
	}
	event := appended[0]
	if event.Type != contractseed.EventTypeContractSeedCreated {
		t.Fatalf("event type = %q, want %q", event.Type, contractseed.EventTypeContractSeedCreated)
	}
	if event.EntityType != contractseed.EntityTypeContractSeed {
		t.Fatalf("entity type = %q, want %q", event.EntityType, contractseed.EntityTypeContractSeed)
	}
	if event.EntityID != string(created.ID) {
		t.Fatalf("entity id = %q, want %q", event.EntityID, created.ID)
	}
	if event.OrganizationID != goal.OrganizationID {
		t.Fatalf("organization_id = %q, want %q", event.OrganizationID, goal.OrganizationID)
	}
	if event.ProjectID != goal.ProjectID {
		t.Fatalf("project_id = %q, want %q", event.ProjectID, goal.ProjectID)
	}
	if event.RepoBindingID != goal.RepoBindingID {
		t.Fatalf("repo_binding_id = %q, want %q", event.RepoBindingID, goal.RepoBindingID)
	}

	var payload spine.ContractSeed
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatalf("unmarshal contract_seed.created payload: %v", err)
	}
	if payload.ID != created.ID {
		t.Fatalf("payload id = %q, want %q", payload.ID, created.ID)
	}
	if payload.ContractID != created.ContractID {
		t.Fatalf("payload contract_id = %q, want %q", payload.ContractID, created.ContractID)
	}
}

func TestServiceRejectsDuplicateSeedForGoal(t *testing.T) {
	service, goals, _, _, events := seedService(t)
	goal := validSeedableGoal()
	if err := goals.Create(context.Background(), goal); err != nil {
		t.Fatalf("Create goal: %v", err)
	}
	if _, err := service.Create(context.Background(), goal.ID); err != nil {
		t.Fatalf("first Create() error = %v", err)
	}

	_, err := service.Create(context.Background(), goal.ID)
	if !errors.Is(err, contractseed.ErrAlreadySeeded) {
		t.Fatalf("second Create() error = %v, want ErrAlreadySeeded", err)
	}
	if got := len(events.Events()); got != 1 {
		t.Fatalf("events length = %d, want 1", got)
	}
}

func TestServiceRejectsGoalNotReadyForContractSeed(t *testing.T) {
	service, goals, _, _, _ := seedService(t)
	goal := validSeedableGoal()
	goal.State = spine.GoalStateNeedsClarification
	if err := goals.Create(context.Background(), goal); err != nil {
		t.Fatalf("Create goal: %v", err)
	}

	_, err := service.Create(context.Background(), goal.ID)
	if !errors.Is(err, contractseed.ErrInvalidGoalState) {
		t.Fatalf("Create() error = %v, want ErrInvalidGoalState", err)
	}
}

func TestServiceRejectsUnknownGoal(t *testing.T) {
	service, _, _, _, _ := seedService(t)

	_, err := service.Create(context.Background(), "missing")
	if !errors.Is(err, contractseed.ErrGoalNotFound) {
		t.Fatalf("Create() error = %v, want ErrGoalNotFound", err)
	}
}

func TestServiceValidatesRequiredGoalFields(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*spine.Goal)
	}{
		{name: "repo_binding_id", mutate: func(goal *spine.Goal) { goal.RepoBindingID = "" }},
		{name: "title", mutate: func(goal *spine.Goal) { goal.Title = "" }},
		{name: "summary", mutate: func(goal *spine.Goal) { goal.Summary = "" }},
		{name: "intent_owner.kind", mutate: func(goal *spine.Goal) { goal.IntentOwner.Kind = "" }},
		{name: "intent_owner.id", mutate: func(goal *spine.Goal) { goal.IntentOwner.ID = "" }},
		{name: "scope_hint", mutate: func(goal *spine.Goal) { goal.ScopeHint = "" }},
		{name: "acceptance_hint", mutate: func(goal *spine.Goal) { goal.AcceptanceHint = "" }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, goals, _, _, events := seedService(t)
			goal := validSeedableGoal()
			goal.ID = spine.GoalID("goal-" + tt.name)
			goal.IntakeID = spine.IntakeID("intake-" + tt.name)
			tt.mutate(&goal)
			if err := goals.Create(context.Background(), goal); err != nil {
				t.Fatalf("Create goal: %v", err)
			}

			_, err := service.Create(context.Background(), goal.ID)
			var validationErr *contractseed.ValidationError
			if !errors.As(err, &validationErr) {
				t.Fatalf("Create() error = %v, want ValidationError", err)
			}
			if got := len(events.Events()); got != 0 {
				t.Fatalf("events length = %d, want 0", got)
			}
		})
	}
}

func TestServiceDoesNotMutateGoal(t *testing.T) {
	service, goals, _, _, _ := seedService(t)
	goal := validSeedableGoal()
	if err := goals.Create(context.Background(), goal); err != nil {
		t.Fatalf("Create goal: %v", err)
	}

	if _, err := service.Create(context.Background(), goal.ID); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	stored, ok, err := goals.Get(context.Background(), goal.ID)
	if err != nil {
		t.Fatalf("goals.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("goal not found")
	}
	if !reflect.DeepEqual(stored, goal) {
		t.Fatalf("stored goal = %#v, want unchanged %#v", stored, goal)
	}
}

func TestServiceDoesNotAppendContractWorkGateProofEvents(t *testing.T) {
	service, goals, _, _, events := seedService(t)
	goal := validSeedableGoal()
	if err := goals.Create(context.Background(), goal); err != nil {
		t.Fatalf("Create goal: %v", err)
	}

	if _, err := service.Create(context.Background(), goal.ID); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	assertNoForbiddenEvents(t, events.Events())
}

func seedService(t *testing.T) (*contractseed.Service, *store.GoalStore, *store.ContractStore, *store.ContractSeedStore, *eventlog.EventLog) {
	t.Helper()

	goals := store.NewGoalStore()
	contracts := store.NewContractStore()
	seeds := store.NewContractSeedStore()
	events := eventlog.NewEventLog()
	service := contractseed.NewService(goals, contracts, seeds, events, fixedClock{now: testTime()}, &sequenceIDs{})
	return service, goals, contracts, seeds, events
}

func validSeedableGoal() spine.Goal {
	return spine.Goal{
		ID:             "goal-1",
		IntakeID:       "intake-1",
		OrganizationID: "018f0000-0000-7000-8000-000000000002",
		ProjectID:      "018f0000-0000-7000-8000-000000000003",
		RepoBindingID:  "018f0000-0000-7000-8000-000000000004",
		Title:          "Refactor CSV export filters",
		Summary:        "Current code duplicates filter logic. Preserve current behavior.",
		ScopeHint:      "Refactor duplicate CSV export filter logic",
		AcceptanceHint: "Existing CSV export behavior is preserved",
		SourceRefs: []spine.SourceRef{
			{Kind: "intake", ID: "intake-1"},
		},
		RequestAuthor: spine.ActorRef{Kind: "user", ID: "018f0000-0000-7000-8000-000000000001"},
		IntentOwner:   spine.ActorRef{Kind: "user", ID: "018f0000-0000-7000-8000-000000000001"},
		State:         spine.GoalStateReadyForContractSeed,
		CreatedAt:     testTime(),
	}
}

func hasSourceRef(refs []spine.SourceRef, kind string, id string) bool {
	for _, ref := range refs {
		if ref.Kind == kind && ref.ID == id {
			return true
		}
	}
	return false
}

func assertNoForbiddenEvents(t *testing.T, events []spine.Event) {
	t.Helper()

	forbidden := map[string]bool{
		"contract.draft_created": true,
		"contract.approved":      true,
		"work_item.created":      true,
		"run.started":            true,
		"gate.decision_written":  true,
		"proof.created":          true,
	}
	for _, event := range events {
		if forbidden[event.Type] {
			t.Fatalf("forbidden event type appended: %s", event.Type)
		}
	}
}

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

type sequenceIDs struct {
	contract     int
	contractSeed int
	event        int
}

func (g *sequenceIDs) NewContractID() (spine.ContractID, error) {
	g.contract++
	return spine.ContractID(fmt.Sprintf("contract-%d", g.contract)), nil
}

func (g *sequenceIDs) NewContractSeedID() (spine.ContractSeedID, error) {
	g.contractSeed++
	return spine.ContractSeedID(fmt.Sprintf("contract-seed-%d", g.contractSeed)), nil
}

func (g *sequenceIDs) NewEventID() (spine.EventID, error) {
	g.event++
	return spine.EventID(fmt.Sprintf("event-%d", g.event)), nil
}

func testTime() time.Time {
	return time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
}
