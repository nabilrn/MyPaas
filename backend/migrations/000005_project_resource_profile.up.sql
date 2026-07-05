ALTER TABLE projects
    ADD COLUMN resource_profile VARCHAR(30) NOT NULL DEFAULT 'custom';

ALTER TABLE projects
    ADD CONSTRAINT projects_resource_profile_check
    CHECK (resource_profile IN ('static', 'go-small', 'node-python', 'compose-main', 'custom'));
