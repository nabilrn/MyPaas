# MyPaaS Documentation

Documentation for the current stable single-host MyPaaS product line.

This index separates **operator/user guidance**, **engineering architecture**, and **historical evidence** so an old qualification record is not mistaken for current product behavior.

## Source of truth

When documents disagree, use this order:

1. current code, database schema, tests, installers, production Compose/configuration, and executable command behavior;
2. current architecture and security-boundary documents;
3. accepted architecture decision records (ADRs), including later amendments;
4. current product scope and roadmap;
5. historical requirements, release notes, audits, and qualification records.

Historical records should remain historically accurate. Do not rewrite them to pretend an old release already had the current contract.

## User and operator documentation

Start here when installing or operating a MyPaaS VM:

| Document | Purpose |
| --- | --- |
| [Installation](installation.md) | Stable version-pinned VM install, prerequisites, runtime choice, post-install verification |
| [Configuration](configuration.md) | Production `.env`, Admin Settings, runtime/profile/update configuration semantics |
| [CLI](cli.md) | Current `mypaas` command surface and authentication boundary |
| [REST API](api.md) | Current API route families, auth boundaries, and deliberate unsupported mutations |
| [MCP](mcp.md) | Local stdio MCP bridge setup and credential lifecycle |
| [Updates](operations/update.md) | Stable release channel, manual update, periodic timer, status and rollback behavior |
| [Backup and restore](operations/backup-restore.md) | Control-plane backup and full disaster-recovery bundle workflow |
| [VM migration](operations/migration.md) | Supported Administration migration path and storage/release-identity boundaries |
| [Production verification](operations/production-verification.md) | What `verify-production.sh` checks and does not prove |
| [Uninstall](operations/uninstall.md) | Destructive removal boundary and required backup precautions |
| [mypaas-statd](STATD.md) | Optional host-native telemetry integration and operations |

## Product and architecture

| Document | Purpose |
| --- | --- |
| [`../PRODUCT.md`](../PRODUCT.md) | Current product scope, audience, capabilities, and non-goals |
| [`../ROADMAP.md`](../ROADMAP.md) | Current product direction and prioritization boundary |
| [Architecture](ARCHITECTURE.md) | Canonical current single-host architecture |
| [Security boundaries](SECURITY_BOUNDARIES.md) | Trust, engine authority, Compose, storage, shell, database, and tenant boundaries |
| [Architecture overview](architecture/overview.md) | Control-plane responsibilities and request paths |
| [Deployment architecture](architecture/deployment.md) | Deployment modes, inspection, lifecycle, routing, and rollback |
| [Networking](architecture/networking.md) | Control/project/routing network separation and Caddy boundaries |
| [Observability](architecture/observability.md) | Metrics, statd fallback, logs, host telemetry, and health surfaces |

## Development and engineering

| Document | Purpose |
| --- | --- |
| [Development](development.md) | Local toolchain, `make` workflow, tests, builds, migrations, and generated code |
| [`../AGENTS.md`](../AGENTS.md) | Repository engineering constraints and current implementation boundaries |
| [Runtime verification](engineering/runtime-verification.md) | Current retained runtime qualification/evidence boundary |
| [Create Project contract](ux/create-project-contract.md) | Current Create Project UX/product contract |
| [Create Project AI auditor prompt](ux/create-project-ai-auditor-prompt.md) | Current audit prompt derived from the UX contract |
| [WebMCP](WEBMCP.md) | Experimental browser `document.modelContext` adapter |

## Architecture decision records

ADRs under [`adr/`](adr/) capture design decisions and their lifecycle. Read status and amendment notes before treating an ADR as current behavior.

Current accepted decisions include DB Studio, Compose configuration/parity/routes, OCI registry deployment/authentication, self-update, migration safety, metrics semantics, GitHub repository access, persistent image storage, and platform setting semantics.

Deferred ADRs record earlier design directions that are **not** current stable product commitments.

## Current product facts

The current stable contract is intentionally narrow:

- one Linux host;
- rootful Podman by default on fresh supported hosts, Docker Engine compatibility mode available;
- Git deployments through Dockerfile, Docker Compose, and static output;
- OCI image deployment, with anonymous pull or one bounded installation-level registry credential;
- Caddy primary routes plus up to four bounded additional Compose HTTP routes;
- project-scoped persistence and optional shared PostgreSQL;
- DB Studio Lite for PostgreSQL, MySQL, MariaDB, and eligible persistent SQLite;
- DB Studio read-only by default, temporary update-only primary-key writes, insert/delete/raw SQL disabled;
- owner-only short-lived host shell and read-only host-wide container inventory;
- backups, restore/migration tooling, CLI, REST API, GitHub webhooks, audit logs, local stdio MCP, and optional browser WebMCP adapter;
- optional `mypaas-statd` host telemetry with Docker-compatible runtime fallback.

The product does not currently claim Kubernetes/multi-node scheduling, control-plane HA, automatic horizontal application scaling, hostile multi-tenant isolation, raw TCP/SSH/UDP routing, a universal workload-capacity number, or a one-click application template catalog.

## Historical records

Historical material is retained for provenance and should not be used as the current operator procedure merely because the file still exists.

Examples:

- [`operations/beta-backup-restore-drill.md`](operations/beta-backup-restore-drill.md) — historical backup/restore qualification procedure;
- [`operations/beta-create-project-audit.md`](operations/beta-create-project-audit.md) — historical beta Create Project audit procedure;
- [`ux/create-project-production-audit-2026-08-13.md`](ux/create-project-production-audit-2026-08-13.md) — historical production UX observation;
- [`releases/`](releases/) — release-specific records and notes;
- [`PRD.md`](PRD.md) — historical product requirements/baseline where retained.

For current operational steps, use the User and operator documentation section above.

## Documentation maintenance rules

When behavior changes:

- update the current operator doc in the same change when commands/semantics changed;
- update architecture/security docs when a trust or execution boundary changed;
- create/amend an ADR for a durable architecture decision;
- preserve historical records rather than silently rewriting their evidence;
- do not publish passwords, tokens, cookies, OAuth secrets, registry credentials, production `.env` values, decrypted project env values, or private application data;
- do not convert benchmark or one-host observations into generic capacity claims.

A command shown in current operator docs should map to an executable path/target in the repository or to an explicit dashboard workflow. If the product lacks a safe public workflow, document that limitation instead of inventing one.
