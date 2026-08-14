package container

import (
	"strings"
	"testing"
)

func TestParseImageVolumeTargetsSortsDeclaredVolumes(t *testing.T) {
	raw := []byte(`[{"Config":{"Volumes":{"/var/lib/app":{},"/app/data":{}}}}]`)

	got, err := parseImageVolumeTargets(raw)
	if err != nil {
		t.Fatalf("parseImageVolumeTargets() error = %v", err)
	}
	want := []string{"/app/data", "/var/lib/app"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("parseImageVolumeTargets() = %v, want %v", got, want)
	}
}

func TestParseImageVolumeTargetsHandlesNullVolumes(t *testing.T) {
	raw := []byte(`[{"Config":{"Volumes":null}}]`)
	got, err := parseImageVolumeTargets(raw)
	if err != nil {
		t.Fatalf("parseImageVolumeTargets() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("parseImageVolumeTargets() = %v, want empty", got)
	}
}

func TestParseImageVolumeTargetsReadsPersistenceLabel(t *testing.T) {
	raw := []byte(`[{"Config":{"Volumes":null,"Labels":{"io.mypaas.persistent-volumes":" /var/lib/app, /app/data "}}}]`)

	got, err := parseImageVolumeTargets(raw)
	if err != nil {
		t.Fatalf("parseImageVolumeTargets() error = %v", err)
	}
	want := []string{"/app/data", "/var/lib/app"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("parseImageVolumeTargets() = %v, want %v", got, want)
	}
}

func TestParseImageVolumeTargetsDeduplicatesLabelAndDockerVolume(t *testing.T) {
	raw := []byte(`[{"Config":{"Volumes":{"/app/data":{}},"Labels":{"io.mypaas.persistent-volumes":"/app/data,/var/lib/app"}}}]`)

	got, err := parseImageVolumeTargets(raw)
	if err != nil {
		t.Fatalf("parseImageVolumeTargets() error = %v", err)
	}
	want := []string{"/app/data", "/var/lib/app"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("parseImageVolumeTargets() = %v, want %v", got, want)
	}
}

func TestRunArgsWithVolumesIncludesStableVolume(t *testing.T) {
	cli := NewDockerCLI("127.0.0.1", "mypaas-projects")
	opts := RunOptions{
		Name:          "mypaas-wa-api-next",
		Image:         "ghcr.io/example/wago@sha256:abc",
		HostPort:      31001,
		ContainerPort: 3000,
		MemoryMB:      512,
		CPULimit:      0.5,
		EnvFile:       "/tmp/project.env",
	}
	volumes := []VolumeMount{{Name: "mypaas-project-data-123", Target: "/app/data"}}

	got, err := cli.runArgsWithVolumes(opts, volumes)
	if err != nil {
		t.Fatalf("runArgsWithVolumes() error = %v", err)
	}
	joined := strings.Join(got, " ")
	if !strings.Contains(joined, "--mount type=volume,source=mypaas-project-data-123,target=/app/data") {
		t.Fatalf("run args missing persistent named volume: %v", got)
	}
	if got[len(got)-1] != opts.Image {
		t.Fatalf("last run arg = %q, want image %q", got[len(got)-1], opts.Image)
	}
}

func TestRunArgsWithVolumesRejectsInvalidVolume(t *testing.T) {
	cli := NewDockerCLI("127.0.0.1")
	_, err := cli.runArgsWithVolumes(RunOptions{
		Name: "demo", Image: "example/demo:latest", HostPort: 3001, ContainerPort: 3000, MemoryMB: 128, CPULimit: 0.5,
	}, []VolumeMount{{Name: "", Target: "/app/data"}})
	if err == nil {
		t.Fatal("runArgsWithVolumes() error = nil, want validation error")
	}
}
