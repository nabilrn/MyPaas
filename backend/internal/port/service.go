package port

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"

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

var dockerPortBindings = runningDockerPortBindings

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
	host := strings.TrimSpace(os.Getenv("DOCKER_BIND_HOST"))
	if host == "" {
		host = "127.0.0.1"
	}
	if dockerHostPortBound(host, port) {
		return false
	}
	if !isLocalBindHost(host) {
		return true
	}
	ln, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(int(port))))
	if err != nil {
		return false
	}
	_ = ln.Close()
	return true
}

func isLocalBindHost(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	return host == "" || host == "127.0.0.1" || host == "localhost" || host == "0.0.0.0" || host == "::" || host == "::1"
}

func dockerHostPortBound(host string, port int32) bool {
	bindings, err := dockerPortBindings(context.Background())
	if err != nil {
		return false
	}
	wantedPort := strconv.Itoa(int(port))
	wantedHost := strings.TrimSpace(host)
	for _, row := range bindings {
		for _, ports := range row.NetworkSettings.Ports {
			for _, binding := range ports {
				if strings.TrimSpace(binding.HostPort) != wantedPort {
					continue
				}
				boundHost := strings.TrimSpace(binding.HostIP)
				if boundHost == "" || boundHost == "0.0.0.0" || boundHost == "::" || boundHost == wantedHost {
					return true
				}
			}
		}
	}
	return false
}

type dockerInspectPortRow struct {
	NetworkSettings struct {
		Ports map[string][]struct {
			HostIP   string `json:"HostIp"`
			HostPort string `json:"HostPort"`
		} `json:"Ports"`
	} `json:"NetworkSettings"`
}

func runningDockerPortBindings(ctx context.Context) ([]dockerInspectPortRow, error) {
	idsRaw, err := exec.CommandContext(ctx, "docker", "ps", "-q").CombinedOutput()
	if err != nil {
		return nil, err
	}
	ids := strings.Fields(string(idsRaw))
	if len(ids) == 0 {
		return nil, nil
	}
	args := append([]string{"inspect"}, ids...)
	out, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
	if err != nil {
		return nil, err
	}
	var rows []dockerInspectPortRow
	if err := json.Unmarshal(out, &rows); err != nil {
		return nil, err
	}
	return rows, nil
}
