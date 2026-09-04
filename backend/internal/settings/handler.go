package settings

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mypaas/internal/backup"
	"mypaas/internal/config"
	"mypaas/internal/db"
	"mypaas/internal/host"
	"mypaas/internal/httpx"
	"mypaas/internal/resourceprofile"
	"mypaas/internal/statd"
)

// settingKeys lists numeric settings with an authoritative live runtime
// consumer. Deployment concurrency remains installation-level because the
// worker semaphore is created at process startup.
var settingKeys = []string{
	"user_ram_quota_gb",
	"user_cpu_quota",
	"max_projects",
	"build_timeout_minutes",
	"profile_static_memory_mb",
	"profile_static_cpu_limit",
	"profile_go_small_memory_mb",
	"profile_go_small_cpu_limit",
	"profile_node_python_memory_mb",
	"profile_node_python_cpu_limit",
	"profile_compose_main_memory_mb",
	"profile_compose_main_cpu_limit",
}

type Handler struct {
	queries           *db.Queries
	cfg               *config.Config
	backupService     *backup.Service
	updateRequestPath string
}

func NewHandler(queries *db.Queries, cfg *config.Config, backupService *backup.Service) *Handler {
	h := &Handler{
		queries:           queries,
		cfg:               cfg,
		backupService:     backupService,
		updateRequestPath: "/run/mypaas/update.request",
	}
	// The DB is the persisted source for owner-edited live settings. Rehydrate
	// the shared config before the HTTP server starts accepting requests so a
	// process restart cannot silently revert quota/build/backup behavior to env values.
	if rows, err := queries.GetAllSettings(context.Background()); err == nil {
		h.applyStoredRows(rows)
	}
	return h
}

// Get returns the effective live platform settings. DB overrides are applied
// to the same shared config before the response is produced so displayed and
// enforced values cannot diverge.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	rows, err := h.queries.GetAllSettings(r.Context())
	if err != nil {
		httpx.DomainError(w, err)
		return
	}
	h.applyStoredRows(rows)

	res := make(map[string]interface{})
	for k, v := range h.defaults() {
		res[k] = v
	}
	res["mcp_api_token"] = h.cfg.ApiToken
	res["cloudflare_configured"] = h.cfg.CloudflareAPIToken != "" && h.cfg.CloudflareZoneID != ""
	res["s3_configured"] = h.cfg.S3Endpoint != "" && h.cfg.S3Bucket != "" && h.cfg.S3AccessKey != "" && h.cfg.S3SecretKey != ""
	res["s3_endpoint"] = h.cfg.S3Endpoint
	res["s3_bucket"] = h.cfg.S3Bucket
	res["s3_region"] = h.cfg.S3Region
	res["build_sha"] = strings.TrimSpace(os.Getenv("MYPAAS_BUILD_SHA"))

	httpx.JSON(w, http.StatusOK, res)
}

func (h *Handler) RegenerateMCPToken(w http.ResponseWriter, r *http.Request) {
	randBytes := make([]byte, 24)
	_, _ = rand.Read(randBytes)
	newToken := "mp_" + hex.EncodeToString(randBytes)

	rawToken, _ := json.Marshal(newToken)
	err := h.queries.UpsertSetting(r.Context(), db.UpsertSettingParams{
		Key:   "mypaas_api_token",
		Value: rawToken,
	})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_WRITE_FAILED", "Failed to save token to database: "+err.Error(), nil)
		return
	}

	h.cfg.ApiToken = newToken
	h.Get(w, r)
}

type cloudflareReq struct {
	Token  string `json:"token"`
	ZoneID string `json:"zone_id"`
}

func (h *Handler) UpdateCloudflareConfig(w http.ResponseWriter, r *http.Request) {
	var req cloudflareReq
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body", nil)
		return
	}

	rawToken, _ := json.Marshal(req.Token)
	if err := h.queries.UpsertSetting(r.Context(), db.UpsertSettingParams{
		Key:   "cloudflare_api_token",
		Value: rawToken,
	}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_WRITE_FAILED", "Failed to save cloudflare token", nil)
		return
	}

	rawZone, _ := json.Marshal(req.ZoneID)
	if err := h.queries.UpsertSetting(r.Context(), db.UpsertSettingParams{
		Key:   "cloudflare_zone_id",
		Value: rawZone,
	}); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "DB_WRITE_FAILED", "Failed to save zone ID", nil)
		return
	}

	h.cfg.CloudflareAPIToken = req.Token
	h.cfg.CloudflareZoneID = req.ZoneID

	h.Get(w, r)
}

type s3Req struct {
	Endpoint  string `json:"s3_endpoint"`
	Bucket    string `json:"s3_bucket"`
	AccessKey string `json:"s3_access_key"`
	SecretKey string `json:"s3_secret_key"`
	Region    string `json:"s3_region"`
}

func (h *Handler) UpdateS3Config(w http.ResponseWriter, r *http.Request) {
	var req s3Req
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_BODY", "Invalid request body", nil)
		return
	}

	req.Endpoint = strings.TrimRight(strings.TrimSpace(req.Endpoint), "/")
	req.Bucket = strings.TrimSpace(req.Bucket)
	req.AccessKey = strings.TrimSpace(req.AccessKey)
	req.Region = strings.TrimSpace(req.Region)
	if req.Region == "" {
		req.Region = "auto"
	}

	validationCtx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	if err := backup.ValidateS3Connection(validationCtx, backup.S3Config{
		Endpoint:  req.Endpoint,
		Bucket:    req.Bucket,
		Region:    req.Region,
		AccessKey: req.AccessKey,
		SecretKey: req.SecretKey,
	}); err != nil {
		httpx.Error(w, http.StatusUnprocessableEntity, "BACKUP_STORAGE_VALIDATION_FAILED", "Could not validate backup storage: "+err.Error(), nil)
		return
	}

	if r.URL.Query().Get("validate") == "1" {
		httpx.JSON(w, http.StatusOK, map[string]interface{}{
			"valid":  true,
			"bucket": req.Bucket,
		})
		return
	}

	settings := map[string]string{
		"s3_endpoint":   req.Endpoint,
		"s3_bucket":     req.Bucket,
		"s3_access_key": req.AccessKey,
		"s3_secret_key": req.SecretKey,
		"s3_region":     req.Region,
	}

	for k, v := range settings {
		raw, _ := json.Marshal(v)
		if err := h.queries.UpsertSetting(r.Context(), db.UpsertSettingParams{
			Key:   k,
			Value: raw,
		}); err != nil {
			httpx.Error(w, http.StatusInternalServerError, "DB_WRITE_FAILED", "Failed to save "+k, nil)
			return
		}
	}

	h.cfg.S3Endpoint = req.Endpoint
	h.cfg.S3Bucket = req.Bucket
	h.cfg.S3AccessKey = req.AccessKey
	h.cfg.S3SecretKey = req.SecretKey
	h.cfg.S3Region = req.Region

	h.Get(w, r)
}

func (h *Handler) TriggerBackup(w http.ResponseWriter, r *http.Request) {
	if h.backupService == nil {
		httpx.Error(w, http.StatusServiceUnavailable, "BACKUP_NOT_CONFIGURED", "Backup service is not available", nil)
		return
	}

	// Run in background to avoid blocking the HTTP response
	go func() {
		// Use a detached context for the background operation
		_, err := h.backupService.Run(context.Background())
		if err != nil {
			// Just log the error, the backup service handles its own logging mostly
		}
	}()

	httpx.JSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}

func (h *Handler) DownloadBackup(w http.ResponseWriter, r *http.Request) {
	if h.backupService == nil {
		httpx.Error(w, http.StatusServiceUnavailable, "BACKUP_NOT_CONFIGURED", "Backup service is not available", nil)
		return
	}

	result, err := h.backupService.Run(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "BACKUP_FAILED", "Failed to generate backup: "+err.Error(), nil)
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filepath.Base(result.DailyPath)))
	w.Header().Set("Content-Type", "application/gzip")
	http.ServeFile(w, r, result.DailyPath)
}

func (h *Handler) UpdateSystem(w http.ResponseWriter, r *http.Request) {
	requestPath := h.updateRequestPath
	if requestPath == "" {
		requestPath = "/run/mypaas/update.request"
	}
	if err := os.MkdirAll(filepath.Dir(requestPath), 0o755); err != nil {
		httpx.Error(w, http.StatusServiceUnavailable, "UPDATE_TRIGGER_UNAVAILABLE", "Host update trigger is unavailable", nil)
		return
	}
	file, err := os.OpenFile(requestPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil && !errors.Is(err, os.ErrExist) {
		httpx.Error(w, http.StatusServiceUnavailable, "UPDATE_TRIGGER_UNAVAILABLE", "Host update trigger is unavailable", nil)
		return
	}
	if file != nil {
		_ = file.Close()
	}

	httpx.JSON(w, http.StatusAccepted, map[string]bool{"queued": true})
}

// Update upserts one or more platform settings and applies the supported
// runtime values to the in-memory config.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var req map[string]float64
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_BODY", "Request body must be a JSON object with numeric values.", nil)
		return
	}
	if err := validateSettings(req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "INVALID_SETTING", err.Error(), nil)
		return
	}

	for key, value := range req {
		raw, err := json.Marshal(value)
		if err != nil {
			httpx.Error(w, http.StatusBadRequest, "INVALID_VALUE", "Cannot encode value for "+key, nil)
			return
		}
		if err := h.queries.UpsertSetting(r.Context(), db.UpsertSettingParams{
			Key:   key,
			Value: raw,
		}); err != nil {
			httpx.DomainError(w, err)
			return
		}
	}

	h.applyToConfig(req)
	h.Get(w, r)
}

func validateSettings(values map[string]float64) error {
	allowed := make(map[string]struct{}, len(settingKeys))
	for _, key := range settingKeys {
		allowed[key] = struct{}{}
	}
	for key, value := range values {
		if _, ok := allowed[key]; !ok {
			return fmt.Errorf("unknown platform setting %q", key)
		}
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return fmt.Errorf("%s must be a finite number", key)
		}
		switch key {
		case "user_ram_quota_gb":
			if value <= 0 || value > 64 {
				return errors.New("user RAM quota must be greater than 0 and at most 64 GB")
			}
		case "user_cpu_quota":
			if value <= 0 || value > 32 {
				return errors.New("user CPU quota must be greater than 0 and at most 32 cores")
			}
		case "max_projects":
			if value < 1 || value > 500 || value != math.Trunc(value) {
				return errors.New("maximum projects must be a whole number between 1 and 500")
			}
		case "build_timeout_minutes":
			if value < 1 || value > 1440 || value != math.Trunc(value) {
				return errors.New("build timeout must be a whole number between 1 and 1440 minutes")
			}
		case "profile_static_memory_mb":
			if value < 64 || value > 32768 || value != math.Trunc(value) {
				return errors.New("static memory must be a whole number between 64 and 32768 MB")
			}
		case "profile_go_small_memory_mb":
			if value < 128 || value > 32768 || value != math.Trunc(value) {
				return errors.New("Go small memory must be a whole number between 128 and 32768 MB")
			}
		case "profile_node_python_memory_mb", "profile_compose_main_memory_mb":
			if value < 256 || value > 32768 || value != math.Trunc(value) {
				return errors.New("Node/Python and Compose memory must be a whole number between 256 and 32768 MB")
			}
		case "profile_static_cpu_limit":
			if value < 0.10 || value > 32 {
				return errors.New("static CPU must be between 0.10 and 32 cores")
			}
		case "profile_go_small_cpu_limit":
			if value < 0.20 || value > 32 {
				return errors.New("Go small CPU must be between 0.20 and 32 cores")
			}
		case "profile_node_python_cpu_limit", "profile_compose_main_cpu_limit":
			if value < 0.35 || value > 32 {
				return errors.New("Node/Python and Compose CPU must be between 0.35 and 32 cores")
			}
		}
	}
	return nil
}

func (h *Handler) applyStoredRows(rows []db.PlatformSetting) {
	values := make(map[string]float64)
	for _, row := range rows {
		switch row.Key {
		case "s3_endpoint", "s3_bucket", "s3_access_key", "s3_secret_key", "s3_region":
			var value string
			if json.Unmarshal(row.Value, &value) != nil {
				continue
			}
			switch row.Key {
			case "s3_endpoint":
				h.cfg.S3Endpoint = strings.TrimRight(strings.TrimSpace(value), "/")
			case "s3_bucket":
				h.cfg.S3Bucket = strings.TrimSpace(value)
			case "s3_access_key":
				h.cfg.S3AccessKey = strings.TrimSpace(value)
			case "s3_secret_key":
				h.cfg.S3SecretKey = value
			case "s3_region":
				h.cfg.S3Region = strings.TrimSpace(value)
				if h.cfg.S3Region == "" {
					h.cfg.S3Region = "auto"
				}
			}
			continue
		}

		if !isSettingKey(row.Key) {
			continue
		}
		var value float64
		if json.Unmarshal(row.Value, &value) != nil {
			continue
		}
		candidate := map[string]float64{row.Key: value}
		if validateSettings(candidate) != nil {
			continue
		}
		values[row.Key] = value
	}
	h.applyToConfig(values)
}

func isSettingKey(key string) bool {
	for _, allowed := range settingKeys {
		if key == allowed {
			return true
		}
	}
	return false
}

type hostStatsResponse struct {
	HostRAMBytes       int64                      `json:"host_ram_bytes"`
	HostCPUCores       int                        `json:"host_cpu_cores"`
	AllocatedRAMMB     int32                      `json:"allocated_ram_mb"`
	AllocatedCPU       float64                    `json:"allocated_cpu"`
	TelemetryStatus    string                     `json:"telemetry_status"`
	TelemetryErrorCode string                     `json:"telemetry_error_code,omitempty"`
	Memory             *statd.HostMemorySnapshot  `json:"memory"`
	CPU                *statd.HostCPUSnapshot     `json:"cpu"`
	Storage            *statd.HostStorageSnapshot `json:"storage"`
	Network            *statd.HostNetworkSnapshot `json:"network"`
}

func hostTelemetryErrorCode(err error) string {
	if err == nil {
		return ""
	}
	var protocolErr *statd.ProtocolError
	if errors.As(err, &protocolErr) {
		if protocolErr.Code != "" {
			return protocolErr.Code
		}
		return "PROTOCOL_ERROR"
	}
	if errors.Is(err, statd.ErrInvalidInput) {
		return "INVALID_CONFIG"
	}
	return "CONNECT_OR_IO_ERROR"
}

// HostStats returns host capacity plus optional host telemetry from mypaas-statd.
// Capacity/allocation data remains usable when host telemetry is disabled or unavailable.
// telemetry_status and telemetry_error_code make the fail-open path observable without
// exposing raw socket or filesystem errors to the dashboard.
func (h *Handler) HostStats(w http.ResponseWriter, r *http.Request) {
	cap := host.GetCapacity()

	usage, err := h.queries.GetGlobalResourceUsage(r.Context())
	var allocatedRAM int32
	var allocatedCPU float64
	if err == nil {
		allocatedRAM = usage.TotalMemoryMb
		if usage.TotalCpu.Valid && usage.TotalCpu.Int != nil {
			cpuVal, _ := usage.TotalCpu.Float64Value()
			allocatedCPU = cpuVal.Float64
		}
	}

	var memory *statd.HostMemorySnapshot
	var cpu *statd.HostCPUSnapshot
	var storage *statd.HostStorageSnapshot
	var network *statd.HostNetworkSnapshot
	telemetryStatus := "disabled"
	telemetryErrorCode := ""
	if socketPath := strings.TrimSpace(os.Getenv("STATD_SOCKET")); socketPath != "" {
		telemetryStatus = "unavailable"
		snapshot, snapshotErr := statd.NewClient(socketPath).HostSnapshot(r.Context())
		if snapshotErr == nil {
			memory = snapshot.Memory
			cpu = snapshot.CPU
			storage = snapshot.Storage
			network = snapshot.Network
			if memory != nil || cpu != nil || storage != nil || network != nil {
				telemetryStatus = "available"
			} else {
				telemetryErrorCode = "EMPTY_SNAPSHOT"
			}
		} else {
			telemetryErrorCode = hostTelemetryErrorCode(snapshotErr)
		}
	}

	httpx.JSON(w, http.StatusOK, hostStatsResponse{
		HostRAMBytes:       cap.TotalRAMBytes,
		HostCPUCores:       cap.TotalCPUCores,
		AllocatedRAMMB:     allocatedRAM,
		AllocatedCPU:       allocatedCPU,
		TelemetryStatus:    telemetryStatus,
		TelemetryErrorCode: telemetryErrorCode,
		Memory:             memory,
		CPU:                cpu,
		Storage:            storage,
		Network:            network,
	})
}

func (h *Handler) defaults() map[string]float64 {
	values := map[string]float64{
		"user_ram_quota_gb":     float64(h.cfg.UserRAMQuotaMB) / 1024,
		"user_cpu_quota":        h.cfg.UserCPUQuota,
		"max_projects":          float64(h.cfg.MaxProjects),
		"build_timeout_minutes": float64(h.cfg.BuildTimeoutMinutes),
	}
	for _, profile := range []struct {
		id     string
		prefix string
	}{
		{resourceprofile.Static, "profile_static"},
		{resourceprofile.GoSmall, "profile_go_small"},
		{resourceprofile.NodePython, "profile_node_python"},
		{resourceprofile.ComposeMain, "profile_compose_main"},
	} {
		current, err := resourceprofile.Get(profile.id)
		if err != nil {
			continue
		}
		values[profile.prefix+"_memory_mb"] = float64(current.MemoryMB)
		values[profile.prefix+"_cpu_limit"] = current.CPULimit
	}
	return values
}

func (h *Handler) applyToConfig(values map[string]float64) {
	if v, ok := values["user_ram_quota_gb"]; ok {
		h.cfg.UserRAMQuotaMB = int32(v * 1024)
	}
	if v, ok := values["user_cpu_quota"]; ok {
		h.cfg.UserCPUQuota = v
	}
	if v, ok := values["max_projects"]; ok {
		h.cfg.MaxProjects = int32(v)
	}
	if v, ok := values["build_timeout_minutes"]; ok {
		h.cfg.BuildTimeoutMinutes = int(v)
	}
	h.applyResourceProfileDefaults(values)
}

func (h *Handler) applyResourceProfileDefaults(values map[string]float64) {
	configured := make(map[string]resourceprofile.Profile)
	for _, profile := range []struct {
		id     string
		prefix string
	}{
		{resourceprofile.Static, "profile_static"},
		{resourceprofile.GoSmall, "profile_go_small"},
		{resourceprofile.NodePython, "profile_node_python"},
		{resourceprofile.ComposeMain, "profile_compose_main"},
	} {
		memory, hasMemory := values[profile.prefix+"_memory_mb"]
		cpu, hasCPU := values[profile.prefix+"_cpu_limit"]
		if !hasMemory && !hasCPU {
			continue
		}
		current, err := resourceprofile.Get(profile.id)
		if err != nil {
			continue
		}
		if hasMemory {
			current.MemoryMB = int32(memory)
		}
		if hasCPU {
			current.CPULimit = cpu
		}
		configured[profile.id] = current
	}
	_ = resourceprofile.ConfigureDefaults(configured)
}
