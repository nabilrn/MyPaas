# ADR-017: Container Registry as a Deployment Source

## Status

Accepted

## Amendment

The original public-image-only authentication boundary in this ADR was amended by [ADR-022](ADR-022-private-registry-auth.md). Current image-mode deployments may use one bounded installation-level credential scoped to a configured registry host. The rest of this ADR remains applicable; the historical text below is retained to show the original decision sequence.

## Context

MyPaas originally models every project as source code from a Git repository. The deployment engine clones a branch, optionally detects a base directory, then builds a Dockerfile, starts a Compose application, or serves static output.

Issue #2 requires a second project source: an already-built OCI container image such as `nginx:latest`, `ghcr.io/org/app:v1.2.0`, or an image hosted by another Docker-compatible public registry.

Treating a registry image as a fake Git repository would leak repository-only concepts into the UI and deployment lifecycle. At the same time, the existing single-container runtime already provides the required port mapping, resource limits, environment file handling, lifecycle actions, logs, metrics, and Caddy routing.

## Decision

MyPaas supports two project source types:

- **Git Repository** — existing Dockerfile, Compose, and static deployment modes.
- **Container Registry** — a pre-built public OCI image deployed with the new `image` deployment mode.

`sourceType` is an API/UI concept derived from deployment mode instead of a second persisted discriminator:

- `deploy_mode = image` → `sourceType = registry`
- every other deployment mode → `sourceType = git`

Registry projects persist an `image_ref` on `projects`. The existing `repo_url` and `branch` columns remain non-null for backwards compatibility; they are not used by image-mode deployments. This avoids a broad nullable-schema migration while keeping current project records and generated sqlc types stable.

The image deployment flow is:

1. validate the image reference;
2. `docker pull` the public image;
3. resolve a repository digest when Docker exposes one and store it on the deployment record;
4. materialize the project's encrypted environment variables into the normal temporary env file;
5. start the image through the existing single-container blue/green switch path;
6. apply project port, memory, CPU, network, and Caddy routing exactly like Dockerfile deployments;
7. record the deployment as running.

Rollback reuses the stored immutable digest when available. If the image is no longer local, MyPaas pulls the recorded reference before switching containers.

## Security boundary

This section records the original ADR-017 boundary. It was later amended by ADR-022 for bounded private-registry authentication.

The original change intentionally supported **public images only**. Private registry credentials were not accepted, persisted, passed on the command line, or written into deployment logs in this change. Registry authentication required a separate credential model and lifecycle because credentials are secrets and may need registry-specific scopes/rotation.

Image references are treated as data, not shell fragments. MyPaas invokes Docker with argument arrays and rejects whitespace, URL schemes, option-like references, NULs, and empty values before execution.

## UI

The New Project flow begins with the existing segmented-choice visual language and asks for a deployment source before source-specific fields:

- Git Repository shows repository URL, branch, base directory, repository inspection, and runtime detection.
- Container Registry shows the image reference and explains that MyPaas will pull rather than build the image.

Shared project controls such as project name, app port, resource profile, environment variables, database option, review panel, and routing remain visually and behaviorally consistent.

Project overview, control panel, and settings display source-aware labels. Git webhook configuration is hidden for registry projects because registry projects have no repository push event.

## Consequences

### Positive

- Public Docker Hub, GHCR, and other OCI-compatible registry images can be deployed without source code.
- Existing container lifecycle, metrics, logs, Caddy, quota, and rollback paths are reused rather than duplicated.
- Existing Git projects require no migration beyond a nullable column and continue using the same API defaults.
- The source model can later grow to private registries without redesigning project deployment modes.

### Trade-offs

- `repo_url` and `branch` remain populated with compatibility values for registry projects even though they are not semantically used.
- Registry deployments do not have commit metadata.
- Automatic redeploy on a registry tag change is not included; users trigger deployment manually.
- The original public-image-only authentication boundary was superseded by the bounded credential model in ADR-022.
