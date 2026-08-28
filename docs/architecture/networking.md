# Networking and Trust Boundaries

> Production network membership, route activation, and privileged control paths.

**Status:** Current  
**Applies to:** `main`  
**Last verified:** 2026-08-28  
**Verified against commit:** `e12f47dd3249e2fdd69df352852ff3c9c3489245`

---

## Network membership

Production creates three distinct external networks.

| Component | `CONTROL_NETWORK` | `PROJECT_NETWORK` | `ROUTING_NETWORK` |
| --- | :---: | :---: | :---: |
| API | ✓ | — | — |
| Dashboard | ✓ | — | — |
| cloudflared | ✓ | — | — |
| PostgreSQL | ✓ | ✓ | — |
| Caddy | ✓ | — | ✓ |
| Normal project workload | — | ✓ | — |
| Publicly routed runtime/service | — | ✓ | ✓ |

The default names are `mypaas-control`, `mypaas-projects`, and `mypaas-routing`. Production validation requires all three names to be distinct.

## Topology

```mermaid
flowchart TB
    Internet["Internet"] --> Delivery["Configured public delivery path"] --> Caddy["Caddy"]

    subgraph Control["CONTROL_NETWORK"]
        Caddy
        Dashboard["Dashboard"]
        API["Go API"]
        Postgres[("PostgreSQL")]
    end

    subgraph Project["PROJECT_NETWORK"]
        Workload["Ordinary project workload"]
        RoutedPrimary["Primary routed runtime"]
        RoutedAdditional["Compose service routed by additional HTTP route"]
        SharedDB["PostgreSQL project-side membership"]
    end

    subgraph Routing["ROUTING_NETWORK"]
        CaddyRoute["Caddy routing-side membership"]
        PrimaryAlias["managed primary route alias"]
        AdditionalAlias["managed additional HTTP-route alias"]
    end

    Caddy --> Dashboard
    Caddy --> API
    Workload --> SharedDB
    RoutedPrimary --> SharedDB
    RoutedAdditional --> SharedDB
    CaddyRoute --> PrimaryAlias
    CaddyRoute --> AdditionalAlias

    Caddy -. "same Caddy container" .-> CaddyRoute
    Postgres -. "same PostgreSQL container" .-> SharedDB
    RoutedPrimary -. "attached when primary route is active" .-> PrimaryAlias
    RoutedAdditional -. "attached only when declared additional route requires it" .-> AdditionalAlias
```

The duplicated labels represent network interfaces on the same logical containers; they are not duplicate services.

## Why three networks

### Control network

The control network carries platform communication. Project workloads are not attached to it, so a normal workload does not automatically receive container-network adjacency to the API, dashboard, cloudflared, or Caddy's control-side membership.

### Project network

The project network is the default workload network. It permits project-service communication and optional shared PostgreSQL access. Caddy is intentionally not a general member of this network.

### Routing network

The routing network is the bounded application data plane. Caddy joins it permanently; a project runtime/service joins it only when MyPaaS activates a public route that needs that target.

This keeps general project workloads away from Caddy while still allowing explicit primary and additional HTTP routes to reach their intended service over container networking.

## Primary route activation

Allocated host ports remain deterministic runtime identity keys for normal primary container-backed routes. They are not the normal production traffic path.

```mermaid
sequenceDiagram
    participant API as MyPaaS API
    participant Engine as Docker-compatible engine
    participant Runtime as Project runtime
    participant Caddy

    API->>Engine: List running containers
    Engine-->>API: Runtime IDs
    API->>Engine: Batch inspect candidates
    Engine-->>API: Published ports + networks
    API->>API: Match allocated host port
    API->>API: Verify PROJECT_NETWORK membership
    API->>API: Derive internal container port
    API->>Engine: Ensure ROUTING_NETWORK managed alias
    Engine-->>API: Attachment ready
    API->>Caddy: Configure alias:internal-port
    Caddy->>Runtime: Proxy over ROUTING_NETWORK
```

The primary managed alias remains based on the allocated project port. Route resolution fails closed if MyPaaS cannot identify the intended running container, verify project-network membership, derive the internal port, or establish the managed routing attachment.

## Additional Compose HTTP routes

Compose projects may declare up to four additional HTTP routes. Each route uses a derived hostname and targets an existing Compose service plus an explicitly declared internal TCP port.

```mermaid
sequenceDiagram
    participant API as MyPaaS API
    participant Engine as Docker-compatible engine
    participant Service as Compose service
    participant Caddy

    API->>API: Validate persisted route contract
    API->>Engine: Resolve target Compose container/service
    Engine-->>API: Container networks + declared runtime target

    alt target already has usable MyPaaS routing attachment
        API->>API: Reuse managed routing attachment
    else target needs routing membership
        API->>Engine: Attach service to ROUTING_NETWORK with managed HTTP-route alias
        Engine-->>API: Attachment ready
    end

    API->>Caddy: Configure derived host -> alias:internal-port
    Caddy->>Service: Proxy over ROUTING_NETWORK
```

Additional-route rules:

- the public hostname is `<project>-<route>.<PUBLIC_DOMAIN>`;
- the target service/port must exist in the resolved Compose contract;
- no extra host port is allocated or published;
- the route is HTTP(S)-only through Caddy;
- no raw TCP, SSH, UDP, or arbitrary-domain forwarding is implied;
- stop/delete remove public routes;
- reconciliation recreates missing routes for eligible running projects.

When multiple HTTP surfaces live on the same container, such as MinIO `9000` and `9001`, MyPaaS can reuse the same routing-network attachment while Caddy targets different internal ports.

When an additional route targets another eligible Compose service, only that service receives the routing-network attachment required for the route.

## Why the primary host port still exists

The published host binding remains useful for primary runtime identity and existing lifecycle/accounting semantics.

```mermaid
flowchart LR
    Port["Allocated host port"] --> Identity["Primary runtime lookup"]
    PrimaryAlias["Managed primary alias"] --> PrimaryTraffic["Primary Caddy traffic"]
    AdditionalAlias["Managed additional alias"] --> AdditionalTraffic["Additional HTTP-route traffic"]
```

Additional Compose HTTP routes do not create another host-port identity. Their target is resolved from the persisted Compose route contract and container/service state.

## Caddy control plane

Caddy's production Admin API is not exposed on TCP port `2019`.

```mermaid
flowchart LR
    API["Go API"] --> Socket["/run/mypaas/caddy-admin.sock"]
    Socket --> Caddy["Caddy Admin API"]
    Runtime["Routed workload"] --> Listener["Caddy HTTP data plane"]
    Runtime -. "no socket mount" .-> Blocked["No Caddy Admin access"]
```

The API and Caddy share `/run/mypaas`; project workloads do not receive that host mount.

## Engine authority boundary

The strongest trust boundary in the current architecture is the Docker-compatible engine socket.

```mermaid
flowchart TB
    subgraph Control["Control plane"]
        API["API container\ncap_drop: ALL\nno-new-privileges"]
    end

    Socket["Docker-compatible engine socket\nHOST AUTHORITY"]
    Engine["Docker Engine / rootful Podman"]
    Workloads["Project workloads"]

    API --> Socket --> Engine --> Workloads
    Workloads -. "socket is never passed through" .-> Denied["No direct engine authority"]
```

Production maps the selected host engine socket into the API at the stable in-container path `/var/run/docker.sock`. Container hardening reduces ambient Linux privilege but does not neutralize engine authority.

## PostgreSQL dual-homing

PostgreSQL intentionally joins both control and project networks because optional shared PostgreSQL provisioning is a platform feature. This is not an accidental topology leak.

That tradeoff means network segmentation should not be described as absolute tenant isolation. Database isolation also depends on generated database/user credentials and PostgreSQL authorization.

## Security invariants

Current production assumptions are:

- the three network names are distinct;
- API, dashboard, and cloudflared are not project-network members;
- API is not a routing-network member;
- Caddy is not a general project-network member;
- only explicitly routed runtimes/services gain routing-network membership;
- additional Compose routes do not publish extra host ports;
- project workloads never receive the engine socket;
- project workloads never receive the Caddy Admin Unix socket;
- route resolution fails closed instead of proxying to an arbitrary fallback address.

These invariants reduce unintended adjacency. They do not replace VM/microVM isolation for mutually hostile tenants.

## Related documents

- [Architecture overview](overview.md)
- [Deployment architecture](deployment.md)
- [Security boundaries](../SECURITY_BOUNDARIES.md)
- [ADR-023: bounded additional Compose HTTP routes](../adr/ADR-023-compose-additional-http-routes.md)
