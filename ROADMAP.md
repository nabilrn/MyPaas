# MyPaaS Product Roadmap

MyPaaS is a simple single-host PaaS for an owner developer or small trusted team. The roadmap optimizes for real application deployment and operation, not benchmark breadth or distributed orchestration.

Current code, tests, accepted ADRs, and real-VM qualification remain the source of truth for implemented behavior. This file defines product direction only.

## Product rule

Prioritize work that is:

- owned by the MyPaaS platform;
- useful for real application deployment or operation;
- reversible and understandable on one host;
- compatible with the existing Docker/Compose/static/OCI deployment engine;
- directly visible as better deployment, database, compatibility, or dashboard UX;
- justified by a real application onboarding problem or concrete product defect.

Do not add architecture merely to create a future scaling story.

## KEEP

These are established product capabilities and should be maintained rather than redesigned without a concrete defect:

- project create, update, delete, redeploy, start, stop, restart, and rollback boundaries;
- Dockerfile deployment;
- Docker Compose deployment, including multi-service inspection and main-service selection;
- static deployment through Caddy;
- OCI image deployment with anonymous pulls and one bounded configured registry credential for image mode;
- public routing through the configured Caddy / Cloudflare delivery path;
- bounded Compose additional HTTP routes with derived hostnames and no extra host-port publication;
- encrypted project environment variables and repository environment discovery;
- project-scoped persistent storage and owned-resource cleanup;
- deployment status, history, logs, metrics, and bounded deployment concurrency;
- project/user resource guardrails;
- shared PostgreSQL provisioning with generated credentials;
- DB Studio safe row browsing/editing and schema metadata for PostgreSQL, MySQL, and MariaDB;
- backup, restore-drill, and migration tooling within documented boundaries;
- the real-world compatibility catalog and runner;
- qualified OSS application templates;
- the existing dashboard information architecture.

## DELIVERED PRODUCTIZATION

### OSS App Templates v1

Delivered on `main`.

The template catalog turns qualified deployment patterns into a user-facing install path without creating a second deployment engine. Initial templates cover representative image, stateful, database-backed, multi-service, and multi-route applications.

### DB Studio schema metadata and ERD

Delivered on `main`.

DB Studio extends beyond row browsing with schema metadata useful for understanding relationships while remaining intentionally smaller and safer than a full SQL IDE.

### Bounded private-registry authentication

Delivered on `main`.

MyPaaS can authenticate OCI image-mode pulls to one configured registry without modifying the operator's persistent Docker credential store. Credentials are scoped by registry host and an isolated temporary Docker configuration is used for login/pull. Pull failures distinguish authentication, permission, rate-limit, and missing-image cases where registry output supports that classification.

The implementation intentionally does not add a registry proxy, pull-through cache, or credential inheritance into project Compose environments. See ADR-022.

### Compatibility status in product UX

Delivered on `main`.

Installable templates expose a stable catalog identity and deployment-pattern guidance without fabricating a live compatibility result. The dashboard can surface persistent-storage expectations, resource guidance, setup requirements, and known platform boundaries.

Compatibility status is not a throughput, concurrent-user, hardware-capacity, or production-readiness claim. Live evidence remains in compatibility run artifacts, issues, or pull requests.

### Bounded Compose additional HTTP routes

Delivered on `main` by PR #157 and real-VM-qualified on exact head `b35176fd0156c8128e988a2ce3a46693a150c61d` before merge.

The primitive supports up to four additional HTTP routes for Compose projects. Routes use platform-derived hostnames, target declared Compose service ports, reuse the routing-network data plane, and do not expose additional host ports.

MinIO is the first qualified application using this capability:

- primary project route -> MinIO S3 API on `9000`;
- derived `console` route -> MinIO Console on `9001`;
- restart/redeploy preserved both routes;
- reconciliation recreated a deliberately removed Console route;
- stop/delete removed public routes and owned runtime resources.

This feature intentionally does not provide raw TCP, SSH, UDP, arbitrary route hostnames, or generic public port forwarding. See ADR-023.

## IMPLEMENT NEXT

There is no broad feature program required before the current beta can be evaluated as a product.

The next work should be **compatibility-driven only**:

- deploy representative real OSS applications from the compatibility catalog;
- classify failures before changing MyPaaS;
- fix only reusable platform-owned gaps or real correctness defects;
- extend templates/env generation only when an application requires a reusable primitive;
- keep documentation and landing-page claims aligned with verified behavior.

Examples of valid template/env improvements when demonstrated by a real app:

- generated secrets;
- generated public URL/host values;
- required/default env fields;
- explicit resource warnings;
- documented persistent-storage requirements.

Do not add template-specific application code patches.

## DEFER

These remain possible future product ideas, but are not active implementation targets without a concrete application requirement.

### Static asset cache policy

Potentially useful for known immutable build assets, but incorrect caching is a correctness bug. Revisit only as a narrow delivery feature with framework-safe semantics.

### Registry pull cache / mirror

Current bounded authentication and diagnostics are sufficient. A registry cache introduces storage, garbage-collection, and freshness responsibilities that are not justified by the current product scope.

### Caddy-specific delivery telemetry

Existing application logs, runtime metrics, and optional Cloudflare analytics are sufficient for current operation. Add Caddy-specific telemetry only if it answers a concrete operator question that current observability cannot answer.

### Restore UI

The destructive recovery path should remain operator-oriented until the existing backup/restore workflow has a clear, safe UI contract.

### Multi-database DB Studio selector

Useful for Compose projects that intentionally contain multiple SQL databases. Keep any future implementation project-local and bounded to PostgreSQL/MySQL/MariaDB; do not turn it into a global DBA connection manager.

## OUT OF TARGET FEATURE SCOPE

The following are not active MyPaaS product targets:

- repeated throughput benchmarking or broad k6 matrices;
- direct-vs-tunnel performance retesting without a concrete product defect;
- kernel, sysctl, or NIC tuning programs;
- control-plane micro-optimization without profiling evidence of a user-visible bottleneck;
- application replica experiments as a roadmap item;
- generic Cloudflare tuning without a product feature;
- vague `production ready` milestones that cannot be tested as a specific capability;
- Kubernetes, Nomad, Swarm, service mesh, distributed scheduler, or multi-node orchestration plans;
- hostile multi-tenant isolation claims;
- generic raw TCP/SSH/UDP routing or arbitrary public port forwarding;
- user-count, RPS, or hardware-capacity promises derived from compatibility fixtures.

Historical experiments may remain in Git history, closed pull requests, historical PRD/release notes, or archived evidence. They must not be presented as current product direction.

## DB Studio boundary

DB Studio should remain a focused application-data tool.

Keep improving only when a real project requires it:

- table/column detail;
- foreign-key relationships;
- indexes and constraints;
- ERD/schema graph;
- safe row editing;
- project-local database selection when multiple supported SQL services are present.

Keep out of scope unless the product direction changes:

- full SQL IDE/query workbench;
- schema migration designer;
- database user/grant administration;
- replication/VACUUM/cluster DBA tooling;
- MongoDB administration;
- Redis administration;
- general-purpose external database connection management.

## Compatibility policy

The compatibility suite answers one question: **can MyPaaS correctly host this declared application pattern within its documented boundary?**

A `PASS` means the declared deployment and smoke/lifecycle checks worked on the tested host. It does not establish throughput, concurrent-user capacity, enterprise readiness, or a minimum universal hardware specification.

Use compatibility failures to discover product gaps. Fix a gap only when the capability is platform-owned, reusable, and appropriate for a single-host PaaS.
