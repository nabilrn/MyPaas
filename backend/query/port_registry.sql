-- name: AcquireAvailablePort :one
-- Lock one available port for atomic allocation. Must be called inside a transaction.
-- Caller updates the row with SetPortInUse immediately after.
SELECT port
FROM port_registry
WHERE status = 'available'
ORDER BY port
LIMIT 1
FOR UPDATE SKIP LOCKED;

-- name: SetPortInUse :exec
UPDATE port_registry
SET status      = 'in_use',
    project_id  = $2,
    assigned_at = NOW()
WHERE port = $1;

-- name: ReleasePortByProject :exec
UPDATE port_registry
SET status      = 'available',
    project_id  = NULL,
    assigned_at = NULL
WHERE project_id = $1;

-- name: ReleasePort :exec
UPDATE port_registry
SET status      = 'available',
    project_id  = NULL,
    assigned_at = NULL
WHERE port = $1;

-- name: GetPortByProject :one
SELECT port, project_id, status, assigned_at
FROM port_registry
WHERE project_id = $1;

-- name: ListInUsePorts :many
SELECT port, project_id, status, assigned_at
FROM port_registry
WHERE status = 'in_use'
ORDER BY port;
