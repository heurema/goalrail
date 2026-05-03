package store

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

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

func TestWithPostgresTxNilPoolWithoutExistingTransaction(t *testing.T) {
	err := withPostgresTx(context.Background(), nil, pgx.TxOptions{IsoLevel: pgx.ReadCommitted}, func(context.Context) error {
		t.Fatal("transaction callback should not be called")
		return nil
	})
	if err == nil {
		t.Fatal("withPostgresTx() error = nil, want nil pool error")
	}
	if got, want := err.Error(), "postgres transaction pool is nil"; got != want {
		t.Fatalf("withPostgresTx() error = %q, want %q", got, want)
	}
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
