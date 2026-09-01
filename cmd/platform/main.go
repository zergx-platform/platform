package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"

	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"forgejo.develop.10.199.64.20.nip.io/abc-protocol/sdk-go/bus"
	"forgejo.develop.10.199.64.20.nip.io/abc-protocol/sdk-go/extension"
	"forgejo.develop.10.199.64.20.nip.io/abc-protocol/sdk-go/manifest"
	natsbus "forgejo.develop.10.199.64.20.nip.io/abc-protocol/sdk-go/transport/nats"
	"forgejo.develop.10.199.64.20.nip.io/zergx/go-shared/env"
	"forgejo.develop.10.199.64.20.nip.io/zergx/platform/internal/aggregate"
	"forgejo.develop.10.199.64.20.nip.io/zergx/platform/internal/auth"
	"forgejo.develop.10.199.64.20.nip.io/zergx/platform/internal/cors"
	"forgejo.develop.10.199.64.20.nip.io/zergx/platform/internal/proxy"
	"forgejo.develop.10.199.64.20.nip.io/zergx/platform/internal/sessionstate"
	"forgejo.develop.10.199.64.20.nip.io/zergx/platform/internal/upstream"
	"forgejo.develop.10.199.64.20.nip.io/zergx/platform/web"
)

func main() {
	up := upstream.FromEnv(os.Getenv)
	api := &aggregate.API{Up: up}

	// Session-state store: message facts from the agent + read watermarks of
	// our own, both carried by the abc Bus. Optional: startup continues when
	// NATS is unreachable (chat list renders without preview/unread) and the
	// bus reconnect logic inside the SDK retries in the background.
	var states *sessionstate.Store
	var natsBus bus.Bus
	if os.Getenv("ZERGX_DISABLE_NATS") != "1" {
		natsURL := os.Getenv("NATS_URL")
		if natsURL == "" {
			natsURL = "nats://nats.zergx.svc.cluster.local:4222"
		}
		b, err := natsbus.Connect(natsURL)
		if err != nil {
			slog.Warn("nats connect failed — chat-list extras disabled", "err", err)
		} else {
			natsBus = b
			states = sessionstate.New(b, up.Agent)
		}
	}
	api.States = states

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
		pair{"/api/v1/zergx-config", up.Agent.Base},
		// jjlab passthroughs (diff/file/file-log/file-diff/delete/org…)
		pair{"/api/v1/repos", up.Repo.Base},
		pair{"/git", up.Repo.Base},
		pair{"/api/v1/git-blame", up.Repo.Base},
		pair{"/api/v1/git-diff", up.Repo.Base},
		pair{"/api/v1/git-show", up.Repo.Base},
		// repo-extension ops surface
		pair{"/api/v1/session-map", up.RepoExt.Base},
		// ops-extension (sandboxes + deployments + infra + images)
		pair{"/api/v1/sandboxes", up.Ops.Base},
		pair{"/api/v1/deployments", up.Ops.Base},
		pair{"/api/v1/infra", up.Ops.Base},
		pair{"/api/v1/images", up.Ops.Base},
		pair{"/api/v1/builds", up.Ops.Base}, // build/publish task status + SSE log stream
		pair{"/api/v1/status", up.Ops.Base},
		pair{"/api/v1/publish-specs", up.Ops.Base},
		pair{"/api/v1/containerfile-templates", up.Ops.Base},
		// artifact (packages + OCI)
		pair{"/api/v1/packages", up.Artifact.Base},
		pair{"/api/v1/packages/publish", up.Ops.Base},
		pair{"/v2", up.Artifact.Base},
	)
	if err != nil {
		panic(err)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger, middleware.Recoverer)
	// CORS must run before auth so browser preflight OPTIONS is answered
	// (204 + allow headers) instead of hitting the Bearer-token gate.
	r.Use(cors.Middleware(os.Getenv("CORS_ALLOW_ORIGIN")))
	// Platform auth: accepts the fixed pre-shared token (GATEWAY_TOKEN) or a
	// signed login token minted by POST /api/v1/auth/login. Disabled when the
	// platform credential is empty.
	authenticator := auth.New(
		os.Getenv("PLATFORM_CREDENTIAL"),
		os.Getenv("GATEWAY_TOKEN"),
		os.Getenv("PLATFORM_TOKEN_SECRET"),
	)
	r.Use(authenticator.Middleware())

	// Login is unauthenticated (it is how you obtain a token): mount before
	// the auth middleware takes effect on other routes. Since auth's Middleware
	// exempts /api/v1/health only, register login explicitly and exempt its
	// path inside the authenticator is not needed — we mount it as another
	// pre-auth route via the same chi router (login runs first).
	r.Post("/api/v1/auth/login", handleLogin(authenticator))

	r.Get("/api/v1/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"name":"platform"}`))
	})

	// Attach the extension shell AFTER the server wiring: a minimal manifest
	// (no tools/hooks) gives the state store the SDK's canonical
	// session-variable writer for read watermarks.
	if states != nil {
		const platformManifest = `id: platform
version: 0.1.0
`
		if m, err := manifest.ParseManifest([]byte(platformManifest)); err != nil {
			slog.Warn("platform manifest parse failed — read watermarks disabled", "err", err)
		} else {
			states.SetExt(extension.New(natsBus, m.BuildConfig(manifest.Bindings{})))
		}
	}

	// Aggregate handlers take precedence; the proxy table serves the rest.
	r.Mount("/api/v1", aggregateRouter(api, table))

	if states != nil {
		// A minimal extension shell (no tools, no hooks) gives the store the
		// SDK's canonical session-variable writer for its read watermarks.
		const platformManifest = `id: platform
version: 0.1.0
`
		if m, err := manifest.ParseManifest([]byte(platformManifest)); err == nil {
			states.SetExt(extension.New(natsBus, m.BuildConfig(manifest.Bindings{})))
		} else {
			slog.Warn("platform manifest parse failed — read watermarks disabled", "err", err)
		}
	}

	// OCI registry surface: /v2 is the Docker Distribution API root served by
	// artifact (e.g. /v2/_catalog for the UI's image-catalog view). It was
	// registered in the prefix table but never mounted, so /v2/*
	// fell through to the SPA fallback and 404'd.
	proxy.Mount(r, table, "/v2")

	// SPA static: embedded dist (WEB_DIST overrides with a local dir for dev)
	if d := os.Getenv("WEB_DIST"); d != "" {
		fileServer(d, r)
	} else if dist, err := webui.Dist(); err == nil {
		fileServerFS(dist, r)
	} else {
		fileServer("web/dist", r)
	}

	port := env.Or("ZERGX_PORT", "8080")
	srv := &http.Server{Addr: ":" + port, Handler: r}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()
	fmt.Printf("[platform] listening on :%s (agent=%s repo=%s repoext=%s ops=%s artifact=%s)\n",
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
		if ownedByAggregate(path, r.Method) {
			aggHandler.ServeHTTP(w, r)
			return
		}
		t.Handler().ServeHTTP(w, r)
	})
	return inner
}

func ownedByAggregate(path, method string) bool {
	exact := []string{
		"/api/v1/repos", "/api/v1/repos/ensure", "/api/v1/repos/ensure-org",
		"/api/v1/repos/clone", "/api/v1/repos/fork",
		"/api/v1/sessions", "/api/v1/fs/list", "/api/v1/fs/read",
		"/api/v1/packages", "/api/v1/packages/list",
	}
	for _, e := range exact {
		if path == e {
			return true
		}
	}
	if strings.HasPrefix(path, "/api/v1/packages/publish") {
		return false // ops-extension publish (not artifact package CRUD)
	}
	if strings.HasPrefix(path, "/api/v1/packages/") {
		return true // {type}/{name}/versions | {type}/{name} deletes
	}
	if strings.HasPrefix(path, "/api/v1/repos/") && strings.HasSuffix(path, "/session") {
		return true // bookmark adoption (repo-extension)
	}
	// DELETE /api/v1/repos/{org}/{repo}/{bookmark}, /api/v1/repos/{org}/{repo},
	// /api/v1/repos/{org} are wrapped by the aggregate layer so their matching
	// agent sessions are also removed (otherwise they linger in "Recent").
	if method == http.MethodDelete && strings.HasPrefix(path, "/api/v1/repos/") {
		rest := strings.TrimPrefix(path, "/api/v1/repos/")
		if rest != "" {
			segs := strings.Split(strings.Trim(rest, "/"), "/")
			if len(segs) >= 1 && len(segs) <= 3 {
				return true
			}
		}
	}
	// /api/v1/sessions/{id}/{action}: id may itself contain ':' but not '/'
	if strings.HasPrefix(path, "/api/v1/sessions/") {
		sub := strings.TrimPrefix(path, "/api/v1/sessions/")
		for _, action := range []string{"messages", "changes", "todos", "fork", "prompt", "compact", "settings", "read"} {
			if strings.HasSuffix(sub, "/"+action) {
				return true
			}
		}
	}
	return false
}

func handleLogin(a *auth.Authenticator) http.HandlerFunc {
	type loginReq struct {
		Credential string `json:"credential"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var b loginReq
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil || b.Credential == "" {
			http.Error(w, `{"ok":false,"error":"credential required"}`, http.StatusBadRequest)
			return
		}
		token, ok := a.Login(b.Credential)
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"ok":false,"error":"invalid credential"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"token":"` + token + `"}`))
	}
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
