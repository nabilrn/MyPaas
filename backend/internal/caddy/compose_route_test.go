package caddy

import (
	"strings"
	"testing"
)

func TestComposeRouteAliasIsDeterministicAndBounded(t *testing.T) {
	first := composeRouteAlias("minio-project", "minio")
	second := composeRouteAlias("minio-project", "minio")
	if first != second {
		t.Fatalf("alias must be deterministic: %q != %q", first, second)
	}
	if !strings.HasPrefix(first, "mypaas-http-") {
		t.Fatalf("unexpected alias prefix: %q", first)
	}
	if len(first) > 63 {
		t.Fatalf("alias exceeds DNS label limit: %d", len(first))
	}
	other := composeRouteAlias("minio-project", "console-sidecar")
	if other == first {
		t.Fatalf("different services must not share aliases: %q", first)
	}
}
