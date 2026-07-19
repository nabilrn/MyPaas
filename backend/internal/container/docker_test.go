package container

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
	"time"
)

func TestParseStatsLine(t *testing.T) {
	line := `{"Name":"mypaas-demo","CPUPerc":"3.45%","MemUsage":"27.5MiB / 512MiB"}`

	metrics, err := parseStatsLine(line)
	if err != nil {
		t.Fatalf("parseStatsLine() error = %v", err)
	}

	if metrics.Service != "mypaas-demo" {
		t.Fatalf("Service = %q, want mypaas-demo", metrics.Service)
	}
	assertFloat(t, metrics.CPUPercent, 3.45)
	assertFloat(t, metrics.MemoryMB, 27.5)
	assertFloat(t, metrics.MemoryLimitMB, 512)
}

func TestParseMemoryUsage(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantUsed  float64
		wantLimit float64
	}{
		{name: "mib", input: "27.5MiB / 512MiB", wantUsed: 27.5, wantLimit: 512},
		{name: "gib", input: "1.25GiB / 8GiB", wantUsed: 1280, wantLimit: 8192},
		{name: "bytes", input: "1048576B / 536870912B", wantUsed: 1, wantLimit: 512},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			used, limit, err := parseMemoryUsage(tt.input)
			if err != nil {
				t.Fatalf("parseMemoryUsage() error = %v", err)
			}
			assertFloat(t, used, tt.wantUsed)
			assertFloat(t, limit, tt.wantLimit)
		})
	}
}

func TestFormatUptime(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{name: "seconds", duration: 45 * time.Second, want: "<1m"},
		{name: "minutes", duration: 17 * time.Minute, want: "17m"},
		{name: "hours", duration: 2*time.Hour + 8*time.Minute, want: "2h 8m"},
		{name: "days", duration: 49*time.Hour + 30*time.Minute, want: "2d 1h"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatUptime(tt.duration); got != tt.want {
				t.Fatalf("formatUptime() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsNoContainerMessage(t *testing.T) {
	tests := []string{
		"Error response from daemon: No such container: mypaas-static-test",
		"No such container: mypaas-demo",
		"container mypaas-demo not found",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			if !isNoContainerMessage(input) {
				t.Fatalf("isNoContainerMessage(%q) = false, want true", input)
			}
		})
	}
}

func TestDockerCLIPortMappingUsesConfiguredBindHost(t *testing.T) {
	cli := NewDockerCLI("0.0.0.0")
	got := cli.portMapping(RunOptions{HostPort: 3001, ContainerPort: 80})
	if got != "0.0.0.0:3001:80" {
		t.Fatalf("portMapping() = %q, want 0.0.0.0:3001:80", got)
	}
}

func TestComposeBaseArgsIncludesEnvFileBeforeCommand(t *testing.T) {
	got := composeBaseArgs("C:/tmp/project/.env")
	want := []string{"compose", "--env-file", "C:/tmp/project/.env"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("composeBaseArgs() = %v, want %v", got, want)
	}
}

func TestComposeConfigArgsIncludesComposeFileAfterEnvFile(t *testing.T) {
	got := composeConfigArgs("C:/tmp/project/.env", "docker-compose.prod.yml")
	want := []string{"compose", "--env-file", "C:/tmp/project/.env", "-f", "docker-compose.prod.yml"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("composeConfigArgs() = %v, want %v", got, want)
	}
}

func TestComposeConfigArgsMultiPreservesOrderAndSkipsEmpty(t *testing.T) {
	got := composeConfigArgsMulti("/tmp/.env", []string{"docker-compose.yml", "", "docker-compose.prod.yml", ""})
	want := []string{"compose", "--env-file", "/tmp/.env", "-f", "docker-compose.yml", "-f", "docker-compose.prod.yml"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("composeConfigArgsMulti() = %v, want %v", got, want)
	}
}

func TestComposeUpFilesOrdersPrimaryThenUserThenMyPaasOverride(t *testing.T) {
	got := composeUpFiles(ComposeUpOptions{
		ComposeFile:  "docker-compose.yml",
		ComposeFiles: []string{"docker-compose.prod.yml", "docker-compose.cache.yml"},
		OverrideFile: "docker-compose.mypaas.override.yml",
	})
	want := []string{
		"docker-compose.yml",
		"docker-compose.prod.yml",
		"docker-compose.cache.yml",
		"docker-compose.mypaas.override.yml",
	}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("composeUpFiles() = %v, want %v", got, want)
	}
}

func TestComposeUpFilesSupportsLegacySingleFileCallers(t *testing.T) {
	got := composeUpFiles(ComposeUpOptions{
		ComposeFile:  "sanitized.json",
		OverrideFile: "override.yml",
	})
	want := []string{"sanitized.json", "override.yml"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("composeUpFiles() = %v, want %v", got, want)
	}
}

func TestIsMypaasInternalEnvFiltersLeakyComposeVars(t *testing.T) {
	for _, key := range []string{"DATABASE_URL", "POSTGRES_PASSWORD", "JWT_SECRET", "CADDY_ADMIN"} {
		t.Run(key, func(t *testing.T) {
			if !isMypaasInternalEnv(key) {
				t.Fatalf("isMypaasInternalEnv(%q) = false, want true", key)
			}
		})
	}
	for _, key := range []string{"PATH", "SystemRoot", "DOCKER_HOST"} {
		t.Run(key, func(t *testing.T) {
			if isMypaasInternalEnv(key) {
				t.Fatalf("isMypaasInternalEnv(%q) = true, want false", key)
			}
		})
	}
}

func TestParseComposeServiceNamesDedupesAndSorts(t *testing.T) {
	got := parseComposeServiceNames("web\n<no value>\ndb\nweb\r\n")
	want := []string{"db", "web"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("parseComposeServiceNames() = %v, want %v", got, want)
	}
}

func TestParseComposeBuildServicesJSON(t *testing.T) {
	raw := []byte(`{
		"services": {
			"web": {"build": {"context": "."}},
			"worker": {"build": "."},
			"db": {"image": "postgres:16"},
			"cache": {"build": null}
		}
	}`)

	got, err := parseComposeBuildServicesJSON(raw)
	if err != nil {
		t.Fatalf("parseComposeBuildServicesJSON() error = %v", err)
	}
	want := []string{"web", "worker"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("parseComposeBuildServicesJSON() = %v, want %v", got, want)
	}
}

func TestRemoveComposeHostPorts(t *testing.T) {
	raw := []byte(`{
		"services": {
			"web": {
				"image": "demo",
				"ports": [{"target": 8080, "published": "8080"}],
				"expose": ["8080"]
			},
			"db": {
				"image": "postgres:16",
				"ports": [{"target": 5432, "published": "5432"}]
			}
		}
	}`)

	out, err := removeComposeHostPorts(raw)
	if err != nil {
		t.Fatalf("removeComposeHostPorts() error = %v", err)
	}
	var doc struct {
		Services map[string]map[string]any `json:"services"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("sanitized json invalid: %v", err)
	}
	if _, ok := doc.Services["web"]["ports"]; ok {
		t.Fatalf("web ports should be removed, got %s", string(out))
	}
	if _, ok := doc.Services["db"]["ports"]; ok {
		t.Fatalf("db ports should be removed, got %s", string(out))
	}
	if _, ok := doc.Services["web"]["expose"]; !ok {
		t.Fatalf("non-host exposure metadata should be preserved, got %s", string(out))
	}
}

func assertFloat(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.001 {
		t.Fatalf("got %f, want %f", got, want)
	}
}
