package store

import (
	"context"
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

func testStoreTime() time.Time {
	return time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
}
