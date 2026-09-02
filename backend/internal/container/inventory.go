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

const runtimeStatsFormat = "{{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}"

// RuntimeContainerMetadata returns every container visible to the
// Docker-compatible runtime without waiting for CPU/RAM telemetry.
func (d *DockerCLI) RuntimeContainerMetadata(ctx context.Context) ([]RuntimeContainer, error) {
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
	if containers == nil {
		containers = []RuntimeContainer{}
	}
	return containers, nil
}

// RuntimeContainers returns every container visible to the Docker-compatible
// runtime with best-effort CPU/RAM telemetry. A stats failure does not hide the
// inventory: metadata is still returned and MetricsAvailable remains false only
// for rows whose stats could not be collected.
func (d *DockerCLI) RuntimeContainers(ctx context.Context) ([]RuntimeContainer, error) {
	containers, err := d.RuntimeContainerMetadata(ctx)
	if err != nil {
		return nil, err
	}
	targets := runtimeStatsTargets(containers)
	if len(targets) > 0 {
		args := []string{"stats", "--no-stream", "--format", runtimeStatsFormat}
		args = append(args, targets...)
		statsOut, statsErr := commandContext(ctx, "docker", args...).CombinedOutput()
		if statsErr == nil {
			mergeRuntimeStats(containers, statsOut)
		}

		// Docker-compatible runtimes can return exit 0 for multi-target stats
		// while omitting or formatting one or more rows differently. Always
		// retry any still-unmatched running container through the existing
		// single-target Stats path, which is also used by project metrics.
		for _, index := range missingRuntimeStatsIndices(containers) {
			metric, err := d.Stats(ctx, containers[index].Name)
			if err != nil {
				continue
			}
			applyRuntimeMetric(&containers[index], metric)
		}
	}

	return containers, nil
}

func runtimeStatsTargets(containers []RuntimeContainer) []string {
	targets := make([]string, 0, len(containers))
	for _, container := range containers {
		if container.State != "running" || strings.TrimSpace(container.Name) == "" {
			continue
		}
		targets = append(targets, container.Name)
	}
	return targets
}

func missingRuntimeStatsIndices(containers []RuntimeContainer) []int {
	indices := make([]int, 0)
	for i := range containers {
		if containers[i].State != "running" || strings.TrimSpace(containers[i].Name) == "" || containers[i].MetricsAvailable {
			continue
		}
		indices = append(indices, i)
	}
	return indices
}

func mergeRuntimeStats(containers []RuntimeContainer, raw []byte) {
	statsByName := make(map[string]Metrics)
	for _, line := range fieldsByLine(string(raw)) {
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
		applyRuntimeMetric(&containers[i], metric)
	}
}

func applyRuntimeMetric(container *RuntimeContainer, metric Metrics) {
	container.CPUPercent = metric.CPUPercent
	container.MemoryMB = metric.MemoryMB
	container.MemoryLimitMB = metric.MemoryLimitMB
	container.MetricsAvailable = true
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
