CREATE TABLE api_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        VARCHAR(100) NOT NULL,
    token_hash  BYTEA NOT NULL UNIQUE,
    token_prefix VARCHAR(20) NOT NULL,
    scopes      TEXT[] NOT NULL,
    expires_at  TIMESTAMP,
    last_used_at TIMESTAMP,
    created_at  TIMESTAMP NOT NULL DEFAULT NOW(),
    revoked_at  TIMESTAMP
);

CREATE INDEX idx_api_tokens_user_id ON api_tokens(user_id);
CREATE INDEX idx_api_tokens_active_hash ON api_tokens(token_hash) WHERE revoked_at IS NULL;
