package deployment

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"mypaas/internal/db"
)

type Response struct {
	ID                string  `json:"id"`
	ProjectID         string  `json:"projectId"`
	CommitSha         *string `json:"commitSha"`
	CommitMessage     *string `json:"commitMessage"`
	Status            string  `json:"status"`
	BuildLog          *string `json:"buildLog"`
	ErrorMsg          *string `json:"errorMsg"`
	ImageTag          *string `json:"imageTag"`
	TriggeredBy       string  `json:"triggeredBy"`
	TriggeredByUserID *string `json:"triggeredByUserId"`
	StartedAt         string  `json:"startedAt"`
	FinishedAt        *string `json:"finishedAt"`
}

func ResponseFromDB(row db.Deployment) Response {
	return Response{
		ID:                row.ID.String(),
		ProjectID:         row.ProjectID.String(),
		CommitSha:         row.CommitSha,
		CommitMessage:     row.CommitMessage,
		Status:            row.Status,
		BuildLog:          row.BuildLog,
		ErrorMsg:          row.ErrorMsg,
		ImageTag:          row.ImageTag,
		TriggeredBy:       row.TriggeredBy,
		TriggeredByUserID: uuidString(row.TriggeredByUserID),
		StartedAt:         formatTimestamp(row.StartedAt.Time, row.StartedAt.Valid),
		FinishedAt:        optionalTimestamp(row.FinishedAt.Time, row.FinishedAt.Valid),
	}
}

func ResponsesFromDB(rows []db.Deployment) []Response {
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

func optionalTimestamp(t time.Time, valid bool) *string {
	if !valid {
		return nil
	}
	formatted := t.UTC().Format(time.RFC3339)
	return &formatted
}
