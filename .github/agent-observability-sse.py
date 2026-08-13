from pathlib import Path


def replace_once(path: str, old: str, new: str) -> None:
    p = Path(path)
    text = p.read_text()
    if old not in text:
        raise SystemExit(f"missing expected block in {path}: {old[:100]!r}")
    p.write_text(text.replace(old, new, 1))


handler = Path("backend/internal/deployment/handler.go")
text = handler.read_text()

text = text.replace(
'''type Handler struct {
\tservice    *Service
\tstatdCache statdRuntimeCache
}

func NewHandler(service *Service) *Handler {
\treturn &Handler{service: service}
}
''',
'''type Handler struct {
\tservice    *Service
\tstatdCache statdRuntimeCache
\tmetricsHub *projectMetricsHub
}

func NewHandler(service *Service) *Handler {
\th := &Handler{service: service}
\th.metricsHub = newProjectMetricsHub(streamPollInterval, func(ctx context.Context, projectID uuid.UUID) (MetricsSnapshotResponse, error) {
\t\tmetrics, err := h.service.PreferredContainerMetricsList(ctx, projectID, &h.statdCache)
\t\tif err != nil {
\t\t\treturn MetricsSnapshotResponse{}, err
\t\t}
\t\treturn MetricsSnapshotFromContainers(metrics), nil
\t})
\treturn h
}
''')

old_metrics = '''func (h *Handler) Metrics(w http.ResponseWriter, r *http.Request) {
\tid, ok := projectID(w, r)
\tif !ok {
\t\treturn
\t}
\tmetrics, err := h.service.PreferredContainerMetricsList(r.Context(), id, &h.statdCache)
\tif err != nil {
\t\thttpx.DomainError(w, err)
\t\treturn
\t}

\tresp := MetricsSnapshotFromContainers(metrics)
\tcfData, _ := h.service.CloudflareAnalytics(r.Context(), id)
\tif cfData != nil {
\t\tts := make([]TimeseriesDataPoint, len(cfData.Timeseries))
\t\tfor i, t := range cfData.Timeseries {
\t\t\tts[i] = TimeseriesDataPoint{
\t\t\t\tTimestamp: t.Timestamp,
\t\t\t\tRequests:  t.Requests,
\t\t\t\tBandwidth: t.Bandwidth,
\t\t\t}
\t\t}

\t\tresp.Analytics = &CloudflareAnalytics{
\t\t\tTotalRequests: cfData.TotalRequests,
\t\t\tBandwidth:     cfData.Bandwidth,
\t\t\tErrors:        cfData.Errors,
\t\t\tTimeseries:    ts,
\t\t}
\t}

\thttpx.JSON(w, http.StatusOK, resp)
}
'''
new_metrics = '''func (h *Handler) Metrics(w http.ResponseWriter, r *http.Request) {
\tid, ok := projectID(w, r)
\tif !ok {
\t\treturn
\t}
\tmetrics, err := h.service.PreferredContainerMetricsList(r.Context(), id, &h.statdCache)
\tif err != nil {
\t\thttpx.DomainError(w, err)
\t\treturn
\t}
\thttpx.JSON(w, http.StatusOK, MetricsSnapshotFromContainers(metrics))
}

func (h *Handler) Analytics(w http.ResponseWriter, r *http.Request) {
\tid, ok := projectID(w, r)
\tif !ok {
\t\treturn
\t}
\tcfData, err := h.service.CloudflareAnalytics(r.Context(), id)
\tif err != nil {
\t\thttpx.DomainError(w, err)
\t\treturn
\t}
\tif cfData == nil {
\t\thttpx.JSON(w, http.StatusOK, CloudflareAnalytics{Timeseries: []TimeseriesDataPoint{}})
\t\treturn
\t}
\tts := make([]TimeseriesDataPoint, len(cfData.Timeseries))
\tfor i, point := range cfData.Timeseries {
\t\tts[i] = TimeseriesDataPoint{
\t\t\tTimestamp: point.Timestamp,
\t\t\tRequests:  point.Requests,
\t\t\tBandwidth: point.Bandwidth,
\t\t}
\t}
\thttpx.JSON(w, http.StatusOK, CloudflareAnalytics{
\t\tTotalRequests: cfData.TotalRequests,
\t\tBandwidth:     cfData.Bandwidth,
\t\tErrors:        cfData.Errors,
\t\tTimeseries:    ts,
\t})
}
'''
if old_metrics not in text:
    raise SystemExit("Metrics handler block not found")
text = text.replace(old_metrics, new_metrics, 1)

old_stream = '''func (h *Handler) Stream(w http.ResponseWriter, r *http.Request) {
\tid, ok := projectID(w, r)
\tif !ok {
\t\treturn
\t}
\tflusher, ok := w.(http.Flusher)
\tif !ok {
\t\thttpx.Error(w, http.StatusInternalServerError, "STREAM_UNSUPPORTED", "Streaming is not supported by this response writer.", nil)
\t\treturn
\t}
\t_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})

\tw.Header().Set("Content-Type", "text/event-stream")
\tw.Header().Set("Cache-Control", "no-cache")
\tw.Header().Set("Connection", "keep-alive")
\tw.Header().Set("X-Accel-Buffering", "no")

\tstream := &projectStream{
\t\thandler:   h,
\t\tprojectID: id,
\t\twriter:    w,
\t\tflusher:   flusher,
\t}
\tif !stream.emitSnapshot(r.Context()) {
\t\treturn
\t}

\tpoll := time.NewTicker(streamPollInterval)
\tdefer poll.Stop()
\theartbeat := time.NewTicker(streamHeartbeat)
\tdefer heartbeat.Stop()

\tfor {
\t\tselect {
\t\tcase <-r.Context().Done():
\t\t\treturn
\t\tcase <-heartbeat.C:
\t\t\tif !stream.heartbeat() {
\t\t\t\treturn
\t\t\t}
\t\tcase <-poll.C:
\t\t\tif !stream.emitSnapshot(r.Context()) {
\t\t\t\treturn
\t\t\t}
\t\t}
\t}
}
'''
new_stream = '''func (h *Handler) Stream(w http.ResponseWriter, r *http.Request) {
\tid, ok := projectID(w, r)
\tif !ok {
\t\treturn
\t}
\tflusher, ok := w.(http.Flusher)
\tif !ok {
\t\thttpx.Error(w, http.StatusInternalServerError, "STREAM_UNSUPPORTED", "Streaming is not supported by this response writer.", nil)
\t\treturn
\t}
\t_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})

\tw.Header().Set("Content-Type", "text/event-stream")
\tw.Header().Set("Cache-Control", "no-cache")
\tw.Header().Set("Connection", "keep-alive")
\tw.Header().Set("X-Accel-Buffering", "no")

\ttopics := parseProjectStreamTopics(r.URL.Query().Get("topics"))
\tstream := &projectStream{
\t\thandler:   h,
\t\tprojectID: id,
\t\twriter:    w,
\t\tflusher:   flusher,
\t\ttopics:    topics,
\t}
\tif !stream.emitSnapshot(r.Context()) {
\t\treturn
\t}

\tvar metrics <-chan MetricsSnapshotResponse
\tvar unsubscribe func()
\tif topics.has(streamTopicMetrics) {
\t\tmetrics, unsubscribe = h.metricsHub.subscribe(id)
\t\tdefer unsubscribe()
\t}

\tpoll := time.NewTicker(streamPollInterval)
\tdefer poll.Stop()
\theartbeat := time.NewTicker(streamHeartbeat)
\tdefer heartbeat.Stop()

\tfor {
\t\tselect {
\t\tcase <-r.Context().Done():
\t\t\treturn
\t\tcase snapshot, open := <-metrics:
\t\t\tif !open || !stream.send("metrics", snapshot) {
\t\t\t\treturn
\t\t\t}
\t\tcase <-heartbeat.C:
\t\t\tif !stream.heartbeat() {
\t\t\t\treturn
\t\t\t}
\t\tcase <-poll.C:
\t\t\tif !stream.emitSnapshot(r.Context()) {
\t\t\t\treturn
\t\t\t}
\t\t}
\t}
}
'''
if old_stream not in text:
    raise SystemExit("Stream handler block not found")
text = text.replace(old_stream, new_stream, 1)

text = text.replace(
'''type projectStream struct {
\thandler              *Handler
\tprojectID            uuid.UUID
\twriter               http.ResponseWriter
\tflusher              http.Flusher
\tlogOffset            map[string]int
\tbuildLogDeploymentID string
\tbuildLogOffset       int
}
''',
'''type projectStream struct {
\thandler              *Handler
\tprojectID            uuid.UUID
\twriter               http.ResponseWriter
\tflusher              http.Flusher
\ttopics               projectStreamTopics
\tlogOffset            map[string]int
\tbuildLogDeploymentID string
\tbuildLogOffset       int
}
''')

old_emit = '''func (s *projectStream) emitSnapshot(ctx context.Context) bool {
\tproject, err := s.handler.service.project(ctx, s.projectID)
\tif err != nil {
\t\tif errors.Is(err, errs.ErrNotFound) {
\t\t\t_ = s.send("status", map[string]string{"status": "deleted"})
\t\t\treturn false
\t\t}
\t\treturn s.send("error", map[string]string{"message": err.Error()})
\t}

\tif !s.send("status", map[string]string{"status": project.Status}) {
\t\treturn false
\t}
\ts.emitMetrics(ctx)
\ts.emitLogs(ctx)
\ts.emitDeployment(ctx)
\treturn true
}

func (s *projectStream) emitMetrics(ctx context.Context) {
\tmetrics, err := s.handler.service.PreferredContainerMetricsList(ctx, s.projectID, &s.handler.statdCache)
\tif err != nil {
\t\treturn
\t}
\tsnapshot := MetricsSnapshotFromContainers(metrics)
\tfor _, item := range snapshot.Items {
\t\t_ = s.send("metrics", item)
\t}
}
'''
new_emit = '''func (s *projectStream) emitSnapshot(ctx context.Context) bool {
\tif s.topics.has(streamTopicStatus) {
\t\tproject, err := s.handler.service.project(ctx, s.projectID)
\t\tif err != nil {
\t\t\tif errors.Is(err, errs.ErrNotFound) {
\t\t\t\t_ = s.send("status", map[string]string{"status": "deleted"})
\t\t\t\treturn false
\t\t\t}
\t\t\treturn s.send("error", map[string]string{"message": err.Error()})
\t\t}
\t\tif !s.send("status", map[string]string{"status": project.Status}) {
\t\t\treturn false
\t\t}
\t}
\tif s.topics.has(streamTopicLogs) {
\t\ts.emitLogs(ctx)
\t}
\tif s.topics.has(streamTopicDeployment) {
\t\ts.emitDeployment(ctx)
\t}
\treturn true
}
'''
if old_emit not in text:
    raise SystemExit("projectStream emit block not found")
text = text.replace(old_emit, new_emit, 1)
handler.write_text(text)

main = Path("backend/cmd/api/main.go")
text = main.read_text()
old = '''\t\tr.Get("/{id}/stream", deploymentHandler.Stream)
\t\tr.Get("/{id}/logs", deploymentHandler.Logs)
\t\tr.Get("/{id}/metrics", deploymentHandler.Metrics)
\t\tr.Get("/{id}/compose-resources", deploymentHandler.ComposeResources)
'''
new = '''\t\tr.Get("/{id}/stream", deploymentHandler.Stream)
\t\tr.Get("/{id}/logs", deploymentHandler.Logs)
\t\tr.Get("/{id}/metrics", deploymentHandler.Metrics)
\t\tr.Get("/{id}/analytics", deploymentHandler.Analytics)
\t\tr.Get("/{id}/compose-resources", deploymentHandler.ComposeResources)
'''
if old not in text:
    raise SystemExit("project metrics route block not found")
main.write_text(text.replace(old, new, 1))

Path("backend/internal/deployment/project_stream_topics.go").write_text(r'''package deployment

import "strings"

type projectStreamTopic string

const (
	streamTopicStatus     projectStreamTopic = "status"
	streamTopicMetrics    projectStreamTopic = "metrics"
	streamTopicLogs       projectStreamTopic = "logs"
	streamTopicDeployment projectStreamTopic = "deployment"
)

type projectStreamTopics map[projectStreamTopic]struct{}

func parseProjectStreamTopics(raw string) projectStreamTopics {
	out := make(projectStreamTopics)
	if strings.TrimSpace(raw) == "" {
		out[streamTopicStatus] = struct{}{}
		out[streamTopicMetrics] = struct{}{}
		out[streamTopicLogs] = struct{}{}
		out[streamTopicDeployment] = struct{}{}
		return out
	}
	for _, value := range strings.Split(raw, ",") {
		switch projectStreamTopic(strings.TrimSpace(strings.ToLower(value))) {
		case streamTopicStatus:
			out[streamTopicStatus] = struct{}{}
		case streamTopicMetrics:
			out[streamTopicMetrics] = struct{}{}
		case streamTopicLogs:
			out[streamTopicLogs] = struct{}{}
		case streamTopicDeployment:
			out[streamTopicDeployment] = struct{}{}
		}
	}
	if len(out) == 0 {
		out[streamTopicStatus] = struct{}{}
	}
	return out
}

func (t projectStreamTopics) has(topic projectStreamTopic) bool {
	_, ok := t[topic]
	return ok
}
''')

Path("backend/internal/deployment/project_metrics_hub.go").write_text(r'''package deployment

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
''')

Path("backend/internal/deployment/project_metrics_hub_test.go").write_text(r'''package deployment

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
''')
