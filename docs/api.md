# REST API

The Go API is the orchestration authority behind the MyPaaS dashboard, CLI, webhooks, and local MCP bridge.

This document is a route-family reference for operators and integrators. It is **not** a generated OpenAPI specification. The current route registration in `backend/cmd/api/main.go`, handler authorization, schema, and tests are authoritative.

## Base URL

A normal public installation exposes API routes below:

```text
https://<your-domain>/api
```

The server also registers the same application route set at its internal root for control-plane compatibility. External integrations should use the configured public `/api` path rather than depending on internal container addressing.

## Health and metrics

Public control-plane health surfaces are:

```text
GET /api/health
GET /api/ready
GET /api/metrics
```

`/api/health` reports process liveness. `/api/ready` checks control-plane PostgreSQL reachability. Public `/api/metrics` is Prometheus-compatible and requires configured Basic Auth when metrics are enabled.

Inside the API/control-network address space the same handlers are also registered at `/health`, `/ready`, and `/metrics`. External clients should not depend on those internal-root paths because production Caddy exposes the API through `/api/*`.

The production verifier checks these surfaces through the intended control-plane paths; see [Production verification](operations/production-verification.md).

## Authentication

MyPaaS uses different credential paths for different clients:

- the dashboard authenticates through GitHub OAuth and the normal browser session;
- API automation endpoints protected by the auth middleware accept the platform's authenticated session/Bearer JWT contract;
- the local stdio MCP bridge uses its dedicated `MYPAAS_API_TOKEN` configured through Administration → MCP;
- `/api/metrics` uses its own Basic Auth credential when configured.

These credential types are not interchangeable.

GitHub OAuth route family:

```text
/api/auth/github/login
/api/auth/github/callback
/api/auth/refresh
/api/auth/me
/api/auth/github/repositories
/api/auth/logout
```

## Project route family

Authenticated project operations include:

```text
/api/projects
/api/projects/detect-mode
/api/projects/detect-compose
/api/projects/{id}
/api/projects/{id}/routes
/api/projects/{id}/deploy
/api/projects/{id}/start
/api/projects/{id}/stop
/api/projects/{id}/restart
/api/projects/{id}/deployments
/api/projects/{id}/logs
/api/projects/{id}/stream
/api/projects/{id}/metrics
/api/projects/{id}/analytics
/api/projects/{id}/env
/api/projects/{id}/compose-resources
```

Project webhook support includes the project webhook endpoint, secret regeneration, and webhook status. Treat webhook secrets as credentials and never expose them in logs or public issue reports.

## Deployment route family

Deployment records and rollback are exposed under:

```text
/api/deployments/{id}
/api/deployments/{id}/rollback
```

A rollback is an explicit deployment action and remains subject to the current deployment/runtime safety contract.

## DB Studio route family

Owner-only DB Studio operations live under:

```text
/api/projects/{id}/db/...
```

The registered route set includes status, schemas, tables, columns, rows, and temporary write-session endpoints.

**Route registration does not imply every HTTP mutation is supported.** The stable contract is intentionally narrower:

- browsing is read-only by default;
- a temporary write session is required for supported writes;
- row update requires a primary key and validated values;
- row insert is disabled;
- row delete is disabled;
- raw SQL is not exposed.

The insert/delete handlers deliberately reject those operations even though compatibility routes exist. See [ADR-015](adr/ADR-015-db-studio-lite.md).

## Owner administration route family

Owner-protected operations include route families for:

- users;
- audit logs;
- settings and host statistics;
- Cloudflare and S3 configuration;
- MCP token regeneration;
- VM migration preparation/status;
- system update requests;
- backup trigger/download;
- host container inventory;
- short-lived host shell sessions.

The current host container inventory is **read-only**. A compatibility DELETE route exists but its handler rejects deletion; workload lifecycle belongs to project-scoped operations instead.

The owner shell is host authority for trusted operators, not a project terminal or generic public SSH endpoint.

## GitHub webhook

Project webhook delivery is accepted through:

```text
/api/webhook/{projectId}
```

Webhook authentication/validation remains handler-controlled. Do not expose webhook secrets in client-side code or public reports.

## CORS and public origin

Production CORS is constrained to configured MyPaaS origins. Development additionally allows the local dashboard origins used by the repository workflow.

Do not design integrations around arbitrary cross-origin browser access; use a trusted server/agent workflow when browser CORS does not match the configured installation origin.

## Errors and compatibility

Clients should rely on HTTP status plus the structured API error code/message returned by current handlers rather than parsing human-readable logs.

Some registered compatibility routes intentionally return `405 Method Not Allowed` because the stable product contract is narrower than the historical route shape. Examples include DB Studio insert/delete and host-container deletion.

## Safety boundary

The API has Docker-compatible engine authority and therefore participates in the host trust boundary. Do not expose internal API/container addresses directly to untrusted networks, and do not bypass the normal Caddy/public-delivery configuration to create an alternate privileged ingress.

See [Security boundaries](SECURITY_BOUNDARIES.md).

## Source reference

For exact current paths, methods, middleware, and handler wiring, inspect:

```text
backend/cmd/api/main.go
```

For integrations that need a more ergonomic automation surface, prefer the [local MCP bridge](mcp.md) where its bounded tool set fits the task.
