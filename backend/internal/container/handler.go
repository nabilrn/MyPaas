package container

import (
	"context"
	"net/http"

	"mypaas/internal/httpx"
)

type RuntimeInventory interface {
	RuntimeContainers(ctx context.Context) ([]RuntimeContainer, error)
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

// Delete intentionally keeps the host-wide inventory read-only. Container
// lifecycle remains project-scoped so removing a stopped project container
// cannot make a later Start operation unrecoverable.
func (h *Handler) Delete(w http.ResponseWriter, _ *http.Request) {
	httpx.Error(w, http.StatusMethodNotAllowed, "CONTAINER_INVENTORY_READ_ONLY", "Container inventory is read-only. Manage application lifecycle from the project page.", nil)
}
