# Beta Create Project runtime-contract audit

**Status:** Historical qualification procedure; retained for regression provenance.  
**Current source of truth:** current frontend/backend implementation, tests, and [`../engineering/beta-readiness-gates.md`](../engineering/beta-readiness-gates.md).

This document records the intent of the Create Project beta audit that has already been completed. It is not an active instruction to repeat a broad UI/runtime audit after unrelated changes.

## Contract that remains important

Create Project must remain fail-closed while source analysis or required configuration is unresolved.

Representative behavior covered by the historical qualification included:

- slow repository analysis;
- stale analysis after Base Directory changes;
- Dockerfile with unresolved app port;
- Compose required environment values;
- Compose Doctor blockers;
- repository-analysis backend failure/timeout;
- static detection;
- OCI image source with required port configuration;
- backend project-creation failure reached from an otherwise valid form.

Current tests are authoritative for the exact state machine and UI behavior.

## Current product surface

Create Project currently supports the product's four deployment paths:

- Git + Dockerfile;
- Git + Docker Compose;
- Git + static output;
- OCI image source.

Compose configuration can include base-directory/monorepo layout, Compose file selection, main-service selection, environment discovery, resource settings, and the bounded additional HTTP-route contract when a template/application requires it.

Image-mode deployment may use the installation-level bounded registry credential defined by ADR-022; this is not a project-level registry credential manager.

## Regression rule

Run the narrow tests/audit path that corresponds to the behavior changed.

A broad Create Project qualification is justified only when a change materially alters the overall source-analysis/readiness/create contract. Do not rerun it merely because an unrelated lifecycle, routing, observability, compatibility, or documentation change landed.

## Evidence rule

Screenshots, traces, HAR/network observations, and other generated audit artifacts are review evidence, not permanent product documentation. Keep them out of source unless a specific PR/release review requires retention.

Never retain credentials, decrypted environment values, cookies, OAuth tokens, registry passwords, or other secrets in audit artifacts.
