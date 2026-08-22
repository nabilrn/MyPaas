package deployment

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"mypaas/internal/db"
)

func TestQueueSummaryFromDB(t *testing.T) {
	t.Parallel()

	now := pgtype.Timestamp{Time: time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC), Valid: true}
	rows := []db.ListDeploymentQueueItemsRow{
		queueRow("queued", now),
		queueRow("cloning", now),
		queueRow("building", now),
		queueRow("starting", now),
		queueRow("failed", now),
	}

	got := QueueSummaryFromDB(rows)
	if got.Queued != 1 || got.Active != 3 || got.Failed24h != 1 {
		t.Fatalf("summary counts = queued %d active %d failed %d", got.Queued, got.Active, got.Failed24h)
	}
	if len(got.Items) != len(rows) {
		t.Fatalf("items = %d, want %d", len(got.Items), len(rows))
	}
	if got.Items[0].StartedAt != "2026-08-22T12:00:00Z" {
		t.Fatalf("startedAt = %q", got.Items[0].StartedAt)
	}
}

func queueRow(status string, startedAt pgtype.Timestamp) db.ListDeploymentQueueItemsRow {
	return db.ListDeploymentQueueItemsRow{
		ID:               uuid.New(),
		ProjectID:        uuid.New(),
		ProjectName:      "example",
		ProjectSubdomain: "example",
		Status:           status,
		TriggeredBy:      "manual",
		StartedAt:        startedAt,
	}
}
