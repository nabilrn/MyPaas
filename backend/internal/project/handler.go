package project

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"mypaas/internal/auth"
	"mypaas/internal/db"
	"mypaas/internal/envvar"
	"mypaas/internal/errs"
	"mypaas/internal/httpx"
)

type Handler struct {
	service           *Service
	cleanup           func(*http.Request, uuid.UUID) error
	updateRouting     func(context.Context, db.Project, db.Project) error
	provisionSharedDB func(context.Context, db.Project) error
	envs              *envvar.Service
}

func NewHandler(
	service *Service,
	cleanup func(*http.Request, uuid.UUID) error,
	updateRouting func(context.Context, db.Project, db.Project) error,
	provisionSharedDB func(context.Context, db.Project) error,
	envs *envvar.Service,
) *Handler {
	return &Handler{service: service, cleanup: cleanup, updateRouting: updateRouting, provisionSharedDB: provisionSharedDB, envs: envs}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	user, err := auth.CurrentUser(r)
	if err != nil {
		httpx.DomainError(w, err)
		return
	}

	projects, err := h.service.List(r.Context(), user.ID)
	if err != nil {
		httpx.DomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, ResponsesFromDB(projects))
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := projectID(w, r)
	if !ok {
		return
	}

	project, err := h.service.Get(r.Context(), id)
	if err != nil {
		httpx.DomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, ResponseFromDB(project))
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	user, err := auth.CurrentUser(r)
	if err != nil {
		httpx.DomainError(w, err)
		return
	}

	var req struct {
		Name            string         `json:"name"`
		RepoURL         string         `json:"repoUrl"`
		Branch          string         `json:"branch"`
		DeployMode      string         `json:"deployMode"`
		ResourceProfile string         `json:"resourceProfile"`
		MainService     *string        `json:"mainService"`
		AppPort         int32          `json:"appPort"`
		MemoryLimitMb   int32          `json:"memoryLimitMb"`
		MemoryMb        int32          `json:"memoryMb"`
		CPULimit        float64        `json:"cpuLimit"`
		SharedPostgres  bool           `json:"sharedPostgres"`
		EnvVars         []envvar.Value `json:"envVars"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_JSON", "Request body must be valid JSON.", nil)
		return
	}
	if req.MemoryLimitMb == 0 {
		req.MemoryLimitMb = req.MemoryMb
	}

	project, err := h.service.Create(r.Context(), CreateInput{
		UserID:          user.ID,
		Name:            req.Name,
		RepoURL:         req.RepoURL,
		Branch:          req.Branch,
		DeployMode:      req.DeployMode,
		ResourceProfile: req.ResourceProfile,
		MainService:     req.MainService,
		AppPort:         req.AppPort,
		MemoryLimitMb:   req.MemoryLimitMb,
		CPULimit:        req.CPULimit,
	})
	if err != nil {
		httpx.DomainError(w, err)
		return
	}
	if len(req.EnvVars) > 0 && h.envs != nil {
		if err := h.envs.BulkUpdate(r.Context(), project.ID, req.EnvVars); err != nil {
			h.cleanupCreatedProject(r, project.ID)
			httpx.DomainError(w, err)
			return
		}
	}
	if req.SharedPostgres && h.provisionSharedDB != nil {
		if err := h.provisionSharedDB(r.Context(), project); err != nil {
			h.cleanupCreatedProject(r, project.ID)
			httpx.DomainError(w, err)
			return
		}
		if refreshed, err := h.service.Get(r.Context(), project.ID); err == nil {
			project = refreshed
		}
	}
	httpx.JSON(w, http.StatusCreated, ResponseFromDB(project))
}

func (h *Handler) cleanupCreatedProject(r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	if h.envs != nil {
		if err := h.envs.DeleteAll(ctx, id); err != nil {
			slog.Warn("delete env vars after failed project create", "projectId", id, "error", err)
		}
	}
	if h.cleanup != nil {
		if err := h.cleanup(r, id); err != nil {
			slog.Warn("cleanup resources after failed project create", "projectId", id, "error", err)
		}
	}
	if err := h.service.Delete(context.Background(), id); err != nil {
		slog.Warn("soft delete project after failed create", "projectId", id, "error", err)
	}
}

func (h *Handler) DetectMode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RepoURL string `json:"repoUrl"`
		Branch  string `json:"branch"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_JSON", "Request body must be valid JSON.", nil)
		return
	}

	result, err := h.service.DetectMode(r.Context(), DetectInput{
		RepoURL: req.RepoURL,
		Branch:  req.Branch,
	})
	if err != nil {
		httpx.DomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, DetectResponseFromResult(result))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := projectID(w, r)
	if !ok {
		return
	}
	before, err := h.service.Get(r.Context(), id)
	if err != nil {
		httpx.DomainError(w, err)
		return
	}

	var req struct {
		Name            string  `json:"name"`
		Branch          string  `json:"branch"`
		ResourceProfile string  `json:"resourceProfile"`
		AppPort         int32   `json:"appPort"`
		MemoryLimitMb   int32   `json:"memoryLimitMb"`
		MemoryMb        int32   `json:"memoryMb"`
		CPULimit        float64 `json:"cpuLimit"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_JSON", "Request body must be valid JSON.", nil)
		return
	}
	if req.MemoryLimitMb == 0 {
		req.MemoryLimitMb = req.MemoryMb
	}

	project, err := h.service.Update(r.Context(), UpdateInput{
		ID:              id,
		Name:            req.Name,
		Branch:          req.Branch,
		ResourceProfile: req.ResourceProfile,
		AppPort:         req.AppPort,
		MemoryLimitMb:   req.MemoryLimitMb,
		CPULimit:        req.CPULimit,
	})
	if err != nil {
		httpx.DomainError(w, err)
		return
	}
	if h.updateRouting != nil {
		if err := h.updateRouting(r.Context(), before, project); err != nil {
			httpx.DomainError(w, err)
			return
		}
	}
	httpx.JSON(w, http.StatusOK, ResponseFromDB(project))
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := projectID(w, r)
	if !ok {
		return
	}
	if h.cleanup != nil {
		if err := h.cleanup(r, id); err != nil {
			httpx.DomainError(w, err)
			return
		}
	}
	if err := h.service.Delete(r.Context(), id); err != nil {
		httpx.DomainError(w, err)
		return
	}
	httpx.NoContent(w)
}

func (h *Handler) RegenerateWebhookSecret(w http.ResponseWriter, r *http.Request) {
	id, ok := projectID(w, r)
	if !ok {
		return
	}
	secret, err := h.service.RegenerateWebhookSecret(r.Context(), id)
	if err != nil {
		httpx.DomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"webhookSecret": secret})
}

func projectID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.DomainError(w, errs.ErrNotFound)
		return uuid.UUID{}, false
	}
	return id, true
}

func IntQuery(r *http.Request, key string, fallback int32) int32 {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed < 0 {
		return fallback
	}
	return int32(parsed)
}
