package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"mypaas/internal/mcpserver"
)

func main() {
	apiURL := strings.TrimRight(strings.TrimSpace(os.Getenv("MYPAAS_INTERNAL_API_URL")), "/")
	if apiURL == "" {
		apiURL = strings.TrimRight(strings.TrimSpace(os.Getenv("MYPAAS_URL")), "/")
	}
	if apiURL == "" {
		apiURL = "http://localhost:8080/api"
	}
	listenAddr := strings.TrimSpace(os.Getenv("MYPAAS_MCP_LISTEN_ADDR"))
	if listenAddr == "" {
		listenAddr = ":8081"
	}

	mcpHandler := mcpserver.NewHandler(apiURL)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.Handle("/mcp", mcpHandler)

	srv := &http.Server{
		Addr:              listenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("remote MCP server started", "addr", listenAddr, "api", apiURL)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)
	select {
	case err := <-errCh:
		slog.Error("remote MCP server failed", "error", err)
		os.Exit(1)
	case sig := <-signals:
		slog.Info("remote MCP server shutting down", "signal", sig)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("remote MCP shutdown failed", "error", err)
		os.Exit(1)
	}
}
