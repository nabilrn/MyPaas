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

func TestMissingRuntimeStatsIndicesRetriesOnlyUnmatchedRunningRows(t *testing.T) {
	containers := []RuntimeContainer{
		{Name: "app-a", State: "running", MetricsAvailable: true},
		{Name: "app-b", State: "running"},
		{Name: "app-c", State: "exited"},
		{Name: "", State: "running"},
		{Name: "app-d", State: "running"},
	}

	got := missingRuntimeStatsIndices(containers)
	want := []int{1, 4}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("missingRuntimeStatsIndices() = %#v, want %#v", got, want)
	}
}

func TestMissingRuntimeStatsIndicesAfterPartialBatchMerge(t *testing.T) {
	containers := []RuntimeContainer{
		{Name: "app-a", State: "running"},
		{Name: "app-b", State: "running"},
	}

	mergeRuntimeStats(containers, []byte("app-a\t0.50%\t64MiB / 256MiB\n"))

	got := missingRuntimeStatsIndices(containers)
	want := []int{1}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("missingRuntimeStatsIndices() = %#v, want %#v", got, want)
	}
}
