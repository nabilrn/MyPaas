# Institutional Readiness Runbook

This runbook defines the evidence required before a single-machine MyPaaS
installation is presented as ready for a medium institution. It does not turn
MyPaaS into a multi-VM platform; it keeps the claim tied to the tested VM,
applications, public path, and workload.

## Scope

Use this runbook for one-VM installations that host organization-scale internal
or public applications, such as faculty sites, department portals, dashboards,
or moderate CRUD systems.

Do not use it to claim hyperscale or hostile multi-tenant readiness. Multi-VM
runtime workers, per-project autoscaling, and stronger tenant isolation remain
separate scale-out work.

## Required Evidence

- Exact MyPaaS commit SHA and image tag.
- Exact application commit SHA or immutable image digest.
- VM CPU, RAM, disk, swap, OS, and container runtime.
- Public delivery mode: `tunnel`, `cloudflare-origin`, or `direct`.
- When using Tunnel: Cloudflare Tunnel protocol and connector count.
- When using a public origin: TLS certificate coverage/expiry and firewall rule evidence.
- Caddy delivery metrics availability.
- Static/header verification for static or hybrid frontend routes.
- `/api/*` cache behavior verification for dynamic routes.
- Deployment queue snapshot with queued, active, and recent failed counts.
- Desired/ready replica count for every scaled Dockerfile/image project.
- Release-command result when `MYPAAS_PLATFORM_RELEASE_COMMAND` is configured.
- Backup manifest showing the control-plane database plus every managed `mypaas_p_%` project database.
- Restore drill result or an explicit restore gap.
- k6 summary tied to the exact workload, runner location, and k6 version.

## Operating Gates

### Gate 1: Platform health

Run production verification and confirm all enabled control-plane services are healthy:

- API
- dashboard
- PostgreSQL
- Caddy
- Cloudflare Tunnel connectors when `PUBLIC_DELIVERY_MODE=tunnel`
- `mypaas-statd` when configured

Do not mark a direct-origin installation unhealthy merely because cloudflared is intentionally absent.

### Gate 2: Delivery path

Confirm the dashboard delivery panel shows Caddy metrics and the selected public
delivery topology. Static assets should show immutable cache headers and
compression where applicable. Dynamic APIs should keep application-defined cache
behavior.

For `cloudflare-origin` and `direct`, verify that only the intended ingress ports
are public and that the Caddy Admin API, MyPaaS API, dashboard, PostgreSQL, and
project runtime ports remain private.

### Gate 3: Deployment queue

Confirm the Projects dashboard shows deployment queue counts. A quiet platform
should show zero queued and active deployments. Recent failed deployments must
remain visible long enough for operators to investigate.

### Gate 4: Runtime readiness and release safety

For Dockerfile/image projects, a replacement runtime must not become routable
until its container readiness gate passes. Images with a Docker/OCI healthcheck
must reach `healthy`; images without one retain the compatibility behavior that
a running container is considered ready.

If `MYPAAS_PLATFORM_RELEASE_COMMAND` is configured, verify that it runs from the
candidate image before the replacement primary is created. The command is a
reserved control-plane setting and must not appear in the application env file.
A non-zero release command must fail the deployment while the previously routed
runtime remains available. Release commands should be idempotent and are
intended for database/schema work that does not require application-local
persistent volumes.

Compose applications use their existing main-service healthcheck/readiness
contract. Arbitrary Compose release commands are intentionally not inferred.

### Gate 5: Replica topology

Replica count is opt-in through `MYPAAS_PLATFORM_REPLICA_COUNT` for Dockerfile
and image projects. Validate all of the following before treating a project as
scaled:

- desired count is between 1 and the supported platform maximum;
- the primary and every secondary replica run the same deployment image/digest;
- every secondary is running and, when a healthcheck exists, healthy;
- Caddy contains all ready upstreams after reconciliation;
- stopped/deleted projects have no orphaned secondary replicas;
- images declaring persistent `VOLUME` targets are rejected for multi-replica mode;
- aggregate replica reservations remain within the user RAM/CPU quota.

Route ownership is deliberately disjoint. Static and Compose routes are repaired
by the canonical route reconciler. Dockerfile/image routes are owned exclusively
by replica reconciliation for both desired count 1 and desired count greater
than 1. This prevents a periodic canonical repair pass from briefly replacing a
healthy multi-upstream route with a primary-only route.

### Gate 6: Backup and database isolation

Scheduled backup archives must include the legacy control-plane `database.sql`
for compatibility, a manifest, role metadata, and a dump for every managed
project database matching `mypaas_p_%`. A backup that only contains the MyPaaS
control-plane database does not satisfy this gate.

For shared PostgreSQL projects, verify that the managed project role has a
connection ceiling and that `PUBLIC` cannot connect directly to the project
database. Do not raise PostgreSQL global connection limits merely to mask a
pooling problem.

#### Role-aware disposable restore drill

Qualification must restore into an isolated disposable PostgreSQL target; never
overwrite the live control or project databases. Use a PostgreSQL major version
compatible with the source backup and keep the extracted archive and
`databases/roles.sql` owner-readable only because role dumps may contain password
verifiers.

For a clean isolated restore target:

1. Extract the backup to a private temporary directory and confirm the selected
   project dump is listed in `manifest.json`.
2. As the disposable cluster superuser, restore `databases/roles.sql` before the
   project database dump. If those role names already exist, use a clean
   disposable cluster rather than mutating production roles to make the test
   pass.
3. Create a disposable database for the selected managed project and make the
   corresponding managed project role its owner.
4. Restore the custom project dump with `pg_restore --no-owner --no-privileges`.
   The backup intentionally excludes object ownership/privileges from the dump,
   so cluster/database access policy must be re-established explicitly.
5. Reapply the managed database connection boundary: revoke `CONNECT` from
   `PUBLIC`, then grant `CONNECT` to the matching managed project role.
6. Verify expected schema/data as a privileged verifier, then verify the managed
   project role itself can connect and read the restored fixture data.
7. Destroy the disposable database/cluster after evidence is recorded.

Do not restore the archived `.env`/`dot-env` into the disposable target unless a
specific recovery test requires it. Never commit role dumps, database dumps,
credentials, or extracted backup contents as qualification evidence; record only
sanitized commands, manifest inventory, and PASS/FAIL results.

### Gate 7: Application class

Classify each benchmarked application:

- static public site
- SPA static site
- Next.js runtime
- API-only service
- Compose application
- generic container

Compare results only within the same application class and workload.

### Gate 8: Workload proof

Record smoke, baseline, normal, and stress results. Stop escalation when
correctness fails. Conclusions must name the workload, VM shape, public delivery
mode, and bottleneck layer.

Do not start the final mixed-load qualification while deployments, replica
reconciliation, backup/restore, or release-command validation is unresolved.
Load generation must run outside the MyPaaS VM.

## Incident Triage

Use layer evidence before tuning:

- High app CPU or memory: inspect the application first.
- High Caddy p95 with low app pressure: inspect proxy/static delivery and response size.
- Low Caddy latency but high client latency: inspect Cloudflare edge/Tunnel or the direct network path and benchmark runner.
- Replica desired/ready mismatch: inspect candidate image, healthcheck, quota, and routing-network attachment before adding more replicas.
- Release-command failure: keep the previous runtime serving, inspect the migration command offline, and do not bypass the gate by routing the failed candidate.
- Rising queue depth: reduce concurrent deploys, inspect build logs, or defer load testing until deploy work drains.
- High disk usage: inspect container logs, old images, static artifacts, Compose workspaces, and backup retention before increasing disk allocation.

## Claim Format

Use constrained claims:

```text
On <VM shape>, MyPaaS <SHA> handled <workload> for <application SHA> at
<profile> with <error rate>, <p95 latency>, and <bandwidth>, through
<delivery mode/topology>. The observed bottleneck was <layer>.
```

Avoid unconstrained claims such as "supports 50,000 users" without workload,
duration, client location, and correctness criteria.
