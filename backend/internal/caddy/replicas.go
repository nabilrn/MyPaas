package caddy

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type ReplicaUpstream struct {
	Dial string
}

func reverseProxyHandlersMany(dials []string) []map[string]any {
	upstreams := make([]map[string]any, 0, len(dials))
	seen := make(map[string]struct{}, len(dials))
	for _, dial := range dials {
		dial = strings.TrimSpace(dial)
		if dial == "" {
			continue
		}
		if _, ok := seen[dial]; ok {
			continue
		}
		seen[dial] = struct{}{}
		upstreams = append(upstreams, map[string]any{"dial": dial})
	}
	return []map[string]any{
		{
			"handler": "encode",
			"encodings": map[string]any{
				"gzip": map[string]any{},
				"zstd": map[string]any{},
			},
		},
		{
			"handler":   "reverse_proxy",
			"upstreams": upstreams,
			"load_balancing": map[string]any{
				"selection_policy": map[string]any{"policy": "round_robin"},
				"try_duration":     "10s",
				"try_interval":     "250ms",
			},
		},
	}
}

func (c *Client) AddReplicaRoute(ctx context.Context, host string, primaryPort int32, replicas []ReplicaUpstream) error {
	primaryDial, err := c.upstreamDial(ctx, primaryPort)
	if err != nil {
		return err
	}
	dials := []string{primaryDial}
	for _, replica := range replicas {
		if strings.TrimSpace(replica.Dial) != "" {
			dials = append(dials, replica.Dial)
		}
	}
	if len(dials) == 0 {
		return fmt.Errorf("replica route requires at least one upstream")
	}
	route, err := json.Marshal(map[string]any{
		"match":    []map[string]any{{"host": []string{host}}},
		"handle":   reverseProxyHandlersMany(dials),
		"terminal": true,
	})
	if err != nil {
		return err
	}
	return c.replaceHostRoute(ctx, host, route)
}

func (c *Client) AddHybridReplicaRoute(ctx context.Context, host, root string, primaryPort int32, replicas []ReplicaUpstream) error {
	primaryDial, err := c.upstreamDial(ctx, primaryPort)
	if err != nil {
		return err
	}
	dials := []string{primaryDial}
	for _, replica := range replicas {
		if strings.TrimSpace(replica.Dial) != "" {
			dials = append(dials, replica.Dial)
		}
	}
	route, err := json.Marshal(map[string]any{
		"match": []map[string]any{{"host": []string{host}}},
		"handle": []map[string]any{
			{
				"handler": "subroute",
				"routes": []map[string]any{
					{
						"match":  []map[string]any{{"path": []string{"/api/*"}}},
						"handle": reverseProxyHandlersMany(dials),
					},
					{"handle": staticFileHandlers(root)},
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
