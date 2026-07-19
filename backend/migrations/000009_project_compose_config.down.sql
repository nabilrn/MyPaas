ALTER TABLE projects
    DROP CONSTRAINT IF EXISTS chk_compose_only_when_relevant;

ALTER TABLE projects
    DROP CONSTRAINT IF EXISTS chk_compose_workdir_safe;

ALTER TABLE projects
    DROP CONSTRAINT IF EXISTS chk_compose_file_path_safe;

ALTER TABLE projects
    DROP COLUMN IF EXISTS compose_file_path,
    DROP COLUMN IF EXISTS compose_override_paths,
    DROP COLUMN IF EXISTS compose_profiles,
    DROP COLUMN IF EXISTS compose_workdir;
