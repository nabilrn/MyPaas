package container

import (
	"fmt"
	"strings"
)

type VolumeMount struct {
	Name   string
	Target string
}

func (d *DockerCLI) runArgsWithVolumes(opts RunOptions, volumes []VolumeMount) ([]string, error) {
	args := []string{
		"run", "-d",
		"--name", opts.Name,
		"-p", d.portMapping(opts),
		"--memory", fmt.Sprintf("%dm", opts.MemoryMB),
		"--cpus", fmt.Sprintf("%.2f", opts.CPULimit),
		"--restart", "unless-stopped",
	}
	if d.projectNetwork != "" {
		args = append(args, "--network", d.projectNetwork)
	} else {
		args = append(args, "--add-host", "host.docker.internal:host-gateway")
	}
	for _, volume := range volumes {
		name := strings.TrimSpace(volume.Name)
		target := strings.TrimSpace(volume.Target)
		if name == "" || target == "" {
			return nil, fmt.Errorf("volume name and target are required")
		}
		args = append(args, "--mount", fmt.Sprintf("type=volume,source=%s,target=%s", name, target))
	}
	if opts.EnvFile != "" {
		args = append(args, "--env-file", opts.EnvFile)
	}
	args = append(args, opts.Image)
	return args, nil
}
