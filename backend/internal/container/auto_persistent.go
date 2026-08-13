package container

import (
	"context"
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
)

var replacementSuffixPattern = regexp.MustCompile(`-[0-9a-f]{8}-[0-9a-f]{3}$`)

func stableRuntimeName(name string) string {
	name = strings.TrimSpace(name)
	return replacementSuffixPattern.ReplaceAllString(name, "")
}

func runtimeVolumeName(containerName, target string) string {
	sum := sha256.Sum256([]byte(target))
	return fmt.Sprintf("%s-data-%x", stableRuntimeName(containerName), sum[:6])
}

func (d *DockerCLI) persistentVolumesForRun(ctx context.Context, opts RunOptions) ([]VolumeMount, error) {
	targets, err := d.ImageVolumeTargets(ctx, opts.Image)
	if err != nil {
		return nil, err
	}
	volumes := make([]VolumeMount, 0, len(targets))
	for _, target := range targets {
		target = strings.TrimSpace(target)
		if target == "" {
			continue
		}
		volumes = append(volumes, VolumeMount{
			Name:   runtimeVolumeName(opts.Name, target),
			Target: target,
		})
	}
	return volumes, nil
}
