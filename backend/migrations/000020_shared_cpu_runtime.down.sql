-- Restore the legacy profile CPU defaults when rolling back the shared-CPU
-- runtime contract. Exact historical per-project custom CPU values cannot be
-- reconstructed after the forward normalization.
UPDATE projects
SET cpu_limit = CASE resource_profile
    WHEN 'static' THEN 0.01
    WHEN 'go-small' THEN 0.20
    WHEN 'node-python' THEN 0.35
    WHEN 'compose-main' THEN 0.35
    ELSE 0.50
END
WHERE cpu_limit = 0;
