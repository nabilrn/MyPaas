package project

import (
	"net/http"

	"mypaas/internal/httpx"
)

func (h *Handler) Routes(w http.ResponseWriter, r *http.Request) {
	id, ok := projectID(w, r)
	if !ok {
		return
	}
	routes, err := h.service.AdditionalRoutes(r.Context(), id)
	if err != nil {
		httpx.DomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, routes)
}

func (h *Handler) SetRoutes(w http.ResponseWriter, r *http.Request) {
	id, ok := projectID(w, r)
	if !ok {
		return
	}
	var req struct {
		Routes []AdditionalRoute `json:"routes"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_JSON", "Request body must be valid JSON.", nil)
		return
	}
	routes, err := h.service.SetAdditionalRoutes(r.Context(), id, req.Routes)
	if err != nil {
		httpx.DomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, routes)
}
