package aggregate

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRouteMatchingColonIDs(t *testing.T) {
	fb := newFakeBackends()
	defer fb.Close()
	h := fb.api(t)

	for _, p := range []string{
		"/sessions/acme:api:main/messages?limit=5",
		"/sessions/acme%3Aapi%3Amain/messages?limit=5",
		"/sessions/acme:api:main/changes",
		"/sessions/acme:api:main/todos",
	} {
		req := httptest.NewRequest("GET", p, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		t.Logf("%s → %d", p, rec.Code)
		if rec.Code == 404 {
			t.Errorf("404 on %s", p)
		}
	}

	// prompt ownership: POST must reach the handler (the fake agent 404s
	// the upstream call, so accept 502 as "routed" — only 404/405 mean the
	// route itself is broken).
	req := httptest.NewRequest("POST", "/sessions/acme:api:main/prompt", strings.NewReader(`{"prompt":"hi"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code == 404 || rec.Code == 405 {
		t.Errorf("prompt route broken: %d", rec.Code)
	}
}
