# ADR-012: Idle Sleep and Wake-on-Request

Date: 2026-07-03

Status: Proposed for after-deploy

## Context

MyPaas runs on a single VM with limited RAM. Some personal projects will be accessed rarely, so keeping every container running wastes memory. The pre-deploy gate already reduces configured limits through resource profiles, but automatic sleep/wake adds routing and state-management risk that should not block the first VM deploy.

## Decision

Idle sleep and wake-on-request will be implemented after MyPaas is live and stable.

The first design should:

- Mark sleep state explicitly in `projects.status` or a dedicated lifecycle field.
- Stop eligible project containers after an inactivity window.
- Keep Caddy routes active and point sleeping projects to a lightweight wake handler in the MyPaas API.
- On first request, enqueue a wake job, start the container or Compose project, restore the project route, and return a clear cold-start response.
- Exclude projects with active deployments, recent failures, or user-disabled sleep.

## Consequences

This keeps the pre-deploy scope focused on stable deploy, routing, logs, metrics, backup, and quota. It also avoids introducing request buffering, race handling, and user-facing cold-start semantics before dogfooding proves the baseline.

## Follow-up

Before implementation, define the lifecycle state machine and Caddy wake route shape in a dedicated technical design.
