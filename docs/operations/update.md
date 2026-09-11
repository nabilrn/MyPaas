# Updating MyPaaS

Production VM updates are host-side and release-aware. The supported stable channel follows published GitHub releases rather than every new commit on `main`.

For a published release, the dispatcher reads the release's full Git target SHA, cross-checks the corresponding remote tag resolves to the same commit, fetches that exact SHA, and then uses the same SHA for source plus immutable API/dashboard image selection. This closes the mutable-tag race inside the normal GitHub release/repository trust boundary; it is not a separate signed-source attestation system.

## Stable update policy

For production, use:

```text
AUTO_UPDATE_CHANNEL=release
AUTO_UPDATE_INCLUDE_PRERELEASES=false
```

The updater resolves the latest stable release, requires a full 40-character release target SHA, verifies the remote release tag resolves to that same SHA, fetches the exact commit, validates ancestry, waits for immutable API/dashboard images for that SHA, applies the matching source revision, and verifies the resulting control plane.

`AUTO_UPDATE_CHANNEL=main` is an explicit development-host option. Do not use it as the normal production stable policy.

## Checkout path

Bootstrap installs into `$HOME/MyPaas` by default. For a custom installation, set `MYPAAS_INSTALL_DIR` to the same path that was used during bootstrap:

```bash
export MYPAAS_INSTALL_DIR="${MYPAAS_INSTALL_DIR:-$HOME/MyPaas}"
cd "$MYPAAS_INSTALL_DIR"
```

## Check current updater policy

```bash
cat /etc/mypaas/update.env
```

The policy file is managed by `scripts/configure-auto-update.sh`; do not edit generated systemd units as the primary configuration path.

## Run one update check manually

Use the installed systemd service so `/etc/mypaas/update.env`, host privileges, locking, and status-file behavior match dashboard/scheduled updates:

```bash
sudo systemctl start mypaas-update.service
sudo systemctl status mypaas-update.service --no-pager
```

Inspect the result with:

```bash
cat /run/mypaas/update/status
journalctl -u mypaas-update.service
```

Do not replace this with a manual `git pull` followed by arbitrary Compose commands. `scripts/update-dispatch.sh` is the service implementation detail; direct interactive execution can bypass the policy environment loaded by systemd.

If the installed revision is already current, the updater still reconciles host dependencies such as `mypaas-statd` and relevant runtime environment drift.

## Enable periodic stable updates

Periodic polling is opt-in:

```bash
export MYPAAS_INSTALL_DIR="${MYPAAS_INSTALL_DIR:-$HOME/MyPaas}"
cd "$MYPAAS_INSTALL_DIR"
AUTO_UPDATE_ENABLED=true \
AUTO_UPDATE_INTERVAL_MINUTES=30 \
AUTO_UPDATE_CHANNEL=release \
AUTO_UPDATE_INCLUDE_PRERELEASES=false \
bash scripts/configure-auto-update.sh
```

This installs/enables the update timer in addition to the host update path trigger.

The update path trigger is installed even when periodic polling is disabled so the authenticated dashboard can request a host-side update without receiving direct script execution or Docker-socket authority.

## Disable periodic polling

Keep dashboard-triggered/manual update support while disabling the timer:

```bash
export MYPAAS_INSTALL_DIR="${MYPAAS_INSTALL_DIR:-$HOME/MyPaas}"
cd "$MYPAAS_INSTALL_DIR"
AUTO_UPDATE_ENABLED=false \
AUTO_UPDATE_CHANNEL=release \
bash scripts/configure-auto-update.sh
```

## Inspect status and logs

```bash
cat /run/mypaas/update/status
systemctl status mypaas-update.path
systemctl status mypaas-update.timer
journalctl -u mypaas-update.service
```

The status file can report states including checking, updating, succeeded, failed, rolled back, blocked, or idle.

If periodic updates are disabled, the timer may not exist/be enabled; the path trigger should remain available on a supported systemd installation.

## Update safety behavior

The current updater intentionally:

- refuses a dirty installer-managed checkout;
- resolves release metadata to a full target SHA before mutation;
- cross-checks the release tag against that same target SHA;
- fetches and verifies the exact release SHA rather than using a fetched tag as runtime authority;
- refuses an implicit downgrade or unrelated history on the stable release path;
- waits for immutable target API and dashboard images;
- preflights migrations and existing runtime/network state;
- preserves the existing Docker/Podman engine contract;
- reconciles `mypaas-statd`;
- tags currently running API/dashboard images locally before a full update when possible;
- deploys the resolved source and images at one Git identity;
- runs production verification after deployment;
- attempts a best-effort runtime/checkout rollback if deployment or verification fails.

Automatic rollback cannot make an arbitrary forward database migration inherently reversible. Keep backups enabled and maintain an external recovery copy for important installations.

## Frontend-only fast path

When the resolved target changes only `frontend/`, the dispatcher can update the dashboard without restarting unchanged API dependencies. The target dashboard image is still required and rollback is verified independently.

This is an updater implementation detail; operators should invoke the configured update service rather than selecting the fast path manually.

## Qualifying a prerelease

Only use this on a test/qualification host:

```bash
export MYPAAS_INSTALL_DIR="${MYPAAS_INSTALL_DIR:-$HOME/MyPaas}"
cd "$MYPAAS_INSTALL_DIR"
AUTO_UPDATE_CHANNEL=release \
AUTO_UPDATE_INCLUDE_PRERELEASES=true \
bash scripts/configure-auto-update.sh
```

Do not enable prereleases on a stable production installation unless that risk is intentional.

## Development-host channel

For a host intentionally following a development ref:

```bash
export MYPAAS_INSTALL_DIR="${MYPAAS_INSTALL_DIR:-$HOME/MyPaas}"
cd "$MYPAAS_INSTALL_DIR"
AUTO_UPDATE_CHANNEL=main \
AUTO_UPDATE_REF=main \
bash scripts/configure-auto-update.sh
```

This is not the normal stable operator path. The `main` channel intentionally follows the configured development ref rather than release metadata.

## After a manual recovery or policy change

Run production verification when you have manually repaired/reconciled runtime state:

```bash
export MYPAAS_INSTALL_DIR="${MYPAAS_INSTALL_DIR:-$HOME/MyPaas}"
cd "$MYPAAS_INSTALL_DIR"
sudo env ENV_FILE="$MYPAAS_INSTALL_DIR/.env" \
  bash "$MYPAAS_INSTALL_DIR/scripts/verify-production.sh"
```

## Related documents

- [Installation](../installation.md)
- [Configuration](../configuration.md)
- [Backup and restore](backup-restore.md)
- [Production verification](production-verification.md)
- [ADR-018: automatic self-update](../adr/ADR-018-automatic-self-update.md)
