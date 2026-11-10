// Command migrate applies embedded SQL migrations with a version journal.
//
// Usage:
//
//	DB_DSN=postgres://... migrate up        # apply pending *.up.sql in order
//	DB_DSN=postgres://... migrate status    # list applied/pending versions
//
// Rollbacks are intentionally manual: review backend/migrations/*.down.sql and
// apply the matching file with psql, then delete its schema_migrations row.
// Seed data (seed.sql) is local-dev only and is never applied by this tool;
// provision the first prod admin with `adminctl create-admin` instead.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"avibe-platform/backend/internal/repository"
	"avibe-platform/backend/migrations"
)

var upMigrations = migrations.UpFS

const journalDDL = `CREATE TABLE IF NOT EXISTS schema_migrations (
	version TEXT PRIMARY KEY,
	applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`

func main() {
	flag.Parse()
	if flag.NArg() != 1 || (flag.Arg(0) != "up" && flag.Arg(0) != "status") {
		log.Fatalf("usage: %s [up|status] (DB_DSN env required)", os.Args[0])
	}
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		log.Fatal("DB_DSN is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	db, err := repository.OpenDB(dsn)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	files, err := upMigrations.ReadDir(".")
	if err != nil {
		log.Fatalf("read embedded migrations: %v", err)
	}
	var versions []string
	bodies := map[string][]byte{}
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		body, err := upMigrations.ReadFile(f.Name())
		if err != nil {
			log.Fatalf("read %s: %v", f.Name(), err)
		}
		versions = append(versions, f.Name())
		bodies[f.Name()] = body
	}
	sort.Strings(versions)

	if _, err := db.ExecContext(ctx, journalDDL); err != nil {
		log.Fatalf("journal ddl: %v", err)
	}
	applied, err := appliedVersions(ctx, db)
	if err != nil {
		log.Fatalf("journal read: %v", err)
	}

	if flag.Arg(0) == "status" {
		for _, v := range versions {
			state := "pending"
			if applied[v] {
				state = "applied"
			}
			fmt.Printf("%s %s\n", state, v)
		}
		return
	}

	for _, v := range versions {
		if applied[v] {
			continue
		}
		log.Printf("applying %s", v)
		if err := applyOne(ctx, db, v, bodies[v]); err != nil {
			log.Fatalf("apply %s: %v", v, err)
		}
		log.Printf("applied %s", v)
	}
	log.Print("migrate up: done")
}

func appliedVersions(ctx context.Context, db *sql.DB) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out[v] = true
	}
	return out, rows.Err()
}

func applyOne(ctx context.Context, db *sql.DB, version string, body []byte) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, string(body)); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, version); err != nil {
		return err
	}
	return tx.Commit()
}
