# ADR-018: Automatic self-update for VM installs

## Status

Proposed

## Context

Production VM installs are repository-backed and currently update by rerunning `scripts/bootstrap.sh`. The bootstrap uses a shallow fetch followed by an `ff-only` merge. When the upstream branch is rewritten or squash/force-updated, an existing shallow checkout can fail with `refusing to merge unrelated histories` even though the install directory is intentionally managed by MyPaas.

The production Compose file also consumes mutable `:latest` API/dashboard images. Pulling `latest` immediately after observing a new Git commit creates a race: the source checkout can advance before the image publish workflow for that commit has completed.

MyPaas needs an updater that keeps source, migrations, Compose configuration, and application images on the same revision, without requiring Watchtower or giving another container access to the Docker socket. Production installs also need a conservative update source: an arbitrary new `main` commit must not become an installable production update before it has been intentionally published as a release.

## Decision

1. Existing installer-managed checkouts are synchronized with `git fetch` + `git reset --hard` after refusing dirty working trees. The checkout is not treated as a user-development clone.
2. Stable installations use `AUTO_UPDATE_CHANNEL=release` and resolve published GitHub Releases. `AUTO_UPDATE_CHANNEL=main` remains an explicit development-host option. Prereleases are excluded unless `AUTO_UPDATE_INCLUDE_PRERELEASES=true`.
3. A resolved release is accepted only when its target commit is a descendant of the currently installed commit. The updater refuses implicit downgrades or unrelated history.
4. The image publish workflow publishes `latest` plus immutable full-Git-SHA tags. A frontend-only main revision reuses the immutable API image from its parent SHA and aliases it to the new SHA instead of rebuilding unchanged backend code.
5. `docker-compose.prod.yml` accepts `MYPAAS_IMAGE_TAG`, defaulting to `latest` for backwards compatibility.
6. `scripts/update-vm.sh` remains the full platform updater. It:
   - serializes updates with a host lock;
   - refuses dirty checkouts;
   - fetches the resolved target ref;
   - waits for both SHA-tagged images before changing the checkout;
   - preflights migrations and the existing control-plane runtime;
   - tags the currently running API/dashboard images locally for runtime rollback;
   - resets the managed checkout to the target SHA;
   - deploys using the same SHA image tag;
   - verifies API/Caddy/CLI health;
   - restores the previous checkout and locally tagged runtime images when deployment or verification fails.
7. `scripts/update-dispatch.sh` resolves the selected channel before delegating. A frontend-only target uses `scripts/update-dashboard.sh`, which recreates only the dashboard and verifies or rolls it back without restarting API dependencies.
8. The host updater writes an atomic status snapshot under `/run/mypaas/update/status` with `checking`, `updating`, `succeeded`, `failed`, `rolled_back`, `blocked`, or `idle` state. The dashboard receives only `/run/mypaas/update` as a read-only mount; it does not receive the parent `/run/mypaas` directory or the Caddy admin socket.
9. The host-side systemd service and path trigger are always installed on supported VMs so the authenticated API can queue a manual update through `/run/mypaas/update.request`. The API never executes host scripts from inside its container.
10. The owner-only dashboard notification surface reads the host status through a SvelteKit server route, authenticates the browser session against the internal API, and discovers published releases from GitHub. UI success is based on updater status rather than inferring success from an API restart.
11. Release publication is a qualification gate. A `release/v*` publish request may contain only its release-notes file; the release tag targets the current `main` SHA only after an exact-SHA CI run has succeeded and immutable API and dashboard images for that SHA exist in GHCR.
12. Automatic polling is opt-in. The timer is installed only when `AUTO_UPDATE_ENABLED=true`. No Watchtower-style Docker-socket watcher is used.

## Consequences

### Positive

- Stable VM installs advance only to intentionally published releases rather than every `main` commit.
- Prereleases can be qualified explicitly without exposing them to stable installs.
- Automatic updates cannot deploy a Git revision before its exact API and dashboard artifacts are available.
- Source/config/migrations and application images stay pinned to one revision during an update.
- Frontend-only revisions keep the dashboard-only runtime fast path while preserving an immutable API identity for the release SHA.
- Owner UI can report the updater's actual terminal result, including rollback and blocked states.
- The dashboard can inspect update status without receiving the Caddy admin socket.
- The updater remains host-side and does not introduce another privileged Docker-socket container.

### Trade-offs

- Automatic rollback is best effort. A forward database migration may not be reversible by simply restoring the previous application image. Operators should keep MyPaas backups enabled.
- SHA image tags and frontend-only aliases consume registry metadata in addition to `latest`.
- Stable release discovery depends on GitHub availability; a discovery failure leaves the running installation unchanged.
- systemd is required for dashboard-triggered and scheduled updates; `scripts/update-vm.sh` can still be run manually on other Linux init systems.

## Operations

Enable automatic stable-release checks on an installed VM:

```bash
cd ~/MyPaas
AUTO_UPDATE_ENABLED=true AUTO_UPDATE_INTERVAL_MINUTES=30 AUTO_UPDATE_CHANNEL=release bash scripts/configure-auto-update.sh
```

Temporarily qualify a prerelease on a test VM:

```bash
cd ~/MyPaas
AUTO_UPDATE_CHANNEL=release AUTO_UPDATE_INCLUDE_PRERELEASES=true bash scripts/configure-auto-update.sh
```

Use the development branch only on a development host:

```bash
cd ~/MyPaas
AUTO_UPDATE_CHANNEL=main AUTO_UPDATE_REF=main bash scripts/configure-auto-update.sh
```

Inspect updater policy, status, and logs:

```bash
cat /etc/mypaas/update.env
cat /run/mypaas/update/status
journalctl -u mypaas-update.service
```

When `AUTO_UPDATE_ENABLED=true`, inspect the periodic timer separately:

```bash
systemctl status mypaas-update.timer
```

Run one release-aware update check manually:

```bash
bash scripts/update-dispatch.sh
```
