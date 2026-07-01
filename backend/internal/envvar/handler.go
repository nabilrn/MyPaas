package envvar

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"mypaas/internal/errs"
	"mypaas/internal/httpx"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	id, ok := projectID(w, r)
	if !ok {
		return
	}
	rows, err := h.service.List(r.Context(), id)
	if err != nil {
		httpx.DomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, ResponsesFromDB(rows))
}

func (h *Handler) BulkUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := projectID(w, r)
	if !ok {
		return
	}

	var req struct {
		Vars      []Value `json:"vars"`
		Variables []Value `json:"variables"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_JSON", "Request body must be valid JSON.", nil)
		return
	}
	values := req.Vars
	if len(values) == 0 {
		values = req.Variables
	}
	if err := h.service.BulkUpdate(r.Context(), id, values); err != nil {
		httpx.DomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := projectID(w, r)
	if !ok {
		return
	}
	if err := h.service.Delete(r.Context(), id, chi.URLParam(r, "key")); err != nil {
		httpx.DomainError(w, err)
		return
	}
	httpx.NoContent(w)
}

func projectID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.DomainError(w, errs.ErrNotFound)
		return uuid.UUID{}, false
	}
	return id, true
}
