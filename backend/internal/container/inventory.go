package container

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// RuntimeContainer is a host-wide read-only view of a container visible through
// the Docker-compatible runtime. It intentionally includes platform/control
// plane containers and unrelated host containers so owners can diagnose the
// whole single-node runtime from one page.
type RuntimeContainer struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	Image            string  `json:"image"`
	State            string  `json:"state"`
	Status           string  `json:"status"`
	ComposeProject   string  `json:"composeProject"`
	Service          string  `json:"service"`
	CPUPercent       float64 `json:"cpu"`
	MemoryMB         float64 `json:"memoryMb"`
	MemoryLimitMB    float64 `json:"memoryLimitMb"`
	MetricsAvailable bool    `json:"metricsAvailable"`
}

type dockerPSLine struct {
	ID     string `json:"ID"`
	Names  string `json:"Names"`
	Image  string `json:"Image"`
	State  string `json:"State"`
	Status string `json:"Status"`
	Labels string `json:"Labels"`
}

// RuntimeContainers returns every container visible to the Docker-compatible
// runtime. A stats failure does not hide the inventory: metadata is still
// returned and MetricsAvailable remains false for the affected rows.
func (d *DockerCLI) RuntimeContainers(ctx context.Context) ([]RuntimeContainer, error) {
	out, err := commandContext(ctx, "docker", "ps", "-a", "--no-trunc", "--format", "{{json .}}").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("docker ps inventory: %w: %s", err, strings.TrimSpace(string(out)))
	}

	containers := make([]RuntimeContainer, 0)
	for _, line := range fieldsByLine(string(out)) {
		var raw dockerPSLine
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			return nil, fmt.Errorf("parse docker ps inventory: %w", err)
		}
		containers = append(containers, RuntimeContainer{
			ID:             strings.TrimSpace(raw.ID),
			Name:           strings.TrimSpace(raw.Names),
			Image:          strings.TrimSpace(raw.Image),
			State:          strings.ToLower(strings.TrimSpace(raw.State)),
			Status:         strings.TrimSpace(raw.Status),
			ComposeProject: dockerLabel(raw.Labels, "com.docker.compose.project"),
			Service:        dockerLabel(raw.Labels, "com.docker.compose.service"),
		})
	}

	statsOut, statsErr := commandContext(ctx, "docker", "stats", "--no-stream", "--format", "{{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}").CombinedOutput()
	if statsErr == nil {
		statsByName := make(map[string]Metrics)
		for _, line := range fieldsByLine(string(statsOut)) {
			metric, err := parseStatsLine(line)
			if err != nil {
				continue
			}
			statsByName[metric.Service] = metric
		}
		for i := range containers {
			metric, ok := statsByName[containers[i].Name]
			if !ok {
				continue
			}
			containers[i].CPUPercent = metric.CPUPercent
			containers[i].MemoryMB = metric.MemoryMB
			containers[i].MemoryLimitMB = metric.MemoryLimitMB
			containers[i].MetricsAvailable = true
		}
	}

	if containers == nil {
		containers = []RuntimeContainer{}
	}
	return containers, nil
}

func dockerLabel(labels, key string) string {
	for _, item := range strings.Split(labels, ",") {
		parts := strings.SplitN(strings.TrimSpace(item), "=", 2)
		if len(parts) == 2 && parts[0] == key {
			return strings.TrimSpace(parts[1])
		}
	}
	return ""
}
