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
	"github.com/heurema/goalrail/apps/server/internal/spine"
)

func TestServiceCreatesContractDraftFromSeed(t *testing.T) {
	service, seeds, contracts, drafts, _ := draftService(t)
	seed := validDraftableSeed()
	storeSeedWithContract(t, seeds, contracts, seed)

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
	if created.ContractID != seed.ContractID {
		t.Fatalf("contract_id = %q, want %q", created.ContractID, seed.ContractID)
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
	contract, ok, err := contracts.Get(context.Background(), seed.ContractID)
	if err != nil {
		t.Fatalf("contracts.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("contract not found")
	}
	if contract.State != spine.ContractStateDraft {
		t.Fatalf("contract state = %q, want %q", contract.State, spine.ContractStateDraft)
	}
	if contract.CurrentDraftID == nil || *contract.CurrentDraftID != created.ID {
		t.Fatalf("contract current_draft_id = %v, want %q", contract.CurrentDraftID, created.ID)
	}
}

func TestServiceAppendsContractDraftCreatedEvent(t *testing.T) {
	service, seeds, contracts, _, events := draftService(t)
	seed := validDraftableSeed()
	storeSeedWithContract(t, seeds, contracts, seed)

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
	if payload.ContractID != created.ContractID {
		t.Fatalf("payload contract_id = %q, want %q", payload.ContractID, created.ContractID)
	}
}

func TestServiceCreateUsesRequiredTransactionRunner(t *testing.T) {
	service, seeds, contracts, drafts, events := draftService(t)
	txRunner := service.TxRunner.(*fakeTransactionRunner)
	outerCtx := context.WithValue(context.Background(), txContextKey{}, "outer")
	seed := validDraftableSeed()
	storeSeedWithContract(t, seeds, contracts, seed)

	if _, err := service.Create(outerCtx, seed.ID); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if txRunner.calls != 1 {
		t.Fatalf("TxRunner calls = %d, want 1", txRunner.calls)
	}
	if drafts.createCtx != txRunner.txCtx {
		t.Fatal("Drafts.Create did not receive transaction context")
	}
	if contracts.markDraftCreatedCtx != txRunner.txCtx {
		t.Fatal("Contracts.MarkDraftCreated did not receive transaction context")
	}
	if events.appendCtx != txRunner.txCtx {
		t.Fatal("Events.Append did not receive transaction context")
	}
	if drafts.createCtx == outerCtx || contracts.markDraftCreatedCtx == outerCtx || events.appendCtx == outerCtx {
		t.Fatal("transactional create writes used outer context")
	}
}

func TestServiceCreateFailedCreateDoesNotRunPostFailureDuplicateLookup(t *testing.T) {
	service, seeds, contracts, drafts, _ := draftService(t)
	txRunner := service.TxRunner.(*fakeTransactionRunner)
	createErr := errors.New("create failed")
	drafts.createErr = createErr
	seed := validDraftableSeed()
	storeSeedWithContract(t, seeds, contracts, seed)

	_, err := service.Create(txRunner.txCtx, seed.ID)
	if !errors.Is(err, createErr) {
		t.Fatalf("Create() error = %v, want original create error", err)
	}
	if got := len(drafts.getBySeedCtxs); got != 1 {
		t.Fatalf("Drafts.GetByContractSeedID calls = %d, want preflight only", got)
	}
}

func TestServiceRejectsDuplicateDraftForSeed(t *testing.T) {
	service, seeds, contracts, _, events := draftService(t)
	seed := validDraftableSeed()
	storeSeedWithContract(t, seeds, contracts, seed)
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
	service, seeds, contracts, _, _ := draftService(t)
	seed := validDraftableSeed()
	seed.State = "superseded"
	storeSeedWithContract(t, seeds, contracts, seed)

	_, err := service.Create(context.Background(), seed.ID)
	if !errors.Is(err, contractdraft.ErrInvalidSeedState) {
		t.Fatalf("Create() error = %v, want ErrInvalidSeedState", err)
	}
}

func TestServiceRejectsUnknownSeed(t *testing.T) {
	service, _, _, _, _ := draftService(t)

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
			service, seeds, contracts, _, events := draftService(t)
			seed := validDraftableSeed()
			seed.ID = spine.ContractSeedID("contract-seed-" + tt.name)
			seed.GoalID = spine.GoalID("goal-" + tt.name)
			tt.mutate(&seed)
			storeSeedWithContract(t, seeds, contracts, seed)

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
	service, seeds, contracts, _, _ := draftService(t)
	seed := validDraftableSeed()
	storeSeedWithContract(t, seeds, contracts, seed)

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
	service, seeds, contracts, _, events := draftService(t)
	seed := validDraftableSeed()
	storeSeedWithContract(t, seeds, contracts, seed)

	if _, err := service.Create(context.Background(), seed.ID); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	assertNoForbiddenEvents(t, events.Events())
}

func TestServiceUpdatesEditableDraftFields(t *testing.T) {
	service, seeds, contracts, _, events := draftService(t)
	seed := validDraftableSeed()
	storeSeedWithContract(t, seeds, contracts, seed)
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
	service, seeds, contracts, _, _ := draftService(t)
	seed := validDraftableSeed()
	storeSeedWithContract(t, seeds, contracts, seed)
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
	service, seeds, contracts, _, events := draftService(t)
	seed := validDraftableSeed()
	storeSeedWithContract(t, seeds, contracts, seed)
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
	service, seeds, contracts, _, _ := draftService(t)
	seed := validDraftableSeed()
	storeSeedWithContract(t, seeds, contracts, seed)
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
	service, seeds, contracts, _, _ := draftService(t)
	seed := validDraftableSeed()
	storeSeedWithContract(t, seeds, contracts, seed)
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
	service, seeds, contracts, _, _ := draftService(t)
	seed := validDraftableSeed()
	storeSeedWithContract(t, seeds, contracts, seed)
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
	service, _, _, drafts, _ := draftService(t)
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
	service, seeds, contracts, _, events := draftService(t)
	seed := validDraftableSeed()
	storeSeedWithContract(t, seeds, contracts, seed)
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
	service, seeds, contracts, _, events := draftService(t)
	seed := validDraftableSeed()
	storeSeedWithContract(t, seeds, contracts, seed)
	created, err := service.Create(context.Background(), seed.ID)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := service.Update(context.Background(), created.ID, updateRequest(t, `{"title": "Reviewed"}`)); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	assertNoForbiddenEvents(t, events.Events())
}

func TestServiceUpdateUsesRequiredTransactionRunner(t *testing.T) {
	service, seeds, contracts, drafts, events := draftService(t)
	txRunner := service.TxRunner.(*fakeTransactionRunner)
	outerCtx := context.WithValue(context.Background(), txContextKey{}, "outer")
	seed := validDraftableSeed()
	storeSeedWithContract(t, seeds, contracts, seed)
	created, err := service.Create(context.Background(), seed.ID)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	txRunner.calls = 0
	events.appendCtx = nil

	if _, err := service.Update(outerCtx, created.ID, updateRequest(t, `{"title": "Reviewed"}`)); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if txRunner.calls != 1 {
		t.Fatalf("TxRunner calls = %d, want 1", txRunner.calls)
	}
	if drafts.updateCtx != txRunner.txCtx {
		t.Fatal("Drafts.Update did not receive transaction context")
	}
	if events.appendCtx != txRunner.txCtx {
		t.Fatal("Events.Append did not receive transaction context")
	}
	if drafts.updateCtx == outerCtx || events.appendCtx == outerCtx {
		t.Fatal("transactional update writes used outer context")
	}
}

func TestServiceMarksDraftReadyForApproval(t *testing.T) {
	service, seeds, contracts, drafts, _ := draftService(t)
	seed := validDraftableSeed()
	storeSeedWithContract(t, seeds, contracts, seed)
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
	contract, ok, err := contracts.Get(context.Background(), seed.ContractID)
	if err != nil {
		t.Fatalf("contracts.Get() error = %v", err)
	}
	if !ok {
		t.Fatal("contract not found")
	}
	if contract.State != spine.ContractStateReadyForApproval {
		t.Fatalf("contract state = %q, want %q", contract.State, spine.ContractStateReadyForApproval)
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
			service, _, _, drafts, events := draftService(t)
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
	service, _, _, drafts, _ := draftService(t)
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
	service, _, _, drafts, _ := draftService(t)
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
	service, seeds, contracts, _, events := draftService(t)
	seed := validDraftableSeed()
	storeSeedWithContract(t, seeds, contracts, seed)
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

func TestServiceMarkReadyForApprovalUsesRequiredTransactionRunner(t *testing.T) {
	service, seeds, contracts, drafts, events := draftService(t)
	txRunner := service.TxRunner.(*fakeTransactionRunner)
	outerCtx := context.WithValue(context.Background(), txContextKey{}, "outer")
	seed := validDraftableSeed()
	storeSeedWithContract(t, seeds, contracts, seed)
	created, err := service.Create(context.Background(), seed.ID)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	txRunner.calls = 0
	events.appendCtx = nil

	if _, err := service.MarkReadyForApproval(outerCtx, created.ID, readyForApprovalRequest()); err != nil {
		t.Fatalf("MarkReadyForApproval() error = %v", err)
	}

	if txRunner.calls != 1 {
		t.Fatalf("TxRunner calls = %d, want 1", txRunner.calls)
	}
	if drafts.markReadyForApprovalCtx != txRunner.txCtx {
		t.Fatal("Drafts.MarkReadyForApproval did not receive transaction context")
	}
	if contracts.markReadyForApprovalCtx != txRunner.txCtx {
		t.Fatal("Contracts.MarkReadyForApproval did not receive transaction context")
	}
	if events.appendCtx != txRunner.txCtx {
		t.Fatal("Events.Append did not receive transaction context")
	}
	if drafts.markReadyForApprovalCtx == outerCtx || contracts.markReadyForApprovalCtx == outerCtx || events.appendCtx == outerCtx {
		t.Fatal("transactional ready-for-approval writes used outer context")
	}
}

func TestServiceReadyForApprovalDoesNotAppendApprovalWorkGateProofEvents(t *testing.T) {
	service, seeds, contracts, _, events := draftService(t)
	seed := validDraftableSeed()
	storeSeedWithContract(t, seeds, contracts, seed)
	created, err := service.Create(context.Background(), seed.ID)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := service.MarkReadyForApproval(context.Background(), created.ID, readyForApprovalRequest()); err != nil {
		t.Fatalf("MarkReadyForApproval() error = %v", err)
	}
	assertNoForbiddenEvents(t, events.Events())
}

func draftService(t *testing.T) (*contractdraft.Service, *fakeContractSeedStore, *fakeContractStore, *fakeContractDraftStore, *fakeEventLog) {
	t.Helper()

	seeds := newFakeContractSeedStore()
	contracts := newFakeContractStore()
	drafts := newFakeContractDraftStore()
	events := newFakeEventLog()
	service := contractdraft.NewService(seeds, contracts, drafts, events, newFakeTransactionRunner(), fixedClock{now: testTime()}, &sequenceIDs{})
	return service, seeds, contracts, drafts, events
}

type txContextKey struct{}

type fakeTransactionRunner struct {
	calls int
	txCtx context.Context
}

func newFakeTransactionRunner() *fakeTransactionRunner {
	return &fakeTransactionRunner{txCtx: context.WithValue(context.Background(), txContextKey{}, "tx")}
}

func (r *fakeTransactionRunner) RunReadCommitted(_ context.Context, fn func(context.Context) error) error {
	r.calls++
	return fn(r.txCtx)
}

type fakeContractSeedStore struct {
	seeds map[spine.ContractSeedID]spine.ContractSeed
}

func newFakeContractSeedStore() *fakeContractSeedStore {
	return &fakeContractSeedStore{seeds: map[spine.ContractSeedID]spine.ContractSeed{}}
}

func (s *fakeContractSeedStore) Create(_ context.Context, seed spine.ContractSeed) error {
	s.seeds[seed.ID] = seed
	return nil
}

func (s *fakeContractSeedStore) Get(_ context.Context, id spine.ContractSeedID) (spine.ContractSeed, bool, error) {
	seed, ok := s.seeds[id]
	return seed, ok, nil
}

type fakeContractStore struct {
	contracts               map[spine.ContractID]spine.Contract
	markDraftCreatedCtx     context.Context
	markReadyForApprovalCtx context.Context
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

func (s *fakeContractStore) MarkDraftCreated(ctx context.Context, id spine.ContractID, draftID spine.ContractDraftID, updatedAt time.Time) error {
	s.markDraftCreatedCtx = ctx
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
	s.markReadyForApprovalCtx = ctx
	contract, ok := s.contracts[id]
	if !ok {
		return nil
	}
	contract.State = spine.ContractStateReadyForApproval
	contract.UpdatedAt = updatedAt.UTC()
	s.contracts[id] = contract
	return nil
}

type fakeContractDraftStore struct {
	drafts                  map[spine.ContractDraftID]spine.ContractDraft
	bySeed                  map[spine.ContractSeedID]spine.ContractDraftID
	createCtx               context.Context
	updateCtx               context.Context
	markReadyForApprovalCtx context.Context
	createErr               error
	getBySeedCtxs           []context.Context
}

func newFakeContractDraftStore() *fakeContractDraftStore {
	return &fakeContractDraftStore{
		drafts: map[spine.ContractDraftID]spine.ContractDraft{},
		bySeed: map[spine.ContractSeedID]spine.ContractDraftID{},
	}
}

func (s *fakeContractDraftStore) Create(ctx context.Context, draft spine.ContractDraft) error {
	s.createCtx = ctx
	if s.createErr != nil {
		return s.createErr
	}
	s.drafts[draft.ID] = draft
	s.bySeed[draft.ContractSeedID] = draft.ID
	return nil
}

func (s *fakeContractDraftStore) Update(ctx context.Context, draft spine.ContractDraft) error {
	s.updateCtx = ctx
	existing, ok := s.drafts[draft.ID]
	if !ok {
		return errors.New("contract draft not found")
	}
	draft.ContractID = existing.ContractID
	draft.ContractSeedID = existing.ContractSeedID
	draft.GoalID = existing.GoalID
	draft.RepoBindingID = existing.RepoBindingID
	draft.State = existing.State
	draft.CreatedAt = existing.CreatedAt
	s.drafts[draft.ID] = draft
	return nil
}

func (s *fakeContractDraftStore) MarkReadyForApproval(ctx context.Context, draft spine.ContractDraft) error {
	s.markReadyForApprovalCtx = ctx
	existing, ok := s.drafts[draft.ID]
	if !ok {
		return errors.New("contract draft not found")
	}
	existing.State = spine.ContractDraftStateReadyForApproval
	s.drafts[draft.ID] = existing
	return nil
}

func (s *fakeContractDraftStore) Get(_ context.Context, id spine.ContractDraftID) (spine.ContractDraft, bool, error) {
	draft, ok := s.drafts[id]
	return draft, ok, nil
}

func (s *fakeContractDraftStore) GetByContractSeedID(ctx context.Context, id spine.ContractSeedID) (spine.ContractDraft, bool, error) {
	s.getBySeedCtxs = append(s.getBySeedCtxs, ctx)
	draftID, ok := s.bySeed[id]
	if !ok {
		return spine.ContractDraft{}, false, nil
	}
	draft, ok := s.drafts[draftID]
	return draft, ok, nil
}

type fakeEventLog struct {
	events    []spine.Event
	appendCtx context.Context
}

func newFakeEventLog() *fakeEventLog {
	return &fakeEventLog{}
}

func (l *fakeEventLog) Append(ctx context.Context, event spine.Event) error {
	l.appendCtx = ctx
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
		ContractID:                 "contract-1",
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
		ContractID:     "contract-1",
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

func storeSeedWithContract(t *testing.T, seeds *fakeContractSeedStore, contracts *fakeContractStore, seed spine.ContractSeed) {
	t.Helper()
	if err := contracts.Create(context.Background(), contractForSeed(seed)); err != nil {
		t.Fatalf("Create contract: %v", err)
	}
	if err := seeds.Create(context.Background(), seed); err != nil {
		t.Fatalf("Create seed: %v", err)
	}
}

func contractForSeed(seed spine.ContractSeed) spine.Contract {
	currentSeedID := seed.ID
	return spine.Contract{
		ID:             seed.ContractID,
		OrganizationID: seed.OrganizationID,
		ProjectID:      seed.ProjectID,
		RepoBindingID:  seed.RepoBindingID,
		GoalID:         seed.GoalID,
		State:          spine.ContractStateSeeded,
		CurrentSeedID:  &currentSeedID,
		CreatedAt:      testTime(),
		UpdatedAt:      testTime(),
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
