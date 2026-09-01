-- name: GetUserByEmail :one
SELECT id, email, github_id, github_username, avatar_url, role, created_at, last_login_at
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, email, github_id, github_username, avatar_url, role, created_at, last_login_at
FROM users
WHERE id = $1;

-- name: CreateUser :one
INSERT INTO users (email, github_id, github_username, avatar_url, role)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, email, github_id, github_username, avatar_url, role, created_at, last_login_at;

-- name: UpdateLastLogin :exec
UPDATE users
SET last_login_at = NOW()
WHERE id = $1;

-- name: UpdateUserGithubProfile :one
UPDATE users
SET github_id = $2,
    github_username = $3,
    avatar_url = $4,
    last_login_at = NOW()
WHERE id = $1
RETURNING id, email, github_id, github_username, avatar_url, role, created_at, last_login_at;

-- name: ListUsers :many
SELECT id, email, github_id, github_username, avatar_url, role, created_at, last_login_at
FROM users
ORDER BY created_at DESC;

-- name: GetMasterUserID :one
SELECT id
FROM users
ORDER BY created_at ASC, id ASC
LIMIT 1;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;

-- name: GetGithubAccessToken :one
SELECT github_access_token_encrypted, github_access_token_nonce
FROM users
WHERE id = $1;

-- name: SetGithubAccessToken :exec
UPDATE users
SET github_access_token_encrypted = $2,
    github_access_token_nonce = $3
WHERE id = $1;

-- name: ClearGithubAccessToken :exec
UPDATE users
SET github_access_token_encrypted = NULL,
    github_access_token_nonce = NULL
WHERE id = $1;
