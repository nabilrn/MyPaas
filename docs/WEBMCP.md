# WebMCP site tools

MyPaaS exposes a bounded subset of its existing operations as experimental WebMCP site tools when the dashboard is opened in a browser that implements `document.modelContext`.

WebMCP is an adapter over the existing MyPaaS REST API. It does not create a second deployment engine, authentication system, or host-control path.

## Authentication model

The existing stdio MCP bridge remains available for headless clients and uses `MYPAAS_API_TOKEN`.

WebMCP uses the signed-in dashboard browser session instead. Tools are registered only after `/auth/me` succeeds, and the underlying API requests continue to use the same authenticated cookies and server-side authorization rules as the dashboard.

No WebMCP tool is exposed cross-origin with `exposedTo`.

## Tool surface

Authenticated users receive:

- `list_projects`
- `get_project`
- `list_deployments`
- `get_deployment`
- `get_logs`
- `get_metrics_snapshot`
- `get_quota`
- `run_diagnostics`
- `deploy_project`
- `start_project`
- `stop_project`
- `restart_project`

Owners additionally receive:

- `get_host_stats`
- `list_containers`
- `get_database_schema`

The browser-facing names intentionally align with the existing MCP bridge where the operation already exists.

## Safety boundary

WebMCP v1 deliberately does not register tools for:

- project deletion;
- environment-value reveal;
- environment mutation;
- DB Studio writes;
- platform update;
- backup/restore mutation;
- arbitrary shell or host commands.

Deployment/start/stop/restart tools are marked as state-changing (`readOnlyHint: false`). Read tools use `readOnlyHint: true`.

Tools returning application logs or user/application-defined metadata use `untrustedContentHint: true` so supporting agents can treat the output as an untrusted-data boundary.

## Diagnostics

`run_diagnostics` is a read-only project diagnostic bundle. It gathers independently:

- project state;
- latest deployment;
- current runtime metrics;
- configured HTTP routes;
- recent logs.

A failed sub-check is returned as a bounded error inside the diagnostic result instead of hiding the remaining evidence.

## Browser compatibility

WebMCP support is feature-detected. Browsers without `document.modelContext` receive the normal MyPaaS dashboard with no behavior change and no additional dependency.

The WebMCP standard is experimental, so this adapter should remain small and isolated under `frontend/src/lib/webmcp/`.
