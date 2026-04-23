-- name: GetDeploymentByID :one
SELECT id, project_id, commit_sha, status, error_message, started_at, completed_at, created_at
FROM deployments
WHERE id = $1;

-- name: ListDeploymentsByProject :many
SELECT id, project_id, commit_sha, status, error_message, started_at, completed_at, created_at
FROM deployments
WHERE project_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CreateDeployment :one
INSERT INTO deployments (project_id, commit_sha, status)
VALUES ($1, $2, 'pending')
RETURNING id, project_id, commit_sha, status, error_message, started_at, completed_at, created_at;

-- name: UpdateDeploymentStatus :exec
UPDATE deployments
SET status = $2, updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: UpdateDeploymentStarted :exec
UPDATE deployments
SET status = 'building', started_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: UpdateDeploymentCompleted :exec
UPDATE deployments
SET status = $2, completed_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: UpdateDeploymentFailed :exec
UPDATE deployments
SET status = 'failed', error_message = $2, completed_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: GetLatestDeployment :one
SELECT id, project_id, commit_sha, status, error_message, started_at, completed_at, created_at
FROM deployments
WHERE project_id = $1 AND status = 'deployed'
ORDER BY created_at DESC
LIMIT 1;

-- name: GetPreviousDeployment :one
SELECT id, project_id, commit_sha, status, error_message, started_at, completed_at, created_at
FROM deployments
WHERE project_id = $1 AND status = 'deployed' AND id != $2
ORDER BY created_at DESC
LIMIT 1;
