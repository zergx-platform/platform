// Package proxy: prefix-table streaming reverse proxy + SPA fallback.
package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
)

// Backend routes one path prefix to an upstream service.
type Backend struct {
	Prefix string
	Target *url.URL
}

// Table is the ordered prefix table. Longest prefix wins.
type Table struct {
	Backends []Backend
}

func NewTable(pairs ...struct {
	Prefix string
	URL    string
}) (*Table, error) {
	t := &Table{}
	for _, p := range pairs {
		u, err := url.Parse(p.URL)
		if err != nil {
			return nil, err
		}
		t.Backends = append(t.Backends, Backend{Prefix: p.Prefix, Target: u})
	}
	return t, nil
}

// Match returns the backend for the longest matching prefix, or nil.
func (t *Table) Match(path string) *Backend {
	var best *Backend
	for i := range t.Backends {
		b := &t.Backends[i]
		if path == b.Prefix || strings.HasPrefix(path, b.Prefix+"/") {
			if best == nil || len(b.Prefix) > len(best.Prefix) {
				best = b
			}
		}
	}
	return best
}

// Handler streams requests to the matched backend. SSE-friendly:
// FlushInterval=-1 flushes immediately; bodies are never buffered.
func (t *Table) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b := t.Match(r.URL.Path)
		if b == nil {
			http.NotFound(w, r)
			return
		}
		rp := &httputil.ReverseProxy{
			Director: func(req *http.Request) {
				req.URL.Scheme = b.Target.Scheme
				req.URL.Host = b.Target.Host
				req.Host = b.Target.Host
			},
			FlushInterval: -1,
		}
		rp.ServeHTTP(w, r)
	})
}

// Router mounts the proxy under the given prefixes (each prefix subtree
// falls through to the table), leaving other routes to the caller.
func Mount(r chi.Router, t *Table, prefixes ...string) {
	h := t.Handler()
	for _, p := range prefixes {
		r.Mount(p, h)
	}
}
