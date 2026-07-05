package deployment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"mypaas/internal/auth"
	"mypaas/internal/container"
	"mypaas/internal/errs"
	"mypaas/internal/httpx"
	"mypaas/internal/project"
)

const (
	streamPollInterval = 5 * time.Second
	streamHeartbeat    = 30 * time.Second
	streamLogTail      = 200
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Trigger(w http.ResponseWriter, r *http.Request) {
	user, err := auth.CurrentUser(r)
	if err != nil {
		httpx.DomainError(w, err)
		return
	}
	id, ok := projectID(w, r)
	if !ok {
		return
	}
	deployment, err := h.service.TriggerDockerfile(r.Context(), id, user.ID)
	if err != nil {
		httpx.DomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusAccepted, ResponseFromDB(deployment))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	id, ok := projectID(w, r)
	if !ok {
		return
	}
	rows, err := h.service.ListByProject(r.Context(), id, project.IntQuery(r, "limit", 20), project.IntQuery(r, "offset", 0))
	if err != nil {
		httpx.DomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, ResponsesFromDB(rows))
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.DomainError(w, errs.ErrNotFound)
		return
	}
	row, err := h.service.Get(r.Context(), id)
	if err != nil {
		httpx.DomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, ResponseFromDB(row))
}

func (h *Handler) Rollback(w http.ResponseWriter, r *http.Request) {
	user, err := auth.CurrentUser(r)
	if err != nil {
		httpx.DomainError(w, err)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.DomainError(w, errs.ErrNotFound)
		return
	}
	deployment, err := h.service.Rollback(r.Context(), id, user.ID)
	if err != nil {
		httpx.DomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, ResponseFromDB(deployment))
}

func (h *Handler) Start(w http.ResponseWriter, r *http.Request) {
	h.lifecycle(w, r, h.service.Start)
}

func (h *Handler) Stop(w http.ResponseWriter, r *http.Request) {
	h.lifecycle(w, r, h.service.Stop)
}

func (h *Handler) Restart(w http.ResponseWriter, r *http.Request) {
	h.lifecycle(w, r, h.service.Restart)
}

func (h *Handler) Logs(w http.ResponseWriter, r *http.Request) {
	id, ok := projectID(w, r)
	if !ok {
		return
	}
	lines, err := h.service.ContainerLogLines(r.Context(), id, int(project.IntQuery(r, "tail", 500)))
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			httpx.JSON(w, http.StatusOK, EmptyLogsResponse())
			return
		}
		httpx.DomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, LogsResponseFromContainer(lines))
}

func (h *Handler) Metrics(w http.ResponseWriter, r *http.Request) {
	id, ok := projectID(w, r)
	if !ok {
		return
	}
	metrics, err := h.service.ContainerMetricsList(r.Context(), id)
	if err != nil {
		httpx.DomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, MetricsSnapshotFromContainers(metrics))
}

func (h *Handler) ComposeResources(w http.ResponseWriter, r *http.Request) {
	id, ok := projectID(w, r)
	if !ok {
		return
	}
	resources, err := h.service.ComposeResources(r.Context(), id)
	if err != nil {
		httpx.DomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, resources)
}

func (h *Handler) ResetComposeResources(w http.ResponseWriter, r *http.Request) {
	id, ok := projectID(w, r)
	if !ok {
		return
	}
	if err := h.service.ResetComposeResources(r.Context(), id); err != nil {
		httpx.DomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) Stream(w http.ResponseWriter, r *http.Request) {
	id, ok := projectID(w, r)
	if !ok {
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		httpx.Error(w, http.StatusInternalServerError, "STREAM_UNSUPPORTED", "Streaming is not supported by this response writer.", nil)
		return
	}
	_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	stream := &projectStream{
		handler:   h,
		projectID: id,
		writer:    w,
		flusher:   flusher,
	}
	if !stream.emitSnapshot(r.Context()) {
		return
	}

	poll := time.NewTicker(streamPollInterval)
	defer poll.Stop()
	heartbeat := time.NewTicker(streamHeartbeat)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-heartbeat.C:
			if !stream.heartbeat() {
				return
			}
		case <-poll.C:
			if !stream.emitSnapshot(r.Context()) {
				return
			}
		}
	}
}

func (h *Handler) lifecycle(w http.ResponseWriter, r *http.Request, fn func(context.Context, uuid.UUID) error) {
	id, ok := projectID(w, r)
	if !ok {
		return
	}
	if err := fn(r.Context(), id); err != nil {
		httpx.DomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func projectID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.DomainError(w, errs.ErrNotFound)
		return uuid.UUID{}, false
	}
	return id, true
}

type LogsResponse struct {
	Lines []string          `json:"lines"`
	Items []LogLineResponse `json:"items"`
}

type LogLineResponse struct {
	Service string `json:"service"`
	Line    string `json:"line"`
}

func EmptyLogsResponse() LogsResponse {
	return LogsResponse{
		Lines: []string{},
		Items: []LogLineResponse{},
	}
}

func LogsResponseFromContainer(lines []container.ComposeLogLine) LogsResponse {
	response := LogsResponse{
		Lines: make([]string, 0, len(lines)),
		Items: make([]LogLineResponse, 0, len(lines)),
	}
	for _, line := range lines {
		response.Lines = append(response.Lines, line.Line)
		response.Items = append(response.Items, LogLineResponse{
			Service: line.Service,
			Line:    line.Line,
		})
	}
	return response
}

type projectStream struct {
	handler              *Handler
	projectID            uuid.UUID
	writer               http.ResponseWriter
	flusher              http.Flusher
	logOffset            map[string]int
	buildLogDeploymentID string
	buildLogOffset       int
}

func (s *projectStream) emitSnapshot(ctx context.Context) bool {
	project, err := s.handler.service.project(ctx, s.projectID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			_ = s.send("status", map[string]string{"status": "deleted"})
			return false
		}
		return s.send("error", map[string]string{"message": err.Error()})
	}

	if !s.send("status", map[string]string{"status": project.Status}) {
		return false
	}
	s.emitMetrics(ctx)
	s.emitLogs(ctx)
	s.emitDeployment(ctx)
	return true
}

func (s *projectStream) emitMetrics(ctx context.Context) {
	metrics, err := s.handler.service.ContainerMetricsList(ctx, s.projectID)
	if err != nil {
		return
	}
	snapshot := MetricsSnapshotFromContainers(metrics)
	for _, item := range snapshot.Items {
		_ = s.send("metrics", item)
		if item.Uptime != "" {
			_ = s.send("status", map[string]string{"status": "running", "uptime": item.Uptime})
		}
	}
}

func (s *projectStream) emitLogs(ctx context.Context) {
	lines, err := s.handler.service.ContainerLogLines(ctx, s.projectID, streamLogTail)
	if err != nil {
		return
	}
	if s.logOffset == nil {
		s.logOffset = make(map[string]int)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	byService := make(map[string][]container.ComposeLogLine)
	for _, line := range lines {
		byService[line.Service] = append(byService[line.Service], line)
	}
	for service, serviceLines := range byService {
		offset := s.logOffset[service]
		if len(serviceLines) < offset {
			offset = 0
		}
		for _, line := range serviceLines[offset:] {
			_ = s.send("log", map[string]string{
				"service":   line.Service,
				"line":      line.Line,
				"timestamp": now,
			})
		}
		s.logOffset[service] = len(serviceLines)
	}
}

func (s *projectStream) emitDeployment(ctx context.Context) {
	deployment, ok, err := s.handler.service.activeDeployment(ctx, s.projectID)
	if err == nil && ok {
		_ = s.send("deployment", ResponseFromDB(deployment))
		s.emitBuildLog(deployment.ID.String(), deployment.BuildLog)
		return
	}
	rows, err := s.handler.service.ListByProject(ctx, s.projectID, 1, 0)
	if err == nil && len(rows) > 0 {
		_ = s.send("deployment", ResponseFromDB(rows[0]))
		s.emitBuildLog(rows[0].ID.String(), rows[0].BuildLog)
	}
}

func (s *projectStream) emitBuildLog(deploymentID string, buildLog *string) {
	if buildLog == nil || strings.TrimSpace(*buildLog) == "" {
		return
	}
	if s.buildLogDeploymentID != deploymentID {
		s.buildLogDeploymentID = deploymentID
		s.buildLogOffset = 0
	}

	lines := splitBuildLog(*buildLog)
	if len(lines) < s.buildLogOffset {
		s.buildLogOffset = 0
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for _, line := range lines[s.buildLogOffset:] {
		_ = s.send("deployment-log", map[string]string{
			"service":   "build",
			"line":      line,
			"timestamp": now,
		})
	}
	s.buildLogOffset = len(lines)
}

func splitBuildLog(value string) []string {
	lines := strings.Split(strings.ReplaceAll(value, "\r\n", "\n"), "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func (s *projectStream) send(event string, data any) bool {
	payload, err := json.Marshal(data)
	if err != nil {
		payload = []byte(`{"message":"failed to encode stream event"}`)
	}
	if _, err := fmt.Fprintf(s.writer, "event: %s\ndata: %s\n\n", event, payload); err != nil {
		return false
	}
	s.flusher.Flush()
	return true
}

func (s *projectStream) heartbeat() bool {
	if _, err := fmt.Fprint(s.writer, ": heartbeat\n\n"); err != nil {
		return false
	}
	s.flusher.Flush()
	return true
}
