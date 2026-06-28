package handler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/domain/aidproject"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/repository"
)

func seedProject(t *testing.T) string {
	t.Helper()
	id, err := repository.New(pool).UpsertProject(context.Background(), aidproject.AidProject{
		Source: "s", ExternalID: "w1", Title: "Original", Region: "DC", Contact: "+58-555",
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// PATCH updates only the provided field and requires auth.
func TestPatchProject(t *testing.T) {
	api := newAPI(t)
	id := seedProject(t)

	resp := api.Patch("/api/v1/projects/"+id, bearer(t, "admin@blazeaid"),
		map[string]any{"title": "Updated title"})
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	if b := resp.Body.String(); !contains(b, "Updated title") {
		t.Fatalf("expected updated title, got %s", b)
	}

	// Region (not in patch) must be preserved.
	got, err := repository.New(pool).GetProject(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Updated title" || got.Region != "DC" {
		t.Fatalf("patch should change only title; got title=%q region=%q", got.Title, got.Region)
	}
}

// PUT replaces the full record (and requires auth).
func TestPutProject(t *testing.T) {
	api := newAPI(t)
	id := seedProject(t)

	resp := api.Put("/api/v1/projects/"+id, bearer(t, "admin@blazeaid"),
		map[string]any{"title": "Replaced", "region": "Vargas", "status": "closed"})
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}

	got, err := repository.New(pool).GetProject(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Replaced" || got.Region != "Vargas" || got.Status != "closed" {
		t.Fatalf("PUT did not replace fields: %+v", got)
	}

	noauth := api.Put("/api/v1/projects/"+id, map[string]any{"title": "x"})
	if noauth.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", noauth.Code)
	}
}

func TestPatchProject_RequiresAuth(t *testing.T) {
	api := newAPI(t)
	id := seedProject(t)
	resp := api.Patch("/api/v1/projects/"+id, map[string]any{"title": "x"})
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", resp.Code)
	}
}

// DELETE soft-deletes: 204, then the row disappears from reads.
func TestDeleteProject_SoftDelete(t *testing.T) {
	api := newAPI(t)
	id := seedProject(t)

	resp := api.Delete("/api/v1/projects/"+id, bearer(t, "admin@blazeaid"))
	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", resp.Code, resp.Body.String())
	}

	get := api.Get("/api/v1/projects/" + id)
	if get.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", get.Code)
	}
	list := api.Get("/api/v1/projects")
	if b := list.Body.String(); !contains(b, "\"total\":0") {
		t.Fatalf("expected total 0 after delete, got %s", b)
	}

	// Deleting again is 404 (already gone).
	again := api.Delete("/api/v1/projects/"+id, bearer(t, "admin@blazeaid"))
	if again.Code != http.StatusNotFound {
		t.Fatalf("expected 404 on second delete, got %d", again.Code)
	}
}

// PII: contact is hidden for anonymous readers, visible with a valid JWT.
func TestPII_ContactGatedByAuth(t *testing.T) {
	api := newAPI(t)
	id := seedProject(t)

	anon := api.Get("/api/v1/projects/" + id)
	if b := anon.Body.String(); contains(b, "+58-555") {
		t.Fatalf("anonymous read must NOT expose contact, got %s", b)
	}

	authed := api.Get("/api/v1/projects/"+id, bearer(t, "admin@blazeaid"))
	if b := authed.Body.String(); !contains(b, "+58-555") {
		t.Fatalf("authenticated read must expose contact, got %s", b)
	}
}
