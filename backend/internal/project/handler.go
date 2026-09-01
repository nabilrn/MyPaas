package project

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"mypaas/internal/auth"
	"mypaas/internal/db"
	"mypaas/internal/envvar"
	"mypaas/internal/errs"
	"mypaas/internal/github"
	"mypaas/internal/httpx"
	"mypaas/internal/repopath"
)

type Handler struct {
	service           *Service
	cleanup           func(*http.Request, uuid.UUID) error
	updateRouting     func(context.Context, db.Project, db.Project) error
	provisionSharedDB func(context.Context, db.Project) error
	envs              *envvar.Service
	github            github.TokenReader
}

func NewHandler(
	service *Service,
	cleanup func(*http.Request, uuid.UUID) error,
	updateRouting func(context.Context, db.Project, db.Project) error,
	provisionSharedDB func(context.Context, db.Project) error,
	envs *envvar.Service,
	githubTokens ...github.TokenReader,
) *Handler {
	var githubTokenReader github.TokenReader
	if len(githubTokens) > 0 {
		githubTokenReader = githubTokens[0]
	}
	return &Handler{service: service, cleanup: cleanup, updateRouting: updateRouting, provisionSharedDB: provisionSharedDB, envs: envs, github: githubTokenReader}
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
		Name                 string          `json:"name"`
		SourceType           string          `json:"sourceType"`
		RepoURL              string          `json:"repoUrl"`
		ImageRef             *string         `json:"imageRef"`
		Branch               string          `json:"branch"`
		DeployMode           string          `json:"deployMode"`
		ResourceProfile      string          `json:"resourceProfile"`
		MainService          *string         `json:"mainService"`
		AppPort              int32           `json:"appPort"`
		MemoryLimitMb        int32           `json:"memoryLimitMb"`
		MemoryMb             int32           `json:"memoryMb"`
		CPULimit             float64         `json:"cpuLimit"`
		SharedPostgres       bool            `json:"sharedPostgres"`
		EnvVars              []envvar.Value  `json:"envVars"`
		ComposeFilePath      *string         `json:"composeFilePath"`
		ComposeOverridePaths []string        `json:"composeOverridePaths"`
		ComposeProfiles      []string        `json:"composeProfiles"`
		ComposeWorkdir       *string         `json:"composeWorkdir"`
		ServiceResources     json.RawMessage `json:"serviceResources"`
		StaticFrontendPath   *string         `json:"staticFrontendPath"`
		BaseDirectory        *string         `json:"baseDirectory"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_JSON", "Request body must be valid JSON.", nil)
		return
	}
	if req.BaseDirectory != nil {
		if err := repopath.Validate(*req.BaseDirectory); err != nil {
			httpx.DomainError(w, err)
			return
		}
	}
	if req.StaticFrontendPath != nil {
		if err := repopath.Validate(*req.StaticFrontendPath); err != nil {
			httpx.DomainError(w, err)
			return
		}
	}
	if req.MemoryLimitMb == 0 {
		req.MemoryLimitMb = req.MemoryMb
	}
	githubAccessToken, err := h.githubAccessToken(r.Context(), user.ID, req.SourceType != "registry" && req.DeployMode != "image")
	if err != nil {
		httpx.DomainError(w, err)
		return
	}

	project, err := h.service.CreateValidated(r.Context(), CreateValidationInput{
		Project: CreateInput{
			UserID:               user.ID,
			Name:                 req.Name,
			SourceType:           req.SourceType,
			RepoURL:              req.RepoURL,
			ImageRef:             req.ImageRef,
			Branch:               req.Branch,
			DeployMode:           req.DeployMode,
			ResourceProfile:      req.ResourceProfile,
			MainService:          req.MainService,
			AppPort:              req.AppPort,
			MemoryLimitMb:        req.MemoryLimitMb,
			CPULimit:             req.CPULimit,
			ComposeFilePath:      req.ComposeFilePath,
			ComposeOverridePaths: req.ComposeOverridePaths,
			ComposeProfiles:      req.ComposeProfiles,
			ComposeWorkdir:       req.ComposeWorkdir,
			ServiceResources:     req.ServiceResources,
			StaticFrontendPath:   req.StaticFrontendPath,
			BaseDirectory:        req.BaseDirectory,
			GitHubAccessToken:    githubAccessToken,
		},
		EnvVars:        req.EnvVars,
		SharedPostgres: req.SharedPostgres,
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

func (h *Handler) githubAccessToken(ctx context.Context, userID uuid.UUID, required bool) (string, error) {
	if !required || h.github == nil {
		return "", nil
	}
	accessToken, err := h.github.AccessToken(ctx, userID)
	if errors.Is(err, errs.ErrNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return accessToken, nil
}

func (h *Handler) DetectMode(w http.ResponseWriter, r *http.Request) {
	var userID uuid.UUID
	if h.github != nil {
		user, err := auth.CurrentUser(r)
		if err != nil {
			httpx.DomainError(w, err)
			return
		}
		userID = user.ID
	}
	var req struct {
		RepoURL       string `json:"repoUrl"`
		Branch        string `json:"branch"`
		InspectOnly   bool   `json:"inspectOnly"`
		BaseDirectory string `json:"baseDirectory"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_JSON", "Request body must be valid JSON.", nil)
		return
	}
	githubAccessToken, err := h.githubAccessToken(r.Context(), userID, true)
	if err != nil {
		httpx.DomainError(w, err)
		return
	}

	result, err := h.service.DetectModeValidated(r.Context(), DetectInput{
		RepoURL:           req.RepoURL,
		Branch:            req.Branch,
		InspectOnly:       req.InspectOnly,
		BaseDirectory:     req.BaseDirectory,
		GitHubAccessToken: githubAccessToken,
	})
	if err != nil {
		httpx.DomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, DetectResponseFromResult(result))
}

func (h *Handler) DetectCompose(w http.ResponseWriter, r *http.Request) {
	var userID uuid.UUID
	if h.github != nil {
		user, err := auth.CurrentUser(r)
		if err != nil {
			httpx.DomainError(w, err)
			return
		}
		userID = user.ID
	}
	var req struct {
		RepoURL       string `json:"repoUrl"`
		Branch        string `json:"branch"`
		BaseDirectory string `json:"baseDirectory"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_JSON", "Request body must be valid JSON.", nil)
		return
	}
	githubAccessToken, err := h.githubAccessToken(r.Context(), userID, true)
	if err != nil {
		httpx.DomainError(w, err)
		return
	}

	result, err := h.service.DetectCompose(r.Context(), DetectComposeInput{
		RepoURL:           req.RepoURL,
		Branch:            req.Branch,
		BaseDirectory:     req.BaseDirectory,
		GitHubAccessToken: githubAccessToken,
	})
	if err != nil {
		httpx.DomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, DetectComposeResponseFromResult(result))
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
		Name                 string          `json:"name"`
		Branch               string          `json:"branch"`
		ImageRef             *string         `json:"imageRef"`
		ResourceProfile      string          `json:"resourceProfile"`
		MainService          *string         `json:"mainService"`
		AppPort              int32           `json:"appPort"`
		MemoryLimitMb        int32           `json:"memoryLimitMb"`
		MemoryMb             int32           `json:"memoryMb"`
		CPULimit             float64         `json:"cpuLimit"`
		ComposeFilePath      *string         `json:"composeFilePath"`
		ComposeOverridePaths []string        `json:"composeOverridePaths"`
		ComposeProfiles      []string        `json:"composeProfiles"`
		ComposeWorkdir       *string         `json:"composeWorkdir"`
		ServiceResources     json.RawMessage `json:"serviceResources"`
		StaticFrontendPath   optionalString  `json:"staticFrontendPath"`
		BaseDirectory        optionalString  `json:"baseDirectory"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_JSON", "Request body must be valid JSON.", nil)
		return
	}
	if req.BaseDirectory.Set && req.BaseDirectory.Value != nil {
		if err := repopath.Validate(*req.BaseDirectory.Value); err != nil {
			httpx.DomainError(w, err)
			return
		}
	}
	if req.StaticFrontendPath.Set && req.StaticFrontendPath.Value != nil {
		if err := repopath.Validate(*req.StaticFrontendPath.Value); err != nil {
			httpx.DomainError(w, err)
			return
		}
	}
	if req.MemoryLimitMb == 0 {
		req.MemoryLimitMb = req.MemoryMb
	}
	staticFrontendPath := req.StaticFrontendPath.Resolve(before.StaticFrontendPath)
	baseDirectory := req.BaseDirectory.Resolve(before.BaseDirectory)

	project, err := h.service.UpdateValidated(r.Context(), UpdateInput{
		ID:                   id,
		Name:                 req.Name,
		Branch:               req.Branch,
		ImageRef:             req.ImageRef,
		ResourceProfile:      req.ResourceProfile,
		MainService:          req.MainService,
		AppPort:              req.AppPort,
		MemoryLimitMb:        req.MemoryLimitMb,
		CPULimit:             req.CPULimit,
		ComposeFilePath:      req.ComposeFilePath,
		ComposeOverridePaths: req.ComposeOverridePaths,
		ComposeProfiles:      req.ComposeProfiles,
		ComposeWorkdir:       req.ComposeWorkdir,
		ServiceResources:     req.ServiceResources,
		StaticFrontendPath:   staticFrontendPath,
		BaseDirectory:        baseDirectory,
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
			slog.Warn("project cleanup encountered errors during deletion", "projectId", id, "error", err)
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
