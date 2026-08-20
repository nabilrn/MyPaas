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

Cloudflare connector replicas remain future P2 performance-mode work and are
not implemented as part of private-beta correctness hardening.
