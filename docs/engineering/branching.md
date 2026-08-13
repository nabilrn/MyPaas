# Branching Guidelines

MyPaas branch names must make the work domain obvious. Avoid broad bot-generated branch names such as `agent/...` for durable work.

## Domain prefixes

Use one of these prefixes:

- `core/` — product-critical platform behavior: updates, backups, deployment, runtime, ports, DB Studio, auth, migrations.
- `infra/` — VM, Docker, GHCR, Caddy, Cloudflare, statd, host cleanup, installer, systemd.
- `ux/` — dashboard, Create Project, project detail, settings, visual consistency, accessibility.
- `test/` — performance, benchmarks, resilience, e2e, Playwright audits, load and concurrency tests.
- `docs/` — ADRs, runbooks, architecture, beta readiness, operational documentation.
- `chore/` — repository hygiene, dependency maintenance, mechanical cleanup.
- `fix/` — narrow urgent bugfixes when the domain is unclear or intentionally cross-cutting.

## Naming pattern

Use:

```text
<domain>/<short-outcome>
```

Examples:

```text
core/update-release-safety
core/backup-restore-drill
core/dbstudio-compose-env-detection
infra/docker-cache-retention
infra/ghcr-sha-publish-guard
test/perf-many-projects
test/resilience-concurrent-deploys
ux/create-project-source-flow
docs/beta-readiness-gates
chore/repo-branch-cleanup
```

## PR titles

Use the same domain language in PR titles:

```text
core: harden VM update release flow
core: add backup restore validation
infra: prune Docker build cache safely
test: add many-project performance harness
docs: define beta readiness gates
```

## Cleanup rules

- Delete merged branches immediately.
- Delete closed/unmerged experiment branches unless the owner explicitly keeps them.
- Rename surviving long-lived work to a domain prefix before continuing it.
- Do not create new `agent/*` branches for normal repo work.
- One branch should have one clear outcome. Split unrelated fixes into separate branches.

## Beta readiness workstreams

Use these branch families for pre-beta core work:

- `core/update-*` for self-update, rollback, release image consistency, and update verification.
- `core/backup-*` for backup scheduling, restore, disaster recovery, and backup integrity checks.
- `test/perf-*` for many-project benchmarks, resource pressure, and throughput.
- `test/resilience-*` for concurrent deploy/redeploy/delete, failure injection, and recovery drills.
- `infra/docker-*` for image/cache retention and disk pressure controls.
