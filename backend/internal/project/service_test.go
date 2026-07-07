package project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"mypaas/internal/envdiscover"
)

func TestInferDockerfileAppPortFromExpose(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "Dockerfile"), "FROM alpine\nEXPOSE 8080/tcp\n")

	got := inferDockerfileAppPort(dir, nil)
	if got != 8080 {
		t.Fatalf("inferDockerfileAppPort() = %d, want 8080", got)
	}
}

func TestInferDockerfileAppPortFromEnvReference(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "Dockerfile"), "FROM node:20\nENV PORT=4173\nEXPOSE $PORT\n")

	got := inferDockerfileAppPort(dir, nil)
	if got != 4173 {
		t.Fatalf("inferDockerfileAppPort() = %d, want 4173", got)
	}
}

func TestInferDockerfileAppPortFromDiscoveredEnv(t *testing.T) {
	dir := t.TempDir()
	value := "9000"
	writeFile(t, filepath.Join(dir, "Dockerfile"), "FROM alpine\n")

	got := inferDockerfileAppPort(dir, []envdiscover.Var{{Key: "PORT", DefaultValue: &value}})
	if got != 9000 {
		t.Fatalf("inferDockerfileAppPort() = %d, want 9000", got)
	}
}

func TestParseComposePortsUsesContainerTarget(t *testing.T) {
	raw, err := json.Marshal([]map[string]any{{
		"published": "3000",
		"target":    8080,
		"protocol":  "tcp",
	}})
	if err != nil {
		t.Fatal(err)
	}

	got := parseComposePorts(raw, nil)
	if got != 8080 {
		t.Fatalf("parseComposePorts() = %d, want 8080", got)
	}
}

func TestParseComposeExposeUsesFirstPort(t *testing.T) {
	raw, err := json.Marshal([]string{"7000", "9000"})
	if err != nil {
		t.Fatal(err)
	}

	got := parseComposeExpose(raw, nil)
	if got != 7000 {
		t.Fatalf("parseComposeExpose() = %d, want 7000", got)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}
