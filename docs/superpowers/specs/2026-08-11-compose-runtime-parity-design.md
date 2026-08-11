# Compose Runtime Parity Design

## Problem

Registry/image deployments and repository/Compose deployments do not currently apply the same runtime guarantees. Image deployments explicitly pass the MyPaas project env file to `docker run`, while Compose deployments only pass `--env-file` to the Compose CLI. That file is used for Compose interpolation and does not automatically become the container environment unless the Compose service declares `environment`/`env_file`.

Compose deployments also do not explicitly refresh image-only services and are marked `running` immediately after `docker compose up -d` succeeds, even if the public service exits or remains unhealthy.

## Design

1. The generated MyPaas Compose override injects the project env file into the selected main service using `env_file`. This preserves the repository Compose file as the source of topology while making project-level environment variables behave consistently with registry/image mode.
2. Compose deployment uses `--pull always` when running normal deployments so services declared with `image:` refresh floating tags. Rollback continues to use `--no-build` and does not force a newer image over a recorded rollback target.
3. After `docker compose up -d`, MyPaas waits for the selected main service to become ready before switching Caddy and persisting `running`. A running container without a healthcheck is considered ready. A container with a healthcheck must become `healthy`. `exited`, `dead`, or `unhealthy` fail deployment. The wait is bounded and context-aware.
4. Readiness inspection belongs in `internal/container`; deployment orchestration only selects the main service and decides when routing/status changes occur.

## Security

The env file remains created with mode `0600`. Its values are not copied into deployment logs. Compose security sanitization remains unchanged.

## Compatibility

- Existing Compose `environment:` entries still work and can override values according to Docker Compose merge semantics.
- Named volumes, networks, profiles, build contexts, and MyPaas port sanitization remain unchanged.
- Image deployments are unchanged.
- Compose rollback does not force a registry refresh.

## Verification

Regression tests cover generated `env_file`, Compose pull policy argument construction, readiness-state parsing, and readiness gating before route activation. The repository CI (`go test ./...`, frontend checks, script checks) must pass before merge.
