# Architecture Overview

> System context and major responsibilities for the current MyPaaS control plane.

**Status:** Current  
**Applies to:** `main`  
**Last verified:** 2026-08-28  
**Verified against commit:** `e12f47dd3249e2fdd69df352852ff3c9c3489245`

---

## System context

```mermaid
flowchart LR
    Operator["Owner / collaborator"] --> Browser["Browser"]
    GitHub["GitHub"] --> Webhook["Webhook"]
    Automation["CLI / MCP"] --> API["Go API"]

    Browser --> Edge["Configured public delivery path"]
    Webhook --> Edge
    Edge --> Caddy["Caddy"]
    Caddy --> Dashboard["SvelteKit dashboard"]
    Caddy --> API

    API --> DB[("PostgreSQL")]
    API --> Engine["Docker-compatible engine contract\nPodman default on fresh hosts"]
    API --> CaddyAdmin["Caddy Admin Unix socket"]
    API --> Statd["optional mypaas-statd Unix socket"]

    Engine --> Apps["Container-backed projects"]
    Caddy --> Apps
    Caddy --> Static["Static releases"]
```

Caddy is the front door for dashboard/API/webhook traffic, static projects, primary project routes, and bounded additional Compose HTTP routes.

## Control-plane responsibilities

### Go API

The API is the orchestration authority. Its responsibilities include:

- GitHub OAuth and authorization;
- project configuration and deployment state;
- repository inspection;
- Dockerfile, Compose, static, and OCI image deployments;
- bounded private-registry authentication for image-mode pulls;
- lifecycle actions and rollback;
- encrypted environment-variable management;
- resource quotas and settings;
- primary and additional Caddy route reconciliation;
- project-scoped persistent-storage management;
- backups and VM migration;
- DB Studio and optional shared PostgreSQL provisioning;
- CLI/MCP endpoints;
- runtime metrics integration and host telemetry.

Because orchestration requires the Docker-compatible engine socket, compromise of the API must be treated as compromise of the host boundary.

### Dashboard

The SvelteKit dashboard is an operator UI over the API. It does not directly orchestrate the engine, configure Caddy, or read host cgroups.

### PostgreSQL

PostgreSQL stores control-plane state. It is also optionally used to provision project-specific shared databases and users, so it intentionally participates in both control and project networks.

### Caddy

Caddy has two distinct roles:

1. stable ingress for dashboard/API/webhook traffic;
2. data-plane routing and static-file serving for projects.

Project routing includes the primary project hostname and, for eligible Compose projects, up to four additional platform-derived HTTP hostnames. Additional routes never grant access to the Caddy Admin socket.

The production Admin API is reachable only over `/run/mypaas/caddy-admin.sock`, shared with the API container through `/run/mypaas`.

### cloudflared

`cloudflared` is the default production outbound tunnel client. It joins the control network and sends incoming tunnel traffic to Caddy. The product model remains a configured public delivery path; Cloudflare-specific provider behavior is not treated as application capacity.

### mypaas-statd

`mypaas-statd` is an optional host-native systemd daemon. It reads bounded host/cgroup telemetry and exposes snapshots over `/run/mypaas/statd.sock`. Runtime metrics fall back to the Docker-compatible engine path when statd is disabled or unavailable.

## Request paths

### Dashboard and API

```mermaid
sequenceDiagram
    actor User
    participant Edge as Public delivery path
    participant Caddy
    participant UI as SvelteKit dashboard
    participant API as Go API

    User->>Edge: HTTPS request
    Edge->>Caddy: Request
    alt dashboard route
        Caddy->>UI: Proxy request
        UI-->>Caddy: HTML / app response
    else /api/* or /webhook/*
        Caddy->>API: Proxy request
        API-->>Caddy: API response
    end
    Caddy-->>Edge: Response
    Edge-->>User: HTTPS response
```

### Container-backed project

```mermaid
sequenceDiagram
    actor Client
    participant Edge as Public delivery path
    participant Caddy
    participant App as Routed runtime/service

    Client->>Edge: Project hostname request
    Edge->>Caddy: Request
    Caddy->>App: ROUTING_NETWORK alias + internal port
    App-->>Caddy: Application response
    Caddy-->>Edge: Response
    Edge-->>Client: HTTPS response
```

Primary routes and additional Compose HTTP routes use the same bounded routing-network data plane. Additional routes may target another port on the same container or a different declared Compose service without publishing another host port.

### Static project

```mermaid
sequenceDiagram
    actor Client
    participant Edge as Public delivery path
    participant Caddy
    participant Files as Static release directory

    Client->>Edge: Project request
    Edge->>Caddy: Request
    Caddy->>Files: Read active static release
    Files-->>Caddy: File content
    Caddy-->>Edge: Response
    Edge-->>Client: HTTPS response
```

Static projects have no persistent application container and therefore do not use runtime container metrics.

## Engine portability

MyPaaS does not maintain separate Docker-specific and Podman-specific orchestration implementations.

```mermaid
flowchart LR
    Backend["MyPaaS backend"] --> DockerCmd["docker / docker compose command surface"]
    DockerCmd --> Compat["Docker-compatible socket"]
    Compat --> Podman["Rootful Podman\ndefault"]
    Compat --> DockerEngine["Docker Engine\ncompatibility mode"]
```

Fresh supported installations default to rootful Podman. Production normalizes the selected host engine socket into `/var/run/docker.sock` inside the API container so the orchestration command contract remains stable.

## Registry boundary

OCI image-mode projects support:

- anonymous pulls from compatible registries;
- one optional installation-level credential scoped to a configured registry host.

The authenticated pull uses an isolated temporary Docker configuration and does not modify the host user's persistent Docker credentials. Compose image pulls do not inherit this credential automatically.

No registry proxy, mirror, pull-through cache, generic credential broker, or multi-registry credential UI is provided.

## Scope

The current architecture intentionally does not provide:

- Kubernetes scheduling;
- multi-node placement or HA control-plane failover;
- automatic horizontal autoscaling;
- hostile-tenant VM/microVM isolation;
- generic raw TCP, SSH, or UDP public routing;
- arbitrary per-route custom domains;
- registry proxy/cache/mirror behavior;
- a second native Podman orchestration backend.

Those are product/architecture changes, not hidden assumptions of the current implementation.

## Related documents

- [Networking and trust boundaries](networking.md)
- [Deployment architecture](deployment.md)
- [Observability architecture](observability.md)
- [Security boundaries](../SECURITY_BOUNDARIES.md)
- [mypaas-statd integration](../STATD.md)
- [ADR-022: bounded private-registry authentication](../adr/ADR-022-private-registry-auth.md)
- [ADR-023: bounded additional Compose HTTP routes](../adr/ADR-023-compose-additional-http-routes.md)
