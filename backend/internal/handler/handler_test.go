package handler_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/handler"
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

func newAPI(t *testing.T) humatest.TestAPI {
	t.Helper()
	if err := testdb.Truncate(context.Background(), pool); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := handler.New(repository.New(pool), "test-secret", false, logger)
	_, api := humatest.New(t)
	h.Register(api)
	return api
}

// Happy path: a valid project ingests, returns 200 and an upserted result.
func TestIngestProject_HappyPath(t *testing.T) {
	api := newAPI(t)

	resp := api.Post("/api/v1/ingest/project", map[string]any{
		"source":      "twitter",
		"external_id": "tw-1",
		"title":       "Red mesh Caracas",
		"region":      "DC",
	})

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}

	var count int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM aid_projects WHERE source='twitter' AND external_id='tw-1'`).
		Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected the row to be persisted, found %d", count)
	}
}

// Error path: a body missing the required `title` is rejected by Huma validation
// with 422 before reaching the database.
func TestIngestProject_ValidationError(t *testing.T) {
	api := newAPI(t)

	resp := api.Post("/api/v1/ingest/project", map[string]any{
		"source":      "twitter",
		"external_id": "tw-1",
		// title omitted on purpose
	})

	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for missing title, got %d: %s", resp.Code, resp.Body.String())
	}
}

// magic-login in non-production mode returns the token in the body (beta) and
// persists it.
func TestMagicLogin_DevExposesToken(t *testing.T) {
	api := newAPI(t)

	resp := api.Post("/api/v1/magic-login", map[string]any{"email": "vol@example.org"})
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	if body := resp.Body.String(); !contains(body, "\"token\"") {
		t.Fatalf("expected token in dev response, got %s", body)
	}

	var count int
	if err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM magic_tokens WHERE email='vol@example.org'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected token persisted, found %d", count)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
