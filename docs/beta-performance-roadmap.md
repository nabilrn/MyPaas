# Beta Performance Roadmap

This roadmap captures benchmark follow-ups from the `v0.5.0-beta.2` single-VM
System Under Test preflight. These items are intentionally staged so private
beta hardening stays focused on correctness before adding orchestration
complexity.

## P0: Beta Correctness And Release Hygiene

- Publish GHCR release images for every MyPaaS release tag/SHA.
- Keep installer and deploy scripts fail-closed for malformed production env.
- Validate `PUBLIC_DOMAIN` as a bare hostname, not a URL.
- Accept documented legacy bootstrap aliases only when they map cleanly to the
  current canonical env names.
- Document the GitHub OAuth callback URL expected by production installs.
- Keep the k6 benchmark suite deterministic and free from false-negative CRUD
  errors.

## P1: Latency Work Without Runtime Complexity

- Apply default Caddy delivery rules for static projects and hybrid frontends:
  immutable caching for framework/static asset paths, `no-cache` for the HTML
  entry point, and gzip/zstd compression for compressible assets.
- Add public API read-cache guidance for tenant-safe profile, study program,
  lecturer, event, banner, and published news endpoints.
- Audit database indexes for public faculty read endpoints.
- Expose or document Caddy access/performance logs for route-level latency
  diagnosis.
- Keep deployment concurrency tunable for small VMs.
- Add VM profile presets such as `small`, `balanced`, and `benchmark`.

## P2: Performance Mode

- Keep optional `CLOUDFLARE_TUNNEL_CONNECTORS` support for controlled
  one-variable tunnel connector experiments.
- Surface Cloudflare Tunnel connector count and health in the dashboard.
- Add benchmark result summary rendering for k6 JSON output.
- Add a documented deploy-under-load workflow using selected project IDs.
- Surface framework and delivery profile detection in the project detection API
  so operators can distinguish static, SPA, API-only, generic container,
  Compose, and Next.js runtime deployments before benchmarking.

## P3: Single-Host Scale Guardrails

- Surface Cloudflare Tunnel connector count and protocol in owner-only
  delivery telemetry so benchmark runs can be tied to the actual public path.
- Keep project detection results visible in the create-project flow so owners
  know whether an app is static, SPA, API-only, Compose, generic container, or
  Next.js runtime before choosing benchmark expectations.
- Support managed or external PostgreSQL for heavier production workloads.
- Add a build queue dashboard with pending/running/failed states.

## P4: Institutional Qualification

- Maintain repeatable benchmark profiles for representative institutional
  workloads: mostly-static public sites, dynamic API reads, authenticated CRUD,
  and heavier frontend applications.
- Record each run with exact MyPaaS SHA, application SHA, VM shape, tunnel
  topology, route type, cache headers, and client-side k6 version.
- Classify conclusions by layer: application, VM, Caddy, Cloudflare Tunnel,
  Cloudflare edge, external database, and load generator.
- Treat multi-VM runtime workers and per-project autoscaling as next-level
  architecture, not as a prerequisite for the single-machine institutional
  target.

## P5: Scale-Out Architecture

- Expose a build/deployment queue snapshot before adding worker scheduling so
  owners can see queued, active, and recently failed deploy work from the
  dashboard.
- Add multi-VM runtime workers.
- Add per-project autoscaling only after worker scheduling is mature.

## P6: Institutional Operations

- Define tested operating envelopes per workload class instead of one generic
  concurrent-user claim.
- Add runbooks for capacity review, backup/restore drills, incident triage,
  Cloudflare path checks, and application-specific optimization reviews.
- Add release gates that require passing smoke deploys, static header checks,
  delivery telemetry availability, queue visibility, backup verification, and
  benchmark summary provenance before an installation is called
  institution-ready.

## Current Benchmark Interpretation

The current performance interpretation is provisional and is based on the
external k6 preflight retained in `nabilrn/mypaas-test-vibecoder` under
`benchmarks/k6/PROOF.md`. Release decisions should use that sanitized proof
summary together with the exact MyPaaS release/SHA, fixture SHAs, VM profile,
and public-vs-origin path used by the run.

The working result is that controlled private beta correctness is acceptable at
the tested normal loads, while the public static+API path remains latency-bound
and the deliberately brutal 1000 VU public spike exceeds the current tunnel
path capacity. These are workload-specific observations, not universal capacity
claims.

Cloudflare connector replicas are available as an explicit experiment knob, not
as automatic orchestration. Current qualification should still treat the
single-VM deployment, one Caddy instance, and one application container as the
baseline unless an experiment states otherwise.
