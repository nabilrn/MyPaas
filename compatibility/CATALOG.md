# MyPaaS Real-World Compatibility Suite

This suite measures whether MyPaaS can correctly host representative real-world open-source workloads. It is a compatibility suite, not a throughput or capacity benchmark.

## Rules

- A `PASS` means the application deployed and its declared smoke checks succeeded on the tested MyPaaS host.
- A failure must be classified as a MyPaaS defect, an upstream application/configuration problem, a host-resource limit, or an intentional platform boundary before it becomes accepted evidence.
- Do not convert a passing fixture into an RPS, concurrent-user, project-count, or hardware-capacity claim.
- Prefer upstream Docker/Compose deployment patterns and public OCI images. Compatibility manifests may only adapt host-specific details such as bind mounts, platform-owned routing, and secrets; they must not patch application code.
- Run heavy workloads separately on modest hosts. Resource exhaustion is not automatically a MyPaaS defect.
- Additional public endpoints remain HTTP(S) routes owned by MyPaaS/Caddy. The suite does not reinterpret them as permission for arbitrary host-port, TCP, SSH, or UDP exposure.

## Workload classes

| Class | Representative applications | What it exercises |
| --- | --- | --- |
| Simple image | Excalidraw | public OCI image, one HTTP runtime, routing |
| Source Dockerfile | drawDB | Git clone, Docker build, route activation |
| Stateful single service | Uptime Kuma, Meilisearch | named-volume persistence, runtime lifecycle |
| App + database | Umami, Ghost | Compose, SQL service, readiness, env |
| Realtime/stateful app | Directus | persistent storage plus WebSocket-capable routing |
| Developer platform | Forgejo | persistent HTTP application; SSH remains an intentional protocol boundary |
| Multi-service application | NocoDB | app, worker, PostgreSQL, Redis, named volumes |
| Automation platform | n8n | state, background execution, environment configuration |
| Document platform | Paperless-ngx | web app, PostgreSQL, Redis, persistent media/data |
| Knowledge platform | Outline | app, PostgreSQL, Redis, required secrets/auth configuration |
| Agent gateway | OpenClaw | OCI image, persistent state, env-heavy setup, security boundaries |
| Heavy media platform | Immich | multiple runtimes, database, cache, ML process, storage |
| Heavy all-in-one platform | Appsmith | large application footprint and persistent state |
| Multi-route storage | MinIO | primary S3 HTTP API plus a separately routed HTTP Console without an extra host-port binding |

The machine-readable source of truth is [`catalog.json`](catalog.json).

## Additional-route qualification

An application may declare `execution.additionalRoutes` when it needs another HTTP surface. Each entry names a route, an existing Compose service, and an internal container port. `routeSmoke` then checks the derived hostname independently from the primary project hostname.

For a candidate branch that changes compatibility manifests before merge, set:

```bash
MYPAAS_COMPAT_REPO_BRANCH=core/compose-http-routes
```

The runner applies that override only to workloads sourced from the MyPaaS repository. External upstream repositories keep their declared branches. Release catalog defaults remain `main`.

MinIO is the first route-aware candidate:

```text
<project>.<domain>         -> minio:9000  # S3 API
<project>-console.<domain> -> minio:9001  # Web Console
```

The catalog entry being runnable is not itself a `PASS`. Live VM evidence must still prove both endpoints and lifecycle cleanup.

## Result vocabulary

- `untested` — catalogued but not yet run on the target MyPaaS host.
- `pass` — deployment and declared checks completed successfully.
- `fail-unclassified` — automated run failed; human/evidence triage is still required.
- `fail-platform` — reproducible MyPaaS defect.
- `fail-app` — upstream application/configuration problem unrelated to MyPaaS.
- `fail-resource` — host CPU/RAM/disk capacity was the limiting factor.
- `blocked` — the application requires a capability MyPaaS intentionally does not provide.

Live results belong in run artifacts, issues, or pull requests. They should not be committed as permanent capacity claims.
