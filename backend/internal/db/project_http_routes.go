package db

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

func (q *Queries) GetProjectAdditionalRoutes(ctx context.Context, projectID uuid.UUID) (json.RawMessage, error) {
	row := q.db.QueryRow(ctx, `
SELECT additional_routes
FROM projects
WHERE id = $1 AND deleted_at IS NULL
`, projectID)
	var routes json.RawMessage
	if err := row.Scan(&routes); err != nil {
		return nil, err
	}
	return routes, nil
}

func (q *Queries) SetProjectAdditionalRoutes(ctx context.Context, projectID uuid.UUID, routes json.RawMessage) error {
	_, err := q.db.Exec(ctx, `
UPDATE projects
SET additional_routes = $2,
    updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL
`, projectID, routes)
	return err
}
