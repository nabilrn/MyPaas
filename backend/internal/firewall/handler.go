package firewall

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"mypaas/internal/container"
	"mypaas/internal/httpx"
	"mypaas/internal/port"
)

type Handler struct {
	ports    *port.Service
	client   *Client
	bindHost string
}

type Overview struct {
	BindHost    string                       `json:"bindHost"`
	Allocations []port.Allocation            `json:"allocations"`
	Firewall    Status                       `json:"firewall"`
	Containers  []container.RuntimeContainer `json:"containers,omitempty"`
}

func NewHandler(ports *port.Service, client *Client, bindHost string) *Handler {
	return &Handler{ports: ports, client: client, bindHost: strings.TrimSpace(bindHost)}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	allocations, err := h.ports.ListInUse(r.Context())
	if err != nil {
		httpx.DomainError(w, err)
		return
	}

	status, err := h.client.Status(r.Context())
	if err != nil {
		status = Status{Available: false, Active: false, Rules: []Rule{}}
	}
	if status.Rules == nil {
		status.Rules = []Rule{}
	}
	if allocations == nil {
		allocations = []port.Allocation{}
	}

	var runtimeContainers []container.RuntimeContainer
	if r.URL.Query().Get("includeContainers") == "true" {
		runtimeContainers, err = container.NewDockerCLI(h.bindHost).RuntimeContainers(r.Context())
		if err != nil {
			httpx.Error(w, http.StatusServiceUnavailable, "CONTAINER_INVENTORY_FAILED", err.Error(), nil)
			return
		}
	}

	httpx.JSON(w, http.StatusOK, Overview{
		BindHost:    h.bindHost,
		Allocations: allocations,
		Firewall:    status,
		Containers:  runtimeContainers,
	})
}

func (h *Handler) Allow(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Port     int32  `json:"port"`
		Protocol string `json:"protocol"`
	}
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_JSON", "Request body must be valid JSON.", nil)
		return
	}
	if err := h.client.Allow(r.Context(), req.Port, req.Protocol); err != nil {
		httpx.Error(w, http.StatusServiceUnavailable, "FIREWALL_OPERATION_FAILED", err.Error(), nil)
		return
	}
	httpx.NoContent(w)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	value, err := strconv.Atoi(chi.URLParam(r, "port"))
	if err != nil || value < 1 || value > 65535 {
		httpx.Error(w, http.StatusBadRequest, "INVALID_PORT", "Port must be between 1 and 65535.", nil)
		return
	}
	protocol := chi.URLParam(r, "protocol")
	if err := h.client.Delete(r.Context(), int32(value), protocol); err != nil {
		httpx.Error(w, http.StatusServiceUnavailable, "FIREWALL_OPERATION_FAILED", err.Error(), nil)
		return
	}
	httpx.NoContent(w)
}
