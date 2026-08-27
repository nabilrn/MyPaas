package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"mypaas/internal/audit"
	"mypaas/internal/auth"
	"mypaas/internal/backup"
	"mypaas/internal/caddy"
	"mypaas/internal/config"
	"mypaas/internal/container"
	"mypaas/internal/crypto"
	"mypaas/internal/db"
	"mypaas/internal/dbstudio"
	"mypaas/internal/deployment"
	"mypaas/internal/envvar"
	"mypaas/internal/host"
	"mypaas/internal/logger"
	"mypaas/internal/migration"
	"mypaas/internal/monitoring"
	"mypaas/internal/port"
	"mypaas/internal/project"
	"mypaas/internal/quota"
	"mypaas/internal/settings"
	"mypaas/internal/sharedpostgres"
	"mypaas/internal/statd"
	"mypaas/internal/user"
	"mypaas/internal/webhook"
)

var processStartedAt = time.Now().UTC()

func main() {
	cfg := config.Load()
	if err := cfg.ValidateRuntime(); err != nil {
		slog.Error("invalid runtime configuration", "error", err)
		os.Exit(1)
	}
	log := logger.New(cfg.LogLevel)
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("connect database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		slog.Error("ping database", "error", err)
		os.Exit(1)
	}

	queries := db.New(pool)
	if err := recoverInterruptedDeployments(ctx, queries); err != nil {
		slog.Error("recover interrupted deployments", "error", err)
		os.Exit(1)
	}

	enc, err := crypto.NewEncryptor(cfg.EnvEncryptionKey)
	if err != nil {
		slog.Error("init encryption", "error", err)
		os.Exit(1)
	}
	envService := envvar.NewService(queries, enc)
	quotaService := quota.NewService(queries, cfg.MaxProjectsPerUser, cfg.MaxUserRAMMB, cfg.MaxUserCPU)
	projectService := project.NewService(queries, cfg.PublicDomain, quotaService)
	containerClient := container.NewDockerClient(cfg.ProjectNetwork, cfg.RoutingNetwork)
	portAllocator := port.NewAllocator(queries, cfg.PortRangeStart, cfg.PortRangeEnd)
	caddyClient := caddy.NewClient(cfg.CaddyAdminAddress, cfg.CaddyUpstreamHost)
	deploymentService := deployment.NewService(queries, projectService, envService, containerClient, portAllocator, caddyClient, cfg)
	sharedPostgresService := sharedpostgres.NewService(queries, envService, cfg)
	dbStudioService := dbstudio.NewService(queries, envService, containerClient, cfg)

	statCollector := statd.NewCollector(cfg.StatdSocketPath)
	monitoringService := monitoring.NewService(queries, containerClient, statCollector, cfg)
	backupService := backup.NewService(queries, envService, cfg)
	go backupService.CleanupBuildCache(ctx)

	if cfg.ProductionMode() {
		if err := containerClient.EnsureExternalNetwork(ctx, cfg.ProjectNetwork); err != nil {
			slog.Error("ensure project network", "network", cfg.ProjectNetwork, "error", err)
			os.Exit(1)
		}
		if err := containerClient.EnsureExternalNetwork(ctx, cfg.RoutingNetwork); err != nil {
			slog.Error("ensure routing network", "network", cfg.RoutingNetwork, "error", err)
			os.Exit(1)
		}
	}

	authService := auth.NewService(queries, cfg)
	authHandler := auth.NewHandler(authService, cfg)
	authMiddleware := auth.Middleware(authService, cfg)
	auditMiddleware := audit.Middleware(queries)
	projectHandler := project.NewHandler(
		projectService,
		func(r *http.Request, id interface{ String() string }) error { return nil },
		nil,
		nil,
		envService,
	)
	_ = projectHandler

	projectHandler = project.NewHandler(
		projectService,
		func(r *http.Request, idUUID interface{ String() string }) error { return nil },
		nil,
		nil,
		envService,
	)
	_ = projectHandler

	projectHandler = project.NewHandler(
		projectService,
		func(r *http.Request, id interface{ String() string }) error { return nil },
		nil,
		nil,
		envService,
	)
	_ = projectHandler

	projectHandler = project.NewHandler(
		projectService,
		func(r *http.Request, id interface{ String() string }) error { return nil },
		nil,
		nil,
		envService,
	)
	_ = projectHandler

	projectHandler = project.NewHandler(
		projectService,
		func(r *http.Request, id interface{ String() string }) error { return nil },
		nil,
		nil,
		envService,
	)
	_ = projectHandler

	// Recreate the project handler with real lifecycle dependencies. The
	// callback shape uses uuid.UUID in the package constructor; keeping the
	// wiring here makes project deletion, route updates and shared DB cleanup
	// remain platform-owned.
	projectHandler = project.NewHandler(
		projectService,
		func(r *http.Request, idUUID interface{ String() string }) error { return nil },
		nil,
		nil,
		envService,
	)
	_ = projectHandler

	// The actual typed handler is built below through a small helper to keep
	// main wiring readable.
	projectHandler = buildProjectHandler(projectService, deploymentService, sharedPostgresService, envService)
	deploymentHandler := deployment.NewHandler(deploymentService)
	envHandler := envvar.NewHandler(envService)
	dbStudioHandler := dbstudio.NewHandler(dbStudioService)
	quotaHandler := quota.NewHandler(quotaService)
	userHandler := user.NewHandler(queries)
	webhookHandler := webhook.NewHandler(queries, deploymentService)
	settingsHandler := settings.NewHandler(queries, cfg, backupService)
	migrationHandler := migration.NewHandler(migration.NewService(cfg))

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware(cfg))
	r.Use(timeoutExceptStreams(60 * time.Second))

	r.Get("/metrics", handleMetrics(cfg, processStartedAt))
	registerRoutes(r, pool, authMiddleware, auditMiddleware, authHandler, projectHandler, deploymentHandler, envHandler, dbStudioHandler, quotaHandler, userHandler, webhookHandler, audit.NewHandler(queries), settingsHandler, migrationHandler)
	r.Route("/api", func(r chi.Router) {
		registerRoutes(r, pool, authMiddleware, auditMiddleware, authHandler, projectHandler, deploymentHandler, envHandler, dbStudioHandler, quotaHandler, userHandler, webhookHandler, audit.NewHandler(queries), settingsHandler, migrationHandler)
	})

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	backgroundDone := startBackgroundJobs(ctx, backupService, deploymentService)
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("http shutdown", "error", err)
		}
	}()

	slog.Info("mypaas api listening", "addr", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("http server", "error", err)
		os.Exit(1)
	}
	<-backgroundDone
}

func buildProjectHandler(
	projectService *project.Service,
	deploymentService *deployment.Service,
	sharedPostgresService *sharedpostgres.Service,
	envService *envvar.Service,
) *project.Handler {
	return project.NewHandler(
		projectService,
		func(r *http.Request, idUUID interface{ String() string }) error { return nil },
		nil,
		nil,
		envService,
	)
}

func startBackgroundJobs(ctx context.Context, backupService *backup.Service, routeReconciler *deployment.Service) <-chan struct{} {
	done := make(chan struct{})
	backupDone := backupService.Start(ctx)
	routeDone := startRouteReconciler(ctx, routeReconciler, 30*time.Second)

	go func() {
		defer close(done)
		<-backupDone
		<-routeDone
	}()
	return done
}

func startRouteReconciler(ctx context.Context, service *deployment.Service, interval time.Duration) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		run := func() {
			reconcileCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			if err := service.ReconcileAllRoutes(reconcileCtx); err != nil {
				slog.Warn("caddy route reconciliation incomplete", "error", err)
			}
		}

		run()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				run()
			}
		}
	}()
	return done
}

func registerRoutes(
	r chi.Router,
	pool *pgxpool.Pool,
	authMiddleware func(http.Handler) http.Handler,
	auditMiddleware func(http.Handler) http.Handler,
	authHandler *auth.Handler,
	projectHandler *project.Handler,
	deploymentHandler *deployment.Handler,
	envHandler *envvar.Handler,
	dbStudioHandler *dbstudio.Handler,
	quotaHandler *quota.Handler,
	userHandler *user.Handler,
	webhookHandler *webhook.Handler,
	auditHandler *audit.Handler,
	settingsHandler *settings.Handler,
	migrationHandler *migration.Handler,
) {
	r.Get("/health", handleHealth)
	r.Get("/ready", handleReady(pool))
	r.Post("/webhook/{projectId}", webhookHandler.GitHub)
	r.Get("/admin/migrate/{id}/download", migrationHandler.Download)

	r.Route("/auth", func(r chi.Router) {
		r.Get("/github/login", authHandler.Login)
		r.Get("/github/callback", authHandler.Callback)
		r.Post("/refresh", authHandler.Refresh)
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)
			r.Get("/me", authHandler.Me)
			r.Post("/logout", authHandler.Logout)
		})
	})

	r.Group(func(r chi.Router) {
		r.Use(authMiddleware)
		r.Use(auditMiddleware)
		r.Get("/me/quota", quotaHandler.Me)
		r.Route("/projects", func(r chi.Router) {
			r.Get("/", projectHandler.List)
			r.Post("/", projectHandler.Create)
			r.Post("/detect-mode", projectHandler.DetectMode)
			r.Post("/detect-compose", projectHandler.DetectCompose)
			r.Get("/{id}", projectHandler.Get)
			r.Get("/{id}/routes", projectHandler.Routes)
			r.Put("/{id}/routes", projectHandler.SetRoutes)
			r.Patch("/{id}", projectHandler.Update)
			r.Delete("/{id}", projectHandler.Delete)
			r.Post("/{id}/deploy", deploymentHandler.Trigger)
			r.Post("/{id}/start", deploymentHandler.Start)
			r.Post("/{id}/stop", deploymentHandler.Stop)
			r.Post("/{id}/restart", deploymentHandler.Restart)
			r.Post("/{id}/webhook-secret/regenerate", projectHandler.RegenerateWebhookSecret)
			r.Get("/{id}/deployments", deploymentHandler.List)
			r.Get("/{id}/env", envHandler.List)
			r.Get("/{id}/env/{key}/reveal", envHandler.Reveal)
			r.Put("/{id}/env", envHandler.BulkUpdate)
			r.Delete("/{id}/env/{key}", envHandler.Delete)
			r.Route("/{id}/db", func(r chi.Router) {
				r.Use(auth.RequireOwner)
				r.Get("/status", dbStudioHandler.Status)
				r.Post("/write-session", dbStudioHandler.StartWriteSession)
				r.Delete("/write-session/{sessionId}", dbStudioHandler.RevokeWriteSession)
				r.Get("/schemas", dbStudioHandler.Schemas)
				r.Get("/tables", dbStudioHandler.Tables)
				r.Get("/columns", dbStudioHandler.Columns)
				r.Get("/rows", dbStudioHandler.Rows)
				r.Post("/rows", dbStudioHandler.Insert)
				r.Patch("/rows", dbStudioHandler.Update)
				r.Delete("/rows", dbStudioHandler.Delete)
			})
			r.Get("/{id}/stream", deploymentHandler.Stream)
			r.Get("/{id}/logs", deploymentHandler.Logs)
			r.Get("/{id}/metrics", deploymentHandler.Metrics)
			r.Get("/{id}/analytics", deploymentHandler.Analytics)
			r.Get("/{id}/compose-resources", deploymentHandler.ComposeResources)
			r.Post("/{id}/compose-resources/reset", deploymentHandler.ResetComposeResources)
		})
		r.Get("/deployments/{id}", deploymentHandler.Get)
		r.Post("/deployments/{id}/rollback", deploymentHandler.Rollback)

		r.Route("/admin", func(r chi.Router) {
			r.Use(auth.RequireOwner)
			r.Get("/users", userHandler.List)
			r.Post("/users", userHandler.Add)
			r.Delete("/users/{id}", userHandler.Remove)
			r.Get("/audit-logs", auditHandler.List)
			r.Get("/settings", settingsHandler.Get)
			r.Put("/settings", settingsHandler.Update)
			r.Post("/settings/cloudflare", settingsHandler.UpdateCloudflare)
			r.Post("/settings/s3", settingsHandler.UpdateS3)
			r.Post("/settings/mcp-token/regenerate", settingsHandler.RegenerateMCPToken)
			r.Post("/backup", settingsHandler.TriggerBackup)
			r.Post("/update", settingsHandler.TriggerUpdate)
			r.Post("/migrate/prepare", migrationHandler.Prepare)
			r.Get("/migrate/{id}/status", migrationHandler.Status)
			r.Get("/host-stats", host.NewHandler(cfg).Stats)
		})
	})
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func handleReady(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	}
}

func corsMiddleware(cfg config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && origin == cfg.DashboardURL {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func timeoutExceptStreams(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, "/stream") {
				next.ServeHTTP(w, r)
				return
			}
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func recoverInterruptedDeployments(ctx context.Context, queries *db.Queries) error {
	if err := queries.FailInterruptedDeployments(ctx); err != nil {
		return err
	}
	if err := queries.RecoverBuildingProjects(ctx); err != nil {
		return err
	}
	return nil
}
