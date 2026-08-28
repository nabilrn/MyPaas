# Branching Guidelines

MyPaaS branch names should make the work domain and outcome obvious. Avoid broad bot-generated branch names such as `agent/...` for durable work.

## Domain prefixes

Use one of these prefixes:

- `core/` — product-critical platform behavior: deployment, runtime, routing, persistence, DB Studio, auth, migrations, recovery.
- `infra/` — VM/runtime integration, GHCR, Caddy, Cloudflare, statd, host cleanup, installer, systemd.
- `ux/` — dashboard, Create Project, project detail, settings, templates, accessibility.
- `test/` — targeted regression, compatibility, resilience, or e2e work tied to a concrete behavior.
- `docs/` — ADRs, architecture, compatibility records, runbooks, operational documentation.
- `chore/` — repository hygiene, dependency maintenance, mechanical cleanup.
- `fix/` — narrow urgent bugfixes when the domain is intentionally cross-cutting or unclear.

## Naming pattern

Use:

```text
<domain>/<short-outcome>
```

Examples:

```text
core/compose-http-routes
core/backup-restore-safety
infra/podman-socket-recovery
ux/template-env-guidance
test/minio-route-regression
docs/sync-post-pr157
chore/repo-branch-cleanup
```

Do not create broad branch programs merely to repeat performance matrices or historical qualification phases.

## PR titles

Use the same domain language in PR titles:

```text
core: harden Compose route lifecycle
infra: fix Podman socket resolution
ux: clarify template environment setup
test: cover MinIO route reconciliation
docs: sync current product documentation
```

## Cleanup rules

- Delete merged branches when practical.
- Delete closed/unmerged experiment branches unless the owner explicitly keeps them.
- Rename surviving long-lived work to a domain prefix before continuing it.
- Do not create new `agent/*` branches for normal repository work.
- One branch should represent one domain + one outcome.
- Split unrelated product/runtime changes rather than hiding them inside a qualification branch.

## Testing branches

A `test/` branch must answer a concrete product question or protect a known behavior.

Good examples:

- compatibility qualification for a real OSS workload;
- regression coverage for a confirmed defect;
- targeted lifecycle/recovery verification after a runtime change;
- e2e coverage for a user-visible workflow.

Do not maintain active branch families for generic throughput, many-project, direct-vs-tunnel, kernel, or resource-pressure benchmarking unless a specific product defect requires that measurement.

## Source of truth

For current product direction and qualification policy, use:

- [`AGENTS.md`](../../AGENTS.md)
- [`PRODUCT.md`](../../PRODUCT.md)
- [`ROADMAP.md`](../../ROADMAP.md)
- [`beta-readiness-gates.md`](beta-readiness-gates.md)
- [`../../compatibility/CATALOG.md`](../../compatibility/CATALOG.md)

Historical planning documents are not current branching programs.
