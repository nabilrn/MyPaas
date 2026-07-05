ALTER TABLE projects
    DROP CONSTRAINT IF EXISTS projects_name_key,
    DROP CONSTRAINT IF EXISTS projects_subdomain_key,
    DROP CONSTRAINT IF EXISTS projects_allocated_port_key;

CREATE UNIQUE INDEX IF NOT EXISTS idx_projects_name_active_unique
    ON projects(name)
    WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_projects_subdomain_active_unique
    ON projects(subdomain)
    WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_projects_allocated_port_active_unique
    ON projects(allocated_port)
    WHERE allocated_port IS NOT NULL
      AND deleted_at IS NULL;
