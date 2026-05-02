package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

type postgresTxFunc func(context.Context) error

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
	ExecReadCommitted(ctx context.Context, fn postgresTxFunc) error
	WithTx(ctx context.Context, opts pgx.TxOptions, fn postgresTxFunc) error
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

func (t pgxpoolTransactor) ExecReadCommitted(ctx context.Context, fn postgresTxFunc) error {
	return t.WithTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted}, fn)
}

func (t pgxpoolTransactor) WithTx(ctx context.Context, opts pgx.TxOptions, fn postgresTxFunc) error {
	return withPostgresTx(ctx, t.pool, opts, fn)
}

type PostgresTransactionalIntakeStore struct {
	base       *PostgresIntakeStore
	events     *PostgresEventLog
	transactor postgresTransactor
}

func NewPostgresTransactionalIntakeStore(pool *pgxpool.Pool) *PostgresTransactionalIntakeStore {
	db := newPostgresDB(pool)
	return newPostgresTransactionalIntakeStore(
		NewPostgresIntakeStoreWithExecutorAndQuerier(db, db),
		NewPostgresEventLogWithExecutorAndQuerier(db, db),
		pgxpoolTransactor{pool: pool},
	)
}

func newPostgresTransactionalIntakeStore(base *PostgresIntakeStore, events *PostgresEventLog, transactor postgresTransactor) *PostgresTransactionalIntakeStore {
	return &PostgresTransactionalIntakeStore{
		base:       base,
		events:     events,
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
	if s.transactor == nil {
		return fmt.Errorf("postgres transactor is nil")
	}
	return s.transactor.ExecReadCommitted(ctx, func(txCtx context.Context) error {
		if err := s.base.Create(txCtx, record); err != nil {
			return err
		}
		if err := s.events.Append(txCtx, event); err != nil {
			return err
		}
		return nil
	})
}

type PostgresTransactionalGoalStore struct {
	base       *PostgresGoalStore
	events     *PostgresEventLog
	transactor postgresTransactor
}

func NewPostgresTransactionalGoalStore(pool *pgxpool.Pool) *PostgresTransactionalGoalStore {
	db := newPostgresDB(pool)
	return newPostgresTransactionalGoalStore(
		NewPostgresGoalStoreWithExecutorAndQuerier(db, db),
		NewPostgresEventLogWithExecutorAndQuerier(db, db),
		pgxpoolTransactor{pool: pool},
	)
}

func newPostgresTransactionalGoalStore(base *PostgresGoalStore, events *PostgresEventLog, transactor postgresTransactor) *PostgresTransactionalGoalStore {
	return &PostgresTransactionalGoalStore{
		base:       base,
		events:     events,
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
	if s.transactor == nil {
		return fmt.Errorf("postgres transactor is nil")
	}
	return s.transactor.ExecReadCommitted(ctx, func(txCtx context.Context) error {
		if err := s.base.Create(txCtx, created); err != nil {
			return err
		}
		for _, event := range eventsToAppend {
			if err := s.events.Append(txCtx, event); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *PostgresTransactionalGoalStore) UpdateReadinessWithEvents(ctx context.Context, id spine.GoalID, state spine.GoalState, reasonCodes []spine.GoalReadinessReasonCode, eventsToAppend []spine.Event) (spine.Goal, bool, error) {
	if s.transactor == nil {
		return spine.Goal{}, false, fmt.Errorf("postgres transactor is nil")
	}
	var updated spine.Goal
	var ok bool
	err := s.transactor.ExecReadCommitted(ctx, func(txCtx context.Context) error {
		var err error
		updated, ok, err = s.base.UpdateReadiness(txCtx, id, state, reasonCodes)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		for _, event := range eventsToAppend {
			if err := s.events.Append(txCtx, event); err != nil {
				return err
			}
		}
		return nil
	})
	return updated, ok, err
}

type PostgresTransactionalContractSeedStore struct {
	base       *PostgresContractSeedStore
	contracts  *PostgresContractStore
	events     *PostgresEventLog
	transactor postgresTransactor
}

func NewPostgresTransactionalContractSeedStore(pool *pgxpool.Pool) *PostgresTransactionalContractSeedStore {
	db := newPostgresDB(pool)
	return newPostgresTransactionalContractSeedStore(
		NewPostgresContractSeedStoreWithExecutorAndQuerier(db, db),
		NewPostgresContractStoreWithExecutorAndQuerier(db, db),
		NewPostgresEventLogWithExecutorAndQuerier(db, db),
		pgxpoolTransactor{pool: pool},
	)
}

func newPostgresTransactionalContractSeedStore(base *PostgresContractSeedStore, contracts *PostgresContractStore, events *PostgresEventLog, transactor postgresTransactor) *PostgresTransactionalContractSeedStore {
	return &PostgresTransactionalContractSeedStore{
		base:       base,
		contracts:  contracts,
		events:     events,
		transactor: transactor,
	}
}

func (s *PostgresTransactionalContractSeedStore) Create(ctx context.Context, created spine.ContractSeed) error {
	return s.base.Create(ctx, created)
}

func (s *PostgresTransactionalContractSeedStore) Get(ctx context.Context, id spine.ContractSeedID) (spine.ContractSeed, bool, error) {
	return s.base.Get(ctx, id)
}

func (s *PostgresTransactionalContractSeedStore) GetByGoalID(ctx context.Context, id spine.GoalID) (spine.ContractSeed, bool, error) {
	return s.base.GetByGoalID(ctx, id)
}

func (s *PostgresTransactionalContractSeedStore) CreateContractWithSeedAndEvent(ctx context.Context, contract spine.Contract, created spine.ContractSeed, event spine.Event) error {
	if s.transactor == nil {
		return fmt.Errorf("postgres transactor is nil")
	}
	if s.contracts == nil {
		return fmt.Errorf("postgres contract store is nil")
	}
	return s.transactor.ExecReadCommitted(ctx, func(txCtx context.Context) error {
		if err := s.contracts.Create(txCtx, contract); err != nil {
			return err
		}
		if err := s.base.Create(txCtx, created); err != nil {
			return err
		}
		if err := s.events.Append(txCtx, event); err != nil {
			return err
		}
		return nil
	})
}

func (s *PostgresTransactionalContractSeedStore) CreateWithEvent(ctx context.Context, created spine.ContractSeed, event spine.Event) error {
	if s.transactor == nil {
		return fmt.Errorf("postgres transactor is nil")
	}
	return s.transactor.ExecReadCommitted(ctx, func(txCtx context.Context) error {
		if err := s.base.Create(txCtx, created); err != nil {
			return err
		}
		if err := s.events.Append(txCtx, event); err != nil {
			return err
		}
		return nil
	})
}

type PostgresTransactionalContractDraftStore struct {
	base       *PostgresContractDraftStore
	contracts  *PostgresContractStore
	events     *PostgresEventLog
	transactor postgresTransactor
}

func NewPostgresTransactionalContractDraftStore(pool *pgxpool.Pool) *PostgresTransactionalContractDraftStore {
	db := newPostgresDB(pool)
	return newPostgresTransactionalContractDraftStore(
		NewPostgresContractDraftStoreWithExecutorAndQuerier(db, db),
		NewPostgresContractStoreWithExecutorAndQuerier(db, db),
		NewPostgresEventLogWithExecutorAndQuerier(db, db),
		pgxpoolTransactor{pool: pool},
	)
}

func newPostgresTransactionalContractDraftStore(base *PostgresContractDraftStore, contracts *PostgresContractStore, events *PostgresEventLog, transactor postgresTransactor) *PostgresTransactionalContractDraftStore {
	return &PostgresTransactionalContractDraftStore{
		base:       base,
		contracts:  contracts,
		events:     events,
		transactor: transactor,
	}
}

func (s *PostgresTransactionalContractDraftStore) Create(ctx context.Context, created spine.ContractDraft) error {
	return s.base.Create(ctx, created)
}

func (s *PostgresTransactionalContractDraftStore) Update(ctx context.Context, updated spine.ContractDraft) error {
	return s.base.Update(ctx, updated)
}

func (s *PostgresTransactionalContractDraftStore) MarkReadyForApproval(ctx context.Context, updated spine.ContractDraft) error {
	return s.base.MarkReadyForApproval(ctx, updated)
}

func (s *PostgresTransactionalContractDraftStore) Get(ctx context.Context, id spine.ContractDraftID) (spine.ContractDraft, bool, error) {
	return s.base.Get(ctx, id)
}

func (s *PostgresTransactionalContractDraftStore) GetByContractSeedID(ctx context.Context, id spine.ContractSeedID) (spine.ContractDraft, bool, error) {
	return s.base.GetByContractSeedID(ctx, id)
}

func (s *PostgresTransactionalContractDraftStore) CreateWithContractUpdateAndEvent(ctx context.Context, created spine.ContractDraft, event spine.Event, updatedAt time.Time) error {
	if s.transactor == nil {
		return fmt.Errorf("postgres transactor is nil")
	}
	if s.contracts == nil {
		return fmt.Errorf("postgres contract store is nil")
	}
	return s.transactor.ExecReadCommitted(ctx, func(txCtx context.Context) error {
		if err := s.base.Create(txCtx, created); err != nil {
			return err
		}
		if err := s.contracts.MarkDraftCreated(txCtx, created.ContractID, created.ID, updatedAt); err != nil {
			return err
		}
		if err := s.events.Append(txCtx, event); err != nil {
			return err
		}
		return nil
	})
}

func (s *PostgresTransactionalContractDraftStore) CreateWithEvent(ctx context.Context, created spine.ContractDraft, event spine.Event) error {
	if s.transactor == nil {
		return fmt.Errorf("postgres transactor is nil")
	}
	return s.transactor.ExecReadCommitted(ctx, func(txCtx context.Context) error {
		if err := s.base.Create(txCtx, created); err != nil {
			return err
		}
		if err := s.events.Append(txCtx, event); err != nil {
			return err
		}
		return nil
	})
}

func (s *PostgresTransactionalContractDraftStore) UpdateWithEvent(ctx context.Context, updated spine.ContractDraft, event spine.Event) error {
	if s.transactor == nil {
		return fmt.Errorf("postgres transactor is nil")
	}
	return s.transactor.ExecReadCommitted(ctx, func(txCtx context.Context) error {
		if err := s.base.Update(txCtx, updated); err != nil {
			return err
		}
		if err := s.events.Append(txCtx, event); err != nil {
			return err
		}
		return nil
	})
}

func (s *PostgresTransactionalContractDraftStore) MarkReadyForApprovalWithEvent(ctx context.Context, updated spine.ContractDraft, event spine.Event) error {
	if s.transactor == nil {
		return fmt.Errorf("postgres transactor is nil")
	}
	return s.transactor.ExecReadCommitted(ctx, func(txCtx context.Context) error {
		if err := s.base.MarkReadyForApproval(txCtx, updated); err != nil {
			return err
		}
		if err := s.events.Append(txCtx, event); err != nil {
			return err
		}
		return nil
	})
}

func (s *PostgresTransactionalContractDraftStore) MarkReadyForApprovalWithContractUpdateAndEvent(ctx context.Context, updated spine.ContractDraft, event spine.Event, updatedAt time.Time) error {
	if s.transactor == nil {
		return fmt.Errorf("postgres transactor is nil")
	}
	if s.contracts == nil {
		return fmt.Errorf("postgres contract store is nil")
	}
	return s.transactor.ExecReadCommitted(ctx, func(txCtx context.Context) error {
		if err := s.base.MarkReadyForApproval(txCtx, updated); err != nil {
			return err
		}
		if err := s.contracts.MarkReadyForApproval(txCtx, updated.ContractID, updatedAt); err != nil {
			return err
		}
		if err := s.events.Append(txCtx, event); err != nil {
			return err
		}
		return nil
	})
}

type PostgresTransactionalApprovedContractStore struct {
	base       *PostgresApprovedContractStore
	contracts  *PostgresContractStore
	events     *PostgresEventLog
	transactor postgresTransactor
}

func NewPostgresTransactionalApprovedContractStore(pool *pgxpool.Pool) *PostgresTransactionalApprovedContractStore {
	db := newPostgresDB(pool)
	return newPostgresTransactionalApprovedContractStore(
		NewPostgresApprovedContractStoreWithExecutorAndQuerier(db, db),
		NewPostgresContractStoreWithExecutorAndQuerier(db, db),
		NewPostgresEventLogWithExecutorAndQuerier(db, db),
		pgxpoolTransactor{pool: pool},
	)
}

func newPostgresTransactionalApprovedContractStore(base *PostgresApprovedContractStore, contracts *PostgresContractStore, events *PostgresEventLog, transactor postgresTransactor) *PostgresTransactionalApprovedContractStore {
	return &PostgresTransactionalApprovedContractStore{
		base:       base,
		contracts:  contracts,
		events:     events,
		transactor: transactor,
	}
}

func (s *PostgresTransactionalApprovedContractStore) Create(ctx context.Context, approved spine.ApprovedContract) error {
	return s.base.Create(ctx, approved)
}

func (s *PostgresTransactionalApprovedContractStore) Get(ctx context.Context, id spine.ApprovedContractID) (spine.ApprovedContract, bool, error) {
	return s.base.Get(ctx, id)
}

func (s *PostgresTransactionalApprovedContractStore) GetByContractDraftID(ctx context.Context, id spine.ContractDraftID) (spine.ApprovedContract, bool, error) {
	return s.base.GetByContractDraftID(ctx, id)
}

func (s *PostgresTransactionalApprovedContractStore) CreateWithContractUpdateAndEvent(ctx context.Context, approved spine.ApprovedContract, event spine.Event, updatedAt time.Time) error {
	if s.transactor == nil {
		return fmt.Errorf("postgres transactor is nil")
	}
	if s.contracts == nil {
		return fmt.Errorf("postgres contract store is nil")
	}
	return s.transactor.ExecReadCommitted(ctx, func(txCtx context.Context) error {
		if err := s.base.Create(txCtx, approved); err != nil {
			return err
		}
		if err := s.contracts.MarkApproved(txCtx, approved.ContractID, approved.ID, updatedAt); err != nil {
			return err
		}
		if err := s.events.Append(txCtx, event); err != nil {
			return err
		}
		return nil
	})
}

func (s *PostgresTransactionalApprovedContractStore) CreateWithEvent(ctx context.Context, approved spine.ApprovedContract, event spine.Event) error {
	if s.transactor == nil {
		return fmt.Errorf("postgres transactor is nil")
	}
	return s.transactor.ExecReadCommitted(ctx, func(txCtx context.Context) error {
		if err := s.base.Create(txCtx, approved); err != nil {
			return err
		}
		if err := s.events.Append(txCtx, event); err != nil {
			return err
		}
		return nil
	})
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
