# MyPaas — Self-Hosted Deployment Platform

> Deploy Git repositories and public OCI images to your own Linux VM with managed builds, routing, environment configuration, logs, metrics, databases, backups, and rollback.

MyPaas is a single-host self-hosted PaaS for an owner developer or a small trusted team. The goal is to provide a Vercel/Railway-style deployment workflow while keeping control of the VM, container engine, persistent data, and network path.

The control plane is a Go API, SvelteKit dashboard, PostgreSQL, Caddy, and Cloudflare Tunnel. Container-backed workloads use a Docker-compatible CLI/API contract and can run on **Docker Engine or Podman**. Static projects are served directly by Caddy and do not keep an application container running.

> This README documents behavior implemented on `main`. Experimental branches and unmerged integrations are intentionally excluded.

---

## Architecture at a Glance

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

- **Git repository deployments** with Dockerfile, Docker Compose, and static deployment modes.
- **Public OCI image deployments** from Docker Hub, GHCR, and compatible public registries.
- **Repository inspection** for remote branches, repository tree preview, runtime files, environment templates, ports, and Compose candidates.
- **Monorepo/base-directory support** for applications below the repository root.
- **Flexible Compose projects** with Compose files outside the repository root, chained override files, profiles, explicit working directories, selectable public/main service, and per-service resource overrides.
- **Compose preflight analysis** for service selection, ports, build contexts, environment requirements, and unsafe configuration patterns.
- **Static applications without a persistent runtime container**, including static SPA builds when a build script is present.
- **Hybrid projects** where a static frontend is served by Caddy while a container-backed service handles the application backend.
- **Encrypted environment variables** using AES-256-GCM, including nested `.env.example`/`.env.sample`/`.env.template` discovery, service attribution, conflict detection, paste/upload import, reveal, update, and delete flows.
- **GitHub webhook deployments** with HMAC verification, branch filtering, delivery logging, and rate limiting for Git-backed projects.
- **Automatic Caddy routing and route reconciliation** so running projects are restored after control-plane/Caddy restarts.

### Runtime operations

- **Deployment history** with build logs and current deployment state.
- **Start, stop, and restart** lifecycle actions for deployed projects.
- **Realtime SSE events** for project status, runtime metrics, logs, deployment state, and build logs.
- **Per-service Compose logs and metrics** with dashboard service selection/filtering.
- **CPU and memory controls** with user quotas, project resource profiles, and optional custom limits.
- **Rollback** for successful Dockerfile, Compose, and registry-image deployments.
- **Static recovery by redeploy/roll-forward** rather than the runtime rollback action used by container-backed modes.
- **Optional Cloudflare Analytics** for project request, bandwidth, error, and timeseries data when the API token and zone are configured from platform settings.

### Data and platform operations

- **Optional shared PostgreSQL provisioning** that creates a project-specific database/user and injects an encrypted `DATABASE_URL`.
- **DB Studio Lite** for PostgreSQL, MySQL, and MariaDB connections discovered from project environment configuration, including schemas, tables, columns, paginated/searchable rows, enum filters, and owner-only temporary write sessions for guarded row mutations.
- **Scheduled PostgreSQL backups** with daily/weekly retention.
- **Scoped cleanup of unused MyPaas-managed images** on a configurable schedule.
- **GitHub OAuth + user whitelist**, owner/collaborator roles, mutation audit logging, and admin user management.
- **Prometheus-compatible API process metrics**; in production, `/metrics` requires configured Basic Auth credentials. The API also exposes `/health` and `/ready`.
- **Host/resource settings** for quotas, concurrent deploys, defaults, and build timeout.

### Operator and automation tooling

- **`mypaas` CLI** for configuration, admin users, project list/deploy/logs, and manual backups.
- **Local MCP bridge for AI agents** with project inspection/create/update, lifecycle, deployments, rollback, logs, metrics, environment variables, quota, and host-stat tools.
- **Opt-in automatic self-updates** through systemd with revision-pinned GHCR images and best-effort runtime rollback.
- **VM migration export/import tooling** for the control-plane database, shared project databases, `.env`, and selected persistent MyPaas directories. Treat this as an operator workflow and validate it for workloads with actively changing persistent files before relying on it as a consistent live-volume snapshot.

> Container Registry deployment currently targets **public images**. Private registry credential management is outside the current implementation.

---

## Deployment Modes

| Source | Deploy mode | What MyPaas does | Historical rollback |
| --- | --- | --- | --- |
| Git | `dockerfile` | Builds the repository Dockerfile, starts a replacement container, switches Caddy, then retires the previous runtime | Yes |
| Git | `compose` | Resolves the configured Compose files/profiles/workdir, applies MyPaas routing/resource overrides, and manages the multi-service project | Yes |
| Git | `static` | Publishes static output to `/var/lib/mypaas/static` and serves it through Caddy without a persistent app container | Redeploy the target revision instead |
| Registry | `image` | Pulls a public OCI image and runs it through the normal MyPaas resource/routing lifecycle | Yes, using the recorded image reference/digest |

Registry projects do not use Git webhooks because there is no Git source to watch.

### Runtime detection

For Git projects, detection follows the repository contents rather than inventing a runtime:

1. discover and rank Compose files;
2. otherwise use a root Dockerfile when present;
3. otherwise detect an existing static site/static SPA;
4. use Nixpacks planning only as an additional **inspection signal** for provider/framework detection.

Nixpacks is **not a MyPaas deployment mode** on `main`. If a backend/SSR application has neither Compose nor a production Dockerfile, MyPaas rejects the deploy configuration and asks for a Dockerfile instead of silently generating an opaque runtime.

---

## Container Engine: Docker or Podman

MyPaas keeps a Docker-compatible control-plane contract. This is why environment variables and parts of the Go code still use names such as `DOCKER_SOCKET` and `DockerCLI` even when the host engine is Podman.

### Docker Engine

The normal bootstrap command currently installs/uses Docker Engine by default:

```bash
curl -fsSL https://raw.githubusercontent.com/nabilrn/MyPaas/main/scripts/bootstrap.sh | bash
```

### Podman Engine

For a new Ubuntu/Debian VM using Podman:

```bash
curl -fsSL https://raw.githubusercontent.com/nabilrn/MyPaas/main/scripts/bootstrap.sh | env USE_PODMAN=true bash
```

Or from an existing clone:

```bash
bash scripts/install-vm.sh --podman
```

Podman mode intentionally keeps the Docker CLI and Compose plugin as compatibility clients. The installer:

1. installs Podman;
2. enables `podman.socket`;
3. keeps `docker`/`docker compose` as the command surface expected by MyPaas;
4. points `/var/run/docker.sock` at `/run/podman/podman.sock`.

Therefore commands such as `docker compose`, `docker inspect`, and `docker stats` may still appear in operations and source code while **Podman is the actual engine**.

### Existing Docker host → Podman helper is destructive

`scripts/migrate-to-podman.sh` switches an existing host from Docker Engine to Podman, but it does **not** migrate Docker engine-local containers or named volumes. The production Compose stack stores the MyPaas PostgreSQL control-plane database in the `postgres_data` named volume, so this helper must not be treated as a safe in-place production migration.

For a stateful production installation, export/back up MyPaas and plan an explicit restore before changing engines. Prefer provisioning a fresh VM directly with Podman and using the migration/export workflow. Treat `scripts/migrate-to-podman.sh` as a disposable/development-host helper until the script explicitly preserves engine-local volumes.

For disposable hosts only:

```bash
cd ~/MyPaas
bash scripts/migrate-to-podman.sh
```

---

## Quick Start

### Production VM

Automatic dependency installation currently targets fresh **Ubuntu/Debian** hosts.

Docker Engine:

```bash
curl -fsSL https://raw.githubusercontent.com/nabilrn/MyPaas/main/scripts/bootstrap.sh | bash
```

Podman Engine:

```bash
curl -fsSL https://raw.githubusercontent.com/nabilrn/MyPaas/main/scripts/bootstrap.sh | env USE_PODMAN=true bash
```

The bootstrap flow:

1. installs Git when required;
2. checks out `main` into `~/MyPaas`;
3. installs the selected container-engine dependencies and Nixpacks CLI when required;
4. starts the browser setup wizard by default;
5. writes the production `.env`;
6. prepares persistent host directories under `/var/lib/mypaas` and `/tmp/mypaas/builds`;
7. creates the project network;
8. starts PostgreSQL and runs database migrations;
9. starts `docker-compose.prod.yml` using the Docker-compatible command surface;
10. configures the optional self-update service and leaves production verification available through `scripts/verify-production.sh`.

During browser setup, the wizard binds to `127.0.0.1`. It can be exposed temporarily through a token-protected Cloudflare Quick Tunnel. If that is unavailable, use SSH forwarding:

```bash
ssh -L 8787:127.0.0.1:8787 <user>@<vm-ip>
```

### Non-interactive production install

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

Add `USE_PODMAN=true` to the `env` arguments when provisioning the host with Podman.

### Existing installation / manual upgrade

The bootstrap command is also the supported manual checkout/update path:

```bash
curl -fsSL https://raw.githubusercontent.com/nabilrn/MyPaas/main/scripts/bootstrap.sh | bash
```

Installer-managed checkouts must be clean. Bootstrap fetches the configured upstream ref and resets the managed checkout to that fetched revision so rewritten/squashed upstream history does not require a local Git merge.

---

## Production Control Plane

`docker-compose.prod.yml` starts five control-plane services:

- `postgres` — MyPaas system database and optional shared project databases;
- `api` — Go API, deployment worker, CLI binary, scheduled backup/image-cleanup jobs, and route reconciliation;
- `dashboard` — SvelteKit dashboard;
- `caddy` — dashboard/API/project routing and static file serving;
- `cloudflared` — outbound Cloudflare Tunnel client.

Persistent application data lives outside the Git checkout. Important host paths include:

```text
/var/lib/mypaas/volumes
/var/lib/mypaas/compose
/var/lib/mypaas/static
/var/lib/mypaas/backups
/tmp/mypaas/builds
```

The control-plane PostgreSQL data itself is stored in the container engine's named `postgres_data` volume, so include that state in any engine-migration plan rather than assuming all persistent data is under `/var/lib/mypaas`.

Do not treat `~/MyPaas` as the persistent data directory.

---

## Git and Compose Deployments

### Repository inspection

Before project creation, MyPaas can inspect the selected Git branch, show the repository tree, and detect runtime-related files. The selected branch is the source of truth for detection.

Supported source configuration includes:

- repository root or `baseDirectory`;
- ranked Compose file candidates;
- `composeFilePath`;
- ordered `composeOverridePaths`;
- `composeProfiles`;
- `composeWorkdir`;
- public/main Compose service;
- application port;
- static frontend path for hybrid projects;
- project and per-service resource limits.

### Environment discovery

MyPaas scans common templates such as:

```text
.env.example
.env.sample
.env.template
.env.local.example
```

For Compose/monorepo repositories it can discover nested templates, attribute variables to services, identify conflicting defaults, and generate per-service `.env` files at deployment time without overwriting a committed `.env` beside the template.

Environment values persisted by MyPaas are encrypted before storage.

---

## Container Registry Deployments

Registry projects use `sourceType=registry` and `deployMode=image`.

Examples:

```text
nginx:latest
ghcr.io/example/my-api:v1.4.0
ghcr.io/example/my-api@sha256:<digest>
```

The project lifecycle pulls the public image, applies environment/resource settings, maps the configured application port, records a digest when available, and routes it through Caddy.

Registry projects currently do **not** provide private-registry credential storage and do not accept Git webhook deployment triggers.

---

## Database Features

### Shared PostgreSQL

When explicitly requested for a project and enabled globally, MyPaas can provision a dedicated database and role inside the platform PostgreSQL service and store the generated connection URL as the project's encrypted `DATABASE_URL`.

The database/role are cleaned up with the project lifecycle.

### DB Studio Lite

The owner-only Database tab can discover project database connections from `DATABASE_URL`, PostgreSQL variables, MySQL variables, MariaDB variables, or common component-style DB environment variables.

Implemented operations include:

- connection status;
- schema/table/column browsing;
- paginated row browsing;
- server-side search and enum filters;
- temporary write sessions;
- guarded insert/update/delete operations;
- mutation audit records.

Write access is time-limited and must be explicitly enabled; read access does not implicitly grant row mutation.

---

## Logs, Metrics, and Observability

Project operations expose:

- runtime CPU and memory snapshots;
- uptime;
- per-service Compose metrics;
- recent runtime logs;
- live log events over SSE;
- deployment/build logs;
- project/deployment status events;
- optional Cloudflare request/bandwidth/error analytics.

The control-plane API also exposes:

```text
/health
/ready
/metrics
```

`/metrics` is Prometheus-compatible. In production it is served only when metrics Basic Auth credentials are configured, and valid credentials are required to read it.

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

Backups use PostgreSQL custom-format dumps and maintain separate daily/weekly retention sets. Image cleanup is scoped to unused MyPaas-managed images rather than indiscriminately pruning every host image.

Run a manual backup through the CLI verification path with:

```bash
RUN_BACKUP=true bash scripts/verify-production.sh
```

---

## CLI

The production API image includes `/app/mypaas`, and local builds create `backend/bin/mypaas`.

Build locally:

```bash
make build-backend
```

Examples:

```bash
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

MyPaas includes a stdio MCP server under `backend/cmd/mcp`. The MCP bridge is intended to run on the developer's local machine and call the remote MyPaas API using an API token generated from **Admin → Settings**.

From a local clone:

```bash
cd backend
MYPAAS_URL=https://mypaas.example.com/api \
MYPAAS_API_TOKEN=<token> \
go run ./cmd/mcp
```

The current server exposes tools for repository inspection, project creation/settings, deploy/start/stop/restart, deployments, rollback, logs, metrics, environment variables, quota, and host statistics.

Keep the MCP token private. Regenerating it invalidates agents using the previous token.

---

## VM Migration Tooling

Admin Settings can prepare an export containing:

- the MyPaas system PostgreSQL database;
- shared `mypaas_p_*` project databases and roles when available;
- the production `.env` when accessible;
- `/var/lib/mypaas/volumes`;
- `/var/lib/mypaas/compose`;
- `/var/lib/mypaas/static`;
- an export manifest.

The resulting archive is token-protected and expires after **24 hours**. A new VM can import it through `scripts/install-vm.sh --migrate-url <url>`.

The database portion uses logical dumps. Persistent directories are archived from the host filesystem, so applications with write-heavy file/volume state should be quiesced or otherwise validated for consistency before treating the archive as a point-in-time application snapshot.

---

## Automatic Self-Updates

Automatic updates are **opt-in**. After bootstrap has installed the updater, enable it with:

```bash
cd ~/MyPaas
AUTO_UPDATE_ENABLED=true \
AUTO_UPDATE_INTERVAL_MINUTES=30 \
bash scripts/configure-auto-update.sh
```

The updater tracks a Git ref, waits for API and dashboard GHCR images tagged with the matching commit SHA, and deploys source/configuration and images from the same revision rather than blindly following `latest`.

Useful commands:

```bash
# Check/apply immediately
bash scripts/update-vm.sh

# Inspect schedule
systemctl status mypaas-update.timer

# Logs
journalctl -u mypaas-update.service

# Disable
AUTO_UPDATE_ENABLED=false bash scripts/configure-auto-update.sh
```

The updater rejects dirty managed checkouts and performs best-effort runtime rollback when deployment/verification fails. Database migrations can still be forward-only, so backups remain necessary.

---

## Cloudflare Requirements

For public dashboard and project routes:

- add the MyPaas domain to Cloudflare DNS;
- create/configure a Cloudflare Tunnel;
- route the dashboard/root hostname and wildcard project hostname to Caddy (`HTTP` → `caddy:80`);
- create proxied root and wildcard DNS records pointing to the tunnel hostname.

A registrar transfer is not required; changing the authoritative nameservers is sufficient when the domain is registered elsewhere.

Cloudflare Tunnel credentials are configured separately from the optional Cloudflare Analytics API token/zone settings.

---

## Production Operations

The production scripts use the `docker`/`docker compose` command surface. On a Podman installation those commands are compatibility clients talking to `podman.socket`.

Useful commands:

```bash
# Deploy the current checkout
bash scripts/deploy-to-vm.sh

# Verify containers, API readiness, Caddy Admin API, and CLI
bash scripts/verify-production.sh

# Also verify a manual PostgreSQL backup
RUN_BACKUP=true bash scripts/verify-production.sh

# Inspect control plane
docker compose -f docker-compose.prod.yml ps
docker compose -f docker-compose.prod.yml logs -f
```

Useful installer flags/environment:

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
- **Docker CLI + Compose plugin** with either Docker Engine or a compatible Podman socket for container-backed development workflows.
- **Caddy 2** when testing routing outside the development Compose stack.
- **Nixpacks CLI** when exercising the repository-detection fallback that inspects provider/framework metadata.

The Makefile currently invokes the command name `docker`, so Podman-based local development must provide the same compatibility command/socket arrangement used by the VM installer.

### Setup

```bash
git clone https://github.com/nabilrn/MyPaas.git
cd MyPaas
cp .env.example .env
```

Start development dependencies and database migrations:

```bash
make dev
```

Run API and dashboard separately:

```bash
make backend-dev
```

```bash
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

Pull requests are checked by GitHub Actions with backend tests, frontend unit/type/build checks, deployment-script checks, bootstrap regression tests, and production Compose rendering.

---

## Project Structure

```text
MyPaas/
├── backend/
│   ├── cmd/
│   │   ├── api/                Go HTTP API
│   │   ├── cli/                mypaas CLI
│   │   └── mcp/                local stdio MCP bridge
│   ├── internal/               deployment, container, backup, DB Studio, auth, etc.
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

Start from `.env.example`; it remains the authoritative list of environment-backed configuration.

Core examples:

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
# In Podman mode this path is bridged to /run/podman/podman.sock.
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

Optional Cloudflare Analytics credentials are managed through platform settings rather than the Tunnel token above.

Do not commit production `.env`, generated tokens, database credentials, or encryption keys.

---

## Current Boundaries

MyPaas deliberately remains smaller than a general-purpose cloud platform:

- single-host VM control plane;
- no Kubernetes scheduler;
- no multi-node orchestration or autoscaling in `main`;
- no private-registry credential manager;
- Podman support is through the Docker-compatible CLI/socket contract rather than a separate native Podman backend;
- Nixpacks is an inspection aid, not an automatic backend/SSR buildpack deployment mode;
- static deployments use redeploy/roll-forward instead of the historical container rollback action.

These constraints are intentional unless a measured operational need justifies additional complexity.

---

## Documentation

- **[Product Requirements](docs/PRD.md)** — product scope and behavior.
- **[Architecture](docs/ARCHITECTURE.md)** — technical design and system diagrams.
- **[Architecture Decisions](docs/adr/)** — recorded architectural decisions.
- **[Product direction](PRODUCT.md)** — user, purpose, and UI principles.
- **[Conventions](AGENTS.md)** — repository-wide engineering constraints for agents/contributors.
- **[Changelog](CHANGELOG.md)** — notable implementation changes.

When documentation disagrees with executable behavior, treat the current `main` code, schema, tests, and installer as the source of truth and update the stale documentation.

---

## Contributing

1. Create a focused branch from the latest `main`.
2. Keep changes scoped to one bug or feature where practical.
3. Add/update regression tests for behavior changes.
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

### Project domain does not resolve

Verify Cloudflare Tunnel public-hostname routes plus the proxied root and wildcard DNS records.

### Production stack verification fails

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
- **Documentation:** start with `docs/`, ADRs, and this README.
- **Security issues:** use the repository's private security contact/process instead of publishing sensitive details in a public issue.

---

**Implementation audit:** 2026-08-10
