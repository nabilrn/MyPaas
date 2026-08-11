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

- `CONTROL_NETWORK` (default `mypaas-control`) for API, dashboard, Cloudflare Tunnel, Caddy control-plane connectivity, and PostgreSQL control-plane access;
- `PROJECT_NETWORK` (default `mypaas-projects`) for MyPaaS-managed workloads, shared PostgreSQL access, and Caddy's application data plane.

The API, dashboard, and Cloudflare Tunnel do not join the project network. Ordinary project containers therefore do not receive direct container-network reachability to those control-plane services.

Caddy intentionally joins both networks because it is the reverse-proxy data-plane endpoint. That dual-homing does not expose its administrative API: production disables TCP admin access and configures Caddy Admin on the shared Unix socket `/run/mypaas/caddy-admin.sock`. The API and Caddy share `/run/mypaas`; project workloads do not receive that host mount.

PostgreSQL also intentionally joins both networks while shared PostgreSQL provisioning is enabled. That dual-homing is a product feature, not a general bridge between the networks. Project database credentials remain scoped per provisioned project database/user.

Managed application ports remain bound to the private project-network gateway. Caddy reaches those ports from its project-network attachment while its API/dashboard upstreams remain available through the control network.

## Caddy administration

The production Caddyfile configures:

```text
admin unix//run/mypaas/caddy-admin.sock
```

There is no production host/container mapping for TCP port `2019`. The Go Caddy client supports the Unix admin endpoint directly with an HTTP transport whose dialer connects to the Unix socket.

This separates Caddy's required project-facing HTTP data plane from its privileged configuration plane. Project workloads may reach Caddy's normal HTTP listener when network policy allows it, but they do not receive the Caddy Admin socket.

## User Compose execution

Repository Compose files are not passed directly to the engine as trusted configuration. MyPaaS renders the final Compose model and enforces its host-isolation policy before execution.

The security policy rejects host-escape features including privileged containers, host/container namespace sharing, host bind mounts, engine socket mounts, devices, added capabilities, GPUs, custom runtimes, external networks/volumes, unsafe build entitlements, build SSH/secrets, and privileged lifecycle hooks. MyPaaS also strips repository-defined host ports and container names before applying its managed runtime override.

This policy is an important single-host isolation boundary, but it is not equivalent to a VM, microVM, or Kubernetes multi-tenant sandbox.

## Native telemetry daemon

`mypaas-statd` is host-native by design. It reads cgroup v2 data and exposes a bounded protocol over `/run/mypaas/statd.sock`. The API receives the statd socket directory but does not receive host `/proc` or `/sys/fs/cgroup` mounts.

Statd failure is non-fatal: MyPaaS falls back to the Docker-compatible metrics path. Production metrics expose statd availability and fallback/error counters so that fallback is observable instead of silent.

## Trust model

The current production target is a single administrative owner or a small trusted team deploying workloads onto one host. Before treating arbitrary external users as mutually untrusted tenants, additional isolation work would be required, potentially including per-tenant VM/microVM boundaries, stronger database isolation, host resource governance, and a narrower control-plane engine authority model.
