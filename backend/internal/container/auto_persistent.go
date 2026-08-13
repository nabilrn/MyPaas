package container

import (
	"context"
	"crypto/sha256"
	"fmt"
	pathpkg "path"
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

func normalizeRuntimeVolumeTarget(target string) (string, error) {
	target = strings.TrimSpace(target)
	if target == "" || !pathpkg.IsAbs(target) {
		return "", fmt.Errorf("image volume target must be an absolute path")
	}
	cleaned := pathpkg.Clean(target)
	if cleaned == "/" {
		return "", fmt.Errorf("image volume target cannot be the container root")
	}
	return cleaned, nil
}

func (d *DockerCLI) persistentVolumesForRun(ctx context.Context, opts RunOptions) ([]VolumeMount, error) {
	targets, err := d.ImageVolumeTargets(ctx, opts.Image)
	if err != nil {
		return nil, err
	}
	volumes := make([]VolumeMount, 0, len(targets))
	for _, rawTarget := range targets {
		target, err := normalizeRuntimeVolumeTarget(rawTarget)
		if err != nil {
			return nil, fmt.Errorf("invalid image volume %q: %w", rawTarget, err)
		}
		name := runtimeVolumeName(opts.Name, target)
		if err := d.EnsureVolume(ctx, name); err != nil {
			return nil, err
		}
		volumes = append(volumes, VolumeMount{Name: name, Target: target})
	}
	return volumes, nil
}
