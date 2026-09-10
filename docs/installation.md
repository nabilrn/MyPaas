# Installation

This document describes the supported production installation path for the current stable MyPaaS line.

MyPaaS targets one Linux host. Fresh supported installs are **rootful Podman first** and use the Docker-compatible command/socket contract expected by the control plane. Docker Engine remains an explicit compatibility mode.

## Before you install

Prepare:

- a Linux VM you control;
- a public hostname for the MyPaaS dashboard, for example `mypaas.example.com`;
- a GitHub OAuth application with Client ID and Client Secret;
- the GitHub primary email that should become the initial owner;
- a Cloudflare Tunnel token for the configured public delivery path.

The default GitHub callback URL is:

```text
https://<your-domain>/api/auth/github/callback
```

Automatic dependency installation is implemented for Ubuntu/Debian-family hosts through `apt-get`. Other Linux distributions require the necessary runtime and dependencies to be prepared manually before using the installer.

The default `mypaas-statd` release artifact currently supports `linux-amd64`. Source installation is available as an explicit alternative for statd; statd itself can also be skipped.

## Install the stable release

Pin both the bootstrap script and checkout ref to the same stable release. For `v0.7.0`:

```bash
curl -fL \
  https://raw.githubusercontent.com/nabilrn/MyPaas/v0.7.0/scripts/bootstrap.sh \
  -o /tmp/mypaas-bootstrap.sh
MYPAAS_REF=v0.7.0 bash /tmp/mypaas-bootstrap.sh
```

The bootstrap process:

1. requires Linux;
2. installs Git automatically on supported apt-based hosts when needed;
3. creates or updates the installer-managed checkout at `$HOME/MyPaas` by default;
4. preserves the detected Docker/Podman engine on an existing installation and refuses an implicit in-place engine switch;
5. starts `scripts/install-vm.sh`;
6. defaults to rootful Podman on a fresh host;
7. starts the browser install wizard by default.

The installer creates the production `.env`. **Do not use `cp .env.example .env` as the production installation procedure.** `.env.example` is a development/configuration reference and contains defaults that intentionally differ from generated production values.

## Browser wizard

The bootstrap installer starts the browser wizard unless `INSTALL_WIZARD=false` is supplied. The wizard collects the installation identity and credentials required by production setup and writes the generated `.env` with restrictive permissions.

Important production values include:

- `PUBLIC_DOMAIN`;
- `OWNER_EMAIL`;
- `GITHUB_CLIENT_ID`;
- `GITHUB_CLIENT_SECRET`;
- `GITHUB_CALLBACK_URL`;
- `CLOUDFLARE_TUNNEL_TOKEN`.

Secrets such as the PostgreSQL password, JWT secret, encryption key, and metrics password are generated when not supplied explicitly.

## Docker Engine compatibility mode

To choose Docker Engine for a **fresh** installation:

```bash
USE_PODMAN=false MYPAAS_REF=v0.7.0 bash /tmp/mypaas-bootstrap.sh
```

Do not use `USE_PODMAN` to switch an existing installation between Docker and Podman in place. The bootstrap intentionally refuses an engine mismatch when it detects existing MyPaaS runtime state. Engine changes belong to the VM migration boundary; see [VM migration](operations/migration.md) and [ADR-019](adr/ADR-019-migration-safety-boundaries.md).

## Optional installer controls

The bootstrap forwards supported `install-vm.sh` environment settings. Common controls include:

| Setting | Default | Purpose |
| --- | --- | --- |
| `MYPAAS_INSTALL_DIR` | `$HOME/MyPaas` | Installer-managed checkout |
| `MYPAAS_REF` | `main` | Branch/tag/commit to install; pin a stable tag for production |
| `INSTALL_WIZARD` | `true` in bootstrap | Start browser setup wizard |
| `USE_PODMAN` | `true` | Fresh-install runtime choice |
| `INSTALL_STATD` | `true` | Install optional host telemetry daemon |
| `STATD_INSTALL_MODE` | `release` | `release` or explicit `source` build |
| `AUTO_UPDATE_ENABLED` | `false` | Enable periodic update timer during configuration |

For non-interactive installation, all required production values must be supplied through environment variables because the installer cannot prompt without a TTY.

## Host state created by MyPaaS

Production installation creates host-managed paths including:

```text
/var/lib/mypaas/volumes
/var/lib/mypaas/compose
/var/lib/mypaas/static
/var/lib/mypaas/backups
/tmp/mypaas/builds
```

The production stack also uses three distinct Docker-compatible networks, defaulting to:

```text
mypaas-control
mypaas-projects
mypaas-routing
```

See [Architecture](ARCHITECTURE.md) for the network and trust model.

## Verify the installation

After installation:

```bash
cd ~/MyPaas
ENV_FILE=.env bash scripts/verify-production.sh
```

The verifier checks control-plane containers, network boundaries, API readiness, Caddy ingress/admin socket behavior, dashboard release assets, release identity, optional statd, and the bundled CLI. See [Production verification](operations/production-verification.md) for the complete boundary.

Then open:

```text
https://<your-domain>
```

and sign in through the configured GitHub OAuth application.

## Next steps

- [Configuration](configuration.md)
- [Updates](operations/update.md)
- [Backup and restore](operations/backup-restore.md)
- [VM migration](operations/migration.md)
- [Production verification](operations/production-verification.md)
