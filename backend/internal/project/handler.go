package project

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"mypaas/internal/auth"
	"mypaas/internal/errs"
	"mypaas/internal/httpx"
)

type Handler struct {
	service *Service
	cleanup func(*http.Request, uuid.UUID) error
}

func NewHandler(service *Service, cleanup func(*http.Request, uuid.UUID) error) *Handler {
	return &Handler{service: service, cleanup: cleanup}
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
		Name          string  `json:"name"`
		RepoURL       string  `json:"repoUrl"`
		Branch        string  `json:"branch"`
		DeployMode    string  `json:"deployMode"`
		MainService   *string `json:"mainService"`
		AppPort       int32   `json:"appPort"`
		MemoryLimitMb int32   `json:"memoryLimitMb"`
		MemoryMb      int32   `json:"memoryMb"`
		CPULimit      float64 `json:"cpuLimit"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_JSON", "Request body must be valid JSON.", nil)
		return
	}
	if req.MemoryLimitMb == 0 {
		req.MemoryLimitMb = req.MemoryMb
	}

	project, err := h.service.Create(r.Context(), CreateInput{
		UserID:        user.ID,
		Name:          req.Name,
		RepoURL:       req.RepoURL,
		Branch:        req.Branch,
		DeployMode:    req.DeployMode,
		MainService:   req.MainService,
		AppPort:       req.AppPort,
		MemoryLimitMb: req.MemoryLimitMb,
		CPULimit:      req.CPULimit,
	})
	if err != nil {
		httpx.DomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, ResponseFromDB(project))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := projectID(w, r)
	if !ok {
		return
	}

	var req struct {
		Name          string  `json:"name"`
		Branch        string  `json:"branch"`
		AppPort       int32   `json:"appPort"`
		MemoryLimitMb int32   `json:"memoryLimitMb"`
		MemoryMb      int32   `json:"memoryMb"`
		CPULimit      float64 `json:"cpuLimit"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_JSON", "Request body must be valid JSON.", nil)
		return
	}
	if req.MemoryLimitMb == 0 {
		req.MemoryLimitMb = req.MemoryMb
	}

	project, err := h.service.Update(r.Context(), UpdateInput{
		ID:            id,
		Name:          req.Name,
		Branch:        req.Branch,
		AppPort:       req.AppPort,
		MemoryLimitMb: req.MemoryLimitMb,
		CPULimit:      req.CPULimit,
	})
	if err != nil {
		httpx.DomainError(w, err)
		return
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
