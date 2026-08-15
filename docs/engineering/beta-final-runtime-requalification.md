# Final Beta Runtime Requalification Runbook

This runbook defines the final runtime qualification required before publishing a replacement MyPaaS beta release after PR #121 (`fix: harden beta release gates`).

## Immutable candidate

Qualify the merged PR #121 state, not the PR head:

- PR #121 head: `8e8c462cc0b5a2d5a7d9c3eafcac43f3b107061d`
- final candidate / merge SHA: `f66a823a0e57dbc0b89e9e8681140e343df20afd`

All qualifying evidence must be attributable to the same exact candidate SHA. If a product defect requires a code fix and therefore produces a new merge SHA, stop qualification and rerun the affected gates against the new candidate.

## Primary runtime target

- VM: `172.105.118.30`
- SSH: `root@172.105.118.30`
- Domain: `https://malala.tech`

MyPaaS is already running on this VM, but the deployed version is unknown. Do **not** reinstall or update blindly.

Before mutation, fingerprint the existing installation and record:

- installation/repository path;
- `git rev-parse HEAD`, current branch/ref, and dirty state;
- configured safe build/image identity (`MYPAAS_BUILD_SHA`, `MYPAAS_IMAGE_TAG`) without dumping secrets;
- actual API and dashboard container images;
- `/health`, `/ready`, dashboard and Caddy state;
- one representative existing project route when available;
- running containers and persistent volumes;
- host OS/kernel, CPU, RAM, disk, Docker and Compose versions;
- whether backup/R2 configuration exists, without recording secret values.

If the current runtime is already the exact candidate, qualify it in place. If it is older/different, preserve the source identity and use the repository-supported updater path to move to the exact candidate; that real transition should become Phase 1 evidence.

## Authoritative documents

Read these before execution:

- `docs/engineering/beta-readiness-master-plan.md`
- `docs/engineering/beta-readiness-gates.md`
- `docs/ux/create-project-contract.md`

The gate document is authoritative. Repository CI being green is necessary but not sufficient. `BLOCKED_ON_VM_EVIDENCE` is not `PASS`.

## Durable evidence contract

Create the run directory before testing:

```text
artifacts/beta-readiness/<UTC_TIMESTAMP>-f66a823a0e57/
```

Recommended structure:

```text
preflight/
phase-1-update-release/
phase-2-backup-restore/
phase-3-performance/
phase-4-resilience/
phase-5-retention/
phase-6-create-project/
phase-7-dbstudio/
final-summary.md
final-summary.json
```

Every phase must persist `summary.md` and `summary.json` plus sanitized raw logs/screenshots/traces when applicable. Do not leave final evidence only in terminal scrollback or agent chat.

Never persist passwords, JWTs, cookies, OAuth secrets, GitHub tokens, Cloudflare tokens, R2 access/secret keys, decrypted environment values, database passwords, or tunnel credentials.

## Phase 0 — Preflight and provenance

Capture the existing VM state before mutation, then verify the target identity.

Expected final candidate:

```text
f66a823a0e57dbc0b89e9e8681140e343df20afd
```

If the VM is on an older version, record that source SHA/runtime identity and use it for the update test. Do not wipe useful project state; existing routes are valuable update/rollback sentinels.

## Phase 1 — Update / release safety

Run the full runtime gate.

Validate:

- immutable target API and dashboard images exist;
- supported N -> exact target update succeeds;
- checkout, API image, dashboard image and runtime build SHA identify the target;
- `/health`, `/ready`, dashboard, Caddy and an existing project route remain healthy;
- a deliberately missing target image fails closed and leaves checkout/runtime unchanged;
- a controlled post-update verification failure triggers rollback;
- rollback restores the previous checkout and previous working runtime images;
- an existing route still works after rollback;
- rollback does not depend on pulling a nonexistent remote rollback tag;
- after the drill, return the VM to the exact candidate and verify identity again.

Use existing supported scripts such as `scripts/update-vm.sh`, `scripts/deploy-to-vm.sh`, and `scripts/verify-production.sh` instead of inventing a new updater.

## Phase 2 — Backup / restore

This gate must cover current local/manual backup behavior and the newer R2/S3-compatible path.

### Local/manual backup

Verify that the backup includes and safely handles:

- control-plane PostgreSQL;
- required production config;
- `/var/lib/mypaas/static`;
- managed persistent volumes;
- manifest and checksums;
- restrictive permissions for sensitive files;
- checksum verification;
- secret-safe reports.

Prefer the existing `scripts/backup-restore.py` tooling.

### Cloudflare R2 / S3-compatible backup

Verify:

- configuration persists correctly;
- credentials are never exposed;
- manual remote backup uploads successfully;
- backups can be listed/discovered and the intended object selected;
- the object can be downloaded and verified;
- invalid credentials and missing objects return actionable sanitized errors.

### Backup-first installer / restore flow

Verify backup detection, R2 restore selection, state transitions, error reporting, successful completion, background cleanup, and final success behavior.

### Fresh restore requirement

A full `PASS` requires restore to a genuinely fresh/disposable MyPaaS installation. Do not wipe `172.105.118.30` merely to simulate a fresh target.

If no second fresh VM is available, complete all safe source-side backup/R2 checks but mark the fresh-restore requirement `BLOCKED_ON_VM_EVIDENCE`; do not call Phase 2 `PASS`.

A true fresh restore must verify login, project inventory, deployment history, routes, static artifacts, persistent volume data, normal encrypted-environment behavior, DB Studio, and a machine-readable restore report.

## Phase 3 — Many-project performance

Reuse the existing benchmark harness.

Run sequentially:

1. 10 projects;
2. 25 projects only if the 10-project batch genuinely passes;
3. 50 projects only if the 25-project batch genuinely passes.

Fixtures must cover static, Dockerfile, and Compose app+database deployments.

Record create/deploy timings, route verification, failure rate, configured thresholds, p95 where supported, CPU, memory, storage growth, Docker image growth, BuildKit/cache growth, volume growth, and explicit capacity limits.

Do not put a raw immutable SHA into a project `branch` field; the project branch must remain clonable while the report independently records the immutable tested SHA.

Distinguish product, harness, environment, and edge/network failures; do not reinterpret Cloudflare or harness failures as MyPaaS performance failures.

## Phase 4 — Concurrent-deploy resilience

Reuse the existing resilience harness.

Exercise concurrent creates, initial deploys and redeploys with healthy work mixed with an intentionally failing deployment. Use webhook burst only when the documented destructive scenario explicitly enables it.

Verify:

- healthy projects settle correctly;
- intentional failures fail as expected;
- no duplicate host ports;
- no unexpected stuck/non-terminal deployments beyond timeout;
- final state is internally consistent;
- unrelated/sentinel routes remain healthy;
- a failed deployment does not replace a previously healthy runtime;
- routing is consistent after workers settle.

Persist JSON and Markdown evidence.

## Phase 5 — Docker / cache retention

Requalify because updater and rollback image lifecycle changed.

Verify dry-run and controlled cleanup, MyPaaS-managed image scoping, preservation of running and rollback-critical images, recent-image retention, BuildKit cleanup, before/after Docker/cache usage, surviving application routes, successful redeploy after cleanup, and viability of the updater rollback path after retention cleanup.

Do not run broad destructive host cleanup against unrelated data.

## Phase 6 — Create Project runtime contract

Run the production Playwright regression audit against `https://malala.tech` using `docs/ux/create-project-contract.md` as the behavioral source of truth.

Cover at minimum:

- static Git repo;
- Dockerfile repo;
- Compose app+DB;
- GHCR/container registry;
- subdirectory/base-directory flow;
- invalid repository;
- required environment values;
- missing port;
- stale, slow and in-flight analysis;
- backend analysis failure/timeout;
- Compose Doctor blockers;
- creation failure and re-analysis;
- Advanced Settings behavior.

Mandatory registry regression:

```text
ghcr.io/stefanprodan/podinfo:latest
```

The Container Port field must remain visible and retain `8080` after typing; the user must not need to collapse/reopen Advanced Settings.

Use representative viewports `1366x768`, `1440x900`, and `390x844`.

Evidence should include screenshots, accessibility/ARIA representation, console/network findings, geometry, Playwright traces, and HAR where practical. Every summary must record the tested SHA and runtime build SHA.

## Phase 7 — DB Studio Compose reliability

Run real controlled Compose smoke tests for PostgreSQL, MySQL and MariaDB using existing fixtures.

Verify real Compose service/network resolution, correct credentials without leakage, actionable incomplete-credential errors, successful connections, expected engine/driver, default read-only mode, explicit expiring write sessions, and generated smoke-project names within backend limits.

Where practical, verify DB Studio again after restore.

## Efficient execution order

Use this order:

1. preflight/runtime fingerprint;
2. update/release safety;
3. backup/R2/installer checks;
4. Create Project E2E;
5. DB Studio Compose;
6. concurrent resilience;
7. 10-project performance;
8. 25-project performance;
9. 50-project performance;
10. Docker/cache retention final drill;
11. final evidence consolidation.

Do not start heavy load tests while basic runtime flows are failing.

## Defect handling

If a genuine product defect is found:

1. stop the affected gate;
2. save exact sanitized evidence;
3. identify the root cause;
4. do not patch `main` directly;
5. create a focused branch/PR following repository branching rules;
6. add regression coverage and run CI;
7. if merged, treat the resulting merge SHA as a new candidate;
8. rerun the directly affected gate and any other gate invalidated by that subsystem change.

If a harness/environment/Cloudflare/DNS/credential problem occurs, classify and preserve it accurately; do not silently convert missing evidence into `PASS`.

## Per-phase summary

Each phase should record at minimum:

```json
{
  "testedGitSha": "f66a823a0e57dbc0b89e9e8681140e343df20afd",
  "runtimeBuildSha": "f66a823a0e57dbc0b89e9e8681140e343df20afd",
  "targetEnvironment": {
    "vm": "172.105.118.30",
    "domain": "https://malala.tech"
  },
  "startedAt": "...",
  "finishedAt": "...",
  "status": "PASS|FAIL|BLOCKED_ON_VM_EVIDENCE",
  "failures": [],
  "blockedReason": null
}
```

## Final release decision

Create `final-summary.md` and `final-summary.json` containing the exact tested/runtime SHA, target environment, timestamps, each gate state, overall result, and release recommendation.

Only declare `BETA_READY` when all seven runtime gates are `PASS` against the same exact candidate SHA.

Any `FAIL`, `BLOCKED_ON_VM_EVIDENCE`, or `NOT_RUN` means `NOT_BETA_READY`.

Do not create, move, or reuse a beta tag during this run. A replacement prerelease such as `v0.5.0-beta.1` requires explicit approval after the evidence is reviewed.
