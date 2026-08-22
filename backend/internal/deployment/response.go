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

type QueueItemResponse struct {
	ID               string  `json:"id"`
	ProjectID        string  `json:"projectId"`
	ProjectName      string  `json:"projectName"`
	ProjectSubdomain string  `json:"projectSubdomain"`
	Status           string  `json:"status"`
	TriggeredBy      string  `json:"triggeredBy"`
	StartedAt        string  `json:"startedAt"`
	FinishedAt       *string `json:"finishedAt"`
	ErrorMsg         *string `json:"errorMsg"`
}

type QueueSummaryResponse struct {
	Queued    int32               `json:"queued"`
	Active    int32               `json:"active"`
	Failed24h int32               `json:"failed24h"`
	Items     []QueueItemResponse `json:"items"`
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

func QueueSummaryFromDB(rows []db.ListDeploymentQueueItemsRow) QueueSummaryResponse {
	resp := QueueSummaryResponse{Items: make([]QueueItemResponse, 0, len(rows))}
	for _, row := range rows {
		switch row.Status {
		case "queued":
			resp.Queued++
		case "cloning", "building", "starting":
			resp.Active++
		case "failed":
			resp.Failed24h++
		}
		resp.Items = append(resp.Items, QueueItemResponse{
			ID:               row.ID.String(),
			ProjectID:        row.ProjectID.String(),
			ProjectName:      row.ProjectName,
			ProjectSubdomain: row.ProjectSubdomain,
			Status:           row.Status,
			TriggeredBy:      row.TriggeredBy,
			StartedAt:        formatTimestamp(row.StartedAt.Time, row.StartedAt.Valid),
			FinishedAt:       optionalTimestamp(row.FinishedAt.Time, row.FinishedAt.Valid),
			ErrorMsg:         row.ErrorMsg,
		})
	}
	return resp
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
