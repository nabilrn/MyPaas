# Backup and Restore

MyPaaS has two backup paths with different scopes. Do not treat them as interchangeable.

## 1. Control-plane backup

The bundled CLI can run the normal control-plane backup service:

```bash
mypaas backup
```

Production can also schedule backups through the platform backup settings. This path is suitable for the control-plane backup contract but is not the full disaster-recovery bundle described below.

## 2. Full disaster-recovery bundle

`scripts/backup-restore.py` creates a broader recovery bundle containing:

- the production `.env` configuration;
- a custom-format control-plane PostgreSQL dump;
- static project artifacts;
- Compose workspaces;
- MyPaaS-managed named volumes;
- Compose volumes associated with active MyPaaS projects when discoverable by the supported ownership rules.

Because the bundle contains the production `.env`, it contains **secrets**. Store and transfer it as sensitive operational data.

The script still contains some historical qualification-oriented command labels (`source-preflight` and `validate-fixture-manifest`). The ordinary `backup`, `verify`, and `restore` procedures below are the operator-facing DR path; fixture-oriented commands remain historical/qualification helpers.

## Checkout path

Bootstrap installs into `$HOME/MyPaas` by default. For a custom installation, use the same checkout path that bootstrap used:

```bash
export MYPAAS_INSTALL_DIR="${MYPAAS_INSTALL_DIR:-$HOME/MyPaas}"
cd "$MYPAAS_INSTALL_DIR"
```

## Inspect the full-backup contract

```bash
python3 scripts/backup-restore.py plan
```

This command is non-mutating and does not require engine-volume access.

## Create a full bundle

Full backup needs host privileges on the default rootful runtime because it writes below `/var/lib/mypaas/backups`, talks to the rootful engine, and reads managed volume mountpoints.

```bash
export MYPAAS_INSTALL_DIR="${MYPAAS_INSTALL_DIR:-$HOME/MyPaas}"
cd "$MYPAAS_INSTALL_DIR"
sudo python3 scripts/backup-restore.py backup \
  --install-dir "$MYPAAS_INSTALL_DIR"
```

By default the output is written below:

```text
/var/lib/mypaas/backups/full-<timestamp>-<source-sha>
```

### Running persistent volumes

If managed project volumes are mounted by running containers, the tool refuses to take a potentially inconsistent snapshot unless quiescing is explicitly requested.

For a controlled snapshot that may stop and restart affected application containers during backup:

```bash
export MYPAAS_INSTALL_DIR="${MYPAAS_INSTALL_DIR:-$HOME/MyPaas}"
cd "$MYPAAS_INSTALL_DIR"
sudo python3 scripts/backup-restore.py backup \
  --install-dir "$MYPAAS_INSTALL_DIR" \
  --quiesce-managed-containers
```

Plan an application maintenance window before using this option on stateful workloads. The tool restarts containers it stopped, but application-level consistency requirements remain the operator's responsibility.

## Verify a bundle before restore

Verification is non-mutating and checks the manifest plus file checksums. Bundles created under the default root-owned backup directory may require `sudo` to read:

```bash
sudo python3 scripts/backup-restore.py verify \
  --bundle /var/lib/mypaas/backups/full-<timestamp>-<source-sha>
```

A copied/exported bundle should be verified again on the destination before restoration.

## Restore

Restore is destructive and requires explicit confirmation. The default rootful installation requires host privileges for engine/database/volume access:

```bash
export MYPAAS_INSTALL_DIR="${MYPAAS_INSTALL_DIR:-$HOME/MyPaas}"
cd "$MYPAAS_INSTALL_DIR"
sudo python3 scripts/backup-restore.py restore \
  --bundle /path/to/full-backup \
  --install-dir "$MYPAAS_INSTALL_DIR" \
  --confirm-restore
```

The current default requires the destination checkout Git SHA to match the backup's `sourceGitSha`. Preserve that guard unless you are deliberately performing a separately qualified compatibility recovery.

The restore path:

1. verifies the bundle before mutation;
2. replaces the target production configuration with the bundled configuration, keeping a pre-restore copy if one already exists;
3. restores static and Compose workspace state;
4. restores captured managed volumes;
5. restores the control-plane PostgreSQL database;
6. recreates the API service when present;
7. writes a restore report.

After a restore, redeploy/reconcile the production stack as required and run the production verifier before declaring recovery complete.

```bash
export MYPAAS_INSTALL_DIR="${MYPAAS_INSTALL_DIR:-$HOME/MyPaas}"
cd "$MYPAAS_INSTALL_DIR"
sudo env ENV_FILE="$MYPAAS_INSTALL_DIR/.env" \
  bash "$MYPAAS_INSTALL_DIR/scripts/verify-production.sh"
```

## Recovery boundaries

A full bundle is not evidence that every application database or external dependency is transactionally consistent. Application data outside MyPaaS-owned/captured storage is outside this bundle.

Do not rely on a backup stored only on the same VM that it protects. Keep at least one recovery copy outside the host for important installations.

Do not publish:

- `private/config.env`;
- the full backup archive/bundle;
- database dumps containing private data;
- restore reports if your surrounding workflow has added sensitive material.

## Historical qualification helpers

The DR tool also contains fixture-oriented `source-preflight` and `validate-fixture-manifest` commands retained for controlled qualification workflows. They are not required for ordinary operator backup/restore and should not be presented as the normal stable procedure.

The older [`beta-backup-restore-drill.md`](beta-backup-restore-drill.md) is retained as historical qualification evidence, not as the current operator runbook.

## Related documents

- [Production verification](production-verification.md)
- [VM migration](migration.md)
- [Updates](update.md)
- [Security boundaries](../SECURITY_BOUNDARIES.md)
