package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
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

	DockerSocket string
	CaddyAdmin   string

	FrontendURL  string
	PublicDomain string
	OwnerEmail   string

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

	cfg := &Config{
		DatabaseURL: req("DATABASE_URL"),
		Port:        port,
		Environment: envStr("ENVIRONMENT", "development"),

		JWTSecret: req("JWT_SECRET"),

		GitHubClientID:     req("GITHUB_CLIENT_ID"),
		GitHubClientSecret: req("GITHUB_CLIENT_SECRET"),
		GitHubCallbackURL:  req("GITHUB_CALLBACK_URL"),

		EncryptionKey: req("ENCRYPTION_KEY"),

		DockerSocket: envStr("DOCKER_SOCKET", "/var/run/docker.sock"),
		CaddyAdmin:   envStr("CADDY_ADMIN", "127.0.0.1:2019"),

		FrontendURL:  envStr("FRONTEND_URL", "http://localhost:5173"),
		PublicDomain: envStr("PUBLIC_DOMAIN", "localhost"),
		OwnerEmail:   envStr("OWNER_EMAIL", ""),

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
