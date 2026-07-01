package envvar

import (
	"time"

	"mypaas/internal/db"
)

type Response struct {
	ID        string `json:"id"`
	ProjectID string `json:"projectId"`
	Key       string `json:"key"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

func ResponseFromDB(row db.EnvVar) Response {
	return Response{
		ID:        row.ID.String(),
		ProjectID: row.ProjectID.String(),
		Key:       row.Key,
		CreatedAt: formatTimestamp(row.CreatedAt.Time, row.CreatedAt.Valid),
		UpdatedAt: formatTimestamp(row.UpdatedAt.Time, row.UpdatedAt.Valid),
	}
}

func ResponsesFromDB(rows []db.EnvVar) []Response {
	out := make([]Response, 0, len(rows))
	for _, row := range rows {
		out = append(out, ResponseFromDB(row))
	}
	return out
}

func formatTimestamp(t time.Time, valid bool) string {
	if !valid {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
