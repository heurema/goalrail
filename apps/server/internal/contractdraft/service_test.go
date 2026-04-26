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
	if created.OrganizationID != seed.OrganizationID {
		t.Fatalf("organization_id = %q, want %q", created.OrganizationID, seed.OrganizationID)
	}
	if created.ProjectID != seed.ProjectID {
		t.Fatalf("project_id = %q, want %q", created.ProjectID, seed.ProjectID)
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
	if event.OrganizationID != seed.OrganizationID {
		t.Fatalf("organization_id = %q, want %q", event.OrganizationID, seed.OrganizationID)
	}
	if event.ProjectID != seed.ProjectID {
		t.Fatalf("project_id = %q, want %q", event.ProjectID, seed.ProjectID)
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

func TestServiceUpdatesEditableDraftFields(t *testing.T) {
	service, seeds, _, events := draftService(t)
	seed := validDraftableSeed()
	if err := seeds.Create(context.Background(), seed); err != nil {
		t.Fatalf("Create seed: %v", err)
	}
	created, err := service.Create(context.Background(), seed.ID)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	updated, err := service.Update(context.Background(), created.ID, updateRequest(t, `{
		"title": "Reviewed draft title",
		"intent_summary": "Reviewed summary",
		"proposed_scope": ["Reviewed scope"],
		"proposed_acceptance_criteria": ["Reviewed acceptance"],
		"proposed_non_goals": [],
		"proposed_constraints": ["No schema changes"],
		"proposed_expected_checks": ["go test ./..."],
		"proposed_proof_expectations": ["Attach test output"],
		"risk_hints": ["Low risk"]
	}`))
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if updated.State != spine.ContractDraftStateDraft {
		t.Fatalf("state = %q, want %q", updated.State, spine.ContractDraftStateDraft)
	}
	if updated.Title != "Reviewed draft title" {
		t.Fatalf("title = %q, want reviewed title", updated.Title)
	}
	if updated.IntentSummary != "Reviewed summary" {
		t.Fatalf("intent_summary = %q, want reviewed summary", updated.IntentSummary)
	}
	if !reflect.DeepEqual(updated.ProposedScope, []string{"Reviewed scope"}) {
		t.Fatalf("proposed_scope = %#v", updated.ProposedScope)
	}
	if !reflect.DeepEqual(updated.ProposedAcceptanceCriteria, []string{"Reviewed acceptance"}) {
		t.Fatalf("proposed_acceptance_criteria = %#v", updated.ProposedAcceptanceCriteria)
	}
	if !reflect.DeepEqual(updated.ProposedNonGoals, []string{}) {
		t.Fatalf("proposed_non_goals = %#v, want empty slice", updated.ProposedNonGoals)
	}
	if !reflect.DeepEqual(updated.ProposedConstraints, []string{"No schema changes"}) {
		t.Fatalf("proposed_constraints = %#v", updated.ProposedConstraints)
	}
	if !reflect.DeepEqual(updated.ProposedExpectedChecks, []string{"go test ./..."}) {
		t.Fatalf("proposed_expected_checks = %#v", updated.ProposedExpectedChecks)
	}
	if !reflect.DeepEqual(updated.ProposedProofExpectations, []string{"Attach test output"}) {
		t.Fatalf("proposed_proof_expectations = %#v", updated.ProposedProofExpectations)
	}
	if !reflect.DeepEqual(updated.RiskHints, []string{"Low risk"}) {
		t.Fatalf("risk_hints = %#v", updated.RiskHints)
	}
	if updated.ContractSeedID != created.ContractSeedID || updated.GoalID != created.GoalID || updated.RepoBindingID != created.RepoBindingID {
		t.Fatalf("identity fields changed: got %#v, want seed/goal/repo unchanged from %#v", updated, created)
	}
	if !reflect.DeepEqual(updated.SourceRefs, created.SourceRefs) {
		t.Fatalf("source_refs = %#v, want unchanged %#v", updated.SourceRefs, created.SourceRefs)
	}
	if updated.CreatedAt != created.CreatedAt {
		t.Fatalf("created_at = %s, want %s", updated.CreatedAt, created.CreatedAt)
	}
	if got := countEventType(events.Events(), contractdraft.EventTypeContractDraftUpdated); got != 1 {
		t.Fatalf("contract_draft.updated events = %d, want 1", got)
	}
}

func TestServiceUpdateAllowsEmptyCoreDraftArrays(t *testing.T) {
	service, seeds, _, _ := draftService(t)
	seed := validDraftableSeed()
	if err := seeds.Create(context.Background(), seed); err != nil {
		t.Fatalf("Create seed: %v", err)
	}
	created, err := service.Create(context.Background(), seed.ID)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	updated, err := service.Update(context.Background(), created.ID, updateRequest(t, `{
		"proposed_scope": [],
		"proposed_acceptance_criteria": [],
		"proposed_proof_expectations": []
	}`))
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if updated.ProposedScope == nil || len(updated.ProposedScope) != 0 {
		t.Fatalf("proposed_scope = %#v, want explicit empty slice", updated.ProposedScope)
	}
	if updated.ProposedAcceptanceCriteria == nil || len(updated.ProposedAcceptanceCriteria) != 0 {
		t.Fatalf("proposed_acceptance_criteria = %#v, want explicit empty slice", updated.ProposedAcceptanceCriteria)
	}
	if updated.ProposedProofExpectations == nil || len(updated.ProposedProofExpectations) != 0 {
		t.Fatalf("proposed_proof_expectations = %#v, want explicit empty slice", updated.ProposedProofExpectations)
	}
}

func TestServiceRejectsNonEditableUpdateField(t *testing.T) {
	service, seeds, _, events := draftService(t)
	seed := validDraftableSeed()
	if err := seeds.Create(context.Background(), seed); err != nil {
		t.Fatalf("Create seed: %v", err)
	}
	created, err := service.Create(context.Background(), seed.ID)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	_, err = service.Update(context.Background(), created.ID, updateRequest(t, `{"state": "ready_for_approval"}`))
	var nonEditableErr *contractdraft.NonEditableFieldError
	if !errors.As(err, &nonEditableErr) {
		t.Fatalf("Update() error = %v, want NonEditableFieldError", err)
	}
	if got := countEventType(events.Events(), contractdraft.EventTypeContractDraftUpdated); got != 0 {
		t.Fatalf("contract_draft.updated events = %d, want 0", got)
	}
}

func TestServiceRejectsUnknownUpdateField(t *testing.T) {
	service, seeds, _, _ := draftService(t)
	seed := validDraftableSeed()
	if err := seeds.Create(context.Background(), seed); err != nil {
		t.Fatalf("Create seed: %v", err)
	}
	created, err := service.Create(context.Background(), seed.ID)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	_, err = service.Update(context.Background(), created.ID, updateRequest(t, `{"unknown": "value"}`))
	var unknownErr *contractdraft.UnknownFieldError
	if !errors.As(err, &unknownErr) {
		t.Fatalf("Update() error = %v, want UnknownFieldError", err)
	}
}

func TestServiceRejectsMissingUpdatedBy(t *testing.T) {
	service, seeds, _, _ := draftService(t)
	seed := validDraftableSeed()
	if err := seeds.Create(context.Background(), seed); err != nil {
		t.Fatalf("Create seed: %v", err)
	}
	created, err := service.Create(context.Background(), seed.ID)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	tests := []struct {
		name      string
		updatedBy spine.ActorRef
		field     string
	}{
		{name: "missing_kind", updatedBy: spine.ActorRef{ID: "dev_1"}, field: "updated_by.kind"},
		{name: "missing_id", updatedBy: spine.ActorRef{Kind: "user"}, field: "updated_by.id"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := updateRequest(t, `{"title": "Reviewed"}`)
			input.UpdatedBy = tt.updatedBy

			_, err := service.Update(context.Background(), created.ID, input)
			var validationErr *contractdraft.ValidationError
			if !errors.As(err, &validationErr) {
				t.Fatalf("Update() error = %v, want ValidationError", err)
			}
			if validationErr.Field != tt.field {
				t.Fatalf("validation field = %q, want %q", validationErr.Field, tt.field)
			}
		})
	}
}

func TestServiceRejectsEmptyUpdateChanges(t *testing.T) {
	service, seeds, _, _ := draftService(t)
	seed := validDraftableSeed()
	if err := seeds.Create(context.Background(), seed); err != nil {
		t.Fatalf("Create seed: %v", err)
	}
	created, err := service.Create(context.Background(), seed.ID)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	_, err = service.Update(context.Background(), created.ID, spine.ContractDraftUpdateRequest{
		UpdatedBy: spine.ActorRef{Kind: "user", ID: "dev_1"},
		Changes:   map[string]json.RawMessage{},
	})
	var validationErr *contractdraft.ValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("Update() error = %v, want ValidationError", err)
	}
	if validationErr.Field != "changes" {
		t.Fatalf("validation field = %q, want changes", validationErr.Field)
	}
}

func TestServiceRejectsUpdateWhenDraftNotDraft(t *testing.T) {
	service, _, drafts, _ := draftService(t)
	created := validStoredDraft()
	created.State = "ready_for_approval"
	if err := drafts.Create(context.Background(), created); err != nil {
		t.Fatalf("drafts.Create() error = %v", err)
	}

	_, err := service.Update(context.Background(), created.ID, updateRequest(t, `{"title": "Reviewed"}`))
	if !errors.Is(err, contractdraft.ErrInvalidDraftState) {
		t.Fatalf("Update() error = %v, want ErrInvalidDraftState", err)
	}
}

func TestServiceAppendsContractDraftUpdatedEvent(t *testing.T) {
	service, seeds, _, events := draftService(t)
	seed := validDraftableSeed()
	if err := seeds.Create(context.Background(), seed); err != nil {
		t.Fatalf("Create seed: %v", err)
	}
	created, err := service.Create(context.Background(), seed.ID)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	updated, err := service.Update(context.Background(), created.ID, updateRequest(t, `{
		"proposed_acceptance_criteria": ["Reviewed acceptance"],
		"proposed_scope": ["Reviewed scope"]
	}`))
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	appended := events.Events()
	if got := countEventType(appended, contractdraft.EventTypeContractDraftUpdated); got != 1 {
		t.Fatalf("contract_draft.updated events = %d, want 1", got)
	}
	event := appended[len(appended)-1]
	if event.Type != contractdraft.EventTypeContractDraftUpdated {
		t.Fatalf("event type = %q, want %q", event.Type, contractdraft.EventTypeContractDraftUpdated)
	}
	if event.EntityType != contractdraft.EntityTypeContractDraft {
		t.Fatalf("entity type = %q, want %q", event.EntityType, contractdraft.EntityTypeContractDraft)
	}
	if event.EntityID != string(updated.ID) {
		t.Fatalf("entity id = %q, want %q", event.EntityID, updated.ID)
	}
	if event.OrganizationID != updated.OrganizationID {
		t.Fatalf("organization_id = %q, want %q", event.OrganizationID, updated.OrganizationID)
	}
	if event.ProjectID != updated.ProjectID {
		t.Fatalf("project_id = %q, want %q", event.ProjectID, updated.ProjectID)
	}
	if event.RepoBindingID != updated.RepoBindingID {
		t.Fatalf("repo_binding_id = %q, want %q", event.RepoBindingID, updated.RepoBindingID)
	}

	var payload struct {
		ContractDraftID spine.ContractDraftID `json:"contract_draft_id"`
		ChangedFields   []string              `json:"changed_fields"`
		UpdatedBy       spine.ActorRef        `json:"updated_by"`
		PreviousValues  map[string]any        `json:"previous_values"`
		NewValues       map[string]any        `json:"new_values"`
		UpdatedAt       time.Time             `json:"updated_at"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatalf("unmarshal contract_draft.updated payload: %v", err)
	}
	if payload.ContractDraftID != updated.ID {
		t.Fatalf("payload contract_draft_id = %q, want %q", payload.ContractDraftID, updated.ID)
	}
	if !reflect.DeepEqual(payload.ChangedFields, []string{"proposed_acceptance_criteria", "proposed_scope"}) {
		t.Fatalf("changed_fields = %#v", payload.ChangedFields)
	}
	if payload.UpdatedBy.Kind != "user" || payload.UpdatedBy.ID != "dev_1" {
		t.Fatalf("updated_by = %#v, want audit user", payload.UpdatedBy)
	}
	if _, ok := payload.PreviousValues["proposed_scope"]; !ok {
		t.Fatalf("previous_values missing proposed_scope: %#v", payload.PreviousValues)
	}
	if _, ok := payload.NewValues["proposed_scope"]; !ok {
		t.Fatalf("new_values missing proposed_scope: %#v", payload.NewValues)
	}
}

func TestServiceUpdateDoesNotAppendApprovalWorkGateProofEvents(t *testing.T) {
	service, seeds, _, events := draftService(t)
	seed := validDraftableSeed()
	if err := seeds.Create(context.Background(), seed); err != nil {
		t.Fatalf("Create seed: %v", err)
	}
	created, err := service.Create(context.Background(), seed.ID)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := service.Update(context.Background(), created.ID, updateRequest(t, `{"title": "Reviewed"}`)); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	assertNoForbiddenEvents(t, events.Events())
}

func TestServiceMarksDraftReadyForApproval(t *testing.T) {
	service, seeds, drafts, _ := draftService(t)
	seed := validDraftableSeed()
	if err := seeds.Create(context.Background(), seed); err != nil {
		t.Fatalf("Create seed: %v", err)
	}
	created, err := service.Create(context.Background(), seed.ID)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	updated, err := service.MarkReadyForApproval(context.Background(), created.ID, readyForApprovalRequest())
	if err != nil {
		t.Fatalf("MarkReadyForApproval() error = %v", err)
	}

	if updated.State != spine.ContractDraftStateReadyForApproval {
		t.Fatalf("state = %q, want %q", updated.State, spine.ContractDraftStateReadyForApproval)
	}
	expected := created
	expected.State = spine.ContractDraftStateReadyForApproval
	if !reflect.DeepEqual(updated, expected) {
		t.Fatalf("updated draft = %#v, want only state changed %#v", updated, expected)
	}

	stored, ok, err := drafts.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("drafts.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("stored draft not found")
	}
	if !reflect.DeepEqual(stored, expected) {
		t.Fatalf("stored draft = %#v, want %#v", stored, expected)
	}
}

func TestServiceRejectsIncompleteDraftForReadyForApproval(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*spine.ContractDraft)
		reason string
	}{
		{name: "title", mutate: func(draft *spine.ContractDraft) { draft.Title = "" }, reason: contractdraft.ReasonMissingTitle},
		{name: "intent_summary", mutate: func(draft *spine.ContractDraft) { draft.IntentSummary = "" }, reason: contractdraft.ReasonMissingIntentSummary},
		{name: "repo_binding_id", mutate: func(draft *spine.ContractDraft) { draft.RepoBindingID = "" }, reason: contractdraft.ReasonMissingRepoBindingID},
		{name: "contract_seed_id", mutate: func(draft *spine.ContractDraft) { draft.ContractSeedID = "" }, reason: contractdraft.ReasonMissingContractSeedID},
		{name: "goal_id", mutate: func(draft *spine.ContractDraft) { draft.GoalID = "" }, reason: contractdraft.ReasonMissingGoalID},
		{name: "proposed_scope", mutate: func(draft *spine.ContractDraft) { draft.ProposedScope = nil }, reason: contractdraft.ReasonMissingProposedScope},
		{name: "proposed_acceptance_criteria", mutate: func(draft *spine.ContractDraft) { draft.ProposedAcceptanceCriteria = []string{" "} }, reason: contractdraft.ReasonMissingProposedAcceptanceCriteria},
		{name: "proposed_proof_expectations", mutate: func(draft *spine.ContractDraft) { draft.ProposedProofExpectations = []string{} }, reason: contractdraft.ReasonMissingProposedProofExpectations},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, _, drafts, events := draftService(t)
			created := validStoredDraft()
			created.ID = spine.ContractDraftID("contract-draft-" + tt.name)
			created.ContractSeedID = spine.ContractSeedID("contract-seed-" + tt.name)
			tt.mutate(&created)
			if err := drafts.Create(context.Background(), created); err != nil {
				t.Fatalf("drafts.Create() error = %v", err)
			}

			_, err := service.MarkReadyForApproval(context.Background(), created.ID, readyForApprovalRequest())
			var completenessErr *contractdraft.CompletenessError
			if !errors.As(err, &completenessErr) {
				t.Fatalf("MarkReadyForApproval() error = %v, want CompletenessError", err)
			}
			if !containsString(completenessErr.ReasonCodes, tt.reason) {
				t.Fatalf("reason codes = %#v, want %q", completenessErr.ReasonCodes, tt.reason)
			}
			if got := countEventType(events.Events(), contractdraft.EventTypeContractDraftMarkedReadyForApproval); got != 0 {
				t.Fatalf("marked ready events = %d, want 0", got)
			}
		})
	}
}

func TestServiceRejectsMissingMarkedBy(t *testing.T) {
	service, _, drafts, _ := draftService(t)
	created := validStoredDraft()
	if err := drafts.Create(context.Background(), created); err != nil {
		t.Fatalf("drafts.Create() error = %v", err)
	}

	tests := []struct {
		name     string
		markedBy spine.ActorRef
		field    string
	}{
		{name: "missing_kind", markedBy: spine.ActorRef{ID: "dev_1"}, field: "marked_by.kind"},
		{name: "missing_id", markedBy: spine.ActorRef{Kind: "user"}, field: "marked_by.id"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.MarkReadyForApproval(context.Background(), created.ID, spine.ContractDraftReadyForApprovalRequest{MarkedBy: tt.markedBy})
			var validationErr *contractdraft.ValidationError
			if !errors.As(err, &validationErr) {
				t.Fatalf("MarkReadyForApproval() error = %v, want ValidationError", err)
			}
			if validationErr.Field != tt.field {
				t.Fatalf("validation field = %q, want %q", validationErr.Field, tt.field)
			}
		})
	}
}

func TestServiceRejectsReadyForApprovalWhenDraftNotDraft(t *testing.T) {
	service, _, drafts, _ := draftService(t)
	created := validStoredDraft()
	created.State = spine.ContractDraftStateReadyForApproval
	if err := drafts.Create(context.Background(), created); err != nil {
		t.Fatalf("drafts.Create() error = %v", err)
	}

	_, err := service.MarkReadyForApproval(context.Background(), created.ID, readyForApprovalRequest())
	if !errors.Is(err, contractdraft.ErrInvalidDraftState) {
		t.Fatalf("MarkReadyForApproval() error = %v, want ErrInvalidDraftState", err)
	}
}

func TestServiceAppendsContractDraftMarkedReadyForApprovalEvent(t *testing.T) {
	service, seeds, _, events := draftService(t)
	seed := validDraftableSeed()
	if err := seeds.Create(context.Background(), seed); err != nil {
		t.Fatalf("Create seed: %v", err)
	}
	created, err := service.Create(context.Background(), seed.ID)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	updated, err := service.MarkReadyForApproval(context.Background(), created.ID, readyForApprovalRequest())
	if err != nil {
		t.Fatalf("MarkReadyForApproval() error = %v", err)
	}

	appended := events.Events()
	if got := countEventType(appended, contractdraft.EventTypeContractDraftMarkedReadyForApproval); got != 1 {
		t.Fatalf("marked ready events = %d, want 1", got)
	}
	event := appended[len(appended)-1]
	if event.Type != contractdraft.EventTypeContractDraftMarkedReadyForApproval {
		t.Fatalf("event type = %q, want %q", event.Type, contractdraft.EventTypeContractDraftMarkedReadyForApproval)
	}
	if event.EntityType != contractdraft.EntityTypeContractDraft {
		t.Fatalf("entity type = %q, want %q", event.EntityType, contractdraft.EntityTypeContractDraft)
	}
	if event.EntityID != string(updated.ID) {
		t.Fatalf("entity id = %q, want %q", event.EntityID, updated.ID)
	}

	var payload struct {
		ContractDraftID spine.ContractDraftID    `json:"contract_draft_id"`
		MarkedBy        spine.ActorRef           `json:"marked_by"`
		ReasonCodes     []string                 `json:"reason_codes"`
		PreviousState   spine.ContractDraftState `json:"previous_state"`
		NewState        spine.ContractDraftState `json:"new_state"`
		MarkedAt        time.Time                `json:"marked_at"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatalf("unmarshal contract_draft.marked_ready_for_approval payload: %v", err)
	}
	if payload.ContractDraftID != updated.ID {
		t.Fatalf("payload contract_draft_id = %q, want %q", payload.ContractDraftID, updated.ID)
	}
	if payload.MarkedBy.Kind != "user" || payload.MarkedBy.ID != "dev_1" {
		t.Fatalf("marked_by = %#v, want audit user", payload.MarkedBy)
	}
	if len(payload.ReasonCodes) != 0 {
		t.Fatalf("reason_codes = %#v, want empty", payload.ReasonCodes)
	}
	if payload.PreviousState != spine.ContractDraftStateDraft {
		t.Fatalf("previous_state = %q, want draft", payload.PreviousState)
	}
	if payload.NewState != spine.ContractDraftStateReadyForApproval {
		t.Fatalf("new_state = %q, want ready_for_approval", payload.NewState)
	}
	if !payload.MarkedAt.Equal(testTime()) {
		t.Fatalf("marked_at = %s, want %s", payload.MarkedAt, testTime())
	}
}

func TestServiceReadyForApprovalDoesNotAppendApprovalWorkGateProofEvents(t *testing.T) {
	service, seeds, _, events := draftService(t)
	seed := validDraftableSeed()
	if err := seeds.Create(context.Background(), seed); err != nil {
		t.Fatalf("Create seed: %v", err)
	}
	created, err := service.Create(context.Background(), seed.ID)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := service.MarkReadyForApproval(context.Background(), created.ID, readyForApprovalRequest()); err != nil {
		t.Fatalf("MarkReadyForApproval() error = %v", err)
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

func updateRequest(t *testing.T, changesJSON string) spine.ContractDraftUpdateRequest {
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

func readyForApprovalRequest() spine.ContractDraftReadyForApprovalRequest {
	return spine.ContractDraftReadyForApprovalRequest{
		MarkedBy: spine.ActorRef{Kind: "user", ID: "dev_1", DisplayName: "Developer"},
	}
}

func validStoredDraft() spine.ContractDraft {
	return spine.ContractDraft{
		ID:                         "contract-draft-1",
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
		State:     spine.ContractDraftStateDraft,
		CreatedAt: testTime(),
	}
}

func validDraftableSeed() spine.ContractSeed {
	return spine.ContractSeed{
		ID:             "contract-seed-1",
		OrganizationID: "organization-1",
		ProjectID:      "project-1",
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

func countEventType(events []spine.Event, eventType string) int {
	count := 0
	for _, event := range events {
		if event.Type == eventType {
			count++
		}
	}
	return count
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
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
