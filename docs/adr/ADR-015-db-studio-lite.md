# ADR-015: DB Studio Lite

## Status

Accepted

## Context

MyPaas is used to learn and operate small deployed projects. Users often need to inspect or lightly adjust production data after a deploy without opening SSH, DataGrip, DBeaver, or container shells.

Full database IDE functionality would add too much security and UX surface for the current single-VM scope. The stable release therefore keeps DB Studio intentionally narrower than a general-purpose database editor.

## Decision

Add a project-level DB Studio Lite:

- Support PostgreSQL, MySQL, MariaDB, and persistent SQLite databases.
- Discover server-database connection details from encrypted project environment variables. Prefer explicit SQLite paths from those variables, with conservative runtime discovery as a fallback for eligible persistent mounts.
- Provide schema/table browsing, paginated row viewing, table-scoped string search, schema metadata, and ERD inspection.
- Keep raw SQL console out of scope.
- Keep row insertion and row deletion disabled for the stable write boundary.
- Allow only explicit temporary write mode for row updates.
- Allow updates only when a table has a primary key.
- Keep primary-key and generated columns immutable from DB Studio.
- Allow mutation only for recognized scalar types. Complex/rich/custom types remain read-only.
- Require explicit `NULL` semantics for nullable values.
- Do not accept manual temporal strings in the stable editor. Temporal columns may use database-side `CURRENT_TIMESTAMP`, `CURRENT_DATE`, or `CURRENT_TIME`.
- Validate mutation values server-side and fail closed for unknown, generated, primary-key, unsupported, or invalid values.
- Block system schemas; SQLite exposes only its `main` schema through DB Studio.
- Quote identifiers through driver-specific adapters after validating them against introspection results.
- Audit successful write actions.

Row search is intentionally simple: one server-side string query scoped to the currently selected schema/table. Search updates are debounced by the dashboard; no apply/reset workflow or enum-filter toolbar is part of the stable contract.

This feature may use dynamic SQL only inside `internal/dbstudio` adapters because it targets user project databases with dynamic schemas. MyPaas application database queries remain sqlc-managed.

SQLite has a different runtime contract from server databases. DB Studio accepts SQLite only when the resolved database file is inside a persistent mount of a container-backed project. A file that exists only in the container writable layer is rejected because it is not durable across recreate/redeploy. The control-plane API does not mount container-engine storage directly; SQLite operations run through a short-lived, network-disabled helper container that shares the project's existing mounts.

## Consequences

- The dashboard remains useful for inspection and cautious production-data correction without becoming a full database IDE.
- Stable DB Studio writes are intentionally update-only and fail closed when mutation semantics are ambiguous.
- Insert/delete, raw SQL, complex datatype editors, and broader database-client features can be reconsidered after explicit qualification instead of being exposed speculatively.
- Compose server-database access requires MyPaas API to reach the project Compose network; the service can connect the API container to the project default network when needed.
- SQLite projects must place the database under a persistent runtime mount. An explicit supported database environment variable remains authoritative; when it is absent, DB Studio may discover a bounded, positively identified SQLite candidate through the isolated helper.
- SQL Server, Oracle, MongoDB, and Redis remain out of scope for this ADR.
