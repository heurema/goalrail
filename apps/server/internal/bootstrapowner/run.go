package bootstrapowner

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func RunWithPool(ctx context.Context, pool *pgxpool.Pool, input Input) (Result, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return Result{}, err
	}
	defer tx.Rollback(ctx)

	result, err := NewService(NewPostgresStoreWithExecutorAndQuerier(tx, tx)).BootstrapOwner(ctx, input)
	if err != nil {
		return Result{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Result{}, err
	}
	return result, nil
}
