package caddy

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAddRoutePostsNewRoute(t *testing.T) {
	const routesPath = "/config/apps/http/servers/srv0/routes"

	var postedMethod string
	var postedBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == routesPath:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"match":[{"host":["old.localhost"]}],"handle":[],"terminal":true}]`))
		case r.Method == http.MethodPost && r.URL.Path == routesPath:
			postedMethod = r.Method
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

	client := NewClient(server.URL, "host.docker.internal")
	if err := client.AddRoute(context.Background(), "new.localhost", 3456); err != nil {
		t.Fatalf("AddRoute returned error: %v", err)
	}

	if postedMethod != http.MethodPost {
		t.Fatalf("method = %q, want POST", postedMethod)
	}
	if !strings.Contains(postedBody, `"new.localhost"`) {
		t.Fatalf("posted body does not contain new host: %s", postedBody)
	}
	if !strings.Contains(postedBody, `"host.docker.internal:3456"`) {
		t.Fatalf("posted body does not contain configured upstream: %s", postedBody)
	}
}

func TestAddRoutePatchesExistingRouteByIndex(t *testing.T) {
	const routesPath = "/config/apps/http/servers/srv0/routes"

	var patchedPath string
	var patchedBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == routesPath:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"match":[{"host":["old.localhost"]}],"handle":[],"terminal":true},{"match":[{"host":["app.localhost"]}],"handle":[],"terminal":true}]`))
		case r.Method == http.MethodPatch && r.URL.Path == routesPath+"/1":
			patchedPath = r.URL.Path
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read patch body: %v", err)
			}
			patchedBody = string(body)
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected caddy request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "host.docker.internal")
	if err := client.AddRoute(context.Background(), "app.localhost", 3456); err != nil {
		t.Fatalf("AddRoute returned error: %v", err)
	}

	if patchedPath != routesPath+"/1" {
		t.Fatalf("patched path = %q, want %q", patchedPath, routesPath+"/1")
	}
	if !strings.Contains(patchedBody, `"app.localhost"`) {
		t.Fatalf("patched body does not contain host: %s", patchedBody)
	}
}

func TestAddFileServerRoutePatchesStaticRoot(t *testing.T) {
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

	client := NewClient(server.URL, "host.docker.internal")
	if err := client.AddFileServerRoute(context.Background(), "static.localhost", "/var/lib/mypaas/static/project-id"); err != nil {
		t.Fatalf("AddFileServerRoute returned error: %v", err)
	}

	for _, want := range []string{`"static.localhost"`, `"/var/lib/mypaas/static/project-id"`, `"handler":"file_server"`} {
		if !strings.Contains(postedBody, want) {
			t.Fatalf("posted body does not contain %s: %s", want, postedBody)
		}
	}
}
