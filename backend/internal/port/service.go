package port

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"mypaas/internal/db"
	"mypaas/internal/errs"
)

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

func (s *Service) Allocate(ctx context.Context, projectID uuid.UUID) (int32, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("begin port allocation: %w", err)
	}
	defer tx.Rollback(ctx)

	queries := db.New(tx)
	port, err := queries.AcquireAvailablePort(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, errs.ErrPortPoolExhausted
		}
		return 0, err
	}
	if !canBind(port) {
		return 0, fmt.Errorf("%w: selected port %d is already bound outside registry", errs.ErrPortPoolExhausted, port)
	}

	if err := queries.SetPortInUse(ctx, db.SetPortInUseParams{
		Port:      port,
		ProjectID: pgUUID(projectID),
	}); err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit port allocation: %w", err)
	}
	return port, nil
}

func (s *Service) Release(ctx context.Context, projectID uuid.UUID) error {
	queries := db.New(s.pool)
	return queries.ReleasePortByProject(ctx, pgUUID(projectID))
}

func (s *Service) ReleasePort(ctx context.Context, port int32) error {
	queries := db.New(s.pool)
	return queries.ReleasePort(ctx, port)
}

func pgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func canBind(port int32) bool {
	ln, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(int(port))))
	if err != nil {
		return false
	}
	_ = ln.Close()
	return true
}
