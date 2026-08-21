package caddy

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
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

func TestRuntimeProxyHandlersSeparateStaticDeliveryFromAPI(t *testing.T) {
	handlers := runtimeProxyHandlers("project-gateway:3000")
	if len(handlers) != 1 {
		t.Fatalf("top-level handler count = %d, want 1", len(handlers))
	}
	if got := handlers[0]["handler"]; got != "subroute" {
		t.Fatalf("top-level handler = %v, want subroute", got)
	}

	routes, ok := handlers[0]["routes"].([]map[string]any)
	if !ok {
		t.Fatalf("routes have unexpected type %T", handlers[0]["routes"])
	}
	if len(routes) != 3 {
		t.Fatalf("route count = %d, want 3", len(routes))
	}

	apiMatch, ok := routes[0]["match"].([]map[string]any)
	if !ok || len(apiMatch) != 1 {
		t.Fatalf("API match has unexpected shape: %#v", routes[0]["match"])
	}
	apiPaths, ok := apiMatch[0]["path"].([]string)
	if !ok || len(apiPaths) != 1 || apiPaths[0] != "/api/*" {
		t.Fatalf("API path matcher = %#v, want [/api/*]", apiMatch[0]["path"])
	}
	apiHandle, ok := routes[0]["handle"].([]map[string]any)
	if !ok || len(apiHandle) != 1 {
		t.Fatalf("API handle has unexpected shape: %#v", routes[0]["handle"])
	}
	if got := apiHandle[0]["handler"]; got != "reverse_proxy" {
		t.Fatalf("API handler = %v, want reverse_proxy only", got)
	}
	if _, exists := apiHandle[0]["response"]; exists {
		t.Fatalf("API reverse proxy unexpectedly contains response-header policy: %#v", apiHandle[0])
	}
	if _, exists := apiHandle[0]["encodings"]; exists {
		t.Fatalf("API reverse proxy unexpectedly contains encoding policy: %#v", apiHandle[0])
	}

	staticMatch, ok := routes[1]["match"].([]map[string]any)
	if !ok || len(staticMatch) != 1 {
		t.Fatalf("static match has unexpected shape: %#v", routes[1]["match"])
	}
	staticPaths, ok := staticMatch[0]["path"].([]string)
	if !ok || len(staticPaths) != 1 || staticPaths[0] != "/_next/static/*" {
		t.Fatalf("static path matcher = %#v, want [/_next/static/*]", staticMatch[0]["path"])
	}
	staticHandle, ok := routes[1]["handle"].([]map[string]any)
	if !ok || len(staticHandle) != 3 {
		t.Fatalf("static handle has unexpected shape: %#v", routes[1]["handle"])
	}
	if got := staticHandle[0]["handler"]; got != "headers" {
		t.Fatalf("static first handler = %v, want headers", got)
	}
	response, ok := staticHandle[0]["response"].(map[string]any)
	if !ok {
		t.Fatalf("static header response has unexpected shape: %#v", staticHandle[0]["response"])
	}
	if deferred, ok := response["deferred"].(bool); !ok || !deferred {
		t.Fatalf("static header policy deferred = %#v, want true", response["deferred"])
	}
	set, ok := response["set"].(map[string][]string)
	if !ok {
		t.Fatalf("static header set has unexpected shape: %#v", response["set"])
	}
	cacheControl := set["Cache-Control"]
	if len(cacheControl) != 1 || cacheControl[0] != immutableAssetCacheControl {
		t.Fatalf("static Cache-Control = %#v, want %q", cacheControl, immutableAssetCacheControl)
	}
	if got := staticHandle[1]["handler"]; got != "encode" {
		t.Fatalf("static second handler = %v, want encode", got)
	}
	encodings, ok := staticHandle[1]["encodings"].(map[string]any)
	if !ok {
		t.Fatalf("static encodings have unexpected shape: %#v", staticHandle[1]["encodings"])
	}
	if _, ok := encodings["gzip"]; !ok {
		t.Fatalf("static encodings do not include gzip: %#v", encodings)
	}
	if got := staticHandle[2]["handler"]; got != "reverse_proxy" {
		t.Fatalf("static third handler = %v, want reverse_proxy", got)
	}

	fallbackHandle, ok := routes[2]["handle"].([]map[string]any)
	if !ok || len(fallbackHandle) != 2 {
		t.Fatalf("fallback handle has unexpected shape: %#v", routes[2]["handle"])
	}
	if got := fallbackHandle[0]["handler"]; got != "encode" {
		t.Fatalf("fallback first handler = %v, want encode", got)
	}
	if got := fallbackHandle[1]["handler"]; got != "reverse_proxy" {
		t.Fatalf("fallback second handler = %v, want reverse_proxy", got)
	}
	if _, exists := fallbackHandle[0]["response"]; exists {
		t.Fatalf("fallback unexpectedly contains cache header policy: %#v", fallbackHandle[0])
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

func TestClientUsesUnixAdminSocket(t *testing.T) {
	const routesPath = "/config/apps/http/servers/srv0/routes"
	socketPath := filepath.Join(t.TempDir(), "caddy-admin.sock")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("listen unix socket: %v", err)
	}

	var postedBody string
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == routesPath:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[]`))
		case r.Method == http.MethodPost && r.URL.Path == routesPath:
			body, readErr := io.ReadAll(r.Body)
			if readErr != nil {
				t.Fatalf("read unix post body: %v", readErr)
			}
			postedBody = string(body)
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected unix caddy request: %s %s", r.Method, r.URL.Path)
		}
	})}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })

	client := NewClient("unix/"+socketPath, "project-gateway")
	if err := client.AddRoute(context.Background(), "unix.localhost", 4567); err != nil {
		t.Fatalf("AddRoute over Unix socket returned error: %v", err)
	}
	if !strings.Contains(postedBody, `"project-gateway:4567"`) {
		t.Fatalf("Unix admin request did not preserve upstream host: %s", postedBody)
	}
}
