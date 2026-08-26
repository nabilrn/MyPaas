package caddy

import (
	"strings"
	"testing"
)

func TestComposeRouteAliasIsDeterministicAndBounded(t *testing.T) {
	first := composeRouteAlias("minio-project", "console")
	second := composeRouteAlias("minio-project", "console")
	if first != second {
		t.Fatalf("alias must be deterministic: %q != %q", first, second)
	}
	if !strings.HasPrefix(first, "mypaas-http-console-") {
		t.Fatalf("unexpected alias prefix: %q", first)
	}
	if len(first) > 63 {
		t.Fatalf("alias exceeds DNS label limit: %d", len(first))
	}
	other := composeRouteAlias("minio-project", "metrics")
	if other == first {
		t.Fatalf("different routes must not share aliases: %q", first)
	}
}
