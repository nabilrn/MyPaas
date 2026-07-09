# ADR-015: DB Studio Lite

## Status

Accepted

## Context

MyPaas is used to learn and operate small deployed projects. Users often need to inspect or lightly adjust production data after a deploy without opening SSH, DataGrip, DBeaver, or container shells.

Full database IDE functionality would add too much security and UX surface for the current single-VM scope.

## Decision

Add a project-level DB Studio Lite:

- Support PostgreSQL, MySQL, and MariaDB first.
- Discover connection details from encrypted project environment variables.
- Provide schema/table browsing, paginated row viewing, insert, update, and delete.
- Keep raw SQL console out of MVP.
- Require explicit temporary write mode before insert/update/delete.
- Allow update/delete only when a table has a primary key.
- Block system schemas.
- Quote identifiers through driver-specific adapters after validating them against introspection results.
- Audit write actions.

This feature may use dynamic SQL only inside `internal/dbstudio` adapters because it targets user project databases with dynamic schemas. MyPaas application database queries remain sqlc-managed.

## Consequences

- The dashboard can offer a Prisma Studio-like workflow for small CRUD tasks.
- The first version stays intentionally limited and safer than a full DB client.
- Compose database access requires MyPaas API to reach the project Compose network; the service can connect the API container to the project default network when needed.
- SQLite, SQL Server, Oracle, MongoDB, and Redis are out of scope for this ADR.
