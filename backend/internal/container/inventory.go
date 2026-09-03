package container

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// RuntimeContainerNetwork is a compact network attachment shown in the host
// inventory. The inventory is read-only; network lifecycle remains project-owned.
type RuntimeContainerNetwork struct {
	Name      string `json:"name"`
	IPAddress string `json:"ipAddress"`
}

// RuntimeContainer is a host-wide read-only view of a container visible through
// the Docker-compatible runtime. It intentionally includes platform/control
// plane containers and unrelated host containers so owners can diagnose the
// whole single-node runtime from one page.
type RuntimeContainer struct {
	ID               string                    `json:"id"`
	Name             string                    `json:"name"`
	Image            string                    `json:"image"`
	State            string                    `json:"state"`
	Status           string                    `json:"status"`
	Uptime           string                    `json:"uptime"`
	Health           string                    `json:"health"`
	Ports            string                    `json:"ports"`
	RestartCount     int                       `json:"restartCount"`
	DetailsAvailable bool                      `json:"detailsAvailable"`
	Networks         []RuntimeContainerNetwork `json:"networks"`
	ComposeProject   string                    `json:"composeProject"`
	Service          string                    `json:"service"`
	CPUPercent       float64                   `json:"cpu"`
	CPUAvailable     bool                      `json:"cpuAvailable"`
	MemoryMB         float64                   `json:"memoryMb"`
	MemoryLimitMB    float64                   `json:"memoryLimitMb"`
	MemoryAvailable  bool                      `json:"memoryAvailable"`
	MetricsAvailable bool                      `json:"metricsAvailable"`
}

type dockerPSLine struct {
	ID         string `json:"ID"`
	Names      string `json:"Names"`
	Image      string `json:"Image"`
	State      string `json:"State"`
	Status     string `json:"Status"`
	RunningFor string `json:"RunningFor"`
	Ports      string `json:"Ports"`
	Labels     string `json:"Labels"`
}

type runtimeInventoryInspectRow struct {
	ID           string `json:"Id"`
	RestartCount int    `json:"RestartCount"`
	State        struct {
		Health *struct {
			Status string `json:"Status"`
		} `json:"Health"`
	} `json:"State"`
	NetworkSettings struct {
		Networks map[string]struct {
			IPAddress string `json:"IPAddress"`
		} `json:"Networks"`
	} `json:"NetworkSettings"`
}

const (
	runtimeStatsFormat              = "{{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}"
	runtimeTelemetryRefreshEvery    = 15 * time.Second
	runtimeTelemetryTimeout         = 12 * time.Second
	runtimeBulkTelemetryTimeout     = 4 * time.Second
	runtimeFallbackTelemetryTimeout = 2 * time.Second
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
			Uptime:         runtimeUptime(raw.RunningFor, raw.Status),
			Health:         runtimeHealth(raw.Status),
			Ports:          strings.TrimSpace(raw.Ports),
			Networks:       []RuntimeContainerNetwork{},
			ComposeProject: dockerLabel(raw.Labels, "com.docker.compose.project"),
			Service:        dockerLabel(raw.Labels, "com.docker.compose.service"),
		})
	}
	if containers == nil {
		containers = []RuntimeContainer{}
	}

	// Restart count and network attachments are useful diagnostics but should
	// never make the fast inventory unavailable. One batched inspect enriches
	// all rows; failures leave DetailsAvailable false instead of failing the page.
	_ = d.enrichRuntimeContainerDetails(ctx, containers)
	return containers, nil
}

func (d *DockerCLI) enrichRuntimeContainerDetails(ctx context.Context, containers []RuntimeContainer) error {
	ids := make([]string, 0, len(containers))
	for _, item := range containers {
		if strings.TrimSpace(item.ID) != "" {
			ids = append(ids, item.ID)
		}
	}
	if len(ids) == 0 {
		return nil
	}

	args := append([]string{"inspect"}, ids...)
	out, err := commandContext(ctx, "docker", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker inspect inventory details: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return applyRuntimeContainerDetails(containers, out)
}

func applyRuntimeContainerDetails(containers []RuntimeContainer, raw []byte) error {
	var details []runtimeInventoryInspectRow
	if err := json.Unmarshal(raw, &details); err != nil {
		return fmt.Errorf("decode docker inventory details: %w", err)
	}
	byID := make(map[string]runtimeInventoryInspectRow, len(details))
	for _, item := range details {
		if id := strings.TrimSpace(item.ID); id != "" {
			byID[id] = item
		}
	}

	for i := range containers {
		item, ok := byID[containers[i].ID]
		if !ok {
			continue
		}
		containers[i].DetailsAvailable = true
		containers[i].RestartCount = item.RestartCount
		if item.State.Health != nil && strings.TrimSpace(item.State.Health.Status) != "" {
			containers[i].Health = strings.ToLower(strings.TrimSpace(item.State.Health.Status))
		}
		containers[i].Networks = runtimeNetworks(item.NetworkSettings.Networks)
	}
	return nil
}

func runtimeNetworks(networks map[string]struct {
	IPAddress string `json:"IPAddress"`
}) []RuntimeContainerNetwork {
	out := make([]RuntimeContainerNetwork, 0, len(networks))
	for name, network := range networks {
		out = append(out, RuntimeContainerNetwork{
			Name:      strings.TrimSpace(name),
			IPAddress: strings.TrimSpace(network.IPAddress),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func runtimeHealth(status string) string {
	value := strings.ToLower(status)
	switch {
	case strings.Contains(value, "unhealthy"):
		return "unhealthy"
	case strings.Contains(value, "health: starting"):
		return "starting"
	case strings.Contains(value, "healthy"):
		return "healthy"
	default:
		return ""
	}
}

func runtimeUptime(runningFor, status string) string {
	status = strings.TrimSpace(status)
	if strings.HasPrefix(status, "Up ") {
		value := strings.TrimSpace(strings.TrimPrefix(status, "Up "))
		if index := strings.Index(value, " ("); index >= 0 {
			value = strings.TrimSpace(value[:index])
		}
		if value != "" {
			return value
		}
	}
	return strings.TrimSpace(runningFor)
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
	stats, err := d.collectRuntimeStats(ctx, containers)
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

// collectRuntimeStats prefers one targeted bulk stats call, retries through the
// runtime-wide bulk form when a Docker-compatible runtime omits rows, then uses
// a single-target command only for still-missing samples. All of this stays on
// the background path; user requests never wait on stats.
func (d *DockerCLI) collectRuntimeStats(ctx context.Context, containers []RuntimeContainer) (map[string]Metrics, error) {
	targets := runtimeStatsTargets(containers)
	if len(targets) == 0 {
		return map[string]Metrics{}, nil
	}

	targetSet := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		targetSet[target] = struct{}{}
	}
	stats := make(map[string]Metrics, len(targets))

	bulkOut, bulkErr := runRuntimeStatsCommand(ctx, targets)
	if bulkErr == nil {
		mergeTargetRuntimeStats(stats, parseRuntimeStats(bulkOut, len(targets)), targetSet)
	}

	if len(stats) < len(targets) && ctx.Err() == nil {
		allOut, allErr := runRuntimeStatsCommand(ctx, nil)
		if allErr == nil {
			mergeTargetRuntimeStats(stats, parseRuntimeStats(allOut, len(targets)), targetSet)
		}
	}

	for _, target := range targets {
		if _, ok := stats[target]; ok || ctx.Err() != nil {
			continue
		}
		fallbackCtx, cancel := context.WithTimeout(ctx, runtimeFallbackTelemetryTimeout)
		fallbackOut, err := runRuntimeStatsCommand(fallbackCtx, []string{target})
		cancel()
		if err == nil {
			mergeTargetRuntimeStats(stats, parseRuntimeStats(fallbackOut, 1), targetSet)
		}
	}

	if len(stats) == 0 && bulkErr != nil {
		return nil, fmt.Errorf("docker stats inventory: %w: %s", bulkErr, strings.TrimSpace(string(bulkOut)))
	}
	return stats, nil
}

func runRuntimeStatsCommand(ctx context.Context, targets []string) ([]byte, error) {
	bulkCtx, cancel := context.WithTimeout(ctx, runtimeBulkTelemetryTimeout)
	defer cancel()
	args := []string{"stats", "--no-stream", "--format", runtimeStatsFormat}
	args = append(args, targets...)
	return commandContext(bulkCtx, "docker", args...).CombinedOutput()
}

func mergeTargetRuntimeStats(dst, source map[string]Metrics, targets map[string]struct{}) {
	for name, metric := range source {
		if _, ok := targets[name]; !ok {
			continue
		}
		dst[name] = metric
	}
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
		metric, err := parseRuntimeInventoryStatsLine(line)
		if err != nil {
			continue
		}
		stats[metric.Service] = metric
	}
	return stats
}

// parseRuntimeInventoryStatsLine is deliberately more tolerant than the
// project-metrics parser. Podman can legitimately report `--` for one metric
// while still returning useful values for another; the host inventory keeps
// the partial sample instead of discarding the whole container row.
func parseRuntimeInventoryStatsLine(line string) (Metrics, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return Metrics{}, ErrNoContainer
	}

	name := ""
	cpuValue := ""
	memoryValue := ""
	if strings.HasPrefix(line, "{") {
		var raw dockerStatsLine
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			return Metrics{}, fmt.Errorf("parse docker stats inventory: %w", err)
		}
		name = strings.TrimSpace(raw.Name)
		cpuValue = raw.CPUPerc
		memoryValue = raw.MemUsage
	} else {
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) != 3 {
			return Metrics{}, fmt.Errorf("parse docker stats inventory columns: expected name, CPU, and memory usage")
		}
		name = strings.TrimSpace(parts[0])
		cpuValue = parts[1]
		memoryValue = parts[2]
	}
	if name == "" {
		return Metrics{}, fmt.Errorf("parse docker stats inventory: missing container name")
	}

	cpu, cpuAvailable, err := parseRuntimeMetricPercent(cpuValue)
	if err != nil {
		return Metrics{}, err
	}
	used, limit, memoryAvailable, err := parseRuntimeMetricMemory(memoryValue)
	if err != nil {
		return Metrics{}, err
	}
	if !cpuAvailable {
		cpu = math.NaN()
	}
	if !memoryAvailable {
		used = math.NaN()
	}

	return Metrics{
		Service:       name,
		CPUPercent:    cpu,
		MemoryMB:      used,
		MemoryLimitMB: limit,
	}, nil
}

func parseRuntimeMetricPercent(value string) (float64, bool, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || trimmed == "--" {
		return 0, false, nil
	}
	parsed, err := parsePercent(trimmed)
	if err != nil {
		return 0, false, err
	}
	return parsed, true, nil
}

func parseRuntimeMetricMemory(value string) (float64, float64, bool, error) {
	parts := strings.Split(value, "/")
	if len(parts) != 2 {
		return 0, 0, false, fmt.Errorf("parse memory usage %q", value)
	}

	used := math.NaN()
	limit := math.NaN()
	usedText := strings.TrimSpace(parts[0])
	limitText := strings.TrimSpace(parts[1])
	if usedText != "" && usedText != "--" {
		parsed, err := parseMemoryMB(usedText)
		if err != nil {
			return 0, 0, false, err
		}
		used = parsed
	}
	if limitText != "" && limitText != "--" {
		parsed, err := parseMemoryMB(limitText)
		if err != nil {
			return 0, 0, false, err
		}
		limit = parsed
	}
	return used, limit, !math.IsNaN(used), nil
}

func applyRuntimeMetric(container *RuntimeContainer, metric Metrics) {
	container.CPUAvailable = !math.IsNaN(metric.CPUPercent)
	container.MemoryAvailable = !math.IsNaN(metric.MemoryMB)
	if container.CPUAvailable {
		container.CPUPercent = metric.CPUPercent
	} else {
		container.CPUPercent = 0
	}
	if container.MemoryAvailable {
		container.MemoryMB = metric.MemoryMB
	} else {
		container.MemoryMB = 0
	}
	if !math.IsNaN(metric.MemoryLimitMB) {
		container.MemoryLimitMB = metric.MemoryLimitMB
	} else {
		container.MemoryLimitMB = 0
	}
	container.MetricsAvailable = container.CPUAvailable || container.MemoryAvailable
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
