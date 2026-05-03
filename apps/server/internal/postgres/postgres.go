package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/heurema/goalrail/apps/server/internal/config"
	"github.com/heurema/goalrail/apps/server/internal/postgres/migrations"
)

var ErrDatabaseNotConfigured = errors.New("database_not_configured")

const migrationsDir = "."

func OpenPool(ctx context.Context, db config.DatabaseConfig) (*pgxpool.Pool, error) {
	if !db.Configured() {
		return nil, ErrDatabaseNotConfigured
	}

	cfg, err := db.PGXPoolConfig()
	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("open postgres pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return pool, nil
}

func OpenSQLDB(db config.DatabaseConfig) (*sql.DB, error) {
	if !db.Configured() {
		return nil, ErrDatabaseNotConfigured
	}

	cfg, err := db.PGXConfig()
	if err != nil {
		return nil, err
	}
	return stdlib.OpenDB(*cfg), nil
}

func MigrateUp(ctx context.Context, cfg config.DatabaseConfig) error {
	db, err := OpenSQLDB(cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}
	if err := goose.UpContext(ctx, db, migrationsDir); err != nil {
		return fmt.Errorf("run postgres migrations: %w", err)
	}
	return nil
}
