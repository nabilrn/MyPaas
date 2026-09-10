# MCP

MyPaaS exposes a **local stdio MCP bridge** for trusted coding/operations agents. The bridge runs on the same machine as the agent, then calls the configured MyPaaS REST API.

This is not a remotely hosted `/mcp` service and it does not create a second orchestration engine.

## Supported bridge

Source:

```text
backend/cmd/mcp/main.go
```

The bridge reads:

```text
MYPAAS_URL
MYPAAS_API_TOKEN
```

`MYPAAS_URL` should point to the installation API, normally:

```text
https://<your-domain>/api
```

The token is a dedicated MCP/API bridge credential. Keep it secret.

## Get or rotate the MCP token

Use the owner dashboard:

```text
Administration → MCP
```

The page shows the current bridge target and MCP credential. Use **Regenerate** when the credential must be rotated.

Regeneration invalidates the previous token, so every connected agent must be updated afterward.

Do **not** add an arbitrary `MYPAAS_API_TOKEN` to the production `.env` and restart the API container as an MCP setup procedure. The current supported token lifecycle is managed by the Administration → MCP settings flow.

## Run the bridge

On the machine running your agent, clone MyPaaS and run the Go bridge over stdio:

```bash
git clone https://github.com/nabilrn/MyPaas.git
cd MyPaas
MYPAAS_URL=https://<your-domain>/api \
MYPAAS_API_TOKEN=<your-token> \
go run ./backend/cmd/mcp
```

For reproducible use, check out the release/source revision you intend to run rather than assuming the repository default branch is immutable.

A repository example configuration is available at:

```text
.agents/mcp/mypaas/mcp_config.json
```

Adapt its command format to the MCP client you use.

## Verify safely

Start with a read operation, for example asking the connected agent to list MyPaaS projects. Confirm it is connected to the intended installation before allowing a state-changing operation.

The current bridge exposes project/deployment/observability/environment operations. Exact tool registration in the MCP source is authoritative.

Treat deployment, restart, rollback, environment mutation, and other write operations as explicit state changes. Agents should not perform them unless the operator requested the action.

## Credential boundary

The MCP token is not the same thing as:

- a GitHub OAuth client secret;
- a project webhook secret;
- a project environment secret;
- a metrics Basic Auth password;
- the JWT stored by the standalone CLI.

Do not copy one credential type into another configuration path.

Never commit a real MCP token to Git, paste it into public issues, or include it in screenshots/logs.

## WebMCP

The dashboard also contains an **experimental browser WebMCP adapter** for browsers that implement `document.modelContext`. It uses the signed-in dashboard browser session rather than the stdio MCP token.

WebMCP is feature-detected and deliberately exposes a bounded tool surface. Browsers without the experimental API continue to use the normal dashboard.

See [WebMCP site tools](WEBMCP.md).

## Related documents

- [REST API](api.md)
- [Security boundaries](SECURITY_BOUNDARIES.md)
- [WebMCP](WEBMCP.md)
