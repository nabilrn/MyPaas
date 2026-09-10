# ADR-012: Idle Sleep and Wake-on-Request

Date: 2026-07-03

Status: Deferred

Resolution: Deferred from the current stable product contract. This document records an earlier design direction; MyPaaS does not currently implement idle sleep or wake-on-request. Revisit only if a concrete workload or operator problem justifies the additional routing and lifecycle complexity.

## Context

MyPaas runs on a single VM with limited RAM. Some personal projects will be accessed rarely, so keeping every container running wastes memory. The pre-deploy gate already reduces configured limits through resource profiles, but automatic sleep/wake adds routing and state-management risk that should not block the first VM deploy.

## Decision

Idle sleep and wake-on-request was proposed as a possible post-deploy capability.

If reconsidered, the design should:

- Mark sleep state explicitly in `projects.status` or a dedicated lifecycle field.
- Stop eligible project containers after an inactivity window.
- Keep Caddy routes active and point sleeping projects to a lightweight wake handler in the MyPaas API.
- On first request, enqueue a wake job, start the container or Compose project, restore the project route, and return a clear cold-start response.
- Exclude projects with active deployments, recent failures, or user-disabled sleep.

## Consequences

Deferring this keeps the stable product focused on explicit deploy, routing, logs, metrics, backup, quota, and lifecycle behavior. It also avoids adding request buffering, race handling, and user-facing cold-start semantics without a measured need.

## Follow-up

No implementation is committed. If a concrete need reopens this decision, define the lifecycle state machine and Caddy wake route shape in a new/current technical design before implementation.
