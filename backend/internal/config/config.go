package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	DatabaseURL string
	Port        int
	Environment string

	JWTSecret string

	GitHubClientID     string
	GitHubClientSecret string
	GitHubCallbackURL  string

	EncryptionKey string

	DockerSocket      string
	DockerBindHost    string
	ProjectNetwork    string
	CaddyAdmin        string
	CaddyUpstreamHost string
	StaticRoot        string
	CaddyStaticRoot   string

	FrontendURL  string
	PublicDomain string
	OwnerEmail   string

	MetricsUsername string
	MetricsPassword string

	BackupEnabled        bool
	BackupDir            string
	BackupDailyAt        string
	BackupTimeoutMinutes int
	BackupKeepDaily      int
	BackupKeepWeekly     int
	BackupWeeklyDay      time.Weekday

	SharedPostgresEnabled bool
	SharedPostgresHost    string
	SharedPostgresPort    int
	SharedPostgresSSLMode string

	ImageCleanupEnabled bool
	ImageCleanupUntil   string
	ImageCleanupWeekday time.Weekday

	MaxConcurrentDeploys int
	BuildTimeoutMinutes  int
	UserRAMQuotaMB       int32
	UserCPUQuota         float64
	MaxProjects          int32
}

func Load() (*Config, error) {
	var missing []string

	req := func(key string) string {
		v := os.Getenv(key)
		if v == "" {
			missing = append(missing, key)
		}
		return v
	}

	port, err := envInt("API_PORT", 8080)
	if err != nil {
		return nil, fmt.Errorf("API_PORT: %w", err)
	}
	maxDeploys, err := envInt("MAX_CONCURRENT_DEPLOYS", 2)
	if err != nil {
		return nil, fmt.Errorf("MAX_CONCURRENT_DEPLOYS: %w", err)
	}
	buildTimeout, err := envInt("BUILD_TIMEOUT_MINUTES", 15)
	if err != nil {
		return nil, fmt.Errorf("BUILD_TIMEOUT_MINUTES: %w", err)
	}
	ramQuotaGB, err := envFloat("USER_RAM_QUOTA_GB", 6)
	if err != nil {
		return nil, fmt.Errorf("USER_RAM_QUOTA_GB: %w", err)
	}
	cpuQuota, err := envFloat("USER_CPU_QUOTA", 3)
	if err != nil {
		return nil, fmt.Errorf("USER_CPU_QUOTA: %w", err)
	}
	maxProjects, err := envInt("MAX_PROJECTS", 20)
	if err != nil {
		return nil, fmt.Errorf("MAX_PROJECTS: %w", err)
	}
	backupEnabled, err := envBool("BACKUP_ENABLED", true)
	if err != nil {
		return nil, fmt.Errorf("BACKUP_ENABLED: %w", err)
	}
	backupTimeout, err := envInt("BACKUP_TIMEOUT_MINUTES", 30)
	if err != nil {
		return nil, fmt.Errorf("BACKUP_TIMEOUT_MINUTES: %w", err)
	}
	backupKeepDaily, err := envInt("BACKUP_KEEP_DAILY", 7)
	if err != nil {
		return nil, fmt.Errorf("BACKUP_KEEP_DAILY: %w", err)
	}
	backupKeepWeekly, err := envInt("BACKUP_KEEP_WEEKLY", 4)
	if err != nil {
		return nil, fmt.Errorf("BACKUP_KEEP_WEEKLY: %w", err)
	}
	backupWeeklyDay, err := envWeekday("BACKUP_WEEKLY_DAY", time.Sunday)
	if err != nil {
		return nil, fmt.Errorf("BACKUP_WEEKLY_DAY: %w", err)
	}
	projectNetwork := envStr("PROJECT_NETWORK", "")
	sharedPostgresHostDefault := "host.docker.internal"
	if projectNetwork != "" {
		sharedPostgresHostDefault = "postgres"
	}
	sharedPostgresEnabled, err := envBool("SHARED_POSTGRES_ENABLED", true)
	if err != nil {
		return nil, fmt.Errorf("SHARED_POSTGRES_ENABLED: %w", err)
	}
	sharedPostgresPort, err := envInt("SHARED_POSTGRES_PORT", 5432)
	if err != nil {
		return nil, fmt.Errorf("SHARED_POSTGRES_PORT: %w", err)
	}
	imageCleanupEnabled, err := envBool("IMAGE_CLEANUP_ENABLED", true)
	if err != nil {
		return nil, fmt.Errorf("IMAGE_CLEANUP_ENABLED: %w", err)
	}
	imageCleanupWeekday, err := envWeekday("IMAGE_CLEANUP_WEEKDAY", time.Sunday)
	if err != nil {
		return nil, fmt.Errorf("IMAGE_CLEANUP_WEEKDAY: %w", err)
	}

	cfg := &Config{
		DatabaseURL: req("DATABASE_URL"),
		Port:        port,
		Environment: envStr("ENVIRONMENT", "development"),

		JWTSecret: req("JWT_SECRET"),

		GitHubClientID:     req("GITHUB_CLIENT_ID"),
		GitHubClientSecret: req("GITHUB_CLIENT_SECRET"),
		GitHubCallbackURL:  req("GITHUB_CALLBACK_URL"),

		EncryptionKey: req("ENCRYPTION_KEY"),

		DockerSocket:      envStr("DOCKER_SOCKET", "/var/run/docker.sock"),
		DockerBindHost:    envStr("DOCKER_BIND_HOST", "127.0.0.1"),
		ProjectNetwork:    projectNetwork,
		CaddyAdmin:        envStr("CADDY_ADMIN", "127.0.0.1:2019"),
		CaddyUpstreamHost: envStr("CADDY_UPSTREAM_HOST", "host.docker.internal"),
		StaticRoot:        envPath("STATIC_ROOT", "/var/lib/mypaas/static"),
		CaddyStaticRoot:   envStr("CADDY_STATIC_ROOT", envStr("STATIC_ROOT", "/var/lib/mypaas/static")),

		FrontendURL:  envStr("FRONTEND_URL", "http://localhost:5173"),
		PublicDomain: envStr("PUBLIC_DOMAIN", "localhost"),
		OwnerEmail:   envStr("OWNER_EMAIL", ""),

		MetricsUsername: envStr("METRICS_USERNAME", ""),
		MetricsPassword: envStr("METRICS_PASSWORD", ""),

		BackupEnabled:        backupEnabled,
		BackupDir:            envPath("BACKUP_DIR", "/var/lib/mypaas/backups"),
		BackupDailyAt:        envStr("BACKUP_DAILY_AT", "02:00"),
		BackupTimeoutMinutes: backupTimeout,
		BackupKeepDaily:      backupKeepDaily,
		BackupKeepWeekly:     backupKeepWeekly,
		BackupWeeklyDay:      backupWeeklyDay,

		SharedPostgresEnabled: sharedPostgresEnabled,
		SharedPostgresHost:    envStr("SHARED_POSTGRES_HOST", sharedPostgresHostDefault),
		SharedPostgresPort:    sharedPostgresPort,
		SharedPostgresSSLMode: envStr("SHARED_POSTGRES_SSLMODE", "disable"),

		ImageCleanupEnabled: imageCleanupEnabled,
		ImageCleanupUntil:   envStr("IMAGE_CLEANUP_UNTIL", "168h"),
		ImageCleanupWeekday: imageCleanupWeekday,

		MaxConcurrentDeploys: maxDeploys,
		BuildTimeoutMinutes:  buildTimeout,
		UserRAMQuotaMB:       int32(ramQuotaGB * 1024),
		UserCPUQuota:         cpuQuota,
		MaxProjects:          int32(maxProjects),
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return cfg, nil
}

func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

func envStr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envPath(key, def string) string {
	value := envStr(key, def)
	if value == "" || isAbsolutePath(value) {
		return value
	}
	configDir := strings.TrimSpace(os.Getenv("MYPAAS_CONFIG_DIR"))
	if configDir == "" {
		return value
	}
	return filepath.Clean(filepath.Join(configDir, value))
}

func isAbsolutePath(value string) bool {
	return filepath.IsAbs(value) || strings.HasPrefix(value, "/") || strings.HasPrefix(value, "\\")
}

func envInt(key string, def int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("invalid integer %q", v)
	}
	return n, nil
}

func envFloat(key string, def float64) (float64, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	n, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number %q", v)
	}
	return n, nil
}

func envBool(key string, def bool) (bool, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "y", "on":
		return true, nil
	case "0", "false", "no", "n", "off":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean %q", v)
	}
}

func envWeekday(key string, def time.Weekday) (time.Weekday, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "sunday", "sun":
		return time.Sunday, nil
	case "monday", "mon":
		return time.Monday, nil
	case "tuesday", "tue":
		return time.Tuesday, nil
	case "wednesday", "wed":
		return time.Wednesday, nil
	case "thursday", "thu":
		return time.Thursday, nil
	case "friday", "fri":
		return time.Friday, nil
	case "saturday", "sat":
		return time.Saturday, nil
	default:
		return 0, fmt.Errorf("invalid weekday %q", v)
	}
}
