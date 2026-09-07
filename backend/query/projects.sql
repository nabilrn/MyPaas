-- name: GetProjectByID :one
SELECT *
FROM projects
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetProjectByName :one
SELECT *
FROM projects
WHERE name = $1 AND deleted_at IS NULL;

-- name: ListProjectsByUser :many
SELECT *
FROM projects
WHERE user_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: ListRoutableProjects :many
SELECT *
FROM projects
WHERE status = 'running'
  AND deleted_at IS NULL
ORDER BY created_at ASC;

-- name: CountProjectsByUser :one
SELECT COUNT(*)
FROM projects
WHERE user_id = $1
  AND deleted_at IS NULL;

-- name: CreateProject :one
INSERT INTO projects (
    user_id, name, repo_url, branch, subdomain, deploy_mode,
    resource_profile, main_service, app_port, webhook_secret, memory_limit_mb, cpu_limit,
    compose_file_path, compose_override_paths, compose_profiles, compose_workdir,
    service_resources, static_frontend_path, base_directory, image_ref
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
RETURNING *;

-- name: UpdateProject :exec
UPDATE projects
SET name                 = $2,
    subdomain            = $3,
    branch               = $4,
    resource_profile     = $5,
    app_port             = $6,
    memory_limit_mb      = $7,
    cpu_limit            = $8,
    main_service         = $9,
    compose_file_path    = $10,
    compose_override_paths = $11,
    compose_profiles     = $12,
    compose_workdir      = $13,
    service_resources    = $14,
    static_frontend_path = $15,
    base_directory       = $16,
    image_ref            = $17,
    updated_at           = NOW()
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

-- name: GetGlobalResourceUsage :one
SELECT
    COALESCE(SUM(memory_limit_mb) FILTER (WHERE deploy_mode <> 'static'), 0)::INT      AS total_memory_mb,
    COALESCE(MAX(cpu_limit) FILTER (WHERE deploy_mode <> 'static'), 0.0)::NUMERIC(6,2) AS total_cpu
FROM projects
WHERE deleted_at IS NULL;
