package deployment

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"mypaas/internal/container"
	"mypaas/internal/db"
	"mypaas/internal/statd"
)

const (
	statdSocketEnv               = "STATD_SOCKET"
	statdRuntimeCacheMaxProjects = 128
)

type statdRuntimeCacheEntry struct {
	activeDeployment      [16]byte
	activeDeploymentValid bool
	runtimes              []container.RuntimeProcess
}

// statdRuntimeCache is owned by the deployment Handler. It remembers only
// cold-path Docker runtime metadata needed to address statd snapshots. The
// cache is deliberately small and bounded; it is not a source of truth.
type statdRuntimeCache struct {
	mu      sync.RWMutex
	entries map[uuid.UUID]statdRuntimeCacheEntry
}

func (c *statdRuntimeCache) get(project db.Project) ([]container.RuntimeProcess, bool) {
	if c == nil {
		return nil, false
	}
	c.mu.RLock()
	entry, ok := c.entries[project.ID]
	c.mu.RUnlock()
	if !ok || entry.activeDeploymentValid != project.ActiveDeploymentID.Valid ||
		entry.activeDeployment != project.ActiveDeploymentID.Bytes {
		return nil, false
	}
	return cloneRuntimeProcesses(entry.runtimes), true
}

func (c *statdRuntimeCache) put(project db.Project, runtimes []container.RuntimeProcess) {
	if c == nil || len(runtimes) == 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entries == nil {
		c.entries = make(map[uuid.UUID]statdRuntimeCacheEntry)
	}
	if _, exists := c.entries[project.ID]; !exists && len(c.entries) >= statdRuntimeCacheMaxProjects {
		clear(c.entries)
	}
	c.entries[project.ID] = statdRuntimeCacheEntry{
		activeDeployment:      project.ActiveDeploymentID.Bytes,
		activeDeploymentValid: project.ActiveDeploymentID.Valid,
		runtimes:              cloneRuntimeProcesses(runtimes),
	}
}

func (c *statdRuntimeCache) invalidate(projectID uuid.UUID) {
	if c == nil {
		return
	}
	c.mu.Lock()
	delete(c.entries, projectID)
	c.mu.Unlock()
}

func cloneRuntimeProcesses(runtimes []container.RuntimeProcess) []container.RuntimeProcess {
	if len(runtimes) == 0 {
		return nil
	}
	out := make([]container.RuntimeProcess, len(runtimes))
	copy(out, runtimes)
	return out
}

func singleRuntimeFromCache(cache *statdRuntimeCache, project db.Project, service string) (container.RuntimeProcess, bool) {
	runtimes, ok := cache.get(project)
	if !ok || len(runtimes) != 1 {
		return container.RuntimeProcess{}, false
	}
	runtime := runtimes[0]
	if runtime.PID <= 0 || strings.TrimSpace(runtime.Service) != strings.TrimSpace(service) {
		return container.RuntimeProcess{}, false
	}
	return runtime, true
}

// PreferredContainerMetricsList uses mypaas-statd when it is explicitly
// configured and falls back to the existing Docker metrics path on any statd
// availability/integration failure. This keeps rollout reversible.
func (s *Service) PreferredContainerMetricsList(ctx context.Context, projectID uuid.UUID, cache *statdRuntimeCache) ([]container.Metrics, error) {
	project, err := s.project(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if project.DeployMode == "static" || !hasLiveRuntime(project.Status) {
		return s.ContainerMetricsList(ctx, projectID)
	}

	socketPath := strings.TrimSpace(os.Getenv(statdSocketEnv))
	if socketPath == "" {
		return s.ContainerMetricsList(ctx, projectID)
	}

	client := statd.NewClient(socketPath)
	var metrics []container.Metrics
	if project.DeployMode == "compose" {
		metrics, err = s.composeStatdMetrics(ctx, project, client, cache)
	} else {
		metrics, err = s.singleStatdMetrics(ctx, project, client, cache)
	}
	if err == nil && len(metrics) > 0 {
		if statd.MarkAvailable(true) == statd.AvailabilityRecovered {
			slog.Info("statd metrics path recovered")
		}
		return metrics, nil
	}

	transition := statd.MarkAvailable(false)
	statd.RecordFallback()
	if transition == statd.AvailabilityInitialUnavailable || transition == statd.AvailabilityLost {
		slog.Warn("statd metrics unavailable; using Docker fallback",
			"projectId", project.ID,
			"mode", project.DeployMode,
			"error", err,
		)
	} else {
		slog.Debug("statd metrics unavailable; using Docker fallback",
			"projectId", project.ID,
			"mode", project.DeployMode,
			"error", err,
		)
	}
	return s.ContainerMetricsList(ctx, projectID)
}

func (s *Service) singleStatdMetrics(ctx context.Context, project db.Project, client *statd.Client, cache *statdRuntimeCache) ([]container.Metrics, error) {
	service := mainService(project)
	if strings.TrimSpace(service) == "" {
		service = "app"
	}
	id, err := statdRuntimeID(project.ID, service)
	if err != nil {
		return nil, err
	}

	// Keep StartedAt/PID in the same bounded handler-owned cache used by Compose.
	// A cold cache performs one inspect, while steady-state statd reads do not
	// need Docker process discovery merely to preserve uptime.
	runtime, cached := singleRuntimeFromCache(cache, project, service)
	if !cached {
		runtime, err = s.docker.RuntimeProcess(ctx, containerName(project.Name))
		if err != nil {
			return nil, err
		}
		runtime.Service = service
		cache.put(project, []container.RuntimeProcess{runtime})
	}

	snapshot, snapshotErr := client.Snapshot(ctx, id)
	recordUnexpectedSnapshotError(snapshotErr)
	if snapshotErr != nil || snapshot.Stale {
		if snapshot.Stale {
			_ = client.Unregister(ctx, id)
		}
		if err := registerAndSnapshot(ctx, client, id, runtime.PID, &snapshot); err != nil {
			if !cached || !runtimeIdentityMayBeStale(err) {
				return nil, err
			}

			// A cached PID can become invalid after an out-of-band runtime
			// replacement. Refresh only after statd explicitly rejects the
			// cached identity; daemon/socket failures go directly to fallback.
			cache.invalidate(project.ID)
			freshRuntime, discoveryErr := s.docker.RuntimeProcess(ctx, containerName(project.Name))
			if discoveryErr != nil {
				return nil, fmt.Errorf("refresh single runtime metadata after statd failure: %w", discoveryErr)
			}
			freshRuntime.Service = service
			if retryErr := registerAndSnapshot(ctx, client, id, freshRuntime.PID, &snapshot); retryErr != nil {
				return nil, retryErr
			}
			runtime = freshRuntime
			cache.put(project, []container.RuntimeProcess{freshRuntime})
		}
	}

	return []container.Metrics{metricFromStatd(service, snapshot, runtime.StartedAt)}, nil
}

func (s *Service) composeStatdMetrics(ctx context.Context, project db.Project, client *statd.Client, cache *statdRuntimeCache) ([]container.Metrics, error) {
	runtimes, cached := cache.get(project)
	if !cached {
		var err error
		runtimes, err = s.docker.ComposeRuntimeProcesses(ctx, composeProjectName(project.Name))
		if err != nil {
			return nil, err
		}
	}

	metrics, err := composeStatdSnapshots(ctx, project, client, runtimes)
	if err == nil {
		cache.put(project, runtimes)
		return metrics, nil
	}
	if !cached || !runtimeIdentityMayBeStale(err) {
		return nil, err
	}

	// Cached PIDs normally let statd re-register after a daemon restart without
	// touching Docker. Re-discover only when statd explicitly rejects the
	// cached identity (container replacement/restart outside known lifecycle).
	cache.invalidate(project.ID)
	freshRuntimes, discoveryErr := s.docker.ComposeRuntimeProcesses(ctx, composeProjectName(project.Name))
	if discoveryErr != nil {
		return nil, fmt.Errorf("refresh compose runtime metadata after statd failure: %w", discoveryErr)
	}
	metrics, err = composeStatdSnapshots(ctx, project, client, freshRuntimes)
	if err != nil {
		return nil, err
	}
	cache.put(project, freshRuntimes)
	return metrics, nil
}

func composeStatdSnapshots(ctx context.Context, project db.Project, client *statd.Client, runtimes []container.RuntimeProcess) ([]container.Metrics, error) {
	if client == nil || len(runtimes) == 0 {
		return nil, errors.New("statd client and compose runtimes are required")
	}
	metrics := make([]container.Metrics, 0, len(runtimes))
	for _, runtime := range runtimes {
		id, err := statdRuntimeID(project.ID, runtime.Service)
		if err != nil {
			return nil, err
		}
		snapshot, snapshotErr := client.Snapshot(ctx, id)
		recordUnexpectedSnapshotError(snapshotErr)
		if snapshotErr != nil || snapshot.Stale {
			if snapshot.Stale {
				_ = client.Unregister(ctx, id)
			}
			if err := registerAndSnapshot(ctx, client, id, runtime.PID, &snapshot); err != nil {
				return nil, err
			}
		}
		metrics = append(metrics, metricFromStatd(runtime.Service, snapshot, runtime.StartedAt))
	}
	sort.Slice(metrics, func(i, j int) bool { return metrics[i].Service < metrics[j].Service })
	return metrics, nil
}

func recordUnexpectedSnapshotError(err error) {
	if err == nil {
		return
	}
	var protocolErr *statd.ProtocolError
	if errors.As(err, &protocolErr) && protocolErr.Code == "NOT_FOUND" {
		return
	}
	statd.RecordSnapshotError()
}

func runtimeIdentityMayBeStale(err error) bool {
	var protocolErr *statd.ProtocolError
	return errors.As(err, &protocolErr) && protocolErr.Code == "REGISTER_FAILED"
}

func registerAndSnapshot(ctx context.Context, client *statd.Client, id string, pid int, out *statd.Snapshot) error {
	if client == nil || out == nil {
		return errors.New("statd client and output are required")
	}
	if err := client.Register(ctx, id, pid); err != nil {
		var protocolErr *statd.ProtocolError
		if errors.As(err, &protocolErr) && protocolErr.Code == "REGISTER_FAILED" {
			if unregisterErr := client.Unregister(ctx, id); unregisterErr != nil {
				statd.RecordRegistrationError()
				return fmt.Errorf("replace statd registration: unregister: %w", unregisterErr)
			}
			if retryErr := client.Register(ctx, id, pid); retryErr != nil {
				statd.RecordRegistrationError()
				return fmt.Errorf("replace statd registration: register: %w", retryErr)
			}
		} else {
			statd.RecordRegistrationError()
			return err
		}
	}
	snapshot, err := client.Snapshot(ctx, id)
	if err != nil {
		statd.RecordSnapshotError()
		return err
	}
	*out = snapshot
	return nil
}

func statdRuntimeID(projectID uuid.UUID, service string) (string, error) {
	service = strings.TrimSpace(service)
	if service == "" {
		return "", errors.New("statd runtime service is required")
	}
	id := projectID.String() + ":" + service
	if len(id) > 127 {
		return "", fmt.Errorf("statd runtime id exceeds 127 bytes")
	}
	return id, nil
}

func metricFromStatd(service string, snapshot statd.Snapshot, startedAt time.Time) container.Metrics {
	cpu := 0.0
	if snapshot.CPU.Percent != nil {
		cpu = *snapshot.CPU.Percent
	}
	memoryLimitMB := 0.0
	if snapshot.Memory.MaxBytes != nil {
		memoryLimitMB = bytesToMiB(*snapshot.Memory.MaxBytes)
	}
	uptime := "unknown"
	if !startedAt.IsZero() {
		uptime = formatRuntimeUptime(time.Since(startedAt))
	}
	return container.Metrics{
		Service:       service,
		CPUPercent:    cpu,
		MemoryMB:      bytesToMiB(snapshot.Memory.CurrentBytes),
		MemoryLimitMB: memoryLimitMB,
		Uptime:        uptime,
		CollectedAt:   time.Now().UTC(),
	}
}

func bytesToMiB(value uint64) float64 {
	return float64(value) / (1024.0 * 1024.0)
}

func formatRuntimeUptime(duration time.Duration) string {
	if duration < 0 {
		return "unknown"
	}
	if duration < time.Minute {
		return "<1m"
	}
	if duration < time.Hour {
		return fmt.Sprintf("%dm", int(duration.Minutes()))
	}
	if duration < 24*time.Hour {
		hours := int(duration.Hours())
		minutes := int(duration.Minutes()) % 60
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	days := int(duration.Hours()) / 24
	hours := int(duration.Hours()) % 24
	return fmt.Sprintf("%dd %dh", days, hours)
}
