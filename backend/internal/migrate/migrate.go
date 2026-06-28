// Package migrate applies the embedded SQL schema migrations on boot. It tracks
// applied versions in schema_migrations and serializes concurrent app instances
// with a Postgres advisory lock, so it is safe to run from both api and worker
// against a managed Postgres (Neon/Supabase/RDS) — no external migration tool.
package migrate

import (
	"context"
	"fmt"
	"io/fs"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
)

// advisoryLockKey is an arbitrary app-wide key so only one instance migrates at a time.
const advisoryLockKey int64 = 947563021

// Run applies any embedded migrations not yet recorded in schema_migrations.
func Run(ctx context.Context, pool *pgxpool.Pool, files fs.FS) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", advisoryLockKey); err != nil {
		return fmt.Errorf("acquire migration lock: %w", err)
	}
	defer func() {
		// Best-effort unlock on a fresh context so it runs even if ctx is done.
		_, _ = conn.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", advisoryLockKey)
	}()

	if _, err := conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version    text PRIMARY KEY,
		applied_at timestamptz NOT NULL DEFAULT now()
	)`); err != nil {
		return fmt.Errorf("ensure schema_migrations: %w", err)
	}

	names, err := fs.Glob(files, "*.sql")
	if err != nil {
		return err
	}
	sort.Strings(names)

	for _, name := range names {
		var applied bool
		if err := conn.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)`, name).Scan(&applied); err != nil {
			return err
		}
		if applied {
			continue
		}

		body, err := fs.ReadFile(files, name)
		if err != nil {
			return err
		}
		// No args -> pgx uses the simple protocol, which runs the multi-statement
		// file as one implicit transaction (all-or-nothing).
		if _, err := conn.Exec(ctx, string(body)); err != nil {
			return fmt.Errorf("apply %s: %w", name, err)
		}
		if _, err := conn.Exec(ctx,
			`INSERT INTO schema_migrations (version) VALUES ($1)`, name); err != nil {
			return fmt.Errorf("record %s: %w", name, err)
		}
	}
	return nil
}
