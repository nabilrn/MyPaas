package envdiscover

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverEnvFilesAndComposeVariables(t *testing.T) {
	workspace := t.TempDir()
	if err := os.WriteFile(filepath.Join(workspace, ".env.example"), []byte(`
APP_NAME=demo
SECRET_KEY=sample
export PUBLIC_URL="https://example.test" # comment
`), 0640); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "compose.yml"), []byte(`
services:
  app:
    environment:
      DATABASE_URL: ${DATABASE_URL}
      PORT: ${PORT:-3000}
      CACHE_DSN: ${CACHE_DSN:?required}
`), 0640); err != nil {
		t.Fatal(err)
	}

	vars, err := Discover(workspace, "compose.yml")
	if err != nil {
		t.Fatalf("Discover returned error: %v", err)
	}
	byKey := make(map[string]Var, len(vars))
	for _, item := range vars {
		byKey[item.Key] = item
	}

	if got := byKey["APP_NAME"].DefaultValue; got == nil || *got != "demo" {
		t.Fatalf("APP_NAME default = %v, want demo", got)
	}
	if !byKey["SECRET_KEY"].Sensitive || byKey["SECRET_KEY"].DefaultValue != nil {
		t.Fatalf("SECRET_KEY should be sensitive without copied default: %#v", byKey["SECRET_KEY"])
	}
	if got := byKey["PORT"].DefaultValue; got == nil || *got != "3000" {
		t.Fatalf("PORT default = %v, want 3000", got)
	}
	if !byKey["DATABASE_URL"].Sensitive {
		t.Fatalf("DATABASE_URL should be sensitive")
	}
	if !byKey["CACHE_DSN"].Sensitive {
		t.Fatalf("CACHE_DSN should be sensitive")
	}
}
