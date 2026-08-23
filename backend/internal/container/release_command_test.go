package container

import (
	"slices"
	"testing"
)

func TestReleaseCommandArgsUsesCandidateRuntimeContract(t *testing.T) {
	opts := RunOptions{
		Name:          "mypaas-demo-01234567-abc",
		Image:         "mypaas/demo:abc123",
		ContainerPort: 8080,
		MemoryMB:      512,
		CPULimit:      0.5,
		EnvFile:       "/tmp/deploy/.env",
	}
	args := releaseCommandArgs(opts, "mypaas-projects", "npm run migrate")

	for _, required := range []string{
		"--rm",
		"mypaas-demo-01234567-abc-release",
		"--network",
		"mypaas-projects",
		"--env-file",
		"/tmp/deploy/.env",
		"max-size=20m",
		"mypaas/demo:abc123",
		"sh",
		"-lc",
		"npm run migrate",
	} {
		if !slices.Contains(args, required) {
			t.Fatalf("releaseCommandArgs() missing %q: %v", required, args)
		}
	}
}

func TestReleaseCommandArgsUsesHostGatewayWithoutProjectNetwork(t *testing.T) {
	args := releaseCommandArgs(RunOptions{
		Name:      "mypaas-demo-candidate",
		Image:     "example/app:latest",
		MemoryMB:  256,
		CPULimit:  0.25,
		EnvFile:   "/tmp/.env",
	}, "", "echo ok")

	if !slices.Contains(args, "host.docker.internal:host-gateway") {
		t.Fatalf("releaseCommandArgs() missing host-gateway fallback: %v", args)
	}
}
