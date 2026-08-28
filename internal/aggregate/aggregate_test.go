package aggregate

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"forgejo.develop.10.199.64.20.nip.io/zergx/gateway-go/internal/upstream"
	"forgejo.develop.10.199.64.20.nip.io/zergx/go-shared/naming"
)

type fakeBackends struct {
	agent, repo, repoExt, ops, memory *httptest.Server
}

func newFakeBackends() *fakeBackends {
	fb := &fakeBackends{}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/sessions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`{"sessions":[
				{"name":"acme:api:main","model":"m1","preset":"p1","tip_id":"t1","input_tokens":10,"output_tokens":5,"total_tokens":15,"created_at":"2026-01-01","updated_at":"2026-01-02"},
				{"name":"hi","model":"","preset":"","tip_id":null,"created_at":"2026-01-01","updated_at":"2026-01-01"}
			]}`))
			return
		}
		var b map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&b)
		name, _ := b["name"].(string)
		_, _ = w.Write([]byte(`{"ok":true,"session_name":"` + name + `"}`))
	})
	mux.HandleFunc("/api/v1/sessions/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/messages") {
			_, _ = w.Write([]byte(`{"messages":[
				{"id":"m1","role":"user","content":"hi","tool_name":"","created_at":"2026-01-01"},
				{"id":"m2","role":"assistant","content":"","tool_name":"write","created_at":"2026-01-01"}
			]}`))
			return
		}
		http.NotFound(w, r)
	})
	fb.agent = httptest.NewServer(mux)

	mux = http.NewServeMux()
	mux.HandleFunc("/api/v1/repos", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"orgs":[{"org":"acme","repos":[{"repo":"api","bookmarks":[{"branch":"main"},{"branch":"dev"}]}]}]}`))
	})
	mux.HandleFunc("/api/v1/repos/ensure", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	mux.HandleFunc("/api/v1/repos/acme/api/bookmarks", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true,"branch":"dev"}`))
	})
	mux.HandleFunc("/api/v1/repos/acme/api/main/tree", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tree":[
			{"path":"README.md","type":"blob","size":10},
			{"path":"src","type":"tree","size":0},
			{"path":"src/main.go","type":"blob","size":100},
			{"path":"src/util.go","type":"blob","size":50}
		]}`))
	})
	mux.HandleFunc("/api/v1/repos/acme/api/main/contents/hello.txt", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"content":"aGVsbG8gd29ybGQ=","encoding":"base64","sha":"x","size":11}`))
	})
	mux.HandleFunc("/api/v1/repos/acme/api/log", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("rev") != "main" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"commits":[
			{"change_id":"c2","commit_id":"k2","author":"zergx","timestamp":"2026-01-02","message":"edit hello"},
			{"change_id":"c1","commit_id":"k1","author":"zergx","timestamp":"2026-01-01","message":"write hello"},
			{"change_id":"c0","commit_id":"k0","author":"zergx","timestamp":"2026-01-01","message":"initial commit"}
		]}`))
	})
	fb.repo = httptest.NewServer(mux)

	mux = http.NewServeMux()
	mux.HandleFunc("/api/v1/repos", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"orgs":[{"org":"acme","repos":[{"repo":"api","managed":true,"bookmarks":[{"branch":"main","session_name":"acme:api:main"},{"branch":"dev","session_name":null}]}]}]}`))
	})
	mux.HandleFunc("/api/v1/session-map", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("session") == "acme:api:main" {
			_, _ = w.Write([]byte(`{"org":"acme","repo":"api","bookmark":"main"}`))
			return
		}
		http.NotFound(w, r)
	})
	fb.repoExt = httptest.NewServer(mux)

	mux = http.NewServeMux()
	mux.HandleFunc("/api/v1/containers", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`{"containers":[{"container_id":"cid-1","pod_name":"pod-1","worker_url":"http://w1","status":"Running"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"container":{"id":"cid-2","name":"pod-2","worker_url":"http://w2","status":"Running"}}`))
	})
	fb.ops = httptest.NewServer(mux)

	mux = http.NewServeMux()
	mux.HandleFunc("/api/v1/todos", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("session_id") == "acme:api:main" {
			_, _ = w.Write([]byte(`{"todos":[{"id":"1","content":"x"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"todos":[]}`))
	})
	fb.memory = httptest.NewServer(mux)

	return fb
}

func (fb *fakeBackends) Close() {
	fb.agent.Close()
	fb.repo.Close()
	fb.repoExt.Close()
	fb.ops.Close()
	fb.memory.Close()
}

func (fb *fakeBackends) api(t *testing.T) http.Handler {
	t.Helper()
	return Router(&API{Up: &upstream.Upstreams{
		Agent:   upstream.New(fb.agent.URL),
		Repo:    upstream.New(fb.repo.URL),
		RepoExt: upstream.New(fb.repoExt.URL),
		Ops:     upstream.New(fb.ops.URL),
		Memory:  upstream.New(fb.memory.URL),
	}})
}

func do(t *testing.T, h http.Handler, method, path string, body string) (int, map[string]interface{}) {
	t.Helper()
	var rd *strings.Reader
	if body != "" {
		rd = strings.NewReader(body)
	} else {
		rd = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, rd)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var v map[string]interface{}
	_ = json.NewDecoder(rec.Body).Decode(&v)
	return rec.Code, v
}

func TestListReposMergesJJRepoExtAgent(t *testing.T) {
	fb := newFakeBackends()
	defer fb.Close()
	h := fb.api(t)

	code, v := do(t, h, "GET", "/repos", "")
	if code != 200 {
		t.Fatalf("code=%d", code)
	}
	orgs := v["orgs"].([]interface{})
	org := orgs[0].(map[string]interface{})
	if org["org"] != "acme" {
		t.Fatalf("org=%v", org["org"])
	}
	repos := org["repos"].([]interface{})
	bms := repos[0].(map[string]interface{})["bookmarks"].([]interface{})
	if len(bms) != 2 {
		t.Fatalf("bookmarks=%d", len(bms))
	}
	main := bms[0].(map[string]interface{})
	if main["branch"] != "main" {
		t.Fatalf("branch=%v", main["branch"])
	}
	sess := main["session"].(map[string]interface{})
	if sess["session_id"] != "acme:api:main" || sess["model"] != "m1" {
		t.Fatalf("session=%v", sess)
	}
	if bms[1].(map[string]interface{})["session"] != nil {
		t.Fatal("unbound dev bookmark should have null session")
	}
}

func TestListSessionsParsesNames(t *testing.T) {
	fb := newFakeBackends()
	defer fb.Close()
	h := fb.api(t)

	_, v := do(t, h, "GET", "/sessions", "")
	sessions := v["sessions"].([]interface{})
	s0 := sessions[0].(map[string]interface{})
	if s0["id"] != "acme:api:main" || s0["org"] != "acme" || s0["repo"] != "api" || s0["branch"] != "main" {
		t.Fatalf("derived=%v %v %v %v", s0["id"], s0["org"], s0["repo"], s0["branch"])
	}
	s1 := sessions[1].(map[string]interface{})
	if s1["org"] != "" || s1["branch"] != "" {
		t.Fatalf("non-derived should be empty: %v", s1)
	}
}

func TestCreateSessionDerivesName(t *testing.T) {
	fb := newFakeBackends()
	defer fb.Close()
	h := fb.api(t)

	code, v := do(t, h, "POST", "/sessions", `{"org":"acme","repo":"api","branch":"feat"}`)
	if code != 200 {
		t.Fatalf("code=%d %v", code, v)
	}
	s := v["session"].(map[string]interface{})
	if s["id"] != "acme:api:feat" {
		t.Fatalf("id=%v", s["id"])
	}
}

func TestForkSessionRequiresDerivedName(t *testing.T) {
	fb := newFakeBackends()
	defer fb.Close()
	h := fb.api(t)

	code, _ := do(t, h, "POST", "/sessions/hi/fork", `{"branch":"x"}`)
	if code != 400 {
		t.Fatalf("non-derived fork should 400, got %d", code)
	}
}

func TestSessionChangesFansOutRepoExtAndJJ(t *testing.T) {
	fb := newFakeBackends()
	defer fb.Close()
	h := fb.api(t)

	code, v := do(t, h, "GET", "/sessions/acme:api:main/changes", "")
	if code != 200 {
		t.Fatalf("code=%d", code)
	}
	changes := v["changes"].([]interface{})
	if len(changes) != 2 {
		t.Fatalf("changes=%d (initial commit should be filtered)", len(changes))
	}
	c0 := changes[0].(map[string]interface{})
	if c0["change_id"] != "c2" {
		t.Fatalf("newest first expected, got %v", c0["change_id"])
	}
}

func TestSessionTodosProxiesMemory(t *testing.T) {
	fb := newFakeBackends()
	defer fb.Close()
	h := fb.api(t)

	_, v := do(t, h, "GET", "/sessions/acme:api:main/todos", "")
	todos := v["todos"].([]interface{})
	if len(todos) != 1 {
		t.Fatalf("todos=%v", v)
	}
}

func TestFsListDepthOne(t *testing.T) {
	fb := newFakeBackends()
	defer fb.Close()
	h := fb.api(t)

	_, v := do(t, h, "GET", "/fs/list?org=acme&repo=api&branch=main&path=&depth=1", "")
	entries := v["entries"].([]interface{})
	// root level: README.md + src/ (collapsed dir)
	if len(entries) != 2 {
		t.Fatalf("entries=%v", v)
	}
	src := entries[0].(map[string]interface{})
	if src["name"] != "src" || src["is_dir"] != true {
		t.Fatalf("dir collapse failed: %v", src)
	}

	_, v = do(t, h, "GET", "/fs/list?org=acme&repo=api&branch=main&path=src&depth=1", "")
	entries = v["entries"].([]interface{})
	if len(entries) != 2 {
		t.Fatalf("src entries=%v", v)
	}
}

func TestFsReadDecodesBase64(t *testing.T) {
	fb := newFakeBackends()
	defer fb.Close()
	h := fb.api(t)

	_, v := do(t, h, "GET", "/fs/read?org=acme&repo=api&branch=main&path=hello.txt", "")
	if v["content"] != "hello world" {
		t.Fatalf("content=%v", v["content"])
	}
}

func TestForkRepoSameRepoOnly(t *testing.T) {
	fb := newFakeBackends()
	defer fb.Close()
	h := fb.api(t)

	code, _ := do(t, h, "POST", "/repos/fork", `{"source_org":"acme","source_repo":"api","target_org":"other","target_repo":"api"}`)
	if code != 400 {
		t.Fatalf("cross-repo fork should 400, got %d", code)
	}

	code, v := do(t, h, "POST", "/repos/fork", `{"source_org":"acme","source_repo":"api","source_branch":"main","target_org":"acme","target_repo":"api","target_branch":"dev2"}`)
	if code != 200 {
		t.Fatalf("same-repo fork code=%d %v", code, v)
	}
	if v["session"].(map[string]interface{})["id"] != "acme:api:dev2" {
		t.Fatalf("session=%v", v)
	}
}

var _ = chi.URLParam
var _ = url.Values{}

func TestInvalidComponentsRejectedFast(t *testing.T) {
	fb := newFakeBackends()
	defer fb.Close()
	h := fb.api(t)

	// (method, path, body, offender field)
	cases := []struct {
		method, path, body string
	}{
		{"POST", "/sessions", `{"org":"a:b","repo":"ok","branch":"main"}`},
		{"POST", "/sessions", `{"org":"ok","repo":"x..y","branch":"main"}`},
		{"POST", "/sessions", `{"org":"ok","repo":"ok","branch":"-lead"}`},
		{"POST", "/sessions", `{"org":"ok","repo":"ok","branch":"name.lock"}`},
		{"POST", "/sessions", `{"org":"ok","repo":"ok","branch":"trailing."}`},
		{"POST", "/sessions/acme:api:main/fork", `{"branch":"bad:name"}`},
		{"POST", "/repos/ensure-org", `{"org":"sp ace"}`},
		{"POST", "/repos/ensure", `{"org":"ok","repo":"a..b"}`},
		{"POST", "/repos/clone", `{"org":"ok","repo":"中文","git_url":"https://x"}`},
		{"POST", "/repos/fork", `{"source_org":"ok","source_repo":"a","source_branch":"main","target_org":"ok","target_repo":"a","target_branch":"x:y"}`},
	}
	for i, c := range cases {
		code, v := do(t, h, c.method, c.path, c.body)
		if code != 400 {
			t.Fatalf("case %d (%s %s): expected 400, got %d %v", i, c.method, c.path, code, v)
		}
		if msg, _ := v["error"].(string); !strings.Contains(msg, "invalid") {
			t.Fatalf("case %d: error should name the offender: %v", i, v)
		}
	}

	// valid components still pass through
	code, _ := do(t, h, "POST", "/sessions", `{"org":"acme","repo":"my.repo","branch":"feat-1.2"}`)
	if code != 200 {
		t.Fatalf("valid dotted components should pass, got %d", code)
	}
}

func TestValidComponentRules(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"main", true}, {"my.repo", true}, {"v1.2", true}, {"feat-x", true},
		{"feat/a", true}, {"feat/a/b", true},
		{"a", true}, {"", false}, {"-lead", false}, {"has:colon", false},
		{"dot..dot", false}, {"trail.", false}, {"name.lock", false},
		{"sp ace", false}, {"/lead", false}, {"trail/", false}, {"a//b", false},
		{strings.Repeat("a", 128), true}, {strings.Repeat("a", 129), false},
	}
	for _, c := range cases {
		if got := naming.ValidComponent(c.in); got != c.want {
			t.Errorf("naming.ValidComponent(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

// TestEscapedSessionIDsForwardDecoded pins the UI's encodeURIComponent'd
// session ids: chi returns the raw %3A segment, so the gateway must decode
// before matching/re-forwarding. Before the fix, timelines failed with a
// double-encoded session-map lookup, todos read a stale key that never
// matched the NATS-written rows, and fork always rejected the name.
func TestEscapedSessionIDsForwardDecoded(t *testing.T) {
	var agentPath, agentQuery, mapSession, todoSession string

	agent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		agentPath = r.URL.EscapedPath()
		agentQuery = r.URL.RawQuery
		if strings.HasSuffix(r.URL.Path, "/messages") {
			_, _ = w.Write([]byte(`{"messages":[
				{"id":"m1","role":"user","content":"hi","tool_name":"","created_at":"2026-01-01"},
				{"id":"m2","role":"assistant","content":"ok","tool_name":"","created_at":"2026-01-01"}
			]}`))
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer agent.Close()

	repoExt := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mapSession = r.URL.Query().Get("session")
		_, _ = w.Write([]byte(`{"org":"acme","repo":"api","bookmark":"main"}`))
	}))
	defer repoExt.Close()

	repo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"commits":[
			{"change_id":"c2","commit_id":"k2","author":"a","timestamp":"t","message":"work"},
			{"change_id":"c1","commit_id":"k1","author":"a","timestamp":"t","message":"initial commit"}
		]}`))
	}))
	defer repo.Close()

	memory := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		todoSession = r.URL.Query().Get("session_id")
		_, _ = w.Write([]byte(`{"todos":[{"id":"1","content":"x"}]}`))
	}))
	defer memory.Close()

	h := Router(&API{Up: &upstream.Upstreams{
		Agent:   upstream.New(agent.URL),
		Repo:    upstream.New(repo.URL),
		RepoExt: upstream.New(repoExt.URL),
		Ops:     upstream.New(repoExt.URL),
		Memory:  upstream.New(memory.URL),
	}})

	// messages: no limit/before provided — the forwarded query must be EMPTY
	// (an explicit `limit=` made the agent coerce NaN and return zero rows),
	// and the id must survive as exactly one level of escaping.
	code, v := do(t, h, "GET", "/sessions/acme%3Aapi%3Amain/messages", "")
	if code != 200 {
		t.Fatalf("messages code=%d", code)
	}
	if n := len(v["messages"].([]interface{})); n != 2 {
		t.Fatalf("messages n=%d (empty-string query bug)", n)
	}
	if agentQuery != "" {
		t.Fatalf("forwarded query %q, want empty (no limit=/before= keys)", agentQuery)
	}
	if dec, err := url.PathUnescape(strings.TrimSuffix(strings.TrimPrefix(agentPath, "/api/v1/sessions/"), "/messages")); err != nil || dec != "acme:api:main" {
		t.Fatalf("agent path %q does not carry the id exactly once", agentPath)
	}

	// changes: repo-extension must receive the DECODED session name once.
	code, v = do(t, h, "GET", "/sessions/acme%3Aapi%3Amain/changes", "")
	if code != 200 {
		t.Fatalf("changes code=%d (%v)", code, v)
	}
	if mapSession != "acme:api:main" {
		t.Fatalf("session-map forwarded %q, want acme:api:main (single decode)", mapSession)
	}
	if n := len(v["changes"].([]interface{})); n != 1 {
		t.Fatalf("changes n=%d", n)
	}

	// todos: memory must be queried with the decoded key that matches the
	// NATS-written rows.
	code, _ = do(t, h, "GET", "/sessions/acme%3Aapi%3Amain/todos", "")
	if code != 200 || todoSession != "acme:api:main" {
		t.Fatalf("todos code=%d session=%q", code, todoSession)
	}

	// fork: parseTriple needs the decoded name; the agent call must carry
	// the id exactly once.
	code, _ = do(t, h, "POST", "/sessions/acme%3Aapi%3Amain/fork", `{"branch":"dev2"}`)
	if code != 200 {
		t.Fatalf("fork with escaped id code=%d (was always 400 before decode)", code)
	}
	if dec, err := url.PathUnescape(strings.TrimSuffix(strings.TrimPrefix(agentPath, "/api/v1/sessions/"), "/fork")); err != nil || dec != "acme:api:main" {
		t.Fatalf("agent fork path %q malformed", agentPath)
	}
}

// TestPackageRoutesAcceptEscapedNames pins scoped package names
// (encodeURIComponent turns '@'/'/' into %40/%2F): matching and forwarding
// must happen on the decoded name.
func TestPackageRoutesAcceptEscapedNames(t *testing.T) {
	var deletedRepo string
	artifact := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/pkgs/system/packages" {
			switch r.Method {
			case http.MethodDelete:
				deletedRepo = r.URL.Query().Get("repo")
				_, _ = w.Write([]byte(`{"ok":true}`))
				return
			default:
				_, _ = w.Write([]byte(`{"packages":[
					{"format":"npm","repository":"@bazel/runfiles","versions":["6.5.0"]}
				]}`))
				return
			}
		}
		http.NotFound(w, r)
	}))
	defer artifact.Close()

	h := Router(&API{Up: &upstream.Upstreams{
		Artifact: upstream.New(artifact.URL),
	}})

	code, v := do(t, h, "GET", "/packages/npm/%40bazel%2Frunfiles/versions", "")
	if code != 200 {
		t.Fatalf("versions code=%d (%v) — scoped name must be decoded before matching", code, v)
	}
	data := v["data"].(map[string]interface{})
	if data["name"] != "@bazel/runfiles" {
		t.Fatalf("name=%v", data["name"])
	}

	code, _ = do(t, h, "DELETE", "/packages/npm/%40bazel%2Frunfiles", "")
	if code != 200 || deletedRepo != "@bazel/runfiles" {
		t.Fatalf("delete code=%d repo=%q", code, deletedRepo)
	}
}

// TestFsDefaultsToConventionalBranch pins the empty-branch default: repos
// that only carry dev/master must resolve to a real bookmark instead of the
// hardcoded "main" (which 404'd every tree/read).
func TestFsDefaultsToConventionalBranch(t *testing.T) {
	repo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/repos" {
			_, _ = w.Write([]byte(`{"orgs":[{"org":"build","repos":[{"repo":"svc","bookmarks":[{"branch":"master"},{"branch":"dev"}]}]}]}`))
			return
		}
		// Whatever branch the gateway asks the tree for, echo it back.
		if strings.Contains(r.URL.Path, "/tree") {
			used := strings.Split(r.URL.Path, "/")[6]
			_, _ = w.Write([]byte(`{"tree":[{"path":"go.mod","type":"blob","size":10}]}`))
			_ = used
			w.Header().Set("X-Used-Branch", used)
			return
		}
		http.NotFound(w, r)
	}))
	defer repo.Close()

	h := Router(&API{Up: &upstream.Upstreams{Repo: upstream.New(repo.URL)}})
	code, v := do(t, h, "GET", "/fs/list?org=build&repo=svc", "")
	if code != 200 {
		t.Fatalf("code=%d (%v)", code, v)
	}
	// The default cache now holds master for build/svc.
	dbMu.Lock()
	cached := dbCache["build/svc"].branch
	dbMu.Unlock()
	if cached != "master" {
		t.Fatalf("default branch = %q, want master (main absent, master before dev)", cached)
	}
}
