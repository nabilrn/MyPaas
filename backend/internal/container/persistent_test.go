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

func TestRunArgsWithBindMountsIncludesStableMounts(t *testing.T) {
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
	mounts := []BindMount{{Source: "/var/lib/mypaas/volumes/project-id/app/data", Target: "/app/data"}}

	got, err := cli.runArgsWithBindMounts(opts, mounts)
	if err != nil {
		t.Fatalf("runArgsWithBindMounts() error = %v", err)
	}
	joined := strings.Join(got, " ")
	if !strings.Contains(joined, "--mount type=bind,src=/var/lib/mypaas/volumes/project-id/app/data,dst=/app/data") {
		t.Fatalf("run args missing persistent bind mount: %v", got)
	}
	if got[len(got)-1] != opts.Image {
		t.Fatalf("last run arg = %q, want image %q", got[len(got)-1], opts.Image)
	}
}

func TestRunArgsWithBindMountsRejectsInvalidMount(t *testing.T) {
	cli := NewDockerCLI("127.0.0.1")
	_, err := cli.runArgsWithBindMounts(RunOptions{
		Name: "demo", Image: "example/demo:latest", HostPort: 3001, ContainerPort: 3000, MemoryMB: 128, CPULimit: 0.5,
	}, []BindMount{{Source: "", Target: "/app/data"}})
	if err == nil {
		t.Fatal("runArgsWithBindMounts() error = nil, want validation error")
	}
}
