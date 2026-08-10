package deployment

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"mypaas/internal/statd"
)

func TestStatdRuntimeID(t *testing.T) {
	projectID := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	got, err := statdRuntimeID(projectID, "api")
	if err != nil {
		t.Fatalf("statdRuntimeID() error = %v", err)
	}
	if got != "11111111-2222-3333-4444-555555555555:api" {
		t.Fatalf("statdRuntimeID() = %q", got)
	}

	if _, err := statdRuntimeID(projectID, ""); err == nil {
		t.Fatal("expected empty service error")
	}
	if _, err := statdRuntimeID(projectID, strings.Repeat("x", 100)); err == nil {
		t.Fatal("expected overlong runtime id error")
	}
}

func TestMetricFromStatd(t *testing.T) {
	cpu := 12.5
	memoryMax := uint64(512 * 1024 * 1024)
	snapshot := statd.Snapshot{
		Valid: true,
		CPU: statd.CPUSnapshot{Percent: &cpu},
		Memory: statd.MemorySnapshot{
			CurrentBytes: 64 * 1024 * 1024,
			MaxBytes:     &memoryMax,
		},
	}
	startedAt := time.Now().Add(-(2*time.Hour + 8*time.Minute))

	metric := metricFromStatd("api", snapshot, startedAt)
	if metric.Service != "api" {
		t.Fatalf("Service = %q, want api", metric.Service)
	}
	if metric.CPUPercent != 12.5 {
		t.Fatalf("CPUPercent = %f, want 12.5", metric.CPUPercent)
	}
	if metric.MemoryMB != 64 || metric.MemoryLimitMB != 512 {
		t.Fatalf("memory = %f/%f MiB", metric.MemoryMB, metric.MemoryLimitMB)
	}
	if metric.Uptime != "2h 8m" {
		t.Fatalf("Uptime = %q, want 2h 8m", metric.Uptime)
	}
	if metric.CollectedAt.IsZero() {
		t.Fatal("CollectedAt must be set")
	}
}

func TestMetricFromStatdNullableValues(t *testing.T) {
	snapshot := statd.Snapshot{
		Valid: true,
		Memory: statd.MemorySnapshot{
			CurrentBytes: 1024,
		},
	}
	metric := metricFromStatd("worker", snapshot, time.Time{})
	if metric.CPUPercent != 0 {
		t.Fatalf("CPUPercent = %f, want 0 while unavailable", metric.CPUPercent)
	}
	if metric.MemoryLimitMB != 0 {
		t.Fatalf("MemoryLimitMB = %f, want 0 for unlimited", metric.MemoryLimitMB)
	}
	if metric.Uptime != "unknown" {
		t.Fatalf("Uptime = %q, want unknown", metric.Uptime)
	}
}

func TestFormatRuntimeUptime(t *testing.T) {
	tests := []struct {
		name string
		in   time.Duration
		want string
	}{
		{name: "negative", in: -time.Second, want: "unknown"},
		{name: "seconds", in: 30 * time.Second, want: "<1m"},
		{name: "minutes", in: 17 * time.Minute, want: "17m"},
		{name: "hours", in: 2*time.Hour + 8*time.Minute, want: "2h 8m"},
		{name: "days", in: 49*time.Hour + 30*time.Minute, want: "2d 1h"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatRuntimeUptime(tt.in); got != tt.want {
				t.Fatalf("formatRuntimeUptime() = %q, want %q", got, tt.want)
			}
		})
	}
}
