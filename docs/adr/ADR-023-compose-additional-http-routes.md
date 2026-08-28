# ADR-023: Bounded additional HTTP routes for Compose projects

**Status:** Accepted and qualified  
**Date:** 2026-08-27  
**Qualified:** 2026-08-28

## Context

Some self-hosted applications expose more than one HTTP surface. MinIO makes the gap concrete: the S3 API normally listens on port `9000` while the web Console listens on `9001`.

MyPaaS historically gave each project one primary public hostname and one selected application port. Solving the MinIO case with arbitrary host-port publishing would weaken the existing routing and ownership model. A generic TCP proxy would also create a much larger security/lifecycle contract than the product needs.

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

Before persisting the contract, MyPaaS re-reads the project's persisted repository, branch, base directory, and Compose configuration. The resolved Compose model must prove that:

1. the requested service exists;
2. the target port is explicitly declared by that service through `ports` or `expose`;
3. the route does not duplicate the primary `mainService:appPort` target.

Client-side template metadata is not trusted as runtime authority.

## Data plane

Additional routes do not allocate extra host ports.

Caddy reaches workloads through `ROUTING_NETWORK` and internal container ports. If the target Compose container already has a usable MyPaaS-managed routing attachment, the additional route reuses it. This is required for applications such as MinIO where both public HTTP surfaces terminate on the same container.

If another eligible Compose service is not yet attached to `ROUTING_NETWORK`, MyPaaS attaches that service with a deterministic managed HTTP-route alias. A target that cannot be safely resolved or attached fails closed.

## Lifecycle

Additional routes are reconciled from persisted project state alongside the primary route lifecycle.

- initial Compose deployment reconciles required additional routes **before** the deployment is marked successful;
- start/restart/redeploy reconcile declared routes synchronously;
- running Compose projects should have all declared additional routes present;
- stopped/non-running projects should not retain active public routes;
- API/Caddy interruption can be repaired by periodic reconciliation;
- project deletion removes additional routes and persisted route configuration after cleanup.

The first version intentionally keeps the route contract immutable after first deployment. Mutable route contracts require explicit rollback/lifecycle semantics and should be added only if a real application demonstrates the need.

## Product integration

The first product template using this primitive is MinIO:

- primary route -> `minio:9000` for the S3 API;
- `console` route -> `minio:9001` for the web Console;
- `MINIO_BROWSER_REDIRECT_URL` is derived from the managed Console hostname;
- root credentials are generated through the existing template secret flow.

## Real-VM qualification

The candidate was qualified on VM `172.104.61.180` using exact head:

```text
b35176fd0156c8128e988a2ce3a46693a150c61d
```

All required gates passed before PR #157 merged:

- production deployment automatically used the host Podman socket through the stable in-container `/var/run/docker.sock` mount;
- primary MinIO health returned HTTP `200`;
- Console returned HTTP `200` and its hostname was present in Caddy configuration;
- Console port `9001` was not published as an additional host port;
- restart preserved primary and Console routes;
- redeploy preserved primary and Console routes;
- deleting the Console Caddy route followed by reconciliation recreated it;
- stop removed both public routes;
- delete removed routes, container, volume, and network cleanly.

Qualification evidence was recorded as:

```text
artifacts/pr157-minio-qualification-090944.json
```

The evidence was attached to the PR conversation; generated VM evidence is not required to remain committed in the source tree.

PR #157 then merged to `main` as merge commit:

```text
e12f47dd3249e2fdd69df352852ff3c9c3489245
```

## Consequences

This solves multi-HTTP-surface applications without creating a generic port-forwarding product.

It deliberately does **not** solve:

- Forgejo SSH on port `22`;
- databases exposed directly to the Internet;
- game-server UDP;
- arbitrary TCP services;
- arbitrary custom route domains;
- generic public host-port exposure.

Those remain outside the current MyPaaS routing contract.
