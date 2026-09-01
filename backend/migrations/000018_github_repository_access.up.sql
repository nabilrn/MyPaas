ALTER TABLE users
    ADD COLUMN github_access_token_encrypted TEXT,
    ADD COLUMN github_access_token_nonce TEXT;
