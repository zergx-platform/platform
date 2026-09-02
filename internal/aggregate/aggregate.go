// Package aggregate: the platform-owned API face. Each handler fans out to
// multiple backends (agent /  / repo-extension / ops-extension /
// memory-tools) and merges their results into the UI contract.
package aggregate

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"forgejo.develop.10.199.64.20.nip.io/zergx/go-shared/naming"

	"forgejo.develop.10.199.64.20.nip.io/zergx/platform/internal/sessionstate"
	"forgejo.develop.10.199.64.20.nip.io/zergx/platform/internal/upstream"
)

type API struct {
	Up *upstream.Upstreams
	// States carries per-session message facts + read watermarks off the
	// abc Bus. Optional: nil in tests → sessions render without the
	// chat-list extras (preview/unread).
	States *sessionstate.Store
}

func Router(a *API) http.Handler {
	r := chi.NewRouter()
	r.Get("/repos", a.listRepos)
	r.Post("/repos/ensure", a.ensureRepo)
	r.Post("/repos/ensure-org", a.ensureOrg)
	r.Post("/repos/clone", a.cloneRepo)
	r.Post("/repos/fork", a.forkRepo)
	r.Post("/repos/{org}/{repo}/bookmarks/{bm}/session", a.adoptSession)
	r.Delete("/repos/{org}/{repo}/{bookmark}", a.deleteBookmark)
	r.Delete("/repos/{org}/{repo}", a.deleteRepo)
	r.Delete("/repos/{org}", a.deleteOrg)
	r.Get("/sessions", a.listSessions)
	r.Post("/sessions", a.createSession)
	r.Post("/sessions/{id}/prompt", a.sessionPrompt)
	r.Post("/sessions/{id}/read", a.sessionRead)
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
	typ, name := pathParam(r, "type"), pathParam(r, "name")
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
	typ, name := pathParam(r, "type"), pathParam(r, "name")
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

// pathParam decodes a chi URL parameter. chi hands back the raw (still
// percent-encoded) segment, while the UI encodes session ids and package
// names with encodeURIComponent — so every consumer must decode before
// matching, splitting or re-forwarding. The old code compared/forwarded the
// escaped form, which broke every id containing ':' (session timelines,
// todos, fork) and any package name containing '@' or '/'.
func pathParam(r *http.Request, key string) string {
	v := chi.URLParam(r, key)
	if u, err := url.PathUnescape(v); err == nil {
		return u
	}
	return v
}

// sessPath builds a downstream agent path with the id safely (re-)escaped
// exactly once.
func sessPath(id, suffix string) string {
	return "/api/v1/sessions/" + url.PathEscape(id) + suffix
}

// qOpt builds query values, skipping pairs with empty values. An empty
// `limit=`/`before=` is not "unset" to a strict downstream validator: it
// fails number coercion and the route silently returns zero messages.
func qOpt(kv ...string) url.Values {
	q := url.Values{}
	for i := 0; i+1 < len(kv); i += 2 {
		if kv[i+1] != "" {
			q.Set(kv[i], kv[i+1])
		}
	}
	return q
}

// recSession maps an agent session row into the UI Session shape, filling
// org/repo/branch from the name convention. All agent-owned fields pass
// through verbatim (the agent is the source of truth); the platform only adds
// the workspace naming split and the legacy-null cursor/worker placeholders the
// UI still reads.
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
		"id":                 name,
		"org":                org,
		"repo":               repo,
		"branch":             branch,
		"model":              str("model"),
		"preset":             str("preset"),
		"tip_id":             m["tip_id"],
		"max_turns":          num("max_turns"),
		"system_prompt":      str("system_prompt"),
		"input_tokens":       num("input_tokens"),
		"output_tokens":      num("output_tokens"),
		"total_tokens":       num("total_tokens"),
		"last_input_tokens":  num("last_input_tokens"),
		"last_output_tokens": num("last_output_tokens"),
		"created_at":         str("created_at"),
		"updated_at":         str("updated_at"),
	}
}

// ---- GET /repos: merge repo-extension tree (bookmarks + session binding)
// with agent sessions (model/preset). repo-extension already fans out the new
// jjlab directory (`GET /repos` + per-repo `GET /branches`), so the platform
// no longer re-walks the jj tree itself.

// repoExtTree is repo-extension's GET /repos shape: org → repo → bookmarks
// with an optional bound session_name.
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
	var ext repoExtTree
	if err := a.Up.RepoExt.JSON(ctx, http.MethodGet, "/api/v1/repos", nil, nil, &ext); err != nil {
		badGateway(w, "repo-extension", err)
		return
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
	for _, o := range ext.Orgs {
		repos := []map[string]interface{}{}
		for _, rp := range o.Repos {
			bms := []map[string]interface{}{}
			for _, bm := range rp.Bookmarks {
				var sess interface{}
				if sn, ok := bm.SessionName.(string); ok && sn != "" {
					if row, live := byName[sn]; live {
						s := recSession(row)
						sess = map[string]interface{}{
							"session_id":    sn,
							"branch":        bm.Branch,
							"message_count": 0,
							"model":         s["model"],
							"preset":        s["preset"],
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
	org, repo, bm := pathParam(r, "org"), pathParam(r, "repo"), pathParam(r, "bm")
	path := "/api/v1/repos/" + url.PathEscape(org) + "/" + url.PathEscape(repo) + "/bookmarks/" + url.PathEscape(bm) + "/session"
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
	if a.States != nil {
		names := make([]string, 0, len(list.Sessions))
		for _, s := range list.Sessions {
			if n, ok := s["name"].(string); ok {
				names = append(names, n)
			}
		}
		a.States.RefineUnread(r.Context(), names)
	}
	snap := map[string]sessionstate.Session{}
	if a.States != nil {
		names := make([]string, 0, len(list.Sessions))
		for _, s := range list.Sessions {
			if n, ok := s["name"].(string); ok {
				names = append(names, n)
			}
		}
		snap = a.States.Snapshot(names)
	}
	for _, s := range list.Sessions {
		row := recSession(s)
		if a.States != nil {
			if n, ok := s["name"].(string); ok {
				if ss, ok := snap[n]; ok {
					mergeSessionState(row, ss)
				}
			}
		}
		out = append(out, row)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"sessions": out})
}

// mergeSessionState folds bus-carried chat-list facts into a UI session row.
func mergeSessionState(row map[string]interface{}, ss sessionstate.Session) {
	if ss.LastMessageAt != "" {
		row["last_message_at"] = ss.LastMessageAt
	}
	if ss.LastMessagePrev != "" {
		row["last_message_preview"] = ss.LastMessagePrev
	}
	if ss.UnreadCount > 0 {
		row["unread_count"] = ss.UnreadCount
		row["unread_calculated"] = ss.UnreadCalculated
	}
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
	id := pathParam(r, "id")
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
	if err := a.Up.Agent.JSON(r.Context(), http.MethodPost, sessPath(id, "/fork"), map[string]interface{}{"name": name}, nil, nil); err != nil {
		badGateway(w, "agent", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"session": recSession(map[string]interface{}{"name": name})})
}

// compactSession forwards a manual compaction request to the agent.
func (a *API) compactSession(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r, "id")
	var res struct {
		OK bool `json:"ok"`
	}
	if err := a.Up.Agent.JSON(r.Context(), http.MethodPost, sessPath(id, "/compact"), nil, nil, &res); err != nil {
		badGateway(w, "agent", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": res.OK})
}

// sessionSettings forwards PATCH /sessions/{id}/settings to the agent and
// wraps the returned session in the zergx UI shape (org/repo/branch split
// from the session name).
func (a *API) sessionSettings(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r, "id")
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	var res struct {
		Session map[string]interface{} `json:"session"`
	}
	if err := a.Up.Agent.JSON(r.Context(), http.MethodPatch, sessPath(id, "/settings"), body, nil, &res); err != nil {
		badGateway(w, "agent", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"session": recSession(res.Session)})
}

// sessionPrompt forwards the prompt and synthesizes {ok, messageId}: the
// agent only replies {ok}; the UI needs the id to swap its optimistic
// pending user message.
// sessionRead records the read watermark on the platform side (bus vars KV
// under our extension id). The agent is never called: read/unread semantics
// live entirely here.
func (a *API) sessionRead(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r, "id")
	if a.States == nil {
		// States unavailable (e.g. tests): accept and no-op so the UI flow
		// keeps working.
		writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
		return
	}
	if err := a.States.MarkRead(r.Context(), id); err != nil {
		badGateway(w, "bus", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}

func (a *API) sessionPrompt(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r, "id")
	var b struct {
		Prompt string `json:"prompt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil || b.Prompt == "" {
		writeErr(w, http.StatusBadRequest, "prompt required")
		return
	}
	if err := a.Up.Agent.JSON(r.Context(), http.MethodPost, sessPath(id, "/prompt"), b, nil, nil); err != nil {
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
	if err := a.Up.Agent.JSON(r.Context(), http.MethodGet, sessPath(id, "/messages"), nil, upstream.Q("limit", "10"), &msgs); err == nil {
		// messages come oldest-first (chain walk, ORDER BY depth DESC); the
		// LAST user row is the just-persisted prompt. Taking the first row
		// anchored the UI's optimistic bubble to the oldest user message.
		for _, m := range msgs.Messages {
			if m.Role == "user" {
				messageId = m.ID
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "messageId": messageId})
}

// listMessages adapts agent message rows into the UI Message shape with
// parts built from role/content/tool_name.
func (a *API) listMessages(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r, "id")
	q := qOpt("limit", r.URL.Query().Get("limit"), "before", r.URL.Query().Get("before"))
	var res struct {
		Messages []struct {
			ID        string `json:"id"`
			Role      string `json:"role"`
			Content   string `json:"content"`
			ToolName  string `json:"tool_name"`
			ToolParts []struct {
				Type   string      `json:"type"`
				Name   string      `json:"name"`
				Input  interface{} `json:"input"`
				Result string      `json:"result"`
			} `json:"tool_parts"`
			CreatedAt string `json:"created_at"`
		} `json:"messages"`
	}
	if err := a.Up.Agent.JSON(r.Context(), http.MethodGet, sessPath(id, "/messages"), nil, q, &res); err != nil {
		badGateway(w, "agent", err)
		return
	}
	out := []map[string]interface{}{}
	for _, m := range res.Messages {
		parts := []map[string]interface{}{}
		if len(m.ToolParts) > 0 {
			for _, tp := range m.ToolParts {
				parts = append(parts, map[string]interface{}{
					"type": "tool",
					"tool": tp.Name,
					"state": map[string]interface{}{
						"status": "complete",
						"input":  tp.Input,
						"output": tp.Result,
					},
				})
			}
		} else if m.Role == "compaction" {
			parts = append(parts, map[string]interface{}{
				"type": "compaction",
				"text": m.Content,
			})
		} else if m.ToolName != "" {
			parts = append(parts, map[string]interface{}{
				"type": "tool",
				"tool": m.ToolName,
				"state": map[string]interface{}{
					"status": "complete",
					"output": m.Content,
				},
			})
		} else if m.Content != "" {
			parts = append(parts, map[string]interface{}{"type": "text", "text": m.Content})
		}
		// Guarantee a non-null parts slice: a tool message with neither
		// tool_parts nor content would otherwise serialize as JSON null and
		// break the UI's zod validation for the entire batch.
		if parts == nil {
			parts = []map[string]interface{}{}
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
// jjlab log on that bookmark. jj shape (change_id/commit_id/author/
// timestamp/message), newest first.
func (a *API) sessionChanges(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := pathParam(r, "id")
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
			ChangeID    string `json:"change_id"`
			CommitID    string `json:"sha"`
			Author      string `json:"author"`
			Timestamp   string `json:"timestamp"`
			Description string `json:"description"`
		} `json:"commits"`
	}
	if err := a.Up.Repo.JSON(ctx, http.MethodGet, "/api/v1/repos/"+org+"/"+repo+"/commits", nil, upstream.Q("limit", "100", "rev", bm), &log); err != nil {
		badGateway(w, "jjlab", err)
		return
	}
	// Skip the bootstrap commit (README initial) — the UI timeline starts at
	// real work; keep everything, the UI renders messages anyway.
	changes := []map[string]interface{}{}
	for _, c := range log.Commits {
		if strings.HasPrefix(c.Description, "initial commit") {
			continue
		}
		changes = append(changes, map[string]interface{}{
			"change_id": c.ChangeID,
			"commit_id": c.CommitID,
			"author":    c.Author,
			"timestamp": c.Timestamp,
			"message":   c.Description,
		})
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"changes": changes})
}

func (a *API) sessionTodos(w http.ResponseWriter, r *http.Request) {
	id := pathParam(r, "id")
	var res map[string]interface{}
	if err := a.Up.Memory.JSON(r.Context(), http.MethodGet, "/api/v1/todos", nil, upstream.Q("session_id", id), &res); err != nil {
		badGateway(w, "memory-tools", err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// ---- fs: jj tree/contents adapted to the zergx fs contract ----

// defaultBranchCache maps org/repo -> resolved default bookmark.
var (
	dbMu    sync.Mutex
	dbCache = map[string]dbEntry{}
)

type dbEntry struct {
	branch  string
	expires time.Time
}

// resolveBranch returns the given branch, or the repo's conventional default
// bookmark when empty. The new jjlab has no bookmarks in its `GET /repos`
// directory, so resolve via `GET /repos/{org}/{repo}/branches` (each branch
// carries name+sha) and pick main/master/dev/first.
func (a *API) resolveBranch(ctx context.Context, org, repo, branch string) string {
	if branch != "" {
		return branch
	}
	key := org + "/" + repo
	dbMu.Lock()
	if e, ok := dbCache[key]; ok && time.Now().Before(e.expires) {
		dbMu.Unlock()
		return e.branch
	}
	dbMu.Unlock()

	var branches struct {
		Branches []struct {
			Name string `json:"name"`
			Sha  string `json:"sha"`
		} `json:"branches"`
	}
	out := ""
	if err := a.Up.Repo.JSON(ctx, http.MethodGet,
		"/api/v1/repos/"+url.PathEscape(org)+"/"+url.PathEscape(repo)+"/branches",
		nil, nil, &branches); err == nil {
		names := map[string]bool{}
		var first string
		for _, b := range branches.Branches {
			if first == "" {
				first = b.Name
			}
			names[b.Name] = true
		}
		for _, pref := range []string{"main", "master", "dev"} {
			if names[pref] {
				out = pref
				break
			}
		}
		if out == "" {
			out = first
		}
	}
	dbMu.Lock()
	dbCache[key] = dbEntry{branch: out, expires: time.Now().Add(30 * time.Second)}
	dbMu.Unlock()
	return out
}

func (a *API) fsList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	org, repo, branch, path := q.Get("org"), q.Get("repo"), q.Get("branch"), q.Get("path")
	if org == "" || repo == "" {
		writeErr(w, http.StatusBadRequest, "org/repo required")
		return
	}
	branch = a.resolveBranch(r.Context(), org, repo, branch)
	// New jjlab `GET /repos/{org}/{repo}/tree/{sha}` lists the whole tree
	// (kind "tree"/"file"); the sha may be empty (default) or a branch name
	// (resolve_snapshot resolves tags/bookmarks/sha). Use the branch name as
	// the rev.
	var tree struct {
		Tree []struct {
			Path string `json:"path"`
			Kind string `json:"kind"`
			Size int64  `json:"size"`
		} `json:"tree"`
	}
	if err := a.Up.Repo.JSON(r.Context(), http.MethodGet, "/api/v1/repos/"+url.PathEscape(org)+"/"+url.PathEscape(repo)+"/tree/"+url.PathEscape(branch), nil, nil, &tree); err != nil {
		badGateway(w, "jjlab", err)
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
		isDir := e.Kind == "tree"
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
	branch = a.resolveBranch(r.Context(), org, repo, branch)
	// New jjlab `GET /repos/{org}/{repo}/contents/{path}?ref=` returns a
	// Gitea-style entry (base64 content) at a snapshot rev (bookmark/sha).
	var res struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	pth := "/api/v1/repos/" + url.PathEscape(org) + "/" + url.PathEscape(repo) + "/contents/" + path
	if err := a.Up.Repo.JSON(r.Context(), http.MethodGet, pth, nil, upstream.Q("ref", branch), &res); err != nil {
		badGateway(w, "jjlab", err)
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

// ensureOrg: forward to jjlab, which materializes the (possibly empty) org —
// it creates the org directory + DB row, and jjlab's `GET /repos` lists
// empty orgs too, so the frontend "All repositories" tree shows them.
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
	if err := a.Up.Repo.JSON(r.Context(), http.MethodPost,
		"/api/v1/repos/ensure-org", map[string]interface{}{"org": b.Org}, nil, &res); err != nil {
		badGateway(w, "jjlab", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "org": b.Org})
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
	// New jjlab: POST /repos/{org}/{repo} creates the repo (201) with
	// default_branch; a 409 already-exists is treated as success (idempotent
	// ensure). The org materializes implicitly.
	status, data, err := a.Up.Repo.Raw(r.Context(), http.MethodPost,
		"/api/v1/repos/"+url.PathEscape(b.Org)+"/"+url.PathEscape(b.Repo),
		map[string]interface{}{"default_branch": "main"}, nil)
	if err != nil {
		badGateway(w, "jjlab", err)
		return
	}
	if status != 201 && status != 200 && status != 409 {
		writeErr(w, http.StatusBadGateway, fmt.Sprintf("jjlab create repo: HTTP %d: %.200s", status, data))
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
	if err := a.Up.Repo.JSON(r.Context(), http.MethodPost, "/api/v1/repos/"+url.PathEscape(b.Org)+"/"+url.PathEscape(b.Repo)+"/sync/clone", map[string]interface{}{"url": b.GitURL}, nil, nil); err != nil {
		badGateway(w, "jjlab", err)
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
		"target": b.SourceBranch,
	}
	if err := a.Up.Repo.JSON(r.Context(), http.MethodPost, "/api/v1/repos/"+url.PathEscape(b.SourceOrg)+"/"+url.PathEscape(b.SourceRepo)+"/branches/"+url.PathEscape(b.TargetBranch), body, nil, nil); err != nil {
		badGateway(w, "jjlab", err)
		return
	}
	name := b.TargetOrg + ":" + b.TargetRepo + ":" + b.TargetBranch
	if err := a.Up.Agent.JSON(r.Context(), http.MethodPost, "/api/v1/sessions", map[string]interface{}{"name": name}, nil, nil); err != nil {
		badGateway(w, "agent", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"session": recSession(map[string]interface{}{"name": name})})
}

// deleteBookmark deletes a bookmark in , then removes the matching
// agent session (org:repo:bookmark) so it disappears from "Recent".
func (a *API) deleteBookmark(w http.ResponseWriter, r *http.Request) {
	org := pathParam(r, "org")
	repo := pathParam(r, "repo")
	bookmark := pathParam(r, "bookmark")

	// 1. Delete the branch in jjlab (DELETE /repos/{org}/{repo}/branches/{name}).
	path := "/api/v1/repos/" + url.PathEscape(org) + "/" + url.PathEscape(repo) + "/branches/" + url.PathEscape(bookmark)
	if err := a.Up.Repo.JSON(r.Context(), http.MethodDelete, path, nil, nil, nil); err != nil {
		badGateway(w, "jjlab", err)
		return
	}

	// 2. Delete the agent session if it exists (best-effort; a missing session
	// is fine). Interrupting a running turn first is handled by the agent.
	name := org + ":" + repo + ":" + bookmark
	_ = a.Up.Agent.JSON(r.Context(), http.MethodDelete, sessPath(name, ""), nil, nil, nil)

	writeJSON(w, http.StatusOK, map[string]interface{}{"deleted": 1})
}

// deleteRepo deletes a repo in , then removes every agent session
// under org:repo:* so they disappear from "Recent".
func (a *API) deleteRepo(w http.ResponseWriter, r *http.Request) {
	org := pathParam(r, "org")
	repo := pathParam(r, "repo")

	path := "/api/v1/repos/" + url.PathEscape(org) + "/" + url.PathEscape(repo)
	if err := a.Up.Repo.JSON(r.Context(), http.MethodDelete, path, nil, nil, nil); err != nil {
		badGateway(w, "jjlab", err)
		return
	}

	a.deleteSessionsFor(r, org+":"+repo+":")

	writeJSON(w, http.StatusOK, map[string]interface{}{"deleted": 1})
}

// deleteOrg: the new jjlab has no org-level delete (orgs vanish when their
// last repo is deleted). Deleting an org therefore deletes every repo under
// it, then removes the matching agent sessions.
func (a *API) deleteOrg(w http.ResponseWriter, r *http.Request) {
	org := pathParam(r, "org")

	// List the org's repos from the new jjlab directory, delete each repo.
	var tree struct {
		Orgs []struct {
			Org   string `json:"org"`
			Repos []struct {
				Repo string `json:"repo"`
			} `json:"repos"`
		} `json:"orgs"`
	}
	if err := a.Up.Repo.JSON(r.Context(), http.MethodGet, "/api/v1/repos", nil, nil, &tree); err == nil {
		for _, o := range tree.Orgs {
			if o.Org != org {
				continue
			}
			for _, rp := range o.Repos {
				path := "/api/v1/repos/" + url.PathEscape(org) + "/" + url.PathEscape(rp.Repo)
				_ = a.Up.Repo.JSON(r.Context(), http.MethodDelete, path, nil, nil, nil)
			}
		}
	}

	a.deleteSessionsFor(r, org+":")

	writeJSON(w, http.StatusOK, map[string]interface{}{"deleted": 1})
}

// deleteSessionsFor lists agent sessions and best-effort deletes every one
// whose name starts with `prefix` ("org:repo:" or "org:").
func (a *API) deleteSessionsFor(r *http.Request, prefix string) {
	var list struct {
		Sessions []map[string]interface{} `json:"sessions"`
	}
	if err := a.Up.Agent.JSON(r.Context(), http.MethodGet, "/api/v1/sessions", nil, nil, &list); err != nil {
		return // best-effort: sessions will be reaped by reconcile later
	}
	for _, s := range list.Sessions {
		name, _ := s["name"].(string)
		if strings.HasPrefix(name, prefix) {
			_ = a.Up.Agent.JSON(r.Context(), http.MethodDelete, sessPath(name, ""), nil, nil, nil)
		}
	}
}

var _ = context.Background
