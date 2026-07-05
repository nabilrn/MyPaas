# ADR-013: Autosizing as Recommendation

Date: 2026-07-03

Status: Proposed for after-deploy

## Context

Resource profiles give safe starting limits, but real memory and CPU usage can differ by project. Fully automatic autosizing could surprise the owner by changing deploy behavior, exceeding quota expectations, or destabilizing apps during early dogfooding.

## Decision

MyPaas will start with autosizing recommendations, not automatic enforcement.

The recommendation engine should:

- Use historical Docker Stats samples per project and service.
- Compute p95 memory and CPU over a rolling window.
- Compare actual usage against configured limits.
- Suggest lower or higher limits with a clear reason.
- Require explicit user approval before applying changes.

## Consequences

The owner stays in control of quota and deployment behavior. MyPaas can still guide projects toward tighter resource limits once real runtime data exists.

## Follow-up

Add persistent metrics storage before implementing recommendations. Current live snapshots are enough for dashboard display but not enough for historical p95 calculation.
