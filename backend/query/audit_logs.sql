-- name: CreateAuditLog :exec
INSERT INTO audit_logs (
    user_id, action, resource_type, resource_id, metadata, ip_address, user_agent
) VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: ListAuditLogs :many
SELECT id, user_id, action, resource_type, resource_id, metadata, ip_address, user_agent, created_at
FROM audit_logs
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;
