package deployment

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInjectComposeEnvFileAddsProjectEnvToMainService(t *testing.T) {
	dir := t.TempDir()
	overridePath := filepath.Join(dir, "docker-compose.mypaas.override.yml")
	envPath := filepath.Join(dir, "project.env")

	if err := writeComposeOverride(overridePath, "wago", "127.0.0.1:3201:3000", 512, 0.5, "mypaas-dev", "", nil); err != nil {
		t.Fatalf("writeComposeOverride() error = %v", err)
	}
	if err := injectComposeEnvFile(context.Background(), overridePath, envPath); err != nil {
		t.Fatalf("injectComposeEnvFile() error = %v", err)
	}

	raw, err := os.ReadFile(overridePath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	if !strings.Contains(content, "    env_file:\n      - "+strconvQuote(envPath)+"\n") {
		t.Fatalf("override missing project env_file, got:\n%s", content)
	}
}

func TestInjectComposeEnvFileIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	overridePath := filepath.Join(dir, "docker-compose.mypaas.override.yml")
	envPath := filepath.Join(dir, "project.env")

	if err := writeComposeOverride(overridePath, "app", "127.0.0.1:3202:8080", 512, 0.5, "", "", nil); err != nil {
		t.Fatalf("writeComposeOverride() error = %v", err)
	}
	if err := injectComposeEnvFile(context.Background(), overridePath, envPath); err != nil {
		t.Fatal(err)
	}
	if err := injectComposeEnvFile(context.Background(), overridePath, envPath); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(overridePath)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(raw), "env_file:"); got != 1 {
		t.Fatalf("env_file count = %d, want 1\n%s", got, string(raw))
	}
}

func TestInjectComposeEnvFileHonorsCancellation(t *testing.T) {
	dir := t.TempDir()
	overridePath := filepath.Join(dir, "docker-compose.mypaas.override.yml")
	envPath := filepath.Join(dir, "project.env")

	if err := writeComposeOverride(overridePath, "app", "127.0.0.1:3203:8080", 512, 0.5, "", "", nil); err != nil {
		t.Fatalf("writeComposeOverride() error = %v", err)
	}
	before, err := os.ReadFile(overridePath)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := injectComposeEnvFile(ctx, overridePath, envPath); !errors.Is(err, context.Canceled) {
		t.Fatalf("injectComposeEnvFile() error = %v, want context.Canceled", err)
	}

	after, err := os.ReadFile(overridePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatalf("override changed after canceled context:\n%s", string(after))
	}
}

func strconvQuote(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `\"`) + `"`
}
