// Package repository holds database/sql (pgx stdlib) and go-redis access.
package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/redis/go-redis/v9"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// DBTX abstracts *sql.DB and *sql.Tx so repositories can run inside a
// transaction. Both types satisfy it; see Repos.WithTx.
type DBTX interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// OpenDB opens a PostgreSQL connection pool via the pgx stdlib driver.
func OpenDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	return db, nil
}

// NewRedisClient builds a go-redis client for addr (host:port).
func NewRedisClient(addr string) *redis.Client {
	return redis.NewClient(&redis.Options{Addr: addr})
}
