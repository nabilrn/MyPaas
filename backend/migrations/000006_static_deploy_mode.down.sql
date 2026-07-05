UPDATE projects
SET deploy_mode = 'dockerfile',
    resource_profile = 'node-python',
    updated_at = NOW()
WHERE deploy_mode = 'static';

ALTER TABLE projects
    DROP CONSTRAINT IF EXISTS projects_deploy_mode_check;

ALTER TABLE projects
    ADD CONSTRAINT projects_deploy_mode_check
    CHECK (deploy_mode IN ('dockerfile', 'compose'));
