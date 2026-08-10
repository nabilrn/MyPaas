# MyPaas — Self-Hosted Deployment Platform

> Deploy Git repositories and public OCI images to your own Linux VM with managed builds, routing, environment configuration, logs, metrics, databases, backups, and recovery tooling.

MyPaas is a single-host self-hosted PaaS for an owner developer or a small trusted team. It aims to provide a Vercel/Railway-style deployment workflow while keeping control of the VM, container engine, persistent data, and network path.

The control plane is built with a Go API, SvelteKit dashboard, PostgreSQL, Caddy, and Cloudflare Tunnel. Container-backed workloads use a Docker-compatible CLI/API contract and can run on **Docker Engine or Podman**. Static projects are served directly by Caddy without a persistent application container.

> This README documents behavior implemented on `main`. Experimental branches and unmerged integrations are intentionally excluded.

---

## Architecture

```text
Internet
   |
Cloudflare Tunnel
   |
 Caddy
   |---------------------> SvelteKit dashboard
   |---------------------> Go API
   |                         |
   |                         +--> PostgreSQL control-plane database
   |                         +--> Caddy Admin API
   |                         +--> Docker-compatible CLI/socket
   |                                  |
   |                                  +--> Docker Engine
   |                                  `--> Podman socket compatibility
   |
   `----------------------> project routes
                              |--> container-backed applications
                              `--> static files under /var/lib/mypaas/static
```

MyPaas currently targets a **single Linux VM**, not a Kubernetes cluster. The runtime abstraction is intentionally small: the backend invokes the `docker` CLI and Docker-compatible socket contract even when Podman is the actual engine.

---

## Implemented Capabilities

### Deployment and routing

- **Git repository deployments** using Dockerfile, Docker Compose, or static mode.
- **Public OCI image deployments** from Docker Hub, GHCR, and compatible public registries.
- **Repository inspection** for branches, repository tree, runtime files, environment templates, ports, and Compose candidates.
- **Monorepo/base-directory support** for applications below the repository root.
- **Flexible Compose configuration** with non-root Compose files, ordered overrides, profiles, explicit working directories, selectable public/main service, and per-service resource overrides.
- **Compose preflight analysis** for services, ports, build contexts, required environment variables, and unsafe configuration patterns.
- **Static applications without a persistent runtime container**, including static SPA builds when a build script is available.
- **Hybrid projects** where a static frontend is served by Caddy while a container-backed service handles the backend.
- **Encrypted environment variables** using AES-256-GCM with nested template discovery, service attribution, conflict detection, paste/upload import, reveal, update, and delete flows.
- **GitHub webhook deployments** with HMAC verification, branch filtering, delivery logging, and rate limiting for Git-backed projects.
- **Automatic Caddy routing and route reconciliation** for projects whose desired state is running.

### Runtime operations

- Deployment history and build logs.
- Start, stop, and restart lifecycle actions.
- Realtime SSE events for status, metrics, logs, deployment state, and build logs.
- Per-service Compose logs and metrics.
- CPU and memory quotas, resource profiles, and custom limits.
- Historical rollback for successful Dockerfile, Compose, and registry-image deployments.
- Static recovery through redeploy/roll-forward rather than container rollback.
- Optional Cloudflare Analytics for request, bandwidth, error, and timeseries data.

### Data and platform operations

- **Optional shared PostgreSQL provisioning** with a project-specific database/user and encrypted `DATABASE_URL` injection.
- **DB Studio Lite** for PostgreSQL, MySQL, and MariaDB, including schema/table/column browsing, paginated/searchable rows, enum filters, and owner-only temporary write sessions.
- **Scheduled PostgreSQL backups** with daily/weekly retention.
- **Scoped cleanup of unused MyPaas-managed images**.
- **GitHub OAuth + user whitelist**, owner/collaborator roles, audit logs, and admin user management.
- **Prometheus-compatible API process metrics** plus `/health` and `/ready` endpoints.
- Host/resource settings for quotas, concurrent deploys, defaults, and build timeout.

### Operator and automation tooling

- **`mypaas` CLI** for configuration, users, project list/deploy/logs, and manual backups.
- **Local MCP bridge for AI agents** with repository inspection, project management, lifecycle, deployment, rollback, logs, metrics, environment, quota, and host-stat tools.
- **Opt-in automatic self-updates** through systemd with revision-pinned GHCR images and verification.
- **VM migration export/import** with storage safety preflight and runtime quiescing.

> Registry deployment currently targets **public images**. Private registry credential management is outside the current implementation.

---

## Deployment Modes

| Source | Mode | Behavior | Historical rollback |
| --- | --- | --- | --- |
| Git | `dockerfile` | Build the repository Dockerfile and run the resulting image | Yes |
| Git | `compose` | Resolve configured Compose files/profiles/workdir and manage the multi-service project | Yes |
| Git | `static` | Publish static output and serve it directly through Caddy | Redeploy target revision |
| Registry | `image` | Pull a public OCI image and run it through the normal resource/routing lifecycle | Yes |

Registry projects do not use Git webhooks because there is no Git source to watch.

### Runtime detection

Git repository detection follows the source rather than inventing a runtime:

1. discover and rank Compose files;
2. otherwise use a root Dockerfile when present;
3. otherwise detect a static site/static SPA;
4. use Nixpacks planning only as an additional **inspection signal** for provider/framework detection.

Nixpacks is **not a MyPaas deployment mode** on `main`. If a backend/SSR application has neither Compose nor a production Dockerfile, MyPaas rejects the deploy configuration and asks for a Dockerfile instead of silently generating an opaque runtime.

---

## Container Engine: Docker or Podman

MyPaas keeps a Docker-compatible control-plane contract. This is why environment variables and source types still use names such as `DOCKER_SOCKET` and `DockerCLI` even when Podman is the actual host engine.

### Docker Engine

Docker Engine remains the default bootstrap path:

```bash
curl -fsSL https://raw.githubusercontent.com/nabilrn/MyPaas/main/scripts/bootstrap.sh | bash
```

### Podman Engine

For a fresh Ubuntu/Debian VM using Podman:

```bash
curl -fsSL https://raw.githubusercontent.com/nabilrn/MyPaas/main/scripts/bootstrap.sh | env USE_PODMAN=true bash
```

Or from an existing clean checkout:

```bash
bash scripts/install-vm.sh --podman
```

Podman mode intentionally keeps the Docker CLI and Compose plugin as compatibility clients. The installer enables `podman.socket` and points `/var/run/docker.sock` at `/run/podman/podman.sock`. Commands such as `docker compose`, `docker inspect`, and `docker stats` can therefore still appear while **Podman is the actual engine**.

### In-place Docker → Podman migration

In-place engine replacement is **not supported**. Docker Engine and Podman keep engine-local containers and named volumes in different storage, so changing the socket is not a safe state migration.

`scripts/migrate-to-podman.sh` is retained only as a fail-closed compatibility stub and refuses normal execution.

For a stateful installation, use the supported flow:

1. prepare a VM migration package from **Admin → Settings → VM Migration**;
2. provision a fresh VM with `USE_PODMAN=true`;
3. restore the migration package on the new VM.

---

## Quick Start

Automatic dependency installation currently targets fresh **Ubuntu/Debian** hosts.

### Docker Engine

```bash
curl -fsSL https://raw.githubusercontent.com/nabilrn/MyPaas/main/scripts/bootstrap.sh | bash
```

### Podman Engine

```bash
curl -fsSL https://raw.githubusercontent.com/nabilrn/MyPaas/main/scripts/bootstrap.sh | env USE_PODMAN=true bash
```

The bootstrap flow:

1. installs Git when required;
2. checks out `main` into `~/MyPaas`;
3. installs the selected container-engine dependencies and Nixpacks CLI when required;
4. starts the browser setup wizard by default;
5. writes the production `.env`;
6. prepares persistent host directories;
7. creates the project network;
8. starts PostgreSQL and runs migrations;
9. starts the production Compose stack through the Docker-compatible command surface;
10. configures optional self-updates and leaves verification available through `scripts/verify-production.sh`.

The setup wizard binds to `127.0.0.1`. It can be exposed temporarily through a token-protected Cloudflare Quick Tunnel. SSH forwarding remains the fallback:

```bash
ssh -L 8787:127.0.0.1:8787 <user>@<vm-ip>
```

### Non-interactive install

```bash
curl -fsSL https://raw.githubusercontent.com/nabilrn/MyPaas/main/scripts/bootstrap.sh | env \
  INSTALL_WIZARD=false \
  PUBLIC_DOMAIN=mypaas.example.com \
  OWNER_EMAIL=you@example.com \
  GITHUB_CLIENT_ID=your_client_id \
  GITHUB_CLIENT_SECRET=your_client_secret \
  CLOUDFLARE_TUNNEL_TOKEN=your_tunnel_token \
  bash
```

Add `USE_PODMAN=true` when provisioning with Podman.

### Existing installation / manual upgrade

```bash
curl -fsSL https://raw.githubusercontent.com/nabilrn/MyPaas/main/scripts/bootstrap.sh | bash
```

Installer-managed checkouts must be clean. Bootstrap fetches the configured upstream ref and resets the managed checkout to the fetched revision.

---

## Production Control Plane

`docker-compose.prod.yml` runs five control-plane services:

- `postgres` — system database and optional shared project databases;
- `api` — Go API, deployment workers, CLI, backup/image-cleanup jobs, and route reconciliation;
- `dashboard` — SvelteKit dashboard;
- `caddy` — dashboard/API/project routing and static serving;
- `cloudflared` — outbound Cloudflare Tunnel client.

Important host-managed paths include:

```text
/var/lib/mypaas/volumes
/var/lib/mypaas/compose
/var/lib/mypaas/static
/var/lib/mypaas/backups
/tmp/mypaas/builds
```

The control-plane PostgreSQL data itself uses the container engine's `postgres_data` named volume. Do not treat `~/MyPaas` as the persistent-data directory.

---

## Git, Compose, and Environment Detection

Before project creation, MyPaas can inspect the selected Git branch and show the repository tree. Supported source configuration includes:

- repository root or `baseDirectory`;
- ranked Compose candidates;
- `composeFilePath`;
- ordered `composeOverridePaths`;
- `composeProfiles`;
- `composeWorkdir`;
- public/main Compose service;
- application port;
- optional static frontend path;
- project and per-service resource limits.

Environment discovery scans common templates including:

```text
.env.example
.env.sample
.env.template
.env.local.example
```

For Compose/monorepo repositories, MyPaas can discover nested templates, attribute variables to services, identify conflicting defaults, and generate service-local `.env` files during deployment without overwriting a committed `.env` beside the template.

Persisted environment values are encrypted before storage.

---

## Container Registry Deployments

Registry projects use `sourceType=registry` and `deployMode=image`.

Examples:

```text
nginx:latest
ghcr.io/example/my-api:v1.4.0
ghcr.io/example/my-api@sha256:<digest>
```

MyPaas pulls the configured public image, applies environment/resource settings, maps the application port, records the resolved image identity when available, and routes it through Caddy.

Private-registry credential storage is not currently implemented.

---

## Database Features

### Shared PostgreSQL

When explicitly requested and globally enabled, MyPaas can provision a dedicated database and role inside the platform PostgreSQL service and persist the generated connection URL as the project's encrypted `DATABASE_URL`.

### DB Studio Lite

The owner-only Database view can discover PostgreSQL, MySQL, and MariaDB connections from project environment configuration.

Implemented operations include:

- connection status;
- schema/table/column browsing;
- paginated row browsing;
- server-side search and enum filters;
- temporary write sessions;
- guarded insert/update/delete;
- mutation audit records.

Write access is time-limited and must be explicitly enabled.

---

## Logs, Metrics, and Observability

Project operations expose runtime CPU/memory snapshots, uptime, per-service Compose metrics, recent logs, live SSE logs/events, deployment/build logs, and optional Cloudflare request/bandwidth/error analytics.

The control-plane API exposes:

```text
/health
/ready
/metrics
```

`/metrics` is Prometheus-compatible. In production, metrics Basic Auth credentials must be configured and valid credentials are required to read the endpoint.

---

## Backups and Cleanup

The API starts a background scheduler when backup and/or image cleanup is enabled.

Default production configuration includes:

```text
BACKUP_ENABLED=true
BACKUP_DIR=/var/lib/mypaas/backups
BACKUP_DAILY_AT=02:00
BACKUP_KEEP_DAILY=7
BACKUP_KEEP_WEEKLY=4
IMAGE_CLEANUP_ENABLED=true
IMAGE_CLEANUP_UNTIL=168h
```

Backups use PostgreSQL custom-format dumps and maintain daily/weekly retention sets. Image cleanup is scoped to unused MyPaas-managed images.

To include a manual backup check in production verification:

```bash
RUN_BACKUP=true bash scripts/verify-production.sh
```

---

## VM Migration Safety

The supported exporter lives in **Admin → Settings → VM Migration**. The older standalone `scripts/migrate-export.sh` is retired and refuses normal execution so migration safety logic cannot drift between two implementations.

Before any runtime is stopped, the backend performs a storage preflight.

### Supported state

The export contains:

- the MyPaas system PostgreSQL database;
- shared `mypaas_p_*` databases and roles when available;
- the production `.env` when accessible;
- `/var/lib/mypaas/volumes`;
- `/var/lib/mypaas/compose`;
- `/var/lib/mypaas/static`;
- an export manifest.

For running container-backed projects, the exporter:

1. performs storage preflight;
2. stops existing runtimes **without changing their desired project status in PostgreSQL**;
3. creates logical database dumps and the filesystem archive;
4. restarts every runtime it stopped;
5. marks the migration package `ready` only after restoration succeeds.

Static projects have no application runtime to stop. If quiescing fails partway through, already-stopped runtimes are restored before the export fails. Export failures also trigger a deferred runtime restoration attempt.

The resulting download token expires after **24 hours**. A new VM can import the package through:

```bash
bash scripts/install-vm.sh --migrate-url <migration-url>
```

### Engine-managed Compose volumes

MyPaas deliberately **fails closed** when a project Compose container has an engine-managed named/external volume. Those volumes do not live in the host-managed MyPaas directory tree and cannot be safely assumed portable between Docker Engine and Podman.

The exporter reports the affected volume names and stops before creating downtime. Move persistent data to bind mounts under `/var/lib/mypaas/volumes` or migrate the listed engine volumes separately before retrying.

See [ADR-019](docs/adr/ADR-019-migration-safety-boundaries.md) for the safety rationale.

---

## CLI

The production API image includes `/app/mypaas`; local builds create `backend/bin/mypaas`.

```bash
make build-backend

backend/bin/mypaas config set api-url https://mypaas.example.com
backend/bin/mypaas config set token <token>
backend/bin/mypaas project list
backend/bin/mypaas project deploy <name>
backend/bin/mypaas project logs <name>
backend/bin/mypaas user list
backend/bin/mypaas backup
```

Run `mypaas help` for the current command surface.

---

## AI Agent Integration (MCP)

MyPaas includes a stdio MCP server under `backend/cmd/mcp`. It is intended to run on the developer's local machine and call the remote MyPaas API using an API token generated from **Admin → Settings**.

```bash
cd backend
MYPAAS_URL=https://mypaas.example.com/api \
MYPAAS_API_TOKEN=<token> \
go run ./cmd/mcp
```

The MCP server exposes repository inspection, project create/update, lifecycle, deployment/rollback, logs, metrics, environment, quota, and host-stat tools.

Keep the MCP token private. Regenerating it invalidates clients using the previous token.

---

## Automatic Self-Updates

Automatic updates are **opt-in**:

```bash
cd ~/MyPaas
AUTO_UPDATE_ENABLED=true \
AUTO_UPDATE_INTERVAL_MINUTES=30 \
bash scripts/configure-auto-update.sh
```

The updater tracks a Git ref, waits for API and dashboard GHCR images tagged with the matching commit SHA, then deploys matching source/configuration and images together instead of blindly following `latest`.

Useful commands:

```bash
bash scripts/update-vm.sh
systemctl status mypaas-update.timer
journalctl -u mypaas-update.service
AUTO_UPDATE_ENABLED=false bash scripts/configure-auto-update.sh
```

Dirty managed checkouts are rejected. Runtime rollback is best-effort; database migrations can still be forward-only, so backups remain necessary.

---

## Cloudflare Requirements

For public dashboard and project routes:

- add the MyPaas domain to Cloudflare DNS;
- configure a Cloudflare Tunnel;
- route the root/dashboard hostname and wildcard project hostname to Caddy (`HTTP` → `caddy:80`);
- create proxied root and wildcard DNS records pointing to the tunnel hostname.

A registrar transfer is not required. Cloudflare Tunnel credentials are separate from the optional Cloudflare Analytics API token/zone settings.

---

## Production Operations

Production scripts use the `docker`/`docker compose` command surface. On a Podman installation those commands are compatibility clients talking to `podman.socket`.

```bash
# Deploy current checkout
bash scripts/deploy-to-vm.sh

# Verify containers, API readiness, Caddy Admin API, and CLI
bash scripts/verify-production.sh

# Verify a manual PostgreSQL backup too
RUN_BACKUP=true bash scripts/verify-production.sh

# Inspect control plane
docker compose -f docker-compose.prod.yml ps
docker compose -f docker-compose.prod.yml logs -f
```

Useful installer options:

```bash
SKIP_DEPLOY=true bash scripts/install-vm.sh
FORCE_ENV=true bash scripts/install-vm.sh
SKIP_DOCKER_INSTALL=true bash scripts/install-vm.sh
INSTALL_WIZARD=true bash scripts/install-vm.sh
WIZARD_PUBLIC_TUNNEL=false INSTALL_WIZARD=true bash scripts/install-vm.sh
bash scripts/install-vm.sh --podman
bash scripts/install-vm.sh --migrate-url <migration-url>
```

---

## Local Development

### Prerequisites

- **Go 1.25.5** — matches `backend/go.mod`.
- **Node.js 22** — matches repository CI.
- **pnpm 10.22.0** — declared by the frontend package.
- **PostgreSQL 16**.
- **Docker CLI + Compose plugin** with Docker Engine or a compatible Podman socket for container-backed workflows.
- **Caddy 2** when testing routing outside the development Compose stack.
- **Nixpacks CLI** when exercising repository-detection provider inspection.

The Makefile currently invokes the command name `docker`, so Podman-based local development must expose the same compatibility command/socket arrangement used by the VM installer.

### Setup

```bash
git clone https://github.com/nabilrn/MyPaas.git
cd MyPaas
cp .env.example .env
make dev
```

Run API and dashboard separately:

```bash
make backend-dev
make frontend-dev
```

### Tests and build

```bash
make test
make lint
make build
```

Useful targets:

```bash
make test-backend
make test-frontend
make test-coverage
make migrate-up
make migrate-down
make sqlc
make verify-prod
make help
```

Pull requests are checked by GitHub Actions with backend tests, frontend unit/type/build checks, Bash syntax, all `scripts/*_test.py` regressions, and production Compose rendering.

---

## Project Structure

```text
MyPaas/
├── backend/
│   ├── cmd/
│   │   ├── api/                Go HTTP API
│   │   ├── cli/                mypaas CLI
│   │   └── mcp/                local stdio MCP bridge
│   ├── internal/               deployment, container, migration, DB Studio, auth, etc.
│   ├── migrations/             PostgreSQL schema migrations
│   └── query/                  sqlc queries
├── frontend/                   SvelteKit dashboard
├── docs/                       PRD, architecture, ADRs, and audits
├── scripts/                    bootstrap, install, deploy, verify, update, migration tooling
├── docker-compose.dev.yml      local dependencies
├── docker-compose.prod.yml     production control plane
├── Caddyfile.*                 routing configuration
└── Makefile                    development and operations targets
```

---

## Configuration

Start from `.env.example`; it is the authoritative environment-backed configuration reference.

```bash
# Application/database
DATABASE_URL=postgres://user:pass@localhost:5432/mypaas_dev
ENVIRONMENT=development

# GitHub OAuth
GITHUB_CLIENT_ID=your_id
GITHUB_CLIENT_SECRET=your_secret
OWNER_EMAIL=you@example.com

# Security
JWT_SECRET=your_256bit_secret
ENCRYPTION_KEY=your_base64_encoded_32byte_key

# Docker-compatible engine contract
DOCKER_SOCKET=/var/run/docker.sock
DOCKER_BIND_HOST=0.0.0.0
PROJECT_NETWORK=mypaas-dev

# Cloudflare Tunnel
CLOUDFLARE_TUNNEL_TOKEN=your_tunnel_token

# Shared PostgreSQL
SHARED_POSTGRES_ENABLED=true

# Backup/cleanup
BACKUP_ENABLED=true
IMAGE_CLEANUP_ENABLED=true
```

Do not commit production `.env`, generated tokens, database credentials, or encryption keys.

---

## Current Boundaries

MyPaas deliberately remains smaller than a general-purpose cloud platform:

- single-host VM control plane;
- no Kubernetes scheduler;
- no multi-node orchestration or autoscaling in `main`;
- no private-registry credential manager;
- Podman support is through the Docker-compatible CLI/socket contract rather than a separate native Podman backend;
- no supported in-place Docker Engine → Podman state migration;
- built-in VM export rejects engine-managed Compose named/external volumes rather than silently copying them;
- Nixpacks is an inspection aid, not an automatic backend/SSR buildpack deployment mode;
- static deployments use redeploy/roll-forward instead of historical container rollback.

These constraints are intentional unless a measured operational need justifies additional complexity.

---

## Documentation

- **[Product Requirements](docs/PRD.md)** — product scope and behavior.
- **[Architecture](docs/ARCHITECTURE.md)** — technical design and diagrams.
- **[Architecture Decisions](docs/adr/)** — recorded architectural decisions, including [ADR-019](docs/adr/ADR-019-migration-safety-boundaries.md).
- **[Product direction](PRODUCT.md)** — user, purpose, and UI principles.
- **[Engineering conventions](AGENTS.md)** — repository-wide constraints for agents/contributors.
- **[Changelog](CHANGELOG.md)** — notable implementation changes.

When documentation disagrees with executable behavior, treat current `main` code, schema, tests, and installer behavior as the source of truth and update the stale documentation.

---

## Contributing

1. Create a focused branch from the latest `main`.
2. Keep changes scoped where practical.
3. Add or update regression tests for behavior changes.
4. Run the relevant backend/frontend/script checks.
5. Open a pull request with implementation impact and validation results.
6. Keep documentation synchronized with executable behavior.

For production-sensitive changes, prefer explicit failure and reversible rollout paths over hidden fallback behavior.

---

## Troubleshooting

### Bootstrap reports a dirty checkout

Preserve, commit, or remove local changes before running bootstrap/update. Installer-managed checkouts intentionally refuse to overwrite local modifications.

### Verify Podman compatibility

```bash
systemctl status podman.socket --no-pager
readlink -f /var/run/docker.sock
docker info
```

On a Podman host, `/var/run/docker.sock` should resolve to the Podman API socket configured by the installer.

### VM migration reports engine-managed volumes

The exporter intentionally stops before downtime. Move the listed persistent volumes to supported bind mounts under `/var/lib/mypaas/volumes`, or migrate those engine volumes separately, then run the export again.

### Project domain does not resolve

Verify Cloudflare Tunnel public-hostname routes plus proxied root and wildcard DNS records.

### Production verification fails

```bash
docker compose -f docker-compose.prod.yml ps
docker compose -f docker-compose.prod.yml logs --tail=200
bash scripts/verify-production.sh
```

### Local development database needs a reset

```bash
make docker-reset
make migrate-up
```

---

## License

MIT — see [LICENSE](LICENSE).

---

## Getting Help

- **Bug reports:** open a GitHub issue with reproduction steps and relevant logs.
- **Feature requests:** open an issue describing the workflow and expected behavior.
- **Documentation:** start with this README, `docs/`, and the ADRs.
- **Security issues:** use the repository's private security contact/process instead of publishing sensitive details in a public issue.

---

**Implementation audit:** 2026-08-10
