package settings

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mypaas/internal/config"
	"mypaas/internal/db"
	"mypaas/internal/statd"
)

func TestUpdateSystemQueuesHostRequest(t *testing.T) {
	requestPath := filepath.Join(t.TempDir(), "run", "update.request")
	h := &Handler{updateRequestPath: requestPath}

	for attempt := 0; attempt < 2; attempt++ {
		response := httptest.NewRecorder()
		h.UpdateSystem(response, httptest.NewRequest(http.MethodPost, "/admin/update", nil))
		if response.Code != http.StatusAccepted {
			t.Fatalf("attempt %d status = %d, want %d: %s", attempt+1, response.Code, http.StatusAccepted, response.Body.String())
		}
	}
	if _, err := os.Stat(requestPath); err != nil {
		t.Fatalf("update request was not created: %v", err)
	}
}

func TestUpdateS3ConfigValidateOnlyDoesNotRequirePersistence(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodHead, http.MethodPut:
			w.WriteHeader(http.StatusOK)
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	payload := `{"s3_endpoint":"` + server.URL + `","s3_bucket":"mypaas-backups","s3_region":"auto","s3_access_key":"test-access-key","s3_secret_key":"test-secret-key"}`
	request := httptest.NewRequest(http.MethodPost, "/admin/settings/s3?validate=1", strings.NewReader(payload))
	response := httptest.NewRecorder()
	h := &Handler{cfg: &config.Config{}}
	h.UpdateS3Config(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"valid":true`) {
		t.Fatalf("response = %s, want valid=true", response.Body.String())
	}
}

func TestUpdateS3ConfigRejectsUnreachableStorage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	payload := `{"s3_endpoint":"` + server.URL + `","s3_bucket":"mypaas-backups","s3_region":"auto","s3_access_key":"bad-access-key","s3_secret_key":"bad-secret-key"}`
	request := httptest.NewRequest(http.MethodPost, "/admin/settings/s3?validate=1", strings.NewReader(payload))
	response := httptest.NewRecorder()
	h := &Handler{cfg: &config.Config{}}
	h.UpdateS3Config(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusUnprocessableEntity, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "BACKUP_STORAGE_VALIDATION_FAILED") {
		t.Fatalf("response = %s, want validation error code", response.Body.String())
	}
}

func TestApplyStoredRowsRehydratesS3Config(t *testing.T) {
	cfg := &config.Config{}
	h := &Handler{cfg: cfg}
	h.applyStoredRows([]db.PlatformSetting{
		{Key: "s3_endpoint", Value: json.RawMessage(`"https://account.r2.cloudflarestorage.com/"`)},
		{Key: "s3_bucket", Value: json.RawMessage(`"mypaas-backups"`)},
		{Key: "s3_region", Value: json.RawMessage(`"auto"`)},
		{Key: "s3_access_key", Value: json.RawMessage(`"access"`)},
		{Key: "s3_secret_key", Value: json.RawMessage(`"secret"`)},
	})

	if cfg.S3Endpoint != "https://account.r2.cloudflarestorage.com" || cfg.S3Bucket != "mypaas-backups" || cfg.S3Region != "auto" {
		t.Fatalf("storage config not rehydrated: endpoint=%q bucket=%q region=%q", cfg.S3Endpoint, cfg.S3Bucket, cfg.S3Region)
	}
	if cfg.S3AccessKey != "access" || cfg.S3SecretKey != "secret" {
		t.Fatal("storage credentials were not rehydrated")
	}
}

func TestHostTelemetryErrorCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "nil", err: nil, want: ""},
		{name: "protocol", err: &statd.ProtocolError{Code: "INVALID_REQUEST"}, want: "INVALID_REQUEST"},
		{name: "protocol without code", err: &statd.ProtocolError{}, want: "PROTOCOL_ERROR"},
		{name: "invalid config", err: statd.ErrInvalidInput, want: "INVALID_CONFIG"},
		{name: "wrapped invalid config", err: errors.Join(errors.New("outer"), statd.ErrInvalidInput), want: "INVALID_CONFIG"},
		{name: "io", err: errors.New("dial unix: permission denied"), want: "CONNECT_OR_IO_ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hostTelemetryErrorCode(tt.err); got != tt.want {
				t.Fatalf("hostTelemetryErrorCode() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateSettings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		values  map[string]float64
		wantErr bool
	}{
		{name: "valid", values: map[string]float64{"user_ram_quota_gb": 4, "user_cpu_quota": 2, "max_projects": 20, "build_timeout_minutes": 15}},
		{name: "valid resource defaults", values: map[string]float64{"profile_static_memory_mb": 128, "profile_static_cpu_limit": 0.01, "profile_compose_main_memory_mb": 512, "profile_compose_main_cpu_limit": 0.5}},
		{name: "static CPU below floor", values: map[string]float64{"profile_static_cpu_limit": 0.009}, wantErr: true},
		{name: "static memory below floor", values: map[string]float64{"profile_static_memory_mb": 32}, wantErr: true},
		{name: "compose CPU below floor", values: map[string]float64{"profile_compose_main_cpu_limit": 0.2}, wantErr: true},
		{name: "deployment concurrency is installation level", values: map[string]float64{"max_concurrent_deploys": 2}, wantErr: true},
		{name: "zero quota", values: map[string]float64{"user_ram_quota_gb": 0}, wantErr: true},
		{name: "fractional projects", values: map[string]float64{"max_projects": 2.5}, wantErr: true},
		{name: "zero timeout", values: map[string]float64{"build_timeout_minutes": 0}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateSettings(tt.values)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateSettings() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSettingsDefaultsExposeRuntimeBackedValues(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		UserRAMQuotaMB:       4096,
		UserCPUQuota:         2,
		MaxProjects:          12,
		MaxConcurrentDeploys: 3,
		BuildTimeoutMinutes:  20,
	}
	h := &Handler{cfg: cfg}
	defaults := h.defaults()

	for _, key := range []string{"project_default_ram_mb", "project_default_cpu", "max_concurrent_deploys"} {
		if _, ok := defaults[key]; ok {
			t.Fatalf("%s must not be exposed without a live authoritative consumer", key)
		}
	}
	if got := defaults["user_ram_quota_gb"]; got != 4 {
		t.Fatalf("user_ram_quota_gb = %v, want 4", got)
	}
	if got := defaults["profile_static_memory_mb"]; got != 64 {
		t.Fatalf("profile_static_memory_mb = %v, want 64", got)
	}
	if got := defaults["profile_compose_main_cpu_limit"]; got != 0.35 {
		t.Fatalf("profile_compose_main_cpu_limit = %v, want 0.35", got)
	}
}

func TestApplyToConfig(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{}
	h := &Handler{cfg: cfg}
	h.applyToConfig(map[string]float64{
		"user_ram_quota_gb":     3.5,
		"user_cpu_quota":        1.5,
		"max_projects":          15,
		"build_timeout_minutes": 25,
	})

	if cfg.UserRAMQuotaMB != 3584 {
		t.Fatalf("UserRAMQuotaMB = %d, want 3584", cfg.UserRAMQuotaMB)
	}
	if cfg.UserCPUQuota != 1.5 || cfg.MaxProjects != 15 || cfg.BuildTimeoutMinutes != 25 {
		t.Fatalf("runtime config not updated: %+v", cfg)
	}
}
