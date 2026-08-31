package shell

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

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

func (h *Handler) Start(w http.ResponseWriter, r *http.Request) {
	info, err := h.service.Start(r.Context())
	if err != nil {
		httpx.DomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, info)
}

func (h *Handler) Stream(w http.ResponseWriter, r *http.Request) {
	id, err := sessionID(r)
	if err != nil {
		httpx.DomainError(w, err)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		httpx.Error(w, http.StatusInternalServerError, "STREAM_UNSUPPORTED", "Streaming is not supported by this response writer.", nil)
		return
	}
	events, done, _, unsubscribe, err := h.service.Subscribe(id)
	if err != nil {
		httpx.DomainError(w, err)
		return
	}
	defer unsubscribe()
	_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	if !sendEvent(w, flusher, Event{Type: "ready", Data: "Shell session connected."}) {
		return
	}

	for {
		select {
		case <-r.Context().Done():
			return
		case event := <-events:
			if !sendEvent(w, flusher, event) {
				return
			}
		case <-done:
			return
		}
	}
}

func (h *Handler) Input(w http.ResponseWriter, r *http.Request) {
	id, err := sessionID(r)
	if err != nil {
		httpx.DomainError(w, err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxInputBytes+1024)
	var req struct {
		Data string `json:"data"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_JSON", "Shell input must be valid JSON.", nil)
		return
	}
	if err := h.service.Write(id, req.Data); err != nil {
		httpx.DomainError(w, err)
		return
	}
	httpx.NoContent(w)
}

func (h *Handler) Stop(w http.ResponseWriter, r *http.Request) {
	id, err := sessionID(r)
	if err != nil {
		httpx.DomainError(w, err)
		return
	}
	if err := h.service.Stop(id); err != nil {
		httpx.DomainError(w, err)
		return
	}
	httpx.NoContent(w)
}

func sessionID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uuid.Nil, errs.ErrShellSessionNotFound
	}
	return id, nil
}

func sendEvent(w http.ResponseWriter, flusher http.Flusher, event Event) bool {
	payload, err := json.Marshal(event.Data)
	if err != nil {
		return false
	}
	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, payload); err != nil {
		return false
	}
	flusher.Flush()
	return true
}
