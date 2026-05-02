package store

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

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

func TestPostgresClarificationRequestStoreCreateBuildsDurableInsert(t *testing.T) {
	ctx := context.Background()
	exec := &recordingProjectContextExecer{}
	store := NewPostgresClarificationRequestStoreWithExecutorAndQuerier(exec, nil)

	if err := store.Create(ctx, validPostgresClarificationRequest()); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if len(exec.calls) != 1 {
		t.Fatalf("Exec calls = %d, want 1", len(exec.calls))
	}
	call := exec.calls[0]
	if !strings.Contains(call.sql, "INSERT INTO clarification_requests") {
		t.Fatalf("SQL = %q, want clarification_requests insert", call.sql)
	}
	if !strings.Contains(call.sql, "FROM goals") {
		t.Fatalf("SQL = %q, want goal context select", call.sql)
	}
	if got, want := len(call.args), 8; got != want {
		t.Fatalf("args len = %d, want %d", got, want)
	}
}

func TestPostgresClarificationRequestStoreCreateDetectsMissingGoal(t *testing.T) {
	ctx := context.Background()
	exec := &recordingProjectContextExecer{tag: pgconn.NewCommandTag("INSERT 0 0")}
	store := NewPostgresClarificationRequestStoreWithExecutorAndQuerier(exec, nil)

	err := store.Create(ctx, validPostgresClarificationRequest())
	if !errors.Is(err, errPostgresClarificationGoalNotFound) {
		t.Fatalf("Create() error = %v, want errPostgresClarificationGoalNotFound", err)
	}
}

func TestPostgresClarificationRequestStoreGetOpenByGoalIDScansRequest(t *testing.T) {
	ctx := context.Background()
	query := &recordingProjectContextQuerier{row: fakeProjectContextRow{values: validClarificationRequestRowValues()}}
	store := NewPostgresClarificationRequestStoreWithExecutorAndQuerier(&recordingProjectContextExecer{}, query)

	request, ok, err := store.GetOpenByGoalID(ctx, "018f0000-0000-7000-8000-000000000201")
	if err != nil {
		t.Fatalf("GetOpenByGoalID() error = %v", err)
	}
	if !ok {
		t.Fatal("GetOpenByGoalID() ok = false, want true")
	}
	if request.ID != "018f0000-0000-7000-8000-000000000a01" {
		t.Fatalf("ID = %q, want persisted clarification request id", request.ID)
	}
	if len(request.ReasonCodes) != 1 || request.ReasonCodes[0] != spine.GoalReadinessReasonMissingScopeHint {
		t.Fatalf("ReasonCodes = %#v, want missing_scope_hint", request.ReasonCodes)
	}
	if len(request.Questions) != 1 || request.Questions[0].MapsTo != spine.ClarificationMapsToGoalScopeHint {
		t.Fatalf("Questions = %#v, want persisted question mapping", request.Questions)
	}
	if request.Target.ActorRef == nil || request.Target.ActorRef.ID != "018f0000-0000-7000-8000-000000000001" {
		t.Fatalf("Target = %#v, want persisted target actor", request.Target)
	}
	if !strings.Contains(query.calls[0].sql, "FROM clarification_requests") {
		t.Fatalf("SQL = %q, want clarification_requests select", query.calls[0].sql)
	}
	if !strings.Contains(query.calls[0].sql, "state = $2") {
		t.Fatalf("SQL = %q, want open state filter", query.calls[0].sql)
	}
}

func TestPostgresClarificationRequestStoreUpdateStateReturnsAnsweredRequest(t *testing.T) {
	ctx := context.Background()
	values := validClarificationRequestRowValues()
	values[2] = spine.ClarificationRequestStateAnswered
	query := &recordingProjectContextQuerier{row: fakeProjectContextRow{values: values}}
	store := NewPostgresClarificationRequestStoreWithExecutorAndQuerier(&recordingProjectContextExecer{}, query)

	request, ok, err := store.UpdateState(ctx, "018f0000-0000-7000-8000-000000000a01", spine.ClarificationRequestStateAnswered)
	if err != nil {
		t.Fatalf("UpdateState() error = %v", err)
	}
	if !ok {
		t.Fatal("UpdateState() ok = false, want true")
	}
	if request.State != spine.ClarificationRequestStateAnswered {
		t.Fatalf("State = %q, want answered", request.State)
	}
	if !strings.Contains(query.calls[0].sql, "UPDATE clarification_requests SET") {
		t.Fatalf("SQL = %q, want clarification_requests update", query.calls[0].sql)
	}
}

func TestPostgresClarificationRequestStoreMapsDuplicateOpenRequest(t *testing.T) {
	ctx := context.Background()
	store := NewPostgresClarificationRequestStoreWithExecutorAndQuerier(uniqueViolationExecer{constraint: "clarification_requests_one_open_per_goal_idx"}, nil)

	err := store.Create(ctx, validPostgresClarificationRequest())
	if err != ErrClarificationRequestAlreadyOpen {
		t.Fatalf("Create() error = %v, want ErrClarificationRequestAlreadyOpen", err)
	}
}

func TestPostgresClarificationAnswerStoreCreateBuildsDurableInsert(t *testing.T) {
	ctx := context.Background()
	exec := &recordingProjectContextExecer{}
	store := NewPostgresClarificationAnswerStoreWithExecutorAndQuerier(exec, nil)

	if err := store.Create(ctx, validPostgresClarificationAnswer()); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if len(exec.calls) != 1 {
		t.Fatalf("Exec calls = %d, want 1", len(exec.calls))
	}
	call := exec.calls[0]
	if !strings.Contains(call.sql, "INSERT INTO clarification_answers") {
		t.Fatalf("SQL = %q, want clarification_answers insert", call.sql)
	}
	if !strings.Contains(call.sql, "FROM clarification_requests") {
		t.Fatalf("SQL = %q, want clarification request context select", call.sql)
	}
	if got, want := len(call.args), 6; got != want {
		t.Fatalf("args len = %d, want %d", got, want)
	}
}

func TestPostgresClarificationAnswerStoreGetByRequestIDScansAnswer(t *testing.T) {
	ctx := context.Background()
	query := &recordingProjectContextQuerier{row: fakeProjectContextRow{values: validClarificationAnswerRowValues()}}
	store := NewPostgresClarificationAnswerStoreWithExecutorAndQuerier(&recordingProjectContextExecer{}, query)

	answer, ok, err := store.GetByRequestID(ctx, "018f0000-0000-7000-8000-000000000a01")
	if err != nil {
		t.Fatalf("GetByRequestID() error = %v", err)
	}
	if !ok {
		t.Fatal("GetByRequestID() ok = false, want true")
	}
	if answer.ID != "018f0000-0000-7000-8000-000000000b01" {
		t.Fatalf("ID = %q, want persisted answer id", answer.ID)
	}
	if answer.State != spine.ClarificationAnswerStateRecorded {
		t.Fatalf("State = %q, want recorded", answer.State)
	}
	if len(answer.Answers) != 1 || answer.Answers[0].Value != "Persisted scope" {
		t.Fatalf("Answers = %#v, want persisted answer item", answer.Answers)
	}
	if answer.SubmittedBy.ID != "018f0000-0000-7000-8000-000000000001" {
		t.Fatalf("SubmittedBy = %#v, want persisted actor", answer.SubmittedBy)
	}
	if !strings.Contains(query.calls[0].sql, "WHERE clarification_request_id = $1") {
		t.Fatalf("SQL = %q, want request id lookup", query.calls[0].sql)
	}
}

func TestPostgresClarificationAnswerStoreMarkAppliedPersistsActorAndTimestamp(t *testing.T) {
	ctx := context.Background()
	exec := &recordingProjectContextExecer{}
	store := NewPostgresClarificationAnswerStoreWithExecutorAndQuerier(exec, nil)

	marked, err := store.MarkApplied(
		ctx,
		"018f0000-0000-7000-8000-000000000b01",
		spine.ActorRef{Kind: "user", ID: "018f0000-0000-7000-8000-000000000001"},
		testStoreTime(),
	)
	if err != nil {
		t.Fatalf("MarkApplied() error = %v", err)
	}
	if !marked {
		t.Fatal("MarkApplied() marked = false, want true")
	}
	if len(exec.calls) != 1 {
		t.Fatalf("Exec calls = %d, want 1", len(exec.calls))
	}
	call := exec.calls[0]
	for _, want := range []string{"UPDATE clarification_answers SET", "applied = $1", "applied_by =", "applied_at =", "WHERE id ="} {
		if !strings.Contains(call.sql, want) {
			t.Fatalf("SQL = %q, want %q", call.sql, want)
		}
	}
}

func TestPostgresClarificationAnswerStoreMapsDuplicateRequestAnswer(t *testing.T) {
	ctx := context.Background()
	store := NewPostgresClarificationAnswerStoreWithExecutorAndQuerier(uniqueViolationExecer{constraint: "clarification_answers_request_id_unique"}, nil)

	err := store.Create(ctx, validPostgresClarificationAnswer())
	if err != ErrClarificationAnswerAlreadyRecorded {
		t.Fatalf("Create() error = %v, want ErrClarificationAnswerAlreadyRecorded", err)
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
	if got, want := len(call.args), 15; got != want {
		t.Fatalf("args len = %d, want %d", got, want)
	}
}

func TestPostgresContractStoreCreateBuildsDurableInsert(t *testing.T) {
	ctx := context.Background()
	exec := &recordingProjectContextExecer{}
	store := NewPostgresContractStoreWithExecutorAndQuerier(exec, nil)

	if err := store.Create(ctx, validPostgresContract()); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if len(exec.calls) != 1 {
		t.Fatalf("Exec calls = %d, want 1", len(exec.calls))
	}
	call := exec.calls[0]
	if !strings.Contains(call.sql, "INSERT INTO contracts") {
		t.Fatalf("SQL = %q, want contracts insert", call.sql)
	}
	if got, want := len(call.args), 11; got != want {
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
	if got, want := len(call.args), 20; got != want {
		t.Fatalf("args len = %d, want %d", got, want)
	}
	assertJSONBytesEqual(t, call.args[9], `["Persist contract seeds","Preserve draft arrays"]`)
	assertJSONBytesEqual(t, call.args[12], `["Seed can be read after restart","Draft array values are retained"]`)
	assertJSONBytesEqual(t, call.args[14], `["Provide evidence that acceptance criteria were checked.","Show persisted draft arrays"]`)
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

func TestPostgresApprovedContractStoreCreateBuildsDurableInsert(t *testing.T) {
	ctx := context.Background()
	exec := &recordingProjectContextExecer{}
	store := NewPostgresApprovedContractStoreWithExecutorAndQuerier(exec, nil)

	if err := store.Create(ctx, validPostgresApprovedContract()); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if len(exec.calls) != 1 {
		t.Fatalf("Exec calls = %d, want 1", len(exec.calls))
	}
	call := exec.calls[0]
	if !strings.Contains(call.sql, "INSERT INTO approved_contracts") {
		t.Fatalf("SQL = %q, want approved_contracts insert", call.sql)
	}
	if got, want := len(call.args), 23; got != want {
		t.Fatalf("args len = %d, want %d", got, want)
	}
	assertJSONBytesEqual(t, call.args[10], `["Persist contract seeds","Preserve draft arrays"]`)
	assertJSONBytesEqual(t, call.args[13], `["Seed can be read after restart","Draft array values are retained"]`)
	assertJSONBytesEqual(t, call.args[15], `["Provide evidence that acceptance criteria were checked.","Show persisted draft arrays"]`)
	assertJSONBytesEqual(t, call.args[17], `{"kind":"user","id":"018f0000-0000-7000-8000-000000000001"}`)
}

func TestPostgresApprovedContractStoreGetByContractDraftIDScansPersistedApprovedContract(t *testing.T) {
	ctx := context.Background()
	query := &recordingProjectContextQuerier{row: fakeProjectContextRow{values: validApprovedContractRowValues()}}
	store := NewPostgresApprovedContractStoreWithExecutorAndQuerier(&recordingProjectContextExecer{}, query)

	approved, ok, err := store.GetByContractDraftID(ctx, "018f0000-0000-7000-8000-000000000501")
	if err != nil {
		t.Fatalf("GetByContractDraftID() error = %v", err)
	}
	if !ok {
		t.Fatal("GetByContractDraftID() ok = false, want true")
	}
	if approved.ID != "018f0000-0000-7000-8000-000000000601" {
		t.Fatalf("ID = %q, want persisted approved contract id", approved.ID)
	}
	if approved.State != spine.ApprovedContractStateApproved {
		t.Fatalf("State = %q, want approved", approved.State)
	}
	if got, want := approved.Scope, []string{"Persist contract seeds", "Preserve draft arrays"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Scope = %#v, want %#v", got, want)
	}
	if approved.ApprovedBy.ID != "018f0000-0000-7000-8000-000000000001" {
		t.Fatalf("ApprovedBy = %#v, want persisted actor", approved.ApprovedBy)
	}
	if !strings.Contains(query.calls[0].sql, "FROM approved_contracts") {
		t.Fatalf("SQL = %q, want approved_contracts select", query.calls[0].sql)
	}
	if !strings.Contains(query.calls[0].sql, "WHERE contract_draft_id = $1") {
		t.Fatalf("SQL = %q, want contract_draft_id lookup", query.calls[0].sql)
	}
}

func TestPostgresWorkItemStoreCreateBuildsDurableInsert(t *testing.T) {
	ctx := context.Background()
	exec := &recordingProjectContextExecer{}
	store := NewPostgresWorkItemStoreWithExecutorAndQuerier(exec, nil)

	if err := store.Create(ctx, validPostgresWorkItem()); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if len(exec.calls) != 1 {
		t.Fatalf("Exec calls = %d, want 1", len(exec.calls))
	}
	call := exec.calls[0]
	if !strings.Contains(call.sql, "INSERT INTO work_items") {
		t.Fatalf("SQL = %q, want work_items insert", call.sql)
	}
	if got, want := len(call.args), 19; got != want {
		t.Fatalf("args len = %d, want %d", got, want)
	}
	assertJSONBytesEqual(t, call.args[10], `["Persist contract seeds","Preserve draft arrays"]`)
	assertJSONBytesEqual(t, call.args[11], `["acceptance_criteria[0]","acceptance_criteria[1]"]`)
	assertJSONBytesEqual(t, call.args[12], `["proof_expectations[0]","proof_expectations[1]"]`)
}

func TestPostgresWorkItemStoreGetByApprovedContractIDScansPersistedWorkItem(t *testing.T) {
	ctx := context.Background()
	query := &recordingProjectContextQuerier{row: fakeProjectContextRow{values: validWorkItemRowValues()}}
	store := NewPostgresWorkItemStoreWithExecutorAndQuerier(&recordingProjectContextExecer{}, query)

	item, ok, err := store.GetByApprovedContractID(ctx, "018f0000-0000-7000-8000-000000000601")
	if err != nil {
		t.Fatalf("GetByApprovedContractID() error = %v", err)
	}
	if !ok {
		t.Fatal("GetByApprovedContractID() ok = false, want true")
	}
	if item.ID != "018f0000-0000-7000-8000-000000000701" {
		t.Fatalf("ID = %q, want persisted work item id", item.ID)
	}
	if item.Status != spine.WorkItemStatusPlanned {
		t.Fatalf("Status = %q, want planned", item.Status)
	}
	if item.PlanID != "018f0000-0000-7000-8000-000000000801" || item.ProposalID != "018f0000-0000-7000-8000-000000000901" {
		t.Fatalf("planning trace = %q/%q, want plan/proposal ids", item.PlanID, item.ProposalID)
	}
	if got, want := item.Scope, []string{"Persist contract seeds", "Preserve draft arrays"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Scope = %#v, want %#v", got, want)
	}
	if item.OrderIndex == nil || *item.OrderIndex != 1 {
		t.Fatalf("OrderIndex = %#v, want 1", item.OrderIndex)
	}
	if len(item.SourceRefs) != 1 || item.SourceRefs[0].Kind != "approved_contract" {
		t.Fatalf("SourceRefs = %#v, want approved_contract ref", item.SourceRefs)
	}
	if !strings.Contains(query.calls[0].sql, "FROM work_items") {
		t.Fatalf("SQL = %q, want work_items select", query.calls[0].sql)
	}
	if !strings.Contains(query.calls[0].sql, "WHERE approved_contract_id = $1") {
		t.Fatalf("SQL = %q, want approved_contract_id lookup", query.calls[0].sql)
	}
}

func TestPostgresWorkItemPlanStoreCreateGetAndMark(t *testing.T) {
	ctx := context.Background()
	exec := &recordingProjectContextExecer{}
	query := &recordingProjectContextQuerier{row: fakeProjectContextRow{values: validWorkItemPlanRowValues()}}
	store := NewPostgresWorkItemPlanStoreWithExecutorAndQuerier(exec, query)

	if err := store.Create(ctx, validPostgresWorkItemPlan()); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if len(exec.calls) != 1 || !strings.Contains(exec.calls[0].sql, "INSERT INTO work_item_plans") {
		t.Fatalf("create SQL calls = %#v, want work_item_plans insert", exec.calls)
	}
	plan, ok, err := store.Get(ctx, "018f0000-0000-7000-8000-000000000801")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !ok {
		t.Fatal("Get() ok = false, want true")
	}
	if plan.State != spine.WorkItemPlanStateQueued || plan.RequestedBy.ID == "" {
		t.Fatalf("plan state/requested_by = %q/%#v, want queued actor", plan.State, plan.RequestedBy)
	}
	if !strings.Contains(query.calls[0].sql, "FROM work_item_plans") {
		t.Fatalf("get SQL = %q, want work_item_plans select", query.calls[0].sql)
	}
	if err := store.MarkProposalSubmitted(ctx, plan.ID, testStoreTime()); err != nil {
		t.Fatalf("MarkProposalSubmitted() error = %v", err)
	}
	if !strings.Contains(exec.calls[1].sql, "UPDATE work_item_plans") {
		t.Fatalf("mark SQL = %q, want work_item_plans update", exec.calls[1].sql)
	}
}

func TestPostgresWorkItemPlanProposalStoreCreateGetAndMark(t *testing.T) {
	ctx := context.Background()
	exec := &recordingProjectContextExecer{}
	query := &recordingProjectContextQuerier{row: fakeProjectContextRow{values: validWorkItemPlanProposalRowValues()}}
	store := NewPostgresWorkItemPlanProposalStoreWithExecutorAndQuerier(exec, query)

	if err := store.Create(ctx, validPostgresWorkItemPlanProposal()); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if len(exec.calls) != 1 || !strings.Contains(exec.calls[0].sql, "INSERT INTO work_item_plan_proposals") {
		t.Fatalf("create SQL calls = %#v, want work_item_plan_proposals insert", exec.calls)
	}
	proposal, ok, err := store.Get(ctx, "018f0000-0000-7000-8000-000000000901")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !ok {
		t.Fatal("Get() ok = false, want true")
	}
	if proposal.State != spine.WorkItemProposalStateSubmitted || len(proposal.ProposedTasks) != 1 {
		t.Fatalf("proposal state/tasks = %q/%d, want submitted/1", proposal.State, len(proposal.ProposedTasks))
	}
	if !strings.Contains(query.calls[0].sql, "FROM work_item_plan_proposals") {
		t.Fatalf("get SQL = %q, want work_item_plan_proposals select", query.calls[0].sql)
	}
	if err := store.MarkAccepted(ctx, proposal.ID, spine.ActorRef{Kind: "user", ID: "acceptor"}, testStoreTime()); err != nil {
		t.Fatalf("MarkAccepted() error = %v", err)
	}
	if !strings.Contains(exec.calls[1].sql, "UPDATE work_item_plan_proposals") {
		t.Fatalf("mark SQL = %q, want work_item_plan_proposals update", exec.calls[1].sql)
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

func validPostgresClarificationRequest() spine.ClarificationRequest {
	now := testStoreTime()
	return spine.ClarificationRequest{
		ID:          "018f0000-0000-7000-8000-000000000a01",
		GoalID:      "018f0000-0000-7000-8000-000000000201",
		ReasonCodes: []spine.GoalReadinessReasonCode{spine.GoalReadinessReasonMissingScopeHint},
		Questions: []spine.ClarificationQuestion{
			{
				ID:         "018f0000-0000-7000-8000-000000000a11",
				Text:       "What is the intended scope at a high level?",
				WhyNeeded:  "A scope hint is required before contract seed readiness.",
				AnswerType: spine.ClarificationAnswerTypeText,
				MapsTo:     spine.ClarificationMapsToGoalScopeHint,
			},
		},
		Target: spine.ClarificationTarget{
			Role:     spine.ClarificationTargetRoleIntentOwner,
			ActorRef: &spine.ActorRef{Kind: "user", ID: "018f0000-0000-7000-8000-000000000001"},
		},
		State:     spine.ClarificationRequestStateOpen,
		CreatedAt: now,
	}
}

func validPostgresClarificationAnswer() spine.ClarificationAnswer {
	now := testStoreTime()
	return spine.ClarificationAnswer{
		ID:        "018f0000-0000-7000-8000-000000000b01",
		RequestID: "018f0000-0000-7000-8000-000000000a01",
		GoalID:    "018f0000-0000-7000-8000-000000000201",
		Answers: []spine.ClarificationAnswerItem{
			{QuestionID: "018f0000-0000-7000-8000-000000000a11", Value: "Persisted scope"},
		},
		SubmittedBy: spine.ActorRef{Kind: "user", ID: "018f0000-0000-7000-8000-000000000001"},
		State:       spine.ClarificationAnswerStateRecorded,
		CreatedAt:   now,
	}
}

func validPostgresContractSeed() spine.ContractSeed {
	now := testStoreTime()
	return spine.ContractSeed{
		ID:             "018f0000-0000-7000-8000-000000000401",
		OrganizationID: "018f0000-0000-7000-8000-000000000002",
		ProjectID:      "018f0000-0000-7000-8000-000000000003",
		RepoBindingID:  "018f0000-0000-7000-8000-000000000004",
		ContractID:     "018f0000-0000-7000-8000-000000000301",
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

func validPostgresContract() spine.Contract {
	now := testStoreTime()
	currentSeedID := spine.ContractSeedID("018f0000-0000-7000-8000-000000000401")
	return spine.Contract{
		ID:             "018f0000-0000-7000-8000-000000000301",
		OrganizationID: "018f0000-0000-7000-8000-000000000002",
		ProjectID:      "018f0000-0000-7000-8000-000000000003",
		RepoBindingID:  "018f0000-0000-7000-8000-000000000004",
		GoalID:         "018f0000-0000-7000-8000-000000000201",
		State:          spine.ContractStateSeeded,
		CurrentSeedID:  &currentSeedID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func validPostgresContractDraft() spine.ContractDraft {
	now := testStoreTime()
	return spine.ContractDraft{
		ID:                         "018f0000-0000-7000-8000-000000000501",
		OrganizationID:             "018f0000-0000-7000-8000-000000000002",
		ProjectID:                  "018f0000-0000-7000-8000-000000000003",
		RepoBindingID:              "018f0000-0000-7000-8000-000000000004",
		ContractID:                 "018f0000-0000-7000-8000-000000000301",
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

func validPostgresApprovedContract() spine.ApprovedContract {
	draft := validPostgresContractDraft()
	return spine.ApprovedContract{
		ID:                 "018f0000-0000-7000-8000-000000000601",
		OrganizationID:     draft.OrganizationID,
		ProjectID:          draft.ProjectID,
		RepoBindingID:      draft.RepoBindingID,
		ContractID:         draft.ContractID,
		ContractDraftID:    draft.ID,
		ContractSeedID:     draft.ContractSeedID,
		GoalID:             draft.GoalID,
		Title:              draft.Title,
		IntentSummary:      draft.IntentSummary,
		Scope:              draft.ProposedScope,
		NonGoals:           draft.ProposedNonGoals,
		Constraints:        draft.ProposedConstraints,
		AcceptanceCriteria: draft.ProposedAcceptanceCriteria,
		ExpectedChecks:     draft.ProposedExpectedChecks,
		ProofExpectations:  draft.ProposedProofExpectations,
		RiskHints:          draft.RiskHints,
		ApprovedBy:         spine.ActorRef{Kind: "user", ID: "018f0000-0000-7000-8000-000000000001"},
		ApprovedAt:         testStoreTime(),
		SourceRefs: []spine.SourceRef{
			{Kind: "contract_draft", ID: string(draft.ID)},
			{Kind: "contract_seed", ID: string(draft.ContractSeedID)},
		},
		State: spine.ApprovedContractStateApproved,
	}
}

func validPostgresWorkItem() spine.WorkItem {
	approved := validPostgresApprovedContract()
	orderIndex := 1
	return spine.WorkItem{
		ID:                   "018f0000-0000-7000-8000-000000000701",
		OrganizationID:       approved.OrganizationID,
		ProjectID:            approved.ProjectID,
		ContractID:           approved.ContractID,
		ApprovedContractID:   approved.ID,
		PlanID:               "018f0000-0000-7000-8000-000000000801",
		ProposalID:           "018f0000-0000-7000-8000-000000000901",
		RepoBindingID:        approved.RepoBindingID,
		Title:                approved.Title,
		Summary:              approved.IntentSummary,
		Scope:                approved.Scope,
		AcceptanceRefs:       []string{"acceptance_criteria[0]", "acceptance_criteria[1]"},
		ProofExpectationRefs: []string{"proof_expectations[0]", "proof_expectations[1]"},
		Status:               spine.WorkItemStatusPlanned,
		OwnerHint:            "platform",
		OrderIndex:           &orderIndex,
		SourceRefs: []spine.SourceRef{
			{Kind: "approved_contract", ID: string(approved.ID)},
		},
		CreatedAt: testStoreTime(),
	}
}

func validPostgresWorkItemPlan() spine.WorkItemPlan {
	approved := validPostgresApprovedContract()
	return spine.WorkItemPlan{
		ID:                 "018f0000-0000-7000-8000-000000000801",
		OrganizationID:     approved.OrganizationID,
		ProjectID:          approved.ProjectID,
		ContractID:         approved.ContractID,
		ApprovedContractID: approved.ID,
		RepoBindingID:      approved.RepoBindingID,
		State:              spine.WorkItemPlanStateQueued,
		RequestedBy:        spine.ActorRef{Kind: "user", ID: "018f0000-0000-7000-8000-000000000001"},
		CreatedAt:          testStoreTime(),
		UpdatedAt:          testStoreTime(),
	}
}

func validPostgresWorkItemPlanProposal() spine.WorkItemPlanProposal {
	plan := validPostgresWorkItemPlan()
	orderIndex := 0
	return spine.WorkItemPlanProposal{
		ID:                 "018f0000-0000-7000-8000-000000000901",
		PlanID:             plan.ID,
		OrganizationID:     plan.OrganizationID,
		ProjectID:          plan.ProjectID,
		ContractID:         plan.ContractID,
		ApprovedContractID: plan.ApprovedContractID,
		RepoBindingID:      plan.RepoBindingID,
		State:              spine.WorkItemProposalStateSubmitted,
		SubmittedBy:        spine.ActorRef{Kind: "worker", ID: "planner-worker-1"},
		Planner:            map[string]any{"kind": "goalrail_worker", "id": "planner-worker-1"},
		SourceSnapshotRefs: []spine.SourceRef{{Kind: "approved_contract", ID: string(plan.ApprovedContractID)}},
		Rationale:          "Split durable work item planning.",
		ProposedTasks: []spine.ProposedWorkItem{
			{
				Title:                "Persist task one",
				Summary:              "Create the first durable task.",
				Scope:                []string{"Persist task"},
				AcceptanceRefs:       []string{"acceptance_criteria[0]"},
				ProofExpectationRefs: []string{"proof_expectations[0]"},
				OrderIndex:           &orderIndex,
				SourceRefs:           []spine.SourceRef{{Kind: "approved_contract", ID: string(plan.ApprovedContractID)}},
			},
		},
		CreatedAt: testStoreTime(),
		UpdatedAt: testStoreTime(),
	}
}

func validWorkItemPlanRowValues() []any {
	plan := validPostgresWorkItemPlan()
	return []any{
		string(plan.ID),
		string(plan.OrganizationID),
		string(plan.ProjectID),
		string(plan.ContractID),
		string(plan.ApprovedContractID),
		string(plan.RepoBindingID),
		string(plan.State),
		[]byte(`{"kind":"user","id":"018f0000-0000-7000-8000-000000000001"}`),
		testStoreTime(),
		testStoreTime(),
	}
}

func validWorkItemPlanProposalRowValues() []any {
	proposal := validPostgresWorkItemPlanProposal()
	return []any{
		string(proposal.ID),
		string(proposal.PlanID),
		string(proposal.OrganizationID),
		string(proposal.ProjectID),
		string(proposal.ContractID),
		string(proposal.ApprovedContractID),
		string(proposal.RepoBindingID),
		string(proposal.State),
		[]byte(`{"kind":"worker","id":"planner-worker-1"}`),
		[]byte(`{"kind":"goalrail_worker","id":"planner-worker-1"}`),
		[]byte(`[{"kind":"approved_contract","id":"018f0000-0000-7000-8000-000000000601"}]`),
		proposal.Rationale,
		[]byte(`[{"title":"Persist task one","summary":"Create the first durable task.","scope":["Persist task"],"acceptance_refs":["acceptance_criteria[0]"],"proof_expectation_refs":["proof_expectations[0]"],"order_index":0,"source_refs":[{"kind":"approved_contract","id":"018f0000-0000-7000-8000-000000000601"}]}]`),
		[]byte(nil),
		nil,
		testStoreTime(),
		testStoreTime(),
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

func validClarificationRequestRowValues() []any {
	return []any{
		"018f0000-0000-7000-8000-000000000a01",
		"018f0000-0000-7000-8000-000000000201",
		spine.ClarificationRequestStateOpen,
		[]byte(`["missing_scope_hint"]`),
		[]byte(`[{"id":"018f0000-0000-7000-8000-000000000a11","text":"What is the intended scope at a high level?","why_needed":"A scope hint is required before contract seed readiness.","answer_type":"text","maps_to":"goal.scope_hint"}]`),
		[]byte(`{"role":"intent_owner","actor_ref":{"kind":"user","id":"018f0000-0000-7000-8000-000000000001"}}`),
		testStoreTime(),
	}
}

func validClarificationAnswerRowValues() []any {
	return []any{
		"018f0000-0000-7000-8000-000000000b01",
		"018f0000-0000-7000-8000-000000000a01",
		"018f0000-0000-7000-8000-000000000201",
		[]byte(`{"kind":"user","id":"018f0000-0000-7000-8000-000000000001"}`),
		[]byte(`[{"question_id":"018f0000-0000-7000-8000-000000000a11","value":"Persisted scope"}]`),
		testStoreTime(),
	}
}

func validContractSeedRowValues() []any {
	return []any{
		"018f0000-0000-7000-8000-000000000401",
		"018f0000-0000-7000-8000-000000000002",
		"018f0000-0000-7000-8000-000000000003",
		"018f0000-0000-7000-8000-000000000004",
		"018f0000-0000-7000-8000-000000000301",
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
		"018f0000-0000-7000-8000-000000000301",
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

func validApprovedContractRowValues() []any {
	return []any{
		"018f0000-0000-7000-8000-000000000601",
		"018f0000-0000-7000-8000-000000000002",
		"018f0000-0000-7000-8000-000000000003",
		"018f0000-0000-7000-8000-000000000004",
		"018f0000-0000-7000-8000-000000000301",
		"018f0000-0000-7000-8000-000000000501",
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
		[]byte(`{"kind":"user","id":"018f0000-0000-7000-8000-000000000001"}`),
		testStoreTime(),
		[]byte(`[{"kind":"contract_draft","id":"018f0000-0000-7000-8000-000000000501"},{"kind":"contract_seed","id":"018f0000-0000-7000-8000-000000000401"}]`),
		string(spine.ApprovedContractStateApproved),
	}
}

func validWorkItemRowValues() []any {
	return []any{
		"018f0000-0000-7000-8000-000000000701",
		"018f0000-0000-7000-8000-000000000002",
		"018f0000-0000-7000-8000-000000000003",
		"018f0000-0000-7000-8000-000000000301",
		"018f0000-0000-7000-8000-000000000601",
		"018f0000-0000-7000-8000-000000000801",
		"018f0000-0000-7000-8000-000000000901",
		"018f0000-0000-7000-8000-000000000004",
		"Persist seed",
		"Make contract seed durable",
		[]byte(`["Persist contract seeds","Preserve draft arrays"]`),
		[]byte(`["acceptance_criteria[0]","acceptance_criteria[1]"]`),
		[]byte(`["proof_expectations[0]","proof_expectations[1]"]`),
		string(spine.WorkItemStatusPlanned),
		"platform",
		int32(1),
		[]byte(`[{"kind":"approved_contract","id":"018f0000-0000-7000-8000-000000000601"}]`),
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

type uniqueViolationExecer struct {
	constraint string
}

func (e uniqueViolationExecer) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, &pgconn.PgError{Code: "23505", ConstraintName: e.constraint}
}
