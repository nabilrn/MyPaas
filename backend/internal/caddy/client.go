package caddy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(adminAddress string) *Client {
	adminAddress = strings.TrimPrefix(adminAddress, "http://")
	return &Client{
		baseURL: "http://" + adminAddress,
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
				"dial": fmt.Sprintf("127.0.0.1:%d", port),
			}},
		}},
		"terminal": true,
	})
	if err != nil {
		return err
	}

	routes, err := c.routes(ctx)
	if err != nil {
		return err
	}
	next := []json.RawMessage{route}
	for _, existing := range routes {
		if !routeMatchesHost(existing, host) {
			next = append(next, existing)
		}
	}
	return c.putJSON(ctx, "/config/apps/http/servers/srv0/routes", next)
}

func (c *Client) RemoveRoute(ctx context.Context, host string) error {
	routes, err := c.routes(ctx)
	if err != nil {
		return err
	}

	filtered := make([]json.RawMessage, 0, len(routes))
	for _, route := range routes {
		if !routeMatchesHost(route, host) {
			filtered = append(filtered, route)
		}
	}
	return c.putJSON(ctx, "/config/apps/http/servers/srv0/routes", filtered)
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

func (c *Client) putJSON(ctx context.Context, path string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("caddy put config: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("caddy put config returned %s", resp.Status)
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
