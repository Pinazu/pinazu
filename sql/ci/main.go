package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	pq_compat "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func main() {
	// Connect to the database
	pool, err := pgxpool.New(context.Background(), os.Getenv("POSTGRES_URL"))
	if err != nil {
		panic(fmt.Errorf("failed to connect to the database: %w", err))
	}
	defer pool.Close()
	goose.SetBaseFS(os.DirFS("sql"))
	if err := goose.SetDialect("postgres"); err != nil {
		panic(fmt.Errorf("failed to set goose dialect: %w", err))
	}
	if err := goose.Up(pq_compat.OpenDBFromPool(pool), "migrations"); err != nil {
		panic(fmt.Errorf("failed to run migrations: %w", err))
	}
}
