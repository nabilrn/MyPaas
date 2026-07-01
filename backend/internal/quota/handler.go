package quota

import (
	"net/http"

	"mypaas/internal/auth"
	"mypaas/internal/httpx"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	user, err := auth.CurrentUser(r)
	if err != nil {
		httpx.DomainError(w, err)
		return
	}
	usage, err := h.service.Usage(r.Context(), user.ID)
	if err != nil {
		httpx.DomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, usage)
}
