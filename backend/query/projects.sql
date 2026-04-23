-- name: GetProjectByID :one
SELECT id, owner_id, name, git_url, mode, port, main_service, description, created_at, updated_at
FROM projects
WHERE id = $1;

-- name: ListProjectsByOwner :many
SELECT id, owner_id, name, git_url, mode, port, main_service, description, created_at, updated_at
FROM projects
WHERE owner_id = $1
ORDER BY created_at DESC;

-- name: ListAllProjects :many
SELECT id, owner_id, name, git_url, mode, port, main_service, description, created_at, updated_at
FROM projects
ORDER BY created_at DESC;

-- name: CreateProject :one
INSERT INTO projects (owner_id, name, git_url, mode, port, main_service, description)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, owner_id, name, git_url, mode, port, main_service, description, created_at, updated_at;

-- name: UpdateProject :exec
UPDATE projects
SET name = COALESCE($2, name),
    git_url = COALESCE($3, git_url),
    mode = COALESCE($4, mode),
    port = COALESCE($5, port),
    main_service = COALESCE($6, main_service),
    description = COALESCE($7, description),
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: DeleteProject :exec
DELETE FROM projects
WHERE id = $1;

-- name: GetProjectByName :one
SELECT id, owner_id, name, git_url, mode, port, main_service, description, created_at, updated_at
FROM projects
WHERE owner_id = $1 AND name = $2;
