package store

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

func TestPostgresIntakeStoreCreateBuildsDurableInsert(t *testing.T) {
	ctx := context.Background()
	exec := &recordingProjectContextExecer{}
	store := NewPostgresIntakeStoreWithExecutorAndQuerier(exec, nil)

	if err := store.Create(ctx, validPostgresIntakeRecord()); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if len(exec.calls) != 1 {
		t.Fatalf("Exec calls = %d, want 1", len(exec.calls))
	}
	call := exec.calls[0]
	if !strings.Contains(call.sql, "INSERT INTO intake_records") {
		t.Fatalf("SQL = %q, want intake_records insert", call.sql)
	}
	if !strings.Contains(call.sql, "$1") {
		t.Fatalf("SQL = %q, want PostgreSQL placeholders", call.sql)
	}
	if got, want := len(call.args), 13; got != want {
		t.Fatalf("args len = %d, want %d", got, want)
	}
}

func TestPostgresIntakeStoreGetScansRecord(t *testing.T) {
	ctx := context.Background()
	now := testStoreTime()
	query := &recordingProjectContextQuerier{
		row: fakeProjectContextRow{
			values: []any{
				"018f0000-0000-7000-8000-000000000101",
				"018f0000-0000-7000-8000-000000000002",
				"018f0000-0000-7000-8000-000000000003",
				"018f0000-0000-7000-8000-000000000004",
				[]byte(`{"kind":"codex_skill"}`),
				"Persist intake",
				"Make it durable",
				[]byte(`{"kind":"user","id":"018f0000-0000-7000-8000-000000000001"}`),
				[]byte(`{"kind":"user","id":"018f0000-0000-7000-8000-000000000001"}`),
				spine.IntakeStateReceived,
				false,
				now,
			},
		},
	}
	store := NewPostgresIntakeStoreWithExecutorAndQuerier(&recordingProjectContextExecer{}, query)

	record, ok, err := store.Get(ctx, "018f0000-0000-7000-8000-000000000101")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !ok {
		t.Fatal("Get() ok = false, want true")
	}
	if record.ID != "018f0000-0000-7000-8000-000000000101" {
		t.Fatalf("ID = %q, want persisted id", record.ID)
	}
	if record.OrganizationID != "018f0000-0000-7000-8000-000000000002" {
		t.Fatalf("OrganizationID = %q, want persisted organization", record.OrganizationID)
	}
	if record.Source.Kind != "codex_skill" {
		t.Fatalf("Source.Kind = %q, want codex_skill", record.Source.Kind)
	}
	if record.CreatedAt != now {
		t.Fatalf("CreatedAt = %v, want %v", record.CreatedAt, now)
	}
	if !strings.Contains(query.calls[0].sql, "FROM intake_records") {
		t.Fatalf("SQL = %q, want intake_records select", query.calls[0].sql)
	}
}

func TestPostgresGoalStoreCreateBuildsDurableInsert(t *testing.T) {
	ctx := context.Background()
	exec := &recordingProjectContextExecer{}
	store := NewPostgresGoalStoreWithExecutorAndQuerier(exec, nil)

	if err := store.Create(ctx, validPostgresGoal()); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if len(exec.calls) != 1 {
		t.Fatalf("Exec calls = %d, want 1", len(exec.calls))
	}
	call := exec.calls[0]
	if !strings.Contains(call.sql, "INSERT INTO goals") {
		t.Fatalf("SQL = %q, want goals insert", call.sql)
	}
	if got, want := len(call.args), 16; got != want {
		t.Fatalf("args len = %d, want %d", got, want)
	}
}

func TestPostgresGoalStoreGetByIntakeIDScansPersistedGoal(t *testing.T) {
	ctx := context.Background()
	query := &recordingProjectContextQuerier{row: fakeProjectContextRow{values: validGoalRowValues()}}
	store := NewPostgresGoalStoreWithExecutorAndQuerier(&recordingProjectContextExecer{}, query)

	goal, ok, err := store.GetByIntakeID(ctx, "018f0000-0000-7000-8000-000000000101")
	if err != nil {
		t.Fatalf("GetByIntakeID() error = %v", err)
	}
	if !ok {
		t.Fatal("GetByIntakeID() ok = false, want true")
	}
	if goal.ID != "018f0000-0000-7000-8000-000000000201" {
		t.Fatalf("ID = %q, want persisted goal id", goal.ID)
	}
	if goal.IntakeID != "018f0000-0000-7000-8000-000000000101" {
		t.Fatalf("IntakeID = %q, want persisted intake id", goal.IntakeID)
	}
	if goal.SourceRefs[0].Kind != "intake" {
		t.Fatalf("SourceRefs[0].Kind = %q, want intake", goal.SourceRefs[0].Kind)
	}
	if !strings.Contains(query.calls[0].sql, "WHERE intake_id = $1") {
		t.Fatalf("SQL = %q, want intake_id lookup", query.calls[0].sql)
	}
}

func TestPostgresGoalStoreUpdateReadinessScansPersistedState(t *testing.T) {
	ctx := context.Background()
	query := &recordingProjectContextQuerier{row: fakeProjectContextRow{values: validGoalRowValues()}}
	store := NewPostgresGoalStoreWithExecutorAndQuerier(&recordingProjectContextExecer{}, query)

	goal, ok, err := store.UpdateReadiness(ctx, "018f0000-0000-7000-8000-000000000201", spine.GoalStateNeedsClarification, []spine.GoalReadinessReasonCode{
		spine.GoalReadinessReasonMissingScopeHint,
	})
	if err != nil {
		t.Fatalf("UpdateReadiness() error = %v", err)
	}
	if !ok {
		t.Fatal("UpdateReadiness() ok = false, want true")
	}
	if goal.State != spine.GoalStateNeedsClarification {
		t.Fatalf("State = %q, want needs_clarification", goal.State)
	}
	if len(goal.LastReadinessReasonCodes) != 1 || goal.LastReadinessReasonCodes[0] != spine.GoalReadinessReasonMissingScopeHint {
		t.Fatalf("LastReadinessReasonCodes = %#v, want persisted missing_scope_hint", goal.LastReadinessReasonCodes)
	}
	if !strings.Contains(query.calls[0].sql, "UPDATE goals SET") {
		t.Fatalf("SQL = %q, want goals update", query.calls[0].sql)
	}
	if !strings.Contains(query.calls[0].sql, "RETURNING") {
		t.Fatalf("SQL = %q, want returning persisted goal", query.calls[0].sql)
	}
}

func TestPostgresEventLogAppendBuildsDurableInsert(t *testing.T) {
	ctx := context.Background()
	exec := &recordingProjectContextExecer{}
	events := NewPostgresEventLogWithExecutorAndQuerier(exec, nil)

	err := events.Append(ctx, spine.Event{
		ID:             "018f0000-0000-7000-8000-000000000301",
		Type:           "intake.received",
		OrganizationID: "018f0000-0000-7000-8000-000000000002",
		ProjectID:      "018f0000-0000-7000-8000-000000000003",
		RepoBindingID:  "018f0000-0000-7000-8000-000000000004",
		EntityType:     "IntakeRecord",
		EntityID:       "018f0000-0000-7000-8000-000000000101",
		Timestamp:      testStoreTime(),
		Payload:        []byte(`{"id":"018f0000-0000-7000-8000-000000000101"}`),
	})
	if err != nil {
		t.Fatalf("Append() error = %v", err)
	}

	if len(exec.calls) != 1 {
		t.Fatalf("Exec calls = %d, want 1", len(exec.calls))
	}
	call := exec.calls[0]
	if !strings.Contains(call.sql, "INSERT INTO events") {
		t.Fatalf("SQL = %q, want events insert", call.sql)
	}
	if got, want := len(call.args), 10; got != want {
		t.Fatalf("args len = %d, want %d", got, want)
	}
}

func TestPostgresContractSeedStoreCreateBuildsDurableInsert(t *testing.T) {
	ctx := context.Background()
	exec := &recordingProjectContextExecer{}
	store := NewPostgresContractSeedStoreWithExecutorAndQuerier(exec, nil)

	if err := store.Create(ctx, validPostgresContractSeed()); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if len(exec.calls) != 1 {
		t.Fatalf("Exec calls = %d, want 1", len(exec.calls))
	}
	call := exec.calls[0]
	if !strings.Contains(call.sql, "INSERT INTO contract_seeds") {
		t.Fatalf("SQL = %q, want contract_seeds insert", call.sql)
	}
	if got, want := len(call.args), 14; got != want {
		t.Fatalf("args len = %d, want %d", got, want)
	}
}

func TestPostgresContractSeedStoreGetByGoalIDScansPersistedSeed(t *testing.T) {
	ctx := context.Background()
	query := &recordingProjectContextQuerier{row: fakeProjectContextRow{values: validContractSeedRowValues()}}
	store := NewPostgresContractSeedStoreWithExecutorAndQuerier(&recordingProjectContextExecer{}, query)

	seed, ok, err := store.GetByGoalID(ctx, "018f0000-0000-7000-8000-000000000201")
	if err != nil {
		t.Fatalf("GetByGoalID() error = %v", err)
	}
	if !ok {
		t.Fatal("GetByGoalID() ok = false, want true")
	}
	if seed.ID != "018f0000-0000-7000-8000-000000000401" {
		t.Fatalf("ID = %q, want persisted seed id", seed.ID)
	}
	if seed.OrganizationID != "018f0000-0000-7000-8000-000000000002" {
		t.Fatalf("OrganizationID = %q, want persisted organization", seed.OrganizationID)
	}
	if seed.SourceRefs[0].Kind != "goal" {
		t.Fatalf("SourceRefs[0].Kind = %q, want goal", seed.SourceRefs[0].Kind)
	}
	if !strings.Contains(query.calls[0].sql, "FROM contract_seeds") {
		t.Fatalf("SQL = %q, want contract_seeds select", query.calls[0].sql)
	}
	if !strings.Contains(query.calls[0].sql, "WHERE goal_id = $1") {
		t.Fatalf("SQL = %q, want goal_id lookup", query.calls[0].sql)
	}
}

func TestPostgresContractDraftStoreCreateBuildsDurableInsert(t *testing.T) {
	ctx := context.Background()
	exec := &recordingProjectContextExecer{}
	store := NewPostgresContractDraftStoreWithExecutorAndQuerier(exec, nil)

	if err := store.Create(ctx, validPostgresContractDraft()); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if len(exec.calls) != 1 {
		t.Fatalf("Exec calls = %d, want 1", len(exec.calls))
	}
	call := exec.calls[0]
	if !strings.Contains(call.sql, "INSERT INTO contract_drafts") {
		t.Fatalf("SQL = %q, want contract_drafts insert", call.sql)
	}
	if got, want := len(call.args), 19; got != want {
		t.Fatalf("args len = %d, want %d", got, want)
	}
	assertJSONBytesEqual(t, call.args[8], `["Persist contract seeds","Preserve draft arrays"]`)
	assertJSONBytesEqual(t, call.args[11], `["Seed can be read after restart","Draft array values are retained"]`)
	assertJSONBytesEqual(t, call.args[13], `["Provide evidence that acceptance criteria were checked.","Show persisted draft arrays"]`)
}

func TestPostgresContractDraftStoreUpdateBuildsDurableUpdate(t *testing.T) {
	ctx := context.Background()
	exec := &recordingProjectContextExecer{}
	store := NewPostgresContractDraftStoreWithExecutorAndQuerier(exec, nil)

	updated := validPostgresContractDraft()
	updated.Title = "Reviewed persisted draft"
	updated.ProposedNonGoals = []string{}
	updated.ProposedScope = []string{"Reviewed persisted scope"}
	updated.ProposedAcceptanceCriteria = []string{"Reviewed persisted acceptance"}

	if err := store.Update(ctx, updated); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if len(exec.calls) != 1 {
		t.Fatalf("Exec calls = %d, want 1", len(exec.calls))
	}
	call := exec.calls[0]
	if !strings.Contains(call.sql, "UPDATE contract_drafts") {
		t.Fatalf("SQL = %q, want contract_drafts update", call.sql)
	}
	if !strings.Contains(call.sql, "WHERE id =") {
		t.Fatalf("SQL = %q, want id lookup", call.sql)
	}
	if strings.Contains(call.sql, "contract_seed_id =") || strings.Contains(call.sql, "goal_id =") || strings.Contains(call.sql, "repo_binding_id =") || strings.Contains(call.sql, "state =") || strings.Contains(call.sql, "source_refs =") || strings.Contains(call.sql, "created_at =") {
		t.Fatalf("SQL = %q, should not update identity/source/state fields", call.sql)
	}
	if got, want := len(call.args), 11; got != want {
		t.Fatalf("args len = %d, want %d", got, want)
	}
	assertJSONBytesEqual(t, call.args[2], `["Reviewed persisted scope"]`)
	assertJSONBytesEqual(t, call.args[3], `[]`)
	assertJSONBytesEqual(t, call.args[5], `["Reviewed persisted acceptance"]`)
}

func TestPostgresContractDraftStoreMarkReadyForApprovalBuildsStateOnlyUpdate(t *testing.T) {
	ctx := context.Background()
	exec := &recordingProjectContextExecer{}
	store := NewPostgresContractDraftStoreWithExecutorAndQuerier(exec, nil)

	updated := validPostgresContractDraft()
	updated.State = spine.ContractDraftStateReadyForApproval

	if err := store.MarkReadyForApproval(ctx, updated); err != nil {
		t.Fatalf("MarkReadyForApproval() error = %v", err)
	}

	if len(exec.calls) != 1 {
		t.Fatalf("Exec calls = %d, want 1", len(exec.calls))
	}
	call := exec.calls[0]
	if !strings.Contains(call.sql, "UPDATE contract_drafts") {
		t.Fatalf("SQL = %q, want contract_drafts update", call.sql)
	}
	if !strings.Contains(call.sql, "state =") {
		t.Fatalf("SQL = %q, want state update", call.sql)
	}
	if !strings.Contains(call.sql, "updated_at =") {
		t.Fatalf("SQL = %q, want updated_at update", call.sql)
	}
	if !strings.Contains(call.sql, "WHERE id =") {
		t.Fatalf("SQL = %q, want id lookup", call.sql)
	}
	if strings.Contains(call.sql, "title =") ||
		strings.Contains(call.sql, "intent_summary =") ||
		strings.Contains(call.sql, "contract_seed_id =") ||
		strings.Contains(call.sql, "goal_id =") ||
		strings.Contains(call.sql, "repo_binding_id =") ||
		strings.Contains(call.sql, "source_refs =") ||
		strings.Contains(call.sql, "created_at =") ||
		strings.Contains(call.sql, "proposed_scope =") ||
		strings.Contains(call.sql, "proposed_acceptance_criteria =") ||
		strings.Contains(call.sql, "proposed_proof_expectations =") {
		t.Fatalf("SQL = %q, should update state only", call.sql)
	}
	if got, want := len(call.args), 3; got != want {
		t.Fatalf("args len = %d, want %d", got, want)
	}
	if call.args[0] != spine.ContractDraftStateReadyForApproval {
		t.Fatalf("state arg = %#v, want ready_for_approval", call.args[0])
	}
}

func TestPostgresContractDraftStoreGetByContractSeedIDScansPersistedDraft(t *testing.T) {
	ctx := context.Background()
	query := &recordingProjectContextQuerier{row: fakeProjectContextRow{values: validContractDraftRowValues()}}
	store := NewPostgresContractDraftStoreWithExecutorAndQuerier(&recordingProjectContextExecer{}, query)

	draft, ok, err := store.GetByContractSeedID(ctx, "018f0000-0000-7000-8000-000000000401")
	if err != nil {
		t.Fatalf("GetByContractSeedID() error = %v", err)
	}
	if !ok {
		t.Fatal("GetByContractSeedID() ok = false, want true")
	}
	if draft.ID != "018f0000-0000-7000-8000-000000000501" {
		t.Fatalf("ID = %q, want persisted draft id", draft.ID)
	}
	if draft.OrganizationID != "018f0000-0000-7000-8000-000000000002" {
		t.Fatalf("OrganizationID = %q, want persisted organization", draft.OrganizationID)
	}
	if got, want := draft.ProposedScope, []string{"Persist contract seeds", "Preserve draft arrays"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ProposedScope = %#v, want %#v", got, want)
	}
	if got, want := draft.ProposedAcceptanceCriteria, []string{"Seed can be read after restart", "Draft array values are retained"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ProposedAcceptanceCriteria = %#v, want %#v", got, want)
	}
	if got, want := draft.ProposedProofExpectations, []string{"Provide evidence that acceptance criteria were checked.", "Show persisted draft arrays"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ProposedProofExpectations = %#v, want %#v", got, want)
	}
	if len(draft.SourceRefs) != 2 || draft.SourceRefs[0].Kind != "contract_seed" {
		t.Fatalf("SourceRefs = %#v, want contract_seed refs", draft.SourceRefs)
	}
	if !strings.Contains(query.calls[0].sql, "FROM contract_drafts") {
		t.Fatalf("SQL = %q, want contract_drafts select", query.calls[0].sql)
	}
	if !strings.Contains(query.calls[0].sql, "WHERE contract_seed_id = $1") {
		t.Fatalf("SQL = %q, want contract_seed_id lookup", query.calls[0].sql)
	}
}

func validPostgresIntakeRecord() spine.IntakeRecord {
	now := testStoreTime()
	return spine.IntakeRecord{
		ID:             "018f0000-0000-7000-8000-000000000101",
		OrganizationID: "018f0000-0000-7000-8000-000000000002",
		ProjectID:      "018f0000-0000-7000-8000-000000000003",
		RepoBindingID:  "018f0000-0000-7000-8000-000000000004",
		Source: spine.IntakeSource{
			Kind: "codex_skill",
		},
		Title:         "Persist intake",
		Body:          "Make it durable",
		RequestAuthor: spine.ActorRef{Kind: "user", ID: "018f0000-0000-7000-8000-000000000001"},
		IntentOwner:   spine.ActorRef{Kind: "user", ID: "018f0000-0000-7000-8000-000000000001"},
		State:         spine.IntakeStateReceived,
		CreatedAt:     now,
	}
}

func validPostgresGoal() spine.Goal {
	now := testStoreTime()
	return spine.Goal{
		ID:             "018f0000-0000-7000-8000-000000000201",
		IntakeID:       "018f0000-0000-7000-8000-000000000101",
		OrganizationID: "018f0000-0000-7000-8000-000000000002",
		ProjectID:      "018f0000-0000-7000-8000-000000000003",
		RepoBindingID:  "018f0000-0000-7000-8000-000000000004",
		Title:          "Persist goal",
		Summary:        "Make the goal durable",
		SourceRefs: []spine.SourceRef{
			{Kind: "intake", ID: "018f0000-0000-7000-8000-000000000101"},
		},
		RequestAuthor: spine.ActorRef{Kind: "user", ID: "018f0000-0000-7000-8000-000000000001"},
		IntentOwner:   spine.ActorRef{Kind: "user", ID: "018f0000-0000-7000-8000-000000000001"},
		State:         spine.GoalStateCreated,
		CreatedAt:     now,
	}
}

func validPostgresContractSeed() spine.ContractSeed {
	now := testStoreTime()
	return spine.ContractSeed{
		ID:             "018f0000-0000-7000-8000-000000000401",
		OrganizationID: "018f0000-0000-7000-8000-000000000002",
		ProjectID:      "018f0000-0000-7000-8000-000000000003",
		RepoBindingID:  "018f0000-0000-7000-8000-000000000004",
		GoalID:         "018f0000-0000-7000-8000-000000000201",
		Title:          "Persist seed",
		IntentSummary:  "Make contract seed durable",
		IntentOwner:    spine.ActorRef{Kind: "user", ID: "018f0000-0000-7000-8000-000000000001"},
		ScopeHint:      "Persist contract seeds",
		AcceptanceHint: "Seed can be read after restart",
		SourceRefs: []spine.SourceRef{
			{Kind: "goal", ID: "018f0000-0000-7000-8000-000000000201"},
		},
		State:     spine.ContractSeedStateCreated,
		CreatedAt: now,
	}
}

func validPostgresContractDraft() spine.ContractDraft {
	now := testStoreTime()
	return spine.ContractDraft{
		ID:                         "018f0000-0000-7000-8000-000000000501",
		OrganizationID:             "018f0000-0000-7000-8000-000000000002",
		ProjectID:                  "018f0000-0000-7000-8000-000000000003",
		RepoBindingID:              "018f0000-0000-7000-8000-000000000004",
		ContractSeedID:             "018f0000-0000-7000-8000-000000000401",
		GoalID:                     "018f0000-0000-7000-8000-000000000201",
		Title:                      "Persist seed",
		IntentSummary:              "Make contract seed durable",
		ProposedScope:              []string{"Persist contract seeds", "Preserve draft arrays"},
		ProposedNonGoals:           []string{},
		ProposedConstraints:        []string{},
		ProposedAcceptanceCriteria: []string{"Seed can be read after restart", "Draft array values are retained"},
		ProposedExpectedChecks:     []string{},
		ProposedProofExpectations:  []string{"Provide evidence that acceptance criteria were checked.", "Show persisted draft arrays"},
		RiskHints:                  []string{},
		SourceRefs: []spine.SourceRef{
			{Kind: "contract_seed", ID: "018f0000-0000-7000-8000-000000000401"},
			{Kind: "goal", ID: "018f0000-0000-7000-8000-000000000201"},
		},
		State:     spine.ContractDraftStateDraft,
		CreatedAt: now,
	}
}

func validGoalRowValues() []any {
	return []any{
		"018f0000-0000-7000-8000-000000000201",
		"018f0000-0000-7000-8000-000000000002",
		"018f0000-0000-7000-8000-000000000003",
		"018f0000-0000-7000-8000-000000000004",
		"018f0000-0000-7000-8000-000000000101",
		"Persist goal",
		"Make the goal durable",
		"",
		"",
		[]byte(`[{"kind":"intake","id":"018f0000-0000-7000-8000-000000000101"}]`),
		[]byte(`{"kind":"user","id":"018f0000-0000-7000-8000-000000000001"}`),
		[]byte(`{"kind":"user","id":"018f0000-0000-7000-8000-000000000001"}`),
		spine.GoalStateNeedsClarification,
		[]byte(`["missing_scope_hint"]`),
		testStoreTime(),
	}
}

func validContractSeedRowValues() []any {
	return []any{
		"018f0000-0000-7000-8000-000000000401",
		"018f0000-0000-7000-8000-000000000002",
		"018f0000-0000-7000-8000-000000000003",
		"018f0000-0000-7000-8000-000000000004",
		"018f0000-0000-7000-8000-000000000201",
		"Persist seed",
		"Make contract seed durable",
		[]byte(`{"kind":"user","id":"018f0000-0000-7000-8000-000000000001"}`),
		"Persist contract seeds",
		"Seed can be read after restart",
		[]byte(`[{"kind":"goal","id":"018f0000-0000-7000-8000-000000000201"}]`),
		string(spine.ContractSeedStateCreated),
		testStoreTime(),
	}
}

func validContractDraftRowValues() []any {
	return []any{
		"018f0000-0000-7000-8000-000000000501",
		"018f0000-0000-7000-8000-000000000002",
		"018f0000-0000-7000-8000-000000000003",
		"018f0000-0000-7000-8000-000000000004",
		"018f0000-0000-7000-8000-000000000401",
		"018f0000-0000-7000-8000-000000000201",
		"Persist seed",
		"Make contract seed durable",
		[]byte(`["Persist contract seeds","Preserve draft arrays"]`),
		[]byte(`[]`),
		[]byte(`[]`),
		[]byte(`["Seed can be read after restart","Draft array values are retained"]`),
		[]byte(`[]`),
		[]byte(`["Provide evidence that acceptance criteria were checked.","Show persisted draft arrays"]`),
		[]byte(`[]`),
		[]byte(`[{"kind":"contract_seed","id":"018f0000-0000-7000-8000-000000000401"},{"kind":"goal","id":"018f0000-0000-7000-8000-000000000201"}]`),
		string(spine.ContractDraftStateDraft),
		testStoreTime(),
	}
}

func testStoreTime() time.Time {
	return time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
}

func assertJSONBytesEqual(t *testing.T, got any, want string) {
	t.Helper()
	gotBytes, ok := got.([]byte)
	if !ok {
		t.Fatalf("json arg type = %T, want []byte", got)
	}
	if string(gotBytes) != want {
		t.Fatalf("json arg = %s, want %s", gotBytes, want)
	}
}
