package deployment

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestProjectMetricsHubSharesSamplerAcrossSubscribers(t *testing.T) {
	var calls atomic.Int32
	projectID := uuid.New()
	hub := newProjectMetricsHub(20*time.Millisecond, func(context.Context, uuid.UUID) (MetricsSnapshotResponse, error) {
		calls.Add(1)
		return MetricsSnapshotResponse{CollectedAt: time.Now().UTC().Format(time.RFC3339)}, nil
	})

	first, unsubscribeFirst := hub.subscribe(projectID)
	defer unsubscribeFirst()
	second, unsubscribeSecond := hub.subscribe(projectID)
	defer unsubscribeSecond()

	awaitMetricSnapshot(t, first)
	awaitMetricSnapshot(t, second)
	start := calls.Load()
	if start < 1 {
		t.Fatal("expected sampler to run")
	}

	time.Sleep(55 * time.Millisecond)
	got := calls.Load() - start
	if got > 4 {
		t.Fatalf("expected one shared sampler loop, got %d additional samples", got)
	}
}

func TestProjectMetricsHubStopsAfterLastSubscriber(t *testing.T) {
	var calls atomic.Int32
	projectID := uuid.New()
	hub := newProjectMetricsHub(10*time.Millisecond, func(context.Context, uuid.UUID) (MetricsSnapshotResponse, error) {
		calls.Add(1)
		return MetricsSnapshotResponse{}, nil
	})

	stream, unsubscribe := hub.subscribe(projectID)
	awaitMetricSnapshot(t, stream)
	unsubscribe()
	before := calls.Load()
	time.Sleep(35 * time.Millisecond)
	after := calls.Load()
	if after > before+1 {
		t.Fatalf("sampler kept running after final unsubscribe: before=%d after=%d", before, after)
	}
}

func TestParseProjectStreamTopics(t *testing.T) {
	topics := parseProjectStreamTopics("status,metrics")
	if !topics.has(streamTopicStatus) || !topics.has(streamTopicMetrics) {
		t.Fatal("expected requested status and metrics topics")
	}
	if topics.has(streamTopicLogs) || topics.has(streamTopicDeployment) {
		t.Fatal("unexpected expensive stream topics")
	}

	legacy := parseProjectStreamTopics("")
	for _, topic := range []projectStreamTopic{streamTopicStatus, streamTopicMetrics, streamTopicLogs, streamTopicDeployment} {
		if !legacy.has(topic) {
			t.Fatalf("legacy stream should keep %s for backward compatibility", topic)
		}
	}
}

func awaitMetricSnapshot(t *testing.T, stream <-chan MetricsSnapshotResponse) MetricsSnapshotResponse {
	t.Helper()
	select {
	case snapshot := <-stream:
		return snapshot
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for metrics snapshot")
		return MetricsSnapshotResponse{}
	}
}
