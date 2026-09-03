package container

import (
	"reflect"
	"testing"
)

func TestRuntimeStatsTargetsOnlyRunningContainers(t *testing.T) {
	containers := []RuntimeContainer{
		{Name: "app-a", State: "running"},
		{Name: "app-b", State: "exited"},
		{Name: "app-c", State: "running"},
		{Name: "", State: "running"},
	}

	got := runtimeStatsTargets(containers)
	want := []string{"app-a", "app-c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("runtimeStatsTargets() = %#v, want %#v", got, want)
	}
}

func TestMergeRuntimeStatsMarksOnlyMatchedRowsAvailable(t *testing.T) {
	containers := []RuntimeContainer{
		{Name: "app-a", State: "running"},
		{Name: "app-b", State: "running"},
	}

	mergeRuntimeStats(containers, []byte("app-a\t1.25%\t128MiB / 512MiB\n"))

	if !containers[0].MetricsAvailable || !containers[0].CPUAvailable || !containers[0].MemoryAvailable {
		t.Fatal("expected app-a CPU and memory metrics to be available")
	}
	if containers[0].CPUPercent != 1.25 {
		t.Fatalf("CPUPercent = %v, want 1.25", containers[0].CPUPercent)
	}
	if containers[0].MemoryMB != 128 || containers[0].MemoryLimitMB != 512 {
		t.Fatalf("memory = %v / %v, want 128 / 512", containers[0].MemoryMB, containers[0].MemoryLimitMB)
	}
	if containers[1].MetricsAvailable {
		t.Fatal("expected unmatched app-b metrics to remain unavailable")
	}
}

func TestRuntimeStatsKeepsMemoryWhenPodmanCPUIsUnavailable(t *testing.T) {
	containers := []RuntimeContainer{{Name: "app-a", State: "running"}}

	mergeRuntimeStats(containers, []byte("app-a\t--\t128MiB / 512MiB\n"))

	if !containers[0].MetricsAvailable {
		t.Fatal("expected partial telemetry sample to remain available")
	}
	if containers[0].CPUAvailable {
		t.Fatal("CPUAvailable = true, want false for Podman -- sample")
	}
	if !containers[0].MemoryAvailable {
		t.Fatal("MemoryAvailable = false, want true")
	}
	if containers[0].CPUPercent != 0 {
		t.Fatalf("CPUPercent = %v, want presentation fallback 0", containers[0].CPUPercent)
	}
	if containers[0].MemoryMB != 128 || containers[0].MemoryLimitMB != 512 {
		t.Fatalf("memory = %v / %v, want 128 / 512", containers[0].MemoryMB, containers[0].MemoryLimitMB)
	}
}

func TestRuntimeStatsTreatsFullyUnavailableSampleAsUnavailable(t *testing.T) {
	containers := []RuntimeContainer{{Name: "app-a", State: "running"}}

	mergeRuntimeStats(containers, []byte("app-a\t--\t-- / --\n"))

	if containers[0].MetricsAvailable || containers[0].CPUAvailable || containers[0].MemoryAvailable {
		t.Fatalf("unexpected telemetry availability: %+v", containers[0])
	}
}

func TestApplyCachedRuntimeTelemetry(t *testing.T) {
	cli := NewDockerCLI("127.0.0.1")
	cli.runtimeTelemetry["app-a"] = Metrics{CPUPercent: 1.25, MemoryMB: 128, MemoryLimitMB: 512}
	containers := []RuntimeContainer{
		{Name: "app-a", State: "running"},
		{Name: "app-b", State: "running"},
	}

	cli.applyCachedRuntimeTelemetry(containers)
	if !containers[0].MetricsAvailable || !containers[0].CPUAvailable || !containers[0].MemoryAvailable {
		t.Fatal("expected cached app-a metrics to be available")
	}
	if containers[0].CPUPercent != 1.25 || containers[0].MemoryMB != 128 || containers[0].MemoryLimitMB != 512 {
		t.Fatalf("cached metrics = %+v", containers[0])
	}
	if containers[1].MetricsAvailable {
		t.Fatal("expected cache miss for app-b")
	}
}

func TestApplyCachedRuntimeTelemetrySkipsStoppedContainer(t *testing.T) {
	cli := NewDockerCLI("127.0.0.1")
	cli.runtimeTelemetry["app-a"] = Metrics{CPUPercent: 9.5, MemoryMB: 256, MemoryLimitMB: 512}
	containers := []RuntimeContainer{{Name: "app-a", State: "exited"}}

	cli.applyCachedRuntimeTelemetry(containers)

	if containers[0].MetricsAvailable {
		t.Fatal("stopped container must not expose stale cached telemetry")
	}
	if containers[0].CPUPercent != 0 || containers[0].MemoryMB != 0 || containers[0].MemoryLimitMB != 0 {
		t.Fatalf("stopped container inherited stale telemetry: %+v", containers[0])
	}
}

func TestRuntimeHealth(t *testing.T) {
	tests := map[string]string{
		"Up 2 minutes (healthy)":          "healthy",
		"Up 2 minutes (unhealthy)":        "unhealthy",
		"Up 2 minutes (health: starting)": "starting",
		"Up 2 minutes":                    "",
	}
	for input, want := range tests {
		if got := runtimeHealth(input); got != want {
			t.Fatalf("runtimeHealth(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestRuntimeUptime(t *testing.T) {
	if got := runtimeUptime("2 hours ago", "Up 41 minutes (healthy)"); got != "41 minutes" {
		t.Fatalf("runtimeUptime() = %q, want %q", got, "41 minutes")
	}
	if got := runtimeUptime("3 hours ago", "Exited (0) 1 minute ago"); got != "3 hours ago" {
		t.Fatalf("runtimeUptime() fallback = %q, want %q", got, "3 hours ago")
	}
}

func TestApplyRuntimeContainerDetails(t *testing.T) {
	rows := []RuntimeContainer{{ID: "abc", Name: "app", Networks: []RuntimeContainerNetwork{}}}
	raw := []byte(`[{"Id":"abc","RestartCount":4,"State":{"Health":{"Status":"healthy"}},"NetworkSettings":{"Networks":{"mypaas-routing":{"IPAddress":"10.89.2.4"},"mypaas-control":{"IPAddress":"10.89.0.7"}}}}]`)

	if err := applyRuntimeContainerDetails(rows, raw); err != nil {
		t.Fatalf("applyRuntimeContainerDetails() error = %v", err)
	}
	if !rows[0].DetailsAvailable {
		t.Fatal("DetailsAvailable = false, want true")
	}
	if rows[0].RestartCount != 4 {
		t.Fatalf("RestartCount = %d, want 4", rows[0].RestartCount)
	}
	if rows[0].Health != "healthy" {
		t.Fatalf("Health = %q, want healthy", rows[0].Health)
	}
	if len(rows[0].Networks) != 2 || rows[0].Networks[0].Name != "mypaas-control" || rows[0].Networks[1].Name != "mypaas-routing" {
		t.Fatalf("Networks = %#v, want sorted network attachments", rows[0].Networks)
	}
}

func TestMergeTargetRuntimeStatsFiltersUnrelatedContainers(t *testing.T) {
	dst := map[string]Metrics{}
	source := map[string]Metrics{
		"app":   {Service: "app", CPUPercent: 1.5},
		"other": {Service: "other", CPUPercent: 9.9},
	}
	targets := map[string]struct{}{"app": {}}

	mergeTargetRuntimeStats(dst, source, targets)
	if len(dst) != 1 || dst["app"].CPUPercent != 1.5 {
		t.Fatalf("merged stats = %#v, want only app", dst)
	}
}
