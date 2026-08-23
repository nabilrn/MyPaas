package caddy

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestStaticFileHandlersApplyProfileAwareCachingCompressionAndFallback(t *testing.T) {
	raw, err := json.Marshal(staticFileHandlers("/srv/site"))
	if err != nil {
		t.Fatalf("marshal handlers: %v", err)
	}
	body := string(raw)

	for _, want := range []string{
		`"root":"/srv/site"`,
		`"handler":"encode"`,
		`"gzip":{}`,
		`"zstd":{}`,
		`"/_next/static/*"`,
		`"/_astro/*"`,
		`"/_nuxt/*"`,
		`"/_app/immutable/*"`,
		`"/assets/*-*.*"`,
		`"Cache-Control":["public, max-age=31536000, immutable"]`,
		`"Cache-Control":["public, max-age=0, must-revalidate"]`,
		`"path":["/.mypaas-*"]`,
		`"handler":"static_response","status_code":404`,
		`"file":{"try_files":["{http.request.uri.path}"]}`,
		`"try_files":["{http.request.uri.path}","{http.request.uri.path}/index.html"]`,
		`"path":["/_next/static/*","/_astro/*","/_nuxt/*","/_app/*","/assets/*","/static/*"]`,
		`"try_files":["/.mypaas-spa-fallback"]`,
		`"handler":"rewrite","uri":"/index.html"`,
		`"handler":"file_server"`,
		`"precompressed":{"br":{},"gzip":{}}`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("handlers do not contain %s: %s", want, body)
		}
	}

	for _, unsafe := range []string{
		`"/*.js"`,
		`"/*.css"`,
		`"/*.svg"`,
		`"try_files":["{http.request.uri.path}","{http.request.uri.path}/index.html","/index.html"]`,
	} {
		if strings.Contains(body, unsafe) {
			t.Fatalf("handlers contain unsafe/global behavior %s: %s", unsafe, body)
		}
	}
}

func TestStaticRewriteRoutesServeFilesBeforeTerminating(t *testing.T) {
	handlers := staticFileHandlers("/srv/site")
	subroute, ok := handlers[3]["routes"].([]map[string]any)
	if !ok {
		t.Fatalf("static subroute routes are missing: %#v", handlers[3])
	}

	for _, index := range []int{2, 4} {
		route := subroute[index]
		if route["terminal"] != true {
			t.Fatalf("route %d must remain terminal", index)
		}
		handle, ok := route["handle"].([]map[string]any)
		if !ok || len(handle) != 2 {
			t.Fatalf("route %d handle = %#v, want rewrite and file_server", index, route["handle"])
		}
		if handle[0]["handler"] != "rewrite" {
			t.Fatalf("route %d first handler = %#v, want rewrite", index, handle[0])
		}
		if handle[1]["handler"] != "file_server" {
			t.Fatalf("route %d second handler = %#v, want file_server", index, handle[1])
		}
	}
}
