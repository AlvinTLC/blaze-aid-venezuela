package server

import (
	"net/http"
	"time"

	"github.com/go-chi/httprate"
)

// rateLimit limits requests per client IP to rpm per minute, returning 429 when
// exceeded. rpm <= 0 disables limiting.
func rateLimit(rpm int) func(http.Handler) http.Handler {
	if rpm <= 0 {
		return func(next http.Handler) http.Handler { return next }
	}
	return httprate.LimitByIP(rpm, time.Minute)
}

// bodyLimit rejects requests whose body exceeds maxBytes with 413, and caps
// reads (for chunked/unknown-length bodies) via MaxBytesReader. maxBytes <= 0
// disables the cap.
func bodyLimit(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if maxBytes <= 0 {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > maxBytes {
				http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}
