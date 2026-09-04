package dbstudio

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"mypaas/internal/auth"
	"mypaas/internal/errs"
	"mypaas/internal/httpx"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	projectID, userID, ok := ids(w, r)
	if !ok {
		return
	}
	if strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("probe")), "false") {
		out, err := h.service.ConfigurationStatus(r.Context(), projectID)
		if err != nil {
			httpx.DomainError(w, err)
			return
		}
		httpx.JSON(w, http.StatusOK, out)
		return
	}
	out, err := h.service.Status(r.Context(), projectID, userID)
	if err != nil {
		httpx.DomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, out)
}

func (h *Handler) StartWriteSession(w http.ResponseWriter, r *http.Request) {
	projectID, userID, ok := ids(w, r)
	if !ok {
		return
	}
	var req struct {
		TTLMinutes int `json:"ttlMinutes"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_JSON", "Request body must be valid JSON.", nil)
		return
	}
	session, err := h.service.StartWriteSession(r.Context(), projectID, userID, time.Duration(req.TTLMinutes)*time.Minute)
	if err != nil {
		httpx.DomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, session)
}

func (h *Handler) RevokeWriteSession(w http.ResponseWriter, r *http.Request) {
	projectID, userID, ok := ids(w, r)
	if !ok {
		return
	}
	sessionID, err := uuid.Parse(chi.URLParam(r, "sessionId"))
	if err != nil {
		httpx.DomainError(w, errs.ErrNotFound)
		return
	}
	if err := h.service.RevokeWriteSession(r.Context(), projectID, userID, sessionID); err != nil {
		httpx.DomainError(w, err)
		return
	}
	httpx.NoContent(w)
}

func (h *Handler) Schemas(w http.ResponseWriter, r *http.Request) {
	projectID, ok := projectID(w, r)
	if !ok {
		return
	}
	out, err := h.service.Schemas(r.Context(), projectID)
	writeResult(w, out, err)
}

func (h *Handler) Tables(w http.ResponseWriter, r *http.Request) {
	projectID, ok := projectID(w, r)
	if !ok {
		return
	}
	out, err := h.service.Tables(r.Context(), projectID, r.URL.Query().Get("schema"))
	writeResult(w, out, err)
}

func (h *Handler) Columns(w http.ResponseWriter, r *http.Request) {
	projectID, ok := projectID(w, r)
	if !ok {
		return
	}
	schema := r.URL.Query().Get("schema")
	table := r.URL.Query().Get("table")
	if strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("details")), "true") {
		out, err := h.service.TableDetails(r.Context(), projectID, schema, table)
		writeResult(w, out, err)
		return
	}
	out, err := h.service.Columns(r.Context(), projectID, schema, table)
	writeResult(w, out, err)
}

func (h *Handler) Rows(w http.ResponseWriter, r *http.Request) {
	projectID, ok := projectID(w, r)
	if !ok {
		return
	}
	query := RowQuery{
		Schema: r.URL.Query().Get("schema"),
		Table:  r.URL.Query().Get("table"),
		Limit:  intQuery(r, "limit", 100),
		Offset: intQuery(r, "offset", 0),
		Search: strings.TrimSpace(r.URL.Query().Get("search")),
	}
	out, err := h.service.Rows(r.Context(), projectID, query)
	writeResult(w, out, err)
}

func (h *Handler) Insert(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Allow", "GET, PATCH")
	httpx.Error(
		w,
		http.StatusMethodNotAllowed,
		"DBSTUDIO_INSERT_DISABLED",
		"DB Studio row insertion is disabled for the stable write boundary. Use the application or a dedicated database client.",
		nil,
	)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	h.mutate(w, r, h.service.Update)
}

func (h *Handler) Delete(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Allow", "GET, PATCH")
	httpx.Error(
		w,
		http.StatusMethodNotAllowed,
		"DBSTUDIO_DELETE_DISABLED",
		"DB Studio row deletion is disabled. Use the application or a dedicated database client for destructive row operations.",
		nil,
	)
}

func (h *Handler) mutate(w http.ResponseWriter, r *http.Request, fn func(context.Context, uuid.UUID, uuid.UUID, Mutation) error) {
	projectID, userID, ok := ids(w, r)
	if !ok {
		return
	}
	var req Mutation
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_JSON", "Request body must be valid JSON.", nil)
		return
	}
	if err := fn(r.Context(), projectID, userID, req); err != nil {
		httpx.DomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func ids(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	projectID, ok := projectID(w, r)
	if !ok {
		return uuid.UUID{}, uuid.UUID{}, false
	}
	user, err := auth.CurrentUser(r)
	if err != nil {
		httpx.DomainError(w, err)
		return uuid.UUID{}, uuid.UUID{}, false
	}
	return projectID, user.ID, true
}

func projectID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.DomainError(w, errs.ErrNotFound)
		return uuid.UUID{}, false
	}
	return id, true
}

func writeResult(w http.ResponseWriter, data any, err error) {
	if err != nil {
		httpx.DomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, data)
}

func intQuery(r *http.Request, key string, fallback int) int {
	value, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil {
		return fallback
	}
	return value
}
