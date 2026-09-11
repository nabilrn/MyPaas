# ADR-013: Autosizing as Recommendation

Date: 2026-07-03

Status: Deferred

Resolution: Deferred from the current stable product contract. MyPaaS does not currently provide a historical p95 autosizing recommendation engine, and this ADR is not a commitment to add one. Reconsider only from measured workload/operator need.

## Context

Resource profiles give safe starting limits, but real memory and CPU usage can differ by project. Fully automatic autosizing could surprise the owner by changing deploy behavior, exceeding quota expectations, or destabilizing apps during early dogfooding.

## Decision

The earlier direction was to prefer autosizing **recommendations**, not automatic enforcement.

If reconsidered, a recommendation engine should:

- Use historical runtime samples per project and service.
- Compute a defensible percentile/window from persisted measurements.
- Compare actual usage against configured limits.
- Suggest lower or higher limits with a clear reason.
- Require explicit user approval before applying changes.

## Consequences

Deferring this keeps the owner in control of quota and deployment behavior and avoids presenting live snapshots as a historical sizing model.

Current resource profiles and explicit per-project overrides remain the stable configuration contract.

## Follow-up

No implementation is committed. A future proposal must first define persistent metrics semantics, evidence quality, and the concrete operator problem it solves.
