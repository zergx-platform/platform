// Package auth provides the gateway's opt-in pre-shared-token middleware.
//
// When GATEWAY_TOKEN is set (non-empty), every /api/v1/** route — including
// the SSE stream — requires `Authorization: Bearer <token>`. A constant-time
// compare prevents timing side channels. /api/v1/health stays open so
// liveness probes never need a token. When GATEWAY_TOKEN is empty the
// middleware is a no-op (trusted-network default), so a dev gateway keeps
// working exactly as before.
package auth

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// Middleware returns an auth middleware for the given fixed token.
// An empty token disables authentication entirely.
func Middleware(token string) func(http.Handler) http.Handler {
	// Trim surrounding quotes (a Kubernetes env literal may carry them) and
	// whitespace; a truly empty token means "auth disabled".
	token = strings.Trim(strings.TrimSpace(token), `"'`)

	return func(next http.Handler) http.Handler {
		if token == "" {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only API routes require auth; the SPA and static assets are
			// public (they contain no secrets without the API).
			if !strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/api/v1/health" {
				next.ServeHTTP(w, r)
				return
			}
			if !authorized(r, token) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"ok":false,"error":"unauthorized"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// authorized reports whether the request carries a matching bearer token.
func authorized(r *http.Request, token string) bool {
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(h) <= len(prefix) || !strings.EqualFold(h[:len(prefix)], prefix) {
		return false
	}
	got := strings.TrimSpace(h[len(prefix):])
	return subtle.ConstantTimeCompare([]byte(got), []byte(token)) == 1
}
