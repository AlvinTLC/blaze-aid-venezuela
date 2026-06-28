// Package testdb provides an ephemeral PostgreSQL (TimescaleDB + pgvector) with
// the BlazeAid schema applied, for integration tests. It uses testcontainers-go,
// falling back to TEST_DATABASE_URL when set (e.g. CI without a Docker daemon).
package testdb

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// New returns a ready-to-use pool plus a cleanup func that closes the pool and
// terminates the container (no-op terminate when using TEST_DATABASE_URL).
func New(ctx context.Context) (*pgxpool.Pool, func(), error) {
	if url := os.Getenv("TEST_DATABASE_URL"); url != "" {
		pool, err := pgxpool.New(ctx, url)
		if err != nil {
			return nil, nil, fmt.Errorf("connect TEST_DATABASE_URL: %w", err)
		}
		return pool, func() { pool.Close() }, nil
	}

	// Explicit cleanup below; the reaper is unnecessary and can be blocked in
	// constrained sandboxes.
	if _, ok := os.LookupEnv("TESTCONTAINERS_RYUK_DISABLED"); !ok {
		_ = os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")
	}

	migrations, err := migrationFiles()
	if err != nil {
		return nil, nil, err
	}

	ctr, err := postgres.Run(ctx, "timescale/timescaledb-ha:pg16",
		postgres.WithDatabase("blazeaid"),
		postgres.WithUsername("blazeaid"),
		postgres.WithPassword("blazeaid"),
		postgres.WithInitScripts(migrations...),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(120*time.Second),
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("start postgres container: %w", err)
	}

	url, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = ctr.Terminate(ctx)
		return nil, nil, fmt.Errorf("connection string: %w", err)
	}

	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		_ = ctr.Terminate(ctx)
		return nil, nil, fmt.Errorf("connect pool: %w", err)
	}

	cleanup := func() {
		pool.Close()
		_ = ctr.Terminate(ctx)
	}
	return pool, cleanup, nil
}

// Truncate clears every data table so each test starts from a clean slate.
func Truncate(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx,
		`TRUNCATE aid_projects, resources, missing_persons, volunteers, webhooks_log, events, magic_tokens`)
	return err
}

// migrationFiles returns every migrations/*.sql in lexical order, resolved
// relative to this source file so it works from any package's test.
func migrationFiles() ([]string, error) {
	_, thisFile, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(thisFile), "..", "..", "migrations")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(files)
	return files, nil
}
