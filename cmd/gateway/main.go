package main

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"forgejo.develop.10.199.64.20.nip.io/rucoder/gateway-go/internal/aggregate"
	"forgejo.develop.10.199.64.20.nip.io/rucoder/gateway-go/internal/proxy"
	"forgejo.develop.10.199.64.20.nip.io/rucoder/gateway-go/internal/upstream"
	"forgejo.develop.10.199.64.20.nip.io/rucoder/gateway-go/web"
)

func main() {
	env := func(k string) string { return os.Getenv(k) }
	up := upstream.FromEnv(env)
	api := &aggregate.API{Up: up}

	type pair = struct {
		Prefix string
		URL    string
	}
	table, err := proxy.NewTable(
		// agent-owned surfaces not adapted by the aggregate layer
		pair{"/api/v1/sessions", up.Agent.Base}, // aggregate wins on its exact subpaths first
		pair{"/api/v1/presets", up.Agent.Base},
		pair{"/api/v1/providers", up.Agent.Base},
		pair{"/api/v1/models", up.Agent.Base},
		pair{"/api/v1/config", up.Agent.Base},
		pair{"/api/v1/tool-config", up.Agent.Base},
		pair{"/api/v1/tools", up.Agent.Base},
		pair{"/api/v1/recore-config", up.Agent.Base},
		// jj-server passthroughs (diff/file/file-log/file-diff/delete/org…)
		pair{"/api/v1/repos", up.Repo.Base},
		pair{"/git", up.Repo.Base},
		// repo-extension ops surface
		pair{"/api/v1/session-map", up.RepoExt.Base},
		// ops-extension (containers subpaths beyond the adapted list/create,
		// sandbox, deploy, infra, images)
		pair{"/api/v1/containers", up.Ops.Base},
		pair{"/api/v1/sandbox", up.Ops.Base},
		pair{"/api/v1/deploy", up.Ops.Base},
		pair{"/api/v1/infra", up.Ops.Base},
		pair{"/api/v1/images", up.Ops.Base},
		pair{"/api/v1/podman", up.Ops.Base},
		// artifact (packages + OCI)
		pair{"/api/v1/packages", up.Artifact.Base},
		pair{"/v2", up.Artifact.Base},
		// browser
		pair{"/api/v1/browser", up.Browser.Base},
	)
	if err != nil {
		panic(err)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger, middleware.Recoverer)

	r.Get("/api/v1/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"name":"gateway-go"}`))
	})

	// Aggregate handlers take precedence; the proxy table serves the rest.
	r.Mount("/api/v1", aggregateRouter(api, table))

	// SPA static: embedded dist (WEB_DIST overrides with a local dir for dev)
	if d := os.Getenv("WEB_DIST"); d != "" {
		fileServer(d, r)
	} else if dist, err := webui.Dist(); err == nil {
		fileServerFS(dist, r)
	} else {
		fileServer("web/dist", r)
	}

	port := envOr("RUCODER_PORT", "8080")
	srv := &http.Server{Addr: ":" + port, Handler: r}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()
	fmt.Printf("[gateway-go] listening on :%s (agent=%s repo=%s repoext=%s ops=%s artifact=%s)\n",
		port, up.Agent.Base, up.Repo.Base, up.RepoExt.Base, up.Ops.Base, up.Artifact.Base)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		panic(err)
	}
}

// aggregateRouter merges the aggregate API routes with the proxy table under
// /api/v1: exact aggregate routes are matched first; everything else falls
// through to the streaming proxy.
func aggregateRouter(api *aggregate.API, t *proxy.Table) http.Handler {
	inner := chi.NewRouter()
	agg := aggregate.Router(api)

	inner.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
		// Try the aggregate router first on exact paths it owns.
		aggHandler := agg
		path := r.URL.Path
		if ownedByAggregate(path) {
			aggHandler.ServeHTTP(w, r)
			return
		}
		t.Handler().ServeHTTP(w, r)
	})
	return inner
}

func ownedByAggregate(path string) bool {
	exact := []string{
		"/api/v1/repos", "/api/v1/repos/ensure", "/api/v1/repos/ensure-org",
		"/api/v1/repos/clone", "/api/v1/repos/fork",
		"/api/v1/sessions", "/api/v1/fs/list", "/api/v1/fs/read",
		"/api/v1/containers", "/api/v1/packages", "/api/v1/packages/list",
	}
	for _, e := range exact {
		if path == e {
			return true
		}
	}
	if strings.HasPrefix(path, "/api/v1/packages/") {
		return true // {type}/{name}/versions | {type}/{name} deletes
	}
	// /api/v1/sessions/{id}/{action}: id may itself contain ':' but not '/'
	if strings.HasPrefix(path, "/api/v1/sessions/") {
		sub := strings.TrimPrefix(path, "/api/v1/sessions/")
		for _, action := range []string{"messages", "changes", "todos", "fork"} {
			if strings.HasSuffix(sub, "/"+action) {
				return true
			}
		}
	}
	return false
}

func fileServer(dir string, r chi.Router) {
	serve(http.FileServer(http.Dir(dir)), r)
}

func fileServerFS(fsys fs.FS, r chi.Router) {
	serve(http.FileServer(http.FS(fsys)), r)
}

func serve(handler http.Handler, r chi.Router) {
	r.NotFound(func(w http.ResponseWriter, req *http.Request) {
		// SPA: non-API paths fall back to index.html
		if strings.HasPrefix(req.URL.Path, "/api/") || strings.HasPrefix(req.URL.Path, "/v2/") {
			http.NotFound(w, req)
			return
		}
		req.URL.Path = "/"
		handler.ServeHTTP(w, req)
	})
	r.Handle("/*", handler)
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
