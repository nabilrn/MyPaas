# Deployment Architecture

> Source inspection, deployment execution, lifecycle state, and public route activation.

**Status:** Current  
**Applies to:** `main`  
**Last verified:** 2026-08-13  
**Verified against commit:** `f76102997089a3f1a3b5e7d9f4326582ff22e02c`

---

## Supported deployment modes

| Source | Deploy mode | Execution model | Historical rollback |
| --- | --- | --- | --- |
| Git | `dockerfile` | Build repository Dockerfile, start managed runtime, activate route | Yes |
| Git | `compose` | Render/validate Compose, apply MyPaaS override, start services, route selected service | Yes |
| Git | `static` | Build when needed, publish static release, serve directly with Caddy | Redeploy/roll-forward |
| Registry | `image` | Pull public OCI image, start managed runtime, activate route | Yes |

Nixpacks is an inspection signal, not a deployment mode. A backend/SSR repository without Compose or a production Dockerfile is not silently turned into an opaque generated runtime.

## Common pipeline

```mermaid
flowchart TB
    Trigger["Manual deploy / GitHub webhook / rollback / registry deploy"]
    Persist["Persist deployment + queue work"]
    Prepare["Resolve source and project configuration"]
    Mode{"Deploy mode"}

    Dockerfile["Build Dockerfile image"]
    Compose["Render + security-validate Compose\napply managed override"]
    Static["Run bounded static build when required\npublish atomic release"]
    Image["Pull public OCI image"]

    Runtime["Start or replace runtime"]
    Route["Resolve runtime + activate Caddy route"]
    StaticRoute["Point Caddy at active static release"]
    Commit["Commit active deployment / project state"]
    Cleanup["Clean temporary and superseded state"]
    Failed["Record failure"]

    Trigger --> Persist --> Prepare --> Mode
    Mode -->|dockerfile| Dockerfile --> Runtime
    Mode -->|compose| Compose --> Runtime
    Mode -->|static| Static --> StaticRoute
    Mode -->|image| Image --> Runtime
    Runtime --> Route --> Commit
    StaticRoute --> Commit
    Commit --> Cleanup

    Dockerfile -. error .-> Failed
    Compose -. error .-> Failed
    Static -. error .-> Failed
    Image -. error .-> Failed
    Runtime -. error .-> Failed
    Route -. error .-> Failed
```

The worker model is intentionally bounded rather than an external distributed scheduler. The current platform remains single-host.

## Repository inspection

Project creation and deployment configuration inspect the selected Git source instead of inferring a runtime from marketing-style framework detection.

```mermaid
flowchart TB
    Repo["Selected Git branch"] --> Tree["Inspect repository tree"]
    Tree --> Compose{"Compose candidate?"}
    Compose -->|yes| ComposePlan["Resolve files / overrides / profiles / workdir\nservices / ports / required env / issues"]
    Compose -->|no| Dockerfile{"Dockerfile available?"}
    Dockerfile -->|yes| DockerPlan["Dockerfile deployment"]
    Dockerfile -->|no| Static{"Static site / SPA?"}
    Static -->|yes| StaticPlan["Static deployment"]
    Static -->|no| Nixpacks["Nixpacks provider/framework inspection signal"]
    Nixpacks --> Require["Require explicit production Dockerfile for backend/SSR"]
```

Compose is treated as untrusted configuration. The final rendered model is evaluated before execution, and MyPaaS strips repository-defined host ports and container names before applying its managed runtime override.

## Compose execution boundary

The current policy rejects configuration that could bypass the intended host boundary, including:

- privileged containers;
- host/container namespace sharing;
- Docker/Podman socket mounts;
- host bind mounts;
- devices and GPUs;
- added Linux capabilities;
- custom runtimes;
- external networks and external volumes;
- unsafe build entitlements;
- build SSH and build secrets;
- privileged lifecycle hooks.

Safe engine-managed named volumes are allowed by Compose sanitization. Project environment values are passed through generated project env files. Compose subprocesses receive a fail-closed host-environment allowlist rather than inheriting arbitrary API process credentials.

## Deployment status model

The current API/frontend contract exposes these deployment status values:

```mermaid
stateDiagram-v2
    [*] --> queued
    queued --> cloning: Git source
    queued --> building: mode may skip clone
    cloning --> building
    building --> starting
    starting --> running

    queued --> failed
    cloning --> failed
    building --> failed
    starting --> failed

    running --> stopped
    running --> rolled_back: superseded by rollback

    stopped --> [*]
    rolled_back --> [*]
    failed --> [*]
```

The diagram shows a typical progression while documenting the public status vocabulary: `queued`, `cloning`, `building`, `starting`, `running`, `failed`, `stopped`, and `rolled_back`. Individual deployment modes can skip stages that do not apply.

A rollback action is an explicit deployment trigger; the deployment being superseded can be recorded as `rolled_back` while the rollback operation proceeds through its own deployment path.

Project state is a separate model with `pending`, `building`, `running`, `stopped`, and `crashed` states.

## Runtime replacement and route activation

Dockerfile and image deployments can replace a runtime while keeping routing identity independent of a stable container name.

```mermaid
sequenceDiagram
    participant Worker as Deployment worker
    participant Engine as Docker-compatible engine
    participant New as Replacement runtime
    participant Caddy
    participant DB as PostgreSQL state

    Worker->>Engine: Build or pull image
    Worker->>Engine: Start replacement runtime with allocated host binding
    Engine-->>Worker: Runtime started

    Worker->>Engine: List + batch inspect running containers
    Engine-->>Worker: Published bindings + network memberships
    Worker->>Worker: Match allocated port and verify PROJECT_NETWORK
    Worker->>Engine: Attach replacement to ROUTING_NETWORK with managed alias
    Engine-->>Worker: Routing attachment ready
    Worker->>Caddy: Configure alias:internal-port
    Caddy-->>Worker: Route accepted
    Worker->>DB: Mark deployment active / project running
```

The important property is ordering: route resolution identifies the replacement by its allocated host binding, not by an assumed final container name.

## Production routing path

```mermaid
flowchart LR
    HostPort["Allocated host port"] --> Lookup["Runtime lookup key"]
    Lookup --> Inspect["Engine inspection"]
    Inspect --> Verify["Verify PROJECT_NETWORK"]
    Verify --> Attach["Attach ROUTING_NETWORK\nalias mypaas-port-{port}"]
    Attach --> Caddy["Caddy upstream\nalias:internal-port"]
    Caddy --> Runtime["Application runtime"]
```

Route resolution is fail-closed. MyPaaS does not silently proxy to an arbitrary host IP when the expected runtime cannot be validated.

## Static release path

Static projects bypass the container runtime after any required build step.

```mermaid
sequenceDiagram
    participant Worker as Deployment worker
    participant Builder as Ephemeral Node builder
    participant Files as Static release storage
    participant Caddy

    opt build script required
        Worker->>Builder: Run bounded static build
        Builder-->>Worker: Build output
    end
    Worker->>Files: Publish new release
    Worker->>Files: Atomically switch active release
    Worker->>Caddy: Ensure static route
    Caddy->>Files: Serve active release directly
```

The static Node builder has a simple safety ceiling of 2 GiB RAM, 2 CPUs, and 512 PIDs. The deployment context supplies the outer build timeout.

## Rollback semantics

Dockerfile, Compose, and registry-image deployments keep historical deployment state that can be used for rollback. Static projects recover through a target-revision redeploy/roll-forward model rather than pretending that a removed container runtime exists.

A rollback is itself an explicit deployment action and should remain visible in deployment history/audit behavior.

## Failure handling principles

- persist deployment state before asynchronous work begins;
- reject unsafe Compose input before engine execution;
- do not mark a deployment active before its required runtime/static route is ready;
- fail closed when runtime route identity cannot be resolved;
- preserve explicit failure status and build/error information;
- keep cleanup separate from the correctness of activating the new deployment.

## Related documents

- [Architecture overview](overview.md)
- [Networking and trust boundaries](networking.md)
- [Observability architecture](observability.md)
- [Security boundaries](../SECURITY_BOUNDARIES.md)
