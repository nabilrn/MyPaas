# ADR-014: Optional Single-Host Replicas

Date: 2026-07-03

Status: Deferred

Resolution: Deferred from the current stable product contract. MyPaaS does not currently provide application replicas or horizontal scaling. This document preserves an earlier design option only; it is not an active roadmap commitment.

## Context

Some stateless projects may benefit from multiple local replicas on the same VM for restart smoothness or basic load distribution. This is not high availability because the VM remains a single failure domain. It also increases port allocation, Caddy route complexity, and quota accounting.

## Decision

Optional single-host replicas were considered as a possible future capability, initially limited to stateless Dockerfile projects.

If reconsidered, the design should:

- Treat replicas as an explicit per-project setting.
- Allocate runtime identity safely for each replica.
- Configure Caddy with multiple validated upstreams for the project host.
- Count every replica against configured memory and CPU quota.
- Define Compose service-level semantics before supporting Compose replicas.

## Consequences

Deferring replicas preserves the current single-host explicit-lifecycle model and avoids implying Kubernetes, Docker Swarm, Nomad, multi-node orchestration, or automatic horizontal scaling.

A single VM remains one failure domain regardless of local replica count.

## Follow-up

No implementation is committed. A future proposal must define rollback, logs, metrics, routing, health, quota, and cleanup behavior and demonstrate a concrete need before implementation.
