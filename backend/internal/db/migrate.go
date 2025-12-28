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
var migrationsFS embed.FS

func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	goose.SetBaseFS(migrationsFS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("goose.SetDialect: %w", err)
	}

	sqldb := stdlib.OpenDBFromPool(pool)
	defer sqldb.Close()

	if err := goose.UpContext(ctx, sqldb, "migrations"); err != nil {
		return fmt.Errorf("goose.Up: %w", err)
	}
	return nil
}
