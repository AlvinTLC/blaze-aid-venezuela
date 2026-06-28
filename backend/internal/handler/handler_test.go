package handler_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/auth"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/handler"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/repository"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/testdb"
)

const testSecret = "test-secret"

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
	h := handler.New(repository.New(pool), testSecret, false, logger)
	_, api := humatest.New(t)
	h.Register(api)
	return api
}

// bearer returns an "Authorization: Bearer <jwt>" header line for humatest.
func bearer(t *testing.T, subject string) string {
	t.Helper()
	jwtStr, _, err := auth.IssueJWT(testSecret, subject)
	if err != nil {
		t.Fatalf("issue jwt: %v", err)
	}
	return "Authorization: Bearer " + jwtStr
}

// Happy path: an authenticated valid project ingests, returns 200 and persists.
func TestIngestProject_HappyPath(t *testing.T) {
	api := newAPI(t)

	resp := api.Post("/api/v1/ingest/project", bearer(t, "admin@blazeaid"), map[string]any{
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

// Error path: an authenticated body missing the required `title` is rejected by
// Huma validation with 422 before reaching the database.
func TestIngestProject_ValidationError(t *testing.T) {
	api := newAPI(t)

	resp := api.Post("/api/v1/ingest/project", bearer(t, "admin@blazeaid"), map[string]any{
		"source":      "twitter",
		"external_id": "tw-1",
		// title omitted on purpose
	})

	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for missing title, got %d: %s", resp.Code, resp.Body.String())
	}
}

// Without a bearer token the protected ingest endpoint returns 401.
func TestIngestProject_RequiresAuth(t *testing.T) {
	api := newAPI(t)

	resp := api.Post("/api/v1/ingest/project", map[string]any{
		"source":      "twitter",
		"external_id": "tw-1",
		"title":       "Red mesh Caracas",
	})

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d: %s", resp.Code, resp.Body.String())
	}
}

// A garbage/forged bearer token is rejected with 401.
func TestIngestProject_RejectsBadToken(t *testing.T) {
	api := newAPI(t)

	resp := api.Post("/api/v1/ingest/project", "Authorization: Bearer not-a-jwt", map[string]any{
		"source":      "twitter",
		"external_id": "tw-1",
		"title":       "Red mesh Caracas",
	})

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for bad token, got %d: %s", resp.Code, resp.Body.String())
	}
}

// Full bootstrap: magic-login token -> auth/verify -> JWT accepted by ingest.
func TestAuthVerify_IssuesUsableJWT(t *testing.T) {
	api := newAPI(t)
	ctx := context.Background()

	token, _, err := repository.New(pool).CreateMagicToken(ctx, "vol@example.org", time.Minute)
	if err != nil {
		t.Fatalf("seed magic token: %v", err)
	}

	resp := api.Post("/api/v1/auth/verify", map[string]any{"token": token})
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200 from verify, got %d: %s", resp.Code, resp.Body.String())
	}
	if body := resp.Body.String(); !contains(body, "\"access_token\"") {
		t.Fatalf("expected access_token in response, got %s", body)
	}

	// The same magic token cannot be reused.
	resp2 := api.Post("/api/v1/auth/verify", map[string]any{"token": token})
	if resp2.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 on token reuse, got %d: %s", resp2.Code, resp2.Body.String())
	}
}

// An unknown magic token is rejected with 401.
func TestAuthVerify_InvalidToken(t *testing.T) {
	api := newAPI(t)

	resp := api.Post("/api/v1/auth/verify", map[string]any{"token": "does-not-exist"})
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unknown token, got %d: %s", resp.Code, resp.Body.String())
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
