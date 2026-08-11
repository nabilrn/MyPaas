# ADR-020: Compose Runtime Parity

## Status

Accepted — 2026-08-11

## Context

MyPaas supports both direct registry-image deployments and repository-driven Docker Compose deployments. These paths had inconsistent runtime guarantees.

Registry-image deployments pass the project env file to `docker run --env-file`, so project environment variables enter the application container. Compose deployments passed the same file only through `docker compose --env-file`; Docker Compose uses that file for interpolation, but it does not automatically inject every variable into a service container. A repository whose Compose service omitted `environment` or `env_file` therefore behaved differently from the equivalent registry-image deployment.

Compose deployments also started the stack without explicitly refreshing image-only services, and MyPaas routed traffic as soon as `docker compose up -d` returned successfully. A main service could exit or become unhealthy after the Compose command returned while the project was still recorded as `running`, producing an origin 502 through Caddy/Cloudflare.

## Decision

MyPaas gives repository/Compose deployments the same application-runtime guarantees as registry-image deployments:

1. The generated MyPaas override adds the project env file to the selected public/main service through `env_file`. The user repository remains responsible for service topology; MyPaas remains responsible for project-level runtime configuration.
2. Normal Compose deployments explicitly refresh remote image-only services with `docker compose pull --ignore-buildable` before `up`. Buildable services remain handled by the existing build path. Compose rollback does not perform this pull, so it cannot silently replace a recorded rollback target with a newer floating image.
3. MyPaas waits for the selected main service before switching the Caddy route or persisting `running`. A running container without a Docker healthcheck is ready. A service with a healthcheck must reach `healthy`. Terminal `exited`, `dead`, or `unhealthy` states fail the deployment; transient startup states are polled with a bounded timeout.
4. Start and restart lifecycle actions apply the same readiness gate before restoring/confirming the public route.

## Consequences

- Project env variables now behave consistently between image and Compose sources.
- Repository deployments using floating image tags receive current remote images on normal deploys.
- A successful Compose CLI exit is no longer confused with application readiness.
- Existing Compose security validation, port sanitization, profiles, named volumes, networks, and build behavior remain unchanged.
- Repositories can still declare their own `environment` values; normal Docker Compose merge/precedence semantics apply.
- Deployments with slow healthchecks must become healthy within the bounded readiness timeout or fail instead of exposing an unhealthy origin.

## Verification

Regression tests cover env-file injection, Compose pull argument construction, and readiness-state evaluation. Backend `go test ./...` must pass, followed by the repository CI suite before the PR is considered ready to merge.
