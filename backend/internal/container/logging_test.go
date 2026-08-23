package container

import (
	"encoding/json"
	"slices"
	"testing"
)

func TestRunArgsWithVolumesAppliesManagedLogging(t *testing.T) {
	cli := NewDockerCLI("127.0.0.1", "mypaas-projects")
	args, err := cli.runArgsWithVolumes(RunOptions{
		Name:          "mypaas-demo",
		Image:         "example/demo:latest",
		HostPort:      3456,
		ContainerPort: 8080,
		MemoryMB:      512,
		CPULimit:      0.5,
	}, []VolumeMount{{Name: "demo-data", Target: "/data"}})
	if err != nil {
		t.Fatalf("runArgsWithVolumes() error = %v", err)
	}
	for _, value := range []string{"--log-driver", ManagedLogDriver, "--log-opt", "max-size=" + ManagedLogMaxSize} {
		if !slices.Contains(args, value) {
			t.Fatalf("runArgsWithVolumes() missing %q: %v", value, args)
		}
	}
}

func TestSanitizeComposeConfigAddsManagedLoggingWhenMissing(t *testing.T) {
	out, err := sanitizeComposeConfig([]byte(`{"services":{"web":{"image":"example/web:latest"}}}`))
	if err != nil {
		t.Fatalf("sanitizeComposeConfig() error = %v", err)
	}
	var doc struct {
		Services map[string]struct {
			Logging struct {
				Driver  string            `json:"driver"`
				Options map[string]string `json:"options"`
			} `json:"logging"`
		} `json:"services"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatal(err)
	}
	logging := doc.Services["web"].Logging
	if logging.Driver != ManagedLogDriver || logging.Options["max-size"] != ManagedLogMaxSize {
		t.Fatalf("unexpected managed logging config: %+v", logging)
	}
}

func TestSanitizeComposeConfigPreservesCustomLogging(t *testing.T) {
	out, err := sanitizeComposeConfig([]byte(`{"services":{"web":{"image":"example/web:latest","logging":{"driver":"journald","options":{"tag":"custom"}}}}}`))
	if err != nil {
		t.Fatalf("sanitizeComposeConfig() error = %v", err)
	}
	var doc struct {
		Services map[string]struct {
			Logging struct {
				Driver  string            `json:"driver"`
				Options map[string]string `json:"options"`
			} `json:"logging"`
		} `json:"services"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatal(err)
	}
	logging := doc.Services["web"].Logging
	if logging.Driver != "journald" || logging.Options["tag"] != "custom" {
		t.Fatalf("custom logging config was overwritten: %+v", logging)
	}
	if _, exists := logging.Options["max-size"]; exists {
		t.Fatalf("managed max-size should not be injected into custom logging: %+v", logging)
	}
}
