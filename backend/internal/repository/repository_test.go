package repository_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/domain/aidproject"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/domain/resource"
	syncdom "github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/domain/sync"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/repository"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/testdb"
)

var pool *pgxpool.Pool

func TestMain(m *testing.M) {
	ctx := context.Background()
	p, cleanup, err := testdb.New(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "testdb setup failed:", err)
		os.Exit(1)
	}
	pool = p
	code := m.Run()
	cleanup()
	os.Exit(code)
}

func reset(t *testing.T) {
	t.Helper()
	if err := testdb.Truncate(context.Background(), pool); err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

// UpsertProject must be idempotent on (source, external_id): a second call with
// the same key updates the row in place (same id, same row count) and advances
// updated_at while preserving created_at.
func TestUpsertProject_Idempotent(t *testing.T) {
	reset(t)
	ctx := context.Background()
	repo := repository.New(pool)

	p := aidproject.AidProject{Source: "twitter", ExternalID: "tw-1", Title: "v1", Region: "DC"}
	id1, err := repo.UpsertProject(ctx, p)
	if err != nil {
		t.Fatalf("first upsert: %v", err)
	}

	time.Sleep(3 * time.Millisecond) // ensure now() differs between the two txns
	p.Title = "v2"
	id2, err := repo.UpsertProject(ctx, p)
	if err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	if id1 != id2 {
		t.Fatalf("expected same id on re-upsert, got %q then %q", id1, id2)
	}

	var count int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM aid_projects WHERE source=$1 AND external_id=$2`,
		p.Source, p.ExternalID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 row, got %d", count)
	}

	var title string
	var created, updated time.Time
	if err := pool.QueryRow(ctx,
		`SELECT title, created_at, updated_at FROM aid_projects WHERE id=$1`, id1).
		Scan(&title, &created, &updated); err != nil {
		t.Fatal(err)
	}
	if title != "v2" {
		t.Fatalf("expected title updated to %q, got %q", "v2", title)
	}
	if !updated.After(created) {
		t.Fatalf("expected updated_at (%s) to advance past created_at (%s)", updated, created)
	}
}

// SyncSince must page by the updated_at cursor: ascending order, respect limit,
// and a second page from the last cursor returns the remainder with no overlap.
func TestSyncSince_Cursor(t *testing.T) {
	reset(t)
	ctx := context.Background()
	repo := repository.New(pool)

	const total = 5
	for i := 0; i < total; i++ {
		_, err := repo.UpsertResource(ctx, resource.Resource{
			Source:     "form",
			ExternalID: fmt.Sprintf("r-%d", i),
			Name:       fmt.Sprintf("res-%d", i),
		})
		if err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
		time.Sleep(2 * time.Millisecond) // distinct updated_at per row
	}

	page1, err := repo.SyncSince(ctx, time.Time{}, 3)
	if err != nil {
		t.Fatalf("page1: %v", err)
	}
	if len(page1) != 3 {
		t.Fatalf("expected 3 changes on page1, got %d", len(page1))
	}
	for i := 1; i < len(page1); i++ {
		if page1[i].UpdatedAt.Before(page1[i-1].UpdatedAt) {
			t.Fatalf("page1 not ascending by updated_at at index %d", i)
		}
	}

	cursor := page1[len(page1)-1].UpdatedAt
	page2, err := repo.SyncSince(ctx, cursor, 10)
	if err != nil {
		t.Fatalf("page2: %v", err)
	}
	if len(page2) != total-3 {
		t.Fatalf("expected %d changes on page2, got %d", total-3, len(page2))
	}

	// Union of both pages must cover all rows with no duplicates.
	seen := make(map[string]bool, total)
	for _, c := range append(append([]string{}, ids(page1)...), ids(page2)...) {
		if seen[c] {
			t.Fatalf("duplicate id across pages: %s", c)
		}
		seen[c] = true
	}
	if len(seen) != total {
		t.Fatalf("expected %d distinct ids across pages, got %d", total, len(seen))
	}
}

func ids(changes []syncdom.Change) []string {
	out := make([]string, len(changes))
	for i, c := range changes {
		out[i] = c.ID
	}
	return out
}
