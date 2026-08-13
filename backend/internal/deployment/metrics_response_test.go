package deployment

import (
	"testing"
	"time"

	"mypaas/internal/container"
)

func TestMetricsSnapshotSkipsSyntheticIdleSamples(t *testing.T) {
	now := time.Date(2026, 8, 13, 9, 0, 0, 0, time.UTC)
	resp := MetricsSnapshotFromContainers([]container.Metrics{
		{Service: "static", CPUPercent: 0, MemoryMB: 0, MemoryLimitMB: 64, Uptime: "n/a", CollectedAt: now},
		{Service: "app", CPUPercent: 3.5, MemoryMB: 12, MemoryLimitMB: 256, Uptime: "2m", CollectedAt: now.Add(time.Second)},
	})
	if len(resp.Items) != 1 {
		t.Fatalf("expected one live runtime metric, got %d", len(resp.Items))
	}
	if resp.Items[0].Service != "app" {
		t.Fatalf("expected app metric, got %q", resp.Items[0].Service)
	}
	if resp.CollectedAt != now.Add(time.Second).Format(time.RFC3339) {
		t.Fatalf("unexpected collectedAt %q", resp.CollectedAt)
	}
}

func TestMetricsSnapshotWithOnlyIdleSamplesHasNoCollectionTime(t *testing.T) {
	resp := MetricsSnapshotFromContainers([]container.Metrics{{Service: "static", Uptime: "n/a", CollectedAt: time.Now().UTC()}})
	if len(resp.Items) != 0 {
		t.Fatalf("expected no runtime metrics, got %d", len(resp.Items))
	}
	if resp.CollectedAt != "" {
		t.Fatalf("expected empty collectedAt, got %q", resp.CollectedAt)
	}
}
