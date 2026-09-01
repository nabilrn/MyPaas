package container

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"mypaas/internal/httpx"
)

type RuntimeInventory interface {
	RuntimeContainers(ctx context.Context) ([]RuntimeContainer, error)
	RemoveStopped(ctx context.Context, id string) error
}

type Handler struct {
	runtime RuntimeInventory
}

func NewHandler(runtime RuntimeInventory) *Handler {
	return &Handler{runtime: runtime}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	containers, err := h.runtime.RuntimeContainers(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusServiceUnavailable, "CONTAINER_INVENTORY_FAILED", err.Error(), nil)
		return
	}
	httpx.JSON(w, http.StatusOK, struct {
		Containers []RuntimeContainer `json:"containers"`
	}{Containers: containers})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.runtime.RemoveStopped(r.Context(), chi.URLParam(r, "id")); err != nil {
		httpx.DomainError(w, err)
		return
	}
	httpx.NoContent(w)
}
