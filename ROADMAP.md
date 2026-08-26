# MyPaaS Product Roadmap

MyPaaS is a simple single-host PaaS for an owner developer or small trusted team. The roadmap optimizes for real application deployment and operation, not benchmark breadth or distributed orchestration.

Current code, tests, and accepted ADRs remain the source of truth for implemented behavior. This file only defines product direction.

## Product rule

Prioritize work that is:

- owned by the MyPaaS platform;
- useful for real application deployment or operation;
- reversible and understandable on one host;
- compatible with the existing Docker/Compose/static/OCI deployment engine;
- directly visible as better deployment, database, compatibility, or dashboard UX.

Do not add architecture merely to create a future scaling story.

## KEEP

These are established product capabilities and should be maintained rather than redesigned without a concrete defect:

- project create, update, delete, redeploy, start, stop, restart, and rollback boundaries;
- Dockerfile deployment;
- Docker Compose deployment, including multi-service inspection and main-service selection;
- public OCI image deployment;
- static deployment through Caddy;
- public routing through the configured Caddy / Cloudflare delivery path;
- encrypted project environment variables and repository environment discovery;
- project-scoped persistent storage and owned-resource cleanup;
- deployment status, history, logs, metrics, and bounded deployment concurrency;
- project/user resource guardrails;
- shared PostgreSQL provisioning with generated credentials;
- DB Studio safe row browsing/editing for PostgreSQL, MySQL, and MariaDB;
- backup, restore-drill, and migration tooling within documented boundaries;
- the real-world compatibility catalog and runner;
- the existing dashboard information architecture.

## DELIVERED PRODUCTIZATION

### OSS App Templates v1

Delivered on `main`.

The template catalog turns qualified deployment patterns into a user-facing install path without creating a second deployment engine. Initial templates cover representative image, stateful, database-backed, and multi-service applications.

### DB Studio schema metadata and ERD

Delivered on `main`.

DB Studio now extends beyond row browsing with schema metadata useful for understanding relationships while remaining intentionally smaller and safer than a full SQL IDE.

## IMPLEMENT NEXT

### 1. Private registry authentication and pull diagnostics

Support authenticated OCI image pulls with narrowly scoped credentials and actionable failures for authentication, permission, rate-limit, and missing-image cases.

Boundaries:

- no registry proxy;
- no pull-through cache in the first implementation;
- no credential leakage into project Compose environments;
- no persistent modification of an operator's normal Docker credential store.

### 2. Compatibility status in product UX

Surface curated compatibility information in the dashboard/template experience:

- supported deployment pattern;
- known platform boundary;
- expected persistent storage;
- declared resource guidance;
- important setup requirements.

A compatibility result must never become a throughput, user-count, or hardware-capacity claim.

### 3. Template/env improvements driven by real applications

Extend templates only when an application requires a reusable platform primitive, for example:

- generated secrets;
- generated public URL/host values;
- required/default env fields;
- explicit resource warnings;
- documented persistent-storage requirements.

Do not add template-specific application code patches.

## DEFER

Keep these as valid future product ideas, but do not keep active implementation programs for them until a real application demonstrates the need.

### Multiple public routes / ports per project

Useful for applications such as a web UI plus an additional public protocol or endpoint. Implement only with an explicit route model and clear security ownership; do not bolt arbitrary host-port exposure onto the current single-route model.

### Static asset cache policy

Potentially useful for known immutable build assets, but incorrect caching is a correctness bug. Revisit only as a narrow delivery feature with framework-safe semantics.

### Registry pull cache / mirror

Authentication and clear rate-limit diagnostics come first. A registry cache introduces storage, garbage-collection, and freshness responsibilities that are not justified by the current product scope.

### Caddy-specific delivery telemetry

Existing application logs, runtime metrics, and optional Cloudflare analytics are sufficient for current operation. Add Caddy-specific telemetry only if it answers a concrete operator question that current observability cannot answer.

### Restore UI

The destructive recovery path should remain operator-oriented until the existing backup/restore workflow has a clear, safe UI contract.

### Multi-database DB Studio selector

Useful for Compose projects that intentionally contain multiple SQL databases. Keep the first implementation project-local and bounded to PostgreSQL/MySQL/MariaDB; do not turn it into a global DBA connection manager.

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
- user-count, RPS, or hardware-capacity promises derived from compatibility fixtures.

Historical experiments may remain in Git history, closed pull requests, or archived evidence. They must not be presented as current product direction.

## DB Studio boundary

DB Studio should remain a focused application-data tool.

Keep improving:

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

A PASS means the declared deployment and smoke checks worked on the tested host. It does not establish throughput, concurrent-user capacity, enterprise readiness, or a minimum universal hardware specification.

Use compatibility failures to discover product gaps. Fix a gap only when the capability is platform-owned, reusable, and appropriate for a single-host PaaS.
