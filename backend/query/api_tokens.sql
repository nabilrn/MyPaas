-- name: CreateAPIToken :one
INSERT INTO api_tokens (user_id, name, token_hash, token_prefix, scopes, expires_at)
VALUES (sqlc.arg(user_id), sqlc.arg(name), sqlc.arg(token_hash), sqlc.arg(token_prefix), sqlc.arg(scopes), sqlc.narg(expires_at))
RETURNING *;

-- name: ListAPITokens :many
SELECT *
FROM api_tokens
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: RevokeAPIToken :execrows
UPDATE api_tokens
SET revoked_at = COALESCE(revoked_at, NOW())
WHERE id = sqlc.arg(id)
  AND user_id = sqlc.arg(user_id);

-- name: RevokeActiveAPITokensByName :exec
UPDATE api_tokens
SET revoked_at = NOW()
WHERE user_id = sqlc.arg(user_id)
  AND name = sqlc.arg(name)
  AND revoked_at IS NULL;

-- name: AuthenticateAPIToken :one
UPDATE api_tokens
SET last_used_at = NOW()
WHERE token_hash = $1
  AND revoked_at IS NULL
  AND (expires_at IS NULL OR expires_at > NOW())
RETURNING *;
