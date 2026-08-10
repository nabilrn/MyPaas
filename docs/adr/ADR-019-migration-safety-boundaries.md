# ADR-019: VM Migration Safety Boundaries

## Status

Accepted

## Context

MyPaas supports exporting control-plane state and host-managed project data to move an installation to a new Linux VM. The original implementation had two unsafe ambiguities:

1. the Admin migration UI said running project containers were stopped during export, while the backend exporter did not quiesce them;
2. standalone Docker-to-Podman and migration scripts could bypass the backend workflow and did not preserve engine-managed named volumes.

A successful-looking archive that silently omits mutable data is worse than an explicit unsupported-state error.

## Decision

The backend Admin VM Migration workflow is the single supported export path.

Before any runtime is stopped, the exporter performs a container-storage preflight. If a MyPaas Compose project has an engine-managed volume mount, export fails closed and reports the affected volume names. Operators must either move persistent data to bind mounts under `/var/lib/mypaas/volumes` or migrate those engine volumes separately.

When preflight passes, the exporter:

1. reads projects whose desired state is `running`;
2. skips static projects because they have no application runtime;
3. stops existing Dockerfile, registry-image, and Compose runtimes directly through the container-engine adapter without changing project status in PostgreSQL;
4. creates logical PostgreSQL dumps and archives supported host-managed paths;
5. attempts to start every runtime that it stopped;
6. marks the archive `ready` only after runtime restoration succeeds.

If quiescing fails partway through, already-stopped runtimes are restarted before the export fails. If archive creation fails, a deferred restoration attempt runs. A runtime restoration failure makes the export fail rather than presenting the archive as ready.

The old `scripts/migrate-export.sh` and `scripts/migrate-to-podman.sh` entry points remain only as fail-closed compatibility stubs. In-place Docker Engine to Podman migration is not supported because engine-local container and named-volume storage is not portable merely by changing the API socket.

Fresh Podman hosts remain supported through `USE_PODMAN=true` / `scripts/install-vm.sh --podman`.

## Consequences

### Positive

- mutable bind-mounted project data is quiesced before archive creation;
- project desired state remains `running` in the database dump;
- failed runtime restoration cannot be mistaken for a successful migration;
- engine-managed Compose volumes cannot be silently omitted;
- destructive Docker-to-Podman daemon replacement is no longer exposed as a supported migration command;
- migration behavior has focused Go and script regression tests enforced by CI.

### Trade-offs

- Compose projects using named or external engine volumes must migrate those volumes separately or switch to host bind mounts before using the built-in VM exporter;
- running container-backed projects experience a maintenance window while the archive is created;
- this design deliberately avoids a more complex cross-engine volume-copy subsystem until there is a measured need for one.

## Rejected alternatives

### Automatically copy every engine volume between Docker and Podman

Rejected for now. Docker and Podman storage layouts, ownership, labels, external-volume semantics, and application-specific consistency requirements make a generic automatic copy path substantially more complex and risky than the current single-host project needs justify.

### Keep the legacy standalone exporter

Rejected. Maintaining two migration implementations would allow their safety behavior to drift again.

### Change project status to `stopped` during export

Rejected. That would write the temporary maintenance state into the database dump and lose the operator's intended post-restore desired state.
