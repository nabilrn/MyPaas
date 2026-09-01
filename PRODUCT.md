# Product

MyPaaS is a **single-host self-hosted deployment platform** for an owner developer or a small trusted team.

Its job is to make common deployment and operations work repeatable on a Linux server without hiding ownership of the host, container engine, data, or routing path.

## Current scope

MyPaaS can:

- deploy Git repositories with Dockerfile, Docker Compose, or static output;
- list repositories available to the connected GitHub account from the New Project form and deploy private GitHub repositories;
- deploy OCI-image projects with anonymous pulls or one bounded installation-level credential for a configured registry;
- inspect repository structure and configuration before creation, including base-directory / monorepo layouts;
- manage encrypted environment variables and project resource settings;
- manage primary project routing through Caddy;
- provide up to four derived additional HTTP routes for Compose projects, targeting declared service ports without publishing extra host ports;
- expose deployment history, logs, metrics, and lifecycle actions;
- provide a short-lived owner-only host shell for trusted VM operators;
- support rollback for compatible container-backed deployments;
- provide project-scoped persistent storage and owned-resource cleanup;
- provide PostgreSQL provisioning, DB Studio Lite, backups, restore, and migration tooling;
- expose CLI, REST API, webhooks, and an optional local MCP bridge;
- provide OSS application templates backed by a compatibility catalog;
- use optional `mypaas-statd` telemetry with an engine-metrics fallback.

Fresh supported Linux installations default to rootful Podman through the Docker-compatible command/socket contract used by the control plane. Docker Engine remains an explicit compatibility mode.

## GitHub repository access

GitHub OAuth is also the repository connection used by the project picker. The picker lists repositories the signed-in administrator can access, including private repositories, and applies the selected repository's clone URL and default branch to project creation.

The control plane encrypts the OAuth access token at rest. Repository Git operations receive it only through an ephemeral Git HTTP authorization header; it is not stored in a project checkout, passed into a workload, or included in command arguments or logs.

## Bounded registry authentication

Image-mode deployments can optionally authenticate to **one configured registry host** using installation-level credentials. The credential is applied only when the requested image host matches the configured registry and is isolated from the operator's persistent Docker credential store.

This is intentionally not a general credential manager:

- Compose image pulls do not receive these credentials automatically;
- no registry proxy, mirror, pull-through cache, credential broker, or registry UI is provided;
- multiple per-project/per-registry credentials are outside the current contract.

See [`docs/adr/ADR-022-private-registry-auth.md`](docs/adr/ADR-022-private-registry-auth.md).

## Bounded Compose HTTP routes

A Compose project may declare up to four additional HTTP routes when a real application needs multiple HTTP surfaces.

Each route:

- uses a platform-derived hostname `<project>-<route>.<public-domain>`;
- targets an existing Compose service and a TCP port declared by `ports` or `expose`;
- is routed internally through Caddy and `ROUTING_NETWORK`;
- does not allocate or publish an additional host port.

The first version is deliberately HTTP-only, Compose-only, and immutable after first deployment. Raw TCP, SSH, UDP, arbitrary hostnames, and arbitrary public port exposure are not part of this capability.

MinIO's S3 API + Console path is the first real-VM-qualified use of this primitive. See [`docs/adr/ADR-023-compose-additional-http-routes.md`](docs/adr/ADR-023-compose-additional-http-routes.md).

## Boundaries

MyPaaS currently does **not** provide:

- multi-node scheduling or cluster orchestration;
- control-plane high availability;
- hostile multi-tenant isolation;
- automatic horizontal application scaling;
- generic raw TCP/SSH/UDP public routing;
- arbitrary custom domains per additional route;
- registry proxy/cache/mirror behavior or a general multi-registry credential manager;
- supported in-place Docker-to-Podman state migration;
- a universal application-capacity guarantee.

Application and build capacity depend on the workload and on the CPU, memory, storage, network, database, and other processes sharing the host. Project count, concurrent users, RPS, or a particular VM size are not fixed capabilities of MyPaaS.

## Product principles

1. Keep deployment state and failure state explicit.
2. Prefer deterministic configuration over guessed automation.
3. Do not replace a healthy runtime with a failed deployment.
4. Keep recovery, rollback, backup, and cleanup operable.
5. Prefer simple single-host mechanisms over distributed-system complexity without a demonstrated need.
6. Add platform primitives only when a real application demonstrates the gap.
7. Keep public claims narrower than what the implementation and current evidence support.

## Security and operations

The MyPaaS API has privileged container-engine authority. The current trust model is therefore an owner or small trusted team, not mutually hostile tenants.

Whitelisted accounts are owners. The first account is the master account and cannot be removed; owner accounts cannot remove other owner accounts. The owner-only Shell page exposes a short-lived host shell for trusted VM operators. It is a control-plane operation, not public SSH or generic project TCP/SSH forwarding.

Host sizing, application architecture, provider availability, infrastructure security, and off-host recovery material remain operator responsibilities.

See [`docs/SECURITY_BOUNDARIES.md`](docs/SECURITY_BOUNDARIES.md) for the detailed trust model.
