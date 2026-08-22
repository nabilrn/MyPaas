package settings

import (
	"net/http"

	"mypaas/internal/caddy"
	"mypaas/internal/httpx"
)

type deliveryStatsResponse struct {
	Status     string               `json:"status"`
	ErrorCode  string               `json:"error_code,omitempty"`
	Caddy      *caddy.DeliveryStats `json:"caddy"`
	Cloudflare cloudflareTunnelInfo `json:"cloudflare"`
}

type cloudflareTunnelInfo struct {
	Protocol   string `json:"protocol"`
	Connectors int    `json:"connectors"`
}

// DeliveryStats exposes owner-only, low-cardinality delivery-path telemetry.
// Caddy's metrics endpoint stays private on the existing admin Unix socket;
// the dashboard receives only the compact snapshot needed for rate/latency
// derivation and never receives raw access logs.
func (h *Handler) DeliveryStats(w http.ResponseWriter, r *http.Request) {
	client := caddy.NewClient(h.cfg.CaddyAdmin, h.cfg.CaddyUpstreamHost)
	stats, err := client.DeliveryStats(r.Context())
	if err != nil {
		httpx.JSON(w, http.StatusOK, deliveryStatsResponse{
			Status:     "unavailable",
			ErrorCode:  "CADDY_METRICS_UNAVAILABLE",
			Cloudflare: h.cloudflareTunnelInfo(),
		})
		return
	}

	httpx.JSON(w, http.StatusOK, deliveryStatsResponse{
		Status:     "available",
		Caddy:      &stats,
		Cloudflare: h.cloudflareTunnelInfo(),
	})
}

func (h *Handler) cloudflareTunnelInfo() cloudflareTunnelInfo {
	return cloudflareTunnelInfo{
		Protocol:   h.cfg.CloudflareTunnelProtocol,
		Connectors: h.cfg.CloudflareTunnelConnectors,
	}
}
