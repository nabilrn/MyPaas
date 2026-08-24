-- name: GetDeploymentByID :one
SELECT id, project_id, commit_sha, commit_message, status, build_log, error_msg, image_tag,
       triggered_by, triggered_by_user_id, started_at, finished_at
FROM deployments
WHERE id = $1;

-- name: ListDeploymentsByProject :many
SELECT id, project_id, commit_sha, commit_message, status, build_log, error_msg, image_tag,
       triggered_by, triggered_by_user_id, started_at, finished_at
FROM deployments
WHERE project_id = $1
ORDER BY started_at DESC
LIMIT $2 OFFSET $3;

-- name: ListDeploymentQueueItems :many
SELECT d.id, d.project_id, p.name AS project_name, p.subdomain AS project_subdomain,
       d.status, d.triggered_by, d.started_at, d.finished_at, d.error_msg
FROM deployments d
JOIN projects p ON p.id = d.project_id
WHERE p.deleted_at IS NULL
  AND (
    d.status IN ('queued', 'cloning', 'building', 'starting')
    OR (
      d.status = 'failed'
      AND d.finished_at > NOW() - INTERVAL '24 hours'
    )
  )
ORDER BY
  CASE d.status
    WHEN 'queued' THEN 0
    WHEN 'cloning' THEN 1
    WHEN 'building' THEN 2
    WHEN 'starting' THEN 3
    ELSE 4
  END,
  d.started_at DESC
LIMIT $1;

-- name: GetActiveDeploymentByProject :one
SELECT id, project_id, commit_sha, commit_message, status, build_log, error_msg, image_tag,
       triggered_by, triggered_by_user_id, started_at, finished_at
FROM deployments
WHERE project_id = $1
  AND status IN ('queued', 'cloning', 'building', 'starting')
ORDER BY started_at DESC
LIMIT 1;

-- name: CreateDeployment :one
INSERT INTO deployments (
    project_id, commit_sha, commit_message,
    status, triggered_by, triggered_by_user_id, image_tag
) VALUES ($1, $2, $3, 'queued', $4, $5, $6)
RETURNING id, project_id, commit_sha, commit_message, status, build_log, error_msg, image_tag,
          triggered_by, triggered_by_user_id, started_at, finished_at;

-- name: UpdateDeploymentStatus :exec
UPDATE deployments
SET status = $2
WHERE id = $1;

-- name: SetDeploymentBuildInfo :exec
UPDATE deployments
SET commit_sha = $2,
    commit_message = $3,
    image_tag = $4
WHERE id = $1;

-- name: AppendBuildLog :exec
UPDATE deployments
SET build_log = COALESCE(build_log, '') || $2
WHERE id = $1;

-- name: FinishDeployment :exec
UPDATE deployments
SET status      = $2,
    finished_at = NOW()
WHERE id = $1;

-- name: FailDeployment :exec
UPDATE deployments
SET status      = 'failed',
    error_msg   = $2,
    finished_at = NOW()
WHERE id = $1;

-- name: FailInterruptedDeployments :exec
UPDATE deployments
SET status      = 'failed',
    error_msg   = $1,
    finished_at = NOW()
WHERE status IN ('queued', 'cloning', 'building', 'starting');

-- name: GetLatestRunningDeployment :one
SELECT id, project_id, commit_sha, commit_message, status, build_log, error_msg, image_tag,
       triggered_by, triggered_by_user_id, started_at, finished_at
FROM deployments
WHERE project_id = $1 AND status = 'running'
ORDER BY started_at DESC, id DESC
LIMIT 1;

-- name: GetRollbackTarget :one
-- Returns the most recent successfully-run deployment that is not the current one.
SELECT id, project_id, commit_sha, commit_message, status, build_log, error_msg, image_tag,
       triggered_by, triggered_by_user_id, started_at, finished_at
FROM deployments
WHERE project_id = $1
  AND status     = 'running'
  AND id        != $2
ORDER BY started_at DESC
LIMIT 1;

-- name: CountDeploymentsByProject :one
SELECT COUNT(*) FROM deployments WHERE project_id = $1;

-- name: PruneOldDeployments :exec
-- Keep the 20 most recent; delete the rest.
DELETE FROM deployments
WHERE deployments.project_id = $1
  AND deployments.id NOT IN (
      SELECT kept.id FROM deployments AS kept
      WHERE kept.project_id = $1
      ORDER BY kept.started_at DESC
      LIMIT 20
  );
