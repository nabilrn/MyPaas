package audit

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"mypaas/internal/auth"
	"mypaas/internal/db"
)

type Service struct {
	queries *db.Queries
}

type Entry struct {
	UserID       uuid.UUID
	Action       string
	ResourceType *string
	ResourceID   uuid.UUID
	Metadata     map[string]any
	IPAddress    *netip.Addr
	UserAgent    *string
}

func NewService(queries *db.Queries) *Service {
	return &Service{queries: queries}
}

func (s *Service) Log(ctx context.Context, entry Entry) error {
	metadata, err := json.Marshal(entry.Metadata)
	if err != nil {
		metadata = []byte(`{}`)
	}
	var resourceID pgtype.UUID
	if entry.ResourceID != uuid.Nil {
		resourceID = pgtype.UUID{Bytes: entry.ResourceID, Valid: true}
	}
	return s.queries.CreateAuditLog(ctx, db.CreateAuditLogParams{
		UserID:       pgtype.UUID{Bytes: entry.UserID, Valid: entry.UserID != uuid.Nil},
		Action:       entry.Action,
		ResourceType: entry.ResourceType,
		ResourceID:   resourceID,
		Metadata:     metadata,
		IpAddress:    entry.IPAddress,
		UserAgent:    entry.UserAgent,
	})
}

func (s *Service) List(ctx context.Context, limit, offset int32) ([]db.AuditLog, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return s.queries.ListAuditLogs(ctx, db.ListAuditLogsParams{Limit: limit, Offset: offset})
}

func Middleware(service *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !shouldAudit(r) {
				next.ServeHTTP(w, r)
				return
			}

			recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(recorder, r)

			user, err := auth.CurrentUser(r)
			if err != nil {
				return
			}
			action, resourceType, resourceID := classify(r)
			if action == "" {
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if err := service.Log(ctx, Entry{
				UserID:       user.ID,
				Action:       action,
				ResourceType: resourceType,
				ResourceID:   resourceID,
				Metadata: map[string]any{
					"method": r.Method,
					"path":   r.URL.Path,
					"status": recorder.status,
				},
				IPAddress: remoteIP(r),
				UserAgent: stringPtr(r.UserAgent()),
			}); err != nil {
				slog.Warn("write audit log", "error", err, "action", action)
			}
		})
	}
}

func shouldAudit(r *http.Request) bool {
	switch r.Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func classify(r *http.Request) (string, *string, uuid.UUID) {
	path := r.URL.Path
	id := firstUUID(chi.URLParam(r, "id"), chi.URLParam(r, "projectId"))

	switch {
	case r.Method == http.MethodPost && strings.HasSuffix(path, "/deploy"):
		return "deployment.triggered", stringPtr("project"), id
	case r.Method == http.MethodPost && strings.HasSuffix(path, "/start"):
		return "project.started", stringPtr("project"), id
	case r.Method == http.MethodPost && strings.HasSuffix(path, "/stop"):
		return "project.stopped", stringPtr("project"), id
	case r.Method == http.MethodPost && strings.HasSuffix(path, "/restart"):
		return "project.restarted", stringPtr("project"), id
	case r.Method == http.MethodPost && strings.HasSuffix(path, "/rollback"):
		return "deployment.rollback", stringPtr("deployment"), id
	case r.Method == http.MethodPost && strings.HasSuffix(path, "/webhook-secret/regenerate"):
		return "project.webhook_secret_regenerated", stringPtr("project"), id
	case r.Method == http.MethodPost && strings.HasSuffix(path, "/projects"):
		return "project.created", stringPtr("project"), uuid.Nil
	case r.Method == http.MethodPatch && strings.Contains(path, "/projects/"):
		return "project.updated", stringPtr("project"), id
	case r.Method == http.MethodDelete && strings.Contains(path, "/projects/"):
		return "project.deleted", stringPtr("project"), id
	case r.Method == http.MethodPut && strings.HasSuffix(path, "/env"):
		return "project.env_updated", stringPtr("project"), id
	case r.Method == http.MethodDelete && strings.Contains(path, "/env/"):
		return "project.env_deleted", stringPtr("project"), id
	case strings.Contains(path, "/projects/") && strings.Contains(path, "/db/write-session"):
		return "project.dbstudio_write_session", stringPtr("project"), id
	case strings.Contains(path, "/projects/") && strings.Contains(path, "/db/rows"):
		return "project.dbstudio_rows_mutated", stringPtr("project"), id
	case r.Method == http.MethodPost && strings.HasSuffix(path, "/admin/users"):
		return "admin.user_added", stringPtr("user"), uuid.Nil
	case r.Method == http.MethodDelete && strings.Contains(path, "/admin/users/"):
		return "admin.user_removed", stringPtr("user"), id
	default:
		return strings.ToLower(r.Method) + "." + strings.Trim(strings.ReplaceAll(path, "/", "."), "."), nil, id
	}
}

func firstUUID(values ...string) uuid.UUID {
	for _, value := range values {
		if id, err := uuid.Parse(value); err == nil {
			return id
		}
	}
	return uuid.Nil
}

func remoteIP(r *http.Request) *netip.Addr {
	host := r.RemoteAddr
	if value := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); value != "" {
		host = strings.TrimSpace(strings.Split(value, ",")[0])
	} else if parsed, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		host = parsed
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return nil
	}
	return &addr
}

func stringPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
