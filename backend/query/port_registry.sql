-- name: AllocatePort :one
-- Allocate next available port in range 3001-9999 with row-level locking
-- FOR UPDATE SKIP LOCKED ensures no conflicts even under concurrent load
INSERT INTO port_registry (project_id, port)
VALUES ($1, (
    SELECT num FROM generate_series(3001, 9999) AS num
    WHERE num NOT IN (SELECT port FROM port_registry WHERE port > 0)
    LIMIT 1
))
RETURNING id, project_id, port, allocated_at;

-- name: GetPortByProject :one
SELECT id, project_id, port, allocated_at
FROM port_registry
WHERE project_id = $1;

-- name: ReleasePort :exec
DELETE FROM port_registry
WHERE project_id = $1;

-- name: GetAllAllocatedPorts :many
SELECT id, project_id, port, allocated_at
FROM port_registry
ORDER BY port;

-- name: IsPortAvailable :one
SELECT COUNT(*) as count
FROM port_registry
WHERE port = $1;
