# CLI

MyPaaS includes a small Go CLI named `mypaas`. The current CLI is intentionally narrower than the dashboard and REST surface.

## Build

From the repository root:

```bash
make build-backend
```

This produces:

```text
backend/bin/mypaas
```

The production API image also contains the CLI at `/app/mypaas`.

## Configuration

The CLI stores its local configuration at:

```text
~/.mypaas/config.yml
```

The file is created with owner-only permissions. Configure the API URL and an existing valid JWT:

```bash
mypaas config set api-url https://<your-domain>/api
mypaas config set token <jwt>
mypaas config show
```

`config show` reports whether a token is set without printing it.

### Authentication boundary

The current CLI **does not implement a login command or issue its own JWT**. API-backed CLI commands require a valid Bearer JWT obtained through an existing trusted authentication/automation flow.

Do not invent a long-lived token by copying cookies, database values, OAuth secrets, or the MCP API token into the CLI configuration. The MCP token is a separate bridge credential with its own contract.

If you do not already have a supported JWT provisioning flow for your automation, use the dashboard or MCP bridge instead of constructing credentials manually.

## User commands

```bash
mypaas user list
mypaas user add user@example.com
mypaas user remove <user-id>
```

These call owner-protected administration endpoints and therefore require an authenticated owner JWT.

## Project commands

List projects:

```bash
mypaas project list
```

Deploy by project name:

```bash
mypaas project deploy <project-name>
```

Read recent logs:

```bash
mypaas project logs --tail 200 <project-name>
```

Follow logs by polling the current log endpoint:

```bash
mypaas project logs --follow <project-name>
```

The current project CLI surface is limited to `list`, `deploy`, and `logs`; other lifecycle/configuration operations remain available through the dashboard, REST API, or MCP surface.

## Backup command

```bash
mypaas backup
```

Unlike the API-backed user/project commands, this is a local host operation. It loads the MyPaaS runtime configuration and invokes the control-plane backup service directly. Run it in an environment where the production configuration and Docker-compatible runtime are available.

For a fuller disaster-recovery bundle that includes production configuration, static/Compose state, and managed volumes, use [Backup and restore](operations/backup-restore.md) instead.

## Default API URL

Without an explicit CLI configuration, the API URL defaults to:

```text
http://localhost:8080
```

For a normal remote installation, set the public API base explicitly:

```bash
mypaas config set api-url https://<your-domain>/api
```

## Command reference

The executable help is authoritative:

```bash
mypaas help
```

Current top-level commands are:

```text
config
user
project
backup
```

## Security notes

- `~/.mypaas/config.yml` contains the configured JWT in plaintext and must be treated as a secret file.
- Do not publish its contents in bug reports.
- Prefer short-lived/trusted automation credentials where the surrounding workflow supports them.
- The CLI does not weaken the same owner/user authorization enforced by the API.

## Related documents

- [REST API](api.md)
- [MCP](mcp.md)
- [Backup and restore](operations/backup-restore.md)
- [Security boundaries](SECURITY_BOUNDARIES.md)
