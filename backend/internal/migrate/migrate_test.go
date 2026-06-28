package migrate_test

import (
	"context"
	"testing"

	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/migrate"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/testdb"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/migrations"
)

// Run on an empty DB creates the schema and records versions; a second run is a
// no-op (idempotent), proving safe re-runs on every boot.
func TestRun_AppliesAndIsIdempotent(t *testing.T) {
	ctx := context.Background()
	pool, cleanup, err := testdb.NewBare(ctx)
	if err != nil {
		t.Fatalf("bare db: %v", err)
	}
	defer cleanup()

	if err := migrate.Run(ctx, pool, migrations.FS); err != nil {
		t.Fatalf("first migrate: %v", err)
	}

	// Schema is present.
	var tables int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM information_schema.tables
		WHERE table_schema='public' AND table_name IN ('aid_projects','events','webhooks_log','magic_tokens')`).
		Scan(&tables); err != nil {
		t.Fatal(err)
	}
	if tables != 4 {
		t.Fatalf("expected 4 core tables created, got %d", tables)
	}

	var firstCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM schema_migrations`).Scan(&firstCount); err != nil {
		t.Fatal(err)
	}
	if firstCount < 2 {
		t.Fatalf("expected >=2 recorded migrations, got %d", firstCount)
	}

	// Second run must be a no-op and not error.
	if err := migrate.Run(ctx, pool, migrations.FS); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
	var secondCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM schema_migrations`).Scan(&secondCount); err != nil {
		t.Fatal(err)
	}
	if secondCount != firstCount {
		t.Fatalf("re-run changed migration count: %d -> %d", firstCount, secondCount)
	}
}
