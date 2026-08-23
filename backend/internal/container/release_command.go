package container

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
)

const releaseCommandFileSuffix = ".mypaas-release-command"

// runReleaseCommandIfConfigured executes an opt-in one-off command from the
// candidate image before the primary runtime is created. The command runs with
// the same application env file and project network, but without persistent
// volumes. It is intended for idempotent database/schema release work.
func (d *DockerCLI) runReleaseCommandIfConfigured(ctx context.Context, opts RunOptions) error {
	if strings.TrimSpace(opts.EnvFile) == "" {
		return nil
	}
	commandPath := opts.EnvFile + releaseCommandFileSuffix
	raw, err := os.ReadFile(commandPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read release command metadata: %w", err)
	}
	command := strings.TrimSpace(string(raw))
	if command == "" {
		return nil
	}

	name := strings.TrimSpace(opts.Name) + "-release"
	if err := d.Remove(ctx, name); err != nil {
		return fmt.Errorf("remove stale release command container: %w", err)
	}
	cmd := commandContext(ctx, "docker", releaseCommandArgs(opts, d.projectNetwork, command)...)
	if err := cmd.Run(); err != nil {
		// Deliberately do not include stdout/stderr in the returned error: release
		// commands frequently invoke migration tooling whose output may contain
		// credentials or connection strings.
		return fmt.Errorf("release command failed: %w", err)
	}
	return nil
}

func releaseCommandArgs(opts RunOptions, projectNetwork, command string) []string {
	args := []string{
		"run", "--rm",
		"--name", strings.TrimSpace(opts.Name) + "-release",
		"--memory", fmt.Sprintf("%dm", opts.MemoryMB),
		"--cpus", fmt.Sprintf("%.2f", opts.CPULimit),
		"--log-driver", ManagedLogDriver,
		"--log-opt", "max-size=" + ManagedLogMaxSize,
	}
	if strings.TrimSpace(projectNetwork) != "" {
		args = append(args, "--network", strings.TrimSpace(projectNetwork))
	} else {
		args = append(args, "--add-host", "host.docker.internal:host-gateway")
	}
	return append(args, "--env-file", opts.EnvFile, opts.Image, "sh", "-lc", command)
}
