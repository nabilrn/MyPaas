# ADR-021: Persistent Image Storage

## Status

Accepted

## Context

Registry-based image deployments replace containers during deploy and rollback. Application data stored only in the container writable layer is therefore disposable.

Docker image `VOLUME` declarations are a useful persistence signal, but requiring an application image to declare `VOLUME` can create an anonymous Docker volume when the same image is run outside MyPaas. Some applications deliberately require the orchestrator to attach a stable volume and must continue to fail closed when that does not happen.

## Decision

MyPaas treats either of these image metadata sources as a request for project-scoped durable storage:

1. Docker `Config.Volumes` targets.
2. The optional image label `io.mypaas.persistent-volumes`, containing a comma-separated list of container paths.

Targets from both sources are trimmed, deduplicated, sorted, and passed through the same validation. Every target must be an absolute container path and cannot be `/`.

For each accepted target, MyPaas derives the existing deterministic Docker-managed named-volume identity from the stable runtime name plus target path, creates it if needed, and attaches it with `--mount type=volume`. Redeploy and rollback therefore reattach the same volume even when the replacement container name changes.

The label is only metadata. It does not cause Docker itself to create an anonymous volume, so an image can keep a fail-closed runtime check for an explicitly attached durable mount.

MyPaas does not allow image metadata to request arbitrary host bind paths. Image-declared persistence always resolves to Docker-managed named volumes owned by MyPaas.

## Consequences

- Existing images that declare Docker `VOLUME` continue to work unchanged.
- Images that intentionally avoid Docker `VOLUME` can still opt into stable MyPaas storage using the label.
- Stateless images remain unchanged when neither metadata source is present.
- Invalid relative/root targets fail deployment rather than silently falling back to disposable storage.
- The metadata contract stays narrow and requires no project-schema migration or new storage service.
