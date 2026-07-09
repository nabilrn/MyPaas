package dbstudio

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"mypaas/internal/audit"
	"mypaas/internal/db"
	"mypaas/internal/envvar"
	"mypaas/internal/errs"
)

type Service struct {
	queries *db.Queries
	envs    *envvar.Service
	audit   *audit.Service
}

func NewService(queries *db.Queries, envs *envvar.Service, auditService *audit.Service) *Service {
	return &Service{queries: queries, envs: envs, audit: auditService}
}

func (s *Service) Status(ctx context.Context, projectID, userID uuid.UUID) (Status, error) {
	conn, err := s.resolve(ctx, projectID)
	if err != nil {
		if errors.Is(err, errs.ErrValidation) {
			message := trimDomainPrefix(err)
			return Status{Configured: !strings.Contains(message, "no supported database"), Message: message}, nil
		}
		return Status{}, err
	}
	session := s.activeSession(ctx, projectID, userID)
	if err := s.withConnection(ctx, conn, func(_ *dbHandle) error { return nil }); err != nil {
		return Status{Configured: true, Connected: false, Message: err.Error(), Connection: &conn, WriteAccess: session}, nil
	}
	return Status{Configured: true, Connected: true, Message: "Connected", Connection: &conn, WriteAccess: session}, nil
}

func (s *Service) Schemas(ctx context.Context, projectID uuid.UUID) ([]Schema, error) {
	return withResult(ctx, s, projectID, func(handle *dbHandle) ([]Schema, error) {
		return handle.adapter.Schemas(ctx, handle.conn)
	})
}

func (s *Service) Tables(ctx context.Context, projectID uuid.UUID, schema string) ([]Table, error) {
	return withResult(ctx, s, projectID, func(handle *dbHandle) ([]Table, error) {
		return handle.adapter.Tables(ctx, handle.conn, schema)
	})
}

func (s *Service) Columns(ctx context.Context, projectID uuid.UUID, schema, table string) ([]Column, error) {
	return withResult(ctx, s, projectID, func(handle *dbHandle) ([]Column, error) {
		return handle.adapter.Columns(ctx, handle.conn, schema, table)
	})
}

func (s *Service) Rows(ctx context.Context, projectID uuid.UUID, query RowQuery) (RowPage, error) {
	return withResult(ctx, s, projectID, func(handle *dbHandle) (RowPage, error) {
		return handle.adapter.Rows(ctx, handle.conn, query)
	})
}

func (s *Service) StartWriteSession(ctx context.Context, projectID, userID uuid.UUID, ttl time.Duration) (WriteSession, error) {
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	if ttl > 4*time.Hour {
		ttl = 4 * time.Hour
	}
	if _, err := s.queries.GetProjectByID(ctx, projectID); err != nil {
		return WriteSession{}, projectErr(err)
	}
	expiresAt := time.Now().Add(ttl).UTC()
	row, err := s.queries.CreateDBStudioSession(ctx, db.CreateDBStudioSessionParams{
		ProjectID: projectID,
		UserID:    userID,
		ExpiresAt: pgtype.Timestamp{Time: expiresAt, Valid: true},
	})
	if err != nil {
		return WriteSession{}, err
	}
	return sessionResponse(row), nil
}

func (s *Service) RevokeWriteSession(ctx context.Context, projectID, userID, sessionID uuid.UUID) error {
	return s.queries.RevokeDBStudioSession(ctx, db.RevokeDBStudioSessionParams{
		ID: sessionID, ProjectID: projectID, UserID: userID,
	})
}

func (s *Service) Insert(ctx context.Context, projectID, userID uuid.UUID, mutation Mutation) error {
	if err := s.requireWrite(ctx, projectID, userID); err != nil {
		return err
	}
	conn, err := s.resolve(ctx, projectID)
	if err != nil {
		return err
	}
	err = s.withConnection(ctx, conn, func(handle *dbHandle) error {
		return handle.adapter.Insert(ctx, handle.conn, mutation)
	})
	return s.auditMutation(ctx, userID, projectID, "dbstudio.row_inserted", mutation, err)
}

func (s *Service) Update(ctx context.Context, projectID, userID uuid.UUID, mutation Mutation) error {
	if err := s.requireWrite(ctx, projectID, userID); err != nil {
		return err
	}
	conn, err := s.resolve(ctx, projectID)
	if err != nil {
		return err
	}
	err = s.withConnection(ctx, conn, func(handle *dbHandle) error {
		return handle.adapter.Update(ctx, handle.conn, mutation)
	})
	return s.auditMutation(ctx, userID, projectID, "dbstudio.row_updated", mutation, err)
}

func (s *Service) Delete(ctx context.Context, projectID, userID uuid.UUID, mutation Mutation) error {
	if err := s.requireWrite(ctx, projectID, userID); err != nil {
		return err
	}
	conn, err := s.resolve(ctx, projectID)
	if err != nil {
		return err
	}
	err = s.withConnection(ctx, conn, func(handle *dbHandle) error {
		return handle.adapter.Delete(ctx, handle.conn, mutation)
	})
	return s.auditMutation(ctx, userID, projectID, "dbstudio.row_deleted", mutation, err)
}

func (s *Service) resolve(ctx context.Context, projectID uuid.UUID) (Connection, error) {
	project, err := s.queries.GetProjectByID(ctx, projectID)
	if err != nil {
		return Connection{}, projectErr(err)
	}
	envs, err := s.envs.DecryptedMap(ctx, projectID)
	if err != nil {
		return Connection{}, err
	}
	return resolveConnection(ctx, project, envs)
}

type dbHandle struct {
	conn    *sql.DB
	adapter Adapter
}

func (s *Service) withConnection(ctx context.Context, conn Connection, fn func(*dbHandle) error) error {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	sqlDB, adapter, err := openConnection(ctx, conn)
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	return fn(&dbHandle{conn: sqlDB, adapter: adapter})
}

func withResult[T any](ctx context.Context, s *Service, projectID uuid.UUID, fn func(*dbHandle) (T, error)) (T, error) {
	var zero T
	conn, err := s.resolve(ctx, projectID)
	if err != nil {
		return zero, err
	}
	var out T
	err = s.withConnection(ctx, conn, func(handle *dbHandle) error {
		var err error
		out, err = fn(handle)
		return err
	})
	return out, err
}

func (s *Service) requireWrite(ctx context.Context, projectID, userID uuid.UUID) error {
	_, err := s.queries.GetActiveDBStudioSession(ctx, db.GetActiveDBStudioSessionParams{ProjectID: projectID, UserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%w: enable write mode before changing database rows", errs.ErrForbidden)
	}
	return err
}

func (s *Service) activeSession(ctx context.Context, projectID, userID uuid.UUID) *WriteSession {
	row, err := s.queries.GetActiveDBStudioSession(ctx, db.GetActiveDBStudioSessionParams{ProjectID: projectID, UserID: userID})
	if err != nil {
		return nil
	}
	session := sessionResponse(row)
	return &session
}

func (s *Service) auditMutation(ctx context.Context, userID, projectID uuid.UUID, action string, mutation Mutation, err error) error {
	if err != nil || s.audit == nil {
		return err
	}
	_ = s.audit.Log(context.Background(), audit.Entry{
		UserID: userID, Action: action, ResourceType: stringPtr("project"), ResourceID: projectID,
		Metadata: map[string]any{"schema": mutation.Schema, "table": mutation.Table},
	})
	return nil
}

func projectErr(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return errs.ErrNotFound
	}
	return err
}

func sessionResponse(row db.DbStudioSession) WriteSession {
	return WriteSession{ID: row.ID.String(), ExpiresAt: row.ExpiresAt.Time}
}

func trimDomainPrefix(err error) string {
	return strings.TrimPrefix(err.Error(), errs.ErrValidation.Error()+": ")
}

func stringPtr(value string) *string {
	return &value
}
