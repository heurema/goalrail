package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

type postgresDBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type postgresTx interface {
	postgresDBTX
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

type postgresTransactor interface {
	BeginTx(ctx context.Context) (postgresTx, error)
}

type pgxpoolTransactor struct {
	pool *pgxpool.Pool
}

func (t pgxpoolTransactor) BeginTx(ctx context.Context) (postgresTx, error) {
	return t.pool.Begin(ctx)
}

type PostgresTransactionalIntakeStore struct {
	base       *PostgresIntakeStore
	transactor postgresTransactor
}

func NewPostgresTransactionalIntakeStore(pool *pgxpool.Pool) *PostgresTransactionalIntakeStore {
	return newPostgresTransactionalIntakeStore(NewPostgresIntakeStore(pool), pgxpoolTransactor{pool: pool})
}

func newPostgresTransactionalIntakeStore(base *PostgresIntakeStore, transactor postgresTransactor) *PostgresTransactionalIntakeStore {
	return &PostgresTransactionalIntakeStore{
		base:       base,
		transactor: transactor,
	}
}

func (s *PostgresTransactionalIntakeStore) Create(ctx context.Context, record spine.IntakeRecord) error {
	return s.base.Create(ctx, record)
}

func (s *PostgresTransactionalIntakeStore) Get(ctx context.Context, id spine.IntakeID) (spine.IntakeRecord, bool, error) {
	return s.base.Get(ctx, id)
}

func (s *PostgresTransactionalIntakeStore) CreateWithEvent(ctx context.Context, record spine.IntakeRecord, event spine.Event) error {
	return withPostgresTx(ctx, s.transactor, func(tx postgresTx) error {
		intakes := NewPostgresIntakeStoreWithExecutorAndQuerier(tx, tx)
		events := NewPostgresEventLogWithExecutorAndQuerier(tx, tx)
		if err := intakes.Create(ctx, record); err != nil {
			return err
		}
		if err := events.Append(ctx, event); err != nil {
			return err
		}
		return nil
	})
}

type PostgresTransactionalGoalStore struct {
	base       *PostgresGoalStore
	transactor postgresTransactor
}

func NewPostgresTransactionalGoalStore(pool *pgxpool.Pool) *PostgresTransactionalGoalStore {
	return newPostgresTransactionalGoalStore(NewPostgresGoalStore(pool), pgxpoolTransactor{pool: pool})
}

func newPostgresTransactionalGoalStore(base *PostgresGoalStore, transactor postgresTransactor) *PostgresTransactionalGoalStore {
	return &PostgresTransactionalGoalStore{
		base:       base,
		transactor: transactor,
	}
}

func (s *PostgresTransactionalGoalStore) Create(ctx context.Context, created spine.Goal) error {
	return s.base.Create(ctx, created)
}

func (s *PostgresTransactionalGoalStore) Get(ctx context.Context, id spine.GoalID) (spine.Goal, bool, error) {
	return s.base.Get(ctx, id)
}

func (s *PostgresTransactionalGoalStore) GetByIntakeID(ctx context.Context, id spine.IntakeID) (spine.Goal, bool, error) {
	return s.base.GetByIntakeID(ctx, id)
}

func (s *PostgresTransactionalGoalStore) UpdateReadiness(ctx context.Context, id spine.GoalID, state spine.GoalState, reasonCodes []spine.GoalReadinessReasonCode) (spine.Goal, bool, error) {
	return s.base.UpdateReadiness(ctx, id, state, reasonCodes)
}

func (s *PostgresTransactionalGoalStore) UpdateHints(ctx context.Context, id spine.GoalID, update spine.GoalHintUpdate) (spine.Goal, bool, error) {
	return s.base.UpdateHints(ctx, id, update)
}

func (s *PostgresTransactionalGoalStore) CreateWithEvents(ctx context.Context, created spine.Goal, eventsToAppend []spine.Event) error {
	return withPostgresTx(ctx, s.transactor, func(tx postgresTx) error {
		goals := NewPostgresGoalStoreWithExecutorAndQuerier(tx, tx)
		events := NewPostgresEventLogWithExecutorAndQuerier(tx, tx)
		if err := goals.Create(ctx, created); err != nil {
			return err
		}
		for _, event := range eventsToAppend {
			if err := events.Append(ctx, event); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *PostgresTransactionalGoalStore) UpdateReadinessWithEvents(ctx context.Context, id spine.GoalID, state spine.GoalState, reasonCodes []spine.GoalReadinessReasonCode, eventsToAppend []spine.Event) (spine.Goal, bool, error) {
	var updated spine.Goal
	var ok bool
	err := withPostgresTx(ctx, s.transactor, func(tx postgresTx) error {
		goals := NewPostgresGoalStoreWithExecutorAndQuerier(tx, tx)
		events := NewPostgresEventLogWithExecutorAndQuerier(tx, tx)
		var err error
		updated, ok, err = goals.UpdateReadiness(ctx, id, state, reasonCodes)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		for _, event := range eventsToAppend {
			if err := events.Append(ctx, event); err != nil {
				return err
			}
		}
		return nil
	})
	return updated, ok, err
}

func withPostgresTx(ctx context.Context, transactor postgresTransactor, fn func(postgresTx) error) error {
	if transactor == nil {
		return fmt.Errorf("postgres transactor is nil")
	}
	tx, err := transactor.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin postgres transaction: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit postgres transaction: %w", err)
	}
	committed = true
	return nil
}
