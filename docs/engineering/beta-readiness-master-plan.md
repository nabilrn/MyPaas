# Beta Readiness Master Plan

MyPaas should not be called beta until update safety, backup/restore, performance, resilience, disk retention, and critical creation flows have reproducible evidence.

This plan is intentionally focused on release readiness, not feature polish.

## Goals

- Finish the smallest set of core reliability work needed before beta.
- Keep work grouped by domain branch names from `docs/engineering/branching.md`.
- Produce repeatable evidence for update, backup, restore, performance, concurrency, and critical UX/runtime flows.
- Avoid broad UI redesign while release blockers are still unknown.

## Workstreams

### 1. `core/update-release-safety`

Purpose: MyPaas updates must never silently deploy the wrong image, and failed updates must be recoverable.

Scope:

- Continue deploying API/dashboard by immutable commit SHA image tags.
- Add post-update verification for API health, dashboard reachability, Caddy route reconciliation, and existing project reachability.
- Add an explicit failed-update rollback drill.
- Surface the running build SHA/version in admin or settings.
- Document manual and automatic update runbooks.

Acceptance criteria:

- Updating from version N to N+1 deploys images matching the target Git SHA.
- If the target image is missing, the update leaves the running installation unchanged.
- If post-update verification fails, rollback restores the previous checkout/runtime images.
- The owner can see the running build/version from the dashboard or API.

### 2. `core/backup-restore`

Purpose: MyPaas must be recoverable from VM loss or corrupted state.

Scope:

- Backup MyPaas control-plane PostgreSQL.
- Backup required production configuration safely.
- Backup static artifacts under `/var/lib/mypaas/static`.
- Backup managed Docker volumes for user projects.
- Restore to a fresh VM.
- Verify login, project list, encrypted env readability, routes, deployments, and DB Studio after restore.

Acceptance criteria:

- A fresh VM can be restored from backup without manual database surgery.
- Restored projects retain routes, env metadata, deployment history, and persistent data.
- Restore verification produces a machine-readable report.
- Secrets are not printed in backup logs or reports.

### 3. `test/perf-many-projects`

Purpose: Establish the real limit of the current VM before beta.

Scope:

- Add fixture projects for static, Dockerfile, and Compose app+db deployments.
- Deploy 10, 25, and 50 project batches.
- Capture deploy duration, build duration, route reconciliation time, CPU, memory, disk growth, image/cache growth, and failure rate.
- Generate JSON and Markdown reports.

Acceptance criteria:

- Performance test can run repeatedly without manual setup beyond credentials/VM target.
- Report shows pass/fail thresholds for each project count.
- Disk growth is attributed to images, BuildKit cache, volumes, logs, or artifacts.
- Bottlenecks are documented before release decisions.

### 4. `test/resilience-concurrent-deploys`

Purpose: Verify MyPaas remains consistent under concurrent user and webhook activity.

Scope:

- Simulate concurrent create/deploy/redeploy operations.
- Simulate webhook bursts.
- Mix failed builds with successful builds.
- Read logs, metrics, and DB Studio while deploys are active.
- Verify project locks, deployment queueing, port allocation, Caddy routes, and final states.

Acceptance criteria:

- No duplicate port assignments.
- No stuck deployments after mixed success/failure runs.
- Existing project routes stay reachable during unrelated deploys.
- Failed deploys do not break successful project runtime state.
- Resilience report includes failed requests, state transitions, and final consistency checks.

### 5. `infra/docker-cache-retention`

Purpose: Prevent disk growth from repeated builds and deploys.

Scope:

- Add scheduled BuildKit cache pruning.
- Add old image retention policy.
- Keep active runtime images and recent successful deployment images.
- Add dry-run cleanup output.
- Surface disk pressure warnings in admin/settings.

Acceptance criteria:

- Cleanup never removes images used by running containers.
- Cleanup preserves enough recent images for rollback policy.
- Build cache reclaim can be run manually and by schedule.
- Disk report separates images, BuildKit cache, volumes, logs, and MyPaas artifacts.

### 6. `core/create-project-runtime-contract`

Purpose: Stabilize critical project creation modes without redesigning the UI.

Scope:

- Keep the source-first Create Project contract as the source of truth.
- Audit Git, GHCR/registry, Compose, static, Dockerfile, and subdirectory flows.
- Keep production audit non-destructive.
- Use deterministic mock audit for unsafe edge cases.
- Fix only blocking runtime/state bugs discovered by evidence.

Acceptance criteria:

- Git, registry/GHCR, Compose, static, Dockerfile, and subdirectory creation are covered.
- Production audit collects screenshots, ARIA, network, console, geometry, and trace evidence.
- Mock audit covers slow analysis, failures, stale analysis, missing port, required env, and base-directory cases.
- Create cannot become ready when analysis is stale or incomplete.

### 7. `core/dbstudio-compose-reliability`

Purpose: DB Studio should reliably detect and connect to supported Compose databases.

Scope:

- Keep Compose DB service env fallback.
- Add smoke tests for MariaDB, MySQL, and PostgreSQL Compose fixtures.
- Handle duplicate project names and stale project records more clearly.
- Show actionable errors when DB credentials are incomplete.
- Preserve safe read-only default behavior.

Acceptance criteria:

- DB Studio detects DB credentials from project env or Compose DB service env.
- DB Studio connects to Compose DB service network without relying on fake fallback network names.
- Missing/incomplete credentials produce clear user-facing errors.
- Write session remains explicit and time-limited.

### 8. `docs/beta-readiness-gates`

Purpose: Define the release gate for beta.

Scope:

- Document pass/fail gates for update, backup/restore, performance, resilience, disk retention, Create Project, and DB Studio.
- List known limitations and acceptable beta caveats.
- Link reports generated by test harnesses.

Acceptance criteria:

- Beta readiness can be evaluated from a checklist with evidence links.
- The owner can see which blockers remain.
- Known limitations are explicit instead of discovered from production use.

## Fastest execution order

1. `docs/beta-readiness-gates`
2. `test/perf-many-projects`
3. `test/resilience-concurrent-deploys`
4. `infra/docker-cache-retention`
5. `core/update-release-safety`
6. `core/backup-restore`
7. `core/create-project-runtime-contract`
8. `core/dbstudio-compose-reliability`

Reasoning: start with gates and harnesses first, then use the evidence to prioritize fixes. Implementing backup/update refinements before load and resilience evidence risks optimizing the wrong failure mode.

## Operating rules

- Do not start broad UI redesign work until beta blockers are known.
- Prefer small focused PRs grouped by the branch domains above.
- Every reliability PR should include validation results or a reason validation is blocked.
- Delete merged branches immediately.
- Keep production tests non-destructive unless explicitly marked as controlled integration tests.
