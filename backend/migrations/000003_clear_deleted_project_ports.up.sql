UPDATE projects
SET allocated_port = NULL,
    active_deployment_id = NULL,
    status = 'stopped',
    updated_at = NOW()
WHERE deleted_at IS NOT NULL
  AND allocated_port IS NOT NULL;
