# ADR-023: Bounded additional HTTP routes for Compose projects

**Status:** Accepted candidate; requires VM qualification before merge  
**Date:** 2026-08-27

## Context

Some self-hosted applications expose more than one HTTP surface. MinIO is the first catalogued workload that makes the gap concrete: the S3 API normally listens on port `9000` while the web Console listens on `9001`.

MyPaaS currently gives each project one primary public hostname and one selected application port. Solving the MinIO case with arbitrary host-port publishing would weaken the existing Caddy ownership and routing-network boundary. A generic TCP proxy would also create a much larger security and lifecycle contract than the product currently needs.

## Decision

MyPaaS supports a bounded set of **additional HTTP routes** for Docker Compose projects.

Each route contains:

- a short route name;
- an existing Compose service name;
- an internal TCP container port declared by that service through Compose `ports` or `expose`.

The public hostname is platform-derived:

```text
<project>-<route>.<PUBLIC_DOMAIN>
```

The primary project route remains:

```text
<project>.<PUBLIC_DOMAIN>
```

Additional routes have these hard boundaries:

- Compose projects only;
- HTTP(S) through Caddy only;
- maximum four additional routes per project;
- no arbitrary public hostname;
- no additional host-port binding;
- no raw TCP forwarding;
- no SSH routing;
- no UDP routing;
- no project access to the Caddy Admin socket;
- route contract is immutable after the first deployment.

## Route validation

Before persisting the contract, MyPaaS re-reads the project's persisted repository, branch, base directory, and Compose file. The resolved Compose configuration must prove that:

1. the requested service exists;
2. the target port is explicitly declared by that service through `ports` or `expose`;
3. the route does not duplicate the primary `mainService:appPort` target.

Client-side template metadata is therefore not trusted as runtime authority.

## Data plane

Additional routes do not allocate extra host ports.

Caddy continues to reach workloads through `ROUTING_NETWORK`. If the target Compose container already has a MyPaaS-managed routing alias, the additional route reuses it. This is required for applications such as MinIO where both public HTTP surfaces terminate on the same container: adding the Console route must not disconnect or replace the primary S3 API alias.

If a non-primary Compose service is not yet attached to `ROUTING_NETWORK`, MyPaaS attaches that single service with a deterministic `mypaas-http-*` alias. A container that is already attached without any MyPaaS-managed alias fails closed rather than being disconnected and rewritten.

## Lifecycle

Additional routes are reconciled from persisted project state alongside the primary Caddy route reconciliation loop.

- `running` Compose project: declared additional routes are present;
- stopped/non-running project: declared additional routes are removed from Caddy;
- API/Caddy restart: reconciliation recreates missing routes from persisted state;
- deleted project: routes are removed and the persisted route list is cleared after cleanup.

The first version intentionally keeps the route contract immutable after first deploy. Mutable routing can be added later only if a real workload requires it and lifecycle rollback semantics are explicit.

## Product integration

The first product template using this primitive is MinIO:

- primary route -> `minio:9000` for the S3 API;
- `console` route -> `minio:9001` for the web Console;
- `MINIO_BROWSER_REDIRECT_URL` is derived from the managed Console hostname;
- root credentials are generated through the existing template secret flow.

## Consequences

This solves multi-HTTP-surface applications without creating a generic port-forwarding product. It deliberately does not solve Forgejo SSH, databases exposed directly to the Internet, game-server UDP, arbitrary TCP services, or custom per-route domains.

A code-side pass is insufficient to declare the feature qualified. A real VM must still prove primary/additional routing, lifecycle reconciliation, absence of extra host-port exposure, and cleanup before this ADR loses its candidate qualifier.
