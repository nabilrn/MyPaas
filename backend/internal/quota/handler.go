package quota

import (
	"net/http"
	"strconv"
	"strings"

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
	var usage Usage
	if includeRuntime(r) {
		usage, err = h.service.UsageWithRuntime(r.Context(), user.ID)
	} else {
		usage, err = h.service.Usage(r.Context(), user.ID)
	}
	if err != nil {
		httpx.DomainError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, usage)
}

func includeRuntime(r *http.Request) bool {
	value := strings.TrimSpace(r.URL.Query().Get("includeRuntime"))
	if value == "" {
		return false
	}
	enabled, err := strconv.ParseBool(value)
	return err == nil && enabled
}
