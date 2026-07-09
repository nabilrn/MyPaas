-- name: CreateDBStudioSession :one
INSERT INTO db_studio_sessions (
    project_id, user_id, mode, expires_at
) VALUES ($1, $2, 'write', $3)
RETURNING id, project_id, user_id, mode, expires_at, revoked_at, created_at;

-- name: GetActiveDBStudioSession :one
SELECT id, project_id, user_id, mode, expires_at, revoked_at, created_at
FROM db_studio_sessions
WHERE project_id = $1
  AND user_id = $2
  AND mode = 'write'
  AND revoked_at IS NULL
  AND expires_at > NOW()
ORDER BY expires_at DESC
LIMIT 1;

-- name: RevokeDBStudioSession :exec
UPDATE db_studio_sessions
SET revoked_at = NOW()
WHERE id = $1
  AND project_id = $2
  AND user_id = $3
  AND revoked_at IS NULL;
