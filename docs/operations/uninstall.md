# Uninstall

`scripts/uninstall-vm.sh` is a **destructive host cleanup tool**. It is not a normal troubleshooting command.

The script explicitly requires the operator to type:

```text
DESTROY
```

before proceeding.

## What it removes

The current uninstall path removes MyPaaS-owned state including:

- the production control-plane Compose stack, associated volumes, and images selected by that stack;
- MyPaaS control/project/routing networks;
- host update systemd units and update policy/status/request state;
- `mypaas-statd` by default;
- `/var/lib/mypaas`, including MyPaaS-managed volumes/static/Compose/backup paths;
- `/tmp/mypaas/builds`;
- the production `.env`;
- the installer-managed source checkout itself.

It also retains cleanup logic for legacy MyPaaS firewall-helper state from older installations.

The script warns that projects, databases, configuration, and backups on the host can be permanently deleted. Treat that warning literally.

## Before uninstalling

If any data matters:

1. create a full recovery bundle;
2. verify the bundle;
3. copy it **off the VM**;
4. verify the external copy is accessible;
5. record any application/external data that is outside MyPaaS-managed storage.

See [Backup and restore](backup-restore.md).

Do not rely on `/var/lib/mypaas/backups` as your only backup immediately before uninstall: that directory is removed by the uninstall path.

## Run uninstall

From the current MyPaaS checkout:

```bash
cd ~/MyPaas
bash scripts/uninstall-vm.sh
```

Review the warning and type `DESTROY` only when permanent removal is intended.

## Keep mypaas-statd

The script supports preserving the statd host service while removing the main MyPaaS installation:

```bash
cd ~/MyPaas
REMOVE_STATD=false bash scripts/uninstall-vm.sh
```

Use this only when another intended workflow still owns/uses that service. Normal complete removal uses the default `REMOVE_STATD=true`.

## What uninstall does not do

Do not assume this script removes unrelated host packages or unrelated containers simply because Docker/Podman is installed. Its purpose is MyPaaS-owned runtime/state cleanup.

Conversely, data in external providers or manually managed host paths may survive because it is outside MyPaaS ownership. Review those resources separately.

## Reinstall

A later fresh installation should use the version-pinned production path in [Installation](../installation.md), not a retained copy of the deleted production `.env` unless restoring an intentional trusted backup.
