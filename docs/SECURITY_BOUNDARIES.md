# MyPaaS Security Boundaries

MyPaaS is a single-host deployment platform. Its security model separates the control plane from project workloads, but it does not treat the control-plane API itself as a sandboxed or least-privilege container-engine client.

## Container-engine socket

The API mounts the configured Docker-compatible engine socket because deployment orchestration, runtime lifecycle, inspection, logs, metrics fallback, image management, and Compose operations require engine authority.

Access to that socket is effectively host-level container-engine authority. Compromise of the API process must therefore be treated as compromise of the MyPaaS host boundary, even though the API container drops Linux capabilities and enables `no-new-privileges`.

The production API container therefore:

- is never exposed directly on a public host interface;
- joins only the control-plane network;
- uses `no-new-privileges:true`;
- drops all ambient container capabilities;
- does not pass the engine socket to project workloads.

A socket proxy is intentionally not introduced yet. It would add another privileged component and a large engine-API authorization surface. Revisit that choice only if the deployment engine can be reduced to a small, auditable API subset.

## Network separation

Production uses two external container networks:

- `CONTROL_NETWORK` (default `mypaas-control`) for API, dashboard, Caddy, Cloudflare Tunnel, and PostgreSQL control-plane access;
- `PROJECT_NETWORK` (default `mypaas-projects`) for MyPaaS-managed workloads that need platform-provided connectivity.

The API, dashboard, Caddy, and Cloudflare Tunnel do not join the project network. This prevents ordinary project containers from receiving direct container-network reachability to control-plane services such as the Caddy Admin API.

PostgreSQL intentionally joins both networks while shared PostgreSQL provisioning is enabled. That dual-homing is a product feature, not a general bridge between the networks. Project database credentials remain scoped per provisioned project database/user.

Caddy reaches deployed applications through MyPaaS-managed host port bindings. Fresh installs derive `DOCKER_BIND_HOST` from the private control-network gateway rather than placing Caddy on the project network.

## User Compose execution

Repository Compose files are not passed directly to the engine as trusted configuration. MyPaaS renders the final Compose model and enforces its host-isolation policy before execution.

The security policy rejects host-escape features including privileged containers, host/container namespace sharing, host bind mounts, engine socket mounts, devices, added capabilities, GPUs, custom runtimes, external networks/volumes, unsafe build entitlements, build SSH/secrets, and privileged lifecycle hooks. MyPaaS also strips repository-defined host ports and container names before applying its managed runtime override.

This policy is an important single-host isolation boundary, but it is not equivalent to a VM, microVM, or Kubernetes multi-tenant sandbox.

## Native telemetry daemon

`mypaas-statd` is host-native by design. It reads cgroup v2 data and exposes a bounded protocol over `/run/mypaas/statd.sock`. The API receives the statd socket directory but does not receive host `/proc` or `/sys/fs/cgroup` mounts.

Statd failure is non-fatal: MyPaaS falls back to the Docker-compatible metrics path. Production metrics expose statd availability and fallback/error counters so that fallback is observable instead of silent.

## Trust model

The current production target is a single administrative owner or a small trusted team deploying workloads onto one host. Before treating arbitrary external users as mutually untrusted tenants, additional isolation work would be required, potentially including per-tenant VM/microVM boundaries, stronger database isolation, host resource governance, and a narrower control-plane engine authority model.
