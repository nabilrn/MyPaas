DROP INDEX IF EXISTS idx_projects_allocated_port_active_unique;
DROP INDEX IF EXISTS idx_projects_subdomain_active_unique;
DROP INDEX IF EXISTS idx_projects_name_active_unique;

ALTER TABLE projects
    ADD CONSTRAINT projects_name_key UNIQUE (name),
    ADD CONSTRAINT projects_subdomain_key UNIQUE (subdomain),
    ADD CONSTRAINT projects_allocated_port_key UNIQUE (allocated_port);
