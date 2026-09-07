package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMachineTokenAllows(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		scopes []string
		want   bool
	}{
		{"auth probe", http.MethodGet, "/api/auth/me", nil, true},
		{"project read", http.MethodGet, "/api/projects/abc", []string{"projects:read"}, true},
		{"project read denied", http.MethodGet, "/api/projects/abc", []string{"projects:write"}, false},
		{"project update", http.MethodPatch, "/api/projects/abc", []string{"projects:write"}, true},
		{"project delete denied", http.MethodDelete, "/api/projects/abc", []string{"projects:write"}, false},
		{"deploy", http.MethodPost, "/api/projects/abc/deploy", []string{"deployments:write"}, true},
		{"logs", http.MethodGet, "/api/projects/abc/logs", []string{"logs:read"}, true},
		{"metrics", http.MethodGet, "/api/projects/abc/metrics", []string{"metrics:read"}, true},
		{"env list", http.MethodGet, "/api/projects/abc/env", []string{"env:read"}, true},
		{"env set", http.MethodPut, "/api/projects/abc/env", []string{"env:write"}, true},
		{"env delete", http.MethodDelete, "/api/projects/abc/env/API_KEY", []string{"env:write"}, true},
		{"secret reveal denied", http.MethodGet, "/api/projects/abc/env/API_KEY/reveal", []string{"env:read", "env:write"}, false},
		{"db studio denied", http.MethodGet, "/api/projects/abc/db/tables", []string{"projects:read", "admin:read"}, false},
		{"routes denied", http.MethodPut, "/api/projects/abc/routes", []string{"projects:write"}, false},
		{"webhook denied", http.MethodPost, "/api/projects/abc/webhook-secret/regenerate", []string{"projects:write"}, false},
		{"analytics denied", http.MethodGet, "/api/projects/abc/analytics", []string{"metrics:read"}, false},
		{"host stats without admin", http.MethodGet, "/api/admin/host-stats", []string{"projects:read"}, false},
		{"host stats with admin", http.MethodGet, "/api/admin/host-stats", []string{"admin:read"}, true},
		{"unknown route fails closed", http.MethodGet, "/api/admin/new-powerful-feature", []string{"admin:read", "projects:read", "projects:write"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, tt.path, nil)
			if got := machineTokenAllows(r, tt.scopes); got != tt.want {
				t.Fatalf("machineTokenAllows(%s %s, %v) = %v, want %v", tt.method, tt.path, tt.scopes, got, tt.want)
			}
		})
	}
}
