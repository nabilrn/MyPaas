package deployment

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"mypaas/internal/auth"
	"mypaas/internal/errs"
	"mypaas/internal/httpx"
	"mypaas/internal/project"
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
	lines, err := h.service.ContainerLogs(r.Context(), id, int(project.IntQuery(r, "tail", 500)))
	if err != nil {
		httpx.DomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"lines": lines})
}

func (h *Handler) Metrics(w http.ResponseWriter, r *http.Request) {
	id, ok := projectID(w, r)
	if !ok {
		return
	}
	metrics, err := h.service.ContainerMetrics(r.Context(), id)
	if err != nil {
		httpx.DomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, MetricsSnapshotFromContainer(metrics))
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
