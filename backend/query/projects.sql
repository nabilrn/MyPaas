-- name: GetProjectByID :one
SELECT id, user_id, name, repo_url, branch, subdomain, deploy_mode, main_service,
       app_port, webhook_secret, allocated_port, memory_limit_mb, cpu_limit,
       status, active_deployment_id, created_at, updated_at, deleted_at, resource_profile
FROM projects
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetProjectByName :one
SELECT id, user_id, name, repo_url, branch, subdomain, deploy_mode, main_service,
       app_port, webhook_secret, allocated_port, memory_limit_mb, cpu_limit,
       status, active_deployment_id, created_at, updated_at, deleted_at, resource_profile
FROM projects
WHERE name = $1 AND deleted_at IS NULL;

-- name: ListProjectsByUser :many
SELECT id, user_id, name, repo_url, branch, subdomain, deploy_mode, main_service,
       app_port, webhook_secret, allocated_port, memory_limit_mb, cpu_limit,
       status, active_deployment_id, created_at, updated_at, deleted_at, resource_profile
FROM projects
WHERE user_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: CountProjectsByUser :one
SELECT COUNT(*)
FROM projects
WHERE user_id = $1
  AND deleted_at IS NULL;

-- name: CreateProject :one
INSERT INTO projects (
    user_id, name, repo_url, branch, subdomain, deploy_mode,
    resource_profile, main_service, app_port, webhook_secret, memory_limit_mb, cpu_limit
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING id, user_id, name, repo_url, branch, subdomain, deploy_mode, main_service,
          app_port, webhook_secret, allocated_port, memory_limit_mb, cpu_limit,
          status, active_deployment_id, created_at, updated_at, deleted_at, resource_profile;

-- name: UpdateProject :exec
UPDATE projects
SET name            = $2,
    subdomain       = $3,
    branch          = $4,
    resource_profile = $5,
    app_port        = $6,
    memory_limit_mb = $7,
    cpu_limit       = $8,
    updated_at      = NOW()
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateProjectStatus :exec
UPDATE projects
SET status     = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: ResetBuildingProjects :exec
UPDATE projects
SET status     = 'pending',
    updated_at = NOW()
WHERE status = 'building'
  AND deleted_at IS NULL;

-- name: UpdateProjectWebhookSecret :one
UPDATE projects
SET webhook_secret = $2,
    updated_at     = NOW()
WHERE id = $1
  AND deleted_at IS NULL
RETURNING webhook_secret;

-- name: SetProjectActiveDeployment :exec
UPDATE projects
SET active_deployment_id = $2,
    status               = $3,
    updated_at           = NOW()
WHERE id = $1;

-- name: SetProjectAllocatedPort :exec
UPDATE projects
SET allocated_port = $2,
    updated_at     = NOW()
WHERE id = $1;

-- name: SoftDeleteProject :exec
UPDATE projects
SET allocated_port       = NULL,
    active_deployment_id = NULL,
    status               = 'stopped',
    deleted_at           = NOW(),
    updated_at           = NOW()
WHERE id = $1;

-- name: GetTotalResourcesByUser :one
SELECT
    COALESCE(SUM(memory_limit_mb), 0)::INT      AS total_memory_mb,
    COALESCE(SUM(cpu_limit), 0.0)::NUMERIC(6,2) AS total_cpu
FROM projects
WHERE user_id = $1
  AND deleted_at IS NULL;

-- name: GetTotalResourcesByUserExcludingProject :one
SELECT
    COALESCE(SUM(memory_limit_mb), 0)::INT      AS total_memory_mb,
    COALESCE(SUM(cpu_limit), 0.0)::NUMERIC(6,2) AS total_cpu
FROM projects
WHERE user_id = $1
  AND id <> $2
  AND deleted_at IS NULL;
