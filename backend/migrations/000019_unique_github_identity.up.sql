WITH ranked_bindings AS (
    SELECT
        id,
        ROW_NUMBER() OVER (
            PARTITION BY github_id
            ORDER BY last_login_at DESC NULLS LAST, created_at DESC, id DESC
        ) AS binding_rank
    FROM users
    WHERE github_id IS NOT NULL
)
UPDATE users AS u
SET github_id = NULL,
    github_username = NULL,
    github_access_token_encrypted = NULL,
    github_access_token_nonce = NULL
FROM ranked_bindings AS ranked
WHERE u.id = ranked.id
  AND ranked.binding_rank > 1;

CREATE UNIQUE INDEX IF NOT EXISTS users_github_id_unique
ON users (github_id)
WHERE github_id IS NOT NULL;
