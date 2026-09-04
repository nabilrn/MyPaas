package migration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const migrationEngineVolumeStage = "/var/lib/mypaas/volumes/.migration-engine-volumes"

var engineVolumeNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)

type engineVolumeMount struct {
	Name        string
	ContainerID string
	Destination string
	Running     bool
}

type inspectedContainerStorage struct {
	ID    string `json:"Id"`
	Name  string `json:"Name"`
	State struct {
		Running bool `json:"Running"`
	} `json:"State"`
	Config struct {
		Labels map[string]string `json:"Labels"`
	} `json:"Config"`
	Mounts []struct {
		Type        string `json:"Type"`
		Name        string `json:"Name"`
		Destination string `json:"Destination"`
	} `json:"Mounts"`
}

func inspectProjectEngineVolumeMounts(ctx context.Context) ([]engineVolumeMount, error) {
	idsRaw, err := exec.CommandContext(ctx, "docker", "ps", "-aq").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("docker ps: %w: %s", err, strings.TrimSpace(string(idsRaw)))
	}
	ids := strings.Fields(string(idsRaw))
	if len(ids) == 0 {
		return nil, nil
	}

	args := append([]string{"inspect"}, ids...)
	inspectRaw, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("docker inspect: %w: %s", err, strings.TrimSpace(string(inspectRaw)))
	}
	return projectEngineVolumeMounts(inspectRaw)
}

func projectEngineVolumeMounts(raw []byte) ([]engineVolumeMount, error) {
	var containers []inspectedContainerStorage
	if err := json.Unmarshal(raw, &containers); err != nil {
		return nil, fmt.Errorf("decode docker inspect: %w", err)
	}

	byName := make(map[string]*engineVolumeMount)
	for _, item := range containers {
		project := strings.TrimSpace(item.Config.Labels["com.docker.compose.project"])
		if !strings.HasPrefix(project, "mypaas-") || isPlatformComposeService(item.Config.Labels) {
			continue
		}
		containerID := strings.TrimSpace(item.ID)
		if containerID == "" {
			containerID = strings.TrimPrefix(strings.TrimSpace(item.Name), "/")
		}
		for _, mount := range item.Mounts {
			if mount.Type != "volume" {
				continue
			}
			name := strings.TrimSpace(mount.Name)
			destination := strings.TrimSpace(mount.Destination)
			if name == "" || destination == "" || containerID == "" {
				continue
			}
			if !engineVolumeNamePattern.MatchString(name) {
				return nil, fmt.Errorf("unsafe engine-managed volume name %q", name)
			}
			if _, ok := byName[name]; !ok {
				byName[name] = &engineVolumeMount{
					Name:        name,
					ContainerID: containerID,
					Destination: destination,
				}
			}
		}
	}

	// A volume can have more than one consumer. Refuse a snapshot while any
	// consumer is still running, even if the MyPaaS Compose service chosen as
	// the copy source has already stopped.
	for _, item := range containers {
		if !item.State.Running {
			continue
		}
		for _, mount := range item.Mounts {
			if mount.Type != "volume" {
				continue
			}
			if volume := byName[strings.TrimSpace(mount.Name)]; volume != nil {
				volume.Running = true
			}
		}
	}

	volumes := make([]engineVolumeMount, 0, len(byName))
	for _, volume := range byName {
		volumes = append(volumes, *volume)
	}
	sort.Slice(volumes, func(i, j int) bool { return volumes[i].Name < volumes[j].Name })
	return volumes, nil
}

func captureProjectEngineVolumes(ctx context.Context) error {
	volumes, err := inspectProjectEngineVolumeMounts(ctx)
	if err != nil {
		return fmt.Errorf("inspect engine-managed volumes: %w", err)
	}

	if err := os.RemoveAll(migrationEngineVolumeStage); err != nil {
		return fmt.Errorf("clear engine-volume staging: %w", err)
	}
	if len(volumes) == 0 {
		return nil
	}
	if err := os.MkdirAll(migrationEngineVolumeStage, 0700); err != nil {
		return fmt.Errorf("create engine-volume staging: %w", err)
	}

	for _, volume := range volumes {
		if volume.Running {
			return fmt.Errorf("engine-managed volume %s is still mounted by a running container; refusing an inconsistent migration snapshot", volume.Name)
		}
		destination := filepath.Join(migrationEngineVolumeStage, volume.Name)
		if err := os.MkdirAll(destination, 0700); err != nil {
			return fmt.Errorf("create staging directory for %s: %w", volume.Name, err)
		}
		source := fmt.Sprintf("%s:%s/.", volume.ContainerID, strings.TrimRight(volume.Destination, "/"))
		out, err := exec.CommandContext(ctx, "docker", "cp", source, destination).CombinedOutput()
		if err != nil {
			return fmt.Errorf("capture engine-managed volume %s: docker cp: %w: %s", volume.Name, err, strings.TrimSpace(string(out)))
		}
	}
	return nil
}

func cleanupProjectEngineVolumeStage() error {
	if err := os.RemoveAll(migrationEngineVolumeStage); err != nil {
		return fmt.Errorf("remove engine-volume staging: %w", err)
	}
	return nil
}
