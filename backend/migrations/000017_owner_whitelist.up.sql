-- MyPaaS is an owner-only control plane. Existing collaborators keep access
-- but receive the same administrative permissions as every whitelisted user.
UPDATE users
SET role = 'owner'
WHERE role <> 'owner';

ALTER TABLE users
    ADD CONSTRAINT users_role_owner_only CHECK (role = 'owner');
