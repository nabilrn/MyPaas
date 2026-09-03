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

	if !containers[0].MetricsAvailable {
		t.Fatal("expected app-a metrics to be available")
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

func TestApplyCachedRuntimeTelemetry(t *testing.T) {
	cli := NewDockerCLI("127.0.0.1")
	cli.runtimeTelemetry["app-a"] = Metrics{CPUPercent: 1.25, MemoryMB: 128, MemoryLimitMB: 512}
	containers := []RuntimeContainer{
		{Name: "app-a", State: "running"},
		{Name: "app-b", State: "running"},
	}

	cli.applyCachedRuntimeTelemetry(containers)
	if !containers[0].MetricsAvailable {
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
