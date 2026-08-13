package caddy

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAddFileServerRouteAddsSPAFallback(t *testing.T) {
	const routesPath = "/config/apps/http/servers/srv0/routes"

	var postedBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == routesPath:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[]`))
		case r.Method == http.MethodPost && r.URL.Path == routesPath:
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read post body: %v", err)
			}
			postedBody = string(body)
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected caddy request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "runtime")
	if err := client.AddFileServerRoute(context.Background(), "static.localhost", "/var/lib/mypaas/static/project-id"); err != nil {
		t.Fatalf("AddFileServerRoute returned error: %v", err)
	}

	for _, want := range []string{
		`"root":"/var/lib/mypaas/static/project-id"`,
		`"try_files":["{http.request.uri.path}","{http.request.uri.path}/","/index.html"]`,
		`"handler":"rewrite","uri":"{http.matchers.file.relative}"`,
		`"handler":"file_server"`,
	} {
		if !strings.Contains(postedBody, want) {
			t.Fatalf("posted body does not contain %s: %s", want, postedBody)
		}
	}
}

func TestAddHybridRouteKeepsAPIProxyAheadOfSPAFallback(t *testing.T) {
	const routesPath = "/config/apps/http/servers/srv0/routes"

	var postedBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == routesPath:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[]`))
		case r.Method == http.MethodPost && r.URL.Path == routesPath:
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read post body: %v", err)
			}
			postedBody = string(body)
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected caddy request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "runtime")
	if err := client.AddHybridRoute(context.Background(), "hybrid.localhost", "/var/lib/mypaas/static/project-id", 4567); err != nil {
		t.Fatalf("AddHybridRoute returned error: %v", err)
	}

	apiIndex := strings.Index(postedBody, `"path":["/api/*"]`)
	fallbackIndex := strings.Index(postedBody, `"try_files":["{http.request.uri.path}","{http.request.uri.path}/","/index.html"]`)
	if apiIndex < 0 || fallbackIndex < 0 {
		t.Fatalf("hybrid route is missing API proxy or SPA fallback: %s", postedBody)
	}
	if apiIndex >= fallbackIndex {
		t.Fatalf("SPA fallback appears before API proxy: %s", postedBody)
	}
	if !strings.Contains(postedBody, `"runtime:4567"`) {
		t.Fatalf("hybrid route does not contain configured API upstream: %s", postedBody)
	}
}
