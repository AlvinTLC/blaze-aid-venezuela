package handler_test

import (
	"net/http"
	"testing"
)

// Errors use the {error:{code,message}} envelope (not RFC7807).
func TestErrorEnvelope(t *testing.T) {
	api := newAPI(t)

	// 404 detail endpoint.
	resp := api.Get("/api/v1/projects/00000000-0000-0000-0000-000000000000")
	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.Code)
	}
	body := resp.Body.String()
	for _, want := range []string{`"error"`, `"code":404`, `"message"`} {
		if !contains(body, want) {
			t.Fatalf("404 body missing %q:\n%s", want, body)
		}
	}
	if contains(body, `"$schema"`) || contains(body, `"title"`) {
		t.Fatalf("error body still looks like RFC7807:\n%s", body)
	}

	// 422 validation error also uses the envelope.
	bad := api.Post("/api/v1/ingest/project", bearer(t, "admin@blazeaid"),
		map[string]any{"source": "x", "external_id": "y"}) // missing title
	if bad.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422, got %d", bad.Code)
	}
	if !contains(bad.Body.String(), `"error"`) || !contains(bad.Body.String(), `"code":422`) {
		t.Fatalf("422 body not in envelope:\n%s", bad.Body.String())
	}
}
