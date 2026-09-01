# ADR-018: Automatic self-update for VM installs

## Status

Proposed

## Context

Production VM installs are repository-backed and currently update by rerunning `scripts/bootstrap.sh`. The bootstrap uses a shallow fetch followed by an `ff-only` merge. When the upstream branch is rewritten or squash/force-updated, an existing shallow checkout can fail with `refusing to merge unrelated histories` even though the install directory is intentionally managed by MyPaas.

The production Compose file also consumes mutable `:latest` API/dashboard images. Pulling `latest` immediately after observing a new Git commit creates a race: the source checkout can advance before the image publish workflow for that commit has completed.

MyPaas needs an updater that keeps source, migrations, Compose configuration, and application images on the same revision, without requiring Watchtower or giving another container access to the Docker socket.

## Decision

1. Existing installer-managed checkouts are synchronized with `git fetch` + `git reset --hard FETCH_HEAD` after refusing dirty working trees and verifying the configured origin. The checkout is not treated as a user-development clone.
2. The image publish workflow publishes both `latest` and an immutable full Git SHA tag for API and dashboard images.
3. `docker-compose.prod.yml` accepts `MYPAAS_IMAGE_TAG`, defaulting to `latest` for backwards compatibility.
4. `scripts/update-vm.sh`:
   - serializes updates with a host lock;
   - refuses dirty checkouts;
   - fetches the configured ref;
   - does nothing when the revision is unchanged;
   - waits for both SHA-tagged images before changing the checkout;
   - tags the currently running API/dashboard images locally for best-effort runtime rollback;
   - resets the managed checkout to the target SHA;
   - deploys using the same SHA image tag;
   - verifies API/Caddy/CLI health;
   - restores the previous checkout and locally tagged runtime images if deployment/verification fails.
5. The host-side systemd service and path trigger are always installed on supported VMs so the authenticated dashboard can queue a manual update through `/run/mypaas/update.request`. The API never executes host scripts from inside its container.
6. Automatic polling is opt-in. The timer is installed only when `AUTO_UPDATE_ENABLED=true`. The default interval is 30 minutes and the minimum accepted interval is 5 minutes.
7. No Watchtower-style Docker socket watcher is used. Updates remain coordinated through MyPaas' own migration and verification scripts.

## Consequences

### Positive

- Upstream history rewrites no longer break rerunning the bootstrap on a clean installer-managed checkout.
- Automatic updates cannot deploy a Git revision before its API and dashboard artifacts are published.
- Source/config/migrations and application images are pinned to one revision during an automatic update.
- The updater is host-side and does not introduce another privileged Docker-socket container.
- Scheduled polling remains disabled unless automatic updates are explicitly enabled; the owner-triggered host update path remains available.

### Trade-offs

- Automatic rollback is best effort. A forward database migration may not be reversible by simply restoring the previous application image. Operators should keep MyPaas backups enabled.
- SHA image tags consume registry metadata in addition to `latest`.
- systemd is required for dashboard-triggered and scheduled updates; `scripts/update-vm.sh` can still be run manually on other Linux init systems.
- Mutable development branches are supported because the current project uses `main` operationally, but stable release tags remain preferable for conservative production environments.

## Operations

Enable automatic updates on an installed VM:

```bash
cd ~/MyPaas
AUTO_UPDATE_ENABLED=true AUTO_UPDATE_INTERVAL_MINUTES=30 bash scripts/configure-auto-update.sh
```

Inspect the timer and logs:

```bash
systemctl status mypaas-update.timer
journalctl -u mypaas-update.service
```

Run one update check manually:

```bash
bash scripts/update-vm.sh
```
