package config

import (
	"path/filepath"
	"testing"
)

func TestEnvPathResolvesRelativePathFromConfigDir(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("MYPAAS_CONFIG_DIR", configDir)
	t.Setenv("STATIC_ROOT", "./.run/static")

	got := envPath("STATIC_ROOT", "/var/lib/mypaas/static")
	want := filepath.Join(configDir, ".run", "static")
	if got != want {
		t.Fatalf("envPath() = %q, want %q", got, want)
	}
}

func TestEnvPathKeepsAbsolutePath(t *testing.T) {
	t.Setenv("MYPAAS_CONFIG_DIR", t.TempDir())
	t.Setenv("STATIC_ROOT", "/var/lib/mypaas/static")

	got := envPath("STATIC_ROOT", "")
	if got != "/var/lib/mypaas/static" {
		t.Fatalf("envPath() = %q, want /var/lib/mypaas/static", got)
	}
}
