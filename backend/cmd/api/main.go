package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
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
	deploymentService := deployment.NewService(cfg, queries, envService, portService, caddy.NewClient(cfg.CaddyAdmin), container.NewDockerCLI())
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
	r.Use(middleware.Timeout(60 * time.Second))

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
