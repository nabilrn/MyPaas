-- name: ListEnvVarsByProject :many
SELECT id, project_id, key, value_encrypted, value_nonce, created_at, updated_at
FROM env_vars
WHERE project_id = $1
ORDER BY key;

-- name: GetEnvVar :one
SELECT id, project_id, key, value_encrypted, value_nonce, created_at, updated_at
FROM env_vars
WHERE project_id = $1 AND key = $2;

-- name: UpsertEnvVar :one
INSERT INTO env_vars (project_id, key, value_encrypted, value_nonce)
VALUES ($1, $2, $3, $4)
ON CONFLICT (project_id, key) DO UPDATE
    SET value_encrypted = EXCLUDED.value_encrypted,
        value_nonce     = EXCLUDED.value_nonce,
        updated_at      = NOW()
RETURNING id, project_id, key, value_encrypted, value_nonce, created_at, updated_at;

-- name: DeleteEnvVar :exec
DELETE FROM env_vars
WHERE project_id = $1 AND key = $2;

-- name: DeleteAllEnvVars :exec
DELETE FROM env_vars
WHERE project_id = $1;
