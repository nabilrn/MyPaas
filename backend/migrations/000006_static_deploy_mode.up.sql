ALTER TABLE projects
    DROP CONSTRAINT IF EXISTS projects_deploy_mode_check;

ALTER TABLE projects
    ADD CONSTRAINT projects_deploy_mode_check
    CHECK (deploy_mode IN ('dockerfile', 'compose', 'static'));
