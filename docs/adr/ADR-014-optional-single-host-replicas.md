# ADR-014: Optional Single-Host Replicas

Date: 2026-07-03

Status: Proposed for after-deploy

## Context

Some stateless projects may benefit from multiple local replicas on the same VM for restart smoothness or basic load distribution. This is not high availability because the VM remains a single failure domain. It also increases port allocation, Caddy route complexity, and quota accounting.

## Decision

MyPaas may support optional single-host replicas after the first live deploy, limited to stateless Dockerfile projects at first.

The design should:

- Treat replicas as an explicit per-project setting.
- Allocate one internal port per replica.
- Configure Caddy with multiple upstreams for the project host.
- Count every replica against configured memory and CPU quota.
- Disable replicas for Compose projects until service-level semantics are designed.

## Consequences

This avoids Kubernetes, Docker Swarm, Nomad, or multi-node orchestration while leaving a path for practical single-VM smoothing later. It does not change the MVP commitment: one VM, Docker/Compose, and explicit resource limits.

## Follow-up

Define rollback, logs, metrics, and health behavior for replicated projects before implementation.
