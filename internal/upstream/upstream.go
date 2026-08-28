// Package upstream holds thin HTTP clients for the services behind the
// gateway. Each client returns raw decoded JSON plus status; shape mapping
// lives in aggregate.
package upstream

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	Base string
	HC   *http.Client
}

func New(base string) *Client {
	return &Client{Base: base, HC: &http.Client{Timeout: 30 * time.Second}}
}

// Raw performs a request and returns status + body bytes (caller decodes).
func (c *Client) Raw(ctx context.Context, method, path string, body interface{}, query url.Values) (int, []byte, error) {
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		rd = bytes.NewReader(b)
	}
	u := c.Base + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, u, rd)
	if err != nil {
		return 0, nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.HC.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}
	return resp.StatusCode, data, nil
}

// JSON performs a request and decodes the response body into out.
func (c *Client) JSON(ctx context.Context, method, path string, body interface{}, query url.Values, out interface{}) error {
	status, data, err := c.Raw(ctx, method, path, body, query)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	if status >= 400 {
		return fmt.Errorf("%s %s: HTTP %d: %.200s", method, path, status, data)
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(data, out)
}

func Q(kv ...string) url.Values {
	q := url.Values{}
	for i := 0; i+1 < len(kv); i += 2 {
		q.Set(kv[i], kv[i+1])
	}
	return q
}

// Upstreams bundles the service clients.
type Upstreams struct {
	Agent    *Client // agent-ts (sessions, presets, providers, models, config)
	Repo     *Client // jjlab (repos, contents, git inspection)
	RepoExt  *Client // repo-extension (session-map, workspace ops surface)
	Ops      *Client // ops-extension (containers, sandbox, build, infra)
	Artifact *Client // artifact (packages, OCI /v2)
	Browser  *Client // browser-extension
	Memory   *Client // memory-tools (todos)
}

func FromEnv(env func(string) string) *Upstreams {
	// or returns the first non-empty env hit; the LAST argument is the
	// fallback when no env var is set.
	or := func(keys ...string) string {
		for _, k := range keys[:len(keys)-1] {
			if v := env(k); v != "" {
				return v
			}
		}
		return keys[len(keys)-1]
	}
	return &Upstreams{
		// Prefer the gateway-specific names; fall back to the chart's
		// ZERGX_*_URL service map (external-secret).
		Agent:    New(or("AGENT_URL", "ZERGX_AGENT_URL", "http://agent.zergx.svc.cluster.local:80")),
		Repo:     New(or("REPO_URL", "ZERGX_REPO_MANAGER_URL", "http://jjlab.zergx.svc.cluster.local:80")),
		RepoExt:  New(or("REPOEXT_URL", "ZERGX_REPOEXT_URL", "http://repo-extension.zergx.svc.cluster.local:80")),
		Ops:      New(or("OPS_URL", "ZERGX_EXECUTOR_URL", "http://ops-extension.zergx.svc.cluster.local:80")),
		Artifact: New(or("ARTIFACT_URL", "ZERGX_ZOT_URL", "ZERGX_REGISTRY_URL", "http://artifact.zergx.svc.cluster.local")),
		Browser:  New(or("BROWSER_URL", "ZERGX_BROWSER_URL", "http://browser-extension.zergx.svc.cluster.local:80")),
		Memory:   New(or("MEMORY_URL", "ZERGX_MEMORY_URL", "http://memory-tools.zergx.svc.cluster.local:80")),
	}
}
