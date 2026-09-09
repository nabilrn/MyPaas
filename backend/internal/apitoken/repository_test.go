package apitoken

import (
	"reflect"
	"testing"
)

func TestNormalizeScopes(t *testing.T) {
	got, err := NormalizeScopes([]string{"env:write", "projects:read", "env:write", " logs:read "})
	if err != nil {
		t.Fatalf("NormalizeScopes returned error: %v", err)
	}
	want := []string{"env:write", "logs:read", "projects:read"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("NormalizeScopes = %v, want %v", got, want)
	}
}

func TestNormalizeScopesRejectsUnknown(t *testing.T) {
	if _, err := NormalizeScopes([]string{"host:shell"}); err == nil {
		t.Fatal("NormalizeScopes accepted an unsupported privileged scope")
	}
}

func TestDefaultMCPScopesExcludeAdmin(t *testing.T) {
	if HasScope(DefaultMCPScopes, "admin:read") {
		t.Fatal("default MCP scopes must not include admin:read")
	}
	if !HasScope(DefaultMCPScopes, "projects:read") || !HasScope(DefaultMCPScopes, "deployments:write") {
		t.Fatal("default MCP scopes lost expected project/deployment capabilities")
	}
}
