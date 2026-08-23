package aggregate

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"forgejo.develop.10.199.64.20.nip.io/rucoder/gateway-go/internal/upstream"
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
	mux.HandleFunc("/api/v1/repos/bookmark-from", func(w http.ResponseWriter, r *http.Request) {
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
	mux.HandleFunc("/api/v1/repos/acme/api/main/log", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"commits":[
			{"change_id":"c2","commit_id":"k2","author":"rucoder","timestamp":"2026-01-02","message":"edit hello"},
			{"change_id":"c1","commit_id":"k1","author":"rucoder","timestamp":"2026-01-01","message":"write hello"},
			{"change_id":"c0","commit_id":"k0","author":"rucoder","timestamp":"2026-01-01","message":"initial commit"}
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

func TestContainersAdapted(t *testing.T) {
	fb := newFakeBackends()
	defer fb.Close()
	h := fb.api(t)

	_, v := do(t, h, "GET", "/containers", "")
	cs := v["containers"].([]interface{})
	c := cs[0].(map[string]interface{})
	if c["id"] != "cid-1" || c["name"] != "pod-1" || c["kind"] != "worker" {
		t.Fatalf("container=%v", c)
	}

	code, v := do(t, h, "POST", "/containers", `{"image":"x"}`)
	if code != 200 {
		t.Fatalf("create code=%d", code)
	}
	c2 := v["container"].(map[string]interface{})
	if c2["kind"] != "worker" || c2["service_url"] != nil {
		t.Fatalf("create defaults missing: %v", c2)
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
