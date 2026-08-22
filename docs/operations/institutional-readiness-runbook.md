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
- Cloudflare Tunnel protocol and connector count.
- Caddy delivery metrics availability.
- Static/header verification for static or hybrid frontend routes.
- `/api/*` cache behavior verification for dynamic routes.
- Deployment queue snapshot with queued, active, and recent failed counts.
- Backup/restore drill result or an explicit backup gap.
- k6 summary tied to the exact workload, runner location, and k6 version.

## Operating Gates

### Gate 1: Platform health

Run production verification and confirm all control-plane services are healthy:

- API
- dashboard
- PostgreSQL
- Caddy
- Cloudflare Tunnel connectors
- `mypaas-statd`

### Gate 2: Delivery path

Confirm the dashboard delivery panel shows Caddy metrics and the current
Cloudflare Tunnel topology. Static assets should show immutable cache headers
and compression where applicable. Dynamic APIs should keep application-defined
cache behavior.

### Gate 3: Deployment queue

Confirm the Projects dashboard shows deployment queue counts. A quiet platform
should show zero queued and active deployments. Recent failed deployments must
remain visible long enough for operators to investigate.

### Gate 4: Application class

Classify each benchmarked application:

- static public site
- SPA static site
- Next.js runtime
- API-only service
- Compose application
- generic container

Compare results only within the same application class and workload.

### Gate 5: Workload proof

Record smoke, baseline, normal, and stress results. Stop escalation when
correctness fails. Conclusions must name the workload, VM shape, Cloudflare
path, and bottleneck layer.

## Incident Triage

Use layer evidence before tuning:

- High app CPU or memory: inspect the application first.
- High Caddy p95 with low app pressure: inspect proxy/static delivery and
  response size.
- Low Caddy latency but high client latency: inspect Cloudflare edge, Tunnel,
  ISP path, and benchmark runner.
- Rising queue depth: reduce concurrent deploys, inspect build logs, or defer
  load testing until deploy work drains.
- High disk usage: prune old images, inspect static artifacts, and verify
  backup retention.

## Claim Format

Use constrained claims:

```text
On <VM shape>, MyPaaS <SHA> handled <workload> for <application SHA> at
<profile> with <error rate>, <p95 latency>, and <bandwidth>, through
<Cloudflare topology>. The observed bottleneck was <layer>.
```

Avoid unconstrained claims such as "supports 50,000 users" without workload,
duration, client location, and correctness criteria.
