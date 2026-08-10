package deployment

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"mypaas/internal/container"
	"mypaas/internal/db"
	"mypaas/internal/statd"
)

const statdSocketEnv = "STATD_SOCKET"

// PreferredContainerMetricsList uses mypaas-statd when it is explicitly
// configured and falls back to the existing Docker metrics path on any statd
// availability/integration failure. This keeps rollout reversible.
func (s *Service) PreferredContainerMetricsList(ctx context.Context, projectID uuid.UUID) ([]container.Metrics, error) {
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
		metrics, err = s.composeStatdMetrics(ctx, project, client)
	} else {
		metrics, err = s.singleStatdMetrics(ctx, project, client)
	}
	if err == nil && len(metrics) > 0 {
		return metrics, nil
	}

	slog.Debug("statd metrics unavailable; using Docker fallback",
		"projectId", project.ID,
		"mode", project.DeployMode,
		"error", err,
	)
	return s.ContainerMetricsList(ctx, projectID)
}

func (s *Service) singleStatdMetrics(ctx context.Context, project db.Project, client *statd.Client) ([]container.Metrics, error) {
	service := mainService(project)
	if strings.TrimSpace(service) == "" {
		service = "app"
	}
	id, err := statdRuntimeID(project.ID, service)
	if err != nil {
		return nil, err
	}

	snapshot, err := client.Snapshot(ctx, id)
	startedAt := time.Time{}
	if err != nil || snapshot.Stale {
		runtime, runtimeErr := s.docker.RuntimeProcess(ctx, containerName(project.Name))
		if runtimeErr != nil {
			return nil, runtimeErr
		}
		startedAt = runtime.StartedAt
		if snapshot.Stale {
			_ = client.Unregister(ctx, id)
		}
		if err := registerAndSnapshot(ctx, client, id, runtime.PID, &snapshot); err != nil {
			return nil, err
		}
	}

	return []container.Metrics{metricFromStatd(service, snapshot, startedAt)}, nil
}

func (s *Service) composeStatdMetrics(ctx context.Context, project db.Project, client *statd.Client) ([]container.Metrics, error) {
	runtimes, err := s.docker.ComposeRuntimeProcesses(ctx, composeProjectName(project.Name))
	if err != nil {
		return nil, err
	}

	metrics := make([]container.Metrics, 0, len(runtimes))
	for _, runtime := range runtimes {
		id, err := statdRuntimeID(project.ID, runtime.Service)
		if err != nil {
			return nil, err
		}
		snapshot, snapshotErr := client.Snapshot(ctx, id)
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

func registerAndSnapshot(ctx context.Context, client *statd.Client, id string, pid int, out *statd.Snapshot) error {
	if client == nil || out == nil {
		return errors.New("statd client and output are required")
	}
	if err := client.Register(ctx, id, pid); err != nil {
		var protocolErr *statd.ProtocolError
		if errors.As(err, &protocolErr) && protocolErr.Code == "REGISTER_FAILED" {
			if unregisterErr := client.Unregister(ctx, id); unregisterErr != nil {
				return fmt.Errorf("replace statd registration: unregister: %w", unregisterErr)
			}
			if retryErr := client.Register(ctx, id, pid); retryErr != nil {
				return fmt.Errorf("replace statd registration: register: %w", retryErr)
			}
		} else {
			return err
		}
	}
	snapshot, err := client.Snapshot(ctx, id)
	if err != nil {
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
