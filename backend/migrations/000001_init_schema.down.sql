ALTER TABLE projects DROP CONSTRAINT IF EXISTS fk_projects_active_deployment;

DROP TABLE IF EXISTS webhook_deliveries;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS port_registry;
DROP TABLE IF EXISTS env_vars;
DROP TABLE IF EXISTS deployments;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS users;
