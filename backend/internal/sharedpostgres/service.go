package sharedpostgres

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"mypaas/internal/config"
	"mypaas/internal/db"
	"mypaas/internal/envvar"
	"mypaas/internal/errs"
)

type Service struct {
	pool *pgxpool.Pool
	cfg  *config.Config
	envs *envvar.Service
}

func NewService(pool *pgxpool.Pool, cfg *config.Config, envs *envvar.Service) *Service {
	return &Service{pool: pool, cfg: cfg, envs: envs}
}

func (s *Service) Provision(ctx context.Context, project db.Project) error {
	if !s.cfg.SharedPostgresEnabled {
		return fmt.Errorf("%w: shared PostgreSQL provisioning is disabled", errs.ErrValidation)
	}

	name := databaseName(project.ID)
	role := roleName(project.ID)
	password, err := randomPassword()
	if err != nil {
		return fmt.Errorf("generate database password: %w", err)
	}

	if err := s.ensureRole(ctx, role, password); err != nil {
		return err
	}
	if err := s.ensureDatabase(ctx, name, role); err != nil {
		_ = s.Cleanup(context.Background(), project.ID)
		return err
	}

	if err := s.envs.BulkUpdate(ctx, project.ID, []envvar.Value{{
		Key:   "DATABASE_URL",
		Value: s.databaseURL(name, role, password),
	}}); err != nil {
		_ = s.Cleanup(context.Background(), project.ID)
		return err
	}
	return nil
}

func (s *Service) Cleanup(ctx context.Context, projectID uuid.UUID) error {
	if !s.cfg.SharedPostgresEnabled {
		return nil
	}
	name := databaseName(projectID)
	role := roleName(projectID)
	quotedName := pgx.Identifier{name}.Sanitize()
	quotedRole := pgx.Identifier{role}.Sanitize()

	if _, err := s.pool.Exec(ctx, `
SELECT pg_terminate_backend(pid)
FROM pg_stat_activity
WHERE datname = $1
  AND pid <> pg_backend_pid()
`, name); err != nil {
		return fmt.Errorf("terminate shared postgres connections: %w", err)
	}
	if _, err := s.pool.Exec(ctx, "DROP DATABASE IF EXISTS "+quotedName); err != nil {
		return fmt.Errorf("drop shared postgres database: %w", err)
	}
	if _, err := s.pool.Exec(ctx, "DROP ROLE IF EXISTS "+quotedRole); err != nil {
		return fmt.Errorf("drop shared postgres role: %w", err)
	}
	return nil
}

func (s *Service) ensureRole(ctx context.Context, role, password string) error {
	var exists bool
	if err := s.pool.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)", role).Scan(&exists); err != nil {
		return fmt.Errorf("check shared postgres role: %w", err)
	}
	quotedRole := pgx.Identifier{role}.Sanitize()
	if exists {
		if _, err := s.pool.Exec(ctx, "ALTER ROLE "+quotedRole+" LOGIN PASSWORD "+quoteLiteral(password)); err != nil {
			return fmt.Errorf("alter shared postgres role: %w", err)
		}
		return nil
	}
	if _, err := s.pool.Exec(ctx, "CREATE ROLE "+quotedRole+" LOGIN PASSWORD "+quoteLiteral(password)); err != nil {
		return fmt.Errorf("create shared postgres role: %w", err)
	}
	return nil
}

func (s *Service) ensureDatabase(ctx context.Context, name, owner string) error {
	var exists bool
	if err := s.pool.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", name).Scan(&exists); err != nil {
		return fmt.Errorf("check shared postgres database: %w", err)
	}
	if exists {
		return nil
	}
	quotedName := pgx.Identifier{name}.Sanitize()
	quotedOwner := pgx.Identifier{owner}.Sanitize()
	if _, err := s.pool.Exec(ctx, "CREATE DATABASE "+quotedName+" OWNER "+quotedOwner); err != nil {
		return fmt.Errorf("create shared postgres database: %w", err)
	}
	return nil
}

func (s *Service) databaseURL(name, user, password string) string {
	host := strings.TrimSpace(s.cfg.SharedPostgresHost)
	if host == "" {
		host = "host.docker.internal"
	}
	port := s.cfg.SharedPostgresPort
	if port <= 0 {
		port = 5432
	}
	sslMode := strings.TrimSpace(s.cfg.SharedPostgresSSLMode)
	if sslMode == "" {
		sslMode = "disable"
	}
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, password),
		Host:   net.JoinHostPort(host, fmt.Sprint(port)),
		Path:   "/" + name,
	}
	query := u.Query()
	query.Set("sslmode", sslMode)
	u.RawQuery = query.Encode()
	return u.String()
}

func databaseName(projectID uuid.UUID) string {
	return "mypaas_p_" + strings.ReplaceAll(projectID.String(), "-", "")
}

func roleName(projectID uuid.UUID) string {
	return databaseName(projectID) + "_user"
}

func randomPassword() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func quoteLiteral(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}
