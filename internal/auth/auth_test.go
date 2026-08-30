package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddlewareDisabled(t *testing.T) {
	h := Middleware("")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(204)
	}))
	r := httptest.NewRequest(http.MethodGet, "/api/v1/sessions", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 204 {
		t.Fatalf("disabled auth code = %d, want 204", w.Code)
	}
}

func TestMiddlewareRequiresToken(t *testing.T) {
	h := Middleware("secret-token")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(204)
	}))

	cases := []struct {
		name   string
		header string
		path   string
		want   int
	}{
		{"no header", "", "/api/v1/sessions", 401},
		{"wrong token", "Bearer wrong", "/api/v1/sessions", 401},
		{"wrong scheme", "Basic secret-token", "/api/v1/sessions", 401},
		{"right token", "Bearer secret-token", "/api/v1/sessions", 204},
		{"health exempt", "", "/api/v1/health", 204},
		{"static exempt", "", "/assets/app.js", 204},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, c.path, nil)
			if c.header != "" {
				r.Header.Set("Authorization", c.header)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != c.want {
				t.Fatalf("code = %d, want %d", w.Code, c.want)
			}
		})
	}
}
