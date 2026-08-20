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

- Add Cloudflare/static asset cache guidance for CSS, JavaScript, SVG, image,
  and immutable asset paths.
- Add public API read-cache guidance for tenant-safe profile, study program,
  lecturer, event, banner, and published news endpoints.
- Audit database indexes for public faculty read endpoints.
- Expose or document Caddy access/performance logs for route-level latency
  diagnosis.
- Keep deployment concurrency tunable for small VMs.
- Add VM profile presets such as `small`, `balanced`, and `benchmark`.

## P2: Performance Mode

- Add optional `CLOUDFLARED_REPLICAS` support for multiple connectors attached
  to the same Cloudflare Tunnel.
- Surface Cloudflare Tunnel connector count and health in the dashboard.
- Add benchmark result summary rendering for k6 JSON output.
- Add a documented deploy-under-load workflow using selected project IDs.
- Add project-level static cache configuration where supported by routing.

## P3: Scale-Out Architecture

- Add multi-VM runtime workers.
- Support managed or external PostgreSQL for heavier production workloads.
- Add a build queue dashboard with pending/running/failed states.
- Add per-project autoscaling only after worker scheduling is mature.
- Consider Kubernetes as an optional enterprise runtime, not as a beta-default
  dependency.

## Current Benchmark Interpretation

The `v0.5.0-beta.2` SUT stayed healthy under smoke, API-only normal load,
static+API normal load, admin CRUD load, and a deliberately brutal 1000 VU
public spike. The brutal spike saturated the public/tunnel path but did not
exhaust VM CPU/RAM or crash the platform.

Therefore, Cloudflare connector replicas belong in a future performance mode,
not as a private beta blocker.
