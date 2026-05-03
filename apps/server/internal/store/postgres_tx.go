package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresTxFunc func(context.Context) error

type postgresDBTX interface {
	postgresExecer
	postgresRowQuerier
	postgresRowsQuerier
}

type postgresTxContextKey struct{}

func contextWithPostgresTx(ctx context.Context, tx postgresDBTX) context.Context {
	return context.WithValue(ctx, postgresTxContextKey{}, tx)
}

func postgresTxFromContext(ctx context.Context) (postgresDBTX, bool) {
	tx, ok := ctx.Value(postgresTxContextKey{}).(postgresDBTX)
	return tx, ok
}

type postgresDB struct {
	pool *pgxpool.Pool
}

func newPostgresDB(pool *pgxpool.Pool) postgresDB {
	return postgresDB{pool: pool}
}

func (db postgresDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if tx, ok := postgresTxFromContext(ctx); ok {
		return tx.Exec(ctx, sql, args...)
	}
	return db.pool.Exec(ctx, sql, args...)
}

func (db postgresDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if tx, ok := postgresTxFromContext(ctx); ok {
		return tx.Query(ctx, sql, args...)
	}
	return db.pool.Query(ctx, sql, args...)
}

func (db postgresDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if tx, ok := postgresTxFromContext(ctx); ok {
		return tx.QueryRow(ctx, sql, args...)
	}
	return db.pool.QueryRow(ctx, sql, args...)
}

type pgxpoolTransactor struct {
	pool *pgxpool.Pool
}

// PostgresTransactionRunner runs service transactions against a Postgres pool.
type PostgresTransactionRunner struct {
	transactor pgxpoolTransactor
}

// NewPostgresTransactionRunner creates a transaction runner backed by pool.
func NewPostgresTransactionRunner(pool *pgxpool.Pool) *PostgresTransactionRunner {
	return &PostgresTransactionRunner{
		transactor: pgxpoolTransactor{pool: pool},
	}
}

// RunReadCommitted runs fn in a read committed Postgres transaction.
func (r *PostgresTransactionRunner) RunReadCommitted(ctx context.Context, fn func(context.Context) error) error {
	return r.transactor.RunReadCommitted(ctx, postgresTxFunc(fn))
}

func (t pgxpoolTransactor) RunReadCommitted(ctx context.Context, fn postgresTxFunc) error {
	return t.WithTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted}, fn)
}

func (t pgxpoolTransactor) WithTx(ctx context.Context, opts pgx.TxOptions, fn postgresTxFunc) error {
	return withPostgresTx(ctx, t.pool, opts, fn)
}

func withPostgresTx(ctx context.Context, pool *pgxpool.Pool, opts pgx.TxOptions, fn postgresTxFunc) error {
	if _, ok := postgresTxFromContext(ctx); ok {
		return fn(ctx)
	}
	if pool == nil {
		return fmt.Errorf("postgres transaction pool is nil")
	}

	tx, err := pool.BeginTx(ctx, opts)
	if err != nil {
		return fmt.Errorf("begin postgres transaction: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	txCtx := contextWithPostgresTx(ctx, tx)
	if err := fn(txCtx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit postgres transaction: %w", err)
	}
	committed = true
	return nil
}
