package tokenapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"mypaas/internal/apitoken"
	"mypaas/internal/auth"
	"mypaas/internal/httpx"
)

type Handler struct {
	repo *apitoken.Repository
}

func NewHandler(repo *apitoken.Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	user, err := auth.CurrentUser(r)
	if err != nil {
		httpx.DomainError(w, err)
		return
	}
	tokens, err := h.repo.List(r.Context(), user.ID)
	if err != nil {
		httpx.DomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"tokens":        tokens,
		"allowedScopes": apitoken.AllowedScopes(),
		"defaultScopes": append([]string(nil), apitoken.DefaultMCPScopes...),
	})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	user, err := auth.CurrentUser(r)
	if err != nil {
		httpx.DomainError(w, err)
		return
	}
	var req struct {
		Name      string     `json:"name"`
		Scopes    []string   `json:"scopes"`
		ExpiresAt *time.Time `json:"expiresAt"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_JSON", "Request body must be valid JSON.", nil)
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		req.Name = "MCP agent"
	}
	if len(req.Scopes) == 0 {
		req.Scopes = append([]string(nil), apitoken.DefaultMCPScopes...)
	}
	created, err := h.repo.Create(r.Context(), user.ID, req.Name, req.Scopes, req.ExpiresAt)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_API_TOKEN", err.Error(), nil)
		return
	}
	// The raw bearer credential exists only in this creation response. List and
	// audit surfaces expose only the display prefix, never the secret.
	httpx.JSON(w, http.StatusCreated, created)
}

func (h *Handler) Revoke(w http.ResponseWriter, r *http.Request) {
	user, err := auth.CurrentUser(r)
	if err != nil {
		httpx.DomainError(w, err)
		return
	}
	tokenID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusNotFound, "API_TOKEN_NOT_FOUND", "API token not found.", nil)
		return
	}
	if err := h.repo.Revoke(r.Context(), user.ID, tokenID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.Error(w, http.StatusNotFound, "API_TOKEN_NOT_FOUND", "API token not found.", nil)
			return
		}
		httpx.DomainError(w, err)
		return
	}
	httpx.NoContent(w)
}
