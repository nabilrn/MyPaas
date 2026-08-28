# MyPaaS Template Compatibility Retest

- Environment: staging VM `172.104.61.180`
- Domain: `malala.tech`
- Candidate SHA: `77b914071dc82ffcd3b833aefa9344aa35017a80`
- Scope: `directus`, `n8n`, `ghost`, `paperless-ngx`, `openclaw`, `umami`
- Production VM was not modified.

## Raw run results

| Template | Deploy | Health | Public route | Raw runner label |
| --- | --- | --- | --- | --- |
| Directus | running | healthy | `/server/ping` passed | `PASS_USABLE` |
| n8n | failed | not reached | not tested | `FAIL_TEMPLATE` |
| Ghost | running | container running | persistent `503` | `FAIL_TEMPLATE` |
| Paperless-ngx | running | container running | persistent `502` | `FAIL_MYPAAS_PLATFORM` |
| OpenClaw | failed | readiness timeout at 1m | not tested | `FAIL_TEMPLATE` |
| Umami | failed | readiness timeout at 1m | not tested | `FAIL_TEMPLATE` |

The JSON files preserve the labels emitted during the run. They are raw evidence, not the final ownership classification.

## Post-review classification

| Template | Reviewed verdict | Reason |
| --- | --- | --- |
| Directus | `PASS_BASELINE` | Public `/server/ping` plus restart/redeploy/stop-start lifecycle checks passed. Application login/write sentinel persistence was not exercised. |
| n8n | `FAIL_UNCLASSIFIED` | `docker.io/n8nio/n8n:2.36.7` pulled successfully with staging Podman, but the MyPaaS `docker compose pull --ignore-buildable` path exited `18`. The preserved artifact does not contain enough Compose-provider diagnostics to assign ownership. |
| Ghost | `FAIL_UNCLASSIFIED` | Public HTTP failed with persistent `503`, but the preserved run probed `/` while the declared qualification contract is `/ghost/`. The failure is real, but template/platform ownership was not established. |
| Paperless-ngx | `FAIL_UNCLASSIFIED` | Public HTTP failed with persistent `502`. No internal application probe versus Caddy/routing probe was preserved, so platform ownership was not established. |
| OpenClaw | `FAIL_PLATFORM` | The gateway was still `running/health=starting` when MyPaaS terminated readiness at its fixed 1-minute bound. The template health contract can legitimately extend beyond that bound. |
| Umami | `FAIL_PLATFORM` | The service was still `running/health=starting` when MyPaaS terminated readiness at its fixed 1-minute bound. The template health contract can legitimately extend beyond that bound. |

## Diagnostics

- n8n image `docker.io/n8nio/n8n:2.36.7` was pulled successfully by staging Podman. Digest: `sha256:770da605a7dfdda55838fb2b66b701435690ffcce5d3067585fc7e3cb17b168f`.
- n8n still failed through the MyPaaS Compose path with pull exit status `18`; see `n8n-rerun.json`.
- Ghost remained publicly unavailable with `503` after deployment reached `running`; the preserved probe path does not match the declared `/ghost/` contract.
- Paperless-ngx remained publicly unavailable with `502`; internal-vs-routing probes were not captured.
- OpenClaw and Umami remained in `running/health=starting` until the then-existing 1-minute MyPaaS readiness bound expired.

## Follow-up scope

The corrective source change raises the generic Compose readiness floor and adds the previously missing rootful-Podman `docker compose pull --ignore-buildable` CI path. The old staging run must not be reinterpreted as proof that n8n, Ghost, Paperless-ngx, OpenClaw, or Umami are fundamentally unsupported.

Only affected paths require targeted retest after the source fix. No performance benchmark or production change is part of this qualification.

Persistent 502 failures observed in this run: `1`

Persistent 503 failures observed in this run: `1`

The JSON files in this directory contain sanitized deployment evidence only; secrets and environment values are excluded.
