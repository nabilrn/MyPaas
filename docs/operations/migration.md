# VM Migration

The **Administration → Migration** workflow is the supported MyPaaS VM export path. It is designed to move a single-host installation while failing closed on storage states the built-in exporter cannot preserve safely.

See [ADR-019](../adr/ADR-019-migration-safety-boundaries.md) for the authoritative storage-safety boundary.

## Before migration

Before preparing an export:

- create and verify a current backup;
- plan a maintenance window for running container-backed projects;
- confirm the destination Linux host can run the intended Docker-compatible engine;
- review persistent storage, especially Compose named/external volumes;
- keep the source VM available until the destination has passed verification.

## Storage preflight

The exporter performs its portability preflight **before** stopping application runtimes.

If an active MyPaaS Compose project uses engine-managed volume state that the built-in migration package cannot guarantee to preserve, export fails closed and identifies the affected volumes.

In that case, move the application to supported host-managed storage or migrate the engine volume separately using an application-specific/engine-specific procedure. Do not bypass the preflight merely to obtain a successful-looking archive.

## Prepare the export

From the owner dashboard:

```text
Administration → Migration
```

Use the dashboard workflow to prepare the migration package and wait for it to reach the ready state.

For running Dockerfile, image, and Compose projects, the exporter preserves desired state while temporarily quiescing existing runtimes for the archive operation. Static projects have no runtime to stop. A failed quiesce/archive/runtime restoration causes the export to fail rather than presenting the package as ready.

## Destination import

When the package is ready, Administration shows two separate copy actions:

1. **Copy command** — contains no migration token and pins the destination checkout to the concrete 40-character `MYPAAS_BUILD_SHA` reported by the running source control plane.
2. **Copy migration URL** — contains the temporary download credential and must be treated as a secret.

Run the generated command on the destination VM. It initializes a new checkout, fetches only the source build SHA, verifies `FETCH_HEAD` matches that SHA, checks out detached, and starts:

```bash
bash scripts/install-migration.sh
```

The helper does **not** accept the remote migration URL as a command-line argument. On an interactive terminal it prompts for the URL with input hidden. For controlled automation it can read the URL from stdin. It passes the secret URL to `curl` through stdin/config input, downloads into a private temporary directory, then gives the existing installer only a local `file://` archive path.

The migration URL is sensitive access to a secret-bearing package. The token is not consumed after one successful download; it remains usable until the export expires (currently 24 hours). Clipboard contents and any external automation that stores the URL must therefore be treated as credentials and cleared/rotated by expiry.

Example non-interactive shape for a trusted automation environment:

```bash
printf '%s\n' "$MYPAAS_MIGRATION_URL" | bash scripts/install-migration.sh
```

Do not put the migration URL directly into shell arguments, copied command text, process argv, or ordinary shell history.

### Checkout path

The generated command defaults to `$HOME/MyPaas` and honors `MYPAAS_INSTALL_DIR` when explicitly set. It refuses to overwrite an existing path so migration does not silently mutate another checkout.

### Release identity

Migration restore is pinned to the exact source build SHA, not the repository default branch or a mutable release tag. If the source API does not expose a concrete 40-character build SHA, the dashboard blocks generation of the destination restore command rather than guessing a revision.

This is an identity-consistency guard within the configured GitHub repository trust boundary; it is not a separate signed-source attestation system.

## Engine changes

Do not switch an existing installation from Docker Engine to Podman (or the reverse) by changing the socket and rerunning bootstrap in place. Bootstrap refuses detected split/mismatched runtime state.

A fresh destination host may use the supported runtime choice, but application/engine-managed storage still has to satisfy the migration portability boundary. Changing engine does not magically make Docker-local named volumes portable.

The old standalone `scripts/migrate-export.sh` and `scripts/migrate-to-podman.sh` paths are compatibility stubs and are **not** supported migration entry points.

## After import

On the destination:

1. confirm the production stack is running;
2. verify the configured public domain/delivery path now reaches the destination;
3. run production verification;
4. inspect project desired/runtime state;
5. verify stateful application data explicitly;
6. verify DB Studio/database access where used;
7. only retire the source VM after the destination has passed the required checks.

Production verifier:

```bash
export MYPAAS_INSTALL_DIR="${MYPAAS_INSTALL_DIR:-$HOME/MyPaas}"
cd "$MYPAAS_INSTALL_DIR"
sudo env ENV_FILE="$MYPAAS_INSTALL_DIR/.env" \
  bash "$MYPAAS_INSTALL_DIR/scripts/verify-production.sh"
```

A control-plane verifier pass does not prove every application-specific persistent dataset is correct. Validate important workloads independently.

## Related documents

- [Backup and restore](backup-restore.md)
- [Production verification](production-verification.md)
- [Installation](../installation.md)
- [ADR-019: VM Migration Safety Boundaries](../adr/ADR-019-migration-safety-boundaries.md)
