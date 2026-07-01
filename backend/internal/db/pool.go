package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Connect(ctx context.Context, url string) (*pgxpool.Pool, error) {
	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(connectCtx, url)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	if err := pool.Ping(connectCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}

	return pool, nil
}
