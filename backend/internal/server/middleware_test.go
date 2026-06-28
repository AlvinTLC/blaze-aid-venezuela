package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestBodyLimit(t *testing.T) {
	h := bodyLimit(10)(okHandler())

	// Under the cap -> passes.
	small := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("hello"))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, small)
	if rec.Code != http.StatusOK {
		t.Fatalf("small body: expected 200, got %d", rec.Code)
	}

	// Over the cap (Content-Length) -> 413.
	big := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("this body is way too long"))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, big)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("big body: expected 413, got %d", rec.Code)
	}
}

func TestBodyLimit_Disabled(t *testing.T) {
	h := bodyLimit(0)(okHandler())
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("x", 1000)))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("disabled cap: expected 200, got %d", rec.Code)
	}
}

func TestRateLimit(t *testing.T) {
	h := rateLimit(2)(okHandler())

	do := func() int {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "1.2.3.4:5678"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}

	if c := do(); c != http.StatusOK {
		t.Fatalf("req 1: expected 200, got %d", c)
	}
	if c := do(); c != http.StatusOK {
		t.Fatalf("req 2: expected 200, got %d", c)
	}
	if c := do(); c != http.StatusTooManyRequests {
		t.Fatalf("req 3: expected 429, got %d", c)
	}
}

func TestRateLimit_Disabled(t *testing.T) {
	h := rateLimit(0)(okHandler())
	for i := 0; i < 50; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "9.9.9.9:1"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("disabled limiter req %d: expected 200, got %d", i, rec.Code)
		}
	}
}
