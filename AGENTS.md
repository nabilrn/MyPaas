# AGENTS.md — MyPaaS

Persistent engineering context for agents working in this repository. Keep this file concise, implementation-grounded, and consistent with current `main`.

## Product identity

MyPaaS is a **single-host self-hosted PaaS** for an owner developer or a small trusted team.

Current deployment sources/modes:

- Git repository -> Dockerfile;
- Git repository -> Docker Compose;
- Git repository -> static deployment;
- OCI registry -> image deployment.

Current platform capabilities include:

- repository inspection, base-directory/monorepo configuration, and Compose Doctor;
- encrypted project environment variables and resource settings;
- lifecycle actions, deployment history, logs, metrics, rollback, and cleanup;
- Caddy routing and route reconciliation;
- project-scoped persistent storage;
- optional shared PostgreSQL and DB Studio Lite for PostgreSQL/MySQL/MariaDB;
- backup/restore/migration tooling;
- CLI, REST API, webhooks, audit logs, and optional local MCP bridge;
- OSS templates + compatibility catalog;
- optional `mypaas-statd` telemetry;
- rootful Podman by default on fresh supported Linux hosts, with Docker Engine as compatibility mode.

## Source of truth

When files disagree, use this order:

1. current code, database schema/migrations, tests, installers, and production configuration;
2. `README.md`, `PRODUCT.md`, and current files under `docs/architecture/` / `docs/SECURITY_BOUNDARIES.md`;
3. accepted ADRs;
4. `ROADMAP.md`;
5. historical requirements/release notes.

`docs/PRD.md` is a **historical baseline**. It is not the current runtime source of truth.

Files under `docs/releases/` describe their named historical release and must not override current `main` behavior.

## Product boundaries

Do not introduce or claim these without an explicit product-direction change:

- Kubernetes, Nomad, Swarm, service mesh, distributed scheduling, or multi-node orchestration;
- control-plane HA;
- automatic horizontal scaling;
- hostile multi-tenant isolation;
- generic raw TCP/SSH/UDP routing;
- arbitrary public host-port forwarding;
- universal project-count, concurrent-user, RPS, or VM-size capacity guarantees;
- broad kernel/sysctl/NIC tuning programs;
- repeated throughput/k6 matrices without a concrete product defect;
- speculative framework/platform support that is not driven by a real application gap.

Compatibility work answers: **can MyPaaS correctly host this declared application pattern within the documented single-host boundary?** It is not a capacity benchmark.

## Container runtime contract

Fresh supported Linux installs are Podman-first.

The backend intentionally keeps a Docker-compatible orchestration surface:

```text
MyPaaS backend
  -> docker / docker compose command contract
  -> Docker-compatible socket
  -> rootful Podman (default) or Docker Engine (compatibility)
```

Production maps the selected host engine socket into the API container at the stable path:

```text
/var/run/docker.sock
```

Do not create separate Docker and Podman orchestration backends unless a demonstrated incompatibility requires it.

Treat engine-socket access as host authority.

## Network and routing contract

Production uses three distinct networks:

- `CONTROL_NETWORK` — API/dashboard/cloudflared/PostgreSQL/Caddy;
- `PROJECT_NETWORK` — project workloads + optional shared PostgreSQL;
- `ROUTING_NETWORK` — Caddy + explicitly routed workloads/services.

Security invariants:

- API is not a project/routing-network member;
- Caddy is not a general project-network member;
- project workloads never receive the engine socket;
- project workloads never receive the Caddy Admin Unix socket;
- production Caddy Admin uses `/run/mypaas/caddy-admin.sock`, not published TCP `2019`;
- route resolution fails closed.

### Primary routes

A normal container-backed project uses its allocated host port as a runtime lookup/identity key. Production Caddy traffic uses a managed `ROUTING_NETWORK` alias + internal container port.

### Additional Compose HTTP routes

ADR-023 is delivered and real-VM-qualified.

A Compose project may declare up to four additional routes:

```text
<project>-<route>.<PUBLIC_DOMAIN>
```

Rules:

- Compose only;
- HTTP(S) through Caddy only;
- target must be an existing Compose service and a TCP port declared by `ports` or `expose`;
- no additional host-port allocation/publication;
- no arbitrary route hostname;
- no raw TCP/SSH/UDP;
- route contract immutable after first deployment;
- additional routes must be synchronously reconciled before initial Compose deployment is marked successful;
- lifecycle and periodic reconciliation must preserve/remove routes according to project state.

MinIO `9000` primary + `9001` Console is the first qualified pattern.

## Registry contract

OCI image-mode deployment supports:

- anonymous registry pulls;
- one optional installation-level credential for a configured registry host (ADR-022).

Authenticated pulls use an isolated temporary Docker configuration and must not modify the operator's persistent Docker credentials.

Do not claim or silently implement:

- general per-project/per-registry credential management;
- credential inheritance into Compose services;
- registry proxy/cache/mirror behavior.

## Compose trust boundary

Repository Compose is untrusted input. Preserve the existing render/sanitize/validate path.

Known rejected host-bypass classes include:

- privileged mode;
- host/container namespace sharing;
- engine socket mounts;
- host bind mounts;
- devices/GPUs;
- added Linux capabilities;
- custom runtimes;
- external networks/volumes;
- unsafe build entitlements;
- build SSH/secrets;
- privileged lifecycle hooks.

Safe engine-managed named volumes are allowed where current sanitization permits them.

Do not pass arbitrary control-plane environment variables into Compose subprocesses.

## Static deployment

Static deployment is a first-class mode. Static projects are published atomically and served directly by Caddy without a persistent application container.

Do not force static sites into Node/Nginx containers merely to match a container-centric pattern.

## Persistence and cleanup

Important host state includes:

```text
/var/lib/mypaas/volumes
/var/lib/mypaas/compose
/var/lib/mypaas/static
/var/lib/mypaas/backups
```

Project deletion/cleanup must remove only owned resources and preserve documented recovery boundaries.

Do not assume engine-managed volume state is portable between Docker and Podman; migration preflight owns that decision.

## Development discipline

- Read the existing implementation before proposing architecture.
- Prefer small reusable platform primitives over application-specific patches.
- Fix real defects narrowly.
- Do not redesign unrelated UI while fixing backend/runtime behavior.
- Do not create a second deployment engine for templates/compatibility fixtures.
- Keep functions/components consistent with existing package/component conventions.
- Do not introduce a new dependency without a clear need.
- Never log secrets, JWTs, OAuth tokens, registry passwords, webhook secrets, or decrypted environment values.
- Keep errors explicit and wrapped with operational context.
- Preserve context cancellation on I/O and subprocess paths.

## Testing rule

Run tests proportional to the change.

For source changes, use the relevant backend/frontend/script/Compose/Podman gates already present in the repository. Re-run a real VM qualification only when the change materially touches the behavior that qualification covers.

Do **not** repeat unrelated runtime matrices merely to preserve a historical gate count.

When an OSS compatibility run fails:

1. classify the failure;
2. determine whether it is a MyPaaS defect, application/config issue, host-resource limit, or intentional boundary;
3. fix only platform-owned reusable defects/gaps;
4. rerun the affected path.

## Branching

Use narrow domain branches:

- `core/`
- `infra/`
- `ux/`
- `test/`
- `docs/`
- `chore/`
- narrow `fix/`

One branch should represent one domain + one outcome. Delete merged branches when practical.

## Common commands

```bash
# backend
cd backend
go test ./...
go test -race ./...
go build ./cmd/api

# frontend
cd frontend
pnpm install
pnpm check
pnpm test
pnpm build

# repository
make test
make build
```

Use the repository's existing scripts/workflows for production Compose validation, compatibility checks, and Podman compatibility rather than inventing parallel harnesses.

## Documentation rule

When a feature changes current product behavior, update the smallest relevant set of:

- `README.md`;
- `PRODUCT.md`;
- `ROADMAP.md` when product direction/delivery state changes;
- current architecture/security docs;
- the accepted ADR;
- compatibility docs when the capability affects workload support;
- `CHANGELOG.md`.

Do not rewrite historical PRD/release records to pretend they were always current.

## Current direction

The core beta feature target is implemented. New work should primarily come from real OSS application qualification, actual user/operator friction, or reproducible defects.

Do not create new roadmap work just to keep development active.
