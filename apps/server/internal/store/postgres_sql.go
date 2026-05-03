package store

import (
	"context"
	"fmt"

	squirrel "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
)

func execSQL(ctx context.Context, exec postgresExecer, op string, sqlizer squirrel.Sqlizer) error {
	if exec == nil {
		return fmt.Errorf("%s executor is nil", op)
	}
	sqlText, args, err := sqlizer.ToSql()
	if err != nil {
		return fmt.Errorf("%s SQL: %w", op, err)
	}
	if _, err := exec.Exec(ctx, sqlText, args...); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func execUpdate(ctx context.Context, exec postgresExecer, op string, notFoundErr error, sqlizer squirrel.Sqlizer) error {
	if exec == nil {
		return fmt.Errorf("%s executor is nil", op)
	}
	sqlText, args, err := sqlizer.ToSql()
	if err != nil {
		return fmt.Errorf("%s SQL: %w", op, err)
	}
	result, err := exec.Exec(ctx, sqlText, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if result.RowsAffected() == 0 {
		return notFoundErr
	}
	return nil
}

func queryRow(ctx context.Context, query postgresRowQuerier, op string, sqlizer squirrel.Sqlizer) (pgx.Row, error) {
	if query == nil {
		return nil, fmt.Errorf("%s query executor is nil", op)
	}
	sqlText, args, err := sqlizer.ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s SQL: %w", op, err)
	}
	return query.QueryRow(ctx, sqlText, args...), nil
}
