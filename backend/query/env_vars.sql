-- name: GetEnvVarsByProject :many
SELECT id, project_id, key, value_encrypted, created_at, updated_at
FROM env_vars
WHERE project_id = $1
ORDER BY key;

-- name: GetEnvVar :one
SELECT id, project_id, key, value_encrypted, created_at, updated_at
FROM env_vars
WHERE project_id = $1 AND key = $2;

-- name: CreateEnvVar :one
INSERT INTO env_vars (project_id, key, value_encrypted)
VALUES ($1, $2, $3)
RETURNING id, project_id, key, value_encrypted, created_at, updated_at;

-- name: UpdateEnvVar :exec
UPDATE env_vars
SET value_encrypted = $3, updated_at = CURRENT_TIMESTAMP
WHERE project_id = $1 AND key = $2;

-- name: DeleteEnvVar :exec
DELETE FROM env_vars
WHERE project_id = $1 AND key = $2;

-- name: DeleteEnvVarsByProject :exec
DELETE FROM env_vars
WHERE project_id = $1;
