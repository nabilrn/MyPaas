package caddy

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAddRoutePatchesExistingRoutes(t *testing.T) {
	const routesPath = "/config/apps/http/servers/srv0/routes"

	var patchedMethod string
	var patchedBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == routesPath:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"match":[{"host":["old.localhost"]}],"handle":[],"terminal":true}]`))
		case r.Method == http.MethodPatch && r.URL.Path == routesPath:
			patchedMethod = r.Method
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
	if err := client.AddRoute(context.Background(), "new.localhost", 3456); err != nil {
		t.Fatalf("AddRoute returned error: %v", err)
	}

	if patchedMethod != http.MethodPatch {
		t.Fatalf("method = %q, want PATCH", patchedMethod)
	}
	if !strings.Contains(patchedBody, `"new.localhost"`) {
		t.Fatalf("patched body does not contain new host: %s", patchedBody)
	}
	if !strings.Contains(patchedBody, `"old.localhost"`) {
		t.Fatalf("patched body does not preserve existing routes: %s", patchedBody)
	}
	if !strings.Contains(patchedBody, `"host.docker.internal:3456"`) {
		t.Fatalf("patched body does not contain configured upstream: %s", patchedBody)
	}
}
