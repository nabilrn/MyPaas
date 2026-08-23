package envvar

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteEnvFileSeparatesReleaseCommandMetadata(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	values := map[string]string{
		"DATABASE_URL":      "postgres://app:secret@postgres/app",
		ReleaseCommandKey:    "npm run migrate",
		"MYPAAS_PLATFORM_X": "internal",
	}

	if err := WriteEnvFile(path, values); err != nil {
		t.Fatalf("WriteEnvFile() error = %v", err)
	}
	runtimeEnv, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(runtimeEnv)
	if !strings.Contains(text, "DATABASE_URL=") {
		t.Fatalf("runtime env missing application variable: %q", text)
	}
	if strings.Contains(text, "MYPAAS_PLATFORM_") || strings.Contains(text, "npm run migrate") {
		t.Fatalf("platform metadata leaked into runtime env: %q", text)
	}

	release, err := os.ReadFile(path + ReleaseCommandFileSuffix)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(release); got != "npm run migrate" {
		t.Fatalf("release sidecar = %q", got)
	}
}

func TestWriteEnvFileRemovesStaleReleaseCommandMetadata(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path+ReleaseCommandFileSuffix, []byte("old command"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := WriteEnvFile(path, map[string]string{"APP_ENV": "production"}); err != nil {
		t.Fatalf("WriteEnvFile() error = %v", err)
	}
	if _, err := os.Stat(path + ReleaseCommandFileSuffix); !os.IsNotExist(err) {
		t.Fatalf("stale release metadata still exists: %v", err)
	}
}
