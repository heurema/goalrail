package store

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

func TestPostgresTransactionalIntakeStoreRollsBackWhenEventAppendFails(t *testing.T) {
	ctx := context.Background()
	tx := &recordingPostgresTx{failExecCall: 2}
	db := &recordingPostgresDB{}
	transactor := &recordingPostgresTransactor{tx: tx}
	store := newPostgresTransactionalIntakeStore(
		NewPostgresIntakeStoreWithExecutorAndQuerier(db, db),
		NewPostgresEventLogWithExecutorAndQuerier(db, db),
		transactor,
	)

	err := store.CreateWithEvent(ctx, validPostgresIntakeRecord(), validPostgresEvent("intake.received", "IntakeRecord", "018f0000-0000-7000-8000-000000000101"))
	if err == nil {
		t.Fatal("CreateWithEvent() error = nil, want failure")
	}
	if got, want := len(transactor.isoLevels), 1; got != want {
		t.Fatalf("transaction count = %d, want %d", got, want)
	}
	if transactor.isoLevels[0] != pgx.ReadCommitted {
		t.Fatalf("transaction isolation = %s, want %s", transactor.isoLevels[0], pgx.ReadCommitted)
	}
	if got := len(db.fallbackExecCalls); got != 0 {
		t.Fatalf("fallback Exec calls = %d, want 0", got)
	}
	if tx.commitCalls != 0 {
		t.Fatalf("Commit calls = %d, want 0", tx.commitCalls)
	}
	if tx.rollbackCalls != 1 {
		t.Fatalf("Rollback calls = %d, want 1", tx.rollbackCalls)
	}
	if got, want := len(tx.execCalls), 2; got != want {
		t.Fatalf("Exec calls = %d, want %d", got, want)
	}
	if !strings.Contains(tx.execCalls[0].sql, "INSERT INTO intake_records") {
		t.Fatalf("first SQL = %q, want intake insert", tx.execCalls[0].sql)
	}
	if !strings.Contains(tx.execCalls[1].sql, "INSERT INTO events") {
		t.Fatalf("second SQL = %q, want event insert", tx.execCalls[1].sql)
	}
}

func TestPostgresTransactionalGoalStoreRollsBackWhenPromotionEventAppendFails(t *testing.T) {
	ctx := context.Background()
	tx := &recordingPostgresTx{failExecCall: 3}
	db := &recordingPostgresDB{}
	store := newPostgresTransactionalGoalStore(
		NewPostgresGoalStoreWithExecutorAndQuerier(db, db),
		NewPostgresEventLogWithExecutorAndQuerier(db, db),
		&recordingPostgresTransactor{tx: tx},
	)

	err := store.CreateWithEvents(ctx, validPostgresGoal(), []spine.Event{
		validPostgresEvent("goal.created", "Goal", "018f0000-0000-7000-8000-000000000201"),
		validPostgresEvent("intake.promoted_to_goal", "IntakeRecord", "018f0000-0000-7000-8000-000000000101"),
	})
	if err == nil {
		t.Fatal("CreateWithEvents() error = nil, want failure")
	}
	if tx.commitCalls != 0 {
		t.Fatalf("Commit calls = %d, want 0", tx.commitCalls)
	}
	if tx.rollbackCalls != 1 {
		t.Fatalf("Rollback calls = %d, want 1", tx.rollbackCalls)
	}
	if got := len(db.fallbackExecCalls); got != 0 {
		t.Fatalf("fallback Exec calls = %d, want 0", got)
	}
	if got, want := len(tx.execCalls), 3; got != want {
		t.Fatalf("Exec calls = %d, want %d", got, want)
	}
	if !strings.Contains(tx.execCalls[0].sql, "INSERT INTO goals") {
		t.Fatalf("first SQL = %q, want goal insert", tx.execCalls[0].sql)
	}
	if !strings.Contains(tx.execCalls[1].sql, "INSERT INTO events") || !strings.Contains(tx.execCalls[2].sql, "INSERT INTO events") {
		t.Fatalf("event SQL = %q / %q, want event inserts", tx.execCalls[1].sql, tx.execCalls[2].sql)
	}
}

func TestPostgresTransactionalGoalStoreRollsBackWhenReadinessEventAppendFails(t *testing.T) {
	ctx := context.Background()
	tx := &recordingPostgresTx{
		failExecCall: 2,
		row:          fakeProjectContextRow{values: validGoalRowValues()},
	}
	db := &recordingPostgresDB{}
	store := newPostgresTransactionalGoalStore(
		NewPostgresGoalStoreWithExecutorAndQuerier(db, db),
		NewPostgresEventLogWithExecutorAndQuerier(db, db),
		&recordingPostgresTransactor{tx: tx},
	)

	_, ok, err := store.UpdateReadinessWithEvents(
		ctx,
		"018f0000-0000-7000-8000-000000000201",
		spine.GoalStateNeedsClarification,
		[]spine.GoalReadinessReasonCode{spine.GoalReadinessReasonMissingScopeHint},
		[]spine.Event{
			validPostgresEvent("goal.readiness_checked", "Goal", "018f0000-0000-7000-8000-000000000201"),
			validPostgresEvent("goal.marked_needs_clarification", "Goal", "018f0000-0000-7000-8000-000000000201"),
		},
	)
	if err == nil {
		t.Fatal("UpdateReadinessWithEvents() error = nil, want failure")
	}
	if !ok {
		t.Fatal("UpdateReadinessWithEvents() ok = false, want true before rollback")
	}
	if tx.commitCalls != 0 {
		t.Fatalf("Commit calls = %d, want 0", tx.commitCalls)
	}
	if tx.rollbackCalls != 1 {
		t.Fatalf("Rollback calls = %d, want 1", tx.rollbackCalls)
	}
	if got := len(db.fallbackExecCalls) + len(db.fallbackQueryRowCalls); got != 0 {
		t.Fatalf("fallback DB calls = %d, want 0", got)
	}
	if got, want := len(tx.queryRowCalls), 1; got != want {
		t.Fatalf("QueryRow calls = %d, want %d", got, want)
	}
	if !strings.Contains(tx.queryRowCalls[0].sql, "UPDATE goals SET") {
		t.Fatalf("QueryRow SQL = %q, want goals update", tx.queryRowCalls[0].sql)
	}
	if got, want := len(tx.execCalls), 2; got != want {
		t.Fatalf("Exec calls = %d, want %d", got, want)
	}
}

func TestPostgresTransactionalContractSeedStoreRollsBackWhenEventAppendFails(t *testing.T) {
	ctx := context.Background()
	tx := &recordingPostgresTx{failExecCall: 2}
	db := &recordingPostgresDB{}
	store := newPostgresTransactionalContractSeedStore(
		NewPostgresContractSeedStoreWithExecutorAndQuerier(db, db),
		NewPostgresContractStoreWithExecutorAndQuerier(db, db),
		NewPostgresEventLogWithExecutorAndQuerier(db, db),
		&recordingPostgresTransactor{tx: tx},
	)

	err := store.CreateWithEvent(ctx, validPostgresContractSeed(), validPostgresEvent("contract_seed.created", "ContractSeed", "018f0000-0000-7000-8000-000000000401"))
	if err == nil {
		t.Fatal("CreateWithEvent() error = nil, want failure")
	}
	if tx.commitCalls != 0 {
		t.Fatalf("Commit calls = %d, want 0", tx.commitCalls)
	}
	if tx.rollbackCalls != 1 {
		t.Fatalf("Rollback calls = %d, want 1", tx.rollbackCalls)
	}
	if got := len(db.fallbackExecCalls); got != 0 {
		t.Fatalf("fallback Exec calls = %d, want 0", got)
	}
	if got, want := len(tx.execCalls), 2; got != want {
		t.Fatalf("Exec calls = %d, want %d", got, want)
	}
	if !strings.Contains(tx.execCalls[0].sql, "INSERT INTO contract_seeds") {
		t.Fatalf("first SQL = %q, want contract seed insert", tx.execCalls[0].sql)
	}
	if !strings.Contains(tx.execCalls[1].sql, "INSERT INTO events") {
		t.Fatalf("second SQL = %q, want event insert", tx.execCalls[1].sql)
	}
}

func TestPostgresTransactionalContractDraftStoreRollsBackWhenEventAppendFails(t *testing.T) {
	ctx := context.Background()
	tx := &recordingPostgresTx{failExecCall: 2}
	db := &recordingPostgresDB{}
	store := newPostgresTransactionalContractDraftStore(
		NewPostgresContractDraftStoreWithExecutorAndQuerier(db, db),
		NewPostgresContractStoreWithExecutorAndQuerier(db, db),
		NewPostgresEventLogWithExecutorAndQuerier(db, db),
		&recordingPostgresTransactor{tx: tx},
	)

	err := store.CreateWithEvent(ctx, validPostgresContractDraft(), validPostgresEvent("contract_draft.created", "ContractDraft", "018f0000-0000-7000-8000-000000000501"))
	if err == nil {
		t.Fatal("CreateWithEvent() error = nil, want failure")
	}
	if tx.commitCalls != 0 {
		t.Fatalf("Commit calls = %d, want 0", tx.commitCalls)
	}
	if tx.rollbackCalls != 1 {
		t.Fatalf("Rollback calls = %d, want 1", tx.rollbackCalls)
	}
	if got := len(db.fallbackExecCalls); got != 0 {
		t.Fatalf("fallback Exec calls = %d, want 0", got)
	}
	if got, want := len(tx.execCalls), 2; got != want {
		t.Fatalf("Exec calls = %d, want %d", got, want)
	}
	if !strings.Contains(tx.execCalls[0].sql, "INSERT INTO contract_drafts") {
		t.Fatalf("first SQL = %q, want contract draft insert", tx.execCalls[0].sql)
	}
	if !strings.Contains(tx.execCalls[1].sql, "INSERT INTO events") {
		t.Fatalf("second SQL = %q, want event insert", tx.execCalls[1].sql)
	}
}

func TestPostgresTransactionalContractDraftStoreRollsBackUpdateWhenEventAppendFails(t *testing.T) {
	ctx := context.Background()
	tx := &recordingPostgresTx{failExecCall: 2}
	db := &recordingPostgresDB{}
	store := newPostgresTransactionalContractDraftStore(
		NewPostgresContractDraftStoreWithExecutorAndQuerier(db, db),
		NewPostgresContractStoreWithExecutorAndQuerier(db, db),
		NewPostgresEventLogWithExecutorAndQuerier(db, db),
		&recordingPostgresTransactor{tx: tx},
	)

	err := store.UpdateWithEvent(ctx, validPostgresContractDraft(), validPostgresEvent("contract_draft.updated", "ContractDraft", "018f0000-0000-7000-8000-000000000501"))
	if err == nil {
		t.Fatal("UpdateWithEvent() error = nil, want failure")
	}
	if tx.commitCalls != 0 {
		t.Fatalf("Commit calls = %d, want 0", tx.commitCalls)
	}
	if tx.rollbackCalls != 1 {
		t.Fatalf("Rollback calls = %d, want 1", tx.rollbackCalls)
	}
	if got := len(db.fallbackExecCalls); got != 0 {
		t.Fatalf("fallback Exec calls = %d, want 0", got)
	}
	if got, want := len(tx.execCalls), 2; got != want {
		t.Fatalf("Exec calls = %d, want %d", got, want)
	}
	if !strings.Contains(tx.execCalls[0].sql, "UPDATE contract_drafts") {
		t.Fatalf("first SQL = %q, want contract draft update", tx.execCalls[0].sql)
	}
	if !strings.Contains(tx.execCalls[1].sql, "INSERT INTO events") {
		t.Fatalf("second SQL = %q, want event insert", tx.execCalls[1].sql)
	}
}

func TestPostgresTransactionalContractDraftStoreRollsBackMarkReadyForApprovalWhenEventAppendFails(t *testing.T) {
	ctx := context.Background()
	tx := &recordingPostgresTx{failExecCall: 2}
	db := &recordingPostgresDB{}
	store := newPostgresTransactionalContractDraftStore(
		NewPostgresContractDraftStoreWithExecutorAndQuerier(db, db),
		NewPostgresContractStoreWithExecutorAndQuerier(db, db),
		NewPostgresEventLogWithExecutorAndQuerier(db, db),
		&recordingPostgresTransactor{tx: tx},
	)

	draft := validPostgresContractDraft()
	draft.State = spine.ContractDraftStateReadyForApproval
	err := store.MarkReadyForApprovalWithEvent(ctx, draft, validPostgresEvent("contract_draft.marked_ready_for_approval", "ContractDraft", "018f0000-0000-7000-8000-000000000501"))
	if err == nil {
		t.Fatal("MarkReadyForApprovalWithEvent() error = nil, want failure")
	}
	if tx.commitCalls != 0 {
		t.Fatalf("Commit calls = %d, want 0", tx.commitCalls)
	}
	if tx.rollbackCalls != 1 {
		t.Fatalf("Rollback calls = %d, want 1", tx.rollbackCalls)
	}
	if got := len(db.fallbackExecCalls); got != 0 {
		t.Fatalf("fallback Exec calls = %d, want 0", got)
	}
	if got, want := len(tx.execCalls), 2; got != want {
		t.Fatalf("Exec calls = %d, want %d", got, want)
	}
	if !strings.Contains(tx.execCalls[0].sql, "UPDATE contract_drafts") {
		t.Fatalf("first SQL = %q, want contract draft update", tx.execCalls[0].sql)
	}
	if !strings.Contains(tx.execCalls[0].sql, "state =") {
		t.Fatalf("first SQL = %q, want state update", tx.execCalls[0].sql)
	}
	if !strings.Contains(tx.execCalls[1].sql, "INSERT INTO events") {
		t.Fatalf("second SQL = %q, want event insert", tx.execCalls[1].sql)
	}
}

func TestPostgresTransactionalApprovedContractStoreRollsBackWhenEventAppendFails(t *testing.T) {
	ctx := context.Background()
	tx := &recordingPostgresTx{failExecCall: 2}
	db := &recordingPostgresDB{}
	store := newPostgresTransactionalApprovedContractStore(
		NewPostgresApprovedContractStoreWithExecutorAndQuerier(db, db),
		NewPostgresContractStoreWithExecutorAndQuerier(db, db),
		NewPostgresEventLogWithExecutorAndQuerier(db, db),
		&recordingPostgresTransactor{tx: tx},
	)

	err := store.CreateWithEvent(ctx, validPostgresApprovedContract(), validPostgresEvent("contract.approved", "ApprovedContract", "018f0000-0000-7000-8000-000000000601"))
	if err == nil {
		t.Fatal("CreateWithEvent() error = nil, want failure")
	}
	if tx.commitCalls != 0 {
		t.Fatalf("Commit calls = %d, want 0", tx.commitCalls)
	}
	if tx.rollbackCalls != 1 {
		t.Fatalf("Rollback calls = %d, want 1", tx.rollbackCalls)
	}
	if got := len(db.fallbackExecCalls); got != 0 {
		t.Fatalf("fallback Exec calls = %d, want 0", got)
	}
	if got, want := len(tx.execCalls), 2; got != want {
		t.Fatalf("Exec calls = %d, want %d", got, want)
	}
	if !strings.Contains(tx.execCalls[0].sql, "INSERT INTO approved_contracts") {
		t.Fatalf("first SQL = %q, want approved contract insert", tx.execCalls[0].sql)
	}
	if !strings.Contains(tx.execCalls[1].sql, "INSERT INTO events") {
		t.Fatalf("second SQL = %q, want event insert", tx.execCalls[1].sql)
	}
}

func TestPostgresTransactionalWorkItemStoreRollsBackWhenEventAppendFails(t *testing.T) {
	ctx := context.Background()
	tx := &recordingPostgresTx{failExecCall: 2}
	db := &recordingPostgresDB{}
	store := newPostgresTransactionalWorkItemStore(
		NewPostgresWorkItemStoreWithExecutorAndQuerier(db, db),
		NewPostgresEventLogWithExecutorAndQuerier(db, db),
		&recordingPostgresTransactor{tx: tx},
	)

	err := store.CreateWithEvent(ctx, validPostgresWorkItem(), validPostgresEvent("work_item.created", "WorkItem", "018f0000-0000-7000-8000-000000000701"))
	if err == nil {
		t.Fatal("CreateWithEvent() error = nil, want failure")
	}
	if tx.commitCalls != 0 {
		t.Fatalf("Commit calls = %d, want 0", tx.commitCalls)
	}
	if tx.rollbackCalls != 1 {
		t.Fatalf("Rollback calls = %d, want 1", tx.rollbackCalls)
	}
	if got := len(db.fallbackExecCalls); got != 0 {
		t.Fatalf("fallback Exec calls = %d, want 0", got)
	}
	if got, want := len(tx.execCalls), 2; got != want {
		t.Fatalf("Exec calls = %d, want %d", got, want)
	}
	if !strings.Contains(tx.execCalls[0].sql, "INSERT INTO work_items") {
		t.Fatalf("first SQL = %q, want work item insert", tx.execCalls[0].sql)
	}
	if !strings.Contains(tx.execCalls[1].sql, "INSERT INTO events") {
		t.Fatalf("second SQL = %q, want event insert", tx.execCalls[1].sql)
	}
}

func TestPostgresTransactionalClarificationStoreRollsBackRequestCreationWhenEventAppendFails(t *testing.T) {
	ctx := context.Background()
	tx := &recordingPostgresTx{failExecCall: 2}
	db := &recordingPostgresDB{}
	transactor := &recordingPostgresTransactor{tx: tx}
	store := newPostgresTransactionalClarificationStore(
		NewPostgresClarificationRequestStoreWithExecutorAndQuerier(db, db),
		NewPostgresClarificationAnswerStoreWithExecutorAndQuerier(db, db),
		NewPostgresGoalStoreWithExecutorAndQuerier(db, db),
		NewPostgresEventLogWithExecutorAndQuerier(db, db),
		transactor,
	)

	err := store.CreateRequestWithEvent(ctx, validPostgresClarificationRequest(), validPostgresEvent("clarification.requested", "ClarificationRequest", "018f0000-0000-7000-8000-000000000a01"))
	if err == nil {
		t.Fatal("CreateRequestWithEvent() error = nil, want failure")
	}
	if got, want := len(transactor.isoLevels), 1; got != want {
		t.Fatalf("transaction count = %d, want %d", got, want)
	}
	if transactor.isoLevels[0] != pgx.ReadCommitted {
		t.Fatalf("transaction isolation = %s, want %s", transactor.isoLevels[0], pgx.ReadCommitted)
	}
	if tx.commitCalls != 0 {
		t.Fatalf("Commit calls = %d, want 0", tx.commitCalls)
	}
	if tx.rollbackCalls != 1 {
		t.Fatalf("Rollback calls = %d, want 1", tx.rollbackCalls)
	}
	if got := len(db.fallbackExecCalls) + len(db.fallbackQueryRowCalls); got != 0 {
		t.Fatalf("fallback DB calls = %d, want 0", got)
	}
	if got, want := len(tx.execCalls), 2; got != want {
		t.Fatalf("Exec calls = %d, want %d", got, want)
	}
	if !strings.Contains(tx.execCalls[0].sql, "INSERT INTO clarification_requests") {
		t.Fatalf("first SQL = %q, want clarification request insert", tx.execCalls[0].sql)
	}
	if !strings.Contains(tx.execCalls[1].sql, "INSERT INTO events") {
		t.Fatalf("second SQL = %q, want event insert", tx.execCalls[1].sql)
	}
}

func TestPostgresTransactionalClarificationStoreRollsBackAnswerRecordingWhenEventAppendFails(t *testing.T) {
	ctx := context.Background()
	values := validClarificationRequestRowValues()
	values[2] = spine.ClarificationRequestStateAnswered
	tx := &recordingPostgresTx{
		failExecCall: 2,
		row:          fakeProjectContextRow{values: values},
	}
	db := &recordingPostgresDB{}
	store := newPostgresTransactionalClarificationStore(
		NewPostgresClarificationRequestStoreWithExecutorAndQuerier(db, db),
		NewPostgresClarificationAnswerStoreWithExecutorAndQuerier(db, db),
		NewPostgresGoalStoreWithExecutorAndQuerier(db, db),
		NewPostgresEventLogWithExecutorAndQuerier(db, db),
		&recordingPostgresTransactor{tx: tx},
	)

	_, err := store.RecordAnswerWithEvents(ctx, validPostgresClarificationAnswer(), []spine.Event{
		validPostgresEvent("clarification.answer_recorded", "ClarificationAnswer", "018f0000-0000-7000-8000-000000000b01"),
		validPostgresEvent("clarification.request_answered", "ClarificationRequest", "018f0000-0000-7000-8000-000000000a01"),
	})
	if err == nil {
		t.Fatal("RecordAnswerWithEvents() error = nil, want failure")
	}
	if tx.commitCalls != 0 {
		t.Fatalf("Commit calls = %d, want 0", tx.commitCalls)
	}
	if tx.rollbackCalls != 1 {
		t.Fatalf("Rollback calls = %d, want 1", tx.rollbackCalls)
	}
	if got := len(db.fallbackExecCalls) + len(db.fallbackQueryRowCalls); got != 0 {
		t.Fatalf("fallback DB calls = %d, want 0", got)
	}
	if got, want := len(tx.execCalls), 2; got != want {
		t.Fatalf("Exec calls = %d, want %d", got, want)
	}
	if got, want := len(tx.queryRowCalls), 1; got != want {
		t.Fatalf("QueryRow calls = %d, want %d", got, want)
	}
	if !strings.Contains(tx.execCalls[0].sql, "INSERT INTO clarification_answers") {
		t.Fatalf("first SQL = %q, want clarification answer insert", tx.execCalls[0].sql)
	}
	if !strings.Contains(tx.queryRowCalls[0].sql, "UPDATE clarification_requests") {
		t.Fatalf("QueryRow SQL = %q, want clarification request update", tx.queryRowCalls[0].sql)
	}
	if !strings.Contains(tx.execCalls[1].sql, "INSERT INTO events") {
		t.Fatalf("second SQL = %q, want event insert", tx.execCalls[1].sql)
	}
}

func TestPostgresTransactionalClarificationStoreReturnsFalseWhenRequestAnsweredUpdateFindsNoRow(t *testing.T) {
	ctx := context.Background()
	tx := &recordingPostgresTx{
		row: fakeProjectContextRow{err: pgx.ErrNoRows},
	}
	db := &recordingPostgresDB{}
	store := newPostgresTransactionalClarificationStore(
		NewPostgresClarificationRequestStoreWithExecutorAndQuerier(db, db),
		NewPostgresClarificationAnswerStoreWithExecutorAndQuerier(db, db),
		NewPostgresGoalStoreWithExecutorAndQuerier(db, db),
		NewPostgresEventLogWithExecutorAndQuerier(db, db),
		&recordingPostgresTransactor{tx: tx},
	)

	ok, err := store.RecordAnswerWithEvents(ctx, validPostgresClarificationAnswer(), []spine.Event{
		validPostgresEvent("clarification.answer_recorded", "ClarificationAnswer", "018f0000-0000-7000-8000-000000000b01"),
	})
	if err != nil {
		t.Fatalf("RecordAnswerWithEvents() error = %v, want nil", err)
	}
	if ok {
		t.Fatal("RecordAnswerWithEvents() ok = true, want false")
	}
	if tx.commitCalls != 0 {
		t.Fatalf("Commit calls = %d, want 0", tx.commitCalls)
	}
	if tx.rollbackCalls != 1 {
		t.Fatalf("Rollback calls = %d, want 1", tx.rollbackCalls)
	}
	if got, want := len(tx.execCalls), 1; got != want {
		t.Fatalf("Exec calls = %d, want %d", got, want)
	}
	if got, want := len(tx.queryRowCalls), 1; got != want {
		t.Fatalf("QueryRow calls = %d, want %d", got, want)
	}
}

func TestPostgresTransactionalClarificationStoreRollsBackApplicationWhenEventAppendFails(t *testing.T) {
	ctx := context.Background()
	tx := &recordingPostgresTx{
		failExecCall: 2,
		row:          fakeProjectContextRow{values: validGoalRowValues()},
	}
	db := &recordingPostgresDB{}
	store := newPostgresTransactionalClarificationStore(
		NewPostgresClarificationRequestStoreWithExecutorAndQuerier(db, db),
		NewPostgresClarificationAnswerStoreWithExecutorAndQuerier(db, db),
		NewPostgresGoalStoreWithExecutorAndQuerier(db, db),
		NewPostgresEventLogWithExecutorAndQuerier(db, db),
		&recordingPostgresTransactor{tx: tx},
	)

	_, marked, goalOK, err := store.ApplyAnswerWithGoalHintsAndEvents(
		ctx,
		"018f0000-0000-7000-8000-000000000b01",
		"018f0000-0000-7000-8000-000000000201",
		spine.GoalHintUpdate{ScopeHint: stringPtrForStoreTest("Updated scope")},
		spine.ActorRef{Kind: "user", ID: "018f0000-0000-7000-8000-000000000001"},
		testStoreTime(),
		[]spine.Event{
			validPostgresEvent("clarification.answer_applied_to_goal", "ClarificationAnswer", "018f0000-0000-7000-8000-000000000b01"),
			validPostgresEvent("goal.hints_updated", "Goal", "018f0000-0000-7000-8000-000000000201"),
		},
	)
	if err == nil {
		t.Fatal("ApplyAnswerWithGoalHintsAndEvents() error = nil, want failure")
	}
	if !marked {
		t.Fatal("marked = false, want true before rollback")
	}
	if !goalOK {
		t.Fatal("goalOK = false, want true before rollback")
	}
	if tx.commitCalls != 0 {
		t.Fatalf("Commit calls = %d, want 0", tx.commitCalls)
	}
	if tx.rollbackCalls != 1 {
		t.Fatalf("Rollback calls = %d, want 1", tx.rollbackCalls)
	}
	if got, want := len(tx.execCalls), 2; got != want {
		t.Fatalf("Exec calls = %d, want %d", got, want)
	}
	if got, want := len(tx.queryRowCalls), 1; got != want {
		t.Fatalf("QueryRow calls = %d, want %d", got, want)
	}
	if !strings.Contains(tx.execCalls[0].sql, "UPDATE clarification_answers") {
		t.Fatalf("first SQL = %q, want clarification answer update", tx.execCalls[0].sql)
	}
	if !strings.Contains(tx.queryRowCalls[0].sql, "UPDATE goals SET") {
		t.Fatalf("QueryRow SQL = %q, want goal hints update", tx.queryRowCalls[0].sql)
	}
	if !strings.Contains(tx.execCalls[1].sql, "INSERT INTO events") {
		t.Fatalf("second SQL = %q, want event insert", tx.execCalls[1].sql)
	}
}

func TestPostgresTransactionalClarificationStoreRollsBackApplicationWhenGoalUpdateFails(t *testing.T) {
	ctx := context.Background()
	tx := &recordingPostgresTx{
		row: fakeProjectContextRow{err: pgx.ErrNoRows},
	}
	db := &recordingPostgresDB{}
	store := newPostgresTransactionalClarificationStore(
		NewPostgresClarificationRequestStoreWithExecutorAndQuerier(db, db),
		NewPostgresClarificationAnswerStoreWithExecutorAndQuerier(db, db),
		NewPostgresGoalStoreWithExecutorAndQuerier(db, db),
		NewPostgresEventLogWithExecutorAndQuerier(db, db),
		&recordingPostgresTransactor{tx: tx},
	)

	_, marked, goalOK, err := store.ApplyAnswerWithGoalHintsAndEvents(
		ctx,
		"018f0000-0000-7000-8000-000000000b01",
		"018f0000-0000-7000-8000-000000000201",
		spine.GoalHintUpdate{ScopeHint: stringPtrForStoreTest("Updated scope")},
		spine.ActorRef{Kind: "user", ID: "018f0000-0000-7000-8000-000000000001"},
		testStoreTime(),
		[]spine.Event{validPostgresEvent("goal.hints_updated", "Goal", "018f0000-0000-7000-8000-000000000201")},
	)
	if err != nil {
		t.Fatalf("ApplyAnswerWithGoalHintsAndEvents() error = %v, want nil with goalOK=false", err)
	}
	if !marked {
		t.Fatal("marked = false, want true before rollback")
	}
	if goalOK {
		t.Fatal("goalOK = true, want false")
	}
	if tx.commitCalls != 0 {
		t.Fatalf("Commit calls = %d, want 0", tx.commitCalls)
	}
	if tx.rollbackCalls != 1 {
		t.Fatalf("Rollback calls = %d, want 1", tx.rollbackCalls)
	}
	if got, want := len(tx.execCalls), 1; got != want {
		t.Fatalf("Exec calls = %d, want %d", got, want)
	}
	if got, want := len(tx.queryRowCalls), 1; got != want {
		t.Fatalf("QueryRow calls = %d, want %d", got, want)
	}
}

func TestPostgresTransactionalGoalStoreCommitsPromotionWithEvents(t *testing.T) {
	ctx := context.Background()
	tx := &recordingPostgresTx{}
	db := &recordingPostgresDB{}
	store := newPostgresTransactionalGoalStore(
		NewPostgresGoalStoreWithExecutorAndQuerier(db, db),
		NewPostgresEventLogWithExecutorAndQuerier(db, db),
		&recordingPostgresTransactor{tx: tx},
	)

	if err := store.CreateWithEvents(ctx, validPostgresGoal(), []spine.Event{
		validPostgresEvent("goal.created", "Goal", "018f0000-0000-7000-8000-000000000201"),
		validPostgresEvent("intake.promoted_to_goal", "IntakeRecord", "018f0000-0000-7000-8000-000000000101"),
	}); err != nil {
		t.Fatalf("CreateWithEvents() error = %v", err)
	}
	if tx.commitCalls != 1 {
		t.Fatalf("Commit calls = %d, want 1", tx.commitCalls)
	}
	if tx.rollbackCalls != 0 {
		t.Fatalf("Rollback calls = %d, want 0", tx.rollbackCalls)
	}
	if got := len(db.fallbackExecCalls); got != 0 {
		t.Fatalf("fallback Exec calls = %d, want 0", got)
	}
}

func TestWithPostgresTxReusesExistingTransactionFromContext(t *testing.T) {
	ctx := context.Background()
	tx := &recordingPostgresTx{}
	txCtx := contextWithPostgresTx(ctx, tx)

	called := false
	err := withPostgresTx(txCtx, nil, pgx.TxOptions{IsoLevel: pgx.Serializable}, func(ctx context.Context) error {
		called = true
		fromContext, ok := postgresTxFromContext(ctx)
		if !ok {
			t.Fatal("postgres tx missing from context")
		}
		if fromContext != tx {
			t.Fatal("postgres tx from context was replaced")
		}
		_, err := fromContext.Exec(ctx, "select 1")
		return err
	})
	if err != nil {
		t.Fatalf("withPostgresTx() error = %v", err)
	}
	if !called {
		t.Fatal("transaction callback was not called")
	}
	if tx.commitCalls != 0 {
		t.Fatalf("Commit calls = %d, want 0", tx.commitCalls)
	}
	if tx.rollbackCalls != 0 {
		t.Fatalf("Rollback calls = %d, want 0", tx.rollbackCalls)
	}
	if got, want := len(tx.execCalls), 1; got != want {
		t.Fatalf("Exec calls = %d, want %d", got, want)
	}
}

func validPostgresEvent(eventType string, entityType string, entityID string) spine.Event {
	return spine.Event{
		ID:             "018f0000-0000-7000-8000-000000000301",
		Type:           eventType,
		OrganizationID: "018f0000-0000-7000-8000-000000000002",
		ProjectID:      "018f0000-0000-7000-8000-000000000003",
		RepoBindingID:  "018f0000-0000-7000-8000-000000000004",
		EntityType:     entityType,
		EntityID:       entityID,
		Timestamp:      testStoreTime(),
		Payload:        []byte(`{"id":"` + entityID + `"}`),
	}
}

func stringPtrForStoreTest(value string) *string {
	return &value
}

type recordingPostgresTransactor struct {
	tx         *recordingPostgresTx
	beginCalls int
	beginErr   error
	isoLevels  []pgx.TxIsoLevel
}

func (t *recordingPostgresTransactor) RunReadCommitted(ctx context.Context, fn postgresTxFunc) error {
	return t.WithTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted}, fn)
}

func (t *recordingPostgresTransactor) WithTx(ctx context.Context, opts pgx.TxOptions, fn postgresTxFunc) error {
	if _, ok := postgresTxFromContext(ctx); ok {
		return fn(ctx)
	}
	t.beginCalls++
	t.isoLevels = append(t.isoLevels, opts.IsoLevel)
	if t.beginErr != nil {
		return t.beginErr
	}

	committed := false
	defer func() {
		if !committed {
			_ = t.tx.Rollback(ctx)
		}
	}()

	if err := fn(contextWithPostgresTx(ctx, t.tx)); err != nil {
		return err
	}
	if err := t.tx.Commit(ctx); err != nil {
		return err
	}
	committed = true
	return nil
}

type recordingPostgresDB struct {
	fallbackExecCalls     []recordedExecCall
	fallbackQueryRowCalls []recordedExecCall
}

func (db *recordingPostgresDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if tx, ok := postgresTxFromContext(ctx); ok {
		return tx.Exec(ctx, sql, args...)
	}
	db.fallbackExecCalls = append(db.fallbackExecCalls, recordedExecCall{
		sql:  sql,
		args: append([]any(nil), args...),
	})
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}

func (db *recordingPostgresDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if tx, ok := postgresTxFromContext(ctx); ok {
		return tx.Query(ctx, sql, args...)
	}
	return nil, errors.New("unexpected fallback query")
}

func (db *recordingPostgresDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if tx, ok := postgresTxFromContext(ctx); ok {
		return tx.QueryRow(ctx, sql, args...)
	}
	db.fallbackQueryRowCalls = append(db.fallbackQueryRowCalls, recordedExecCall{
		sql:  sql,
		args: append([]any(nil), args...),
	})
	return fakeProjectContextRow{err: pgx.ErrNoRows}
}

type recordingPostgresTx struct {
	execCalls     []recordedExecCall
	queryRowCalls []recordedExecCall
	failExecCall  int
	row           pgx.Row
	commitCalls   int
	rollbackCalls int
}

func (tx *recordingPostgresTx) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	tx.execCalls = append(tx.execCalls, recordedExecCall{
		sql:  sql,
		args: append([]any(nil), args...),
	})
	if tx.failExecCall == len(tx.execCalls) {
		return pgconn.CommandTag{}, errors.New("forced exec failure")
	}
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}

func (tx *recordingPostgresTx) Query(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
	return nil, errors.New("unexpected query")
}

func (tx *recordingPostgresTx) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	tx.queryRowCalls = append(tx.queryRowCalls, recordedExecCall{
		sql:  sql,
		args: append([]any(nil), args...),
	})
	if tx.row != nil {
		return tx.row
	}
	return fakeProjectContextRow{err: pgx.ErrNoRows}
}

func (tx *recordingPostgresTx) Commit(context.Context) error {
	tx.commitCalls++
	return nil
}

func (tx *recordingPostgresTx) Rollback(context.Context) error {
	tx.rollbackCalls++
	return nil
}
