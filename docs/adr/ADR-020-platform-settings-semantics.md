# ADR-020: Platform settings must have one authoritative runtime consumer

## Status

Accepted

## Context

The owner-facing Admin Settings page exposed `project_default_ram_mb`, `project_default_cpu`, and `max_concurrent_deploys` as if changing them altered live platform behavior.

That was misleading:

- new-project resources are selected by MyPaas resource profiles (`static`, `go-small`, `node-python`, `compose-main`, or explicit custom values), so the two global project-default fields were a second, unused source of truth;
- deployment concurrency is established by a process-start semaphore from `MAX_CONCURRENT_DEPLOYS`, so changing a database value cannot resize the active worker limiter;
- numeric owner overrides are persisted in `platform_settings`, but they must also be rehydrated into the shared runtime config after an API restart or the UI and enforcement state can diverge.

## Decision

Admin Settings exposes build timeout and resource-profile defaults. The former platform-limit controls are not shown in the dashboard; installation-level quota values remain supported for compatibility and enforcement.

`project_default_ram_mb` and `project_default_cpu` are not part of the editable Admin Settings contract. Resource profiles remain the single source of truth for new-project defaults. Owners may raise the memory and CPU defaults for `static`, `go-small`, `node-python`, and `compose-main`, but cannot lower them below the built-in profile floors. Changes are persisted, rehydrated on API startup, and used by REST, MCP, Create Project, and Project Settings. Explicit per-project overrides remain available.

`max_concurrent_deploys` is no longer exposed as a live Admin Settings control. It remains an installation-level setting through `MAX_CONCURRENT_DEPLOYS` until MyPaas has a deliberately resizable runtime limiter.

Persisted live numeric settings are re-applied to the shared config when the settings handler is constructed and whenever settings are read. Invalid or obsolete database keys are ignored rather than becoming effective runtime configuration.

Both API and frontend validate the supported limits. Unknown Admin Settings keys are rejected by the update endpoint.

## Consequences

- The owner UI can no longer claim configuration that MyPaas does not enforce.
- New-project resource defaults remain runtime-profile-specific and owner-configurable above fixed floors.
- Saved quota and build-timeout overrides survive API restarts as effective runtime values.
- Deployment concurrency changes continue to require installation configuration and an API restart.
- Old database rows for removed keys may remain harmlessly stored; they are not returned or applied by the live settings contract.
