package container

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
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

const (
	runtimeStatsFormat           = "{{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}"
	runtimeTelemetryRefreshEvery = 15 * time.Second
	runtimeTelemetryTimeout      = 5 * time.Second
)

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
// runtime and merges the latest best-effort CPU/RAM telemetry snapshot. It does
// not invoke docker stats on the request path; telemetry is refreshed by
// StartRuntimeTelemetry so inventory latency stays bounded as hosts grow.
func (d *DockerCLI) RuntimeContainers(ctx context.Context) ([]RuntimeContainer, error) {
	containers, err := d.RuntimeContainerMetadata(ctx)
	if err != nil {
		return nil, err
	}
	d.applyCachedRuntimeTelemetry(containers)
	return containers, nil
}

// StartRuntimeTelemetry refreshes one bulk telemetry snapshot in the
// background. The returned channel closes when the caller's context is done.
func (d *DockerCLI) StartRuntimeTelemetry(ctx context.Context, interval time.Duration) <-chan struct{} {
	done := make(chan struct{})
	if interval <= 0 {
		interval = runtimeTelemetryRefreshEvery
	}

	go func() {
		defer close(done)
		refresh := func() {
			refreshCtx, cancel := context.WithTimeout(ctx, runtimeTelemetryTimeout)
			defer cancel()
			_ = d.refreshRuntimeTelemetry(refreshCtx)
		}

		refresh()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				refresh()
			}
		}
	}()

	return done
}

func (d *DockerCLI) refreshRuntimeTelemetry(ctx context.Context) error {
	containers, err := d.RuntimeContainerMetadata(ctx)
	if err != nil {
		return err
	}
	stats, err := collectRuntimeStats(ctx, containers)
	if err != nil {
		return err
	}

	// Treat each successful collection as an immutable snapshot. Merging into
	// the previous map would retain metrics for stopped/deleted containers and
	// can leak those values to a later container that reuses the same name.
	d.telemetryMu.Lock()
	d.runtimeTelemetry = stats
	d.telemetryMu.Unlock()
	return nil
}

func collectRuntimeStats(ctx context.Context, containers []RuntimeContainer) (map[string]Metrics, error) {
	targets := runtimeStatsTargets(containers)
	if len(targets) == 0 {
		return map[string]Metrics{}, nil
	}

	args := []string{"stats", "--no-stream", "--format", runtimeStatsFormat}
	args = append(args, targets...)
	statsOut, err := commandContext(ctx, "docker", args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("docker stats inventory: %w: %s", err, strings.TrimSpace(string(statsOut)))
	}

	return parseRuntimeStats(statsOut, len(targets)), nil
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

func (d *DockerCLI) applyCachedRuntimeTelemetry(containers []RuntimeContainer) {
	d.telemetryMu.RLock()
	defer d.telemetryMu.RUnlock()
	for i := range containers {
		// Metadata is authoritative for lifecycle state. A container may stop
		// between telemetry refreshes, so never render the last running sample
		// on a row that is no longer running.
		if containers[i].State != "running" {
			continue
		}
		metric, ok := d.runtimeTelemetry[containers[i].Name]
		if !ok {
			continue
		}
		applyRuntimeMetric(&containers[i], metric)
	}
}

func mergeRuntimeStats(containers []RuntimeContainer, raw []byte) {
	statsByName := parseRuntimeStats(raw, 0)
	for i := range containers {
		metric, ok := statsByName[containers[i].Name]
		if !ok {
			continue
		}
		applyRuntimeMetric(&containers[i], metric)
	}
}

func parseRuntimeStats(raw []byte, capacity int) map[string]Metrics {
	if capacity < 0 {
		capacity = 0
	}
	stats := make(map[string]Metrics, capacity)
	for _, line := range fieldsByLine(string(raw)) {
		metric, err := parseStatsLine(line)
		if err != nil {
			continue
		}
		stats[metric.Service] = metric
	}
	return stats
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
