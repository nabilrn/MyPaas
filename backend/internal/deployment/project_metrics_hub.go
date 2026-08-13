package deployment

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

type projectMetricsSampler func(context.Context, uuid.UUID) (MetricsSnapshotResponse, error)

type projectMetricsHub struct {
	mu       sync.Mutex
	interval time.Duration
	sample   projectMetricsSampler
	topics   map[uuid.UUID]*projectMetricsTopic
}

type projectMetricsTopic struct {
	cancel      context.CancelFunc
	subscribers map[chan MetricsSnapshotResponse]struct{}
	latest      MetricsSnapshotResponse
	hasLatest   bool
}

func newProjectMetricsHub(interval time.Duration, sampler projectMetricsSampler) *projectMetricsHub {
	return &projectMetricsHub{
		interval: interval,
		sample:   sampler,
		topics:   make(map[uuid.UUID]*projectMetricsTopic),
	}
}

func (h *projectMetricsHub) subscribe(projectID uuid.UUID) (<-chan MetricsSnapshotResponse, func()) {
	ch := make(chan MetricsSnapshotResponse, 1)

	h.mu.Lock()
	topic := h.topics[projectID]
	if topic == nil {
		ctx, cancel := context.WithCancel(context.Background())
		topic = &projectMetricsTopic{
			cancel:      cancel,
			subscribers: make(map[chan MetricsSnapshotResponse]struct{}),
		}
		h.topics[projectID] = topic
		go h.run(ctx, projectID, topic)
	}
	topic.subscribers[ch] = struct{}{}
	if topic.hasLatest {
		ch <- topic.latest
	}
	h.mu.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			h.mu.Lock()
			current := h.topics[projectID]
			if current == topic {
				delete(topic.subscribers, ch)
				if len(topic.subscribers) == 0 {
					delete(h.topics, projectID)
					topic.cancel()
				}
			}
			close(ch)
			h.mu.Unlock()
		})
	}
	return ch, unsubscribe
}

func (h *projectMetricsHub) run(ctx context.Context, projectID uuid.UUID, topic *projectMetricsTopic) {
	h.collect(ctx, projectID, topic)
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.collect(ctx, projectID, topic)
		}
	}
}

func (h *projectMetricsHub) collect(ctx context.Context, projectID uuid.UUID, topic *projectMetricsTopic) {
	snapshot, err := h.sample(ctx, projectID)
	if err != nil {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	if h.topics[projectID] != topic {
		return
	}
	topic.latest = snapshot
	topic.hasLatest = true
	for subscriber := range topic.subscribers {
		select {
		case subscriber <- snapshot:
		default:
			select {
			case <-subscriber:
			default:
			}
			select {
			case subscriber <- snapshot:
			default:
			}
		}
	}
}
