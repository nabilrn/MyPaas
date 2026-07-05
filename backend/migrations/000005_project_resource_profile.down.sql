ALTER TABLE projects
    DROP CONSTRAINT IF EXISTS projects_resource_profile_check;

ALTER TABLE projects
    DROP COLUMN IF EXISTS resource_profile;
