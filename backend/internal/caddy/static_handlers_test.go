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
		`"try_files":["{http.request.uri.path}","{http.request.uri.path}/index.html"]`,
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
		`"/static/*"`,
		`"/assets/*"`,
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
