-- name: CreateWebhookDelivery :one
INSERT INTO webhook_deliveries (
    project_id,
    github_delivery_id,
    signature_valid,
    event_type,
    branch,
    processed,
    deployment_id
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, project_id, github_delivery_id, signature_valid, event_type,
          branch, processed, deployment_id, received_at;

-- name: MarkWebhookDeliveryProcessed :exec
UPDATE webhook_deliveries
SET processed     = TRUE,
    deployment_id = $2
WHERE id = $1;
