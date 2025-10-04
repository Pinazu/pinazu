package db

import (
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	pq_compat "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func MigrateDb(db *pgxpool.Pool) error {
	// Set the base filesystem for goose migrations
	goose.SetBaseFS(os.DirFS("sql"))

	// Set the dialect to PostgreSQL
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	// Run the migrations
	if err := goose.Up(pq_compat.OpenDBFromPool(db), "migrations"); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	return nil
}
