# MyPaaS Real-World Compatibility Suite

This suite measures whether MyPaaS can correctly host representative real-world open-source workloads. It is a compatibility suite, not a throughput or capacity benchmark.

## Rules

- A `PASS` means the application deployed and its declared smoke/lifecycle checks succeeded on the tested MyPaaS host.
- A failure must be classified as a MyPaaS defect, an upstream application/configuration problem, a host-resource limit, or an intentional platform boundary before it becomes accepted evidence.
- Do not convert a passing fixture into an RPS, concurrent-user, project-count, or hardware-capacity claim.
- Prefer upstream Docker/Compose deployment patterns and public OCI images. Compatibility manifests may adapt host-specific details such as persistence, platform-owned routing, and secrets; they must not patch application code.
- Run heavy workloads separately on modest hosts. Resource exhaustion is not automatically a MyPaaS defect.
- Additional public endpoints remain HTTP(S) routes owned by MyPaaS/Caddy. The suite does not reinterpret them as permission for arbitrary host-port, TCP, SSH, or UDP exposure.

## Workload classes

| Class | Representative applications | What it exercises |
| --- | --- | --- |
| Simple image | Excalidraw | OCI image, one HTTP runtime, routing |
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
| Multi-route storage | MinIO | primary S3 HTTP API plus separately routed HTTP Console without an extra host-port binding |

The machine-readable workload definitions are in [`catalog.json`](catalog.json).

## Additional HTTP-route compatibility

An application may declare `execution.additionalRoutes` when it needs another HTTP surface. Each entry names a route, an existing Compose service, and an internal container port. `routeSmoke` checks the derived hostname independently from the primary project hostname.

The platform contract is defined by [ADR-023](../docs/adr/ADR-023-compose-additional-http-routes.md):

- Compose only;
- maximum four additional routes;
- platform-derived hostnames;
- service/port must exist in the resolved Compose contract;
- no additional host-port publication;
- HTTP(S) only through Caddy;
- no raw TCP, SSH, UDP, or arbitrary-domain routing.

MinIO is the first qualified route-aware workload:

```text
<project>.<domain>         -> minio:9000  # S3 API
<project>-console.<domain> -> minio:9001  # Web Console
```

## MinIO qualification record

MinIO's route-aware path was qualified on VM `172.104.61.180` using exact MyPaaS head:

```text
b35176fd0156c8128e988a2ce3a46693a150c61d
```

before PR #157 merged to `main`.

Verified behavior:

- primary `/minio/health/live` returned HTTP `200`;
- Console returned HTTP `200`;
- primary and Console hostnames were present in Caddy configuration;
- Console `9001` was not published as an additional host port;
- restart and redeploy preserved both routes;
- reconciliation recreated a deliberately removed Console route;
- stop removed both public routes;
- delete cleaned routes, container, volume, and network.

Evidence identifier:

```text
artifacts/pr157-minio-qualification-090944.json
```

The evidence is retained with the PR/qualification record rather than treated as a permanent capacity claim in source documentation.

## Result vocabulary

- `untested` — catalogued but not yet run on the target MyPaaS host.
- `pass` — deployment and declared checks completed successfully.
- `fail-unclassified` — automated run failed; human/evidence triage is still required.
- `fail-platform` — reproducible MyPaaS defect.
- `fail-app` — upstream application/configuration problem unrelated to MyPaaS.
- `fail-resource` — host CPU/RAM/disk capacity was the limiting factor.
- `blocked` — the application requires a capability MyPaaS intentionally does not provide.

Live results belong in run artifacts, issues, pull requests, or release qualification records. They should not be converted into permanent throughput or hardware-capacity claims.

## Current compatibility direction

Use representative OSS applications to discover real product gaps. When a compatibility run fails:

1. classify the failure;
2. fix MyPaaS only if the defect/gap is platform-owned and reusable;
3. rerun only the affected path;
4. document limitations instead of adding speculative platform scope.

Do not turn compatibility work back into broad performance benchmarking or an endless application-count target.
