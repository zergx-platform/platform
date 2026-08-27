// Package aggregate: the gateway-owned API face. Each handler fans out to
// multiple backends (agent / jj-server / repo-extension / ops-extension /
// memory-tools) and merges their results into the UI contract.
package aggregate

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"forgejo.develop.10.199.64.20.nip.io/rucoder/go-shared/naming"

	"forgejo.develop.10.199.64.20.nip.io/rucoder/gateway-go/internal/upstream"
)

type API struct {
	Up *upstream.Upstreams
}

func Router(a *API) http.Handler {
	r := chi.NewRouter()
	r.Get("/repos", a.listRepos)
	r.Post("/repos/ensure", a.ensureRepo)
	r.Post("/repos/ensure-org", a.ensureOrg)
	r.Post("/repos/clone", a.cloneRepo)
	r.Post("/repos/fork", a.forkRepo)
	r.Post("/repos/{org}/{repo}/bookmarks/{bm}/session", a.adoptSession)
	r.Get("/sessions", a.listSessions)
	r.Post("/sessions", a.createSession)
	r.Post("/sessions/{id}/prompt", a.sessionPrompt)
	r.Get("/sessions/{id}/messages", a.listMessages)
	r.Post("/sessions/{id}/compact", a.compactSession)
	r.Get("/sessions/{id}/changes", a.sessionChanges)
	r.Get("/sessions/{id}/todos", a.sessionTodos)
	r.Post("/sessions/{id}/fork", a.forkSession)
	r.Patch("/sessions/{id}/settings", a.sessionSettings)
	r.Get("/fs/list", a.fsList)
	r.Get("/fs/read", a.fsRead)
	r.Get("/packages", a.packageTypes)
	r.Get("/packages/list", a.packageList)
	r.Get("/packages/{type}/{name}/versions", a.packageVersions)
	r.Delete("/packages/{type}/{name}", a.packageDelete)
	r.Delete("/packages/{type}/{name}/{version}", a.packageVersionDelete)
	return r
}

// ---- packages: adapted from the artifact system API ----

type pkgSummary struct {
	Format     string   `json:"format"`
	Repository string   `json:"repository"`
	Versions   []string `json:"versions"`
	Source     string   `json:"source"`
	Upstream   string   `json:"upstream"`
}

func (a *API) fetchPackages(ctx context.Context) ([]pkgSummary, error) {
	var res struct {
		Packages []pkgSummary `json:"packages"`
	}
	if err := a.Up.Artifact.JSON(ctx, http.MethodGet, "/pkgs/system/packages", nil, nil, &res); err != nil {
		return nil, err
	}
	return res.Packages, nil
}

func (a *API) packageTypes(w http.ResponseWriter, r *http.Request) {
	pkgs, err := a.fetchPackages(r.Context())
	if err != nil {
		badGateway(w, "artifact", err)
		return
	}
	seen := map[string]int{}
	for _, p := range pkgs {
		seen[p.Format]++
	}
	types := []map[string]interface{}{}
	for f, n := range seen {
		types = append(types, map[string]interface{}{
			"type":     f,
			"packages": n,
			"upstream": defaultUpstream(f),
		})
	}
	sort.Slice(types, func(i, j int) bool {
		return types[i]["type"].(string) < types[j]["type"].(string)
	})
	writeJSON(w, http.StatusOK, map[string]interface{}{"types": types})
}

// defaultUpstream mirrors the artifact's well-known format -> public registry
// map so the UI can show an upstream for each ecosystem.
func defaultUpstream(format string) string {
	m := map[string]string{
		"cargo":    "https://crates.io",
		"composer": "https://repo.packagist.org",
		"conan":    "https://center.conan.io",
		"go":       "https://proxy.golang.org",
		"helm":     "https://charts.helm.sh/stable",
		"hex":      "https://repo.hex.pm",
		"maven":    "https://repo.maven.apache.org/maven2",
		"npm":      "https://registry.npmjs.org",
		"nuget":    "https://api.nuget.org",
		"pub":      "https://pub.dev",
		"pypi":     "https://pypi.org",
		"rubygems": "https://rubygems.org",
		"swift":    "https://api.spm.swift.org",
		"oci":      "https://registry-1.docker.io",
	}
	return m[format]
}

func (a *API) packageList(w http.ResponseWriter, r *http.Request) {
	pkgs, err := a.fetchPackages(r.Context())
	if err != nil {
		badGateway(w, "artifact", err)
		return
	}
	q := r.URL.Query()
	typ, query := q.Get("type"), strings.ToLower(q.Get("q"))
	limit := 50
	offset := 0
	if v, err := strconv.Atoi(q.Get("limit")); err == nil && v > 0 {
		limit = v
	}
	if v, err := strconv.Atoi(q.Get("offset")); err == nil && v > 0 {
		offset = v
	}
	filtered := []map[string]interface{}{}
	for _, p := range pkgs {
		if typ != "" && p.Format != typ {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(p.Repository), query) {
			continue
		}
		latest := ""
		if len(p.Versions) > 0 {
			latest = p.Versions[len(p.Versions)-1]
		}
		filtered = append(filtered, map[string]interface{}{
			"name": p.Repository, "type": p.Format,
			"latest_version": latest, "versions": len(p.Versions),
		})
	}
	total := len(filtered)
	end := offset + limit
	if end > total {
		end = total
	}
	if offset > total {
		offset = total
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok": true,
		"data": map[string]interface{}{
			"packages": filtered[offset:end], "total": total,
			"offset": offset, "limit": limit,
		},
	})
}

func (a *API) packageVersions(w http.ResponseWriter, r *http.Request) {
	typ, name := chi.URLParam(r, "type"), chi.URLParam(r, "name")
	pkgs, err := a.fetchPackages(r.Context())
	if err != nil {
		badGateway(w, "artifact", err)
		return
	}
	for _, p := range pkgs {
		if p.Format == typ && p.Repository == name {
			versions := []map[string]interface{}{}
			for _, v := range p.Versions {
				versions = append(versions, map[string]interface{}{"version": v})
			}
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"ok": true,
				"data": map[string]interface{}{
					"name": name, "type": typ, "versions": versions,
				},
			})
			return
		}
	}
	writeErr(w, http.StatusNotFound, "package not found")
}

func (a *API) packageDelete(w http.ResponseWriter, r *http.Request) {
	typ, name := chi.URLParam(r, "type"), chi.URLParam(r, "name")
	repo := name
	if typ == "oci" || typ == "generic" {
		// repo keys may carry namespaces verbatim; no transformation
		repo = name
	}
	if err := a.Up.Artifact.JSON(r.Context(), http.MethodDelete, "/pkgs/system/packages", nil, upstream.Q("repo", repo), nil); err != nil {
		badGateway(w, "artifact", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}

func (a *API) packageVersionDelete(w http.ResponseWriter, r *http.Request) {
	// The system API deletes whole repositories only.
	writeErr(w, http.StatusNotImplemented, "per-version delete is not supported by the artifact system API")
}

// ---- shared helpers ----

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]interface{}{"ok": false, "error": msg})
}

func badGateway(w http.ResponseWriter, what string, err error) {
	writeErr(w, http.StatusBadGateway, what+": "+err.Error())
}

// parseTriple splits an org:repo:bookmark session name; ok=false when the
// name is not workspace-derived.
func parseTriple(name string) (org, repo, branch string, ok bool) {
	parts := strings.Split(name, ":")
	if len(parts) != 3 {
		return "", "", "", false
	}
	return parts[0], parts[1], parts[2], true
}

// recSession maps an agent session row into the UI Session shape, filling
// org/repo/branch from the name convention.
func recSession(m map[string]interface{}) map[string]interface{} {
	str := func(k string) interface{} {
		v, _ := m[k].(string)
		return v
	}
	num := func(k string) interface{} {
		if v, ok := m[k].(float64); ok {
			return v
		}
		return 0
	}
	name, _ := m["name"].(string)
	org, repo, branch, _ := parseTriple(name)
	return map[string]interface{}{
		"id":             name,
		"org":            org,
		"repo":           repo,
		"branch":         branch,
		"model":          str("model"),
		"preset":         str("preset"),
		"parent_id":      nil,
		"tip_id":         m["tip_id"],
		"fork_at_msg_id": nil,
		"worker_url":     nil,
		"container_id":   nil,
		"max_turns":      nil,
		"system_prompt":  nil,
		"base_image":     nil,
		"unread":         0,
		"input_tokens":   num("input_tokens"),
		"output_tokens":  num("output_tokens"),
		"total_tokens":   num("total_tokens"),
		"created_at":     str("created_at"),
		"updated_at":     str("updated_at"),
	}
}

// ---- GET /repos: merge jj tree + repo-extension bindings + agent sessions ----

type jjTree struct {
	Orgs []struct {
		Org   string `json:"org"`
		Repos []struct {
			Repo      string `json:"repo"`
			Bookmarks []struct {
				Branch string `json:"branch"`
			} `json:"bookmarks"`
		} `json:"repos"`
	} `json:"orgs"`
}

type repoExtTree struct {
	Orgs []struct {
		Org   string `json:"org"`
		Repos []struct {
			Repo      string `json:"repo"`
			Bookmarks []struct {
				Branch      string      `json:"branch"`
				SessionName interface{} `json:"session_name"`
			} `json:"bookmarks"`
		} `json:"repos"`
	} `json:"orgs"`
}

func (a *API) listRepos(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var tree jjTree
	if err := a.Up.Repo.JSON(ctx, http.MethodGet, "/api/v1/repos", nil, nil, &tree); err != nil {
		badGateway(w, "jj-server", err)
		return
	}
	var ext repoExtTree
	_ = a.Up.RepoExt.JSON(ctx, http.MethodGet, "/api/v1/repos", nil, nil, &ext) // best-effort

	// session_name per org/repo/bookmark from repo-extension
	bound := map[string]string{}
	for _, o := range ext.Orgs {
		for _, rp := range o.Repos {
			for _, bm := range rp.Bookmarks {
				if s, ok := bm.SessionName.(string); ok && s != "" {
					bound[o.Org+"/"+rp.Repo+"/"+bm.Branch] = s
				}
			}
		}
	}

	// agent sessions by name (for model/preset on bound bookmarks)
	var sessList struct {
		Sessions []map[string]interface{} `json:"sessions"`
	}
	_ = a.Up.Agent.JSON(ctx, http.MethodGet, "/api/v1/sessions", nil, nil, &sessList)
	byName := map[string]map[string]interface{}{}
	for _, s := range sessList.Sessions {
		if n, ok := s["name"].(string); ok {
			byName[n] = s
		}
	}

	orgs := []map[string]interface{}{}
	for _, o := range tree.Orgs {
		repos := []map[string]interface{}{}
		for _, rp := range o.Repos {
			bms := []map[string]interface{}{}
			for _, bm := range rp.Bookmarks {
				var sess interface{}
				if sn, ok := bound[o.Org+"/"+rp.Repo+"/"+bm.Branch]; ok {
					if row, live := byName[sn]; live {
						s := recSession(row)
						sess = map[string]interface{}{
							"session_id":    sn,
							"branch":        bm.Branch,
							"message_count": 0,
							"model":         s["model"],
							"preset":        s["preset"],
							"parent_id":     nil,
						}
					} else {
						// bound but agent session gone (drift window)
						sess = nil
					}
				}
				bms = append(bms, map[string]interface{}{
					"branch":  bm.Branch,
					"session": sess,
				})
			}
			repos = append(repos, map[string]interface{}{"repo": rp.Repo, "bookmarks": bms})
		}
		orgs = append(orgs, map[string]interface{}{"org": o.Org, "repos": repos})
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"orgs": orgs})
}

// adoptSession: clicking an unbound bookmark in the UI binds it to a
// (created) workspace session via repo-extension's adoption endpoint.
func (a *API) adoptSession(w http.ResponseWriter, r *http.Request) {
	org, repo, bm := chi.URLParam(r, "org"), chi.URLParam(r, "repo"), chi.URLParam(r, "bm")
	path := "/api/v1/repos/" + org + "/" + repo + "/bookmarks/" + bm + "/session"
	var res map[string]interface{}
	if err := a.Up.RepoExt.JSON(r.Context(), http.MethodPost, path, nil, nil, &res); err != nil {
		badGateway(w, "repo-extension", err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// ---- sessions ----

func (a *API) listSessions(w http.ResponseWriter, r *http.Request) {
	var list struct {
		Sessions []map[string]interface{} `json:"sessions"`
	}
	if err := a.Up.Agent.JSON(r.Context(), http.MethodGet, "/api/v1/sessions", nil, nil, &list); err != nil {
		badGateway(w, "agent", err)
		return
	}
	out := []map[string]interface{}{}
	for _, s := range list.Sessions {
		out = append(out, recSession(s))
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"sessions": out})
}

func (a *API) createSession(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Org    string `json:"org"`
		Repo   string `json:"repo"`
		Branch string `json:"branch"`
		Model  string `json:"model"`
		Preset string `json:"preset"`
	}
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if b.Org == "" || b.Repo == "" || b.Branch == "" {
		writeErr(w, http.StatusBadRequest, "org/repo/branch required")
		return
	}
	if bad, ok := naming.Require(
		[2]string{"org", b.Org}, [2]string{"repo", b.Repo}, [2]string{"branch", b.Branch},
	); !ok {
		writeErr(w, http.StatusBadRequest, "invalid "+bad+" name: must match [A-Za-z0-9][A-Za-z0-9._-]{0,127} without ':'/'..'/trailing '.'/'.lock'")
		return
	}
	name := b.Org + ":" + b.Repo + ":" + b.Branch
	body := map[string]interface{}{"name": name}
	if b.Model != "" {
		body["model"] = b.Model
	}
	if b.Preset != "" {
		body["preset"] = b.Preset
	}
	// agent replies {ok, session_name}; synthesize the UI session from the
	// known inputs (a follow-up GET would race the lifecycle fan-out).
	if err := a.Up.Agent.JSON(r.Context(), http.MethodPost, "/api/v1/sessions", body, nil, nil); err != nil {
		badGateway(w, "agent", err)
		return
	}
	row := map[string]interface{}{"name": name, "model": b.Model, "preset": b.Preset}
	writeJSON(w, http.StatusOK, map[string]interface{}{"session": recSession(row)})
}

func (a *API) forkSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var b struct {
		Branch string `json:"branch"`
	}
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil || b.Branch == "" {
		writeErr(w, http.StatusBadRequest, "branch required")
		return
	}
	if bad, ok := naming.Require([2]string{"branch", b.Branch}); !ok {
		writeErr(w, http.StatusBadRequest, "invalid "+bad+" name: must match [A-Za-z0-9][A-Za-z0-9._-]{0,127} without ':'/'..'/trailing '.'/'.lock'")
		return
	}
	org, repo, _, ok := parseTriple(id)
	if !ok {
		writeErr(w, http.StatusBadRequest, "session name is not org:repo:bookmark — cannot fork into a workspace branch")
		return
	}
	name := org + ":" + repo + ":" + b.Branch
	if err := a.Up.Agent.JSON(r.Context(), http.MethodPost, "/api/v1/sessions/"+id+"/fork", map[string]interface{}{"name": name}, nil, nil); err != nil {
		badGateway(w, "agent", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"session": recSession(map[string]interface{}{"name": name})})
}

// compactSession forwards a manual compaction request to the agent.
func (a *API) compactSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var res struct {
		OK bool `json:"ok"`
	}
	if err := a.Up.Agent.JSON(r.Context(), http.MethodPost, "/api/v1/sessions/"+id+"/compact", nil, nil, &res); err != nil {
		badGateway(w, "agent", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": res.OK})
}

// sessionSettings forwards PATCH /sessions/{id}/settings to the agent and
// wraps the returned session in the recoder UI shape (org/repo/branch split
// from the session name).
func (a *API) sessionSettings(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	var res struct {
		Session map[string]interface{} `json:"session"`
	}
	if err := a.Up.Agent.JSON(r.Context(), http.MethodPatch, "/api/v1/sessions/"+id+"/settings", body, nil, &res); err != nil {
		badGateway(w, "agent", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"session": recSession(res.Session)})
}

// sessionPrompt forwards the prompt and synthesizes {ok, messageId}: the
// agent only replies {ok}; the UI needs the id to swap its optimistic
// pending user message.
func (a *API) sessionPrompt(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var b struct {
		Prompt string `json:"prompt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil || b.Prompt == "" {
		writeErr(w, http.StatusBadRequest, "prompt required")
		return
	}
	if err := a.Up.Agent.JSON(r.Context(), http.MethodPost, "/api/v1/sessions/"+id+"/prompt", b, nil, nil); err != nil {
		badGateway(w, "agent", err)
		return
	}
	// The user message is the chain tip after the turn enqueues; read it
	// back so the UI can anchor its optimistic bubble.
	var msgs struct {
		Messages []struct {
			ID   string `json:"id"`
			Role string `json:"role"`
		} `json:"messages"`
	}
	messageId := ""
	if err := a.Up.Agent.JSON(r.Context(), http.MethodGet, "/api/v1/sessions/"+id+"/messages", nil, upstream.Q("limit", "10"), &msgs); err == nil {
		// messages come newest-first; the first user row is this prompt
		for _, m := range msgs.Messages {
			if m.Role == "user" {
				messageId = m.ID
				break
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "messageId": messageId})
}

// listMessages adapts agent message rows into the UI Message shape with
// parts built from role/content/tool_name.
func (a *API) listMessages(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	q := upstream.Q("limit", r.URL.Query().Get("limit"), "before", r.URL.Query().Get("before"))
	var res struct {
		Messages []struct {
			ID        string `json:"id"`
			Role      string `json:"role"`
			Content   string `json:"content"`
			ToolName  string `json:"tool_name"`
			CreatedAt string `json:"created_at"`
		} `json:"messages"`
	}
	if err := a.Up.Agent.JSON(r.Context(), http.MethodGet, "/api/v1/sessions/"+id+"/messages", nil, q, &res); err != nil {
		badGateway(w, "agent", err)
		return
	}
	out := []map[string]interface{}{}
	for _, m := range res.Messages {
		var parts []map[string]interface{}
		if m.Role == "compaction" {
			parts = []map[string]interface{}{{
				"type": "compaction",
				"text": m.Content,
			}}
		} else if m.ToolName != "" {
			parts = []map[string]interface{}{{
				"type": "tool",
				"tool": m.ToolName,
				"state": map[string]interface{}{
					"status": "complete",
					"output": m.Content,
				},
			}}
		} else if m.Content != "" || len(out) == 0 {
			parts = []map[string]interface{}{{"type": "text", "text": m.Content}}
		}
		out = append(out, map[string]interface{}{
			"id":         m.ID,
			"role":       m.Role,
			"parts":      parts,
			"created_at": m.CreatedAt,
		})
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"messages": out})
}

// sessionChanges: the timeline. Fan-out: repo-extension session-map →
// jj-server log on that bookmark. jj shape (change_id/commit_id/author/
// timestamp/message), newest first.
func (a *API) sessionChanges(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	var m map[string]interface{}
	if err := a.Up.RepoExt.JSON(ctx, http.MethodGet, "/api/v1/session-map", nil, upstream.Q("session", id), &m); err != nil {
		badGateway(w, "repo-extension", err)
		return
	}
	org, _ := m["org"].(string)
	repo, _ := m["repo"].(string)
	bm, _ := m["bookmark"].(string)
	if org == "" || repo == "" || bm == "" {
		writeJSON(w, http.StatusOK, map[string]interface{}{"changes": []interface{}{}})
		return
	}
	var log struct {
		Commits []struct {
			ChangeID  string `json:"change_id"`
			CommitID  string `json:"commit_id"`
			Author    string `json:"author"`
			Timestamp string `json:"timestamp"`
			Message   string `json:"message"`
		} `json:"commits"`
	}
	if err := a.Up.Repo.JSON(ctx, http.MethodGet, "/api/v1/repos/"+org+"/"+repo+"/log", nil, upstream.Q("limit", "100", "rev", bm), &log); err != nil {
		badGateway(w, "jj-server", err)
		return
	}
	// Skip the bootstrap commit (README initial) — the UI timeline starts at
	// real work; keep everything, the UI renders messages anyway.
	changes := []map[string]interface{}{}
	for _, c := range log.Commits {
		if strings.HasPrefix(c.Message, "initial commit") {
			continue
		}
		changes = append(changes, map[string]interface{}{
			"change_id": c.ChangeID,
			"commit_id": c.CommitID,
			"author":    c.Author,
			"timestamp": c.Timestamp,
			"message":   c.Message,
		})
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"changes": changes})
}

func (a *API) sessionTodos(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var res map[string]interface{}
	if err := a.Up.Memory.JSON(r.Context(), http.MethodGet, "/api/v1/todos", nil, upstream.Q("session_id", id), &res); err != nil {
		badGateway(w, "memory-tools", err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// ---- fs: jj tree/contents adapted to the recoder fs contract ----

func (a *API) fsList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	org, repo, branch, path := q.Get("org"), q.Get("repo"), q.Get("branch"), q.Get("path")
	if org == "" || repo == "" {
		writeErr(w, http.StatusBadRequest, "org/repo required")
		return
	}
	if branch == "" {
		branch = "main"
	}
	var tree struct {
		Tree []struct {
			Path string `json:"path"`
			Type string `json:"type"`
			Size int64  `json:"size"`
		} `json:"tree"`
	}
	if err := a.Up.Repo.JSON(r.Context(), http.MethodGet, "/api/v1/repos/"+org+"/"+repo+"/"+branch+"/tree", nil, nil, &tree); err != nil {
		badGateway(w, "jj-server", err)
		return
	}
	prefix := strings.Trim(path, "/")
	entries := []map[string]interface{}{}
	seen := map[string]bool{}
	for _, e := range tree.Tree {
		p := e.Path
		if prefix != "" {
			if !strings.HasPrefix(p, prefix+"/") {
				continue
			}
			p = strings.TrimPrefix(p, prefix+"/")
		}
		name := p
		isDir := e.Type == "tree"
		if idx := strings.IndexByte(p, '/'); idx >= 0 {
			name = p[:idx]
			isDir = true
		}
		key := name + "\x00" + map[bool]string{true: "d", false: "f"}[isDir]
		if seen[key] {
			continue
		}
		seen[key] = true
		full := name
		if prefix != "" {
			full = prefix + "/" + name
		}
		entries = append(entries, map[string]interface{}{
			"name":   name,
			"path":   full,
			"is_dir": isDir,
			"size":   e.Size,
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		ei, _ := entries[i]["is_dir"].(bool)
		ej, _ := entries[j]["is_dir"].(bool)
		if ei != ej {
			return ei
		}
		ni, _ := entries[i]["name"].(string)
		nj, _ := entries[j]["name"].(string)
		return ni < nj
	})
	writeJSON(w, http.StatusOK, map[string]interface{}{"entries": entries})
}

func (a *API) fsRead(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	org, repo, branch, path := q.Get("org"), q.Get("repo"), q.Get("branch"), q.Get("path")
	if org == "" || repo == "" || path == "" {
		writeErr(w, http.StatusBadRequest, "org/repo/path required")
		return
	}
	if branch == "" {
		branch = "main"
	}
	var res struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	pth := "/api/v1/repos/" + org + "/" + repo + "/" + branch + "/contents/" + path
	if err := a.Up.Repo.JSON(r.Context(), http.MethodGet, pth, nil, nil, &res); err != nil {
		badGateway(w, "jj-server", err)
		return
	}
	content := res.Content
	if res.Encoding == "base64" {
		if raw, err := base64.StdEncoding.DecodeString(res.Content); err == nil {
			content = string(raw)
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"content": content})
}

// ---- repo lifecycle adaptations ----

func (a *API) ensureOrg(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Org string `json:"org"`
	}
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil || b.Org == "" {
		writeErr(w, http.StatusBadRequest, "org required")
		return
	}
	if bad, ok := naming.Require([2]string{"org", b.Org}); !ok {
		writeErr(w, http.StatusBadRequest, "invalid "+bad+" name: must match [A-Za-z0-9][A-Za-z0-9._-]{0,127} without ':'/'..'/trailing '.'/'.lock'")
		return
	}
	var res map[string]interface{}
	if err := a.Up.Repo.JSON(r.Context(), http.MethodPost, "/api/v1/repos/ensure-org", b, nil, &res); err != nil {
		badGateway(w, "jj-server", err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// ensureRepo: jj ensure (repo + main) then eagerly create the workspace
// session via the agent (name convention triggers repo-extension binding).
func (a *API) ensureRepo(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Org  string `json:"org"`
		Repo string `json:"repo"`
	}
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil || b.Org == "" || b.Repo == "" {
		writeErr(w, http.StatusBadRequest, "org/repo required")
		return
	}
	if bad, ok := naming.Require(
		[2]string{"org", b.Org}, [2]string{"repo", b.Repo},
	); !ok {
		writeErr(w, http.StatusBadRequest, "invalid "+bad+" name: must match [A-Za-z0-9][A-Za-z0-9._-]{0,127} without ':'/'..'/trailing '.'/'.lock'")
		return
	}
	if err := a.Up.Repo.JSON(r.Context(), http.MethodPost, "/api/v1/repos/ensure", b, nil, nil); err != nil {
		badGateway(w, "jj-server", err)
		return
	}
	var created map[string]interface{}
	name := b.Org + ":" + b.Repo + ":main"
	if err := a.Up.Agent.JSON(r.Context(), http.MethodPost, "/api/v1/sessions", map[string]interface{}{"name": name}, nil, &created); err != nil {
		badGateway(w, "agent", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "session_id": name})
}

func (a *API) cloneRepo(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Org    string `json:"org"`
		Repo   string `json:"repo"`
		GitURL string `json:"git_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil || b.Org == "" || b.Repo == "" || b.GitURL == "" {
		writeErr(w, http.StatusBadRequest, "org/repo/git_url required")
		return
	}
	if bad, ok := naming.Require(
		[2]string{"org", b.Org}, [2]string{"repo", b.Repo},
	); !ok {
		writeErr(w, http.StatusBadRequest, "invalid "+bad+" name: must match [A-Za-z0-9][A-Za-z0-9._-]{0,127} without ':'/'..'/trailing '.'/'.lock'")
		return
	}
	if err := a.Up.Repo.JSON(r.Context(), http.MethodPost, "/api/v1/repos/clone", b, nil, nil); err != nil {
		badGateway(w, "jj-server", err)
		return
	}
	name := b.Org + ":" + b.Repo + ":main"
	_ = a.Up.Agent.JSON(r.Context(), http.MethodPost, "/api/v1/sessions", map[string]interface{}{"name": name}, nil, nil)
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "session_id": name})
}

// forkRepo: same-repo branch fork (source branch → target branch), workspace
// session created eagerly. Cross-repo forks are not supported by jj
// POST /repos/{org}/{repo}/bookmarks; rejected with a clear error.
func (a *API) forkRepo(w http.ResponseWriter, r *http.Request) {
	var b struct {
		SourceOrg    string `json:"source_org"`
		SourceRepo   string `json:"source_repo"`
		SourceBranch string `json:"source_branch"`
		TargetOrg    string `json:"target_org"`
		TargetRepo   string `json:"target_repo"`
		TargetBranch string `json:"target_branch"`
	}
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if b.SourceBranch == "" {
		b.SourceBranch = "main"
	}
	if b.TargetBranch == "" {
		b.TargetBranch = b.SourceBranch
	}
	if bad, ok := naming.Require(
		[2]string{"source_org", b.SourceOrg}, [2]string{"source_repo", b.SourceRepo},
		[2]string{"target_org", b.TargetOrg}, [2]string{"target_repo", b.TargetRepo},
		[2]string{"source_branch", b.SourceBranch}, [2]string{"target_branch", b.TargetBranch},
	); !ok {
		writeErr(w, http.StatusBadRequest, "invalid "+bad+" name: must match [A-Za-z0-9][A-Za-z0-9._-]{0,127} without ':'/'..'/trailing '.'/'.lock'")
		return
	}
	if b.SourceOrg != b.TargetOrg || b.SourceRepo != b.TargetRepo {
		writeErr(w, http.StatusBadRequest, "cross-repo fork is not supported; fork within the same repo (branch-level)")
		return
	}
	body := map[string]interface{}{
		"rev": b.SourceBranch, "branch": b.TargetBranch,
	}
	if err := a.Up.Repo.JSON(r.Context(), http.MethodPost, "/api/v1/repos/"+b.SourceOrg+"/"+b.SourceRepo+"/bookmarks", body, nil, nil); err != nil {
		badGateway(w, "jj-server", err)
		return
	}
	name := b.TargetOrg + ":" + b.TargetRepo + ":" + b.TargetBranch
	if err := a.Up.Agent.JSON(r.Context(), http.MethodPost, "/api/v1/sessions", map[string]interface{}{"name": name}, nil, nil); err != nil {
		badGateway(w, "agent", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"session": recSession(map[string]interface{}{"name": name})})
}

var _ = context.Background
