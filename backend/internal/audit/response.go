package audit

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"mypaas/internal/db"
)

type Response struct {
	ID           string         `json:"id"`
	UserID       *string        `json:"userId"`
	Action       string         `json:"action"`
	ResourceType *string        `json:"resourceType"`
	ResourceID   *string        `json:"resourceId"`
	Metadata     map[string]any `json:"metadata"`
	IPAddress    *string        `json:"ipAddress"`
	UserAgent    *string        `json:"userAgent"`
	CreatedAt    string         `json:"createdAt"`
}

func ResponseFromDB(row db.AuditLog) Response {
	metadata := map[string]any{}
	if len(row.Metadata) > 0 {
		_ = json.Unmarshal(row.Metadata, &metadata)
	}
	var ipAddress *string
	if row.IpAddress != nil {
		formatted := row.IpAddress.String()
		ipAddress = &formatted
	}
	return Response{
		ID:           row.ID.String(),
		UserID:       uuidString(row.UserID),
		Action:       row.Action,
		ResourceType: row.ResourceType,
		ResourceID:   uuidString(row.ResourceID),
		Metadata:     metadata,
		IPAddress:    ipAddress,
		UserAgent:    row.UserAgent,
		CreatedAt:    formatTimestamp(row.CreatedAt.Time, row.CreatedAt.Valid),
	}
}

func ResponsesFromDB(rows []db.AuditLog) []Response {
	out := make([]Response, 0, len(rows))
	for _, row := range rows {
		out = append(out, ResponseFromDB(row))
	}
	return out
}

func uuidString(value pgtype.UUID) *string {
	if !value.Valid {
		return nil
	}
	id := uuid.UUID(value.Bytes)
	formatted := id.String()
	return &formatted
}

func formatTimestamp(t time.Time, valid bool) string {
	if !valid {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
