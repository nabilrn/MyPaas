-- CPU is host-shared rather than reserved per project.
-- Keep the legacy cpu_limit column for API/schema compatibility, using 0 as
-- the Docker/Podman no-limit value for existing projects.
UPDATE projects
SET cpu_limit = 0
WHERE cpu_limit <> 0;
