-- The owner-only policy is intentionally not downgraded automatically.
ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_role_owner_only;
