package deployment

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"

	"mypaas/internal/config"
	"mypaas/internal/db"
)

func TestWriteComposeOverrideReplacesExistingPorts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "docker-compose.mypaas.override.yml")

	if err := writeComposeOverride(path, "app", "127.0.0.1:3001:8080", 512, 0.5, "mypaas-dev", "mypaas/demo-app:abc123"); err != nil {
		t.Fatalf("writeComposeOverride() error = %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	if !strings.Contains(content, "ports: !override") {
		t.Fatalf("override should replace compose ports, got:\n%s", content)
	}
	if !strings.Contains(content, `"127.0.0.1:3001:8080"`) {
		t.Fatalf("override missing MyPaas port mapping, got:\n%s", content)
	}
	if !strings.Contains(content, `name: "mypaas-dev"`) {
		t.Fatalf("override missing platform network, got:\n%s", content)
	}
	if !strings.Contains(content, `image: "mypaas/demo-app:abc123"`) {
		t.Fatalf("override missing immutable image tag, got:\n%s", content)
	}
}

func TestStaticCaddyPathUsesContainerPathSeparators(t *testing.T) {
	projectID := uuid.MustParse("741151a3-ee89-4357-a797-289b50be2431")
	service := &Service{
		cfg: &config.Config{
			CaddyStaticRoot: `\var\lib\mypaas\static`,
		},
	}

	got := service.staticCaddyPath(db.Project{ID: projectID})
	want := "/var/lib/mypaas/static/741151a3-ee89-4357-a797-289b50be2431"
	if got != want {
		t.Fatalf("staticCaddyPath() = %q, want %q", got, want)
	}
}

func TestIdleMetricsUsesProjectShapeWithoutDocker(t *testing.T) {
	mainService := "web"
	project := db.Project{
		DeployMode:    "compose",
		MainService:   &mainService,
		MemoryLimitMb: 256,
		Status:        "stopped",
	}

	items := idleMetrics(project)
	if len(items) != 1 {
		t.Fatalf("idleMetrics() returned %d items, want 1", len(items))
	}
	got := items[0]
	if got.Service != "web" {
		t.Fatalf("Service = %q, want web", got.Service)
	}
	if got.MemoryLimitMB != 256 {
		t.Fatalf("MemoryLimitMB = %v, want 256", got.MemoryLimitMB)
	}
	if got.CPUPercent != 0 || got.MemoryMB != 0 || got.Uptime != "n/a" {
		t.Fatalf("idle metrics should be zeroed with n/a uptime, got %+v", got)
	}
}

func TestHasLiveRuntime(t *testing.T) {
	for _, status := range []string{"running", "building"} {
		t.Run(status, func(t *testing.T) {
			if !hasLiveRuntime(status) {
				t.Fatalf("hasLiveRuntime(%q) = false, want true", status)
			}
		})
	}
	for _, status := range []string{"pending", "stopped", "crashed"} {
		t.Run(status, func(t *testing.T) {
			if hasLiveRuntime(status) {
				t.Fatalf("hasLiveRuntime(%q) = true, want false", status)
			}
		})
	}
}
