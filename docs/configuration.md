# Configuration

MyPaaS has more than one configuration layer. Treating every setting as a live environment variable produces misleading operator behavior, so this document separates installation-time configuration from owner-editable runtime settings.

## Configuration layers

| Layer | Authority | Examples | Apply semantics |
| --- | --- | --- | --- |
| Production environment | installer-managed `.env` and host configuration | domain, OAuth credentials, runtime/socket/network identity, `MAX_CONCURRENT_DEPLOYS` | generally process/startup level; restart/redeploy may be required |
| Admin Settings | control-plane settings persisted in PostgreSQL | build timeout and resource-profile defaults | applied through the supported settings API/UI |
| Per-project configuration | project records and encrypted environment values | resource override, application env, routes | project lifecycle/deploy semantics |
| Update policy | `/etc/mypaas/update.env` | release channel, interval, prerelease policy | consumed by host-side updater |

The current code, installer, schema, and accepted ADRs remain authoritative if a setting changes after this document is published.

## Production `.env`

`scripts/install-vm.sh` generates the production `.env`. Do not copy `.env.example` as a production recipe.

The generated environment includes categories such as:

- installation identity: `PUBLIC_DOMAIN`, `FRONTEND_URL`, `OWNER_EMAIL`;
- GitHub OAuth: `GITHUB_CLIENT_ID`, `GITHUB_CLIENT_SECRET`, `GITHUB_CALLBACK_URL`;
- public delivery: `CLOUDFLARE_TUNNEL_TOKEN`;
- control-plane database and generated secrets;
- Docker-compatible runtime/socket and network names;
- installation quotas and deployment worker concurrency;
- metrics authentication;
- optional shared PostgreSQL;
- backup/image-cleanup policy;
- update defaults;
- Caddy/static/statd host paths.

Production secrets belong only in trusted configuration/storage. Never paste `.env`, OAuth secrets, registry credentials, JWTs, encryption keys, database passwords, MCP tokens, or decrypted project environment values into public issues.

## `.env.example`

`.env.example` is a **development and configuration reference**. It intentionally contains local defaults such as development database URLs, local frontend/API addresses, and local Caddy settings alongside documented production-related keys.

Use it to discover supported environment names, not as a replacement for the production installer.

## Container runtime and networks

Fresh installation defaults to rootful Podman through the Docker-compatible contract. Docker Engine is an explicit compatibility mode.

Important production runtime settings include:

```text
DOCKER_SOCKET=/var/run/docker.sock
CONTROL_NETWORK=mypaas-control
PROJECT_NETWORK=mypaas-projects
ROUTING_NETWORK=mypaas-routing
```

All three production network names must be distinct. Do not switch an existing installation between Docker and Podman by editing the socket/runtime setting in place; use the migration boundary instead.

## Resource profiles and quotas

New projects use resource-profile defaults rather than one global RAM/CPU default. The current profile families include:

```text
static
go-small
node-python
compose-main
```

Owner-editable profile defaults may be raised but cannot be reduced below the built-in safety floors. Explicit per-project overrides remain possible within the platform's validation and quota boundaries.

`project_default_ram_mb` and `project_default_cpu` are **not** current Admin Settings controls.

`MAX_CONCURRENT_DEPLOYS` is installation-level configuration. The active deployment limiter is established when the API process starts, so changing this value requires applying the production configuration and restarting/recreating the API process; it is intentionally not presented as a live Admin Settings control.

See [ADR-026](adr/ADR-026-platform-settings-semantics.md).

## Private registry credential

OCI image-mode deployments pull anonymously by default. An installation can configure one bounded registry credential:

```text
MYPAAS_REGISTRY_HOST
MYPAAS_REGISTRY_USERNAME
MYPAAS_REGISTRY_PASSWORD
```

All three values belong to the same installation-level credential contract. MyPaaS uses the credential only when the requested image registry host matches the configured host and performs authentication with an isolated temporary Docker configuration.

The credential does **not** become a generic multi-registry credential store and is not automatically inherited by Compose service pulls. See [ADR-022](adr/ADR-022-private-registry-auth.md).

## Image-mode persistent storage

An OCI image can request MyPaaS-managed durable storage using either:

- Docker image `VOLUME` declarations; or
- the image label `io.mypaas.persistent-volumes` containing a comma-separated list of absolute container paths.

These paths resolve to MyPaaS-owned Docker-compatible named volumes. Image metadata cannot request arbitrary host bind paths through this contract. See [ADR-025](adr/ADR-025-persistent-image-storage.md).

## Self-update policy

The host updater persists its policy in:

```text
/etc/mypaas/update.env
```

The stable operator channel is:

```text
AUTO_UPDATE_CHANNEL=release
AUTO_UPDATE_INCLUDE_PRERELEASES=false
```

`AUTO_UPDATE_CHANNEL=main` is a development-host option, not the normal stable production policy. Periodic updates are opt-in through `AUTO_UPDATE_ENABLED=true`; the host path trigger remains available for dashboard-triggered updates even when the timer is disabled.

Use `scripts/configure-auto-update.sh` to change updater policy rather than editing systemd units directly. See [Updates](operations/update.md).

## Applying startup-level changes

Do not assume an environment-file edit is hot-reloaded. For settings consumed at process startup, apply the intended configuration through the production deployment/update path and run verification afterward.

For the special case of `MAX_CONCURRENT_DEPLOYS`, an API restart/recreation is required for the new limiter to take effect.

When changing network identity, runtime engine, production secrets, or other host-level topology, prefer the documented installer/update/migration path over ad-hoc container commands.

## Related documents

- [Installation](installation.md)
- [Updates](operations/update.md)
- [Architecture](ARCHITECTURE.md)
- [Security boundaries](SECURITY_BOUNDARIES.md)
- [ADR-022: private registry authentication](adr/ADR-022-private-registry-auth.md)
- [ADR-025: persistent image storage](adr/ADR-025-persistent-image-storage.md)
- [ADR-026: platform settings semantics](adr/ADR-026-platform-settings-semantics.md)
