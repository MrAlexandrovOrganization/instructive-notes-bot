package db

import (
	"context"
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrations embed.FS

// NewPool creates a new PostgreSQL connection pool.
func NewPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	if err = pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return pool, nil
}

// Migrate runs database migrations using goose.
func Migrate(pool *pgxpool.Pool) error {
	goose.SetBaseFS(migrations)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set dialect: %w", err)
	}
	db := stdlib.OpenDBFromPool(pool)
	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	return db.Close()
}
