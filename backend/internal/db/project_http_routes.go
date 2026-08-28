package db

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

type ProjectAdditionalRouteConfig struct {
	ID         uuid.UUID
	Name       string
	Status     string
	DeployMode string
	Routes     json.RawMessage
	Deleted    bool
}

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

func (q *Queries) GetProjectAdditionalRouteConfig(ctx context.Context, projectID uuid.UUID) (ProjectAdditionalRouteConfig, error) {
	row := q.db.QueryRow(ctx, `
SELECT id, name, status, deploy_mode, additional_routes, deleted_at IS NOT NULL
FROM projects
WHERE id = $1
`, projectID)
	var item ProjectAdditionalRouteConfig
	if err := row.Scan(&item.ID, &item.Name, &item.Status, &item.DeployMode, &item.Routes, &item.Deleted); err != nil {
		return ProjectAdditionalRouteConfig{}, err
	}
	return item, nil
}

func (q *Queries) ProjectHTTPHostLabelInUse(ctx context.Context, label string, excludeProjectID uuid.UUID) (bool, error) {
	row := q.db.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1
    FROM projects AS p
    WHERE p.deleted_at IS NULL
      AND p.id <> $2
      AND (
          p.name = $1
          OR EXISTS (
              SELECT 1
              FROM jsonb_array_elements(p.additional_routes) AS route
              WHERE p.name || '-' || (route ->> 'name') = $1
          )
      )
)
`, label, excludeProjectID)
	var inUse bool
	if err := row.Scan(&inUse); err != nil {
		return false, err
	}
	return inUse, nil
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

func (q *Queries) ClearProjectAdditionalRoutes(ctx context.Context, projectID uuid.UUID) error {
	_, err := q.db.Exec(ctx, `
UPDATE projects
SET additional_routes = '[]'::jsonb,
    updated_at = NOW()
WHERE id = $1
`, projectID)
	return err
}

func (q *Queries) ListProjectAdditionalRouteConfigs(ctx context.Context) ([]ProjectAdditionalRouteConfig, error) {
	rows, err := q.db.Query(ctx, `
SELECT id, name, status, deploy_mode, additional_routes, deleted_at IS NOT NULL
FROM projects
WHERE additional_routes <> '[]'::jsonb
ORDER BY created_at ASC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	configs := make([]ProjectAdditionalRouteConfig, 0)
	for rows.Next() {
		var item ProjectAdditionalRouteConfig
		if err := rows.Scan(&item.ID, &item.Name, &item.Status, &item.DeployMode, &item.Routes, &item.Deleted); err != nil {
			return nil, err
		}
		configs = append(configs, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return configs, nil
}