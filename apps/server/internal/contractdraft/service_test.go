package contractdraft_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/contractdraft"
	"github.com/heurema/goalrail/apps/server/internal/eventlog"
	"github.com/heurema/goalrail/apps/server/internal/spine"
	"github.com/heurema/goalrail/apps/server/internal/store"
)

func TestServiceCreatesContractDraftFromSeed(t *testing.T) {
	service, seeds, drafts, _ := draftService(t)
	seed := validDraftableSeed()
	if err := seeds.Create(context.Background(), seed); err != nil {
		t.Fatalf("Create seed: %v", err)
	}

	created, err := service.Create(context.Background(), seed.ID)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if created.State != spine.ContractDraftStateDraft {
		t.Fatalf("state = %q, want %q", created.State, spine.ContractDraftStateDraft)
	}
	if created.ContractSeedID != seed.ID {
		t.Fatalf("contract_seed_id = %q, want %q", created.ContractSeedID, seed.ID)
	}
	if created.GoalID != seed.GoalID {
		t.Fatalf("goal_id = %q, want %q", created.GoalID, seed.GoalID)
	}
	if created.RepoBindingID != seed.RepoBindingID {
		t.Fatalf("repo_binding_id = %q, want %q", created.RepoBindingID, seed.RepoBindingID)
	}
	if created.Title != seed.Title {
		t.Fatalf("title = %q, want %q", created.Title, seed.Title)
	}
	if created.IntentSummary != seed.IntentSummary {
		t.Fatalf("intent_summary = %q, want %q", created.IntentSummary, seed.IntentSummary)
	}
	if !reflect.DeepEqual(created.ProposedScope, []string{seed.ScopeHint}) {
		t.Fatalf("proposed_scope = %#v, want seed scope hint", created.ProposedScope)
	}
	if !reflect.DeepEqual(created.ProposedAcceptanceCriteria, []string{seed.AcceptanceHint}) {
		t.Fatalf("proposed_acceptance_criteria = %#v, want seed acceptance hint", created.ProposedAcceptanceCriteria)
	}
	if len(created.ProposedNonGoals) != 0 {
		t.Fatalf("proposed_non_goals = %#v, want empty", created.ProposedNonGoals)
	}
	if len(created.ProposedConstraints) != 0 {
		t.Fatalf("proposed_constraints = %#v, want empty", created.ProposedConstraints)
	}
	if len(created.ProposedExpectedChecks) != 0 {
		t.Fatalf("proposed_expected_checks = %#v, want empty", created.ProposedExpectedChecks)
	}
	if !reflect.DeepEqual(created.ProposedProofExpectations, []string{contractdraft.DefaultProofExpectation}) {
		t.Fatalf("proposed_proof_expectations = %#v, want default", created.ProposedProofExpectations)
	}
	if len(created.RiskHints) != 0 {
		t.Fatalf("risk_hints = %#v, want empty", created.RiskHints)
	}
	if !hasSourceRef(created.SourceRefs, "contract_seed", string(seed.ID)) {
		t.Fatalf("source_refs = %#v, want contract_seed ref", created.SourceRefs)
	}
	if !hasSourceRef(created.SourceRefs, "goal", string(seed.GoalID)) {
		t.Fatalf("source_refs = %#v, want goal ref", created.SourceRefs)
	}
	if !hasSourceRef(created.SourceRefs, "intake", "intake-1") {
		t.Fatalf("source_refs = %#v, want intake ref", created.SourceRefs)
	}

	stored, ok, err := drafts.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("drafts.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("stored draft not found")
	}
	if stored.ID != created.ID {
		t.Fatalf("stored draft id = %q, want %q", stored.ID, created.ID)
	}
}

func TestServiceAppendsContractDraftCreatedEvent(t *testing.T) {
	service, seeds, _, events := draftService(t)
	seed := validDraftableSeed()
	if err := seeds.Create(context.Background(), seed); err != nil {
		t.Fatalf("Create seed: %v", err)
	}

	created, err := service.Create(context.Background(), seed.ID)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	appended := events.Events()
	if len(appended) != 1 {
		t.Fatalf("events length = %d, want 1", len(appended))
	}
	event := appended[0]
	if event.Type != contractdraft.EventTypeContractDraftCreated {
		t.Fatalf("event type = %q, want %q", event.Type, contractdraft.EventTypeContractDraftCreated)
	}
	if event.EntityType != contractdraft.EntityTypeContractDraft {
		t.Fatalf("entity type = %q, want %q", event.EntityType, contractdraft.EntityTypeContractDraft)
	}
	if event.EntityID != string(created.ID) {
		t.Fatalf("entity id = %q, want %q", event.EntityID, created.ID)
	}
	if event.RepoBindingID != seed.RepoBindingID {
		t.Fatalf("repo_binding_id = %q, want %q", event.RepoBindingID, seed.RepoBindingID)
	}

	var payload spine.ContractDraft
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatalf("unmarshal contract_draft.created payload: %v", err)
	}
	if payload.ID != created.ID {
		t.Fatalf("payload id = %q, want %q", payload.ID, created.ID)
	}
}

func TestServiceRejectsDuplicateDraftForSeed(t *testing.T) {
	service, seeds, _, events := draftService(t)
	seed := validDraftableSeed()
	if err := seeds.Create(context.Background(), seed); err != nil {
		t.Fatalf("Create seed: %v", err)
	}
	if _, err := service.Create(context.Background(), seed.ID); err != nil {
		t.Fatalf("first Create() error = %v", err)
	}

	_, err := service.Create(context.Background(), seed.ID)
	if !errors.Is(err, contractdraft.ErrAlreadyDrafted) {
		t.Fatalf("second Create() error = %v, want ErrAlreadyDrafted", err)
	}
	if got := len(events.Events()); got != 1 {
		t.Fatalf("events length = %d, want 1", got)
	}
}

func TestServiceRejectsSeedNotCreated(t *testing.T) {
	service, seeds, _, _ := draftService(t)
	seed := validDraftableSeed()
	seed.State = "superseded"
	if err := seeds.Create(context.Background(), seed); err != nil {
		t.Fatalf("Create seed: %v", err)
	}

	_, err := service.Create(context.Background(), seed.ID)
	if !errors.Is(err, contractdraft.ErrInvalidSeedState) {
		t.Fatalf("Create() error = %v, want ErrInvalidSeedState", err)
	}
}

func TestServiceRejectsUnknownSeed(t *testing.T) {
	service, _, _, _ := draftService(t)

	_, err := service.Create(context.Background(), "missing")
	if !errors.Is(err, contractdraft.ErrContractSeedNotFound) {
		t.Fatalf("Create() error = %v, want ErrContractSeedNotFound", err)
	}
}

func TestServiceValidatesRequiredSeedFields(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*spine.ContractSeed)
	}{
		{name: "goal_id", mutate: func(seed *spine.ContractSeed) { seed.GoalID = "" }},
		{name: "repo_binding_id", mutate: func(seed *spine.ContractSeed) { seed.RepoBindingID = "" }},
		{name: "title", mutate: func(seed *spine.ContractSeed) { seed.Title = "" }},
		{name: "intent_summary", mutate: func(seed *spine.ContractSeed) { seed.IntentSummary = "" }},
		{name: "intent_owner.kind", mutate: func(seed *spine.ContractSeed) { seed.IntentOwner.Kind = "" }},
		{name: "intent_owner.id", mutate: func(seed *spine.ContractSeed) { seed.IntentOwner.ID = "" }},
		{name: "scope_hint", mutate: func(seed *spine.ContractSeed) { seed.ScopeHint = "" }},
		{name: "acceptance_hint", mutate: func(seed *spine.ContractSeed) { seed.AcceptanceHint = "" }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, seeds, _, events := draftService(t)
			seed := validDraftableSeed()
			seed.ID = spine.ContractSeedID("contract-seed-" + tt.name)
			seed.GoalID = spine.GoalID("goal-" + tt.name)
			tt.mutate(&seed)
			if err := seeds.Create(context.Background(), seed); err != nil {
				t.Fatalf("Create seed: %v", err)
			}

			_, err := service.Create(context.Background(), seed.ID)
			var validationErr *contractdraft.ValidationError
			if !errors.As(err, &validationErr) {
				t.Fatalf("Create() error = %v, want ValidationError", err)
			}
			if got := len(events.Events()); got != 0 {
				t.Fatalf("events length = %d, want 0", got)
			}
		})
	}
}

func TestServiceDoesNotMutateSeed(t *testing.T) {
	service, seeds, _, _ := draftService(t)
	seed := validDraftableSeed()
	if err := seeds.Create(context.Background(), seed); err != nil {
		t.Fatalf("Create seed: %v", err)
	}

	if _, err := service.Create(context.Background(), seed.ID); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	stored, ok, err := seeds.Get(context.Background(), seed.ID)
	if err != nil {
		t.Fatalf("seeds.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("seed not found")
	}
	if !reflect.DeepEqual(stored, seed) {
		t.Fatalf("stored seed = %#v, want unchanged %#v", stored, seed)
	}
}

func TestServiceDoesNotAppendApprovalWorkGateProofEvents(t *testing.T) {
	service, seeds, _, events := draftService(t)
	seed := validDraftableSeed()
	if err := seeds.Create(context.Background(), seed); err != nil {
		t.Fatalf("Create seed: %v", err)
	}

	if _, err := service.Create(context.Background(), seed.ID); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	assertNoForbiddenEvents(t, events.Events())
}

func draftService(t *testing.T) (*contractdraft.Service, *store.ContractSeedStore, *store.ContractDraftStore, *eventlog.EventLog) {
	t.Helper()

	seeds := store.NewContractSeedStore()
	drafts := store.NewContractDraftStore()
	events := eventlog.NewEventLog()
	service := contractdraft.NewService(seeds, drafts, events, fixedClock{now: testTime()}, &sequenceIDs{})
	return service, seeds, drafts, events
}

func validDraftableSeed() spine.ContractSeed {
	return spine.ContractSeed{
		ID:             "contract-seed-1",
		GoalID:         "goal-1",
		RepoBindingID:  "repo-binding-1",
		Title:          "Refactor CSV export filters",
		IntentSummary:  "Current code duplicates filter logic. Preserve current behavior.",
		IntentOwner:    spine.ActorRef{Kind: "user", ID: "dev_1"},
		ScopeHint:      "Refactor duplicate CSV export filter logic",
		AcceptanceHint: "Existing CSV export behavior is preserved",
		SourceRefs: []spine.SourceRef{
			{Kind: "goal", ID: "goal-1"},
			{Kind: "intake", ID: "intake-1"},
		},
		State:     spine.ContractSeedStateCreated,
		CreatedAt: testTime(),
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
		"contract.approved":     true,
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

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

type sequenceIDs struct {
	contractDraft int
	event         int
}

func (g *sequenceIDs) NewContractDraftID() (spine.ContractDraftID, error) {
	g.contractDraft++
	return spine.ContractDraftID(fmt.Sprintf("contract-draft-%d", g.contractDraft)), nil
}

func (g *sequenceIDs) NewEventID() (spine.EventID, error) {
	g.event++
	return spine.EventID(fmt.Sprintf("event-%d", g.event)), nil
}

func testTime() time.Time {
	return time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
}
