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

## Inspect the full-backup contract

From the installer-managed checkout:

```bash
cd ~/MyPaas
python3 scripts/backup-restore.py plan
```

This command is non-mutating.

## Create a full bundle

The DR tool has an internal default install path used by controlled tooling. Bootstrap installations normally live in `~/MyPaas`, so pass the actual checkout explicitly:

```bash
cd ~/MyPaas
python3 scripts/backup-restore.py backup \
  --install-dir "$PWD"
```

By default the output is written below:

```text
/var/lib/mypaas/backups/full-<timestamp>-<source-sha>
```

### Running persistent volumes

If managed project volumes are mounted by running containers, the tool refuses to take a potentially inconsistent snapshot unless quiescing is explicitly requested.

For a controlled snapshot that may stop and restart affected application containers during backup:

```bash
cd ~/MyPaas
python3 scripts/backup-restore.py backup \
  --install-dir "$PWD" \
  --quiesce-managed-containers
```

Plan an application maintenance window before using this option on stateful workloads. The tool restarts containers it stopped, but application-level consistency requirements remain the operator's responsibility.

## Verify a bundle before restore

Verification is non-mutating and checks the manifest plus file checksums:

```bash
python3 scripts/backup-restore.py verify \
  --bundle /var/lib/mypaas/backups/full-<timestamp>-<source-sha>
```

A copied/exported bundle should be verified again on the destination before restoration.

## Restore

Restore is destructive and requires explicit confirmation:

```bash
cd ~/MyPaas
python3 scripts/backup-restore.py restore \
  --bundle /path/to/full-backup \
  --install-dir "$PWD" \
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
cd ~/MyPaas
ENV_FILE=.env bash scripts/verify-production.sh
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
