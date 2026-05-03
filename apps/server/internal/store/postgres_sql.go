package store

import (
	"context"
	"fmt"

	squirrel "github.com/Masterminds/squirrel"
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
