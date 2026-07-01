package deployment

import (
	"time"

	"mypaas/internal/container"
)

type MetricsSnapshotResponse struct {
	Items       []ContainerMetricsResponse `json:"items"`
	CollectedAt string                     `json:"collectedAt"`
}

type ContainerMetricsResponse struct {
	Service       string  `json:"service"`
	CPU           float64 `json:"cpu"`
	MemoryMB      float64 `json:"memoryMb"`
	MemoryLimitMB float64 `json:"memoryLimitMb"`
	Uptime        string  `json:"uptime"`
}

func MetricsSnapshotFromContainer(metrics container.Metrics) MetricsSnapshotResponse {
	return MetricsSnapshotResponse{
		Items: []ContainerMetricsResponse{
			{
				Service:       metrics.Service,
				CPU:           metrics.CPUPercent,
				MemoryMB:      metrics.MemoryMB,
				MemoryLimitMB: metrics.MemoryLimitMB,
				Uptime:        metrics.Uptime,
			},
		},
		CollectedAt: formatCollectedAt(metrics.CollectedAt),
	}
}

func formatCollectedAt(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
