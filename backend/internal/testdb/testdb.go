// Package testdb provides an ephemeral PostgreSQL (TimescaleDB + pgvector) for
// integration tests. New applies the real embedded migrator (same path as boot);
// NewBare returns an empty database. Falls back to TEST_DATABASE_URL when set.
package testdb

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/migrate"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/migrations"
)

// New returns a pool with the full schema applied via the embedded migrator.
func New(ctx context.Context) (*pgxpool.Pool, func(), error) {
	pool, cleanup, err := NewBare(ctx)
	if err != nil {
		return nil, nil, err
	}
	if err := migrate.Run(ctx, pool, migrations.FS); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("apply migrations: %w", err)
	}
	return pool, cleanup, nil
}

// NewBare returns a pool to an empty database (no schema applied).
func NewBare(ctx context.Context) (*pgxpool.Pool, func(), error) {
	if url := os.Getenv("TEST_DATABASE_URL"); url != "" {
		pool, err := pgxpool.New(ctx, url)
		if err != nil {
			return nil, nil, fmt.Errorf("connect TEST_DATABASE_URL: %w", err)
		}
		return pool, func() { pool.Close() }, nil
	}

	if _, ok := os.LookupEnv("TESTCONTAINERS_RYUK_DISABLED"); !ok {
		_ = os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")
	}

	ctr, err := postgres.Run(ctx, "timescale/timescaledb-ha:pg16",
		postgres.WithDatabase("blazeaid"),
		postgres.WithUsername("blazeaid"),
		postgres.WithPassword("blazeaid"),
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
