package deployment

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestInjectComposeEnvFileAddsProjectEnvToMainService(t *testing.T) {
	dir := t.TempDir()
	overridePath := filepath.Join(dir, "docker-compose.mypaas.override.yml")
	envPath := filepath.Join(dir, "project.env")
	if err := os.WriteFile(envPath, []byte("APP_ENV=production\n"), 0o600); err != nil {
		t.Fatal(err)
	}

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

	envRaw, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(envRaw), "COMPOSE_PARALLEL_LIMIT=1\n") {
		t.Fatalf("project env missing safe compose parallel limit, got:\n%s", string(envRaw))
	}
}

func TestInjectComposeEnvFileIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	overridePath := filepath.Join(dir, "docker-compose.mypaas.override.yml")
	envPath := filepath.Join(dir, "project.env")
	if err := os.WriteFile(envPath, []byte("APP_ENV=production\n"), 0o600); err != nil {
		t.Fatal(err)
	}

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
	envRaw, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(envRaw), "COMPOSE_PARALLEL_LIMIT="); got != 1 {
		t.Fatalf("COMPOSE_PARALLEL_LIMIT count = %d, want 1\n%s", got, string(envRaw))
	}
}

func TestEnsureComposeParallelLimitRespectsExplicitProjectValue(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, "project.env")
	if err := os.WriteFile(envPath, []byte("COMPOSE_PARALLEL_LIMIT=2\nAPP_ENV=production\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := ensureComposeParallelLimit(envPath); err != nil {
		t.Fatalf("ensureComposeParallelLimit() error = %v", err)
	}

	raw, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "COMPOSE_PARALLEL_LIMIT=2\nAPP_ENV=production\n" {
		t.Fatalf("explicit compose parallel limit changed:\n%s", string(raw))
	}
}

func TestInjectComposeEnvFileHonorsCancellation(t *testing.T) {
	dir := t.TempDir()
	overridePath := filepath.Join(dir, "docker-compose.mypaas.override.yml")
	envPath := filepath.Join(dir, "project.env")
	if err := os.WriteFile(envPath, []byte("APP_ENV=production\n"), 0o600); err != nil {
		t.Fatal(err)
	}

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
	envRaw, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(envRaw), "COMPOSE_PARALLEL_LIMIT=") {
		t.Fatalf("env changed after canceled context:\n%s", string(envRaw))
	}
}

func strconvQuote(value string) string {
	return strconv.Quote(value)
}
