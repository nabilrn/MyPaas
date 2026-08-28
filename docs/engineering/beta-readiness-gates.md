# Beta Runtime Verification

This document records controlled checks used while hardening the MyPaaS beta. It is a reliability and qualification record for platform behavior, not a capacity benchmark.

## What these checks mean

A passing check means the tested MyPaaS behavior worked in the named controlled scenario. It does **not** establish a universal project count, concurrent-user count, RPS ceiling, application size, or minimum production server specification.

Application capacity depends on the application itself and on available CPU, memory, storage, network, database behavior, build requirements, and other workloads sharing the host.

## Retained runtime checks

| Area | What was verified |
| --- | --- |
| Update / release safety | Controlled update, missing-image safety, rollback, health verification, build identity, and route reconciliation. |
| Backup / restore | Fresh-host restoration of control-plane data, configuration, static artifacts, managed persistent data, routes, encrypted environment usability, deployment history, and DB Studio state. |
| Concurrent deployment reliability | Concurrent create/deploy/redeploy, intentional failure, protected routes, webhook activity, and final runtime/port consistency. |
| Image and cache retention | Cleanup scope, protection of running/rollback images and persistent volumes, dry-run/apply behavior, and post-cleanup deployment/rollback. |
| Create Project contract | Static, Dockerfile, Compose, subdirectory, registry image, required environment, missing-port, stale-analysis, timeout, and invalid-repository behavior. |
| DB Studio | PostgreSQL, MySQL, and MariaDB Compose connectivity, project-network resolution, read-only default, and expiring write sessions. |
| Docker/Podman runtime contract | Production Compose rendering, explicit routing-network aliases, rootful Podman compatibility, and stable Docker-compatible API socket behavior. |
| Compose additional HTTP routes | Primary + secondary route activation, no extra secondary host-port publication, lifecycle persistence, reconciliation recovery, and cleanup. |

These checks remain useful regression targets only when their corresponding runtime paths change materially.

## Final PR #157 qualification

PR #157 added bounded additional HTTP routes for Compose projects and used MinIO as the concrete product qualification path.

Target VM:

```text
172.104.61.180
```

Exact qualified head:

```text
b35176fd0156c8128e988a2ce3a46693a150c61d
```

Source-side gates on that head included backend tests, Go race detection, frontend checks/build, script regressions, compatibility runner/catalog checks, production Compose rendering, Docker routing-alias checks, and rootful Podman compatibility.

The final real-VM qualification passed all required behavior:

- production deployment automatically selected the live host Podman socket and exposed it to the API through the stable in-container `/var/run/docker.sock` path;
- primary MinIO health returned HTTP `200`;
- Console returned HTTP `200` and its hostname existed in Caddy configuration;
- Console port `9001` was not published as an additional host port;
- restart preserved primary and Console routes;
- redeploy preserved primary and Console routes;
- after deliberately removing the Console route from Caddy, reconciliation recreated it;
- stop removed both public routes;
- delete removed routes, container, volume, and network.

Evidence identifier:

```text
artifacts/pr157-minio-qualification-090944.json
```

The evidence was retained with the PR qualification record. PR #157 then merged to `main` as:

```text
e12f47dd3249e2fdd69df352852ff3c9c3489245
```

This qualification establishes the correctness of the bounded additional-HTTP-route behavior in that scenario. It does not establish general network throughput or host capacity.

## Historical defects found during qualification

Qualification work has found real product defects that were subsequently fixed, including:

- Compose empty-health handling causing deployment timeout and port-state divergence;
- Dockerfile side-by-side redeploy attempting to reuse an active runtime port;
- a DB Studio test fixture incorrectly treating an immutable commit SHA as a branch;
- production deploy workers targeting a stale Docker socket path on a rootful Podman host;
- initial successful Compose deployment completing before the secondary Caddy route had been synchronously reconciled.

The last two defects were discovered during the first PR #157 VM run and corrected before the passing rerun.

These defects are retained as regression history. They should not be converted into broader performance or scalability claims.

## Evidence handling

Generated VM logs, screenshots, load output, host snapshots, and other run artifacts are test evidence, not product documentation. They should normally stay outside the source tree or be attached to the specific issue, pull request, or release investigation that needs them.

For a controlled runtime check, record only what is needed to reproduce and interpret the result:

- tested Git SHA;
- relevant environment shape;
- scenario and expected behavior;
- pass/fail result;
- failure details when applicable.

Never store credentials, decrypted environment values, cookies, or other secrets in evidence.

## Regression rule

Re-run a controlled runtime check when a change materially touches the behavior it covers. Do not re-run unrelated scenarios merely to preserve a historical gate count.

A failure caused by application resource demand or insufficient host capacity must be classified separately from a MyPaaS correctness failure unless evidence shows the platform itself caused the fault.

For compatibility-driven development, rerun the affected application path after a platform fix rather than restarting broad benchmark matrices.

## Current boundary

MyPaaS remains a single-host self-hosted platform for an owner developer or a small trusted team. It is not a multi-node HA scheduler, a hostile multi-tenant isolation boundary, or a guarantee that arbitrary workloads will fit on a given server.
