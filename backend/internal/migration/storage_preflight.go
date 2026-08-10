package migration

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

type StoragePreflight interface {
	Check(context.Context) error
}

type dockerStoragePreflight struct{}

func newStoragePreflight() StoragePreflight {
	return dockerStoragePreflight{}
}

func (dockerStoragePreflight) Check(ctx context.Context) error {
	idsRaw, err := exec.CommandContext(ctx, "docker", "ps", "-aq").CombinedOutput()
	if err != nil {
		return fmt.Errorf("inspect container storage: docker ps: %w: %s", err, strings.TrimSpace(string(idsRaw)))
	}
	ids := strings.Fields(string(idsRaw))
	if len(ids) == 0 {
		return nil
	}

	args := append([]string{"inspect"}, ids...)
	inspectRaw, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("inspect container storage: docker inspect: %w: %s", err, strings.TrimSpace(string(inspectRaw)))
	}

	volumes, err := projectEngineVolumes(inspectRaw)
	if err != nil {
		return fmt.Errorf("inspect container storage: %w", err)
	}
	if len(volumes) == 0 {
		return nil
	}

	return fmt.Errorf(
		"migration export blocked: engine-managed volumes are attached to MyPaas Compose projects: %s; move persistent data to bind mounts under /var/lib/mypaas/volumes or migrate these volumes separately before exporting",
		strings.Join(volumes, ", "),
	)
}

func projectEngineVolumes(raw []byte) ([]string, error) {
	var containers []struct {
		Config struct {
			Labels map[string]string `json:"Labels"`
		} `json:"Config"`
		Mounts []struct {
			Type string `json:"Type"`
			Name string `json:"Name"`
		} `json:"Mounts"`
	}
	if err := json.Unmarshal(raw, &containers); err != nil {
		return nil, fmt.Errorf("decode docker inspect: %w", err)
	}

	seen := make(map[string]struct{})
	for _, item := range containers {
		project := strings.TrimSpace(item.Config.Labels["com.docker.compose.project"])
		if !strings.HasPrefix(project, "mypaas-") {
			continue
		}
		for _, mount := range item.Mounts {
			if mount.Type != "volume" {
				continue
			}
			name := strings.TrimSpace(mount.Name)
			if name != "" {
				seen[name] = struct{}{}
			}
		}
	}

	volumes := make([]string, 0, len(seen))
	for name := range seen {
		volumes = append(volumes, name)
	}
	sort.Strings(volumes)
	return volumes, nil
}
