# MyPaaS Template Compatibility Retest

- Environment: staging VM `172.104.61.180`
- Domain: `malala.tech`
- Original candidate SHA: `77b914071dc82ffcd3b833aefa9344aa35017a80`
- Corrective qualification later tested SHA `0c13215bc999ae27fe38f61d3ef346a021705bc2` before the Compose registry-auth fix.
- Production VM was not modified.

## Raw original run results

| Template | Deploy | Health | Public route | Raw runner label |
| --- | --- | --- | --- | --- |
| Directus | running | healthy | `/server/ping` passed | `PASS_USABLE` |
| n8n | failed | not reached | not tested | `FAIL_TEMPLATE` |
| Ghost | running | container running | persistent `503` | `FAIL_TEMPLATE` |
| Paperless-ngx | running | container running | persistent `502` | `FAIL_MYPAAS_PLATFORM` |
| OpenClaw | failed | readiness timeout at 1m | not tested | `FAIL_TEMPLATE` |
| Umami | failed | readiness timeout at 1m | not tested | `FAIL_TEMPLATE` |

The per-template JSON files preserve the labels emitted during the original run. They are raw observations, not final ownership classification.

## Reviewed classification

| Template | Reviewed verdict | Reason |
| --- | --- | --- |
| Directus | `PASS_BASELINE` | Public `/server/ping` plus restart/redeploy/stop-start lifecycle checks passed. Application login/write sentinel persistence was not exercised. |
| n8n | `FAIL_UNCLASSIFIED` | The original pull failure did not establish template/platform ownership. A later targeted staging run at `0c13215...` reproduced an unauthenticated Docker Hub rate limit before container startup. |
| Ghost | `FAIL_UNCLASSIFIED` | Public HTTP failed with persistent `503`, but the preserved run probed `/` while the declared qualification contract is `/ghost/`. |
| Paperless-ngx | `FAIL_UNCLASSIFIED` | Public HTTP failed with persistent `502`. No internal application probe versus Caddy/routing probe was preserved. |
| OpenClaw | `FAIL_PLATFORM` | The gateway was still `running/health=starting` when the old MyPaaS one-minute readiness bound terminated the deployment. |
| Umami | `FAIL_PLATFORM` | The service was still `running/health=starting` when the old MyPaaS one-minute readiness bound terminated the deployment. |

## Later targeted staging finding

The isolated PR checkout was prepared without modifying the existing dirty staging checkout. Podman 5.4.2 and its rootful Docker-compatible socket were operational. n8n deployment reached MyPaaS ComposePull but failed on an unauthenticated Docker Hub `toomanyrequests` rate limit before containers started. Cleanup succeeded and the remaining applications were not run under the stop rule.

Source review then found a generic MyPaaS inconsistency: direct image pulls already supported `MYPAAS_REGISTRY_HOST`, `MYPAAS_REGISTRY_USERNAME`, and `MYPAAS_REGISTRY_PASSWORD` through a temporary isolated Docker config, while `docker compose pull --ignore-buildable` did not use that registry-auth path. The current PR branch corrects that generic Compose behavior; no n8n-specific pull fallback was added.

## Follow-up scope

Retest only n8n, Ghost, Paperless-ngx, OpenClaw, and Umami from the current PR head after source gates pass. The compatibility runner must set `MYPAAS_COMPAT_REPO_BRANCH=fix/template-compat-contracts` so projects clone the candidate branch rather than defaulting to `main`.

No performance benchmark, production mutation, scaling claim, or broad application matrix is part of this qualification.

Persistent HTTP 502/503 remains a hard failure and requires internal, routing-network, and public probes before ownership is assigned.
