package caddy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL      string
	upstreamHost string
	http         *http.Client
}

func NewClient(adminAddress, upstreamHost string) *Client {
	adminAddress = strings.TrimPrefix(adminAddress, "http://")
	if upstreamHost == "" {
		upstreamHost = "127.0.0.1"
	}
	return &Client{
		baseURL:      "http://" + adminAddress,
		upstreamHost: upstreamHost,
		http: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) AddRoute(ctx context.Context, host string, port int32) error {
	route, err := json.Marshal(map[string]any{
		"match": []map[string]any{{"host": []string{host}}},
		"handle": []map[string]any{{
			"handler": "reverse_proxy",
			"upstreams": []map[string]any{{
				"dial": fmt.Sprintf("%s:%d", c.upstreamHost, port),
			}},
		}},
		"terminal": true,
	})
	if err != nil {
		return err
	}
	return c.replaceHostRoute(ctx, host, route)
}

func (c *Client) AddFileServerRoute(ctx context.Context, host, root string) error {
	route, err := json.Marshal(map[string]any{
		"match": []map[string]any{{"host": []string{host}}},
		"handle": []map[string]any{
			{
				"handler":     "file_server",
				"root":        root,
				"index_names": []string{"index.html"},
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
