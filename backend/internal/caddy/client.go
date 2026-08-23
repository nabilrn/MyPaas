package caddy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

const spaFallbackMarker = "/.mypaas-spa-fallback"

type Client struct {
	baseURL      string
	upstreamHost string
	http         *http.Client
}

func NewClient(adminAddress, upstreamHost string) *Client {
	adminAddress = strings.TrimSpace(adminAddress)
	if upstreamHost == "" {
		upstreamHost = "127.0.0.1"
	}

	httpClient := &http.Client{Timeout: 10 * time.Second}
	baseURL := ""
	if strings.HasPrefix(adminAddress, "unix/") {
		socketPath := strings.TrimPrefix(adminAddress, "unix/")
		transport := &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				dialer := net.Dialer{}
				return dialer.DialContext(ctx, "unix", socketPath)
			},
		}
		httpClient.Transport = transport
		// HTTP still frames requests over the Unix stream; this host is only a
		// local request URL placeholder and is never resolved on the network.
		baseURL = "http://caddy-admin"
	} else {
		adminAddress = strings.TrimPrefix(adminAddress, "http://")
		baseURL = "http://" + adminAddress
	}

	return &Client{
		baseURL:      baseURL,
		upstreamHost: upstreamHost,
		http:         httpClient,
	}
}

func reverseProxyHandlers(dial string) []map[string]any {
	return []map[string]any{
		{
			"handler": "encode",
			"encodings": map[string]any{
				"gzip": map[string]any{},
				"zstd": map[string]any{},
			},
		},
		{
			"handler": "reverse_proxy",
			"upstreams": []map[string]any{{
				"dial": dial,
			}},
			"load_balancing": map[string]any{
				"try_duration": "10s",
				"try_interval": "250ms",
			},
		},
	}
}

func (c *Client) AddRoute(ctx context.Context, host string, port int32) error {
	dial, err := c.upstreamDial(ctx, port)
	if err != nil {
		return err
	}
	route, err := json.Marshal(map[string]any{
		"match":    []map[string]any{{"host": []string{host}}},
		"handle":   reverseProxyHandlers(dial),
		"terminal": true,
	})
	if err != nil {
		return err
	}
	return c.replaceHostRoute(ctx, host, route)
}

func staticFileHandlers(root string) []map[string]any {
	assetNamespaces := []string{
		"/_next/static/*",
		"/_astro/*",
		"/_nuxt/*",
		"/_app/*",
		"/assets/*",
		"/static/*",
	}
	immutablePaths := []string{
		"/_next/static/*",
		"/_astro/*",
		"/_nuxt/*",
		"/_app/immutable/*",
		"/assets/*-*.*",
	}

	return []map[string]any{
		{
			"handler": "vars",
			"root":    root,
		},
		{
			// Static responses are revalidated by default. Only framework-owned,
			// fingerprinted asset namespaces below receive an immutable lifetime.
			"handler": "headers",
			"response": map[string]any{
				"set": map[string][]string{
					"Cache-Control": {"public, max-age=0, must-revalidate"},
				},
			},
		},
		{
			"handler": "encode",
			"encodings": map[string]any{
				"gzip": map[string]any{},
				"zstd": map[string]any{},
			},
		},
		{
			"handler": "subroute",
			"routes": []map[string]any{
				{
					// Deployment metadata must never become public site content.
					"match": []map[string]any{{"path": []string{"/.mypaas-*"}}},
					"handle": []map[string]any{{
						"handler":     "static_response",
						"status_code": 404,
					}},
					"terminal": true,
				},
				{
					// Apply immutable caching only when the requested fingerprinted asset
					// actually exists. A missing hashed asset must not turn into a cached
					// SPA shell or immutable 404.
					"match": []map[string]any{{
						"path": immutablePaths,
						"file": map[string]any{
							"try_files": []string{"{http.request.uri.path}"},
						},
					}},
					"handle": []map[string]any{{
						"handler": "headers",
						"response": map[string]any{
							"set": map[string][]string{
								"Cache-Control": {"public, max-age=31536000, immutable"},
							},
						},
					}},
				},
				{
					// Real files and directory indexes win before any SPA history
					// fallback. Mark this route terminal so a deployed SPA marker cannot
					// overwrite an already resolved asset/page with index.html.
					"match": []map[string]any{{
						"file": map[string]any{
							"try_files": []string{
								"{http.request.uri.path}",
								"{http.request.uri.path}/index.html",
							},
						},
					}},
					"handle": []map[string]any{{
						"handler": "rewrite",
						"uri":     "{http.matchers.file.relative}",
					}},
					"terminal": true,
				},
				{
					// Missing files in known static namespaces are always genuine misses;
					// never satisfy them with an SPA shell.
					"match": []map[string]any{{"path": assetNamespaces}},
					"handle": []map[string]any{{
						"handler":     "static_response",
						"status_code": 404,
					}},
					"terminal": true,
				},
				{
					// CopyDir writes the marker only for recognized client-rendered SPA
					// builds. Astro/prerendered/static HTML releases have no marker and
					// therefore retain ordinary 404 behavior for unknown routes.
					"match": []map[string]any{{
						"file": map[string]any{
							"try_files": []string{spaFallbackMarker},
						},
					}},
					"handle": []map[string]any{{
						"handler": "rewrite",
						"uri":     "/index.html",
					}},
					"terminal": true,
				},
			},
		},
		{
			"handler":     "file_server",
			"index_names": []string{"index.html"},
			"precompressed": map[string]any{
				"br":   map[string]any{},
				"gzip": map[string]any{},
			},
		},
	}
}

func (c *Client) AddFileServerRoute(ctx context.Context, host, root string) error {
	route, err := json.Marshal(map[string]any{
		"match":    []map[string]any{{"host": []string{host}}},
		"handle":   staticFileHandlers(root),
		"terminal": true,
	})
	if err != nil {
		return err
	}
	return c.replaceHostRoute(ctx, host, route)
}

func (c *Client) AddHybridRoute(ctx context.Context, host, root string, port int32) error {
	dial, err := c.upstreamDial(ctx, port)
	if err != nil {
		return err
	}
	route, err := json.Marshal(map[string]any{
		"match": []map[string]any{{"host": []string{host}}},
		"handle": []map[string]any{
			{
				"handler": "subroute",
				"routes": []map[string]any{
					{
						"match":  []map[string]any{{"path": []string{"/api/*"}}},
						"handle": reverseProxyHandlers(dial),
					},
					{
						"handle": staticFileHandlers(root),
					},
				},
			},
		},
		"terminal": true,
	})
	if err != nil {
		return err
	}
	return c.replaceHostRoute(ctx, host, route)
}

func (c *Client) replaceHostRoute(ctx context.Context, host string, route json.RawMessage) error {
	routes, err := c.routes(ctx)
	if err != nil {
		return err
	}

	for idx, existing := range routes {
		if routeMatchesHost(existing, host) {
			return c.patchJSON(ctx, fmt.Sprintf("/config/apps/http/servers/srv0/routes/%d", idx), route)
		}
	}

	return c.postJSON(ctx, "/config/apps/http/servers/srv0/routes", route)
}

func (c *Client) RemoveRoute(ctx context.Context, host string) error {
	routes, err := c.routes(ctx)
	if err != nil {
		return err
	}

	for idx := len(routes) - 1; idx >= 0; idx-- {
		if routeMatchesHost(routes[idx], host) {
			if err := c.delete(ctx, fmt.Sprintf("/config/apps/http/servers/srv0/routes/%d", idx)); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Client) routes(ctx context.Context) ([]json.RawMessage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/config/apps/http/servers/srv0/routes", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("caddy get routes: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("caddy get routes returned %s", resp.Status)
	}

	var routes []json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&routes); err != nil {
		return nil, fmt.Errorf("decode caddy routes: %w", err)
	}
	return routes, nil
}

func (c *Client) patchJSON(ctx context.Context, path string, payload any) error {
	return c.sendJSON(ctx, http.MethodPatch, path, payload)
}

func (c *Client) postJSON(ctx context.Context, path string, payload any) error {
	return c.sendJSON(ctx, http.MethodPost, path, payload)
}

func (c *Client) sendJSON(ctx context.Context, method, path string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("caddy patch config: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		detail := strings.TrimSpace(string(respBody))
		if detail == "" {
			return fmt.Errorf("caddy patch config returned %s", resp.Status)
		}
		return fmt.Errorf("caddy patch config returned %s: %s", resp.Status, detail)
	}
	return nil
}

func (c *Client) delete(ctx context.Context, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("caddy delete config: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		detail := strings.TrimSpace(string(respBody))
		if detail == "" {
			return fmt.Errorf("caddy delete config returned %s", resp.Status)
		}
		return fmt.Errorf("caddy delete config returned %s: %s", resp.Status, detail)
	}
	return nil
}

func routeMatchesHost(raw json.RawMessage, host string) bool {
	var route struct {
		Match []struct {
			Host []string `json:"host"`
		} `json:"match"`
	}
	if err := json.Unmarshal(raw, &route); err != nil {
		return false
	}
	for _, matcher := range route.Match {
		for _, item := range matcher.Host {
			if item == host {
				return true
			}
	}
	}
	return false
}
