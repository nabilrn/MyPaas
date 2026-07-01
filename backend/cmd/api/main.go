package main

import (
	"context"
	"crypto/subtle"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"mypaas/internal/auth"
	"mypaas/internal/caddy"
	"mypaas/internal/config"
	"mypaas/internal/container"
	"mypaas/internal/crypto"
	"mypaas/internal/db"
	"mypaas/internal/deployment"
	"mypaas/internal/envvar"
	"mypaas/internal/logger"
	"mypaas/internal/port"
	"mypaas/internal/project"
	"mypaas/internal/quota"
	"mypaas/internal/user"
	"mypaas/internal/webhook"
)

var processStartedAt = time.Now()

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run() error {
	// Load .env if present; not fatal if missing (production uses real env vars).
	for _, path := range []string{".env", "../.env", "../../.env", "../../../.env"} {
		if err := godotenv.Load(path); err == nil {
			break
		}
	}

	logger.Setup(os.Getenv("LOG_LEVEL"))

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	pool, err := db.Connect(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	defer pool.Close()

	tokenService, err := auth.NewTokenService(cfg.JWTSecret)
	if err != nil {
		return fmt.Errorf("jwt: %w", err)
	}
	cipher, err := crypto.NewAESGCM(cfg.EncryptionKey)
	if err != nil {
		return fmt.Errorf("crypto: %w", err)
	}
	queries := db.New(pool)
	if err := seedOwner(context.Background(), queries, cfg.OwnerEmail); err != nil {
		return fmt.Errorf("seed owner: %w", err)
	}
	if err := recoverInterruptedDeployments(context.Background(), queries); err != nil {
		return fmt.Errorf("recover interrupted deployments: %w", err)
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      buildRouter(cfg, pool, tokenService, cipher),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("server started", "addr", srv.Addr, "env", cfg.Environment)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	select {
	case err := <-errCh:
		return fmt.Errorf("server: %w", err)
	case sig := <-quit:
		slog.Info("shutting down", "signal", sig)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}

	slog.Info("server stopped cleanly")
	return nil
}

func buildRouter(cfg *config.Config, pool *pgxpool.Pool, tokenService *auth.TokenService, cipher *crypto.AESGCM) http.Handler {
	queries := db.New(pool)
	authHandler := auth.NewHandler(cfg, queries, tokenService)
	authMiddleware := auth.Middleware(tokenService, queries)
	envService := envvar.NewService(queries, cipher)
	portService := port.NewService(pool)
	quotaService := quota.NewService(queries, cfg)
	deploymentService := deployment.NewService(cfg, queries, envService, portService, caddy.NewClient(cfg.CaddyAdmin, cfg.CaddyUpstreamHost), container.NewDockerCLI(cfg.DockerBindHost))
	projectHandler := project.NewHandler(project.NewService(queries, cfg.PublicDomain, quotaService), func(r *http.Request, id uuid.UUID) error {
		return deploymentService.CleanupProject(r.Context(), id)
	})
	deploymentHandler := deployment.NewHandler(deploymentService)
	envHandler := envvar.NewHandler(envService)
	quotaHandler := quota.NewHandler(quotaService)
	userHandler := user.NewHandler(queries)
	webhookHandler := webhook.NewHandler(queries, deploymentService)

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware(cfg))
	r.Use(timeoutExceptStreams(60 * time.Second))

	r.Get("/metrics", handleMetrics(cfg, processStartedAt))
	registerRoutes(r, pool, authMiddleware, authHandler, projectHandler, deploymentHandler, envHandler, quotaHandler, userHandler, webhookHandler)
	r.Route("/api", func(r chi.Router) {
		registerRoutes(r, pool, authMiddleware, authHandler, projectHandler, deploymentHandler, envHandler, quotaHandler, userHandler, webhookHandler)
	})

	return r
}

func registerRoutes(
	r chi.Router,
	pool *pgxpool.Pool,
	authMiddleware func(http.Handler) http.Handler,
	authHandler *auth.Handler,
	projectHandler *project.Handler,
	deploymentHandler *deployment.Handler,
	envHandler *envvar.Handler,
	quotaHandler *quota.Handler,
	userHandler *user.Handler,
	webhookHandler *webhook.Handler,
) {
	r.Get("/health", handleHealth)
	r.Get("/ready", handleReady(pool))
	r.Post("/webhook/{projectId}", webhookHandler.GitHub)

	r.Route("/auth", func(r chi.Router) {
		r.Get("/github/login", authHandler.Login)
		r.Get("/github/callback", authHandler.Callback)
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)
			r.Get("/me", authHandler.Me)
			r.Post("/refresh", authHandler.Refresh)
			r.Post("/logout", authHandler.Logout)
		})
	})

	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		r.Get("/me/quota", quotaHandler.Me)
		r.Route("/projects", func(r chi.Router) {
			r.Get("/", projectHandler.List)
			r.Post("/", projectHandler.Create)
			r.Get("/{id}", projectHandler.Get)
			r.Patch("/{id}", projectHandler.Update)
			r.Delete("/{id}", projectHandler.Delete)
			r.Post("/{id}/deploy", deploymentHandler.Trigger)
			r.Post("/{id}/start", deploymentHandler.Start)
			r.Post("/{id}/stop", deploymentHandler.Stop)
			r.Post("/{id}/restart", deploymentHandler.Restart)
			r.Post("/{id}/webhook-secret/regenerate", projectHandler.RegenerateWebhookSecret)
			r.Get("/{id}/deployments", deploymentHandler.List)
			r.Get("/{id}/env", envHandler.List)
			r.Put("/{id}/env", envHandler.BulkUpdate)
			r.Delete("/{id}/env/{key}", envHandler.Delete)
			r.Get("/{id}/stream", deploymentHandler.Stream)
			r.Get("/{id}/logs", deploymentHandler.Logs)
			r.Get("/{id}/metrics", deploymentHandler.Metrics)
		})
		r.Route("/deployments", func(r chi.Router) {
			r.Get("/{id}", deploymentHandler.Get)
			r.Post("/{id}/rollback", deploymentHandler.Rollback)
		})
		r.Route("/admin", func(r chi.Router) {
			r.Use(auth.RequireOwner)
			r.Get("/users", userHandler.List)
			r.Post("/users", userHandler.Add)
			r.Delete("/users/{id}", userHandler.Remove)
		})
	})
}

func corsMiddleware(cfg *config.Config) func(http.Handler) http.Handler {
	allowed := allowedOrigins(cfg)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := strings.TrimRight(r.Header.Get("Origin"), "/")
			if origin != "" {
				if _, ok := allowed[origin]; !ok {
					if r.Method == http.MethodOptions {
						http.Error(w, "CORS origin is not allowed", http.StatusForbidden)
						return
					}
				} else {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Credentials", "true")
					w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
					w.Header().Add("Vary", "Origin")
				}
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func allowedOrigins(cfg *config.Config) map[string]struct{} {
	origins := make(map[string]struct{})
	addOrigin := func(origin string) {
		origin = strings.TrimRight(strings.TrimSpace(origin), "/")
		if origin != "" {
			origins[origin] = struct{}{}
		}
	}
	addOrigin(cfg.FrontendURL)
	if cfg.PublicDomain != "" && cfg.PublicDomain != "localhost" {
		addOrigin("https://" + cfg.PublicDomain)
		addOrigin("https://dashboard." + cfg.PublicDomain)
	}
	if cfg.IsDevelopment() {
		addOrigin("http://localhost:3000")
		addOrigin("http://localhost:5173")
	}
	return origins
}

func timeoutExceptStreams(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		wrapped := middleware.Timeout(timeout)(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, "/stream") {
				next.ServeHTTP(w, r)
				return
			}
			wrapped.ServeHTTP(w, r)
		})
	}
}

func handleMetrics(cfg *config.Config, startedAt time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !metricsAuthConfigured(cfg) && !cfg.IsDevelopment() {
			http.Error(w, "metrics basic auth is not configured", http.StatusServiceUnavailable)
			return
		}
		if metricsAuthConfigured(cfg) && !metricsAuthOK(cfg, r) {
			w.Header().Set("WWW-Authenticate", `Basic realm="mypaas metrics"`)
			http.Error(w, "metrics authentication required", http.StatusUnauthorized)
			return
		}

		var mem runtime.MemStats
		runtime.ReadMemStats(&mem)

		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		uptime := time.Since(startedAt).Seconds()
		_, _ = fmt.Fprintf(w, "# HELP mypaas_api_up Whether the MyPaas API process is serving requests.\n")
		_, _ = fmt.Fprintf(w, "# TYPE mypaas_api_up gauge\nmypaas_api_up 1\n")
		_, _ = fmt.Fprintf(w, "# HELP mypaas_api_uptime_seconds Seconds since the API process started.\n")
		_, _ = fmt.Fprintf(w, "# TYPE mypaas_api_uptime_seconds counter\nmypaas_api_uptime_seconds %.0f\n", uptime)
		_, _ = fmt.Fprintf(w, "# HELP mypaas_go_goroutines Current goroutine count.\n")
		_, _ = fmt.Fprintf(w, "# TYPE mypaas_go_goroutines gauge\nmypaas_go_goroutines %d\n", runtime.NumGoroutine())
		_, _ = fmt.Fprintf(w, "# HELP mypaas_go_heap_alloc_bytes Current heap allocation in bytes.\n")
		_, _ = fmt.Fprintf(w, "# TYPE mypaas_go_heap_alloc_bytes gauge\nmypaas_go_heap_alloc_bytes %d\n", mem.HeapAlloc)
	}
}

func metricsAuthConfigured(cfg *config.Config) bool {
	return cfg.MetricsUsername != "" && cfg.MetricsPassword != ""
}

func metricsAuthOK(cfg *config.Config, r *http.Request) bool {
	user, pass, ok := r.BasicAuth()
	if !ok {
		return false
	}
	userOK := subtle.ConstantTimeCompare([]byte(user), []byte(cfg.MetricsUsername)) == 1
	passOK := subtle.ConstantTimeCompare([]byte(pass), []byte(cfg.MetricsPassword)) == 1
	return userOK && passOK
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func handleReady(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := pool.Ping(r.Context()); err != nil {
			slog.Warn("readiness check failed", "error", err)
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"status":"not_ready","reason":"database unreachable"}`))
			return
		}
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	}
}

func seedOwner(ctx context.Context, queries *db.Queries, email string) error {
	if email == "" {
		return nil
	}
	if _, err := queries.GetUserByEmail(ctx, email); err == nil {
		return nil
	} else if err != pgx.ErrNoRows {
		return err
	}
	_, err := queries.CreateUser(ctx, db.CreateUserParams{
		Email: email,
		Role:  "owner",
	})
	return err
}

func recoverInterruptedDeployments(ctx context.Context, queries *db.Queries) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	msg := "deployment interrupted by API restart"
	if err := queries.FailInterruptedDeployments(ctx, &msg); err != nil {
		return err
	}
	if err := queries.ResetBuildingProjects(ctx); err != nil {
		return err
	}
	return nil
}
