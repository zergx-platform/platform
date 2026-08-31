// Package auth: platform authentication. Two credential forms share the same
// `Authorization: Bearer` header:
//
//  1. The fixed, env-configured pre-shared token (GATEWAY_TOKEN) — the legacy
//     bootstrap/service credential. When set it is accepted as-is so existing
//     clients and service-to-service callers keep working.
//  2. Signed login tokens minted by POST /api/v1/auth/login: the user presents
//     the platform `user:password` credential ONCE and receives a 30-day
//     HMAC-signed token to cache locally.
//
// The HMAC signing key is derived from the platform credential, so no separate
// secret has to be provisioned.
package auth

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"strings"
	"time"
)

// Authenticator validates both the fixed pre-shared token and signed login
// tokens against one credential.
type Authenticator struct {
	credential string
	uid        string
	token      string
	signingKey []byte
}

// New builds an Authenticator from the platform credential (login) and the
// optional fixed pre-shared token. The uid is derived from the user part of
// the credential.
func New(credential, token, secret string) *Authenticator {
	if secret == "" {
		secret = deriveSecret(credential)
	}
	uid := "root"
	if i := strings.IndexByte(credential, ':'); i >= 0 {
		uid = credential[:i]
	}
	return &Authenticator{
		credential: credential,
		uid:        uid,
		token:      strings.Trim(strings.TrimSpace(token), `"'`),
		signingKey: []byte(secret),
	}
}

// Login checks the supplied credential against the configured one and returns
// a signed 30-day token on success.
func (a *Authenticator) Login(supplied string) (string, bool) {
	if a.credential != "" && subtle.ConstantTimeCompare(
		[]byte(supplied), []byte(a.credential),
	) == 1 {
		return mint(a.signingKey, a.uid, time.Now()), true
	}
	return "", false
}

// Allow reports whether the authorization header carries a valid credential:
// either the fixed pre-shared token or a freshly-signed login token.
func (a *Authenticator) Allow(h string) bool {
	const prefix = "Bearer "
	if len(h) <= len(prefix) || !strings.EqualFold(h[:len(prefix)], prefix) {
		return false
	}
	got := strings.TrimSpace(h[len(prefix):])
	if a.token != "" &&
		subtle.ConstantTimeCompare([]byte(got), []byte(a.token)) == 1 {
		return true
	}
	_, ok := verify(a.signingKey, got, time.Now())
	return ok
}

// Middleware wraps the Authenticator as an http middleware: /api/** routes
// (except /api/v1/health) require a valid credential.
func (a *Authenticator) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasPrefix(r.URL.Path, "/api/") ||
				r.URL.Path == "/api/v1/health" ||
				r.URL.Path == "/api/v1/auth/login" {
				next.ServeHTTP(w, r)
				return
			}
			if !a.Allow(r.Header.Get("Authorization")) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"ok":false,"error":"unauthorized"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// deriveSecret hashes the credential into a stable token-signing key so no
// separate secret needs provisioning.
func deriveSecret(credential string) string {
	h := sha256.Sum256([]byte("platform-auth-v1\x00" + credential))
	return string(h[:])
}