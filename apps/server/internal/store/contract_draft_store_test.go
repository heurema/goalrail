package store_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/spine"
	"github.com/heurema/goalrail/apps/server/internal/store"
)

func TestContractDraftStoreCreateAndGet(t *testing.T) {
	draftStore := store.NewContractDraftStore()
	created := validContractDraft()

	if err := draftStore.Create(context.Background(), created); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, ok, err := draftStore.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !ok {
		t.Fatal("Get() ok = false, want true")
	}
	if !reflect.DeepEqual(got, created) {
		t.Fatalf("Get() = %#v, want %#v", got, created)
	}

	bySeed, ok, err := draftStore.GetByContractSeedID(context.Background(), created.ContractSeedID)
	if err != nil {
		t.Fatalf("GetByContractSeedID() error = %v", err)
	}
	if !ok {
		t.Fatal("GetByContractSeedID() ok = false, want true")
	}
	if !reflect.DeepEqual(bySeed, created) {
		t.Fatalf("GetByContractSeedID() = %#v, want %#v", bySeed, created)
	}
}

func TestContractDraftStorePreventsDuplicateDraftForSeed(t *testing.T) {
	draftStore := store.NewContractDraftStore()
	created := validContractDraft()
	if err := draftStore.Create(context.Background(), created); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	duplicate := created
	duplicate.ID = "contract-draft-2"
	if err := draftStore.Create(context.Background(), duplicate); err != store.ErrContractDraftAlreadyDrafted {
		t.Fatalf("duplicate Create() error = %v, want %v", err, store.ErrContractDraftAlreadyDrafted)
	}
}

func TestContractDraftStoreUpdatePreservesIdentityAndSourceFields(t *testing.T) {
	draftStore := store.NewContractDraftStore()
	created := validContractDraft()
	if err := draftStore.Create(context.Background(), created); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	updated := created
	updated.ContractSeedID = "different-seed"
	updated.GoalID = "different-goal"
	updated.RepoBindingID = "different-repo-binding"
	updated.SourceRefs = []spine.SourceRef{{Kind: "forbidden", ID: "source"}}
	updated.State = "ready_for_approval"
	updated.CreatedAt = created.CreatedAt.Add(time.Hour)
	updated.Title = "Reviewed title"
	updated.ProposedScope = []string{"Reviewed scope"}
	updated.ProposedNonGoals = []string{}

	if err := draftStore.Update(context.Background(), updated); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, ok, err := draftStore.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !ok {
		t.Fatal("Get() ok = false, want true")
	}
	if got.ContractSeedID != created.ContractSeedID {
		t.Fatalf("contract_seed_id = %q, want %q", got.ContractSeedID, created.ContractSeedID)
	}
	if got.GoalID != created.GoalID {
		t.Fatalf("goal_id = %q, want %q", got.GoalID, created.GoalID)
	}
	if got.RepoBindingID != created.RepoBindingID {
		t.Fatalf("repo_binding_id = %q, want %q", got.RepoBindingID, created.RepoBindingID)
	}
	if !reflect.DeepEqual(got.SourceRefs, created.SourceRefs) {
		t.Fatalf("source_refs = %#v, want %#v", got.SourceRefs, created.SourceRefs)
	}
	if got.State != created.State {
		t.Fatalf("state = %q, want %q", got.State, created.State)
	}
	if got.CreatedAt != created.CreatedAt {
		t.Fatalf("created_at = %s, want %s", got.CreatedAt, created.CreatedAt)
	}
	if got.Title != "Reviewed title" {
		t.Fatalf("title = %q, want reviewed title", got.Title)
	}
	if !reflect.DeepEqual(got.ProposedScope, []string{"Reviewed scope"}) {
		t.Fatalf("proposed_scope = %#v, want reviewed scope", got.ProposedScope)
	}
	if !reflect.DeepEqual(got.ProposedNonGoals, []string{}) {
		t.Fatalf("proposed_non_goals = %#v, want empty slice", got.ProposedNonGoals)
	}
}

func TestContractDraftStoreUpdateUnknownDraft(t *testing.T) {
	draftStore := store.NewContractDraftStore()

	if err := draftStore.Update(context.Background(), validContractDraft()); err != store.ErrContractDraftNotFound {
		t.Fatalf("Update() error = %v, want %v", err, store.ErrContractDraftNotFound)
	}
}

func TestContractDraftStoreMarkReadyForApprovalPreservesIdentitySourceAndProposedFields(t *testing.T) {
	draftStore := store.NewContractDraftStore()
	created := validContractDraft()
	if err := draftStore.Create(context.Background(), created); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	updated := created
	updated.ContractSeedID = "different-seed"
	updated.GoalID = "different-goal"
	updated.RepoBindingID = "different-repo-binding"
	updated.Title = "Forbidden title mutation"
	updated.IntentSummary = "Forbidden summary mutation"
	updated.ProposedScope = []string{"Forbidden scope mutation"}
	updated.ProposedAcceptanceCriteria = []string{"Forbidden acceptance mutation"}
	updated.ProposedProofExpectations = []string{"Forbidden proof mutation"}
	updated.SourceRefs = []spine.SourceRef{{Kind: "forbidden", ID: "source"}}
	updated.CreatedAt = created.CreatedAt.Add(time.Hour)
	updated.State = spine.ContractDraftStateReadyForApproval

	if err := draftStore.MarkReadyForApproval(context.Background(), updated); err != nil {
		t.Fatalf("MarkReadyForApproval() error = %v", err)
	}

	got, ok, err := draftStore.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !ok {
		t.Fatal("Get() ok = false, want true")
	}
	if got.State != spine.ContractDraftStateReadyForApproval {
		t.Fatalf("state = %q, want %q", got.State, spine.ContractDraftStateReadyForApproval)
	}

	expected := created
	expected.State = spine.ContractDraftStateReadyForApproval
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("stored draft = %#v, want identity/source/proposed unchanged %#v", got, expected)
	}
}

func TestContractDraftStoreMarkReadyForApprovalUnknownDraft(t *testing.T) {
	draftStore := store.NewContractDraftStore()

	if err := draftStore.MarkReadyForApproval(context.Background(), validContractDraft()); err != store.ErrContractDraftNotFound {
		t.Fatalf("MarkReadyForApproval() error = %v, want %v", err, store.ErrContractDraftNotFound)
	}
}

func validContractDraft() spine.ContractDraft {
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
		CreatedAt: time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC),
	}
}
