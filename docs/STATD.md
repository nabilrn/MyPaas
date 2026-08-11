# mypaas-statd Integration

`mypaas-statd` is the preferred runtime metrics path for live Dockerfile and Compose projects when the host has the daemon installed and `STATD_SOCKET` configured.

It is intentionally a host-native systemd daemon, not a MyPaas control-plane container and not a GHCR sidecar image. The daemon reads host cgroup v2 counters and serves cached snapshots over a local Unix socket.

## Production installation

Production installation is version-pinned rather than following a mutable Git branch.

The default VM installer targets:

```text
STATD_INSTALL_MODE=release
STATD_VERSION=v0.1.0
STATD_RELEASE_BASE_URL=https://github.com/nabilrn/mypaas-statd/releases/download
```

For the initial v0.1 release, the supported prebuilt artifact is:

```text
mypaas-statd-linux-amd64.tar.gz
SHA256SUMS
```

The installer downloads both files, requires an exact checksum entry for the selected artifact, verifies it with `sha256sum -c`, extracts the bundle, verifies that the bundled `VERSION` equals `STATD_VERSION`, installs the binary and systemd unit, then waits for `/run/mypaas/statd.sock` after enabling the service.

The production environment uses:

```text
STATD_SOCKET=/run/mypaas/statd.sock
```

Set `INSTALL_STATD=false` to skip the daemon and use the Docker-compatible metrics path only.

### Explicit source fallback

Source installation remains available for development, forks, and architectures without a published release artifact:

```text
STATD_INSTALL_MODE=source
STATD_REPO_URL=https://github.com/nabilrn/mypaas-statd.git
STATD_REF=<branch-tag-or-commit>
STATD_DIR=/opt/mypaas-statd
```

Source mode fetches the requested ref and checks out the resolved commit detached. It does not silently continue after a failed pull or reuse an unresolved stale checkout.

## Compatibility contract

The compatibility boundary is intentionally small:

| MyPaaS integration | statd release | Protocol | Host requirement |
| --- | --- | --- | --- |
| current production hardening | `v0.1.0` | v1 | Linux, cgroup v2; prebuilt artifact: amd64 |

Protocol v1 is negotiated on every new Unix connection through `hello`. A protocol mismatch is an integration failure and MyPaaS falls back to the Docker-compatible metrics path rather than fabricating telemetry.

Future statd releases may remain compatible with protocol v1, but MyPaaS production installation must continue to pin a known release until that compatibility has been validated. Do not point production installation at mutable `main`.

## Runtime flow

```plantuml
@startuml
title MyPaas runtime metrics with mypaas-statd

actor "Dashboard / REST / SSE client" as Client
participant "MyPaas API container" as API
participant "Docker-compatible CLI/API\n(Podman socket)" as Docker
participant "mypaas-statd\nsystemd daemon" as Statd
database "cgroup v2\n/sys/fs/cgroup" as Cgroup

Client -> API: GET /api/projects/:id/metrics\nor SSE metrics tick
API -> API: read STATD_SOCKET

alt STATD_SOCKET configured and project has live runtime
  API -> Docker: cold-path inspect\ncontainer PID/service metadata
  Docker --> API: runtime PID + service name + StartedAt
  API -> API: bounded runtime metadata cache
  API -> Statd: hello + register <project-id>:<service>, pid
  Statd -> Cgroup: open/read runtime cgroup
  Statd --> API: cached snapshot\ncpu/memory/pids
  API --> Client: metrics response
else statd disabled, unavailable, static project, or unusable snapshot
  API -> Docker: Docker-compatible metrics fallback
  Docker --> API: metrics
  API --> Client: metrics response
end
@enduml
```

The steady-state statd path avoids spawning Docker/Podman process-discovery commands on every metrics refresh. Single-container and Compose runtime metadata retain `StartedAt` in the bounded handler-owned cache, so uptime does not require steady-state Docker inspection.

The fallback path remains available so rollout is reversible. A cached PID that no longer works is invalidated and rediscovered only after the cached identity actually fails.

Runtime ID format:

```text
<project-uuid>:<service-name>
```

Example:

```text
784283cd-0b53-42eb-bd9a-e1c729e86f41:app
```

## Operational visibility

Fallback is intentionally non-fatal, but it must not be silent.

MyPaaS exposes these low-cardinality Prometheus metrics:

```text
mypaas_statd_available
mypaas_statd_fallback_total
mypaas_statd_snapshot_errors_total
mypaas_statd_registration_errors_total
```

`mypaas_statd_available` reflects the last exercised statd integration state in the API process. Cold `NOT_FOUND` snapshots used to discover/register a runtime are expected protocol flow and are not counted as operational snapshot failures.

Logging is transition-aware:

- the first unavailable state or healthy -> unavailable transition is logged at warning level;
- repeated fallback while already unavailable is debug-only to avoid SSE log spam;
- unavailable -> healthy recovery is logged at info level.

When `STATD_SOCKET` is configured, `scripts/verify-production.sh` also requires the host systemd service to be active, the Unix socket to exist, and the API container to see the mounted socket.

## Benchmark evidence

The accepted Phase 4 real-host benchmark evidence is preserved in the `nabilrn/mypaas-statd` repository:

```text
benchmarks/results/phase4-debian13-podman-2026-08-10/
```

Evidence source:

```text
https://github.com/nabilrn/mypaas-statd/tree/main/benchmarks/results/phase4-debian13-podman-2026-08-10
```

Tested statd commit:

```text
cf8843545ea19ecf9a54049e21b2fe609e49d58d
```

Environment:

- Debian GNU/Linux 13
- kernel `6.12.88+deb13-amd64`
- rootful Podman 5.4.2
- Docker-compatible path `/var/run/docker.sock -> /run/podman/podman.sock`
- no Docker Engine/dockerd

Method:

- one rootful Podman Alpine workload
- warmup: 50 samples
- recorded iterations: 500 per trial
- trials: 3
- baseline: `docker stats --no-stream` through the Podman-backed Docker-compatible command/socket path
- statd path: protocol v1 over `/run/mypaas/statd.sock`, using connect + hello + snapshot per sample

The raw JSON files remain the source of truth. The table below repeats the recorded values without rounding.

| Run | Path | p50_ms | p95_ms | p99_ms | mean_ms | max_ms | wall_seconds | process_spawns |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 1 | Docker-compatible CLI | 41.039357499999994 | 51.9851865 | 57.27985834999998 | 42.131678146 | 68.288171 | 21.067599074 | 500 |
| 1 | mypaas-statd | 0.7803485 | 0.8812820499999998 | 1.0528069899999999 | 0.793765936 | 1.255629 | 0.397203313 | 0 |
| 2 | Docker-compatible CLI | 41.5558045 | 53.618609249999956 | 64.31991137 | 43.004707342 | 72.804908 | 21.504005963 | 500 |
| 2 | mypaas-statd | 0.7968685 | 1.0064879999999998 | 1.1074274999999998 | 0.826836158 | 4.199228 | 0.413757872 | 0 |
| 3 | Docker-compatible CLI | 43.2664495 | 55.993713 | 64.04532325 | 43.954954898000004 | 68.152107 | 21.978995991 | 500 |
| 3 | mypaas-statd | 0.820184 | 1.08817145 | 1.2006058099999999 | 0.870551032 | 1.544936 | 0.43590697 | 0 |

Additional observations from the same evidence:

| Run | Docker CLI child CPU seconds | statd daemon CPU seconds | statd RSS before | statd RSS after | statd voluntary context switches | statd involuntary context switches |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| 1 | 19.248283 | 0.03 | 1859584 | 1859584 | 1973 | 27 |
| 2 | 19.720067 | 0.03 | 1859584 | 1859584 | 1963 | 30 |
| 3 | 19.956553 | 0.03999999999999998 | 1859584 | 1859584 | 1982 | 12 |

Correctness checks in the evidence compared statd snapshots against raw cgroup v2 files for memory, PID, and CPU counter semantics. Runtime disappearance was also tested: statd returned `NOT_FOUND` after the container stopped and the daemon remained active.

Protocol v1 does not expose a sampler timestamp, so MyPaas documentation must not claim metric freshness or metric age from these benchmark files.

## Publish model

`mypaas-statd` remains host-native. Publishing it as a required container image would require privileged host PID/cgroup mounts and would make the validated model more complex.

The supported distribution order is:

1. versioned GitHub Release artifact + `SHA256SUMS` for production installation;
2. Debian package later if packaging maturity warrants it;
3. explicit source installation for development/forks/unsupported prebuilt architectures.

MyPaas API and dashboard remain GHCR images. `mypaas-statd` remains a host tool consumed through `STATD_SOCKET`.
