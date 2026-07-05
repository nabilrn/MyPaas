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
	return MetricsSnapshotFromContainers([]container.Metrics{metrics})
}

func MetricsSnapshotFromContainers(metrics []container.Metrics) MetricsSnapshotResponse {
	items := make([]ContainerMetricsResponse, 0, len(metrics))
	var collectedAt time.Time
	for _, item := range metrics {
		if collectedAt.IsZero() || item.CollectedAt.After(collectedAt) {
			collectedAt = item.CollectedAt
		}
		items = append(items, ContainerMetricsResponse{
			Service:       item.Service,
			CPU:           item.CPUPercent,
			MemoryMB:      item.MemoryMB,
			MemoryLimitMB: item.MemoryLimitMB,
			Uptime:        item.Uptime,
		})
	}
	return MetricsSnapshotResponse{
		Items:       items,
		CollectedAt: formatCollectedAt(collectedAt),
	}
}

func formatCollectedAt(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
