package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/heurema/goalrail/apps/server/internal/spine"
)

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
	return s.transactor.RunReadCommitted(ctx, func(txCtx context.Context) error {
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
	return s.transactor.RunReadCommitted(ctx, func(txCtx context.Context) error {
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
	err := s.transactor.RunReadCommitted(ctx, func(txCtx context.Context) error {
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

type PostgresTransactionalWorkItemStore struct {
	base       *PostgresWorkItemStore
	events     *PostgresEventLog
	transactor postgresTransactor
}

func NewPostgresTransactionalWorkItemStore(pool *pgxpool.Pool) *PostgresTransactionalWorkItemStore {
	db := newPostgresDB(pool)
	return newPostgresTransactionalWorkItemStore(
		NewPostgresWorkItemStoreWithExecutorAndQuerier(db, db),
		NewPostgresEventLogWithExecutorAndQuerier(db, db),
		pgxpoolTransactor{pool: pool},
	)
}

func newPostgresTransactionalWorkItemStore(base *PostgresWorkItemStore, events *PostgresEventLog, transactor postgresTransactor) *PostgresTransactionalWorkItemStore {
	return &PostgresTransactionalWorkItemStore{
		base:       base,
		events:     events,
		transactor: transactor,
	}
}

func (s *PostgresTransactionalWorkItemStore) Create(ctx context.Context, item spine.WorkItem) error {
	return s.base.Create(ctx, item)
}

func (s *PostgresTransactionalWorkItemStore) Get(ctx context.Context, id spine.WorkItemID) (spine.WorkItem, bool, error) {
	return s.base.Get(ctx, id)
}

func (s *PostgresTransactionalWorkItemStore) GetByApprovedContractID(ctx context.Context, id spine.ApprovedContractID) (spine.WorkItem, bool, error) {
	return s.base.GetByApprovedContractID(ctx, id)
}

func (s *PostgresTransactionalWorkItemStore) CreateWithEvent(ctx context.Context, item spine.WorkItem, event spine.Event) error {
	if s.transactor == nil {
		return fmt.Errorf("postgres transactor is nil")
	}
	return s.transactor.RunReadCommitted(ctx, func(txCtx context.Context) error {
		if err := s.base.Create(txCtx, item); err != nil {
			return err
		}
		if err := s.events.Append(txCtx, event); err != nil {
			return err
		}
		return nil
	})
}
