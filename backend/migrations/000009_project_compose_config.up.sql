-- Flexible Compose configuration: subdirectory compose files, override chains, profiles.
-- All columns nullable so existing dockerfile/static projects and root-only compose
-- projects keep working unchanged (NULL -> fall back to repo-root scan).
ALTER TABLE projects
    ADD COLUMN compose_file_path     VARCHAR(255),                                   -- repo-relative path to the primary compose file, e.g. "infra/docker-compose.yml"
    ADD COLUMN compose_override_paths VARCHAR[]   NOT NULL DEFAULT '{}',             -- additional -f files (repo-relative), applied before MyPaas override
    ADD COLUMN compose_profiles       VARCHAR[]   NOT NULL DEFAULT '{}',             -- COMPOSE_PROFILES values for this project
    ADD COLUMN compose_workdir        VARCHAR(255);                                  -- repo-relative cwd override; defaults to the compose file's directory

ALTER TABLE projects
    ADD CONSTRAINT chk_compose_file_path_safe
    CHECK (
        compose_file_path IS NULL
        OR (compose_file_path NOT LIKE '/%'                            -- no absolute paths
            AND compose_file_path NOT LIKE '%..%'                      -- no traversal segments
            AND compose_file_path NOT LIKE '%\%'                       -- no backslashes (POSIX-style only)
            AND length(compose_file_path) <= 255)
    );

ALTER TABLE projects
    ADD CONSTRAINT chk_compose_workdir_safe
    CHECK (
        compose_workdir IS NULL
        OR (compose_workdir NOT LIKE '/%'
            AND compose_workdir NOT LIKE '%..%'
            AND compose_workdir NOT LIKE '%\%'
            AND length(compose_workdir) <= 255)
    );

ALTER TABLE projects
    ADD CONSTRAINT chk_compose_only_when_relevant
    CHECK (
        (compose_file_path IS NULL AND compose_override_paths = '{}' AND compose_profiles = '{}' AND compose_workdir IS NULL)
        OR deploy_mode = 'compose'
    );
