package container

import (
	"context"
	"fmt"
	"strings"
)

func (d *DockerCLI) ImageVolumeTargets(ctx context.Context, image string) ([]string, error) {
	image = strings.TrimSpace(image)
	if image == "" {
		return nil, fmt.Errorf("container image is required")
	}
	out, err := commandContext(ctx, "docker", "image", "inspect", image).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("docker image inspect volumes: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return parseImageVolumeTargets(out)
}

func (d *DockerCLI) EnsureVolume(ctx context.Context, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("volume name is required")
	}
	out, err := commandContext(ctx, "docker", "volume", "create", "--label", ManagedImageLabel, name).CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker volume create: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (d *DockerCLI) RunWithVolumes(ctx context.Context, opts RunOptions, volumes []VolumeMount, log func(string)) error {
	args, err := d.runArgsWithVolumes(opts, volumes)
	if err != nil {
		return err
	}
	return runLogged(ctx, "", log, "docker", args...)
}
