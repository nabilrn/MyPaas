package db

import (
	"context"

	"github.com/google/uuid"
)

// WebhookProjectStatus summarizes delivery evidence for one project without
// loading individual delivery rows. It mirrors the public webhook status
// contract: any verified delivery means connected, an unverified delivery means
// issue, and no delivery means unverified.
type WebhookProjectStatus struct {
	ProjectID   uuid.UUID
	HasDelivery bool
	HasVerified bool
}

func (q *Queries) ListWebhookProjectStatusesByUser(ctx context.Context, userID uuid.UUID) ([]WebhookProjectStatus, error) {
	rows, err := q.db.Query(ctx, `
SELECT
	p.id,
	COUNT(wd.id) > 0 AS has_delivery,
	COALESCE(BOOL_OR(wd.signature_valid), FALSE) AS has_verified
FROM projects p
LEFT JOIN webhook_deliveries wd ON wd.project_id = p.id
WHERE p.user_id = $1
  AND p.deleted_at IS NULL
GROUP BY p.id
ORDER BY p.id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]WebhookProjectStatus, 0)
	for rows.Next() {
		var item WebhookProjectStatus
		if err := rows.Scan(&item.ProjectID, &item.HasDelivery, &item.HasVerified); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
