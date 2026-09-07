package container

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStripGeneratedComposeCPUHardCaps(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "docker-compose.mypaas.override.yml")
	input := `services:
  "app":
    mem_limit: 512m
    cpus: 0.50
  "worker":
    mem_limit: 256m
    cpus: 0.25
`
	if err := os.WriteFile(path, []byte(input), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := stripGeneratedComposeCPUHardCaps(ComposeUpOptions{WorkDir: dir, OverrideFile: filepath.Base(path)}); err != nil {
		t.Fatalf("stripGeneratedComposeCPUHardCaps() error = %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(raw)
	if strings.Contains(got, "cpus:") {
		t.Fatalf("override still contains CPU hard cap:\n%s", got)
	}
	if !strings.Contains(got, "mem_limit: 512m") || !strings.Contains(got, "mem_limit: 256m") {
		t.Fatalf("memory limits were not preserved:\n%s", got)
	}
}

func TestSanitizeComposeConfigStripsCPUHardLimits(t *testing.T) {
	raw := []byte(`{
  "services": {
    "app": {
      "image": "example/app:latest",
      "cpus": 0.5,
      "cpu_quota": 50000,
      "cpu_period": 100000,
      "cpuset": "0",
      "mem_limit": 536870912,
      "deploy": {
        "resources": {
          "limits": {"cpus": "0.50", "memory": "512M"},
          "reservations": {"cpus": "0.10", "memory": "128M"}
        }
      }
    }
  }
}`)

	out, err := sanitizeComposeConfig(raw)
	if err != nil {
		t.Fatalf("sanitizeComposeConfig() error = %v", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatal(err)
	}
	service := doc["services"].(map[string]any)["app"].(map[string]any)
	for _, key := range []string{"cpus", "cpu_quota", "cpu_period", "cpuset"} {
		if _, ok := service[key]; ok {
			t.Fatalf("service still contains %s: %#v", key, service[key])
		}
	}
	if _, ok := service["mem_limit"]; !ok {
		t.Fatal("memory limit was removed with CPU limits")
	}

	deploy := service["deploy"].(map[string]any)
	resources := deploy["resources"].(map[string]any)
	for _, bucket := range []string{"limits", "reservations"} {
		values := resources[bucket].(map[string]any)
		if _, ok := values["cpus"]; ok {
			t.Fatalf("%s still contains cpus: %#v", bucket, values["cpus"])
		}
		if _, ok := values["memory"]; !ok {
			t.Fatalf("%s memory was removed", bucket)
		}
	}
}
