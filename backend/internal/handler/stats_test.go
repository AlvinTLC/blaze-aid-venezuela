package handler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/domain/aidproject"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/domain/resource"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/repository"
)

func TestStats(t *testing.T) {
	api := newAPI(t)
	ctx := context.Background()
	repo := repository.New(pool)

	if _, err := repo.UpsertProject(ctx, aidproject.AidProject{
		Source: "s", ExternalID: "p1", Title: "Mesh", Region: "DC", Status: "active", Contact: "+58-1",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.UpsertProject(ctx, aidproject.AidProject{
		Source: "s", ExternalID: "p2", Title: "Agua", Region: "Miranda", Status: "closed",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.UpsertResource(ctx, resource.Resource{
		Source: "s", ExternalID: "r1", Name: "Agua", Region: "DC",
	}); err != nil {
		t.Fatal(err)
	}

	resp := api.Get("/api/v1/stats")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}

	body := resp.Body.String()
	for _, want := range []string{
		`"data"`, `"counts"`, `"projects":2`, `"resources":1`,
		`"by_status"`, `"by_region"`, `"recent"`, `"timeline"`,
		// by_region is region-outer: { "<region>": { ..., "total": N } }
		`"DC"`, `"total"`,
	} {
		if !contains(body, want) {
			t.Fatalf("stats body missing %q:\n%s", want, body)
		}
	}

	// Public endpoint must not leak contact PII in recent rows.
	if contains(body, "+58-1") {
		t.Fatalf("stats leaked contact PII:\n%s", body)
	}
}
