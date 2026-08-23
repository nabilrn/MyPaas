package caddy

import "testing"

func TestReverseProxyHandlersManyDeduplicatesUpstreams(t *testing.T) {
	handlers := reverseProxyHandlersMany([]string{"app-a:8080", "app-b:8080", "app-a:8080", ""})
	if len(handlers) != 2 {
		t.Fatalf("handlers = %d, want 2", len(handlers))
	}
	proxy := handlers[1]
	upstreams, ok := proxy["upstreams"].([]map[string]any)
	if !ok {
		t.Fatalf("upstreams = %#v", proxy["upstreams"])
	}
	if len(upstreams) != 2 {
		t.Fatalf("upstreams = %d, want 2", len(upstreams))
	}
	if upstreams[0]["dial"] != "app-a:8080" || upstreams[1]["dial"] != "app-b:8080" {
		t.Fatalf("unexpected upstreams: %#v", upstreams)
	}
	lb := proxy["load_balancing"].(map[string]any)
	policy := lb["selection_policy"].(map[string]any)
	if policy["policy"] != "round_robin" {
		t.Fatalf("selection policy = %#v", policy)
	}
}
