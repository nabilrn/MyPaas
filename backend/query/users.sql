-- name: GetUserByEmail :one
SELECT id, email, github_username, is_admin, created_at, updated_at
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, email, github_username, is_admin, created_at, updated_at
FROM users
WHERE id = $1;

-- name: CreateUser :one
INSERT INTO users (email, github_username, is_admin)
VALUES ($1, $2, COALESCE($3, FALSE))
RETURNING id, email, github_username, is_admin, created_at, updated_at;

-- name: ListUsers :many
SELECT id, email, github_username, is_admin, created_at, updated_at
FROM users
ORDER BY created_at DESC;

-- name: UpdateUserAdmin :exec
UPDATE users
SET is_admin = $2, updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;
