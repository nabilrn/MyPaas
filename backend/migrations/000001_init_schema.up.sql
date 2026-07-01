-- User whitelist
CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           VARCHAR(255) UNIQUE NOT NULL,
    github_id       VARCHAR(50),
    github_username VARCHAR(100),
    avatar_url      TEXT,
    role            VARCHAR(20) NOT NULL DEFAULT 'owner', -- owner | collaborator
    created_at      TIMESTAMP NOT NULL DEFAULT NOW(),
    last_login_at   TIMESTAMP
);

-- Project metadata
CREATE TABLE projects (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id              UUID NOT NULL REFERENCES users(id),
    name                 VARCHAR(30) UNIQUE NOT NULL,
    repo_url             TEXT NOT NULL,
    branch               VARCHAR(100) NOT NULL DEFAULT 'main',
    subdomain            VARCHAR(100) UNIQUE NOT NULL,
    deploy_mode          VARCHAR(20) NOT NULL CHECK (deploy_mode IN ('dockerfile', 'compose')),
    main_service         VARCHAR(100),
    app_port             INT NOT NULL,
    webhook_secret       TEXT NOT NULL,
    allocated_port       INT UNIQUE,
    memory_limit_mb      INT NOT NULL DEFAULT 512,
    cpu_limit            NUMERIC(3,2) NOT NULL DEFAULT 0.50,
    status               VARCHAR(20) NOT NULL DEFAULT 'pending'
                           CHECK (status IN ('pending', 'running', 'stopped', 'crashed', 'building')),
    active_deployment_id UUID,
    created_at           TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at           TIMESTAMP
);

CREATE INDEX idx_projects_user_id ON projects(user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_projects_status  ON projects(status)  WHERE deleted_at IS NULL;

-- Deployment history
CREATE TABLE deployments (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id           UUID NOT NULL REFERENCES projects(id),
    commit_sha           VARCHAR(40),
    commit_message       TEXT,
    status               VARCHAR(20) NOT NULL
                           CHECK (status IN ('queued', 'cloning', 'building', 'starting',
                                             'running', 'failed', 'stopped', 'rolled_back')),
    build_log            TEXT,
    error_msg            TEXT,
    image_tag            VARCHAR(255),
    triggered_by         VARCHAR(20) NOT NULL CHECK (triggered_by IN ('manual', 'webhook', 'rollback')),
    triggered_by_user_id UUID REFERENCES users(id),
    started_at           TIMESTAMP NOT NULL DEFAULT NOW(),
    finished_at          TIMESTAMP
);

CREATE INDEX idx_deployments_project_id ON deployments(project_id);
CREATE INDEX idx_deployments_status     ON deployments(status);

-- FK added after deployments table exists
ALTER TABLE projects
    ADD CONSTRAINT fk_projects_active_deployment
    FOREIGN KEY (active_deployment_id) REFERENCES deployments(id);

-- Environment variables (encrypted at rest, AES-256-GCM)
CREATE TABLE env_vars (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id      UUID NOT NULL REFERENCES projects(id),
    key             VARCHAR(100) NOT NULL,
    value_encrypted TEXT NOT NULL,
    value_nonce     TEXT NOT NULL, -- base64-encoded GCM nonce
    created_at      TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (project_id, key)
);

-- Port pool (pre-populated by migration 000002)
CREATE TABLE port_registry (
    port        INT PRIMARY KEY,
    project_id  UUID REFERENCES projects(id),
    status      VARCHAR(20) NOT NULL DEFAULT 'available'
                  CHECK (status IN ('available', 'in_use', 'reserved')),
    assigned_at TIMESTAMP
);

-- Audit log
CREATE TABLE audit_logs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID REFERENCES users(id),
    action        VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50),
    resource_id   UUID,
    metadata      JSONB,
    ip_address    INET,
    user_agent    TEXT,
    created_at    TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_user_id    ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at DESC);

-- Webhook delivery log (for debugging & deduplication)
CREATE TABLE webhook_deliveries (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id         UUID NOT NULL REFERENCES projects(id),
    github_delivery_id VARCHAR(100),
    signature_valid    BOOLEAN NOT NULL,
    event_type         VARCHAR(50),
    branch             VARCHAR(100),
    processed          BOOLEAN NOT NULL DEFAULT FALSE,
    deployment_id      UUID REFERENCES deployments(id),
    received_at        TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_webhook_deliveries_project_id ON webhook_deliveries(project_id);
CREATE INDEX idx_webhook_deliveries_processed  ON webhook_deliveries(processed);
