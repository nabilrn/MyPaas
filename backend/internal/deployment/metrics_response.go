package deployment

import (
	"strings"
	"time"

	"mypaas/internal/container"
)

type MetricsSnapshotResponse struct {
	Items       []ContainerMetricsResponse `json:"items"`
	Analytics   *CloudflareAnalytics       `json:"analytics,omitempty"`
	CollectedAt string                     `json:"collectedAt"`
}

type CloudflareAnalytics struct {
	TotalRequests int                   `json:"total_requests"`
	Bandwidth     int                   `json:"bandwidth"`
	Errors        int                   `json:"errors"`
	Timeseries    []TimeseriesDataPoint `json:"timeseries"`
}

type TimeseriesDataPoint struct {
	Timestamp string `json:"timestamp"`
	Requests  int    `json:"requests"`
	Bandwidth int    `json:"bandwidth"`
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
		if strings.EqualFold(strings.TrimSpace(item.Uptime), "n/a") {
			continue
		}
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
