# ADR-018: Automatic self-update for VM installs

## Status

Accepted / Implemented

Implementation note: this decision is part of the current stable VM-install contract. Stable installations use the release-aware host updater described below; periodic polling remains opt-in.

## Context

Production VM installs are repository-backed and update through the host-side updater rather than an arbitrary `git pull`. Existing installer-managed checkouts can be shallow, and upstream branch history may be rewritten or squash-updated, so checkout synchronization must remain deliberate and fail closed on dirty working trees.

The production Compose file also supports mutable compatibility tags such as `:latest`, but release updates must not select source independently from the immutable API/dashboard artifacts qualified for that release.

MyPaaS needs an updater that keeps source, migrations, Compose configuration, and application images on the same revision, without requiring Watchtower or giving another container access to the Docker socket. Production installs also need a conservative update source: an arbitrary new `main` commit must not become an installable production update before it has been intentionally published as a release.

## Decision

1. Existing installer-managed checkouts are synchronized with `git fetch` + `git reset --hard` after refusing dirty working trees. The checkout is not treated as a user-development clone.
2. Stable installations use `AUTO_UPDATE_CHANNEL=release` and resolve published GitHub Releases. `AUTO_UPDATE_CHANNEL=main` remains an explicit development-host option. Prereleases are excluded unless `AUTO_UPDATE_INCLUDE_PRERELEASES=true`.
3. A release-channel update requires the published release to identify a full 40-character Git target SHA. The dispatcher cross-checks the corresponding remote release tag resolves to the same commit, fetches that exact SHA, and verifies `FETCH_HEAD` before using it as runtime authority. This is an identity-consistency guard inside the GitHub release/repository trust boundary, not an independent signed-attestation system.
4. A resolved target is accepted only when its target commit is a descendant of the currently installed commit. The updater refuses implicit downgrades or unrelated history.
5. The image publish workflow publishes `latest` plus immutable full-Git-SHA tags. A frontend-only main revision reuses the immutable API image from its parent SHA and aliases it to the new SHA instead of rebuilding unchanged backend code.
6. `docker-compose.prod.yml` accepts `MYPAAS_IMAGE_TAG`, defaulting to `latest` for backwards compatibility.
7. `scripts/update-vm.sh` remains the full platform updater. It:
   - serializes updates with a host lock;
   - refuses dirty checkouts;
   - fetches the already-resolved target identity;
   - waits for both SHA-tagged images before changing the checkout;
   - preflights migrations and the existing control-plane runtime;
   - tags the currently running API/dashboard images locally for runtime rollback;
   - resets the managed checkout to the target SHA;
   - deploys using the same SHA image tag;
   - verifies API/Caddy/CLI health;
   - restores the previous checkout and locally tagged runtime images when deployment or verification fails.
8. `scripts/update-dispatch.sh` resolves the selected channel before delegating. A frontend-only target uses `scripts/update-dashboard.sh`, which recreates only the dashboard and verifies or rolls it back without restarting API dependencies.
9. The host updater writes an atomic status snapshot under `/run/mypaas/update/status` with `checking`, `updating`, `succeeded`, `failed`, `rolled_back`, `blocked`, or `idle` state. The dashboard receives only `/run/mypaas/update` as a read-only mount; it does not receive the parent `/run/mypaas` directory or the Caddy admin socket.
10. The host-side systemd service and path trigger are always installed on supported VMs so the authenticated API can queue a manual update through `/run/mypaas/update.request`. The API never executes host scripts from inside its container.
11. The owner-only dashboard notification surface reads the host status through a SvelteKit server route, authenticates the browser session against the internal API, and discovers published releases from GitHub. UI success is based on updater status rather than inferring success from an API restart.
12. Release publication is a qualification gate. A `release/v*` publish request may contain only its release-notes file; the release targets the current `main` SHA only after an exact-SHA CI run has succeeded and immutable API and dashboard images for that SHA exist in GHCR.
13. Automatic polling is opt-in. The timer is installed only when `AUTO_UPDATE_ENABLED=true`. No Watchtower-style Docker-socket watcher is used.

## Consequences

### Positive

- Stable VM installs advance only to intentionally published releases rather than every `main` commit.
- A moved release tag cannot silently select a different commit from the published release target; mismatched release metadata/tag identity blocks the update.
- Prereleases can be qualified explicitly without exposing them to stable installs.
- Automatic updates cannot deploy a Git revision before its exact API and dashboard artifacts are available.
- Source/config/migrations and application images stay pinned to one revision during an update.
- Frontend-only revisions keep the dashboard-only runtime fast path while preserving an immutable API identity for the release SHA.
- Owner UI can report the updater's actual terminal result, including rollback and blocked states.
- The dashboard can inspect update status without receiving the Caddy admin socket.
- The updater remains host-side and does not introduce another privileged Docker-socket container.

### Trade-offs

- Release identity still trusts GitHub release metadata plus the configured Git remote; this ADR does not add an independent signature/cosign trust root.
- Automatic rollback is best effort. A forward database migration may not be reversible by simply restoring the previous application image. Operators should keep MyPaaS backups enabled.
- SHA image tags and frontend-only aliases consume registry metadata in addition to `latest`.
- Stable release discovery depends on GitHub availability; a discovery failure leaves the running installation unchanged.
- systemd is required for dashboard-triggered and scheduled updates; `scripts/update-vm.sh` can still be run manually on other Linux init systems.

## Operations

The current operator runbook is [`../operations/update.md`](../operations/update.md). Use the installed service for a manual production check so it receives the persisted policy and host privileges:

```bash
sudo systemctl start mypaas-update.service
sudo systemctl status mypaas-update.service --no-pager
```

Enable automatic stable-release checks on an installed VM:

```bash
export MYPAAS_INSTALL_DIR="${MYPAAS_INSTALL_DIR:-$HOME/MyPaas}"
cd "$MYPAAS_INSTALL_DIR"
AUTO_UPDATE_ENABLED=true AUTO_UPDATE_INTERVAL_MINUTES=30 AUTO_UPDATE_CHANNEL=release bash scripts/configure-auto-update.sh
```

Temporarily qualify a prerelease on a test VM:

```bash
export MYPAAS_INSTALL_DIR="${MYPAAS_INSTALL_DIR:-$HOME/MyPaas}"
cd "$MYPAAS_INSTALL_DIR"
AUTO_UPDATE_CHANNEL=release AUTO_UPDATE_INCLUDE_PRERELEASES=true bash scripts/configure-auto-update.sh
```

Use the development branch only on a development host:

```bash
export MYPAAS_INSTALL_DIR="${MYPAAS_INSTALL_DIR:-$HOME/MyPaas}"
cd "$MYPAAS_INSTALL_DIR"
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
