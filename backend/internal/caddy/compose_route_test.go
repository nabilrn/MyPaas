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

func TestExistingManagedRoutingAliasReusesPrimaryOrAdditionalAlias(t *testing.T) {
	cases := []struct {
		name    string
		aliases []string
		want    string
	}{
		{name: "primary route alias", aliases: []string{"mypaas-port-31001", "container-name"}, want: "mypaas-port-31001"},
		{name: "additional route alias", aliases: []string{"service", "mypaas-http-a1b2c3"}, want: "mypaas-http-a1b2c3"},
		{name: "no managed alias", aliases: []string{"service", "container-name"}, want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := existingManagedRoutingAlias(tc.aliases); got != tc.want {
				t.Fatalf("existingManagedRoutingAlias() = %q, want %q", got, tc.want)
			}
		})
	}
}
