ALTER TABLE users
    DROP COLUMN IF EXISTS github_access_token_encrypted,
    DROP COLUMN IF EXISTS github_access_token_nonce;
