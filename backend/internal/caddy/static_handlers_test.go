package caddy

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestStaticFileHandlersIncludeHistoryFallback(t *testing.T) {
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
		`"Cache-Control":["public, max-age=31536000, immutable"]`,
		`"Cache-Control":["no-cache"]`,
		`"try_files":["{http.request.uri.path}","{http.request.uri.path}/index.html","/index.html"]`,
		`"handler":"rewrite","uri":"{http.matchers.file.relative}"`,
		`"handler":"file_server"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("handlers do not contain %s: %s", want, body)
		}
	}
}
