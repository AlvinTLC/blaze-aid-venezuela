package repository_test

import (
	"context"
	"errors"
	"testing"

	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/domain/aidproject"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/repository"
)

func TestListProjects_FilterPaginate(t *testing.T) {
	reset(t)
	ctx := context.Background()
	repo := repository.New(pool)

	seed := []aidproject.AidProject{
		{Source: "s", ExternalID: "p1", Title: "Mesh network", Region: "DC", Status: "active", Category: "connectivity"},
		{Source: "s", ExternalID: "p2", Title: "Water trucks", Region: "DC", Status: "active", Category: "water"},
		{Source: "s", ExternalID: "p3", Title: "Shelter build", Region: "Miranda", Status: "closed", Category: "shelter"},
	}
	for _, p := range seed {
		if _, err := repo.UpsertProject(ctx, p); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	// Filter by region.
	items, total, err := repo.ListProjects(ctx, repository.ListParams{Region: "DC"})
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(items) != 2 {
		t.Fatalf("region=DC: expected total=2 items=2, got total=%d items=%d", total, len(items))
	}

	// Free-text + category.
	items, total, err = repo.ListProjects(ctx, repository.ListParams{Q: "mesh"})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(items) != 1 || items[0].ExternalID != "p1" {
		t.Fatalf("q=mesh: expected the mesh project, got total=%d items=%d", total, len(items))
	}

	// Pagination: limit keeps total but caps the page.
	items, total, err = repo.ListProjects(ctx, repository.ListParams{Region: "DC", Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(items) != 1 {
		t.Fatalf("limit=1: expected total=2 items=1, got total=%d items=%d", total, len(items))
	}
}

func TestGetProject_NotFound(t *testing.T) {
	reset(t)
	_, err := repository.New(pool).GetProject(context.Background(), "00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
