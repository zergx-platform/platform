// Package cors adds cross-origin support to the platform API.
//
// The embedded Svelte UI is served same-origin and never needs this, but the
// standalone Flutter web client is hosted on a different domain and its
// browser requests cross the origin boundary. Without CORS headers the
// browser blocks every /api/v1/** fetch (after a preflight that the auth
// middleware would otherwise reject with 401).
package cors

import (
	"net/http"
	"strings"
)

// Middleware returns a CORS middleware that allows a caller-configurable
// origin (CORS_ALLOW_ORIGIN, default: reflect the request Origin — i.e.
// allow any origin; the platform's Bearer-token auth carries no cookies, so
// credentialed "allow all" semantics are not a concern here).
func Middleware(allowOrigin string) func(http.Handler) http.Handler {
	allowOrigin = strings.TrimSpace(allowOrigin)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only API surfaces need CORS; static files come from the
			// platform itself and are same-origin.
			if !strings.HasPrefix(r.URL.Path, "/api/") && !strings.HasPrefix(r.URL.Path, "/v2/") {
				next.ServeHTTP(w, r)
				return
			}

			origin := r.Header.Get("Origin")
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			allowed := "*"
			if allowOrigin != "" {
				allowed = allowOrigin
			}

			h := w.Header()
			if allowed == "*" {
				h.Set("Access-Control-Allow-Origin", "*")
			} else {
				// Restrict to the configured origin; only emit the header
				// when the request actually originates from it.
				if origin != allowed {
					next.ServeHTTP(w, r)
					return
				}
				h.Set("Access-Control-Allow-Origin", allowed)
				h.Add("Vary", "Origin")
			}
			h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")

			// Preflight: answer directly so the auth middleware (and the
			// handler) never see the OPTIONS request.
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}