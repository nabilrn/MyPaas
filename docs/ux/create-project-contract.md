# Create Project UX Contract

**Status:** Current UX behavior contract for the Create Project surface.  
**Overall product source of truth:** current code/tests, `README.md`, `PRODUCT.md`, architecture docs, and accepted ADRs.

This document defines the intended interaction contract for Create Project. It is not a separate deployment engine specification and must not override current backend validation.

## Intent

Create Project is a single-page, source-first flow:

1. the user chooses or provides a deploy source;
2. MyPaaS inspects the source when inspection applies;
3. MyPaaS resolves the deployment shape;
4. required configuration appears before optional configuration;
5. the project can be created only after current analysis/configuration is complete and valid.

Do not introduce a wizard without a demonstrated UX need. Keep the existing single-page control-plane flow.

## Supported source paths

The current product supports:

- Git repository -> Dockerfile;
- Git repository -> Docker Compose;
- Git repository -> static deployment;
- OCI registry -> image deployment.

Installable OSS templates feed these same deployment primitives. Templates must not become a second deployment engine.

## Required behavior

- Git repository input starts/feeds repository inspection.
- Runtime/deployment type is detected or resolved from the selected source rather than hidden behind framework marketing labels.
- Project name may be suggested from repository/image/template context until the user edits it.
- Deployment type remains visible in the normal flow.
- Environment detection remains visible when relevant.
- Required configuration appears before optional configuration.
- Advanced settings are discoverable but secondary.
- Base Directory normally remains advanced unless root detection fails or the user needs a manual override.
- Create cannot become ready while required analysis is stale, incomplete, scheduled, or in flight.
- Re-analysis invalidates stale repository/runtime/environment results.
- Blocking Compose Doctor findings prevent readiness and surface an actionable message.
- Required environment values block readiness until provided or satisfied by a managed capability.
- OCI image flows expose a required container port when it cannot be inferred.

## Current bounded feature contracts

### Registry image authentication

ADR-022 provides one optional **installation-level** credential for a configured registry host.

Create Project must not present this as arbitrary per-project/multi-registry credential management. Anonymous image pulls remain valid when no matching configured credential is required.

### Compose additional HTTP routes

ADR-023 provides up to four bounded additional HTTP routes for Compose projects.

When a template/application uses this primitive, the UI may surface the derived endpoints/setup requirements, but backend validation remains authoritative:

- target must be an existing Compose service;
- target port must be declared by `ports` or `expose`;
- hostname is platform-derived;
- no additional host port is published;
- no raw TCP/SSH/UDP routing;
- current route contract is immutable after first deployment.

## Audit expectations

Use audits only when the Create Project behavior itself changes materially or a concrete UX defect needs evidence.

A production audit answers: "What does the deployed UI actually do?"

A mocked audit answers: "Does a difficult/unsafe state behave correctly?"

Production audits should be non-destructive by default. Mocked audits may simulate failures/timeouts/stale analysis, missing env/ports, Compose blockers, static detection, source-mode differences, and project-creation failure.

Do not repeat broad audit matrices after unrelated runtime, routing, observability, compatibility, or documentation changes.

## Evidence checkpoints

When a controlled audit is justified, stable checkpoint names may include:

- `00-initial`
- `01-source-entered`
- `02-analyzing`
- `03-runtime-detected`
- `04-configuration-required`
- `05-ready`
- `06-submitting`
- `07-created`

Collect only evidence needed to prove the targeted behavior: screenshot/ARIA state, visible controls/text, readiness state, relevant console/network failures, and geometry when layout is the issue.

Generated audit evidence is not permanent product documentation and must not contain credentials, decrypted env values, cookies, OAuth tokens, or registry passwords.
