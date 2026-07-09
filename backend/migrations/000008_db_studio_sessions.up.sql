CREATE TABLE db_studio_sessions (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id),
    mode       VARCHAR(20) NOT NULL CHECK (mode IN ('write')),
    expires_at TIMESTAMP NOT NULL,
    revoked_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_db_studio_sessions_project_user_active
ON db_studio_sessions(project_id, user_id, expires_at DESC)
WHERE revoked_at IS NULL;
