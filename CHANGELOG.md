# Changelog

All notable changes to MyPaaS are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/) and the project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- Stable release updates now resolve the published release to a full Git SHA, cross-check the corresponding remote tag resolves to the same commit, fetch that exact SHA, and keep source plus immutable API/dashboard images on one release identity.
- Bootstrap accepts a full 40-character Git SHA and uses a verified detached checkout for exact-revision installs.
- VM migration destination instructions are pinned to the source control plane's concrete build SHA instead of cloning the repository default branch.

### Security

- Migration restore no longer embeds the temporary migration download token in the generated destination command or installer argv. `scripts/install-migration.sh` reads the sensitive URL from a hidden prompt/stdin, passes it to `curl` through protected input, downloads to a private temporary directory, and gives the existing installer only a local archive path.
- Release/update identity checks fail closed when published release metadata, the remote release tag, or the fetched exact commit disagree. This is consistency hardening within the GitHub release/repository trust boundary; it does not add a separate signing authority.

## [0.7.0] - 2026-09-09

First stable release line after the `v0.5.x` beta and `v0.6.0-rc.x` hardening series.

### Added

- Git deployment through Dockerfile, Docker Compose, and static output modes.
- OCI image deployment with anonymous pulls and one bounded installation-level registry credential.
- GitHub repository picker and private-repository deployment through the authenticated administrator connection.
- Repository inspection, base-directory/monorepo support, Compose discovery, Compose Doctor, environment discovery, and required configuration checks.
- Encrypted project environment variables and source-aware resource settings.
- Deployment history, logs, metrics, lifecycle actions, rollback, route reconciliation, and bounded deployment concurrency.
- Caddy project routing with one primary route and up to four bounded additional Compose HTTP routes using platform-derived hostnames.
- Project-scoped persistent storage and owned-resource cleanup.
- Optional shared PostgreSQL provisioning.
- DB Studio Lite for PostgreSQL, MySQL, MariaDB, and eligible persistent SQLite databases.
- DB Studio schema/table browsing, paginated rows, table-scoped string search, schema metadata/ERD, and temporary write sessions for validated primary-key row updates.
- Owner-only short-lived host shell with audit-safe session handling.
- Host-wide read-only container inventory with metadata-first loading.
- Backup, restore, migration, image/cache retention, audit log, CLI, REST API, webhook, and optional local MCP bridge surfaces.
- Optional `mypaas-statd` host telemetry integration with Docker-compatible engine fallback.
- Guarded release workflow that requires current-main identity, exact-SHA successful CI, and immutable API/dashboard images before publishing a release.

### Changed

- Fresh supported Linux installations are rootful Podman-first while retaining Docker Engine as an explicit compatibility mode through the existing Docker-compatible command/socket contract.
- Production routing uses explicit control/project/routing network boundaries and managed runtime aliases rather than treating published host ports as the normal Caddy data path.
- Project CPU values are shared scheduler ceilings rather than dedicated CPU reservations; host/admin copy and resource accounting reflect that model.
- Static projects are served directly by Caddy and do not consume container-runtime CPU/RAM quota accounting.
- Container inventory metadata is independent from per-project telemetry so host inventory does not block on collecting runtime CPU/RAM samples.
- SQLC generation is pinned and reproducible, with CI checking committed generated state.
- Repository qualification is framed around correctness contracts and targeted regression evidence instead of universal throughput/capacity claims.
- Current documentation defines MyPaaS as a stable single-host platform for an owner developer or small trusted team and keeps historical beta/RC material explicitly historical.

### Security

- Repository Compose input is rendered, sanitized, and validated before execution; known host-escape features remain rejected.
- Production Caddy administration uses a Unix socket instead of a published TCP admin endpoint.
- GitHub and registry credentials remain control-plane scoped and are not passed to project workloads.
- DB Studio stable writes are intentionally update-only. Row insertion, row deletion, and raw SQL are disabled; persistent SQLite must resolve inside an eligible persistent mount and uses an isolated helper path.
- Host shell access remains owner-only and is treated as host authority rather than a project terminal.

### Removed

- Owner-facing generic Ports/firewall management and arbitrary public port mutation.
- Obsolete benchmark-only framing and harness residue that did not represent product correctness.
- Public product claims for one-click application templates/catalogs, universal RPS/user capacity, automatic horizontal scaling, or multi-node orchestration.

### Fixed

The stable line includes the accumulated RC hardening for:

- rootful Podman socket resolution and explicit production network aliases;
- Caddy-to-API routing after control-plane recreation;
- static-project `0.01` CPU profile handling;
- shared-CPU host/quota semantics;
- Compose service readiness and route activation ordering;
- deployment failure-state preservation and lifecycle route consistency;
- persistent SQLite DB Studio discovery/helper execution across Docker-compatible Podman behavior;
- container-inventory responsiveness and metadata loading;
- reproducible backend database code generation and stale generated-state detection.

## Pre-stable history

The beta and release-candidate history is intentionally preserved as historical evidence rather than duplicated into the current stable contract:

- named release notes under [`docs/releases/`](docs/releases/);
- published GitHub releases under the repository's Releases page;
- merged pull requests and Git history;
- historical qualification/runbook documents that are explicitly marked historical.

Those records describe the product at their named point in time and do not override the current implementation, accepted ADRs, or stable product documentation.
