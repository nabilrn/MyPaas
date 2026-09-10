# MyPaaS MCP Bridge

This directory contains an example client configuration for the current **local stdio MCP bridge**.

Use the canonical setup guide in [`docs/mcp.md`](../../../docs/mcp.md).

## Supported setup

1. Sign in to the MyPaaS owner dashboard.
2. Open **Administration → MCP**.
3. Copy the bridge target and current MCP token, or regenerate the token when rotation is intended.
4. On the machine running the agent, use a MyPaaS checkout containing `backend/cmd/mcp`.
5. Configure the agent to run the bridge over stdio with:

```text
MYPAAS_URL=https://<your-domain>/api
MYPAAS_API_TOKEN=<your-token>
```

The example [`mcp_config.json`](mcp_config.json) shows the repository command shape. Adapt it to the MCP client you use.

The bridge itself can be started directly for testing:

```bash
MYPAAS_URL=https://<your-domain>/api \
MYPAAS_API_TOKEN=<your-token> \
go run ./backend/cmd/mcp
```

Start with a read action such as listing projects before allowing state-changing operations.

## Token lifecycle

`MYPAAS_API_TOKEN` is managed through the current Administration → MCP settings flow. **Do not** use the old procedure of adding an arbitrary token to the production `.env` and restarting a container.

Regenerating the MCP token invalidates the previous credential. Update every connected agent after rotation.

Keep the token secret and never commit a real value to this directory.

## Current boundary

The supported headless MCP integration is a local stdio bridge that calls the configured MyPaaS REST API. It is not a remotely hosted `/mcp` endpoint.

For the experimental browser adapter, see [`docs/WEBMCP.md`](../../../docs/WEBMCP.md).
