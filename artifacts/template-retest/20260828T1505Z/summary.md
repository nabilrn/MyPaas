# MyPaaS Template Compatibility Retest

- Environment: staging VM `172.104.61.180`
- Domain: `malala.tech`
- Candidate SHA: `77b914071dc82ffcd3b833aefa9344aa35017a80`
- Scope: `directus`, `n8n`, `ghost`, `paperless-ngx`, `openclaw`, `umami`
- Production VM was not modified.

| Template | Deploy | Health | Public route | Verdict |
| --- | --- | --- | --- | --- |
| Directus | running | healthy | `/server/ping` passed | PASS_USABLE* |
| n8n | failed | not reached | not tested | FAIL_TEMPLATE |
| Ghost | running | container running | persistent `503` | FAIL_TEMPLATE |
| Paperless-ngx | running | container running | persistent `502` | FAIL_MYPAAS_PLATFORM** |
| OpenClaw | failed | readiness timeout at 1m | not tested | FAIL_TEMPLATE |
| Umami | failed | readiness timeout at 1m | not tested | FAIL_TEMPLATE |

## Diagnostics

- n8n image `docker.io/n8nio/n8n:2.36.7` was pulled successfully by staging Podman. Digest: `sha256:770da605a7dfdda55838fb2b66b701435690ffcce5d3067585fc7e3cb17b168f`.
- n8n still failed through the MyPaaS Compose path with pull exit status `18`; see `n8n-rerun.json`.
- Ghost remained publicly unavailable with `503` after deployment reached `running`.
- Paperless-ngx remained publicly unavailable with `502`; internal-vs-routing probes were not captured, so platform ownership needs a follow-up diagnostic run.
- OpenClaw and Umami remained in `running/health=starting` until the existing 1-minute MyPaaS readiness bound expired. The timeout was not changed.

`*` Directus baseline route/lifecycle qualification passed; application credential and sentinel persistence checks were not implemented by the runner.

`**` Paperless-ngx is a provisional classification pending internal reachability evidence.

Persistent 502 failures: `1`

Persistent 503 failures: `1`

The JSON files in this directory contain sanitized deployment evidence only; secrets and environment values are excluded.
