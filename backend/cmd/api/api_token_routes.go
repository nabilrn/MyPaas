package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"mypaas/internal/apitoken"
	"mypaas/internal/db"
	"mypaas/internal/tokenapi"
)

func registerAPITokenRoutes(r chi.Router, pool *pgxpool.Pool) {
	handler := tokenapi.NewHandler(apitoken.NewRepository(db.New(pool)))
	r.Get("/api-tokens", handler.List)
	r.Post("/api-tokens", handler.Create)
	r.Delete("/api-tokens/{id}", handler.Revoke)
}
